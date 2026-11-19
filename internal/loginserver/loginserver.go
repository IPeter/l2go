// Package loginserver implements the login server: account login, the game
// server list and the hand-over to a game server (C1 protocol).
package loginserver

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/crypt/blowfish"
	"github.com/IPeter/l2go/internal/loginserver/clientpackets"
	"github.com/IPeter/l2go/internal/loginserver/serverpackets"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/storage"
)

const (
	// DefaultIdleTimeout closes client connections that send nothing.
	DefaultIdleTimeout = 5 * time.Minute
	writeTimeout       = 10 * time.Second
	defaultBcryptCost  = 10
)

var (
	errWrongCredentials = errors.New("wrong username or password")
	errAccessDenied     = errors.New("account may not log in")
)

// Server is the login server. Serve or Run may be called only once.
type Server struct {
	cfg      config.Config
	store    storage.AccountStore
	logger   *slog.Logger
	cipher   *blowfish.Cipher
	internal []netip.Prefix

	idleTimeout time.Duration
	bcryptCost  int

	clients netutil.Registry[*client]
	links   netutil.Registry[*gameServerLink]
	stats   stats
}

type stats struct {
	accountsCreated       atomic.Uint32
	accountCreateFailures atomic.Uint32
	logins                atomic.Uint32
	failedLogins          atomic.Uint32
	hackAttempts          atomic.Uint32
}

// Stats is a snapshot of the server counters.
type Stats struct {
	AccountsCreated       uint32
	AccountCreateFailures uint32
	Logins                uint32
	FailedLogins          uint32
	HackAttempts          uint32
	OnlineClients         int
	GameServers           int
}

// New returns a login server for cfg, which should be validated. Accounts
// are read from and created in store.
func New(cfg config.Config, store storage.AccountStore, logger *slog.Logger) (*Server, error) {
	cipher, err := blowfish.NewCipher(staticKey)
	if err != nil {
		return nil, fmt.Errorf("loginserver: %w", err)
	}
	var internal []netip.Prefix
	for _, n := range cfg.LoginServer.InternalNetworks {
		p, err := netip.ParsePrefix(n)
		if err != nil {
			return nil, fmt.Errorf("loginserver: internal network: %w", err)
		}
		internal = append(internal, p)
	}
	return &Server{
		cfg:         cfg,
		store:       store,
		logger:      logger,
		cipher:      cipher,
		internal:    internal,
		idleTimeout: DefaultIdleTimeout,
		bcryptCost:  defaultBcryptCost,
	}, nil
}

// Stats returns the current counters.
func (s *Server) Stats() Stats {
	return Stats{
		AccountsCreated:       s.stats.accountsCreated.Load(),
		AccountCreateFailures: s.stats.accountCreateFailures.Load(),
		Logins:                s.stats.logins.Load(),
		FailedLogins:          s.stats.failedLogins.Load(),
		HackAttempts:          s.stats.hackAttempts.Load(),
		OnlineClients:         s.clients.Len(),
		GameServers:           s.links.Len(),
	}
}

// Run listens on the configured addresses and serves until ctx is done.
func (s *Server) Run(ctx context.Context) error {
	var lc net.ListenConfig
	clientLn, err := lc.Listen(ctx, "tcp", s.cfg.LoginServer.ClientListen)
	if err != nil {
		return fmt.Errorf("loginserver: listen for clients: %w", err)
	}
	gsLn, err := lc.Listen(ctx, "tcp", s.cfg.LoginServer.GameServerListen)
	if err != nil {
		_ = clientLn.Close()
		return fmt.Errorf("loginserver: listen for game servers: %w", err)
	}
	return s.Serve(ctx, clientLn, gsLn)
}

