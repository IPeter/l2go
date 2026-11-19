// Package gameserver implements the game server for the Interlude client
// (protocol 746): the handshake, the session check with the login server,
// the character list and character creation with its Lua script hook.
package gameserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/storage"
)

const (
	// DefaultIdleTimeout closes client connections that send nothing.
	DefaultIdleTimeout = 15 * time.Minute
	// DefaultVerifyTimeout bounds the session key check with the login
	// server.
	DefaultVerifyTimeout = 5 * time.Second

	writeTimeout = 10 * time.Second
)

// Scripts runs script hooks. *scripting.Engine implements it.
type Scripts interface {
	Call(ctx context.Context, ev scripting.Event) (scripting.Result, error)
}

// LoginLink is the game server's connection to the login server.
// *link.Client implements it.
type LoginLink interface {
	// VerifySession asks the login server whether k are the current session
	// keys of account. It returns an error wrapping link.ErrNotConnected
	// while the link is down.
	VerifySession(ctx context.Context, account string, k session.Keys) (bool, error)
	// SetPlayers reports the number of players on this server.
	SetPlayers(n int)
}

// Deps are the services a game server uses.
type Deps struct {
	// Characters stores the player characters. Required.
	Characters storage.CharacterStore
	// IDs hands out the object IDs of new characters and their items.
	// Required.
	IDs storage.IDAllocator
	// Data is the static game data (templates, starting items). Required.
	Data *data.Data
	// Login checks session keys and receives the player count. Required.
	Login LoginLink
	// Scripts runs the script hooks. Nil allows everything.
	Scripts Scripts
}

// Server is one game server. Serve or Run may be called only once.
type Server struct {
	gs      config.GameServer
	chars   storage.CharacterStore
	ids     storage.IDAllocator
	data    *data.Data
	login   LoginLink
	scripts Scripts
	logger  *slog.Logger

	// templates are the starting classes NewCharacterSuccess lists.
	templates []data.Template

	idleTimeout   time.Duration
	verifyTimeout time.Duration
	rand          io.Reader        // key source; nil means crypto/rand
	now           func() time.Time // replaced in tests

	clients      netutil.Registry[*client]
	hackAttempts atomic.Uint32
	accountLocks keyedMutex

	playersMu sync.Mutex
	players   int // authenticated clients
}

// New returns the game server with the given ID from cfg, which should be
// validated. The caller runs the scripts engine and the login link and
// keeps them running until Run or Serve returns.
func New(cfg config.Config, id int, deps Deps, logger *slog.Logger) (*Server, error) {
	gs, ok := cfg.GameServerByID(id)
	if !ok {
		return nil, fmt.Errorf("gameserver: no game server with id %d in the config", id)
	}
	switch {
	case deps.Characters == nil:
		return nil, errors.New("gameserver: no character store")
	case deps.IDs == nil:
		return nil, errors.New("gameserver: no object ID allocator")
	case deps.Data == nil:
		return nil, errors.New("gameserver: no game data")
	case deps.Login == nil:
		return nil, errors.New("gameserver: no login server link")
	}
	templates := startingTemplates(deps.Data)
	if len(templates) == 0 {
		return nil, errors.New("gameserver: the game data has no template for any starting class")
	}
	return &Server{
		gs:            gs,
		chars:         deps.Characters,
		ids:           deps.IDs,
		data:          deps.Data,
		login:         deps.Login,
		scripts:       deps.Scripts,
		logger:        logger.With("server", gs.ID),
		templates:     templates,
		idleTimeout:   DefaultIdleTimeout,
		verifyTimeout: DefaultVerifyTimeout,
		now:           time.Now,
	}, nil
}

// OnlineClients returns the number of connected clients.
func (s *Server) OnlineClients() int { return s.clients.Len() }

// Players returns the number of clients that passed AuthLogin.
func (s *Server) Players() int {
	s.playersMu.Lock()
	defer s.playersMu.Unlock()
	return s.players
}

// HackAttempts returns how many malformed or impossible packets clients
// sent.
func (s *Server) HackAttempts() uint32 { return s.hackAttempts.Load() }

