package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// callPc is Mobius CallPc (Summon Friend): a player caster asks the target
// to come, at the price of itemCount of itemID; the world's
// skills.Summoner does the checks, the ConfirmDlg and the teleport.
type callPc struct {
	skills.Base
	itemID, itemCount int
}

// callParty is Mobius CallParty: the caster's party members come to it.
type callParty struct{ skills.Base }

func init() {
	register(data.EffectCallPc, func(def *data.SkillEffect) (skills.Effect, error) {
		id, err := def.Params.Int("itemId", 0)
		if err != nil {
			return nil, err
		}
		count, err := def.Params.Int("itemCount", 0)
		if err != nil {
			return nil, err
		}
		return &callPc{Base: skills.NewBase(def), itemID: id, itemCount: count}, nil
	})
	register(data.EffectCallParty, func(def *data.SkillEffect) (skills.Effect, error) {
		return &callParty{Base: skills.NewBase(def)}, nil
	})
}

func (*callPc) Instant() bool { return true }

func (e *callPc) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effector == nil || effected == nil || sameCreature(effector, effected) {
		return
	}
	if s, ok := info.Env().World.(skills.Summoner); ok {
		s.CallPc(effector, effected, e.itemID, e.itemCount)
	}
}

func (*callParty) Instant() bool { return true }

func (*callParty) OnStart(info *skills.BuffInfo) {
	effector := info.Effector()
	if effector == nil {
		return
	}
	if s, ok := info.Env().World.(skills.Summoner); ok {
		s.CallParty(effector)
	}
}
