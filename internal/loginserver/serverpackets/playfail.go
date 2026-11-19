package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewPlayFail builds PlayFail. The Interlude reason is a single byte.
func NewPlayFail(reason PlayFailReason) []byte {
	w := packets.NewWriter(2)
	w.WriteUint8(OpPlayFail)
	w.WriteUint8(uint8(reason))
	return w.Bytes()
}
