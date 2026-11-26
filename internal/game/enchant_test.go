package game

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/enchant"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// Items of the enchant tests (Mobius item IDs).
const (
	enchSaber          = 123  // Saber: D, rhand, 743 crystals
	enchSpiritSword    = 131  // Spirit Sword: C weapon
	enchCompositeArmor = 60   // Composite Armor: C full armor, 576 crystals
	enchNecklaceOfSeal = 119  // Necklace of Seal: C accessory
	enchHeroWeapon     = 6611 // Infinity Blade: a hero weapon
	enchScrollWeaponD  = 955  // Scroll: Enchant Weapon (Grade D)
	enchScrollArmorC   = 952  // Scroll: Enchant Armor (Grade C)
	enchBlessedWeaponD = 6575 // Blessed Scroll: Enchant Weapon (Grade D)
	enchCrystalWeaponD = 957  // Crystal Scroll: Enchant Weapon (Grade D)
	enchCrystalD       = 1458 // Crystal: D-Grade
	enchCrystalC       = 1459 // Crystal: C-Grade
)

// newEnchantEnv is an item test server with the enchant data of data/ and
// no flood limit, unless configure sets one.
func newEnchantEnv(t *testing.T, configure func(gs *config.GameServer)) *itemEnv {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "data", filepath.FromSlash(enchant.File)))
	if err != nil {
		t.Fatalf("read the enchant data: %v", err)
	}
	return newItemEnv(t, func(gs *config.GameServer) {
		writeDialogFile(t, filepath.Join(gs.DataDir, filepath.FromSlash(enchant.File)), string(b))
		gs.Enchant.FloodEnchantItem = 0
		if configure != nil {
			configure(gs)
		}
	})
}

// enchanted is an item of a character's inventory at an enchant level.
func enchanted(itemID, level int32) storage.Item {
	return storage.Item{ItemID: itemID, Count: 1, Enchant: level}
}

// script makes the enchant rolls come from draws.
func (e *itemEnv) script(draws ...rnd.Draw) {
	e.onLoop(func() { e.srv.enchant.rnd = rnd.NewScripted(e.t, draws...) })
}

// enchantWith uses the scroll and sends RequestEnchantItem for the item; it
// returns the packets of the enchant.
func (e *itemEnv) enchantWith(sess *fakeSession, scroll, item *inventory.Item) []testclient.Packet {
	e.t.Helper()
	e.useItem(sess, scroll.ObjectID())
	return e.handle(sess, clientpackets.OpRequestEnchantItem, ints(item.ObjectID()))
}

// enchantOf returns the enchant level of p's item with the object ID, -1
// when p does not have it.
func (e *itemEnv) enchantOf(p *Player, objectID int32) int32 {
	lvl := int32(-1)
	e.onLoop(func() {
		if it := p.inv.ByObjectID(objectID); it != nil {
			lvl = it.Enchant()
		}
	})
	return lvl
}

// enchantResults returns the EnchantResult values among ps.
func enchantResults(ps []testclient.Packet) []int32 {
	var out []int32
	for _, r := range only[testclient.EnchantResult](ps) {
		out = append(out, r.Result)
	}
	return out
}

// itemChanges returns the changes of every InventoryUpdate among ps, one
// slice per packet, as "change:itemID+enchant".
func itemChanges(ps []testclient.Packet) [][]string {
	var out [][]string
	for _, u := range only[testclient.InventoryUpdate](ps) {
		var l []string
		for _, c := range u.Items {
			change := [...]string{"", "added", "modified", "removed"}[c.Change]
			l = append(l, change+":"+strconv.Itoa(int(c.ItemID))+"+"+strconv.Itoa(int(c.Enchant)))
		}
		out = append(out, l)
	}
	return out
}

// TestEnchantScrollUse: a scroll opens the enchant window.
func TestEnchantScrollUse(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart, enchanted(enchScrollWeaponD, 0))
	scroll := e.itemOf(p, enchScrollWeaponD)
	ps := e.useItem(sess, scroll.ObjectID())
	if got := testclient.Names(ps); !slices.Equal(got, []string{"ChooseInventoryItem", "SystemMessage"}) {
		t.Fatalf("packets %v", got)
	}
	if c := only[testclient.ChooseInventoryItem](ps)[0]; c.ItemID != enchScrollWeaponD {
		t.Errorf("ChooseInventoryItem %+v", c)
	}
	if ids := sysMsgIDsOf(ps); ids[0] != int32(sysMsgSelectItemToEnchant) {
		t.Errorf("message %v", ids)
	}
}

