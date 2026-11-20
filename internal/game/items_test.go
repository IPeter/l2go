package game

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// TestUseItemUnequipsAndEquips: double-clicking the worn weapon takes it
// off ("... has been disarmed.", lower P.Atk, CharInfo for the others) and
// puts it on again ("You have equipped your ...").
func TestUseItemUnequipsAndEquips(t *testing.T) {
	e := newItemEnv(t, nil)
	sa, pa := e.enterWith("hero", tiStart)
	sb, _ := e.enterWith("watcher", eastOf(tiStart, 100))
	sa.take(t)
	sword := e.itemOf(pa, itemSquireSword)
	var armed int32
	e.onLoop(func() { armed = pa.stats.View().PAtk })

	ps := e.useItem(sa, sword.ObjectID())
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgDisarmed)}) {
		t.Errorf("unequip messages %v, want disarmed", ids)
	}
	if e.wornIn(pa, storage.PaperdollRHand) != 0 {
		t.Error("the sword is still worn")
	}
	if len(only[testclient.InventoryUpdate](ps)) != 0 {
		t.Error("InventoryUpdate came at once; Mobius sends it 100 ms later")
	}
	sb.take(t)
	e.advance(inventoryUpdateDelay)
	ps = sa.take(t)
	ius := only[testclient.InventoryUpdate](ps)
	if len(ius) != 1 || len(ius[0].Items) != 1 || ius[0].Items[0].ObjectID != sword.ObjectID() || ius[0].Items[0].Equipped || ius[0].Items[0].Change != testclient.ItemModified {
		t.Errorf("InventoryUpdate %+v, want the sword modified and not equipped", ius)
	}
	uis := only[testclient.UserInfo](ps)
	if len(uis) == 0 || uis[len(uis)-1].PAtk >= armed {
		t.Errorf("UserInfo after unequip %+v, want P.Atk below %d", uis, armed)
	}
	if len(only[testclient.CharInfoPacket](sb.take(t))) == 0 {
		t.Error("the other player saw no CharInfo")
	}

	ps = e.useItem(sa, sword.ObjectID())
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgEquippedS1)}) {
		t.Errorf("equip messages %v, want equipped", ids)
	}
	if e.wornIn(pa, storage.PaperdollRHand) != itemSquireSword {
		t.Error("the sword is not worn again")
	}
}