// addPlayers changes the player count and reports it to the login server.
// The lock keeps the reports in order.
func (s *Server) addPlayers(delta int) {
	s.playersMu.Lock()
	defer s.playersMu.Unlock()
	s.players += delta
	s.login.SetPlayers(s.players)
}

// Run listens for clients and serves until ctx is done.
func (s *Server) Run(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.gs.ListenAddr())
	if err != nil {
		return fmt.Errorf("gameserver: listen for clients: %w", err)
	}
	return s.Serve(ctx, ln)
}

// Serve accepts clients on ln until ctx is done. It then closes the
// listener and every connection, waits for all handlers to finish and
// returns nil. It returns an error if the listener fails. Serve takes
// ownership of ln.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.logger.Info("game server listening", "name", s.gs.Name, "addr", ln.Addr().String(),
		"protocols", s.gs.AllowedProtocols)

	var wg sync.WaitGroup
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
	s.clients.CloseAll()
	wg.Wait()

	s.logger.Info("game server stopped", "hackAttempts", s.hackAttempts.Load())
	if err != nil {
		return fmt.Errorf("gameserver: %w", err)
	}
	return nil
}

func (s *Server) serveClient(ctx context.Context, conn net.Conn) {
	logger := s.logger.With("remote", conn.RemoteAddr().String())
	key, err := gamecrypt.NewKey(s.rand)
	if err != nil {
		logger.Error("cannot make a session key", "err", err)
		_ = conn.Close()
		return
	}
	c := newClient(conn, key, s.idleTimeout, logger)
	id, ok := s.clients.Add(c)
	if !ok {
		_ = conn.Close()
		return
	}
	defer func() {
		s.clients.Remove(id)
		_ = c.Close()
		if c.state == stateAuthenticated {
			s.addPlayers(-1)
		}
	}()
	logger.Info("client connected", "online", s.clients.Len())

	if err := s.handshake(c); err != nil {
		switch {
		case ctx.Err() != nil:
		case errors.Is(err, errPing):
			logger.Debug("ping from a client")
		default:
			logger.Info("handshake failed", "err", err)
		}
		return
	}
	for {
		opcode, data, err := c.receive()
		if err != nil {
			if ctx.Err() == nil {
				c.logger.Info("client disconnected", "err", err)
			}
			return
		}
		if err := s.handlePacket(ctx, c, opcode, data); err != nil {
			if ctx.Err() == nil {
				c.logger.Warn("closing client connection", "err", err)
			}
			return
		}
	}
}

// errPing ends a connection that only checked whether the server is up.
var errPing = errors.New("ping")

// handshake reads ProtocolVersion and answers with KeyPacket, which turns
// on the cipher. It returns an error if the connection must close.
func (s *Server) handshake(c *client) error {
	opcode, data, err := c.receive()
	if err != nil {
		return err
	}
	if opcode != clientpackets.OpProtocolVersion {
		s.hackAttempts.Add(1)
		return fmt.Errorf("first packet has opcode %s, want ProtocolVersion", opcodeString(opcode))
	}
	pv, err := clientpackets.ParseProtocolVersion(data)
	if err != nil {
		s.hackAttempts.Add(1)
		return err
	}
	if pv.Version == clientpackets.PingVersion {
		return errPing
	}
	serverID := int32(s.gs.ID) //nolint:gosec // validated to 1-255
	if !slices.Contains(s.gs.AllowedProtocols, pv.Version) {
		if err := c.send(serverpackets.NewKeyPacket(c.key, false, serverID)); err != nil {
			return err
		}
		return fmt.Errorf("protocol version %d is not allowed (allowed: %v)", pv.Version, s.gs.AllowedProtocols)
	}
	c.logger.Debug("protocol version accepted", "version", pv.Version)
	return c.send(serverpackets.NewKeyPacket(c.key, true, serverID))
}

