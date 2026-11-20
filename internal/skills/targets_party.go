package skills

import "github.com/IPeter/l2go/internal/data"

// The party target types of M9 (Plan 12 Phase 3) besides PARTY,
// PARTY_MEMBER and PARTY_NOTME (targets.go).
func init() {
	targetHandlers[data.TargetPartyOther] = targetPartyOther
	targetHandlers[data.TargetTargetParty] = targetTargetParty
}

// The force buffs PARTY_OTHER gives only to fighters (426) or only to
// mystics (427).
const (
	skillForceBuffFighter = 426
	skillForceBuffMystic  = 427
)

// targetPartyOther is PARTY_OTHER: another living member of the caster's
// party; the force buffs 426 and 427 only reach fighters and mystics. Any
// other target is incorrect.
func targetPartyOther(env *Env, s *Skill, caster Creature, _ bool, target Creature) []Creature {
	if target == nil || same(target, caster) || !inParty(env, caster) || !inParty(env, target) || !sameParty(env, caster, target) {
		return incorrectTarget(caster)
	}
	if target.IsDead() {
		return nil
	}
	if tp := asPlayer(target); tp != nil {
		switch s.ID {
		case skillForceBuffFighter:
			if tp.IsMageClass() {
				return nil
			}
		case skillForceBuffMystic:
			if !tp.IsMageClass() {
				return nil
			}
		}
	}
	return []Creature{target}
}

// targetTargetParty is TARGET_PARTY: the target's party, every member and
// summon within the affect range of the target, or the target alone when it
// has no party. A target that is not a player gives nothing (Mobius fails
// on it).
func targetTargetParty(env *Env, s *Skill, caster Creature, _ bool, target Creature) []Creature {
	if target == nil || target.IsDead() || same(target, caster) {
		return incorrectTarget(caster)
	}
	if !isPlayer(target) {
		return nil
	}
	if !inParty(env, target) {
		return []Creature{target}
	}
	var list []Creature
	radius := s.AffectRange
	for _, m := range env.World.Party(target) {
		if m == nil {
			continue
		}
		if addCharacter(target, m, radius, false) {
			list = append(list, m)
		}
		if sm := summonOf(env, m); addCharacter(target, sm, radius, false) {
			list = append(list, sm)
		}
	}
	return list
}
