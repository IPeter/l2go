package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// The manor's effects (Plan 14 M11k): Sow (Sowing, skill 2097) and
// Harvesting (Harvester, skill 2098). Both are instant; the world's
// skills.ManorWorld does the work (the seed, the odds, the crop); without
// one they do nothing.

// sow is Mobius Sow.
type sow struct{ skills.Base }

// harvesting is Mobius Harvesting.
type harvesting struct{ skills.Base }

func init() {
	register(data.EffectSow, func(def *data.SkillEffect) (skills.Effect, error) {
		return &sow{Base: skills.NewBase(def)}, nil
	})
	register(data.EffectHarvesting, func(def *data.SkillEffect) (skills.Effect, error) {
		return &harvesting{Base: skills.NewBase(def)}, nil
	})
}

func manorWorld(info *skills.BuffInfo) skills.ManorWorld {
	mw, _ := info.Env().World.(skills.ManorWorld)
	return mw
}

func (*sow) Instant() bool { return true }

// OnStart is Sow.onStart: a player sows on a monster.
func (*sow) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if !isPlayer(effector) || !isMonster(effected) {
		return
	}
	if mw := manorWorld(info); mw != nil {
		mw.Sow(effector, effected)
	}
}

func (*harvesting) Instant() bool { return true }

// OnStart is Harvesting.onStart: a player harvests a dead monster.
func (*harvesting) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if !isPlayer(effector) || !isMonster(effected) || !effected.IsDead() {
		return
	}
	if mw := manorWorld(info); mw != nil {
		mw.Harvest(effector, effected)
	}
}
