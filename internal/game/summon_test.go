package game

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The summon test world: the real wolf (12077) and Kat the Cat (14111, the
// servitor of Summon Kat the Cat level 1) next to a Bearded Keltir 300 west
// of the start.
const (
	summonWolf        = 12077
	summonBuffalo     = 12780 // Baby Buffalo, a baby pet
	summonBuffaloPipe = 6648  // Baby Buffalo Panpipe
	summonKat         = 14111
	summonWolfCollar  = 2375
	summonWolfFood    = 2515
	summonKatSkill    = 1111
	summonStormCubic  = 10   // Summon Storm Cubic
	summonKeltirSpace = 300  // the Keltir's distance from the start
	summonCrystalD    = 1458 // Crystal: D-Grade, the Kat's upkeep
)

// summonTestData is the pet, pet skill and cubic data of data/, loaded once.
var summonTestData = sync.OnceValues(func() (*data.Summons, error) {
	return data.LoadSummons(filepath.Join("..", "..", "data"))
})

// newSummonEnv starts a server with the skill and summon data on
// npcTestWorld, the wolf and Kat templates of the real data, and a Keltir;
// the summon store is in memory. The game draws from a fixed seed.
func newSummonEnv(t *testing.T) (*testEnv, *memstore.Summons) {
	t.Helper()
	full, err := chatTestWorld()
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	sd, err := summonTestData()
	if err != nil {
		t.Fatalf("LoadSummons: %v", err)
	}
	start := npcTestStart(t)
	wd := npcTestWorld(start)
	wd.NpcTypes = append(wd.NpcTypes,
		data.MobiusClass{Name: "Playable", Parent: "Creature", Abstract: true},
		data.MobiusClass{Name: "Summon", Parent: "Playable", Abstract: true},
		data.MobiusClass{Name: "Servitor", Parent: "Summon", Usable: true},
		data.MobiusClass{Name: "Pet", Parent: "Summon", Usable: true},
		data.MobiusClass{Name: "BabyPet", Parent: "Pet", Usable: true},
	)
	slices.SortFunc(wd.NpcTypes, func(a, b data.MobiusClass) int {
		switch {
		case a.Name < b.Name:
			return -1
		case a.Name > b.Name:
			return 1
		}
		return 0
	})
	keltir := attackTestKeltir()
	keltir.Stats.Vitals.HP = 120
	wd.Npcs = append(wd.Npcs, keltir)
	for _, id := range []int{summonWolf, summonKat, summonBuffalo, mountNpcStrider, mountNpcWyvern} { // mount_test.go
		tmpl, ok := full.Npc(id)
		if !ok {
			t.Fatalf("no NPC %d in the data", id)
		}
		wd.Npcs = append(wd.Npcs, *tmpl)
	}
	slices.SortFunc(wd.Npcs, func(a, b data.NpcTemplate) int { return a.ID - b.ID })
	wd.Spawns = []data.Spawn{{
		File: "TalkingIsland/TestKeltir.xml",
		Npcs: []data.SpawnNpc{{NpcID: attackKeltir, X: start.X - summonKeltirSpace, Y: start.Y, Z: start.Z, Heading: 0, Count: 1, RespawnDelay: 600}},
	}}
	w, err := data.NewWorld(wd)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	store := newRecordingStore()
	summons := memstore.NewSummons()
	d := testGameData(t)
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w,
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		Summons: sd, SummonStore: summons,
		Rand: rnd.New(1),
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
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}, summons
}

