package game

import (
	"math"

	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/world"
)

// The summon AI: Mobius SummonAI on CreatureAI, with the NPC movement
// (npc_move.go), swing (attack.go) and cast (cast.go) of the NPC base. The
// monster AI's think slots never see a summon; the summon tick drives the
// follows (CreatureFollowTaskManager) and the avoid task.

// summonIntention is what a summon's AI does (Mobius Intention).
type summonIntention uint8

// The intentions a summon uses.
const (
	summonIdle summonIntention = iota
	summonActive
	summonFollowing
	summonAttacking
	summonCasting
	summonMovingTo
	summonPickingUp
)

// summonAI is a summon's AI state (Mobius SummonAI and the CreatureAI parts
// it uses).
type summonAI struct {
	intention summonIntention
	// startFollow is Mobius _startFollow: whether the summon goes back to
	// following its owner.
	startFollow bool
	// followTarget is what it follows; followRange the range (-1: a random
	// 50-100 each time).
	followTarget Creature
	followRange  int
	attackTarget Creature
	castSkill    *skills.Skill
	castTarget   Creature
	// lastAttack is the attack a cast interrupted (_lastAttack).
	lastAttack Creature
	// pickUp is the ground item a pet walks to.
	pickUp world.Object
	// avoiding is set while the avoid task runs; startAvoid when a hit asked
	// for a step aside.
	avoiding, startAvoid bool
	thinking             bool
}

// summonTick runs every 100 ms: the avoid task of each summon, the attack
// follow every 500 ms and the normal follow every second.
func (s *Server) summonTick() {
	f := &s.summons
	f.ticks++
	attack := f.ticks%5 == 0
	normal := f.ticks%10 == 0
	for _, sm := range append([]*Summon(nil), f.list...) {
		if sm.gone || !sm.npc.InWorld() || sm.npc.Dead {
			continue
		}
		s.summonAvoidTick(sm)
		switch sm.ai.intention {
		case summonAttacking, summonCasting, summonPickingUp:
			if attack {
				s.summonThink(sm)
			}
		case summonFollowing:
			if normal {
				s.summonFollowTick(sm)
			}
		}
	}
}

// summonFollowTick is CreatureFollowTaskManager.follow for a summon in the
// FOLLOW intention: within its range it waits; farther than 3,000 it gives
// up (idle, not following); otherwise it walks up to the target.
func (s *Server) summonFollowTick(sm *Summon) {
	t := sm.ai.followTarget
	if t == nil || !creatureInWorld(t) {
		sm.follow = false
		s.summonSetIdle(sm)
		return
	}
	rng := sm.ai.followRange
	if rng < 0 {
		rng = s.rnd.Between(summon.FollowMinRange, summon.FollowMaxRange)
	}
	n := sm.npc
	if within3D(n.Position(), t.Position(), rng) {
		return
	}
	if !within3D(n.Position(), t.Position(), summon.FollowGiveUpRange) {
		sm.follow = false
		s.summonSetIdle(sm)
		return
	}
	if !n.Running() {
		s.SetNpcRunning(n, true)
	}
	s.NpcMoveToPawn(n, t, int32(rng)) //nolint:gosec // small
}

// summonSetFollowStatus is Mobius Summon.setFollowStatus: following its
// owner, or idle.
func (s *Server) summonSetFollowStatus(sm *Summon, follow bool) {
	sm.follow = follow
	if follow {
		s.summonSetFollow(sm, sm.owner)
	} else {
		s.summonSetIdle(sm)
	}
}

// summonSetFollow is SummonAI.setIntentionFollow: the summon follows target.
func (s *Server) summonSetFollow(sm *Summon, target Creature) {
	if target == nil || sm.npc.Dead {
		return
	}
	sm.ai.avoiding = true
	s.npcClientStopMoving(sm.npc, nil)
	sm.ai.intention = summonFollowing
	sm.ai.followTarget, sm.ai.followRange = target, -1
	s.summonFollowTick(sm)
}

// summonStopFollow is AbstractAI.stopFollow.
func (s *Server) summonStopFollow(sm *Summon) {
	sm.ai.followTarget = nil
	if sm.ai.intention == summonFollowing {
		sm.ai.intention = summonActive
	}
}

// summonSetIdle is SummonAI.setIntentionIdle: no more following, then
// ACTIVE.
func (s *Server) summonSetIdle(sm *Summon) {
	s.summonStopFollow(sm)
	sm.ai.startFollow = false
	sm.ai.avoiding = false
	s.summonSetActive(sm)
}

