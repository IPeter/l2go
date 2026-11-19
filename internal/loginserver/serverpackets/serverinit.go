package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewInitPacket builds Init, the first packet of a login connection. It is
// sent without checksum or encryption.
func NewInitPacket() []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(0x00)                           // Init
	w.WriteBytes([]byte{0x9c, 0x77, 0xed, 0x03}) // session ID (fixed)
	w.WriteBytes([]byte{0x5a, 0x78, 0x00, 0x00}) // protocol version 0x785a
	return w.Bytes()
}
