package effects

import (
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// consumeBody is Mobius ConsumeBody, the corpse skills: a dead NPC's corpse
// goes at once, or, for a skill with START effects, after the skill's hit
// and cool time scaled by the caster's speed (AtkSpd).
type consumeBody struct{ skills.Base }

func init() {
	register(data.EffectConsumeBody, func(def *data.SkillEffect) (skills.Effect, error) {
		return &consumeBody{Base: skills.NewBase(def)}, nil
	})
}

func (*consumeBody) Instant() bool { return true }

func (*consumeBody) OnStart(info *skills.BuffInfo) {
	effector, effected, sk := info.Effector(), info.Effected(), info.Skill()
	if effector == nil || effected == nil || !isNpc(effected) || !effected.IsDead() {
		return
	}
	if sk.HasEffects(data.ScopeStart) {
		ms := skills.AtkSpd(effector, sk, float64(sk.HitTime+sk.CoolTime))
		effected.Decay(time.Duration(ms) * time.Millisecond)
		return
	}
	effected.Decay(0)
}
