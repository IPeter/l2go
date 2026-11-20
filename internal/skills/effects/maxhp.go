package effects

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// maxHp is Mobius MaxHp: while it lasts, the maximum HP gains power (type
// DIFF, the default) or power percent (type PER), as an order-1 function of
// its own. With heal, the HP rises by the same amount and the effected is
// told ("S1 HP has been restored.", or by whom). When it ends, HP above the
// new maximum is cut.
type maxHp struct {
	skills.Base
	per   bool
	power float64
	heal  bool
}

func init() {
	register(data.EffectMaxHp, func(def *data.SkillEffect) (skills.Effect, error) {
		var per bool
		switch t := def.Params.String("type", "DIFF"); t {
		case "DIFF":
		case "PER":
			per = true
		default:
			return nil, fmt.Errorf("type %q: want DIFF or PER", t)
		}
		power, err := def.Params.Int("power", 0)
		if err != nil {
			return nil, err
		}
		heal, err := def.Params.Bool("heal", false)
		if err != nil {
			return nil, err
		}
		e := &maxHp{Base: skills.NewBase(def), per: per, power: float64(power), heal: heal}
		if per {
			e.power = 1 + float64(power)/100.0
		}
		return e, nil
	})
}

// owner is the owner of the effect's function on a creature: Mobius owns it
// by the effect object, which a creature holds once per skill level.
func (*maxHp) owner(sk *skills.Skill) stats.Owner {
	return stats.Owner{Kind: stats.OwnerEffect, ID: -int64(sk.ReuseHash())}
}

func (e *maxHp) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	currentHP := effected.CurrentHP()
	amount := e.power
	m := stats.Modifier{Stat: stats.MaxHP, Op: stats.OpAdd, Order: 1, Value: e.power, Owner: e.owner(info.Skill())}
	if e.per {
		maxHP := float64(effected.MaxHP())
		m.Op = stats.OpMul
		_ = effected.AddModifiers(m)
		if e.heal {
			amount = float64((e.power - 1) * maxHP)
			effected.SetCurrentHP(currentHP + amount)
		}
	} else {
		_ = effected.AddModifiers(m)
		if e.heal {
			effected.SetCurrentHP(currentHP + e.power)
		}
	}
	if !e.heal {
		return
	}
	if effector != nil && !sameCreature(effector, effected) {
		skills.Send(effected, skills.MsgS2HPRestoredByS1, sp.TextParam(effector.Name()), sp.NumberParam(javaInt(amount)))
	} else {
		skills.Send(effected, skills.MsgS1HPRestored, sp.NumberParam(javaInt(amount)))
	}
}

func (e *maxHp) OnExit(info *skills.BuffInfo) {
	effected := info.Effected()
	effected.RemoveOwner(e.owner(info.Skill()))
	if maxHP := float64(effected.MaxHP()); effected.CurrentHP() > maxHP {
		effected.SetCurrentHP(maxHP)
	}
}
