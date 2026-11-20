package testclient

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The Mobius server packets of trades and private stores, transcribed from
// their writeImpl line by line.

// mobiusTradeStart is TradeStart.writeImpl.
func mobiusTradeStart(s TradeStart) []byte {
	w := msg(0x1e)
	w.i32(s.PartnerID)
	w.u16(uint16(len(s.Items)))
	for _, it := range s.Items {
		w.u16(it.Type1)
		w.i32(it.ObjectID)
		w.i32(it.ItemID) // getId
		w.i32(it.Count)
		w.u16(it.Type2)
		w.u16(it.CustomType1) // ?: 0
		w.i32(it.BodyPart)
		w.u16(it.Enchant)
		w.u16(it.Reserved) // 0
		w.u16(it.CustomType2)
	}
	return w.b
}

// mobiusTradeAdd is TradeOwnAdd.writeImpl (0x20) and
// TradeOtherAdd.writeImpl (0x21).
func mobiusTradeAdd(op byte, it TradeItem) []byte {
	w := msg(op)
	w.u16(1) // items added count
	w.u16(it.Type1)
	w.i32(it.ObjectID)
	w.i32(it.ItemID) // display ID
	w.i32(it.Count)
	w.u16(it.Type2)
	w.u16(it.CustomType1)
	w.i32(it.BodyPart)
	w.u16(it.Enchant)
	w.u16(it.Reserved) // 0
	w.u16(it.CustomType2)
	return w.b
}

// mobiusTradeUpdate is TradeUpdate.writeImpl.
func mobiusTradeUpdate(u TradeUpdate) []byte {
	w := msg(0x74)
	w.u16(uint16(len(u.Items)))
	for _, it := range u.Items {
		w.u16(it.Change) // stackable ? 3 : 2
		w.u16(it.Type1)
		w.i32(it.ObjectID)
		w.i32(it.ItemID)
		w.i32(it.Count) // aveCount
		w.u16(it.Type2)
		w.u16(it.Reserved1) // 0
		w.i32(it.BodyPart)
		w.u16(it.Enchant)
		w.u16(it.Reserved2) // 0
		w.u16(it.Reserved3) // 0
	}
	return w.b
}

// mobiusStoreSellItem writes an entry of PrivateStoreManageListSell or
// PrivateStoreListSell; withReference adds the reference price.
func mobiusStoreSellItem(w *wire, it StoreSellItem, withReference bool) {
	w.i32(it.Type2)
	w.i32(it.ObjectID)
	w.i32(it.ItemID)
	w.i32(it.Count)
	w.u16(it.Reserved) // 0
	w.u16(it.Enchant)
	w.u16(it.CustomType2) // 0 in the sell list section of the manage list
	w.i32(it.BodyPart)
	w.i32(it.Price)
	if withReference {
		w.i32(it.ReferencePrice)
	}
}

// mobiusPrivateStoreManageListSell is PrivateStoreManageListSell.writeImpl.
func mobiusPrivateStoreManageListSell(l PrivateStoreManageListSell) []byte {
	w := msg(0x9a)
	w.i32(l.ObjectID)
	w.i32(b2i(l.PackageSale))
	w.i32(l.Adena)
	w.i32(int32(len(l.Available)))
	for _, it := range l.Available {
		mobiusStoreSellItem(w, it, false)
	}
	w.i32(int32(len(l.Items)))
	for _, it := range l.Items {
		mobiusStoreSellItem(w, it, true)
	}
	return w.b
}

// mobiusPrivateStoreListSell is PrivateStoreListSell.writeImpl.
func mobiusPrivateStoreListSell(l PrivateStoreListSell) []byte {
	w := msg(0x9b)
	w.i32(l.SellerID)
	w.i32(b2i(l.PackageSale))
	w.i32(l.Adena)
	w.i32(int32(len(l.Items)))
	for _, it := range l.Items {
		mobiusStoreSellItem(w, it, true)
	}
	return w.b
}

