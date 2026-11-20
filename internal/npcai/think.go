package npcai

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
)

// Config holds the NPC.ini keys the decisions read.
type Config struct {
	// RaidChaosTime, GrandChaosTime and MinionChaosTime are the thinks in
	// combat after which raid bosses, grand bosses and raid minions may
	// change target (NPC.ini; the Java fallback is 10).
	RaidChaosTime, GrandChaosTime, MinionChaosTime int
}

// DefaultConfig returns the NPC.ini defaults: chaos after 30 thinks.
func DefaultConfig() Config {
	return Config{RaidChaosTime: 30, GrandChaosTime: 30, MinionChaosTime: 30}
}

// State is what an NPC's AI remembers between thinks (Mobius AttackableAI
// fields). The game keeps one per NPC; the zero value is a fresh AI.
type State struct {
	// ChaosTime counts the thinks since the last chaos target change
	// (_chaosTime).
	ChaosTime int
}

// Thinker makes the decisions: Rnd is the world loop's random source, World
// answers about the world. The zero Config is not usable; start from
// DefaultConfig.
type Thinker struct {
	Rnd    rnd.Source
	World  World
	Config Config
}

// The thinkActive constants.
const (
	// RandomWalkRate: an idle NPC wanders, or a minion near its leader
	// buffs itself, when Get(30) == 0 (RANDOM_WALK_RATE).
	RandomWalkRate = 30
	// FollowOffset and RaidFollowOffset are how far a minion lets its
	// leader go before it follows (200, raid minions 500);
	// FollowMinRadius is the inner radius of the point it walks to.
	FollowOffset     = 200
	RaidFollowOffset = 500
	FollowMinRadius  = 30
	// SpawnRadius and SpawnMinRadius place minions around their master
	// (MinionList SPAWN_OFFSET_RADIUS, MINIMUM_SPAWN_RADIUS).
	SpawnRadius    = 200
	SpawnMinRadius = 30
)

// The thinkAttack constants.
const (
	// suicideHP: suicide skills are used below 30 % HP.
	suicideHP = 30
	// chaosHate is the hate targetReconsider() adds without a most hated
	// enemy.
	chaosHate = 2000
	// movingSlack is what effectTargetReconsider and targetReconsider()
	// take off the distance to a moving creature; castMovingSlack what
	// cast() takes off.
	movingSlack     = 70
	castMovingSlack = 30
)

// hasSkillChance is Npc.hasSkillChance: Get(100) < Between(min, max) of the
// template's skill chances (7 and 15 by default); two draws, in that order.
func (t *Thinker) hasSkillChance(npc Npc) bool {
	p := npc.AIProfile()
	roll := t.Rnd.Get(100)
	return roll < t.Rnd.Between(p.MinSkillChance, p.MaxSkillChance)
}

// canCast is checkSkillCastConditions: not casting (unless the skill casts
// alongside), more MP than the skill takes, the skill not disabled, and not
// muted for it (static skills ignore mutes).
func canCast(npc Npc, sk *skills.Skill) bool {
	if npc.IsCastingNow() && !sk.SimultaneousCast {
		return false
	}
	if float64(sk.MPConsume) >= npc.CurrentMP() {
		return false
	}
	if npc.IsSkillDisabled(sk) {
		return false
	}
	return sk.IsStatic() || !((sk.IsMagic() && npc.IsMuted()) || npc.IsPhysicalMuted())
}

// isParty is CreatureAI.isParty: the skill targets the party.
func isParty(sk *skills.Skill) bool { return sk.TargetType == data.TargetParty }

// Suicide is thinkAttack's suicide step, after the sight check: below 30 %
// HP (as an int percent), an NPC with suicide skills picks one (Get(size)),
// and when target is within its affect range (2D, plus both collision
// radii) and hasSkillChance passes, tries it with cast(). A Decision with an
// action ends the think; ActNone goes on (with Pawn, after that move).
func (t *Thinker) Suicide(npc Npc, target Creature) Decision {
	list := npc.AIProfile().Skills(ScopeSuicide)
	if len(list) == 0 || int(hpPercent(npc)) >= suicideHP {
		return none
	}
	sk := list[t.Rnd.Get(len(list))]
	if !inRange(sk.AffectRange, npc, target) || !t.hasSkillChance(npc) {
		return none
	}
	return t.cast(npc, target, sk)
}

