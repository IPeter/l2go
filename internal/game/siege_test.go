package game

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The siege scenario tests run one castle, Gludio (ID 1), 10,000 east of
// the Talking Island village, on a fake clock in UTC.

const (
	siegeTestCT       = 13002
	siegeTestFlame    = 13004
	siegeTestGuard    = 35085
	siegeTestArtefact = 35063
	siegeTestMerc     = 35010
	siegeTestTicket   = 3960
)

// siegeTestCenter is the castle's center.
func siegeTestCenter() data.Location {
	return data.Location{X: int(tiVillage.X) + 10000, Y: int(tiVillage.Y), Z: int(tiVillage.Z)}
}

// siegeTestWorld is the world of the castle: its siege, castle and
// headquarters zones, the artefact and the NPC templates.
func siegeTestWorld() data.WorldData {
	c := siegeTestCenter()
	cuboid := func(x0, y0, x1, y1 int) data.Shape {
		return data.Shape{Type: data.ShapeCuboid, MinZ: c.Z - 2000, MaxZ: c.Z + 2000, Points: []data.Point{{X: x0, Y: y0}, {X: x1, Y: y1}}}
	}
	castleID := []data.ZoneParam{{Name: "castleId", Value: "1"}}
	return data.WorldData{
		NpcTypes: []data.MobiusClass{
			{Name: "Artefact", Parent: "Npc", Usable: true},
			{Name: "Attackable", Parent: "Npc", Usable: true},
			{Name: "ControlTower", Parent: "Tower", Usable: true},
			{Name: "Creature", Parent: "WorldObject", Abstract: true},
			{Name: "Defender", Parent: "Attackable", Usable: true},
			{Name: "FlameTower", Parent: "Tower", Usable: true},
			{Name: "Folk", Parent: "Npc", Usable: true},
			{Name: "Npc", Parent: "Creature"},
			{Name: "SiegeFlag", Parent: "Npc", Usable: true},
			{Name: "Tower", Parent: "Npc", Abstract: true},
		},
		Npcs: []data.NpcTemplate{
			npcTestTemplate(siegeTestCT, "ControlTower", "Control Tower", 70),
			npcTestTemplate(siegeTestFlame, "FlameTower", "Flame Control Tower", 70),
			npcTestTemplate(siegeTestGuard, "Defender", "Gludio Guard", 60),
			npcTestTemplate(siegeTestMerc, "Defender", "Mercenary", 60),
			npcTestTemplate(siegeTestArtefact, "Artefact", "Holy Artifact", 20),
			npcTestTemplate(siegeHQNpc, "SiegeFlag", "Headquarters", 1),
		},
		Spawns: []data.Spawn{{
			File: "Castles/Gludio.xml",
			Npcs: []data.SpawnNpc{{NpcID: siegeTestArtefact, X: c.X, Y: c.Y + 500, Z: c.Z, Count: 1, RespawnDelay: 60}},
		}},
		ZoneTypes: []data.MobiusClass{
			{Name: "CastleZone", Parent: "ResidenceZone", Usable: true},
			{Name: "ResidenceZone", Parent: "ZoneRespawn", Abstract: true},
			{Name: "ZoneRespawn", Parent: "ZoneType", Abstract: true},
			{Name: "HqZone", Parent: "ZoneType", Usable: true},
			{Name: "SiegeZone", Parent: "ZoneType", Usable: true},
			{Name: "ZoneType", Abstract: true},
		},
		Zones: []data.Zone{
			{ID: 70001, Name: "test_siege", Type: "SiegeZone", File: "castle_siege.xml", Shape: cuboid(c.X-3000, c.Y-3000, c.X+3000, c.Y+3000), Params: castleID},
			{ID: 70002, Name: "test_castle", Type: "CastleZone", File: "castle_hall.xml", Shape: cuboid(c.X-1000, c.Y-1000, c.X+1000, c.Y+1000), Params: castleID,
				Spawns: []data.ZoneSpawn{{X: c.X + 100, Y: c.Y, Z: c.Z}, {X: c.X + 200, Y: c.Y, Z: c.Z, Type: data.ZoneSpawnChaotic}}},
			{ID: 70003, Name: "test_hq", Type: "HqZone", File: "castle_hq.xml", Shape: cuboid(c.X+1500, c.Y-1000, c.X+2500, c.Y+1000), Params: castleID},
		},
		MapRegions: []data.MapRegion{{
			Name: data.DefaultRespawnRegion, Town: "Talking Island Village", LocID: 910, Castle: 1, BBS: 1, File: "talking_island_town.xml",
			RespawnPoints: []data.Location{{X: int(tiVillage.X), Y: int(tiVillage.Y), Z: int(tiVillage.Z)}},
		}},
	}
}

