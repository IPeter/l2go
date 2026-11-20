// Package npcai holds the monster AI decisions of Mobius AttackableAI that
// use skills (Plan 13, decision A10), as pure functions over small
// interfaces: the game implements Creature, Npc and World over its own
// objects, injects the random source, and carries out the Decision it gets
// back. The package imports no world, game or server code, so every branch
// can be tested with a scripted source (rnd.NewScripted) that pins Mobius's
// draw order.
//
// What is here:
//
//   - Profile: the AI skill lists of an NPC template, built once at load by
//     the rules of Mobius NpcData (cast range <= 150 is short range, the
//     effect types pick the scopes), in the order of Mobius's HashMap of
//     the template's skills; the Skill0N_* parameter gates; the skill
//     chances, AI type and clans.
//   - Thinker: the steps of thinkAttack that use skills (Suicide, Chaos,
//     UseSkills), the immobilized branch (MovementDisabled), the three
//     target reconsiders, and the buffs and leader following of thinkActive.
//   - RingPoint, FollowPoint and SpawnPoint: MinionList's formation around a
//     leader.
//   - DropsCast: Monster.doCast's refusal of a skill without a negative
//     effect aimed at a player.
//
// The game keeps the rest of thinkAttack (the leash, the attack timeout,
// sight, the crowd step, the archer step, the reach check) and calls the
// steps here at the points where Mobius runs them, so the draws stay in
// Mobius's order:
//
//	most hated, movement disabled -> MovementDisabled (then return)
//	dead target, timeout, sight
//	Suicide (a Cast ends the think)
//	crowd step, archer step
//	Chaos (raids, grand bosses, raid minions)
//	UseSkills (a Cast or MoveToPawn ends the think)
//	reach check, TargetReconsider(false), attack
//
// Nothing here is safe for concurrent use; the world loop calls it.
package npcai
