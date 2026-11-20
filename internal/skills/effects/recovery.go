package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// recovery is Mobius Recovery: a player's death penalty drops a level.
type recovery struct{ skills.Base }

func init() {
	register(data.EffectRecovery, func(def *data.SkillEffect) (skills.Effect, error) {
		return &recovery{Base: skills.NewBase(def)}, nil
	})
}

func (*recovery) Instant() bool { return true }

func (*recovery) OnStart(info *skills.BuffInfo) {
	if isPlayer(info.Effected()) {
		info.Effected().AsPlayer().ReduceDeathPenaltyBuffLevel()
	}
}