// siegeTestCastles is Gludio with three control towers, a flame tower and
// two guards.
func siegeTestCastles(t *testing.T) *data.Castles {
	t.Helper()
	c := siegeTestCenter()
	tower := func(id, dx, dy int) data.TowerSpawn {
		return data.TowerSpawn{NpcID: id, X: c.X + dx, Y: c.Y + dy, Z: c.Z}
	}
	d, err := data.NewCastles(data.CastleData{
		Castles: []data.Castle{{ID: 1, Name: "Gludio", SiegeEnabled: true, SiegeDay: 0, SiegeHour: 16, MaxConcurrent: 5,
			ControlTowers:  []data.TowerSpawn{tower(siegeTestCT, 600, 0), tower(siegeTestCT, -600, 0), tower(siegeTestCT, 0, 600)},
			FlameTowers:    []data.TowerSpawn{tower(siegeTestFlame, 0, -600)},
			MaxMercenaries: 100}},
		Guards: []data.SiegeGuard{
			{CastleID: 1, ID: 1, NpcID: siegeTestGuard, X: c.X + 550, Y: c.Y, Z: c.Z, RespawnDelay: 60},
			{CastleID: 1, ID: 2, NpcID: siegeTestGuard, X: c.X - 550, Y: c.Y, Z: c.Z, RespawnDelay: 0},
		},
		Tickets: []data.MercTicket{{ItemID: siegeTestTicket, NpcID: siegeTestMerc, CastleID: 1, MaxPerType: 10}},
	})
	if err != nil {
		t.Fatalf("NewCastles: %v", err)
	}
	return d
}

// siegeTestStart is a Wednesday noon: the first siege is on Sunday 6
// December 2026 at 16:00 (two weeks after the Sunday of that week).
var siegeTestStart = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

// newSiegeEnv starts a server with the castle; with prev, on prev's store
// (a restart).
func newSiegeEnv(t *testing.T, prev *testEnv, start time.Time) *testEnv {
	t.Helper()
	d := testGameData(t)
	w, err := data.NewWorld(siegeTestWorld())
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	dataDir := t.TempDir()
	writeDialogFile(t, filepath.Join(dataDir, "html", "admin", "castle.htm"), "<html><body>CASTLE %castleName%</body></html>")
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = dataDir
	store, ids := newRecordingStore(), memstore.NewIDs()
	if prev != nil {
		store, ids = prev.store, prev.ids
	}
	clock := world.NewFakeClock(start)
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w,
		Characters: store, Clans: store.Clans(), IDs: ids, Clock: clock, CheckLoopOwner: true,
		Passives: testPassives(t), Categories: testCategories(t),
		Castles: siegeTestCastles(t), CastleStore: store, TimeZone: time.UTC, Rand: rnd.New(7),
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
	e := &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
	e.advance(2 * time.Second) // the siege tasks start once the loop runs
	return e
}

// siegeLeader enters the world with a clan leader of a clan of level 4.
func (e *testEnv) siegeLeader(account, name, clanName string) (*fakeSession, *Player, *Clan) {
	e.t.Helper()
	sess, p := e.enterClanChar(clanChar{account: account, name: name, level: 40})
	c := e.foundClan(p, clanName)
	e.onLoop(func() { c.Level = 4 })
	return sess, p, c
}

// moveTo teleports p to a point near the castle center.
func (e *testEnv) moveTo(sess *fakeSession, p *Player, dx, dy int) {
	e.t.Helper()
	c := siegeTestCenter()
	e.onLoop(func() {
		if err := e.srv.Teleport(p, world.Position{X: int32(c.X + dx), Y: int32(c.Y + dy), Z: int32(c.Z)}); err != nil {
			e.t.Errorf("Teleport: %v", err)
		}
	})
	siegeAppear(e, sess)
}

