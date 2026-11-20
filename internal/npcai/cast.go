package npcai

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// The target types cast() and its helpers tell apart.
func isAura(tt data.TargetType) bool {
	return tt == data.TargetAura || tt == data.TargetBehindAura || tt == data.TargetFrontAura
}

// isAuraOrCorpse adds AURA_CORPSE_MOB to isAura.
func isAuraOrCorpse(tt data.TargetType) bool { return isAura(tt) || tt == data.TargetAuraCorpseMob }

func isArea(tt data.TargetType) bool {
	return tt == data.TargetArea || tt == data.TargetBehindArea || tt == data.TargetFrontArea
}

// The effect types of cast()'s branches.
var (
	dispelTypes  = []skills.EffectType{skills.TypeDispel, skills.TypeDispelBySlot}
	damageTypes  = []skills.EffectType{skills.TypePhysicalAttack, skills.TypePhysicalAttackHPLink, skills.TypeMagicalAttack, skills.TypeDeathLink, skills.TypeHPDrain}
	disableTypes = []skills.EffectType{skills.TypeStun, skills.TypeRoot, skills.TypeParalyze, skills.TypeMute, skills.TypeFear}
	dotTypes     = []skills.EffectType{skills.TypeDmgOverTime, skills.TypeDmgOverTimePercent}
)

// The draws of cast()'s clan searches.
const (
	clanHealDivisor = 10 // Get(100) < (100 − HP%) / 10
	partyChance     = 20 // Get(100) <= 20
)

// cast is AttackableAI.cast(sk) against target (the attack target): the
// skill's effect types pick how it is used. ActNone when it is not used;
// Pawn set when Mobius started moving to a leader on the way.
func (t *Thinker) cast(npc Npc, target Creature, sk *skills.Skill) Decision {
	if !canCast(npc, sk) || !npc.AIProfile().AllowsCast(t.Rnd, npc, sk, target) || target == nil {
		return none
	}
	c := caster{t: t, npc: npc, target: target, sk: sk}
	d := c.decide()
	d.Pawn, d.PawnRange = c.pawn, c.pawnRange
	return d
}

// caster holds one cast() evaluation.
type caster struct {
	t      *Thinker
	npc    Npc
	target Creature
	sk     *skills.Skill
	// dist2 is the distance to the target less its collision radius (and
	// 30 when it moves); srange the cast range plus the NPC's radius.
	dist2, srange float64
	// pawn is a leader Mobius started moving to.
	pawn      Creature
	pawnRange int
}

// current is a cast at the NPC's target (the attack target).
func (c *caster) current() Decision {
	return Decision{Action: ActCast, Skill: c.sk, Target: c.target, StopMove: true}
}

// seesInRange: the target is in sight, alive and within srange.
func (c *caster) seesInRange() bool {
	return c.t.World.CanSee(c.npc, c.target) && !c.target.IsDead() && c.dist2 <= c.srange
}

func (c *caster) decide() Decision {
	t, npc, target, sk := c.t, c.npc, c.target, c.sk
	coll := npc.TemplateCollision()
	c.dist2 = dist2D(npc, target) - float64(target.TemplateCollision())
	c.srange = float64(sk.CastRange + coll)
	if target.IsMoving() {
		c.dist2 -= castMovingSlack
	}
	tt := sk.TargetType

	if sk.IsContinuous() {
		if !sk.IsDebuff {
			if !npc.AffectedBySkill(sk.ID) {
				return castOn(sk, npc, target)
			}
			if tt == data.TargetSelf {
				return none
			}
			if tt == data.TargetOne {
				if o := t.effectTargetReconsider(npc, target, sk, true); o != nil {
					return castOn(sk, o, target)
				}
			}
			if t.canParty(npc, sk) {
				return castOn(sk, npc, target)
			}
		} else if d := c.debuff(isAuraOrCorpse, true); d.Action != ActNone {
			return d
		}
	}

	if sk.HasEffectType(dispelTypes...) {
		if tt == data.TargetOne {
			if target.HasBuffEffect() && c.seesInRange() {
				return c.current()
			}
			if o := t.effectTargetReconsider(npc, target, sk, false); o != nil {
				return castOn(sk, o, target)
			}
		} else if t.canAOE(npc, target, sk) {
			if isAura(tt) && t.World.CanSee(npc, target) {
				return c.current()
			}
			if isArea(tt) && c.seesInRange() {
				return c.current()
			}
		}
	}

	if sk.HasEffectType(skills.TypeHeal) {
		if d := c.heal(); d.Action != ActNone {
			return d
		}
	}

	if sk.HasEffectType(damageTypes...) {
		if d := c.damage(); d.Action != ActNone {
			return d
		}
	}

	if sk.HasEffectType(skills.TypeSleep) {
		if tt == data.TargetOne {
			reach := float64(npc.PhysicalAttackRange() + coll + target.TemplateCollision())
			if !target.IsDead() && c.dist2 <= c.srange && (c.dist2 > reach || target.IsMoving()) && !target.AffectedBySkill(sk.ID) {
				return c.current()
			}
			if t.effectTargetReconsider(npc, target, sk, false) != nil {
				return c.current()
			}
		} else if t.canAOE(npc, target, sk) {
			if isAura(tt) {
				return c.current()
			}
			if isArea(tt) && c.seesInRange() {
				return c.current()
			}
		}
	}

	if sk.HasEffectType(disableTypes...) {
		if d := c.debuff(isAura, false); d.Action != ActNone {
			return d
		}
	}

	if sk.HasEffectType(dotTypes...) {
		if d := c.debuff(isAuraOrCorpse, true); d.Action != ActNone {
			return d
		}
	}

	if sk.HasEffectType(skills.TypeResurrection) {
		if d := c.res(); d.Action != ActNone {
			return d
		}
	}

	return c.damage()
}

