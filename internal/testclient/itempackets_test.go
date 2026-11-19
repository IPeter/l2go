package testclient

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The Mobius server packets of Milestone 5, transcribed from their
// writeImpl line by line.

// mobiusSpawnItem is SpawnItem.writeImpl.
func mobiusSpawnItem(s SpawnItem) []byte {
	w := msg(0x0b)
	w.i32(s.ObjectID)
	w.i32(s.ItemID) // display ID
	w.i32(s.X).i32(s.Y).i32(s.Z)
	w.i32(b2i(s.Stackable)) // only show the count of a stackable item
	w.i32(s.Count)
	w.i32(s.Unknown) // c2: 0
	return w.b
}

// mobiusDropItem is DropItem.writeImpl.
func mobiusDropItem(d DropItem) []byte {
	w := msg(0x0c)
	w.i32(d.DropperID)
	w.i32(d.ObjectID)
	w.i32(d.ItemID) // display ID
	w.i32(d.X).i32(d.Y).i32(d.Z)
	w.i32(b2i(d.Stackable))
	w.i32(d.Count)
	w.i32(d.Unknown) // 1
	return w.b
}

// mobiusGetItem is GetItem.writeImpl.
func mobiusGetItem(g GetItem) []byte {
	return msg(0x0d).i32(g.PlayerID).i32(g.ObjectID).i32(g.X).i32(g.Y).i32(g.Z).b
}

// mobiusEquipUpdate is EquipUpdate.writeImpl: d change, d object, d the
// slot number of the body part.
func mobiusEquipUpdate(e EquipUpdate) []byte {
	return msg(0x4b).i32(e.Change).i32(e.ObjectID).i32(e.Slot).b
}

// mobiusBuyList is BuyList.writeImpl. listed is the product count Mobius
// writes, that of the whole buylist; items are the products it writes.
func mobiusBuyList(adena, listID int32, listed int, items []BuyListItem) []byte {
	w := msg(0x11)
	w.i32(adena) // current money
	w.i32(listID)
	w.u16(uint16(listed))
	for _, it := range items {
		w.u16(it.Type1)
		w.i32(it.ObjectID) // 0
		w.i32(it.ItemID)
		w.i32(it.Count) // stock left, 0 without a limit
		w.u16(it.Type2)
		w.u16(it.Equipped) // isEquipped: 0
		w.i32(it.BodyPart) // 0 for type 1 TYPE1_ITEM_QUESTITEM_ADENA
		w.u16(it.Enchant)  // 0
		w.u16(it.Reserved1)
		w.u16(it.Reserved2)
		w.i32(it.Price) // price * (1 + tax)
	}
	return w.b
}

// mobiusSellList is SellList.writeImpl.
func mobiusSellList(l SellList) []byte {
	w := msg(0x10)
	w.i32(l.Adena)
	w.i32(l.ListID) // 0
	w.u16(uint16(len(l.Items)))
	for _, it := range l.Items {
		w.u16(it.Type1)
		w.i32(it.ObjectID)
		w.i32(it.ItemID)
		w.i32(it.Count)
		w.u16(it.Type2)
		w.u16(it.CustomType1)
		w.i32(it.BodyPart)
		w.u16(it.Enchant)
		w.u16(it.CustomType2)
		w.u16(it.Reserved) // 0
		w.i32(it.Price)    // reference price / 2
	}
	return w.b
}

// mobiusMultiSellList is MultiSellList.writeImpl.
func mobiusMultiSellList(l MultiSellList) []byte {
	w := msg(0xd0)
	w.i32(l.ListID)
	w.i32(l.Page) // page started from 1
	w.i32(b2i(l.Finished))
	w.i32(l.PageSize) // 40
	w.i32(int32(len(l.Entries)))
	for _, e := range l.Entries {
		w.i32(e.EntryID)
		w.i32(e.Reserved1) // C6: 0
		w.i32(e.Reserved2) // C6: 0
		w.u8(e.Reserved3)  // 1
		w.u16(uint16(len(e.Products)))
		w.u16(uint16(len(e.Ingredients)))
		for _, p := range e.Products {
			w.u16(uint16(p.ItemID)) // display ID as a short
			w.i32(p.BodyPart)
			w.u16(p.Type2) // 65535 without a template
			w.i32(p.Count)
			w.u16(p.Enchant)
			w.i32(p.AugmentationID)
			w.i32(p.Mana) // 0
		}
		for _, in := range e.Ingredients {
			w.u16(uint16(in.ItemID))
			w.u16(in.Type2)
			w.i32(in.Count)
			w.u16(in.Enchant)
			w.i32(in.AugmentationID)
			w.i32(in.Mana)
		}
	}
	return w.b
}

