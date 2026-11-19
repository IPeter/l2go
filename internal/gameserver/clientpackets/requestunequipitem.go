package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestUnEquipItem is the opcode of RequestUnEquipItem.
const OpRequestUnEquipItem = 0x11

// RequestUnEquipItem takes off the item worn in a slot, for example when
// the player drags it out of the equipment window.
type RequestUnEquipItem struct {
	// Slot is the slot to empty, as the client sends it. Mobius reads it as a
	// paperdoll slot index (BodyPart.fromPaperdollSlot; the numbering of
	// storage.PaperdollSlot, 7 is the right hand) and does nothing for a
	// value without a body part: below 0, above 16, or the face slot 14. The
	// Mobius C4 version reads the same int as a body part mask instead.
	Slot int32
}

// ParseRequestUnEquipItem parses a RequestUnEquipItem body (Mobius
// RequestUnEquipItem.readImpl).
func ParseRequestUnEquipItem(data []byte) (RequestUnEquipItem, error) {
	r := packets.NewReader(data)
	u := RequestUnEquipItem{Slot: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestUnEquipItem{}, fmt.Errorf("RequestUnEquipItem: %w", err)
	}
	return u, nil
}
