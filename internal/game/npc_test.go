package game

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The NPCs of npcTestWorld. The test characters enter at the first
// creation point of the Human Fighter; the NPCs stand around it.
const (
	npcTestGremlin    = 20001 // Monster, level 5: 5 in the territory, respawn 30 ± 10 s
	npcTestNightWolf  = 21000 // Monster: 2 by night only
	npcTestDaySparrow = 21001 // Monster: 1 by day only
	npcTestDoor       = 24000 // Door, not an NPC class: never spawned
	npcTestRaidBoss   = 25001 // RaidBoss: never spawned from the lists
	npcTestLector     = 30001 // Merchant, 1000 east of the start, random heading, respawn 60 s
	npcTestRoxxy      = 30006 // Teleporter, 100 east of the start, heading 16384, respawn 60 s
	npcTestFriendly   = 30100 // FriendlyMob, 300 south of the start
	npcTestCustom     = 30200 // Folk in a custom list

	npcTestTerritory = "test_field"
)

// npcTestTemplate returns a plain template. With STR 40, DEX 19, CON 43,
// WIT 20 and MEN 10 the bonuses make: max HP 100 × 1.58 = 158, max MP
// 50 × 1.11 = 55, attack speed 300 × 1.0, casting speed 333.
func npcTestTemplate(id int, typ, name string, level int) data.NpcTemplate {
	return data.NpcTemplate{
		ID: id, DisplayID: id, Level: level, Type: typ, Name: name, Race: data.RaceHuman, Sex: "MALE",
		Stats: data.NpcStats{
			STR: 40, INT: 21, DEX: 19, WIT: 20, CON: 43, MEN: 10,
			Vitals:  data.NpcVitals{HP: 100, MP: 50},
			Attack:  data.NpcAttack{Physical: 10, Magical: 5, Critical: 4, Speed: 300, Type: data.WeaponSword, Range: 40, Width: 120},
			Defence: data.NpcDefence{Physical: 40, Magical: 30},
			Speed:   data.NpcSpeed{Walk: 50, Run: 120, SwimWalk: 50, SwimRun: 120, FlyWalk: 50, FlyRun: 120},
		},
		Status:    data.NpcStatus{Attackable: true, Targetable: true, Talkable: true, ShowName: true, RandomWalk: true, CanMove: true},
		Collision: data.NpcCollision{Radius: 8, Height: 25},
		AI:        data.NpcAI{Type: "FIGHTER", MinSkillChance: 7, MaxSkillChance: 15},
	}
}

// npcTestSquare returns an NPoly square of side 2*half around (x, y).
func npcTestSquare(x, y, half, minZ, maxZ int) data.Shape {
	return data.Shape{Type: data.ShapeNPoly, MinZ: minZ, MaxZ: maxZ, Points: []data.Point{
		{X: x - half, Y: y - half}, {X: x + half, Y: y - half}, {X: x + half, Y: y + half}, {X: x - half, Y: y + half},
	}}
}

