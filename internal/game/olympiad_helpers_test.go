package game

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The Olympiad tests run a server with the world data of data/ (its zones
// hold the 22 stadium zones), the two first stadiums of Mobius, fixture
// pages of the Grand Olympiad Manager and an olympiad store, on a fake clock
// in UTC.

// classDuelist (88, newbie_test.go) is a third class of the human
// fighters.
const (
	classAdventurer   = 93
	itemGatePass      = 6651
	itemInfinityBlade = 6611
	heroSkillID       = 395 // Heroic Miracle
)

// olyStadium0 is the first stadium's centre.
var olyStadium0 = world.Position{X: -20814, Y: -21189, Z: -3030}

// olympiadTestPages are the manager's pages, cut to their placeholders and
// links.
var olympiadTestPages = map[string]string{
	"default/31688.htm": `<html><body>manager <a action="bypass -h npc_%objectId%_Olympiad 1">observe</a>` +
		`<a action="bypass -h npc_%objectId%_Olympiad 4">heroes</a></body></html>`,
	"olympiad/noble_main.htm": `<html><body>noble main` +
		`<a action="bypass -h npc_%objectId%_OlympiadNoble 1">cancel</a>` +
		`<a action="bypass -h npc_%objectId%_OlympiadNoble 2">list</a>` +
		`<a action="bypass -h npc_%objectId%_OlympiadNoble 3">points</a>` +
		`<a action="bypass -h npc_%objectId%_OlympiadNoble 4">join</a>` +
		`<a action="bypass -h npc_%objectId%_OlympiadNoble 5">classed</a>` +
		`<a action="bypass -h npc_%objectId%_OlympiadNoble 6">settle?</a>` +
		`<a action="bypass -h npc_%objectId%_OlympiadNoble 9">last points</a>` +
		`<a action="bypass -h npc_%objectId%_OlympiadDesc 2a">desc</a>` +
		`<a action="bypass -h npc_%objectId%_Olympiad 1">observe</a>` +
		`<a action="bypass -h npc_%objectId%_Olympiad 4">heroes</a>` +
		`<a action="bypass -h npc_%objectId%_Olympiad 2_88">ranking</a></body></html>`,
	"olympiad/noble_desc2a.htm":     `<html><body>desc 2a</body></html>`,
	"olympiad/noble_desc5.htm":      `<html><body>subclass page</body></html>`,
	"olympiad/noble_desc6.htm":      `<html><body>not a third class noble</body></html>`,
	"olympiad/noble_registered.htm": `<html><body>waiting %listClassed% %listNonClassedTeam% %listNonClassed%</body></html>`,
	"olympiad/noble_points1.htm":    `<html><body>points %points%</body></html>`,
	"olympiad/noble_points2.htm":    `<html><body>last points %points%</body></html>`,
	"olympiad/noble_settle.htm":     `<html><body>settle <a action="bypass -h npc_%objectId%_OlympiadNoble 10">settle</a></body></html>`,
	"olympiad/noble_nopoints.htm":   `<html><body>no points</body></html>`,
	"olympiad/olympiad_observe1.htm": `<html><body>arenas [%title1%] [%title2%]` +
		`<a action="bypass -h npc_%objectId%_Olympiad 3_0">arena 1</a></body></html>`,
	"olympiad/olympiad_observe2.htm": `<html><body>list %state1% %players1% %state2%` +
		`<a action="bypass OlympiadArenaChange 1">2</a></body></html>`,
	"olympiad/olympiad_ranking.htm": `<html><body>ranking %place1% %rank1% %place2% %rank2% %place3%</body></html>`,
	"olympiad/hero_main.htm":        `<html><body>hero main</body></html>`,
	"olympiad/herohistory.htm":      `<html><body>%heroname% %win% %loos% %draw% %list%%buttprev%%buttnext%</body></html>`,
	"olympiad/herodiary.htm":        `<html><body>%heroname% says %message% %list%%buttprev%%buttnext%</body></html>`,
}

// olyEnvOptions configure newOlympiadEnv.
type olyEnvOptions struct {
	prev      *testEnv // a restart: prev's characters and object IDs
	start     time.Time
	store     storage.OlympiadStore // nil: a new memstore
	configure func(o *config.Olympiad)
}

// newOlympiadEnv starts a server for the Olympiad tests.
func newOlympiadEnv(t *testing.T, o olyEnvOptions) (*testEnv, storage.OlympiadStore) {
	t.Helper()
	d := testGameData(t)
	w := testWorldData(t)
	dataDir := t.TempDir()
	for name, content := range olympiadTestPages {
		writeDialogFile(t, filepath.Join(dataDir, "html", filepath.FromSlash(name)), content)
	}
	stadiums, err := json.Marshal([]data.OlympiadStadium{{X: -20814, Y: -21189, Z: -3030}, {X: -120324, Y: -225077, Z: -3331}})
	if err != nil {
		t.Fatal(err)
	}
	writeDialogFile(t, filepath.Join(dataDir, filepath.FromSlash(data.OlympiadStadiumsFile)), string(stadiums))
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = dataDir
	if o.configure != nil {
		o.configure(&gs.Olympiad)
	}
	store, ids := newRecordingStore(), memstore.NewIDs()
	if o.prev != nil {
		store, ids = o.prev.store, o.prev.ids
	}
	if o.store == nil {
		o.store = memstore.NewOlympiad()
	}
	if o.start.IsZero() {
		o.start = testStart
	}
	clock := world.NewFakeClock(o.start)
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w,
		Characters: store, Clans: store.Clans(), IDs: ids, Clock: clock, CheckLoopOwner: true,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		OlympiadStore: o.store, TimeZone: time.UTC, Rand: rnd.New(11),
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}, o.store
}

