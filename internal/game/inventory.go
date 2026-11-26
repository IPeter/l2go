package game

import (
	"math"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
)

// Inventory is a player's inventory (internal/inventory, Mobius
// PlayerInventory): the carried items, the paperdoll with the Mobius equip
// rules, weight and limits. It adds the packet views to it. Change it only
// through the Server's item methods (GiveItem, DestroyItem, ...) or call
// Server.ItemsChanged afterwards, so that the client, the weight penalty
// and the save follow.
type Inventory struct {
	*inventory.Inventory
}

// PacketPaperdoll returns the worn items as UserInfo and CharInfo show them:
// object IDs and display IDs by slot.
func (inv Inventory) PacketPaperdoll() serverpackets.Paperdoll {
	var d serverpackets.Paperdoll
	if inv.Inventory == nil {
		return d
	}
	for slot := range storage.PaperdollTotalSlots {
		if it := inv.Paperdoll(storage.PaperdollSlot(slot)); it != nil {
			d[slot] = serverpackets.EquippedItem{ObjectID: it.ObjectID(), ItemID: int32(it.Template().DisplayID)} //nolint:gosec // item IDs fit in int32
		}
	}
	return d
}

// ItemInfos returns the items as ItemList lists them, in inventory order.
func (inv Inventory) ItemInfos() []serverpackets.ItemInfo {
	if inv.Inventory == nil {
		return nil
	}
	return itemInfos(inv.Items())
}

// Load returns the weight of the carried items as the packets show it
// (Mobius getCurrentLoad).
func (inv Inventory) Load() int32 {
	if inv.Inventory == nil {
		return 0
	}
	return int32(min(inv.Weight(), math.MaxInt32)) //nolint:gosec // clamped to the int32 range
}

// itemInfos returns items as the item packets list them.
func itemInfos(items []*inventory.Item) []serverpackets.ItemInfo {
	infos := make([]serverpackets.ItemInfo, 0, len(items))
	for _, it := range items {
		infos = append(infos, ItemInfo(it))
	}
	return infos
}

// ItemInfo returns one item as the item packets write it (Mobius ItemInfo
// and AbstractItemPacket.writeItem): the template's list types and body
// part mask, the display ID, the count cut to an int, the enchant level and
// the augmentation and the mana left (-1 for items without mana).
func ItemInfo(it *inventory.Item) serverpackets.ItemInfo {
	t := it.Template()
	s := it.Storage()
	//nolint:gosec // list types are 0-5, IDs and masks fit in int32, the rest is clamped
	return serverpackets.ItemInfo{
		Type1:          int16(t.Type1),
		ObjectID:       it.ObjectID(),
		ItemID:         int32(t.DisplayID),
		Count:          itemCount32(it.Count()),
		Type2:          int16(t.Type2),
		CustomType1:    int16(s.CustomType1),
		Equipped:       it.Equipped(),
		BodyPart:       int32(t.BodyPartMask),
		Enchant:        int16(min(max(it.Enchant(), math.MinInt16), math.MaxInt16)),
		CustomType2:    int16(s.CustomType2),
		AugmentationID: it.Augmentation(),
		Mana:           it.Mana(),
	}
}

// storageItems returns every item p owns as it is stored: the carried
// ones, those in the warehouse and those the data has no template for.
func (p *Player) storageItems() []storage.Item {
	var out []storage.Item
	if p.inv.Inventory != nil {
		for _, it := range p.inv.Items() {
			out = append(out, it.Storage())
		}
	}
	if p.items.warehouse != nil {
		for _, it := range p.items.warehouse.Items() {
			out = append(out, it.Storage())
		}
	}
	if p.items.pet != nil {
		for _, it := range p.items.pet.Items() {
			out = append(out, it.Storage())
		}
	}
	return append(out, p.items.unknown...)
}

// takeItemChanges returns the changes to p's items since the last call,
// for a save.
func (p *Player) takeItemChanges() storage.ItemChanges {
	if p.items.tracker == nil {
		return storage.ItemChanges{}
	}
	return p.items.tracker.Take()
}
