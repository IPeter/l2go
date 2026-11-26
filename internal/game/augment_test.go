package game

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/augment"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// Items and NPCs of the augmentation tests (Mobius IDs).
const (
	itemStormbringer = 72   // C-grade sword, 916 crystals: removal costs 95,000
	itemCaliburs     = 75   // C-grade sword, 1720 crystals: removal costs 150,000
	itemLifeStone46  = 8723 // Life Stone: level 46, no grade
	itemLifeStone76  = 8732 // Life Stone: level 76, no grade
	itemGemstoneD    = 2130
	npcHelton        = 30678 // Blacksmith Helton (augmentation)

	optPDef1545   = 1     // P. Def. + 15.45
	optActive3188 = 14578 // active skill 3188 (a purple option of 8723)
	optPassive    = 14575 // passive skill 3252
	optCharm      = 14568 // attack trigger: Charm (3081), 1 %
	skillActive   = 3188
	skillPassive  = 3252
	skillCharm    = 3081
)

// augmentEnv starts a server with the data of data/ and enters a level 46
// fighter next to a blacksmith with a sword, life stones, gemstones and
// adena. It returns the session, the player and the blacksmith.
func augmentEnv(t *testing.T, level int32) (*testEnv, *fakeSession, *Player, *Npc) {
	t.Helper()
	e := newItemSkillEnv(t, testWorldData(t), nil)
	if e.srv.augment.tables == nil {
		t.Fatal("no augmentation data in data/augment")
	}
	sess, p := enterWithItems(t, e, "smith", level, tiVillage,
		storage.Item{ItemID: itemStormbringer, Count: 1},
		storage.Item{ItemID: itemCaliburs, Count: 1},
		storage.Item{ItemID: itemLifeStone46, Count: 2},
		storage.Item{ItemID: itemLifeStone76, Count: 1},
		storage.Item{ItemID: itemGemstoneD, Count: 45},
		storage.Item{ItemID: adenaID, Count: 200_000},
	)
	n := e.spawnNpc(npcHelton, eastOf(tiVillage, 60))
	e.onLoop(func() { p.LastNpc = n.ID() })
	return e, sess, p, n
}

func exOn(e *testEnv, sess *fakeSession, sub uint16, vs ...int32) []testclient.Packet {
	e.t.Helper()
	return handleExOn(e, sess, sub, ints(vs...))
}

// augmentationOf returns an item's augmentation, read on the loop.
func augmentationOf(e *testEnv, p *Player, objectID int32) augment.Augmentation {
	var a int32
	e.onLoop(func() {
		if it := p.inv.ByObjectID(objectID); it != nil {
			a = it.Augmentation()
		}
	})
	return augment.Augmentation(a)
}

// pdefAndSkill returns p's P. Def. and the level of a skill, on the loop.
func pdefAndSkill(e *testEnv, p *Player, skillID int) (float64, int) {
	var pdef float64
	var level int
	e.onLoop(func() {
		pdef = float64(p.stats.View().PDef)
		level = p.SkillLevel(skillID)
	})
	return pdef, level
}

