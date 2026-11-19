package netutil

import (
	"encoding/hex"
	"log/slog"
	"net"
	"net/netip"
)

// Hex logs bytes as a hex string. The string is only built when the record
// is actually logged, so debug dumps cost nothing at higher levels.
type Hex []byte

// LogValue implements slog.LogValuer.
func (h Hex) LogValue() slog.Value { return slog.StringValue(hex.EncodeToString(h)) }

// RemoteIP returns the peer IP of conn with any IPv4-in-IPv6 mapping
// removed, or the zero Addr if it cannot be determined.
func RemoteIP(conn net.Conn) netip.Addr {
	addr := conn.RemoteAddr()
	if addr == nil {
		return netip.Addr{}
	}
	if ta, ok := addr.(*net.TCPAddr); ok {
		return ta.AddrPort().Addr().Unmap()
	}
	ap, err := netip.ParseAddrPort(addr.String())
	if err != nil {
		return netip.Addr{}
	}
	return ap.Addr().Unmap()
}