// TestEquipRules goes through Mobius Inventory.equipItem with UseItem.
func TestEquipRules(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart)
	use := func(it *inventory.Item) []testclient.Packet { return e.useItem(sess, it.ObjectID()) }
	worn := func(slot storage.PaperdollSlot) int32 { return e.wornIn(p, slot) }
	all := func(id int32) []*inventory.Item {
		var out []*inventory.Item
		e.onLoop(func() { out = slices.Clone(p.inv.AllByItemID(id)) })
		return out
	}
	shield := e.give(p, itemLeatherShield, 1)
	claymore := e.give(p, itemClaymore, 1)
	bow := e.give(p, itemBow, 1)
	e.give(p, itemWoodenArrow, 100)
	armor := e.give(p, itemCompositeArmor, 1)
	pants := e.itemOf(p, itemSquirePants)
	sword := e.itemOf(p, itemSquireSword)
	e.give(p, itemEarring, 3)
	e.give(p, itemMagicRing, 3)
	formal := e.give(p, itemFormalWear, 1)
	helmet := e.give(p, itemLeatherHelmet, 1)
	sess.take(t)

	// A shield goes next to a one-handed weapon.
	use(shield)
	if worn(storage.PaperdollLHand) != itemLeatherShield || worn(storage.PaperdollRHand) != itemSquireSword {
		t.Error("shield and sword are not both worn")
	}
	// A two-handed weapon takes the shield off; a shield takes it off.
	use(claymore)
	if worn(storage.PaperdollRHand) != itemClaymore || worn(storage.PaperdollLHand) != 0 {
		t.Error("the two-handed sword did not take the shield off")
	}
	use(shield)
	if worn(storage.PaperdollLHand) != itemLeatherShield || worn(storage.PaperdollRHand) != 0 {
		t.Error("the shield did not take the two-handed sword off")
	}
	// A bow puts arrows of its grade into the left hand; they come off with
	// it.
	use(bow)
	if worn(storage.PaperdollRHand) != itemBow || worn(storage.PaperdollLHand) != itemWoodenArrow {
		t.Errorf("bow: right %d left %d, want the bow and its arrows", worn(storage.PaperdollRHand), worn(storage.PaperdollLHand))
	}
	// RequestUnEquipItem leaves arrows in the left hand alone.
	e.handle(sess, clientpackets.OpRequestUnEquipItem, ints(int32(storage.PaperdollLHand)))
	if worn(storage.PaperdollLHand) != itemWoodenArrow {
		t.Error("RequestUnEquipItem took the arrows off")
	}
	use(bow)
	if worn(storage.PaperdollRHand) != 0 || worn(storage.PaperdollLHand) != 0 {
		t.Error("the arrows stayed when the bow came off")
	}
	// Full armor takes the legs off, legs take full armor off.
	use(armor)
	if worn(storage.PaperdollChest) != itemCompositeArmor || worn(storage.PaperdollLegs) != 0 {
		t.Error("the full armor did not take the pants off")
	}
	use(pants)
	if worn(storage.PaperdollLegs) != itemSquirePants || worn(storage.PaperdollChest) != 0 {
		t.Error("the pants did not take the full armor off")
	}
	// Earrings and rings: the left one, the right one, then the left again.
	for _, pair := range []struct {
		id          int32
		left, right storage.PaperdollSlot
	}{{itemEarring, storage.PaperdollLEar, storage.PaperdollREar}, {itemMagicRing, storage.PaperdollLFinger, storage.PaperdollRFinger}} {
		items := all(pair.id)
		if len(items) != 3 {
			t.Fatalf("%d items of %d, want 3", len(items), pair.id)
		}
		var left, right *inventory.Item
		for i, it := range items {
			use(it)
			e.onLoop(func() { left, right = p.inv.Paperdoll(pair.left), p.inv.Paperdoll(pair.right) })
			switch i {
			case 0:
				if left != it || right != nil {
					t.Errorf("item %d: first one not on the left", pair.id)
				}
			case 1:
				if left != items[0] || right != it {
					t.Errorf("item %d: second one not on the right", pair.id)
				}
			case 2:
				if left != it || right != items[1] {
					t.Errorf("item %d: third one did not replace the left", pair.id)
				}
			}
		}
	}
	// Formal wear takes off the weapon and the legs; then nothing goes on
	// the head; a weapon takes the formal wear off.
	use(sword)
	use(formal)
	if worn(storage.PaperdollChest) != itemFormalWear || worn(storage.PaperdollRHand) != 0 || worn(storage.PaperdollLegs) != 0 {
		t.Error("formal wear did not take the weapon and pants off")
	}
	ps := use(helmet)
	if worn(storage.PaperdollHead) != 0 || !slices.Contains(sysMsgIDsOf(ps), int32(sysMsgEquipConditionNotMet)) {
		t.Errorf("a helmet went on over formal wear (messages %v)", sysMsgIDsOf(ps))
	}
	use(sword)
	if worn(storage.PaperdollChest) != 0 || worn(storage.PaperdollRHand) != itemSquireSword {
		t.Error("the sword did not take the formal wear off")
	}
	// RequestUnEquipItem takes the weapon off by its paperdoll slot.
	ps = e.handle(sess, clientpackets.OpRequestUnEquipItem, ints(int32(storage.PaperdollRHand)))
	if worn(storage.PaperdollRHand) != 0 || !slices.Equal(sysMsgIDsOf(ps), []int32{int32(sysMsgDisarmed)}) {
		t.Errorf("RequestUnEquipItem: right hand %d, messages %v", worn(storage.PaperdollRHand), sysMsgIDsOf(ps))
	}
}

// TestUseItemRefusalsAndHandlers: quest items, item conditions, lures and
// books.
func TestUseItemRefusalsAndHandlers(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart)

	letter := e.give(p, itemDarinsLetter, 1)
	sess.take(t)
	if ids := sysMsgIDsOf(e.useItem(sess, letter.ObjectID())); !slices.Equal(ids, []int32{int32(sysMsgCannotUseQuestItems)}) {
		t.Errorf("quest item messages %v", ids)
	}

	// An item whose conditions a Human Fighter fails stays off.
	var refused int32
	e.onLoop(func() {
		for id := range e.srv.items.conds {
			tmpl, _ := e.data.ItemTemplate(id)
			if tmpl == nil || !inventory.NewItem(1, tmpl, 1).Equipable() {
				continue
			}
			if !e.srv.items.conds.Check(id, itemCondition{p: p, cats: e.srv.items.categories}).OK {
				refused = int32(id)
				return
			}
		}
	})
	if refused == 0 {
		t.Fatal("no equipable item with conditions a Human Fighter fails")
	}
	it := e.give(p, refused, 1)
	sess.take(t)
	ps := e.useItem(sess, it.ObjectID())
	if len(only[testclient.SystemMessage](ps)) == 0 {
		t.Errorf("item %d with failing conditions gave no message: %q", refused, testclient.Names(ps))
	}
	e.onLoop(func() {
		if it.Equipped() {
			t.Errorf("item %d went on despite its conditions", refused)
		}
	})

	// A lure goes into the left hand of a fishing rod.
	rod := e.give(p, itemFishingRod, 1)
	lure := e.give(p, itemLure, 10)
	e.useItem(sess, rod.ObjectID())
	e.useItem(sess, lure.ObjectID())
	if e.wornIn(p, storage.PaperdollLHand) != itemLure {
		t.Error("the lure is not in the left hand")
	}

	// A book shows its help page, bound to the item.
	book := e.give(p, itemTutorialGuide, 1)
	sess.take(t)
	ps = e.useItem(sess, book.ObjectID())
	pages := only[testclient.NpcHtmlMessage](ps)
	if len(pages) != 1 || pages[0].ItemID != itemTutorialGuide || pages[0].HTML != bookPage || pages[0].ObjectID != 0 {
		t.Errorf("book page %+v", pages)
	}
	checkNames(t, "book", ps, "NpcHtmlMessage", "ActionFailed")
}