// mobiusPrivateStoreManageListBuy is PrivateStoreManageListBuy.writeImpl.
func mobiusPrivateStoreManageListBuy(l PrivateStoreManageListBuy) []byte {
	w := msg(0xb7)
	w.i32(l.ObjectID)
	w.i32(l.Adena)
	w.i32(int32(len(l.Inventory)))
	for _, it := range l.Inventory {
		w.i32(it.ItemID).u16(it.Enchant).i32(it.Count).i32(it.ReferencePrice)
		w.u16(it.Reserved).i32(it.BodyPart).u16(it.Type2)
	}
	w.i32(int32(len(l.Items)))
	for _, it := range l.Items {
		w.i32(it.ItemID).u16(it.Enchant).i32(it.Count).i32(it.ReferencePrice)
		w.u16(it.Reserved).i32(it.BodyPart).u16(it.Type2)
		w.i32(it.Price)      // your price
		w.i32(it.StorePrice) // fixed store price
	}
	return w.b
}

// mobiusPrivateStoreListBuy is PrivateStoreListBuy.writeImpl.
func mobiusPrivateStoreListBuy(l PrivateStoreListBuy) []byte {
	w := msg(0xb8)
	w.i32(l.StorePlayerID)
	w.i32(l.Adena)
	w.i32(int32(len(l.Items)))
	for _, it := range l.Items {
		w.i32(it.ObjectID)
		w.i32(it.ItemID)
		w.u16(it.Enchant)
		w.i32(it.Count) // max possible sell amount
		w.i32(it.ReferencePrice)
		w.u16(it.Reserved) // 0
		w.i32(it.BodyPart)
		w.u16(it.Type2)
		w.i32(it.Price)      // buyer's price
		w.i32(it.StoreCount) // maximum possible trade count
	}
	return w.b
}

// Test entries of the trade and store tests.
var (
	testTradeSword = TradeItem{Type1: 0, ObjectID: 0x10000016, ItemID: 2369, Count: 1, Type2: 0,
		CustomType1: 11, BodyPart: 0x4000, Enchant: 3, CustomType2: 12}
	testTradeShots = TradeItem{Type1: 4, ObjectID: 0x10000017, ItemID: 1835, Count: 300, Type2: 5}
	testStoreSword = StoreSellItem{Type2: 0, ObjectID: 0x10000016, ItemID: 2369, Count: 1, Enchant: 3,
		CustomType2: 12, BodyPart: 0x4000, Price: 500000, ReferencePrice: 138}
	testStoreShots = StoreSellItem{Type2: 5, ObjectID: 0x10000017, ItemID: 1835, Count: 300, Price: 8, ReferencePrice: 7}
)

func TestDecodeTradePackets(t *testing.T) {
	update := TradeUpdate{Items: []TradeUpdateItem{
		{Change: TradeUpdateRemove, ObjectID: 0x10000016, ItemID: 2369, Count: 1, BodyPart: 0x4000, Enchant: 3},
		{Change: TradeUpdateModify, Type1: 4, ObjectID: 0x10000017, ItemID: 1835, Count: 180, Type2: 5},
	}}
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"TradeStart", mobiusTradeStart(TradeStart{PartnerID: testOtherID, Items: []TradeItem{testTradeSword, testTradeShots}}),
			TradeStart{PartnerID: testOtherID, Items: []TradeItem{testTradeSword, testTradeShots}}},
		{"TradeStart empty", mobiusTradeStart(TradeStart{PartnerID: 5}), TradeStart{PartnerID: 5}},
		{"TradeOwnAdd", mobiusTradeAdd(0x20, testTradeSword), TradeOwnAdd{Item: testTradeSword}},
		{"TradeOtherAdd", mobiusTradeAdd(0x21, testTradeShots), TradeOtherAdd{Item: testTradeShots}},
		// TradeDone.writeImpl: d num.
		{"TradeDone canceled", msg(0x22).i32(0).b, TradeDone{Result: TradeDoneCanceled}},
		{"TradeDone finished", msg(0x22).i32(1).b, TradeDone{Result: TradeDoneFinished}},
		// SendTradeRequest.writeImpl: d sender.
		{"SendTradeRequest", msg(0x5e).i32(testOtherID).b, SendTradeRequest{SenderID: testOtherID}},
		{"TradeUpdate", mobiusTradeUpdate(update), update},
		{"TradeUpdate empty", mobiusTradeUpdate(TradeUpdate{}), TradeUpdate{}},
		// TradeOtherDone.writeImpl: the opcode only.
		{"TradeOtherDone", msg(0x7c).b, TradeOtherDone{}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}
}

