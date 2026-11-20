package testclient

// The server packets of trades and private stores (Milestone 9). Every
// reader follows the writeImpl of the Mobius class of the same name, field
// by field.

// Game server opcodes of trades and private stores, from Mobius
// ServerPackets. The server TradeDone is OpTradeDoneServer; OpTradeDone is
// the client one.
const (
	OpTradeStart                 byte = 0x1e
	OpTradeOwnAdd                byte = 0x20
	OpTradeOtherAdd              byte = 0x21
	OpTradeDoneServer            byte = 0x22
	OpSendTradeRequest           byte = 0x5e
	OpTradeUpdate                byte = 0x74
	OpTradeOtherDone             byte = 0x7c // TRADE_PRESS_OTHER_OK
	OpPrivateStoreManageListSell byte = 0x9a
	OpPrivateStoreListSell       byte = 0x9b
	OpPrivateStoreMsgSell        byte = 0x9c
	OpPrivateStoreManageListBuy  byte = 0xb7 // PRIVATE_STORE_BUY_MANAGE_LIST
	OpPrivateStoreListBuy        byte = 0xb8 // PRIVATE_STORE_BUY_LIST
	OpPrivateStoreMsgBuy         byte = 0xb9 // PRIVATE_STORE_BUY_MSG
)

// Results of the server TradeDone.
const (
	TradeDoneCanceled int32 = 0
	TradeDoneFinished int32 = 1
)

// Change types of TradeUpdate.
const (
	TradeUpdateRemove uint16 = 2
	TradeUpdateModify uint16 = 3
)

// TradeItem is an item of TradeStart, TradeOwnAdd and TradeOtherAdd.
type TradeItem struct {
	Type1    uint16
	ObjectID int32
	// ItemID is the template ID in TradeStart and the display ID in
	// TradeOwnAdd and TradeOtherAdd.
	ItemID int32
	Count  int32
	Type2  uint16
	// CustomType1 is a short TradeStart writes as 0.
	CustomType1 uint16
	BodyPart    int32
	Enchant     uint16
	// Reserved is a short Mobius writes as 0.
	Reserved    uint16
	CustomType2 uint16
}

func readTradeItem(f *fields) TradeItem {
	var it TradeItem
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
	return it
}

// TradeStart opens the trade window (0x1e) with the items the player can
// trade.
type TradeStart struct {
	PartnerID int32
	Items     []TradeItem
}

// PacketName implements Packet.
func (TradeStart) PacketName() string { return "TradeStart" }

// Find returns the entry of the inventory item objectID.
func (s TradeStart) Find(objectID int32) (TradeItem, bool) {
	for _, it := range s.Items {
		if it.ObjectID == objectID {
			return it, true
		}
	}
	return TradeItem{}, false
}

func readTradeStart(f *fields) Packet {
	var p TradeStart
	p.PartnerID = f.i32()
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		p.Items = append(p.Items, readTradeItem(f))
	}
	return p
}

// readTradeAdd reads TradeOwnAdd and TradeOtherAdd: h 1, then the item.
func readTradeAdd(f *fields) TradeItem {
	if n := f.count16(); f.ok() && n != 1 {
		f.failf("%d items, Mobius writes 1", n)
		return TradeItem{}
	}
	return readTradeItem(f)
}

// TradeOwnAdd shows an item the player put into the trade (0x20). Count is
// the count put in.
type TradeOwnAdd struct {
	Item TradeItem
}

// PacketName implements Packet.
func (TradeOwnAdd) PacketName() string { return "TradeOwnAdd" }

func readTradeOwnAdd(f *fields) Packet { return TradeOwnAdd{Item: readTradeAdd(f)} }

// TradeOtherAdd shows an item the partner put into the trade (0x21).
type TradeOtherAdd struct {
	Item TradeItem
}

// PacketName implements Packet.
func (TradeOtherAdd) PacketName() string { return "TradeOtherAdd" }

func readTradeOtherAdd(f *fields) Packet { return TradeOtherAdd{Item: readTradeAdd(f)} }

// TradeDone closes the trade window (0x22): TradeDoneCanceled or
// TradeDoneFinished.
type TradeDone struct {
	Result int32
}

// PacketName implements Packet.
func (TradeDone) PacketName() string { return "TradeDone" }

func readTradeDone(f *fields) Packet { return TradeDone{Result: f.i32()} }

// SendTradeRequest asks the player to trade with SenderID (0x5e). The
// player answers with AnswerTradeRequest.
type SendTradeRequest struct {
	SenderID int32
}

// PacketName implements Packet.
func (SendTradeRequest) PacketName() string { return "SendTradeRequest" }

func readSendTradeRequest(f *fields) Packet { return SendTradeRequest{SenderID: f.i32()} }

// TradeUpdateItem is one entry of TradeUpdate: an item of the player's
// trade list, as it is left in the inventory side of the window.
type TradeUpdateItem struct {
	// Change is TradeUpdateModify for a stackable item with some left, else
	// TradeUpdateRemove.
	Change   uint16
	Type1    uint16
	ObjectID int32
	ItemID   int32 // the template ID
	// Count is what is left: the inventory count less the trade count, 1
	// when nothing is left.
	Count int32
	Type2 uint16
	// Reserved1, Reserved2 and Reserved3 are shorts Mobius writes as 0.
	Reserved1 uint16
	BodyPart  int32
	Enchant   uint16
	Reserved2 uint16
	Reserved3 uint16
}

