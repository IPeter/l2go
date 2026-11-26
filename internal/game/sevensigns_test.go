package game

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/sevensigns"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The Seven Signs on the fake clock (Plan 14 M11l): a week from recruiting
// through the competition to the seal validation, the seals' effects, the
// dungeons, a restart, the festival and the rift.

const (
	ssqTestDawnPriest  = 31078
	ssqTestGatekeeper  = 31095
	ssqTestGuide       = 31127 // Dawn festival guide, level 31 or lower
	ssqTestRiftGuard   = 31865
	ssqTestFragment    = 7079
	ssqTestOffering    = 5901
	ssqTestDuskGuide   = 31137
	ssqTestVictorOfWar = 5074
	ssqTestVanquished  = 5075
)

// Monday 2026-11-16 18:00 UTC: a period change.
var ssqMonday = time.Date(2026, 11, 16, 18, 0, 0, 0, time.UTC)

var loadSSQTestData = sync.OnceValues(func() (*data.SevenSigns, error) {
	return data.LoadSevenSigns(filepath.Join("..", "..", "data"))
})

// ssqTestWorld has the priests, a dungeon gatekeeper, the festival guides
// and a rift guard, all at start.
func ssqTestWorld(start data.Location) data.WorldData {
	return data.WorldData{
		NpcTypes: []data.MobiusClass{
			{Name: "Creature", Parent: "WorldObject", Abstract: true},
			{Name: "Npc", Parent: "Creature"},
			{Name: "Folk", Parent: "Npc", Usable: true},
			{Name: "SignsPriest", Parent: "Npc", Usable: true},
			{Name: "DawnPriest", Parent: "SignsPriest", Usable: true},
			{Name: "DungeonGatekeeper", Parent: "Npc", Usable: true},
			{Name: "FestivalGuide", Parent: "Npc", Usable: true},
		},
		Npcs: []data.NpcTemplate{
			npcTestTemplate(ssqTestDawnPriest, "DawnPriest", "Priest of Dawn", 70),
			npcTestTemplate(ssqTestGatekeeper, "DungeonGatekeeper", "Gatekeeper Ziggurat", 70),
			npcTestTemplate(ssqTestGuide, "FestivalGuide", "Festival Guide", 70),
			npcTestTemplate(ssqTestDuskGuide, "FestivalGuide", "Festival Guide", 70),
			npcTestTemplate(ssqTestRiftGuard, "Folk", "Guardian of the Border", 70),
		},
		Spawns: []data.Spawn{{
			File: "TalkingIsland/TestPriest.xml",
			Npcs: []data.SpawnNpc{{NpcID: ssqTestDawnPriest, X: start.X - 4000, Y: start.Y, Z: start.Z, Count: 1, RespawnDelay: 60}},
		}},
		MapRegions: []data.MapRegion{{
			Name: data.DefaultRespawnRegion, Town: "Talking Island Village", LocID: 910, Castle: 1, BBS: 1, File: "talking_island_town.xml",
			RespawnPoints: []data.Location{start},
		}},
	}
}

// ssqEnvOptions configure newSSQEnv.
type ssqEnvOptions struct {
	store storage.SevenSignsStore // nil: a new memstore
	start time.Time
	cfg   func(c *config.SevenSigns)
}

// newSSQEnv starts a server with the Seven Signs data, the store and the
// calendar in UTC.
func newSSQEnv(t *testing.T, o ssqEnvOptions) (*testEnv, storage.SevenSignsStore) {
	t.Helper()
	d := testGameData(t)
	sd, err := loadSSQTestData()
	if err != nil {
		t.Fatalf("load the seven signs data: %v", err)
	}
	w, err := data.NewWorld(ssqTestWorld(npcTestStart(t)))
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = filepath.Join("..", "..", "data")
	if o.cfg != nil {
		o.cfg(&gs.SevenSigns)
	}
	if o.store == nil {
		o.store = memstore.NewSevenSigns()
	}
	clock := world.NewFakeClock(o.start)
	ids := memstore.NewIDs()
	store := newRecordingStore()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w,
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		SevenSigns: sd, SevenSignsStore: o.store, TimeZone: time.UTC, Rand: rnd.New(7),
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
	e := &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
	e.sync() // the calendar starts on the loop
	return e, o.store
}

func (e *testEnv) ssqState() sevensigns.Status {
	var st sevensigns.Status
	e.onLoop(func() { st = e.srv.sevenSigns.state.Status })
	return st
}

func (e *testEnv) ssqNext() time.Time {
	var t time.Time
	e.onLoop(func() { t = e.srv.sevenSigns.next })
	return t
}

func (e *testEnv) ssqRecord(sess *fakeSession, page int) testclient.SSQStatus {
	e.t.Helper()
	sess.take(e.t)
	e.srv.Handle(sess, clientpackets.OpRequestSSQStatus, []byte{byte(page)})
	e.sync()
	for _, pk := range sess.take(e.t) {
		if r, ok := pk.(testclient.SSQStatus); ok {
			return r
		}
	}
	e.t.Fatalf("no SSQStatus for page %d", page)
	return testclient.SSQStatus{}
}