// siegeAppear finishes a teleport (the client's Appearing).
func siegeAppear(e *testEnv, sess *fakeSession) {
	e.t.Helper()
	e.srv.Handle(sess, clientpackets.OpAppearing, nil)
	e.sync()
}

// gludio returns the castle.
func (e *testEnv) gludio() *Castle {
	var c *Castle
	e.onLoop(func() { c = e.srv.CastleByID(1) })
	return c
}

func joinSiege(e *testEnv, sess *fakeSession, attacker bool) []testclient.Packet {
	a := int32(0)
	if attacker {
		a = 1
	}
	return e.clanSend(sess, clientpackets.OpRequestJoinSiege, clanInts(1, a, 1))
}

func TestCastleSiegeFull(t *testing.T) {
	e := newSiegeEnv(t, nil, siegeTestStart)
	c := e.gludio()
	firstSiege := time.Date(2026, 12, 6, 16, 0, 0, 0, time.UTC)
	e.onLoop(func() {
		if !c.SiegeDate().Equal(firstSiege) || c.siege.registrationOver {
			t.Errorf("siege date %v (registration over %v), want %v and open", c.SiegeDate(), c.siege.registrationOver, firstSiege)
		}
	})

	sa, a, clanA := e.siegeLeader("alice", "Alice", "Lions")
	sb, b, clanB := e.siegeLeader("bob", "Bob", "Wolves")
	sl, _, clanL := e.siegeLeader("carol", "Carol", "Lambs")
	e.onLoop(func() { clanL.Level = 3 })

	// Registration: level 3 refused, no defenders for an NPC castle, two
	// attackers.
	clanWantMsg(t, "level 3", joinSiege(e, sl, true), residence.MsgClanLevelTooLow)
	if ps := joinSiege(e, sb, false); len(only[testclient.SiegeInfo](ps)) != 1 {
		t.Fatalf("a defender of an NPC castle gets %q, want the siege info", testclient.Names(ps))
	}
	joinSiege(e, sa, true)
	ps := joinSiege(e, sb, true)
	if info := only[testclient.SiegeInfo](ps); len(info) != 1 || info[0].SiegeDate != int32(firstSiege.Unix()) {
		t.Fatalf("siege info after joining: %+v", info)
	}
	clanWantMsg(t, "twice", joinSiege(e, sb, true), residence.MsgAlreadyRequested)
	ps = e.clanSend(sa, clientpackets.OpRequestSiegeAttackerList, clanInts(1))
	if l := only[testclient.SiegeAttackerList](ps); len(l) != 1 || len(l[0].Clans) != 2 {
		t.Fatalf("attacker list %+v", l)
	}

	// 24 hours before, registration closes.
	e.advance(firstSiege.Sub(siegeTestStart) - 24*time.Hour + time.Second)
	e.onLoop(func() {
		if !c.siege.registrationOver {
			t.Error("registration still open a day before the siege")
		}
	})
	e.onLoop(func() { clanL.Level = 4 })
	clanWantMsg(t, "after the deadline", joinSiege(e, sl, true), residence.MsgRegistrationDeadline)

	// Everyone stands in the castle; at the start they go to town.
	e.moveTo(sa, a, 0, 0)
	e.moveTo(sb, b, 100, 0)
	e.advance(24 * time.Hour)
	siegeAppear(e, sa)
	siegeAppear(e, sb)
	e.onLoop(func() {
		if !c.siege.inProgress {
			t.Error("the siege did not start")
			return
		}
		if got := len(c.siege.controlTowers) + len(c.siege.flameTowers); got != 4 {
			t.Errorf("%d towers, want 4", got)
		}
		if got := siegeGuardsUp(c); got != 2 {
			t.Errorf("%d guards, want 2", got)
		}
		if c.Contains(a.Position()) || c.Contains(b.Position()) {
			t.Errorf("attackers still in the castle zone")
		}
		if e.srv.siegeState(a) != 1 || e.srv.siegeState(b) != 1 {
			t.Errorf("siege states %d %d, want attackers", e.srv.siegeState(a), e.srv.siegeState(b))
		}
	})

	// Back on the battlefield: the PvP and siege flags, a headquarters in
	// the HQ zone, the relation bits.
	e.moveTo(sa, a, 2000, 0)
	e.onLoop(func() {
		if !a.Zones.Inside(ZoneSiege) || !a.Zones.Inside(ZonePvP) || !a.Zones.Inside(ZoneHQ) {
			t.Errorf("zones %v", a.Zones.IDs())
		}
		e.srv.buildHeadquarters(a)
		if sc := c.siege.reg.Attacker(clanA.ID); sc == nil || len(sc.Flags) != 1 {
			t.Errorf("no headquarters: %+v", sc)
		}
		if r := a.relationTo(b); r&serverpackets.RelationInSiege == 0 || r&serverpackets.RelationAttacker == 0 || r&serverpackets.RelationAlly == 0 {
			t.Errorf("relation %#x", r)
		}
	})

	// Alice dies there and restarts at her headquarters.
	e.onLoop(func() { e.srv.Kill(a, nil) })
	e.onLoop(func() {
		var d serverpackets.Die
		e.srv.residenceDieOptions(a, &d)
		if !d.ToSiegeHQ || d.ToCastle {
			t.Errorf("Die options %+v, want the headquarters", d)
		}
	})
	sa.take(t)
	e.clanSend(sa, clientpackets.OpRequestRestartPoint, clanInts(clientpackets.RestartPointSiegeHQ))
	siegeAppear(e, sa)
	e.onLoop(func() {
		if a.IsDead() || !c.Contains(a.Position()) {
			t.Errorf("Alice dead %v at %v, want alive at her headquarters", a.IsDead(), a.Position())
		}
	})

	// A tower falls: its guards no longer respawn.
	e.onLoop(func() {
		if len(c.siege.controlTowers) == 0 {
			t.Error("no control towers")
			return
		}
		e.srv.Kill(c.siege.controlTowers[0], a)
		if c.siege.ctCount != 2 {
			t.Errorf("control tower count %d, want 2", c.siege.ctCount)
		}
	})

	// Alice engraves the ruler: mid victory, the Lions defend now.
	e.onLoop(func() {
		e.srv.EngraveCastle(c, clanA)
		if c.OwnerID() != clanA.ID || clanA.CastleID != 1 || !c.siege.inProgress || !c.firstMidVictory {
			t.Errorf("after the engraving: owner %d, clan castle %d, in progress %v", c.OwnerID(), clanA.CastleID, c.siege.inProgress)
		}
		if !c.siege.reg.IsDefender(clanA.ID) || !c.siege.reg.IsAttacker(clanB.ID) {
			t.Errorf("sides after the mid victory: defenders %v", siegeClanIDs(c.siege.reg.Defenders()))
		}
		if e.srv.siegeState(a) != 2 {
			t.Errorf("Alice's siege state %d, want defender", e.srv.siegeState(a))
		}
	})

	// The siege ends after 120 minutes: the Lions keep Gludio and win 1,500
	// reputation; the next siege is two weeks later.
	e.advance(2*time.Hour + time.Second)
	nextSiege := time.Date(2026, 12, 20, 16, 0, 0, 0, time.UTC)
	e.onLoop(func() {
		if c.siege.inProgress {
			t.Error("the siege did not end")
		}
		if clanA.Reputation != 1500 {
			t.Errorf("Lions' reputation %d, want 1500", clanA.Reputation)
		}
		if !c.SiegeDate().Equal(nextSiege) {
			t.Errorf("next siege %v, want %v", c.SiegeDate(), nextSiege)
		}
		if siegeGuardsUp(c) != 0 || len(c.siege.controlTowers) != 0 {
			t.Errorf("guards or towers left")
		}
		if e.srv.siegeState(a) != 0 || a.Zones.Inside(ZoneSiege) {
			t.Errorf("Alice still in the siege")
		}
	})

	// The owner and the date survive a restart.
	e.flushClanWrites(clanA.ID)
	e.stop()
	e2 := newSiegeEnv(t, e, e.clock.Now())
	c2 := e2.gludio()
	e2.onLoop(func() {
		if c2.OwnerID() != clanA.ID {
			t.Errorf("owner after a restart %d, want %d", c2.OwnerID(), clanA.ID)
		}
		if !c2.SiegeDate().Equal(nextSiege) {
			t.Errorf("siege date after a restart %v, want %v", c2.SiegeDate(), nextSiege)
		}
	})
}

