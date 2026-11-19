package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// RequestServerList asks for the game server list. It shows the login key
// pair from LoginOk.
type RequestServerList struct {
	LoginOK1, LoginOK2 int32
}

// ParseRequestServerList parses a RequestServerList body (at least 8
// bytes; the byte after the keys is ignored).
func ParseRequestServerList(data []byte) (RequestServerList, error) {
	r := packets.NewReader(data)
	p := RequestServerList{LoginOK1: r.ReadInt32(), LoginOK2: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestServerList{}, fmt.Errorf("RequestServerList: %w", err)
	}
	return p, nil
}
