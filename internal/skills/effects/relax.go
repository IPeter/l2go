package effects

import (
	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// msgSkillDeactivatedHPFull is "That skill has been de-activated as HP was
// fully recovered."; msgSkillRemovedLackOfMP (140) comes with Wave B.
const msgSkillDeactivatedHPFull sp.SystemMessageID = 175

// relax is Mobius Relax (the RELAXING flag and type): a player sits down,
// an NPC rests. Each tick costs power × the ticks multiplier MP; a toggle
// ends when the target dies or stands up, its HP is full ("That skill has
// been de-activated as HP was fully recovered.") or its MP is short ("Your
// skill was removed due to a lack of MP.").
type relax struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectRelax, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &relax{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*relax) Type() skills.EffectType { return skills.TypeRelaxing }

func (*relax) Flags() skills.EffectFlags { return skills.FlagRelaxing.Mask() }

func (*relax) OnStart(info *skills.BuffInfo) { sitOrRest(info.Effected()) }

func (e *relax) OnActionTime(info *skills.BuffInfo) bool {
	effected, sk := info.Effected(), info.Skill()
	if effected.IsDead() {
		return false
	}
	if isPlayer(effected) && !effected.AsPlayer().IsSitting() {
		return false
	}
	if effected.CurrentHP()+1 > float64(effected.MaxRecoverableHP()) && sk.IsToggle() {
		skills.Send(effected, msgSkillDeactivatedHPFull)
		return false
	}
	manaDam := e.power * e.TicksMultiplier(info.Env())
	if manaDam > effected.CurrentMP() && sk.IsToggle() {
		skills.Send(effected, msgSkillRemovedLackOfMP)
		return false
	}
	reduceMP(effected, manaDam)
	return sk.IsToggle()
}

// sitOrRest is the start of the resting effects: a player sits down
// (sitDown(false)), anything else rests (setIntentionRest).
func sitOrRest(c skills.Creature) {
	if isPlayer(c) {
		c.AsPlayer().SitDown(false)
	} else {
		c.NotifyAI(skills.AIRest, nil)
	}
}
