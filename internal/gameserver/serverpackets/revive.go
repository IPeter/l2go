package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpRevive is the opcode of Revive.
const OpRevive = 0x07

// NewRevivePacket builds Revive (Mobius Revive.writeImpl): the dead creature
// with the given object ID gets up.
func NewRevivePacket(objectID int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpRevive)
	w.WriteInt32(objectID)
	return w.Bytes()
}
