package effects

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// rejuvenationSkill is the skill whose heal says "Rejuvenating HP." (4051).
const rejuvenationSkill = 4051

// healEffect is Mobius Heal: power plus sqrt(shot multiplier × M.Atk) for
// non-static skills, through the target's healEffect stat. A mystic player
// or a summon with spiritshots adds the skill's MP cost (× 2.4 blessed) and
// multiplies M.Atk by 2 (4 blessed); an NPC with spiritshots adds 2.4 × the
// MP cost and × 4; anyone else × 2 (× 4 blessed). The heal stops at the
// recoverable maximum; the target is told how much.
type healEffect struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectHeal, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &healEffect{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*healEffect) Type() skills.EffectType { return skills.TypeHeal }

func (*healEffect) Instant() bool { return true }

func (e *healEffect) OnStart(info *skills.BuffInfo) {
	effector, effected, sk := info.Effector(), info.Effected(), info.Skill()
	if effected == nil || effected.IsDead() || isDoor(effected) || effected.IsInvul() {
		return
	}
	amount := e.power
	staticShotBonus := 0.0
	mAtkMul := 1
	sps := sk.IsMagic() && effector.ChargedShot(skills.Spiritshot)
	bss := sk.IsMagic() && effector.ChargedShot(skills.BlessedSpiritshot)
	p := effector.AsPlayer()
	switch {
	case (sps || bss) && isPlayer(effector) && p.IsMageClass(), isSummon(effector):
		staticShotBonus = float64(sk.MPConsume)
		mAtkMul = 2
		if bss {
			mAtkMul = 4
			staticShotBonus *= 2.4
		}
	case (sps || bss) && isNpc(effector):
		staticShotBonus = 2.4 * float64(sk.MPConsume)
		mAtkMul = 4
	default:
		if bss {
			mAtkMul *= 4
		} else {
			mAtkMul++
		}
	}
	if !sk.IsStatic() {
		amount += staticShotBonus + math.Sqrt(float64(float64(mAtkMul)*effector.MAtk(effector, nil)))
		amount = effected.Calc(stats.HealEffect, amount, nil, nil)
	}
	amount = heal(amount, effected.MaxRecoverableHP(), effected.CurrentHP())
	if amount != 0 {
		effected.SetCurrentHP(amount + effected.CurrentHP())
	}
	if !isPlayer(effected) {
		return
	}
	switch {
	case sk.ID == rejuvenationSkill:
		skills.Send(effected, skills.MsgRejuvenatingHP)
	case isPlayer(effector) && !sameCreature(effector, effected):
		skills.Send(effected, skills.MsgS2HPRestoredByS1, sp.TextParam(effector.Name()), sp.NumberParam(javaInt(amount)))
	default:
		skills.Send(effected, skills.MsgS1HPRestored, sp.NumberParam(javaInt(amount)))
	}
}
