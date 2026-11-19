package serverpackets

import "testing"

// Mobius WareHouseDepositList.writeImpl and WareHouseWithdrawalList.writeImpl
// (the same bytes after the opcode):
//
//	c 0x41 (deposit) or 0x42 (withdrawal)
//	h warehouse type; d adena; h item count
//	per item: h type1; d object ID; d item ID; d count; h type2;
//	  h custom type 1; d body part mask; h enchant; h 0; h custom type 2;
//	  d object ID; augmented: d id & 0xFFFF, d id >> 16; else q 0
//
// Equipped and the mana are not sent.
func TestWarehouseListPackets(t *testing.T) {
	// An augmentation ID is never negative in Mobius; this one pins the
	// shift with the sign.
	odd := m5Shots()
	odd.ObjectID, odd.AugmentationID = 0x10000004, -2
	l := WarehouseList{Type: WarehousePrivate, Adena: 1234567, Items: []ItemInfo{m5Sword(), m5Shots(), odd}}
	body := func(g *m3Golden) {
		g.h(1).d(1234567).h(3)
		g.h(0).d(0x10000002).d(2369).d(1).h(0).h(11).d(0x80).h(3).h(0).h(12)
		g.d(0x10000002).d(0x5678).d(0x1234)
		g.h(4).d(0x10000003).d(1835).d(300).h(5).h(21).d(0).h(0).h(0).h(22)
		g.d(0x10000003).q(0)
		g.h(4).d(0x10000004).d(1835).d(300).h(5).h(21).d(0).h(0).h(0).h(22)
		g.d(0x10000004).d(0xfffe).d(-1)
	}

	var deposit m3Golden
	deposit.c(0x41)
	body(&deposit)
	m3Check(t, "WareHouseDepositList", NewWareHouseDepositListPacket(l), deposit.b)

	var withdrawal m3Golden
	withdrawal.c(0x42)
	body(&withdrawal)
	m3Check(t, "WareHouseWithdrawalList", NewWareHouseWithdrawalListPacket(l), withdrawal.b)

	m3Check(t, "empty clan deposit",
		NewWareHouseDepositListPacket(WarehouseList{Type: WarehouseClan, Adena: 9}),
		unhex(t, "41 0200 09000000 0000"))
	m3Check(t, "empty freight withdrawal",
		NewWareHouseWithdrawalListPacket(WarehouseList{Type: WarehouseFreight}),
		unhex(t, "42 0400 00000000 0000"))
}
