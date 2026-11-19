package serverpackets

import "testing"

// Mobius GetItem.writeImpl: c 0x0D; d player object ID, item object ID;
// d x, y, z of the item.
func TestGetItemPacket(t *testing.T) {
	got := NewGetItemPacket(GetItem{PlayerID: 0x10000001, ObjectID: 0x10000005, X: -84023, Y: 244581, Z: -3730})
	var g m3Golden
	g.c(0x0d).d(0x10000001).d(0x10000005).d(-84023).d(244581).d(-3730)
	m3Check(t, "GetItem", got, g.b)

	m3Check(t, "GetItem at the origin",
		NewGetItemPacket(GetItem{PlayerID: 1, ObjectID: 2}),
		unhex(t, "0d 01000000 02000000 00000000 00000000 00000000"))
}