// mobiusWarehouseList is WareHouseDepositList.writeImpl (op 0x41) and
// WareHouseWithdrawalList.writeImpl (op 0x42).
func mobiusWarehouseList(op byte, l WarehouseList) []byte {
	w := msg(op)
	w.u16(l.Type)
	w.i32(l.Adena)
	w.u16(uint16(len(l.Items)))
	for _, it := range l.Items {
		w.u16(it.Type1)
		w.i32(it.ObjectID)
		w.i32(it.ItemID)
		w.i32(it.Count)
		w.u16(it.Type2)
		w.u16(it.CustomType1)
		w.i32(it.BodyPart)
		w.u16(it.Enchant)
		w.u16(it.Reserved) // 0
		w.u16(it.CustomType2)
		w.i32(it.ObjectID2) // the object ID again
		if it.AugmentationLow == 0 && it.AugmentationHigh == 0 {
			w.i64(0) // not augmented
		} else {
			w.i32(it.AugmentationLow).i32(it.AugmentationHigh)
		}
	}
	return w.b
}

// Test products and items of the M5 tests.
var (
	testShortSword = BuyListItem{Type1: 0, ItemID: 1, Type2: 0, BodyPart: 0x0080, Price: 649}
	testPotion     = BuyListItem{Type1: 4, ItemID: 1060, Type2: 5, Price: 41}
)

func TestDecodeItemPackets(t *testing.T) {
	sellItem := SellListItem{Type1: 4, ObjectID: 0x10000015, ItemID: 1060, Count: 10, Type2: 5, Price: 20}
	sword := SellListItem{Type1: 0, ObjectID: 0x10000016, ItemID: 1, Count: 1, Type2: 0, BodyPart: 0x0080, Enchant: 3, Price: 245}
	stored := WarehouseListItem{Type1: 0, ObjectID: 0x10000020, ItemID: 2369, Count: 1, BodyPart: 0x4000,
		ObjectID2: 0x10000020, AugmentationLow: 0x1234, AugmentationHigh: 0x0005}
	adena := WarehouseListItem{Type1: 4, ObjectID: 0x10000014, ItemID: 57, Count: 5000, Type2: 4, ObjectID2: 0x10000014}

	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"SpawnItem", mobiusSpawnItem(SpawnItem{ObjectID: 0x10000030, ItemID: 57, X: 1, Y: -2, Z: 3, Stackable: true, Count: 120}),
			SpawnItem{ObjectID: 0x10000030, ItemID: 57, X: 1, Y: -2, Z: 3, Stackable: true, Count: 120}},
		{"SpawnItem not stackable", mobiusSpawnItem(SpawnItem{ObjectID: 0x10000031, ItemID: 1, Count: 1}),
			SpawnItem{ObjectID: 0x10000031, ItemID: 1, Count: 1}},
		{"DropItem", mobiusDropItem(DropItem{DropperID: 0x60000001, ObjectID: 0x10000030, ItemID: 57, X: 10, Y: 20, Z: -30,
			Stackable: true, Count: 25, Unknown: 1}),
			DropItem{DropperID: 0x60000001, ObjectID: 0x10000030, ItemID: 57, X: 10, Y: 20, Z: -30, Stackable: true, Count: 25, Unknown: 1}},
		{"GetItem", mobiusGetItem(GetItem{PlayerID: 0x10000001, ObjectID: 0x10000030, X: 10, Y: 20, Z: -30}),
			GetItem{PlayerID: 0x10000001, ObjectID: 0x10000030, X: 10, Y: 20, Z: -30}},
		{"EquipUpdate", mobiusEquipUpdate(EquipUpdate{Change: 1, ObjectID: 0x10000016, Slot: 7}),
			EquipUpdate{Change: 1, ObjectID: 0x10000016, Slot: 7}},
		// EnchantResult.writeImpl: d result.
		{"EnchantResult", msg(0x81).i32(0).b, EnchantResult{}},
		{"EnchantResult 2", msg(0x81).i32(2).b, EnchantResult{Result: 2}},
		{"BuyList", mobiusBuyList(5000, 3, 2, []BuyListItem{testShortSword, testPotion}),
			BuyList{Adena: 5000, ListID: 3, Listed: 2, Items: []BuyListItem{testShortSword, testPotion}}},
		{"BuyList empty", mobiusBuyList(0, 7, 0, nil), BuyList{ListID: 7}},
		{"SellList", mobiusSellList(SellList{Adena: 5000, Items: []SellListItem{sellItem, sword}}),
			SellList{Adena: 5000, Items: []SellListItem{sellItem, sword}}},
		{"SellList empty", mobiusSellList(SellList{Adena: 1}), SellList{Adena: 1}},
		{"WareHouseDepositList", mobiusWarehouseList(0x41, WarehouseList{Type: WarehousePrivate, Adena: 5000, Items: []WarehouseListItem{adena, stored}}),
			WareHouseDepositList{WarehouseList{Type: WarehousePrivate, Adena: 5000, Items: []WarehouseListItem{adena, stored}}}},
		{"WareHouseWithdrawalList", mobiusWarehouseList(0x42, WarehouseList{Type: WarehouseFreight, Adena: 7, Items: []WarehouseListItem{stored}}),
			WareHouseWithdrawalList{WarehouseList{Type: WarehouseFreight, Adena: 7, Items: []WarehouseListItem{stored}}}},
		{"WareHouseWithdrawalList empty", mobiusWarehouseList(0x42, WarehouseList{Type: WarehouseClan}),
			WareHouseWithdrawalList{WarehouseList{Type: WarehouseClan}}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}
}

