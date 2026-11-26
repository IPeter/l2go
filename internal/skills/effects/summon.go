package effects

import (
	"errors"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// The summoning handlers (Plan 14 M11f): Summon, SummonPet, SummonCubic,
// CubicMastery and Unsummon. The world's skills.SummonWorld makes the
// servitors, pets and cubics; without one they do nothing.

// summonServitor is Mobius Summon: a player effector without a summon gets
// a servitor of npcId.
type summonServitor struct {
	skills.Base
	p skills.ServitorParams
}

// summonPet is Mobius SummonPet: the pet of the collar a player used.
type summonPet struct{ skills.Base }

// summonCubic is Mobius SummonCubic: a cubic for the effected player.
type summonCubic struct {
	skills.Base
	p skills.CubicParams
}

// cubicMastery is Mobius CubicMastery: while it lasts, a player keeps up to
// cubicCount cubics.
type cubicMastery struct {
	skills.Base
	count int
}

// unsummon is Mobius Unsummon: a summon goes away, with a chance roll.
type unsummon struct {
	skills.Base
	chance int
}

// errSummonNoParams is Mobius's "Summon effect without parameters!".
var errSummonNoParams = errors.New("summon effect without parameters")

func init() {
	register(data.EffectSummon, func(def *data.SkillEffect) (skills.Effect, error) {
		if len(def.Params) == 0 {
			return nil, errSummonNoParams
		}
		npcID, err := def.Params.Int("npcId", 0)
		if err != nil {
			return nil, err
		}
		if _, ok := def.Params.Get("npcId"); !ok {
			return nil, errors.New("summon effect without npcId")
		}
		mult, err := def.Params.Float("expMultiplier", 1)
		if err != nil {
			return nil, err
		}
		p := skills.ServitorParams{NpcID: npcID, ExpMultiplier: float32(mult)}
		for _, f := range []struct {
			name string
			def  int
			v    *int
		}{
			{"consumeItemId", 0, &p.ConsumeItemID},
			{"consumeItemCount", 1, &p.ConsumeItemCount},
			{"consumeItemInterval", 0, &p.ConsumeItemInterval},
			{"lifeTime", 3600, &p.LifeTime},
		} {
			if *f.v, err = def.Params.Int(f.name, f.def); err != nil {
				return nil, err
			}
		}
		return &summonServitor{Base: skills.NewBase(def), p: p}, nil
	})
	register(data.EffectSummonPet, func(def *data.SkillEffect) (skills.Effect, error) {
		return &summonPet{Base: skills.NewBase(def)}, nil
	})
	register(data.EffectSummonCubic, func(def *data.SkillEffect) (skills.Effect, error) {
		var p skills.CubicParams
		var err error
		if p.ID, err = def.Params.Int("cubicId", -1); err != nil {
			return nil, err
		}
		skillLevel, err := def.Params.Int("cubicSkillLevel", 0)
		if err != nil {
			return nil, err
		}
		if p.Level, err = def.Params.Int("cubicLvl", skillLevel); err != nil {
			return nil, err
		}
		for _, f := range []struct {
			name string
			def  int
			v    *int
		}{
			{"cubicPower", 0, &p.Power},
			{"cubicDuration", 0, &p.Duration},
			{"cubicDelay", 0, &p.Delay},
			{"cubicMaxCount", -1, &p.MaxCount},
			{"cubicSkillChance", 0, &p.SkillChance},
		} {
			if *f.v, err = def.Params.Int(f.name, f.def); err != nil {
				return nil, err
			}
		}
		return &summonCubic{Base: skills.NewBase(def), p: p}, nil
	})
	register(data.EffectCubicMastery, func(def *data.SkillEffect) (skills.Effect, error) {
		n, err := def.Params.Int("cubicCount", 1)
		if err != nil {
			return nil, err
		}
		return &cubicMastery{Base: skills.NewBase(def), count: n}, nil
	})
	register(data.EffectUnsummon, func(def *data.SkillEffect) (skills.Effect, error) {
		n, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &unsummon{Base: skills.NewBase(def), chance: n}, nil
	})
}

// summonWorld returns the world's SummonWorld, nil without one.
func summonWorld(info *skills.BuffInfo) skills.SummonWorld {
	sw, _ := info.Env().World.(skills.SummonWorld)
	return sw
}

func (*summonServitor) Instant() bool { return true }

// OnStart is Mobius Summon.onStart: a player effector without a summon gets
// the servitor; the world checks the summon.
func (e *summonServitor) OnStart(info *skills.BuffInfo) {
	effector := info.Effector()
	if !isPlayer(effector) {
		return
	}
	if sw := summonWorld(info); sw != nil {
		sw.SummonServitor(effector, info.Skill(), e.p)
	}
}

// Type is SUMMON_PET.
func (*summonPet) Type() skills.EffectType { return skills.TypeSummonPet }

func (*summonPet) Instant() bool { return true }

// OnStart is Mobius SummonPet.onStart: both sides players, the effected
// alive.
func (*summonPet) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effector == nil || effected == nil || !isPlayer(effector) || !isPlayer(effected) || effected.IsAlikeDead() {
		return
	}
	if sw := summonWorld(info); sw != nil {
		sw.SummonPet(effector, effected, info.Skill())
	}
}

func (*summonCubic) Instant() bool { return true }

// OnStart is Mobius SummonCubic.onStart: a living player effected; a
// negative cubic ID does nothing.
func (e *summonCubic) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if effected == nil || !isPlayer(effected) || effected.IsAlikeDead() || e.p.ID < 0 {
		return
	}
	if sw := summonWorld(info); sw != nil {
		sw.SummonCubic(info.Effector(), effected, info.Skill(), e.p)
	}
}

// CanStart is Mobius CubicMastery.canStart: a player effected.
func (*cubicMastery) CanStart(info *skills.BuffInfo) bool {
	return info.Effector() != nil && isPlayer(info.Effected())
}

// OnStart raises the cubic limit.
func (e *cubicMastery) OnStart(info *skills.BuffInfo) {
	if sw := summonWorld(info); sw != nil {
		sw.SetMaxCubics(info.Effected(), e.count)
	}
}

// OnActionTime keeps a passive mastery.
func (*cubicMastery) OnActionTime(info *skills.BuffInfo) bool { return info.Skill().IsPassive() }

// OnExit puts the limit back to 1.
func (*cubicMastery) OnExit(info *skills.BuffInfo) {
	if sw := summonWorld(info); sw != nil {
		sw.SetMaxCubics(info.Effected(), 1)
	}
}

// CalcSuccess is Mobius Unsummon.calcSuccess: with a magic level of 0, or
// one at least the target's level − 9, chance × the attribute and trait
// bonuses beats 100 × nextDouble().
func (e *unsummon) CalcSuccess(info *skills.BuffInfo) bool {
	effector, effected, sk := info.Effector(), info.Effected(), info.Skill()
	if sk.MagicLevel > 0 && effected.Level()-9 > sk.MagicLevel {
		return false
	}
	chance := float64(e.chance) * skills.AttributeBonus(effector, effected, sk) * skills.GeneralTraitBonus(effector, effected, sk.Trait, false)
	return chance > info.Env().Rand.Float64()*100
}

func (*unsummon) Instant() bool { return true }

// OnStart sends a summon away.
func (*unsummon) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if !isSummon(effected) {
		return
	}
	if sw := summonWorld(info); sw != nil {
		sw.Unsummon(effected)
	}
}
