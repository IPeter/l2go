package testclient

import (
	"cmp"
	"context"
	"errors"
	"math"
	"slices"
	"time"
)

// The world AI part of the client (Plan 13, Phase 10.9): the cinematic
// packets of the boss scripts, the doors, static objects and boats of the
// world model, the game clock, and the BoardShip and WaitForNight helpers.
// The door and vehicle packets themselves are in door.go and vehicle.go.

// Game server opcodes of the cinematic packets (Mobius ServerPackets).
const (
	OpEarthquake    byte = 0xc4
	OpSpecialCamera byte = 0xc7
)

// ExOpRedSky is the Ex sub-opcode of ExRedSky.
const ExOpRedSky uint16 = 0x40

// The decoder entries are added here rather than in the shared tables, so
// the world AI packets stay in one file.
func init() {
	serverPackets[OpEarthquake] = serverPacket{"Earthquake", readEarthquake}
	serverPackets[OpSpecialCamera] = serverPacket{"SpecialCamera", readSpecialCamera}
	exServerPackets[ExOpRedSky] = serverPacket{"ExRedSky", readExRedSky}
}

// SpecialCamera moves the camera around a creature (0xc7), for the boss
// cinematics.
type SpecialCamera struct {
	ObjectID                int32
	Force, Angle1, Angle2   int32
	Time, Duration          int32
	RelYaw, RelPitch        int32
	Wide, RelAngle, Unknown int32
}

// PacketName implements Packet.
func (SpecialCamera) PacketName() string { return "SpecialCamera" }

func readSpecialCamera(f *fields) Packet {
	var p SpecialCamera
	p.ObjectID = f.i32()
	p.Force = f.i32()
	p.Angle1 = f.i32()
	p.Angle2 = f.i32()
	p.Time = f.i32()
	p.Duration = f.i32()
	p.RelYaw = f.i32()
	p.RelPitch = f.i32()
	p.Wide = f.i32()
	p.RelAngle = f.i32()
	p.Unknown = f.i32()
	return p
}

// Earthquake shakes the screens around a point (0xc4). Mobius writes a last
// int that is always 0.
type Earthquake struct {
	X, Y, Z             int32
	Intensity, Duration int32
}

// PacketName implements Packet.
func (Earthquake) PacketName() string { return "Earthquake" }

// Position returns the point the earthquake shakes.
func (e Earthquake) Position() Position { return Position{e.X, e.Y, e.Z} }

func readEarthquake(f *fields) Packet {
	var p Earthquake
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Intensity = f.i32()
	p.Duration = f.i32()
	if last := f.i32(); f.ok() && last != 0 {
		f.failf("last int %d, want 0", last)
	}
	return p
}

// ExRedSky turns the sky red for Duration seconds (FE:40).
type ExRedSky struct {
	Duration int32
}

// PacketName implements Packet.
func (ExRedSky) PacketName() string { return "ExRedSky" }

func readExRedSky(f *fields) Packet { return ExRedSky{Duration: f.i32()} }

// ----- the world model: doors, static objects, boats -----

// StaticObject is an object StaticObjectInfo showed: a door (see Door) or a
// static object such as a town map or a signboard.
type StaticObject struct {
	ObjectID int32
	// StaticID is the door ID or the static object ID of the data.
	StaticID int32
}

// Door is a door the client sees: its StaticObjectInfo and the last
// DoorStatusUpdate.
type Door struct {
	ObjectID int32
	DoorID   int32
	Closed   bool
	// Damage is 0 (whole) to 6.
	Damage       int32
	Enemy        bool
	MaxHP, CurHP int32
}

// Boat is a boat the client sees (VehicleInfo). Moving is set by
// VehicleDeparture and VehicleStarted 1, with Dest where the leg ends, and
// cleared by VehicleStarted 0 and VehicleInfo (which a moving boat follows
// with its VehicleDeparture). The server does not report the end of a leg,
// so Moving stays set when the boat reached Dest.
type Boat struct {
	ObjectID int32
	Pos      Position
	Heading  int32
	Moving   bool
	Dest     Position
	// MoveSpeed and RotationSpeed are those of the last VehicleDeparture.
	MoveSpeed, RotationSpeed int32
	// Passengers are the players seen boarding it (GetOnVehicle), the
	// character included, sorted by object ID.
	Passengers []int32
}

