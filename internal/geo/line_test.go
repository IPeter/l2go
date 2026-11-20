package geo

import (
	"slices"
	"testing"
)

func TestLine2D(t *testing.T) {
	tests := []struct {
		name           string
		x, y, toX, toY int
		want           [][2]int
	}{
		{"shallow", 0, 0, 3, 1, [][2]int{{0, 0}, {1, 0}, {2, 1}, {3, 1}}},
		{"steep", 0, 0, 1, 3, [][2]int{{0, 0}, {0, 1}, {1, 2}, {1, 3}}},
		{"backwards", 3, 1, 0, 0, [][2]int{{3, 1}, {2, 1}, {1, 0}, {0, 0}}},
		{"diagonal", 0, 0, 2, -2, [][2]int{{0, 0}, {1, -1}, {2, -2}}},
		{"one cell", 5, 5, 5, 5, [][2]int{{5, 5}}},
	}
	for _, tc := range tests {
		var got [][2]int
		for l := newLine2D(tc.x, tc.y, tc.toX, tc.toY); l.next(); {
			got = append(got, [2]int{l.x, l.y})
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: cells %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestLine3D(t *testing.T) {
	tests := []struct {
		name                   string
		x, y, z, toX, toY, toZ int
		want                   [][3]int
	}{
		{"x leads", 0, 0, 0, 4, 2, 1, [][3]int{{0, 0, 0}, {1, 1, 0}, {2, 1, 1}, {3, 2, 1}, {4, 2, 1}}},
		{"z leads", 0, 0, 0, 2, 1, 4, [][3]int{{0, 0, 0}, {1, 0, 1}, {1, 1, 2}, {2, 1, 3}, {2, 1, 4}}},
		{"y leads down", 0, 0, 2, 0, -3, 0, [][3]int{{0, 0, 2}, {0, -1, 1}, {0, -2, 1}, {0, -3, 0}}},
	}
	for _, tc := range tests {
		var got [][3]int
		for l := newLine3D(tc.x, tc.y, tc.z, tc.toX, tc.toY, tc.toZ); l.next(); {
			got = append(got, [3]int{l.x, l.y, l.z})
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: points %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestComputeNSWE(t *testing.T) {
	tests := []struct {
		toX, toY int
		want     int
	}{
		{1, 0, nsweEast},
		{-1, 0, nsweWest},
		{0, 1, nsweSouth},
		{0, -1, nsweNorth},
		{1, 1, nsweSouthEast},
		{1, -1, nsweNorthEast},
		{-1, 1, nsweSouthWest},
		{-1, -1, nsweNorthWest},
		{0, 0, 0},
	}
	for _, tc := range tests {
		if got := computeNSWE(0, 0, tc.toX, tc.toY); got != tc.want {
			t.Errorf("computeNSWE(0, 0, %d, %d) = %d, want %d", tc.toX, tc.toY, got, tc.want)
		}
	}
}

// A search takes the smallest window that fits, Mobius's size classes in
// ascending order whatever order the config lists them in.
func TestTakeNodeBuffer(t *testing.T) {
	opts := DefaultOptions()
	opts.BufferSizes = []int{500, 100, 384, 128, 320, 192, 256}
	g := newGeo(opts)
	tests := []struct{ need, want int }{
		{64, 100},
		{100, 100},
		{101, 128},
		{462, 500},
		{500, 500},
		{501, 0},
	}
	for _, tc := range tests {
		b, i := g.takeNodeBuffer(tc.need)
		got := 0
		if b != nil {
			got = b.size
			g.pools[i].Put(b)
		}
		if got != tc.want {
			t.Errorf("window for %d cells = %d, want %d", tc.need, got, tc.want)
		}
	}
}