// TestBuyListSoldOut checks the Mobius quirk of BuyList: the count is that
// of the whole buylist, but a sold-out product with a limited stock is not
// written.
func TestBuyListSoldOut(t *testing.T) {
	// Three products, the second one sold out: Mobius writes 3, then two
	// entries.
	limited := testPotion
	limited.Count = 4
	body := mobiusBuyList(100, 9, 3, []BuyListItem{testShortSword, limited})
	want := BuyList{Adena: 100, ListID: 9, Listed: 3, Items: []BuyListItem{testShortSword, limited}}
	checkDecode(t, "BuyList with a sold-out product", body, want)

	if it, ok := want.Find(1060); !ok || it.Count != 4 {
		t.Errorf("Find(1060) = %+v, %v", it, ok)
	}
	if _, ok := want.Find(2); ok {
		t.Error("Find found a product that is not there")
	}
}

// TestItemPacketSizes checks the test builders against body sizes counted
// by hand from the Mobius writeImpl, and that the zero packets decode to
// the zero value.
func TestItemPacketSizes(t *testing.T) {
	entry := MultiSellEntry{Reserved3: 1, Products: []MultiSellProduct{{}}, Ingredients: []MultiSellIngredient{{}, {}}}
	tests := []struct {
		name string
		body []byte
		size int
	}{
		{"SpawnItem", mobiusSpawnItem(SpawnItem{}), 32},
		{"DropItem", mobiusDropItem(DropItem{}), 36},
		{"GetItem", mobiusGetItem(GetItem{}), 20},
		{"EquipUpdate", mobiusEquipUpdate(EquipUpdate{}), 12},
		// d d h, then 32 bytes per product: h d d d h h d h h h d.
		{"BuyList empty", mobiusBuyList(0, 0, 0, nil), 10},
		{"BuyList of 2", mobiusBuyList(0, 0, 2, []BuyListItem{{}, {}}), 10 + 2*32},
		// d d h, then 32 bytes per item: h d d d h h d h h h d.
		{"SellList of 1", mobiusSellList(SellList{Items: []SellListItem{{}}}), 10 + 32},
		// d d d d d; an entry d d d c h h; a product h d h d h d d; an
		// ingredient h h d h d d.
		{"MultiSellList empty", mobiusMultiSellList(MultiSellList{}), 20},
		{"MultiSellList of 1", mobiusMultiSellList(MultiSellList{Entries: []MultiSellEntry{entry}}), 20 + 17 + 22 + 2*18},
		// h d h, then 40 bytes per item: h d d d h h d h h h d q.
		{"WareHouseDepositList empty", mobiusWarehouseList(0x41, WarehouseList{}), 8},
		{"WareHouseWithdrawalList of 2", mobiusWarehouseList(0x42, WarehouseList{Items: []WarehouseListItem{{}, {}}}), 8 + 2*40},
	}
	for _, tt := range tests {
		if got := len(tt.body) - 1; got != tt.size {
			t.Errorf("%s body is %d bytes, Mobius writes %d", tt.name, got, tt.size)
		}
	}
	zeros := []struct {
		body []byte
		zero Packet
	}{
		{mobiusSpawnItem(SpawnItem{}), SpawnItem{}},
		{mobiusDropItem(DropItem{}), DropItem{}},
		{mobiusGetItem(GetItem{}), GetItem{}},
		{mobiusEquipUpdate(EquipUpdate{}), EquipUpdate{}},
	}
	for _, tt := range zeros {
		checkDecode(t, tt.zero.PacketName(), tt.body, tt.zero)
	}
}

