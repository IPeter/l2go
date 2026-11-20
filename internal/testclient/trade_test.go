package testclient

import (
	"bytes"
	"testing"
)

// The client packets of trades and private stores as Mobius reads them,
// transcribed from their readImpl.

// mobiusTradeRequest: d object.
func mobiusTradeRequest(objectID int32) []byte { return msg(0x15).i32(objectID).b }

// mobiusAddTradeItem: d trade ID, d object, d count.
func mobiusAddTradeItem(tradeID, objectID, count int32) []byte {
	return msg(0x16).i32(tradeID).i32(objectID).i32(count).b
}

// mobiusTradeDoneRequest is the client TradeDone: d response.
func mobiusTradeDoneRequest(response int32) []byte { return msg(0x17).i32(response).b }

// mobiusAnswerTradeRequest: d response.
func mobiusAnswerTradeRequest(response int32) []byte { return msg(0x44).i32(response).b }

// mobiusSetPrivateStoreListSell: d package sale, d count, then count times
// d object, d count, d price (BATCH_LENGTH 12, exact).
func mobiusSetPrivateStoreListSell(packageSale bool, items ...StoreSellOffer) []byte {
	w := msg(0x74).i32(b2i(packageSale)).i32(int32(len(items)))
	for _, it := range items {
		w.i32(it.ObjectID).i32(it.Count).i32(it.Price)
	}
	return w.b
}

// mobiusSetPrivateStoreListBuy: d count, then count times d item, h
// ("Enchant?"), h ("Price per unit"), d count, d price.
func mobiusSetPrivateStoreListBuy(items ...StoreBuyOffer) []byte {
	w := msg(0x91).i32(int32(len(items)))
	for _, it := range items {
		w.i32(it.ItemID).u16(0).u16(0).i32(it.Count).i32(it.Price)
	}
	return w.b
}

// mobiusRequestPrivateStoreBuy: d store player, d count, then count times
// d object, d count, d price (BATCH_LENGTH 12, exact).
func mobiusRequestPrivateStoreBuy(storeID int32, items ...StorePurchase) []byte {
	w := msg(0x79).i32(storeID).i32(int32(len(items)))
	for _, it := range items {
		w.i32(it.ObjectID).i32(it.Count).i32(it.Price)
	}
	return w.b
}

// mobiusRequestPrivateStoreSell: d store player, d count, then count times
// d object, d item, h, h, d count, d price (BATCH_LENGTH 20, exact).
func mobiusRequestPrivateStoreSell(storeID int32, items ...StoreSale) []byte {
	w := msg(0x96).i32(storeID).i32(int32(len(items)))
	for _, it := range items {
		w.i32(it.ObjectID).i32(it.ItemID).u16(0).u16(0).i32(it.Count).i32(it.Price)
	}
	return w.b
}

// mobiusActionUse is RequestActionUse: d action, d ctrl, c shift.
func mobiusActionUse(actionID int32) []byte { return msg(0x45).i32(actionID).i32(0).u8(0).b }