// worldAI is the part of World this file keeps; World holds it in its ai
// field, under its mutex.
type worldAI struct {
	statics map[int32]StaticObject
	doors   map[int32]Door
	boats   map[int32]Boat
	// boat is the boat the character is on (0: none) and deck its position
	// on the deck, from the character's GetOnVehicle and GetOffVehicle.
	boat int32
	deck Position
	// clock is the game clock CharSelected started.
	clock    GameClock
	hasClock bool
}

// selected starts the game clock from CharSelected's game time.
func (a *worldAI) selected(gameTime int32, now time.Time) {
	a.clock, a.hasClock = GameClock{Minutes: gameTime, At: now}, true
}

// forget drops an object DeleteObject removed.
func (a *worldAI) forget(id int32) {
	delete(a.statics, id)
	delete(a.doors, id)
	delete(a.boats, id)
	if a.boat == id {
		a.boat, a.deck = 0, Position{}
	}
}

// applyWorldAI handles the door, static object and vehicle packets; the
// caller holds w.mu.
func (w *World) applyWorldAI(p Packet) {
	a := &w.ai
	switch v := p.(type) {
	case StaticObjectInfo:
		if a.statics == nil {
			a.statics = make(map[int32]StaticObject)
		}
		a.statics[v.ObjectID] = StaticObject{ObjectID: v.ObjectID, StaticID: v.StaticID}
	case DoorStatusUpdate:
		if a.doors == nil {
			a.doors = make(map[int32]Door)
		}
		a.doors[v.ObjectID] = Door{ObjectID: v.ObjectID, DoorID: v.DoorID, Closed: v.Closed, Damage: v.Damage,
			Enemy: v.Enemy, MaxHP: v.MaxHP, CurHP: v.CurHP}
	case VehicleInfo:
		b := a.boatOf(v.ObjectID)
		b.Pos, b.Heading, b.Moving = v.Position(), v.Heading, false
		a.boats[v.ObjectID] = b
	case VehicleCheckLocation:
		b := a.boatOf(v.ObjectID)
		b.Pos, b.Heading = v.Position(), v.Heading
		a.boats[v.ObjectID] = b
	case VehicleDeparture:
		b := a.boatOf(v.ObjectID)
		b.Moving, b.Dest = true, v.Destination()
		b.MoveSpeed, b.RotationSpeed = v.MoveSpeed, v.RotationSpeed
		a.boats[v.ObjectID] = b
	case VehicleStarted:
		b := a.boatOf(v.ObjectID)
		b.Moving = v.State == 1
		a.boats[v.ObjectID] = b
	case GetOnVehicle:
		a.board(v.ObjectID, v.BoatID)
		if v.ObjectID != 0 && v.ObjectID == w.me.ObjectID {
			a.boat, a.deck = v.BoatID, v.Position()
		}
	case GetOffVehicle:
		a.leave(v.ObjectID, v.BoatID)
		if v.ObjectID != 0 && v.ObjectID == w.me.ObjectID {
			a.boat, a.deck = 0, Position{}
			w.me.Pos = v.Position()
		}
	case MoveToLocationInVehicle:
		if v.ObjectID != 0 && v.ObjectID == w.me.ObjectID && v.BoatID == a.boat {
			a.deck = Position{v.DestX, v.DestY, v.DestZ}
		}
	case StopMoveInVehicle:
		if v.ObjectID != 0 && v.ObjectID == w.me.ObjectID && v.BoatID == a.boat {
			a.deck = Position{v.X, v.Y, v.Z}
		}
	}
}

// boatOf returns the boat id, a new one when the client does not know it.
func (a *worldAI) boatOf(id int32) Boat {
	if a.boats == nil {
		a.boats = make(map[int32]Boat)
	}
	b, ok := a.boats[id]
	if !ok {
		b.ObjectID = id
	}
	return b
}

// board adds the player to the boat's passengers.
func (a *worldAI) board(player, boat int32) {
	for id, b := range a.boats {
		if id != boat && slices.Contains(b.Passengers, player) {
			a.leave(player, id)
		}
	}
	b := a.boatOf(boat)
	if i, found := slices.BinarySearch(b.Passengers, player); !found {
		b.Passengers = slices.Insert(slices.Clone(b.Passengers), i, player)
	}
	a.boats[boat] = b
}

