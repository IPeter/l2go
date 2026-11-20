package game

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The dialog tests run a server with the world data of data/ (its spawn
// lists cut to one block, so the test areas stay quiet), HTML fixture pages
// (the real pages are not in git) and test scripts.

// loadDialogWorld loads the world data once, with one spawn block only.
var loadDialogWorld = sync.OnceValues(func() (*data.World, error) {
	w, err := data.LoadWorld(filepath.Join("..", "..", "data"))
	if err != nil {
		return nil, err
	}
	wd := w.WorldData
	wd.Spawns = wd.Spawns[:1]
	return data.NewWorld(wd)
})

func testWorldData(t *testing.T) *data.World {
	t.Helper()
	w, err := loadDialogWorld()
	if err != nil {
		t.Fatalf("load the world data: %v", err)
	}
	return w
}

// Places and NPCs of the tests.
var (
	// tiVillage is near the first restart point of the Talking Island
	// village, in the middle of its town zone's height (-3800 to -3700).
	tiVillage = world.Position{X: -83990, Y: 243336, Z: -3750}
	// tiStart is the first creation point of the human fighter.
	tiStart = world.Position{X: -71338, Y: 258271, Z: -3104}
)

const (
	npcRoxxy       = 30006 // Teleporter, Talking Island gatekeeper
	npcLector      = 30001 // Merchant
	npcWilford     = 30005 // Warehouse
	npcNewbieGuide = 30598 // Folk
	npcMartin      = 30731 // PetManager
	npcGuard       = 30733 // Guard
)

// dialogEnvOptions configure newDialogEnv.
type dialogEnvOptions struct {
	// pages are HTML pages by path relative to the HTML folder.
	pages map[string]string
	// scripts are Lua scripts by path relative to the scripts folder (also
	// their pages); nil runs without a scripting engine.
	scripts map[string]string
	// configure changes the game server's config.
	configure func(gs *config.GameServer)
	// start is when the fake clock starts; zero means testStart.
	start time.Time
	// rand is the scripts' random source; nil is a random one.
	rand rnd.Source
}

// newDialogEnv starts a server like newEnv, with world data, the HTML pages
// and scripts of opts.
func newDialogEnv(t *testing.T, opts dialogEnvOptions) *testEnv {
	t.Helper()
	d := testGameData(t)
	w := testWorldData(t)
	dataDir := t.TempDir()
	for name, content := range opts.pages {
		writeDialogFile(t, filepath.Join(dataDir, "html", filepath.FromSlash(name)), content)
	}

	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0 // as in newEnv
	gs.DataDir = dataDir
	out := NewScriptOutput()
	var engine *scripting.Engine
	if opts.scripts != nil {
		dir := scripttest.WriteScripts(t, opts.scripts)
		gs.Scripts.Dir = dir
		var err error
		engine, err = scripting.New(t.Context(), scripting.Options{
			Dir: dir, Output: out, World: out, Rand: opts.rand,
			Constants: ScriptConstants(nil, 0, QuestSettings{}),
		})
		if err != nil {
			t.Fatalf("load the test scripts: %v", err)
		}
		t.Cleanup(engine.Close) // runs after the server stopped
	}
	if opts.configure != nil {
		opts.configure(gs)
	}

	store := newRecordingStore()
	start := testStart
	if !opts.start.IsZero() {
		start = opts.start
	}
	clock := world.NewFakeClock(start)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		World:          w,
		Characters:     store,
		IDs:            ids,
		Scripts:        engine,
		ScriptOutput:   out,
		Passives:       testPassives(t),
		Categories:     testCategories(t),
		Clock:          clock,
		CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, stopRun := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(runCtx) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			stopRun()
			if err := <-done; err != nil {
				t.Errorf("Run: %v", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

func writeDialogFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// testChar describes a character for enterChar.
type testChar struct {
	account, name string
	access        storage.AccessLevel
	pos           world.Position
	level         int32
	karma         int32
	adena         int64
	items         []storage.Item // more items; object IDs are given
	quests        []storage.Quest
}

// enterChar creates a human fighter as c describes and enters the world
// with it. The packets sent so far are dropped.
func (e *testEnv) enterChar(c testChar) (*fakeSession, *Player) {
	e.t.Helper()
	ch := e.createCharacter(c.account, c.name, classHumanFighter, c.access)
	if c.pos != (world.Position{}) {
		ch.X, ch.Y, ch.Z = c.pos.X, c.pos.Y, c.pos.Z
	}
	if c.level > 0 {
		ch.Level = c.level
	}
	ch.Karma = c.karma
	ch.Quests = c.quests
	if c.level > 1 {
		ch.Exp = e.srv.ExpForLevel(int(c.level))
	}
	extra := c.items
	if c.adena > 0 {
		extra = append(extra, storage.Item{ItemID: adenaID, Count: c.adena, Slot: storage.PaperdollNone, Mana: storage.NoMana})
	}
	if len(extra) > 0 {
		ids, err := e.ids.NextIDs(e.ctx, len(extra))
		if err != nil {
			e.t.Fatalf("NextIDs: %v", err)
		}
		for i := range extra {
			extra[i].ObjectID = ids[i]
		}
		ch.Items = append(ch.Items, extra...)
	}
	// Straight into the store, so that the test's saves stay unrecorded.
	if err := e.store.Update(e.ctx, &ch); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	sess := newSession(c.account)
	p := e.enter(sess, 0)
	sess.take(e.t)
	return sess, p
}

// spawnNpc puts an NPC of template npcID into the world at pos.
func (e *testEnv) spawnNpc(npcID int, pos world.Position) *Npc {
	e.t.Helper()
	var n *Npc
	e.onLoop(func() {
		var err error
		if n, err = e.srv.SpawnNpc(npcID, pos); err != nil {
			e.t.Errorf("SpawnNpc %d: %v", npcID, err)
		}
	})
	if n == nil {
		e.t.FailNow()
	}
	return n
}

// talk makes p interact with n (the second click, in range) and returns
// the packets p got.
func (e *testEnv) talk(sess *fakeSession, p *Player, n *Npc) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.onLoop(func() { e.srv.runNpcInteract(p, n) })
	return sess.take(e.t)
}

// bypass sends RequestBypassToServer with command and returns the packets
// that came back.
func (e *testEnv) bypass(sess *fakeSession, command string) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.srv.Handle(sess, clientpackets.OpRequestBypassToServer, stringBody(command))
	e.sync()
	return sess.take(e.t)
}

// link sends RequestLinkHtml with link and returns the packets that came
// back.
func (e *testEnv) link(sess *fakeSession, link string) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.srv.Handle(sess, clientpackets.OpRequestLinkHtml, stringBody(link))
	e.sync()
	return sess.take(e.t)
}

// eastOf returns a position dx units east of pos.
func eastOf(pos world.Position, dx int32) world.Position {
	pos.X += dx
	return pos
}

// onePage returns the only NpcHtmlMessage of ps, failing the test unless
// there is exactly one.
func onePage(t *testing.T, ps []testclient.Packet) testclient.NpcHtmlMessage {
	t.Helper()
	pages := only[testclient.NpcHtmlMessage](ps)
	if len(pages) != 1 {
		t.Fatalf("got %d pages in %q, want 1", len(pages), testclient.Names(ps))
	}
	return pages[0]
}

// noPage fails the test if ps has an NpcHtmlMessage.
func noPage(t *testing.T, what string, ps []testclient.Packet) {
	t.Helper()
	if pages := only[testclient.NpcHtmlMessage](ps); len(pages) != 0 {
		t.Errorf("%s: got the page %q, want none", what, pages[0].HTML)
	}
}
