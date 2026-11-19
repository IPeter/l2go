package testclient

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

// The client packets of Milestone 5 as Mobius reads them, transcribed from
// their readImpl.

// mobiusUseItem: d object, d ctrl (readInt() != 0).
func mobiusUseItem(objectID int32, ctrl bool) []byte {
	return msg(0x14).i32(objectID).i32(b2i(ctrl)).b
}

// mobiusUnEquipItem: d slot, a paperdoll index (BodyPart.fromPaperdollSlot).
func mobiusUnEquipItem(slot int32) []byte { return msg(0x11).i32(slot).b }

// mobiusRequestDropItem: d object, d count, d x, d y, d z.
func mobiusRequestDropItem(objectID, count int32, at Position) []byte {
	return msg(0x12).i32(objectID).i32(count).i32(at.X).i32(at.Y).i32(at.Z).b
}

// mobiusRequestDestroyItem: d object, d count.
func mobiusRequestDestroyItem(objectID, count int32) []byte {
	return msg(0x59).i32(objectID).i32(count).b
}

// mobiusRequestBuyItem: d list, d size, then size times d item, d count.
// Mobius reads the list only when size * 8 bytes follow exactly.
func mobiusRequestBuyItem(listID int32, items ...BuyItem) []byte {
	w := msg(0x1f).i32(listID).i32(int32(len(items)))
	for _, it := range items {
		w.i32(it.ItemID).i32(it.Count)
	}
	return w.b
}

// mobiusRequestSellItem: d list, d size, then size times d object, d item,
// d count (BATCH_LENGTH 12).
func mobiusRequestSellItem(listID int32, items ...SellItem) []byte {
	w := msg(0x1e).i32(listID).i32(int32(len(items)))
	for _, it := range items {
		w.i32(it.ObjectID).i32(it.ItemID).i32(it.Count)
	}
	return w.b
}

// mobiusMultiSellChoose: d list, d entry, d amount.
func mobiusMultiSellChoose(listID, entryID, amount int32) []byte {
	return msg(0xa7).i32(listID).i32(entryID).i32(amount).b
}

// mobiusWarehouseRequest is SendWareHouseDepositList (0x31) and
// SendWareHouseWithDrawList (0x32): d size, then size times d object, d
// count (BATCH_LENGTH 8).
func mobiusWarehouseRequest(op byte, items ...WarehouseItem) []byte {
	w := msg(op).i32(int32(len(items)))
	for _, it := range items {
		w.i32(it.ObjectID).i32(it.Count)
	}
	return w.b
}

// mobiusSMItem is a SystemMessage with one item name parameter.
func mobiusSMItem(id, itemID int32) []byte {
	return msg(0x64).i32(id).i32(1).i32(SMParamItemName).i32(itemID).b
}

// enterQuick runs the server side of selecting "Hero" and of a short
// enter-world sequence at pos: UserInfo, the inventory, the extra packets,
// then the end of Mobius's sequence.
func (f *fakeGame) enterQuick(pos Position, items []Item, extra ...[]byte) {
	f.fakeSelect([]CharInfo{testChar("Hero", testMeID)})
	f.expect(mobiusEnterWorld())
	f.send(mobiusUserInfo(testUserInfoAt(pos)))
	f.send(mobiusItemList(ItemList{Items: items}))
	for _, b := range extra {
		f.send(b)
	}
	f.send(mobiusValidateLocation(ValidateLocation{ObjectID: testMeID, X: pos.X, Y: pos.Y, Z: pos.Z}))
	f.send(msg(0x25).b)
	f.send(mobiusSkillList(nil))
}

// clientEnterQuick runs the client side of enterQuick.
func clientEnterQuick(t *testing.T, ctx context.Context, addr string) *GameConn {
	t.Helper()
	c := clientSelect(t, ctx, addr)
	if _, err := c.EnterWorld(ctx); err != nil {
		t.Fatalf("EnterWorld: %v", err)
	}
	return c
}

