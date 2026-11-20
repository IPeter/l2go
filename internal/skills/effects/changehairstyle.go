package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// changeHairStyle is Mobius ChangeHairStyle: a player casting on a living
// player gets the hair style value. As in Mobius, the caster's appearance
// changes, not the target's.
type changeHairStyle struct {
	skills.Base
	value int
}

func init() {
	register(data.EffectChangeHairStyle, func(def *data.SkillEffect) (skills.Effect, error) {
		v, err := def.Params.Int("value", 0)
		if err != nil {
			return nil, err
		}
		return &changeHairStyle{Base: skills.NewBase(def), value: v}, nil
	})
}

func (*changeHairStyle) Instant() bool { return true }

func (e *changeHairStyle) OnStart(info *skills.BuffInfo) {
	if p := appearanceChanger(info); p != nil {
		p.SetHairStyle(e.value)
		p.BroadcastUserInfo()
	}
}

// appearanceChanger is the check of the appearance effects: the caster's
// player when a player casts on a player that is not alike dead, else nil.
func appearanceChanger(info *skills.BuffInfo) skills.Player {
	effector, effected := info.Effector(), info.Effected()
	if effector == nil || effected == nil || !isPlayer(effector) || !isPlayer(effected) || effected.IsAlikeDead() {
		return nil
	}
	return effector.AsPlayer()
}
