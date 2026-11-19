// Package gameserver is the network and session layer of the game server
// for the Interlude client (protocol 746): the handshake and the packet
// cipher, the session check with the login server, the connection states,
// and the character screen (list, create, delete, restore, select). Packets
// of a character in the world go to the game (internal/game), which runs
// them on the world loop.
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
	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/netutil"
	"github.com/IPeter/l2go/internal/session"
	"github.com/IPeter/l2go/internal/storage"
)

const (
	// DefaultIdleTimeout closes client connections that send nothing before
	// entering the world.
	DefaultIdleTimeout = 15 * time.Minute
	// DefaultVerifyTimeout bounds the session key check with the login
	// server.
	DefaultVerifyTimeout = 5 * time.Second
	// DefaultSendQueue bounds the packets waiting for a client's writer. A
	// client that falls further behind is disconnected.
	DefaultSendQueue = 8192

	// writeTimeout bounds writing one batch of packets to a client.
	writeTimeout = 10 * time.Second
)

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

// Deps are the services a game server uses. All are required.
type Deps struct {
	// Characters stores the player characters.
	Characters storage.CharacterStore
	// IDs hands out the object IDs of new characters and their items.
	IDs storage.IDAllocator
	// Data is the static game data (templates, starting items).
	Data *data.Data
	// Login checks session keys and receives the player count.
	Login LoginLink
	// Game is the game world: it selects characters, puts them into the
	// world and handles their packets there. Its loop also runs the
	// character creation script hook. *game.Server implements it.
	Game game.Service
}

// Server is one game server. Serve or Run may be called only once.
type Server struct {
	gs     config.GameServer
	chars  storage.CharacterStore
	ids    storage.IDAllocator
	data   *data.Data
	login  LoginLink
	game   game.Service
	logger *slog.Logger

	// templates are the starting classes NewCharacterSuccess lists.
	templates []data.Template

	idleTimeout   time.Duration
	verifyTimeout time.Duration
	sendQueue     int
	rand          io.Reader        // key source; nil means crypto/rand
	now           func() time.Time // replaced in tests

	clients      netutil.Registry[*client]
	hackAttempts atomic.Uint32
	accountLocks keyedMutex

	playersMu sync.Mutex
	players   int // authenticated clients
}

// New returns the game server with the given ID from cfg, which should be
// validated. The caller runs the game loop and the login link and keeps
// them running until Run or Serve returns.
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
	case deps.Game == nil:
		return nil, errors.New("gameserver: no game service")
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
		game:          deps.Game,
		logger:        logger.With("server", gs.ID),
		templates:     templates,
		idleTimeout:   DefaultIdleTimeout,
		verifyTimeout: DefaultVerifyTimeout,
		sendQueue:     DefaultSendQueue,
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
//
// Each connection that had selected a character reports to the game with
// Disconnected before its handler finishes, so the game loop should stop
// only after Serve returns.
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

// serveClient is the reader of one connection: it does the handshake and
// handles the client's packets in order until the connection ends.
func (s *Server) serveClient(ctx context.Context, conn net.Conn) {
	logger := s.logger.With("remote", conn.RemoteAddr().String())
	key, err := gamecrypt.NewKey(s.rand)
	if err != nil {
		logger.Error("cannot make a session key", "err", err)
		_ = conn.Close()
		return
	}
	c := newClient(ctx, s, conn, key, clientConfig{
		idleTimeout: s.idleTimeout,
		queueLimit:  s.sendQueue,
		logger:      logger,
	})
	id, ok := s.clients.Add(c)
	if !ok {
		_ = c.Close()
		<-c.writerDone
		return
	}
	defer func() {
		<-c.writerDone // the queue is flushed or dropped, and the connection closed
		c.stopBackground()
		if c.selected {
			s.game.Disconnected(c)
		}
		if c.counted {
			s.addPlayers(-1)
		}
		s.clients.Remove(id)
	}()
	logger.Info("client connected", "online", s.clients.Len())

	// c.closing is set when the server closed the connection on purpose
	// (shutdown, CloseAfter, a full queue); those ends are not logged here.
	if err := s.handshake(c); err != nil {
		switch {
		case c.closing.Load():
		case errors.Is(err, errPing):
			logger.Debug("ping from a client")
		default:
			logger.Info("handshake failed", "err", err)
		}
		c.CloseAfter(nil, 0) // sends a refusing KeyPacket first
		return
	}
	for {
		opcode, data, err := c.receive()
		if err != nil {
			if !c.closing.Load() {
				c.log().Info("client disconnected", "err", err)
			}
			_ = c.Close()
			return
		}
		if err := s.handlePacket(c.ctx, c, opcode, data); err != nil {
			switch {
			case c.closing.Load():
			case errors.Is(err, errLoggedOut):
				c.log().Info("client logged out")
			default:
				c.log().Warn("closing client connection", "err", err)
			}
			c.CloseAfter(nil, 0) // sends what is queued, such as LoginFail
			return
		}
	}
}

