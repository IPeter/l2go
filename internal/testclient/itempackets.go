package testclient

// The server packets of Milestone 5: items on the ground, shops, multisell
// and the warehouse. Every reader follows the writeImpl of the Mobius class
// of the same name, field by field.

// SpawnItem shows an item lying on the ground (0x0b), for example to a
// player who comes near it after it fell.
type SpawnItem struct {
	ObjectID int32
	// ItemID is the display ID of the item template.
	ItemID  int32
	X, Y, Z int32
	// Stackable makes the client show Count.
	Stackable bool
	Count     int32
	// Unknown is a word Mobius writes as 0.
	Unknown int32
}

// PacketName implements Packet.
func (SpawnItem) PacketName() string { return "SpawnItem" }

// Position returns where the item lies.
func (s SpawnItem) Position() Position { return Position{s.X, s.Y, s.Z} }

func readSpawnItem(f *fields) Packet {
	var p SpawnItem
	p.ObjectID = f.i32()
	p.ItemID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Stackable = f.flag32("stackable flag")
	p.Count = f.i32()
	p.Unknown = f.i32()
	return p
}

// DropItem shows an item falling to the ground (0x0c): dropped by a player
// or by a monster as loot. Mobius sends it to the players who see the item
// fall; later viewers get SpawnItem.
type DropItem struct {
	// DropperID is the object that dropped the item.
	DropperID int32
	ObjectID  int32
	// ItemID is the display ID of the item template.
	ItemID  int32
	X, Y, Z int32
	// Stackable makes the client show Count.
	Stackable bool
	Count     int32
	// Unknown is a word Mobius writes as 1.
	Unknown int32
}

// PacketName implements Packet.
func (DropItem) PacketName() string { return "DropItem" }

// Position returns where the item fell.
func (d DropItem) Position() Position { return Position{d.X, d.Y, d.Z} }

func readDropItem(f *fields) Packet {
	var p DropItem
	p.DropperID = f.i32()
	p.ObjectID = f.i32()
	p.ItemID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Stackable = f.flag32("stackable flag")
	p.Count = f.i32()
	p.Unknown = f.i32()
	return p
}

// GetItem shows an object picking up a ground item (0x0d). DeleteObject
// for the item follows it.
type GetItem struct {
	// PlayerID is the object that picks the item up.
	PlayerID int32
	ObjectID int32
	// X, Y and Z are where the item lay.
	X, Y, Z int32
}

// PacketName implements Packet.
func (GetItem) PacketName() string { return "GetItem" }

func readGetItem(f *fields) Packet {
	var p GetItem
	p.PlayerID = f.i32()
	p.ObjectID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	return p
}

// EquipUpdate tells that an item was put on or taken off (0x4b). Mobius
// defines it but never sends it: InventoryUpdate and UserInfo show
// equipment changes.
type EquipUpdate struct {
	Change   int32
	ObjectID int32
	// Slot is the client's number for the item's body part: 1 left ear, 2
	// right ear, 3 neck, 4 right finger, 5 left finger, 6 head, 7 right hand,
	// 8 left hand, 9 gloves, 10 chest, 11 legs, 12 feet, 13 back, 14 both
	// hands, 15 hair, 0 for the others.
	Slot int32
}

// PacketName implements Packet.
func (EquipUpdate) PacketName() string { return "EquipUpdate" }

func readEquipUpdate(f *fields) Packet {
	var p EquipUpdate
	p.Change = f.i32()
	p.ObjectID = f.i32()
	p.Slot = f.i32()
	return p
}

// EnchantResult ends an enchant attempt (0x81). The warehouse requests
// send it with result 0 when they cancel one.
type EnchantResult struct {
	Result int32
}

// PacketName implements Packet.
func (EnchantResult) PacketName() string { return "EnchantResult" }

func readEnchantResult(f *fields) Packet { return EnchantResult{Result: f.i32()} }

// buyListItemSize is the size of one BuyList entry.
const buyListItemSize = 32

// BuyListItem is one product of BuyList.
type BuyListItem struct {
	Type1 uint16
	// ObjectID is 0 in Mobius.
	ObjectID int32
	ItemID   int32
	// Count is the stock left of a product with a limited stock, else 0.
	Count int32
	Type2 uint16
	// Equipped is a short Mobius writes as 0.
	Equipped uint16
	// BodyPart is the slot mask of the item, 0 for the items with type 1 of 4
	// (quest items, adena and the other items not worn).
	BodyPart int32
	// Enchant, Reserved1 and Reserved2 are shorts Mobius writes as 0.
	Enchant   uint16
	Reserved1 uint16
	Reserved2 uint16
	// Price is the price with the merchant's tax.
	Price int32
}

// BuyList is a merchant's buy window (0x11).
//
// Mobius writes the number of products of the whole buylist but leaves out
// the products with a limited stock that are sold out, so fewer entries can
// follow than it announces. The reader takes the entries that are there:
// Listed is the announced number, len(Items) the number written.
type BuyList struct {
	// Adena is the adena the player carries.
	Adena  int32
	ListID int32
	Listed int
	Items  []BuyListItem
}

