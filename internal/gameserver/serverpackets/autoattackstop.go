package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpAutoAttackStop is the opcode of AutoAttackStop.
const OpAutoAttackStop = 0x2c

// NewAutoAttackStopPacket builds AutoAttackStop (Mobius
// AutoAttackStop.writeImpl): the creature with the given object ID leaves
// its combat stance, for example 15 seconds after its last attack.
func NewAutoAttackStopPacket(objectID int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpAutoAttackStop)
	w.WriteInt32(objectID)
	return w.Bytes()
}
