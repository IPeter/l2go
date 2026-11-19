package link

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/IPeter/l2go/internal/netutil"
)

// Server defaults, used for zero ServerConfig fields.
const (
	defaultRegisterTimeout = 5 * time.Second
	defaultIdleTimeout     = 30 * time.Second
	defaultWriteTimeout    = 10 * time.Second
)

// ServerConfig configures the login server side of the link.
type ServerConfig struct {
	// GameServers maps the ID (1-255) of each game server that may
	// register to its shared secret. Entries with an ID out of range or an
	// empty secret are ignored.
	GameServers map[int]string

	// RegisterTimeout is how long a new connection has to register before
	// it is closed. Default 5s.
	RegisterTimeout time.Duration
	// IdleTimeout drops a registered game server that sends nothing for
	// this long. It must be longer than the clients' ping interval.
	// Default 30s.
	IdleTimeout time.Duration
	// WriteTimeout bounds each write. Default 10s.
	WriteTimeout time.Duration
}

// Server is the login server side of the link. It accepts game server
// connections, registers them and answers their requests. Its status
// methods are safe for concurrent use. Serve may be called only once.
type Server struct {
	cfg      ServerConfig
	secrets  map[int][]byte // read-only after NewServer
	sessions SessionConsumer
	logger   *slog.Logger

	conns netutil.Registry[net.Conn]

	mu      sync.Mutex
	servers map[int]serverState // one entry per known ID
}

type serverState struct {
	status ServerStatus
	connID uint64 // connection that registered the server, while online
}

// NewServer returns a link server for the game servers in cfg. Session
// checks go to sessions. A nil logger discards logs.
func NewServer(cfg ServerConfig, sessions SessionConsumer, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	cfg.RegisterTimeout = orDefault(cfg.RegisterTimeout, defaultRegisterTimeout)
	cfg.IdleTimeout = orDefault(cfg.IdleTimeout, defaultIdleTimeout)
	cfg.WriteTimeout = orDefault(cfg.WriteTimeout, defaultWriteTimeout)

	s := &Server{
		secrets:  make(map[int][]byte, len(cfg.GameServers)),
		sessions: sessions,
		logger:   logger,
		servers:  make(map[int]serverState, len(cfg.GameServers)),
	}
	for id, secret := range cfg.GameServers {
		switch {
		case id < 1 || id > 255:
			logger.Warn("ignoring game server with an ID out of range", "id", id)
		case secret == "":
			logger.Warn("ignoring game server without a secret", "id", id)
		default:
			s.secrets[id] = []byte(secret)
			s.servers[id] = serverState{}
		}
	}
	cfg.GameServers = nil // the secrets live in s.secrets
	s.cfg = cfg
	return s
}

// Status returns the status of the game server with the given ID. ok is
// false for an ID the server does not know.
func (s *Server) Status(id int) (st ServerStatus, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.servers[id]
	return state.status, ok
}

// Statuses returns the status of every known game server by ID.
func (s *Server) Statuses() map[int]ServerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[int]ServerStatus, len(s.servers))
	for id, state := range s.servers {
		out[id] = state.status
	}
	return out
}

// Serve accepts game servers on ln until ctx is done. It then closes the
// listener and every connection, waits for all handlers to finish and
// returns nil; all game servers are offline by then. It returns an error
// if the listener fails. Serve takes ownership of ln.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.logger.Info("listening for game servers", "addr", ln.Addr().String())

	var wg sync.WaitGroup
	errc := make(chan error, 1)
	wg.Go(func() {
		errc <- netutil.AcceptLoop(ctx, ln, &wg, s.logger, func(c net.Conn) { s.handleConn(ctx, c) })
	})

	var err error
	select {
	case <-ctx.Done():
	case err = <-errc:
	}
	cancel()
	_ = ln.Close()
	s.conns.CloseAll()
	wg.Wait()
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}
	return nil
}

// handleConn serves one game server connection until it fails or ctx is
// done.
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	connID, ok := s.conns.Add(conn)
	if !ok {
		_ = conn.Close()
		return
	}
	defer func() {
		s.conns.Remove(connID)
		_ = conn.Close()
	}()
	logger := s.logger.With("remote", conn.RemoteAddr().String())

	id, err := s.register(conn, connID)
	if err != nil {
		if ctx.Err() == nil {
			logger.Warn("game server registration failed", "err", err)
		}
		return
	}
	defer s.setOffline(id, connID)
	logger = logger.With("gameserver", id)
	logger.Info("game server registered")

	err = s.serveRegistered(conn, id, connID, logger)
	if ctx.Err() == nil {
		logger.Info("game server disconnected", "err", err)
	}
}

