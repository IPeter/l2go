package data

// Seasonal events (Plan 14, M11n): the config.xml of each Mobius
// LongTimeEvent script in dist/game/data/scripts/events/ (Christmas, L2 Day,
// Heavy Medal), converted by cmd/datagen into LongTimeEventsFile and loaded
// with LoadLongTimeEvents. The scripts themselves are Lua
// (scripts/events/<Script>/, scripts/lib/long_time_event.lua); the game
// keeps the drop list, the login notice and the item cleanup.

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

// LongTimeEventsFile is the path of the seasonal event data, relative to the
// data folder.
const LongTimeEventsFile = "events/longtimeevents.json"

// LongTimeEvent is one event's config.xml (Mobius LongTimeEvent.loadConfig).
type LongTimeEvent struct {
	// Script is the event's script name, Mobius's class and folder name
	// (Christmas, L2Day, HeavyMedal): the Lua script of the same name runs
	// it, and the config overrides use it.
	Script string `json:"script"`
	// Name is the event's name attribute ("L2 Day").
	Name string `json:"name"`
	// Active is the event period, "dd MM yyyy-dd MM yyyy", or "dd MM-dd MM"
	// for every year (see ParseEventDates).
	Active string `json:"active"`
	// DropPeriod is when monsters drop the event items, in the same form;
	// empty means the event period.
	DropPeriod string `json:"dropPeriod,omitempty"`
	// Drops are rolled for every monster a player kills while the drops run
	// (Mobius EventDropManager), in this order.
	Drops []EventDrop `json:"drops,omitempty"`
	// Spawns are the NPCs that stand in the world during the event.
	Spawns []EventSpawn `json:"spawns,omitempty"`
	// OnEnter is announced at the start and shown to players logging in
	// during the event; OnEnd is announced at the end. Either may be empty.
	OnEnter string `json:"onEnter,omitempty"`
	OnEnd   string `json:"onEnd,omitempty"`
	// DestroyItemsOnEnd are the item IDs taken from every character at the
	// end of the event (and at every start of the server after it).
	DestroyItemsOnEnd []int `json:"destroyItemsOnEnd,omitempty"`
}

// EventDrop is one entry of an event's drop list (Mobius EventDropHolder).
type EventDrop struct {
	ItemID int `json:"itemId"`
	// Min and Max bound the count, both included.
	Min int `json:"min"`
	Max int `json:"max"`
	// Chance is in percent, 0 to 100.
	Chance float64 `json:"chance"`
	// MinLevel and MaxLevel bound the monster's level (Mobius defaults 1 and
	// Integer.MAX_VALUE).
	MinLevel int `json:"minLevel"`
	MaxLevel int `json:"maxLevel"`
	// MonsterIDs, if any, are the only NPC templates that drop it.
	MonsterIDs []int `json:"monsterIds,omitempty"`
}

// EventSpawn is one NPC of an event's spawn list. It is stored as the array
// [npcId, x, y, z, heading, respawnTime].
type EventSpawn struct {
	NpcID   int
	X, Y, Z int
	Heading int
	// RespawnTime is in seconds; 0 means the NPC does not respawn.
	RespawnTime int
}

// MarshalJSON writes the spawn as [npcId, x, y, z, heading, respawnTime].
func (s EventSpawn) MarshalJSON() ([]byte, error) {
	return json.Marshal([6]int{s.NpcID, s.X, s.Y, s.Z, s.Heading, s.RespawnTime})
}

// UnmarshalJSON reads the array form of MarshalJSON.
func (s *EventSpawn) UnmarshalJSON(b []byte) error {
	var a []int
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	if len(a) != 6 {
		return fmt.Errorf("event spawn %s: want [npcId, x, y, z, heading, respawnTime]", b)
	}
	*s = EventSpawn{NpcID: a[0], X: a[1], Y: a[2], Z: a[3], Heading: a[4], RespawnTime: a[5]}
	return nil
}