// TestDropAndPickUp: a part of an adena stack falls for both players, the
// other picks it up and it disappears for both.
func TestDropAndPickUp(t *testing.T) {
	e := newItemEnv(t, func(gs *config.GameServer) { gs.Economy.FloodDropItem = 0 })
	sa, pa := e.enterWith("hero", tiStart, storage.Item{ItemID: itemAdena, Count: 1000})
	sb, pb := e.enterWith("finder", eastOf(tiStart, 120))
	sa.take(t)
	sb.take(t)
	adena := e.itemOf(pa, itemAdena)
	at := eastOf(tiStart, 60)

	// Too far away, or more than there is, is refused.
	far := eastOf(tiStart, 200)
	ps := e.handle(sa, clientpackets.OpRequestDropItem, ints(adena.ObjectID(), 300, far.X, far.Y, far.Z))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgTooFarToDiscard)}) {
		t.Errorf("far drop messages %v", ids)
	}
	ps = e.handle(sa, clientpackets.OpRequestDropItem, ints(adena.ObjectID(), 5000, at.X, at.Y, at.Z))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgCannotDiscard)}) {
		t.Errorf("too many messages %v", ids)
	}

	ps = e.handle(sa, clientpackets.OpRequestDropItem, ints(adena.ObjectID(), 300, at.X, at.Y, at.Z))
	drops := only[testclient.DropItem](ps)
	if len(drops) != 1 || drops[0].DropperID != pa.ObjectID() || drops[0].Count != 300 || drops[0].ItemID != itemAdena || drops[0].X != at.X {
		t.Fatalf("dropper got %+v", drops)
	}
	ground := drops[0].ObjectID
	if ground == adena.ObjectID() {
		t.Error("a part of a stack kept the stack's object ID")
	}
	if seen := only[testclient.DropItem](sb.take(t)); len(seen) != 1 || seen[0].ObjectID != ground {
		t.Errorf("the other player saw %+v", seen)
	}
	e.advance(inventoryUpdateDelay)
	ius := only[testclient.InventoryUpdate](sa.take(t))
	if len(ius) != 1 || ius[0].Items[0].ObjectID != adena.ObjectID() || ius[0].Items[0].Count != 700 {
		t.Errorf("dropper's InventoryUpdate %+v, want 700 adena left", ius)
	}

	// The other player clicks the item, walks over and picks it up.
	e.srv.Handle(sb, clientpackets.OpAction, npcActionBody(ground, false))
	var got []testclient.GetItem
	for range 50 {
		e.advance(100 * time.Millisecond)
		if got = only[testclient.GetItem](sb.take(t)); len(got) > 0 {
			break
		}
	}
	if len(got) != 1 || got[0].PlayerID != pb.ObjectID() || got[0].ObjectID != ground {
		t.Fatalf("picker got GetItem %+v", got)
	}
	ps = sa.take(t)
	if gi := only[testclient.GetItem](ps); len(gi) != 1 || gi[0].PlayerID != pb.ObjectID() {
		t.Errorf("dropper saw GetItem %+v", gi)
	}
	if del := only[testclient.DeleteObject](ps); len(del) != 1 || del[0].ObjectID != ground {
		t.Errorf("dropper saw DeleteObject %+v", del)
	}
	var have int64
	e.onLoop(func() { have = pb.inv.Adena() })
	if have != 300 {
		t.Errorf("the finder has %d adena, want 300", have)
	}
	e.onLoop(func() {
		if _, ok := e.srv.world.Find(world.ObjectID(ground)); ok {
			t.Error("the item is still in the world")
		}
	})
}