// register runs the challenge-response handshake. On success the server is
// online and owned by connID; the caller must set it offline when done.
func (s *Server) register(conn net.Conn, connID uint64) (int, error) {
	if err := conn.SetDeadline(time.Now().Add(s.cfg.RegisterTimeout)); err != nil {
		return 0, err
	}
	ch := challengeMsg{version: protocolVersion}
	if _, err := rand.Read(ch.challenge[:]); err != nil {
		return 0, fmt.Errorf("challenge: %w", err)
	}
	if err := writeMsg(conn, s.cfg.RegisterTimeout, ch); err != nil {
		return 0, fmt.Errorf("send challenge: %w", err)
	}
	msg, err := readMsg(conn)
	if err != nil {
		return 0, fmt.Errorf("read register: %w", err)
	}
	reg, ok := msg.(registerMsg)
	if !ok {
		return 0, fmt.Errorf("expected a register message, got %T", msg)
	}

	id := int(reg.serverID)
	reason := s.checkRegister(reg, ch.challenge)
	if reason == reasonNone && !s.setOnline(id, connID, int(reg.maxPlayers)) {
		reason = ReasonAlreadyRegistered
	}
	if err := writeMsg(conn, s.cfg.WriteTimeout, registerResultMsg{reason: reason}); err != nil {
		if reason == reasonNone {
			s.setOffline(id, connID)
		}
		return 0, fmt.Errorf("send register result: %w", err)
	}
	if reason != reasonNone {
		return 0, fmt.Errorf("server %d: %w", id, &RejectedError{Reason: reason})
	}
	return id, nil
}

// checkRegister checks a registration against the known servers.
func (s *Server) checkRegister(reg registerMsg, challenge [challengeSize]byte) Reason {
	if reg.version != protocolVersion {
		return ReasonUnsupportedVersion
	}
	secret, ok := s.secrets[int(reg.serverID)]
	if !ok {
		return ReasonUnknownServer
	}
	if !hmac.Equal(reg.mac[:], computeMAC(secret, challenge)) {
		return ReasonWrongSecret
	}
	return reasonNone
}

// serveRegistered handles the messages of a registered game server until
// the connection fails.
func (s *Server) serveRegistered(conn net.Conn, id int, connID uint64, logger *slog.Logger) error {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(s.cfg.IdleTimeout)); err != nil {
			return err
		}
		msg, err := readMsg(conn)
		if err != nil {
			return err
		}
		var reply message
		switch m := msg.(type) {
		case statusMsg:
			s.setPlayers(id, connID, int(m.players), int(m.maxPlayers))
		case verifySessionMsg:
			ok := s.sessions != nil && s.sessions.Consume(m.account, m.keys)
			logger.Debug("session check", "account", m.account, "ok", ok)
			reply = verifySessionResultMsg{requestID: m.requestID, ok: ok}
		case pingMsg:
			reply = pongMsg{}
		case pongMsg:
		default:
			return fmt.Errorf("unexpected %T from a game server", msg)
		}
		if reply != nil {
			if err := writeMsg(conn, s.cfg.WriteTimeout, reply); err != nil {
				return err
			}
		}
	}
}

// setOnline marks server id online for connID. It fails if the server is
// already online.
func (s *Server) setOnline(id int, connID uint64, maxPlayers int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.servers[id].status.Online {
		return false
	}
	s.servers[id] = serverState{
		status: ServerStatus{Online: true, MaxPlayers: maxPlayers},
		connID: connID,
	}
	return true
}

// setOffline marks server id offline if connID still owns it.
func (s *Server) setOffline(id int, connID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.servers[id]
	if !state.status.Online || state.connID != connID {
		return
	}
	s.servers[id] = serverState{status: ServerStatus{MaxPlayers: state.status.MaxPlayers}}
}

// setPlayers records a status update from connID.
func (s *Server) setPlayers(id int, connID uint64, players, maxPlayers int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.servers[id]
	if !state.status.Online || state.connID != connID {
		return
	}
	state.status.Players = players
	state.status.MaxPlayers = maxPlayers
	s.servers[id] = state
}

func orDefault(d, def time.Duration) time.Duration {
	if d <= 0 {
		return def
	}
	return d
}
