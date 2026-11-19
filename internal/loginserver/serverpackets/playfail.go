package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewPlayFailPacket builds PlayFail with one of the Reason constants.
func NewPlayFailPacket(reason uint32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(0x06) // PlayFail
	w.WriteUint32(reason)
	return w.Bytes()
}
