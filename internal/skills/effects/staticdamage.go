package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// staticDamage is Mobius StaticDamage: power damage, whatever the stats; a
// player caster is told the damage.
type staticDamage struct {
	skills.Base
	power int
}

func init() {
	register(data.EffectStaticDamage, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Int("power", 0)
		if err != nil {
			return nil, err
		}
		return &staticDamage{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*staticDamage) Instant() bool { return true }

func (e *staticDamage) OnStart(info *skills.BuffInfo) {
	effector, effected, sk := info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() {
		return
	}
	effected.ReduceCurrentHP(float64(e.power), effector, sk)
	effected.NotifyDamageReceived(float64(e.power), effector, sk, false, false)
	if isPlayer(effector) {
		effector.SendDamageMessage(effected, e.power, false, false, false)
	}
}