func TestDecodePrivateStorePackets(t *testing.T) {
	avail := testStoreShots
	avail.Count, avail.Price, avail.ReferencePrice = 200, 7, 0
	inList := testStoreShots
	inList.Count = 100
	manageSell := PrivateStoreManageListSell{ObjectID: testMeID, PackageSale: true, Adena: 1000,
		Available: []StoreSellItem{testStoreSword, avail}, Items: []StoreSellItem{inList}}
	manageSell.Available[0].ReferencePrice = 0
	listSell := PrivateStoreListSell{SellerID: testOtherID, Adena: 5000, Items: []StoreSellItem{testStoreSword, testStoreShots}}
	manageBuy := PrivateStoreManageListBuy{ObjectID: testMeID, Adena: 50000,
		Inventory: []StoreBuyManageItem{{ItemID: 1835, Count: 300, ReferencePrice: 7, Type2: 5}},
		Items:     []StoreBuyManageItem{{ItemID: 1835, Count: 5000, ReferencePrice: 7, Type2: 5, Price: 8, StorePrice: 7}}}
	listBuy := PrivateStoreListBuy{StorePlayerID: testOtherID, Adena: 77, Items: []StoreBuyItem{
		{ObjectID: 0x10000017, ItemID: 1835, Enchant: 1, Count: 300, ReferencePrice: 7, Type2: 5, Price: 8, StoreCount: 5000},
		{ItemID: 2369, ReferencePrice: 138, BodyPart: 0x4000, Price: 500000, StoreCount: 1},
	}}
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"PrivateStoreManageListSell", mobiusPrivateStoreManageListSell(manageSell), manageSell},
		{"PrivateStoreManageListSell empty", mobiusPrivateStoreManageListSell(PrivateStoreManageListSell{ObjectID: 1}),
			PrivateStoreManageListSell{ObjectID: 1}},
		{"PrivateStoreListSell", mobiusPrivateStoreListSell(listSell), listSell},
		{"PrivateStoreListSell package", mobiusPrivateStoreListSell(PrivateStoreListSell{SellerID: 2, PackageSale: true}),
			PrivateStoreListSell{SellerID: 2, PackageSale: true}},
		{"PrivateStoreManageListBuy", mobiusPrivateStoreManageListBuy(manageBuy), manageBuy},
		{"PrivateStoreListBuy", mobiusPrivateStoreListBuy(listBuy), listBuy},
		// PrivateStoreMsgSell/Buy.writeImpl: d object ID, S message.
		{"PrivateStoreMsgSell", msg(0x9c).i32(testOtherID).str("Cheap SS").b,
			PrivateStoreMsgSell{ObjectID: testOtherID, Message: "Cheap SS"}},
		{"PrivateStoreMsgSell null title", msg(0x9c).i32(testOtherID).u16(0).b, PrivateStoreMsgSell{ObjectID: testOtherID}},
		{"PrivateStoreMsgBuy", msg(0xb9).i32(testOtherID).str("WTB D").b,
			PrivateStoreMsgBuy{ObjectID: testOtherID, Message: "WTB D"}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}

	if it, ok := manageSell.FindAvailable(0x10000017); !ok || it.Count != 200 {
		t.Errorf("FindAvailable = %+v, %v", it, ok)
	}
	if it, ok := listSell.Find(0x10000016); !ok || it.Price != 500000 {
		t.Errorf("Find = %+v, %v", it, ok)
	}
	if _, ok := listSell.Find(1); ok {
		t.Error("Find(1) found an item")
	}
	if it, ok := listBuy.FindItem(2369); !ok || it.StoreCount != 1 {
		t.Errorf("FindItem = %+v, %v", it, ok)
	}
	if _, ok := listBuy.FindItem(57); ok {
		t.Error("FindItem(57) found an item")
	}
	ts := TradeStart{Items: []TradeItem{testTradeSword}}
	if it, ok := ts.Find(0x10000016); !ok || it.ItemID != 2369 {
		t.Errorf("TradeStart.Find = %+v, %v", it, ok)
	}
	if _, ok := ts.Find(1); ok {
		t.Error("TradeStart.Find(1) found an item")
	}
}

