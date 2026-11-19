package world

import (
	"fmt"
	"math"
)

// ObjectID identifies an object. The client uses it in every packet that
// refers to an object.
type ObjectID int32

// FirstRuntimeID starts the IDs of runtime-only objects such as NPCs
// (decision W3). Persistent objects (characters, items) get lower IDs from
// storage, so the ranges never overlap.
const FirstRuntimeID ObjectID = 0x60000000

// IsRuntime reports whether id is in the runtime range.
func (id ObjectID) IsRuntime() bool { return id >= FirstRuntimeID }

// Position is a point in the world plus the direction faced (0–65535).
type Position struct {
	X, Y, Z int32
	Heading int32
}

func (p Position) String() string {
	return fmt.Sprintf("(%d, %d, %d) heading %d", p.X, p.Y, p.Z, p.Heading)
}

// DistanceSq3D returns the squared 3D distance between p and q.
func (p Position) DistanceSq3D(q Position) int64 {
	dx, dy, dz := int64(q.X)-int64(p.X), int64(q.Y)-int64(p.Y), int64(q.Z)-int64(p.Z)
	return dx*dx + dy*dy + dz*dz
}

// Distance3D returns the 3D distance between p and q (Mobius
// calculateDistance3D).
func (p Position) Distance3D(q Position) float64 {
	return math.Sqrt(float64(p.DistanceSq3D(q)))
}

// Distance2D returns the distance between p and q, ignoring Z (Mobius
// calculateDistance2D).
func (p Position) Distance2D(q Position) float64 {
	dx, dy := float64(q.X)-float64(p.X), float64(q.Y)-float64(p.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// InRange3D reports whether q is within r of p, edge included, as Mobius's
// range checks (calculateDistance3D(...) <= range). A negative r matches
// nothing.
func (p Position) InRange3D(q Position, r int32) bool {
	return r >= 0 && p.DistanceSq3D(q) <= int64(r)*int64(r)
}