// siegeGuardsUp counts the castle's guards in the world. Loop-only.
func siegeGuardsUp(c *Castle) int {
	n := 0
	for _, g := range c.siege.guards {
		if g.npc != nil && g.npc.InWorld() {
			n++
		}
	}
	return n
}

func siegeClanIDs(list []*residence.SiegeClan) []int32 {
	var out []int32
	for _, c := range list {
		out = append(out, c.ClanID)
	}
	return out
}

func TestCastleSiegeCancelledWithoutAttackers(t *testing.T) {
	e := newSiegeEnv(t, nil, siegeTestStart)
	c := e.gludio()
	sess := newSession("watcher")
	e.createCharacter("watcher", "Watcher", 0, 0)
	e.enter(sess, 0)
	e.sync()
	sess.take(t)
	e.advance(time.Date(2026, 12, 6, 16, 0, 1, 0, time.UTC).Sub(siegeTestStart))
	clanWantMsg(t, "the cancelled siege", sess.take(t), residence.MsgSiegeCanceledNoInterest)
	e.onLoop(func() {
		if c.siege.inProgress {
			t.Error("a siege without attackers started")
		}
		if want := time.Date(2026, 12, 20, 16, 0, 0, 0, time.UTC); !c.SiegeDate().Equal(want) {
			t.Errorf("next siege %v, want %v", c.SiegeDate(), want)
		}
	})
}

