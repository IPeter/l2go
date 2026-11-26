package effects

import (
	"errors"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// fishingCast is Mobius Fishing (Fishing, skill 1312): a player throws the
// bait; the world's skills.Fisher does the checks and starts the fishing.
type fishingCast struct{ skills.Base }

// fishingPull is Mobius Pumping and Reeling (skills 1313 and 1314): the
// skill's damage goes to the fish the player fights.
type fishingPull struct {
	skills.Base
	power float64
	reel  bool
}

func init() {
	register(data.EffectFishing, func(def *data.SkillEffect) (skills.Effect, error) {
		return &fishingCast{Base: skills.NewBase(def)}, nil
	})
	pull := func(reel bool) skills.Factory {
		return func(def *data.SkillEffect) (skills.Effect, error) {
			if _, ok := def.Params.Get("power"); !ok {
				return nil, errors.New("effect without power")
			}
			power, err := def.Params.Float("power", 0)
			if err != nil {
				return nil, err
			}
			return &fishingPull{Base: skills.NewBase(def), power: power, reel: reel}, nil
		}
	}
	register(data.EffectPumping, pull(false))
	register(data.EffectReeling, pull(true))
}

func (*fishingCast) Instant() bool { return true }

func (*fishingCast) Type() skills.EffectType { return skills.TypeFishingStart }

func (*fishingCast) OnStart(info *skills.BuffInfo) {
	effector := info.Effector()
	if !isPlayer(effector) {
		return
	}
	if f, ok := info.Env().World.(skills.Fisher); ok {
		f.StartFishing(effector)
	}
}

func (*fishingPull) Instant() bool { return true }

func (*fishingPull) Type() skills.EffectType { return skills.TypeFishing }

func (e *fishingPull) OnStart(info *skills.BuffInfo) {
	effector := info.Effector()
	if !isPlayer(effector) {
		return
	}
	f, ok := info.Env().World.(skills.Fisher)
	if !ok {
		return
	}
	if e.reel {
		f.Reel(effector, info.Skill(), e.power)
	} else {
		f.Pump(effector, info.Skill(), e.power)
	}
}
