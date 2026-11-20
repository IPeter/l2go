package game

import (
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// Boats (Plan 13, A14): the vehicle engine of Mobius Vehicle, Boat, BoatAI
// and BoatManager. A boat sails a path point by point, each with its own
// speed and turn speed (a speed of 0 teleports it); passengers move with it;
// a monitor restarts a boat that stood still for a second; at the end of a
// path the boat's engine, its schedule (boat_schedule.go), runs again.
// Players board, walk on the deck and leave with the vehicle requests
// (vehicle_player.go). The schedules are Lua data (scripts/vehicles/).
func init() { registerFeature("vehicle", setupVehicles) }

const (
	// boatMonitorPeriod is the stuck monitor's period (Mobius 1,000 ms).
	boatMonitorPeriod = time.Second
	// boatEngineDelay is Mobius runEngine(10) at the end of a path.
	boatEngineDelay = 10 * time.Millisecond
	// boatRange: players board within 1,000 of the boat, pay within 1,000
	// and leave for a point within 1,000 (Mobius).
	boatRange = 1000
	// boatDockSlots is the size of Mobius BoatManager._docksBusy: dock IDs
	// 1 and 2 fit, the Rune dock (3) does not and is never busy.
	boatDockSlots = 3
	// boatSound is the horn of the docks (Mobius PlaySound of the schedules).
	boatSound = "itemsound.ship_arrival_departure"
	// sysMsgNoTicket is "You do not possess the correct ticket to board the
	// boat." (Mobius YOU_DO_NOT_POSSESS_THE_CORRECT_TICKET_TO_BOARD_THE_BOAT).
	sysMsgNoTicket serverpackets.SystemMessageID = 402
)

// vehicleFeature is the boat state on the Server (Server.vehicles).
// Loop-only.
type vehicleFeature struct {
	// boats holds the boats by object ID (Mobius BoatManager._boats);
	// moving lists those under way.
	boats  map[world.ObjectID]*Boat
	moving []*Boat
	// docks is Mobius _docksBusy.
	docks [boatDockSlots]bool
	// schedules are the running schedules (boat_schedule.go).
	schedules []*boatSchedule
	radius    int
}

// Boat is a ferry in the world (Mobius Boat).
type Boat struct {
	world.Base
	srv *Server
	// name is the schedule's; boatID the Mobius boat ID.
	name   string
	boatID int

	// speed and rotation are the move and turn speeds (VehicleStat).
	speed    float64
	rotation int
	// path is the path under way, runState the point it sails to (Mobius
	// _currentPath, _runState).
	path     []scripting.VehiclePoint
	runState int
	// move is the leg under way, nil while the boat stands.
	move *boatMove
	// monitor is the stuck monitor; monitorPos where it last saw the boat.
	monitor    world.TimerID
	monitorPos world.Position
	// dock is Mobius _dockId; the schedules never set it.
	dock int
	// teleporting is set while the boat teleports its passengers along.
	teleporting bool
	// passengers are the players who paid (Mobius _passengers).
	passengers []*Player
	// engine is the schedule's step (Mobius _engine).
	engine func()
}

var _ InfoProvider = (*Boat)(nil)

// boatMove is a leg of a path (Mobius MoveData).
type boatMove struct {
	dest world.Position
	walk walkTrack
}

// playerVehicle is a player's boat (Mobius Player._vehicle and
// _inVehiclePosition).
type playerVehicle struct {
	boat *Boat
	// pos is where the player stands on the deck, when hasPos.
	pos    world.Position
	hasPos bool
	// peace is set while boarding put the player in the peace zone.
	peace bool
}

// setupVehicles registers the vehicle requests and the movement tick, and
// starts the schedules of the scripts once the world runs.
func setupVehicles(s *Server) error {
	f := &s.vehicles
	f.boats = make(map[world.ObjectID]*Boat)
	f.radius = s.gs.World.BoatBroadcastRadius
	if err := s.setupVehicleRequests(); err != nil {
		return err
	}
	s.loop.OnTick(s.boatTick)
	s.OnTeleport(s.vehicleTeleport)
	s.OnPlayerLeave(s.vehicleLeave)
	if s.scripts != nil {
		routes := s.scripts.VehicleRoutes()
		s.loop.Post(func() { s.StartBoats(routes) })
	}
	return nil
}

// newBoat puts a boat into the world (Mobius BoatManager.getNewBoat), or
// returns nil when boats are not allowed.
func (s *Server) newBoat(name string, boatID int, at world.Position) *Boat {
	if !s.gs.World.AllowBoat {
		return nil
	}
	id, err := s.world.IDs().Next()
	if err != nil {
		s.log.Error("game: no object ID for a boat", "boat", name, "err", err)
		return nil
	}
	b := &Boat{Base: world.NewBase(id, at), srv: s, name: name, boatID: boatID}
	if err := s.AddObject(b); err != nil {
		s.log.Error("game: cannot put a boat into the world", "boat", name, "pos", at, "err", err)
		_ = s.world.IDs().Release(id)
		return nil
	}
	s.vehicles.boats[id] = b
	return b
}

// Boat returns the boat with the object ID (Mobius BoatManager.getBoat).
// Loop-only.
func (s *Server) Boat(id world.ObjectID) (*Boat, bool) {
	b, ok := s.vehicles.boats[id]
	return b, ok
}

// ObjectID returns the boat's object ID as the packets carry it.
func (b *Boat) ObjectID() int32 { return int32(b.ID()) } //nolint:gosec // object IDs fit in int32

// Name returns the boat's schedule name.
func (b *Boat) Name() string { return b.name }

// Moving reports whether the boat is under way (Mobius isMoving).
func (b *Boat) Moving() bool { return b.move != nil }

// Passengers returns the players who paid for the ride. Loop-only.
func (b *Boat) Passengers() []*Player { return slices.Clone(b.passengers) }

// InfoFor is Mobius Boat.sendInfo and BoatAI.describeStateToPlayer:
// VehicleInfo, and VehicleDeparture while the boat moves.
func (b *Boat) InfoFor(*Player) [][]byte {
	out := [][]byte{b.infoPacket()}
	if b.move != nil {
		out = append(out, b.departurePacket())
	}
	return out
}

func (b *Boat) infoPacket() []byte {
	pos := b.Position()
	return serverpackets.NewVehicleInfoPacket(serverpackets.VehicleInfo{
		ObjectID: b.ObjectID(), X: pos.X, Y: pos.Y, Z: pos.Z, Heading: pos.Heading,
	})
}

// departurePacket is Mobius VehicleDeparture: the leg's destination and the
// speeds.
func (b *Boat) departurePacket() []byte {
	var dest world.Position
	if b.move != nil {
		dest = b.move.dest
	}
	return serverpackets.NewVehicleDeparturePacket(serverpackets.VehicleDeparture{
		ObjectID:      b.ObjectID(),
		MoveSpeed:     int32(b.speed), //nolint:gosec // boat speeds are small
		RotationSpeed: int32(b.rotation), //nolint:gosec // small
		X:             dest.X, Y: dest.Y, Z: dest.Z,
	})
}

func (b *Boat) startedPacket(state int32) []byte {
	return serverpackets.NewVehicleStartedPacket(serverpackets.VehicleStarted{ObjectID: b.ObjectID(), State: state})
}

// broadcast sends body to the players who see the boat (Mobius
// broadcastPacket).
func (s *Server) boatBroadcast(b *Boat, body []byte) {
	if b.InWorld() {
		s.world.Broadcast(b, body, false)
	}
}

// boatPointPos is a path point's position.
func boatPointPos(p scripting.VehiclePoint) world.Position {
	return world.Position{X: int32(p.X), Y: int32(p.Y), Z: int32(p.Z)} //nolint:gosec // world coordinates fit
}

// boatExecutePath is Mobius Vehicle.executePath: the boat takes the speeds
// of the first point and sails there.
func (s *Server) boatExecutePath(b *Boat, path []scripting.VehiclePoint) {
	b.runState = 0
	b.path = path
	if len(path) == 0 {
		return // setIntentionActive: nothing for a boat
	}
	p := path[0]
	if p.Speed > 0 {
		b.speed = float64(p.Speed)
	}
	if p.Rotation > 0 {
		b.rotation = p.Rotation
	}
	s.boatMoveTo(b, boatPointPos(p))
}

// boatMoveTo is BoatAI.moveTo: VehicleStarted when the boat stood, the
// walk (Creature.moveToLocation), then VehicleDeparture.
func (s *Server) boatMoveTo(b *Boat, to world.Position) {
	if !b.InWorld() {
		return
	}
	if b.move == nil {
		s.boatBroadcast(b, b.startedPacket(1))
	}
	s.boatMoveToLocation(b, to)
	s.boatBroadcast(b, b.departurePacket())
}

// boatMoveToLocation is Mobius Creature.moveToLocation(x, y, z, 0) for a
// flying creature without geodata: a boat already there has arrived.
func (s *Server) boatMoveToLocation(b *Boat, to world.Position) {
	if !(b.speed > 0) || !b.InWorld() {
		return
	}
	cur := b.Position()
	dx, dy, dz := walkDelta(cur, to.X, to.Y, to.Z)
	distance := math.Hypot(dx, dy)
	vertical := distance == 0 && dz != 0
	if vertical {
		distance = math.Abs(dz)
	}
	if distance < 1 {
		s.boatArrived(b)
		return
	}
	if !vertical {
		b.SetHeading(walkHeading(dx/distance, dy/distance))
	}
	s.boatStartLeg(b, to)
}

// boatStartLeg starts a leg towards to and lists the boat as moving (Mobius
// MovementTaskManager.registerMovingObject).
func (s *Server) boatStartLeg(b *Boat, to world.Position) {
	to.Heading = b.Position().Heading
	b.move = &boatMove{dest: to, walk: walkTrack{started: gameTick(s.Now())}}
	if !slices.Contains(s.vehicles.moving, b) {
		s.vehicles.moving = append(s.vehicles.moving, b)
	}
}

// boatTick is the boats' part of Mobius MovementTaskManager: every boat
// under way advances by the game ticks since its last update, takes its
// passengers along (Vehicle.updatePosition) and, at the end of a leg,
// sails on (notifyActionArrived).
func (s *Server) boatTick(now time.Time) {
	f := &s.vehicles
	if len(f.moving) == 0 {
		return
	}
	tick := gameTick(now)
	for _, b := range slices.Clone(f.moving) {
		s.advanceBoat(b, tick)
	}
	f.moving = slices.DeleteFunc(f.moving, func(b *Boat) bool { return b.move == nil || !b.InWorld() })
}

// advanceBoat moves b along its leg (Mobius Vehicle.updatePosition).
func (s *Server) advanceBoat(b *Boat, tick int64) {
	m := b.move
	if m == nil || !b.InWorld() {
		return
	}
	cur := b.Position()
	next, arrived, ok := m.walk.step(cur, m.dest, b.speed, 0, tick, true)
	if !ok {
		return
	}
	if next != cur {
		if err := s.MoveObject(b, next); err != nil {
			s.log.Warn("game: a boat left the world", "boat", b.name, "pos", next, "err", err)
			b.move = nil
			return
		}
		s.boatCarryPassengers(b)
	}
	if arrived && b.move == m {
		s.boatArrived(b)
	}
}

// boatCarryPassengers puts every passenger where the boat is (Mobius
// player.setXYZ(boat) and revalidateZone).
func (s *Server) boatCarryPassengers(b *Boat) {
	pos := b.Position()
	for _, p := range slices.Clone(b.passengers) {
		if p.vehicle.boat != b || !p.InWorld() {
			continue
		}
		to := pos
		to.Heading = p.Position().Heading
		if err := s.MoveObject(p, to); err != nil {
			s.log.Debug("game: cannot carry a passenger", append(p.logArgs(), "boat", b.name, "err", err)...)
		}
	}
}

// boatArrived is Mobius notifyActionArrived for a boat: it sails to the next
// point of its path; the end of the path runs the engine.
func (s *Server) boatArrived(b *Boat) {
	b.move = nil
	s.boatNextPoint(b)
}

// boatNextPoint is Mobius Vehicle.moveToNextRoutePoint (with Boat's
// VehicleDeparture): the next point of the path at its speeds, or a
// teleport for a speed of 0; past the last point the path ends, the
// monitor stops and the engine runs 10 ms later.
func (s *Server) boatNextPoint(b *Boat) bool {
	b.move = nil
	if b.path != nil {
		b.runState++
		if b.runState < len(b.path) {
			p := b.path[b.runState]
			if b.InWorld() {
				if p.Speed == 0 {
					to := boatPointPos(p)
					to.Heading = int32(p.Rotation) //nolint:gosec // a heading
					s.boatTeleport(b, to)
					s.boatStopMonitor(b)
					b.path = nil
				} else {
					if p.Speed > 0 {
						b.speed = float64(p.Speed)
					}
					if p.Rotation > 0 {
						b.rotation = p.Rotation
					}
					to := boatPointPos(p)
					cur := b.Position()
					dx, dy, _ := walkDelta(cur, to.X, to.Y, to.Z)
					if d := math.Hypot(dx, dy); d > 1 {
						b.SetHeading(walkHeading(dx/d, dy/d))
					}
					s.boatStartLeg(b, to)
					if b.monitor == 0 {
						b.monitor = s.loop.Every(boatMonitorPeriod, func() { s.boatMonitor(b) })
					}
					s.boatBroadcast(b, b.departurePacket())
					return true
				}
			}
		} else {
			s.boatStopMonitor(b)
			b.path = nil
		}
	}
	s.boatRunEngine(b, boatEngineDelay)
	return false
}

// boatMonitor is Mobius's stuck monitor: a boat that has not moved for a
// second sails to its point again (or shows itself again past the last
// one).
func (s *Server) boatMonitor(b *Boat) {
	pos := b.Position()
	still := pos.X == b.monitorPos.X && pos.Y == b.monitorPos.Y && pos.Z == b.monitorPos.Z
	if b.dock > 0 || !still {
		b.monitorPos = pos
		return
	}
	if b.path == nil {
		return
	}
	if b.runState < len(b.path) {
		b.runState = max(0, b.runState-1)
		s.boatNextPoint(b)
		return
	}
	s.boatBroadcast(b, b.infoPacket())
}

func (s *Server) boatStopMonitor(b *Boat) {
	if b.monitor != 0 {
		s.loop.Cancel(b.monitor)
		b.monitor = 0
	}
}

// boatRunEngine is Mobius runEngine(delay).
func (s *Server) boatRunEngine(b *Boat, delay time.Duration) {
	if fn := b.engine; fn != nil {
		s.loop.After(delay, fn)
	}
}

// boatStop is Boat.stopMove: VehicleStarted 0 and VehicleInfo.
func (s *Server) boatStop(b *Boat) {
	b.move = nil
	s.boatBroadcast(b, b.startedPacket(0))
	s.boatBroadcast(b, b.infoPacket())
}

// boatTeleport is Mobius Vehicle.teleToLocation: the boat stops, its
// passengers teleport with it (keeping their boat), and it appears at to.
func (s *Server) boatTeleport(b *Boat, to world.Position) {
	if b.move != nil {
		s.boatStop(b)
	}
	b.teleporting = true
	for _, p := range slices.Clone(b.passengers) {
		if err := s.Teleport(p, to); err != nil {
			s.log.Debug("game: cannot teleport a passenger with its boat", append(p.logArgs(), "err", err)...)
		}
	}
	b.teleporting = false
	s.RemoveObject(b)
	if to.Heading == 0 {
		to.Heading = b.Position().Heading
	}
	_ = b.SetPosition(to)
	if err := s.AddObject(b); err != nil {
		s.log.Error("game: a boat cannot appear after a teleport", "boat", b.name, "pos", to, "err", err)
	}
}

// ----- docks and announcements -----

// dockShip is Mobius BoatManager.dockShip: a dock outside the array (Rune)
// is ignored.
func (s *Server) dockShip(dock int, busy bool) {
	if dock >= 0 && dock < boatDockSlots {
		s.vehicles.docks[dock] = busy
	}
}

// dockBusy is Mobius BoatManager.dockBusy.
func (s *Server) dockBusy(dock int) bool {
	return dock >= 0 && dock < boatDockSlots && s.vehicles.docks[dock]
}

// boatHorn is the horn of a boat at a point (Mobius PlaySound(0,
// "itemsound.ship_arrival_departure", 1, boat, x, y, z)).
func boatHorn(b *Boat, at scripting.Location) []byte {
	return serverpackets.NewPlaySoundPacket(serverpackets.PlaySound{
		Sound: boatSound, Bound: 1, ObjectID: b.ObjectID(),
		X: int32(at.X), Y: int32(at.Y), Z: int32(at.Z), //nolint:gosec // world coordinates fit
	})
}

// boatAnnounce is Mobius BoatManager.broadcastPackets: every player within
// the broadcast radius (2D) of either point gets the announcements, then
// the horn.
func (s *Server) boatAnnounce(b *Boat, a *scripting.VehicleAnnounce) {
	if a == nil {
		return
	}
	bodies := make([][]byte, 0, len(a.Messages)+1)
	for _, id := range a.Messages {
		bodies = append(bodies, serverpackets.NewBoatSayPacket(serverpackets.BoatSay{
			NameID: serverpackets.BoatSayNameID, MessageID: int32(id), //nolint:gosec // message IDs are small
		}))
	}
	if a.Horn != nil {
		bodies = append(bodies, boatHorn(b, *a.Horn))
	}
	r := float64(s.vehicles.radius)
	radSq := r * r
	for p := range s.Players() {
		if !p.InGame() {
			continue
		}
		pos := p.Position()
		near := false
		for _, at := range a.Near {
			dx, dy := float64(pos.X)-float64(at.X), float64(pos.Y)-float64(at.Y)
			if dx*dx+dy*dy < radSq {
				near = true
				break
			}
		}
		if near {
			for _, body := range bodies {
				p.Send(body)
			}
		}
	}
}

// ----- passengers -----

// vehicleID returns the object ID of the boat p is on, 0 for none (UserInfo).
func (p *Player) vehicleID() int32 {
	if b := p.vehicle.boat; b != nil {
		return b.ObjectID()
	}
	return 0
}

// charInfoPosition is the position and boat of CharInfo: a passenger with a
// deck position shows there, on its boat (Mobius CharInfo).
func (p *Player) charInfoPosition() (world.Position, int32) {
	pos := p.Position()
	if v := &p.vehicle; v.boat != nil && v.hasPos {
		return world.Position{X: v.pos.X, Y: v.pos.Y, Z: v.pos.Z, Heading: pos.Heading}, v.boat.ObjectID()
	}
	return pos, 0
}

// getOnVehiclePacket is the GetOnVehicle a viewer gets after a passenger's
// CharInfo (Mobius Player.sendInfo), nil for other players.
func (p *Player) getOnVehiclePacket() []byte {
	v := &p.vehicle
	if v.boat == nil || !v.hasPos {
		return nil
	}
	return serverpackets.NewGetOnVehiclePacket(serverpackets.VehiclePassenger{
		ObjectID: p.ObjectID(), BoatID: v.boat.ObjectID(), X: v.pos.X, Y: v.pos.Y, Z: v.pos.Z,
	})
}

// Vehicle returns the boat p is on, or nil. Loop-only.
func (p *Player) Vehicle() *Boat { return p.vehicle.boat }

// setVehicle is Mobius Player.setVehicle: leaving a boat takes the player
// off its passenger list. Leaving also ends the peace boarding gave (Mobius
// keeps that flag after a teleport or an oust; we drop it).
func (s *Server) setVehicle(p *Player, b *Boat) {
	v := &p.vehicle
	if b == nil && v.boat != nil {
		s.boatRemovePassenger(v.boat, p)
		if v.peace {
			v.peace = false
			p.Zones.Leave(ZonePeace)
		}
	}
	v.boat = b
}

// boatAddPassenger is Mobius Vehicle.addPassenger.
func boatAddPassenger(b *Boat, p *Player) bool {
	if slices.Contains(b.passengers, p) || (p.vehicle.boat != nil && p.vehicle.boat != b) {
		return false
	}
	b.passengers = append(b.passengers, p)
	return true
}

func (s *Server) boatRemovePassenger(b *Boat, p *Player) {
	b.passengers = slices.DeleteFunc(b.passengers, func(o *Player) bool { return o == p })
}

// boatPayForRide is Mobius Vehicle.payForRide: every player on board
// within 1,000 gives the tickets, or is put ashore at the oust point with
// the "no ticket" message; those who paid become passengers.
func (s *Server) boatPayForRide(b *Boat, pay *scripting.VehiclePay) {
	if pay == nil || !b.InWorld() {
		return
	}
	var aboard []*Player
	for v := range s.world.VisiblePlayersInRange(b, boatRange) {
		if p, ok := v.(*Player); ok && p.vehicle.boat == b {
			aboard = append(aboard, p)
		}
	}
	for _, p := range aboard {
		if pay.Item > 0 {
			ticket := p.inv.ByItemID(int32(pay.Item)) //nolint:gosec // item IDs fit in int32
			if ticket == nil || !s.DestroyItem(p, ticket, int64(pay.Count), false) {
				p.SendSystemMessage(sysMsgNoTicket)
				s.teleportAround(p, dataLocation(pay.Oust), maxOffsetOnTeleport)
				continue
			}
		}
		boatAddPassenger(b, p)
	}
}

// dataLocation converts a script position.
func dataLocation(l scripting.Location) data.Location {
	return data.Location{X: l.X, Y: l.Y, Z: l.Z}
}

// boatOustLocation is Mobius Vehicle.getOustLoc: the schedules set none, so
// it is the town's teleport point; here the player's restart point, which
// the boat's position decides.
func (s *Server) boatOustLocation(p *Player) (world.Position, bool) {
	loc, ok := s.restartLocation(p)
	if !ok {
		return world.Position{}, false
	}
	return world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z)}, true //nolint:gosec // world coordinates fit
}

// vehicleTeleport is the OnTeleport hook (Mobius Player.teleToLocation): a
// teleport off the boat, not with it, leaves the boat.
func (s *Server) vehicleTeleport(p *Player, _ world.Position) {
	if b := p.vehicle.boat; b != nil && !b.teleporting {
		s.setVehicle(p, nil)
		p.vehicle.hasPos = false
	}
}

// vehicleLeave is the OnPlayerLeave hook (Mobius Player.deleteMe, then
// Boat.oustPlayer): a passenger who leaves the game is saved ashore.
func (s *Server) vehicleLeave(p *Player) {
	if p.vehicle.boat == nil {
		return
	}
	s.setVehicle(p, nil)
	p.vehicle.hasPos = false
	if to, ok := s.boatOustLocation(p); ok && p.InWorld() {
		to.Heading = p.Position().Heading
		if err := s.MoveObject(p, to); err != nil {
			s.log.Debug("game: cannot put a leaving passenger ashore", append(p.logArgs(), "err", err)...)
		}
	}
}
