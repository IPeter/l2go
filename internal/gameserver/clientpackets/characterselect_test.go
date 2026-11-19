package clientpackets

import (
	"bytes"
	"testing"
)

// Mobius CharacterSelect.readImpl: d slot, h unknown, d unknown,
// d unknown, d unknown (18 bytes).
const m3CharacterSelectSize = 18

func TestParseCharacterSelect(t *testing.T) {
	var b m3Body
	b.d(2).h(-1).d(0x01020304).d(-5).d(7)
	want := CharacterSelect{Slot: 2, Unknown1: -1, Unknown2: 0x01020304, Unknown3: -5, Unknown4: 7}

	got, err := ParseCharacterSelect(b.b)
	m3Parsed(t, "CharacterSelect", got, err, want)

	// Bytes after the fields are ignored, as in Mobius.
	got, err = ParseCharacterSelect(append(b.b, 0xff, 0x00, 0x13))
	m3Parsed(t, "CharacterSelect with trailing data", got, err, want)

	// Short input: Mobius drops the packet.
	m3Truncations(t, "CharacterSelect", b.b, ParseCharacterSelect)

	// Garbage of the right size parses; the slot is checked by the caller.
	got, err = ParseCharacterSelect(bytes.Repeat([]byte{0xff}, m3CharacterSelectSize))
	m3Parsed(t, "CharacterSelect garbage", got, err, CharacterSelect{Slot: -1, Unknown1: -1, Unknown2: -1, Unknown3: -1, Unknown4: -1})
}

func FuzzParseCharacterSelect(f *testing.F) {
	var b m3Body
	b.d(0).h(0).d(0).d(0).d(0)
	m3Fuzz(f, ParseCharacterSelect, m3CharacterSelectSize, b.b, []byte{1, 0, 0, 0})
}
