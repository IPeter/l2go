package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// sweeper is Mobius Sweeper: a player sweeping an attackable it may sweep
// (checkSpoilOwner) whose spoil loot fits its inventory takes the loot; in
// a party each item goes by the party's loot rule, else to the player with
// the obtain message.
type sweeper struct{ skills.Base }

func init() {
	register(data.EffectSweeper, func(def *data.SkillEffect) (skills.Effect, error) {
		return &sweeper{Base: skills.NewBase(def)}, nil
	})
}

func (*sweeper) Instant() bool { return true }

func (*sweeper) OnStart(info *skills.BuffInfo) {
	env, effector, effected := info.Env(), info.Effector(), info.Effected()
	if effector == nil || effected == nil || !isPlayer(effector) || !skills.IsAttackable(effected) {
		return
	}
	p, monster := effector.AsPlayer(), effected.AsAttackable()
	if p == nil || monster == nil || !monster.CheckSpoilOwner(effector, false) {
		return
	}
	if !p.CheckInventorySlotsAndWeight(monster.SpoilLootItems(), false, false) {
		return
	}
	inParty := env.World != nil && env.World.Party(effector) != nil
	for _, item := range monster.TakeSweep() {
		if inParty {
			p.PartyDistributeItem(item, true, effected)
		} else {
			p.AddItem(item.ID, int(item.Count), 0, effected, true)
		}
	}
}
