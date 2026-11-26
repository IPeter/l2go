package game

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/fishing"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/geo/geotest"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The fishing scenarios of M11g: casting conditions and the bait on
// geodata, the bite and the fight with scripted draws, night lures, the
// fishermen's skills and the weekly championship across restarts.

const (
	fishTestRod        = 6529  // Baby Duck Rod
	fishTestLure       = 6519  // Green Colored Lure - Low Grade
	fishTestNightLure  = 8505  // Green Luminous Lure - Low Grade
	fishTestShot       = 6535  // Fishing Shot: non-grade
	fishTestFish       = 6411  // Small Nimble Green Fish
	fishTestFisherman  = 31562 // Klufe
	fishTestMonster    = 18319 // the fishing monster of levels 6-15
	fishTestWaterWest  = 100   // the water starts 100 units east of the start
	fishTestWaterEast  = 400
	fishTestWaterBelow = 50 // the water's top is 50 below the start
)

var loadFishTestData = sync.OnceValues(func() (*data.Fishing, error) {
	return data.LoadFishing(filepath.Join("..", "..", "data"))
})

// fishTestWorld is a lake east of start: a water zone from 100 to 400 units
// east, its top 50 below start, inside a fishing zone; a fisherman and the
// fishing monster.
func fishTestWorld(start data.Location) data.WorldData {
	cuboid := func(x0, y0, x1, y1, z0, z1 int) data.Shape {
		return data.Shape{Type: data.ShapeCuboid, MinZ: z0, MaxZ: z1, Points: []data.Point{{X: x0, Y: y0}, {X: x1, Y: y1}}}
	}
	return data.WorldData{
		NpcTypes: []data.MobiusClass{
			{Name: "Attackable", Parent: "Npc"},
			{Name: "Creature", Parent: "WorldObject", Abstract: true},
			{Name: "Fisherman", Parent: "Merchant", Usable: true},
			{Name: "Folk", Parent: "Npc", Usable: true},
			{Name: "Merchant", Parent: "Folk", Usable: true},
			{Name: "Monster", Parent: "Attackable", Usable: true},
			{Name: "Npc", Parent: "Creature"},
		},
		Npcs: []data.NpcTemplate{
			npcTestTemplate(fishTestMonster, "Monster", "Fishing Monster", 10),
			npcTestTemplate(fishTestFisherman, "Fisherman", "Klufe", 70),
		},
		// The world needs a spawn: a fisherman far west.
		Spawns: []data.Spawn{{
			File: "TalkingIsland/TestFisherman.xml",
			Npcs: []data.SpawnNpc{{NpcID: fishTestFisherman, X: start.X - 4000, Y: start.Y, Z: start.Z, Count: 1, RespawnDelay: 60}},
		}},
		ZoneTypes: []data.MobiusClass{
			{Name: "FishingZone", Parent: "ZoneType", Usable: true},
			{Name: "WaterZone", Parent: "ZoneType", Usable: true},
			{Name: "ZoneType", Abstract: true},
		},
		Zones: []data.Zone{{
			ID: 300001, Name: "test_fishing", Type: "FishingZone", File: "fishing.xml",
			Shape: cuboid(start.X-500, start.Y-500, start.X+600, start.Y+500, start.Z-1000, start.Z+1000),
		}, {
			ID: 300002, Name: "test_lake", Type: "WaterZone", File: "water.xml",
			Shape: cuboid(start.X+fishTestWaterWest, start.Y-200, start.X+fishTestWaterEast, start.Y+200, start.Z-500, start.Z-fishTestWaterBelow),
		}},
		MapRegions: []data.MapRegion{{
			Name: data.DefaultRespawnRegion, Town: "Talking Island Village", LocID: 910, Castle: 1, BBS: 1, File: "talking_island_town.xml",
			RespawnPoints: []data.Location{start},
		}},
	}
}

