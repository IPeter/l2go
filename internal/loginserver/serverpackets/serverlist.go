package serverpackets

import (
	"net/netip"

	"github.com/IPeter/l2go/internal/packets"
)

// Server types for ServerInfo.Type, from Mobius ServerConfig. The client
// shows them as labels next to the server name.
const (
	ServerTypeNormal = 0x01
	ServerTypeRelax  = 0x02
	ServerTypeTest   = 0x04
	ServerTypeFree   = 0x40
)

// MaxServers is the most entries a ServerList can carry.
const MaxServers = 255

// ServerInfo is one entry of the server list.
type ServerInfo struct {
	ID uint8
	// IP is the address announced to this client. It must be IPv4;
	// anything else is sent as 0.0.0.0.
	IP         netip.Addr
	Port       int
	AgeLimit   uint8 // 0, 15 or 18
	PvP        bool
	Players    uint16
	MaxPlayers uint16
	Up         bool
	Type       uint32 // ServerType* bits
	Brackets   bool   // show brackets in front of the name
}

// NewServerList builds ServerList. lastServer is the ID of the server the
// account played on last (0 for none). At most MaxServers entries are sent.
//
// The trailing "characters per server" section is always empty.
func NewServerList(lastServer uint8, servers []ServerInfo) []byte {
	servers = servers[:min(len(servers), MaxServers)]
	w := packets.NewWriter(3 + 21*len(servers) + 3)
	w.WriteUint8(OpServerList)
	w.WriteUint8(uint8(len(servers))) //nolint:gosec // capped at 255 above
	w.WriteUint8(lastServer)

	for _, s := range servers {
		var ip [4]byte
		if a := s.IP.Unmap(); a.Is4() {
			ip = a.As4()
		}
		w.WriteUint8(s.ID)
		w.WriteBytes(ip[:])
		w.WriteUint32(uint32(s.Port)) //nolint:gosec // ports are validated in config
		w.WriteUint8(s.AgeLimit)
		w.WriteUint8(boolByte(s.PvP))
		w.WriteUint16(s.Players)
		w.WriteUint16(s.MaxPlayers)
		w.WriteUint8(boolByte(s.Up))
		w.WriteUint32(s.Type)
		w.WriteUint8(boolByte(s.Brackets))
	}

	w.WriteUint16(0) // unknown
	w.WriteUint8(0)  // characters per server: no data
	return w.Bytes()
}

func boolByte(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
