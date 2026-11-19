package clientpackets

import (
	"bytes"
	"testing"
)

// Mobius ValidatePosition.readImpl: d x, y, z, d heading, d vehicle ID
// (20 bytes).
const m3ValidatePositionSize = 20

func TestParseValidatePosition(t *testing.T) {
	var b m3Body
	b.d(-71338).d(258271).d(-3104).d(49152).d(0)
	want := ValidatePosition{X: -71338, Y: 258271, Z: -3104, Heading: 49152}
	got, err := ParseValidatePosition(b.b)
	m3Parsed(t, "ValidatePosition", got, err, want)

	got, err = ParseValidatePosition(append(b.b, 0x00, 0x00, 0x00, 0x00))
	m3Parsed(t, "ValidatePosition with trailing data", got, err, want)

	m3Truncations(t, "ValidatePosition", b.b, ParseValidatePosition)

	got, err = ParseValidatePosition(bytes.Repeat([]byte{0xff}, m3ValidatePositionSize))
	m3Parsed(t, "ValidatePosition garbage", got, err, ValidatePosition{-1, -1, -1, -1, -1})
}

func FuzzParseValidatePosition(f *testing.F) {
	var b m3Body
	b.d(1).d(2).d(3).d(4).d(5)
	m3Fuzz(f, ParseValidatePosition, m3ValidatePositionSize, b.b)
}