// fishEnvOptions configure newFishingEnv.
type fishEnvOptions struct {
	store storage.FishingStore // nil: a new memstore
	start time.Time            // zero: testStart
	geo   geo.Engine           // nil: no geodata
}

// The championship pages of the fishermen, cut to their placeholders.
var fishTestPages = map[string]string{
	"fisherman/31562.htm": `<html><body>Klufe:<br><a action="bypass -h npc_%objectId%_FishingChampionship">Contest</a>` +
		`<a action="bypass -h npc_%objectId%_FishSkillList">Learn</a><a action="bypass -h npc_%objectId%_FishingReward">Prize</a></body></html>`,
	fishPageWinners: "<html><body>winners %TABLE% %prizeItem% %prizeFirst% %prizeFive% in %refresh% npc %objectId% " +
		`<a action="bypass -h npc_%objectId%_FishingReward">Claim</a></body></html>`,
	fishPageRanking:    "<html><body>ranking %TABLE% %prizeItem% %prizeTwo%</body></html>",
	fishPageRefreshing: "<html><body>being updated</body></html>",
	fishPageReward:     "<html><body>here is your prize</body></html>",
	fishPageNoReward:   "<html><body>no prize for you</body></html>",
}

// newFishingEnv starts a server on fishTestWorld with the skill and fishing
// data, the store and the calendar in UTC.
func newFishingEnv(t *testing.T, o fishEnvOptions) (*testEnv, storage.FishingStore) {
	t.Helper()
	d := testGameData(t)
	fd, err := loadFishTestData()
	if err != nil {
		t.Fatalf("load the fishing data: %v", err)
	}
	w, err := data.NewWorld(fishTestWorld(npcTestStart(t)))
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	dataDir := t.TempDir()
	for name, content := range fishTestPages {
		writeDialogFile(t, filepath.Join(dataDir, "html", filepath.FromSlash(name)), content)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = dataDir
	if o.store == nil {
		o.store = memstore.NewFishing()
	}
	if o.start.IsZero() {
		o.start = testStart
	}
	clock := world.NewFakeClock(o.start)
	ids := memstore.NewIDs()
	store := newRecordingStore()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w, Geo: o.geo,
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		Fishing: fd, FishingStore: o.store, TimeZone: time.UTC, Rand: rnd.New(7),
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
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}, o.store
}

// fishScript gives fishing its own scripted random source.
func (e *testEnv) fishScript(draws ...rnd.Draw) *rnd.Scripted {
	e.t.Helper()
	r := rnd.NewScripted(e.t, draws...)
	e.onLoop(func() { e.srv.fishing.rnd = r })
	return r
}

// fisher describes a fishing character for enterFisher.
type fisher struct {
	account string
	level   int32
	pos     *world.Position // nil: the start
	heading int32
	rod     bool
	lure    int32 // 0: none; worn in the left hand
	lures   int64
	shots   int64
	skills  []storage.CharacterSkill
}

// fishSkills are Fishing, Pumping, Reeling and Fishing Expertise, level 1.
var fishSkills = []storage.CharacterSkill{{ID: 1312, Level: 1}, {ID: 1313, Level: 1}, {ID: 1314, Level: 1}, {ID: 1315, Level: 1}}

