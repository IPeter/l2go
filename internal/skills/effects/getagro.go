package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// getAgro is Mobius GetAgro (Aggression): after a Get(100) roll under
// chance, an attackable target attacks the caster, and the NPCs of its
// clans within its clan help range that can move hate the caster (1
// damage, 200 hate), attack it and run. With affectPlayers a player target
// attacks the caster too, and so do the players within range of it, other
// than the caster, that can move and share neither its party nor its clan.
type getAgro struct {
	skills.Base
	chance        int
	affectPlayers bool
	rng           int
}

func init() {
	register(data.EffectGetAgro, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		affect, err := def.Params.Bool("affectPlayers", false)
		if err != nil {
			return nil, err
		}
		rng, err := def.Params.Int("range", 0)
		if err != nil {
			return nil, err
		}
		return &getAgro{Base: skills.NewBase(def), chance: chance, affectPlayers: affect, rng: rng}, nil
	})
}

func (*getAgro) Type() skills.EffectType { return skills.TypeAggression }

func (*getAgro) Instant() bool { return true }

func (e *getAgro) CalcSuccess(info *skills.BuffInfo) bool {
	return info.Env().Rand.Get(100) < e.chance
}

func (e *getAgro) OnStart(info *skills.BuffInfo) {
	env, effector, effected := info.Env(), info.Effector(), info.Effected()
	if effected == nil {
		return
	}
	switch {
	case skills.IsAttackable(effected):
		effected.NotifyAI(skills.AIAttack, effector)
		att := effected.AsAttackable()
		if att == nil || env.World == nil {
			return
		}
		rng, ok := att.ClanHelpRange()
		if !ok {
			return
		}
		for _, nearby := range env.World.VisibleInRange(effected, rng) {
			na := nearby.AsAttackable()
			if !skills.IsAttackable(nearby) || na == nil {
				continue
			}
			if !nearby.IsMovementDisabled() && na.IsClan(att) {
				na.AddDamageHate(effector, 1, 200)
				nearby.NotifyAI(skills.AIAttack, effector)
				nearby.SetRunning()
			}
		}
	case e.affectPlayers && isPlayer(effected):
		effected.NotifyAI(skills.AIAttack, effector)
		if e.rng <= 0 || !isPlayer(effector) || env.World == nil {
			return
		}
		casterClan := effector.AsPlayer().ClanID()
		for _, nearby := range env.World.VisibleInRange(effected, e.rng) {
			if !isPlayer(nearby) || sameCreature(nearby, effector) || nearby.IsMovementDisabled() {
				continue
			}
			sameParty := env.World.SameParty(nearby, effector)
			clan := nearby.AsPlayer().ClanID()
			sameClan := clan != 0 && casterClan != 0 && clan == casterClan
			if !sameParty && !sameClan {
				nearby.NotifyAI(skills.AIAttack, effector)
			}
		}
	}
}
