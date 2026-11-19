package clientpackets

import (
	"bytes"
	"testing"
)

// Mobius CannotMoveAnymore.readImpl: d x, y, z, d heading (16 bytes).
const m3CannotMoveAnymoreSize = 16

func TestParseCannotMoveAnymore(t *testing.T) {
	var b m3Body
	b.d(100).d(-200).d(-3000).d(1024)
	want := CannotMoveAnymore{X: 100, Y: -200, Z: -3000, Heading: 1024}
	got, err := ParseCannotMoveAnymore(b.b)
	m3Parsed(t, "CannotMoveAnymore", got, err, want)

	got, err = ParseCannotMoveAnymore(append(b.b, 0x09))
	m3Parsed(t, "CannotMoveAnymore with trailing data", got, err, want)

	m3Truncations(t, "CannotMoveAnymore", b.b, ParseCannotMoveAnymore)

	got, err = ParseCannotMoveAnymore(bytes.Repeat([]byte{0x01}, m3CannotMoveAnymoreSize))
	const v = 0x01010101
	m3Parsed(t, "CannotMoveAnymore garbage", got, err, CannotMoveAnymore{v, v, v, v})
}

func FuzzParseCannotMoveAnymore(f *testing.F) {
	var b m3Body
	b.d(1).d(2).d(3).d(4)
	m3Fuzz(f, ParseCannotMoveAnymore, m3CannotMoveAnymoreSize, b.b)
}
