package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// npcClass is what the NPC code needs to know about a template, worked out
// once per NPC ID and shared by all NPCs of it: the stat engine's template
// and the Mobius classes the NPC type belongs to.
type npcClass struct {
	stats *stats.NpcTemplate
	// isNpc is set for an Npc subclass: Mobius instantiates only those
	// (Spawn's constructor refuses doors, static objects, pets...).
	isNpc bool
	// trap is set for a Trap: Mobius Spawn.doSpawn never spawns one (pets
	// and decoys are not Npc subclasses at all).
	trap bool
	// monster is set for a Monster or a subclass (Mobius isMonster): players
	// attack it with a plain click, and its spawn height is corrected.
	monster bool
	// friendly is set for a FriendlyMob, which players with karma attack.
	friendly bool
	// attackable is set for an Attackable or a subclass (Mobius
	// isAttackable): it starts regenerating when its region activates.
	attackable bool
	// raid is set for a RaidBoss or a GrandBoss (Mobius isRaid): it
	// regenerates with the raid multipliers.
	raid bool
	// randomAnim is set when the NPC plays a random animation when talked to
	// (Mobius hasRandomAnimation: the template's randomAnimation, and an AI
	// that is not CORPSE; MaxNpcAnimation is above 0 by default).
	randomAnim bool

	// first holds the stat numbers of a new NPC of the template, worked out
	// with the first one: every new NPC of a template has the same, so the
	// tens of thousands of spawns at startup compute them once per template.
	first   npcStatCache
	firstOK bool
}

// npcStatCache holds the stat numbers an NPC's packets show, for one
// version of its stat set.
type npcStatCache struct {
	view     stats.NpcView
	runSpeed float64 // Mobius getRunSpeed: 0 sends ServerObjectInfo
	maxMP    int32
	version  uint64
}

// npcClassOf returns the shared class of template t, working it out on
// first use. Loop-only (or in setup).
func (s *Server) npcClassOf(t *data.NpcTemplate) *npcClass {
	if c := s.npc.classes[t.ID]; c != nil {
		return c
	}
	c := &npcClass{
		stats:      npcStatsTemplate(t),
		randomAnim: t.Status.RandomAnimation && t.AI.Type != "CORPSE",
	}
	if w := s.worldData; w != nil {
		c.isNpc = w.NpcTypeIs(t.Type, "Npc")
		c.trap = w.NpcTypeIs(t.Type, "Trap")
		c.monster = w.NpcTypeIs(t.Type, "Monster")
		c.friendly = w.NpcTypeIs(t.Type, "FriendlyMob")
		c.attackable = w.NpcTypeIs(t.Type, "Attackable")
		c.raid = w.NpcTypeIs(t.Type, "RaidBoss") || w.NpcTypeIs(t.Type, "GrandBoss")
	}
	if s.npc.classes == nil {
		s.npc.classes = make(map[int]*npcClass)
	}
	s.npc.classes[t.ID] = c
	return c
}

// npcStatsTemplate converts an NPC template for the stat engine: the stats
// element of the NPC XML as Mobius NpcData reads it. The data already holds
// Mobius's defaults (swim speeds default to the ground speeds); the stat
// engine applies the storage types (float HP, int attack and defence) and
// the 0.1 floor of the ground speeds. NPCs have no magic attack speed in the
// data, so the stat engine's default 333 applies.
func npcStatsTemplate(t *data.NpcTemplate) *stats.NpcTemplate {
	st := &t.Stats
	return &stats.NpcTemplate{
		Level: t.Level,
		Base: stats.BaseStats{
			STR: st.STR, CON: st.CON, DEX: st.DEX,
			INT: st.INT, WIT: st.WIT, MEN: st.MEN,
		},
		HP:            st.Vitals.HP,
		MP:            st.Vitals.MP,
		HPRegen:       st.Vitals.HPRegen,
		MPRegen:       st.Vitals.MPRegen,
		PAtk:          st.Attack.Physical,
		MAtk:          st.Attack.Magical,
		PDef:          st.Defence.Physical,
		MDef:          st.Defence.Magical,
		CritRate:      st.Attack.Critical,
		PAtkSpd:       st.Attack.Speed,
		AttackRange:   st.Attack.Range,
		RunSpeed:      st.Speed.Run,
		WalkSpeed:     st.Speed.Walk,
		SwimRunSpeed:  st.Speed.SwimRun,
		SwimWalkSpeed: st.Speed.SwimWalk,
		Collision:     stats.Collision{Radius: t.Collision.Radius, Height: t.Collision.Height},
	}
}