func TestEncodeTradeRequests(t *testing.T) {
	sellOffers := []StoreSellOffer{{0x10000016, 1, 500000}, {0x10000017, 100, 8}}
	buyOffers := []StoreBuyOffer{{1835, 5000, 8}}
	purchases := []StorePurchase{{0x10000116, 1, 500000}}
	sales := []StoreSale{{0x10000017, 1835, 300, 8}, {0x10000018, 1463, 10, 3}}
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"TradeRequest", encodeInt(OpTradeRequest, testOtherID), mobiusTradeRequest(testOtherID)},
		{"TradeRequest bytes", encodeInt(OpTradeRequest, testOtherID), []byte{0x15, 0x02, 0, 0, 0x10}},
		{"AddTradeItem", encodeInts(OpAddTradeItem, 0, 0x10000016, 1), mobiusAddTradeItem(0, 0x10000016, 1)},
		{"TradeDone confirm", encodeInt(OpTradeDone, 1), mobiusTradeDoneRequest(1)},
		{"TradeDone cancel", encodeInt(OpTradeDone, 0), []byte{0x17, 0, 0, 0, 0}},
		{"AnswerTradeRequest", encodeInt(OpAnswerTradeRequest, boolInt32(true)), mobiusAnswerTradeRequest(1)},
		{"RequestPrivateStoreManageSell", encodeOpcode(OpRequestPrivateStoreManageSell), []byte{0x73}},
		{"RequestPrivateStoreQuitSell", encodeOpcode(OpRequestPrivateStoreQuitSell), []byte{0x76}},
		{"RequestPrivateStoreManageBuy", encodeOpcode(OpRequestPrivateStoreManageBuy), []byte{0x90}},
		{"RequestPrivateStoreQuitBuy", encodeOpcode(OpRequestPrivateStoreQuitBuy), []byte{0x93}},
		{"SetPrivateStoreMsgSell", encodeString(OpSetPrivateStoreMsgSell, "SS"), []byte{0x77, 'S', 0, 'S', 0, 0, 0}},
		{"SetPrivateStoreMsgBuy", encodeString(OpSetPrivateStoreMsgBuy, "WTB"), msg(0x94).str("WTB").b},
		{"SetPrivateStoreListSell", encodeSetPrivateStoreListSell(false, sellOffers), mobiusSetPrivateStoreListSell(false, sellOffers...)},
		{"SetPrivateStoreListSell package", encodeSetPrivateStoreListSell(true, sellOffers[:1]), []byte{
			0x74,
			1, 0, 0, 0, // package sale
			1, 0, 0, 0, // one item
			0x16, 0, 0, 0x10, // object 0x10000016
			1, 0, 0, 0, // count 1
			0x20, 0xa1, 0x07, 0x00, // price 500000
		}},
		{"SetPrivateStoreListBuy", encodeSetPrivateStoreListBuy(buyOffers), mobiusSetPrivateStoreListBuy(buyOffers...)},
		{"RequestPrivateStoreBuy", encodeRequestPrivateStoreBuy(testOtherID, purchases), mobiusRequestPrivateStoreBuy(testOtherID, purchases...)},
		{"RequestPrivateStoreSell", encodeRequestPrivateStoreSell(testOtherID, sales), mobiusRequestPrivateStoreSell(testOtherID, sales...)},
		{"RequestActionUse sell store", encodeRequestActionUse(ActionPrivateStoreSell, false, false), mobiusActionUse(10)},
		{"RequestActionUse buy store", encodeRequestActionUse(ActionPrivateStoreBuy, false, false), mobiusActionUse(28)},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
	}

	// Mobius drops a list unless its entries fill the rest of the packet
	// (BATCH_LENGTH); SetPrivateStoreListBuy needs at least 12 bytes per
	// entry and reads 16.
	batches := []struct {
		name          string
		body          []byte
		head, n, size int
	}{
		{"SetPrivateStoreListSell", encodeSetPrivateStoreListSell(false, make([]StoreSellOffer, 3)), 9, 3, 12},
		{"SetPrivateStoreListBuy", encodeSetPrivateStoreListBuy(make([]StoreBuyOffer, 2)), 5, 2, 16},
		{"RequestPrivateStoreBuy", encodeRequestPrivateStoreBuy(1, make([]StorePurchase, 2)), 9, 2, 12},
		{"RequestPrivateStoreSell", encodeRequestPrivateStoreSell(1, make([]StoreSale, 4)), 9, 4, 20},
	}
	for _, b := range batches {
		if got := len(b.body) - b.head; got != b.n*b.size {
			t.Errorf("%s: %d bytes of entries, Mobius reads %d", b.name, got, b.n*b.size)
		}
	}
}