func TestCastleSiegeGM(t *testing.T) {
	e := newSiegeEnv(t, nil, siegeTestStart)
	c := e.gludio()
	_, gm := e.enterClanChar(clanChar{account: "gm", name: "Gm", level: 40, access: storage.AccessLevelAdmin})
	sa, a, clanA := e.siegeLeader("alice", "Alice", "Lions")
	joinSiege(e, sa, true)

	gmRun := func(line string) { e.onLoop(func() { e.srv.RunGMCommand(gm, line) }) }
	gmRun("setsiegetime Gludio now 30")
	e.onLoop(func() {
		if want := e.srv.Now().Add(30 * time.Minute); !c.SiegeDate().Equal(want) {
			t.Errorf("siege date %v, want %v", c.SiegeDate(), want)
		}
	})
	gmRun("startsiege Gludio")
	e.onLoop(func() {
		if !c.siege.inProgress {
			t.Error("//startsiege did not start the siege")
		}
		gm.Target = a
	})
	gmRun("setcastle Gludio")
	e.onLoop(func() {
		// The only attacker took the castle: the siege ends at once.
		if c.OwnerID() != clanA.ID || c.siege.inProgress {
			t.Errorf("after //setcastle: owner %d, in progress %v", c.OwnerID(), c.siege.inProgress)
		}
	})
	gmRun("removecastle Gludio")
	e.onLoop(func() {
		if c.OwnerID() != 0 || clanA.CastleID != 0 {
			t.Errorf("after //removecastle: owner %d, clan castle %d", c.OwnerID(), clanA.CastleID)
		}
	})
}

func TestCastleTax(t *testing.T) {
	e := newSiegeEnv(t, nil, siegeTestStart)
	c := e.gludio()
	_, p, clan := e.siegeLeader("alice", "Alice", "Lions")
	_ = p
	art := npcSpawnedOne(t, e, siegeTestArtefact)
	e.onLoop(func() {
		e.srv.setCastleOwner(c, clan)
		c.SetTaxPercent(10)
		art.priceConfig = &data.PriceConfig{ID: 1, BaseTax: 5, CastleID: 1}
		if got := merchantTaxRate(art); got != 0.15 {
			t.Errorf("tax rate %v, want 0.15", got)
		}
		e.srv.merchantTreasury(art, 1150)
		if c.Treasury() != 115 {
			t.Errorf("treasury %d, want 115", c.Treasury())
		}
	})
}