// PacketName implements Packet.
func (BuyList) PacketName() string { return "BuyList" }

// Find returns the product of item itemID.
func (b BuyList) Find(itemID int32) (BuyListItem, bool) {
	for _, it := range b.Items {
		if it.ItemID == itemID {
			return it, true
		}
	}
	return BuyListItem{}, false
}

func readBuyList(f *fields) Packet {
	var p BuyList
	p.Adena = f.i32()
	p.ListID = f.i32()
	p.Listed = f.count16()
	if !f.ok() {
		return p
	}
	if n := f.r.Len(); n%buyListItemSize != 0 || n/buyListItemSize > p.Listed {
		f.failf("%d bytes of entries for %d products", n, p.Listed)
		return p
	}
	for f.ok() && f.r.Len() > 0 {
		var it BuyListItem
		it.Type1 = f.u16()
		it.ObjectID = f.i32()
		it.ItemID = f.i32()
		it.Count = f.i32()
		it.Type2 = f.u16()
		it.Equipped = f.u16()
		it.BodyPart = f.i32()
		it.Enchant = f.u16()
		it.Reserved1 = f.u16()
		it.Reserved2 = f.u16()
		it.Price = f.i32()
		p.Items = append(p.Items, it)
	}
	return p
}

// SellListItem is one item of SellList.
type SellListItem struct {
	Type1       uint16
	ObjectID    int32
	ItemID      int32
	Count       int32
	Type2       uint16
	CustomType1 uint16
	BodyPart    int32
	Enchant     uint16
	CustomType2 uint16
	// Reserved is a short Mobius writes as 0.
	Reserved uint16
	// Price is what the merchant pays for one: half the reference price.
	Price int32
}

// SellList is a merchant's sell window (0x10): the items of the inventory
// the merchant buys, without the equipped ones.
type SellList struct {
	// Adena is the adena the player carries.
	Adena int32
	// ListID is a word Mobius writes as 0.
	ListID int32
	Items  []SellListItem
}

// PacketName implements Packet.
func (SellList) PacketName() string { return "SellList" }

// Find returns the entry of the inventory item objectID.
func (s SellList) Find(objectID int32) (SellListItem, bool) {
	for _, it := range s.Items {
		if it.ObjectID == objectID {
			return it, true
		}
	}
	return SellListItem{}, false
}

func readSellList(f *fields) Packet {
	var p SellList
	p.Adena = f.i32()
	p.ListID = f.i32()
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		var it SellListItem
		it.Type1 = f.u16()
		it.ObjectID = f.i32()
		it.ItemID = f.i32()
		it.Count = f.i32()
		it.Type2 = f.u16()
		it.CustomType1 = f.u16()
		it.BodyPart = f.i32()
		it.Enchant = f.u16()
		it.CustomType2 = f.u16()
		it.Reserved = f.u16()
		it.Price = f.i32()
		p.Items = append(p.Items, it)
	}
	return p
}

// MultiSellNoType2 is the type 2 Mobius sends for a multisell item without
// an item template.
const MultiSellNoType2 uint16 = 0xffff

// MultiSellProduct is what an entry of MultiSellList gives.
type MultiSellProduct struct {
	// ItemID is the display ID, written as a short. Special IDs are negative.
	ItemID int32
	// BodyPart is the slot mask of the item, 0 for an item without a
	// template.
	BodyPart int32
	// Type2 is MultiSellNoType2 for an item without a template.
	Type2          uint16
	Count          int32
	Enchant        uint16
	AugmentationID int32
	// Mana is a word Mobius writes as 0.
	Mana int32
}

// MultiSellIngredient is what an entry of MultiSellList takes.
type MultiSellIngredient struct {
	// ItemID is the display ID, written as a short. Special IDs are negative.
	ItemID int32
	// Type2 is MultiSellNoType2 for an item without a template.
	Type2          uint16
	Count          int32
	Enchant        uint16
	AugmentationID int32
	// Mana is a word Mobius writes as 0.
	Mana int32
}

// MultiSellEntry is one exchange of MultiSellList.
type MultiSellEntry struct {
	// EntryID is what MultiSellChoose sends back.
	EntryID int32
	// Reserved1 and Reserved2 are words Mobius writes as 0, Reserved3 a byte
	// it writes as 1.
	Reserved1   int32
	Reserved2   int32
	Reserved3   uint8
	Products    []MultiSellProduct
	Ingredients []MultiSellIngredient
}

// MultiSellList is one page of a multisell list (0xd0). Mobius sends one
// packet per 40 entries, the last one with Finished set.
type MultiSellList struct {
	ListID int32
	// Page counts the pages from 1.
	Page     int32
	Finished bool
	// PageSize is 40 in Mobius.
	PageSize int32
	Entries  []MultiSellEntry
}

// PacketName implements Packet.
func (MultiSellList) PacketName() string { return "MultiSellList" }

