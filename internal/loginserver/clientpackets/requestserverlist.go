package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// RequestServerList asks for the game server list.
type RequestServerList struct {
	SessionID []byte
}

// ParseRequestServerList parses a RequestServerList body.
func ParseRequestServerList(data []byte) (RequestServerList, error) {
	r := packets.NewReader(data)
	p := RequestServerList{SessionID: r.ReadBytes(SessionIDSize)}
	if err := r.Err(); err != nil {
		return RequestServerList{}, fmt.Errorf("RequestServerList: %w", err)
	}
	return p, nil
}