// summonSetActive is SummonAI.setIntentionActive: back to following when
// startFollow is set, else ACTIVE (the summon stops and leaves its stance).
func (s *Server) summonSetActive(sm *Summon) {
	sm.ai.avoiding = true
	if sm.ai.startFollow {
		s.summonSetFollow(sm, sm.owner)
		return
	}
	if sm.ai.intention != summonActive {
		sm.ai.intention = summonActive
		sm.ai.attackTarget, sm.ai.castTarget = nil, nil
		s.npcClientStopMoving(sm.npc, nil)
	}
}

// summonSetAttack is SummonAI.setIntentionAttack: the summon fights target.
func (s *Server) summonSetAttack(sm *Summon, target Creature) {
	if target == nil || sm.npc.Dead {
		return
	}
	sm.ai.avoiding = false
	sm.ai.followTarget = nil
	sm.ai.intention = summonAttacking
	sm.ai.attackTarget = target
	sm.npc.ai.target = target
	s.summonThink(sm)
}

// summonSetCast is SummonAI.setIntentionCast: an attack in progress is
// remembered and goes on after the cast.
func (s *Server) summonSetCast(sm *Summon, sk *skills.Skill, target Creature) {
	if sm.ai.intention == summonAttacking {
		sm.ai.lastAttack = sm.ai.attackTarget
	} else {
		sm.ai.lastAttack = nil
	}
	sm.ai.avoiding = false
	sm.ai.followTarget = nil
	sm.ai.intention = summonCasting
	sm.ai.castSkill, sm.ai.castTarget = sk, target
	s.summonThink(sm)
}

// summonSetMoveTo is setIntentionMoveTo: the summon walks to pos.
func (s *Server) summonSetMoveTo(sm *Summon, pos world.Position) {
	sm.ai.followTarget = nil
	sm.ai.intention = summonMovingTo
	s.StopAutoAttack(sm.npc)
	s.NpcMoveTo(sm.npc, pos)
}

// summonThink is SummonAI.notifyActionThink: nothing while casting or with
// every skill disabled; otherwise the ATTACK, CAST or PICK_UP think.
func (s *Server) summonThink(sm *Summon) {
	n := sm.npc
	ai := &sm.ai
	if ai.thinking || n.Dead || !n.InWorld() || s.CastingNow(n) || (n.skill != nil && allSkillsDisabled(&n.skill.skillBase)) {
		return
	}
	ai.thinking = true
	defer func() { ai.thinking = false }()
	switch ai.intention {
	case summonAttacking:
		s.summonThinkAttack(sm)
	case summonCasting:
		s.summonThinkCast(sm)
	case summonPickingUp:
		s.petThinkPickUp(sm) // pet_items.go
	}
}

// summonThinkAttack is SummonAI.thinkAttack: a lost or dead target ends
// the attack; out of reach the summon runs to it; in reach it swings.
func (s *Server) summonThinkAttack(sm *Summon) {
	n, t := sm.npc, sm.ai.attackTarget
	if t == nil || t.IsDead() || !creatureInWorld(t) {
		sm.ai.attackTarget = nil
		s.summonSetActive(sm)
		return
	}
	if s.npcMaybeMoveToPawn(n, t, s.summonAttackRange(sm)) {
		return
	}
	s.npcClientStopMoving(n, nil)
	s.npcDoAttack(n, t)
}

// summonAttackRange is the summon's physical attack range (getPhysicalAttackRange,
// at least 36).
func (s *Server) summonAttackRange(sm *Summon) int {
	return max(sm.npc.Stats.PhysicalAttackRange(), summon.MinAttackRange)
}

// summonThinkCast is SummonAI.thinkCast: a lost target ends the cast; out of
// the skill's range the summon walks up; in range it stops following, goes
// idle (keeping startFollow) and casts.
func (s *Server) summonThinkCast(sm *Summon) {
	n, t, sk := sm.npc, sm.ai.castTarget, sm.ai.castSkill
	if t == nil || sk == nil || (sk.HasNegativeEffect() && sk.AffectRange > 0 && !s.canSee(n, t)) {
		sm.ai.castTarget = nil
		s.summonSetActive(sm)
		return
	}
	if s.npcMaybeMoveToPawn(n, t, sk.CastRange) {
		return
	}
	s.npcClientStopMoving(n, nil)
	val := sm.ai.startFollow
	sm.follow = false
	s.summonSetIdle(sm)
	sm.ai.startFollow = val
	n.ai.target = t
	s.DoCast(n, sk)
}