// The demo of M11d: the blacksmith opens the window, the sword, the life
// stone and 20 Gemstone D go in, RequestRefine augments the sword with the
// scripted options; the bonus (P. Def. and an active skill) is on while the
// sword is worn and off when it comes off; removal costs 95,000 adena.
func TestAugmentRefineAndCancel(t *testing.T) {
	e, sess, p, n := augmentEnv(t, 46)
	sword := objectOf(e, p, itemStormbringer)
	stone := objectOf(e, p, itemLifeStone46)
	gems := objectOf(e, p, itemGemstoneD)

	// The Augment bypass of the blacksmith opens the windows.
	sess.take(t)
	e.onLoop(func() { e.srv.npcCommand(p, n, "Augment 1") })
	if ps := sess.take(t); len(only[testclient.ExShowVariationMakeWindow](ps)) != 1 {
		t.Fatalf("Augment 1: %q", testclient.Names(ps))
	}
	e.onLoop(func() { e.srv.npcCommand(p, n, "Augment 2") })
	if ps := sess.take(t); len(only[testclient.ExShowVariationCancelWindow](ps)) != 1 {
		t.Fatalf("Augment 2: %q", testclient.Names(ps))
	}

	// The sword is worn while it goes into the window. The P. Def. without
	// it and with it (a grade penalty may differ) are the baselines.
	offPDef, _ := pdefAndSkill(e, p, skillActive)
	useItemOn(e, sess, sword)
	basePDef, _ := pdefAndSkill(e, p, skillActive)

	ps := exOn(e, sess, clientpackets.ExOpRequestConfirmTargetItem, sword)
	if got := only[testclient.ExPutItemResultForVariationMake](ps); len(got) != 1 || got[0] != (testclient.ExPutItemResultForVariationMake{ObjectID: sword, ItemID: itemStormbringer, Result: 1}) {
		t.Fatalf("ConfirmTargetItem: %+v %q", got, testclient.Names(ps))
	}
	ps = exOn(e, sess, clientpackets.ExOpRequestConfirmRefinerItem, sword, stone)
	want := testclient.ExPutIntensiveResultForVariationMake{RefinerObjectID: stone, LifeStoneItemID: itemLifeStone46, GemstoneItemID: itemGemstoneD, GemstoneCount: 20, Result: 1}
	if got := only[testclient.ExPutIntensiveResultForVariationMake](ps); len(got) != 1 || got[0] != want {
		t.Fatalf("ConfirmRefinerItem: %+v %q", got, testclient.Names(ps))
	}
	ps = exOn(e, sess, clientpackets.ExOpRequestConfirmGemStone, sword, stone, gems, 19)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{augment.MsgGemstoneCount}) {
		t.Errorf("19 gemstones: messages %v, want %d", ids, augment.MsgGemstoneCount)
	}
	ps = exOn(e, sess, clientpackets.ExOpRequestConfirmGemStone, sword, stone, gems, 20)
	if got := only[testclient.ExPutCommissionResultForVariationMake](ps); len(got) != 1 || got[0] != (testclient.ExPutCommissionResultForVariationMake{GemstoneObjectID: gems, Count: 20, Result: 1}) {
		t.Fatalf("ConfirmGemStone: %+v %q", got, testclient.Names(ps))
	}

	// Rnd.get(10000) = 9999 takes the first low option (P. Def. + 15.45),
	// Rnd.get(100) = 90 the purple category, whose first option has an
	// active skill.
	e.onLoop(func() {
		e.srv.augment.rnd = rnd.NewScripted(t, rnd.Get(10000, 9999), rnd.Get(100, 90), rnd.Get(10000, 9999))
	})
	ps = exOn(e, sess, clientpackets.ExOpRequestRefine, sword, stone, gems, 20)
	if got := only[testclient.ExVariationResult](ps); len(got) != 1 || got[0] != (testclient.ExVariationResult{Stat12: optPDef1545, Stat34: optActive3188, Result: 1}) {
		t.Fatalf("Refine: %+v %q", got, testclient.Names(ps))
	}
	// The worn sword came off first: its InventoryUpdate at once, before the
	// result.
	names := testclient.Names(ps)
	if iu, res := slices.Index(names, "InventoryUpdate"), slices.Index(names, "ExVariationResult"); iu < 0 || iu > res {
		t.Errorf("Refine packets %q, want the unequip InventoryUpdate before the result", names)
	}
	if a := augmentationOf(e, p, sword); a != augment.Make(optPDef1545, optActive3188) {
		t.Errorf("augmentation %d (%d, %d)", a, a.Stat12(), a.Stat34())
	}
	if n1, n2 := countOf(e, p, itemLifeStone46), countOf(e, p, itemGemstoneD); n1 != 1 || n2 != 25 {
		t.Errorf("after refine: %d life stones, %d gemstones; want 1, 25", n1, n2)
	}
	if pdef, lvl := pdefAndSkill(e, p, skillActive); pdef != offPDef || lvl != 0 {
		t.Errorf("unworn after refine: P. Def. %v (without the sword %v), skill level %d", pdef, offPDef, lvl)
	}
	// A second augmentation is refused.
	ps = exOn(e, sess, clientpackets.ExOpRequestConfirmTargetItem, sword)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{augment.MsgAlreadyAugmented}) {
		t.Errorf("augmented sword again: messages %v", ids)
	}

	// Worn, the options apply; taken off, they go.
	ps = useItemOn(e, sess, sword)
	pdef, lvl := pdefAndSkill(e, p, skillActive)
	if d := pdef - basePDef; d < 15 || d > 16 || lvl != 1 {
		t.Errorf("worn: P. Def. +%v (want 15.45 rounded), skill %d level %d", d, skillActive, lvl)
	}
	// The SkillList follows 300 ms later (Mobius sendSkillList), with the
	// option's active skill.
	e.advance(400 * time.Millisecond)
	lists := only[testclient.SkillList](sess.take(t))
	if len(lists) != 1 || !slices.ContainsFunc(lists[0].Skills, func(sk testclient.Skill) bool { return sk.ID == skillActive && sk.Level == 1 }) {
		t.Errorf("equip: SkillList %+v, want one with skill %d", lists, skillActive)
	}
	wantAug := int32(augment.Make(optPDef1545, optActive3188))
	if uis := only[testclient.UserInfo](ps); len(uis) > 0 && uis[len(uis)-1].AugmentationID != wantAug {
		t.Errorf("UserInfo weapon augmentation %d, want %d", uis[len(uis)-1].AugmentationID, wantAug)
	}
	e.onLoop(func() {
		if got := p.weaponAugmentation(); got != wantAug {
			t.Errorf("weapon augmentation %d, want %d", got, wantAug)
		}
	})
	useItemOn(e, sess, sword)
	if pdef, lvl := pdefAndSkill(e, p, skillActive); pdef != offPDef || lvl != 0 {
		t.Errorf("taken off: P. Def. %v (without the sword %v), skill level %d", pdef, offPDef, lvl)
	}

	// Augmented items are not traded, sold or dropped (AltAllowAugmentTrade).
	e.onLoop(func() {
		it := p.inv.ByObjectID(sword)
		if !augmentLocked(it) || tradeAvailable(it) || it.Depositable(false) || !it.Depositable(true) {
			t.Error("an augmented sword can be traded or shared")
		}
	})

	// Removal: the fee by grade and crystals, then the augmentation goes.
	ps = exOn(e, sess, clientpackets.ExOpRequestConfirmCancelItem, sword)
	if got := only[testclient.ExPutItemResultForVariationCancel](ps); len(got) != 1 || got[0].ObjectID != sword || got[0].Price != 95000 {
		t.Fatalf("ConfirmCancelItem: %+v %q", got, testclient.Names(ps))
	}
	adena := countOf(e, p, adenaID)
	ps = exOn(e, sess, clientpackets.ExOpRequestRefineCancel, sword)
	if got := only[testclient.ExVariationCancelResult](ps); len(got) != 1 || got[0] != (testclient.ExVariationCancelResult{CloseWindow: 1, Result: 1}) {
		t.Fatalf("RefineCancel: %+v %q", got, testclient.Names(ps))
	}
	if a, left := augmentationOf(e, p, sword), countOf(e, p, adenaID); a != 0 || left != adena-95000 {
		t.Errorf("after removal: augmentation %d, adena %d (was %d)", a, left, adena)
	}
	if ids := sysMsgIDsOf(ps); !slices.Contains(ids, int32(sysMsgAdenaDisappeared)) {
		t.Errorf("removal messages %v, want the adena one", ids)
	}
	// Not augmented any more: removal refused.
	ps = exOn(e, sess, clientpackets.ExOpRequestRefineCancel, sword)
	if got := only[testclient.ExVariationCancelResult](ps); len(got) != 1 || got[0].Result != 0 || !slices.Contains(sysMsgIDsOf(ps), augment.MsgRemoveNotAugmented) {
		t.Errorf("second removal: %+v %v", got, sysMsgIDsOf(ps))
	}
}