// TestDropRules: a worn item comes off before it falls, whole stacks keep
// their object ID, quest items and no-drop zones refuse.
func TestDropRules(t *testing.T) {
	e := newItemEnv(t, func(gs *config.GameServer) { gs.Economy.FloodDropItem = 0 })
	sess, p := e.enterWith("hero", tiStart)
	// The starting sword cannot be dropped.
	squire := e.itemOf(p, itemSquireSword)
	sess.take(t)
	ps := e.handle(sess, clientpackets.OpRequestDropItem, ints(squire.ObjectID(), 1, tiStart.X, tiStart.Y, tiStart.Z))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgCannotDiscard)}) {
		t.Errorf("dropping the Squire's Sword: messages %v", ids)
	}

	sword := e.give(p, itemLongSword, 1)
	e.useItem(sess, sword.ObjectID())
	e.advance(time.Second)
	sess.take(t)
	ps = e.handle(sess, clientpackets.OpRequestDropItem, ints(sword.ObjectID(), 1, tiStart.X, tiStart.Y, tiStart.Z))
	if drops := only[testclient.DropItem](ps); len(drops) != 1 || drops[0].ObjectID != sword.ObjectID() {
		t.Errorf("worn sword drop %+v", drops)
	}
	if e.wornIn(p, storage.PaperdollRHand) != 0 || e.itemOf(p, itemLongSword) != nil {
		t.Error("the sword is still carried")
	}
	e.advance(itemListDelay)
	ps = sess.take(t)
	if il := only[testclient.ItemList](ps); len(il) != 1 || !il[0].ShowWindow {
		t.Errorf("ItemList after dropping a worn item %+v", il)
	}

	letter := e.give(p, itemDarinsLetter, 1)
	sess.take(t)
	ps = e.handle(sess, clientpackets.OpRequestDropItem, ints(letter.ObjectID(), 1, tiStart.X, tiStart.Y, tiStart.Z))
	if len(only[testclient.DropItem](ps)) != 0 {
		t.Error("a quest item was dropped")
	}

	shirt := e.itemOf(p, itemSquireShirt)
	e.onLoop(func() { p.Zones.Enter(ZoneNoItemDrop) })
	ps = e.handle(sess, clientpackets.OpRequestDropItem, ints(shirt.ObjectID(), 1, tiStart.X, tiStart.Y, tiStart.Z))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgCannotDiscard)}) || len(only[testclient.DropItem](ps)) != 0 {
		t.Errorf("drop in a no-drop zone: messages %v", ids)
	}
}

// TestDestroyItem: the trash can takes part of a stack, a whole worn item,
// and refuses what cannot be destroyed.
func TestDestroyItem(t *testing.T) {
	e := newItemEnv(t, func(gs *config.GameServer) { gs.Economy.FloodTransaction = 0 })
	sess, p := e.enterWith("hero", tiStart, storage.Item{ItemID: itemMithrilOre, Count: 10})
	ore := e.itemOf(p, itemMithrilOre)
	sess.take(t)

	ps := e.handle(sess, clientpackets.OpRequestDestroyItem, ints(ore.ObjectID(), 4))
	checkNames(t, "partial destroy", ps, "InventoryUpdate", "StatusUpdate", "SystemMessage")
	if ius := only[testclient.InventoryUpdate](ps); ius[0].Items[0].Count != 6 || ius[0].Items[0].Change != testclient.ItemModified {
		t.Errorf("InventoryUpdate %+v, want 6 left", ius)
	}
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgS2S1Disappeared)}) {
		t.Errorf("messages %v", ids)
	}

	sword := e.itemOf(p, itemSquireSword)
	ps = e.handle(sess, clientpackets.OpRequestDestroyItem, ints(sword.ObjectID(), 1))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgDisarmed), int32(sysMsgS1Disappeared)}) {
		t.Errorf("worn destroy messages %v", ids)
	}
	ius := only[testclient.InventoryUpdate](ps)
	if len(ius) != 2 || ius[1].Items[0].Change != testclient.ItemRemoved || ius[1].Items[0].ObjectID != sword.ObjectID() {
		t.Errorf("worn destroy InventoryUpdates %+v", ius)
	}
	if e.itemOf(p, itemSquireSword) != nil || e.wornIn(p, storage.PaperdollRHand) != 0 {
		t.Error("the sword is still there")
	}

	ps = e.handle(sess, clientpackets.OpRequestDestroyItem, ints(ore.ObjectID(), 0))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgDestroyCountIncorrect)}) {
		t.Errorf("count 0 messages %v", ids)
	}
	ps = e.handle(sess, clientpackets.OpRequestDestroyItem, ints(0x5fffffff, 1))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgCannotDiscard)}) {
		t.Errorf("unknown item messages %v", ids)
	}
	pants := e.itemOf(p, itemSquirePants)
	ps = e.handle(sess, clientpackets.OpRequestDestroyItem, ints(pants.ObjectID(), 2))
	if msgs := messages(ps); len(msgs) != 1 || !strings.Contains(msgs[0], "kicked") {
		t.Errorf("destroying 2 of an item that does not stack: %q", msgs)
	}
	e.advance(illegalActionKickDelay)
	if _, _, closed := sess.closedWith(t); !closed {
		t.Error("the cheater was not kicked")
	}
}