// Serve accepts clients on clientLn and game servers on gsLn until ctx is
// done. It then closes both listeners and every connection, waits for all
// handlers to finish and returns nil. It returns an error if a listener
// fails. Serve takes ownership of the listeners.
func (s *Server) Serve(ctx context.Context, clientLn, gsLn net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.logger.Info("login server listening", "clients", clientLn.Addr().String(), "gameservers", gsLn.Addr().String())

	var wg sync.WaitGroup
	errc := make(chan error, 2)
	wg.Go(func() {
		errc <- netutil.AcceptLoop(ctx, clientLn, &wg, s.logger, func(c net.Conn) { s.serveClient(ctx, c) })
	})
	wg.Go(func() {
		errc <- netutil.AcceptLoop(ctx, gsLn, &wg, s.logger, func(c net.Conn) { s.serveGameServer(ctx, c) })
	})

	var err error
	select {
	case <-ctx.Done():
	case err = <-errc:
	}
	cancel()
	_ = clientLn.Close()
	_ = gsLn.Close()
	s.clients.CloseAll()
	s.links.CloseAll()
	wg.Wait()

	st := s.Stats()
	s.logger.Info("login server stopped",
		"logins", st.Logins, "failedLogins", st.FailedLogins,
		"accountsCreated", st.AccountsCreated, "hackAttempts", st.HackAttempts)
	if err != nil {
		return fmt.Errorf("loginserver: %w", err)
	}
	return nil
}

func (s *Server) serveClient(ctx context.Context, conn net.Conn) {
	logger := s.logger.With("remote", conn.RemoteAddr().String())
	c, err := newClient(conn, s.cipher, rand.Reader, s.idleTimeout, logger)
	if err != nil {
		logger.Error("cannot set up client", "err", err)
		_ = conn.Close()
		return
	}
	id, ok := s.clients.Add(c)
	if !ok {
		_ = conn.Close()
		return
	}
	defer func() {
		s.clients.Remove(id)
		_ = c.Close()
	}()
	logger.Info("client connected")

	if err := c.sendPlain(serverpackets.NewInitPacket()); err != nil {
		logger.Info("cannot send Init", "err", err)
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
		if err := s.handlePacket(ctx, c, opcode, data); err != nil {
			logger.Warn("closing client connection", "err", err)
			return
		}
	}
}

// handlePacket handles one client packet. An error closes the connection.
func (s *Server) handlePacket(ctx context.Context, c *client, opcode byte, data []byte) error {
	switch opcode {
	case clientpackets.OpRequestAuthLogin:
		return s.handleAuthLogin(ctx, c, data)
	case clientpackets.OpRequestPlay:
		return s.handleRequestPlay(c, data)
	case clientpackets.OpRequestServerList:
		return s.handleServerList(c, data)
	default:
		c.logger.Warn("unknown packet", "opcode", opcode)
		return nil
	}
}

func (s *Server) handleAuthLogin(ctx context.Context, c *client, data []byte) error {
	req, err := clientpackets.ParseRequestAuthLogin(data)
	if err != nil {
		return err
	}
	c.logger.Info("login attempt", "user", req.Username)

	acc, err := s.authenticate(ctx, req.Username, req.Password)
	var reply []byte
	switch {
	case err == nil:
		c.account = &acc
		reply = serverpackets.NewLoginOkPacket(c.sessionID[:])
	case errors.Is(err, errWrongCredentials):
		reply = serverpackets.NewLoginFailPacket(serverpackets.ReasonUserOrPassWrong)
	case errors.Is(err, errAccessDenied):
		reply = serverpackets.NewLoginFailPacket(serverpackets.ReasonAccessFailed)
	default:
		c.logger.Error("login failed", "user", req.Username, "err", err)
		reply = serverpackets.NewLoginFailPacket(serverpackets.ReasonSystemError)
	}
	return c.send(reply)
}

// authenticate checks the credentials and returns the account. With
// AutoCreate on, an unknown username gets a new player account.
func (s *Server) authenticate(ctx context.Context, username, password string) (storage.Account, error) {
	if username == "" {
		s.stats.failedLogins.Add(1)
		return storage.Account{}, errWrongCredentials
	}
	acc, err := s.store.FindByUsername(ctx, username)
	switch {
	case errors.Is(err, storage.ErrNotFound) && s.cfg.LoginServer.AutoCreate:
		return s.createAccount(ctx, username, password)
	case errors.Is(err, storage.ErrNotFound):
		s.stats.failedLogins.Add(1)
		return storage.Account{}, errWrongCredentials
	case err != nil:
		return storage.Account{}, fmt.Errorf("find account: %w", err)
	}
	return s.checkAccount(acc, password)
}