// enterFisher creates a human fighter as f describes (the Squire's Sword
// unworn), enters the world, lets a second pass and drops the packets.
func (e *testEnv) enterFisher(f fisher) (*fakeSession, *Player) {
	e.t.Helper()
	name := strings.ToUpper(f.account[:1]) + f.account[1:]
	c := e.createCharacter(f.account, name, classHumanFighter, storage.AccessLevelPlayer)
	if f.level > 1 {
		c.Level, c.Exp = f.level, e.srv.ExpForLevel(int(f.level))
	}
	if f.pos != nil {
		c.X, c.Y, c.Z = f.pos.X, f.pos.Y, f.pos.Z
	}
	c.Heading = f.heading
	c.Skills = f.skills
	for i := range c.Items {
		if c.Items[i].Slot == storage.PaperdollRHand {
			c.Items[i].Slot = storage.PaperdollNone
		}
	}
	var extra []storage.Item
	if f.rod {
		extra = append(extra, storage.Item{ItemID: fishTestRod, Count: 1, Slot: storage.PaperdollRHand, Mana: storage.NoMana})
	}
	if f.lure != 0 {
		extra = append(extra, storage.Item{ItemID: f.lure, Count: f.lures, Slot: storage.PaperdollLHand, Mana: storage.NoMana})
	}
	if f.shots > 0 {
		extra = append(extra, storage.Item{ItemID: fishTestShot, Count: f.shots, Slot: storage.PaperdollNone, Mana: storage.NoMana})
	}
	if len(extra) > 0 {
		ids, err := e.ids.NextIDs(e.ctx, len(extra))
		if err != nil {
			e.t.Fatalf("NextIDs: %v", err)
		}
		for i := range extra {
			extra[i].ObjectID = ids[i]
		}
		c.Items = append(c.Items, extra...)
	}
	if err := e.store.Update(e.ctx, &c); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	sess := newSession(f.account)
	p := e.enter(sess, 0)
	e.advance(time.Second)
	sess.take(e.t)
	return sess, p
}

// countOf returns how many of an item p has, read on the loop.
func (e *testEnv) countOf(p *Player, itemID int32) int64 {
	var n int64
	e.onLoop(func() { n = p.inv.Count(itemID, -1, true) })
	return n
}

// fishDescribe writes the fishing packets and system messages as short
// lines.
func fishDescribe(ps []testclient.Packet) []string {
	var out []string
	for _, p := range ps {
		switch v := p.(type) {
		case testclient.ExFishingStart:
			out = append(out, fmt.Sprintf("Start type %d at %d,%d,%d night %v", v.FishType, v.X, v.Y, v.Z, v.Night))
		case testclient.ExFishingStartCombat:
			out = append(out, fmt.Sprintf("Combat time %d hp %d mode %d lure %d deceptive %d", v.Time, v.HP, v.Mode, v.LureType, v.Deceptive))
		case testclient.ExFishingHpRegen:
			out = append(out, fmt.Sprintf("Regen time %d hp %d mode %d good %d anim %d pen %d", v.Time, v.HP, v.Mode, v.GoodUse, v.Anim, v.Penalty))
		case testclient.ExFishingEnd:
			out = append(out, fmt.Sprintf("End win %v", v.Win))
		case testclient.PlaySound:
			out = append(out, "Sound "+v.Sound)
		case testclient.SystemMessage:
			if v.ID == 46 { // "You use $s1.", as for any skill but Fishing
				continue
			}
			s := fmt.Sprintf("msg %d", v.ID)
			for _, prm := range v.Params {
				switch {
				case prm.Text != "":
					s += " " + prm.Text
				case len(prm.Numbers) > 0:
					s += fmt.Sprintf(" %v", prm.Numbers)
				default:
					s += fmt.Sprintf(" %d", prm.Number)
				}
			}
			out = append(out, s)
		}
	}
	return out
}

func checkFish(t *testing.T, what string, ps []testclient.Packet, want ...string) {
	t.Helper()
	if got := fishDescribe(ps); !slices.Equal(got, want) {
		t.Errorf("%s:\n got %q\nwant %q", what, got, want)
	}
}

