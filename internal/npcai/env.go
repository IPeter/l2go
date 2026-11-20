package npcai

import (
	"iter"
	"math"

	"github.com/IPeter/l2go/internal/skills"
)

// Point is a world position.
type Point struct{ X, Y, Z int32 }

// Class is what a creature is, as far as the decisions ask (Mobius isPlayer,
// isSummon, isAttackable, instanceof Npc).
type Class uint8

// The classes.
const (
	// ClassOther is any other creature (a door, a static object, ...).
	ClassOther Class = iota
	ClassPlayer
	// ClassSummon is a pet or a servitor (Mobius isSummon).
	ClassSummon
	// ClassNpc is an NPC that is not an Attackable (a folk NPC).
	ClassNpc
	// ClassAttackable is an Attackable: monsters, raid bosses, guards, ...
	ClassAttackable
)

// Of is the class argument of Mobius's World lookups: which creatures a
// lookup returns.
type Of uint8

// The lookups' classes.
const (
	// OfCreature returns every creature (Creature.class).
	OfCreature Of = iota
	// OfNpc returns NPCs and attackables (Npc.class).
	OfNpc
	// OfAttackable returns attackables (Attackable.class).
	OfAttackable
)

// Accepts reports whether a creature of class c is an instance of o.
func (o Of) Accepts(c Class) bool {
	switch o {
	case OfNpc:
		return c == ClassNpc || c == ClassAttackable
	case OfAttackable:
		return c == ClassAttackable
	}
	return true
}

// Rank tells raid bosses, grand bosses and raid minions apart (Mobius
// instanceof RaidBoss, instanceof GrandBoss, isRaidMinion). The class comes
// first: a RaidBoss that is also a raid minion is RankRaidBoss.
type Rank uint8

// The ranks.
const (
	RankNormal Rank = iota
	// RankRaidBoss is a RaidBoss (not a GrandBoss, which Mobius derives
	// from Monster).
	RankRaidBoss
	RankGrandBoss
	// RankRaidMinion is a minion of a raid boss (setIsRaidMinion).
	RankRaidMinion
)

// raid reports Mobius's isRaid() || isRaidMinion(): raid and grand bosses
// and raid minions.
func (r Rank) raid() bool { return r != RankNormal }

// Creature is a creature as the decisions see it: the NPC that thinks, its
// enemies, its leader and the clan members around it. Implementations must
// be comparable, and the same creature must always compare equal (the
// decisions compare targets with ==).
type Creature interface {
	// Loc is the position.
	Loc() Point
	// CurrentHP and MaxHP are the HP (Mobius getCurrentHp, getMaxHp).
	CurrentHP() float64
	MaxHP() int32
	// IsDead is Mobius isDead.
	IsDead() bool
	// IsMoving is Mobius isMoving.
	IsMoving() bool
	// TemplateCollision is the collision radius of the template as an int
	// (Mobius getTemplate().getCollisionRadius()), not the grown one.
	TemplateCollision() int
	// AIClass is what the creature is.
	AIClass() Class
	// AIProfile is the AI profile of an NPC's template, nil for any other
	// creature. The clan tests read it.
	AIProfile() *Profile
	// AffectedBySkill is Mobius isAffectedBySkill: an effect of the skill
	// is on the creature.
	AffectedBySkill(id int) bool
	// HasBuffEffect is getEffectList().getFirstEffect(EffectType.BUFF) !=
	// null: an effect of type BUFF is on the creature.
	HasBuffEffect() bool
	// HasEffects is !getEffectList().isEmpty().
	HasEffects() bool
	// IsImmobilized is Mobius isImmobilized (rooted, ...).
	IsImmobilized() bool
}

