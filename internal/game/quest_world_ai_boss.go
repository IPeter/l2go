package game

import (
	"time"

	"github.com/IPeter/l2go/internal/scripting"
)

// The grand boss scripts' part of the AI API (Plan 13, Phase 10.7, on
// ScriptOutput): useMagic, the NPC's target, HP and MP, a minion's own
// respawn time, a skill's effect point and the player flags the boss
// scripts check.

var _ scripting.BossActions = (*ScriptOutput)(nil)

// NpcUseMagic is Mobius Attackable.useMagic.
func (o *ScriptOutput) NpcUseMagic(npc scripting.NpcRef, skill scripting.Skill) {
	s, n, ok := o.npc(npc)
	if !ok {
		return
	}
	if sk, found := s.Skill(skill.ID, skill.Level); found {
		s.NpcUseMagic(n, sk)
	}
}

// NpcTarget is Mobius getTarget.
func (o *ScriptOutput) NpcTarget(npc scripting.NpcRef) any {
	s, n, ok := o.anyNpc(npc)
	if !ok || n.skill == nil {
		return nil
	}
	if c := s.gameCreature(n.skill.Target()); c != nil {
		return scriptObject(c)
	}
	return nil
}

// NpcSetHPMP is Mobius setCurrentHpMp.
func (o *ScriptOutput) NpcSetHPMP(npc scripting.NpcRef, hp, mp float64) {
	if _, n, ok := o.npc(npc); ok {
		n.SetCurrentHPMP(hp, mp)
	}
}

// NpcMP returns the current and the maximum MP.
func (o *ScriptOutput) NpcMP(npc scripting.NpcRef) (mp, maxMP float64) {
	if n := npcOfRef(npc); n != nil {
		return n.CurMP, float64(n.MaxMP())
	}
	return 0, 0
}

// NpcMinionRespawn is Mobius getLeader().getMinionList().onMinionDie(minion,
// ms) for a dead minion: its respawn task starts again with d.
func (o *ScriptOutput) NpcMinionRespawn(npc scripting.NpcRef, d time.Duration) {
	s, m, ok := o.anyNpc(npc)
	if !ok || !m.Dead || d <= 0 {
		return
	}
	if l := m.boss.leader; l != nil && l.boss.minions != nil {
		s.onMinionDie(l.boss.minions, m, d)
	}
}

// NpcSetSpawnLocation is Mobius getSpawn().setLocation(loc): the spawn's
// point, or for an NPC without a spawn the home its AI returns to.
func (o *ScriptOutput) NpcSetSpawnLocation(npc scripting.NpcRef, loc scripting.Location) {
	_, n, ok := o.anyNpc(npc)
	if !ok {
		return
	}
	pos := worldPosition(loc)
	if sp := n.spawn; sp != nil {
		sp.point, sp.lastPoint = pos, pos
		return
	}
	n.ai.home, n.ai.hasHome = pos, true
}

// NpcSetIntentionCast is Mobius getAI().setIntentionCast(skill, target).
func (o *ScriptOutput) NpcSetIntentionCast(npc scripting.NpcRef, skill scripting.Skill, target any) {
	s, n, ok := o.npc(npc)
	t := creatureOfRef(target)
	if !ok || n.Dead || t == nil {
		return
	}
	if sk, found := s.Skill(skill.ID, skill.Level); found {
		s.NpcSetCast(n, sk, t)
	}
}

// SkillCastRange is skill.getCastRange(); -1 for a skill the data does not
// have.
func (o *ScriptOutput) SkillCastRange(skill scripting.Skill) int {
	if s := o.srv.Load(); s != nil {
		if sk, ok := s.Skill(skill.ID, skill.Level); ok {
			return int(sk.CastRange)
		}
	}
	return -1
}

// PlayerDie is Mobius doDie(killer) for a player.
func (o *ScriptOutput) PlayerDie(p scripting.PlayerRef, killer any) {
	s, pl, ok := o.player(p)
	if !ok || pl.IsDead() {
		return
	}
	k := creatureOfRef(killer)
	if k == nil {
		k = pl
	}
	s.Kill(pl, k)
}

// PlayerTeleportToTown is Mobius teleToLocation(TeleportWhereType.TOWN).
func (o *ScriptOutput) PlayerTeleportToTown(p scripting.PlayerRef) {
	if s, pl, ok := o.player(p); ok {
		if loc, found := s.restartLocation(pl); found {
			s.teleportAround(pl, loc, maxOffsetOnTeleport)
		}
	}
}

// SkillEffectPoint is skill.getEffectPoint(); 0 for a skill the data does
// not have.
func (o *ScriptOutput) SkillEffectPoint(skill scripting.Skill) int {
	if s := o.srv.Load(); s != nil {
		if sk, ok := s.Skill(skill.ID, skill.Level); ok {
			return sk.EffectPoint
		}
	}
	return 0
}

// PlayerIsGM is Mobius isGM.
func (o *ScriptOutput) PlayerIsGM(p scripting.PlayerRef) bool {
	pl := playerOfRef(p)
	return pl != nil && pl.IsGM()
}

// PlayerIsInvisible is Mobius isInvisible.
func (o *ScriptOutput) PlayerIsInvisible(p scripting.PlayerRef) bool {
	pl := playerOfRef(p)
	return pl != nil && pl.Invisible()
}

// PlayerIsHero is Mobius isHero.
func (o *ScriptOutput) PlayerIsHero(p scripting.PlayerRef) bool {
	pl := playerOfRef(p)
	return pl != nil && pl.IsHero()
}