// Chaos is thinkAttack's target reconsider of raid bosses, grand bosses and
// raid minions, after the archer step. Each such think counts in st; past
// the configured number of thinks one draw decides a change:
//
//   - a raid boss: Get(100) <= 100 − HP%, or 100 − 2 × HP% when it has
//     minions;
//   - a grand boss: Get(100) <= 100 − 3 × HP%, or Get(100) <= 10 once that
//     rate is 10 or less;
//   - otherwise (a raid minion, or a boss below its own count but past the
//     minion count): Get(100) <= 100 − 2 × HP%.
//
// On a change, TargetReconsider(true) picks the new target: found, the count
// restarts and the think ends (ActRetarget, stop). Not found, Mobius has
// lost its target and looks again with TargetReconsider(false): found, it
// becomes the target and the think goes on (ActRetarget, not stop); not
// found, the think ends (ActNone, stop). Without a change, the think goes on
// with the same target (ActNone, not stop).
func (t *Thinker) Chaos(st *State, npc Npc) (d Decision, stop bool) {
	rank := npc.Rank()
	if !rank.raid() {
		return none, false
	}
	st.ChaosTime++
	cur, maxHP := npc.CurrentHP(), float64(npc.MaxHP())
	change := false
	switch {
	case rank == RankRaidBoss && st.ChaosTime > t.Config.RaidChaosTime:
		multiplier := 100.0
		if npc.HasMinions() {
			multiplier = 200
		}
		change = float64(t.Rnd.Get(100)) <= 100-cur*multiplier/maxHP
	case rank == RankGrandBoss && st.ChaosTime > t.Config.GrandChaosTime:
		rate := 100 - cur*300/maxHP
		roll := t.Rnd.Get(100)
		if rate <= 10 {
			change = roll <= 10
		} else {
			change = float64(roll) <= rate
		}
	case st.ChaosTime > t.Config.MinionChaosTime:
		change = float64(t.Rnd.Get(100)) <= 100-cur*200/maxHP
	}
	if !change {
		return none, false
	}
	if c := t.TargetReconsider(npc, true); c != nil {
		st.ChaosTime = 0
		return Decision{Action: ActRetarget, Target: c}, true
	}
	if c := t.TargetReconsider(npc, false); c != nil {
		return Decision{Action: ActRetarget, Target: c}, false
	}
	return none, true
}

// UseSkills is thinkAttack's skill use, after Chaos, against target (the
// most hated enemy, the NPC's target and attack target). It runs when the
// NPC stands still or is a mage:
//
//   - with GENERAL skills and HEAL skills: a minion heals its living leader
//     when Get(100) > the leader's HP% (moving to it first when out of
//     reach); then itself when Get(100) < (100 − HP%) / 3; then a dead
//     (sic) clan member in range of a single-target heal when Get(100) <
//     (100 − its HP%) / 10, drawn per candidate, or casts a party heal;
//   - with GENERAL skills and RES skills: a minion resurrects its dead
//     leader; then a dead clan member in range of a single-target
//     resurrection when Get(100) < 10, drawn per candidate, or casts a
//     party resurrection;
//   - SHORT_RANGE skills: when hasSkillChance passes and target is within
//     150 (2D), one picked with Get(size), if target is within its cast
//     range (3D) and the gates allow it;
//   - LONG_RANGE skills: the same, at any distance.
//
// ActCast and ActMoveToPawn end the think; ActNone goes on to the reach
// check and the attack.
func (t *Thinker) UseSkills(npc Npc, target Creature) Decision {
	p := npc.AIProfile()
	if npc.IsMoving() && p.AIType != AIMage {
		return none
	}
	if len(p.Skills(ScopeGeneral)) > 0 {
		if d := t.useHeal(npc, target); d.Action != ActNone {
			return d
		}
		if d := t.useRes(npc, target); d.Action != ActNone {
			return d
		}
	}
	if list := p.Skills(ScopeShortRange); len(list) > 0 && t.hasSkillChance(npc) && dist2D(npc, target) <= ShortRange {
		if d := t.useRanged(npc, target, list); d.Action != ActNone {
			return d
		}
	}
	if list := p.Skills(ScopeLongRange); len(list) > 0 && t.hasSkillChance(npc) {
		return t.useRanged(npc, target, list)
	}
	return none
}

// useRanged picks a short or long range skill and casts it at target when
// target is within its cast range (3D; none: any distance), the NPC can cast
// it and the gates allow it.
func (t *Thinker) useRanged(npc Npc, target Creature, list []*skills.Skill) Decision {
	sk := list[t.Rnd.Get(len(list))]
	if (sk.CastRange < 1 || dist3D(npc, target) < float64(sk.CastRange)) && canCast(npc, sk) &&
		npc.AIProfile().AllowsCast(t.Rnd, npc, sk, target) {
		return castOn(sk, target, nil)
	}
	return none
}

