package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// resurrectionSpecial is Mobius ResurrectionSpecial (a resurrection the
// target carries): when it ends, a player outside the Olympiad is asked to
// accept a resurrection by the caster's player, and a pet's owner one for
// the pet, with power percent.
type resurrectionSpecial struct {
	skills.Base
	power int
}

func init() {
	register(data.EffectResurrectionSpecial, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Int("power", 0)
		if err != nil {
			return nil, err
		}
		return &resurrectionSpecial{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*resurrectionSpecial) Type() skills.EffectType { return skills.TypeResurrectionSpecial }

func (*resurrectionSpecial) Flags() skills.EffectFlags {
	return skills.FlagResurrectionSpecial.Mask()
}

func (e *resurrectionSpecial) OnExit(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	pet := effected.InstanceType().Is(data.InstancePet)
	if !isPlayer(effected) && !pet {
		return
	}
	p := effected.AsPlayer()
	if p == nil || p.InOlympiad() {
		return
	}
	if isPlayer(effected) {
		var caster skills.Creature
		if cp := effector.AsPlayer(); cp != nil {
			caster = creatureOf(cp)
		}
		p.ReviveRequest(caster, false, e.power)
	}
	if pet {
		p.ReviveRequest(creatureOf(p), true, e.power)
	}
}