// leave removes the player from the boat's passengers.
func (a *worldAI) leave(player, boat int32) {
	b, ok := a.boats[boat]
	if !ok {
		return
	}
	b.Passengers = slices.DeleteFunc(slices.Clone(b.Passengers), func(id int32) bool { return id == player })
	a.boats[boat] = b
}

// StaticObjects returns the objects StaticObjectInfo showed (doors among
// them), sorted by object ID.
func (w *World) StaticObjects() []StaticObject {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]StaticObject, 0, len(w.ai.statics))
	for _, s := range w.ai.statics {
		out = append(out, s)
	}
	slices.SortFunc(out, func(a, b StaticObject) int { return cmp.Compare(a.ObjectID, b.ObjectID) })
	return out
}

// Doors returns the doors the client sees (a DoorStatusUpdate came), sorted
// by object ID.
func (w *World) Doors() []Door {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]Door, 0, len(w.ai.doors))
	for _, d := range w.ai.doors {
		out = append(out, d)
	}
	slices.SortFunc(out, func(a, b Door) int { return cmp.Compare(a.ObjectID, b.ObjectID) })
	return out
}

// FindDoor returns the door with the door ID doorID (the data's ID, such as
// 17220001).
func (w *World) FindDoor(doorID int32) (Door, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var (
		best  Door
		found bool
	)
	for _, d := range w.ai.doors {
		if d.DoorID == doorID && (!found || d.ObjectID < best.ObjectID) {
			best, found = d, true
		}
	}
	return best, found
}

// Boats returns the boats the client sees, sorted by object ID.
func (w *World) Boats() []Boat {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]Boat, 0, len(w.ai.boats))
	for _, b := range w.ai.boats {
		b.Passengers = slices.Clone(b.Passengers)
		out = append(out, b)
	}
	slices.SortFunc(out, func(a, b Boat) int { return cmp.Compare(a.ObjectID, b.ObjectID) })
	return out
}

// Boat returns the boat with the object ID id.
func (w *World) Boat(id int32) (Boat, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	b, ok := w.ai.boats[id]
	b.Passengers = slices.Clone(b.Passengers)
	return b, ok
}

// NearestBoat returns the boat nearest to the character (the lower object
// ID on a tie).
func (w *World) NearestBoat() (Boat, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var (
		best     Boat
		bestDist = math.Inf(1)
		found    bool
	)
	for _, b := range w.ai.boats {
		d := b.Pos.DistanceTo(w.me.Pos)
		if !found || d < bestDist || (d == bestDist && b.ObjectID < best.ObjectID) {
			best, bestDist, found = b, d, true
		}
	}
	best.Passengers = slices.Clone(best.Passengers)
	return best, found
}

// OnBoat returns the boat the character is on (its GetOnVehicle, until its
// GetOffVehicle or the boat's DeleteObject) and its position on the deck,
// or 0 when it is on none.
func (w *World) OnBoat() (boatID int32, deck Position) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ai.boat, w.ai.deck
}

// ----- the game clock -----

// Game time (Mobius GameTimeTaskManager): 6 game days per real day, night
// before 06:00.
const (
	// RealGameMinute is how long a game minute lasts in real time.
	RealGameMinute = 10 * time.Second
	// GameMinutesPerDay is the length of a game day in game minutes.
	GameMinutesPerDay = 24 * 60
	// NightEndMinute is when the night ends: game 06:00.
	NightEndMinute = 6 * 60
)

// GameClock is the game time as the client knows it. The server tells the
// time of day once, in CharSelected (minutes since game midnight; Mobius
// never sends ClientSetTime); from then on the client's clock runs at 10
// real seconds per game minute.
type GameClock struct {
	// Minutes is the game time CharSelected gave, At when it came by the
	// client's clock.
	Minutes int32
	At      time.Time
	// Passed is game time the client let pass on top of the real time
	// (WaitForNight with a pass function): a test server's fake clock jumps
	// ahead while the client's clock does not.
	Passed time.Duration
}

