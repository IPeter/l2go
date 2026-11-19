package clientpackets

import "testing"

// Mobius RequestActionUse.readImpl: d action ID, d ctrl (== 1),
// c shift (== 1) (9 bytes).
const m3RequestActionUseSize = 9

func TestParseRequestActionUse(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestActionUse
	}{
		{"sit", new(m3Body).d(ActionUseSitStand).d(0).c(0).b, RequestActionUse{ActionID: 0}},
		{"walk/run", new(m3Body).d(ActionUseWalkRun).d(0).c(0).b, RequestActionUse{ActionID: 1}},
		{"ctrl and shift", new(m3Body).d(16).d(1).c(1).b, RequestActionUse{ActionID: 16, CtrlPressed: true, ShiftPressed: true}},
		// Mobius compares with 1: other values are not pressed.
		{"ctrl 2, shift 255", new(m3Body).d(16).d(2).c(0xff).b, RequestActionUse{ActionID: 16}},
		{"trailing data", new(m3Body).d(1).d(1).c(0).c(9).b, RequestActionUse{ActionID: 1, CtrlPressed: true}},
		{"garbage", []byte{0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0x00, 0x00, 0x01}, RequestActionUse{ActionID: -1, CtrlPressed: true, ShiftPressed: true}},
	}
	for _, tt := range tests {
		got, err := ParseRequestActionUse(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestActionUse", new(m3Body).d(0).d(1).c(1).b, ParseRequestActionUse)
}

func FuzzParseRequestActionUse(f *testing.F) {
	m3Fuzz(f, ParseRequestActionUse, m3RequestActionUseSize, new(m3Body).d(1).d(0).c(0).b)
}