// TestTradePacketSizes checks the test builders against body sizes counted
// by hand from the Mobius writeImpl.
func TestTradePacketSizes(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		size int
	}{
		// d h, then 28 bytes per item: h d d d h h d h h h.
		{"TradeStart of 2", mobiusTradeStart(TradeStart{Items: make([]TradeItem, 2)}), 6 + 2*28},
		// h, then h d d d h h d h h h.
		{"TradeOwnAdd", mobiusTradeAdd(0x20, TradeItem{}), 30},
		// h, then 30 bytes per item: h h d d d h h d h h h.
		{"TradeUpdate of 3", mobiusTradeUpdate(TradeUpdate{Items: make([]TradeUpdateItem, 3)}), 2 + 3*30},
		// d d d, d + 30 per available item (d d d d h h h d d), d + 34 per
		// listed item (the same and d).
		{"PrivateStoreManageListSell", mobiusPrivateStoreManageListSell(PrivateStoreManageListSell{
			Available: make([]StoreSellItem, 2), Items: make([]StoreSellItem, 1)}), 12 + 4 + 2*30 + 4 + 34},
		// d d d d, then 34 bytes per item.
		{"PrivateStoreListSell of 2", mobiusPrivateStoreListSell(PrivateStoreListSell{Items: make([]StoreSellItem, 2)}), 16 + 2*34},
		// d d, d + 22 per inventory item (d h d d h d h), d + 30 per listed
		// item (the same and d d).
		{"PrivateStoreManageListBuy", mobiusPrivateStoreManageListBuy(PrivateStoreManageListBuy{
			Inventory: make([]StoreBuyManageItem, 3), Items: make([]StoreBuyManageItem, 2)}), 8 + 4 + 3*22 + 4 + 2*30},
		// d d d, then 34 bytes per item: d d h d d h d h d d.
		{"PrivateStoreListBuy of 2", mobiusPrivateStoreListBuy(PrivateStoreListBuy{Items: make([]StoreBuyItem, 2)}), 12 + 2*34},
	}
	for _, tt := range tests {
		if got := len(tt.body) - 1; got != tt.size {
			t.Errorf("%s body is %d bytes, Mobius writes %d", tt.name, got, tt.size)
		}
	}
}

// TestTradeOwnAddLayout writes TradeOwnAdd byte by byte.
func TestTradeOwnAddLayout(t *testing.T) {
	body := []byte{
		0x20,
		0x01, 0x00, // one item
		0x00, 0x00, // type1
		0x16, 0x00, 0x00, 0x10, // object 0x10000016
		0x41, 0x09, 0x00, 0x00, // item 2369
		0x01, 0x00, 0x00, 0x00, // count 1
		0x00, 0x00, // type2
		0x0b, 0x00, // custom type 1
		0x00, 0x40, 0x00, 0x00, // body part 0x4000
		0x03, 0x00, // enchant 3
		0x00, 0x00, // 0
		0x0c, 0x00, // custom type 2
	}
	checkDecode(t, "TradeOwnAdd", body, TradeOwnAdd{Item: testTradeSword})
	if !bytes.Equal(mobiusTradeAdd(0x20, testTradeSword), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusTradeAdd(0x20, testTradeSword), body)
	}
}

