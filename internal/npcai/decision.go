package npcai

import (
	"strconv"

	"github.com/IPeter/l2go/internal/skills"
)

// Action is what a Decision tells the NPC to do.
type Action uint8

// The actions.
const (
	// ActNone: nothing; the think goes on (or ends, as the step says).
	ActNone Action = iota
	// ActCast: cast Skill (Mobius doCast). Before it, the NPC stops moving
	// when StopMove is set (clientStopMoving) and takes Target as its
	// target; after it, it takes Restore as its target when Restore is set.
	// The game applies DropsCast for monsters.
	ActCast
	// ActAttack: attack Target (doAttack).
	ActAttack
	// ActMoveToPawn: move to Target, up to Range from it (moveToPawn).
	ActMoveToPawn
	// ActMoveTo: move to To (moveTo).
	ActMoveTo
	// ActRetarget: Target becomes the attack target and the target
	// (setAttackTarget, setTarget). With AddHate, Hate is added to it first
	// (addDamageHate(Target, 0, Hate)).
	ActRetarget
)

var actionNames = [...]string{"none", "cast", "attack", "moveToPawn", "moveTo", "retarget"}

// String returns the action's name.
func (a Action) String() string {
	if int(a) < len(actionNames) {
		return actionNames[a]
	}
	return "Action(" + strconv.Itoa(int(a)) + ")"
}

// Decision is the outcome of a decision step.
type Decision struct {
	Action Action
	// Skill is the skill to cast (ActCast).
	Skill *skills.Skill
	// Target is the NPC's target when it casts (ActCast), the creature to
	// attack (ActAttack), to move to (ActMoveToPawn) or to take as the
	// attack target (ActRetarget).
	Target Creature
	// Restore is the target the NPC takes back once the cast has started
	// (ActCast); nil leaves Target as the target.
	Restore Creature
	// StopMove: stop moving first (ActCast).
	StopMove bool
	// Range is the distance of ActMoveToPawn.
	Range int
	// To is the destination of ActMoveTo.
	To Point
	// AddHate and Hate: hate added to Target (ActRetarget).
	AddHate bool
	Hate    int64
	// Pawn and PawnRange are a moveToPawn Mobius starts and goes on from in
	// the same think (cast() towards a leader to heal or resurrect): when
	// Pawn is set, the game moves to it first, then carries out the
	// Decision (a cast stops that move again).
	Pawn      Creature
	PawnRange int
}

// none is the empty Decision.
var none = Decision{}

// castOn is a cast with the NPC stopping first, Target as its target and
// restore taken back after.
func castOn(sk *skills.Skill, target, restore Creature) Decision {
	return Decision{Action: ActCast, Skill: sk, Target: target, Restore: restore, StopMove: true}
}

// DropsCast is Monster.doCast's guard against monsters buffing players: a
// monster drops the cast of a skill without a negative effect
// (Skill.hasNegativeEffect) while its target is a player. The game checks it
// for monsters (instances of Mobius Monster) when it carries out ActCast,
// with the target the NPC has at that moment.
func DropsCast(sk *skills.Skill, target Creature) bool {
	return !sk.HasNegativeEffect() && target != nil && target.AIClass() == ClassPlayer
}
