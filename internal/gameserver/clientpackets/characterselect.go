package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// CharacterSelect chooses a character to play from the list.
type CharacterSelect struct {
	Slot int32 // the index in the last CharSelectionInfo
	// Unknown values the client added in C4. Mobius reads and ignores
	// them.
	Unknown1 int16
	Unknown2 int32
	Unknown3 int32
	Unknown4 int32
}

// ParseCharacterSelect parses a CharacterSelect body (Mobius
// CharacterSelect.readImpl). Like Mobius it needs all fields, the unknown
// ones too.
func ParseCharacterSelect(data []byte) (CharacterSelect, error) {
	r := packets.NewReader(data)
	c := CharacterSelect{
		Slot:     r.ReadInt32(),
		Unknown1: int16(r.ReadUint16()), //nolint:gosec // same bits as the Java short
		Unknown2: r.ReadInt32(),
		Unknown3: r.ReadInt32(),
		Unknown4: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return CharacterSelect{}, fmt.Errorf("CharacterSelect: %w", err)
	}
	return c, nil
}
