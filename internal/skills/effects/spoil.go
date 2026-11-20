package effects

import (
	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// The system messages of Spoil and Distrust.
const (
	msgInvalidTarget                     sp.SystemMessageID = 109 // Invalid target.
	msgItHasAlreadyBeenSpoiled           sp.SystemMessageID = 357 // It has already been spoiled.
	msgTheSpoilConditionHasBeenActivated sp.SystemMessageID = 612 // The Spoil condition has been activated.
)

// spoil is Mobius Spoil: after the magic success roll, a living monster
// that is not spoiled yet is spoiled for the caster ("The Spoil condition
// has been activated.") and its AI learns it was attacked.
type spoil struct{ skills.Base }

func init() {
	register(data.EffectSpoil, func(def *data.SkillEffect) (skills.Effect, error) {
		return &spoil{Base: skills.NewBase(def)}, nil
	})
}

func (*spoil) Instant() bool { return true }

func (*spoil) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.MagicSuccess(info.Env(), info.Effector(), info.Effected(), info.Skill())
}

func (*spoil) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	att := effected.AsAttackable()
	if !isMonster(effected) || effected.IsDead() || att == nil {
		skills.Send(effector, msgInvalidTarget)
		return
	}
	if att.IsSpoiled() {
		skills.Send(effector, msgItHasAlreadyBeenSpoiled)
		return
	}
	att.SetSpoiler(effector.ObjectID())
	skills.Send(effector, msgTheSpoilConditionHasBeenActivated)
	effected.NotifyAI(skills.AIAttacked, effector)
}
