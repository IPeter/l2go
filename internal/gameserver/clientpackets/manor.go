package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// The manor's client packets (Plan 14 M11k): RequestProcureCropList (D0 09),
// RequestSetSeed (D0 0A), RequestSetCrop (D0 0B) and RequestBuySeed (0xC4).
// RequestManorList (D0 08) is in requestmanorlist.go. 0xC3
// (RequestBuyProcure) has no handler in Mobius and none here.

// Sub-opcodes and opcode of the manor's packets.
const (
	ExOpRequestProcureCropList = 0x09
	ExOpRequestSetSeed         = 0x0a
	ExOpRequestSetCrop         = 0x0b
	OpRequestBuySeed           = 0xc4
)

// Sizes of one entry of the manor's lists (Mobius BATCH_LENGTH).
const (
	procureCropItemSize = 16
	setSeedItemSize     = 12
	setCropItemSize     = 13
	buySeedItemSize     = 8
)

// ProcureCropItem is a crop the player sells to a manor.
type ProcureCropItem struct {
	ObjectID, ItemID int32
	ManorID          int32 // the castle whose manor buys it
	Count            int32
}

// RequestProcureCropList sells crops to the manors. Items is nil when Mobius
// refuses the list: no entries, more than MaxItemsInPacket, a size that does
// not match the data, or an entry with an object or item ID below 1 or a
// negative manor or count.
type RequestProcureCropList struct {
	Items []ProcureCropItem
}

// ParseRequestProcureCropList parses a RequestProcureCropList body (Mobius
// readImpl). Only a body too short for the count is an error.
func ParseRequestProcureCropList(data []byte) (RequestProcureCropList, error) {
	r := packets.NewReader(data)
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestProcureCropList{}, fmt.Errorf("RequestProcureCropList: %w", err)
	}
	var p RequestProcureCropList
	if !itemBatchFits(r, n, procureCropItemSize) {
		return p, nil
	}
	items := make([]ProcureCropItem, n)
	for i := range items {
		it := ProcureCropItem{ObjectID: r.ReadInt32(), ItemID: r.ReadInt32(), ManorID: r.ReadInt32(), Count: r.ReadInt32()}
		if it.ObjectID < 1 || it.ItemID < 1 || it.ManorID < 0 || it.Count < 0 {
			return p, nil
		}
		items[i] = it
	}
	p.Items = items
	return p, nil
}

// SetSeedItem is a seed the lord's castle sells in the next period.
type SetSeedItem struct {
	SeedID, Sales, Price int32
}

// RequestSetSeed sets the seeds a castle's manor sells in the next period.
// Items holds the entries with sales above 0; it is empty when Mobius
// refuses the list (a bad size, an item ID below 1, negative sales or price)
// or none sells anything, and then nothing happens.
type RequestSetSeed struct {
	ManorID int32
	Items   []SetSeedItem
}

// ParseRequestSetSeed parses a RequestSetSeed body (Mobius readImpl). Only a
// body too short for the manor and the count is an error.
func ParseRequestSetSeed(data []byte) (RequestSetSeed, error) {
	r := packets.NewReader(data)
	p := RequestSetSeed{ManorID: r.ReadInt32()}
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestSetSeed{}, fmt.Errorf("RequestSetSeed: %w", err)
	}
	if !itemBatchFits(r, n, setSeedItemSize) {
		return p, nil
	}
	var items []SetSeedItem
	for range n {
		it := SetSeedItem{SeedID: r.ReadInt32(), Sales: r.ReadInt32(), Price: r.ReadInt32()}
		if it.SeedID < 1 || it.Sales < 0 || it.Price < 0 {
			return p, nil
		}
		if it.Sales > 0 {
			items = append(items, it)
		}
	}
	p.Items = items
	return p, nil
}

// SetCropItem is a crop the lord's castle buys in the next period, paid with
// reward type Reward.
type SetCropItem struct {
	CropID, Sales, Price int32
	Reward               int8
}

// RequestSetCrop sets the crops a castle's manor buys in the next period,
// with RequestSetSeed's rules.
type RequestSetCrop struct {
	ManorID int32
	Items   []SetCropItem
}

// ParseRequestSetCrop parses a RequestSetCrop body (Mobius readImpl).
func ParseRequestSetCrop(data []byte) (RequestSetCrop, error) {
	r := packets.NewReader(data)
	p := RequestSetCrop{ManorID: r.ReadInt32()}
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestSetCrop{}, fmt.Errorf("RequestSetCrop: %w", err)
	}
	if !itemBatchFits(r, n, setCropItemSize) {
		return p, nil
	}
	var items []SetCropItem
	for range n {
		it := SetCropItem{CropID: r.ReadInt32(), Sales: r.ReadInt32(), Price: r.ReadInt32(), Reward: int8(r.ReadUint8())} //nolint:gosec // Java's signed byte
		if it.CropID < 1 || it.Sales < 0 || it.Price < 0 {
			return p, nil
		}
		if it.Sales > 0 {
			items = append(items, it)
		}
	}
	p.Items = items
	return p, nil
}

// BuySeedItem is a seed the player buys.
type BuySeedItem struct {
	SeedID, Count int32
}

// RequestBuySeed buys seeds at a manor. Items is nil when Mobius refuses the
// list: a bad size, a count below 1 or an item ID below 1.
type RequestBuySeed struct {
	ManorID int32
	Items   []BuySeedItem
}

// ParseRequestBuySeed parses a RequestBuySeed body (Mobius readImpl).
func ParseRequestBuySeed(data []byte) (RequestBuySeed, error) {
	r := packets.NewReader(data)
	p := RequestBuySeed{ManorID: r.ReadInt32()}
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestBuySeed{}, fmt.Errorf("RequestBuySeed: %w", err)
	}
	if !itemBatchFits(r, n, buySeedItemSize) {
		return p, nil
	}
	items := make([]BuySeedItem, n)
	for i := range items {
		it := BuySeedItem{SeedID: r.ReadInt32(), Count: r.ReadInt32()}
		if it.Count < 1 || it.SeedID < 1 {
			return p, nil
		}
		items[i] = it
	}
	p.Items = items
	return p, nil
}