func TestEncodeItemRequests(t *testing.T) {
	at := Position{-71000, 258000, -3100}
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"UseItem", encodeInts(OpUseItem, 0x10000016, 0), mobiusUseItem(0x10000016, false)},
		{"UseItem bytes", encodeInts(OpUseItem, 0x10000016, 0), []byte{0x14, 0x16, 0, 0, 0x10, 0, 0, 0, 0}},
		{"RequestUnEquipItem", encodeInt(OpRequestUnEquipItem, PaperdollRHand), []byte{0x11, 7, 0, 0, 0}},
		{"RequestDropItem", encodeInts(OpRequestDropItem, 0x10000015, 3, at.X, at.Y, at.Z), mobiusRequestDropItem(0x10000015, 3, at)},
		{"RequestDestroyItem", encodeInts(OpRequestDestroyItem, 0x10000015, 10), []byte{0x59, 0x15, 0, 0, 0x10, 10, 0, 0, 0}},
		{"RequestDestroyItem builder", encodeInts(OpRequestDestroyItem, 1, 2), mobiusRequestDestroyItem(1, 2)},
		{"RequestItemList", encodeOpcode(OpRequestItemList), []byte{0x0f}},
		{"RequestBuyItem", encodeRequestBuyItem(3, []BuyItem{{1060, 5}, {1, 1}}), mobiusRequestBuyItem(3, BuyItem{1060, 5}, BuyItem{1, 1})},
		{"RequestBuyItem bytes", encodeRequestBuyItem(3, []BuyItem{{1060, 5}}), []byte{
			0x1f,
			3, 0, 0, 0, // list 3
			1, 0, 0, 0, // one product
			0x24, 0x04, 0, 0, // item 1060
			5, 0, 0, 0, // count 5
		}},
		{"RequestSellItem", encodeRequestSellItem(0, []SellItem{{0x10000015, 1060, 5}}), mobiusRequestSellItem(0, SellItem{0x10000015, 1060, 5})},
		{"MultiSellChoose", encodeInts(OpMultiSellChoose, 93, 100001, 2), mobiusMultiSellChoose(93, 100001, 2)},
		{"SendWareHouseDepositList", encodeWarehouseItems(OpSendWareHouseDepositList, []WarehouseItem{{0x10000014, 1000}, {0x10000016, 1}}),
			mobiusWarehouseRequest(0x31, WarehouseItem{0x10000014, 1000}, WarehouseItem{0x10000016, 1})},
		{"SendWareHouseWithDrawList", encodeWarehouseItems(OpSendWareHouseWithDrawList, []WarehouseItem{{0x10000020, 1}}),
			mobiusWarehouseRequest(0x32, WarehouseItem{0x10000020, 1})},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
	}

	// Mobius drops a list unless its entries fill the rest of the packet
	// (BATCH_LENGTH).
	batches := []struct {
		name          string
		body          []byte
		head, n, size int
	}{
		{"RequestBuyItem", encodeRequestBuyItem(1, make([]BuyItem, 3)), 9, 3, 8},
		{"RequestSellItem", encodeRequestSellItem(1, make([]SellItem, 2)), 9, 2, 12},
		{"SendWareHouseDepositList", encodeWarehouseItems(OpSendWareHouseDepositList, make([]WarehouseItem, 4)), 5, 4, 8},
		{"SendWareHouseWithDrawList", encodeWarehouseItems(OpSendWareHouseWithDrawList, make([]WarehouseItem, 1)), 5, 1, 8},
	}
	for _, b := range batches {
		if got := len(b.body) - b.head; got != b.n*b.size {
			t.Errorf("%s: %d bytes of entries, Mobius reads %d", b.name, got, b.n*b.size)
		}
	}
}

func TestItemRequestNames(t *testing.T) {
	tests := []struct {
		raw  []byte
		want string
	}{
		{encodeInts(OpUseItem, 1, 0), "UseItem"},
		{encodeInt(OpRequestUnEquipItem, 7), "RequestUnEquipItem"},
		{encodeInts(OpRequestDropItem, 1, 1, 0, 0, 0), "RequestDropItem"},
		{encodeInts(OpRequestDestroyItem, 1, 1), "RequestDestroyItem"},
		{encodeOpcode(OpRequestItemList), "RequestItemList"},
		{encodeRequestBuyItem(1, nil), "RequestBuyItem"},
		{encodeRequestSellItem(1, nil), "RequestSellItem"},
		{encodeInts(OpMultiSellChoose, 1, 1, 1), "MultiSellChoose"},
		{encodeWarehouseItems(OpSendWareHouseDepositList, nil), "SendWareHouseDepositList"},
		{encodeWarehouseItems(OpSendWareHouseWithDrawList, nil), "SendWareHouseWithDrawList"},
	}
	for _, tt := range tests {
		if got := clientPacketName(tt.raw); got != tt.want {
			t.Errorf("clientPacketName(% x) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

// TestAwaitItemChange skips the InventoryUpdates about other items.
func TestAwaitItemChange(t *testing.T) {
	items := testItems()
	adena := items[3]
	adena.Count = 4900
	c := queueConn(0)
	feed(c,
		mobiusItemList(ItemList{Items: items}),
		mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: items[0]}}}),
		actionFailed(),
		mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: items[1]}, {Change: ItemModified, Item: adena}}}),
	)
	ctx := testContext(t)
	ch, err := c.AwaitItemChange(ctx, adena.ObjectID)
	if err != nil || ch.Change != ItemModified || ch.Count != 4900 {
		t.Fatalf("AwaitItemChange = %+v, %v", ch, err)
	}
	if n := c.World().ItemCount(57); n != 4900 {
		t.Errorf("adena %d, want 4900", n)
	}
}