// TestDropItemLayout writes DropItem byte by byte: the dropper comes
// before the item.
func TestDropItemLayout(t *testing.T) {
	body := []byte{
		0x0c,
		0x05, 0x00, 0x00, 0x60, // dropper 0x60000005
		0x30, 0x00, 0x00, 0x10, // object 0x10000030
		0x39, 0x00, 0x00, 0x00, // item 57 (adena)
		0x10, 0x27, 0x00, 0x00, // x 10000
		0xe0, 0xb1, 0xff, 0xff, // y -20000
		0x9c, 0xff, 0xff, 0xff, // z -100
		0x01, 0x00, 0x00, 0x00, // stackable
		0x19, 0x00, 0x00, 0x00, // count 25
		0x01, 0x00, 0x00, 0x00, // 1
	}
	want := DropItem{DropperID: 0x60000005, ObjectID: 0x10000030, ItemID: 57, X: 10000, Y: -20000, Z: -100,
		Stackable: true, Count: 25, Unknown: 1}
	checkDecode(t, "DropItem", body, want)
	if !bytes.Equal(mobiusDropItem(want), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusDropItem(want), body)
	}
	if want.Position() != (Position{10000, -20000, -100}) {
		t.Errorf("Position %v", want.Position())
	}
}

// TestMultiSellListLayout writes a MultiSellList page byte by byte, with a
// product without a template (type 2 65535) and a special ingredient with
// a negative ID (written as a short).
func TestMultiSellListLayout(t *testing.T) {
	body := []byte{
		0xd0,
		0x5d, 0x00, 0x00, 0x00, // list 93
		0x02, 0x00, 0x00, 0x00, // page 2
		0x01, 0x00, 0x00, 0x00, // finished
		0x28, 0x00, 0x00, 0x00, // page size 40
		0x01, 0x00, 0x00, 0x00, // one entry
		// entry 100001: d, d 0, d 0, c 1, h 1 product, h 2 ingredients
		0xa1, 0x86, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x01,
		0x01, 0x00,
		0x02, 0x00,
		// product: h 4300, d 0, h 65535, d 1, h 0, d 0, d 0
		0xcc, 0x10,
		0x00, 0x00, 0x00, 0x00,
		0xff, 0xff,
		0x01, 0x00, 0x00, 0x00,
		0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		// ingredient: h 57, h 4, d 1000, h 0, d 0, d 0
		0x39, 0x00,
		0x04, 0x00,
		0xe8, 0x03, 0x00, 0x00,
		0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		// ingredient: h -200, h 65535, d 5, h 3, d 7, d 0
		0x38, 0xff,
		0xff, 0xff,
		0x05, 0x00, 0x00, 0x00,
		0x03, 0x00,
		0x07, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	want := MultiSellList{ListID: 93, Page: 2, Finished: true, PageSize: 40, Entries: []MultiSellEntry{{
		EntryID: 100001, Reserved3: 1,
		Products: []MultiSellProduct{{ItemID: 4300, Type2: MultiSellNoType2, Count: 1}},
		Ingredients: []MultiSellIngredient{
			{ItemID: 57, Type2: 4, Count: 1000},
			{ItemID: -200, Type2: MultiSellNoType2, Count: 5, Enchant: 3, AugmentationID: 7},
		},
	}}}
	checkDecode(t, "MultiSellList", body, want)
	if !bytes.Equal(mobiusMultiSellList(want), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusMultiSellList(want), body)
	}
}

