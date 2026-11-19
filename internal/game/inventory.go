package game

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// Item is an item a player owns: the stored item and its template.
type Item struct {
	storage.Item
	// Template is the item's template, or nil when the data has none for
	// ItemID. Such an item is kept and saved, but not shown or worn (Mobius
	// skips items it has no template for).
	Template *data.Item
}

// Inventory is a player's items: all of them in storage order, and the worn
// ones by paperdoll slot (Mobius PcInventory). It is read-only until items
// can change in Milestone 5.
type Inventory struct {
	items     []*Item
	paperdoll [storage.PaperdollTotalSlots]*Item
}

// newInventory builds the inventory of a loaded character. Items without a
// template are logged and kept.
func (s *Server) newInventory(c storage.Character, logArgs []any) Inventory {
	var inv Inventory
	inv.items = make([]*Item, 0, len(c.Items))
	for _, si := range c.Items {
		it := &Item{Item: si}
		if t, ok := s.data.ItemTemplate(int(si.ItemID)); ok {
			it.Template = t
		} else {
			s.log.Warn("game: item without a template, not shown", append(logArgs, "item", si.ItemID, "itemObjectId", si.ObjectID)...)
		}
		inv.items = append(inv.items, it)
		if si.Slot.Equipped() && it.Template != nil {
			if prev := inv.paperdoll[si.Slot]; prev != nil {
				s.log.Warn("game: two items in one paperdoll slot, keeping the last", append(logArgs, "slot", si.Slot, "item", prev.ItemID, "other", si.ItemID)...)
			}
			inv.paperdoll[si.Slot] = it
		}
	}
	return inv
}

// Items returns the items in storage order. The slice must not be changed.
func (inv *Inventory) Items() []*Item { return inv.items }

// Paperdoll returns the item worn in slot, or nil.
func (inv *Inventory) Paperdoll(slot storage.PaperdollSlot) *Item {
	if !slot.Equipped() {
		return nil
	}
	return inv.paperdoll[slot]
}

// Weight returns the weight of all items (Mobius Inventory.getTotalWeight),
// capped to the int range.
func (inv *Inventory) Weight() int32 {
	var w int64
	for _, it := range inv.items {
		if it.Template != nil {
			w += int64(it.Template.Weight) * it.Count
		}
	}
	return int32(min(max(w, 0), math.MaxInt32)) //nolint:gosec // clamped to the int32 range
}

// PacketPaperdoll returns the worn items as UserInfo and CharInfo show them:
// object IDs and display IDs by slot.
func (inv *Inventory) PacketPaperdoll() serverpackets.Paperdoll {
	var d serverpackets.Paperdoll
	for slot, it := range inv.paperdoll {
		if it != nil {
			d[slot] = serverpackets.EquippedItem{ObjectID: it.ObjectID, ItemID: int32(it.Template.DisplayID)} //nolint:gosec // item IDs fit in int32
		}
	}
	return d
}

// ItemInfos returns the items as the inventory packets list them, leaving
// out those without a template.
func (inv *Inventory) ItemInfos() []serverpackets.ItemInfo {
	infos := make([]serverpackets.ItemInfo, 0, len(inv.items))
	for _, it := range inv.items {
		if it.Template != nil {
			infos = append(infos, ItemInfo(it))
		}
	}
	return infos
}

// ItemInfo returns one item as the inventory packets write it (Mobius
// ItemInfo): the template's list types and body part mask, the display ID,
// the count cut to an int, and the mana of a shadow item (-1 for others).
// it.Template must not be nil.
func ItemInfo(it *Item) serverpackets.ItemInfo {
	t := it.Template
	mana := int32(-1)
	if t.Duration > 0 {
		mana = int32(min(t.Duration, math.MaxInt32)) //nolint:gosec // clamped to the int32 range
	}
	//nolint:gosec // list types are 0-5, IDs and masks fit in int32, the rest is clamped
	return serverpackets.ItemInfo{
		Type1:    int16(t.Type1),
		ObjectID: it.ObjectID,
		ItemID:   int32(t.DisplayID),
		Count:    int32(min(max(it.Count, 0), math.MaxInt32)),
		Type2:    int16(t.Type2),
		Equipped: it.Slot.Equipped(),
		BodyPart: int32(t.BodyPartMask),
		Enchant:  int16(min(max(it.Enchant, math.MinInt16), math.MaxInt16)),
		Mana:     mana,
	}
}

// storageItems returns the items as they are stored.
func (inv *Inventory) storageItems() []storage.Item {
	if len(inv.items) == 0 {
		return nil
	}
	out := make([]storage.Item, len(inv.items))
	for i, it := range inv.items {
		out[i] = it.Item
	}
	return out
}