// TestWeightPenalty goes through the weight thresholds: 50 % shows the
// icon, 66.6 % halves the run speed, more than the maximum blocks walking.
func TestWeightPenalty(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart)
	var maxLoad, load int64
	var run float64
	e.onLoop(func() {
		maxLoad, load = int64(p.stats.WeightLimit()), p.inv.Weight()
		run = p.stats.RunSpeed()
	})
	sess.take(t)
	ore := func(percent int64) int64 {
		var now int64
		e.onLoop(func() { now = p.inv.Weight() })
		return (maxLoad*percent/100 - now) / 2
	}
	for _, step := range []struct {
		percent int64
		level   int32
		halved  bool
	}{{55, 1, false}, {70, 2, true}, {90, 3, true}, {105, 4, true}} {
		e.give(p, itemMithrilOre, ore(step.percent))
		etc := only[testclient.EtcStatusUpdate](sess.take(t))
		if len(etc) != 1 || etc[0].WeightPenalty != step.level {
			t.Errorf("at %d%%: EtcStatusUpdate %+v, want level %d", step.percent, etc, step.level)
		}
		// The packets show the change through the move multiplier.
		var nowRun float64
		e.onLoop(func() { nowRun = p.stats.RunSpeed() })
		if halved := nowRun < run; halved != step.halved {
			t.Errorf("at %d%%: run speed %v (was %v), halved %v", step.percent, nowRun, run, step.halved)
		}
	}
	if load <= 0 {
		t.Error("no starting load")
	}
	if !e.onLoopBool(p.Overloaded) {
		t.Fatal("not overloaded above the maximum")
	}
	to := eastOf(tiStart, 300)
	ps := e.handle(sess, clientpackets.OpMoveToLocation, walkRequest(to, tiStart, false))
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgTooEncumbered)}) || len(only[testclient.ActionFailed](ps)) != 1 {
		t.Errorf("walking overloaded: %q, messages %v", testclient.Names(ps), ids)
	}
	// Dropping the ore lifts it all.
	e.onLoop(func() {
		it := p.inv.ByItemID(itemMithrilOre)
		e.srv.DestroyItem(p, it, it.Count(), false)
	})
	if etc := only[testclient.EtcStatusUpdate](sess.take(t)); len(etc) != 1 || etc[0].WeightPenalty != 0 || e.onLoopBool(p.Overloaded) {
		t.Errorf("after destroying the ore: %+v", etc)
	}
}

// onLoopBool reads a flag on the loop.
func (e *itemEnv) onLoopBool(f func() bool) bool {
	var v bool
	e.onLoop(func() { v = f() })
	return v
}

// TestGroundExpiry: monster drops vanish after AutoDestroyDroppedItemAfter,
// herbs sooner; protected items and player drops stay.
func TestGroundExpiry(t *testing.T) {
	e := newItemEnv(t, func(gs *config.GameServer) { gs.Items.ListOfProtectedItems = []int{itemMithrilOre} })
	sess, p := e.enterWith("hero", tiStart, storage.Item{ItemID: itemAdena, Count: 100})
	sess.take(t)
	drop := func(itemID int32, playerDrop bool) *GroundItem {
		var g *GroundItem
		e.onLoop(func() {
			t, _ := e.data.ItemTemplate(int(itemID))
			id, _ := e.srv.items.pool.TryNext()
			var err error
			g, err = e.srv.DropOnGround(inventory.NewItem(id, t, 1), eastOf(tiStart, 30), GroundDrop{PlayerDrop: playerDrop})
			if err != nil {
				e.t.Errorf("DropOnGround: %v", err)
			}
		})
		return g
	}
	monster := drop(itemLongSword, false)
	protected := drop(itemMithrilOre, false)
	byPlayer := drop(itemClaymore, true)
	if spawned := only[testclient.SpawnItem](sess.take(t)); len(spawned) != 3 {
		t.Errorf("the player saw %d SpawnItem, want 3", len(spawned))
	}
	e.advance(599 * time.Second)
	if !e.onLoopBool(monster.InWorld) {
		t.Error("the monster drop vanished early")
	}
	e.advance(time.Second)
	if e.onLoopBool(monster.InWorld) {
		t.Error("the monster drop is still there after 600 s")
	}
	if del := only[testclient.DeleteObject](sess.take(t)); len(del) != 1 || del[0].ObjectID != int32(monster.ID()) {
		t.Errorf("DeleteObject %+v", del)
	}
	e.advance(time.Hour)
	if !e.onLoopBool(protected.InWorld) || !e.onLoopBool(byPlayer.InWorld) {
		t.Error("a protected item or a player's drop vanished")
	}
	_ = p
}

