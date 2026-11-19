// Package loginserver implements the login server: account login, the game
// server list and the hand-over to a game server (Interlude protocol), plus
// the link that game servers register on.
//
// A client connection goes through three states, as in L2J Mobius:
//
//  1. connected: the server sends Init (session ID, scrambled RSA key,
//     Blowfish key) and waits for AuthGameGuard, which it answers with
//     GGAuth;
//  2. GameGuard ok: RequestAuthLogin carries the RSA-encrypted credentials.
//     A good login gets LoginOk with the login half of new session keys;
//  3. logged in: RequestServerList and RequestServerLogin, both showing the
//     login keys. RequestServerLogin is answered with PlayOk (the play
//     keys) or PlayFail.
//
// As in Mobius, a packet that is unknown, not allowed in the current state,
// too short, or carries an RSA block that does not decrypt is ignored: it
// gets no answer and the connection stays as it was. A packet that fails
// its checksum closes the connection. So does every failure reply
// (LoginFail, AccountKicked, PlayFail); Mobius keeps the connection open
// after LoginFail, but the client reconnects for its next attempt anyway.
package loginserver

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/crypt/rsakey"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/loginserver/clientpackets"
	"github.com/IPeter/l2go/internal/loginserver/serverpackets"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/session"
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

	// errRejected wraps the reason when the server closes a connection
	// after a failure reply. These are logged at info level.
	errRejected = errors.New("request rejected")
)

// Server is the login server. Serve or Run may be called only once.
type Server struct {
	cfg      config.Config
	store    storage.AccountStore
	logger   *slog.Logger
	keys     *rsakey.Pool
	sessions *session.Store
	link     *link.Server
	statuses serverStatuses // s.link, except in tests
	internal []netip.Prefix

	rnd         io.Reader // session IDs, Blowfish keys and XOR keys
	idleTimeout time.Duration
	bcryptCost  int

	clients netutil.Registry[*client]
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
	// GameServers is the number of game servers registered on the link.
	GameServers int
}

// serverStatuses is the game server status the login server reads from
// the link. *link.Server implements it.
type serverStatuses interface {
	Status(id int) (link.ServerStatus, bool)
	Statuses() map[int]link.ServerStatus
}

// New returns a login server for cfg, which should be validated. Accounts
// are read from and created in store.
//
// New generates cfg.LoginServer.RSAKeyPairs RSA key pairs, which takes a
// moment. The game servers in cfg may register on the link with their
// secrets.
func New(cfg config.Config, store storage.AccountStore, logger *slog.Logger) (*Server, error) {
	return newServer(cfg, store, logger, nil)
}

// newServer is New with a ready key pool; nil generates one.
func newServer(cfg config.Config, store storage.AccountStore, logger *slog.Logger, keys *rsakey.Pool) (*Server, error) {
	var internal []netip.Prefix
	for _, n := range cfg.LoginServer.InternalNetworks {
		p, err := netip.ParsePrefix(n)
		if err != nil {
			return nil, fmt.Errorf("loginserver: internal network: %w", err)
		}
		internal = append(internal, p)
	}
	if keys == nil {
		start := time.Now()
		var err error
		keys, err = rsakey.New(cfg.LoginServer.RSAKeyPairs)
		if err != nil {
			return nil, fmt.Errorf("loginserver: %w", err)
		}
		logger.Info("generated RSA key pairs", "count", keys.Len(), "took", time.Since(start).Round(time.Millisecond))
	}
	sessions := session.NewStore(time.Duration(cfg.LoginServer.SessionTTL))
	secrets := make(map[int]string, len(cfg.GameServers))
	for _, gs := range cfg.GameServers {
		secrets[gs.ID] = gs.Secret
	}
	ls := link.NewServer(link.ServerConfig{GameServers: secrets}, sessions, logger.With("component", "link"))
	return &Server{
		cfg:         cfg,
		store:       store,
		logger:      logger,
		keys:        keys,
		sessions:    sessions,
		link:        ls,
		statuses:    ls,
		internal:    internal,
		rnd:         rand.Reader,
		idleTimeout: DefaultIdleTimeout,
		bcryptCost:  defaultBcryptCost,
	}, nil
}

