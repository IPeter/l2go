package game

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The seasonal events (Plan 14, M11n): the server side of Mobius's
// LongTimeEvent, which scripts/lib/long_time_event.lua drives through the
// long_events module (scripting.LongTimeEvents, implemented here by
// ScriptOutput). The game keeps
//   - the event data (Options.LongTimeEvents) with the config's
//     events.dates on top;
//   - the event drop list (Mobius EventDropManager), rolled through the
//     extra drop hook after a monster's own drops;
//   - the login notice: the enter message shown to players entering the
//     world during the event (Mobius EventAnnouncement; a minimal stand-in
//     until the announcements of M12, not listed by //announces);
//   - the item cleanup at the end (Mobius destroyItemsOnEnd): online
//     players lose the items at once, the database copies go in the
//     background;
//   - the GM commands of AdminEvents: //event_menu lists the seasonal events
//     with Start and Stop buttons, //event_start and //event_stop (and their
//     _menu forms) run one whatever its dates. Mobius's AdminEvents only
//     knows its custom events (which stay out, X8), so its menu would be
//     empty.
func init() { registerFeature("long_time_events", setupLongTimeEvents) }

// The events the GM commands send to an event script (lib/long_time_event.lua).
const (
	longEventStart = "long_time_event_start"
	longEventStop  = "long_time_event_stop"
)

// longEventsFeature is the state of the seasonal events, by script name.
// It lives as long as the server; a script reload keeps it.
type longEventsFeature struct {
	data    *data.LongTimeEvents
	active  map[string]bool
	drops   map[string]bool // the events whose drops run
	notices map[string]longEventNotice
	spawned map[string][]*Npc
}

// longEventNotice is an event's login notice and when it shows (end zero:
// no end).
type longEventNotice struct {
	text       string
	start, end time.Time
}

var _ scripting.LongTimeEvents = (*ScriptOutput)(nil)

func setupLongTimeEvents(s *Server) error {
	f := &s.longEvents
	f.active = make(map[string]bool)
	f.drops = make(map[string]bool)
	f.notices = make(map[string]longEventNotice)
	f.spawned = make(map[string][]*Npc)
	s.OnPlayerEnter(s.showLongEventNotices)
	s.OnExtraDrop(s.longEventDrops)
	var errs []error
	add := func(name string, fn GMCommand) {
		errs = append(errs, s.RegisterGMCommand(name, storage.AccessLevelAdmin, fn))
	}
	add("event_menu", func(p *Player, _ []string) { s.gmEventMenu(p) })
	add("event_start", s.gmEventStart(false))
	add("event_start_menu", s.gmEventStart(true))
	add("event_stop", s.gmEventStop(false))
	add("event_stop_menu", s.gmEventStop(true))
	return errors.Join(errs...)
}

// longEvent resolves a seasonal event: its periods with the config's dates
// on top, on the loop's clock, each date from 00:00 in the server's time
// zone (calendar.go, X5; Mobius uses the JVM's default zone).
func (s *Server) longEvent(script string) (scripting.LongEvent, error) {
	e, ok := s.longEvents.data.Event(script)
	if !ok {
		return scripting.LongEvent{}, scripting.ErrNoLongEvent
	}
	dates := s.gs.Events.Dates[script]
	p, err := e.Period(dates.Active, dates.DropPeriod, s.Calendar().In(s.Now()))
	if err != nil {
		return scripting.LongEvent{}, err
	}
	out := scripting.LongEvent{
		Name: e.Name, Start: p.Start, End: p.End, DropStart: p.DropStart, DropEnd: p.DropEnd,
		OnEnter: e.OnEnter, OnEnd: e.OnEnd,
	}
	for _, sp := range e.Spawns {
		out.Spawns = append(out.Spawns, scripting.LongEventSpawn{
			NpcID:   sp.NpcID,
			Loc:     scripting.Location{X: sp.X, Y: sp.Y, Z: sp.Z, Heading: sp.Heading},
			Respawn: time.Duration(sp.RespawnTime) * time.Second,
		})
	}
	return out, nil
}

// LongEventActive reports whether a seasonal event runs (Mobius
// isEventPeriod). Loop-only.
func (s *Server) LongEventActive(script string) bool { return s.longEvents.active[script] }

// longEventDrops is Mobius EventDropManager.doEventDrop, an extra drop
// hook: for a player no more than eventItemMaxLevelDifference levels above
// the monster, every drop of every running event rolls (in script name
// order) if the monster is one it names and its level is in range:
// Rnd.get(100d) < chance, then a count from min to max, looted or dropped
// as the monster's own drops are; a summon's kill drops for its owner.
func (s *Server) longEventDrops(n *Npc, killer Creature) {
	f := &s.longEvents
	if len(f.drops) == 0 {
		return
	}
	p := s.actingPlayer(killer) // a summon's owner (summon.go): Mobius isPlayable, asPlayer
	if p == nil {
		return
	}
	if p.Level()-n.Level() > s.gs.Events.ItemMaxLevelDifference {
		return
	}
	combat := s.combatSettings()
	for _, script := range slices.Sorted(maps.Keys(f.drops)) {
		e, ok := f.data.Event(script)
		if !ok {
			continue
		}
		for _, d := range e.Drops {
			if len(d.MonsterIDs) > 0 && !slices.Contains(d.MonsterIDs, n.Template.ID) {
				continue
			}
			level := n.Level()
			if level < d.MinLevel || level > d.MaxLevel || !(s.rnd.GetFloat64(100) < d.Chance) {
				continue
			}
			count := s.rnd.Between(d.Min, d.Max)
			t, ok := s.data.ItemTemplate(d.ItemID)
			if !ok || count <= 0 {
				continue
			}
			if (combat.AutoLoot || n.Template.Status.Flying) && p.InGame() {
				s.autoLoot(p, n, t, count)
			} else {
				s.npcDropItem(n, p, t, count)
			}
		}
	}
}

