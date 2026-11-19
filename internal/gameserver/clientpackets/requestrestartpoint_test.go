package clientpackets

import (
	"math"
	"testing"
)

// Mobius RequestRestartPoint.readImpl: d point type (4 bytes).
func TestParseRequestRestartPoint(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want int32
	}{
		{"village", new(m67Body).d(RestartPointVillage).b, 0},
		{"clan hall", new(m67Body).d(RestartPointClanHall).b, 1},
		{"castle", new(m67Body).d(RestartPointCastle).b, 2},
		{"siege HQ", new(m67Body).d(RestartPointSiegeHQ).b, 3},
		{"fixed", new(m67Body).d(RestartPointFixed).b, 4},
		{"jail", new(m67Body).d(RestartPointJail).b, 27},
		// Not checked here: Mobius sends unknown points to the village.
		{"unknown", new(m67Body).d(6).b, 6},
		{"negative", []byte{0xff, 0xff, 0xff, 0xff}, -1},
		{"garbage", []byte{0x00, 0x00, 0x00, 0x80}, math.MinInt32},
		{"trailing data", new(m67Body).d(2).c(7).c(8).b, 2},
	}
	for _, tt := range tests {
		got, err := ParseRequestRestartPoint(tt.body)
		m67Parsed(t, tt.name, got, err, RequestRestartPoint{Point: tt.want})
	}
	m67Truncations(t, "RequestRestartPoint", new(m67Body).d(RestartPointFixed).b, ParseRequestRestartPoint)
}

func FuzzParseRequestRestartPoint(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(RestartPointVillage).b)
	f.Add(new(m67Body).d(RestartPointJail).c(1).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestRestartPoint(data)
		if m67Fuzzed(t, data, got, err, 4, false, nil) && got.Point != m67Int(data, 0) {
			t.Errorf("% x: point %d", data, got.Point)
		}
	})
}
