package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewGGAuth builds GGAuth, the answer to AuthGameGuard. Mobius sends the
// session ID as the response, followed by four zero values.
func NewGGAuth(response int32) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpGGAuth)
	w.WriteInt32(response)
	for range 4 {
		w.WriteUint32(0)
	}
	return w.Bytes()
}
