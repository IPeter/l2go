package clientpackets

import "testing"

// Mobius FinishRotating.readImpl: d degree, d unknown (8 bytes).

func TestParseFinishRotating(t *testing.T) {
	var b m3Body
	b.d(32768).d(0)
	want := FinishRotating{Degree: 32768}
	got, err := ParseFinishRotating(b.b)
	m3Parsed(t, "FinishRotating", got, err, want)

	got, err = ParseFinishRotating(append(b.b, 0x07, 0x08))
	m3Parsed(t, "FinishRotating with trailing data", got, err, want)

	m3Truncations(t, "FinishRotating", b.b, ParseFinishRotating)

	got, err = ParseFinishRotating([]byte{0x01, 0x02, 0x03, 0x04, 0xff, 0xff, 0xff, 0xff})
	m3Parsed(t, "FinishRotating garbage", got, err, FinishRotating{Degree: 0x04030201, Unknown: -1})
}

func FuzzParseFinishRotating(f *testing.F) {
	m3Fuzz(f, ParseFinishRotating, 8, []byte{0, 0x80, 0, 0, 0, 0, 0, 0})
}