// TestPrivateStoreListBuyLayout writes a PrivateStoreListBuy entry byte by
// byte: the enchant short sits between the item ID and the count.
func TestPrivateStoreListBuyLayout(t *testing.T) {
	body := []byte{
		0xb8,
		0x02, 0x00, 0x00, 0x10, // store 0x10000002
		0x4d, 0x00, 0x00, 0x00, // adena 77
		0x01, 0x00, 0x00, 0x00, // one item
		0x17, 0x00, 0x00, 0x10, // object 0x10000017
		0x2b, 0x07, 0x00, 0x00, // item 1835
		0x01, 0x00, // enchant 1
		0x2c, 0x01, 0x00, 0x00, // count 300
		0x07, 0x00, 0x00, 0x00, // reference price 7
		0x00, 0x00, // 0
		0x00, 0x00, 0x00, 0x00, // body part
		0x05, 0x00, // type2
		0x08, 0x00, 0x00, 0x00, // price 8
		0x88, 0x13, 0x00, 0x00, // store count 5000
	}
	want := PrivateStoreListBuy{StorePlayerID: testOtherID, Adena: 77, Items: []StoreBuyItem{
		{ObjectID: 0x10000017, ItemID: 1835, Enchant: 1, Count: 300, ReferencePrice: 7, Type2: 5, Price: 8, StoreCount: 5000},
	}}
	checkDecode(t, "PrivateStoreListBuy", body, want)
	if !bytes.Equal(mobiusPrivateStoreListBuy(want), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusPrivateStoreListBuy(want), body)
	}
}

func TestDecodeTradePacketErrors(t *testing.T) {
	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"TradeOwnAdd two items", append(msg(0x20).u16(2).b, mobiusTradeAdd(0x20, testTradeSword)[3:]...),
			"TradeOwnAdd: 2 items, Mobius writes 1"},
		{"TradeOtherAdd short", mobiusTradeAdd(0x21, testTradeSword)[:20], "TradeOtherAdd: packets: short packet"},
		{"TradeStart count too high", msg(0x1e).i32(1).u16(1).b, "TradeStart: packets: short packet"},
		{"TradeDone extra byte", msg(0x22).i32(1).u8(0).b, "TradeDone: 1 bytes after the last field"},
		{"TradeOtherDone extra byte", msg(0x7c).u8(0).b, "TradeOtherDone: 1 bytes after the last field"},
		{"TradeUpdate cut entry", mobiusTradeUpdate(TradeUpdate{Items: make([]TradeUpdateItem, 1)})[:20],
			"TradeUpdate: packets: short packet"},
		{"PrivateStoreListSell package 2", msg(0x9b).i32(1).i32(2).i32(0).i32(0).b,
			"PrivateStoreListSell: package sale flag is 2"},
		{"PrivateStoreListSell negative count", msg(0x9b).i32(1).i32(0).i32(0).i32(-1).b,
			"PrivateStoreListSell: item count -1"},
		{"PrivateStoreManageListSell without the sell list", msg(0x9a).i32(1).i32(0).i32(0).i32(0).b,
			"PrivateStoreManageListSell: packets: short packet"},
		{"PrivateStoreManageListSell negative sell list", msg(0x9a).i32(1).i32(0).i32(0).i32(0).i32(-2).b,
			"PrivateStoreManageListSell: sell list count -2"},
		{"PrivateStoreManageListBuy negative inventory", msg(0xb7).i32(1).i32(0).i32(-1).i32(0).b,
			"PrivateStoreManageListBuy: inventory count -1"},
		{"PrivateStoreListBuy item missing", msg(0xb8).i32(1).i32(0).i32(1).b, "PrivateStoreListBuy: packets: short packet"},
		{"PrivateStoreMsgBuy without a terminator", msg(0xb9).i32(1).u16('A').b, "PrivateStoreMsgBuy"},
	}
	for _, tt := range tests {
		_, err := DecodeServerPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
			continue
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v does not wrap ErrMalformed", tt.name, err)
		}
	}
}
