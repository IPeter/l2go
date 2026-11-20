package game

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The AI scripts' world on the game (Plan 13, Phase 10.6): NPC control,
// minions, chests, world scans, zones, grand bosses, cinematics and the
// movement events, through ScriptOutput as the scripting engine calls it.

// The script switches: walking, the core AI, immobility, invulnerability
// (damage over time still hurts), mortality, and the random walking that a
// respawn resets.
func TestScriptAINpcSwitches(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 200, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	var running, coreOff, immobile, moved, invul, mortal bool
	var hpAfterHit, hpAfterDot, maxHP float64
	e.onLoop(func() {
		e.srv.SetNpcRunning(k, true)
		o.NpcSetWalking(ref)
		running = o.NpcIsRunning(ref)
		o.NpcDisableCoreAI(ref, true)
		coreOff = o.NpcIsCoreAIDisabled(ref) && k.NpcCoreAIDisabled()
		o.NpcSetImmobilized(ref, true)
		immobile = npcMovementDisabled(k)
		o.NpcMoveTo(ref, scripting.Location{X: start.X + 600, Y: start.Y, Z: start.Z})
		moved = k.Move != nil
		o.NpcSetImmobilized(ref, false)

		maxHP = k.CurHP
		o.NpcSetInvul(ref, true)
		invul = o.NpcIsInvul(ref)
		k.ReduceHP(10, p)
		hpAfterHit = k.CurHP
		k.reduceHP(10, p, true)
		hpAfterDot = k.CurHP
		o.NpcSetInvul(ref, false)
	})
	if running || !coreOff || !immobile || moved {
		t.Errorf("running %v, core AI off %v, immobilized %v, walked %v; want false, true, true, false", running, coreOff, immobile, moved)
	}
	if !invul || hpAfterHit != maxHP || hpAfterDot != maxHP-10 {
		t.Errorf("invul %v: HP %v after a hit, %v after a DOT tick; want %v and %v", invul, hpAfterHit, hpAfterDot, maxHP, maxHP-10)
	}

	var dead bool
	var hp float64
	e.onLoop(func() {
		o.NpcSetMortal(ref, false)
		mortal = o.NpcIsMortal(ref)
		k.ReduceHP(k.CurHP+50, p)
		dead, hp = k.Dead, k.CurHP
	})
	if mortal || dead || hp != 0 {
		t.Errorf("immortal: mortal %v, dead %v at %v HP; want an NPC alive at 0 HP", mortal, dead, hp)
	}

	var walks, walksAfter, back bool
	e.onLoop(func() {
		o.NpcSetMortal(ref, true)
		o.NpcSetRandomWalking(ref, true)
		walks = e.srv.npcRandomWalks(k)
		o.NpcSetRespawn(ref, true)
		e.srv.DespawnNpc(k)
	})
	e.advance(time.Second)
	e.onLoop(func() { back, walksAfter = k.InWorld(), e.srv.npcRandomWalks(k) })
	if !walks || !back || walksAfter {
		t.Errorf("random walking %v after set_random_walking(true), %v after a respawn (back %v); want true, then the template's false", walks, walksAfter, back)
	}
}

