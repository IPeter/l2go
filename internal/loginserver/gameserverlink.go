package loginserver

import (
	"context"
	"net"

	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/packets"
)

// gameServerLink is a connected game server. The C1-era link only carries
// a register packet; Plan 03 replaces it with an authenticated protocol.
type gameServerLink struct {
	conn net.Conn
}

func (l *gameServerLink) Close() error { return l.conn.Close() }

// opRegister is the only game server packet L2Go sends today.
const opRegister = 0x00

func (s *Server) serveGameServer(ctx context.Context, conn net.Conn) {
	logger := s.logger.With("gameserver", conn.RemoteAddr().String())
	link := &gameServerLink{conn: conn}
	id, ok := s.links.Add(link)
	if !ok {
		_ = conn.Close()
		return
	}
	defer func() {
		s.links.Remove(id)
		_ = link.Close()
	}()
	logger.Info("game server connected")

	for {
		body, err := packets.ReadPacket(conn, packets.MaxBodySize)
		if err != nil {
			if ctx.Err() == nil {
				logger.Info("game server disconnected", "err", err)
			}
			return
		}
		logger.Debug("game server packet", "data", netutil.Hex(body))
		switch body[0] {
		case opRegister:
			logger.Info("game server sent a register request")
		default:
			logger.Warn("unknown game server packet", "opcode", body[0])
		}
	}
}
