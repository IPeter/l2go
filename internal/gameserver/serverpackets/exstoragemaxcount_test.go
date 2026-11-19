package serverpackets

import "testing"

func TestExStorageMaxCountPacket(t *testing.T) {
	s := ExStorageMaxCount{
		Inventory: 80, Warehouse: 100, ClanWarehouse: 150, PrivateSell: 3,
		PrivateBuy: 4, DwarfRecipe: 0, CommonRecipe: 50, InventoryExtra: 0,
	}
	// Mobius ExStorageMaxCount.writeImpl: c 0xFE, h 0x2E, d inventory,
	// d warehouse, d clan warehouse, d private sell, d private buy,
	// d dwarven recipes, d common recipes, d extra inventory slots.
	var g m3Golden
	g.c(0xfe).h(0x2e)
	g.d(80).d(100).d(150).d(3).d(4).d(0).d(50).d(0)
	m3Check(t, "ExStorageMaxCount", NewExStorageMaxCountPacket(s), g.b)
}