func TestTradeRequestNames(t *testing.T) {
	tests := []struct {
		raw  []byte
		want string
	}{
		{encodeInt(OpTradeRequest, 1), "TradeRequest"},
		{encodeInts(OpAddTradeItem, 0, 1, 1), "AddTradeItem"},
		{encodeInt(OpTradeDone, 1), "TradeDone"},
		{encodeInt(OpAnswerTradeRequest, 1), "AnswerTradeRequest"},
		{encodeOpcode(OpRequestPrivateStoreManageSell), "RequestPrivateStoreManageSell"},
		{encodeSetPrivateStoreListSell(false, nil), "SetPrivateStoreListSell"},
		{encodeOpcode(OpRequestPrivateStoreQuitSell), "RequestPrivateStoreQuitSell"},
		{encodeString(OpSetPrivateStoreMsgSell, ""), "SetPrivateStoreMsgSell"},
		{encodeRequestPrivateStoreBuy(1, nil), "RequestPrivateStoreBuy"},
		{encodeOpcode(OpRequestPrivateStoreManageBuy), "RequestPrivateStoreManageBuy"},
		{encodeSetPrivateStoreListBuy(nil), "SetPrivateStoreListBuy"},
		{encodeOpcode(OpRequestPrivateStoreQuitBuy), "RequestPrivateStoreQuitBuy"},
		{encodeString(OpSetPrivateStoreMsgBuy, ""), "SetPrivateStoreMsgBuy"},
		{encodeRequestPrivateStoreSell(1, nil), "RequestPrivateStoreSell"},
	}
	for _, tt := range tests {
		if got := clientPacketName(tt.raw); got != tt.want {
			t.Errorf("clientPacketName(% x) = %q, want %q", tt.raw, got, tt.want)
		}
	}
	servers := []struct {
		op   byte
		want string
	}{
		{OpTradeStart, "TradeStart"},
		{OpTradeOwnAdd, "TradeOwnAdd"},
		{OpTradeOtherAdd, "TradeOtherAdd"},
		{OpTradeDoneServer, "TradeDone"},
		{OpSendTradeRequest, "SendTradeRequest"},
		{OpTradeUpdate, "TradeUpdate"},
		{OpTradeOtherDone, "TradeOtherDone"},
		{OpPrivateStoreManageListSell, "PrivateStoreManageListSell"},
		{OpPrivateStoreListSell, "PrivateStoreListSell"},
		{OpPrivateStoreMsgSell, "PrivateStoreMsgSell"},
		{OpPrivateStoreManageListBuy, "PrivateStoreManageListBuy"},
		{OpPrivateStoreListBuy, "PrivateStoreListBuy"},
		{OpPrivateStoreMsgBuy, "PrivateStoreMsgBuy"},
	}
	for _, tt := range servers {
		if got := serverPacketName([]byte{tt.op}); got != tt.want {
			t.Errorf("serverPacketName(%#x) = %q, want %q", tt.op, got, tt.want)
		}
	}
}