var (
	// errPing ends a connection that only checked whether the server is up.
	errPing = errors.New("ping")
	// errLoggedOut ends a connection that sent Logout on the character
	// screen.
	errLoggedOut = errors.New("logged out")
)

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
		c.Send(serverpackets.NewKeyPacket(c.key, false, serverID))
		return fmt.Errorf("protocol version %d is not allowed (allowed: %v)", pv.Version, s.gs.AllowedProtocols)
	}
	c.log().Debug("protocol version accepted", "version", pv.Version)
	c.Send(serverpackets.NewKeyPacket(c.key, true, serverID))
	// Every client packet after KeyPacket is encrypted, so the input cipher
	// is on from the start, like a client's.
	c.in = gamecrypt.NewClientCipher(c.key)
	return nil
}

// handlePacket handles one client packet. As in Mobius GamePacketHandler,
// a packet that is not allowed in the connection's state is dropped with a
// debug log, and so is everything after CloseAfter. An error closes the
// connection.
//
// The character screen is handled here. In the world, EnterWorld,
// RequestRestart, Logout and every other packet go to the game.
func (s *Server) handlePacket(ctx context.Context, c *client, opcode byte, data []byte) error {
	if c.closing.Load() {
		c.log().Debug("packet while closing, dropped", "opcode", opcodeString(opcode))
		return nil
	}
	if opcode == clientpackets.OpProtocolVersion {
		c.log().Debug("ProtocolVersion after the handshake, dropped")
		return nil
	}
	st := c.State()
	if !allowed(opcode, st) {
		c.log().Debug("packet not allowed in this state, dropped",
			"opcode", opcodeString(opcode), "state", stateName(st), "size", len(data)+1)
		return nil
	}
	switch opcode {
	case clientpackets.OpAuthLogin:
		return s.authLogin(ctx, c, data)
	case clientpackets.OpNewCharacter:
		c.Send(serverpackets.NewCharacterSuccessPacket(s.templates))
		return nil
	case clientpackets.OpCharacterCreate:
		return s.characterCreate(ctx, c, data)
	case clientpackets.OpCharacterDelete:
		return s.characterDelete(ctx, c, data)
	case clientpackets.OpCharacterRestore:
		return s.characterRestore(ctx, c, data)
	case clientpackets.OpCharacterSelect:
		return s.characterSelect(ctx, c, data)
	case clientpackets.OpLogout:
		if st == game.StateInGame {
			s.game.Logout(c)
			return nil
		}
		return errLoggedOut // Mobius closes at once when no character is in the world
	case clientpackets.OpEnterWorld:
		return s.enterWorld(c, data)
	case clientpackets.OpRequestRestart:
		s.game.Restart(c)
		return nil
	case clientpackets.OpEx:
		ex, err := clientpackets.ParseExPacket(data)
		if err != nil {
			s.hackAttempts.Add(1)
			return err
		}
		s.game.HandleEx(c, ex.SubOpcode, ex.Body)
		return nil
	default:
		s.game.Handle(c, opcode, data)
		return nil
	}
}

// allowed reports whether a packet with opcode may arrive in state st: the
// connection states of Mobius ClientPackets. Every packet not listed,
// RequestRestart and the extended packets (all of Mobius ExClientPackets)
// belong to the world.
func allowed(opcode byte, st game.State) bool {
	switch opcode {
	case clientpackets.OpAuthLogin:
		return st == game.StateConnected
	case clientpackets.OpNewCharacter, clientpackets.OpCharacterCreate, clientpackets.OpCharacterDelete,
		clientpackets.OpCharacterRestore, clientpackets.OpCharacterSelect:
		return st == game.StateAuthenticated
	case clientpackets.OpLogout:
		return st == game.StateAuthenticated || st == game.StateInGame
	case clientpackets.OpEnterWorld:
		return st == game.StateEntering
	default:
		// TODO(M9): RequestPledgeCrest (0x68) is allowed on the character
		// screen too, for the clan crests of the list.
		return st == game.StateInGame
	}
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
	c.logger.Store(c.log().With("account", account))

	keys := auth.Keys()
	vctx, cancel := context.WithTimeout(ctx, s.verifyTimeout)
	ok, err := s.login.VerifySession(vctx, account, keys)
	cancel()
	if err == nil && ok {
		c.auth.Store(&authInfo{account: account, keys: keys})
		c.SetState(game.StateAuthenticated)
		c.counted = true
		s.addPlayers(1)
		c.log().Info("client authenticated")
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
	c.Send(serverpackets.NewLoginFailPacket(serverpackets.LoginFailSystemErrorLoginLater))
	return err
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
		c.Send(serverpackets.NewCharCreateFailPacket(reason))
		return nil
	}
	if err := s.sendCharacterList(ctx, c); err != nil {
		return err
	}
	c.Send(serverpackets.NewCharCreateOkPacket())
	return nil
}

// enterWorld hands EnterWorld to the game once per character selection.
// The game sends the enter-world packets and sets StateInGame.
func (s *Server) enterWorld(c *client, data []byte) error {
	ew, err := clientpackets.ParseEnterWorld(data)
	if err != nil {
		s.hackAttempts.Add(1)
		return err
	}
	if c.enterSent.Swap(true) {
		c.log().Debug("EnterWorld repeated, dropped")
		return nil
	}
	c.log().Debug("entering the world", "tracert", ew.Tracert)
	s.game.EnterWorld(c)
	return nil
}

// opcodeString formats an opcode for logs.
func opcodeString(op byte) string { return fmt.Sprintf("0x%02x", op) }
