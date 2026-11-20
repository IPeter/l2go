package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// sleep is Mobius Sleep: the SLEEP flag. The attack, cast and movement stop
// and the AI is told; when it ends, a player's AI becomes active and an
// NPC's thinks again. Damage wakes it through the skill's removedOnDamage.
type sleep struct{ skills.Base }

func init() {
	register(data.EffectSleep, func(def *data.SkillEffect) (skills.Effect, error) {
		return &sleep{Base: skills.NewBase(def)}, nil
	})
}

func (*sleep) Type() skills.EffectType { return skills.TypeSleep }

func (*sleep) Flags() skills.EffectFlags { return skills.FlagSleep.Mask() }

func (*sleep) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	effected.AbortAttack()
	effected.AbortCast()
	effected.StopMove()
	effected.NotifyAI(skills.AISleeping, nil)
}

func (*sleep) OnExit(info *skills.BuffInfo) {
	effected := info.Effected()
	if isPlayer(effected) {
		effected.NotifyAI(skills.AIActive, nil)
	} else {
		effected.NotifyAI(skills.AIThink, nil)
	}
}
