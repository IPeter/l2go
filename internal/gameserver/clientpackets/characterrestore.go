package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// CharacterRestore cancels the deletion of a character.
type CharacterRestore struct {
	Slot int32 // the index in the last CharSelectionInfo
}

// ParseCharacterRestore parses a CharacterRestore body (Mobius
// CharacterRestore.readImpl).
func ParseCharacterRestore(data []byte) (CharacterRestore, error) {
	r := packets.NewReader(data)
	c := CharacterRestore{Slot: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return CharacterRestore{}, fmt.Errorf("CharacterRestore: %w", err)
	}
	return c, nil
}
