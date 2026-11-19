// Package gameserver implements the game server: the client handshake,
// the character list and character creation (C1 protocol), plus the link
// to the login server.
package gameserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/crypt/xor"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/packets"
)

const (
	// DefaultIdleTimeout closes client connections that send nothing.
	DefaultIdleTimeout = 15 * time.Minute
	// MinProtocolVersion is the oldest client protocol accepted (C1).
	MinProtocolVersion = 419

	writeTimeout = 10 * time.Second
	dialTimeout  = 10 * time.Second
)

// Server is one game server. Serve or Run may be called only once.
type Server struct {
	cfg    config.Config
	gs     config.GameServer
	logger *slog.Logger

	idleTimeout time.Duration

	clients      netutil.Registry[*client]
	hackAttempts atomic.Uint32
}

// New returns the game server with the given ID from cfg, which should be
// validated.
func New(cfg config.Config, id int, logger *slog.Logger) (*Server, error) {
	gs, ok := cfg.GameServerByID(id)
	if !ok {
		return nil, fmt.Errorf("gameserver: no game server with id %d in the config", id)
	}
	return &Server{
		cfg:         cfg,
		gs:          gs,
		logger:      logger.With("server", gs.ID),
		idleTimeout: DefaultIdleTimeout,
	}, nil
}

// OnlineClients returns the number of connected clients.
func (s *Server) OnlineClients() int { return s.clients.Len() }

// Run connects to the login server, listens for clients and serves until
// ctx is done. It fails if the login server cannot be reached.
func (s *Server) Run(ctx context.Context) error {
	addr := s.cfg.LoginServer.GameServerAddr()
	d := net.Dialer{Timeout: dialTimeout}
	link, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("gameserver: connect to login server at %s: %w", addr, err)
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.gs.ListenAddr())
	if err != nil {
		_ = link.Close()
		return fmt.Errorf("gameserver: listen for clients: %w", err)
	}
	return s.Serve(ctx, ln, link)
}

// Serve accepts clients on ln and talks to the login server over link
// until ctx is done. It then closes the listener, the link and every
// connection, waits for all handlers to finish and returns nil. It returns
// an error if the listener fails. Serve takes ownership of ln and link.
func (s *Server) Serve(ctx context.Context, ln net.Listener, link net.Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.logger.Info("game server listening", "name", s.gs.Name, "addr", ln.Addr().String(), "loginserver", link.RemoteAddr().String())

	var wg sync.WaitGroup
	wg.Go(func() { s.serveLoginLink(ctx, link) })
	errc := make(chan error, 1)
	wg.Go(func() {
		errc <- netutil.AcceptLoop(ctx, ln, &wg, s.logger, func(c net.Conn) { s.serveClient(ctx, c) })
	})

	var err error
	select {
	case <-ctx.Done():
	case err = <-errc:
	}
	cancel()
	_ = ln.Close()
	_ = link.Close()
	s.clients.CloseAll()
	wg.Wait()

	s.logger.Info("game server stopped", "hackAttempts", s.hackAttempts.Load())
	if err != nil {
		return fmt.Errorf("gameserver: %w", err)
	}
	return nil
}

// registerPacket announces the game server on the C1-era link.
var registerPacket = []byte{0x00, 0x01, 0x02}

// serveLoginLink registers with the login server and logs what it sends.
// If the link drops, clients keep being served.
func (s *Server) serveLoginLink(ctx context.Context, link net.Conn) {
	if err := packets.WritePacket(link, registerPacket); err != nil {
		if ctx.Err() == nil {
			s.logger.Warn("cannot register with the login server", "err", err)
		}
		return
	}
	for {
		body, err := packets.ReadPacket(link, packets.MaxBodySize)
		if err != nil {
			if ctx.Err() == nil {
				s.logger.Warn("lost connection to the login server", "err", err)
			}
			return
		}
		s.logger.Debug("login server packet", "data", netutil.Hex(body))
		s.logger.Warn("unknown login server packet", "opcode", body[0])
	}
}

// supportedProtocol reports whether a client protocol version is accepted.
func supportedProtocol(v uint32) bool { return v >= MinProtocolVersion }

func (s *Server) serveClient(ctx context.Context, conn net.Conn) {
	logger := s.logger.With("remote", conn.RemoteAddr().String())
	c := newClient(conn, s.idleTimeout, logger)
	id, ok := s.clients.Add(c)
	if !ok {
		_ = conn.Close()
		return
	}
	defer func() {
		s.clients.Remove(id)
		_ = c.Close()
	}()
	logger.Info("client connected", "online", s.clients.Len())

	if err := s.handshake(c); err != nil {
		if ctx.Err() == nil {
			logger.Info("handshake failed", "err", err)
		}
		return
	}
	for {
		opcode, data, err := c.receive()
		if err != nil {
			if ctx.Err() == nil {
				logger.Info("client disconnected", "err", err)
			}
			return
		}
		if err := s.handlePacket(c, opcode, data); err != nil {
			logger.Warn("closing client connection", "err", err)
			return
		}
	}
}

// handshake checks the protocol version and sends the XOR key.
func (s *Server) handshake(c *client) error {
	_, data, err := c.receivePlain()
	if err != nil {
		return err
	}
	pv, err := clientpackets.ParseProtocolVersion(data)
	if err != nil {
		return err
	}
	if !supportedProtocol(pv.Version) {
		return fmt.Errorf("protocol version %d is older than %d", pv.Version, MinProtocolVersion)
	}
	return c.sendPlain(serverpackets.NewCryptInitPacket(xor.StaticKey))
}

// handlePacket handles one client packet. An error closes the connection.
func (s *Server) handlePacket(c *client, opcode byte, data []byte) error {
	switch opcode {
	case clientpackets.OpAuthLogin:
		c.logger.Info("client requests the character list")
		return c.send(serverpackets.NewCharListPacket())

	case clientpackets.OpNewCharacter:
		return c.send(serverpackets.NewCharTemplatePacket())

	case clientpackets.OpCharacterCreate:
		ch, err := clientpackets.ParseCharacterCreate(data)
		if err != nil {
			s.hackAttempts.Add(1)
			return err
		}
		c.logger.Info("character created", "name", ch.Name, "race", ch.Race, "class", ch.ClassID)
		if err := c.send(serverpackets.NewCharCreateOkPacket()); err != nil {
			return err
		}
		return c.send(serverpackets.NewCharListPacket())

	default:
		c.logger.Warn("unknown packet", "opcode", opcode)
		return nil
	}
}
