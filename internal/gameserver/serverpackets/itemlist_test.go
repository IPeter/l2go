package serverpackets

import "testing"

func TestItemListPacketEmpty(t *testing.T) {
	// c 0x1B, h show window, h count.
	m3Check(t, "empty ItemList", NewItemListPacket(ItemList{}), unhex(t, "1b 0000 0000"))
	m3Check(t, "empty ItemList with window", NewItemListPacket(ItemList{ShowWindow: true}), unhex(t, "1b 0100 0000"))
}

func TestItemListPacket(t *testing.T) {
	l := ItemList{
		ShowWindow: true,
		Items: []ItemInfo{
			{
				// An equipped, enchanted, augmented sword.
				Type1: 0, ObjectID: 0x10000002, ItemID: 2369, Count: 1, Type2: 0,
				CustomType1: 0, Equipped: true, BodyPart: 0x4000, Enchant: 3,
				CustomType2: 0, AugmentationID: 0x00112233, Mana: -1,
			},
			{
				// Adena in the inventory.
				Type1: 4, ObjectID: 0x10000003, ItemID: 57, Count: 1234567, Type2: 4,
				CustomType1: 5, Equipped: false, BodyPart: 0, Enchant: 0,
				CustomType2: 6, AugmentationID: 0, Mana: 30,
			},
		},
	}
	// Mobius ItemList.writeImpl: c 0x1B, h show window, h item count, then
	// per item (AbstractItemPacket.writeItem): h type1, d object ID,
	// d item ID, d count, h type2, h custom type 1, h equipped,
	// d body part mask, h enchant, h custom type 2, d augmentation ID,
	// d mana.
	var g m3Golden
	g.c(0x1b).h(1).h(2)
	g.h(0).d(0x10000002).d(2369).d(1).h(0).h(0).h(1).d(0x4000).h(3).h(0).d(0x00112233).d(-1)
	g.h(4).d(0x10000003).d(57).d(1234567).h(4).h(5).h(0).d(0).h(0).h(6).d(0).d(30)
	m3Check(t, "ItemList", NewItemListPacket(l), g.b)
}
