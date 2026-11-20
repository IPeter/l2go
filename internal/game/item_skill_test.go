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
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// Items of the item skill tests (Mobius item IDs).
const (
	itemSoulshotNG    = 1835 // Soulshot: No Grade (SoulShots, skill 2039)
	itemSoulshotD     = 1463 // Soulshot: D-grade
	itemHealingPotion = 1061 // Healing Potion (ItemSkills, skill 2032, reuse 10 s)
	itemSoE           = 736  // Scroll of Escape (skill 2013, a 20 s cast)
	itemHerbOfLife    = 8600 // Herb of Life (skill 2278)
	itemCharmNG       = 8515 // Charm of Courage: No Grade (reuse 20 min)
	itemWoodenChest   = 23   // Wooden Breastplate
	itemWoodenLegs    = 2386 // Wooden Gaiters
	itemWoodenHelmet  = 43   // Wooden Helmet
	itemBastardSword  = 69   // a D-grade sword
)

// lowRand is a random source that always gives the lowest answer: every
// swing hits.
type lowRand struct{}

func (lowRand) Get(int) int                          { return 0 }
func (lowRand) Between(lo, _ int) int                { return lo }
func (lowRand) GetInt64(int64) int64                 { return 0 }
func (lowRand) BetweenInt64(lo, _ int64) int64       { return lo }
func (lowRand) GetFloat32(float32) float32           { return 0 }
func (lowRand) BetweenFloat32(lo, _ float32) float32 { return lo }
func (lowRand) GetFloat64(float64) float64           { return 0 }
func (lowRand) BetweenFloat64(lo, _ float64) float64 { return lo }
func (lowRand) Int32() int32                         { return 0 }
func (lowRand) Int64() int64                         { return 0 }
func (lowRand) Float32() float32                     { return 0 }
func (lowRand) Float64() float64                     { return 0 }
func (lowRand) Bool() bool                           { return false }
func (lowRand) Gaussian() float64                    { return 0 }

var _ rnd.Source = lowRand{}

