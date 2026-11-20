package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// dispelAll is Mobius DispelAll: every buff, dance, toggle and debuff of
// the target ends (stopAllEffects).
//
// Mobius's custom CancelReturn (off by default, W16) is not ported: the
// effects never come back.
type dispelAll struct{ skills.Base }

func init() {
	register(data.EffectDispelAll, func(def *data.SkillEffect) (skills.Effect, error) {
		return &dispelAll{Base: skills.NewBase(def)}, nil
	})
}

func (*dispelAll) Type() skills.EffectType { return skills.TypeDispel }

func (*dispelAll) Instant() bool { return true }

func (*dispelAll) OnStart(info *skills.BuffInfo) {
	if effected := info.Effected(); effected != nil {
		effected.Effects().StopAll()
	}
}
