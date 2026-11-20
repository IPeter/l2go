package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// fear is Mobius Fear: the FEAR flag on players, summons and attackables
// other than siege defenders, siege flags and siege weapons. A cast that can
// still be aborted is, and the AI runs from the caster; it is told again
// every 5 ticks, whatever the data says.
type fear struct{ skills.Base }

func init() {
	register(data.EffectFear, func(def *data.SkillEffect) (skills.Effect, error) {
		return &fear{Base: skills.NewBase(def)}, nil
	})
}

func (*fear) Type() skills.EffectType { return skills.TypeFear }

func (*fear) Flags() skills.EffectFlags { return skills.FlagFear.Mask() }

// Ticks is Fear.getTicks: always 5.
func (*fear) Ticks() int { return 5 }

func (*fear) CanStart(info *skills.BuffInfo) bool {
	effected := info.Effected()
	if isPlayer(effected) || isSummon(effected) {
		return true
	}
	t := effected.InstanceType()
	return skills.IsAttackable(effected) && !t.Is(data.InstanceDefender) && !t.Is(data.InstanceSiegeFlag) &&
		effected.Race() != data.RaceSiegeWeapon
}

func (*fear) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if effected.IsCastingNow() && effected.CanAbortCast() {
		effected.AbortCast()
	}
	effected.NotifyAI(skills.AIAfraidStart, info.Effector())
}

func (*fear) OnActionTime(info *skills.BuffInfo) bool {
	info.Effected().NotifyAI(skills.AIAfraid, info.Effector())
	return false
}
