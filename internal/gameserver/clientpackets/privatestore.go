package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// The client packets of private stores: a player who sells (a sell store
// or a package sale) or buys (a buy store) from the spot where he sits.

// Opcodes of the private store packets (Mobius ClientPackets).
const (
	OpRequestPrivateStoreManageSell = 0x73
	OpSetPrivateStoreListSell       = 0x74
	OpRequestPrivateStoreQuitSell   = 0x76
	OpSetPrivateStoreMsgSell        = 0x77
	OpRequestPrivateStoreBuy        = 0x79
	OpRequestPrivateStoreManageBuy  = 0x90
	OpSetPrivateStoreListBuy        = 0x91
	OpRequestPrivateStoreQuitBuy    = 0x93
	OpSetPrivateStoreMsgBuy         = 0x94
	OpRequestPrivateStoreSell       = 0x96
)

// Action IDs of RequestActionUse that open the setup window of a store.
const (
	ActionUsePrivateStoreSell = 10
	ActionUsePrivateStoreBuy  = 28
)

// PrivateStoreMessageMaxLength is the longest store message Mobius takes,
// in UTF-16 code units (MAX_MSG_LENGTH of SetPrivateStoreMsgSell and
// SetPrivateStoreMsgBuy). A longer one is punished as an illegal action.
const PrivateStoreMessageMaxLength = 29

// Entry sizes of the private store lists (Mobius BATCH_LENGTH, except for
// SetPrivateStoreListBuy, see there).
const (
	storeListSellItemSize    = 12 // d object ID, d count, d price
	storeListBuyItemSize     = 16 // d item ID, h, h, d count, d price
	storeListBuyBatchLength  = 12 // the BATCH_LENGTH SetPrivateStoreListBuy checks
	storeBuyRequestItemSize  = 12 // d object ID, d count, d price
	storeSellRequestItemSize = 20 // d object ID, d item ID, h, h, d count, d price
)

// RequestPrivateStoreManageSell asks for the setup window of a sell store.
// Mobius reads nothing.
type RequestPrivateStoreManageSell struct{}

// ParseRequestPrivateStoreManageSell parses a RequestPrivateStoreManageSell
// body. It never fails.
func ParseRequestPrivateStoreManageSell([]byte) (RequestPrivateStoreManageSell, error) {
	return RequestPrivateStoreManageSell{}, nil
}

// RequestPrivateStoreQuitSell closes the sell store or its setup window.
// Mobius reads nothing.
type RequestPrivateStoreQuitSell struct{}

// ParseRequestPrivateStoreQuitSell parses a RequestPrivateStoreQuitSell
// body. It never fails.
func ParseRequestPrivateStoreQuitSell([]byte) (RequestPrivateStoreQuitSell, error) {
	return RequestPrivateStoreQuitSell{}, nil
}

// RequestPrivateStoreManageBuy asks for the setup window of a buy store.
// Mobius reads nothing.
type RequestPrivateStoreManageBuy struct{}

// ParseRequestPrivateStoreManageBuy parses a RequestPrivateStoreManageBuy
// body. It never fails.
func ParseRequestPrivateStoreManageBuy([]byte) (RequestPrivateStoreManageBuy, error) {
	return RequestPrivateStoreManageBuy{}, nil
}

// RequestPrivateStoreQuitBuy closes the buy store or its setup window.
// Mobius reads nothing.
type RequestPrivateStoreQuitBuy struct{}

// ParseRequestPrivateStoreQuitBuy parses a RequestPrivateStoreQuitBuy body.
// It never fails.
func ParseRequestPrivateStoreQuitBuy([]byte) (RequestPrivateStoreQuitBuy, error) {
	return RequestPrivateStoreQuitBuy{}, nil
}

// SetPrivateStoreMsgSell sets the message shown over a sell store.
type SetPrivateStoreMsgSell struct {
	// Message is read like the Mobius readString: it never fails and runs
	// to the end of the data without a terminator. Mobius refuses one
	// longer than PrivateStoreMessageMaxLength.
	Message string
}

// ParseSetPrivateStoreMsgSell parses a SetPrivateStoreMsgSell body (Mobius
// SetPrivateStoreMsgSell.readImpl): S message. It never fails.
func ParseSetPrivateStoreMsgSell(data []byte) (SetPrivateStoreMsgSell, error) {
	return SetPrivateStoreMsgSell{Message: htmlActionText(data)}, nil
}