// TradeUpdate updates the inventory side of the trade window (0x74) after
// the player added an item.
type TradeUpdate struct {
	Items []TradeUpdateItem
}

// PacketName implements Packet.
func (TradeUpdate) PacketName() string { return "TradeUpdate" }

func readTradeUpdate(f *fields) Packet {
	var p TradeUpdate
	n := f.count16()
	for i := 0; i < n && f.ok(); i++ {
		var it TradeUpdateItem
		it.Change = f.u16()
		it.Type1 = f.u16()
		it.ObjectID = f.i32()
		it.ItemID = f.i32()
		it.Count = f.i32()
		it.Type2 = f.u16()
		it.Reserved1 = f.u16()
		it.BodyPart = f.i32()
		it.Enchant = f.u16()
		it.Reserved2 = f.u16()
		it.Reserved3 = f.u16()
		p.Items = append(p.Items, it)
	}
	return p
}

// TradeOtherDone tells that the partner confirmed the trade (0x7c).
type TradeOtherDone struct{}

// PacketName implements Packet.
func (TradeOtherDone) PacketName() string { return "TradeOtherDone" }

func readTradeOtherDone(*fields) Packet { return TradeOtherDone{} }

// StoreSellItem is an item of a sell store window or of its setup window.
type StoreSellItem struct {
	// Type2 is written as an int.
	Type2    int32
	ObjectID int32
	ItemID   int32 // the template ID
	Count    int32
	// Reserved is a short Mobius writes as 0.
	Reserved uint16
	Enchant  uint16
	// CustomType2 is written as 0 for the items already in the sell list
	// of PrivateStoreManageListSell.
	CustomType2 uint16
	BodyPart    int32
	// Price is the unit price; for the available items of
	// PrivateStoreManageListSell it is the reference price.
	Price int32
	// ReferencePrice is the template's reference price, 0 for the available
	// items of PrivateStoreManageListSell, which do not carry it.
	ReferencePrice int32
}

// readStoreSellItem reads a sell store entry; withReference reads the
// reference price at its end.
func readStoreSellItem(f *fields, withReference bool) StoreSellItem {
	var it StoreSellItem
	it.Type2 = f.i32()
	it.ObjectID = f.i32()
	it.ItemID = f.i32()
	it.Count = f.i32()
	it.Reserved = f.u16()
	it.Enchant = f.u16()
	it.CustomType2 = f.u16()
	it.BodyPart = f.i32()
	it.Price = f.i32()
	if withReference {
		it.ReferencePrice = f.i32()
	}
	return it
}

func readStoreSellItems(f *fields, field string, withReference bool) []StoreSellItem {
	var items []StoreSellItem
	n := f.count32(field)
	for i := 0; i < n && f.ok(); i++ {
		items = append(items, readStoreSellItem(f, withReference))
	}
	return items
}

func findStoreSellItem(items []StoreSellItem, objectID int32) (StoreSellItem, bool) {
	for _, it := range items {
		if it.ObjectID == objectID {
			return it, true
		}
	}
	return StoreSellItem{}, false
}

// PrivateStoreManageListSell is the setup window of a sell store (0x9a).
type PrivateStoreManageListSell struct {
	ObjectID    int32
	PackageSale bool
	Adena       int32
	// Available are the inventory items that can still be offered.
	Available []StoreSellItem
	// Items are the items already in the sell list.
	Items []StoreSellItem
}

// PacketName implements Packet.
func (PrivateStoreManageListSell) PacketName() string { return "PrivateStoreManageListSell" }

// FindAvailable returns the available entry of the inventory item
// objectID.
func (l PrivateStoreManageListSell) FindAvailable(objectID int32) (StoreSellItem, bool) {
	return findStoreSellItem(l.Available, objectID)
}

func readPrivateStoreManageListSell(f *fields) Packet {
	var p PrivateStoreManageListSell
	p.ObjectID = f.i32()
	p.PackageSale = f.flag32("package sale flag")
	p.Adena = f.i32()
	p.Available = readStoreSellItems(f, "available count", false)
	if f.ok() {
		p.Items = readStoreSellItems(f, "sell list count", true)
	}
	return p
}

// PrivateStoreListSell is the window of a sell store (0x9b), as a buyer
// sees it.
type PrivateStoreListSell struct {
	SellerID    int32
	PackageSale bool
	// Adena is the adena the buyer carries.
	Adena int32
	Items []StoreSellItem
}

// PacketName implements Packet.
func (PrivateStoreListSell) PacketName() string { return "PrivateStoreListSell" }

// Find returns the entry of the seller's item objectID.
func (l PrivateStoreListSell) Find(objectID int32) (StoreSellItem, bool) {
	return findStoreSellItem(l.Items, objectID)
}