// enterNoble creates a level 76 human fighter of class (made a noble when
// noble is set) at the village and enters the world with it.
func (e *testEnv) enterNoble(account string, class int, noble bool, items ...storage.Item) (*fakeSession, *Player) {
	e.t.Helper()
	name := strings.ToUpper(account[:1]) + account[1:]
	sess, p := e.enterClanChar(clanChar{account: account, name: name, level: 76, class: classHumanFighter, items: items})
	e.onLoop(func() {
		if class != classHumanFighter {
			e.srv.ChangeClass(p, class)
		}
		if noble {
			p.SetNoble(true)
		}
	})
	e.sync()
	sess.take(e.t)
	return sess, p
}

// olyManager spawns a Grand Olympiad Manager next to p and talks to it.
func (e *testEnv) olyManager(sess *fakeSession, p *Player) (*Npc, []testclient.Packet) {
	e.t.Helper()
	n := e.spawnNpc(olympiadManagerNpc, eastOf(e.position(p), 30))
	return n, e.talk(sess, p, n)
}

// olyBypass sends "npc_<n>_<command>" after the manager's page offered it.
func (e *testEnv) olyBypass(sess *fakeSession, p *Player, n *Npc, command string) []testclient.Packet {
	e.t.Helper()
	e.talk(sess, p, n)
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_"+command)
}

// appear finishes a teleport (the client's Appearing).
func (e *testEnv) appear(sess *fakeSession) {
	e.t.Helper()
	e.srv.Handle(sess, clientpackets.OpAppearing, nil)
	e.sync()
}

// waitOlympiadWrites waits until the olympiad writer is idle.
func (e *testEnv) waitOlympiadWrites() {
	e.t.Helper()
	var w *olympiadWriter
	e.onLoop(func() { w = e.srv.olympiad.writer })
	if err := w.wait(e.ctx); err != nil {
		e.t.Fatal(err)
	}
}

// hasSysMsg reports whether ps has the system message id with its number
// (when num >= 0) or its text (when text is set).
func hasSysMsg(ps []testclient.Packet, id int32, text string, num int64) bool {
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID != id {
			continue
		}
		okText := text == ""
		okNum := num < 0
		for _, prm := range m.Params {
			if text != "" && prm.Text == text {
				okText = true
			}
			if num >= 0 && prm.Number == num {
				okNum = true
			}
		}
		if okText && okNum {
			return true
		}
	}
	return false
}
