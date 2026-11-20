package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The server packets of private stores.

// Opcodes of the private store packets (Mobius ServerPackets).
const (
	OpPrivateStoreManageListSell = 0x9a
	OpPrivateStoreListSell       = 0x9b
	OpPrivateStoreMsgSell        = 0x9c
	OpPrivateStoreManageListBuy  = 0xb7 // Mobius PRIVATE_STORE_BUY_MANAGE_LIST
	OpPrivateStoreListBuy        = 0xb8 // Mobius PRIVATE_STORE_BUY_LIST
	OpPrivateStoreMsgBuy         = 0xb9 // Mobius PRIVATE_STORE_BUY_MSG
)

// Entry sizes of the private store lists.
const (
	storeSellAvailableSize = 30 // PrivateStoreManageListSell, section 2
	storeSellItemSize      = 34 // PrivateStoreManageListSell section 3, PrivateStoreListSell
	storeBuyInventorySize  = 22 // PrivateStoreManageListBuy, section 2
	storeBuyItemSize       = 30 // PrivateStoreManageListBuy, section 3
	storeBuyListItemSize   = 34 // PrivateStoreListBuy
)

// StoreItem is an item of a private store list or of a store's setup
// window (a Mobius TradeItem). Every packet sends the template ID as the
// item ID.
type StoreItem struct {
	ObjectID int32
	ItemID   int32 // the item template ID
	Count    int32
	// Type2 is the template's type 2 (see ItemInfo).
	Type2 int16
	// BodyPart is the body part mask of the template, 0 for items that are
	// not worn.
	BodyPart    int32
	Enchant     int16
	CustomType2 int16
	// Price is the unit price the store owner set; in the available items
	// of PrivateStoreManageListSell it is the reference price.
	Price int32
	// ReferencePrice is the template's reference price (data.Item.Price).
	ReferencePrice int32
}

// PrivateStoreManageListSell is the setup window of a sell store.
type PrivateStoreManageListSell struct {
	ObjectID    int32 // the store owner
	PackageSale bool
	Adena       int32 // the adena the owner carries
	// Available are the inventory items the owner can still offer (Mobius
	// getAvailableItems(sellList)): a stackable item with its count less
	// what the list already holds, and Price set to the reference price.
	Available []StoreItem
	// Items are the items already in the sell list.
	Items []StoreItem
}

// NewPrivateStoreManageListSellPacket builds PrivateStoreManageListSell
// (Mobius PrivateStoreManageListSell.writeImpl). The sell list items are
// written without their CustomType2.
func NewPrivateStoreManageListSellPacket(l PrivateStoreManageListSell) []byte {
	w := packets.NewWriter(21 + len(l.Available)*storeSellAvailableSize + len(l.Items)*storeSellItemSize)
	w.WriteUint8(OpPrivateStoreManageListSell)
	w.WriteInt32(l.ObjectID)
	putBool32(w, l.PackageSale)
	w.WriteInt32(l.Adena)
	putCount32(w, len(l.Available))
	for _, it := range l.Available {
		w.WriteInt32(int32(it.Type2))
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
		w.WriteUint16(0)
		putInt16(w, it.Enchant)
		putInt16(w, it.CustomType2)
		w.WriteInt32(it.BodyPart)
		w.WriteInt32(it.Price)
	}
	putCount32(w, len(l.Items))
	for _, it := range l.Items {
		w.WriteInt32(int32(it.Type2))
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
		w.WriteUint16(0)
		putInt16(w, it.Enchant)
		w.WriteUint16(0)
		w.WriteInt32(it.BodyPart)
		w.WriteInt32(it.Price)
		w.WriteInt32(it.ReferencePrice)
	}
	return w.Bytes()
}

// PrivateStoreListSell is the window of a sell store, as a buyer sees it.
type PrivateStoreListSell struct {
	SellerID int32
	// PackageSale is set for a package sale: the buyer takes everything.
	PackageSale bool
	Adena       int32 // the adena the buyer carries
	Items       []StoreItem
}

// NewPrivateStoreListSellPacket builds PrivateStoreListSell (Mobius
// PrivateStoreListSell.writeImpl, without the sell-buffs custom feature).
func NewPrivateStoreListSellPacket(l PrivateStoreListSell) []byte {
	w := packets.NewWriter(17 + len(l.Items)*storeSellItemSize)
	w.WriteUint8(OpPrivateStoreListSell)
	w.WriteInt32(l.SellerID)
	putBool32(w, l.PackageSale)
	w.WriteInt32(l.Adena)
	putCount32(w, len(l.Items))
	for _, it := range l.Items {
		w.WriteInt32(int32(it.Type2))
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
		w.WriteUint16(0)
		putInt16(w, it.Enchant)
		putInt16(w, it.CustomType2)
		w.WriteInt32(it.BodyPart)
		w.WriteInt32(it.Price)
		w.WriteInt32(it.ReferencePrice)
	}
	return w.Bytes()
}

