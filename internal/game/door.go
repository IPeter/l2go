package game

import (
	"errors"
	"math"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/world"
)

// The door feature (Plan 13, Phase 10.3): the doors, static objects and
// fences of Options.Objects as world objects (this file, staticobject.go,
// fence.go), their packets (A12), the open and close rules with their loop
// timers, the door and fence lookups of the composed geodata view (A2), the
// GM commands (door_gm.go) and the scripts' DoorControl (door_script.go).
//
// Doors, static objects and fences are game objects of their own types:
// they never move, attack or cast (Mobius Door and StaticObject override
// moveToLocation, doAttack and doCast with nothing, and DoorAI thinks
// nothing), so no other feature drives them.
func init() { registerFeature("doors", setupDoors) }

// doorDamageSteps is the number of damage pictures DoorStatusUpdate shows
// (Mobius Door.getDamage: 0 to 6).
const doorDamageSteps = 6

// doorFeature is the door feature's state on the Server (Server.doors).
// Loop-only, except data.
type doorFeature struct {
	// data is Options.Objects, nil without world objects.
	data *data.Objects

	// byID holds the doors by door ID; list holds them in ID order.
	byID map[int]*Door
	list []*Door
	// near holds, per region, the doors spawned in its 3×3×3
	// neighbourhood (Mobius WorldRegion.addDoor on every surrounding
	// region): a segment's doors are looked up from its start's region.
	near map[world.RegionKey][]*Door

	fences    []*Fence
	fenceNear map[world.RegionKey][]*Fence // as near
	statics   []*StaticObject

	// shown holds the door IDs //showdoors drew for each GM.
	shown map[*Player]map[int]bool
	// approaches holds the players walking up to a static object to use it.
	approaches map[*Player]*staticApproach
}

// Door is a door in the world (Mobius Door). It lives on the world loop.
type Door struct {
	world.Base

	// Template is the door's data.
	Template *data.Door

	srv        *Server
	open       bool
	curHP      float64
	maxHP      int32
	attackable bool // Mobius _isAttackableDoor
	targetable bool
	shape      geo.Door

	// autoClose closes an opened door after its close time (Mobius
	// _autoCloseTask); timer is the open/close cycle of a door that opens by
	// time (Mobius TimerOpen). 0: none.
	autoClose world.TimerID
	timer     world.TimerID
}

var (
	_ InfoProvider  = (*Door)(nil)
	_ objectClicker = (*Door)(nil)
	_ GeoAlwaysSeen = (*Door)(nil)
)

// objectClicker is a world object with a click handler of its own (Mobius
// IActionClickHandler), run by a click of p on it (target.go onAction).
// interact is false for a Shift-click of a non-GM.
type objectClicker interface {
	click(p *Player, interact bool)
}

// setupDoors spawns the doors, static objects and fences of the world
// object data, in Mobius's load order (DoorData, StaticObjectData,
// FenceData), and registers the GM commands and the hooks.
func setupDoors(s *Server) error {
	f := &s.doors
	f.byID = make(map[int]*Door)
	f.near = make(map[world.RegionKey][]*Door)
	f.fenceNear = make(map[world.RegionKey][]*Fence)
	f.shown = make(map[*Player]map[int]bool)
	f.approaches = make(map[*Player]*staticApproach)
	if err := s.setupDoorGM(); err != nil {
		return err
	}
	s.OnPlayerLeave(func(p *Player) {
		delete(f.shown, p)
		delete(f.approaches, p)
	})
	s.OnTick(s.checkStaticApproaches)
	s.geo.SetBlockers(geoBlockers{s})
	if f.data == nil {
		return nil
	}
	var errs []error
	for i := range f.data.Doors {
		if err := s.spawnDoor(&f.data.Doors[i]); err != nil {
			errs = append(errs, err)
		}
	}
	for i := range f.data.StaticObjects {
		if err := s.spawnStaticObject(&f.data.StaticObjects[i]); err != nil {
			errs = append(errs, err)
		}
	}
	for i := range f.data.Fences {
		if err := s.spawnFence(&f.data.Fences[i]); err != nil {
			errs = append(errs, err)
		}
	}
	s.log.Info("game: world objects spawned", "doors", len(f.list), "staticObjects", len(f.statics), "fences", len(f.fences))
	return errors.Join(errs...)
}

