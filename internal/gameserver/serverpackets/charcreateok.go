// Package serverpackets builds packets sent by the game server to the
// client (Interlude protocol). Each builder returns the plain body.
//
// Layouts follow the writeImpl methods of the L2J Mobius Interlude server
// packets; opcodes are those of Mobius ServerPackets.
package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// Opcodes of the server packets.
const (
	OpKeyPacket           = 0x00
	OpCharSelectionInfo   = 0x13
	OpLoginFail           = 0x14
	OpNewCharacterSuccess = 0x17
	OpCharCreateOk        = 0x19
	OpCharCreateFail      = 0x1a
)

// NewCharCreateOkPacket builds CharCreateOk.
func NewCharCreateOkPacket() []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpCharCreateOk)
	w.WriteInt32(1) // success
	return w.Bytes()
}