// TestEnchantSuccess: +0 to +1 with a draw that wins: "Your $s1 has been
// successfully enchanted.", EnchantResult 1, then one InventoryUpdate with
// the scroll gone and the sword at +1, then UserInfo at the end of the
// tick.
func TestEnchantSuccess(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart, enchanted(enchScrollWeaponD, 0), enchanted(enchSaber, 0))
	scroll, sword := e.itemOf(p, enchScrollWeaponD), e.itemOf(p, enchSaber)
	e.script(rnd.Float64(0.99))
	ps := e.enchantWith(sess, scroll, sword)
	if got := testclient.Names(ps); !slices.Equal(got, []string{"SystemMessage", "EnchantResult", "InventoryUpdate"}) {
		t.Fatalf("packets %v", got)
	}
	e.advance(200 * time.Millisecond)
	if got := testclient.Names(sess.take(t)); !slices.Contains(got, "UserInfo") {
		t.Errorf("no UserInfo at the end of the tick: %v", got)
	}
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgEnchanted)}) {
		t.Errorf("messages %v", ids)
	}
	if r := enchantResults(ps); !slices.Equal(r, []int32{1}) {
		t.Errorf("EnchantResult %v, want [1]", r)
	}
	if ch := itemChanges(ps); !slices.Equal(ch[0], []string{"removed:955+0", "modified:123+1"}) {
		t.Errorf("InventoryUpdate %v", ch)
	}
	if e.enchantOf(p, sword.ObjectID()) != 1 || e.countOf(p, enchScrollWeaponD) != 0 {
		t.Error("the sword is not +1, or the scroll is still there")
	}
	// The scroll is no longer active: a second request is cancelled.
	ps = e.handle(sess, clientpackets.OpRequestEnchantItem, ints(sword.ObjectID()))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgEnchantCancelled)}) || !slices.Equal(enchantResults(ps), []int32{0}) {
		t.Errorf("second request: %v, %v", testclient.Names(ps), ids)
	}
}

// TestEnchantWornSuccess: a worn sword +3 to +4 adds its new stats (the same
// as putting it on again) and the ItemList follows 250 ms later.
func TestEnchantWornSuccess(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart, enchanted(enchScrollWeaponD, 0), enchanted(enchSaber, 3))
	scroll, sword := e.itemOf(p, enchScrollWeaponD), e.itemOf(p, enchSaber)
	e.useItem(sess, sword.ObjectID())
	if e.wornIn(p, storage.PaperdollRHand) != enchSaber {
		t.Fatal("the sword is not worn")
	}
	var before int32
	e.onLoop(func() { before = p.stats.View().PAtk })
	e.script(rnd.Float64(0.65))
	ps := e.enchantWith(sess, scroll, sword)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgEnchantedLevel)}) {
		t.Errorf("messages %v", ids)
	}
	if m := only[testclient.SystemMessage](ps)[0]; m.Params[0].Number != 3 {
		t.Errorf("the message names +%d, want +3", m.Params[0].Number)
	}
	if r := enchantResults(ps); !slices.Equal(r, []int32{4}) {
		t.Errorf("EnchantResult %v", r)
	}
	var after, again int32
	e.onLoop(func() {
		after = p.stats.View().PAtk
		p.inv.UnequipItem(sword)
		p.inv.Equip(sword, inventory.Wearer{})
		again = p.stats.View().PAtk
	})
	if after <= before || after != again {
		t.Errorf("P.Atk %d before, %d after, %d when put on again", before, after, again)
	}
	e.advance(300 * time.Millisecond)
	if lists := only[testclient.ItemList](sess.take(t)); len(lists) != 1 || lists[0].ShowWindow {
		t.Errorf("ItemList after enchanting a worn item: %+v", lists)
	}
}

