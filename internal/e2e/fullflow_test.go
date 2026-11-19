package e2e

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/loginserver"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
)

const linkSecret = "full-flow-secret"

// stack is the login server and game server 1 running in the test process,
// wired as cmd/l2go wires them, with in-memory storage.
type stack struct {
	login     *loginserver.Server
	game      *gameserver.Server
	accounts  *memstore.Accounts
	chars     *memstore.Characters
	loginAddr string
}

// startStack starts both servers on 127.0.0.1:0 with the real data/ and
// scripts/ folders. Everything stops when the test ends, the game server
// first and the login server last. Server logs go to the test output.
func startStack(t *testing.T) *stack {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(t.Output(), nil))
	clientLn, linkLn, gameLn := listen(t), listen(t), listen(t)

	cfg := config.Default()
	l := &cfg.LoginServer
	l.ClientListen = clientLn.Addr().String()
	l.GameServerListen = linkLn.Addr().String()
	l.RSAKeyPairs = 1 // generating key pairs is slow
	gs := &cfg.GameServers[0]
	gs.Secret = linkSecret
	gs.InternalIP = "127.0.0.1" // the test client connects from 127.0.0.1
	gs.Port = gameLn.Addr().(*net.TCPAddr).Port
	gs.DataDir = filepath.Join("..", "..", "data")
	gs.Scripts.Dir = filepath.Join("..", "..", "scripts")
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	s := &stack{
		accounts:  memstore.NewAccounts(),
		chars:     memstore.NewCharacters(),
		loginAddr: clientLn.Addr().String(),
	}

	ls, err := loginserver.New(cfg, s.accounts, logger.With("role", "login"))
	if err != nil {
		t.Fatalf("loginserver.New: %v", err)
	}
	s.login = ls
	serve(t, "login server", func(ctx context.Context) error { return ls.Serve(ctx, clientLn, linkLn) })

	gameLogger := logger.With("role", "game")
	gsLogger := gameLogger.With("server", gs.ID)
	gameData, err := data.Load(gs.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := scripting.New(t.Context(), scripting.Options{
		Dir:     gs.Scripts.Dir,
		Timeout: time.Duration(gs.Scripts.EventTimeout),
		Logger:  gsLogger,
	})
	if err != nil {
		t.Fatal(err)
	}
	serve(t, "scripts engine", engine.Run)

	loginLink := link.NewClient(link.ClientConfig{
		Addr:       cfg.LoginServer.GameServerAddr(),
		ServerID:   gs.ID,
		Secret:     gs.Secret,
		MaxPlayers: int(gs.Options.MaxPlayers),
		MinBackoff: 10 * time.Millisecond,
		MaxBackoff: 100 * time.Millisecond,
	}, gsLogger)
	serve(t, "login link", loginLink.Run)

	game, err := gameserver.New(cfg, gs.ID, gameserver.Deps{
		Characters: s.chars,
		IDs:        memstore.NewIDs(),
		Data:       gameData,
		Login:      loginLink,
		Scripts:    engine,
	}, gameLogger)
	if err != nil {
		t.Fatalf("gameserver.New: %v", err)
	}
	s.game = game
	serve(t, "game server", func(ctx context.Context) error { return game.Serve(ctx, gameLn) })
	return s
}

// listen opens a listener on a free port. The servers take ownership of
// it; the cleanup only matters if the test fails before that.
func listen(t *testing.T) net.Listener {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln
}

// serve runs fn until the test ends. Cleanups run last-in first-out, so
// what starts last stops first.
func serve(t *testing.T, what string, fn func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- fn(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("%s: %v", what, err)
			}
		case <-time.After(10 * time.Second):
			t.Errorf("%s did not stop within 10s", what)
		}
	})
}

// TestFullFlow runs the real login server and game server together and
// walks the test client through the whole parity checklist (F1-F11).
func TestFullFlow(t *testing.T) {
	s := startStack(t)
	f := flow{
		loginAddr: s.loginAddr,
		// gameAddr is left empty: the client goes where the server list
		// points it.
		account:   "FullFlow",
		password:  "Secret-1",
		charName:  "Frostwind",
		exclusive: true,
		checkAccount: func(t *testing.T, typed string) {
			acc, err := s.accounts.FindByUsername(t.Context(), strings.ToLower(typed))
			if err != nil {
				t.Fatalf("account not stored under the lowercased name: %v", err)
			}
			if acc.AccessLevel != storage.AccessLevelPlayer {
				t.Errorf("new account access level %d, want player", acc.AccessLevel)
			}
			if _, err := s.accounts.FindByUsername(t.Context(), typed); !errors.Is(err, storage.ErrNotFound) {
				t.Errorf("account also stored as typed (%q): %v", typed, err)
			}
		},
		checkCharacter: func(t *testing.T, account, name string) {
			chars, err := s.chars.ListByAccount(t.Context(), account)
			if err != nil || len(chars) != 1 {
				t.Fatalf("stored characters of %q: %+v, %v", account, chars, err)
			}
			ch := chars[0]
			if ch.Name != name || ch.ClassID != 0 || ch.Level != 1 || ch.MaxHP != 80 || ch.MaxMP != 30 {
				t.Errorf("stored character: %+v", ch)
			}
			// Squire's Shirt, Pants and Sword worn; Dagger and Tutorial
			// Guide carried.
			worn := 0
			for _, it := range ch.Items {
				if it.Slot != storage.PaperdollNone {
					worn++
				}
			}
			if len(ch.Items) != 5 || worn != 3 {
				t.Errorf("stored items: %+v, want 5 with 3 worn", ch.Items)
			}
		},
	}
	f.run(t)

	st := s.login.Stats()
	if st.AccountsCreated != 1 || st.FailedLogins != 1 || st.GameServers != 1 {
		t.Errorf("login server stats %+v, want 1 account created, 1 failed login, 1 game server", st)
	}
	if st.HackAttempts != 0 || s.game.HackAttempts() != 0 {
		t.Errorf("hack attempts: login %d, game %d; the flow should cause none", st.HackAttempts, s.game.HackAttempts())
	}
}