// useHeal is the heal part of UseSkills.
func (t *Thinker) useHeal(npc Npc, target Creature) Decision {
	heals := npc.AIProfile().Skills(ScopeHeal)
	if len(heals) == 0 {
		return none
	}
	coll := npc.TemplateCollision()
	if leader := npc.Leader(); leader != nil && !leader.IsDead() && float64(t.Rnd.Get(100)) > hpPercent(leader) {
		for _, sk := range heals {
			if sk.TargetType == data.TargetSelf || !canCast(npc, sk) {
				continue
			}
			r := sk.CastRange + coll + leader.TemplateCollision()
			if !inRange(r, npc, leader) && !isParty(sk) && !npc.IsMovementDisabled() {
				return Decision{Action: ActMoveToPawn, Target: leader, Range: r}
			}
			if t.World.CanSee(npc, leader) {
				return castOn(sk, leader, target)
			}
		}
	}
	if float64(t.Rnd.Get(100)) < (100-hpPercent(npc))/3 {
		for _, sk := range heals {
			if canCast(npc, sk) {
				return castOn(sk, npc, target)
			}
		}
	}
	for _, sk := range heals {
		if !canCast(npc, sk) {
			continue
		}
		if sk.TargetType == data.TargetOne {
			for c := range t.World.Visible(npc, OfAttackable, sk.CastRange+coll) {
				if c.IsDead() && inMyClan(c, npc) && float64(t.Rnd.Get(100)) < (100-hpPercent(c))/10 && t.World.CanSee(npc, c) {
					return castOn(sk, c, target)
				}
			}
		}
		if isParty(sk) {
			return Decision{Action: ActCast, Skill: sk, Target: target, StopMove: true}
		}
	}
	return none
}

// resChance is the chance in 100 to resurrect a dead clan member.
const resChance = 10

// useRes is the resurrection part of UseSkills.
func (t *Thinker) useRes(npc Npc, target Creature) Decision {
	list := npc.AIProfile().Skills(ScopeRes)
	if len(list) == 0 {
		return none
	}
	coll := npc.TemplateCollision()
	if leader := npc.Leader(); leader != nil && leader.IsDead() {
		for _, sk := range list {
			if sk.TargetType == data.TargetSelf || !canCast(npc, sk) {
				continue
			}
			r := sk.CastRange + coll + leader.TemplateCollision()
			if !inRange(r, npc, leader) && !isParty(sk) && !npc.IsMovementDisabled() {
				return Decision{Action: ActMoveToPawn, Target: leader, Range: r}
			}
			if t.World.CanSee(npc, leader) {
				return castOn(sk, leader, target)
			}
		}
	}
	for _, sk := range list {
		if !canCast(npc, sk) {
			continue
		}
		if sk.TargetType == data.TargetOne {
			for c := range t.World.Visible(npc, OfAttackable, sk.CastRange+coll) {
				if c.IsDead() && inMyClan(npc, c) && t.Rnd.Get(100) < resChance && t.World.CanSee(npc, c) {
					return castOn(sk, c, target)
				}
			}
		}
		if isParty(sk) {
			return castOn(sk, npc, target)
		}
	}
	return none
}

// MovementDisabled is thinkAttack's branch for an NPC that cannot move
// (movementDisable), against target (the attack target). One Get(100) picks
// the kinds of skills it tries, in order, each with tryCast: IMMOBILIZE
// below 15 (when target is not immobilized already), COT below 20, DEBUFF
// below 30, NEGATIVE below 40; then ATTACK and UNIVERSAL at any roll. With
// none, it attacks target within its reach (2D, both collision radii) in
// sight, or reconsiders its target (targetReconsider(), ActRetarget or
// ActNone).
func (t *Thinker) MovementDisabled(npc Npc, target Creature) Decision {
	if target == nil {
		return none
	}
	dist := dist2D(npc, target)
	roll := t.Rnd.Get(100)
	if !target.IsImmobilized() && roll < 15 {
		if d := t.tryCast(npc, target, ScopeImmobilize, dist); d.Action != ActNone {
			return d
		}
	}
	steps := [...]struct {
		below int
		scope Scope
	}{{20, ScopeCOT}, {30, ScopeDebuff}, {40, ScopeNegative}}
	for _, s := range steps {
		if roll < s.below {
			if d := t.tryCast(npc, target, s.scope, dist); d.Action != ActNone {
				return d
			}
		}
	}
	// Mobius asks isMovementDisabled() again, which always holds here.
	if p := npc.AIProfile(); npc.IsMovementDisabled() || p.AIType == AIMage || p.AIType == AIHealer {
		if d := t.tryCast(npc, target, ScopeAttack, dist); d.Action != ActNone {
			return d
		}
	}
	if d := t.tryCast(npc, target, ScopeUniversal, dist); d.Action != ActNone {
		return d
	}
	reach := npc.PhysicalAttackRange() + npc.TemplateCollision() + target.TemplateCollision()
	if dist <= float64(reach) && t.World.CanSee(npc, target) {
		return Decision{Action: ActAttack, Target: target}
	}
	return t.reconsiderHated(npc, target)
}