// TestEnchantFailureCrystallizes: a normal scroll fails at +5: the sword is
// crystallized into D crystals, 743 + 90 × 7 − 372 = 1001.
func TestEnchantFailureCrystallizes(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart, enchanted(enchScrollWeaponD, 0), enchanted(enchSaber, 5))
	scroll, sword := e.itemOf(p, enchScrollWeaponD), e.itemOf(p, enchSaber)
	e.script(rnd.Float64(0.66))
	ps := e.enchantWith(sess, scroll, sword)
	want := []string{"SystemMessage", "StatusUpdate", "SystemMessage", "EnchantResult", "InventoryUpdate", "InventoryUpdate"}
	if got := testclient.Names(ps); !slices.Equal(got, want) {
		t.Fatalf("packets %v, want %v", got, want)
	}
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgEnchantCrystallizedLevel), int32(sysMsgEarnedS2S1)}) {
		t.Errorf("messages %v", ids)
	}
	earned := only[testclient.SystemMessage](ps)[1]
	if earned.Params[0].Number != enchCrystalD || earned.Params[1].Number != 1001 {
		t.Errorf("earned %+v, want 1001 D crystals", earned.Params)
	}
	if r := enchantResults(ps); !slices.Equal(r, []int32{0}) {
		t.Errorf("EnchantResult %v", r)
	}
	ch := itemChanges(ps)
	if !slices.Equal(ch[0], []string{"added:1458+0"}) || !slices.Equal(ch[1], []string{"removed:955+0", "removed:123+5", "added:1458+0"}) {
		t.Errorf("InventoryUpdates %v", ch)
	}
	if e.enchantOf(p, sword.ObjectID()) != -1 || e.countOf(p, enchCrystalD) != 1001 {
		t.Error("the sword is still there, or the crystals are wrong")
	}
}

// TestEnchantWornFailure: a worn sword comes off first, in its own
// InventoryUpdate sent at once, then it is crystallized.
func TestEnchantWornFailure(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart, enchanted(enchScrollWeaponD, 0), enchanted(enchSaber, 4), storage.Item{ItemID: enchCrystalD, Count: 10})
	scroll, sword := e.itemOf(p, enchScrollWeaponD), e.itemOf(p, enchSaber)
	e.useItem(sess, sword.ObjectID())
	e.advance(time.Second)
	sess.take(t)
	e.script(rnd.Float64(0.9))
	ps := e.enchantWith(sess, scroll, sword)
	// The removal message and its InventoryUpdate come before the
	// crystallization message (the grade penalty may change in between).
	names := testclient.Names(ps)
	firstIU := slices.Index(names, "InventoryUpdate")
	if names[0] != "SystemMessage" || firstIU < 0 || slices.Index(names[1:], "SystemMessage")+1 < firstIU {
		t.Fatalf("packets %v: the removal and its InventoryUpdate first", names)
	}
	if ids := sysMsgIDsOf(ps); ids[0] != int32(sysMsgEquipmentRemoved) || ids[1] != int32(sysMsgEnchantCrystallizedLevel) {
		t.Errorf("messages %v", ids)
	}
	ch := itemChanges(ps)
	if !slices.Contains(ch[0], "modified:123+4") {
		t.Errorf("the unequip InventoryUpdate %v", ch[0])
	}
	// The crystals go onto the stack: 743 + 90 × 5 − 372 = 821.
	if got := e.countOf(p, enchCrystalD); got != 831 {
		t.Errorf("%d crystals, want 10 + 821", got)
	}
	if e.wornIn(p, storage.PaperdollRHand) != 0 {
		t.Error("something is still worn in the right hand")
	}
}