// PrivateStoreManageListBuy is the setup window of a buy store.
type PrivateStoreManageListBuy struct {
	ObjectID int32 // the store owner
	Adena    int32 // the adena the owner carries
	// Inventory are the owner's items that a buy list can name (Mobius
	// getUniqueItems(false, true, true, false): one per template, without
	// adena, not equipped, tradeable). Their ObjectID, Enchant, CustomType2
	// and Price are not sent; ReferencePrice is sent as the price.
	Inventory []StoreItem
	// Items are the items already in the buy list. Their ObjectID, Enchant
	// and CustomType2 are not sent.
	Items []StoreItem
}

// NewPrivateStoreManageListBuyPacket builds PrivateStoreManageListBuy
// (Mobius PrivateStoreManageListBuy.writeImpl).
func NewPrivateStoreManageListBuyPacket(l PrivateStoreManageListBuy) []byte {
	w := packets.NewWriter(17 + len(l.Inventory)*storeBuyInventorySize + len(l.Items)*storeBuyItemSize)
	w.WriteUint8(OpPrivateStoreManageListBuy)
	w.WriteInt32(l.ObjectID)
	w.WriteInt32(l.Adena)
	putCount32(w, len(l.Inventory))
	for _, it := range l.Inventory {
		w.WriteInt32(it.ItemID)
		w.WriteUint16(0) // no enchanted items in a buy store
		w.WriteInt32(it.Count)
		w.WriteInt32(it.ReferencePrice)
		w.WriteUint16(0)
		w.WriteInt32(it.BodyPart)
		putInt16(w, it.Type2)
	}
	putCount32(w, len(l.Items))
	for _, it := range l.Items {
		w.WriteInt32(it.ItemID)
		w.WriteUint16(0)
		w.WriteInt32(it.Count)
		w.WriteInt32(it.ReferencePrice)
		w.WriteUint16(0)
		w.WriteInt32(it.BodyPart)
		putInt16(w, it.Type2)
		w.WriteInt32(it.Price)
		w.WriteInt32(it.ReferencePrice)
	}
	return w.Bytes()
}

// PrivateStoreListBuy is the window of a buy store, as a seller sees it.
type PrivateStoreListBuy struct {
	StorePlayerID int32
	Adena         int32 // the adena the seller (the viewer) carries
	// Items are the entries of the buy list matched against the viewer's
	// inventory (Mobius TradeList.getAvailableItems(inventory)): ObjectID
	// and Enchant are those of the viewer's item of that template, and
	// Count is what he can sell (the wanted count, cut to what he has, 0
	// when he has none that is not equipped). StoreCount is the count the
	// store still wants. CustomType2 is not sent.
	Items []StoreBuyItem
}

// StoreBuyItem is one entry of PrivateStoreListBuy.
type StoreBuyItem struct {
	StoreItem
	StoreCount int32
}

// NewPrivateStoreListBuyPacket builds PrivateStoreListBuy (Mobius
// PrivateStoreListBuy.writeImpl).
func NewPrivateStoreListBuyPacket(l PrivateStoreListBuy) []byte {
	w := packets.NewWriter(13 + len(l.Items)*storeBuyListItemSize)
	w.WriteUint8(OpPrivateStoreListBuy)
	w.WriteInt32(l.StorePlayerID)
	w.WriteInt32(l.Adena)
	putCount32(w, len(l.Items))
	for _, it := range l.Items {
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		putInt16(w, it.Enchant)
		w.WriteInt32(it.Count)
		w.WriteInt32(it.ReferencePrice)
		w.WriteUint16(0)
		w.WriteInt32(it.BodyPart)
		putInt16(w, it.Type2)
		w.WriteInt32(it.Price)
		w.WriteInt32(it.StoreCount)
	}
	return w.Bytes()
}

// NewPrivateStoreMsgSellPacket builds PrivateStoreMsgSell (Mobius
// PrivateStoreMsgSell.writeImpl): the message over a sell store. An empty
// message stands for Mobius's null title.
func NewPrivateStoreMsgSellPacket(objectID int32, message string) []byte {
	return newStoreMsgPacket(OpPrivateStoreMsgSell, objectID, message)
}

// NewPrivateStoreMsgBuyPacket builds PrivateStoreMsgBuy (Mobius
// PrivateStoreMsgBuy.writeImpl): the message over a buy store.
func NewPrivateStoreMsgBuyPacket(objectID int32, message string) []byte {
	return newStoreMsgPacket(OpPrivateStoreMsgBuy, objectID, message)
}

func newStoreMsgPacket(op uint8, objectID int32, message string) []byte {
	w := packets.NewWriter(7 + 2*len(message))
	w.WriteUint8(op)
	w.WriteInt32(objectID)
	w.WriteString(message)
	return w.Bytes()
}