// debuff is cast()'s branch for continuous debuffs, the disabling effects
// (STUN, ROOT, PARALYZE, MUTE, FEAR; aliveCheck false: Mobius does not check
// the target is alive there) and damage over time: a single target in sight
// and range that does not have the effect yet; an aura or a area when
// canAOE allows it; or, for a single-target skill, when effectTargetReconsider
// finds someone, a cast at the target anyway (Mobius casts at its current
// target, not at the one it found).
func (c *caster) debuff(aura func(data.TargetType) bool, aliveCheck bool) Decision {
	t, npc, target, sk := c.t, c.npc, c.target, c.sk
	tt := sk.TargetType
	aoe := t.canAOE(npc, target, sk)
	if t.World.CanSee(npc, target) && !aoe && (!aliveCheck || !target.IsDead()) && c.dist2 <= c.srange {
		if !target.AffectedBySkill(sk.ID) {
			return c.current()
		}
	} else if aoe {
		if aura(tt) {
			return c.current()
		}
		if isArea(tt) && c.seesInRange() {
			return c.current()
		}
	} else if tt == data.TargetOne {
		if t.effectTargetReconsider(npc, target, sk, false) != nil {
			return c.current()
		}
	}
	return none
}

// damage is cast()'s branch for damage skills, also its last resort: an
// aura that covers the target is cast at once; anything else at the target
// in sight and range, or at whoever skillTargetReconsider finds.
func (c *caster) damage() Decision {
	t, npc, target, sk := c.t, c.npc, c.target, c.sk
	if t.canAura(npc, target, sk) {
		return c.current()
	}
	if c.seesInRange() {
		return c.current()
	}
	if o := t.skillTargetReconsider(npc, target, sk); o != nil {
		return castOn(sk, o, target)
	}
	return none
}

// heal is cast()'s HEAL branch: a minion heals its living leader when
// Get(100) > the leader's HP% and it sees it (moving to it first when out
// of reach); the NPC heals itself when Get(100) < (100 − HP%) / 3; a
// single-target heal goes to a living clan member in range when Get(100) <
// (100 − its HP%) / 10; a party heal is cast when a clan member in its
// affect range is hurt and Get(100) <= 20.
func (c *caster) heal() Decision {
	t, npc, target, sk := c.t, c.npc, c.target, c.sk
	coll := npc.TemplateCollision()
	if leader := npc.Leader(); leader != nil && sk.TargetType != data.TargetSelf {
		if !leader.IsDead() && float64(t.Rnd.Get(100)) > hpPercent(leader) {
			r := sk.CastRange + coll + leader.TemplateCollision()
			if !inRange(r, npc, leader) && !isParty(sk) && !npc.IsMovementDisabled() {
				c.pawn, c.pawnRange = leader, r
			}
			if t.World.CanSee(npc, leader) {
				return castOn(sk, leader, target)
			}
		}
	}
	if float64(t.Rnd.Get(100)) < (100-hpPercent(npc))/3 {
		return castOn(sk, npc, target)
	}
	if sk.TargetType == data.TargetOne {
		for o := range t.World.Visible(npc, OfAttackable, sk.CastRange+coll) {
			if !o.IsDead() && inMyClan(npc, o) && float64(t.Rnd.Get(100)) < (100-hpPercent(o))/clanHealDivisor && t.World.CanSee(npc, o) {
				return castOn(sk, o, target)
			}
		}
	}
	if isParty(sk) {
		for o := range t.World.Visible(npc, OfAttackable, sk.AffectRange+coll) {
			if inMyClan(o, npc) && o.CurrentHP() < float64(o.MaxHP()) && t.Rnd.Get(100) <= partyChance {
				return castOn(sk, npc, target)
			}
		}
	}
	return none
}