// TestWarehouseListLayout checks the tail of a warehouse entry: the object
// ID again, then the augmentation as two words, or a zero long.
func TestWarehouseListLayout(t *testing.T) {
	head := msg(0x42).u16(1).i32(300).u16(1).
		u16(0).i32(0x10000020).i32(2369).i32(1).u16(0).u16(0).i32(0x4000).u16(4).u16(0).u16(0).b
	tests := []struct {
		name      string
		tail      []byte
		low, high int32
	}{
		{"not augmented", []byte{0x20, 0, 0, 0x10, 0, 0, 0, 0, 0, 0, 0, 0}, 0, 0},
		{"augmented", []byte{0x20, 0, 0, 0x10, 0x34, 0x12, 0, 0, 0x05, 0, 0, 0}, 0x1234, 5},
	}
	for _, tt := range tests {
		body := append(append([]byte(nil), head...), tt.tail...)
		want := WareHouseWithdrawalList{WarehouseList{Type: WarehousePrivate, Adena: 300, Items: []WarehouseListItem{{
			ObjectID: 0x10000020, ItemID: 2369, Count: 1, BodyPart: 0x4000, Enchant: 4,
			ObjectID2: 0x10000020, AugmentationLow: tt.low, AugmentationHigh: tt.high,
		}}}}
		checkDecode(t, "WareHouseWithdrawalList "+tt.name, body, want)
		if !bytes.Equal(mobiusWarehouseList(0x42, want.WarehouseList), body) {
			t.Errorf("%s: test builder differs:\n got % x\nwant % x", tt.name, mobiusWarehouseList(0x42, want.WarehouseList), body)
		}
		if it, ok := want.Find(0x10000020); !ok || it.ItemID != 2369 {
			t.Errorf("%s: Find = %+v, %v", tt.name, it, ok)
		}
		if it, ok := want.FindItem(2369); !ok || it.ObjectID != 0x10000020 {
			t.Errorf("%s: FindItem = %+v, %v", tt.name, it, ok)
		}
	}
}

func TestSellListFind(t *testing.T) {
	l := SellList{Items: []SellListItem{{ObjectID: 1, ItemID: 57}, {ObjectID: 2, ItemID: 1060}}}
	if it, ok := l.Find(2); !ok || it.ItemID != 1060 {
		t.Errorf("Find(2) = %+v, %v", it, ok)
	}
	if _, ok := l.Find(3); ok {
		t.Error("Find(3) found an item")
	}
}

func TestDecodeItemPacketErrors(t *testing.T) {
	spawn := mobiusSpawnItem(SpawnItem{})
	spawn[1+5*4] = 2 // the stackable word
	drop := mobiusDropItem(DropItem{})
	drop[1+6*4] = 3
	oneProduct := mobiusBuyList(0, 1, 1, []BuyListItem{testPotion})

	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"SpawnItem stackable 2", spawn, "SpawnItem: stackable flag is 2"},
		{"SpawnItem short", mobiusSpawnItem(SpawnItem{})[:30], "SpawnItem: packets: short packet"},
		{"DropItem stackable 3", drop, "DropItem: stackable flag is 3"},
		{"GetItem extra byte", msg(0x0d).i32(1).i32(2).i32(3).i32(4).i32(5).u8(0).b, "GetItem: 1 bytes after the last field"},
		{"EquipUpdate short", msg(0x4b).i32(1).i32(2).b, "EquipUpdate: packets: short packet"},
		{"BuyList more entries than products", mobiusBuyList(0, 1, 0, []BuyListItem{testPotion}),
			"BuyList: 32 bytes of entries for 0 products"},
		{"BuyList cut entry", oneProduct[:len(oneProduct)-1], "BuyList: 31 bytes of entries for 1 products"},
		{"BuyList without the count", msg(0x11).i32(0).i32(1).u8(0).b, "BuyList: packets: short packet"},
		{"SellList count too high", msg(0x10).i32(0).i32(0).u16(1).b, "SellList: packets: short packet"},
		{"MultiSellList finished 2", msg(0xd0).i32(1).i32(1).i32(2).i32(40).i32(0).b, "MultiSellList: finished flag is 2"},
		{"MultiSellList negative count", msg(0xd0).i32(1).i32(1).i32(1).i32(40).i32(-1).b, "MultiSellList: entry count -1"},
		{"MultiSellList entry missing", msg(0xd0).i32(1).i32(1).i32(1).i32(40).i32(1).b, "MultiSellList: packets: short packet"},
		{"MultiSellList ingredient missing", msg(0xd0).i32(1).i32(1).i32(1).i32(40).i32(1).
			i32(5).i32(0).i32(0).u8(1).u16(0).u16(1).b, "MultiSellList: packets: short packet"},
		{"WareHouseDepositList extra byte", append(mobiusWarehouseList(0x41, WarehouseList{}), 0),
			"WareHouseDepositList: 1 bytes after the last field"},
		{"WareHouseWithdrawalList item missing", msg(0x42).u16(1).i32(0).u16(1).b,
			"WareHouseWithdrawalList: packets: short packet"},
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