// Intentions by their Mobius names, the hate list most hated first, stop
// and reduce hate (which calms the NPC when nobody is hated any more), and
// the heading.
func TestScriptAIIntentionsAndHate(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 200, 0)
	w := aiSpawn(e, aiWolf, start, 300, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	var fresh, attack, moveTo, active string
	var moving, stopped bool
	e.onLoop(func() {
		k.ai.attached = false // as before anything asked for the AI
		fresh = o.NpcIntention(ref)
		o.NpcAddDamageHate(ref, p, 0, 50) // without hate the attack think turns ACTIVE
		o.NpcSetIntention(ref, "ATTACK", p)
		attack = o.NpcIntention(ref)
		loc := scripting.Location{X: start.X + 700, Y: start.Y, Z: start.Z}
		o.NpcSetIntention(ref, "MOVE_TO", &loc)
		moveTo, moving = o.NpcIntention(ref), o.NpcIsMoving(ref)
		o.NpcStopMoving(ref)
		stopped = !o.NpcIsMoving(ref)
		o.NpcSetIntention(ref, "ACTIVE", nil)
		active = o.NpcIntention(ref)
	})
	if fresh != "IDLE" || attack != "ATTACK" || moveTo != "MOVE_TO" || !moving || !stopped || active != "ACTIVE" {
		t.Errorf("intentions %q, %q, %q (moving %v, stopped %v), %q; want IDLE, ATTACK, MOVE_TO, true, true, ACTIVE",
			fresh, attack, moveTo, moving, stopped, active)
	}

	var list []scripting.HateEntry
	var most any
	e.onLoop(func() {
		k.ClearAggroList()
		o.NpcAddDamageHate(ref, w.ScriptRef(), 5, 10)
		o.NpcAddDamageHate(ref, p, 7, 100)
		list = o.NpcHateList(ref)
		most = o.NpcMostHated(ref)
	})
	if len(list) != 2 || list[0].Target != any(p) || list[0].Hate != 100 || list[0].Damage != 7 {
		t.Fatalf("hate list %+v; want Alice (100 hate, 7 damage) first, then the wolf", list)
	}
	if r, ok := list[1].Target.(scripting.NpcRef); !ok || r.ObjectID() != int32(w.ID()) || most != any(p) {
		t.Errorf("second %+v, most hated %v; want the wolf, then Alice most hated", list[1], most)
	}

	var afterStop, afterReduce []scripting.HateEntry
	var calmed int
	var intention string
	var running bool
	var hated int
	e.onLoop(func() {
		o.NpcStopHating(ref, w.ScriptRef())
		afterStop = o.NpcHateList(ref)
		o.NpcReduceHate(ref, p, 30)
		afterReduce = o.NpcHateList(ref)
		e.srv.SetNpcRunning(k, true)
		o.NpcReduceHate(ref, p, 70)
		calmed, intention, running = k.ai.globalAggro, o.NpcIntention(ref), k.Running()
		hated = len(k.Aggro())
	})
	if len(afterStop) != 2 || afterStop[1].Hate != 0 {
		t.Errorf("after stop_hating the wolf: %+v; want it kept with 0 hate", afterStop)
	}
	if len(afterReduce) != 2 || afterReduce[0].Hate != 70 {
		t.Errorf("after reduce_hate 30: %+v; want Alice at 70", afterReduce)
	}
	// An NPC that turns ACTIVE thinks at once, which counts the aggression
	// up by one; this one was ACTIVE already.
	if calmed != npcCalmAggro || intention != "ACTIVE" || running || hated != 0 {
		t.Errorf("calmed: global aggro %d, %s, running %v, %d hated; want -25, ACTIVE, walking, none", calmed, intention, running, hated)
	}

	var heading int32
	e.onLoop(func() {
		o.NpcSetHeading(ref, 12345)
		heading = k.Position().Heading
		o.NpcAddDamageHate(ref, p, 0, 5)
		o.NpcClearAggroList(ref)
		hated = len(k.Aggro())
	})
	if heading != 12345 || hated != 0 {
		t.Errorf("heading %d, %d hated after clear_aggro_list; want 12345, none", heading, hated)
	}
}

// follow walks the NPC up to its target every second, and gives up on a
// target 3,000 or more away.
func TestScriptAIFollow(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 600, 0)
	far := aiSpawn(e, aiWolf, start, 600+3500, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	e.onLoop(func() { o.NpcFollow(ref, p) })
	e.advance(time.Second)
	st := aiStateOf(e, k)
	if !st.moving || st.dest.X >= st.pos.X {
		t.Fatalf("after a second of follow: %+v; want walking west towards Alice", st)
	}

	var task world.TimerID
	e.onLoop(func() {
		o.NpcStopFollow(ref)
		task = k.script.follow
		o.NpcFollow(ref, far.ScriptRef())
	})
	if task != 0 {
		t.Errorf("stop_follow left the task %d", task)
	}
	e.advance(time.Second)
	e.onLoop(func() { task = k.script.follow })
	if task != 0 {
		t.Errorf("the follow of a target 3,500 away still runs (%d)", task)
	}
}