// handlePacket handles one client packet. An error closes the connection.
// Packets that are unknown, not supported yet or not allowed in the
// client's state are logged and ignored, as Mobius does.
func (s *Server) handlePacket(ctx context.Context, c *client, opcode byte, data []byte) error {
	switch opcode {
	case clientpackets.OpAuthLogin:
		if c.state != stateConnected {
			c.logger.Debug("AuthLogin after authentication, ignored")
			return nil
		}
		return s.authLogin(ctx, c, data)

	case clientpackets.OpNewCharacter:
		if !c.requireAuthenticated("NewCharacter") {
			return nil
		}
		return c.send(serverpackets.NewCharacterSuccessPacket(s.templates))

	case clientpackets.OpCharacterCreate:
		if !c.requireAuthenticated("CharacterCreate") {
			return nil
		}
		return s.characterCreate(ctx, c, data)

	case clientpackets.OpCharacterDelete, clientpackets.OpCharacterSelect, clientpackets.OpCharacterRestore:
		c.logger.Info("packet not supported yet, ignored", "opcode", opcodeString(opcode))
		return nil

	default:
		c.logger.Debug("unknown packet, ignored", "opcode", opcodeString(opcode), "size", len(data)+1)
		return nil
	}
}

// requireAuthenticated reports whether c passed AuthLogin, and logs the
// packet as ignored if not.
func (c *client) requireAuthenticated(packet string) bool {
	if c.state == stateAuthenticated {
		return true
	}
	c.logger.Debug("packet before AuthLogin, ignored", "packet", packet)
	return false
}

// authLogin checks the session keys with the login server and sends the
// character list. Keys the login server refuses, or a login server that
// cannot be asked, get LoginFail and close the connection.
func (s *Server) authLogin(ctx context.Context, c *client, data []byte) error {
	auth, err := clientpackets.ParseAuthLogin(data)
	if err != nil {
		s.hackAttempts.Add(1)
		return err
	}
	account := strings.ToLower(auth.Account)
	if account == "" {
		s.hackAttempts.Add(1)
		return errors.New("AuthLogin with an empty account name")
	}
	c.logger = c.logger.With("account", account)

	keys := auth.Keys()
	vctx, cancel := context.WithTimeout(ctx, s.verifyTimeout)
	ok, err := s.login.VerifySession(vctx, account, keys)
	cancel()
	if err == nil && ok {
		c.account = account
		c.keys = keys
		c.state = stateAuthenticated
		s.addPlayers(1)
		c.logger.Info("client authenticated")
		return s.sendCharacterList(ctx, c)
	}

	switch {
	case ctx.Err() != nil:
		return ctx.Err()
	case errors.Is(err, link.ErrNotConnected):
		err = errors.New("cannot check the session keys: the link to the login server is down")
	case errors.Is(err, context.DeadlineExceeded):
		err = fmt.Errorf("cannot check the session keys: the login server did not answer within %v", s.verifyTimeout)
	case err != nil:
		err = fmt.Errorf("cannot check the session keys: %w", err)
	default:
		err = errors.New("the login server refused the session keys")
	}
	if sendErr := c.send(serverpackets.NewLoginFailPacket(serverpackets.LoginFailSystemErrorLoginLater)); sendErr != nil {
		return errors.Join(err, sendErr)
	}
	return err
}

// sendCharacterList sends the account's characters.
func (s *Server) sendCharacterList(ctx context.Context, c *client) error {
	chars, err := s.chars.ListByAccount(ctx, c.account)
	if err != nil {
		return fmt.Errorf("list characters: %w", err)
	}
	return c.send(serverpackets.NewCharSelectionInfoPacket(c.account, c.keys.PlayOK1, chars, s.now()))
}

// characterCreate handles CharacterCreate: CharCreateFail with a reason, or
// the updated character list followed by CharCreateOk. Mobius sends them
// in that order (CharacterCreate.initNewChar sends the list, then runImpl
// sends CharCreateOk).
func (s *Server) characterCreate(ctx context.Context, c *client, data []byte) error {
	req, err := clientpackets.ParseCharacterCreate(data)
	if err != nil {
		s.hackAttempts.Add(1)
		return err
	}
	if reason, ok := s.createCharacter(ctx, c, req); !ok {
		return c.send(serverpackets.NewCharCreateFailPacket(reason))
	}
	if err := s.sendCharacterList(ctx, c); err != nil {
		return err
	}
	return c.send(serverpackets.NewCharCreateOkPacket())
}

// opcodeString formats an opcode for logs.
func opcodeString(op byte) string { return fmt.Sprintf("0x%02x", op) }
