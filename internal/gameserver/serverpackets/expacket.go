package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpExPacket is the opcode of the extended server packets. A 16-bit
// sub-opcode follows it (the second ID of Mobius ServerPackets).
const OpExPacket = 0xfe

// newExWriter returns a Writer that starts with the header of the extended
// packet sub, with room for size more bytes.
func newExWriter(sub uint16, size int) *packets.Writer {
	w := packets.NewWriter(3 + size)
	w.WriteUint8(OpExPacket)
	w.WriteUint16(sub)
	return w
}