// TestEnchantBlessedAndCrystalScrolls: a failed blessed scroll resets the
// item to +0 and keeps it; a crystal scroll succeeds at +16 without a gain;
// a normal scroll is refused at +16.
func TestEnchantBlessedAndCrystalScrolls(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart,
		enchanted(enchBlessedWeaponD, 0), enchanted(enchCrystalWeaponD, 0), enchanted(enchScrollWeaponD, 0),
		enchanted(enchSaber, 6), enchanted(enchSpiritSword, 0))
	sword := e.itemOf(p, enchSaber)

	e.script(rnd.Float64(0.7))
	ps := e.enchantWith(sess, e.itemOf(p, enchBlessedWeaponD), sword)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgBlessedEnchantFailed)}) {
		t.Errorf("blessed failure messages %v", ids)
	}
	if ch := itemChanges(ps); !slices.Equal(ch[0], []string{"removed:6575+0", "modified:123+0"}) {
		t.Errorf("blessed failure InventoryUpdate %v", ch)
	}
	if e.enchantOf(p, sword.ObjectID()) != 0 {
		t.Error("the blessed failure did not reset the sword")
	}

	// A crystal scroll on a +16 sword: success, no gain.
	e.onLoop(func() { sword.SetEnchant(16); e.srv.ItemsChanged(p) })
	e.advance(time.Second)
	sess.take(t)
	e.script(rnd.Float64(0.999))
	ps = e.enchantWith(sess, e.itemOf(p, enchCrystalWeaponD), sword)
	if r := enchantResults(ps); !slices.Equal(r, []int32{16}) {
		t.Errorf("crystal scroll at +16: EnchantResult %v, want [16]", r)
	}
	if ch := itemChanges(ps); !slices.Equal(ch[0], []string{"removed:957+0"}) {
		t.Errorf("crystal scroll at +16: InventoryUpdate %v, want the scroll only", ch)
	}

	// A normal scroll is refused at +16 (its maximum), and keeps the scroll.
	e.script()
	ps = e.enchantWith(sess, e.itemOf(p, enchScrollWeaponD), sword)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgInappropriateEnchant)}) || e.countOf(p, enchScrollWeaponD) != 1 {
		t.Errorf("normal scroll at +16: %v", ids)
	}
	// And on a C weapon (wrong grade).
	ps = e.enchantWith(sess, e.itemOf(p, enchScrollWeaponD), e.itemOf(p, enchSpiritSword))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgInappropriateEnchant)}) || e.countOf(p, enchScrollWeaponD) != 1 {
		t.Errorf("wrong grade: %v", ids)
	}
}

// TestEnchantArmorScrolls: armor scrolls take armor and accessories of
// their grade, a full armor is safe up to +3.
func TestEnchantArmorScrolls(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart,
		enchanted(enchScrollArmorC, 0), enchanted(enchScrollArmorC, 0), enchanted(enchScrollArmorC, 0),
		enchanted(enchCompositeArmor, 3), enchanted(enchNecklaceOfSeal, 3), enchanted(enchSaber, 0))
	armor, neck := e.itemOf(p, enchCompositeArmor), e.itemOf(p, enchNecklaceOfSeal)
	scrolls := func() []*inventory.Item {
		var out []*inventory.Item
		e.onLoop(func() { out = p.inv.AllByItemID(enchScrollArmorC) })
		return out
	}
	// Full armor at +3: 100 %, the highest draw wins.
	e.script(rnd.Float64(0.999999))
	if r := enchantResults(e.enchantWith(sess, scrolls()[0], armor)); !slices.Equal(r, []int32{4}) {
		t.Errorf("full armor +3: %v, want [4]", r)
	}
	// An accessory at +3: 66 %.
	e.script(rnd.Float64(0.66))
	if r := enchantResults(e.enchantWith(sess, scrolls()[0], neck)); !slices.Equal(r, []int32{0}) {
		t.Errorf("accessory +3 with 0.66: %v, want [0]", r)
	}
	// An armor scroll does not take a weapon.
	e.script()
	ps := e.enchantWith(sess, scrolls()[0], e.itemOf(p, enchSaber))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgInappropriateEnchant)}) {
		t.Errorf("armor scroll on a weapon: %v", ids)
	}
}

