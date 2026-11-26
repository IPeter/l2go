package scripting

import (
	"errors"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// The long_events module (Plan 14, M11n): the server side of Mobius's
// LongTimeEvent, which scripts/lib/long_time_event.lua drives. The game
// keeps the event data (with the config's date overrides), the event drop
// list (Mobius EventDropManager), the login notice (EventAnnouncement) and
// the item cleanup; the library decides when they start and stop.

// ErrNoLongEvent means the server has no seasonal event of that script name.
var ErrNoLongEvent = errors.New("no seasonal event of that name")

// LongEvent is a seasonal event as the server resolved it: its periods with
// the config's dates applied, its spawns and messages.
type LongEvent struct {
	// Name is the event's display name ("L2 Day").
	Name string
	// Start and End are the event period, DropStart and DropEnd the drop
	// period (which may end before it starts: then it never drops).
	Start, End         time.Time
	DropStart, DropEnd time.Time
	Spawns             []LongEventSpawn
	// OnEnter and OnEnd are the start and end announcements; either may be
	// empty.
	OnEnter, OnEnd string
}

// LongEventSpawn is one NPC of an event.
type LongEventSpawn struct {
	NpcID int
	Loc   Location
	// Respawn is the respawn delay; 0 means no respawn.
	Respawn time.Duration
}

// LongTimeEvents is what the server keeps of the seasonal events, by script
// name.
type LongTimeEvents interface {
	// LongEvent returns the event (ErrNoLongEvent if there is none), or the
	// error of its dates.
	LongEvent(script string) (LongEvent, error)
	// SetLongEventActive and LongEventActive are Mobius _active /
	// isEventPeriod.
	SetLongEventActive(script string, active bool)
	LongEventActive(script string) bool
	// SetLongEventDrops adds or removes the event's drop list (Mobius
	// EventDropManager.addDrops and removeDrops).
	SetLongEventDrops(script string, on bool)
	// SetLongEventNotice shows text to players entering the world between
	// start and end (end zero: no end); an empty text removes it (Mobius
	// EventAnnouncement).
	SetLongEventNotice(script, text string, start, end time.Time)
	// AnnounceToAll sends text to every player in the game as an
	// announcement (Mobius World.broadcastToAllOnlinePlayers).
	AnnounceToAll(text string)
	// DestroyLongEventItems takes the event's destroyItemsOnEnd items from
	// every character, online or not (Mobius destroyItemsOnEnd).
	DestroyLongEventItems(script string)
	// SpawnLongEvent puts the event's NPCs into the world, each leaving
	// after despawn (0: never), as Mobius spawnNpcs does with addSpawn. The
	// server may do it right after the running handler, so that a long
	// spawn list does not count against the handler's time limit.
	SpawnLongEvent(script string, despawn time.Duration)
	// DespawnLongEvent removes the NPCs SpawnLongEvent put into the world
	// that are still there.
	DespawnLongEvent(script string)
}

// longEventsFuncs is the long_events module.
func (v *vm) longEventsFuncs() map[string]lua.LGFunction {
	events := func(L *lua.LState, fn string) (LongTimeEvents, string) {
		name := L.CheckString(1)
		return worldAs[LongTimeEvents](v, L, fn), name
	}
	ms := func(t time.Time) lua.LValue { return lua.LNumber(t.UnixMilli()) }
	return map[string]lua.LGFunction{
		"config": func(L *lua.LState) int {
			w, name := events(L, "long_events.config")
			e, err := w.LongEvent(name)
			if err != nil {
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			t := L.CreateTable(0, 9)
			t.RawSetString("name", lua.LString(e.Name))
			t.RawSetString("start_ms", ms(e.Start))
			t.RawSetString("end_ms", ms(e.End))
			t.RawSetString("drop_start_ms", ms(e.DropStart))
			t.RawSetString("drop_end_ms", ms(e.DropEnd))
			t.RawSetString("on_enter", lua.LString(e.OnEnter))
			t.RawSetString("on_end", lua.LString(e.OnEnd))
			spawns := L.CreateTable(len(e.Spawns), 0)
			for i, s := range e.Spawns {
				st := L.CreateTable(0, 6)
				st.RawSetString("npc", lua.LNumber(s.NpcID))
				st.RawSetString("x", lua.LNumber(s.Loc.X))
				st.RawSetString("y", lua.LNumber(s.Loc.Y))
				st.RawSetString("z", lua.LNumber(s.Loc.Z))
				st.RawSetString("heading", lua.LNumber(s.Loc.Heading))
				st.RawSetString("respawn_ms", lua.LNumber(s.Respawn.Milliseconds()))
				spawns.RawSetInt(i+1, st)
			}
			t.RawSetString("spawns", spawns)
			L.Push(t)
			return 1
		},
		"set_active": func(L *lua.LState) int {
			w, name := events(L, "long_events.set_active")
			w.SetLongEventActive(name, L.ToBool(2))
			return 0
		},
		"is_active": func(L *lua.LState) int {
			w, name := events(L, "long_events.is_active")
			L.Push(lua.LBool(w.LongEventActive(name)))
			return 1
		},
		"set_drops": func(L *lua.LState) int {
			w, name := events(L, "long_events.set_drops")
			w.SetLongEventDrops(name, L.ToBool(2))
			return 0
		},
		"set_notice": func(L *lua.LState) int {
			const fn = "long_events.set_notice"
			w, name := events(L, fn)
			text := L.OptString(2, "")
			start := time.UnixMilli(optInt(L, 3, fn, 0))
			var end time.Time
			if L.Get(4) != lua.LNil {
				end = time.UnixMilli(checkInt(L, 4, fn))
			}
			w.SetLongEventNotice(name, text, start, end)
			return 0
		},
		"announce": func(L *lua.LState) int {
			text := L.CheckString(1)
			worldAs[LongTimeEvents](v, L, "long_events.announce").AnnounceToAll(text)
			return 0
		},
		"destroy_items": func(L *lua.LState) int {
			w, name := events(L, "long_events.destroy_items")
			w.DestroyLongEventItems(name)
			return 0
		},
		"spawn": func(L *lua.LState) int {
			const fn = "long_events.spawn"
			w, name := events(L, fn)
			w.SpawnLongEvent(name, time.Duration(max(optInt(L, 2, fn, 0), 0))*time.Millisecond)
			return 0
		},
		"despawn": func(L *lua.LState) int {
			w, name := events(L, "long_events.despawn")
			w.DespawnLongEvent(name)
			return 0
		},
	}
}