// SetPrivateStoreMsgBuy sets the message shown over a buy store.
type SetPrivateStoreMsgBuy struct {
	// Message is read like SetPrivateStoreMsgSell.Message.
	Message string
}

// ParseSetPrivateStoreMsgBuy parses a SetPrivateStoreMsgBuy body (Mobius
// SetPrivateStoreMsgBuy.readImpl): S message. It never fails.
func ParseSetPrivateStoreMsgBuy(data []byte) (SetPrivateStoreMsgBuy, error) {
	return SetPrivateStoreMsgBuy{Message: htmlActionText(data)}, nil
}

// SetPrivateStoreListSellItem is one inventory item offered by a sell
// store.
type SetPrivateStoreListSellItem struct {
	ObjectID int32
	Count    int32
	Price    int32 // per unit
}

// SetPrivateStoreListSell opens a sell store with the items of its setup
// window.
type SetPrivateStoreListSell struct {
	// PackageSale is set when the int the client sends is exactly 1: the
	// buyer must take the whole list.
	PackageSale bool
	// Items is nil when Mobius refuses the list: no entries, more than
	// MaxItemsInPacket, a count that does not match the data, or an entry
	// with an object ID or count below 1 or a negative price. Mobius then
	// sends INCORRECT_ITEM_COUNT and closes the store.
	Items []SetPrivateStoreListSellItem
}

// ParseSetPrivateStoreListSell parses a SetPrivateStoreListSell body
// (Mobius SetPrivateStoreListSell.readImpl): d package sale, d count, then
// per item d object ID, d count, d price. Only a body too short for the two
// header ints is an error; a refused list gives nil Items.
func ParseSetPrivateStoreListSell(data []byte) (SetPrivateStoreListSell, error) {
	r := packets.NewReader(data)
	p := SetPrivateStoreListSell{PackageSale: r.ReadInt32() == 1}
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return SetPrivateStoreListSell{}, fmt.Errorf("SetPrivateStoreListSell: %w", err)
	}
	if !itemBatchFits(r, n, storeListSellItemSize) {
		return p, nil
	}
	items := make([]SetPrivateStoreListSellItem, n)
	for i := range items {
		it := SetPrivateStoreListSellItem{ObjectID: r.ReadInt32(), Count: r.ReadInt32(), Price: r.ReadInt32()}
		if it.ObjectID < 1 || it.Count < 1 || it.Price < 0 {
			return p, nil
		}
		items[i] = it
	}
	p.Items = items
	return p, nil
}

// SetPrivateStoreListBuyItem is one item wanted by a buy store.
type SetPrivateStoreListBuyItem struct {
	ItemID int32 // the item template ID
	Count  int32
	Price  int32 // per unit
}

// SetPrivateStoreListBuy opens a buy store with the items of its setup
// window.
type SetPrivateStoreListBuy struct {
	// Items is nil when Mobius refuses the list: no entries, more than
	// MaxItemsInPacket, fewer than 12 bytes per entry (see
	// ParseSetPrivateStoreListBuy), or an entry with an item ID or count
	// below 1 or a negative price. Mobius then sends INCORRECT_ITEM_COUNT
	// and closes the store.
	Items []SetPrivateStoreListBuyItem
}

// ParseSetPrivateStoreListBuy parses a SetPrivateStoreListBuy body (Mobius
// SetPrivateStoreListBuy.readImpl): d count, then per item d item ID, two
// shorts Mobius ignores, d count, d price.
//
// Unlike the other lists, Mobius checks the count against a BATCH_LENGTH
// of 12 although an entry has 16 bytes, and only as an upper bound: bytes
// after the entries are ignored, and a list that passes the check but
// runs out of data inside an entry fails to read, so Mobius drops the
// packet. That case is an error here, like a body without the count. The
// entries are checked as they are read, as in Mobius, so a bad entry
// before the end of the data refuses the list without an error.
func ParseSetPrivateStoreListBuy(data []byte) (SetPrivateStoreListBuy, error) {
	r := packets.NewReader(data)
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return SetPrivateStoreListBuy{}, fmt.Errorf("SetPrivateStoreListBuy: %w", err)
	}
	if n < 1 || n > MaxItemsInPacket || int(n)*storeListBuyBatchLength > r.Len() {
		return SetPrivateStoreListBuy{}, nil
	}
	items := make([]SetPrivateStoreListBuyItem, n)
	for i := range items {
		var it SetPrivateStoreListBuyItem
		it.ItemID = r.ReadInt32()
		r.ReadUint16() // "Enchant?"
		r.ReadUint16() // "Price per unit"
		it.Count = r.ReadInt32()
		it.Price = r.ReadInt32()
		if err := r.Err(); err != nil {
			return SetPrivateStoreListBuy{}, fmt.Errorf("SetPrivateStoreListBuy: %w", err)
		}
		if it.ItemID < 1 || it.Count < 1 || it.Price < 0 {
			return SetPrivateStoreListBuy{}, nil
		}
		items[i] = it
	}
	return SetPrivateStoreListBuy{Items: items}, nil
}

