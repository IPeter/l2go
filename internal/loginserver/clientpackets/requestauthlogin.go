// Package clientpackets parses packets sent by the client to the login
// server (C1 protocol). Data passed to the parsers is the decrypted body
// without the opcode byte.
package clientpackets

import (
	"bytes"
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the client packets.
const (
	OpRequestAuthLogin  = 0x00
	OpRequestPlay       = 0x02
	OpRequestServerList = 0x05
)

// Username and password are fixed-size, zero-padded ASCII fields.
const (
	usernameSize = 14
	passwordSize = 14
)

// RequestAuthLogin carries the login credentials.
type RequestAuthLogin struct {
	Username string
	Password string
}

// ParseRequestAuthLogin parses a RequestAuthLogin body. The zero padding
// is removed from both fields.
func ParseRequestAuthLogin(data []byte) (RequestAuthLogin, error) {
	r := packets.NewReader(data)
	username := r.ReadBytes(usernameSize)
	password := r.ReadBytes(passwordSize)
	if err := r.Err(); err != nil {
		return RequestAuthLogin{}, fmt.Errorf("RequestAuthLogin: %w", err)
	}
	return RequestAuthLogin{Username: cString(username), Password: cString(password)}, nil
}

// cString returns b up to its first zero byte.
func cString(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}
