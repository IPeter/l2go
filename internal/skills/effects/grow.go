package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// grow is Mobius Grow: while it lasts an NPC has its template's grown
// collision radius. Its type is BUFF.
type grow struct{ skills.Base }

func init() {
	register(data.EffectGrow, func(def *data.SkillEffect) (skills.Effect, error) {
		return &grow{Base: skills.NewBase(def)}, nil
	})
}

func (*grow) Type() skills.EffectType { return skills.TypeBuff }

func (*grow) OnStart(info *skills.BuffInfo) {
	if isNpc(info.Effected()) {
		info.Effected().SetCollisionGrown(true)
	}
}

func (*grow) OnExit(info *skills.BuffInfo) {
	if isNpc(info.Effected()) {
		info.Effected().SetCollisionGrown(false)
	}
}
