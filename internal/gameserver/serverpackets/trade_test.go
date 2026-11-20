package serverpackets

import "testing"

// The opcodes are those of Mobius ServerPackets.
func TestTradeAndStoreServerOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"TradeStart", OpTradeStart, 0x1e},
		{"TradeOwnAdd", OpTradeOwnAdd, 0x20},
		{"TradeOtherAdd", OpTradeOtherAdd, 0x21},
		{"TradeDone", OpTradeDone, 0x22},
		{"SendTradeRequest", OpSendTradeRequest, 0x5e},
		{"TradeUpdate", OpTradeUpdate, 0x74},
		{"TradeOtherDone (TRADE_PRESS_OTHER_OK)", OpTradeOtherDone, 0x7c},
		{"PrivateStoreManageListSell", OpPrivateStoreManageListSell, 0x9a},
		{"PrivateStoreListSell", OpPrivateStoreListSell, 0x9b},
		{"PrivateStoreMsgSell", OpPrivateStoreMsgSell, 0x9c},
		{"PrivateStoreManageListBuy (PRIVATE_STORE_BUY_MANAGE_LIST)", OpPrivateStoreManageListBuy, 0xb7},
		{"PrivateStoreListBuy (PRIVATE_STORE_BUY_LIST)", OpPrivateStoreListBuy, 0xb8},
		{"PrivateStoreMsgBuy (PRIVATE_STORE_BUY_MSG)", OpPrivateStoreMsgBuy, 0xb9},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// Mobius TradeStart.writeImpl:
//
//	c 0x1E; d partner object ID; h item count
//	per item: h type1; d object ID; d item ID (getId); d count; h type2;
//	  h 0; d body part mask; h enchant; h 0; h custom type 2
func TestTradeStartPacket(t *testing.T) {
	s := TradeStart{PartnerID: 0x10000009, Items: []ItemInfo{m5Sword(), m5Shots()}}
	var g m3Golden
	g.c(0x1e).d(0x10000009).h(2)
	g.h(0).d(0x10000002).d(2369).d(1).h(0).h(0).d(0x80).h(3).h(0).h(12)
	g.h(4).d(0x10000003).d(1835).d(300).h(5).h(0).d(0).h(0).h(0).h(22)
	m3Check(t, "TradeStart", NewTradeStartPacket(s), g.b)
	if n := len(g.b) - 7; n != 2*tradeStartItemSize {
		t.Errorf("entries of %d bytes, want 2 x %d", n, tradeStartItemSize)
	}

	m3Check(t, "empty TradeStart", NewTradeStartPacket(TradeStart{PartnerID: 5}),
		unhex(t, "1e 05000000 0000"))
}

// Mobius TradeOwnAdd.writeImpl and TradeOtherAdd.writeImpl:
//
//	c 0x20 / 0x21; h 1
//	h type1; d object ID; d display ID; d count; h type2; h custom type 1;
//	d body part mask; h enchant; h 0; h custom type 2
func TestTradeAddPackets(t *testing.T) {
	sword := m5Sword()
	shots := m5Shots()
	shots.Count = 120 // the count put into the trade
	tests := []struct {
		name string
		got  []byte
		op   byte
		it   ItemInfo
	}{
		{"TradeOwnAdd sword", NewTradeOwnAddPacket(sword), 0x20, sword},
		{"TradeOwnAdd shots", NewTradeOwnAddPacket(shots), 0x20, shots},
		{"TradeOtherAdd sword", NewTradeOtherAddPacket(sword), 0x21, sword},
	}
	for _, tt := range tests {
		var g m3Golden
		it := tt.it
		g.c(tt.op).h(1)
		g.h(it.Type1).d(it.ObjectID).d(it.ItemID).d(it.Count).h(it.Type2).h(it.CustomType1)
		g.d(it.BodyPart).h(it.Enchant).h(0).h(it.CustomType2)
		m3Check(t, tt.name, tt.got, g.b)
		if len(tt.got) != tradeAddSize {
			t.Errorf("%s: %d bytes, want %d", tt.name, len(tt.got), tradeAddSize)
		}
	}
	m3Check(t, "TradeOwnAdd bytes", NewTradeOwnAddPacket(sword),
		unhex(t, "20 0100 0000 02000010 41090000 01000000 0000 0b00 80000000 0300 0000 0c00"))
}

// Mobius TradeDone.writeImpl: c 0x22; d num (0 cancel, 1 finish).
// SendTradeRequest.writeImpl: c 0x5E; d sender. TradeOtherDone.writeImpl:
// c 0x7C only.
func TestSmallTradePackets(t *testing.T) {
	m3Check(t, "TradeDone canceled", NewTradeDonePacket(TradeDoneCanceled), unhex(t, "22 00000000"))
	m3Check(t, "TradeDone finished", NewTradeDonePacket(TradeDoneFinished), unhex(t, "22 01000000"))
	m3Check(t, "SendTradeRequest", NewSendTradeRequestPacket(0x10000009), unhex(t, "5e 09000010"))
	m3Check(t, "TradeOtherDone", NewTradeOtherDonePacket(), unhex(t, "7c"))
}

// Mobius TradeUpdate.writeImpl:
//
//	c 0x74; h trade item count
//	per item: aveCount = inventory count - trade count; stackable from the
//	  template; aveCount 0 makes it 1 and not stackable
//	  h stackable ? 3 : 2; h type1; d object ID; d item ID (getId);
//	  d aveCount; h type2; h 0; d body part mask; h enchant; h 0; h 0
func TestTradeUpdatePacket(t *testing.T) {
	sword := m5Sword()
	shots := m5Shots()
	shots.Count = 120
	all := m5Shots()
	all.Count = 300
	odd := m5Shots()
	odd.ObjectID = 0x10000004
	odd.Count = 1
	// The sword is the only one: removed, count 1. 180 shots are left:
	// modified. Then none are left: removed, count 1. The last one is not
	// in the inventory: removed, count -1.
	items := []TradeUpdateItem{
		{Item: sword, InventoryCount: 1},
		{Item: shots, InventoryCount: 300, Stackable: true},
		{Item: all, InventoryCount: 300, Stackable: true},
		{Item: odd, InventoryCount: 0},
	}
	var g m3Golden
	g.c(0x74).h(4)
	g.h(2).h(0).d(0x10000002).d(2369).d(1).h(0).h(0).d(0x80).h(3).h(0).h(0)
	g.h(3).h(4).d(0x10000003).d(1835).d(180).h(5).h(0).d(0).h(0).h(0).h(0)
	g.h(2).h(4).d(0x10000003).d(1835).d(1).h(5).h(0).d(0).h(0).h(0).h(0)
	g.h(2).h(4).d(0x10000004).d(1835).d(-1).h(5).h(0).d(0).h(0).h(0).h(0)
	m3Check(t, "TradeUpdate", NewTradeUpdatePacket(items), g.b)
	if n := len(g.b) - 3; n != 4*tradeUpdateItemSize {
		t.Errorf("entries of %d bytes, want 4 x %d", n, tradeUpdateItemSize)
	}
	m3Check(t, "empty TradeUpdate", NewTradeUpdatePacket(nil), unhex(t, "74 0000"))
}