func readPrivateStoreListSell(f *fields) Packet {
	var p PrivateStoreListSell
	p.SellerID = f.i32()
	p.PackageSale = f.flag32("package sale flag")
	p.Adena = f.i32()
	p.Items = readStoreSellItems(f, "item count", true)
	return p
}

// StoreBuyManageItem is an item of the setup window of a buy store.
type StoreBuyManageItem struct {
	ItemID int32 // the template ID
	// Enchant is a short Mobius writes as 0.
	Enchant        uint16
	Count          int32
	ReferencePrice int32
	// Reserved is a short Mobius writes as 0.
	Reserved uint16
	BodyPart int32
	Type2    uint16
	// Price is the unit price the owner set, and StorePrice the reference
	// price again; both only in the buy list section.
	Price      int32
	StorePrice int32
}

func readStoreBuyManageItems(f *fields, field string, inList bool) []StoreBuyManageItem {
	var items []StoreBuyManageItem
	n := f.count32(field)
	for i := 0; i < n && f.ok(); i++ {
		var it StoreBuyManageItem
		it.ItemID = f.i32()
		it.Enchant = f.u16()
		it.Count = f.i32()
		it.ReferencePrice = f.i32()
		it.Reserved = f.u16()
		it.BodyPart = f.i32()
		it.Type2 = f.u16()
		if inList {
			it.Price = f.i32()
			it.StorePrice = f.i32()
		}
		items = append(items, it)
	}
	return items
}

// PrivateStoreManageListBuy is the setup window of a buy store (0xb7).
type PrivateStoreManageListBuy struct {
	ObjectID int32
	Adena    int32
	// Inventory are the owner's items a buy list can name, one per
	// template.
	Inventory []StoreBuyManageItem
	// Items are the items already in the buy list.
	Items []StoreBuyManageItem
}

// PacketName implements Packet.
func (PrivateStoreManageListBuy) PacketName() string { return "PrivateStoreManageListBuy" }

func readPrivateStoreManageListBuy(f *fields) Packet {
	var p PrivateStoreManageListBuy
	p.ObjectID = f.i32()
	p.Adena = f.i32()
	p.Inventory = readStoreBuyManageItems(f, "inventory count", false)
	if f.ok() {
		p.Items = readStoreBuyManageItems(f, "buy list count", true)
	}
	return p
}

// StoreBuyItem is one entry of PrivateStoreListBuy.
type StoreBuyItem struct {
	// ObjectID and Enchant are those of the viewer's item of the template,
	// 0 when he has none.
	ObjectID int32
	ItemID   int32 // the template ID
	Enchant  uint16
	// Count is what the viewer can sell.
	Count          int32
	ReferencePrice int32
	// Reserved is a short Mobius writes as 0.
	Reserved uint16
	BodyPart int32
	Type2    uint16
	Price    int32
	// StoreCount is the count the store still wants.
	StoreCount int32
}

// PrivateStoreListBuy is the window of a buy store (0xb8), as a seller sees
// it.
type PrivateStoreListBuy struct {
	StorePlayerID int32
	// Adena is the adena the viewer carries.
	Adena int32
	Items []StoreBuyItem
}

// PacketName implements Packet.
func (PrivateStoreListBuy) PacketName() string { return "PrivateStoreListBuy" }

// FindItem returns the entry of item itemID.
func (l PrivateStoreListBuy) FindItem(itemID int32) (StoreBuyItem, bool) {
	for _, it := range l.Items {
		if it.ItemID == itemID {
			return it, true
		}
	}
	return StoreBuyItem{}, false
}

func readPrivateStoreListBuy(f *fields) Packet {
	var p PrivateStoreListBuy
	p.StorePlayerID = f.i32()
	p.Adena = f.i32()
	n := f.count32("item count")
	for i := 0; i < n && f.ok(); i++ {
		var it StoreBuyItem
		it.ObjectID = f.i32()
		it.ItemID = f.i32()
		it.Enchant = f.u16()
		it.Count = f.i32()
		it.ReferencePrice = f.i32()
		it.Reserved = f.u16()
		it.BodyPart = f.i32()
		it.Type2 = f.u16()
		it.Price = f.i32()
		it.StoreCount = f.i32()
		p.Items = append(p.Items, it)
	}
	return p
}

// PrivateStoreMsgSell is the message over a sell store (0x9c).
type PrivateStoreMsgSell struct {
	ObjectID int32
	Message  string
}

// PacketName implements Packet.
func (PrivateStoreMsgSell) PacketName() string { return "PrivateStoreMsgSell" }

func readPrivateStoreMsgSell(f *fields) Packet {
	return PrivateStoreMsgSell{ObjectID: f.i32(), Message: f.str()}
}

// PrivateStoreMsgBuy is the message over a buy store (0xb9).
type PrivateStoreMsgBuy struct {
	ObjectID int32
	Message  string
}

// PacketName implements Packet.
func (PrivateStoreMsgBuy) PacketName() string { return "PrivateStoreMsgBuy" }

func readPrivateStoreMsgBuy(f *fields) Packet {
	return PrivateStoreMsgBuy{ObjectID: f.i32(), Message: f.str()}
}
