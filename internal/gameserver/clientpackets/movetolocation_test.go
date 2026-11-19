package clientpackets

import (
	"bytes"
	"testing"
)

// Mobius MoveToLocation.readImpl: d target x, y, z, d origin x, y, z,
// d movement mode (28 bytes).
const m3MoveToLocationSize = 28

func TestParseMoveToLocation(t *testing.T) {
	var b m3Body
	b.d(-71000).d(258000).d(-3100).d(-71338).d(258271).d(-3104).d(MoveModeMouse)
	want := MoveToLocation{
		TargetX: -71000, TargetY: 258000, TargetZ: -3100,
		OriginX: -71338, OriginY: 258271, OriginZ: -3104,
		Mode: MoveModeMouse,
	}
	got, err := ParseMoveToLocation(b.b)
	m3Parsed(t, "MoveToLocation", got, err, want)

	got, err = ParseMoveToLocation(append(b.b, 0x01))
	m3Parsed(t, "MoveToLocation with trailing data", got, err, want)

	m3Truncations(t, "MoveToLocation", b.b, ParseMoveToLocation)

	// Garbage of the right size parses: the mode and coordinates are
	// checked by the movement code.
	got, err = ParseMoveToLocation(bytes.Repeat([]byte{0x80}, m3MoveToLocationSize))
	const v = -0x7f7f7f80 // 0x80808080
	m3Parsed(t, "MoveToLocation garbage", got, err, MoveToLocation{v, v, v, v, v, v, v})
}

func FuzzParseMoveToLocation(f *testing.F) {
	var b m3Body
	b.d(1).d(2).d(3).d(4).d(5).d(6).d(MoveModeKeyboard)
	m3Fuzz(f, ParseMoveToLocation, m3MoveToLocationSize, b.b)
}
