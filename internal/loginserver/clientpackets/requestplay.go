package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// SessionIDSize is the length of the session ID echoed by the client.
const SessionIDSize = 8

// RequestPlay asks to play on a game server.
type RequestPlay struct {
	SessionID []byte
	ServerID  uint8
}

// ParseRequestPlay parses a RequestPlay body.
func ParseRequestPlay(data []byte) (RequestPlay, error) {
	r := packets.NewReader(data)
	p := RequestPlay{
		SessionID: r.ReadBytes(SessionIDSize),
		ServerID:  r.ReadUint8(),
	}
	if err := r.Err(); err != nil {
		return RequestPlay{}, fmt.Errorf("RequestPlay: %w", err)
	}
	return p, nil
}
