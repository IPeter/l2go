package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// betray is Mobius Betray: a player's skill makes a summon attack its own
// owner while it lasts (the BETRAYED flag); the summon then goes idle. Its
// type is DEBUFF.
type betray struct{ skills.Base }

func init() {
	register(data.EffectBetray, func(def *data.SkillEffect) (skills.Effect, error) {
		return &betray{Base: skills.NewBase(def)}, nil
	})
}

func (*betray) Type() skills.EffectType { return skills.TypeDebuff }

func (*betray) Flags() skills.EffectFlags { return skills.FlagBetrayed.Mask() }

func (*betray) CanStart(info *skills.BuffInfo) bool {
	return isPlayer(info.Effector()) && isSummon(info.Effected())
}

func (*betray) OnStart(info *skills.BuffInfo) {
	var owner skills.Creature
	if p := info.Effected().AsPlayer(); p != nil {
		owner = creatureOf(p)
	}
	info.Effected().NotifyAI(skills.AIAttack, owner)
}

func (*betray) OnExit(info *skills.BuffInfo) {
	info.Effected().NotifyAI(skills.AIIdle, nil)
}
