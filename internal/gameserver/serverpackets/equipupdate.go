package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpEquipUpdate is the opcode of EquipUpdate.
const OpEquipUpdate = 0x4b

// EquipUpdate tells the client that an item was put on or taken off.
// Mobius defines the packet but never sends it: it shows equipment changes
// with InventoryUpdate and UserInfo.
type EquipUpdate struct {
	Change   int32 // sent as given
	ObjectID int32
	// BodyPart is the body part mask of the item template, as in ItemInfo.
	// The packet carries the client's slot number for it.
	BodyPart int32
}

// NewEquipUpdatePacket builds EquipUpdate (Mobius EquipUpdate.writeImpl).
func NewEquipUpdatePacket(e EquipUpdate) []byte {
	w := packets.NewWriter(13)
	w.WriteUint8(OpEquipUpdate)
	w.WriteInt32(e.Change)
	w.WriteInt32(e.ObjectID)
	w.WriteInt32(equipUpdateSlot(e.BodyPart))
	return w.Bytes()
}

// equipUpdateSlot is the slot number EquipUpdate sends for a body part mask,
// as the Mobius switch on the template's body part gives it. Every other
// mask is 0, among them the two-sided ones of earrings ("rear;lear") and
// rings, full armor, underwear and hair2.
func equipUpdateSlot(mask int32) int32 {
	switch mask {
	case 0x0004: // L_EAR
		return 0x01
	case 0x0002: // R_EAR
		return 0x02
	case 0x0008: // NECK
		return 0x03
	case 0x0010: // R_FINGER
		return 0x04
	case 0x0020: // L_FINGER
		return 0x05
	case 0x0040: // HEAD
		return 0x06
	case 0x0080: // R_HAND
		return 0x07
	case 0x0100: // L_HAND
		return 0x08
	case 0x0200: // GLOVES
		return 0x09
	case 0x0400: // CHEST
		return 0x0a
	case 0x0800: // LEGS
		return 0x0b
	case 0x1000: // FEET
		return 0x0c
	case 0x2000: // BACK
		return 0x0d
	case 0x4000: // LR_HAND
		return 0x0e
	case 0x10000: // HAIR
		return 0x0f
	}
	return 0
}