// newItemSkillEnv starts a server with the skill data and the initial
// shortcuts of data/, on the world w. With attackRand the swings draw from
// it and monsters never swing back.
func newItemSkillEnv(t *testing.T, w *data.World, attackRand rnd.Source) *testEnv {
	t.Helper()
	d := testGameData(t)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = filepath.Join("..", "..", "data")
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w, Characters: store, IDs: ids,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		Clock: clock, CheckLoopOwner: true, Rand: rnd.New(1),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if attackRand != nil {
		srv.attack.rnd = attackRand
		srv.ai.doAttack = func(*Npc, Creature) {}
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

// enterWithItems stores a Human Fighter of the level at pos (zero: its
// creation point) with extra items, enters it and drops the packets of the
// first 5 s.
func enterWithItems(t *testing.T, e *testEnv, account string, level int32, pos world.Position, items ...storage.Item) (*fakeSession, *Player) {
	t.Helper()
	c := e.createCharacter(account, account, classHumanFighter, storage.AccessLevelPlayer)
	c.Level = level
	if level > 1 {
		c.Exp = e.srv.ExpForLevel(int(level))
	}
	if pos != (world.Position{}) {
		c.X, c.Y, c.Z = pos.X, pos.Y, pos.Z
	}
	if len(items) > 0 {
		ids, err := e.ids.NextIDs(e.ctx, len(items))
		if err != nil {
			t.Fatal(err)
		}
		for i := range items {
			items[i].ObjectID = ids[i]
			if items[i].Slot == 0 && items[i].Loc == "" {
				items[i].Slot = storage.PaperdollNone
			}
			items[i].Mana = storage.NoMana
		}
		c.Items = append(c.Items, items...)
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

// handleOn sends a client packet and returns what came back after the loop
// ran it.
func handleOn(e *testEnv, sess *fakeSession, op byte, body []byte) []testclient.Packet {
	e.t.Helper()
	e.srv.Handle(sess, op, body)
	e.sync()
	return sess.take(e.t)
}

// handleExOn sends an Ex client packet.
func handleExOn(e *testEnv, sess *fakeSession, sub uint16, body []byte) []testclient.Packet {
	e.t.Helper()
	e.srv.HandleEx(sess, sub, body)
	e.sync()
	return sess.take(e.t)
}

func useItemOn(e *testEnv, sess *fakeSession, objectID int32) []testclient.Packet {
	e.t.Helper()
	return handleOn(e, sess, clientpackets.OpUseItem, ints(objectID, 0))
}

// countOf returns how many items of the template p carries, on the loop.
func countOf(e *testEnv, p *Player, itemID int32) int64 {
	var n int64
	e.onLoop(func() { n = p.inv.Count(itemID, -1, true) })
	return n
}

// objectOf returns the object ID of p's first item of the template.
func objectOf(e *testEnv, p *Player, itemID int32) int32 {
	var id int32
	e.onLoop(func() {
		if it := p.inv.ByItemID(itemID); it != nil {
			id = it.ObjectID()
		}
	})
	if id == 0 {
		e.t.Fatalf("no item %d", itemID)
	}
	return id
}

// textsOf returns the texts of the system messages that carry one.
func textsOf(ps []testclient.Packet) []string {
	var out []string
	for _, m := range only[testclient.SystemMessage](ps) {
		for _, prm := range m.Params {
			if prm.Text != "" {
				out = append(out, prm.Text)
			}
		}
	}
	return out
}

// The shortcut bar: a skill shortcut takes the known level, an item
// shortcut needs the item; deleting a shot's shortcut turns its automatic
// use off and resends the bar; the bar survives a relog; a skill removed
// for good takes its shortcuts with it.
func TestShortcuts(t *testing.T) {
	e := newItemSkillEnv(t, nil, nil)
	sess, p := enterWithItems(t, e, "bar", 1, world.Position{}, storage.Item{ItemID: itemSoulshotNG, Count: 100})
	shot := objectOf(e, p, itemSoulshotNG)
	reg := func(typ, number, id int32) []testclient.Packet {
		return handleOn(e, sess, clientpackets.OpRequestShortcutReg, ints(typ, number, id, 7))
	}

	// Lucky (194) is known at level 1: page 1, slot 1.
	ps := reg(storage.ShortcutSkill, 13, 194)
	regs := only[testclient.ShortcutRegister](ps)
	if len(regs) != 1 || regs[0].Type != storage.ShortcutSkill || regs[0].Slot != 13 || regs[0].ID != 194 || regs[0].Level != 1 {
		t.Fatalf("skill shortcut: %+v", ps)
	}
	if ps := reg(storage.ShortcutSkill, 14, 1177); len(ps) != 0 {
		t.Errorf("an unknown skill's shortcut: %v", testclient.Names(ps))
	}
	if ps := reg(storage.ShortcutAction, 12*11, 2); len(ps) != 0 {
		t.Errorf("page 11: %v", testclient.Names(ps))
	}
	if ps := reg(storage.ShortcutItem, 0, shot); len(only[testclient.ShortcutRegister](ps)) != 1 {
		t.Fatalf("item shortcut: %v", testclient.Names(ps))
	}
	// A missing item: ShortcutRegister anyway, nothing stored.
	if ps := reg(storage.ShortcutItem, 1, 999999); len(only[testclient.ShortcutRegister](ps)) != 1 {
		t.Errorf("missing item: %v", testclient.Names(ps))
	}
	var keys []int
	e.onLoop(func() {
		for _, sc := range p.itemSkill.bar.all() {
			keys = append(keys, sc.key())
		}
	})
	if !slices.Equal(keys, []int{0, 13}) {
		t.Fatalf("bar %v, want slots 0 and 13", keys)
	}

	// The soulshot goes automatic, then its shortcut is deleted.
	ps = handleExOn(e, sess, clientpackets.ExOpRequestAutoSoulShot, ints(itemSoulshotNG, 1))
	if a := only[testclient.ExAutoSoulShot](ps); len(a) != 1 || !a[0].Enabled {
		t.Fatalf("auto on: %v", testclient.Names(ps))
	}
	ps = handleOn(e, sess, clientpackets.OpRequestShortcutDel, ints(0))
	checkNames(t, "deleting the shot's shortcut", ps, "ExAutoSoulShot", "ShortcutInit")
	if a := only[testclient.ExAutoSoulShot](ps)[0]; a.ItemID != itemSoulshotNG || a.Enabled {
		t.Errorf("ExAutoSoulShot %+v, want off", a)
	}
	if init := only[testclient.ShortcutInit](ps)[0]; len(init.Shortcuts) != 1 || init.Shortcuts[0].Slot != 13 {
		t.Errorf("ShortcutInit %+v", init)
	}
	e.onLoop(func() {
		if len(p.AutoShots()) != 0 {
			t.Errorf("auto shots %v after the shortcut went", p.AutoShots())
		}
	})

	// The item shortcut goes with the item.
	reg(storage.ShortcutItem, 5, shot)
	e.onLoop(func() {
		it := p.inv.ByObjectID(shot)
		e.srv.DestroyItem(p, it, it.Count(), false)
	})
	ps = sess.take(t)
	if inits := only[testclient.ShortcutInit](ps); len(inits) != 1 || len(inits[0].Shortcuts) != 1 {
		t.Errorf("after the item went: %v", testclient.Names(ps))
	}

	// A new level updates the skill's shortcuts.
	e.onLoop(func() { e.srv.UpdateSkillShortcuts(p, 194, 2) })
	if r := only[testclient.ShortcutRegister](sess.take(t)); len(r) != 1 || r[0].Level != 2 {
		t.Errorf("UpdateSkillShortcuts: %+v", r)
	}

	// The bar survives a relog.
	c := p.char
	e.srv.Restart(sess)
	saved := e.store.waitSave(t, c.ObjectID)
	want := []storage.Shortcut{{Slot: 1, Page: 1, Type: storage.ShortcutSkill, ID: 194, Level: 2}}
	if !slices.Equal(saved.Shortcuts, want) {
		t.Fatalf("saved %+v, want %+v", saved.Shortcuts, want)
	}
	sess2 := newSession("bar")
	p2 := e.enter(sess2, 0)
	inits := only[testclient.ShortcutInit](sess2.take(t))
	if len(inits) != 1 || len(inits[0].Shortcuts) != 1 || inits[0].Shortcuts[0].ID != 194 || inits[0].Shortcuts[0].Level != 2 {
		t.Fatalf("ShortcutInit at enter: %+v", inits)
	}

	// Forgetting the skill for good deletes its shortcut.
	e.onLoop(func() { e.srv.RemoveSkill(p2, 194, true, true) })
	if inits := only[testclient.ShortcutInit](sess2.take(t)); len(inits) != 1 || len(inits[0].Shortcuts) != 0 {
		t.Errorf("after RemoveSkill: %+v", inits)
	}
}

// A new Human Mystic gets Mobius's initial bar: the actions and the Tutorial
// Guide of every class, Wind Strike on F2 and Self Heal on F10.
func TestNewCharacterShortcuts(t *testing.T) {
	e := newItemSkillEnv(t, nil, nil)
	c := storage.Character{
		ClassID: classHumanMystic,
		Items:   []storage.Item{{ObjectID: 501, ItemID: 5588, Count: 1}},
		Skills:  []storage.CharacterSkill{{ID: 1177, Level: 1}, {ID: 1216, Level: 1}},
	}
	got := e.srv.NewCharacterShortcuts(&c)
	want := []storage.Shortcut{
		{Slot: 0, Type: storage.ShortcutAction, ID: 2},
		{Slot: 3, Type: storage.ShortcutAction, ID: 5},
		{Slot: 10, Type: storage.ShortcutAction, ID: 0},
		{Slot: 11, Type: storage.ShortcutItem, ID: 501},
		{Slot: 1, Type: storage.ShortcutSkill, ID: 1177, Level: 1},
		{Slot: 9, Type: storage.ShortcutSkill, ID: 1216, Level: 1},
	}
	if !slices.Equal(got, want) {
		t.Errorf("shortcuts\n %+v\nwant\n %+v", got, want)
	}
	c.Items, c.ClassID = nil, classHumanFighter
	if got := e.srv.NewCharacterShortcuts(&c); len(got) != 3 {
		t.Errorf("a fighter without the guide: %+v", got)
	}
}

// Manual and automatic soulshots over auto-attacks: the grade check, one
// shot per charge, the Attack's soulshot flag, the discharge by a landed
// swing and the recharge after each hit.
func TestSoulshotsOverAttacks(t *testing.T) {
	e := newAttackEnvWithSkills(t)
	sess, p := enterWithItems(t, e, "shot", 1, world.Position{},
		storage.Item{ItemID: itemSoulshotNG, Count: 10}, storage.Item{ItemID: itemSoulshotD, Count: 5})
	keltir := npcSpawnedOne(t, e, attackKeltir)

	// A D-grade soulshot does not fit the Squire's Sword.
	ps := useItemOn(e, sess, objectOf(e, p, itemSoulshotD))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgSoulshotGrade)}) {
		t.Errorf("D-grade shot: messages %v", ids)
	}
	ng := objectOf(e, p, itemSoulshotNG)
	ps = useItemOn(e, sess, ng)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgSpiritsEnabled)}) {
		t.Errorf("soulshot: messages %v", ids)
	}
	if u := only[testclient.MagicSkillUse](ps); len(u) != 1 || u[0].SkillID != 2039 || u[0].HitTime != 0 {
		t.Errorf("soulshot glow %+v", u)
	}
	useItemOn(e, sess, ng) // already charged: nothing
	if n := countOf(e, p, itemSoulshotNG); n != 9 {
		t.Fatalf("%d soulshots after two uses, want 9", n)
	}
	charged := func() bool {
		var c bool
		e.onLoop(func() { c = e.srv.ChargedShot(p, skills.Soulshot) })
		return c
	}
	if !charged() {
		t.Fatal("the weapon is not charged")
	}

	// The swing shows the soulshot and spends it.
	attackClick(e, sess, keltir, false)
	attackClick(e, sess, keltir, false)
	attacks := only[testclient.Attack](sess.take(t))
	if len(attacks) != 1 || !attacks[0].Hits[0].Soulshot() || attacks[0].Hits[0].Grade() != 0 {
		t.Fatalf("first swing %+v, want a no-grade soulshot", attacks)
	}
	if charged() {
		t.Error("the landed swing did not spend the soulshot")
	}
	e.advance(1200 * time.Millisecond) // the hit, then the next swing
	attacks = only[testclient.Attack](sess.take(t))
	if len(attacks) != 1 || attacks[0].Hits[0].Soulshot() {
		t.Fatalf("second swing %+v, want no soulshot", attacks)
	}

	// Automatic: on, recharged at once, and after every hit.
	ps = handleExOn(e, sess, clientpackets.ExOpRequestAutoSoulShot, ints(itemSoulshotNG, 1))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgAutoUseActivated), int32(sysMsgSpiritsEnabled)}) {
		t.Errorf("auto on: messages %v", ids)
	}
	if n := countOf(e, p, itemSoulshotNG); n != 8 || !charged() {
		t.Fatalf("auto on: %d soulshots, charged %v; want 8 and charged", n, charged())
	}
	e.advance(1200 * time.Millisecond) // the second swing's hit recharges (no-op), the third swing shows it
	attacks = only[testclient.Attack](sess.take(t))
	if len(attacks) == 0 || !attacks[len(attacks)-1].Hits[0].Soulshot() {
		t.Fatalf("third swing %+v, want a soulshot", attacks)
	}
	e.advance(700 * time.Millisecond) // its hit: a new shot
	if n := countOf(e, p, itemSoulshotNG); n != 7 || !charged() {
		t.Errorf("after an automatic hit: %d soulshots, charged %v; want 7 and charged", n, charged())
	}

	// Off.
	ps = handleExOn(e, sess, clientpackets.ExOpRequestAutoSoulShot, ints(itemSoulshotNG, 0))
	if a := only[testclient.ExAutoSoulShot](ps); len(a) != 1 || a[0].Enabled {
		t.Errorf("auto off: %+v", a)
	}
}

