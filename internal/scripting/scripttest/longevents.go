package scripttest

import (
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/scripting"
)

// The seasonal events of the fake world (scripting.LongTimeEvents): the
// events a test adds, with their dates resolved on the fake clock as the
// game does, and a record of what the scripts did with them.

var _ scripting.LongTimeEvents = (*World)(nil)

// longEventsWorld is the seasonal event state of a World.
type longEventsWorld struct {
	events    map[string]longEventEntry
	active    map[string]bool
	drops     map[string]bool
	notices   map[string]LongEventNotice
	spawned   map[string][]int32 // object IDs, by script
	announced []string
	destroyed []string
}

// longEventEntry is an event a test added, with its date override.
type longEventEntry struct {
	event              *data.LongTimeEvent
	active, dropPeriod string
}

// LongEventNotice is a login notice a script set.
type LongEventNotice struct {
	Text       string
	Start, End time.Time
}

func (s *longEventsWorld) init() {
	if s.events == nil {
		s.events = make(map[string]longEventEntry)
		s.active = make(map[string]bool)
		s.drops = make(map[string]bool)
		s.notices = make(map[string]LongEventNotice)
		s.spawned = make(map[string][]int32)
	}
}

// AddLongEvent makes an event known to the scripts. active and dropPeriod,
// if not empty, replace its dates as the config's events.dates do.
func (w *World) AddLongEvent(e *data.LongTimeEvent, active, dropPeriod string) {
	w.longEvents.init()
	w.longEvents.events[e.Script] = longEventEntry{event: e, active: active, dropPeriod: dropPeriod}
}

// LongEventDrops reports whether the event's drops run.
func (w *World) LongEventDrops(script string) bool { return w.longEvents.drops[script] }

// LongEventNoticeOf returns the login notice of an event.
func (w *World) LongEventNoticeOf(script string) (LongEventNotice, bool) {
	n, ok := w.longEvents.notices[script]
	return n, ok
}

// LoginNotices returns the texts a player entering the world now sees.
func (w *World) LoginNotices() []string {
	var out []string
	for _, n := range w.longEvents.notices {
		if !w.now.Before(n.Start) && (n.End.IsZero() || !w.now.After(n.End)) {
			out = append(out, n.Text)
		}
	}
	return out
}

// Announced returns the announcements to every player, in order.
func (w *World) Announced() []string { return w.longEvents.announced }

// DestroyedItemsOf returns the scripts whose items were destroyed, one entry
// per call.
func (w *World) DestroyedItemsOf() []string { return w.longEvents.destroyed }

// LongEvent resolves an added event on the fake clock.
func (w *World) LongEvent(script string) (scripting.LongEvent, error) {
	entry, ok := w.longEvents.events[script]
	if !ok {
		return scripting.LongEvent{}, scripting.ErrNoLongEvent
	}
	e := entry.event
	p, err := e.Period(entry.active, entry.dropPeriod, w.now)
	if err != nil {
		return scripting.LongEvent{}, fmt.Errorf("event %s: %w", script, err)
	}
	out := scripting.LongEvent{
		Name: e.Name, Start: p.Start, End: p.End, DropStart: p.DropStart, DropEnd: p.DropEnd,
		OnEnter: e.OnEnter, OnEnd: e.OnEnd,
	}
	for _, s := range e.Spawns {
		out.Spawns = append(out.Spawns, scripting.LongEventSpawn{
			NpcID:   s.NpcID,
			Loc:     scripting.Location{X: s.X, Y: s.Y, Z: s.Z, Heading: s.Heading},
			Respawn: time.Duration(s.RespawnTime) * time.Second,
		})
	}
	return out, nil
}

// SetLongEventActive records the event's state.
func (w *World) SetLongEventActive(script string, active bool) {
	w.longEvents.init()
	w.longEvents.active[script] = active
}

// LongEventActive returns the event's state.
func (w *World) LongEventActive(script string) bool { return w.longEvents.active[script] }

// SetLongEventDrops records whether the drops run.
func (w *World) SetLongEventDrops(script string, on bool) {
	w.longEvents.init()
	w.longEvents.drops[script] = on
}

// SetLongEventNotice records the login notice.
func (w *World) SetLongEventNotice(script, text string, start, end time.Time) {
	w.longEvents.init()
	if text == "" {
		delete(w.longEvents.notices, script)
		return
	}
	w.longEvents.notices[script] = LongEventNotice{Text: text, Start: start, End: end}
}

// AnnounceToAll records an announcement.
func (w *World) AnnounceToAll(text string) {
	w.longEvents.announced = append(w.longEvents.announced, text)
}

// SpawnLongEvent spawns the event's NPCs as AddSpawn does, at once, and
// keeps them for DespawnLongEvent. The despawn delay is not simulated.
func (w *World) SpawnLongEvent(script string, _ time.Duration) {
	w.longEvents.init()
	entry, ok := w.longEvents.events[script]
	if !ok {
		return
	}
	for _, s := range entry.event.Spawns {
		n := w.AddSpawn(s.NpcID, scripting.Location{X: s.X, Y: s.Y, Z: s.Z, Heading: s.Heading}, false, 0, false)
		w.longEvents.spawned[script] = append(w.longEvents.spawned[script], n.ObjectID())
	}
}

// DespawnLongEvent deletes the event's NPCs still there.
func (w *World) DespawnLongEvent(script string) {
	for _, id := range w.longEvents.spawned[script] {
		if n, ok := w.npcs[id]; ok {
			w.DeleteNpc(n.Ref())
		}
	}
	delete(w.longEvents.spawned, script)
}

// DestroyLongEventItems takes the event's items from the fake players and
// records the call.
func (w *World) DestroyLongEventItems(script string) {
	w.longEvents.destroyed = append(w.longEvents.destroyed, script)
	entry, ok := w.longEvents.events[script]
	if !ok {
		return
	}
	for _, id := range entry.event.DestroyItemsOnEnd {
		for _, p := range w.players {
			delete(p.Items, id)
		}
	}
}