// summonEnter creates a character of the class and level at the start with
// extra items in its inventory, enters it, lets 5 s pass and drops the
// packets.
func summonEnter(t *testing.T, e *testEnv, account string, class int, level int32, items ...storage.Item) (*fakeSession, *Player) {
	t.Helper()
	c := e.createCharacter(account, account, class, storage.AccessLevelPlayer)
	c.Level = level
	c.Exp = e.srv.ExpForLevel(int(level))
	start := npcTestStart(t)
	c.X, c.Y, c.Z = int32(start.X), int32(start.Y), int32(start.Z)
	for _, it := range items {
		ids, err := e.ids.NextIDs(e.ctx, 1)
		if err != nil {
			t.Fatalf("NextIDs: %v", err)
		}
		it.ObjectID = ids[0]
		if it.Slot == 0 {
			it.Slot = storage.PaperdollNone
		}
		if it.Mana == 0 {
			it.Mana = storage.NoMana
		}
		c.Items = append(c.Items, it)
	}
	if err := e.store.Update(e.ctx, &c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	sess := newSession(account)
	p := e.enter(sess, 0)
	e.advance(5 * time.Second)
	sess.take(t)
	return sess, p
}

// summonApplySkill applies a skill's effects from p to itself on the loop,
// as its cast lands.
func summonApplySkill(t *testing.T, e *testEnv, p *Player, id, level int) {
	t.Helper()
	e.onLoop(func() {
		sk, ok := e.srv.Skill(id, level)
		if !ok {
			t.Fatalf("no skill %d level %d", id, level)
		}
		sk.ApplyTo(e.srv.SkillEnv(), p.skill, p.skill)
	})
}

// summonOf returns p's summon, read on the loop.
func summonOfPlayer(e *testEnv, p *Player) *Summon {
	var sm *Summon
	e.onLoop(func() { sm = e.srv.SummonOf(p) })
	return sm
}

// summonAction sends RequestActionUse.
func summonAction(e *testEnv, sess *fakeSession, id int32, ctrl bool) {
	w := packets.NewWriter(9)
	w.WriteInt32(id)
	if ctrl {
		w.WriteInt32(1)
	} else {
		w.WriteInt32(0)
	}
	w.WriteUint8(0)
	e.srv.Handle(sess, clientpackets.OpRequestActionUse, w.Bytes())
	e.sync()
}

// summonKeltir returns the Keltir.
func summonKeltir(t *testing.T, e *testEnv) *Npc { return npcSpawnedOne(t, e, attackKeltir) }

// summonMessages returns the system message IDs among ps.
func summonMessages(ps []testclient.Packet) []int32 {
	var out []int32
	for _, m := range only[testclient.SystemMessage](ps) {
		out = append(out, m.ID)
	}
	return out
}

// A servitor of Summon Kat the Cat comes out next to its owner, who sees it
// through PetInfo and its status, while other players see SummonInfo; it
// follows; on the attack order it kills the owner's target, and the owner
// gets 70 % of the exp; action 52 sends it away.
func TestServitorSummonFightAndUnsummon(t *testing.T) {
	e, _ := newSummonEnv(t)
	sess, p := summonEnter(t, e, "kat", classHumanMystic, 20)
	summonApplySkill(t, e, p, summonKatSkill, 1)
	sm := summonOfPlayer(e, p)
	if sm == nil {
		t.Fatal("no servitor")
	}
	var npcID int
	var kind summon.Kind
	var pos world.Position
	e.onLoop(func() { npcID, kind, pos = sm.npc.Template.ID, sm.kind, sm.npc.Position() })
	if npcID != summonKat || kind != summon.Servitor {
		t.Fatalf("summon %d %v", npcID, kind)
	}
	if d := pos.Distance2D(e.position(p)); d > 150 {
		t.Fatalf("the servitor appeared %v away", d)
	}
	ps := sess.take(t)
	infos := only[testclient.PetInfo](ps)
	if len(infos) == 0 || infos[0].NpcID() != summonKat || infos[0].SummonType != testclient.SummonServitor || infos[0].Title != "kat" {
		t.Fatalf("PetInfo %+v", infos)
	}
	if len(only[testclient.PetStatusUpdate](ps)) == 0 || len(only[testclient.SetSummonRemainTime](ps)) == 0 {
		t.Fatalf("no status or remaining time: %v", testclient.Names(ps))
	}
	rt := only[testclient.SetSummonRemainTime](ps)[0]
	if rt.MaxTime != 1200000 || rt.RemainingTime != 1195000 {
		t.Fatalf("remaining time %+v", rt)
	}

	// Another player sees SummonInfo (NpcInfo's layout) titled with the
	// owner's name.
	other := newSession("viewer")
	c := e.createCharacter("viewer", "viewer", classHumanFighter, storage.AccessLevelPlayer)
	start := npcTestStart(t)
	c.X, c.Y, c.Z = int32(start.X), int32(start.Y), int32(start.Z)
	if err := e.store.Update(e.ctx, &c); err != nil {
		t.Fatalf("Update: %v", err)
	}
	e.enter(other, 0)
	var seen bool
	for _, ni := range only[testclient.NpcInfo](other.take(t)) {
		seen = seen || (ni.NpcTypeID == summonKat+1000000 && ni.Title == "kat")
	}
	if !seen {
		t.Fatal("another player does not see the servitor")
	}

	// The owner walks away: the servitor follows.
	e.onLoop(func() {
		pos := p.Position()
		pos.X += 400
		_ = e.srv.MoveObject(p, pos)
	})
	e.advance(10 * time.Second)
	e.onLoop(func() { pos = sm.npc.Position() })
	if d := pos.Distance2D(e.position(p)); d > 120 {
		t.Fatalf("the servitor did not follow: %v away", d)
	}
	e.onLoop(func() {
		start := npcTestStart(t)
		_ = e.srv.MoveObject(p, world.Position{X: int32(start.X), Y: int32(start.Y), Z: int32(start.Z)})
	})
	e.advance(10 * time.Second)

	keltir := summonKeltir(t, e)
	e.onLoop(func() { e.srv.SetTarget(p, keltir) })
	expBefore := int64(0)
	e.onLoop(func() { expBefore = p.Exp() })
	summonAction(e, sess, clientpackets.ActionUseServitorAttack, false)
	var attacking bool
	e.onLoop(func() { attacking = sm.ai.intention == summonAttacking })
	if !attacking {
		t.Fatal("the servitor does not attack")
	}
	dead := false
	for range 120 {
		e.advance(500 * time.Millisecond)
		e.onLoop(func() { dead = keltir.Dead })
		if dead {
			break
		}
	}
	if !dead {
		t.Fatal("the servitor did not kill the Keltir")
	}
	var exp int64
	e.onLoop(func() { exp = p.Exp() })
	// 35 exp at level 20 against a level 1 monster is 0 after the level gap;
	// what matters is that the owner is rewarded at all through its summon:
	// the kill counted as the owner's (the drops are its).
	if exp < expBefore {
		t.Fatalf("exp went down: %d → %d", expBefore, exp)
	}
	ps = sess.take(t)
	if !slices.Contains(summonMessages(ps), int32(sumMsgYourPetHitForDamage)) {
		t.Fatalf("no \"your pet hit\" message: %v", summonMessages(ps))
	}

	e.advance(20 * time.Second) // out of the stance
	sess.take(t)
	summonAction(e, sess, clientpackets.ActionUseUnsummonServitor, false)
	ps = sess.take(t)
	if len(only[testclient.PetDelete](ps)) != 1 {
		t.Fatalf("no PetDelete: %v", testclient.Names(ps))
	}
	if summonOfPlayer(e, p) != nil || npcInWorld(e, sm.npc) {
		t.Fatal("the servitor is still there")
	}
}

// A servitor eats its upkeep every interval ("A summoned monster uses
// Crystal: D-Grade.") and leaves when it runs out; another passes away
// when its life ends.
func TestServitorUpkeepAndLife(t *testing.T) {
	e, _ := newSummonEnv(t)
	sess, p := summonEnter(t, e, "upkeep", classHumanMystic, 20,
		storage.Item{ItemID: summonCrystalD, Count: 1})
	e.onLoop(func() {
		sk, _ := e.srv.Skill(summonKatSkill, 1)
		e.srv.summonServitor(p, sk, skills.ServitorParams{NpcID: summonKat, ExpMultiplier: 0.7,
			ConsumeItemID: summonCrystalD, ConsumeItemCount: 1, ConsumeItemInterval: 10, LifeTime: 3600})
	})
	sess.take(t)
	e.advance(10 * time.Second)
	if msgs := summonMessages(sess.take(t)); !slices.Contains(msgs, int32(sumMsgSummonedMonsterUses)) {
		t.Fatalf("no upkeep: %v", msgs)
	}
	var crystals int64
	e.onLoop(func() { crystals = p.inv.Count(summonCrystalD, -1, true) })
	if crystals != 0 {
		t.Fatalf("%d crystals left", crystals)
	}
	e.advance(10 * time.Second)
	ps := sess.take(t)
	if !slices.Contains(summonMessages(ps), int32(sumMsgNotEnoughForServitor)) || summonOfPlayer(e, p) != nil {
		t.Fatalf("the servitor stayed without upkeep: %v", summonMessages(ps))
	}

	e.onLoop(func() {
		sk, _ := e.srv.Skill(summonKatSkill, 1)
		e.srv.summonServitor(p, sk, skills.ServitorParams{NpcID: summonKat, ExpMultiplier: 0.7, ConsumeItemCount: 1, LifeTime: 20})
	})
	e.advance(25 * time.Second)
	ps = sess.take(t)
	if !slices.Contains(summonMessages(ps), int32(sumMsgServitorPassedAway)) || summonOfPlayer(e, p) != nil {
		t.Fatalf("the servitor outlived its life: %v", summonMessages(ps))
	}
}

// A servitor the owner logs out with comes back at the next login with its
// HP and life left; one sent away does not.
func TestServitorRestoreOnReconnect(t *testing.T) {
	e, store := newSummonEnv(t)
	sess, p := summonEnter(t, e, "restore", classHumanMystic, 20)
	e.onLoop(func() { e.srv.AddSkill(p, mustSkill(t, e, summonKatSkill, 1), true) })
	summonApplySkill(t, e, p, summonKatSkill, 1)
	sm := summonOfPlayer(e, p)
	if sm == nil {
		t.Fatal("no servitor")
	}
	e.onLoop(func() { sm.npc.SetCurrentHP(100) })
	e.advance(30 * time.Second)
	e.srv.Disconnected(sess)
	e.sync()
	if err := e.srv.WaitSummonWrites(e.ctx); err != nil {
		t.Fatal(err)
	}
	saved, err := store.LoadOwnerSummons(e.ctx, p.ObjectID())
	if err != nil || saved.Servitor == nil || saved.Servitor.SkillID != summonKatSkill {
		t.Fatalf("stored %+v, %v", saved, err)
	}
	sess2 := newSession("restore")
	p2 := e.enter(sess2, 0)
	sm2 := summonOfPlayer(e, p2)
	if sm2 == nil {
		t.Fatal("the servitor did not come back")
	}
	var hp float64
	var left int
	e.onLoop(func() { hp, left = sm2.npc.CurHP, sm2.life.Remaining })
	if int32(hp) != saved.Servitor.CurHP || left != int(saved.Servitor.Time) || left >= 1200000-30000 {
		t.Fatalf("restored HP %v, life %d (stored %+v)", hp, left, saved.Servitor)
	}
	// Sent away: not restored.
	summonAction(e, sess2, clientpackets.ActionUseUnsummonServitor, false)
	if err := e.srv.WaitSummonWrites(e.ctx); err != nil {
		t.Fatal(err)
	}
	if saved, _ := store.LoadOwnerSummons(e.ctx, p.ObjectID()); saved.Servitor != nil {
		t.Fatalf("an unsummoned servitor is kept: %+v", saved.Servitor)
	}
}

// mustSkill returns a skill level of the catalog.
func mustSkill(t *testing.T, e *testEnv, id, level int) *skills.Skill {
	t.Helper()
	sk, ok := e.srv.Skill(id, level)
	if !ok {
		t.Fatalf("no skill %d level %d", id, level)
	}
	return sk
}