// RequestPrivateStoreBuyItem is one item bought from a sell store.
type RequestPrivateStoreBuyItem struct {
	ObjectID int32 // the item in the seller's inventory
	Count    int32
	Price    int32 // the unit price the buyer saw
}

// RequestPrivateStoreBuy buys items from a sell store.
type RequestPrivateStoreBuy struct {
	StorePlayerID int32
	// Items are the items to buy, in the client's order, duplicates
	// included. Mobius collects them in a HashSet whose RequestTrade.equals
	// is inverted, so it keeps every entry but goes through them in hash
	// order (by object ID), not in the client's. Items is nil when Mobius
	// refuses the list: no entries, more than MaxItemsInPacket, a count that
	// does not match the data, or an entry with an object ID or count below
	// 1 or a negative price. Mobius then answers with ActionFailed.
	Items []RequestPrivateStoreBuyItem
}

// ParseRequestPrivateStoreBuy parses a RequestPrivateStoreBuy body (Mobius
// RequestPrivateStoreBuy.readImpl): d store player, d count, then per item
// d object ID, d count, d price. Only a body too short for the two header
// ints is an error; a refused list gives nil Items.
func ParseRequestPrivateStoreBuy(data []byte) (RequestPrivateStoreBuy, error) {
	r := packets.NewReader(data)
	p := RequestPrivateStoreBuy{StorePlayerID: r.ReadInt32()}
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestPrivateStoreBuy{}, fmt.Errorf("RequestPrivateStoreBuy: %w", err)
	}
	if !itemBatchFits(r, n, storeBuyRequestItemSize) {
		return p, nil
	}
	items := make([]RequestPrivateStoreBuyItem, n)
	for i := range items {
		it := RequestPrivateStoreBuyItem{ObjectID: r.ReadInt32(), Count: r.ReadInt32(), Price: r.ReadInt32()}
		if it.ObjectID < 1 || it.Count < 1 || it.Price < 0 {
			return p, nil
		}
		items[i] = it
	}
	p.Items = items
	return p, nil
}

// RequestPrivateStoreSellItem is one item sold to a buy store.
type RequestPrivateStoreSellItem struct {
	ObjectID int32 // the item in the seller's own inventory
	ItemID   int32 // the item template ID
	Count    int32
	Price    int32 // the unit price the seller saw
}

// RequestPrivateStoreSell sells items to a buy store.
type RequestPrivateStoreSell struct {
	StorePlayerID int32
	// Items is nil when Mobius refuses the list: no entries, more than
	// MaxItemsInPacket, a count that does not match the data, or an entry
	// with an object ID, item ID or count below 1 or a negative price.
	// Mobius then answers with ActionFailed.
	Items []RequestPrivateStoreSellItem
}

// ParseRequestPrivateStoreSell parses a RequestPrivateStoreSell body
// (Mobius RequestPrivateStoreSell.readImpl): d store player, d count, then
// per item d object ID, d item ID, two shorts Mobius ignores, d count, d
// price. Only a body too short for the two header ints is an error; a
// refused list gives nil Items.
func ParseRequestPrivateStoreSell(data []byte) (RequestPrivateStoreSell, error) {
	r := packets.NewReader(data)
	p := RequestPrivateStoreSell{StorePlayerID: r.ReadInt32()}
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestPrivateStoreSell{}, fmt.Errorf("RequestPrivateStoreSell: %w", err)
	}
	if !itemBatchFits(r, n, storeSellRequestItemSize) {
		return p, nil
	}
	items := make([]RequestPrivateStoreSellItem, n)
	for i := range items {
		var it RequestPrivateStoreSellItem
		it.ObjectID = r.ReadInt32()
		it.ItemID = r.ReadInt32()
		r.ReadUint16()
		r.ReadUint16()
		it.Count = r.ReadInt32()
		it.Price = r.ReadInt32()
		if it.ObjectID < 1 || it.ItemID < 1 || it.Count < 1 || it.Price < 0 {
			return p, nil
		}
		items[i] = it
	}
	p.Items = items
	return p, nil
}