// The checks of the windows in the game: the NPC of decision X10, the
// player state, the life stone level and the removal fee the player cannot
// pay.
func TestAugmentChecks(t *testing.T) {
	e, sess, p, n := augmentEnv(t, 46)
	sword := objectOf(e, p, itemStormbringer)
	stone76 := objectOf(e, p, itemLifeStone76)
	stone := objectOf(e, p, itemLifeStone46)
	gems := objectOf(e, p, itemGemstoneD)

	// A level 76 stone at level 46.
	ps := exOn(e, sess, clientpackets.ExOpRequestConfirmRefinerItem, sword, stone76)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{augment.MsgNotSuitable}) || len(only[testclient.ExPutIntensiveResultForVariationMake](ps)) != 0 {
		t.Errorf("level 76 stone: %v %q", ids, testclient.Names(ps))
	}
	// Sitting: the state's message, then the step's own (as Mobius sends).
	e.onLoop(func() { p.Sitting = true })
	ps = exOn(e, sess, clientpackets.ExOpRequestConfirmTargetItem, sword)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{augment.MsgSitting, augment.MsgNotSuitable}) {
		t.Errorf("sitting: messages %v", ids)
	}
	ps = exOn(e, sess, clientpackets.ExOpRequestRefine, sword, stone, gems, 20)
	if got := only[testclient.ExVariationResult](ps); len(got) != 1 || got[0].Result != 0 || !slices.Equal(sysMsgIDsOf(ps), []int32{augment.MsgSitting, augment.MsgInappropriateRefine}) {
		t.Errorf("refine sitting: %+v %v", got, sysMsgIDsOf(ps))
	}
	e.onLoop(func() { p.Sitting = false })

	// Decision X10: the last NPC must be a blacksmith nearby.
	far := e.spawnNpc(npcHelton, eastOf(tiVillage, 2000))
	merchant := e.spawnNpc(npcLector, eastOf(tiVillage, 40))
	for _, last := range []*Npc{far, merchant} {
		e.onLoop(func() { p.LastNpc = last.ID() })
		ps = exOn(e, sess, clientpackets.ExOpRequestConfirmTargetItem, sword)
		if len(only[testclient.ExPutItemResultForVariationMake](ps)) != 0 {
			t.Errorf("NPC %d at %v: the sword was accepted", last.Template.ID, last.Position())
		}
		ps = exOn(e, sess, clientpackets.ExOpRequestRefine, sword, stone, gems, 20)
		if got := only[testclient.ExVariationResult](ps); len(got) != 1 || got[0].Result != 0 {
			t.Errorf("NPC %d: refine %+v", last.Template.ID, got)
		}
	}
	if a := augmentationOf(e, p, sword); a != 0 {
		t.Fatalf("augmented without a blacksmith: %d", a)
	}
	e.onLoop(func() { p.LastNpc = n.ID() })

	// A removal the player cannot pay: Caliburs costs 150,000 of 200,000;
	// after it, 50,000 do not pay for another.
	cal := objectOf(e, p, itemCaliburs)
	e.onLoop(func() { e.srv.gmAugment(p, []string{"75", "1"}) }) // only worn items
	if a := augmentationOf(e, p, cal); a != 0 {
		t.Errorf("//augment changed an unworn item: %d", a)
	}
	useItemOn(e, sess, cal)
	e.onLoop(func() { e.srv.gmAugment(p, []string{"75", strconv.Itoa(int(augment.Make(optPDef1545, optPassive)))}) })
	useItemOn(e, sess, cal) // off again
	ps = exOn(e, sess, clientpackets.ExOpRequestConfirmCancelItem, cal)
	if got := only[testclient.ExPutItemResultForVariationCancel](ps); len(got) != 1 || got[0].Price != 150000 {
		t.Fatalf("Caliburs fee: %+v", got)
	}
	exOn(e, sess, clientpackets.ExOpRequestRefineCancel, cal)
	useItemOn(e, sess, cal)
	e.onLoop(func() { e.srv.gmAugment(p, []string{"75", strconv.Itoa(int(augment.Make(optPDef1545, optPassive)))}) })
	useItemOn(e, sess, cal)
	ps = exOn(e, sess, clientpackets.ExOpRequestRefineCancel, cal)
	if got := only[testclient.ExVariationCancelResult](ps); len(got) != 1 || got[0].Result != 0 || augmentationOf(e, p, cal) == 0 {
		t.Errorf("removal without the fee: %+v, augmentation %d", got, augmentationOf(e, p, cal))
	}
}

