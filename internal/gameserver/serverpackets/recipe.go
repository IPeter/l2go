package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The server packets of the recipe books, crafting and the private workshop
// (Plan 14 M11b).

// Opcodes of the recipe packets (Mobius ServerPackets).
const (
	OpRecipeBookItemList   = 0xd6
	OpRecipeItemMakeInfo   = 0xd7
	OpRecipeShopManageList = 0xd8
	OpRecipeShopSellList   = 0xd9
	OpRecipeShopItemInfo   = 0xda
	OpRecipeShopMsg        = 0xdb
)

// RecipeBookItemList is the dwarven or common recipe book.
type RecipeBookItemList struct {
	Dwarven bool
	MaxMP   int32
	// RecipeIDs are the recipes of the book in the order it lists them.
	RecipeIDs []int32
}

// NewRecipeBookItemListPacket builds RecipeBookItemList (Mobius
// RecipeBookItemList.writeImpl): d 0 for the dwarven book, 1 for the common;
// d max MP; d count; per recipe d recipe ID, d its 1-based place.
func NewRecipeBookItemListPacket(b RecipeBookItemList) []byte {
	w := packets.NewWriter(13 + 8*len(b.RecipeIDs))
	w.WriteUint8(OpRecipeBookItemList)
	putBool32(w, !b.Dwarven)
	w.WriteInt32(b.MaxMP)
	putCount32(w, len(b.RecipeIDs))
	for i, id := range b.RecipeIDs {
		w.WriteInt32(id)
		putCount32(w, i+1)
	}
	return w.Bytes()
}

// RecipeItemMakeInfo is the craft window of a recipe of the player's own
// book.
type RecipeItemMakeInfo struct {
	RecipeID int32
	Dwarven  bool
	// CurMP is the current MP cut to an int, MaxMP the maximum.
	CurMP, MaxMP int32
	// Success is the result of the last craft; Mobius sends true when the
	// window opens.
	Success bool
}

// NewRecipeItemMakeInfoPacket builds RecipeItemMakeInfo (Mobius
// RecipeItemMakeInfo.writeImpl): d recipe; d 0 dwarven or 1 common; d
// current MP; d max MP; d success.
func NewRecipeItemMakeInfoPacket(m RecipeItemMakeInfo) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpRecipeItemMakeInfo)
	w.WriteInt32(m.RecipeID)
	putBool32(w, !m.Dwarven)
	w.WriteInt32(m.CurMP)
	w.WriteInt32(m.MaxMP)
	putBool32(w, m.Success)
	return w.Bytes()
}

// RecipeShopItem is a recipe a workshop offers and its fee (Mobius
// ManufactureItem).
type RecipeShopItem struct {
	RecipeID int32
	Price    int32
}

// putRecipeShopItems writes a workshop list: d count; per entry d recipe, d
// 0, d fee.
func putRecipeShopItems(w *packets.Writer, items []RecipeShopItem) {
	putCount32(w, len(items))
	for _, it := range items {
		w.WriteInt32(it.RecipeID)
		w.WriteInt32(0)
		w.WriteInt32(it.Price)
	}
}

// RecipeShopManageList is the setup window of a private workshop.
type RecipeShopManageList struct {
	SellerID int32
	Adena    int32 // the adena the crafter carries
	Dwarven  bool
	// RecipeIDs is the book the window offers.
	RecipeIDs []int32
	// Items is the workshop's list as it stands.
	Items []RecipeShopItem
}

// NewRecipeShopManageListPacket builds RecipeShopManageList (Mobius
// RecipeShopManageList.writeImpl): d crafter; d adena; d 0 dwarven or 1
// common; d count, per recipe d recipe ID, d 1-based place; then the list.
func NewRecipeShopManageListPacket(l RecipeShopManageList) []byte {
	w := packets.NewWriter(21 + 8*len(l.RecipeIDs) + 12*len(l.Items))
	w.WriteUint8(OpRecipeShopManageList)
	w.WriteInt32(l.SellerID)
	w.WriteInt32(l.Adena)
	putBool32(w, !l.Dwarven)
	putCount32(w, len(l.RecipeIDs))
	for i, id := range l.RecipeIDs {
		w.WriteInt32(id)
		putCount32(w, i+1)
	}
	putRecipeShopItems(w, l.Items)
	return w.Bytes()
}

// RecipeShopSellList is the window of a workshop as a customer sees it.
type RecipeShopSellList struct {
	ManufacturerID int32
	// CurMP and MaxMP are the crafter's.
	CurMP, MaxMP int32
	// Adena is what the customer carries.
	Adena int32
	Items []RecipeShopItem
}

// NewRecipeShopSellListPacket builds RecipeShopSellList (Mobius
// RecipeShopSellList.writeImpl): d crafter; d crafter's MP; d max MP; d
// customer's adena; then the list.
func NewRecipeShopSellListPacket(l RecipeShopSellList) []byte {
	w := packets.NewWriter(21 + 12*len(l.Items))
	w.WriteUint8(OpRecipeShopSellList)
	w.WriteInt32(l.ManufacturerID)
	w.WriteInt32(l.CurMP)
	w.WriteInt32(l.MaxMP)
	w.WriteInt32(l.Adena)
	putRecipeShopItems(w, l.Items)
	return w.Bytes()
}

// RecipeShopItemInfo is the craft window of a workshop recipe.
type RecipeShopItemInfo struct {
	ManufacturerID int32
	RecipeID       int32
	// CurMP and MaxMP are the crafter's.
	CurMP, MaxMP int32
}

// NewRecipeShopItemInfoPacket builds RecipeShopItemInfo (Mobius
// RecipeShopItemInfo.writeImpl): d crafter; d recipe; d crafter's MP; d max
// MP; d 0xffffffff.
func NewRecipeShopItemInfoPacket(m RecipeShopItemInfo) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpRecipeShopItemInfo)
	w.WriteInt32(m.ManufacturerID)
	w.WriteInt32(m.RecipeID)
	w.WriteInt32(m.CurMP)
	w.WriteInt32(m.MaxMP)
	w.WriteInt32(-1)
	return w.Bytes()
}

// NewRecipeShopMsgPacket builds RecipeShopMsg (Mobius
// RecipeShopMsg.writeImpl): d crafter; S the workshop's message.
func NewRecipeShopMsgPacket(objectID int32, message string) []byte {
	return newStoreMsgPacket(OpRecipeShopMsg, objectID, message)
}
