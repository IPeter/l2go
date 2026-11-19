package world_test

import (
	"testing"

	"github.com/IPeter/l2go/internal/world"
)

func TestRegionOf(t *testing.T) {
	tests := []struct {
		name    string
		x, y, z int32
		want    world.RegionKey
		ok      bool
	}{
		{"world minimum", world.WorldXMin, world.WorldYMin, world.WorldZMin, world.RegionKey{X: 0, Y: 0, Z: 0}, true},
		{"origin", 0, 0, 0, world.RegionKey{X: 64, Y: 128, Z: 8}, true},
		{"just below the origin", -1, -1, -1, world.RegionKey{X: 63, Y: 127, Z: 7}, true},
		{"last unit of the origin region", 2047, 2047, 1999, world.RegionKey{X: 64, Y: 128, Z: 8}, true},
		{"first unit of the next region", 2048, 2048, 2000, world.RegionKey{X: 65, Y: 129, Z: 9}, true},
		{"first unit of a negative region", -2048, -2048, -2000, world.RegionKey{X: 63, Y: 127, Z: 7}, true},
		{"last unit before it", -2049, -2049, -2001, world.RegionKey{X: 62, Y: 126, Z: 6}, true},
		{"world maximum", world.WorldXMax - 1, world.WorldYMax - 1, world.WorldZMax - 1, world.RegionKey{X: 175, Y: 255, Z: 15}, true},
		{"Z at the top edge", 0, 0, world.WorldZMax, world.RegionKey{X: 64, Y: 128, Z: 15}, true},
		{"Z far above", 0, 0, 100000, world.RegionKey{X: 64, Y: 128, Z: 15}, true},
		{"Z just below the bottom", 0, 0, world.WorldZMin - 1, world.RegionKey{X: 64, Y: 128, Z: 0}, true},
		{"Z far below", 0, 0, -100000, world.RegionKey{X: 64, Y: 128, Z: 0}, true},
		{"Talking Island village", -84318, 244579, -3730, world.RegionKey{X: 22, Y: 247, Z: 6}, true},
		{"Giran", 83400, 147943, -3404, world.RegionKey{X: 104, Y: 200, Z: 6}, true},
		{"X at the maximum", world.WorldXMax, 0, 0, world.RegionKey{}, false},
		{"X below the minimum", world.WorldXMin - 1, 0, 0, world.RegionKey{}, false},
		{"Y at the maximum", 0, world.WorldYMax, 0, world.RegionKey{}, false},
		{"Y below the minimum", 0, world.WorldYMin - 1, 0, world.RegionKey{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := world.RegionOf(tt.x, tt.y, tt.z)
			if got != tt.want || ok != tt.ok {
				t.Errorf("RegionOf(%d, %d, %d) = %v, %v; want %v, %v", tt.x, tt.y, tt.z, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestRegionKeyAdjacent(t *testing.T) {
	k := world.RegionKey{X: 100, Y: 100, Z: 8}
	tests := []struct {
		o    world.RegionKey
		want bool
	}{
		{world.RegionKey{X: 100, Y: 100, Z: 8}, true},
		{world.RegionKey{X: 101, Y: 100, Z: 8}, true},
		{world.RegionKey{X: 99, Y: 99, Z: 7}, true},
		{world.RegionKey{X: 101, Y: 101, Z: 9}, true},
		{world.RegionKey{X: 102, Y: 100, Z: 8}, false},
		{world.RegionKey{X: 100, Y: 98, Z: 8}, false},
		{world.RegionKey{X: 100, Y: 100, Z: 10}, false},
		{world.RegionKey{X: 101, Y: 101, Z: 6}, false},
	}
	for _, tt := range tests {
		if got := k.Adjacent(tt.o); got != tt.want {
			t.Errorf("%v.Adjacent(%v) = %v, want %v", k, tt.o, got, tt.want)
		}
		if got := tt.o.Adjacent(k); got != tt.want {
			t.Errorf("%v.Adjacent(%v) = %v, want %v (not symmetric)", tt.o, k, got, tt.want)
		}
	}
}

func TestHelperAtIsTheRegionCentre(t *testing.T) {
	for _, k := range []world.RegionKey{{X: 0, Y: 0, Z: 0}, {X: 63, Y: 127, Z: 7}, {X: 100, Y: 100, Z: 8}, {X: 175, Y: 255, Z: 15}} {
		p := at(k.X, k.Y, k.Z)
		if got, ok := world.RegionOf(p.X, p.Y, p.Z); !ok || got != k {
			t.Errorf("at(%v) = %v lies in %v, %v", k, p, got, ok)
		}
	}
}

func TestPositionDistances(t *testing.T) {
	p := world.Position{X: 100, Y: 200, Z: -300}
	q := offset(p, 300, 400, 1200)
	if got := p.DistanceSq3D(q); got != 300*300+400*400+1200*1200 {
		t.Errorf("DistanceSq3D = %d", got)
	}
	if got := p.Distance3D(q); got != 1300 {
		t.Errorf("Distance3D = %v, want 1300", got)
	}
	if got := p.Distance2D(q); got != 500 {
		t.Errorf("Distance2D = %v, want 500", got)
	}
	if !p.InRange3D(q, 1300) || p.InRange3D(q, 1299) {
		t.Error("InRange3D must include the edge (1300) and nothing beyond")
	}
	if p.InRange3D(p, -1) {
		t.Error("a negative range matches nothing")
	}
	far := world.Position{X: world.WorldXMax - 1, Y: world.WorldYMax - 1, Z: world.WorldZMax}
	near := world.Position{X: world.WorldXMin, Y: world.WorldYMin, Z: world.WorldZMin}
	if far.DistanceSq3D(near) <= 0 {
		t.Error("DistanceSq3D overflowed across the whole world")
	}
}