// MinutesAt returns the minutes into the game day at now (0-1439).
func (g GameClock) MinutesAt(now time.Time) int32 {
	elapsed := max(now.Sub(g.At), 0) + g.Passed
	m := (int64(g.Minutes) + int64(elapsed/RealGameMinute)) % GameMinutesPerDay
	if m < 0 {
		m += GameMinutesPerDay
	}
	return int32(m)
}

// NightAt reports whether it is night in the game at now (before 06:00).
func (g GameClock) NightAt(now time.Time) bool { return g.MinutesAt(now) < NightEndMinute }

// UntilNight returns the real time from the game time minutes (minutes since
// game midnight) to the next game midnight, or 0 when it is night already.
func UntilNight(minutes int32) time.Duration {
	m := ((minutes % GameMinutesPerDay) + GameMinutesPerDay) % GameMinutesPerDay
	if m < NightEndMinute {
		return 0
	}
	return time.Duration(GameMinutesPerDay-m) * RealGameMinute
}

// GameClock returns the game clock, and false before CharSelected came.
func (w *World) GameClock() (GameClock, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ai.clock, w.ai.hasClock
}

// passGameTime adds d to the time the clock counts as passed.
func (w *World) passGameTime(d time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ai.clock.Passed += d
}

// nightMargin is how far into the night WaitForNight waits: one game minute,
// so that the server's tick has seen midnight and switched to night.
const nightMargin = RealGameMinute

// ErrNoGameClock means WaitForNight ran before CharSelected told the game
// time.
var ErrNoGameClock = errors.New("testclient: no game time yet (CharSelected)")

// WaitForNight lets time pass until it is night in the game (00:00 to
// 06:00) by the client's game clock, one game minute into it, and returns
// how much real time it let pass (0 when it is night already). pass lets d
// pass: nil sleeps (a live server), while a test of an in-process server
// moves its fake clock ahead; the client's game clock then counts d as
// passed.
func (c *GameConn) WaitForNight(ctx context.Context, pass func(ctx context.Context, d time.Duration) error) (time.Duration, error) {
	clock, ok := c.world.GameClock()
	if !ok {
		return 0, ErrNoGameClock
	}
	d := UntilNight(clock.MinutesAt(time.Now()))
	if d == 0 {
		return 0, nil
	}
	d += nightMargin
	if pass == nil {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-t.C:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
		return d, nil
	}
	if err := pass(ctx, d); err != nil {
		return 0, err
	}
	c.world.passGameTime(d)
	return d, nil
}

// ----- boarding -----

// ErrBoardRefused means the server refused to let the character board
// (ActionFailed): the boat is under way, more than 1,000 units away, or
// unknown.
var ErrBoardRefused = errors.New("testclient: the server refused the boarding")

// ErrNoBoat means BoardShip found no boat in sight.
var ErrNoBoat = errors.New("testclient: no boat in sight")

// BoardShip boards the boat boatID (0: the nearest boat the client sees),
// standing at deck, a position on its deck (RequestGetOnVehicle), and
// returns the character's GetOnVehicle. The server lets a character board a
// boat that stands within 1,000 units; a refusal (ActionFailed) gives
// ErrBoardRefused. The packets queued before the request are drained first.
func (c *GameConn) BoardShip(ctx context.Context, boatID int32, deck Position) (GetOnVehicle, error) {
	if boatID == 0 {
		b, ok := c.world.NearestBoat()
		if !ok {
			return GetOnVehicle{}, ErrNoBoat
		}
		boatID = b.ObjectID
	}
	if _, err := c.Drain(); err != nil {
		return GetOnVehicle{}, err
	}
	if err := c.GetOnVehicle(ctx, boatID, deck); err != nil {
		return GetOnVehicle{}, err
	}
	me := c.world.Me().ObjectID
	p, err := c.WaitFor(ctx, func(p Packet) bool {
		switch v := p.(type) {
		case GetOnVehicle:
			return v.ObjectID == me
		case ActionFailed:
			return true
		}
		return false
	})
	if err != nil {
		return GetOnVehicle{}, err
	}
	g, ok := p.(GetOnVehicle)
	if !ok {
		return GetOnVehicle{}, ErrBoardRefused
	}
	return g, nil
}
