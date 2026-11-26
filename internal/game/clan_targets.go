package game

import (
	"github.com/IPeter/l2go/internal/npcai"
	"github.com/IPeter/l2go/internal/skills"
)

// The clan side of the skill engine's world (skills.ClanWorld, Plan 12
// Phase 7): the clan target types (CLAN, CLAN_MEMBER, CORPSE_CLAN,
// PARTY_CLAN) ask it for a player's clan members in the game and for the
// NPC clans of the templates.

var _ skills.ClanWorld = skillWorld{}

// ClanMembers returns the adapters of the members of c's clan in the game,
// in member order; nil when c is not a player in a clan.
func (w skillWorld) ClanMembers(c skills.Creature) []skills.Creature {
	p, ok := w.s.gameCreature(c).(*Player)
	if !ok || p.clan.clan == nil {
		return nil
	}
	var out []skills.Creature
	for _, q := range p.clan.clan.OnlineMembers() {
		if sc := w.s.skillCreatureOf(q); sc != nil {
			out = append(out, sc)
		}
	}
	return out
}

// NpcHasClans reports whether the NPC's template has clans.
func (w skillWorld) NpcHasClans(npc skills.Creature) bool {
	n, ok := w.s.gameCreature(npc).(*Npc)
	return ok && len(w.s.aiClassOf(n).clans) > 0
}

// NpcInMyClan is Mobius Npc.isInMyClan (NpcTemplate.isClan).
func (w skillWorld) NpcInMyClan(npc, other skills.Creature) bool {
	a, okA := w.s.gameCreature(npc).(*Npc)
	b, okB := w.s.gameCreature(other).(*Npc)
	if !okA || !okB {
		return false
	}
	mine := &npcai.Profile{Clans: w.s.aiClassOf(a).clans}
	return mine.IsClan(w.s.aiClassOf(b).clans)
}
