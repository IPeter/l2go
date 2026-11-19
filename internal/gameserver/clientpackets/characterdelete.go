package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// CharacterDelete marks a character for deletion.
type CharacterDelete struct {
	Slot int32 // the index in the last CharSelectionInfo
}

// ParseCharacterDelete parses a CharacterDelete body (Mobius
// CharacterDelete.readImpl).
func ParseCharacterDelete(data []byte) (CharacterDelete, error) {
	r := packets.NewReader(data)
	c := CharacterDelete{Slot: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return CharacterDelete{}, fmt.Errorf("CharacterDelete: %w", err)
	}
	return c, nil
}
