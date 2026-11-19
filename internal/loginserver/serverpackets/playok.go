package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewPlayOkPacket builds PlayOk. The play keys are fixed in C1 L2Go.
func NewPlayOkPacket() []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(0x07)                           // PlayOk
	w.WriteBytes([]byte{0x34, 0x0b, 0x00, 0x01}) // play key 1
	w.WriteBytes([]byte{0x55, 0x66, 0x77, 0x88}) // play key 2
	return w.Bytes()
}