// summonSkillFinished is the OnSkillFinished hook: a summon that finished a
// cast attacks again what it attacked before, or goes back to following
// (SummonAI.notifyActionFinishCasting).
func (s *Server) summonSkillFinished(c Creature, _ *skills.Skill, _ Creature) {
	sm := summonOfNpc(c)
	if sm == nil || sm.npc.Dead {
		return
	}
	if sm.ai.lastAttack == nil {
		s.summonSetFollowStatus(sm, sm.ai.startFollow)
		return
	}
	t := sm.ai.lastAttack
	sm.ai.lastAttack = nil
	s.summonSetAttack(sm, t)
}

// summonArrived is notifyActionArrived for a summon: MOVE_TO turns ACTIVE,
// and the summon thinks.
func (s *Server) summonArrived(sm *Summon) {
	n := sm.npc
	n.Move = nil
	s.npcClientStoppedMoving(n)
	if sm.ai.intention == summonMovingTo {
		s.summonSetActive(sm)
	}
	s.summonThink(sm)
}

// summonAttackEvent is the OnNpcAttackEvent hook for summons: the end of a
// swing and a hit that could not land make it think; a refused swing turns
// it ACTIVE.
func (s *Server) summonAttackEvent(n *Npc, ev AttackEvent) {
	sm := n.summon
	if sm == nil {
		return
	}
	switch ev {
	case NpcAttackReady, NpcAttackCancelled:
		s.summonThink(sm)
	case NpcAttackRefused:
		s.summonSetActive(sm)
	}
}

// summonForget is the OnForget hook: a summon that loses sight of its
// target forgets it.
func (s *Server) summonForget(viewer, object world.Object) {
	sm := summonOfNpc(asCreature(viewer))
	if sm == nil {
		return
	}
	if sm.ai.attackTarget != nil && world.Object(sm.ai.attackTarget) == object {
		sm.ai.attackTarget = nil
		s.summonSetActive(sm)
	}
	if sm.ai.castTarget != nil && world.Object(sm.ai.castTarget) == object {
		sm.ai.castTarget = nil
		s.summonSetActive(sm)
	}
	if sm.npc.ai.target == object {
		sm.npc.ai.target = nil
	}
}

// asCreature returns o as a Creature, nil for anything else.
func asCreature(o world.Object) Creature {
	c, _ := o.(Creature)
	return c
}

// summonAvoidAttack is SummonAI.avoidAttack: hit near its owner by someone
// else, the summon steps aside at its next avoid tick.
func (s *Server) summonAvoidAttack(sm *Summon, attacker Creature) {
	owner := sm.owner
	if owner == nil || attacker == nil || attacker == Creature(owner) {
		return
	}
	if within3D(owner.Position(), sm.npc.Position(), 2*summon.AvoidRadius) {
		sm.ai.startAvoid = true
	}
}

// summonAvoidTick is SummonAI.run: a summon asked to step aside and not
// walking walks to a random point 70 from its owner, on the owner's side.
func (s *Server) summonAvoidTick(sm *Summon) {
	if !sm.ai.avoiding || !sm.ai.startAvoid {
		return
	}
	sm.ai.startAvoid = false
	n := sm.npc
	if n.Move != nil || n.Dead || npcMovementDisabled(n) {
		return
	}
	op, np := sm.owner.Position(), n.Position()
	angle := float64(s.rnd.Between(-90, 90))*math.Pi/180 + math.Atan2(float64(op.Y-np.Y), float64(op.X-np.X))
	to := world.Position{
		X: op.X + int32(summon.AvoidRadius*math.Cos(angle)),
		Y: op.Y + int32(summon.AvoidRadius*math.Sin(angle)),
		Z: np.Z,
	}
	if s.geo.CanMoveToTarget(geoPoint(np), geoPoint(to)) {
		s.NpcMoveTo(n, to)
	}
}

// summonNotifyAI is the summon's side of skillBase.NotifyAI: what effects
// tell a summon (Mobius SummonAI through the effects).
func (s *Server) summonNotifyAI(sm *Summon, ev skills.AIEvent, other Creature) {
	switch ev {
	case skills.AIFollowOwner:
		s.summonSetFollowStatus(sm, true)
	case skills.AISummonAttack, skills.AIAttack:
		if other != nil {
			s.summonSetAttack(sm, other)
		}
	case skills.AIIdle:
		s.summonSetIdle(sm)
	case skills.AIActive:
		s.summonSetActive(sm)
	case skills.AIAttacked:
		if other != nil {
			s.summonAvoidAttack(sm, other)
		}
	}
}