// TestEnchantSeededRolls enchants +3 swords 300 times with a seeded source
// and checks each outcome against the rules rolled on a twin source: the
// game draws exactly the one Float64 of EnchantScroll.calculateSuccess, and
// about 66 % succeed.
func TestEnchantSeededRolls(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart)
	const tries, seed = 300, 42
	e.onLoop(func() { e.srv.enchant.rnd = rnd.New(seed) })
	twin := rnd.New(seed)
	var tbl *enchant.Table
	e.onLoop(func() { tbl = e.srv.enchant.table })
	won := 0
	for i := range tries {
		var scroll, sword *inventory.Item
		e.onLoop(func() {
			scroll, _ = p.inv.Create(enchScrollWeaponD, 1)
			sword, _ = p.inv.Create(enchSaber, 1)
			sword.SetEnchant(3)
			e.srv.ItemsChanged(p)
		})
		want := tbl.Roll(tbl.Scroll(enchScrollWeaponD), enchant.Target{Template: sword.Template(), Enchant: 3, Carried: true}, twin)
		r := enchantResults(e.enchantWith(sess, scroll, sword))
		got := enchant.ResultFailure
		if slices.Equal(r, []int32{4}) {
			got = enchant.ResultSuccess
			won++
		}
		if got != want {
			t.Fatalf("try %d: %v, the rules say %v", i, got, want)
		}
		e.onLoop(func() {
			if it := p.inv.ByObjectID(sword.ObjectID()); it != nil {
				_, _ = p.inv.Destroy(it, 1)
			}
			if c := p.inv.ByItemID(enchCrystalD); c != nil {
				_, _ = p.inv.Destroy(c, c.Count())
			}
			e.srv.ItemsChanged(p)
		})
	}
	if won < tries*56/100 || won > tries*76/100 {
		t.Errorf("%d of %d succeeded, want about 66 %%", won, tries)
	}
}

// TestEnchantAnnouncement: a +7 weapon is announced to the players around
// (not to the enchanter, as Mobius's broadcastMessage), with the firework
// for everyone.
func TestEnchantAnnouncement(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sa, a := e.enterWith("hero", tiVillage, enchanted(enchScrollWeaponD, 0), enchanted(enchSaber, 6))
	sb, _ := e.enterWith("pal", eastOf(tiVillage, 100))
	sa.take(t)
	sb.take(t)
	e.script(rnd.Float64(0.1))
	pa := e.enchantWith(sa, e.itemOf(a, enchScrollWeaponD), e.itemOf(a, enchSaber))
	pb := sb.take(t)
	const text = "hero has successfully enchanted a +7 Saber."
	if slices.Contains(messages(pa), text) {
		t.Error("the enchanter got the announcement")
	}
	if !slices.Contains(messages(pb), text) {
		t.Errorf("the other player's messages %q", messages(pb))
	}
	for who, ps := range map[string][]testclient.Packet{"enchanter": pa, "other": pb} {
		uses := only[testclient.MagicSkillUse](ps)
		if len(uses) != 1 || uses[0].SkillID != fireworkSkill || uses[0].CasterID != a.ObjectID() {
			t.Errorf("%s: firework %+v", who, uses)
		}
	}
	// +8 is not announced.
	e.onLoop(func() { _, _ = a.inv.Create(enchScrollWeaponD, 1); e.srv.ItemsChanged(a) })
	e.script(rnd.Float64(0.1))
	e.enchantWith(sa, e.itemOf(a, enchScrollWeaponD), e.itemOf(a, enchSaber))
	if pb := sb.take(t); len(only[testclient.MagicSkillUse](pb)) != 0 {
		t.Error("+8 was announced")
	}
}

// TestEnchantRefusals: no active scroll, a store, and the flood limit.
func TestEnchantRefusals(t *testing.T) {
	e := newEnchantEnv(t, func(gs *config.GameServer) { gs.Enchant.FloodEnchantItem = config.Duration(1500 * time.Millisecond) })
	sess, p := e.enterWith("hero", tiStart, enchanted(enchScrollWeaponD, 0), enchanted(enchSaber, 0))
	scroll, sword := e.itemOf(p, enchScrollWeaponD), e.itemOf(p, enchSaber)

	// Without a scroll used first.
	ps := e.handle(sess, clientpackets.OpRequestEnchantItem, ints(sword.ObjectID()))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgEnchantCancelled)}) {
		t.Errorf("no scroll: %v", ids)
	}
	// A second request within 1.5 s is ignored.
	e.useItem(sess, scroll.ObjectID())
	if ps := e.handle(sess, clientpackets.OpRequestEnchantItem, ints(sword.ObjectID())); len(ps) != 0 {
		t.Errorf("flooded request answered with %v", testclient.Names(ps))
	}
	e.advance(1500 * time.Millisecond)
	// In a private store.
	e.onLoop(func() { p.store.typ = StoreSell })
	ps = e.handle(sess, clientpackets.OpRequestEnchantItem, ints(sword.ObjectID()))
	e.onLoop(func() { p.store.typ = StoreNone })
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgNoEnchantInStore)}) || !slices.Equal(enchantResults(ps), []int32{0}) {
		t.Errorf("store: %v", ids)
	}
	if e.countOf(p, enchScrollWeaponD) != 1 || e.enchantOf(p, sword.ObjectID()) != 0 {
		t.Error("a refused enchant used the scroll or changed the sword")
	}
}

