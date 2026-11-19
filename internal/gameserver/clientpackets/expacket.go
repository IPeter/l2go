package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpEx is the opcode of the extended client packets. A 16-bit sub-opcode
// follows it (Mobius ExClientPackets).
const OpEx = 0xd0

// Sub-opcodes of the extended client packets.
const (
	ExOpRequestManorList = 0x08
)

// ExPacket is an extended client packet split into its sub-opcode and the
// data after it.
type ExPacket struct {
	SubOpcode uint16
	// Body is the data after the sub-opcode, the input of the parser for
	// that sub-opcode. It shares memory with the data given to
	// ParseExPacket.
	Body []byte
}

// ParseExPacket splits the body of an extended packet (the data after the
// 0xD0 opcode) into its sub-opcode, an unsigned little-endian short as
// Mobius GamePacketHandler reads it, and the rest.
func ParseExPacket(data []byte) (ExPacket, error) {
	r := packets.NewReader(data)
	sub := r.ReadUint16()
	if err := r.Err(); err != nil {
		return ExPacket{}, fmt.Errorf("extended packet: %w", err)
	}
	return ExPacket{SubOpcode: sub, Body: data[2:]}, nil
}