// TestShadowMana: a worn shadow item loses one mana when it goes on and one
// a minute; at 0 it disappears.
func TestShadowMana(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart)
	sword := e.give(p, itemShadowSword, 1)
	var mana int32
	e.onLoop(func() { mana = sword.Mana() })
	if mana != 90 {
		t.Fatalf("new shadow sword has %d mana, want 90", mana)
	}
	sess.take(t)
	e.useItem(sess, sword.ObjectID())
	e.onLoop(func() { mana = sword.Mana() })
	if mana != 89 {
		t.Errorf("mana %d after equipping, want 89", mana)
	}
	e.advance(manaConsumptionRate)
	e.onLoop(func() { mana = sword.Mana() })
	if mana != 88 {
		t.Errorf("mana %d a minute later, want 88", mana)
	}
	e.onLoop(func() { sword.SetMana(2) })
	sess.take(t)
	e.advance(manaConsumptionRate)
	if ids := sysMsgIDsOf(sess.take(t)); !slices.Equal(ids, []int32{int32(sysMsgMana1)}) {
		t.Errorf("at mana 1: messages %v", ids)
	}
	e.advance(manaConsumptionRate + inventoryUpdateDelay)
	ps := sess.take(t)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgMana0)}) {
		t.Errorf("at mana 0: messages %v", ids)
	}
	if e.itemOf(p, itemShadowSword) != nil || e.wornIn(p, storage.PaperdollRHand) != 0 {
		t.Error("the spent shadow sword is still there")
	}
	ius := only[testclient.InventoryUpdate](ps)
	if len(ius) == 0 || ius[len(ius)-1].Items[0].Change != testclient.ItemRemoved || ius[len(ius)-1].Items[0].ObjectID != sword.ObjectID() {
		t.Errorf("InventoryUpdate %+v, want the sword removed", ius)
	}
}