// TestItemFlow uses, takes off, drops, picks up and destroys items against
// a fake server with encryption on, with Mobius's order of answers: the
// inventory changes come last, gathered into one InventoryUpdate.
func TestItemFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	dropAt := Position{-71320, 258290, -3104}
	potion := Item{Type1: 4, ObjectID: 0x10000015, ItemID: 1060, Count: 10, Type2: 5, Mana: -1}
	dagger := Item{Type1: 0, ObjectID: 0x10000016, ItemID: 10, Count: 1, Type2: 0, BodyPart: 0x0080, Mana: -1}
	inv := append(testItems(), potion, dagger)
	sword := inv[2]
	const dropped int32 = 0x10000030

	daggerOn := dagger
	daggerOn.Equipped = true
	swordOff := sword
	swordOff.Equipped = false
	potion7 := potion
	potion7.Count = 7
	final := []Item{inv[0], inv[1], swordOff, inv[3], dagger}

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, inv)

		// Equipping the dagger: UserInfo, the message, then the inventory.
		f.expect(mobiusUseItem(dagger.ObjectID, false))
		f.send(mobiusUserInfo(testUserInfoAt(pos)))
		f.send(mobiusSMItem(SysMsgEquipped, dagger.ItemID))
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{
			{Change: ItemModified, Item: swordOff}, {Change: ItemModified, Item: daggerOn},
		}}))

		// Taking off what the right hand holds.
		f.expect(mobiusUnEquipItem(PaperdollRHand))
		f.send(mobiusUserInfo(testUserInfoAt(pos)))
		f.send(mobiusSMItem(SysMsgDisarmed, dagger.ItemID))
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: dagger}}}))

		// Dropping 3 potions: a new object falls, then the stack shrinks.
		f.expect(mobiusRequestDropItem(potion.ObjectID, 3, dropAt))
		f.send(mobiusDropItem(DropItem{DropperID: testMeID, ObjectID: dropped, ItemID: 1060,
			X: dropAt.X, Y: dropAt.Y, Z: dropAt.Z, Stackable: true, Count: 3, Unknown: 1}))
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: potion7}}}))

		// Picking them up: GetItem, DeleteObject, then the stack grows.
		f.expect(mobiusAction(0x04, dropped, pos, false))
		f.send(mobiusGetItem(GetItem{PlayerID: testMeID, ObjectID: dropped, X: dropAt.X, Y: dropAt.Y, Z: dropAt.Z}))
		f.send(msg(0x12).i32(dropped).i32(1).b)
		f.send(msg(0x64).i32(SysMsgObtainedItems).i32(2).i32(SMParamItemName).i32(1060).i32(SMParamInt).i32(3).b)
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: potion}}}))

		// Destroying the stack: the load, then the inventory.
		f.expect(mobiusRequestDestroyItem(potion.ObjectID, 10))
		f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurLoad, 1500}))
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemRemoved, Item: potion}}}))

		// The inventory window.
		f.expect([]byte{0x0f})
		f.send(mobiusItemList(ItemList{ShowWindow: true, Items: final}))
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	w := c.World()
	if n := len(w.Items()); n != len(inv) {
		t.Fatalf("%d items after entering, want %d", n, len(inv))
	}

	// Equip the dagger.
	if err := c.UseItem(ctx, dagger.ObjectID); err != nil {
		t.Fatal(err)
	}
	ch, err := c.AwaitItemChange(ctx, dagger.ObjectID)
	if err != nil || ch.Change != ItemModified || !ch.Equipped {
		t.Fatalf("AwaitItemChange after UseItem = %+v, %v", ch, err)
	}
	if it, _ := w.Item(sword.ObjectID); it.Equipped {
		t.Error("the sword is still equipped")
	}
	if !slices.ContainsFunc(w.SystemMessages(), func(m SystemMessage) bool { return m.ID == SysMsgEquipped }) {
		t.Error("the equip message is missing")
	}

	// Take it off.
	if err := c.UnequipItem(ctx, PaperdollRHand); err != nil {
		t.Fatal(err)
	}
	if ch, err := c.AwaitItemChange(ctx, dagger.ObjectID); err != nil || ch.Equipped {
		t.Fatalf("AwaitItemChange after UnequipItem = %+v, %v", ch, err)
	}

	// Drop three potions.
	d, err := c.DropItem(ctx, potion.ObjectID, 3, dropAt.X, dropAt.Y, dropAt.Z)
	if err != nil || d.ObjectID != dropped || d.Count != 3 || d.Position() != dropAt || d.DropperID != testMeID {
		t.Fatalf("DropItem = %+v, %v", d, err)
	}
	if g, ok := w.GroundItem(dropped); !ok || g.DropperID != testMeID || g.Count != 3 || g.Pos != dropAt {
		t.Errorf("GroundItem = %+v, %v", g, ok)
	}
	if ch, err := c.AwaitItemChange(ctx, potion.ObjectID); err != nil || ch.Count != 7 {
		t.Fatalf("potions after the drop = %+v, %v", ch, err)
	}
	if n := w.ItemCount(1060); n != 7 {
		t.Errorf("%d potions, want 7", n)
	}

	// Pick them up.
	g, err := c.PickUp(ctx, dropped)
	if err != nil || g.PlayerID != testMeID || g.ObjectID != dropped {
		t.Fatalf("PickUp = %+v, %v", g, err)
	}
	if ch, err := c.AwaitItemChange(ctx, potion.ObjectID); err != nil || ch.Count != 10 {
		t.Fatalf("potions after the pick-up = %+v, %v", ch, err)
	}
	if _, ok := w.GroundItem(dropped); ok {
		t.Error("the potions are still on the ground")
	}

	// Destroy them.
	ch, err = c.DestroyItem(ctx, potion.ObjectID, 10)
	if err != nil || ch.Change != ItemRemoved || ch.ObjectID != potion.ObjectID {
		t.Fatalf("DestroyItem = %+v, %v", ch, err)
	}
	if _, ok := w.Item(potion.ObjectID); ok {
		t.Error("the destroyed potions are still in the inventory")
	}
	if me := w.Me(); me.Load != 1500 {
		t.Errorf("load %d, want 1500", me.Load)
	}

	// Open the inventory window.
	if err := c.RequestItemList(ctx); err != nil {
		t.Fatal(err)
	}
	l, err := Await[ItemList](ctx, c, nil)
	if err != nil || !l.ShowWindow {
		t.Fatalf("ItemList = %+v, %v", l, err)
	}
	if got := w.Items(); !reflect.DeepEqual(got, final) {
		t.Errorf("inventory\n got %+v\nwant %+v", got, final)
	}
}