// npcTestWorld returns a small world around start (see the npcTest
// constants).
func npcTestWorld(start data.Location) data.WorldData {
	fixed := func(id, dx, dy, heading int) data.SpawnNpc {
		return data.SpawnNpc{NpcID: id, X: start.X + dx, Y: start.Y + dy, Z: start.Z, Heading: heading, Count: 1, RespawnDelay: 60}
	}
	roxxy := npcTestTemplate(npcTestRoxxy, "Teleporter", "Roxxy", 20)
	roxxy.Title = "Gatekeeper"
	roxxy.UsingServerSideName, roxxy.UsingServerSideTitle = true, true
	roxxy.Equipment = data.NpcEquipment{RHand: 2369, LHand: 103, WeaponEnchant: 3}
	wolf, sparrow := fixed(npcTestNightWolf, -500, -500, 0), fixed(npcTestDaySparrow, -600, -500, 0)
	wolf.Count, wolf.PeriodOfDay = 2, data.PeriodNight
	sparrow.PeriodOfDay = data.PeriodDay
	return data.WorldData{
		NpcTypes: []data.MobiusClass{
			{Name: "Attackable", Parent: "Npc"},
			{Name: "Creature", Parent: "WorldObject", Abstract: true},
			{Name: "Door", Parent: "Creature", Usable: true},
			{Name: "Folk", Parent: "Npc", Usable: true},
			{Name: "FriendlyMob", Parent: "Attackable", Usable: true},
			{Name: "Merchant", Parent: "Folk", Usable: true},
			{Name: "Monster", Parent: "Attackable", Usable: true},
			{Name: "Npc", Parent: "Creature"},
			{Name: "RaidBoss", Parent: "Monster", Usable: true},
			{Name: "Teleporter", Parent: "Merchant", Usable: true},
		},
		Npcs: []data.NpcTemplate{
			npcTestTemplate(npcTestGremlin, "Monster", "Gremlin", 5),
			npcTestTemplate(npcTestNightWolf, "Monster", "Night Wolf", 3),
			npcTestTemplate(npcTestDaySparrow, "Monster", "Day Sparrow", 2),
			npcTestTemplate(npcTestDoor, "Door", "Gate", 1),
			npcTestTemplate(npcTestRaidBoss, "RaidBoss", "Greyclaw Kutus", 23),
			npcTestTemplate(npcTestLector, "Merchant", "Lector", 20),
			roxxy,
			npcTestTemplate(npcTestFriendly, "FriendlyMob", "Friendly", 10),
			npcTestTemplate(npcTestCustom, "Folk", "Custom Folk", 10),
		},
		Territories: []data.SpawnTerritory{{
			Name: npcTestTerritory, File: "TalkingIsland/TestMonsters.xml",
			Shape: npcTestSquare(start.X+600, start.Y-600, 150, start.Z-100, start.Z+100),
		}},
		Spawns: []data.Spawn{{
			File: "TalkingIsland/TestMonsters.xml", Zone: npcTestTerritory,
			Npcs: []data.SpawnNpc{
				{NpcID: npcTestGremlin, Heading: -1, Count: 5, RespawnDelay: 30, RespawnRandom: 10},
				{NpcID: npcTestRaidBoss, Heading: -1, Count: 1, RespawnDelay: 3600},
				{NpcID: npcTestDoor, Heading: -1, Count: 1, RespawnDelay: 60},
			},
		}, {
			File: "TalkingIsland/TestNPCs.xml", Name: "TestNPCs",
			Npcs: []data.SpawnNpc{
				fixed(npcTestRoxxy, 100, 0, 16384),
				fixed(npcTestLector, 1000, 0, -1),
				fixed(npcTestFriendly, 0, -300, 0),
			},
		}, {
			File: "custom/TestCustom.xml",
			Npcs: []data.SpawnNpc{fixed(npcTestCustom, 0, -500, 0)},
		}, {
			File: "TalkingIsland/TestDayNight.xml",
			Npcs: []data.SpawnNpc{wolf, sparrow},
		}},
		ZoneTypes: []data.MobiusClass{
			{Name: "WaterZone", Parent: "ZoneType", Usable: true},
			{Name: "ZoneType", Abstract: true},
		},
		Zones: []data.Zone{{
			ID: 300001, Name: "far_water", Type: "WaterZone", File: "water.xml",
			Shape: data.Shape{Type: data.ShapeCuboid, MinZ: -4779, MaxZ: -3779, Points: []data.Point{{X: 100000, Y: 100000}, {X: 101000, Y: 101000}}},
		}},
		MapRegions: []data.MapRegion{{
			Name: data.DefaultRespawnRegion, Town: "Talking Island Village", LocID: 910, Castle: 1, BBS: 1, File: "talking_island_town.xml",
			RespawnPoints: []data.Location{start},
		}},
	}
}

// npcTestStart returns the creation point the test characters enter at.
func npcTestStart(t *testing.T) data.Location {
	t.Helper()
	tmpl, ok := testGameData(t).Template(classHumanFighter)
	if !ok || len(tmpl.CreationPoints) == 0 {
		t.Fatalf("no creation point of class %d", classHumanFighter)
	}
	return tmpl.CreationPoints[0]
}

// newNpcEnv starts a server like newEnv, with npcTestWorld as its world
// data, at testStart (06:00 game time: day).
func newNpcEnv(t *testing.T, configure func(gs *config.GameServer), setup ...func(s *Server)) *testEnv {
	t.Helper()
	return newNpcEnvAt(t, testStart, configure, setup...)
}

