package geo

import (
	"slices"
	"testing"
)

func TestNone(t *testing.T) {
	var e Engine = None{}
	from := Point{X: -84318, Y: 244579, Z: -3730}
	to := Point{X: -83000, Y: 243000, Z: -3500}

	if got := e.Height(from.X, from.Y, from.Z); got != from.Z {
		t.Errorf("Height = %d, want the given z %d", got, from.Z)
	}
	if got := e.SpawnHeight(to.X, to.Y, to.Z); got != to.Z {
		t.Errorf("SpawnHeight = %d, want the given z %d", got, to.Z)
	}
	if e.HasGeo(from.X, from.Y) {
		t.Error("HasGeo = true, want false")
	}
	if e.CompletelyBlocked(to.X, to.Y, to.Z) {
		t.Error("CompletelyBlocked = true, want false")
	}
	if !e.CanSeeTarget(from, to) {
		t.Error("CanSeeTarget = false, want true")
	}
	if !e.CanMoveToTarget(from, to) {
		t.Error("CanMoveToTarget = false, want true")
	}
	if got := e.ValidLocation(from, to); got != to {
		t.Errorf("ValidLocation = %v, want the destination %v", got, to)
	}
	if got := e.FindPath(from, to); !slices.Equal(got, []Point{to}) {
		t.Errorf("FindPath = %v, want the straight path [%v]", got, to)
	}
}
