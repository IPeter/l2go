package clientpackets

import (
	"bytes"
	"testing"
)

// Mobius EnterWorld.readImpl: b[32], d, d, d, d, b[32], d, then 5 × 4
// unsigned bytes of traceroute (104 bytes).
const m3EnterWorldSize = 104

// m3EnterWorldBody is an EnterWorld body with the given traceroute and
// recognisable filler in the unknown fields.
func m3EnterWorldBody(tracert [5][4]byte) []byte {
	var b m3Body
	b.b = append(b.b, bytes.Repeat([]byte{0xaa}, 32)...)
	b.d(1).d(2).d(3).d(4)
	b.b = append(b.b, bytes.Repeat([]byte{0xbb}, 32)...)
	b.d(-1)
	for _, ip := range tracert {
		b.b = append(b.b, ip[:]...)
	}
	return b.b
}

func TestParseEnterWorld(t *testing.T) {
	tracert := [5][4]byte{
		{192, 168, 0, 1},
		{10, 0, 0, 138},
		{0, 0, 0, 0},
		{255, 255, 255, 255},
		{1, 2, 3, 4},
	}
	body := m3EnterWorldBody(tracert)
	if len(body) != m3EnterWorldSize {
		t.Fatalf("test body has %d bytes, want %d", len(body), m3EnterWorldSize)
	}

	got, err := ParseEnterWorld(body)
	m3Parsed(t, "EnterWorld", got, err, EnterWorld{Tracert: tracert})

	got, err = ParseEnterWorld(append(body, 0x01, 0x02))
	m3Parsed(t, "EnterWorld with trailing data", got, err, EnterWorld{Tracert: tracert})

	// Short input, down to an empty body: Mobius drops the packet.
	m3Truncations(t, "EnterWorld", body, ParseEnterWorld)

	// Garbage of the right size parses; only the traceroute is kept.
	garbage := bytes.Repeat([]byte{0x5a}, m3EnterWorldSize)
	var want EnterWorld
	for i := range want.Tracert {
		want.Tracert[i] = [4]byte{0x5a, 0x5a, 0x5a, 0x5a}
	}
	got, err = ParseEnterWorld(garbage)
	m3Parsed(t, "EnterWorld garbage", got, err, want)
}

func FuzzParseEnterWorld(f *testing.F) {
	m3Fuzz(f, ParseEnterWorld, m3EnterWorldSize, m3EnterWorldBody([5][4]byte{}), make([]byte, 103))
}