// TestEnchantBlackList: an item of EnchantBlackList cannot be enchanted,
// and //create_item gives it no level.
func TestEnchantBlackList(t *testing.T) {
	e := newEnchantEnv(t, func(gs *config.GameServer) { gs.Enchant.EnchantBlackList = []int{enchSaber} })
	sess, p := e.enterWith("hero", tiStart, enchanted(enchScrollWeaponD, 0), enchanted(enchSaber, 0))
	e.script()
	ps := e.enchantWith(sess, e.itemOf(p, enchScrollWeaponD), e.itemOf(p, enchSaber))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgInappropriateEnchant)}) || e.countOf(p, enchScrollWeaponD) != 1 {
		t.Errorf("black-listed item: %v", ids)
	}
	var lvl int32
	var ok bool
	e.onLoop(func() {
		tmpl, _ := e.srv.data.ItemTemplate(enchSaber)
		lvl, ok = e.srv.createItemEnchant(tmpl, 5)
	})
	if ok || lvl != 0 {
		t.Errorf("createItemEnchant of a black-listed item = %d, %v", lvl, ok)
	}
}

// TestEnchantWarehouseCancels: a warehouse request with a scroll active
// cancels the enchant (Mobius SendWareHouseDepositList).
func TestEnchantWarehouseCancels(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterChar(testChar{account: "gm", name: "Gm", pos: tiStart, access: storage.AccessLevelAdmin, adena: 1000,
		items: []storage.Item{enchanted(enchScrollWeaponD, 0)}})
	e.useItem(sess, e.itemOf(p, enchScrollWeaponD).ObjectID())
	e.onLoop(func() { e.srv.warehouse.active[p] = true })
	adena := e.itemOf(p, adenaID)
	ps := e.handle(sess, clientpackets.OpSendWareHouseDepositList, ints(1, adena.ObjectID(), 10))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgEnchantCancelled)}) || !slices.Equal(enchantResults(ps), []int32{0}) {
		t.Errorf("deposit with a scroll active: %v", testclient.Names(ps))
	}
	if e.countOf(p, adenaID) != 1000 {
		t.Error("the deposit went on")
	}
}

// TestOverEnchantEnter: at enter world a player's equipment above +16 is
// destroyed and the player is told and kicked 5 s later; a GM keeps it.
func TestOverEnchantEnter(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart, enchanted(enchSaber, 17), enchanted(enchCompositeArmor, 16), enchanted(enchNecklaceOfSeal, 20))
	if e.countOf(p, enchSaber) != 0 || e.countOf(p, enchNecklaceOfSeal) != 0 || e.countOf(p, enchCompositeArmor) != 1 {
		t.Errorf("after entering: %d swords, %d necklaces, %d armors; want 0, 0, 1",
			e.countOf(p, enchSaber), e.countOf(p, enchNecklaceOfSeal), e.countOf(p, enchCompositeArmor))
	}
	e.advance(6 * time.Second)
	if _, _, closed := sess.closedWith(t); !closed {
		t.Error("the over-enchanting player was not kicked")
	}

	gsess, gm := e.enterChar(testChar{account: "gm", name: "Gm", pos: tiStart, access: storage.AccessLevelAdmin,
		items: []storage.Item{enchanted(enchSaber, 30)}})
	if e.countOf(gm, enchSaber) != 1 {
		t.Error("the GM's over-enchanted sword was destroyed")
	}
	e.advance(6 * time.Second)
	if _, _, closed := gsess.closedWith(t); closed {
		t.Error("the GM was kicked")
	}
}

