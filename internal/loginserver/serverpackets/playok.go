package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewPlayOk builds PlayOk with the play key pair of the session.
func NewPlayOk(playOK1, playOK2 int32) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpPlayOk)
	w.WriteInt32(playOK1)
	w.WriteInt32(playOK2)
	return w.Bytes()
}
