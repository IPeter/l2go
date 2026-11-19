package serverpackets

// The fixtures of the M5 golden tests (items and economy). The expected
// packets are built with m3Golden and checked with m3Check.

// m5Sword is an equipped, enchanted, augmented sword with a value of its own
// in every ItemInfo field, so that a field sent in the wrong place, or one a
// packet must leave out, shows.
func m5Sword() ItemInfo {
	return ItemInfo{
		Type1: 0, ObjectID: 0x10000002, ItemID: 2369, Count: 1, Type2: 0,
		CustomType1: 11, Equipped: true, BodyPart: 0x80, Enchant: 3,
		CustomType2: 12, AugmentationID: 0x12345678, Mana: 45,
	}
}

// m5Shots is a stack of soulshots.
func m5Shots() ItemInfo {
	return ItemInfo{
		Type1: 4, ObjectID: 0x10000003, ItemID: 1835, Count: 300, Type2: 5,
		CustomType1: 21, CustomType2: 22, Mana: -1,
	}
}

// m5GroundAdena is adena a monster dropped.
func m5GroundAdena() GroundItem {
	return GroundItem{
		ObjectID: 0x10000005, ItemID: 57, X: -84023, Y: 244581, Z: -3730,
		Stackable: true, Count: 1234, DropperID: 0x60000001,
	}
}

// m5IDs returns the numbers from to to, inclusive.
func m5IDs(from, to int32) []int32 {
	var ids []int32
	for id := from; id <= to; id++ {
		ids = append(ids, id)
	}
	return ids
}

// m5Entries returns n multisell entries without items, with IDs 1 to n.
func m5Entries(n int) []MultiSellEntry {
	entries := make([]MultiSellEntry, n)
	for i := range entries {
		entries[i].EntryID = int32(i + 1)
	}
	return entries
}

// m5Page is a MultiSellList page of entries without items, as Mobius writes
// it: the header, then per entry its ID, d 0, d 0, c 1, h 0 products, h 0
// ingredients.
func m5Page(listID, page int32, finished bool, ids ...int32) []byte {
	var fin int32
	if finished {
		fin = 1
	}
	var g m3Golden
	g.c(0xd0).d(listID).d(page).d(fin).d(40).d(int32(len(ids)))
	for _, id := range ids {
		g.d(id).d(0).d(0).c(1).h(0).h(0)
	}
	return g.b
}