// showLongEventNotices is the OnPlayerEnter hook: the enter message of
// every running event, as an announcement with the player's name (Mobius
// AnnouncementsTable.sendAnnouncements for EventAnnouncement).
func (s *Server) showLongEventNotices(p *Player) {
	now := s.Now()
	for _, script := range slices.Sorted(maps.Keys(s.longEvents.notices)) {
		n := s.longEvents.notices[script]
		if now.Before(n.start) || (!n.end.IsZero() && now.After(n.end)) {
			continue
		}
		p.Send(serverpackets.NewCreatureSayPacket(serverpackets.CreatureSay{
			Type: serverpackets.ChatAnnouncement, SenderName: p.Name(), Text: n.text,
		}))
	}
}

// itemPurger is a character store that can delete every item of some
// templates (mongostore and memstore Characters).
type itemPurger interface {
	DeleteItemsByItemID(ctx context.Context, itemIDs []int32) (int64, error)
}

// destroyLongEventItems is Mobius destroyItemsOnEnd: the items leave every
// player in the game now, and every stored character in the background.
func (s *Server) destroyLongEventItems(script string) {
	e, ok := s.longEvents.data.Event(script)
	if !ok || len(e.DestroyItemsOnEnd) == 0 {
		return
	}
	for p := range s.Players() {
		for _, id := range e.DestroyItemsOnEnd {
			s.takeQuestItems(p, id, -1)
		}
	}
	purger, ok := s.chars.(itemPurger)
	if !ok {
		s.log.Warn("game: the character store cannot delete event items", "event", script)
		return
	}
	ids := make([]int32, len(e.DestroyItemsOnEnd))
	for i, id := range e.DestroyItemsOnEnd {
		ids[i] = int32(id) //nolint:gosec // item IDs fit in int32
	}
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
		defer cancel()
		n, err := purger.DeleteItemsByItemID(ctx, ids)
		if err != nil {
			s.log.Error("game: cannot delete the items of an ended event", "event", script, "items", ids, "err", err)
			return
		}
		s.log.Info("event items deleted", "event", script, "items", ids, "deleted", n)
	}()
}

// spawnLongEvent puts an event's NPCs into the world right after the
// running work (so a long list costs no script time), each leaving after
// despawn (0: never): Mobius spawnNpcs, addSpawn without a random offset.
// Mobius's respawnTime, which no shipped event uses, is not supported. An
// event stopped meanwhile spawns nothing.
func (s *Server) spawnLongEvent(script string, despawn time.Duration) {
	e, ok := s.longEvents.data.Event(script)
	if !ok || len(e.Spawns) == 0 {
		return
	}
	s.loop.Post(func() {
		if !s.longEvents.active[script] {
			return
		}
		for _, sp := range e.Spawns {
			if sp.RespawnTime > 0 {
				s.log.Warn("game: event NPC respawns are not supported", "event", script, "npc", sp.NpcID)
			}
			pos := world.Position{X: int32(sp.X), Y: int32(sp.Y), Z: int32(sp.Z), Heading: int32(sp.Heading)} //nolint:gosec // coordinates fit in int32
			n, err := s.SpawnNpc(sp.NpcID, pos)
			if err != nil {
				s.log.Warn("game: an event NPC cannot spawn", "event", script, "npc", sp.NpcID, "err", err)
				continue
			}
			s.longEvents.spawned[script] = append(s.longEvents.spawned[script], n)
			if despawn > 0 {
				s.loop.After(despawn, func() {
					if n.InWorld() {
						s.DespawnNpc(n)
					}
				})
			}
		}
	})
}

// despawnLongEvent removes the NPCs of an event still in the world.
func (s *Server) despawnLongEvent(script string) {
	for _, n := range s.longEvents.spawned[script] {
		if n.InWorld() {
			s.DespawnNpc(n)
		}
	}
	delete(s.longEvents.spawned, script)
}

// ----- the GM commands -----

