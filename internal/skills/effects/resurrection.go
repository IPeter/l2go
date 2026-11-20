package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// resurrection is Mobius Resurrection: a player caster asks the dead
// player, or the owner of the dead pet, to accept (ConfirmDlg; the answer
// revives with power percent of the lost experience back); any other caster
// revives the target at once with ResurrectRestorePercent of power.
type resurrection struct {
	skills.Base
	power int
}

func init() {
	register(data.EffectResurrection, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Int("power", 0)
		if err != nil {
			return nil, err
		}
		return &resurrection{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*resurrection) Type() skills.EffectType { return skills.TypeResurrection }

func (*resurrection) Instant() bool { return true }

func (e *resurrection) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if isPlayer(effector) {
		if p := effected.AsPlayer(); p != nil {
			p.ReviveRequest(effector, effected.InstanceType().Is(data.InstancePet), e.power)
		}
		return
	}
	effected.Revive(skills.ResurrectRestorePercent(float64(e.power), effector))
}
