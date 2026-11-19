package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewLoginFailPacket builds LoginFail with one of the Reason constants.
func NewLoginFailPacket(reason uint32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(0x01) // LoginFail
	w.WriteUint32(reason)
	return w.Bytes()
}