// gmEventMenu shows gm_events.htm with a row per seasonal event that has a
// script (Mobius AdminEvents.showMenu).
func (s *Server) gmEventMenu(p *Player) {
	var list strings.Builder
	if s.longEvents.data != nil {
		for _, e := range s.longEvents.data.Events {
			if s.scripts == nil || !s.scripts.HasScript(e.Script) {
				continue
			}
			list.WriteString(`<tr><td><font color="LEVEL">` + e.Script + `:</font></td><br><td><button value="Start" action="bypass -h admin_event_start_menu ` +
				e.Script + `" width=65 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2"></td><td><button value="Stop" action="bypass -h admin_event_stop_menu ` +
				e.Script + `" width=65 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2"></td></tr>`)
		}
	}
	page, ok := s.htmlFile(p, adminHTMLFolder+"gm_events.htm")
	if !ok {
		page = adminMissingPage(htmlPrefix + adminHTMLFolder + "gm_events.htm")
	}
	s.sendPage(p, newPage(0, strings.ReplaceAll(page, "%LIST%", list.String())))
}

// longEventScript returns the event named by a GM command's argument, or
// false after telling the GM the usage.
func (s *Server) longEventScript(p *Player, args []string, usage string) (string, bool) {
	if len(args) > 0 && s.scripts != nil && s.scripts.HasScript(args[0]) {
		if _, ok := s.longEvents.data.Event(args[0]); ok {
			return args[0], true
		}
	}
	gmSysMessage(p, usage)
	return "", false
}

// gmEventStart is //event_start <event> and //event_start_menu <event>: the
// event starts now, whatever its dates (lib/long_time_event.lua).
func (s *Server) gmEventStart(menu bool) GMCommand {
	return func(p *Player, args []string) {
		if menu {
			s.gmEventMenu(p)
		}
		name, ok := s.longEventScript(p, args, "Usage: //event_start <eventname>")
		if !ok {
			return
		}
		if !s.LongEventActive(name) {
			s.scripts.Event(s.background(), name, longEventStart, p, nil)
		}
		if s.LongEventActive(name) {
			gmSysMessage(p, "Event "+name+" started.")
		} else {
			gmSysMessage(p, "There is problem starting "+name+" event.")
		}
	}
}

// gmEventStop is //event_stop <event> and //event_stop_menu <event>: the
// event ends now and its NPCs leave.
func (s *Server) gmEventStop(menu bool) GMCommand {
	return func(p *Player, args []string) {
		if menu {
			s.gmEventMenu(p)
		}
		name, ok := s.longEventScript(p, args, "Usage: //event_stop <eventname>")
		if !ok {
			return
		}
		if !s.LongEventActive(name) {
			gmSysMessage(p, "There is problem with stoping "+name+" event.")
			return
		}
		s.scripts.Event(s.background(), name, longEventStop, p, nil)
		if s.LongEventActive(name) {
			gmSysMessage(p, "There is problem with stoping "+name+" event.")
		} else {
			gmSysMessage(p, "Event "+name+" stopped.")
		}
	}
}

// ----- the scripts' side (scripting.LongTimeEvents) -----

// LongEvent resolves a seasonal event.
func (o *ScriptOutput) LongEvent(script string) (scripting.LongEvent, error) {
	s := o.srv.Load()
	if s == nil {
		return scripting.LongEvent{}, scripting.ErrNoLongEvent
	}
	return s.longEvent(script)
}

// SetLongEventActive marks an event running or not.
func (o *ScriptOutput) SetLongEventActive(script string, active bool) {
	if s := o.srv.Load(); s != nil {
		if active {
			s.longEvents.active[script] = true
		} else {
			delete(s.longEvents.active, script)
		}
	}
}

// LongEventActive reports whether an event runs.
func (o *ScriptOutput) LongEventActive(script string) bool {
	s := o.srv.Load()
	return s != nil && s.LongEventActive(script)
}

// SetLongEventDrops starts or stops an event's drops.
func (o *ScriptOutput) SetLongEventDrops(script string, on bool) {
	s := o.srv.Load()
	if s == nil {
		return
	}
	if _, ok := s.longEvents.data.Event(script); on && ok {
		s.longEvents.drops[script] = true
	} else {
		delete(s.longEvents.drops, script)
	}
}

// SetLongEventNotice sets or removes an event's login notice.
func (o *ScriptOutput) SetLongEventNotice(script, text string, start, end time.Time) {
	s := o.srv.Load()
	if s == nil {
		return
	}
	if text == "" {
		delete(s.longEvents.notices, script)
		return
	}
	s.longEvents.notices[script] = longEventNotice{text: text, start: start, end: end}
}

// AnnounceToAll announces text to every player in the game.
func (o *ScriptOutput) AnnounceToAll(text string) {
	if s := o.srv.Load(); s != nil {
		s.announceToAll(text, false)
	}
}

// DestroyLongEventItems takes an event's items from every character.
func (o *ScriptOutput) DestroyLongEventItems(script string) {
	if s := o.srv.Load(); s != nil {
		s.destroyLongEventItems(script)
	}
}

// SpawnLongEvent puts an event's NPCs into the world after the handler.
func (o *ScriptOutput) SpawnLongEvent(script string, despawn time.Duration) {
	if s := o.srv.Load(); s != nil {
		s.spawnLongEvent(script, despawn)
	}
}

// DespawnLongEvent removes an event's NPCs.
func (o *ScriptOutput) DespawnLongEvent(script string) {
	if s := o.srv.Load(); s != nil {
		s.despawnLongEvent(script)
	}
}
