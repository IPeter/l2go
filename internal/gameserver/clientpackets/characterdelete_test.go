package clientpackets

import "testing"

// Mobius CharacterDelete.readImpl and CharacterRestore.readImpl: d slot.

func TestParseCharacterDelete(t *testing.T) {
	body := []byte{0x03, 0x00, 0x00, 0x00}
	got, err := ParseCharacterDelete(body)
	m3Parsed(t, "CharacterDelete", got, err, CharacterDelete{Slot: 3})

	got, err = ParseCharacterDelete(append(body, 0xee))
	m3Parsed(t, "CharacterDelete with trailing data", got, err, CharacterDelete{Slot: 3})

	m3Truncations(t, "CharacterDelete", body, ParseCharacterDelete)

	got, err = ParseCharacterDelete([]byte{0xfe, 0xff, 0xff, 0xff})
	m3Parsed(t, "CharacterDelete garbage", got, err, CharacterDelete{Slot: -2})
}

func FuzzParseCharacterDelete(f *testing.F) {
	m3Fuzz(f, ParseCharacterDelete, 4, []byte{0, 0, 0, 0}, []byte{1, 2, 3})
}

func TestParseCharacterRestore(t *testing.T) {
	body := []byte{0x01, 0x00, 0x00, 0x00}
	got, err := ParseCharacterRestore(body)
	m3Parsed(t, "CharacterRestore", got, err, CharacterRestore{Slot: 1})

	got, err = ParseCharacterRestore(append(body, 0x00, 0x00))
	m3Parsed(t, "CharacterRestore with trailing data", got, err, CharacterRestore{Slot: 1})

	m3Truncations(t, "CharacterRestore", body, ParseCharacterRestore)

	got, err = ParseCharacterRestore([]byte{0x00, 0x00, 0x00, 0x80})
	m3Parsed(t, "CharacterRestore garbage", got, err, CharacterRestore{Slot: -0x80000000})
}

func FuzzParseCharacterRestore(f *testing.F) {
	m3Fuzz(f, ParseCharacterRestore, 4, []byte{0, 0, 0, 0}, []byte{9})
}