// TestFishingCatch is the demo: a level 10 human with a Baby Duck Rod and
// green lures casts into the lake, gets a bite after 10 s, pumps the resting
// fish three times and lands it; the fish goes into the inventory and the
// championship.
func TestFishingCatch(t *testing.T) {
	e, _ := newFishingEnv(t, fishEnvOptions{})
	sess, p := e.enterFisher(fisher{account: "angler", level: 10, rod: true, lure: fishTestLure, lures: 10, skills: fishSkills})
	start := npcTestStart(t)
	r := e.fishScript(
		rnd.Between(90, 250, 200), // the bait: 200 units east, in the lake
		rnd.Get(100, 0),           // fish level 1 (Fishing Expertise 1)
		rnd.Get(100, 0),           // group swift
		rnd.Get(1, 0),             // the one fish of level 1, swift, normal
	)
	castUse(e, sess, fishing.SkillFishing, false, false)
	checkFish(t, "cast", sess.take(t),
		"msg 1461",
		fmt.Sprintf("Start type 1 at %d,%d,%d night false", start.X+200, start.Y, start.Z-fishTestWaterBelow),
		"Sound SF_P_01")
	if got := e.countOf(p, fishTestLure); got != 9 {
		t.Errorf("lures left %d, want 9", got)
	}
	var on bool
	e.onLoop(func() { on = e.srv.Fishing(p) && p.immobilized() })
	if !on {
		t.Fatal("not fishing (and immobilized) after the cast")
	}
	r.Done()

	// No bite check before 10 s; then a bite, and the fish rests.
	e.advance(9 * time.Second)
	checkFish(t, "waiting", sess.take(t))
	r.Add(rnd.Get(100, 0), rnd.Get(100, 0)) // bites (guts 50 > 0), rests (0 < 80)
	e.advance(time.Second)
	checkFish(t, "bite", sess.take(t),
		"Combat time 24 hp 100 mode 0 lure 1 deceptive 0", "Sound SF_S_01", "msg 1449")
	r.Done()

	// Pumping a resting fish: (20 + 0.66 + 0.66) × 2 = 42 damage.
	pump := func(what string, hp int) {
		t.Helper()
		r.Add(rnd.Get(100, 0)) // not resisted
		castUse(e, sess, fishing.SkillPumping, false, false)
		checkFish(t, what, sess.take(t), "msg 1465 42", fmt.Sprintf("Regen time %d hp %d mode 0 good 1 anim 1 pen 0", fishTimeLeft(e, p), hp))
		r.Done()
	}
	pump("first pump", 58)
	r.Add(rnd.Get(100, 0)) // the first second: stays resting
	e.advance(2 * time.Second)
	checkFish(t, "two seconds", sess.take(t), "Regen time 23 hp 58 mode 0 good 0 anim 0 pen 0", "Regen time 22 hp 58 mode 0 good 0 anim 0 pen 0")
	pump("second pump", 16)
	r.Add(rnd.Get(100, 0))
	e.advance(2 * time.Second)
	sess.take(t)

	// The third pump lands it: not a monster (50 > 5), 75.5 long.
	r.Add(rnd.Get(100, 0), rnd.Get(100, 50), rnd.Between(60, 89, 75), rnd.Between(0, 1000, 500))
	castUse(e, sess, fishing.SkillPumping, false, false)
	got := fishDescribe(sess.take(t))
	want := []string{"msg 1465 42", "Regen time 20 hp 0 mode 0 good 1 anim 1 pen 0", "msg 1469",
		fmt.Sprintf("msg 30 %d", fishTestFish), // "You have obtained $s1."
		"msg 1847 75.5", "msg 1848", "End win true", "msg 1460"}
	if !slices.Equal(got, want) {
		t.Errorf("landing:\n got %q\nwant %q", got, want)
	}
	r.Done()
	if got := e.countOf(p, fishTestFish); got != 1 {
		t.Errorf("fish in the inventory %d, want 1", got)
	}
	var running []fishing.Fisher
	e.onLoop(func() {
		on = e.srv.Fishing(p) || p.immobilized()
		running, _ = e.srv.fishing.champ.Ranking()
	})
	if on {
		t.Error("still fishing or immobilized after the catch")
	}
	if len(running) != 1 || running[0].Name != "Angler" || running[0].Length != 75.5 {
		t.Errorf("championship ranking %+v, want Angler 75.5", running)
	}
}

