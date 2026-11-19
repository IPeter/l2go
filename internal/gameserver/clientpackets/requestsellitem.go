package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestSellItem is the opcode of RequestSellItem.
const OpRequestSellItem = 0x1e

// sellItemSize is the size of one RequestSellItem entry: d object ID,
// d item ID, d count.
const sellItemSize = 12

// SellItem is one item of RequestSellItem.
type SellItem struct {
	ObjectID int32
	ItemID   int32 // the item template ID
	Count    int32
}

// RequestSellItem sells inventory items to the merchant the player talks
// to.
type RequestSellItem struct {
	// ListID is 0 from the sell window of SellList; Mobius checks any other
	// value as a buylist ID.
	ListID int32
	// Items are the items to sell, in the client's order. Items is nil when
	// Mobius refuses the list: no entries, more than MaxItemsInPacket, a
	// count that does not match the data, or an entry with an object ID,
	// item ID or count below 1. Mobius then answers with ActionFailed.
	Items []SellItem
}

// ParseRequestSellItem parses a RequestSellItem body (Mobius
// RequestSellItem.readImpl). Only a body too short for the list ID and the
// count is an error; a refused list gives nil Items.
func ParseRequestSellItem(data []byte) (RequestSellItem, error) {
	r := packets.NewReader(data)
	p := RequestSellItem{ListID: r.ReadInt32()}
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestSellItem{}, fmt.Errorf("RequestSellItem: %w", err)
	}
	if !itemBatchFits(r, n, sellItemSize) {
		return p, nil
	}
	items := make([]SellItem, n)
	for i := range items {
		it := SellItem{ObjectID: r.ReadInt32(), ItemID: r.ReadInt32(), Count: r.ReadInt32()}
		if it.ObjectID < 1 || it.ItemID < 1 || it.Count < 1 {
			return p, nil
		}
		items[i] = it
	}
	p.Items = items
	return p, nil
}
