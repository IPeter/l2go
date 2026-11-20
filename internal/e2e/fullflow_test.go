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
	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/gameserver"
	"github.com/IPeter/l2go/internal/link"
	"github.com/IPeter/l2go/internal/loginserver"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/world"
)

const linkSecret = "full-flow-secret"

// clockStart is when the world's fake clock starts: noon, so that the
// teleport fees are never halved (Mobius halves them from 20:00 on Mondays
// and Tuesdays), and the game time is the same on every run.
var clockStart = time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)

// stack is the login server and game server 1 running in the test process,
// wired as cmd/l2go wires them, with in-memory storage and a fake clock.
type stack struct {
	login *loginserver.Server
	// server is game server 1's network and session layer; game is its world
	// on the world loop.
	server *gameserver.Server
	game   *game.Server
	// clock drives the world loop. runClock moves it along with real time;
	// passTime jumps ahead.
	clock     *world.FakeClock
	accounts  *memstore.Accounts
	chars     *memstore.Characters
	ids       *memstore.IDs
	loginAddr string
	gameAddr  string
}

// startStack starts both servers on 127.0.0.1:0 with the real data/ and
// scripts/ folders. Everything stops when the test ends, in the order
// cmd/l2go stops it: the game server's connections first, then the world
// loop (which saves every player), the login link, the scripts engine and
// last the login server. Server logs go to the test output.
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
	// Teleports cost adena from level 1, so the walk can try one it cannot
	// pay. Mobius lets players up to level 40 teleport for free.
	gs.NPCs.MaxFreeTeleportLevel = 0
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	s := &stack{
		clock:     world.NewFakeClock(clockStart),
		accounts:  memstore.NewAccounts(),
		chars:     memstore.NewCharacters(),
		ids:       memstore.NewIDs(),
		loginAddr: clientLn.Addr().String(),
		gameAddr:  gameLn.Addr().String(),
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
	worldData, err := data.LoadWorld(gs.DataDir)
	if err != nil {
		t.Fatal(err)
	}

	constants, err := data.LoadConstants(gs.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	// Skills, as cmd/l2go loads them: new characters get their auto-get skills.
	skillData, err := data.LoadSkills(gs.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	// The scripts run on the world loop; what they do to the world (npc:say,
	// the quests' items and states) reaches the players through the game's
	// adapter.
	out := game.NewScriptOutput()
	engine, err := scripting.New(t.Context(), scripting.Options{
		Dir:       gs.Scripts.Dir,
		Timeout:   time.Duration(gs.Scripts.EventTimeout),
		Logger:    gsLogger,
		Output:    out,
		World:     out,
		Constants: game.ScriptConstants(constants, gs.Combat.PartyRange, game.QuestSettings{}),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Cleanups run last-in first-out, so the engine closes after the world
	// loop has stopped calling it.
	t.Cleanup(engine.Close)

	loginLink := link.NewClient(link.ClientConfig{
		Addr:       cfg.LoginServer.GameServerAddr(),
		ServerID:   gs.ID,
		Secret:     gs.Secret,
		MaxPlayers: int(gs.Options.MaxPlayers),
		MinBackoff: 10 * time.Millisecond,
		MaxBackoff: 100 * time.Millisecond,
	}, gsLogger)
	serve(t, "login link", loginLink.Run)

	s.game, err = game.New(game.Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           gameData,
		World:          worldData,
		Characters:     s.chars,
		IDs:            s.ids,
		Skills:         skillData,
		Scripts:        engine,
		ScriptOutput:   out,
		Clock:          s.clock,
		Logger:         gsLogger,
		CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("game.New: %v", err)
	}
	serve(t, "game world", s.game.Run)
	runClock(t, s.clock, s.game.Loop().TickInterval())

	s.server, err = gameserver.New(cfg, gs.ID, gameserver.Deps{
		Characters: s.chars,
		IDs:        s.ids,
		Data:       gameData,
		Login:      loginLink,
		Game:       s.game,
	}, gameLogger)
	if err != nil {
		t.Fatalf("gameserver.New: %v", err)
	}
	serve(t, "game server", func(ctx context.Context) error { return s.server.Serve(ctx, gameLn) })
	return s
}

// runClock moves the fake clock along with real time, a tick at a time,
// until the test ends, so the world loop runs as it would with the real
// clock: timers fire and ticks come.
func runClock(t *testing.T, clock *world.FakeClock, tick time.Duration) {
	t.Helper()
	serve(t, "fake clock", func(ctx context.Context) error {
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				clock.Advance(tick)
			}
		}
	})
}

// passTime lets d of game time pass at once: it moves the fake clock ahead
// a tick at a time and waits after each tick until the world loop has run
// it, so the world goes through every tick in order (movement, visibility,
// timers), only faster than real time.
func (s *stack) passTime(t *testing.T, d time.Duration) {
	t.Helper()
	tick := s.game.Loop().TickInterval()
	for passed := time.Duration(0); passed < d; passed += tick {
		s.clock.Advance(tick)
		// A command runs only after everything due before it: a barrier.
		if err := s.game.OnLoop(t.Context(), func() {}); err != nil {
			t.Fatalf("world loop: %v", err)
		}
	}
}

// makeGM gives the only character of account GM rights (access level 1).
func (s *stack) makeGM(t *testing.T, account string) {
	t.Helper()
	s.updateCharacter(t, account, func(c *storage.Character) { c.AccessLevel = storage.AccessLevelAdmin })
}

// giveItems adds count items of itemID to the inventory of the only
// character of account.
func (s *stack) giveItems(t *testing.T, account string, itemID int32, count int64) {
	t.Helper()
	id, err := s.ids.NextID(t.Context())
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	s.updateCharacter(t, account, func(c *storage.Character) {
		c.Items = append(c.Items, storage.Item{ObjectID: id, ItemID: itemID, Count: count, Slot: storage.PaperdollNone, Mana: storage.NoMana})
	})
}

// updateCharacter changes the only character of account in the store. The
// character must not be in the world, where the game would save over it.
func (s *stack) updateCharacter(t *testing.T, account string, change func(c *storage.Character)) {
	t.Helper()
	chars, err := s.chars.ListByAccount(t.Context(), account)
	if err != nil || len(chars) != 1 {
		t.Fatalf("characters of %q: %+v, %v; want one", account, chars, err)
	}
	c := chars[0]
	change(&c)
	if err := s.chars.Update(t.Context(), &c); err != nil {
		t.Fatalf("updating %s: %v", c.Name, err)
	}
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
// walks two test clients through the parity checklist (F1-F11) and the
// demo checklists of Milestones 3 and 4. The world runs on a fake clock, so
// the long walk takes no real time, and the store makes the second player a
// GM with adena for the GM and gatekeeper steps.
func TestFullFlow(t *testing.T) {
	s := startStack(t)
	f := flow{
		loginAddr: s.loginAddr,
		// gameAddr is left empty: the client goes where the server list
		// points it.
		account:   "FullFlow",
		password:  "Secret-1",
		charName:  "Frostwind",
		account2:  "SecondFlow",
		charName2: "Brightleaf",
		exclusive: true,
		htmlDir:   filepath.Join("..", "..", "data", "html"),
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
			// Full HP and MP, the stats' maxima; CP 0 (Mobius CharacterCreate).
			hp, mp, cp := float64(newFighter.MaxHP), float64(newFighter.MaxMP), float64(newFighter.MaxCP)
			if ch.Name != name || ch.ClassID != 0 || ch.Level != 1 || ch.CurHP != hp || ch.MaxHP != hp ||
				ch.CurMP != mp || ch.MaxMP != mp || ch.CurCP != 0 || ch.MaxCP != cp {
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
		makeGM:    s.makeGM,
		giveItems: s.giveItems,
		passTime:  s.passTime,
	}
	f.run(t)

	st := s.login.Stats()
	if st.AccountsCreated != 2 || st.FailedLogins != 1 || st.GameServers != 1 {
		t.Errorf("login server stats %+v, want 2 accounts created, 1 failed login, 1 game server", st)
	}
	if st.HackAttempts != 0 || s.server.HackAttempts() != 0 {
		t.Errorf("hack attempts: login %d, game %d; the flow should cause none", st.HackAttempts, s.server.HackAttempts())
	}
}