// LongTimeEvents are the checked seasonal events, sorted by script name.
// They must not be modified after NewLongTimeEvents or LoadLongTimeEvents
// returns, and are then safe for concurrent use.
type LongTimeEvents struct {
	Events []LongTimeEvent
}

// LoadLongTimeEvents reads and checks LongTimeEventsFile in dir.
func LoadLongTimeEvents(dir string) (*LongTimeEvents, error) {
	e, err := LoadLongTimeEventsFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return e, nil
}

// LoadLongTimeEventsFS reads and checks LongTimeEventsFile in fsys.
func LoadLongTimeEventsFS(fsys fs.FS) (*LongTimeEvents, error) {
	var list []LongTimeEvent
	if err := readJSON(fsys, LongTimeEventsFile, &list); err != nil {
		return nil, err
	}
	return NewLongTimeEvents(list)
}

// scriptName is what a Lua script name may be.
var scriptName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// NewLongTimeEvents checks the events; the slice is used as it is. Script
// names are unique, sorted and valid script names; the periods parse; drops
// have positive item IDs, 0 ≤ min ≤ max, a chance from 0 to 100 and a level
// range; spawns positive NPC IDs and no negative respawn time.
func NewLongTimeEvents(list []LongTimeEvent) (*LongTimeEvents, error) {
	var p problems
	// Any fixed day checks the syntax of the yearly form.
	probe := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range list {
		e := &list[i]
		where := fmt.Sprintf("%s: event %s", LongTimeEventsFile, e.Script)
		if !scriptName.MatchString(e.Script) {
			p.addf("%s: the script name must be letters, digits and underscores", where)
		}
		if i > 0 && list[i-1].Script >= e.Script {
			p.addf("%s: script names are not sorted and unique", where)
		}
		if _, err := e.Period("", "", probe); err != nil {
			p.addf("%s: %v", where, err)
		}
		for _, d := range e.Drops {
			switch {
			case d.ItemID <= 0:
				p.addf("%s: drop of item %d: the item ID must be positive", where, d.ItemID)
			case d.Min < 0 || d.Min > d.Max:
				p.addf("%s: drop of item %d: count %d-%d", where, d.ItemID, d.Min, d.Max)
			case !(d.Chance >= 0 && d.Chance <= 100):
				p.addf("%s: drop of item %d: chance %v is not 0 to 100", where, d.ItemID, d.Chance)
			case d.MinLevel > d.MaxLevel:
				p.addf("%s: drop of item %d: level %d-%d", where, d.ItemID, d.MinLevel, d.MaxLevel)
			}
		}
		for _, s := range e.Spawns {
			if s.NpcID <= 0 || s.RespawnTime < 0 {
				p.addf("%s: spawn of NPC %d: the NPC ID must be positive and the respawn time not negative", where, s.NpcID)
			}
		}
		for _, id := range e.DestroyItemsOnEnd {
			if id <= 0 {
				p.addf("%s: item %d to destroy: the item ID must be positive", where, id)
			}
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return &LongTimeEvents{Events: list}, nil
}

// Event returns the event of a script.
func (l *LongTimeEvents) Event(script string) (*LongTimeEvent, bool) {
	if l == nil {
		return nil, false
	}
	i, ok := slices.BinarySearchFunc(l.Events, script, func(e LongTimeEvent, s string) int { return cmp.Compare(e.Script, s) })
	if !ok {
		return nil, false
	}
	return &l.Events[i], true
}

// CheckRefs checks that the items and NPCs the events name exist.
func (l *LongTimeEvents) CheckRefs(itemExists, npcExists func(id int) bool) error {
	var p problems
	for _, e := range l.Events {
		where := fmt.Sprintf("%s: event %s", LongTimeEventsFile, e.Script)
		for _, d := range e.Drops {
			if !itemExists(d.ItemID) {
				p.addf("%s: drop of item %d: no such item", where, d.ItemID)
			}
			for _, id := range d.MonsterIDs {
				if !npcExists(id) {
					p.addf("%s: drop of item %d: no such monster %d", where, d.ItemID, id)
				}
			}
		}
		for _, s := range e.Spawns {
			if !npcExists(s.NpcID) {
				p.addf("%s: spawn of NPC %d: no such NPC", where, s.NpcID)
			}
		}
		for _, id := range e.DestroyItemsOnEnd {
			if !itemExists(id) {
				p.addf("%s: item %d to destroy: no such item", where, id)
			}
		}
	}
	return p.err()
}

// EventPeriod is when an event runs and when its drops do.
type EventPeriod struct {
	Start, End         time.Time
	DropStart, DropEnd time.Time
}

// Contains reports whether t is within the event period, both ends included
// (Mobius isWithinRange).
func (p EventPeriod) Contains(t time.Time) bool { return !t.Before(p.Start) && !t.After(p.End) }

// DropsAt reports whether t is within the drop period, both ends included
// (Mobius isWithinDropPeriod).
func (p EventPeriod) DropsAt(t time.Time) bool {
	return !t.Before(p.DropStart) && !t.After(p.DropEnd)
}

// Period returns the event's period and drop period. active, if not empty,
// replaces the configured period, and dropPeriod then replaces the drop
// period (empty: the new event period). Dates are taken at the start of the
// day in now's location; a yearly period ("dd MM-dd MM") is in now's year
// (Mobius LongTimeEvent.loadConfig, with the JVM's default zone). The drop
// period must lie within the event period, and the event must start before
// it ends; a drop period that ends before it starts is allowed and never
// drops (Heavy Medal's).
func (e *LongTimeEvent) Period(active, dropPeriod string, now time.Time) (EventPeriod, error) {
	act, drop := e.Active, e.DropPeriod
	if active != "" {
		act, drop = active, dropPeriod
	}
	if drop == "" {
		drop = act
	}
	var p EventPeriod
	var err error
	if p.Start, p.End, err = ParseEventDates(act, now); err != nil {
		return p, fmt.Errorf("period %q: %w", act, err)
	}
	if p.DropStart, p.DropEnd, err = ParseEventDates(drop, now); err != nil {
		return p, fmt.Errorf("drop period %q: %w", drop, err)
	}
	if p.DropStart.Before(p.Start) || p.DropEnd.After(p.End) {
		return p, fmt.Errorf("drop period %q must be within the active period %q", drop, act)
	}
	if !p.Start.Before(p.End) {
		return p, fmt.Errorf("illegal event period %q", act)
	}
	return p, nil
}

// eventDateLayout is Mobius's "dd MM yyyy".
const eventDateLayout = "02 01 2006"

// ParseEventDates parses an event period: "dd MM yyyy-dd MM yyyy" (21
// characters), or "dd MM-dd MM" (11 characters) in now's year. Each date is
// the start of that day in now's location.
func ParseEventDates(period string, now time.Time) (start, end time.Time, err error) {
	var from, to string
	switch len(period) {
	case 21:
		from, to, _ = strings.Cut(period, "-")
	case 11:
		year := fmt.Sprintf(" %04d", now.Year())
		from, to, _ = strings.Cut(period, "-")
		from, to = from+year, to+year
	default:
		return start, end, fmt.Errorf("want dd MM yyyy-dd MM yyyy or dd MM-dd MM")
	}
	if start, err = time.ParseInLocation(eventDateLayout, strings.TrimSpace(from), now.Location()); err != nil {
		return start, end, fmt.Errorf("start date: %w", err)
	}
	if end, err = time.ParseInLocation(eventDateLayout, strings.TrimSpace(to), now.Location()); err != nil {
		return start, end, fmt.Errorf("end date: %w", err)
	}
	return start, end, nil
}