// spawnDoor makes the door of t and puts it into the world (Mobius
// DoorData.makeDoor): open by default or closed, at full HP, at the lower of
// its pos Z and node Z. A door that opens by time starts its cycle first
// (the Door constructor).
func (s *Server) spawnDoor(t *data.Door) error {
	id, err := s.world.IDs().Next()
	if err != nil {
		return err
	}
	d := &Door{
		Base:       world.NewBase(id, world.Position{X: int32(t.X), Y: int32(t.Y), Z: int32(t.SpawnZ())}), //nolint:gosec // world coordinates fit
		Template:   t,
		srv:        s,
		open:       t.DefaultOpen,
		maxHP:      truncHP(t.HP),
		attackable: t.Attackable,
		targetable: t.Targetable,
		shape:      doorShape(t),
	}
	d.curHP = float64(d.maxHP)
	if t.OpensBy(data.DoorOpenByTime) {
		s.startDoorTimer(d)
	}
	if err := s.AddObject(d); err != nil {
		s.loop.Cancel(d.timer)
		_ = s.world.IDs().Release(id)
		return err
	}
	f := &s.doors
	f.byID[t.ID] = d
	f.list = append(f.list, d)
	if k, ok := world.RegionOf(d.Position().X, d.Position().Y, d.Position().Z); ok {
		for _, nk := range regionNeighbourhood(k) {
			f.near[nk] = append(f.near[nk], d)
		}
	}
	return nil
}

// doorShape is the collision shape of a door for geo.Door.Between.
func doorShape(t *data.Door) geo.Door {
	var g geo.Door
	for i, n := range t.Nodes {
		g.X[i], g.Y[i] = int32(n.X), int32(n.Y) //nolint:gosec // world coordinates fit
	}
	g.ZMin, g.ZMax = int32(t.ZMin()), int32(t.ZMax()) //nolint:gosec // world heights fit
	return g
}

// truncHP is Mobius's (int) of a max HP.
func truncHP(hp float64) int32 {
	return int32(min(max(hp, 0), math.MaxInt32))
}

// regionNeighbourhood returns k and the keys of its 3×3×3 neighbourhood.
// Keys outside the grid are harmless: no position maps to them.
func regionNeighbourhood(k world.RegionKey) []world.RegionKey {
	keys := make([]world.RegionKey, 0, 27)
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				keys = append(keys, world.RegionKey{X: k.X + dx, Y: k.Y + dy, Z: k.Z + dz})
			}
		}
	}
	return keys
}

// ----- lookups -----

// Door returns the door with the door ID (the data's, not the object ID).
// Loop-only.
func (s *Server) Door(doorID int) (*Door, bool) {
	d, ok := s.doors.byID[doorID]
	return d, ok
}

// Doors returns the doors in door ID order. The slice must not be modified.
// Loop-only.
func (s *Server) Doors() []*Door { return s.doors.list }

// DoorsBetween reports whether a closed door stands between from and to
// (Mobius DoorData.checkIfDoorsBetween): of the doors indexed in the
// region of from, one in the instance, closed, with check_collision, whose
// shape the segment crosses (geo.Door.Between; for sight both faces). It
// is the door part of the composed geodata view (A2). Loop-only.
func (s *Server) DoorsBetween(from, to geo.Point, instance int32, sight bool) bool {
	k, ok := world.RegionOf(from.X, from.Y, from.Z)
	if !ok {
		return false
	}
	for _, d := range s.doors.near[k] {
		if d.InstanceID() != instance || d.open || !d.Template.CheckCollision {
			continue
		}
		if d.shape.Between(from, to, sight) {
			return true
		}
	}
	return false
}

// geoBlockers are the doors and fences as the composed geodata view asks
// for them (GeoBlockers, geo.go), in the main world: the view's calls carry
// no instance, and the world has no instances yet.
type geoBlockers struct{ s *Server }

var _ GeoBlockers = geoBlockers{}

func (b geoBlockers) DoorsBetween(from, to geo.Point, sight bool) bool {
	return b.s.DoorsBetween(from, to, 0, sight)
}