// The augmentation is saved with the item; after a relog the worn item's
// bonus is on again, and the item packets show it.
func TestAugmentPersistence(t *testing.T) {
	e, sess, p, _ := augmentEnv(t, 46)
	sword := objectOf(e, p, itemStormbringer)
	useItemOn(e, sess, sword)
	basePDef, _ := pdefAndSkill(e, p, skillActive)
	aug := augment.Make(optPDef1545, optActive3188)
	e.onLoop(func() { e.srv.gmAugment(p, []string{"72", strconv.Itoa(int(aug))}) })
	if pdef, lvl := pdefAndSkill(e, p, skillActive); pdef-basePDef < 15 || lvl != 1 {
		t.Fatalf("//augment on a worn sword: P. Def. +%v, skill level %d", pdef-basePDef, lvl)
	}

	c := p.char
	e.srv.Restart(sess)
	saved := e.store.waitSave(t, c.ObjectID)
	i := slices.IndexFunc(saved.Items, func(it storage.Item) bool { return it.ObjectID == sword })
	if i < 0 || saved.Items[i].Augmentation != int32(aug) || saved.Items[i].Slot != storage.PaperdollRHand {
		t.Fatalf("saved sword: %+v", saved.Items)
	}

	sess2 := newSession("smith")
	p2 := e.enter(sess2, 0)
	ps := sess2.take(t)
	lists := only[testclient.ItemList](ps)
	if len(lists) == 0 {
		t.Fatalf("no ItemList at enter: %q", testclient.Names(ps))
	}
	j := slices.IndexFunc(lists[0].Items, func(it testclient.Item) bool { return it.ObjectID == sword })
	if j < 0 || lists[0].Items[j].AugmentationID != int32(aug) {
		t.Errorf("ItemList sword: %+v", lists[0].Items)
	}
	if pdef, lvl := pdefAndSkill(e, p2, skillActive); pdef-basePDef < 15 || lvl != 1 {
		t.Errorf("after relog: P. Def. +%v, skill level %d", pdef-basePDef, lvl)
	}

	// A class change takes the bonuses off (Mobius setActiveClass).
	e.onLoop(func() { e.srv.RemoveAugmentBonuses(p2) })
	if pdef, lvl := pdefAndSkill(e, p2, skillActive); pdef != basePDef || lvl != 0 {
		t.Errorf("RemoveAugmentBonuses: P. Def. %v (base %v), skill level %d", pdef, basePDef, lvl)
	}
	e.onLoop(func() { e.srv.ApplyAugmentBonuses(p2) })
	if pdef, lvl := pdefAndSkill(e, p2, skillActive); pdef-basePDef < 15 || lvl != 1 {
		t.Errorf("ApplyAugmentBonuses: P. Def. +%v, skill level %d", pdef-basePDef, lvl)
	}
}

