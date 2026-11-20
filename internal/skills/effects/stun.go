package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// stun is Mobius Stun: the STUNNED flag, which raids never get. It starts
// the stun (attack, cast and movement stop); when it ends, an NPC thinks
// again and a summon goes back to its owner, or attacks a caster that is
// still alive unless that caster is an unflagged playable.
type stun struct{ skills.Base }

func init() {
	register(data.EffectStun, func(def *data.SkillEffect) (skills.Effect, error) {
		return &stun{Base: skills.NewBase(def)}, nil
	})
}

func (*stun) Type() skills.EffectType { return skills.TypeStun }

func (*stun) Flags() skills.EffectFlags { return skills.FlagStunned.Mask() }

func (*stun) CanStart(info *skills.BuffInfo) bool {
	effected := info.Effected()
	return effected != nil && !effected.IsRaid()
}

func (*stun) OnStart(info *skills.BuffInfo) { startStunning(info.Effected()) }

func (*stun) OnExit(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effected == nil || effected.IsRaid() {
		return
	}
	// Creature.stopStunning(false).
	thinkUnlessPlayer(effected)
	effected.UpdateAbnormalEffect()
	if !isSummon(effected) {
		return
	}
	switch {
	case effector == nil || effector.IsDead():
		effected.NotifyAI(skills.AIFollowOwner, nil)
	case skills.IsPlayable(effector) && effected.AsPlayer() != nil && effected.AsPlayer().PvpFlag() == 0:
		effected.NotifyAI(skills.AIFollowOwner, nil)
	default:
		effected.NotifyAI(skills.AISummonAttack, effector)
	}
}
