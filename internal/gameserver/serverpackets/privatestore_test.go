package serverpackets

import "testing"

// The fixtures of the private store tests: every field holds a value of its
// own, so that a field sent in the wrong place, or one a packet must leave
// out, shows.

// tsSword is an enchanted sword offered for 500,000 adena.
func tsSword() StoreItem {
	return StoreItem{
		ObjectID: 0x10000002, ItemID: 2369, Count: 1, Type2: 0, BodyPart: 0x80,
		Enchant: 3, CustomType2: 12, Price: 500000, ReferencePrice: 138,
	}
}

// tsShots is a stack of soulshots.
func tsShots() StoreItem {
	return StoreItem{
		ObjectID: 0x10000003, ItemID: 1835, Count: 300, Type2: 5,
		Enchant: 1, CustomType2: 22, Price: 8, ReferencePrice: 7,
	}
}

// Mobius PrivateStoreManageListSell.writeImpl:
//
//	c 0x9A; d owner; d package sale; d adena
//	d available count; per item: d type2; d object ID; d item ID; d count;
//	  h 0; h enchant; h custom type 2; d body part; d price (reference)
//	d sell list count; per item: d type2; d object ID; d item ID; d count;
//	  h 0; h enchant; h 0; d body part; d price; d reference price
func TestPrivateStoreManageListSellPacket(t *testing.T) {
	sword, shots := tsSword(), tsShots()
	// 100 of the 300 are in the list already; Mobius puts the reference
	// price in the price field of the available items.
	avail := shots
	avail.Count = 200
	avail.Price = 7
	inList := shots
	inList.Count = 100
	l := PrivateStoreManageListSell{
		ObjectID: 0x10000001, PackageSale: true, Adena: 123456,
		Available: []StoreItem{sword, avail},
		Items:     []StoreItem{inList},
	}
	var g m3Golden
	g.c(0x9a).d(0x10000001).d(1).d(123456)
	g.d(2)
	g.d(0).d(0x10000002).d(2369).d(1).h(0).h(3).h(12).d(0x80).d(500000)
	g.d(5).d(0x10000003).d(1835).d(200).h(0).h(1).h(22).d(0).d(7)
	g.d(1)
	g.d(5).d(0x10000003).d(1835).d(100).h(0).h(1).h(0).d(0).d(8).d(7)
	m3Check(t, "PrivateStoreManageListSell", NewPrivateStoreManageListSellPacket(l), g.b)
	if n := len(g.b) - 21; n != 2*storeSellAvailableSize+storeSellItemSize {
		t.Errorf("entries of %d bytes", n)
	}

	m3Check(t, "empty PrivateStoreManageListSell",
		NewPrivateStoreManageListSellPacket(PrivateStoreManageListSell{ObjectID: 1, Adena: 2}),
		unhex(t, "9a 01000000 00000000 02000000 00000000 00000000"))
}

// Mobius PrivateStoreListSell.writeImpl:
//
//	c 0x9B; d seller; d packaged; d buyer's adena; d count
//	per item: d type2; d object ID; d item ID; d count; h 0; h enchant;
//	  h custom type 2; d body part; d price; d reference price
func TestPrivateStoreListSellPacket(t *testing.T) {
	l := PrivateStoreListSell{SellerID: 0x10000001, Adena: 999, Items: []StoreItem{tsSword(), tsShots()}}
	var g m3Golden
	g.c(0x9b).d(0x10000001).d(0).d(999).d(2)
	g.d(0).d(0x10000002).d(2369).d(1).h(0).h(3).h(12).d(0x80).d(500000).d(138)
	g.d(5).d(0x10000003).d(1835).d(300).h(0).h(1).h(22).d(0).d(8).d(7)
	m3Check(t, "PrivateStoreListSell", NewPrivateStoreListSellPacket(l), g.b)
	if n := len(g.b) - 17; n != 2*storeSellItemSize {
		t.Errorf("entries of %d bytes", n)
	}

	m3Check(t, "package PrivateStoreListSell",
		NewPrivateStoreListSellPacket(PrivateStoreListSell{SellerID: 3, PackageSale: true, Adena: 4}),
		unhex(t, "9b 03000000 01000000 04000000 00000000"))
}

