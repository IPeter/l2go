package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewLoginOkPacket builds LoginOk. sessionID must hold at least 8 bytes;
// the first 8 are sent.
func NewLoginOkPacket(sessionID []byte) []byte {
	w := packets.NewWriter(33)
	w.WriteUint8(0x03)           // LoginOk
	w.WriteBytes(sessionID[0:4]) // session ID 1/2
	w.WriteBytes(sessionID[4:8]) // session ID 2/2
	w.WriteUint32(0x00)
	w.WriteUint32(0x00)
	w.WriteUint32(0x000003ea)
	w.WriteUint32(0x00)
	w.WriteUint32(0x00)
	w.WriteUint32(0x02)
	return w.Bytes()
}
