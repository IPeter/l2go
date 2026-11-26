package serverpackets

import "testing"

// Mobius RecipeBookItemList.writeImpl:
//
//	c 0xD6; d !isDwarvenCraft; d max MP; d count; per recipe d ID, d index
func TestRecipeBookItemListPacket(t *testing.T) {
	var g m3Golden
	g.c(0xd6).d(0).d(345).d(2)
	g.d(5).d(1)
	g.d(1).d(2)
	m3Check(t, "dwarven RecipeBookItemList",
		NewRecipeBookItemListPacket(RecipeBookItemList{Dwarven: true, MaxMP: 345, RecipeIDs: []int32{5, 1}}), g.b)

	m3Check(t, "empty common RecipeBookItemList",
		NewRecipeBookItemListPacket(RecipeBookItemList{MaxMP: 2}),
		unhex(t, "d6 01000000 02000000 00000000"))
}

// Mobius RecipeItemMakeInfo.writeImpl:
//
//	c 0xD7; d recipe; d !isDwarvenRecipe; d (int) current MP; d max MP;
//	d success
func TestRecipeItemMakeInfoPacket(t *testing.T) {
	var g m3Golden
	g.c(0xd7).d(1).d(0).d(120).d(345).d(1)
	m3Check(t, "RecipeItemMakeInfo", NewRecipeItemMakeInfoPacket(RecipeItemMakeInfo{
		RecipeID: 1, Dwarven: true, CurMP: 120, MaxMP: 345, Success: true}), g.b)

	m3Check(t, "failed common RecipeItemMakeInfo",
		NewRecipeItemMakeInfoPacket(RecipeItemMakeInfo{RecipeID: 2, CurMP: 3, MaxMP: 4}),
		unhex(t, "d7 02000000 01000000 03000000 04000000 00000000"))
}

// Mobius RecipeShopManageList.writeImpl:
//
//	c 0xD8; d crafter; d adena; d !isDwarven; d book count, per recipe d ID,
//	d index; d list count, per entry d recipe, d 0, d cost
func TestRecipeShopManageListPacket(t *testing.T) {
	var g m3Golden
	g.c(0xd8).d(0x10000001).d(123456).d(0)
	g.d(3).d(1).d(1).d(7).d(2).d(9).d(3)
	g.d(2).d(7).d(0).d(1000).d(1).d(0).d(0)
	m3Check(t, "RecipeShopManageList", NewRecipeShopManageListPacket(RecipeShopManageList{
		SellerID: 0x10000001, Adena: 123456, Dwarven: true,
		RecipeIDs: []int32{1, 7, 9},
		Items:     []RecipeShopItem{{RecipeID: 7, Price: 1000}, {RecipeID: 1}},
	}), g.b)

	m3Check(t, "empty RecipeShopManageList",
		NewRecipeShopManageListPacket(RecipeShopManageList{SellerID: 1, Adena: 2}),
		unhex(t, "d8 01000000 02000000 01000000 00000000 00000000"))
}

// Mobius RecipeShopSellList.writeImpl:
//
//	c 0xD9; d crafter; d (int) crafter's MP; d max MP; d buyer's adena;
//	d count, per entry d recipe, d 0, d cost
func TestRecipeShopSellListPacket(t *testing.T) {
	var g m3Golden
	g.c(0xd9).d(0x10000002).d(99).d(300).d(5000)
	g.d(1).d(7).d(0).d(1000)
	m3Check(t, "RecipeShopSellList", NewRecipeShopSellListPacket(RecipeShopSellList{
		ManufacturerID: 0x10000002, CurMP: 99, MaxMP: 300, Adena: 5000,
		Items: []RecipeShopItem{{RecipeID: 7, Price: 1000}},
	}), g.b)
}

// Mobius RecipeShopItemInfo.writeImpl:
//
//	c 0xDA; d crafter; d recipe; d (int) crafter's MP; d max MP;
//	d 0xffffffff
func TestRecipeShopItemInfoPacket(t *testing.T) {
	m3Check(t, "RecipeShopItemInfo",
		NewRecipeShopItemInfoPacket(RecipeShopItemInfo{ManufacturerID: 0x10000002, RecipeID: 7, CurMP: 99, MaxMP: 300}),
		unhex(t, "da 02000010 07000000 63000000 2c010000 ffffffff"))
}

// Mobius RecipeShopMsg.writeImpl: c 0xDB; d crafter; S message.
func TestRecipeShopMsgPacket(t *testing.T) {
	var g m3Golden
	g.c(0xdb).d(0x10000002).s("Arrows 5k")
	m3Check(t, "RecipeShopMsg", NewRecipeShopMsgPacket(0x10000002, "Arrows 5k"), g.b)
	m3Check(t, "empty RecipeShopMsg", NewRecipeShopMsgPacket(3, ""), unhex(t, "db 03000000 0000"))
}
