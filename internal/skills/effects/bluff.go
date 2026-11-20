package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// bluffHeadquarters is the siege headquarters NPC, which Bluff never turns.
const bluffHeadquarters = 35062

// bluff is Mobius Bluff: after its probability roll the target (not a
// headquarters or a raid) turns to the caster's heading, with the rotation
// packets.
type bluff struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectBluff, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &bluff{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (*bluff) Instant() bool { return true }

func (e *bluff) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.Probability(info.Env(), info.Effector(), info.Effected(), info.Skill(), float64(e.chance))
}

func (*bluff) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	// IsRaid covers raid minions (Mobius isRaid || isRaidMinion).
	if effected.NpcID() == bluffHeadquarters || effected.IsRaid() {
		return
	}
	effected.BroadcastStartRotation(effected.Heading(), 1, 65535)
	effected.BroadcastStopRotation(effector.Heading(), 65535)
	effected.SetHeading(effector.Heading())
}
