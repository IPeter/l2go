package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpAutoAttackStart is the opcode of AutoAttackStart.
const OpAutoAttackStart = 0x2b

// NewAutoAttackStartPacket builds AutoAttackStart (Mobius
// AutoAttackStart.writeImpl): the creature with the given object ID enters
// its combat stance. Mobius names the field the target, but it is the
// creature itself.
func NewAutoAttackStartPacket(objectID int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpAutoAttackStart)
	w.WriteInt32(objectID)
	return w.Bytes()
}