func (e *testEnv) hasSkill(p *Player, id int) bool {
	var ok bool
	e.onLoop(func() { ok = p.KnownSkill(id) != nil })
	return ok
}

// TestSevenSignsWeek runs a week: recruiting, the competition with joins
// and stones, the results and the new seal owners, the seal validation with
// the Seal of Strife's skills, the dungeons, the record, a restart, and the
// next cycle.
func TestSevenSignsWeek(t *testing.T) {
	store := memstore.NewSevenSigns()
	// A server stopped in the recruiting period, its change due at 18:15.
	if err := store.SaveSevenSigns(t.Context(), storage.SevenSignsSave{Status: storage.SevenSignsStatus{
		Cycle: 1, Period: int32(sevensigns.PeriodRecruiting), FestivalCycle: 1, AccumulatedBonus: make([]int32, 5),
		Date: ssqMonday.UnixMilli(), NextPeriodChange: ssqMonday.Add(15 * time.Minute).UnixMilli(),
	}}); err != nil {
		t.Fatal(err)
	}
	e, _ := newSSQEnv(t, ssqEnvOptions{store: store, start: ssqMonday.Add(time.Minute)})
	if st := e.ssqState(); st.Period != sevensigns.PeriodRecruiting {
		t.Fatalf("period %v after the start", st.Period)
	}
	sa, a := e.enterChar(testChar{account: "dawny", name: "Dawny", pos: tiStart, level: 40})
	sb, b := e.enterChar(testChar{account: "dusky", name: "Dusky", pos: tiStart, level: 40})

	// 18:15: the competition.
	sa.take(t)
	e.advance(14 * time.Minute)
	if st := e.ssqState(); st.Period != sevensigns.PeriodCompetition {
		t.Fatalf("period %v at 18:15", st.Period)
	}
	if ids := sysMsgIDsOf(sa.take(t)); !slices.Contains(ids, sevensigns.MsgCompetitionBegunVisit) {
		t.Errorf("no competition notice: %v", ids)
	}
	if want := ssqMonday.AddDate(0, 0, 7); !e.ssqNext().Equal(want) {
		t.Errorf("next change %v, want %v", e.ssqNext(), want)
	}

	// Joins and stones, as the priests' script does them.
	e.onLoop(func() {
		o := e.srv.output
		o.SevenSignsJoin(a, int(sevensigns.CabalDawn), int(sevensigns.SealStrife))
		o.SevenSignsJoin(b, int(sevensigns.CabalDusk), int(sevensigns.SealGnosis))
		if got := o.SevenSignsContribute(a, 0, 0, 10); got != 100 {
			t.Errorf("dawn contribution %d", got)
		}
		if got := o.SevenSignsContribute(b, 5, 0, 0); got != 15 {
			t.Errorf("dusk contribution %d", got)
		}
	})
	r := e.ssqRecord(sa, 1)
	if r.Cabal != 2 || r.Seal != 3 || r.StonesTurnedIn != 10 || r.AncientAdenaToCollect != 100 || r.DawnStoneScore != 435 || r.DuskStoneScore != 65 ||
		r.PeriodMsg != sevensigns.MsgQuestEventPeriod || r.PeriodEndMsg != sevensigns.MsgUntilNextMonday {
		t.Errorf("record page 1: %+v", r)
	}
	if r := e.ssqRecord(sa, 4); r.Winner != 2 || len(r.Predictions) != 3 || r.Predictions[2].Owner != 2 {
		t.Errorf("record page 4: %+v", r)
	}

	// Monday 18:00: the results. Dawn won; it takes the Seal of Strife (all
	// its members chose it); Dusk lost, so nobody takes the Seal of Gnosis.
	sb.take(t)
	e.advance(e.ssqNext().Sub(e.clock.Now()))
	st := e.ssqState()
	if st.Period != sevensigns.PeriodResults || st.PreviousWinner != sevensigns.CabalDawn ||
		st.SealOwners[sevensigns.SealStrife] != sevensigns.CabalDawn || st.SealOwners[sevensigns.SealGnosis] != sevensigns.CabalNull {
		t.Fatalf("after the competition: %+v", st)
	}
	ids := sysMsgIDsOf(sb.take(t))
	for _, want := range []int32{sevensigns.MsgCompetitionEnded, 1214, sevensigns.MsgDawnWon} {
		if !slices.Contains(ids, want) {
			t.Errorf("message %d missing from %v", want, ids)
		}
	}
	if r := e.ssqRecord(sa, 4); r.Page != 0 {
		t.Errorf("page 4 answered in the results period: %+v", r)
	}

	// 18:15: the seal validation; the Seal of Strife's skills.
	e.advance(15 * time.Minute)
	if st := e.ssqState(); st.Period != sevensigns.PeriodSealValidation {
		t.Fatalf("period %v", st.Period)
	}
	if !e.hasSkill(a, ssqTestVictorOfWar) || !e.hasSkill(b, ssqTestVanquished) {
		t.Errorf("strife skills: dawn victor %v, dusk vanquished %v", e.hasSkill(a, ssqTestVictorOfWar), e.hasSkill(b, ssqTestVanquished))
	}
	var pdef, mdef float64
	e.onLoop(func() { pdef, mdef = e.srv.SevenSignsGateDefence() })
	if pdef != 1.1 || mdef != 1.1 {
		t.Errorf("gate defence %v %v, want Dawn's 1.1", pdef, mdef)
	}

	// The necropolis: Dawn won but owns no Seal of Avarice, so nobody goes in.
	gk := e.spawnNpc(ssqTestGatekeeper, eastOf(tiStart, 20))
	sa.take(t)
	e.onLoop(func() { e.srv.npcCommand(a, gk, "necro 0") })
	ps := sa.take(t)
	if ids := sysMsgIDsOf(ps); !slices.Contains(ids, sevensigns.MsgOnlyDawn) {
		t.Errorf("necropolis refusal: %v", ids)
	}
	// The catacombs need the Seal of Gnosis: also refused. Open catacombs
	// would let anyone in; here the flag stays off.
	e.onLoop(func() {
		if a.in7sDungeon() {
			t.Error("refused, yet in a dungeon")
		}
		e.srv.sevenSigns.state.SealOwners[sevensigns.SealAvarice] = sevensigns.CabalDawn
		e.srv.npcCommand(a, gk, "necro 0")
		if !a.in7sDungeon() {
			t.Error("a lord of dawn with the Seal of Avarice is not in the necropolis")
		}
		e.srv.npcCommand(b, gk, "necro 0")
		if b.in7sDungeon() {
			t.Error("dusk entered the necropolis")
		}
	})

	// A restart keeps everything, and the next change.
	next := e.ssqNext()
	e.stop()
	e2, _ := newSSQEnv(t, ssqEnvOptions{store: store, start: e.clock.Now()})
	st2 := e2.ssqState()
	if st2.Period != sevensigns.PeriodSealValidation || st2.SealOwners[sevensigns.SealStrife] != sevensigns.CabalDawn ||
		st2.SealOwners[sevensigns.SealAvarice] != sevensigns.CabalDawn || st2.DawnStoneScore != 100 || !e2.ssqNext().Equal(next) {
		t.Fatalf("after the restart: %+v, next %v (want %v)", st2, e2.ssqNext(), next)
	}
	var cabal sevensigns.Cabal
	e2.onLoop(func() { cabal = e2.srv.sevenSigns.state.PlayerCabal(a.ObjectID()) })
	if cabal != sevensigns.CabalDawn {
		t.Errorf("cabal after the restart %v", cabal)
	}

	// The next Monday: a new cycle.
	e2.advance(next.Sub(e2.clock.Now()))
	st2 = e2.ssqState()
	if st2.Period != sevensigns.PeriodRecruiting || st2.Cycle != 2 || st2.DawnStoneScore != 0 || st2.DawnSealTotals != ([4]int{}) {
		t.Fatalf("new cycle: %+v", st2)
	}
	e2.onLoop(func() { cabal = e2.srv.sevenSigns.state.PlayerCabal(a.ObjectID()) })
	if cabal != sevensigns.CabalNull {
		t.Errorf("cabal kept into the new cycle: %v", cabal)
	}
	if !e2.ssqNext().Equal(next.Add(15 * time.Minute)) {
		t.Errorf("recruiting ends %v", e2.ssqNext())
	}
}

