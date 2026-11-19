package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestBuyItem is the opcode of RequestBuyItem.
const OpRequestBuyItem = 0x1f

// MaxBuyCount is the largest count of one product RequestBuyItem may ask
// for (the Mobius readImpl check).
const MaxBuyCount = 10000

// buyItemSize is the size of one RequestBuyItem entry: d item ID, d count.
const buyItemSize = 8

// BuyItem is one product of RequestBuyItem.
type BuyItem struct {
	ItemID int32 // the item template ID
	Count  int32
}

// RequestBuyItem buys products from the buylist the player has open.
type RequestBuyItem struct {
	ListID int32
	// Items are the products to buy, in the client's order. Items is nil
	// when Mobius refuses the list: no entries, more than MaxItemsInPacket,
	// a count that does not match the data, or an entry with an item ID or
	// count below 1 or a count above MaxBuyCount. Mobius then answers with
	// the message "You cannot buy more than 10.000 items." and ActionFailed.
	Items []BuyItem
}

// ParseRequestBuyItem parses a RequestBuyItem body (Mobius
// RequestBuyItem.readImpl). Only a body too short for the list ID and the
// count is an error; a refused list gives nil Items.
func ParseRequestBuyItem(data []byte) (RequestBuyItem, error) {
	r := packets.NewReader(data)
	p := RequestBuyItem{ListID: r.ReadInt32()}
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestBuyItem{}, fmt.Errorf("RequestBuyItem: %w", err)
	}
	if !itemBatchFits(r, n, buyItemSize) {
		return p, nil
	}
	items := make([]BuyItem, n)
	for i := range items {
		it := BuyItem{ItemID: r.ReadInt32(), Count: r.ReadInt32()}
		if it.ItemID < 1 || it.Count < 1 || it.Count > MaxBuyCount {
			return p, nil
		}
		items[i] = it
	}
	p.Items = items
	return p, nil
}