// Stats returns the current counters.
func (s *Server) Stats() Stats {
	online := 0
	for _, st := range s.statuses.Statuses() {
		if st.Online {
			online++
		}
	}
	return Stats{
		AccountsCreated:       s.stats.accountsCreated.Load(),
		AccountCreateFailures: s.stats.accountCreateFailures.Load(),
		Logins:                s.stats.logins.Load(),
		FailedLogins:          s.stats.failedLogins.Load(),
		HackAttempts:          s.stats.hackAttempts.Load(),
		OnlineClients:         s.clients.Len(),
		GameServers:           online,
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

// Serve accepts clients on clientLn and game servers (the link) on gsLn
// until ctx is done. It then closes both listeners and every connection,
// waits for all handlers to finish and returns nil. It returns an error if
// a listener fails. Serve takes ownership of the listeners.
func (s *Server) Serve(ctx context.Context, clientLn, gsLn net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.logger.Info("login server listening", "clients", clientLn.Addr().String(), "gameservers", gsLn.Addr().String())

	var wg sync.WaitGroup
	errc := make(chan error, 2)
	wg.Go(func() {
		errc <- netutil.AcceptLoop(ctx, clientLn, &wg, s.logger, func(c net.Conn) { s.serveClient(ctx, c) })
	})
	wg.Go(func() { errc <- s.link.Serve(ctx, gsLn) })
	wg.Go(func() { s.sessions.Run(ctx) })

	var err error
	select {
	case <-ctx.Done():
	case err = <-errc:
	}
	cancel()
	_ = clientLn.Close()
	s.clients.CloseAll()
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
	c, err := newClient(conn, s.keys.Pick(), s.rnd, s.idleTimeout, logger)
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

	initPacket, err := serverpackets.NewInit(c.sessionID, c.rsa.ScrambledModulus(), c.blowfishKey[:])
	if err != nil {
		logger.Error("cannot build Init", "err", err)
		return
	}
	if err := c.send(initPacket); err != nil {
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
			if errors.Is(err, errRejected) {
				logger.Info("closing client connection", "err", err)
			} else {
				logger.Warn("closing client connection", "err", err)
			}
			return
		}
	}
}

// handlePacket handles one client packet. An error closes the connection.
// Each packet is accepted in one state only. Others are ignored, as Mobius
// LoginPacketHandler does: real clients send packets L2Go does not handle,
// such as the PI agreement requests (0x0e, 0x0f).
func (s *Server) handlePacket(ctx context.Context, c *client, opcode byte, data []byte) error {
	switch {
	case opcode == clientpackets.OpAuthGameGuard && c.state == stateConnected:
		return s.handleAuthGameGuard(c, data)
	case opcode == clientpackets.OpRequestAuthLogin && c.state == stateAuthedGG:
		return s.handleAuthLogin(ctx, c, data)
	case opcode == clientpackets.OpRequestServerList && c.state == stateAuthedLogin:
		return s.handleServerList(c, data)
	case opcode == clientpackets.OpRequestServerLogin && c.state == stateAuthedLogin:
		return s.handleServerLogin(c, data)
	default:
		c.logger.Debug("ignoring packet", "opcode", opcode, "state", c.state.String())
		return nil
	}
}

// ignore logs a packet that cannot be read and drops it, as Mobius does
// when readImpl fails. The connection stays open.
func (c *client) ignore(err error) error {
	c.logger.Debug("ignoring packet", "err", err)
	return nil
}

// reject sends a failure reply and returns an error that closes the
// connection.
func (c *client) reject(reply []byte, format string, args ...any) error {
	if err := c.send(reply); err != nil {
		return err
	}
	return fmt.Errorf("%w: %s", errRejected, fmt.Sprintf(format, args...))
}

func (s *Server) handleAuthGameGuard(c *client, data []byte) error {
	req, err := clientpackets.ParseAuthGameGuard(data)
	if err != nil {
		return c.ignore(err)
	}
	if req.SessionID != c.sessionID {
		s.stats.hackAttempts.Add(1)
		return c.reject(serverpackets.NewLoginFail(serverpackets.LoginFailAccessFailed), "AuthGameGuard with a wrong session ID")
	}
	c.state = stateAuthedGG
	return c.send(serverpackets.NewGGAuth(c.sessionID))
}

func (s *Server) handleAuthLogin(ctx context.Context, c *client, data []byte) error {
	req, err := clientpackets.ParseRequestAuthLogin(data)
	if err != nil {
		return c.ignore(err)
	}
	plain, err := c.rsa.DecryptRaw(req.Encrypted)
	if err != nil {
		// Mobius logs the cipher error and drops the packet.
		s.stats.hackAttempts.Add(1)
		c.logger.Info("cannot decrypt the credentials", "err", err)
		return nil
	}
	creds, err := rsakey.ParseCredentials(plain)
	if err != nil {
		return c.ignore(err)
	}
	username := normalizeUsername(creds.User)
	c.logger.Info("login attempt", "user", username, "newAuth", req.NewAuth())

	acc, err := s.authenticate(ctx, username, creds.Password)
	switch {
	case errors.Is(err, errWrongCredentials):
		return c.reject(serverpackets.NewLoginFail(serverpackets.LoginFailUserOrPassWrong), "user %q: %v", username, err)
	case errors.Is(err, errAccessDenied):
		return c.reject(serverpackets.NewAccountKicked(serverpackets.KickedPermanentlyBanned), "user %q: %v", username, err)
	case err != nil:
		c.logger.Error("login failed", "user", username, "err", err)
		return c.reject(serverpackets.NewLoginFail(serverpackets.LoginFailSystemErrorLoginLater), "user %q: system error", username)
	}

	keys, err := s.sessions.Issue(acc.Username)
	if err != nil {
		c.logger.Error("cannot issue session keys", "user", acc.Username, "err", err)
		return c.reject(serverpackets.NewLoginFail(serverpackets.LoginFailSystemErrorLoginLater), "user %q: system error", username)
	}
	c.account = acc
	c.keys = keys
	c.state = stateAuthedLogin
	c.logger = c.logger.With("user", acc.Username)
	return c.send(serverpackets.NewLoginOk(keys.LoginOK1, keys.LoginOK2))
}

// normalizeUsername returns the account name as it is stored: lower case,
// as Mobius AccountInfo does. The game server sees the same spelling, so
// the session keys match on both sides.
func normalizeUsername(name string) string {
	return strings.ToLower(name)
}

// authenticate checks the credentials and returns the account. username
// must already be normalized. With AutoCreate on, an unknown username gets
// a new player account.
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

// checkLoginKeys reports whether the client shows the login key pair of
// its account's current session keys.
func (s *Server) checkLoginKeys(c *client, loginOK1, loginOK2 int32) bool {
	return s.sessions.CheckLogin(c.account.Username, loginOK1, loginOK2)
}

func (s *Server) handleServerList(c *client, data []byte) error {
	req, err := clientpackets.ParseRequestServerList(data)
	if err != nil {
		return c.ignore(err)
	}
	if !s.checkLoginKeys(c, req.LoginOK1, req.LoginOK2) {
		s.stats.hackAttempts.Add(1)
		return c.reject(serverpackets.NewLoginFail(serverpackets.LoginFailAccessFailed), "RequestServerList with wrong session keys")
	}
	list := s.serverList(netutil.RemoteIP(c.conn), c.accessLevel())
	return c.send(serverpackets.NewServerList(0, list))
}

func (s *Server) handleServerLogin(c *client, data []byte) error {
	req, err := clientpackets.ParseRequestServerLogin(data)
	if err != nil {
		return c.ignore(err)
	}
	if !s.checkLoginKeys(c, req.LoginOK1, req.LoginOK2) {
		s.stats.hackAttempts.Add(1)
		return c.reject(serverpackets.NewLoginFail(serverpackets.LoginFailAccessFailed), "RequestServerLogin with wrong session keys")
	}
	c.logger.Info("client wants to play", "server", req.ServerID)
	if why := s.canPlay(int(req.ServerID), c.accessLevel()); why != "" {
		return c.reject(serverpackets.NewPlayFail(serverpackets.PlayFailServerOverloaded), "server %d: %s", req.ServerID, why)
	}
	return c.send(serverpackets.NewPlayOk(c.keys.PlayOK1, c.keys.PlayOK2))
}

// canPlay checks whether an account with the given access level may join
// game server id, as Mobius GameServerInfo.canLogin does. It returns "" if
// so, or why not.
//
// The server must exist (B3: ID 0 and IDs not in the config are rejected)
// and be registered on the link. Testing servers and full servers only
// take accounts above player level.
func (s *Server) canPlay(id int, level storage.AccessLevel) string {
	gs, ok := s.cfg.GameServerByID(id)
	if !ok {
		s.stats.hackAttempts.Add(1)
		return "unknown server ID"
	}
	st, _ := s.statuses.Status(id)
	switch {
	case !st.Online:
		return "server is offline"
	case gs.Options.Testing && level <= storage.AccessLevelPlayer:
		return "testing server is for admins only"
	}
	if limit := maxPlayers(gs, st); limit > 0 && st.Players >= limit && level <= storage.AccessLevelPlayer {
		return "server is full"
	}
	return ""
}

// maxPlayers returns the player limit the game server reported, or the
// configured one if it reported none.
func maxPlayers(gs config.GameServer, st link.ServerStatus) int {
	if st.MaxPlayers > 0 {
		return st.MaxPlayers
	}
	return int(gs.Options.MaxPlayers)
}

// serverList builds the server list for a client at addr whose account has
// the given access level.
//
// Clients from an internal network get each server's internal IP, others
// its external IP. Up/down, players and the limit come from the link.
// Testing servers are only listed for admins, like Mobius GM-only servers.
func (s *Server) serverList(addr netip.Addr, level storage.AccessLevel) []serverpackets.ServerInfo {
	internal := s.isInternal(addr)
	statuses := s.statuses.Statuses()
	list := make([]serverpackets.ServerInfo, 0, len(s.cfg.GameServers))
	for _, gs := range s.cfg.GameServers {
		if gs.Options.Testing && level <= storage.AccessLevelPlayer {
			continue
		}
		ipText := gs.ExternalIP
		if internal {
			ipText = gs.InternalIP
		}
		ip, _ := netip.ParseAddr(ipText) // validated in config; zero is sent as 0.0.0.0
		st := statuses[gs.ID]
		list = append(list, serverpackets.ServerInfo{
			ID:         uint8(gs.ID), //nolint:gosec // validated to 1-255 in config
			IP:         ip,
			Port:       gs.Port,
			PvP:        true,
			Players:    clampUint16(st.Players),
			MaxPlayers: clampUint16(maxPlayers(gs, st)),
			Up:         st.Online,
			Type:       serverpackets.ServerTypeNormal,
		})
	}
	return list
}

func clampUint16(n int) uint16 {
	return uint16(min(max(n, 0), 0xffff)) //nolint:gosec // clamped
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
