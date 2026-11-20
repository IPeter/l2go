// Package effects holds the effect handlers of the skill engine, one file
// per Mobius handler of dist/game/data/scripts/handlers/skill/effects (Plan
// 10 K2, K3). Each file registers its factory from init(); Register puts
// them all into a skills.Registry. A factory parses the <effect>'s
// parameters once, at load; the effect then keeps no per-creature state.
//
// Waves A, B and C of Plan 10 are ported (Ported lists them); the handlers
// left for later milestones (summons, fishing, sieges...) load as stubs that
// refuse the cast.
package effects

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

var factories = make(map[data.EffectName]skills.Factory)

// register sets the factory of a handler; each handler file calls it from
// init().
func register(name data.EffectName, f skills.Factory) { factories[name] = f }

// Register adds every ported handler to r.
func Register(r *skills.Registry) {
	for name, f := range factories {
		r.Register(name, f)
	}
}

// NewRegistry returns a registry with every ported handler.
func NewRegistry() *skills.Registry {
	r := skills.NewRegistry()
	Register(r)
	return r
}

// Ported returns the names of the ported handlers, sorted.
func Ported() []data.EffectName {
	names := make([]data.EffectName, 0, len(factories))
	for n := range factories {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// heal clamps an amount to what c can still recover, never below 0
// (Math.max(Math.min(amount, max − current), 0)).
func heal(amount float64, maxRecoverable int, current float64) float64 {
	return max(min(amount, float64(maxRecoverable)-current), 0)
}

// javaInt is Java's (int) cast of a double.
func javaInt(x float64) int32 {
	switch {
	case x != x:
		return 0
	case x >= 2147483647:
		return 2147483647
	case x <= -2147483648:
		return -2147483648
	}
	return int32(x)
}

// isPlayer, isDoor, isSummon and isNpc are the Mobius kind checks.
func isPlayer(c skills.Creature) bool {
	return c != nil && c.InstanceType().Is(data.InstancePlayer)
}

func isDoor(c skills.Creature) bool { return c != nil && c.InstanceType().Is(data.InstanceDoor) }

func isSummon(c skills.Creature) bool {
	return c != nil && c.InstanceType().Is(data.InstanceSummon)
}

func isNpc(c skills.Creature) bool { return c != nil && c.InstanceType().Is(data.InstanceNpc) }

// standUp ends a player target's fake death when damage does that
// (FakeDeathDamageStand), as the Mobius damage handlers do first.
func standUp(env *skills.Env, effected skills.Creature) {
	if isPlayer(effected) && effected.AsPlayer().IsFakeDeath() && env.Config.FakeDeathDamageStand {
		effected.StopFakeDeath(true)
	}
}

// sameCreature reports whether a and b are the same creature.
func sameCreature(a, b skills.Creature) bool {
	return a != nil && b != nil && a.ObjectID() == b.ObjectID()
}