// Find returns the entry entryID.
func (l MultiSellList) Find(entryID int32) (MultiSellEntry, bool) {
	for _, e := range l.Entries {
		if e.EntryID == entryID {
			return e, true
		}
	}
	return MultiSellEntry{}, false
}

func readMultiSellList(f *fields) Packet {
	var p MultiSellList
	p.ListID = f.i32()
	p.Page = f.i32()
	p.Finished = f.flag32("finished flag")
	p.PageSize = f.i32()
	n := f.count32("entry count")
	for i := 0; i < n && f.ok(); i++ {
		p.Entries = append(p.Entries, readMultiSellEntry(f))
	}
	return p
}

func readMultiSellEntry(f *fields) MultiSellEntry {
	var e MultiSellEntry
	e.EntryID = f.i32()
	e.Reserved1 = f.i32()
	e.Reserved2 = f.i32()
	e.Reserved3 = f.u8()
	products := f.count16()
	ingredients := f.count16()
	for i := 0; i < products && f.ok(); i++ {
		var pr MultiSellProduct
		pr.ItemID = int32(f.i16())
		pr.BodyPart = f.i32()
		pr.Type2 = f.u16()
		pr.Count = f.i32()
		pr.Enchant = f.u16()
		pr.AugmentationID = f.i32()
		pr.Mana = f.i32()
		e.Products = append(e.Products, pr)
	}
	for i := 0; i < ingredients && f.ok(); i++ {
		var in MultiSellIngredient
		in.ItemID = int32(f.i16())
		in.Type2 = f.u16()
		in.Count = f.i32()
		in.Enchant = f.u16()
		in.AugmentationID = f.i32()
		in.Mana = f.i32()
		e.Ingredients = append(e.Ingredients, in)
	}
	return e
}

// Warehouse types of WareHouseDepositList and WareHouseWithdrawalList.
const (
	WarehousePrivate uint16 = 1
	WarehouseClan    uint16 = 2
	WarehouseCastle  uint16 = 3
	WarehouseFreight uint16 = 4
)

// WarehouseListItem is one item of a warehouse window.
type WarehouseListItem struct {
	Type1       uint16
	ObjectID    int32
	ItemID      int32
	Count       int32
	Type2       uint16
	CustomType1 uint16
	BodyPart    int32
	Enchant     uint16
	// Reserved is a short Mobius writes as 0.
	Reserved    uint16
	CustomType2 uint16
	// ObjectID2 repeats ObjectID.
	ObjectID2 int32
	// AugmentationLow and AugmentationHigh are the low 16 bits of the
	// augmentation ID and the rest shifted down, 0 for an item without one.
	AugmentationLow  int32
	AugmentationHigh int32
}

// WarehouseList is the content of a warehouse window.
type WarehouseList struct {
	// Type is WarehousePrivate, WarehouseClan, WarehouseCastle or
	// WarehouseFreight.
	Type uint16
	// Adena is the adena the player carries.
	Adena int32
	Items []WarehouseListItem
}

// Find returns the entry of the item objectID.
func (l WarehouseList) Find(objectID int32) (WarehouseListItem, bool) {
	for _, it := range l.Items {
		if it.ObjectID == objectID {
			return it, true
		}
	}
	return WarehouseListItem{}, false
}

// FindItem returns the first entry of item itemID.
func (l WarehouseList) FindItem(itemID int32) (WarehouseListItem, bool) {
	for _, it := range l.Items {
		if it.ItemID == itemID {
			return it, true
		}
	}
	return WarehouseListItem{}, false
}

// WareHouseDepositList opens the deposit window (0x41): the items the
// player can store.
type WareHouseDepositList struct {
	WarehouseList
}

// PacketName implements Packet.
func (WareHouseDepositList) PacketName() string { return "WareHouseDepositList" }

// WareHouseWithdrawalList opens the withdrawal window (0x42): the stored
// items.
type WareHouseWithdrawalList struct {
	WarehouseList
}

// PacketName implements Packet.
func (WareHouseWithdrawalList) PacketName() string { return "WareHouseWithdrawalList" }

func readWareHouseDepositList(f *fields) Packet {
	return WareHouseDepositList{readWarehouseList(f)}
}

func readWareHouseWithdrawalList(f *fields) Packet {
	return WareHouseWithdrawalList{readWarehouseList(f)}
}

// readWarehouseList reads the fields both warehouse windows share.
func readWarehouseList(f *fields) WarehouseList {
	var l WarehouseList
	l.Type = f.u16()
	l.Adena = f.i32()
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		var it WarehouseListItem
		it.Type1 = f.u16()
		it.ObjectID = f.i32()
		it.ItemID = f.i32()
		it.Count = f.i32()
		it.Type2 = f.u16()
		it.CustomType1 = f.u16()
		it.BodyPart = f.i32()
		it.Enchant = f.u16()
		it.Reserved = f.u16()
		it.CustomType2 = f.u16()
		it.ObjectID2 = f.i32()
		it.AugmentationLow = f.i32()
		it.AugmentationHigh = f.i32()
		l.Items = append(l.Items, it)
	}
	return l
}
