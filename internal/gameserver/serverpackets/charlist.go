package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewCharListPacket builds an empty CharList.
func NewCharListPacket() []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(0x1f)  // CharList
	w.WriteUint32(0x00) // character count
	return w.Bytes()
}