// tryCast casts the first skill of a scope the NPC can cast that reaches
// target (cast range plus target's collision radius beyond dist, or an aura
// that covers target), in sight, and that the gates allow.
func (t *Thinker) tryCast(npc Npc, target Creature, scope Scope, dist float64) Decision {
	for _, sk := range npc.AIProfile().Skills(scope) {
		if !canCast(npc, sk) || (float64(sk.CastRange+target.TemplateCollision()) <= dist && !t.canAura(npc, target, sk)) {
			continue
		}
		if !t.World.CanSee(npc, target) {
			continue
		}
		if !npc.AIProfile().AllowsCast(t.Rnd, npc, sk, target) {
			continue
		}
		return Decision{Action: ActCast, Skill: sk, Target: target, StopMove: true}
	}
	return none
}

// checkTarget is AttackableAI.checkTarget: a living creature the NPC may
// attack, within its reach (2D, excluded) and in sight when it cannot move.
func (t *Thinker) checkTarget(npc Npc, c Creature) bool {
	if c == nil || c.IsDead() {
		return false
	}
	if npc.IsMovementDisabled() {
		if !insideRadius2D(npc, c, npc.PhysicalAttackRange()+npc.TemplateCollision()+c.TemplateCollision()) {
			return false
		}
		if !t.World.CanSee(npc, c) {
			return false
		}
	}
	return t.World.AutoAttackable(c, npc)
}

// TargetReconsider is AttackableAI.targetReconsider(randomTarget).
//
// With random, it gathers the aggro list's attackers that pass checkTarget,
// then (for an aggressive NPC) the creatures within its aggro range that do
// (one may be listed twice), and picks one with Get(size).
//
// Otherwise, or when that list is empty, Mobius looks for the most hated
// attacker that passes checkTarget but never returns it: it returns nil
// unless no attacker passes, in which case an aggressive NPC takes the first
// creature within its aggro range that passes. This port keeps that.
func (t *Thinker) TargetReconsider(npc Npc, random bool) Creature {
	if random {
		var found []Creature
		for c := range npc.Aggro() {
			if t.checkTarget(npc, c) {
				found = append(found, c)
			}
		}
		if npc.IsAggressive() {
			for c := range t.World.Visible(npc, OfCreature, npc.AggroRange()) {
				if t.checkTarget(npc, c) {
					found = append(found, c)
				}
			}
		}
		if len(found) > 0 {
			return found[t.Rnd.Get(len(found))]
		}
	}
	search := int64(math.MinInt64)
	var best Creature
	for c, hate := range npc.Aggro() {
		if t.checkTarget(npc, c) && hate > search {
			search, best = hate, c
		}
	}
	if best == nil && npc.IsAggressive() {
		for c := range t.World.Visible(npc, OfCreature, npc.AggroRange()) {
			if t.checkTarget(npc, c) {
				return c
			}
		}
	}
	return nil
}

// reconsiderHated is AttackableAI.targetReconsider() of the immobilized
// branch: the most hated enemy, when it is in the aggro list, alive, in
// sight and within the NPC's reach (2D, less the NPC's collision radius,
// less 70 when it moves), gets the hate it already has added again and
// becomes the target (ActRetarget with AddHate). Mobius then scans the
// visible creatures for the most hated enemy when it is not the attack
// target; the immobilized branch always passes the most hated enemy as the
// attack target, so the scan only runs when it is not.
func (t *Thinker) reconsiderHated(npc Npc, attackTarget Creature) Decision {
	most := npc.MostHated()
	hate := func() int64 {
		if most != nil {
			return npc.Hating(most)
		}
		return chaosHate
	}
	self := Creature(npc)
	coll := npc.TemplateCollision()
	for c := range npc.Aggro() {
		if c == nil || c != most || c == self || c.IsDead() || !t.World.CanSee(npc, c) {
			continue
		}
		dist2 := dist2D(npc, c) - float64(coll)
		reach := npc.PhysicalAttackRange() + coll + c.TemplateCollision()
		if c.IsMoving() {
			dist2 -= movingSlack
		}
		if dist2 <= float64(reach) {
			return Decision{Action: ActRetarget, Target: c, AddHate: true, Hate: hate()}
		}
	}
	if npc.IsGuard() || most == nil || most == attackTarget {
		return none
	}
	for c := range t.World.Visible(npc, OfCreature, -1) {
		if c != most || c == self || c.IsDead() || !t.World.CanSee(npc, c) {
			continue
		}
		switch c.AIClass() {
		case ClassPlayer, ClassSummon:
			return Decision{Action: ActRetarget, Target: c, AddHate: true, Hate: hate()}
		case ClassAttackable:
			if npc.AIProfile().Chaos && !inMyClan(c, npc) {
				return Decision{Action: ActRetarget, Target: c, AddHate: true, Hate: hate()}
			}
		}
	}
	return none
}
