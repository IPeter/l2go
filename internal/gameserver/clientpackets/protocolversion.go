// Package clientpackets parses packets sent by the client to the game
// server (Interlude protocol). Data passed to the parsers is the decrypted
// body without the opcode byte.
//
// Layouts follow the readImpl methods of the L2J Mobius Interlude client
// packets. Like Mobius, the parsers ignore bytes after the last field.
package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the client packets (Mobius ClientPackets).
const (
	OpProtocolVersion  = 0x00
	OpAuthLogin        = 0x08
	OpCharacterCreate  = 0x0b
	OpCharacterDelete  = 0x0c
	OpCharacterSelect  = 0x0d
	OpNewCharacter     = 0x0e
	OpCharacterRestore = 0x62
)

// PingVersion is the protocol version a client sends to check that the
// server is up. The server closes the connection without answering.
const PingVersion = -2

// ProtocolVersion is the first packet of a game connection. It is never
// encrypted.
type ProtocolVersion struct {
	Version int32
}

// ParseProtocolVersion parses a ProtocolVersion body. The Interlude client
// sends more data after the version; it is ignored, as in Mobius.
func ParseProtocolVersion(data []byte) (ProtocolVersion, error) {
	r := packets.NewReader(data)
	p := ProtocolVersion{Version: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return ProtocolVersion{}, fmt.Errorf("ProtocolVersion: %w", err)
	}
	return p, nil
}
