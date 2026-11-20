package game

import (
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The on_creature_see scan (Plan 13, decision A5): Mobius
// CreatureSeeTaskManager and Creature.updateSeenCreatures. An NPC whose
// template a script watches with on_creature_see joins the scan when it
// spawns (Npc.onSpawn's initSeenCreatures). Every second each one in an
// active neighbourhood looks around within its template's aggro range (else
// 1,500): every visible creature it has not seen yet is seen, and the
// scripts hear of it. A creature it forgets (out of its regions) can be seen
// again.

// creatureSeeEvent is one on_creature_see the scan raises.
type creatureSeeEvent struct {
	npc  *Npc
	seen Creature
}

// creatureSeeSpawned adds a spawned NPC to the scan when a script watches
// its template (Mobius CREATURE_SEE_IDS).
func (s *Server) creatureSeeSpawned(n *Npc) {
	x := &n.ai.x
	if x.seeing || s.scripts == nil || !s.scripts.Listens("creature_see", n.Template.ID) {
		return
	}
	x.seeing = true
	if x.seen == nil {
		x.seen = make(map[world.Object]struct{}, 1)
	}
	s.ai.x.see = append(s.ai.x.see, n)
}

// creatureSeeScan is CreatureSeeTaskManager's run.
func (s *Server) creatureSeeScan() {
	f := &s.ai.x
	if len(f.see) == 0 {
		return
	}
	// NPCs that left the world leave the scan and forget what they saw
	// (Mobius deleteMe).
	keep := f.see[:0]
	for _, n := range f.see {
		if n.InWorld() {
			keep = append(keep, n)
			continue
		}
		n.ai.x.seeing = false
		clear(n.ai.x.seen)
	}
	clear(f.see[len(keep):])
	f.see = keep

	batch := append(f.seeBatch[:0], f.see...)
	events := f.seeEvents[:0]
	for _, n := range batch {
		events = s.updateSeenCreatures(n, events)
	}
	clear(batch)
	f.seeBatch = batch[:0]
	// The events run after the walk: handlers may move and spawn.
	ctx := s.background()
	for _, ev := range events {
		if s.scripts == nil {
			break
		}
		var seen any
		switch c := ev.seen.(type) {
		case *Player:
			seen = scripting.PlayerRef(c)
		case *Npc:
			seen = c.ScriptRef()
		}
		_ = s.scripts.CreatureSee(ctx, ev.npc.ScriptRef(), seen) // failures are logged
	}
	clear(events)
	f.seeEvents = events[:0]
}

// updateSeenCreatures is Creature.updateSeenCreatures: the creatures n sees
// for the first time, with an event each while a script listens.
func (s *Server) updateSeenCreatures(n *Npc, events []creatureSeeEvent) []creatureSeeEvent {
	if n.Dead || !n.InWorld() || !s.world.NeighborsActive(n) {
		return events
	}
	r := creatureSeeRange
	if ar := n.Template.AI.AggroRange; ar > 0 {
		r = ar
	}
	x := &n.ai.x
	listens := s.scripts != nil && s.scripts.Listens("creature_see", n.Template.ID)
	for o := range s.world.VisibleInRange(n, int32(r)) { //nolint:gosec // aggro ranges are small template values
		c, ok := o.(Creature)
		if !ok || creatureInvisible(o) {
			continue
		}
		if _, seen := x.seen[o]; seen {
			continue
		}
		x.seen[o] = struct{}{}
		if listens {
			events = append(events, creatureSeeEvent{npc: n, seen: c})
		}
	}
	return events
}

// creatureSeeForget is the OnForget hook of the scan (Mobius
// removeSeenCreature from notifyActionForgetObject).
func (s *Server) creatureSeeForget(viewer, object world.Object) {
	if n, ok := viewer.(*Npc); ok && n.ai.x.seeing {
		delete(n.ai.x.seen, object)
	}
}
