package clientpackets

import "testing"

// Mobius StartRotating.readImpl: d degree, d side (8 bytes).

func TestParseStartRotating(t *testing.T) {
	var b m3Body
	b.d(16384).d(-1)
	want := StartRotating{Degree: 16384, Side: -1}
	got, err := ParseStartRotating(b.b)
	m3Parsed(t, "StartRotating", got, err, want)

	got, err = ParseStartRotating(append(b.b, 0x07))
	m3Parsed(t, "StartRotating with trailing data", got, err, want)

	m3Truncations(t, "StartRotating", b.b, ParseStartRotating)

	got, err = ParseStartRotating([]byte{0xff, 0xff, 0xff, 0x7f, 0x00, 0x00, 0x00, 0x80})
	m3Parsed(t, "StartRotating garbage", got, err, StartRotating{Degree: 0x7fffffff, Side: -0x80000000})
}

func FuzzParseStartRotating(f *testing.F) {
	m3Fuzz(f, ParseStartRotating, 8, []byte{0, 0, 0, 0, 1, 0, 0, 0})
}