// newNpcEnvAt is newNpcEnv with the fake clock starting at start.
func newNpcEnvAt(t *testing.T, start time.Time, configure func(gs *config.GameServer), setup ...func(s *Server)) *testEnv {
	t.Helper()
	d := testGameData(t)
	w, err := data.NewWorld(npcTestWorld(npcTestStart(t)))
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	if configure != nil {
		configure(gs)
	}
	store := newRecordingStore()
	clock := world.NewFakeClock(start)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		World:          w,
		Characters:     store,
		IDs:            ids,
		Clock:          clock,
		CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, fn := range setup {
		fn(srv)
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

// npcTalk is a call of the interaction hooks.
type npcTalk struct {
	player string
	npc    int
}

// npcRecordTalks returns a setup function that replaces the interaction
// hooks (the dialog feature's) with one that records the calls in talks
// (read them on the loop), so that the NPC tests see their own packets
// only.
func npcRecordTalks(talks *[]npcTalk) func(s *Server) {
	return func(s *Server) {
		s.npc.interact = nil
		s.OnNpcInteract(func(p *Player, n *Npc) { *talks = append(*talks, npcTalk{p.Name(), n.Template.ID}) })
	}
}

// npcTalks returns a copy of the recorded talks, read on the loop.
func npcTalks(e *testEnv, talks *[]npcTalk) []npcTalk {
	var out []npcTalk
	e.onLoop(func() { out = append(out, *talks...) })
	return out
}

// npcsSpawned returns the NPCs of template id that are in the world, in
// slot order.
func npcsSpawned(e *testEnv, id int) []*Npc {
	var out []*Npc
	e.onLoop(func() {
		for _, sp := range e.srv.npc.spawns {
			if sp.tmpl.ID == id && sp.npc != nil && sp.npc.InWorld() {
				out = append(out, sp.npc)
			}
		}
	})
	return out
}

// npcSpawnedOne returns the one NPC of template id in the world.
func npcSpawnedOne(t *testing.T, e *testEnv, id int) *Npc {
	t.Helper()
	list := npcsSpawned(e, id)
	if len(list) != 1 {
		t.Fatalf("%d NPCs %d in the world, want 1", len(list), id)
	}
	return list[0]
}

// npcInWorld reports whether n is in the world, read on the loop.
func npcInWorld(e *testEnv, n *Npc) bool {
	var in bool
	e.onLoop(func() { in = n.InWorld() })
	return in
}

// npcPosition returns n's position, read on the loop.
func npcPosition(e *testEnv, n *Npc) world.Position {
	var pos world.Position
	e.onLoop(func() { pos = n.Position() })
	return pos
}

// npcActionBody encodes an Action or AttackRequest body: object ID, the
// client's position and the Shift flag.
func npcActionBody(objectID int32, shift bool) []byte {
	w := packets.NewWriter(17)
	w.WriteInt32(objectID)
	w.WriteInt32(0)
	w.WriteInt32(0)
	w.WriteInt32(0)
	if shift {
		w.WriteUint8(1)
	} else {
		w.WriteUint8(0)
	}
	return w.Bytes()
}

// npcTestClick sends a click on o from sess and waits for it.
func npcTestClick(e *testEnv, sess *fakeSession, o world.Object, shift bool) {
	e.srv.Handle(sess, clientpackets.OpAction, npcActionBody(int32(o.ID()), shift))
	e.sync()
}

func TestNpcStatsTemplate(t *testing.T) {
	tmpl := npcTestTemplate(npcTestGremlin, "Monster", "Gremlin", 5)
	tmpl.Stats.Speed.SwimRun, tmpl.Stats.Speed.SwimWalk = 70, 30
	got := npcStatsTemplate(&tmpl)
	want := stats.NpcTemplate{
		Level:         5,
		Base:          stats.BaseStats{STR: 40, CON: 43, DEX: 19, INT: 21, WIT: 20, MEN: 10},
		HP:            100,
		MP:            50,
		PAtk:          10,
		MAtk:          5,
		PDef:          40,
		MDef:          30,
		CritRate:      4,
		PAtkSpd:       300,
		AttackRange:   40,
		RunSpeed:      120,
		WalkSpeed:     50,
		SwimRunSpeed:  70,
		SwimWalkSpeed: 30,
		Collision:     stats.Collision{Radius: 8, Height: 25},
	}
	if *got != want {
		t.Errorf("npcStatsTemplate =\n%+v\nwant\n%+v", *got, want)
	}
}

// TestNpcInfoGolden checks NpcInfo of Roxxy byte by byte (Mobius
// AbstractNpcInfo.NpcInfo.writeImpl).
func TestNpcInfoGolden(t *testing.T) {
	e := newNpcEnv(t, nil)
	var got [][]byte
	e.onLoop(func() {
		tmpl, _ := e.srv.WorldData().Npc(npcTestRoxxy)
		n, err := e.srv.newNpc(tmpl)
		if err != nil {
			t.Fatalf("newNpc: %v", err)
		}
		if err := n.SetID(0x60001234); err != nil {
			t.Fatalf("SetID: %v", err)
		}
		if err := n.SetPosition(world.Position{X: -71238, Y: 258271, Z: -3104, Heading: 16384}); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		got = n.InfoFor(&Player{})
	})

	w := packets.NewWriter(256)
	w.WriteUint8(0x16)
	w.WriteInt32(0x60001234)
	w.WriteInt32(1030006) // display ID + 1,000,000
	w.WriteInt32(0)       // a Teleporter is not attackable
	w.WriteInt32(-71238)
	w.WriteInt32(258271)
	w.WriteInt32(-3104)
	w.WriteInt32(16384)
	w.WriteInt32(0)
	w.WriteInt32(333) // casting speed: the default
	w.WriteInt32(300) // attack speed: 300 × the DEX 19 bonus 1.0
	w.WriteInt32(120) // run
	w.WriteInt32(50)  // walk
	w.WriteInt32(120) // swim run
	w.WriteInt32(50)  // swim walk
	for range 4 {
		w.WriteInt32(0) // fly speeds: not flying
	}
	w.WriteUint64(0x3FF0000000000000) // move multiplier 50 × (1/50) = 1.0
	w.WriteUint64(0x3FF19999A0000000) // attack speed multiplier (float) 1.1
	w.WriteUint64(0x4020000000000000) // collision radius 8
	w.WriteUint64(0x4039000000000000) // collision height 25
	w.WriteInt32(2369)                // right hand
	w.WriteInt32(0)                   // chest: never set for NPCs
	w.WriteInt32(103)                 // left hand
	w.WriteUint8(1)                   // show the name
	w.WriteUint8(0)                   // walking
	w.WriteUint8(0)                   // not in combat
	w.WriteUint8(0)                   // alive
	w.WriteUint8(0)                   // no summon animation
	w.WriteString("Roxxy")            // usingServerSideName
	w.WriteString("Gatekeeper")       // usingServerSideTitle
	for range 8 {
		w.WriteInt32(0) // title colour, PvP flag, karma, abnormal effects, clan, crest, ally, ally crest
	}
	w.WriteUint8(0)                   // on land
	w.WriteUint8(0)                   // no team
	w.WriteUint64(0x4020000000000000) // collision radius again
	w.WriteUint64(0x4039000000000000) // collision height again
	w.WriteInt32(3)                   // the template's weapon enchant
	w.WriteInt32(0)                   // not flying
	if len(got) != 1 || !bytes.Equal(got[0], w.Bytes()) {
		t.Errorf("InfoFor =\n% x\nwant\n% x", got, w.Bytes())
	}
}

func TestNpcInfoNamesAndAttackable(t *testing.T) {
	e := newNpcEnv(t, nil)
	var gremlin, friendly testclient.NpcInfo
	var friendlyToKarma bool
	e.onLoop(func() {
		decode := func(b []byte) testclient.NpcInfo {
			p, err := testclient.DecodeServerPacket(b)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			return p.(testclient.NpcInfo)
		}
		g, _ := e.srv.WorldData().Npc(npcTestGremlin)
		n, err := e.srv.newNpc(g)
		if err != nil {
			t.Fatalf("newNpc: %v", err)
		}
		n.Title = "Own title"
		gremlin = decode(n.InfoFor(&Player{})[0])

		f, _ := e.srv.WorldData().Npc(npcTestFriendly)
		fn, err := e.srv.newNpc(f)
		if err != nil {
			t.Fatalf("newNpc: %v", err)
		}
		friendly = decode(fn.InfoFor(&Player{})[0])
		pk := &Player{}
		pk.char.Karma = 10
		friendlyToKarma = fn.autoAttackableBy(pk)
	})
	// Without server-side names the client shows its own name, and the
	// NPC's own title.
	if gremlin.Name != "" || gremlin.Title != "Own title" || !gremlin.Attackable || gremlin.NpcID() != npcTestGremlin {
		t.Errorf("Gremlin's NpcInfo: name %q title %q attackable %v id %d", gremlin.Name, gremlin.Title, gremlin.Attackable, gremlin.NpcID())
	}
	if friendly.Attackable || !friendlyToKarma {
		t.Errorf("FriendlyMob: attackable %v without karma, %v with karma; want false, true", friendly.Attackable, friendlyToKarma)
	}
}

// TestServerObjectInfoAtRunSpeedZero checks the packet Mobius sends for an
// NPC whose run speed is 0.
func TestServerObjectInfoAtRunSpeedZero(t *testing.T) {
	e := newNpcEnv(t, nil)
	var got [][]byte
	e.onLoop(func() {
		tmpl, _ := e.srv.WorldData().Npc(npcTestRoxxy)
		n, err := e.srv.newNpc(tmpl)
		if err != nil {
			t.Fatalf("newNpc: %v", err)
		}
		if err := n.SetID(0x60000042); err != nil {
			t.Fatalf("SetID: %v", err)
		}
		if err := n.SetPosition(world.Position{X: 10, Y: 20, Z: -30, Heading: 40}); err != nil {
			t.Fatalf("SetPosition: %v", err)
		}
		stop := stats.NewModifier(stats.RunSpd, stats.OpMul, 0, stats.Owner{Kind: stats.OwnerEffect, ID: 1})
		if err := n.Stats.AddModifiers(stop); err != nil {
			t.Fatalf("AddModifiers: %v", err)
		}
		got = n.InfoFor(&Player{})
	})
	w := packets.NewWriter(96)
	w.WriteUint8(0x8c)
	w.WriteInt32(0x60000042)
	w.WriteInt32(1030006)
	w.WriteString("Roxxy")
	w.WriteInt32(0) // not attackable
	w.WriteInt32(10)
	w.WriteInt32(20)
	w.WriteInt32(-30)
	w.WriteInt32(40)
	w.WriteUint64(0x3FF0000000000000) // 1.0
	w.WriteUint64(0x3FF0000000000000) // 1.0
	w.WriteUint64(0x4020000000000000) // collision radius 8
	w.WriteUint64(0x4039000000000000) // collision height 25
	w.WriteInt32(0)                   // HP: only for attackable objects
	w.WriteInt32(0)
	w.WriteInt32(1) // an NPC
	w.WriteInt32(0)
	if len(got) != 1 || !bytes.Equal(got[0], w.Bytes()) {
		t.Errorf("InfoFor =\n% x\nwant\n% x", got, w.Bytes())
	}
}

func TestNpcHPBarCheck(t *testing.T) {
	var c npcHPBarCheck
	c.reset(3520) // a pixel is 10 HP
	for _, step := range []struct {
		hp   float64
		want bool
	}{
		{3515, false}, // within the first pixel
		{3505, true},  // crossed it
		{3501, false},
		{3500, true},
		{3520, true}, // full again
		{3519, false},
		{1, true}, // almost dead: always
	} {
		if got := c.need(step.hp, 3520); got != step.want {
			t.Errorf("need(%v) = %v, want %v", step.hp, got, step.want)
		}
	}
	var small npcHPBarCheck
	small.reset(158)
	if !small.need(157.5, 158) {
		t.Error("a bar shorter than its pixels is refreshed on every change")
	}
}

func TestNpcHPBarFollowsTheTarget(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	npcTestClick(e, sa, gremlin, false)
	sa.take(t)
	sb.take(t)

	e.onLoop(func() { gremlin.SetCurrentHP(100.7) })
	ups := only[testclient.StatusUpdate](sa.take(t))
	if len(ups) != 1 || ups[0].ObjectID != int32(gremlin.ID()) {
		t.Fatalf("Alice got %+v, want the Gremlin's HP bar", ups)
	}
	if hp, _ := ups[0].Value(testclient.StatusCurHP); hp != 100 {
		t.Errorf("current HP %d, want 100", hp)
	}
	if maxHP, _ := ups[0].Value(testclient.StatusMaxHP); maxHP != 158 {
		t.Errorf("max HP %d, want 158", maxHP)
	}
	if ps := sb.take(t); len(ps) != 0 {
		t.Errorf("Bob, who does not target it, got %q", testclient.Names(ps))
	}

	// After dropping the target the bar is no longer sent.
	e.onLoop(func() { e.srv.SetTarget(pa, nil) })
	sa.take(t)
	e.onLoop(func() { gremlin.SetCurrentHP(50) })
	if ups := only[testclient.StatusUpdate](sa.take(t)); len(ups) != 0 {
		t.Errorf("got %+v after dropping the target", ups)
	}
}

func TestNpcScriptRefDisplayID(t *testing.T) {
	e := newNpcEnv(t, nil)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	var id, display int
	var objectID int32
	e.onLoop(func() {
		ref := roxxy.ScriptRef()
		id, objectID = ref.ID(), ref.ObjectID()
		display = ref.(npcDisplayID).DisplayID()
	})
	if id != npcTestRoxxy || display != npcTestRoxxy || objectID != int32(roxxy.ID()) {
		t.Errorf("ScriptRef: id %d, display %d, object %d", id, display, objectID)
	}
}