// TestSaveOrderGiverFirst: when an item moves between characters, the
// giver's save is written before the receiver's, also when the receiver
// already had a save waiting.
func TestSaveOrderGiverFirst(t *testing.T) {
	e := newItemEnv(t, func(gs *config.GameServer) { gs.Economy.FloodDropItem = 0 })
	sa, pa := e.enterWith("giver", tiStart, storage.Item{ItemID: itemAdena, Count: 1000})
	sb, pb := e.enterWith("taker", eastOf(tiStart, 30))
	adena := e.itemOf(pa, itemAdena)
	release := e.store.holdSaves()

	// The giver's first save is being written (held); the receiver's first
	// save waits in line behind it.
	e.give(pa, itemMithrilOre, 5)
	e.advance(100 * time.Millisecond)
	select {
	case <-e.store.started:
	case <-e.ctx.Done():
		t.Fatal("the first save did not start")
	}
	e.give(pb, itemMithrilOre, 5)
	e.advance(100 * time.Millisecond)
	e.handle(sa, clientpackets.OpRequestDropItem, ints(adena.ObjectID(), 400, tiStart.X+10, tiStart.Y, tiStart.Z))
	var g *GroundItem
	e.onLoop(func() {
		for o := range e.srv.world.Around(tiStart, 0) {
			if gi, ok := o.(*GroundItem); ok {
				g = gi
			}
		}
		e.srv.doPickUp(pb, g)
	})
	if g == nil {
		t.Fatal("no item on the ground")
	}
	e.advance(100 * time.Millisecond)
	release()
	sa.take(t)
	sb.take(t)
	deadline := time.Now().Add(10 * time.Second)
	for {
		e.store.mu.Lock()
		n := len(e.store.saves)
		e.store.mu.Unlock()
		if n >= 4 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	e.store.mu.Lock()
	defer e.store.mu.Unlock()
	giverAt, takerAt := -1, -1
	for i, c := range e.store.saves {
		ch := e.store.changes[i]
		switch {
		case c.ObjectID == pa.ObjectID() && slices.ContainsFunc(ch.Upserts, func(it storage.Item) bool { return it.ObjectID == adena.ObjectID() && it.Count == 600 }):
			giverAt = i
		case c.ObjectID == pb.ObjectID() && slices.ContainsFunc(ch.Upserts, func(it storage.Item) bool { return it.ObjectID == int32(g.ID()) }):
			takerAt = i
		}
	}
	if giverAt < 0 || takerAt < 0 || giverAt > takerAt {
		t.Errorf("giver's save at %d, taker's with the item at %d: want the giver first (saves %d)", giverAt, takerAt, len(e.store.saves))
	}
}

// TestItemsSaveLoadRoundTrip: worn, carried and warehouse items come back
// as they were after a relog.
func TestItemsSaveLoadRoundTrip(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart,
		storage.Item{ItemID: itemMithrilOre, Count: 7, Loc: storage.LocWarehouse},
		storage.Item{ItemID: itemAdena, Count: 500})
	var wh []storage.Item
	e.onLoop(func() {
		for _, it := range p.Warehouse().Items() {
			wh = append(wh, it.Storage())
		}
		if p.inv.ByItemID(itemMithrilOre) != nil {
			t.Error("a warehouse item is carried")
		}
	})
	if len(wh) != 1 || wh[0].ItemID != itemMithrilOre || wh[0].Count != 7 {
		t.Fatalf("warehouse %+v", wh)
	}
	// Change things: a new worn item, less adena, an item into the warehouse.
	shield := e.give(p, itemLeatherShield, 1)
	e.useItem(sess, shield.ObjectID())
	e.onLoop(func() {
		e.srv.ReduceAdena(p, 120, false)
		pants := p.inv.ByItemID(itemSquirePants)
		p.inv.UnequipItem(pants)
		if _, err := p.inv.Transfer(pants.ObjectID(), 1, p.Warehouse()); err != nil {
			t.Errorf("Transfer: %v", err)
		}
		e.srv.ItemsChanged(p)
	})
	type state struct {
		carried, warehouse []storage.Item
	}
	read := func(p *Player) state {
		var s state
		e.onLoop(func() {
			for _, it := range p.inv.Items() {
				s.carried = append(s.carried, it.Storage())
			}
			for _, it := range p.Warehouse().Items() {
				s.warehouse = append(s.warehouse, it.Storage())
			}
		})
		sortItems := func(items []storage.Item) {
			slices.SortFunc(items, func(a, b storage.Item) int { return int(a.ObjectID - b.ObjectID) })
		}
		sortItems(s.carried)
		sortItems(s.warehouse)
		return s
	}
	before := read(p)
	e.srv.Logout(sess)
	if err := e.srv.saver.wait(e.ctx, p.ObjectID()); err != nil {
		t.Fatalf("wait: %v", err)
	}
	_, p2 := e.enterWithSession(newSession("hero"))
	after := read(p2)
	if !slices.Equal(before.carried, after.carried) || !slices.Equal(before.warehouse, after.warehouse) {
		t.Errorf("after a relog:\ncarried   %+v\nwant      %+v\nwarehouse %+v\nwant      %+v", after.carried, before.carried, after.warehouse, before.warehouse)
	}
	if e.wornIn(p2, storage.PaperdollLHand) != itemLeatherShield || e.wornIn(p2, storage.PaperdollLegs) != 0 {
		t.Error("the paperdoll did not come back")
	}
}

// enterWithSession enters the first character of the session's account.
func (e *itemEnv) enterWithSession(sess *fakeSession) (*fakeSession, *Player) {
	e.t.Helper()
	p := e.enter(sess, 0)
	sess.take(e.t)
	return sess, p
}

// TestSaveRetry: a failed save is tried again until it works.
func TestSaveRetry(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart)
	sess.take(t)
	e.srv.saver.retryDelay = time.Millisecond
	e.store.mu.Lock()
	e.store.failSaves = 2
	e.store.mu.Unlock()
	e.give(p, itemMithrilOre, 3)
	e.advance(100 * time.Millisecond)
	saved := e.store.waitSave(t, p.ObjectID())
	if !slices.ContainsFunc(saved.Items, func(it storage.Item) bool { return it.ItemID == itemMithrilOre && it.Count == 3 }) {
		t.Errorf("saved items %+v, want the ore", saved.Items)
	}
}

