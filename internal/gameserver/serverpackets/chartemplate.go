package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewCharTemplatePacket builds CharTemplate without templates; the C1
// client does not need them.
func NewCharTemplatePacket() []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(0x23)  // CharTemplate
	w.WriteUint32(0x00) // template count
	return w.Bytes()
}