// fishTimeLeft is the seconds left of p's fight.
func fishTimeLeft(e *testEnv, p *Player) int {
	var n int
	e.onLoop(func() {
		if st := e.srv.fishing.states[p]; st != nil && st.fight != nil {
			n = st.fight.TimeLeft()
		}
	})
	return n
}

// TestFishingCastConditions checks the refusals of the Fishing effect, in
// Mobius's order, none of which uses a lure, and the bait placement.
func TestFishingCastConditions(t *testing.T) {
	e, _ := newFishingEnv(t, fishEnvOptions{})
	start := npcTestStart(t)
	inWater := world.Position{X: int32(start.X) + 200, Y: int32(start.Y), Z: int32(start.Z) - 100}
	for _, c := range []struct {
		name  string
		f     fisher
		draws []rnd.Draw
		want  []string
	}{
		// The skill's own condition (using FISHINGROD) refuses first, with
		// "$s1 cannot be used due to unsuitable terms."; the effect's "You do
		// not have a fishing pole equipped." is not reached, as in Mobius.
		{"no rod", fisher{account: "norod", lure: fishTestLure, lures: 5}, nil, []string{"msg 113 [1312 1]"}},
		{"no lure", fisher{account: "nolure", rod: true}, nil, []string{"msg 1454"}},
		{"under water", fisher{account: "diver", rod: true, lure: fishTestLure, lures: 5, pos: &inWater}, nil, []string{"msg 1455"}},
		// Facing west: no water anywhere from 250 down to 90.
		{"facing away", fisher{account: "west", rod: true, lure: fishTestLure, lures: 5, heading: 32768},
			[]rnd.Draw{rnd.Between(90, 250, 150)}, []string{"msg 1457"}},
	} {
		c.f.skills = fishSkills
		sess, p := e.enterFisher(c.f)
		r := e.fishScript(c.draws...)
		castUse(e, sess, fishing.SkillFishing, false, false)
		checkFish(t, c.name, sess.take(t), c.want...)
		r.Done()
		if c.f.lures > 0 {
			if got := e.countOf(p, c.f.lure); got != c.f.lures {
				t.Errorf("%s: lures %d, want %d (none used)", c.name, got, c.f.lures)
			}
		}
	}

	// A bait too close for the lake (90) falls back to the farthest point
	// that is in it: 250.
	sess, _ := e.enterFisher(fisher{account: "close", rod: true, lure: fishTestLure, lures: 5, skills: fishSkills})
	r := e.fishScript(rnd.Between(90, 250, 90), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(1, 0))
	castUse(e, sess, fishing.SkillFishing, false, false)
	checkFish(t, "fallback", sess.take(t), "msg 1461",
		fmt.Sprintf("Start type 1 at %d,%d,%d night false", start.X+250, start.Y, start.Z-fishTestWaterBelow), "Sound SF_P_01")
	r.Done()
}

