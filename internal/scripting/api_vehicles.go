package scripting

import (
	"fmt"
	"math"
	"slices"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// The vehicles module (Plan 13, A14): the ferry schedules are content, so
// scripts declare them as data (scripts/vehicles/, with lib/boat.lua) and
// the game runs them (the Mobius Boat engine and the BoatManager docks).
// A schedule is the cycle of a Mobius vehicles/Boat*.java script: one step
// per case of its run(), in order, starting over after the last.

// VehiclePoint is a point of a boat's path (Mobius VehiclePathPoint): the
// boat sails there at Speed with the turn speed Rotation. A Speed of 0
// teleports the boat there, facing Rotation.
type VehiclePoint struct {
	X, Y, Z         int
	Speed, Rotation int
}

// Mobius VehiclePathPoint(x, y, z) speeds, for points that give none.
const (
	defaultVehicleSpeed    = 350
	defaultVehicleRotation = 4000
)

// VehicleAnnounce is what the players near a ferry route hear (Mobius
// BoatManager.broadcastPackets): everyone within the broadcast radius of
// either point of Near gets the boat announcements (CreatureSay BOAT with
// the system message IDs, in order), then the horn at Horn when set.
type VehicleAnnounce struct {
	Near     [2]Location
	Messages []int
	Horn     *Location
}

// VehiclePay is Mobius Boat.payForRide: every player on board within 1,000
// of the boat gives Count tickets of Item (none for Item 0); without them
// the player is put ashore at Oust.
type VehiclePay struct {
	Item, Count int
	Oust        Location
}

// VehicleStep is one case of a schedule, done in this order:
//
//   - Await: while that dock is busy, the step waits 5 s and tries again;
//     Busy is announced on the first try and every 36th after.
//   - Undock frees a dock; Dock takes one (Mobius dockShip).
//   - Announce goes to the players near the route; Horn plays the horn
//     around the boat, at that point.
//   - Pay takes the tickets; Path sends the boat on its way.
//   - Wait runs the next step after it. With no Wait the next step runs when
//     the boat reaches the end of a path.
type VehicleStep struct {
	Await    int
	Busy     *VehicleAnnounce
	Undock   int
	Dock     int
	Announce *VehicleAnnounce
	Horn     *Location
	Pay      *VehiclePay
	Path     []VehiclePoint
	Wait     time.Duration
}

// VehicleRoute is a ferry schedule (vehicles.add_route).
type VehicleRoute struct {
	// Name names the route in logs; File is the script that declared it.
	Name, File string
	// BoatID is the Mobius boat ID (getNewBoat's first argument).
	BoatID int
	// Start is where the boat appears, facing Start.Heading.
	Start Location
	// StartDelay is when the first step runs (Mobius runEngine(180000)).
	StartDelay time.Duration
	// Dock is the dock the boat holds at startup, 0 for none.
	Dock  int
	Steps []VehicleStep
}

// VehicleRoutes returns the ferry schedules the loaded scripts declared, in
// load order. The game starts its boats from them once, at startup.
func (e *Engine) VehicleRoutes() []VehicleRoute {
	if e.closed {
		return nil
	}
	return slices.Clone(e.vm.vehicles)
}

// vehiclesFuncs is the vehicles module.
func (v *vm) vehiclesFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{"add_route": v.addVehicleRoute}
}

