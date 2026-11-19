package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpDeleteObject is the opcode of DeleteObject.
const OpDeleteObject = 0x12

// NewDeleteObjectPacket builds DeleteObject (Mobius DeleteObject.writeImpl):
// the client forgets the object, for example when it leaves the viewer's
// visible regions or the world.
func NewDeleteObjectPacket(objectID int32) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpDeleteObject)
	w.WriteInt32(objectID)
	// With 0 a mounted object would only be dismounted; 1 removes it.
	w.WriteInt32(1)
	return w.Bytes()
}
