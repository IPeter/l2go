package serverpackets

import "testing"

func TestInventoryUpdatePacket(t *testing.T) {
	u := InventoryUpdate{Items: []InventoryItem{
		{
			// Adena after a teleport fee.
			Change: ItemModified,
			Item: ItemInfo{
				Type1: 4, ObjectID: 0x10000003, ItemID: 57, Count: 2000, Type2: 4,
				Mana: -1,
			},
		},
		{
			// A used up stack.
			Change: ItemRemoved,
			Item: ItemInfo{
				Type1: 4, ObjectID: 0x10000004, ItemID: 6651, Count: 0, Type2: 5,
				Mana: -1,
			},
		},
	}}
	// Mobius InventoryUpdate.writeImpl: c 0x27, then writeItems: h item count,
	// per item h change (1 add, 2 modify, 3 remove) and the item block of
	// AbstractItemPacket.writeItem: h type1, d object ID, d item ID, d count,
	// h type2, h custom type 1, h equipped, d body part mask, h enchant,
	// h custom type 2, d augmentation ID, d mana.
	var g m3Golden
	g.c(0x27).h(2)
	g.h(2).h(4).d(0x10000003).d(57).d(2000).h(4).h(0).h(0).d(0).h(0).h(0).d(0).d(-1)
	g.h(3).h(4).d(0x10000004).d(6651).d(0).h(5).h(0).h(0).d(0).h(0).h(0).d(0).d(-1)
	m3Check(t, "InventoryUpdate", NewInventoryUpdatePacket(u), g.b)

	// An empty update is just the count.
	m3Check(t, "empty InventoryUpdate", NewInventoryUpdatePacket(InventoryUpdate{}), unhex(t, "27 0000"))
}
