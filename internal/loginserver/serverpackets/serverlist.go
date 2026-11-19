package serverpackets

import (
	"net/netip"

	"github.com/IPeter/l2go/internal/packets"
)

// ServerInfo is one entry of the server list.
type ServerInfo struct {
	ID uint8
	// IP is the address announced to this client. It must be IPv4;
	// anything else is sent as 0.0.0.0.
	IP         netip.Addr
	Port       int
	MaxPlayers uint16
	Testing    bool
}

// NewServerListPacket builds ServerList. At most 255 servers are sent.
func NewServerListPacket(servers []ServerInfo) []byte {
	servers = servers[:min(len(servers), 255)]
	w := packets.NewWriter(3 + 21*len(servers))
	w.WriteUint8(0x04)                // ServerList
	w.WriteUint8(uint8(len(servers))) //nolint:gosec // capped at 255 above
	w.WriteUint8(0x00)                // unused

	for _, s := range servers {
		var ip [4]byte
		if a := s.IP.Unmap(); a.Is4() {
			ip = a.As4()
		}
		w.WriteUint8(s.ID)
		w.WriteBytes(ip[:])
		w.WriteUint32(uint32(s.Port)) //nolint:gosec // ports are validated in config
		w.WriteUint8(0x0f)            // age limit
		w.WriteUint8(0x01)            // PvP allowed
		w.WriteUint16(0)              // players online
		w.WriteUint16(s.MaxPlayers)
		if s.Testing {
			w.WriteUint8(0x00)
		} else {
			w.WriteUint8(0x01)
		}
		w.WriteUint32(0x02) // shows a green clock
	}
	return w.Bytes()
}
