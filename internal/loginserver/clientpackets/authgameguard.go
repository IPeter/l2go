package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// AuthGameGuard is the GameGuard handshake. Only the session ID is used;
// the four GameGuard values after it are read and ignored.
type AuthGameGuard struct {
	SessionID int32
}

// ParseAuthGameGuard parses an AuthGameGuard body (at least 20 bytes).
func ParseAuthGameGuard(data []byte) (AuthGameGuard, error) {
	r := packets.NewReader(data)
	p := AuthGameGuard{SessionID: r.ReadInt32()}
	for range 4 {
		r.ReadUint32()
	}
	if err := r.Err(); err != nil {
		return AuthGameGuard{}, fmt.Errorf("AuthGameGuard: %w", err)
	}
	return p, nil
}
