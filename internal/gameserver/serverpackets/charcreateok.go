// Package serverpackets builds packets sent by the game server to the
// client (C1 protocol). Each builder returns the plain body.
package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewCharCreateOkPacket builds CharCreateOk.
func NewCharCreateOkPacket() []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(0x25)  // CharCreateOk
	w.WriteUint32(0x01) // success
	return w.Bytes()
}