// addVehicleRoute is vehicles.add_route(route): it checks the schedule and
// keeps it for the game. Only while the scripts load.
func (v *vm) addVehicleRoute(L *lua.LState) int {
	const fn = "vehicles.add_route"
	if !v.loading {
		L.RaiseError("%s: routes can only be added while the scripts load", fn)
		return 0
	}
	p := vehicleParser{L: L, fn: fn}
	t := L.CheckTable(1)
	p.keys(t, "route", "name", "boat", "start", "start_delay", "dock", "steps")
	r := VehicleRoute{
		File:       v.currentFile(),
		Name:       p.str(t, "name"),
		BoatID:     p.posInt(t, "boat", "route"),
		StartDelay: p.ms(t, "start_delay", "route"),
		Dock:       p.optInt(t, "dock", "route"),
	}
	r.Start = p.location(t.RawGetString("start"), "start", true)
	steps, ok := t.RawGetString("steps").(*lua.LTable)
	if !ok || steps.Len() == 0 {
		p.fail("route %q: steps must be a non-empty list", r.Name)
	}
	for i := 1; i <= steps.Len(); i++ {
		r.Steps = append(r.Steps, p.step(steps.RawGetInt(i), fmt.Sprintf("route %q step %d", r.Name, i)))
	}
	for _, o := range v.vehicles {
		if o.Name == r.Name {
			p.fail("route %q is already declared (%s)", r.Name, o.File)
		}
		if o.BoatID == r.BoatID {
			p.fail("route %q: boat %d already sails route %q", r.Name, r.BoatID, o.Name)
		}
	}
	v.vehicles = append(v.vehicles, r)
	return 0
}

// vehicleParser reads a route table, raising Lua errors that name the
// faulty field.
type vehicleParser struct {
	L  *lua.LState
	fn string
}

func (p vehicleParser) fail(format string, args ...any) {
	p.L.RaiseError("%s: %s", p.fn, fmt.Sprintf(format, args...))
}

// keys refuses fields other than allowed, to catch typos.
func (p vehicleParser) keys(t *lua.LTable, where string, allowed ...string) {
	t.ForEach(func(k, _ lua.LValue) {
		s, ok := k.(lua.LString)
		if !ok || !slices.Contains(allowed, string(s)) {
			p.fail("%s: unknown field %s", where, k.String())
		}
	})
}

func (p vehicleParser) str(t *lua.LTable, key string) string {
	s, ok := t.RawGetString(key).(lua.LString)
	if !ok || s == "" {
		p.fail("route: %s must be a non-empty string", key)
	}
	return string(s)
}

// number returns val as an integer.
func (p vehicleParser) number(val lua.LValue, where string) int {
	n, ok := val.(lua.LNumber)
	f := float64(n)
	if !ok || f != math.Trunc(f) || math.Abs(f) > math.MaxInt32 {
		p.fail("%s: an integer expected, got %s", where, val.String())
	}
	return int(f)
}

func (p vehicleParser) optInt(t *lua.LTable, key, where string) int {
	val := t.RawGetString(key)
	if val == lua.LNil {
		return 0
	}
	n := p.number(val, where+" "+key)
	if n < 0 {
		p.fail("%s: %s must not be negative", where, key)
	}
	return n
}

func (p vehicleParser) posInt(t *lua.LTable, key, where string) int {
	n := p.optInt(t, key, where)
	if n <= 0 {
		p.fail("%s: %s must be a positive integer", where, key)
	}
	return n
}

func (p vehicleParser) ms(t *lua.LTable, key, where string) time.Duration {
	return time.Duration(p.optInt(t, key, where)) * time.Millisecond
}

// numbers reads a point written as a list {x, y, z, ...} of 3 to max
// integers.
func (p vehicleParser) numbers(val lua.LValue, where string, maxLen int) []int {
	t, ok := val.(*lua.LTable)
	if !ok || t.Len() < 3 || t.Len() > maxLen {
		p.fail("%s: a point {x, y, z, ...} of 3 to %d integers expected", where, maxLen)
	}
	out := make([]int, t.Len())
	for i := range out {
		out[i] = p.number(t.RawGetInt(i+1), where)
	}
	return out
}

// location reads {x, y, z[, heading]}; a path point {x, y, z, speed,
// rotation} serves too (its speeds are ignored) unless heading is set.
func (p vehicleParser) location(val lua.LValue, where string, heading bool) Location {
	if heading {
		n := p.numbers(val, where, 4)
		loc := Location{X: n[0], Y: n[1], Z: n[2]}
		if len(n) == 4 {
			loc.Heading = n[3]
		}
		return loc
	}
	n := p.numbers(val, where, 5)
	return Location{X: n[0], Y: n[1], Z: n[2]}
}

