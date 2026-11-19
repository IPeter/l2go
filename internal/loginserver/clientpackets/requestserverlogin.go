package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// RequestServerLogin asks to play on a game server. It shows the login key
// pair from LoginOk.
type RequestServerLogin struct {
	LoginOK1, LoginOK2 int32
	ServerID           uint8
}

// ParseRequestServerLogin parses a RequestServerLogin body (at least 9
// bytes).
func ParseRequestServerLogin(data []byte) (RequestServerLogin, error) {
	r := packets.NewReader(data)
	p := RequestServerLogin{
		LoginOK1: r.ReadInt32(),
		LoginOK2: r.ReadInt32(),
		ServerID: r.ReadUint8(),
	}
	if err := r.Err(); err != nil {
		return RequestServerLogin{}, fmt.Errorf("RequestServerLogin: %w", err)
	}
	return p, nil
}
