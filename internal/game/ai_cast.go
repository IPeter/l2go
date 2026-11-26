package game

import (
	"github.com/IPeter/l2go/internal/npcai"
	"github.com/IPeter/l2go/internal/skills"
)

// The CAST intention of the monster AI (Mobius AttackableAI.setIntentionCast,
// CreatureAI.changeIntentionToCast and AttackableAI.thinkCast): scripts tell
// an NPC to cast a skill at a target; it walks into range and casts.

// NpcSetCast gives n the CAST intention with sk at target (Mobius
// getAI().setIntention(CAST, skill, target)): the NPC takes target as its
// target; after a swing in progress it thinks at once, and every second until
// it casts. Loop-only.
func (s *Server) NpcSetCast(n *Npc, sk *skills.Skill, target Creature) {
	if n == nil || sk == nil || n.class == nil || !n.class.attackable {
		return
	}
	ai := s.aiOf(n)
	if target != nil {
		ai.target = target
	}
	// CreatureAI.setIntentionCast waits for the swing in progress, so its
	// hit does not land during the cast.
	if st := s.attack.states[n]; st != nil && st.end.After(s.Now()) {
		s.loop.After(st.end.Sub(s.Now()), func() {
			if n.InWorld() && !n.Dead {
				s.npcChangeToCast(n, sk, target)
			}
		})
	} else {
		s.npcChangeToCast(n, sk, target)
	}
	s.startAITask(n)
}

// NpcUseMagic is Mobius Attackable.useMagic, which scripts use to make a
// monster cast (Queen Ant's nurses): when n is alive, not casting and can
// cast sk now (not passive, not disabled, enough MP and HP, not muted for
// it), it takes the CAST intention at the first target of the skill's list.
// Loop-only.
func (s *Server) NpcUseMagic(n *Npc, sk *skills.Skill) {
	if n == nil || sk == nil || n.Dead || n.skill == nil || sk.IsPassive() || s.CastingNow(n) || s.SkillDisabled(n, sk) {
		return
	}
	b := &n.skill.skillBase
	env := s.skills.env
	if b.CurrentMP() < float64(skills.MPConsume(env, b.self, sk)+skills.MPInitialConsume(b.self, sk)) ||
		b.CurrentHP() <= float64(sk.HPConsume) {
		return
	}
	if !sk.IsStatic() {
		if sk.IsMagic() {
			if b.el.IsAffected(skills.FlagMuted) {
				return
			}
		} else if b.el.IsAffected(skills.FlagPhysicalMuted) {
			return
		}
	}
	targets := sk.Targets(env, b.self, true, b.Target())
	if len(targets) == 0 {
		return
	}
	if t := s.gameCreature(targets[0]); t != nil {
		s.NpcSetCast(n, sk, t)
	}
}

// npcChangeToCast is Mobius changeIntentionToCast.
func (s *Server) npcChangeToCast(n *Npc, sk *skills.Skill, target Creature) {
	ai := s.aiOf(n)
	ai.x.castTarget = target
	ai.x.castSkill = sk
	ai.intention = NpcCast
	s.npcThink(n)
}

// npcThinkCast is Mobius AttackableAI.thinkCast: a lost target ends the
// intention (ACTIVE); out of the skill's range the NPC walks up to the
// target; in range it stops, turns ACTIVE and casts.
func (s *Server) npcThinkCast(n *Npc) {
	ai := &n.ai
	target, sk := ai.x.castTarget, ai.x.castSkill
	if s.npcCastTargetLost(n, target, sk) {
		ai.x.castTarget = nil
		return
	}
	if s.npcMaybeMoveToPawn(n, target, sk.CastRange) {
		return
	}
	s.npcClientStopMoving(n, nil)
	s.npcSetActive(n)
	if fn := s.ai.x.doCast; fn != nil {
		fn(n, sk)
		return
	}
	s.DoCast(n, sk)
}

// npcCastTargetLost is Mobius CreatureAI.checkTargetLost for an NPC: no
// target, or out of sight for a harmful skill with an affect range, ends the
// intention.
func (s *Server) npcCastTargetLost(n *Npc, target Creature, sk *skills.Skill) bool {
	if target == nil || sk == nil {
		s.npcSetActive(n)
		return true
	}
	if sk.HasNegativeEffect() && sk.AffectRange > 0 && !s.canSee(n, target) {
		s.npcSetActive(n)
		return true
	}
	return false
}

// npcMaybeMoveToPawn is Mobius CreatureAI.maybeMoveToPawn for an NPC: when
// target is farther than offset plus both collision radii (2D), the NPC runs
// up to it and the think ends; an NPC that cannot move stays (and drops an
// ATTACK intention). Mobius follows the target; the NPC here walks to it
// again on every think.
func (s *Server) npcMaybeMoveToPawn(n *Npc, target Creature, offset int) bool {
	if target == nil || offset < 0 {
		return false
	}
	reach := offset + int(npcCollision(n)) + attackCollision(target)
	if within2D(n.Position(), target.Position(), reach) {
		return false
	}
	if npcMovementDisabled(n) || !(n.Stats.MoveSpeed() > 0) {
		if n.ai.intention == NpcAttack {
			s.npcSetIdle(n)
		}
		return true
	}
	if !n.Running() {
		s.SetNpcRunning(n, true)
	}
	if creatureMoving(target) {
		offset -= 100
	}
	offset = max(offset, 5)
	s.NpcMoveToPawn(n, target, int32(offset)) //nolint:gosec // skill ranges are small
	return true
}

// monsterDropsCast is Monster.doCast(skill, target, targets): a monster
// whose target is a player drops a skill without a negative effect. Mobius
// overrides only that form of doCast, which DoCastAt is; the AI's own casts
// use doCast(skill) and are never dropped.
func (s *Server) monsterDropsCast(c Creature, sk *skills.Skill) bool {
	n, ok := c.(*Npc)
	return ok && n.IsMonster() && npcai.DropsCast(sk, aiObjectView(n.ai.target))
}
