package skilltest

import (
	"math"
	"slices"

	"github.com/IPeter/l2go/internal/skills"
)

// World is a skills.World over a list of creatures.
type World struct {
	// Creatures are the creatures in the world, in iteration order.
	Creatures []skills.Creature
	// Parties are the parties, leader first.
	Parties [][]skills.Creature
	// Summons maps an owner's object ID to its summon.
	Summons map[int32]skills.Creature
	// Hidden lists the pairs (from, to) without line of sight.
	Hidden map[[2]int32]bool
	// Old lists the corpses too old to consume.
	Old map[int32]bool
	// Night is game-time night.
	Night bool
	// NpcClans maps an NPC's object ID to its template clans ("ALL" is in
	// every clan).
	NpcClans map[int32][]string
}

var _ skills.ClanWorld = (*World)(nil)

// ClanMembers returns the players of the world with c's clan (Creature.Clan),
// in world order; nil when c has none.
func (w *World) ClanMembers(c skills.Creature) []skills.Creature {
	cc, ok := c.(*Creature)
	if !ok || cc.Clan == 0 {
		return nil
	}
	var out []skills.Creature
	for _, o := range w.Creatures {
		if oc, ok := o.(*Creature); ok && oc.Clan == cc.Clan && oc.AsPlayer() != nil {
			out = append(out, o)
		}
	}
	return out
}

// NpcHasClans reports whether the NPC has clans in NpcClans.
func (w *World) NpcHasClans(npc skills.Creature) bool { return len(w.NpcClans[npc.ObjectID()]) > 0 }

// NpcInMyClan is NpcTemplate.isClan: npc is in "ALL" or shares a clan with
// other.
func (w *World) NpcInMyClan(npc, other skills.Creature) bool {
	mine, theirs := w.NpcClans[npc.ObjectID()], w.NpcClans[other.ObjectID()]
	if len(mine) == 0 || len(theirs) == 0 {
		return false
	}
	if slices.Contains(mine, "ALL") {
		return true
	}
	return slices.ContainsFunc(theirs, func(c string) bool { return slices.Contains(mine, c) })
}

// Add adds creatures to the world.
func (w *World) Add(cs ...skills.Creature) { w.Creatures = append(w.Creatures, cs...) }

// VisibleInRange returns the other creatures within rng (3D, no collision).
func (w *World) VisibleInRange(c skills.Creature, rng int) []skills.Creature {
	var out []skills.Creature
	for _, o := range w.Creatures {
		if o.ObjectID() == c.ObjectID() {
			continue
		}
		a, b := c.Location(), o.Location()
		dx, dy, dz := float64(a.X-b.X), float64(a.Y-b.Y), float64(a.Z-b.Z)
		if math.Sqrt(dx*dx+dy*dy+dz*dz) <= float64(rng) {
			out = append(out, o)
		}
	}
	return out
}

// Visible returns every other creature.
func (w *World) Visible(c skills.Creature) []skills.Creature {
	var out []skills.Creature
	for _, o := range w.Creatures {
		if o.ObjectID() != c.ObjectID() {
			out = append(out, o)
		}
	}
	return out
}

// CanSee reports line of sight.
func (w *World) CanSee(from, to skills.Creature) bool {
	return !w.Hidden[[2]int32{from.ObjectID(), to.ObjectID()}]
}

// Party returns c's party.
func (w *World) Party(c skills.Creature) []skills.Creature {
	for _, p := range w.Parties {
		if slices.ContainsFunc(p, func(m skills.Creature) bool { return m.ObjectID() == c.ObjectID() }) {
			return p
		}
	}
	return nil
}

// SameParty reports whether a and b share a party.
func (w *World) SameParty(a, b skills.Creature) bool {
	pa, pb := w.Party(a), w.Party(b)
	return pa != nil && pb != nil && pa[0].ObjectID() == pb[0].ObjectID()
}

// Summon returns the summon of owner.
func (w *World) Summon(owner skills.Creature) skills.Creature {
	if s, ok := w.Summons[owner.ObjectID()]; ok {
		return s
	}
	return nil
}

// OldCorpse reports a corpse too old to consume.
func (w *World) OldCorpse(corpse, _ skills.Creature) bool { return w.Old[corpse.ObjectID()] }

// IsNight reports night.
func (w *World) IsNight() bool { return w.Night }

// ValidLocation returns to: there is no geodata.
func (w *World) ValidLocation(_, to skills.Location) skills.Location { return to }