// Passive and trigger options: the passive skill comes and goes with the
// item; a trigger is tried on hits of its kind only, with Rnd.get(100).
func TestAugmentPassiveAndTriggers(t *testing.T) {
	e, sess, p, n := augmentEnv(t, 46)
	sword := objectOf(e, p, itemStormbringer)
	useItemOn(e, sess, sword)
	e.onLoop(func() {
		e.srv.gmAugment(p, []string{"72", strconv.Itoa(int(augment.Make(optPassive, optCharm)))})
	})
	if _, lvl := pdefAndSkill(e, p, skillPassive); lvl != 1 {
		t.Errorf("passive skill level %d, want 1", lvl)
	}
	var triggers []int
	e.onLoop(func() {
		for _, tr := range e.srv.sortedTriggers(p) {
			triggers = append(triggers, tr.skill.ID)
		}
	})
	if !slices.Equal(triggers, []int{skillCharm}) {
		t.Fatalf("triggers %v, want [%d]", triggers, skillCharm)
	}
	// A critical hit does not try an attack trigger: no draw.
	e.onLoop(func() {
		e.srv.attack.rnd = rnd.NewScripted(t)
		e.srv.augmentHitTriggers(p, n, true)
	})
	// A normal hit draws Rnd.get(100); 99 is not below 1.
	e.onLoop(func() {
		e.srv.attack.rnd = rnd.NewScripted(t, rnd.Get(100, 99))
		e.srv.augmentHitTriggers(p, n, false)
	})

	useItemOn(e, sess, sword)
	if _, lvl := pdefAndSkill(e, p, skillPassive); lvl != 0 {
		t.Errorf("unworn: passive skill level %d", lvl)
	}
	e.onLoop(func() {
		if len(e.srv.sortedTriggers(p)) != 0 {
			t.Error("unworn: triggers left")
		}
	})
}