// Leaders and minions: add_minion spawns a minion next to its leader.
func TestScriptAIMinions(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	k := aiSpawn(e, aiKeltir, start, 200, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	var minion, leader scripting.NpcRef
	var minions []scripting.NpcRef
	var none scripting.NpcRef
	e.onLoop(func() {
		minion = o.NpcAddMinion(ref, aiWolf)
		minions = o.NpcMinionsOf(ref)
		if minion != nil {
			leader = o.NpcLeader(minion)
		}
		none = o.NpcLeader(ref)
	})
	if minion == nil || minion.ID() != aiWolf || len(minions) != 1 || minions[0].ObjectID() != minion.ObjectID() {
		t.Fatalf("add_minion = %v, minions %v; want one wolf", minion, minions)
	}
	if leader == nil || leader.ObjectID() != int32(k.ID()) || none != nil {
		t.Errorf("leader of the minion %v, of the master %v; want the Keltir, none", leader, none)
	}
}

// World scans by range and template, the spawn's AI data and point, the
// template parameters, geodata and the game clock.
func TestScriptAIWorldScan(t *testing.T) {
	start := npcTestStart(t)
	block := data.Spawn{
		File: "TalkingIsland/AIScan.xml", Name: "ai_scan_block", AIData: map[string]int{"disableRandomAnimation": 1},
		Npcs: []data.SpawnNpc{{NpcID: aiOrc, X: start.X + 5000, Y: start.Y, Z: start.Z, Heading: 100, Count: 1, RespawnDelay: 60}},
	}
	wd := aiTestWorld(start, block)
	for i := range wd.Npcs {
		if wd.Npcs[i].ID == aiKeltir {
			wd.Npcs[i].Parameters.Params = map[string]string{"Skill01_ID": "4001"}
		}
	}
	e := newAIEnv(t, wd, newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 100, 0)
	near := aiSpawn(e, aiOrc, start, 300, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	var around, all []scripting.NpcRef
	var playersNear, playersFar []scripting.PlayerRef
	var orcs []scripting.NpcRef
	e.onLoop(func() {
		around = o.NpcsAround(ref, 250)
		all = o.NpcsAround(ref, 0)
		playersNear, playersFar = o.PlayersAround(ref, 150), o.PlayersAround(ref, 50)
		orcs = o.NpcsByID(aiOrc)
	})
	if len(around) != 1 || around[0].ObjectID() != int32(near.ID()) {
		t.Errorf("NPCs within 250: %v; want the orc 200 away", around)
	}
	if !slices.ContainsFunc(all, func(r scripting.NpcRef) bool { return r.ObjectID() == int32(near.ID()) }) ||
		slices.ContainsFunc(all, func(r scripting.NpcRef) bool { return r.ObjectID() == int32(k.ID()) }) {
		t.Errorf("NPCs around: %v; want the orc, not the Keltir itself", all)
	}
	if len(playersNear) != 1 || playersNear[0] != scripting.PlayerRef(p) || len(playersFar) != 0 {
		t.Errorf("players within 150: %v, within 50: %v; want Alice, none", playersNear, playersFar)
	}
	if len(orcs) != 2 || orcs[0].ObjectID() >= orcs[1].ObjectID() {
		t.Fatalf("NpcsByID(orc) = %v; want the two orcs by object ID", orcs)
	}

	var blockOrc scripting.NpcRef
	for _, r := range orcs {
		if r.ObjectID() != int32(near.ID()) {
			blockOrc = r
		}
	}
	si, tp := blockOrc.(scripting.NpcSpawnInfo), ref.(scripting.NpcTemplateParams)
	loc, hasLoc := si.SpawnLocation()
	v, hasV := si.AIValue("disableRandomAnimation")
	_, noV := ref.(scripting.NpcSpawnInfo).AIValue("disableRandomAnimation")
	kLoc, kHasLoc := ref.(scripting.NpcSpawnInfo).SpawnLocation()
	param, hasParam := tp.TemplateParam("Skill01_ID")
	_, noParam := tp.TemplateParam("Skill02_ID")
	if !hasLoc || loc.X != start.X+5000 || !hasV || v != 1 || noV {
		t.Errorf("spawn block orc: location %v (%v), AI value %d (%v); Keltir AI value %v", loc, hasLoc, v, hasV, noV)
	}
	if !kHasLoc || kLoc.X != start.X+100 {
		t.Errorf("Keltir spawn location %v (%v); want where SpawnNpc put it", kLoc, kHasLoc)
	}
	if !hasParam || param != "4001" || noParam {
		t.Errorf("template parameter %q (%v), missing one found %v", param, hasParam, noParam)
	}

	var valid scripting.Location
	var ticks, wantTicks int64
	var minutes, wantMinutes int
	var night, wantNight bool
	e.onLoop(func() {
		to := scripting.Location{X: start.X + 900, Y: start.Y + 40, Z: start.Z}
		valid = o.ValidLocation(scripting.Location{X: start.X, Y: start.Y, Z: start.Z}, to)
		gt := e.srv.world.GameTime()
		ticks, minutes, night = o.GameTicks(), o.GameTime(), o.IsNight()
		wantTicks, wantMinutes, wantNight = gt.Ticks(), int(gt.Minutes()), gt.IsNight()
	})
	if valid.X != start.X+900 || valid.Y != start.Y+40 {
		t.Errorf("ValidLocation without geodata = %v; want the destination", valid)
	}
	if ticks != wantTicks || minutes != wantMinutes || night != wantNight {
		t.Errorf("game clock %d ticks, %d min, night %v; want %d, %d, %v", ticks, minutes, night, wantTicks, wantMinutes, wantNight)
	}
}

// Zones by shape, the grand boss store and the live boss.
func TestScriptAIZonesAndGrandBoss(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	k := aiSpawn(e, aiKeltir, start, 100, 0)
	var wet *Npc
	e.onLoop(func() {
		var err error
		wet, err = e.srv.SpawnNpc(aiWolf, world.Position{X: 100500, Y: 100500, Z: -4279})
		if err != nil {
			t.Errorf("SpawnNpc: %v", err)
		}
	})
	o := scriptOutputOf(e)
	ref := k.ScriptRef()

	var in, out, unknown bool
	var inside []scripting.PlayerRef
	e.onLoop(func() {
		in, out, unknown = o.ZoneIsInside(300001, wet.ScriptRef()), o.ZoneIsInside(300001, ref), o.ZoneIsInside(1, wet.ScriptRef())
		inside = o.ZonePlayers(300001)
	})
	if !in || out || unknown || len(inside) != 0 {
		t.Errorf("inside the water %v, Keltir %v, unknown zone %v, players %v; want true, false, false, none", in, out, unknown, inside)
	}

	var live scripting.NpcRef
	var status, unknownStatus bool
	var st int
	var info scripting.BossInfo
	var hasInfo bool
	respawn := testStart.Add(36 * time.Hour).UnixMilli()
	e.onLoop(func() {
		_, unknownStatus = o.BossStatus(99999)
		o.AddBoss(ref)
		live = o.LiveBoss(aiKeltir)
		o.SetBossStatus(aiKeltir, scripting.BossFighting)
		st, status = o.BossStatus(aiKeltir)
		o.SetBossInfo(aiKeltir, scripting.BossInfo{RespawnTime: respawn, X: 1, Y: 2, Z: 3, Heading: 4, HP: 5, MP: 6})
		info, hasInfo = o.BossInfo(aiKeltir)
	})
	if unknownStatus || live == nil || live.ObjectID() != int32(k.ID()) {
		t.Errorf("unknown boss known %v, live boss %v; want false, the Keltir", unknownStatus, live)
	}
	want := scripting.BossInfo{RespawnTime: respawn, X: 1, Y: 2, Z: 3, Heading: 4, HP: 5, MP: 6}
	if !status || st != scripting.BossFighting || !hasInfo || info != want {
		t.Errorf("status %d (%v), info %+v (%v); want 2, %+v", st, status, info, hasInfo, want)
	}

	e.onLoop(func() {
		e.srv.DespawnNpc(k)
		live = o.LiveBoss(aiKeltir)
	})
	if live != nil {
		t.Errorf("live boss %v after it left the world; want none", live)
	}
}

// The cinematics reach one player, the players around an object (a player
// itself included), and stay off a zone nobody is in.
func TestScriptAICinematics(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 100, 0)
	o := scriptOutputOf(e)
	ref := k.ScriptRef()
	kid := int32(k.ID())
	sess.take(t)

	cam := scripting.Camera{Force: 700, Angle1: 13, Angle2: -19, Time: 0, Range: 10000, Duration: 20000}
	e.onLoop(func() {
		o.SpecialCamera(scripting.Audience{Player: p}, ref, cam)
		o.Earthquake(scripting.Audience{Around: ref}, scripting.Location{X: 1, Y: 2, Z: 3}, 20, 10)
		o.PlaySoundFrom(scripting.Audience{Around: p}, "BS01_A", ref)
		o.PlaySoundFrom(scripting.Audience{Player: p}, "BS02_A", nil)
		o.ScreenMessage(scripting.Audience{Player: p}, "Hello", 5, 3000)
		o.ScreenMessage(scripting.Audience{Zone: 300001}, "nobody", 2, 1000)
	})
	ps := sess.take(t)

	var raw [][]byte
	for _, pk := range ps {
		if u, ok := pk.(testclient.Unknown); ok && (u.Opcode == serverpackets.OpSpecialCamera || u.Opcode == serverpackets.OpEarthquake) {
			raw = append(raw, append([]byte{u.Opcode}, u.Body...))
		}
	}
	wantCam := serverpackets.NewSpecialCameraPacket(serverpackets.SpecialCamera{ObjectID: kid, Force: 700, Angle1: 13, Angle2: -19, Duration: 20000})
	wantQuake := serverpackets.NewEarthquakePacket(serverpackets.Earthquake{X: 1, Y: 2, Z: 3, Intensity: 20, Duration: 10})
	if len(raw) != 2 || !bytes.Equal(raw[0], wantCam) || !bytes.Equal(raw[1], wantQuake) {
		t.Errorf("camera and earthquake packets % x; want % x and % x", raw, wantCam, wantQuake)
	}
	sounds := only[testclient.PlaySound](ps)
	var kpos world.Position
	e.onLoop(func() { kpos = k.Position() })
	wantBound := testclient.PlaySound{Type: 1, Sound: "BS01_A", Bound: 1, ObjectID: kid, X: kpos.X, Y: kpos.Y, Z: kpos.Z}
	if len(sounds) != 2 || sounds[0] != wantBound || sounds[1] != (testclient.PlaySound{Sound: "BS02_A"}) {
		t.Errorf("sounds %+v; want %+v and a plain BS02_A", sounds, wantBound)
	}
	msgs := only[testclient.ExShowScreenMessage](ps)
	if len(msgs) != 1 || msgs[0].Text != "Hello" || msgs[0].Position != 5 || msgs[0].Time != 3000 || msgs[0].Type != 1 {
		t.Errorf("screen messages %+v; want one Hello at 5 for 3000 ms", msgs)
	}
}

// ----- scenarios with real scripts on the real game -----

// newScriptAIEnv starts a server with the world data of the dialog tests,
// the skill data and the given scripts (path relative to the scripts
// folder → source). NPCs do not swing.
func newScriptAIEnv(t *testing.T, scripts map[string]string) *testEnv {
	t.Helper()
	d := testGameData(t)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = t.TempDir()
	out := NewScriptOutput()
	dir := scripttest.WriteScripts(t, scripts)
	gs.Scripts.Dir = dir
	engine, err := scripting.New(t.Context(), scripting.Options{
		Dir: dir, Output: out, World: out,
		Constants: ScriptConstants(nil, 0, QuestSettings{}),
	})
	if err != nil {
		t.Fatalf("load the scripts: %v", err)
	}
	t.Cleanup(engine.Close)
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: testWorldData(t),
		Characters: store, IDs: ids, Scripts: engine, ScriptOutput: out,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		Clock: clock, CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.ai.doAttack = func(*Npc, Creature) {}
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
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// readRepoScript reads a script of the repository's scripts folder.
func readRepoScript(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "scripts", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

const (
	chestBox   = 18265 // Treasure Chest, a box (level 21)
	chestMimic = 21801 // Treasure Chest, a mimic
)

// The real Chests script on the real game: a box hit by a player blows up
// once (Treasure Bomb 4143 at level 2, cast on itself), a second hit does
// nothing more; a mimic does not blow up.
func TestScriptAIChestsScenario(t *testing.T) {
	e := newScriptAIEnv(t, map[string]string{
		"ai/monsters/chests.lua": readRepoScript(t, "ai/monsters/chests.lua"),
	})
	sess, p := e.enterChar(testChar{account: "alice", name: "Alice", pos: tiStart})
	box := e.spawnNpc(chestBox, eastOf(tiStart, 80))
	mimic := e.spawnNpc(chestMimic, eastOf(tiStart, -80))
	e.sync() // the scripts start
	sess.take(t)

	hit := func(n *Npc) {
		e.onLoop(func() {
			for _, fn := range e.srv.attack.hit {
				fn(n, p, 5)
			}
			n.ReduceHP(5, p)
		})
		e.sync() // the attack handlers run after the hit
	}
	var isBox, isMimicBox, walks, boxStill bool
	e.onLoop(func() {
		isBox, isMimicBox = box.script.box, mimic.script.box
		walks = e.srv.npcRandomWalks(box) || e.srv.npcRandomWalks(mimic)
		boxStill = npcMovementDisabled(box) && !npcMovementDisabled(mimic)
	})
	if !isBox || isMimicBox || walks || !boxStill {
		t.Fatalf("box %v, mimic a box %v, chests walk %v, only the box cannot move %v", isBox, isMimicBox, walks, boxStill)
	}

	hit(box)
	var interacted bool
	e.onLoop(func() { interacted = box.script.interacted })
	bombs := 0
	for _, m := range only[testclient.MagicSkillUse](sess.take(t)) {
		if m.SkillID == treasureBombSkill && m.CasterID == int32(box.ID()) && m.TargetID == int32(box.ID()) && m.Level == 2 {
			bombs++
		}
	}
	if !interacted || bombs != 1 {
		t.Fatalf("after the first hit: interacted %v, %d Treasure Bombs (level 2, on itself); want true, 1", interacted, bombs)
	}

	e.advance(5 * time.Second) // the bomb lands
	hit(box)
	hit(mimic)
	var mimicInteracted bool
	e.onLoop(func() { mimicInteracted = mimic.script.interacted })
	for _, m := range only[testclient.MagicSkillUse](sess.take(t)) {
		if m.SkillID == treasureBombSkill {
			t.Errorf("another Treasure Bomb from %d", m.CasterID)
		}
	}
	if mimicInteracted {
		t.Error("the mimic counts as interacted")
	}
}

// A script on the real game: on_spawn drives the NPC through the AI API,
// on_move_finished hears the end of a walk, on_teleport a teleport.
func TestScriptAIEventsScenario(t *testing.T) {
	const keltir = 20481
	e := newScriptAIEnv(t, map[string]string{"ai/test/events.lua": `
local s = script.new("AIEventsTest")
s:add_spawn_id(20481)
s:add_move_finished_id(20481)
s:add_teleport_id(20481)
s:on_spawn(function(npc)
  npc:disable_core_ai(true)
  npc:set_invul(true)
  npc:set_random_walking(false)
end)
s:on_move_finished(function(npc) npc:set_script_value(1) end)
s:on_teleport(function(npc) npc:set_script_value(2) end)
`})
	e.sync() // the scripts start
	n := e.spawnNpc(keltir, eastOf(tiStart, 100))
	o := e.srv.ScriptOutput()
	ref := n.ScriptRef()

	var coreOff, invul bool
	e.onLoop(func() { coreOff, invul = n.NpcCoreAIDisabled(), n.isInvul() })
	if !coreOff || !invul {
		t.Fatalf("after on_spawn: core AI off %v, invul %v; want both", coreOff, invul)
	}

	value := func() int {
		var v int
		e.onLoop(func() { v = e.srv.quest.scriptValues[n] })
		return v
	}
	dest := scripting.Location{X: int(tiStart.X) + 160, Y: int(tiStart.Y), Z: int(tiStart.Z)}
	e.onLoop(func() { o.NpcMoveTo(ref, dest) })
	for range 40 {
		if value() == 1 {
			break
		}
		e.advance(250 * time.Millisecond)
	}
	if v := value(); v != 1 {
		t.Fatalf("script value %d after the walk; want 1 from on_move_finished", v)
	}

	e.onLoop(func() { e.srv.TeleportNpc(n, eastOf(tiStart, 400)) })
	e.sync()
	if v := value(); v != 2 {
		t.Errorf("script value %d after the teleport; want 2 from on_teleport", v)
	}
}
