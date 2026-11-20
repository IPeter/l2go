package geo_test

import (
	"testing"

	"github.com/IPeter/l2go/internal/geo"
)

func TestNone(t *testing.T) {
	var e geo.Engine = geo.None{}
	from := geo.Point{X: -84318, Y: 244579, Z: -3730}
	to := geo.Point{X: -83000, Y: 243000, Z: -3500}

	if got := e.Height(from.X, from.Y, from.Z); got != from.Z {
		t.Errorf("Height = %d, want the given z %d", got, from.Z)
	}
	if got := e.SpawnHeight(to.X, to.Y, to.Z); got != to.Z {
		t.Errorf("SpawnHeight = %d, want the given z %d", got, to.Z)
	}
	if e.HasGeo(from.X, from.Y) || e.Pathfinding() || e.CompletelyBlocked(to.X, to.Y, to.Z) {
		t.Error("None has geodata, pathfinding or a blocked cell")
	}
	if !e.CanSeeTarget(from, to) || !e.CanMoveToTarget(from, to) {
		t.Error("None blocks sight or walking")
	}
	if got := e.ValidLocation(from, to); got != to {
		t.Errorf("ValidLocation = %v, want the destination %v", got, to)
	}
	if got := e.FindPath(from, to, true); got != nil {
		t.Errorf("FindPath = %v, want no path", got)
	}
}

// DefaultOptions are Mobius's GeoEngine.ini values.
func TestDefaultOptions(t *testing.T) {
	o := geo.DefaultOptions()
	if !o.Pathfinding || o.LowWeight != 0.5 || o.MediumWeight != 2 || o.HighWeight != 3 || o.DiagonalWeight != 0.707 ||
		o.AdvancedDiagonalStrategy || !o.AvoidObstructedPathNodes || o.MaxPostfilterPasses != 3 ||
		len(o.BufferSizes) != 7 || o.BufferSizes[6] != 500 {
		t.Errorf("DefaultOptions = %+v", o)
	}
}
