package serverpackets

import "testing"

// Mobius SellList.writeImpl:
//
//	c 0x10
//	d adena; d 0; h item count
//	per item: h type1; d object ID; d item ID; d count; h type2;
//	  h custom type 1; d body part mask; h enchant; h custom type 2; h 0;
//	  d reference price / 2
//
// Equipped, the augmentation and the mana are not sent.
func TestSellListPacket(t *testing.T) {
	l := SellList{Adena: 1234567, Items: []SellListItem{
		{Item: m5Sword(), ReferencePrice: 138},
		{Item: m5Shots(), ReferencePrice: 7}, // 3.5, cut to 3
	}}
	var g m3Golden
	g.c(0x10).d(1234567).d(0).h(2)
	g.h(0).d(0x10000002).d(2369).d(1).h(0).h(11).d(0x80).h(3).h(12).h(0).d(69)
	g.h(4).d(0x10000003).d(1835).d(300).h(5).h(21).d(0).h(0).h(22).h(0).d(3)
	m3Check(t, "SellList", NewSellListPacket(l), g.b)

	m3Check(t, "empty SellList",
		NewSellListPacket(SellList{Adena: 5}),
		unhex(t, "10 05000000 00000000 0000"))
}