func (s *Server) checkAccount(acc storage.Account, password string) (storage.Account, error) {
	if bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(password)) != nil {
		s.stats.failedLogins.Add(1)
		return storage.Account{}, errWrongCredentials
	}
	if acc.AccessLevel < storage.AccessLevelPlayer {
		s.stats.failedLogins.Add(1)
		return storage.Account{}, errAccessDenied
	}
	s.stats.logins.Add(1)
	return acc, nil
}

func (s *Server) createAccount(ctx context.Context, username, password string) (storage.Account, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		s.stats.accountCreateFailures.Add(1)
		return storage.Account{}, fmt.Errorf("hash password: %w", err)
	}
	acc := storage.Account{Username: username, Password: string(hash), AccessLevel: storage.AccessLevelPlayer}
	err = s.store.Create(ctx, acc)
	if errors.Is(err, storage.ErrExists) {
		// A concurrent login created it first; check against that account.
		existing, err := s.store.FindByUsername(ctx, username)
		if err != nil {
			return storage.Account{}, fmt.Errorf("find account: %w", err)
		}
		return s.checkAccount(existing, password)
	}
	if err != nil {
		s.stats.accountCreateFailures.Add(1)
		return storage.Account{}, fmt.Errorf("create account: %w", err)
	}
	s.stats.accountsCreated.Add(1)
	s.logger.Info("account created", "user", username)
	return acc, nil
}

func (s *Server) handleRequestPlay(c *client, data []byte) error {
	req, err := clientpackets.ParseRequestPlay(data)
	if err != nil {
		return err
	}
	c.logger.Info("client wants to play", "server", req.ServerID)

	var reply []byte
	switch {
	case !s.canPlay(req.ServerID, c.accessLevel()):
		s.stats.hackAttempts.Add(1)
		reply = serverpackets.NewPlayFailPacket(serverpackets.ReasonAccessFailed)
	case !c.sessionMatches(req.SessionID):
		s.stats.hackAttempts.Add(1)
		reply = serverpackets.NewLoginFailPacket(serverpackets.ReasonAccessFailed)
	default:
		reply = serverpackets.NewPlayOkPacket()
	}
	return c.send(reply)
}

// canPlay reports whether a client with the given access level may join
// game server id. Testing servers only take accounts above player level.
func (s *Server) canPlay(id uint8, level storage.AccessLevel) bool {
	gs, ok := s.cfg.GameServerByID(int(id))
	if !ok {
		return false
	}
	return !gs.Options.Testing || level > storage.AccessLevelPlayer
}

func (s *Server) handleServerList(c *client, data []byte) error {
	req, err := clientpackets.ParseRequestServerList(data)
	if err != nil {
		return err
	}
	if !c.sessionMatches(req.SessionID) {
		s.stats.hackAttempts.Add(1)
		return c.send(serverpackets.NewLoginFailPacket(serverpackets.ReasonAccessFailed))
	}
	return c.send(serverpackets.NewServerListPacket(s.serverList(netutil.RemoteIP(c.conn))))
}

// serverList builds the server list for a client at addr. Clients from an
// internal network get each server's internal IP, others its external IP.
func (s *Server) serverList(addr netip.Addr) []serverpackets.ServerInfo {
	internal := s.isInternal(addr)
	list := make([]serverpackets.ServerInfo, 0, len(s.cfg.GameServers))
	for _, gs := range s.cfg.GameServers {
		ipText := gs.ExternalIP
		if internal {
			ipText = gs.InternalIP
		}
		ip, _ := netip.ParseAddr(ipText) // validated in config; zero is sent as 0.0.0.0
		list = append(list, serverpackets.ServerInfo{
			ID:         uint8(gs.ID), //nolint:gosec // validated to 1-255 in config
			IP:         ip,
			Port:       gs.Port,
			MaxPlayers: gs.Options.MaxPlayers,
			Testing:    gs.Options.Testing,
		})
	}
	return list
}

func (s *Server) isInternal(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, p := range s.internal {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