// TestOverEnchantEnterMessages: the messages, and NONE destroys without a
// word or a kick.
func TestOverEnchantEnterMessages(t *testing.T) {
	for _, punish := range []string{"JAIL", "NONE", "BROADCAST"} {
		e := newEnchantEnv(t, func(gs *config.GameServer) { gs.Enchant.OverEnchantPunishment = punish })
		ch := e.createCharacter("hero", "hero", classHumanFighter, 0)
		ids, err := e.ids.NextIDs(e.ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		ch.Items = append(ch.Items, storage.Item{ObjectID: ids[0], ItemID: enchSaber, Count: 1, Enchant: 17, Slot: storage.PaperdollNone, Mana: storage.NoMana})
		if err := e.store.Update(e.ctx, &ch); err != nil {
			t.Fatal(err)
		}
		sess := newSession("hero")
		p := e.enter(sess, 0)
		ps := sess.take(t)
		msgs := messages(ps)
		screens := only[testclient.ExShowScreenMessage](ps)
		told := slices.Contains(msgs, "[Server]: You have over-enchanted items!") && slices.Contains(msgs, "[Server]: Respect our server rules.") && len(screens) == 1
		if (punish == "NONE") == told {
			t.Errorf("%s: messages %q, %d screen messages", punish, msgs, len(screens))
		}
		if kicked := slices.Contains(msgs, "You will be kicked for illegal action, GM informed."); kicked != (punish == "JAIL") {
			t.Errorf("%s: kick message %v", punish, kicked)
		}
		if e.countOf(p, enchSaber) != 0 {
			t.Errorf("%s: the sword was not destroyed", punish)
		}
	}
}

// TestOverEnchantOnUse: putting on an over-enchanted item destroys it.
func TestOverEnchantOnUse(t *testing.T) {
	e := newEnchantEnv(t, func(gs *config.GameServer) { gs.Enchant.OverEnchantPunishment = "NONE" })
	sess, p := e.enterWith("hero", tiStart, enchanted(enchSaber, 0))
	sword := e.itemOf(p, enchSaber)
	e.onLoop(func() { sword.SetEnchant(25); e.srv.ItemsChanged(p) })
	e.advance(time.Second)
	sess.take(t)
	e.useItem(sess, sword.ObjectID())
	if e.countOf(p, enchSaber) != 0 || e.wornIn(p, storage.PaperdollRHand) == enchSaber {
		t.Error("the over-enchanted sword was put on")
	}
	e.advance(time.Second)
	removed := false
	for _, c := range itemChanges(sess.take(t)) {
		removed = removed || slices.Contains(c, "removed:123+25")
	}
	if !removed {
		t.Error("no InventoryUpdate removed the sword")
	}
}

// TestGMEnchantCaps: //setew caps a player's weapon at +16 and tells the
// GM; //create_item caps the level it gives.
func TestGMEnchantCaps(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sa, gm := e.enterChar(testChar{account: "gm", name: "Gm", pos: tiStart, access: storage.AccessLevelAdmin})
	sb, p := e.enterWith("hero", eastOf(tiStart, 50), enchanted(enchSaber, 0))
	e.useItem(sb, e.itemOf(p, enchSaber).ObjectID())
	sa.take(t)
	e.onLoop(func() { e.srv.SetTarget(gm, p) })
	e.onLoop(func() { e.srv.RunGMCommand(gm, "setew 30") })
	if got := e.enchantOf(p, e.itemOf(p, enchSaber).ObjectID()); got != 16 {
		t.Errorf("//setew 30 gave +%d, want +16", got)
	}
	var says []string
	for _, c := range only[testclient.CreatureSay](sa.take(t)) {
		says = append(says, c.Text)
	}
	if !slices.Contains(says, "Maximum enchantment for weapon items is 16.") {
		t.Errorf("GM messages %q", says)
	}
	e.onLoop(func() { e.srv.RunGMCommand(gm, "create_item 123 1 40") })
	var lvl int32
	e.onLoop(func() { lvl = gm.inv.ByItemID(enchSaber).Enchant() })
	if lvl != 16 {
		t.Errorf("//create_item gave +%d, want +16", lvl)
	}
}

// TestCrystallize: Crystallize (248) at level 2 crystallizes a C armor into
// its crystals; at level 1 it may not; the counts follow the level.
func TestCrystallize(t *testing.T) {
	e := newEnchantEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart, enchanted(enchCompositeArmor, 0), enchanted(enchCompositeArmor, 5), enchanted(enchHeroWeapon, 0))
	setSkill := func(level int) {
		e.onLoop(func() { p.skill.known.put(&skills.Skill{Skill: &data.Skill{ID: skillCrystallize, Level: level}}) })
	}
	crystallize := func(it *inventory.Item, count int32) []testclient.Packet {
		e.advance(2 * time.Second) // the transaction flood limit
		sess.take(t)
		return e.handle(sess, clientpackets.OpRequestCrystallizeItem, ints(it.ObjectID(), count))
	}
	var plain, plus5 *inventory.Item
	e.onLoop(func() {
		for _, it := range p.inv.AllByItemID(enchCompositeArmor) {
			if it.Enchant() == 0 {
				plain = it
			} else {
				plus5 = it
			}
		}
	})

	// Without the skill.
	ps := crystallize(plain, 1)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgCrystallizeLevelTooLow)}) || len(only[testclient.ActionFailed](ps)) != 1 {
		t.Errorf("no skill: %v", testclient.Names(ps))
	}
	// Level 1 is too low for C grade.
	setSkill(1)
	if ids := sysMsgIDsOf(crystallize(plain, 1)); !slices.Equal(ids, []int32{int32(sysMsgCrystallizeLevelTooLow)}) {
		t.Errorf("level 1: %v", ids)
	}
	// Level 2: 576 C crystals.
	setSkill(2)
	ps = crystallize(plain, 1)
	want := []string{"InventoryUpdate", "StatusUpdate", "SystemMessage", "SystemMessage"}
	if got := testclient.Names(ps); !slices.Equal(got, want) {
		t.Fatalf("packets %v, want %v", got, want)
	}
	if ch := itemChanges(ps); !slices.Equal(ch[0], []string{"removed:60+0"}) {
		t.Errorf("InventoryUpdate %v", ch)
	}
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgS1Crystallized), int32(sysMsgEarnedS2S1)}) {
		t.Errorf("messages %v", ids)
	}
	if n := e.countOf(p, enchCrystalC); n != 576 {
		t.Errorf("%d C crystals, want 576", n)
	}
	// A worn +5 armor comes off first: 576 + 6 × 9 = 630 more.
	e.useItem(sess, plus5.ObjectID())
	ps = crystallize(plus5, 1)
	if ids := sysMsgIDsOf(ps); len(ids) != 3 || ids[0] != int32(sysMsgEquipmentRemoved) {
		t.Errorf("worn armor messages %v", ids)
	}
	if n := e.countOf(p, enchCrystalC); n != 576+630 {
		t.Errorf("%d C crystals, want 1206", n)
	}
	// Hero weapons are refused.
	setSkill(5)
	ps = crystallize(e.itemOf(p, enchHeroWeapon), 1)
	if len(only[testclient.ActionFailed](ps)) != 1 || e.countOf(p, enchHeroWeapon) != 1 {
		t.Errorf("hero weapon: %v", testclient.Names(ps))
	}
	// A count below 1 is an illegal action.
	ps = crystallize(e.itemOf(p, enchHeroWeapon), 0)
	if !slices.Contains(messages(ps), "You will be kicked for illegal action, GM informed.") {
		t.Errorf("count 0: %q", messages(ps))
	}
}

// TestEnchantWithoutData: without enchant data nothing is over-enchanted.
func TestEnchantWithoutData(t *testing.T) {
	e := newItemEnv(t, nil)
	_, p := e.enterWith("hero", world.Position{}, enchanted(enchSaber, 40))
	if e.countOf(p, enchSaber) != 1 {
		t.Error("an item was destroyed without enchant data")
	}
}