func (b geoBlockers) FencesBetween(from, to geo.Point) bool {
	return b.s.FencesBetween(from, to, 0)
}

// GeoAlwaysSeen marks a door as a target every line of sight reaches
// (Mobius canSeeTarget(WorldObject, WorldObject): "door as target").
func (*Door) GeoAlwaysSeen() {}

// ----- state -----

// Name returns the door's name.
func (d *Door) Name() string { return d.Template.Name }

// Open reports whether the door is open.
func (d *Door) Open() bool { return d.open }

// Targetable reports whether players other than GMs may click the door.
func (d *Door) Targetable() bool { return d.targetable }

// Level returns the door's level.
func (d *Door) Level() int { return d.Template.Level }

// MaxHP returns the door's maximum HP.
func (d *Door) MaxHP() int32 { return d.maxHP }

// damage is Mobius Door.getDamage: 6 - ceil(HP / max HP × 6), within 0 to 6.
func (d *Door) damage() int32 {
	if d.maxHP <= 0 {
		return 0
	}
	dmg := doorDamageSteps - int32(math.Ceil(d.curHP/float64(d.maxHP)*doorDamageSteps))
	return min(max(dmg, 0), doorDamageSteps)
}

// autoAttackableBy is Mobius Door.isAutoAttackable for a player: an
// attackable door; the siege doors come with the sieges.
func (d *Door) autoAttackableBy(*Player) bool { return d.attackable }

// InfoFor returns what a player that starts to see the door gets (Mobius
// Door.sendInfo): StaticObjectInfo and DoorStatusUpdate (InfoProvider).
func (d *Door) InfoFor(*Player) [][]byte {
	return [][]byte{d.staticInfo(), d.statusUpdate()}
}

func (d *Door) staticInfo() []byte {
	return serverpackets.NewStaticObjectInfoPacket(serverpackets.StaticObjectInfo{
		StaticID: int32(d.Template.ID), //nolint:gosec // door IDs fit
		ObjectID: int32(d.ID()),
	})
}

func (d *Door) statusUpdate() []byte {
	return serverpackets.NewDoorStatusUpdatePacket(serverpackets.DoorStatusUpdate{
		ObjectID: int32(d.ID()),
		Closed:   !d.open,
		Damage:   d.damage(),
		DoorID:   int32(d.Template.ID), //nolint:gosec // door IDs fit
		MaxHP:    d.maxHP,
		CurHP:    int32(d.curHP),
	})
}

// broadcastDoor sends the door's StaticObjectInfo and DoorStatusUpdate to
// every player that sees it (Mobius Door.broadcastStatusUpdate).
func (s *Server) broadcastDoor(d *Door) {
	var info, status []byte
	for v := range s.world.VisiblePlayers(d) {
		if !s.world.CanSee(v, d) {
			continue
		}
		if info == nil {
			info, status = d.staticInfo(), d.statusUpdate()
		}
		world.Send(v, info)
		world.Send(v, status)
	}
}

// ----- opening and closing -----

// openDoor is Mobius Door.openMe: a door of a group opens the group; a
// closed door opens, shows it and starts its auto-close.
func (s *Server) openDoor(d *Door) {
	if g := d.Template.Group; g != "" {
		s.doorGroupOpen(true, g)
		return
	}
	if !d.open {
		s.setDoorOpen(d, true)
		s.broadcastDoor(d)
		s.startDoorAutoClose(d)
	}
}

// closeDoor is Mobius Door.closeMe: the auto-close stops; a door of a group
// closes the group; an open door closes and shows it.
func (s *Server) closeDoor(d *Door) {
	if d.autoClose != 0 {
		s.loop.Cancel(d.autoClose)
		d.autoClose = 0
	}
	if g := d.Template.Group; g != "" {
		s.doorGroupOpen(false, g)
		return
	}
	if d.open {
		s.setDoorOpen(d, false)
		s.broadcastDoor(d)
	}
}