// Mobius PrivateStoreManageListBuy.writeImpl:
//
//	c 0xB7; d owner; d adena
//	d inventory count; per item: d item ID; h 0; d count; d reference
//	  price; h 0; d body part; h type2
//	d buy list count; per item: d item ID; h 0; d count; d reference price;
//	  h 0; d body part; h type2; d price; d reference price
func TestPrivateStoreManageListBuyPacket(t *testing.T) {
	sword, shots := tsSword(), tsShots()
	wanted := shots
	wanted.Count = 5000
	wanted.ObjectID = 0
	l := PrivateStoreManageListBuy{
		ObjectID: 0x10000001, Adena: 50000,
		Inventory: []StoreItem{sword, shots},
		Items:     []StoreItem{wanted},
	}
	var g m3Golden
	g.c(0xb7).d(0x10000001).d(50000)
	g.d(2)
	g.d(2369).h(0).d(1).d(138).h(0).d(0x80).h(0)
	g.d(1835).h(0).d(300).d(7).h(0).d(0).h(5)
	g.d(1)
	g.d(1835).h(0).d(5000).d(7).h(0).d(0).h(5).d(8).d(7)
	m3Check(t, "PrivateStoreManageListBuy", NewPrivateStoreManageListBuyPacket(l), g.b)
	if n := len(g.b) - 17; n != 2*storeBuyInventorySize+storeBuyItemSize {
		t.Errorf("entries of %d bytes", n)
	}

	m3Check(t, "empty PrivateStoreManageListBuy",
		NewPrivateStoreManageListBuyPacket(PrivateStoreManageListBuy{ObjectID: 1, Adena: 2}),
		unhex(t, "b7 01000000 02000000 00000000 00000000"))
}

// Mobius PrivateStoreListBuy.writeImpl:
//
//	c 0xB8; d store owner; d viewer's adena; d count
//	per item: d object ID; d item ID; h enchant; d count; d reference
//	  price; h 0; d body part; h type2; d price; d store count
func TestPrivateStoreListBuyPacket(t *testing.T) {
	shots := tsShots()
	shots.Count = 300 // the viewer has 300 of the 5000 wanted
	none := tsSword()
	none.ObjectID, none.Enchant, none.Count = 0, 0, 0 // the viewer has none
	l := PrivateStoreListBuy{StorePlayerID: 0x10000009, Adena: 777, Items: []StoreBuyItem{
		{StoreItem: shots, StoreCount: 5000},
		{StoreItem: none, StoreCount: 1},
	}}
	var g m3Golden
	g.c(0xb8).d(0x10000009).d(777).d(2)
	g.d(0x10000003).d(1835).h(1).d(300).d(7).h(0).d(0).h(5).d(8).d(5000)
	g.d(0).d(2369).h(0).d(0).d(138).h(0).d(0x80).h(0).d(500000).d(1)
	m3Check(t, "PrivateStoreListBuy", NewPrivateStoreListBuyPacket(l), g.b)
	if n := len(g.b) - 13; n != 2*storeBuyListItemSize {
		t.Errorf("entries of %d bytes", n)
	}

	m3Check(t, "empty PrivateStoreListBuy",
		NewPrivateStoreListBuyPacket(PrivateStoreListBuy{StorePlayerID: 1, Adena: 2}),
		unhex(t, "b8 01000000 02000000 00000000"))
}

// Mobius PrivateStoreMsgSell.writeImpl and PrivateStoreMsgBuy.writeImpl:
// c 0x9C / 0xB9; d object ID; S message (a null title is written as the
// terminator alone).
func TestPrivateStoreMsgPackets(t *testing.T) {
	var g m3Golden
	g.c(0x9c).d(0x10000001).s("Cheap SS")
	m3Check(t, "PrivateStoreMsgSell", NewPrivateStoreMsgSellPacket(0x10000001, "Cheap SS"), g.b)
	g = m3Golden{}
	g.c(0xb9).d(0x10000001).s("WTB Árvíztűrő")
	m3Check(t, "PrivateStoreMsgBuy", NewPrivateStoreMsgBuyPacket(0x10000001, "WTB Árvíztűrő"), g.b)
	m3Check(t, "empty PrivateStoreMsgSell", NewPrivateStoreMsgSellPacket(5, ""), unhex(t, "9c 05000000 0000"))
	m3Check(t, "empty PrivateStoreMsgBuy", NewPrivateStoreMsgBuyPacket(5, ""), unhex(t, "b9 05000000 0000"))
}