// TestFishingWhileFishing checks what a fishing player may not do, and a
// second cast, which cancels the fishing.
func TestFishingWhileFishing(t *testing.T) {
	e, _ := newFishingEnv(t, fishEnvOptions{})
	sess, p := e.enterFisher(fisher{account: "busy", rod: true, lure: fishTestLure, lures: 5, skills: fishSkills})
	// Pumping before any bite is refused before the cast starts.
	castUse(e, sess, fishing.SkillPumping, false, false)
	checkFish(t, "pumping without fishing", sess.take(t), "msg 1462")
	r := e.fishScript(rnd.Between(90, 250, 200), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(1, 0))
	castUse(e, sess, fishing.SkillFishing, false, false)
	sess.take(t)
	r.Done()

	castUse(e, sess, fishing.SkillReeling, false, false)
	checkFish(t, "reeling before the bite", sess.take(t), "msg 1463")
	var lure int32
	e.onLoop(func() { lure = p.inv.ByItemID(fishTestLure).ObjectID() })
	ps := e.handleUse(sess, lure)
	checkFish(t, "an item", ps, "msg 1471")
	start := e.position(p)
	e.srv.Handle(sess, clientpackets.OpMoveToLocation, moveBody(start.X+100, start.Y, start.Z, start))
	e.sync()
	sess.take(t)
	if got := e.position(p); got.X != start.X {
		t.Errorf("walked to %v while fishing", got)
	}

	// A second cast before the bite ends it: the bait is lost.
	castUse(e, sess, fishing.SkillFishing, false, false)
	checkFish(t, "second cast", sess.take(t), "msg 1452", "End win false", "msg 1460", "msg 1458")
	if got := e.countOf(p, fishTestLure); got != 4 {
		t.Errorf("lures %d, want 4", got)
	}
}

// handleUse sends UseItem for an item and returns the packets.
func (e *testEnv) handleUse(sess *fakeSession, objectID int32) []testclient.Packet {
	e.t.Helper()
	e.srv.Handle(sess, clientpackets.OpUseItem, ints(objectID, 0))
	e.sync()
	return sess.take(e.t)
}

// moveBody encodes MoveToLocation from origin to (x, y, z) with the mouse.
func moveBody(x, y, z int32, origin world.Position) []byte {
	return ints(x, y, z, origin.X, origin.Y, origin.Z, 1)
}

// TestFishingNightLure checks that a night lure by day shows fish type -1,
// never bites, and the fishing ends at the fish's wait limit with the bait
// lost.
func TestFishingNightLure(t *testing.T) {
	e, _ := newFishingEnv(t, fishEnvOptions{})
	sess, p := e.enterFisher(fisher{account: "owl", rod: true, lure: fishTestNightLure, lures: 3, skills: fishSkills})
	start := npcTestStart(t)
	r := e.fishScript(rnd.Between(90, 250, 200), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(1, 0))
	castUse(e, sess, fishing.SkillFishing, false, false)
	checkFish(t, "cast by day", sess.take(t), "msg 1461",
		fmt.Sprintf("Start type -1 at %d,%d,%d night true", start.X+200, start.Y, start.Z-fishTestWaterBelow), "Sound SF_P_01")
	// 10 s, then a check every 665 ms (guts check time 5 × 133) without a
	// draw; the first one from 30 s (startCombatTime 20 + 10), at 30.615 s,
	// gives up.
	e.advance(29 * time.Second)
	checkFish(t, "waiting", sess.take(t))
	e.advance(2 * time.Second)
	checkFish(t, "given up", sess.take(t), "msg 1452", "End win false", "msg 1460")
	r.Done()
	var on bool
	e.onLoop(func() { on = e.srv.Fishing(p) })
	if on {
		t.Error("still fishing")
	}
}

// TestFishingAtNight checks that a night lure bites at night, fighting like
// an upper-grade fish (lure type 2, deceptive roll).
func TestFishingAtNight(t *testing.T) {
	// 12:30 UTC is 03:00 game time.
	e, _ := newFishingEnv(t, fishEnvOptions{start: testStart.Add(-30 * time.Minute)})
	sess, _ := e.enterFisher(fisher{account: "bat", rod: true, lure: fishTestNightLure, lures: 3, skills: fishSkills})
	r := e.fishScript(rnd.Between(90, 250, 200), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(1, 0))
	castUse(e, sess, fishing.SkillFishing, false, false)
	if got := only[testclient.ExFishingStart](sess.take(t)); len(got) != 1 || got[0].FishType != 1 || !got[0].Night {
		t.Fatalf("ExFishingStart at night = %+v", got)
	}
	r.Add(rnd.Get(100, 10), rnd.Get(100, 95), rnd.Get(100, 85)) // bites; deceptive; fights
	e.advance(10 * time.Second)
	checkFish(t, "bite", sess.take(t), "Combat time 24 hp 100 mode 1 lure 2 deceptive 1", "Sound SF_S_01", "msg 1449")
	r.Done()
}