// TestShopAndWarehouseFlow buys, sells, exchanges at a multisell and uses
// the warehouse against a fake server, in Mobius's order of answers.
func TestShopAndWarehouseFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	const merchant int32 = 0x60000002
	inv := testItems()
	adena := inv[3]
	potion := Item{Type1: 4, ObjectID: 0x10000040, ItemID: 1060, Count: 5, Type2: 5, Mana: -1}
	adenaBought := adena
	adenaBought.Count -= 5 * testPotion.Price
	afterBuy := []Item{inv[0], inv[1], inv[2], adenaBought, potion}
	adenaSold := adenaBought
	adenaSold.Count += 5 * 20
	afterSell := []Item{inv[0], inv[1], inv[2], adenaSold}
	ancient := Item{Type1: 4, ObjectID: 0x10000041, ItemID: 5575, Count: 1000, Type2: 5, Mana: -1}
	adenaExchanged := adenaSold
	adenaExchanged.Count -= 1000
	afterExchange := []Item{inv[0], inv[1], inv[2], adenaExchanged, ancient}
	adenaDeposited := adenaExchanged
	adenaDeposited.Count -= 1000 + 30 // the amount and the fee
	stored := WarehouseListItem{ObjectID: 0x10000020, ItemID: 2369, Count: 1, BodyPart: 0x4000, ObjectID2: 0x10000020}
	withdrawn := Item{ObjectID: 0x10000020, ItemID: 2369, Count: 1, BodyPart: 0x4000, Mana: -1}

	limited := testPotion
	limited.Count = 50
	sellList := SellList{Adena: adenaBought.Count, Items: []SellListItem{
		{Type1: 4, ObjectID: potion.ObjectID, ItemID: 1060, Count: 5, Type2: 5, Price: 20},
	}}
	page1 := MultiSellList{ListID: 93, Page: 1, PageSize: 40, Entries: []MultiSellEntry{{
		EntryID: 100001, Reserved3: 1,
		Products:    []MultiSellProduct{{ItemID: 5575, Type2: 5, Count: 1000}},
		Ingredients: []MultiSellIngredient{{ItemID: 57, Type2: 4, Count: 1000}},
	}}}
	page2 := MultiSellList{ListID: 93, Page: 2, Finished: true, PageSize: 40, Entries: []MultiSellEntry{{
		EntryID: 4100001, Reserved3: 1,
		Products:    []MultiSellProduct{{ItemID: 57, Type2: 4, Count: 1000}},
		Ingredients: []MultiSellIngredient{{ItemID: 5575, Type2: 5, Count: 1000}},
	}}}
	bypass := func(cmd string) string { return fmt.Sprintf("npc_%d_%s", merchant, cmd) }

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, inv)

		// The buy window lists 3 products and leaves out the sold-out one.
		f.expect(msg(0x21).str(bypass("Buy 3")).b)
		f.send(mobiusBuyList(adena.Count, 3, 3, []BuyListItem{testShortSword, limited}))
		f.send(msg(0x25).b)
		f.expect(mobiusRequestBuyItem(3, BuyItem{1060, 5}))
		f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurLoad, 200}))
		f.send(mobiusItemList(ItemList{ShowWindow: true, Items: afterBuy}))
		f.send(mobiusSystemMessage(SysMsgTransactionComplete))

		// Selling the potions back.
		f.expect(msg(0x21).str(bypass("Sell")).b)
		f.send(mobiusSellList(sellList))
		f.send(msg(0x25).b)
		f.expect(mobiusRequestSellItem(0, SellItem{potion.ObjectID, 1060, 5}))
		f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurLoad, 150}))
		f.send(mobiusItemList(ItemList{ShowWindow: true, Items: afterSell}))
		f.send(mobiusSystemMessage(SysMsgTransactionComplete))

		// A multisell list of two pages, then one exchange.
		f.expect(msg(0x21).str(bypass("multisell 93")).b)
		f.send(mobiusMultiSellList(page1))
		f.send(mobiusMultiSellList(page2))
		f.expect(mobiusMultiSellChoose(93, 100001, 1))
		f.send(mobiusSMItem(SysMsgObtainedItems, 5575))
		f.send(mobiusItemList(ItemList{Items: afterExchange}))
		f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurLoad, 160}))

		// Storing 1000 adena: the load at once, the inventory 100 ms later.
		f.expect(msg(0x21).str(bypass("DepositP")).b)
		f.send(mobiusWarehouseList(0x41, WarehouseList{Type: WarehousePrivate, Adena: adenaExchanged.Count, Items: []WarehouseListItem{
			{Type1: 4, ObjectID: adena.ObjectID, ItemID: 57, Count: adenaExchanged.Count, Type2: 4, ObjectID2: adena.ObjectID},
		}}))
		f.expect(mobiusWarehouseRequest(0x31, WarehouseItem{adena.ObjectID, 1000}))
		f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurLoad, 160}))
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: adenaDeposited}}}))

		// Taking a stored sword back.
		f.expect(msg(0x21).str(bypass("WithdrawP")).b)
		f.send(mobiusWarehouseList(0x42, WarehouseList{Type: WarehousePrivate, Adena: adenaDeposited.Count, Items: []WarehouseListItem{stored}}))
		f.expect(mobiusWarehouseRequest(0x32, WarehouseItem{stored.ObjectID, 1}))
		f.send(mobiusStatusUpdate(testMeID, StatusAttribute{StatusCurLoad, 1660}))
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemAdded, Item: withdrawn}}}))
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	w := c.World()

	// Buy.
	if err := c.Bypass(ctx, bypass("Buy 3")); err != nil {
		t.Fatal(err)
	}
	bl, err := Await[BuyList](ctx, c, nil)
	if err != nil || bl.ListID != 3 || bl.Listed != 3 || len(bl.Items) != 2 || bl.Adena != adena.Count {
		t.Fatalf("BuyList = %+v, %v", bl, err)
	}
	if p, ok := bl.Find(1060); !ok || p.Count != 50 || p.Price != testPotion.Price {
		t.Errorf("potion product = %+v, %v", p, ok)
	}
	l, err := c.BuyItems(ctx, bl.ListID, []BuyItem{{ItemID: 1060, Count: 5}})
	if err != nil || !l.ShowWindow || !reflect.DeepEqual(l.Items, afterBuy) {
		t.Fatalf("BuyItems = %+v, %v", l, err)
	}
	if n := w.ItemCount(1060); n != 5 {
		t.Errorf("%d potions after buying", n)
	}
	if me := w.Me(); me.Load != 200 {
		t.Errorf("load %d after buying, want 200", me.Load)
	}

	// Sell.
	if err := c.Bypass(ctx, bypass("Sell")); err != nil {
		t.Fatal(err)
	}
	sl, err := Await[SellList](ctx, c, nil)
	if err != nil || !reflect.DeepEqual(sl, sellList) {
		t.Fatalf("SellList = %+v, %v", sl, err)
	}
	it, _ := sl.Find(potion.ObjectID)
	l, err = c.SellItems(ctx, sl.ListID, []SellItem{{ObjectID: it.ObjectID, ItemID: it.ItemID, Count: it.Count}})
	if err != nil || !reflect.DeepEqual(l.Items, afterSell) {
		t.Fatalf("SellItems = %+v, %v", l, err)
	}
	if n := w.ItemCount(57); n != int64(adenaSold.Count) {
		t.Errorf("adena %d after selling, want %d", n, adenaSold.Count)
	}

	// Multisell: the two pages come as one list.
	if err := c.Bypass(ctx, bypass("multisell 93")); err != nil {
		t.Fatal(err)
	}
	ml, err := c.AwaitMultiSellList(ctx)
	wantList := MultiSellList{ListID: 93, Page: 2, Finished: true, PageSize: 40,
		Entries: append(slices.Clone(page1.Entries), page2.Entries...)}
	if err != nil || !reflect.DeepEqual(ml, wantList) {
		t.Fatalf("AwaitMultiSellList = %+v, %v", ml, err)
	}
	entry, ok := ml.Find(100001)
	if !ok || entry.Ingredients[0].Count != 1000 {
		t.Fatalf("entry 100001 = %+v, %v", entry, ok)
	}
	if _, ok := ml.Find(1); ok {
		t.Error("Find(1) found an entry")
	}
	l, err = c.MultiSellChoose(ctx, ml.ListID, entry.EntryID, 1)
	if err != nil || l.ShowWindow || !reflect.DeepEqual(l.Items, afterExchange) {
		t.Fatalf("MultiSellChoose = %+v, %v", l, err)
	}

	// Deposit.
	if err := c.Bypass(ctx, bypass("DepositP")); err != nil {
		t.Fatal(err)
	}
	dl, err := Await[WareHouseDepositList](ctx, c, nil)
	if err != nil || dl.Type != WarehousePrivate || len(dl.Items) != 1 {
		t.Fatalf("WareHouseDepositList = %+v, %v", dl, err)
	}
	u, err := c.DepositItems(ctx, []WarehouseItem{{ObjectID: adena.ObjectID, Count: 1000}})
	if err != nil || len(u.Items) != 1 || u.Items[0].Count != adenaDeposited.Count {
		t.Fatalf("DepositItems = %+v, %v", u, err)
	}
	if n := w.ItemCount(57); n != int64(adenaDeposited.Count) {
		t.Errorf("adena %d after the deposit, want %d", n, adenaDeposited.Count)
	}

	// Withdraw.
	if err := c.Bypass(ctx, bypass("WithdrawP")); err != nil {
		t.Fatal(err)
	}
	wl, err := Await[WareHouseWithdrawalList](ctx, c, nil)
	if err != nil {
		t.Fatal(err)
	}
	sw, ok := wl.FindItem(2369)
	if !ok {
		t.Fatalf("the stored sword is not listed: %+v", wl)
	}
	u, err = c.WithdrawItems(ctx, []WarehouseItem{{ObjectID: sw.ObjectID, Count: 1}})
	if err != nil || len(u.Items) != 1 || u.Items[0].Change != ItemAdded {
		t.Fatalf("WithdrawItems = %+v, %v", u, err)
	}
	if got, ok := w.Item(stored.ObjectID); !ok || got != withdrawn {
		t.Errorf("the withdrawn sword = %+v, %v", got, ok)
	}
	if me := w.Me(); me.Load != 1660 {
		t.Errorf("load %d after the withdrawal, want 1660", me.Load)
	}
}