func (p vehicleParser) optLocation(t *lua.LTable, key, where string) *Location {
	val := t.RawGetString(key)
	if val == lua.LNil {
		return nil
	}
	loc := p.location(val, where+" "+key, false)
	return &loc
}

// announce reads {near = {a, b}, messages = {...}, horn = point}.
func (p vehicleParser) announce(val lua.LValue, where string) *VehicleAnnounce {
	t, ok := val.(*lua.LTable)
	if !ok {
		p.fail("%s: a table {near = {a, b}, messages = {...}, horn = point} expected", where)
	}
	p.keys(t, where, "near", "messages", "horn", "dock")
	a := &VehicleAnnounce{Horn: p.optLocation(t, "horn", where)}
	near, ok := t.RawGetString("near").(*lua.LTable)
	if !ok || near.Len() != 2 {
		p.fail("%s: near must list two points", where)
	}
	a.Near[0] = p.location(near.RawGetInt(1), where+" near", false)
	a.Near[1] = p.location(near.RawGetInt(2), where+" near", false)
	switch msgs := t.RawGetString("messages").(type) {
	case *lua.LTable:
		for i := 1; i <= msgs.Len(); i++ {
			if id := p.number(msgs.RawGetInt(i), where+" messages"); id > 0 {
				a.Messages = append(a.Messages, id)
			} else {
				p.fail("%s: message IDs must be positive", where)
			}
		}
	case *lua.LNilType:
	default:
		p.fail("%s: messages must be a list of system message IDs", where)
	}
	return a
}

// step reads one step of a schedule.
func (p vehicleParser) step(val lua.LValue, where string) VehicleStep {
	t, ok := val.(*lua.LTable)
	if !ok {
		p.fail("%s: a table expected", where)
	}
	p.keys(t, where, "await", "undock", "dock", "say", "horn", "pay", "path", "wait")
	st := VehicleStep{
		Undock: p.optInt(t, "undock", where),
		Dock:   p.optInt(t, "dock", where),
		Horn:   p.optLocation(t, "horn", where),
		Wait:   p.ms(t, "wait", where),
	}
	if val := t.RawGetString("await"); val != lua.LNil {
		st.Busy = p.announce(val, where+" await")
		at, _ := val.(*lua.LTable)
		st.Await = p.posInt(at, "dock", where+" await")
	}
	if val := t.RawGetString("say"); val != lua.LNil {
		st.Announce = p.announce(val, where+" say")
		if st.Announce.Horn == nil && len(st.Announce.Messages) == 0 {
			p.fail("%s: say has neither messages nor a horn", where)
		}
	}
	if val := t.RawGetString("pay"); val != lua.LNil {
		pt, ok := val.(*lua.LTable)
		if !ok {
			p.fail("%s: pay must be a table {item = id, count = n, oust = point}", where)
		}
		p.keys(pt, where+" pay", "item", "count", "oust")
		pay := &VehiclePay{Item: p.optInt(pt, "item", where+" pay"), Count: p.optInt(pt, "count", where+" pay")}
		if pay.Count == 0 {
			pay.Count = 1
		}
		pay.Oust = p.location(pt.RawGetString("oust"), where+" pay oust", false)
		st.Pay = pay
	}
	if val := t.RawGetString("path"); val != lua.LNil {
		path, ok := val.(*lua.LTable)
		if !ok || path.Len() == 0 {
			p.fail("%s: path must be a non-empty list of points", where)
		}
		for i := 1; i <= path.Len(); i++ {
			n := p.numbers(path.RawGetInt(i), fmt.Sprintf("%s path point %d", where, i), 5)
			pt := VehiclePoint{X: n[0], Y: n[1], Z: n[2], Speed: defaultVehicleSpeed, Rotation: defaultVehicleRotation}
			if len(n) == 5 {
				pt.Speed, pt.Rotation = n[3], n[4]
			} else if len(n) == 4 {
				p.fail("%s path point %d: give both the speed and the rotation speed", where, i)
			}
			if pt.Speed < 0 || pt.Rotation < 0 {
				p.fail("%s path point %d: speeds must not be negative", where, i)
			}
			st.Path = append(st.Path, pt)
		}
	}
	return st
}