// res is cast()'s RESURRECTION branch. A single-target resurrection: a
// minion casts it at its leader whenever it sees it (Mobius checks the
// leader is dead only before moving to it), then at a dead clan member in
// range when Get(100) < 10. A party resurrection: when an NPC of its clan in
// its affect range is hurt and Get(100) <= 20.
func (c *caster) res() Decision {
	t, npc, target, sk := c.t, c.npc, c.target, c.sk
	coll := npc.TemplateCollision()
	if isParty(sk) {
		for o := range t.World.Visible(npc, OfNpc, sk.AffectRange+coll) {
			if inMyClan(npc, o) && o.CurrentHP() < float64(o.MaxHP()) && t.Rnd.Get(100) <= partyChance {
				return castOn(sk, npc, target)
			}
		}
		return none
	}
	if leader := npc.Leader(); leader != nil && sk.TargetType != data.TargetSelf {
		r := sk.CastRange + coll + leader.TemplateCollision()
		if leader.IsDead() && !inRange(r, npc, leader) && !npc.IsMovementDisabled() {
			c.pawn, c.pawnRange = leader, r
		}
		if t.World.CanSee(npc, leader) {
			return castOn(sk, leader, target)
		}
	}
	for o := range t.World.Visible(npc, OfAttackable, sk.CastRange+coll) {
		if o.IsDead() && inMyClan(npc, o) && t.Rnd.Get(100) < resChance && t.World.CanSee(npc, o) {
			return castOn(sk, o, target)
		}
	}
	return none
}

// notFoe reports Mobius's !(target.isAttackable() && !isChaos()): an
// attackable counts only for a chaos NPC.
func notFoe(npc Npc, o Creature) bool {
	return o.AIClass() == ClassAttackable && !npc.AIProfile().Chaos
}

// canAura is CreatureAI.canAura: an aura skill whose affect range covers the
// attack target.
func (t *Thinker) canAura(npc Npc, target Creature, sk *skills.Skill) bool {
	if !isAuraOrCorpse(sk.TargetType) || target == nil {
		return false
	}
	for o := range t.World.Visible(npc, OfCreature, sk.AffectRange) {
		if o == target {
			return true
		}
	}
	return false
}

// canAOE is CreatureAI.canAOE. Within the skill's affect range, counting
// only creatures the NPC sees and attackables only for a chaos NPC:
//
//   - a dispel aura: nobody around the NPC has the skill's effect;
//   - a dispel area: always (Mobius's loop can only confirm it);
//   - another aura: someone around the NPC has an effect;
//   - another area: nobody around the target has the skill's effect.
func (t *Thinker) canAOE(npc Npc, target Creature, sk *skills.Skill) bool {
	tt := sk.TargetType
	counts := func(o Creature) bool { return t.World.CanSee(npc, o) && !notFoe(npc, o) }
	if sk.HasEffectType(dispelTypes...) {
		switch {
		case isAuraOrCorpse(tt):
			for o := range t.World.Visible(npc, OfCreature, sk.AffectRange) {
				if counts(o) && o.AffectedBySkill(sk.ID) {
					return false
				}
			}
			return true
		case isArea(tt):
			return true
		}
		return false
	}
	switch {
	case isAuraOrCorpse(tt):
		for o := range t.World.Visible(npc, OfCreature, sk.AffectRange) {
			if counts(o) && o.HasEffects() {
				return true
			}
		}
	case isArea(tt) && target != nil:
		for o := range t.World.Visible(target, OfCreature, sk.AffectRange) {
			if counts(o) && o.AffectedBySkill(sk.ID) {
				return false
			}
		}
		return true
	}
	return false
}

