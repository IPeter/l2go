package game

import "github.com/IPeter/l2go/internal/skills"

// Line of sight at Mobius's call sites (Plan 13, Phase 10.2). Most go
// through Server.canSee (Mobius canSeeTarget(WorldObject, WorldObject)):
// player attacks (doAttack, CANNOT_SEE_TARGET), player casts (useMagic),
// targets dropped when a cast lands, aggression, clan help, the monster
// skill targets (heal, resurrection, buffs) and thinkAttack's "cannot see,
// follow within 6,000".

// skillUnseen is the sight rule of Mobius Creature.reduceCurrentHp: a
// player takes no damage from a skill with a cast range, other than a damage
// over time, whose caster cannot see it.
func (s *Server) skillUnseen(p *Player, attacker Creature, sk *skills.Skill) bool {
	return p != nil && sk != nil && sk.CastRange > 0 && attacker != nil && !s.canSee(attacker, p)
}