// Power Strike with automatic soulshots: the skill spends the charged shot
// at its hit and casting charges a new one (SetShotRecharger).
func TestSoulshotsOverPowerStrike(t *testing.T) {
	e := newAttackEnvWithSkills(t)
	sess, p := enterWithItems(t, e, "strike", 5, world.Position{}, storage.Item{ItemID: itemSoulshotNG, Count: 10})
	keltir := npcSpawnedOne(t, e, attackKeltir)
	handleExOn(e, sess, clientpackets.ExOpRequestAutoSoulShot, ints(itemSoulshotNG, 1))
	if n := countOf(e, p, itemSoulshotNG); n != 9 {
		t.Fatalf("%d soulshots after turning them on, want 9", n)
	}
	strike := e.skill(3, 1)
	var started bool
	var hpBefore float64
	e.onLoop(func() {
		e.srv.AddSkill(p, strike, false)
		e.srv.SetTarget(p, keltir)
		hpBefore = keltir.CurrentHP()
		started = e.srv.UseMagic(p, strike, false, false)
	})
	if !started {
		t.Fatal("UseMagic refused Power Strike")
	}
	ps := sess.take(t)
	if u := only[testclient.MagicSkillUse](ps); len(u) != 1 || u[0].SkillID != 3 {
		t.Fatalf("Power Strike: %v", testclient.Names(ps))
	}
	e.advance(3 * time.Second)
	var hpAfter float64
	var charged bool
	e.onLoop(func() {
		hpAfter = keltir.CurrentHP()
		charged = e.srv.ChargedShot(p, skills.Soulshot)
	})
	if hpAfter >= hpBefore {
		t.Errorf("Keltir HP %v → %v: Power Strike did not hit", hpBefore, hpAfter)
	}
	if n := countOf(e, p, itemSoulshotNG); n >= 9 || !charged {
		t.Errorf("after Power Strike: %d soulshots, charged %v; want fewer than 9 and charged", n, charged)
	}
}