// canParty is CreatureAI.canParty: a party skill when some clan member in
// its affect range (in sight) does not have its effect yet.
func (t *Thinker) canParty(npc Npc, sk *skills.Skill) bool {
	if !isParty(sk) {
		return false
	}
	count, affected := 0, 0
	for o := range t.World.Visible(npc, OfAttackable, sk.AffectRange) {
		if !t.World.CanSee(npc, o) || !inMyClan(o, npc) {
			continue
		}
		count++
		if o.AffectedBySkill(sk.ID) {
			affected++
		}
	}
	return affected < count
}

// effectTargetReconsider is AttackableAI.effectTargetReconsider.
//
// For a skill that does not dispel:
//   - negative (a debuff): the first attacker of the attack-by list, alive,
//     in sight and not the attack target, within the cast range (less the
//     NPC's radius, less 70 when it moves) while the attack target lacks the
//     effect; else the first visible player or summon within the cast range
//     of the last attacker measured (0 when none was: a Mobius quirk) that
//     qualifies the same way;
//   - positive (a buff): the first clan member within the cast range, alive,
//     in sight, that lacks the effect.
//
// For a dispel: the first visible player or summon within range while the
// attack target has a buff.
func (t *Thinker) effectTargetReconsider(npc Npc, target Creature, sk *skills.Skill, positive bool) Creature {
	if target == nil {
		return nil
	}
	coll := npc.TemplateCollision()
	slack := func(o Creature, d float64) float64 {
		if o.IsMoving() {
			return d - movingSlack
		}
		return d
	}
	if sk.HasEffectType(dispelTypes...) {
		r := sk.CastRange + coll + target.TemplateCollision()
		for o := range t.World.Visible(npc, OfCreature, r) {
			if o.IsDead() || !t.World.CanSee(npc, o) {
				continue
			}
			d2 := slack(o, dist2D(npc, o)-float64(coll))
			if isPlayerOrSummon(o) && d2 <= float64(sk.CastRange+coll+o.TemplateCollision()) && target.HasBuffEffect() {
				return o
			}
		}
		return nil
	}
	if positive {
		for o := range t.World.Visible(npc, OfAttackable, sk.CastRange+coll) {
			if o.IsDead() || !t.World.CanSee(npc, o) || !inMyClan(o, npc) {
				continue
			}
			d2 := slack(o, dist2D(npc, o)-float64(coll))
			if d2 <= float64(sk.CastRange+coll+o.TemplateCollision()) && !o.AffectedBySkill(sk.ID) {
				return o
			}
		}
		return nil
	}
	quirk := 0
	for o := range npc.AttackBy() {
		if o == nil || o.IsDead() || !t.World.CanSee(npc, o) || o == target {
			continue
		}
		d2 := slack(o, dist2D(npc, o)-float64(coll))
		quirk = sk.CastRange + coll + o.TemplateCollision()
		if d2 <= float64(quirk) && !target.AffectedBySkill(sk.ID) {
			return o
		}
	}
	for o := range t.World.Visible(npc, OfCreature, quirk) {
		if o.IsDead() || !t.World.CanSee(npc, o) {
			continue
		}
		d2 := slack(o, dist2D(npc, o))
		if isPlayerOrSummon(o) && d2 <= float64(sk.CastRange+coll+o.TemplateCollision()) && !target.AffectedBySkill(sk.ID) {
			return o
		}
	}
	return nil
}

// skillTargetReconsider is AttackableAI.skillTargetReconsider: the first
// attacker of the aggro list, alive and in sight, within the cast range
// (less the NPC's radius; the range adds the attack target's radius, as
// Mobius does); else, for anything but a guard, the first visible creature
// within that range in sight that is a player, a summon, or for a chaos NPC
// an attackable outside its clan.
func (t *Thinker) skillTargetReconsider(npc Npc, target Creature, sk *skills.Skill) Creature {
	coll := npc.TemplateCollision()
	r := float64(sk.CastRange + coll + target.TemplateCollision())
	for o := range npc.Aggro() {
		if o == nil || !t.World.CanSee(npc, o) || o.IsDead() {
			continue
		}
		if dist2D(npc, o)-float64(coll) <= r {
			return o
		}
	}
	if npc.IsGuard() {
		return nil
	}
	for o := range t.World.Visible(npc, OfCreature, -1) {
		if dist2D(npc, o) > r || !t.World.CanSee(npc, o) {
			continue
		}
		switch o.AIClass() {
		case ClassPlayer, ClassSummon:
			return o
		case ClassAttackable:
			if npc.AIProfile().Chaos && !inMyClan(o, npc) {
				return o
			}
		}
	}
	return nil
}
