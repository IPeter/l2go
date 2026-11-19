// Package skills is the skill core of L2Go, a port of the L2J Mobius
// (Interlude) skill and effect engine that knows nothing about the world:
//
//   - Catalog compiles the skill data of internal/data into Skill values:
//     every level with its compiled preconditions, its effects per scope and
//     the numbers casting reads (Plan 10 K1, K2).
//   - Conditions are compiled from the data's condition trees into Cond
//     predicates over a ConditionContext (K9). Kinds without a port fail
//     closed, and a skill that needs one refuses to be used.
//   - Effects come from a Registry of factories (map[data.EffectName]Factory,
//     no reflection). The handlers live in internal/skills/effects; a name
//     without a port loads as a stub that refuses the cast.
//   - BuffInfo and EffectList are Mobius's BuffInfo and EffectList, rule for
//     rule (K5, K6): stacking by abnormal type and level, herbs, the buff and
//     dance limits, the buff clock, effect ticks, icons and visual masks.
//   - Passive skills and buffs change stats through stats.Modifier values
//     owned by the skill (passives) or by the BuffInfo (K8).
//   - The formulas of casting (cast time, reuse, MP), of magic and physical
//     skill damage, of effect success and abnormal time are pure functions
//     with Mobius's rounding points and draw order.
//   - The target types (K11) are pure functions over the World interface.
//
// Creatures, the world and time come in through interfaces (Creature,
// Player, World, Scheduler) that internal/game implements; nothing here
// touches the world loop. Everything must run on the loop that owns the
// creatures: no type of this package is safe for concurrent use, except a
// Catalog once built.
package skills