// TestPlayerItemMethods: the methods scripts use.
func TestPlayerItemMethods(t *testing.T) {
	e := newItemEnv(t, nil)
	sess, p := e.enterWith("hero", tiStart, storage.Item{ItemID: itemAdena, Count: 500})
	sess.take(t)
	e.onLoop(func() {
		if p.Adena() != 500 {
			t.Errorf("adena %d", p.Adena())
		}
		if !p.GiveItems(itemMithrilOre, 5) || p.ItemCount(itemMithrilOre) != 5 {
			t.Errorf("give: count %d", p.ItemCount(itemMithrilOre))
		}
		if p.TakeItems(itemMithrilOre, 9) || p.ItemCount(itemMithrilOre) != 0 {
			t.Errorf("take 9 of 5: count %d", p.ItemCount(itemMithrilOre))
		}
		if !p.TakeItems(itemAdena, 100) || p.Adena() != 400 {
			t.Errorf("take adena: %d left", p.Adena())
		}
		if !p.GiveItems(itemAdena, 50) || p.Adena() != 450 {
			t.Errorf("give adena: %d", p.Adena())
		}
		if !p.TakeItems(itemSquireSword, -1) || p.inv.Paperdoll(storage.PaperdollRHand) != nil {
			t.Error("taking the worn sword failed")
		}
	})
	ids := sysMsgIDsOf(sess.take(t))
	for _, want := range []int32{int32(sysMsgEarnedS2S1), int32(sysMsgAdenaDisappeared), int32(sysMsgEarnedS1Adena), int32(sysMsgS1Disappeared)} {
		if !slices.Contains(ids, want) {
			t.Errorf("messages %v lack %d", ids, want)
		}
	}
}

// TestGMItemCommands: //create_item, //give_item_target and //delete_item.
func TestGMItemCommands(t *testing.T) {
	e := newItemEnv(t, nil)
	sa, gm := e.enterChar(testChar{account: "gm", name: "Gm", pos: tiStart, access: storage.AccessLevelAdmin})
	sb, p := e.enterWith("hero", eastOf(tiStart, 50))
	sa.take(t)
	sb.take(t)

	e.onLoop(func() { e.srv.RunGMCommand(gm, "create_item 57 1000") })
	e.onLoop(func() { e.srv.RunGMCommand(gm, "create_item 70 11") })
	e.onLoop(func() { e.srv.RunGMCommand(gm, "create_item 70 1 5") })
	msgs := messages(sa.take(t))
	if len(msgs) != 3 || !strings.Contains(msgs[0], "You have spawned 1000 Adena(57) in Gm inventory.") ||
		msgs[1] != "This item does not stack - Creation aborted." {
		t.Errorf("create_item messages %q", msgs)
	}
	var enchant int32
	e.onLoop(func() {
		if gm.inv.Adena() != 1000 {
			t.Errorf("GM adena %d", gm.inv.Adena())
		}
		enchant = gm.inv.ByItemID(itemClaymore).Enchant()
	})
	if enchant != 5 {
		t.Errorf("created sword +%d, want +5", enchant)
	}

	e.onLoop(func() { e.srv.SetTarget(gm, p) })
	e.onLoop(func() { e.srv.RunGMCommand(gm, "give_item_target 1876 3") })
	if msgs := messages(sb.take(t)); !slices.Contains(msgs, "Admin spawned 3 Mithril Ore in your inventory.") {
		t.Errorf("target messages %q", msgs)
	}
	ore := e.itemOf(p, itemMithrilOre)
	e.onLoop(func() { e.srv.RunGMCommand(gm, "delete_item "+strconv.Itoa(int(ore.ObjectID()))+" 2") })
	e.onLoop(func() {
		if n := p.ItemCount(itemMithrilOre); n != 1 {
			t.Errorf("ore left %d, want 1", n)
		}
	})
}

// TestLuaItemMethods runs the item methods from a script.
func TestLuaItemMethods(t *testing.T) {
	scripts := map[string]string{"npcs/items/items.lua": `
local s = script.new("ItemNpc")
s:on_first_talk(30598, function(npc, player)
  local before = player:adena()
  local took = player:take_items(57, 100)
  local gave = player:give_items(1876, 5)
  return "<html><body>" .. before .. " " .. tostring(took) .. " " .. tostring(gave) .. " " .. player:adena() .. " " .. player:item_count(1876) .. "</body></html>"
end)
`}
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, scripts: scripts})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart, adena: 300})
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiStart, -50))
	ps := e.talk(sess, p, guide)
	if pages := only[testclient.NpcHtmlMessage](ps); len(pages) != 1 || !strings.Contains(pages[0].HTML, "300 true true 200 5") {
		t.Errorf("script page %+v (%q)", pages, testclient.Names(ps))
	}
}