// newAttackEnvWithSkills is newAttackEnv's world (a Bearded Keltir west of
// the start) with the skill data, every swing a hit.
func newAttackEnvWithSkills(t *testing.T) *testEnv {
	t.Helper()
	start := npcTestStart(t)
	wd := npcTestWorld(start)
	wd.Npcs = append(wd.Npcs, attackTestKeltir())
	wd.Spawns = []data.Spawn{{
		File: "TalkingIsland/TestKeltir.xml",
		Npcs: []data.SpawnNpc{{NpcID: attackKeltir, X: start.X - attackKeltirDistance, Y: start.Y, Z: start.Z, Heading: 0, Count: 1, RespawnDelay: 60}},
	}}
	w, err := data.NewWorld(wd)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return newItemSkillEnv(t, w, lowRand{})
}

// A Healing Potion: cast at once, its icon as the short buff, HP every
// tick, and a second potion within 10 s refused with the time left. A Herb
// of Life heals on pick-up and never enters the inventory. A Scroll of
// Escape casts for 20 s and then takes the player to town.
func TestPotionHerbAndScroll(t *testing.T) {
	e := newItemSkillEnv(t, testWorldData(t), nil)
	sess, p := enterWithItems(t, e, "potion", 1, tiStart,
		storage.Item{ItemID: itemHealingPotion, Count: 2}, storage.Item{ItemID: itemSoE, Count: 1})
	var maxHP float64
	e.onLoop(func() {
		maxHP = float64(p.MaxHP())
		p.SetCurrentHP(10)
	})
	sess.take(t)
	hp := func() float64 {
		var v float64
		e.onLoop(func() { v = p.CurrentHP() })
		return v
	}

	potion := objectOf(e, p, itemHealingPotion)
	ps := useItemOn(e, sess, potion)
	if u := only[testclient.MagicSkillUse](ps); len(u) != 1 || u[0].SkillID != 2032 {
		t.Fatalf("potion: %v", testclient.Names(ps))
	}
	if n := countOf(e, p, itemHealingPotion); n != 1 {
		t.Errorf("%d potions left, want 1", n)
	}
	e.advance(500 * time.Millisecond)
	if sb := only[testclient.ShortBuffStatusUpdate](sess.take(t)); len(sb) != 1 || sb[0].SkillID != 2032 || sb[0].Duration != 15 {
		t.Errorf("short buff %+v, want 2032 for 15 s", sb)
	}
	before := hp()
	e.advance(2 * time.Second)
	if after := hp(); after <= before {
		t.Errorf("HP %v → %v: no tick", before, after)
	}

	// Within the potion's 10 s reuse.
	ps = useItemOn(e, sess, potion)
	if txt := textsOf(ps); len(txt) != 1 || !strings.Contains(txt[0], "second(s) remaining in Healing Potion's re-use time") {
		t.Errorf("second potion: %q", txt)
	}
	if n := countOf(e, p, itemHealingPotion); n != 1 {
		t.Errorf("%d potions after the refusal, want 1", n)
	}

	// A herb on the ground.
	e.advance(20 * time.Second)
	sess.take(t)
	e.onLoop(func() { p.SetCurrentHP(10) })
	var g *GroundItem
	e.onLoop(func() {
		tmpl, _ := e.data.ItemTemplate(itemHerbOfLife)
		ids, err := e.ids.NextIDs(e.ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		g, err = e.srv.DropOnGround(inventory.NewItem(ids[0], tmpl, 1), p.Position(), GroundDrop{})
		if err != nil {
			t.Fatal(err)
		}
		e.srv.doPickUp(p, g)
	})
	e.advance(time.Second)
	if got, want := hp(), 10+maxHP*15/100; got < want-1 {
		t.Errorf("HP %v after the herb, want about %v", got, want)
	}
	var onGround bool
	e.onLoop(func() { onGround = g.InWorld() })
	if n := countOf(e, p, itemHerbOfLife); n != 0 || onGround {
		t.Errorf("the herb is still around: %d carried, on the ground %v", n, onGround)
	}
	sess.take(t)

	// Scroll of Escape.
	from := e.position(p)
	ps = useItemOn(e, sess, objectOf(e, p, itemSoE))
	u := only[testclient.MagicSkillUse](ps)
	if len(u) != 1 || u[0].SkillID != 2013 || u[0].HitTime != 20000 {
		t.Fatalf("scroll: %v %+v", testclient.Names(ps), u)
	}
	if n := countOf(e, p, itemSoE); n != 0 {
		t.Errorf("the scroll was not used up at the cast start: %d", n)
	}
	e.advance(19 * time.Second)
	if pos := e.position(p); pos.Distance2D(from) > 100 {
		t.Fatalf("moved before the cast ended: %v", pos)
	}
	sess.take(t)
	e.advance(2 * time.Second)
	// The teleport ends when the client appears; TeleportToLocation tells
	// where to.
	tp := only[testclient.TeleportToLocation](sess.take(t))
	var town data.Location
	e.onLoop(func() { town, _ = e.srv.restartLocation(p) })
	townPos := world.Position{X: int32(town.X), Y: int32(town.Y)}
	if len(tp) != 1 || (world.Position{X: tp[0].X, Y: tp[0].Y}).Distance2D(townPos) > 500 {
		t.Errorf("TeleportToLocation %+v, want the restart point %+v", tp, town)
	}
}

// The Wooden set gives 3500 when complete and loses it when a part comes
// off; a D-grade weapon at level 1 brings the grade penalty (4267 level 1)
// with its icon; carrying over half the weight limit the weight penalty.
func TestArmorSetAndPenalties(t *testing.T) {
	e := newItemSkillEnv(t, testWorldData(t), nil)
	sess, p := enterWithItems(t, e, "set", 1, tiStart,
		storage.Item{ItemID: itemWoodenChest, Count: 1}, storage.Item{ItemID: itemWoodenLegs, Count: 1},
		storage.Item{ItemID: itemWoodenHelmet, Count: 1}, storage.Item{ItemID: itemBastardSword, Count: 1})
	level := func(id int) int {
		var l int
		e.onLoop(func() { l = p.SkillLevel(id) })
		return l
	}
	useItemOn(e, sess, objectOf(e, p, itemWoodenChest))
	useItemOn(e, sess, objectOf(e, p, itemWoodenLegs))
	if level(3500) != 0 {
		t.Fatal("3500 before the set is complete")
	}
	useItemOn(e, sess, objectOf(e, p, itemWoodenHelmet))
	if level(3500) != 1 || level(3006) != 1 {
		t.Fatalf("set skills 3500 %d, 3006 %d; want 1 and 1", level(3500), level(3006))
	}
	e.advance(skillListDelay)
	lists := only[testclient.SkillList](sess.take(t))
	if len(lists) != 1 || !slices.ContainsFunc(lists[0].Skills, func(s testclient.Skill) bool { return s.ID == 3500 }) {
		t.Errorf("SkillList %+v without 3500", lists)
	}
	useItemOn(e, sess, objectOf(e, p, itemWoodenHelmet))
	if level(3500) != 0 {
		t.Error("3500 stayed without the helmet")
	}

	// The grade penalty.
	ps := useItemOn(e, sess, objectOf(e, p, itemBastardSword))
	etc := only[testclient.EtcStatusUpdate](ps)
	if len(etc) != 1 || !etc[0].GradePenalty || level(4267) != 1 {
		t.Fatalf("D-grade sword: EtcStatusUpdate %+v, 4267 level %d", etc, level(4267))
	}
	ps = useItemOn(e, sess, objectOf(e, p, itemBastardSword))
	if etc := only[testclient.EtcStatusUpdate](ps); len(etc) != 1 || etc[0].GradePenalty || level(4267) != 0 {
		t.Errorf("sword off: EtcStatusUpdate %+v, 4267 level %d", etc, level(4267))
	}

	// The weight penalty: over half the limit.
	var limit int
	e.onLoop(func() { limit = p.stats.WeightLimit() })
	tmpl, _ := e.data.ItemTemplate(itemMithrilOre)
	e.onLoop(func() {
		if _, err := p.inv.Create(itemMithrilOre, (int64(limit)*55/100-p.inv.Weight())/int64(tmpl.Weight)); err != nil {
			t.Fatal(err)
		}
		e.srv.ItemsChanged(p)
	})
	if etc := only[testclient.EtcStatusUpdate](sess.take(t)); len(etc) != 1 || etc[0].WeightPenalty != 1 {
		t.Errorf("weight: EtcStatusUpdate %+v, want penalty 1", etc)
	}
}

// A Charm of Courage shows its icon until death; its 20 minute item reuse
// refuses a second one and survives a relog.
func TestCharmOfCourageAndItemReuse(t *testing.T) {
	e := newItemSkillEnv(t, testWorldData(t), nil)
	sess, p := enterWithItems(t, e, "charm", 1, tiStart, storage.Item{ItemID: itemCharmNG, Count: 2})
	charm := objectOf(e, p, itemCharmNG)
	ps := useItemOn(e, sess, charm)
	if etc := only[testclient.EtcStatusUpdate](ps); len(etc) != 1 || !etc[0].CharmOfCourage {
		t.Fatalf("charm: %+v", etc)
	}
	ps = useItemOn(e, sess, charm)
	if txt := textsOf(ps); len(txt) != 1 || !strings.Contains(txt[0], "20 minute(s), 0 second(s) remaining in Charm of Courage: No Grade's re-use time") {
		t.Errorf("second charm: %q", txt)
	}
	if n := countOf(e, p, itemCharmNG); n != 1 {
		t.Errorf("%d charms left, want 1", n)
	}

	e.advance(time.Minute)
	c := p.char
	e.srv.Restart(sess)
	saved := e.store.waitSave(t, c.ObjectID)
	if len(saved.ItemReuse) != 1 || saved.ItemReuse[0].ItemID != itemCharmNG || saved.ItemReuse[0].ReuseDelay != 1200000 {
		t.Fatalf("saved item reuse %+v", saved.ItemReuse)
	}
	sess2 := newSession("charm")
	p2 := e.enter(sess2, 0)
	var left time.Duration
	e.onLoop(func() { left = e.srv.ItemReuseLeft(p2, objectOf2(p2, itemCharmNG)) })
	if left != 19*time.Minute {
		t.Errorf("reuse left after the relog %v, want 19m", left)
	}
	sess2.take(t)
	if ps := useItemOn(e, sess2, objectOf(e, p2, itemCharmNG)); len(textsOf(ps)) != 1 {
		t.Errorf("after the relog: %v", testclient.Names(ps))
	}
	var kept bool
	e.onLoop(func() { kept = p2.CharmOfCourage() })
	if kept {
		t.Error("the charm survived the relog")
	}
}

// objectOf2 is objectOf on the loop.
func objectOf2(p *Player, itemID int32) int32 {
	if it := p.inv.ByItemID(itemID); it != nil {
		return it.ObjectID()
	}
	return 0
}

// A charged shot sits on the weapon item.
func TestChargedShotOnItem(t *testing.T) {
	tmpl, _ := testGameData(t).ItemTemplate(itemSquireSword)
	it := inventory.NewItem(1, tmpl, 1)
	it.SetChargedShot(uint8(skills.Soulshot), true)
	it.SetChargedShot(uint8(skills.BlessedSpiritshot), true)
	if !it.ChargedShot(uint8(skills.Soulshot)) || it.ChargedShot(uint8(skills.Spiritshot)) || !it.ChargedShot(uint8(skills.BlessedSpiritshot)) {
		t.Error("charges")
	}
	it.SetChargedShot(uint8(skills.Soulshot), false)
	if it.ChargedShot(uint8(skills.Soulshot)) || it.ChargedShot(200) {
		t.Error("discharge")
	}
}

func TestReuseText(t *testing.T) {
	for _, c := range []struct {
		d    time.Duration
		want string
	}{
		{9500 * time.Millisecond, "There are 9 second(s) remaining in X's re-use time."},
		{20 * time.Minute, "There are 20 minute(s), 0 second(s) remaining in X's re-use time."},
		{2*time.Hour + 3*time.Minute + 4*time.Second, "There are 2 hour(s), 3 minute(s), and 4 second(s) remaining in X's re-use time."},
	} {
		if got := reuseText(c.d, "X"); got != c.want {
			t.Errorf("%v: %q", c.d, got)
		}
	}
}
