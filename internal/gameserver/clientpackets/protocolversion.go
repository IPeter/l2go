// Package clientpackets parses packets sent by the client to the game
// server (C1 protocol). Data passed to the parsers is the decrypted body
// without the opcode byte.
package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the client packets.
const (
	OpProtocolVersion = 0x00
	OpAuthLogin       = 0x08
	OpCharacterCreate = 0x0b
	OpNewCharacter    = 0x0e
)

// ProtocolVersion is the first packet of a game connection.
type ProtocolVersion struct {
	Version uint32
}

// ParseProtocolVersion parses a ProtocolVersion body.
func ParseProtocolVersion(data []byte) (ProtocolVersion, error) {
	r := packets.NewReader(data)
	p := ProtocolVersion{Version: r.ReadUint32()}
	if err := r.Err(); err != nil {
		return ProtocolVersion{}, fmt.Errorf("ProtocolVersion: %w", err)
	}
	return p, nil
}
