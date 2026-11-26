package scripttest

import (
	"context"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
)

// The fake world of the grand boss scripts (Plan 13, Phase 10.7): useMagic,
// the NPC's target, HP and MP, a minion's own respawn, skill effect points
// and the player flags. Tests set Npc.Target, MP, MaxMP, Player.GM,
// Invisible, Hero and SetSkillEffectPoint, and read what scripts did from
// World.Actions (kinds use_magic, hp_mp, minion_respawn).

var _ scripting.BossActions = (*World)(nil)

// NpcUseMagic records the cast at the NPC's Target (action use_magic, Value
// the skill ID, Level its level); the NPC's intention becomes CAST.
func (w *World) NpcUseMagic(npc scripting.NpcRef, skill scripting.Skill) {
	a := Action{Kind: "use_magic", Object: npc.ObjectID(), Value: skill.ID, Level: skill.Level}
	if n := npcOf(npc); n != nil {
		a.Target = objectID(refOf(n.Target))
		n.Intention = "CAST"
	}
	w.act(a)
}

// NpcTarget returns Target.
func (w *World) NpcTarget(npc scripting.NpcRef) any {
	if n := npcOf(npc); n != nil {
		return refOf(n.Target)
	}
	return nil
}

// NpcSetHPMP sets HP and MP, at most MaxHP and MaxMP when those are set
// (action hp_mp, Value the HP).
func (w *World) NpcSetHPMP(npc scripting.NpcRef, hp, mp float64) {
	if n := npcOf(npc); n != nil {
		if n.MaxHP > 0 {
			hp = min(hp, n.MaxHP)
		}
		if n.MaxMP > 0 {
			mp = min(mp, n.MaxMP)
		}
		n.HP, n.MP = hp, mp
	}
	w.act(Action{Kind: "hp_mp", Object: npc.ObjectID(), Value: int(hp)})
}

// NpcMP returns MP and MaxMP.
func (w *World) NpcMP(npc scripting.NpcRef) (mp, maxMP float64) {
	if n := npcOf(npc); n != nil {
		return n.MP, n.MaxMP
	}
	return 0, 0
}

// NpcMinionRespawn records the respawn time (action minion_respawn, Value
// the milliseconds).
func (w *World) NpcMinionRespawn(npc scripting.NpcRef, d time.Duration) {
	w.act(Action{Kind: "minion_respawn", Object: npc.ObjectID(), Value: int(d / time.Millisecond)})
}

// NpcSetSpawnLocation sets Spawn.
func (w *World) NpcSetSpawnLocation(npc scripting.NpcRef, loc scripting.Location) {
	if n := npcOf(npc); n != nil {
		l := loc
		n.Spawn = &l
	}
	w.act(Action{Kind: "spawn_location", Object: npc.ObjectID(), Loc: loc})
}

// NpcSetIntentionCast records the cast (action cast_intention, Value the
// skill ID, Level its level, Target the target); the intention becomes
// CAST.
func (w *World) NpcSetIntentionCast(npc scripting.NpcRef, skill scripting.Skill, target any) {
	if n := npcOf(npc); n != nil {
		n.Intention = "CAST"
	}
	w.act(Action{Kind: "cast_intention", Object: npc.ObjectID(), Target: objectID(target), Value: skill.ID, Level: skill.Level})
}

// SetSkillCastRange sets a skill's cast range; others are 40.
func (w *World) SetSkillCastRange(skillID, castRange int) {
	if w.castRanges == nil {
		w.castRanges = make(map[int]int)
	}
	w.castRanges[skillID] = castRange
}

// SkillCastRange returns the range SetSkillCastRange set, else 40.
func (w *World) SkillCastRange(s scripting.Skill) int {
	if r, ok := w.castRanges[s.ID]; ok {
		return r
	}
	return 40
}

// PlayerDie sets Dead (action player_die, Target the killer).
func (w *World) PlayerDie(p scripting.PlayerRef, killer any) {
	if pl := playerOf(p); pl != nil {
		pl.Dead = true
	}
	w.act(Action{Kind: "player_die", Object: p.ObjectID(), Target: objectID(killer)})
}

// PlayerTeleportToTown records it (action town).
func (w *World) PlayerTeleportToTown(p scripting.PlayerRef) {
	w.act(Action{Kind: "town", Object: p.ObjectID()})
}

// SetSkillEffectPoint sets a skill's effect point; others are 0.
func (w *World) SetSkillEffectPoint(skillID, point int) {
	if w.effectPoints == nil {
		w.effectPoints = make(map[int]int)
	}
	w.effectPoints[skillID] = point
}

// SkillEffectPoint returns the point SetSkillEffectPoint set.
func (w *World) SkillEffectPoint(s scripting.Skill) int { return w.effectPoints[s.ID] }

// PlayerIsGM reads GM.
func (w *World) PlayerIsGM(p scripting.PlayerRef) bool {
	pl := playerOf(p)
	return pl != nil && pl.GM
}

// PlayerIsInvisible reads Invisible.
func (w *World) PlayerIsInvisible(p scripting.PlayerRef) bool {
	pl := playerOf(p)
	return pl != nil && pl.Invisible
}

// AggroRangeEnter runs the aggro-range handlers: player (or, with
// isSummon, its summon) came within npc's aggro range.
func (h *Harness) AggroRangeEnter(npc *Npc, player *Player, isSummon bool) {
	h.t.Helper()
	h.call("aggro-range-enter", func(ctx context.Context) error {
		return h.engine.AggroRangeEnter(ctx, npc.Ref(), player.Ref(), isSummon)
	}, npc, player)
}

// SpellFinished runs the spell-finished handlers: npc finished casting
// skill at player.
func (h *Harness) SpellFinished(npc *Npc, player *Player, skill scripting.Skill) {
	h.t.Helper()
	h.call("spell-finished", func(ctx context.Context) error {
		return h.engine.SpellFinished(ctx, npc.Ref(), player.Ref(), skill)
	}, npc, player)
}

// PlayerIsHero reads Hero.
func (w *World) PlayerIsHero(p scripting.PlayerRef) bool {
	pl := playerOf(p)
	return pl != nil && pl.Hero
}