// Npc is the attackable that thinks. Its AIProfile must not be nil.
type Npc interface {
	Creature
	// CurrentMP is Mobius getCurrentMp.
	CurrentMP() float64
	// IsCastingNow is Mobius isCastingNow.
	IsCastingNow() bool
	// IsMuted and IsPhysicalMuted are Mobius isMuted and isPhysicalMuted.
	IsMuted() bool
	IsPhysicalMuted() bool
	// IsSkillDisabled is Mobius isSkillDisabled: the skill is on reuse or
	// all skills are disabled.
	IsSkillDisabled(sk *skills.Skill) bool
	// IsMovementDisabled is Mobius isMovementDisabled.
	IsMovementDisabled() bool
	// Leader is the minion's leader, nil when the NPC is not a minion.
	Leader() Creature
	// HasMinions reports a master with minions (Monster.hasMinions).
	HasMinions() bool
	// Rank tells raid bosses, grand bosses and raid minions.
	Rank() Rank
	// IsGuard reports an instance of Mobius Guard.
	IsGuard() bool
	// IsAggressive and AggroRange are Mobius isAggressive and
	// getAggroRange.
	IsAggressive() bool
	AggroRange() int
	// PhysicalAttackRange is Mobius getPhysicalAttackRange.
	PhysicalAttackRange() int
	// AttackBy is the attack-by list (Mobius getAttackByList), in its
	// order.
	AttackBy() iter.Seq[Creature]
	// Aggro is the aggro list (Mobius getAggroList().values()), attacker
	// and hate, in its order. It is empty for a dead NPC (getHateList).
	Aggro() iter.Seq2[Creature, int64]
	// MostHated is Mobius getMostHated (nil for none).
	MostHated() Creature
	// Hating is Mobius getHating.
	Hating(c Creature) int64
}

// World answers the questions about the world around an NPC.
type World interface {
	// CanSee is GeoEngine.canSeeTarget.
	CanSee(from, to Creature) bool
	// Visible returns the creatures of class of that center sees, without
	// center itself, within radius (3D, included) of it, in the order of
	// Mobius's World lookups (the regions around center, then each
	// region's objects). A negative radius has no limit
	// (World.getFirstVisibleObject, forEachVisibleObject). Decisions stop
	// early, so the sequence should be lazy.
	Visible(center Creature, of Of, radius int) iter.Seq[Creature]
	// AutoAttackable is target.isAutoAttackable(by).
	AutoAttackable(target Creature, by Npc) bool
}

// dist2D and dist3D are Mobius calculateDistance2D and 3D.
func dist2D(a, b Creature) float64 {
	pa, pb := a.Loc(), b.Loc()
	dx := float64(pb.X) - float64(pa.X)
	dy := float64(pb.Y) - float64(pa.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func dist3D(a, b Creature) float64 {
	pa, pb := a.Loc(), b.Loc()
	dx := float64(pb.X) - float64(pa.X)
	dy := float64(pb.Y) - float64(pa.Y)
	dz := float64(pb.Z) - float64(pa.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// inRange is LocationUtil.checkIfInRange(r, a, b, false): the 2D distance is
// at most r plus both collision radii.
func inRange(r int, a, b Creature) bool {
	return dist2D(a, b) <= float64(r+a.TemplateCollision()+b.TemplateCollision())
}

// insideRadius2D is Creature.isInsideRadius2D: closer than r (2D, r
// excluded).
func insideRadius2D(a, b Creature, r int) bool { return dist2D(a, b) < float64(r) }

// hpPercent is (getCurrentHp() / getMaxHp()) * 100.
func hpPercent(c Creature) float64 { return c.CurrentHP() / float64(c.MaxHP()) * 100 }

// inMyClan is a.isInMyClan(b) (Mobius Npc.isInMyClan, NpcTemplate.isClan):
// both are NPCs with clans, and a is in the generic clan ALL or shares one
// of b's clans. It is not symmetric.
func inMyClan(a, b Creature) bool {
	pa, pb := a.AIProfile(), b.AIProfile()
	if pa == nil || pb == nil {
		return false
	}
	return pa.IsClan(pb.Clans)
}

// isPlayerOrSummon is Mobius's obj.isPlayer() || obj.isSummon().
func isPlayerOrSummon(c Creature) bool {
	k := c.AIClass()
	return k == ClassPlayer || k == ClassSummon
}
