package testclient

// The server packets of the recipe books, crafting and the private workshop
// (Plan 14 M11b), parsed as Mobius writes them.

// Opcodes of the recipe packets (Mobius ServerPackets).
const (
	OpRecipeBookItemList   byte = 0xd6
	OpRecipeItemMakeInfo   byte = 0xd7
	OpRecipeShopManageList byte = 0xd8
	OpRecipeShopSellList   byte = 0xd9
	OpRecipeShopItemInfo   byte = 0xda
	OpRecipeShopMsg        byte = 0xdb
)

func init() {
	serverPackets[OpRecipeBookItemList] = serverPacket{"RecipeBookItemList", readRecipeBookItemList}
	serverPackets[OpRecipeItemMakeInfo] = serverPacket{"RecipeItemMakeInfo", readRecipeItemMakeInfo}
	serverPackets[OpRecipeShopManageList] = serverPacket{"RecipeShopManageList", readRecipeShopManageList}
	serverPackets[OpRecipeShopSellList] = serverPacket{"RecipeShopSellList", readRecipeShopSellList}
	serverPackets[OpRecipeShopItemInfo] = serverPacket{"RecipeShopItemInfo", readRecipeShopItemInfo}
	serverPackets[OpRecipeShopMsg] = serverPacket{"RecipeShopMsg", readRecipeShopMsg}
}

// RecipeBookItemList is a recipe book (0xd6).
type RecipeBookItemList struct {
	Dwarven bool
	MaxMP   int32
	// RecipeIDs are the recipes in the order sent; Mobius numbers them from
	// 1, which the parser checks.
	RecipeIDs []int32
}

// PacketName implements Packet.
func (RecipeBookItemList) PacketName() string { return "RecipeBookItemList" }

func readRecipeBookItemList(f *fields) Packet {
	p := RecipeBookItemList{Dwarven: !f.flag32("book type"), MaxMP: f.i32()}
	p.RecipeIDs = f.recipeBook()
	return p
}

// recipeBook reads d count, then per recipe d ID and d its 1-based place.
func (f *fields) recipeBook() []int32 {
	n := f.count32("recipe count")
	var ids []int32
	for i := 0; i < n && f.ok(); i++ {
		id, place := f.i32(), f.i32()
		if place != int32(i+1) { //nolint:gosec // small counts
			f.failf("recipe %d is numbered %d, want %d", id, place, i+1)
		}
		ids = append(ids, id)
	}
	return ids
}

// RecipeShopItem is a recipe a workshop offers.
type RecipeShopItem struct {
	RecipeID int32
	Price    int32
}

// recipeShopItems reads d count, then per entry d recipe, d 0, d cost.
func (f *fields) recipeShopItems() []RecipeShopItem {
	n := f.count32("workshop count")
	var items []RecipeShopItem
	for i := 0; i < n && f.ok(); i++ {
		it := RecipeShopItem{RecipeID: f.i32()}
		if zero := f.i32(); zero != 0 {
			f.failf("workshop entry %d has %d where Mobius writes 0", it.RecipeID, zero)
		}
		it.Price = f.i32()
		items = append(items, it)
	}
	return items
}

// RecipeItemMakeInfo is the craft window of a recipe (0xd7).
type RecipeItemMakeInfo struct {
	RecipeID     int32
	Dwarven      bool
	CurMP, MaxMP int32
	Success      bool
}

// PacketName implements Packet.
func (RecipeItemMakeInfo) PacketName() string { return "RecipeItemMakeInfo" }

func readRecipeItemMakeInfo(f *fields) Packet {
	return RecipeItemMakeInfo{
		RecipeID: f.i32(), Dwarven: !f.flag32("book type"), CurMP: f.i32(), MaxMP: f.i32(),
		Success: f.flag32("success"),
	}
}

// RecipeShopManageList is the setup window of a workshop (0xd8).
type RecipeShopManageList struct {
	SellerID  int32
	Adena     int32
	Dwarven   bool
	RecipeIDs []int32
	Items     []RecipeShopItem
}

// PacketName implements Packet.
func (RecipeShopManageList) PacketName() string { return "RecipeShopManageList" }

func readRecipeShopManageList(f *fields) Packet {
	p := RecipeShopManageList{SellerID: f.i32(), Adena: f.i32(), Dwarven: !f.flag32("book type")}
	p.RecipeIDs = f.recipeBook()
	p.Items = f.recipeShopItems()
	return p
}

// RecipeShopSellList is a workshop as a customer sees it (0xd9).
type RecipeShopSellList struct {
	ManufacturerID int32
	CurMP, MaxMP   int32
	Adena          int32
	Items          []RecipeShopItem
}

// PacketName implements Packet.
func (RecipeShopSellList) PacketName() string { return "RecipeShopSellList" }

func readRecipeShopSellList(f *fields) Packet {
	p := RecipeShopSellList{ManufacturerID: f.i32(), CurMP: f.i32(), MaxMP: f.i32(), Adena: f.i32()}
	p.Items = f.recipeShopItems()
	return p
}

// RecipeShopItemInfo is the craft window of a workshop recipe (0xda).
type RecipeShopItemInfo struct {
	ManufacturerID int32
	RecipeID       int32
	CurMP, MaxMP   int32
}

// PacketName implements Packet.
func (RecipeShopItemInfo) PacketName() string { return "RecipeShopItemInfo" }

func readRecipeShopItemInfo(f *fields) Packet {
	p := RecipeShopItemInfo{ManufacturerID: f.i32(), RecipeID: f.i32(), CurMP: f.i32(), MaxMP: f.i32()}
	if last := f.i32(); last != -1 && f.ok() {
		f.failf("last field is %d, Mobius writes 0xffffffff", last)
	}
	return p
}

// RecipeShopMsg is the message over a workshop (0xdb).
type RecipeShopMsg struct {
	ObjectID int32
	Message  string
}

// PacketName implements Packet.
func (RecipeShopMsg) PacketName() string { return "RecipeShopMsg" }

func readRecipeShopMsg(f *fields) Packet {
	return RecipeShopMsg{ObjectID: f.i32(), Message: f.str()}
}

// Find returns the entry of a recipe in the workshop list.
func (l RecipeShopSellList) Find(recipeID int32) (RecipeShopItem, bool) {
	for _, it := range l.Items {
		if it.RecipeID == recipeID {
			return it, true
		}
	}
	return RecipeShopItem{}, false
}