// TestTradeFlow trades with a partner against a fake server with
// encryption on, in Mobius's order of answers: the request, the window,
// an item each side, the partner's confirmation, then the client's.
func TestTradeFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	inv := testItems()
	sword := inv[2]
	start := TradeStart{PartnerID: testOtherID, Items: []TradeItem{
		{Type1: uint16(sword.Type1), ObjectID: sword.ObjectID, ItemID: sword.ItemID, Count: 1, BodyPart: sword.BodyPart},
	}}
	own := start.Items[0]
	other := TradeItem{Type1: 4, ObjectID: 0x10000116, ItemID: 57, Count: 1000, Type2: 4}

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, inv)

		// A request to the partner: Mobius answers the requester with a
		// system message only.
		f.expect(mobiusTradeRequest(testOtherID))
		f.send(mobiusSystemMessage(SysMsgTransactionComplete))

		// The partner asks; the client accepts.
		f.send(msg(0x5e).i32(testOtherID).b)
		f.expect(mobiusAnswerTradeRequest(1))
		f.send(mobiusTradeStart(start))

		// The client puts the sword in: TradeOwnAdd, then TradeUpdate.
		f.expect(mobiusAddTradeItem(0, own.ObjectID, 1))
		f.send(mobiusTradeAdd(0x20, own))
		f.send(mobiusTradeUpdate(TradeUpdate{Items: []TradeUpdateItem{
			{Change: TradeUpdateRemove, Type1: own.Type1, ObjectID: own.ObjectID, ItemID: own.ItemID, Count: 1, BodyPart: own.BodyPart},
		}}))

		// The partner puts adena in and confirms.
		f.send(mobiusTradeAdd(0x21, other))
		f.send(msg(0x7c).b)

		// The client confirms: the trade ends.
		f.expect(mobiusTradeDoneRequest(1))
		f.send(msg(0x22).i32(1).b)

		// A second trade, canceled.
		f.send(msg(0x5e).i32(testOtherID).b)
		f.expect(mobiusAnswerTradeRequest(0))
		f.send(msg(0x5e).i32(testOtherID).b)
		f.expect(mobiusAnswerTradeRequest(1))
		f.send(mobiusTradeStart(TradeStart{PartnerID: testOtherID}))
		f.expect(mobiusTradeDoneRequest(0))
		f.send(msg(0x22).i32(0).b)
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)
	if err := c.RequestTrade(ctx, testOtherID); err != nil {
		t.Fatal(err)
	}
	if _, err := Await[SystemMessage](ctx, c, nil); err != nil {
		t.Fatal(err)
	}

	req, err := Await[SendTradeRequest](ctx, c, nil)
	if err != nil || req.SenderID != testOtherID {
		t.Fatalf("SendTradeRequest = %+v, %v", req, err)
	}
	ts, err := c.AcceptTrade(ctx)
	if err != nil || ts.PartnerID != testOtherID || len(ts.Items) != 1 {
		t.Fatalf("AcceptTrade = %+v, %v", ts, err)
	}
	add, err := c.AddTradeItem(ctx, sword.ObjectID, 1)
	if err != nil || add.Item != own {
		t.Fatalf("AddTradeItem = %+v, %v", add, err)
	}
	if u, err := Await[TradeUpdate](ctx, c, nil); err != nil || len(u.Items) != 1 || u.Items[0].Change != TradeUpdateRemove {
		t.Fatalf("TradeUpdate = %+v, %v", u, err)
	}
	if oa, err := Await[TradeOtherAdd](ctx, c, nil); err != nil || oa.Item != other {
		t.Fatalf("TradeOtherAdd = %+v, %v", oa, err)
	}
	if _, err := Await[TradeOtherDone](ctx, c, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.ConfirmTrade(ctx); err != nil {
		t.Fatal(err)
	}
	if d, err := Await[TradeDone](ctx, c, nil); err != nil || d.Result != TradeDoneFinished {
		t.Fatalf("TradeDone = %+v, %v", d, err)
	}

	if _, err := Await[SendTradeRequest](ctx, c, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.AnswerTradeRequest(ctx, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Await[SendTradeRequest](ctx, c, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AcceptTrade(ctx); err != nil {
		t.Fatal(err)
	}
	d, err := c.CancelTrade(ctx)
	if err != nil || d.Result != TradeDoneCanceled {
		t.Fatalf("CancelTrade = %+v, %v", d, err)
	}
}

// TestPrivateStoreFlow opens a sell and a buy store, then buys from and
// sells to another player's stores, against a fake server.
func TestPrivateStoreFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	inv := testItems()
	sword := inv[2]
	avail := StoreSellItem{ObjectID: sword.ObjectID, ItemID: sword.ItemID, Count: 1, BodyPart: sword.BodyPart, Price: 138}
	manageSell := PrivateStoreManageListSell{ObjectID: testMeID, Adena: 5000, Available: []StoreSellItem{avail}}
	manageBuy := PrivateStoreManageListBuy{ObjectID: testMeID, Adena: 5000,
		Inventory: []StoreBuyManageItem{{ItemID: sword.ItemID, Count: 1, ReferencePrice: 138, BodyPart: sword.BodyPart}}}
	offer := []StoreSellOffer{{ObjectID: sword.ObjectID, Count: 1, Price: 500000}}
	wanted := []StoreBuyOffer{{ItemID: 1835, Count: 5000, Price: 8}}
	theirSell := PrivateStoreListSell{SellerID: testOtherID, Adena: 5000, Items: []StoreSellItem{
		{Type2: 5, ObjectID: 0x10000117, ItemID: 1835, Count: 300, Price: 8, ReferencePrice: 7},
	}}
	theirBuy := PrivateStoreListBuy{StorePlayerID: testOtherID, Adena: 5000, Items: []StoreBuyItem{
		{ObjectID: sword.ObjectID, ItemID: sword.ItemID, Count: 1, ReferencePrice: 138, BodyPart: sword.BodyPart, Price: 1000, StoreCount: 1},
	}}

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, inv)

		// A sell store: the setup window, the message, the list.
		f.expect(mobiusActionUse(10))
		f.send(mobiusPrivateStoreManageListSell(manageSell))
		f.expect(msg(0x77).str("Sword").b)
		f.send(msg(0x9c).i32(testMeID).str("Sword").b)
		f.expect(mobiusSetPrivateStoreListSell(false, offer...))
		f.send(mobiusUserInfo(testUserInfoAt(pos)))
		f.send(msg(0x9c).i32(testOtherID).str("Not mine").b)
		f.send(msg(0x9c).i32(testMeID).str("Sword").b)
		f.expect([]byte{0x76})

		// A buy store.
		f.expect(mobiusActionUse(28))
		f.send(mobiusPrivateStoreManageListBuy(manageBuy))
		f.expect(msg(0x94).str("WTB SS").b)
		f.expect(mobiusSetPrivateStoreListBuy(wanted...))
		f.send(mobiusUserInfo(testUserInfoAt(pos)))
		f.send(msg(0xb9).i32(testMeID).str("WTB SS").b)
		f.expect([]byte{0x93})

		// The setup windows again, from their own packets.
		f.expect([]byte{0x73})
		f.expect([]byte{0x90})

		// Buying from the other player's sell store.
		f.expect(mobiusAction(0x04, testOtherID, pos, false))
		f.send(mobiusPrivateStoreListSell(theirSell))
		f.expect(mobiusRequestPrivateStoreBuy(testOtherID, StorePurchase{0x10000117, 100, 8}))

		// Selling to the other player's buy store.
		f.expect(mobiusAction(0x04, testOtherID, pos, false))
		f.send(mobiusPrivateStoreListBuy(theirBuy))
		f.expect(mobiusRequestPrivateStoreSell(testOtherID, StoreSale{sword.ObjectID, sword.ItemID, 1, 1000}))
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)

	ms, err := c.OpenSellStoreSetup(ctx)
	if err != nil || ms.ObjectID != testMeID || len(ms.Available) != 1 {
		t.Fatalf("OpenSellStoreSetup = %+v, %v", ms, err)
	}
	it, ok := ms.FindAvailable(sword.ObjectID)
	if !ok {
		t.Fatalf("the sword is not available: %+v", ms)
	}
	if err := c.SetSellStoreMessage(ctx, "Sword"); err != nil {
		t.Fatal(err)
	}
	if _, err := Await[PrivateStoreMsgSell](ctx, c, nil); err != nil {
		t.Fatal(err)
	}
	m, err := c.OpenSellStore(ctx, false, []StoreSellOffer{{ObjectID: it.ObjectID, Count: it.Count, Price: 500000}})
	if err != nil || m.ObjectID != testMeID || m.Message != "Sword" {
		t.Fatalf("OpenSellStore = %+v, %v", m, err)
	}
	if err := c.QuitSellStore(ctx); err != nil {
		t.Fatal(err)
	}

	mb, err := c.OpenBuyStoreSetup(ctx)
	if err != nil || len(mb.Inventory) != 1 {
		t.Fatalf("OpenBuyStoreSetup = %+v, %v", mb, err)
	}
	if err := c.SetBuyStoreMessage(ctx, "WTB SS"); err != nil {
		t.Fatal(err)
	}
	bm, err := c.OpenBuyStore(ctx, wanted)
	if err != nil || bm.Message != "WTB SS" {
		t.Fatalf("OpenBuyStore = %+v, %v", bm, err)
	}
	if err := c.QuitBuyStore(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.RequestPrivateStoreManageSell(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.RequestPrivateStoreManageBuy(ctx); err != nil {
		t.Fatal(err)
	}

	ls, err := c.ViewSellStore(ctx, testOtherID)
	if err != nil || ls.SellerID != testOtherID || len(ls.Items) != 1 {
		t.Fatalf("ViewSellStore = %+v, %v", ls, err)
	}
	ss := ls.Items[0]
	if err := c.BuyFromStore(ctx, ls.SellerID, []StorePurchase{{ObjectID: ss.ObjectID, Count: 100, Price: ss.Price}}); err != nil {
		t.Fatal(err)
	}

	lb, err := c.ViewBuyStore(ctx, testOtherID)
	if err != nil || lb.StorePlayerID != testOtherID {
		t.Fatalf("ViewBuyStore = %+v, %v", lb, err)
	}
	bi, ok := lb.FindItem(sword.ItemID)
	if !ok {
		t.Fatalf("the sword is not wanted: %+v", lb)
	}
	if err := c.SellToStore(ctx, lb.StorePlayerID, []StoreSale{{ObjectID: bi.ObjectID, ItemID: bi.ItemID, Count: bi.Count, Price: bi.Price}}); err != nil {
		t.Fatal(err)
	}
}