// TestSevenSignsEnterWorld checks the enter-world notice and the
// //ssq_change command.
func TestSevenSignsEnterWorld(t *testing.T) {
	e, _ := newSSQEnv(t, ssqEnvOptions{start: ssqMonday.Add(time.Hour)})
	c := e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")
	e.enter(sess, 0)
	if ids := sysMsgIDsOf(sess.take(t)); !slices.Contains(ids, sevensigns.MsgCompetitionBegunSpeak) {
		t.Errorf("enter-world notice missing: %v", ids)
	}
	_ = c
	gm := e.createCharacter("admin", "Admin", classHumanFighter, storage.AccessLevelAdmin)
	_ = gm
	sg := newSession("admin")
	e.enter(sg, 0)
	sg.take(t)
	e.srv.Handle(sg, clientpackets.OpSendBypassBuildCmd, stringBody("ssq_change"))
	e.sync()
	if st := e.ssqState(); st.Period != sevensigns.PeriodResults {
		t.Errorf("after //ssq_change: %v", st.Period)
	}
	var texts []string
	for _, pk := range sg.take(t) {
		if m, ok := pk.(testclient.SystemMessage); ok {
			for _, prm := range m.Params {
				texts = append(texts, prm.Text)
			}
		}
	}
	if !strings.Contains(strings.Join(texts, "|"), "Quest Event Results") {
		t.Errorf("//ssq_info text %v", texts)
	}
}
