// Package clientpackets parses packets sent by the client to the login
// server (Interlude protocol). Data passed to the parsers is the decrypted
// body without the opcode byte; it may still hold the zero padding and the
// checksum, which the parsers ignore, as Mobius does.
//
// The layouts follow the readImpl methods of the L2J Mobius Interlude
// login server packets.
package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the client packets (Mobius LoginClientPackets).
const (
	OpRequestAuthLogin   = 0x00
	OpRequestServerLogin = 0x02
	OpRequestServerList  = 0x05
	OpAuthGameGuard      = 0x07
)

// RSABlockSize is the size of one RSA-encrypted credentials block.
const RSABlockSize = 128

// RequestAuthLogin carries the RSA-encrypted credentials.
type RequestAuthLogin struct {
	// Encrypted holds one RSA block (128 bytes), or two (256 bytes) in the
	// newer layout.
	Encrypted []byte
}

// NewAuth reports whether the request uses the newer two-block layout.
func (r RequestAuthLogin) NewAuth() bool { return len(r.Encrypted) == 2*RSABlockSize }

// ParseRequestAuthLogin parses a RequestAuthLogin body. Like Mobius, it
// takes two RSA blocks when at least 256 bytes are left (padding and
// checksum included), otherwise one block.
func ParseRequestAuthLogin(data []byte) (RequestAuthLogin, error) {
	n := RSABlockSize
	if len(data) >= 2*RSABlockSize {
		n = 2 * RSABlockSize
	}
	r := packets.NewReader(data)
	block := r.ReadBytes(n)
	if err := r.Err(); err != nil {
		return RequestAuthLogin{}, fmt.Errorf("RequestAuthLogin: %w", err)
	}
	return RequestAuthLogin{Encrypted: block}, nil
}
