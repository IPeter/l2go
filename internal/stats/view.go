package stats

import "math"

// PlayerView holds every stat number UserInfo and CharInfo send, computed
// as those Mobius packets compute them. It is a plain value: keep the last
// one sent and send UserInfo and CharInfo again only when Equal reports a
// difference.
type PlayerView struct {
	STR, DEX, CON, INT, WIT, MEN int32
	MaxHP, MaxMP, MaxCP          int32
	// WeightLimit is the maximum load (getMaxLoad).
	WeightLimit    int32
	InventoryLimit int32

	PAtk, PAtkSpd, PDef int32
	Evasion, Accuracy   int32
	Crit                int32
	MAtk, MAtkSpd, MDef int32 // MAtkSpd is the casting speed

	// The speeds as the packets send them: each actual speed divided by
	// MoveMultiplier and rounded. The fly speeds repeat run and walk while
	// flying and are 0 otherwise.
	RunSpd, WalkSpd         int32
	SwimRunSpd, SwimWalkSpd int32
	FlyRunSpd, FlyWalkSpd   int32
	// MoveMultiplier is the current speed over the template speed of the
	// current movement type.
	MoveMultiplier float64
	// AttackSpeedMultiplier is a Java float; the packets widen it to a
	// double.
	AttackSpeedMultiplier float32
	// CollisionRadius and CollisionHeight are Java floats widened to
	// doubles.
	CollisionRadius, CollisionHeight float64

	// MoveSpeed is not in the packets: it is the current speed in game
	// units per second, for the movement simulation.
	MoveSpeed float64
}

// Equal reports whether v and o hold the same numbers. Floating-point fields
// compare by their bits, so a NaN equals itself.
func (v PlayerView) Equal(o PlayerView) bool {
	if !sameFloat(v.MoveMultiplier, o.MoveMultiplier) ||
		math.Float32bits(v.AttackSpeedMultiplier) != math.Float32bits(o.AttackSpeedMultiplier) ||
		!sameFloat(v.CollisionRadius, o.CollisionRadius) ||
		!sameFloat(v.CollisionHeight, o.CollisionHeight) ||
		!sameFloat(v.MoveSpeed, o.MoveSpeed) {
		return false
	}
	v.MoveMultiplier, v.AttackSpeedMultiplier, v.CollisionRadius, v.CollisionHeight, v.MoveSpeed = 0, 0, 0, 0, 0
	o.MoveMultiplier, o.AttackSpeedMultiplier, o.CollisionRadius, o.CollisionHeight, o.MoveSpeed = 0, 0, 0, 0, 0
	return v == o
}

// NpcView holds the stat numbers NpcInfo sends (AbstractNpcInfo), the
// maximum HP for the HP bar (StatusUpdate) and the current speed.
type NpcView struct {
	MaxHP   int32
	PAtkSpd int32
	MAtkSpd int32 // casting speed

	// The speeds as NpcInfo sends them; see PlayerView.
	RunSpd, WalkSpd         int32
	SwimRunSpd, SwimWalkSpd int32
	FlyRunSpd, FlyWalkSpd   int32
	MoveMultiplier          float64
	AttackSpeedMultiplier   float32 // a Java float
	// CollisionRadius and CollisionHeight are Java floats widened to
	// doubles.
	CollisionRadius, CollisionHeight float64

	// MoveSpeed is the current speed in game units per second.
	MoveSpeed float64
}

// Equal reports whether v and o hold the same numbers, comparing
// floating-point fields by their bits.
func (v NpcView) Equal(o NpcView) bool {
	if !sameFloat(v.MoveMultiplier, o.MoveMultiplier) ||
		math.Float32bits(v.AttackSpeedMultiplier) != math.Float32bits(o.AttackSpeedMultiplier) ||
		!sameFloat(v.CollisionRadius, o.CollisionRadius) ||
		!sameFloat(v.CollisionHeight, o.CollisionHeight) ||
		!sameFloat(v.MoveSpeed, o.MoveSpeed) {
		return false
	}
	v.MoveMultiplier, v.AttackSpeedMultiplier, v.CollisionRadius, v.CollisionHeight, v.MoveSpeed = 0, 0, 0, 0, 0
	o.MoveMultiplier, o.AttackSpeedMultiplier, o.CollisionRadius, o.CollisionHeight, o.MoveSpeed = 0, 0, 0, 0, 0
	return v == o
}

func sameFloat(a, b float64) bool { return math.Float64bits(a) == math.Float64bits(b) }
