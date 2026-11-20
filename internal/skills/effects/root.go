package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// root is Mobius Root: the ROOTED flag. The creature stops moving and its
// AI is told; when it ends, an NPC thinks again.
type root struct{ skills.Base }

func init() {
	register(data.EffectRoot, func(def *data.SkillEffect) (skills.Effect, error) {
		return &root{Base: skills.NewBase(def)}, nil
	})
}

func (*root) Type() skills.EffectType { return skills.TypeRoot }

func (*root) Flags() skills.EffectFlags { return skills.FlagRooted.Mask() }

func (*root) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	effected.StopMove()
	effected.NotifyAI(skills.AIRooted, nil)
}

func (*root) OnExit(info *skills.BuffInfo) { thinkUnlessPlayer(info.Effected()) }
