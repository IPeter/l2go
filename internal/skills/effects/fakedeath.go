package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// fakeDeath is Mobius FakeDeath, the toggle of the same name: the target
// fakes death; each tick costs power × the ticks multiplier MP, and the
// toggle ends at death or when the MP is short ("Your skill was removed
// due to a lack of MP."). At the end a player leaves fake death (recently
// faked) and everyone sees it stand up.
type fakeDeath struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectFakeDeath, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &fakeDeath{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*fakeDeath) Type() skills.EffectType { return skills.TypeFakeDeath }

func (*fakeDeath) OnStart(info *skills.BuffInfo) { info.Effected().StartFakeDeath() }

func (e *fakeDeath) OnActionTime(info *skills.BuffInfo) bool {
	effected, sk := info.Effected(), info.Skill()
	if effected.IsDead() {
		return false
	}
	manaDam := e.power * e.TicksMultiplier(info.Env())
	if manaDam > effected.CurrentMP() && sk.IsToggle() {
		skills.Send(effected, msgSkillRemovedLackOfMP)
		return false
	}
	reduceMP(effected, manaDam)
	return sk.IsToggle()
}

func (*fakeDeath) OnExit(info *skills.BuffInfo) {
	effected := info.Effected()
	if isPlayer(effected) {
		p := effected.AsPlayer()
		p.SetFakeDeath(false)
		p.SetRecentFakeDeath(true)
	}
	effected.BroadcastFakeDeathEnd()
}