// TestFishingShots checks a fishing shot: it charges the rod, doubles the
// pumping damage and is used up by it.
func TestFishingShots(t *testing.T) {
	e, _ := newFishingEnv(t, fishEnvOptions{})
	sess, p := e.enterFisher(fisher{account: "shooter", rod: true, lure: fishTestLure, lures: 3, shots: 5, skills: fishSkills})
	var shot int32
	e.onLoop(func() { shot = p.inv.ByItemID(fishTestShot).ObjectID() })
	ps := e.handleUse(sess, shot)
	if len(only[testclient.MagicSkillUse](ps)) != 1 || e.countOf(p, fishTestShot) != 4 {
		t.Fatalf("fishing shot: %q, %d left", testclient.Names(ps), e.countOf(p, fishTestShot))
	}
	ps = e.handleUse(sess, shot)
	if len(only[testclient.MagicSkillUse](ps)) != 0 || e.countOf(p, fishTestShot) != 4 {
		t.Errorf("a second shot on a charged rod: %q", testclient.Names(ps))
	}
	r := e.fishScript(rnd.Between(90, 250, 200), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(1, 0))
	castUse(e, sess, fishing.SkillFishing, false, false)
	r.Add(rnd.Get(100, 0), rnd.Get(100, 0))
	e.advance(10 * time.Second)
	sess.take(t)
	r.Add(rnd.Get(100, 0))
	castUse(e, sess, fishing.SkillPumping, false, false)
	checkFish(t, "charged pump", sess.take(t), "msg 1465 85", "Regen time 24 hp 15 mode 0 good 1 anim 1 pen 0")
	var charged bool
	e.onLoop(func() { charged = playerChargedShot(p, skills.FishSoulshot) })
	if charged {
		t.Error("the shot stayed charged")
	}
	r.Done()
}

// TestFishingGeodata casts over a geotest map: a wall between the player
// and the lake hides every bait point; over the lake's lowered bottom the
// bait lands.
func TestFishingGeodata(t *testing.T) {
	start := npcTestStart(t)
	// Columns are 16 units: the lake (100-400 east) starts in column 7.
	open := newGeoMap(t,
		"A......wwwwwwwwwwwwwwwwwww",
	)
	walled := newGeoMap(t,
		"A...#..wwwwwwwwwwwwwwwwwww",
	)
	// The lake's bottom is 64 below the shore: under the water's top, and
	// low enough to see the bait over the shore.
	for _, m := range []*geoMap{open, walled} {
		m.Define('w', geotest.Cell{Floors: []int{int(m.ground) - 64}})
	}
	for _, c := range []struct {
		name string
		m    *geoMap
		want []string
	}{
		{"open", open, []string{"msg 1461", fmt.Sprintf("Start type 1 at %d,%d,%d night false", start.X+200, start.Y, start.Z-fishTestWaterBelow), "Sound SF_P_01"}},
		{"behind a wall", walled, []string{"msg 1457"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e, _ := newFishingEnv(t, fishEnvOptions{geo: c.m.Load(t)})
			pos := world.Position{X: int32(start.X), Y: int32(start.Y), Z: c.m.ground}
			sess, _ := e.enterFisher(fisher{account: "geo", rod: true, lure: fishTestLure, lures: 3, pos: &pos, skills: fishSkills})
			draws := []rnd.Draw{rnd.Between(90, 250, 200)}
			if c.m == open {
				draws = append(draws, rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(1, 0))
			}
			r := e.fishScript(draws...)
			castUse(e, sess, fishing.SkillFishing, false, false)
			checkFish(t, c.name, sess.take(t), c.want...)
			r.Done()
		})
	}
}