// doorGroupOpen is Mobius manageGroupOpen: every door of the group that is
// not in the state changes and shows it; opening starts the auto-close of
// the group's first door only.
func (s *Server) doorGroupOpen(open bool, group string) {
	var first *Door
	for _, id := range s.doors.data.DoorGroup(group) {
		d := s.doors.byID[id]
		if d == nil {
			continue
		}
		if first == nil {
			first = d
		}
		if d.open != open {
			s.setDoorOpen(d, open)
			s.broadcastDoor(d)
		}
	}
	if first != nil && open {
		s.startDoorAutoClose(first)
	}
}

// setDoorOpen is Mobius Door.setOpen: the state changes, then the child
// door follows by its master actions (before the caller shows the change,
// as in Mobius).
func (s *Server) setDoorOpen(d *Door, open bool) {
	d.open = open
	child := d.Template.ChildID
	if child <= 0 {
		return
	}
	c := s.doors.byID[child]
	if c == nil {
		s.log.Warn("game: cannot find the child door", "door", d.Template.ID, "child", child)
		return
	}
	act := c.Template.MasterClose
	if open {
		act = c.Template.MasterOpen
	}
	switch act {
	case data.DoorActOpen:
		s.openDoor(c)
	case data.DoorActClose:
		s.closeDoor(c)
	}
}

// startDoorAutoClose is Mobius startAutoCloseTask: a door with a close time
// that does not open by time and belongs to no clan hall closes itself
// after it; a pending auto-close starts over.
func (s *Server) startDoorAutoClose(d *Door) {
	if !d.Template.AutoCloses() {
		return
	}
	if d.autoClose != 0 {
		s.loop.Cancel(d.autoClose)
	}
	d.autoClose = s.loop.After(time.Duration(d.Template.CloseTime)*time.Second, func() {
		d.autoClose = 0
		if d.open {
			s.closeDoor(d)
		}
	})
}

// startDoorTimer starts the cycle of a door that opens by time (Mobius
// startTimerOpen): the first change comes after the open time when the door
// is open, after the close time when it is closed, plus Rnd(random time).
// (Mobius's first delay is the other way round from the later ones.)
func (s *Server) startDoorTimer(d *Door) {
	t := d.Template
	delay := t.CloseTime
	if d.open {
		delay = t.OpenTime
	}
	s.scheduleDoorTimer(d, delay)
}

// scheduleDoorTimer runs the next change of a timed door after delay
// seconds plus Rnd(random time).
func (s *Server) scheduleDoorTimer(d *Door, delay int) {
	if r := d.Template.RandomTime; r > 0 {
		delay += s.rnd.Get(r)
	}
	d.timer = s.loop.After(time.Duration(delay)*time.Second, func() { s.doorTimer(d) })
}

// doorTimer is Mobius TimerOpen.run: an open door closes, a closed one
// opens; then an open door waits its close time, a closed one its open
// time, each plus Rnd(random time).
func (s *Server) doorTimer(d *Door) {
	d.timer = 0
	if d.open {
		s.closeDoor(d)
	} else {
		s.openDoor(d)
	}
	delay := d.Template.OpenTime
	if d.open {
		delay = d.Template.CloseTime
	}
	s.scheduleDoorTimer(d, delay)
}

// OpenDoor opens the door with the door ID (Mobius openMe) and reports
// whether the door exists. Loop-only.
func (s *Server) OpenDoor(doorID int) bool {
	d, ok := s.doors.byID[doorID]
	if ok {
		s.openDoor(d)
	}
	return ok
}

// CloseDoor closes the door with the door ID (Mobius closeMe) and reports
// whether the door exists. Loop-only.
func (s *Server) CloseDoor(doorID int) bool {
	d, ok := s.doors.byID[doorID]
	if ok {
		s.closeDoor(d)
	}
	return ok
}

// ----- clicks -----

// click is Mobius DoorClick: the first click targets the door. A click on
// the target attacks an attackable door, and lets a clan hall's owners open
// and close it; neither exists yet (attacks on doors come with the sieges,
// clan halls with M11), so the second click does nothing.
func (d *Door) click(p *Player, _ bool) {
	if p.Target != world.Object(d) {
		d.srv.SetTarget(p, d)
	}
}

// hpBar returns the numbers of the door's HP bar as a target (target.go
// targetHPBar).
func (d *Door) hpBar() (maxHP int32, cur float64) { return d.maxHP, d.curHP }
