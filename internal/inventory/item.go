// Package inventory holds a character's items: item instances, containers
// (the inventory and the private warehouse), the paperdoll with the Mobius
// equip rules, weight and slot limits, the changes that feed
// InventoryUpdate, and the save diff of decision I2 (Plan 08, decisions I1
// to I15).
//
// It follows Mobius entity/itemcontainer (ItemContainer, Inventory,
// PlayerInventory, PlayerWarehouse) and entity/item/instance/Item. The
// package is pure: no world, packets or I/O. The game adapts it to the
// world loop, the packets and the saver. Nothing here is safe for
// concurrent use; the world loop owns every container.
package inventory

import (
	"errors"
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
)

// Well-known item IDs (Mobius Inventory).
const (
	AdenaID        int32 = 57
	AncientAdenaID int32 = 5575
)

// DefaultMaxAdena is Mobius MaxAdena (Player.ini).
const DefaultMaxAdena int64 = 2_000_000_000

// LocVoid is the location of an item that is in no container, such as one
// lying on the ground (Mobius ItemLocation.VOID). Such items are not saved.
const LocVoid storage.ItemLocation = "VOID"

// Change is the last change of an item, as InventoryUpdate sends it (Mobius
// Item.ADDED, MODIFIED, REMOVED).
type Change uint8

// The changes. The values are the InventoryUpdate change codes.
const (
	Unchanged Change = 0
	Added     Change = 1
	Modified  Change = 2
	Removed   Change = 3
)

// Errors of the container operations.
var (
	ErrNotFound     = errors.New("inventory: no such item in the container")
	ErrInvalidCount = errors.New("inventory: invalid item count")
	ErrUnknownItem  = errors.New("inventory: unknown item template")
	ErrNoObjectID   = errors.New("inventory: no object ID available")
	ErrInContainer  = errors.New("inventory: the item is in a container")
)

// Item is an item instance (Mobius Item): a template with a count, an
// enchant level, a location and the mana of a shadow item. Only the
// containers change it, so that its container records the change and the
// save diff.
type Item struct {
	objectID int32
	tmpl     *data.Item
	count    int64
	enchant  int32
	loc      storage.ItemLocation
	slot     storage.PaperdollSlot
	mana     int32
	custom1  int32
	custom2  int32

	lastChange Change
	dirty      bool
	c          *Container // the container holding the item, nil if none
}

// NewItem returns an item that is in no container (LocVoid), such as a
// drop: count items of the template, or one if it does not stack, with
// enchant 0 and the template's mana (Mobius ItemManager.createItem).
// Add puts it into a container.
func NewItem(objectID int32, t *data.Item, count int64) *Item {
	it := &Item{
		objectID:   objectID,
		tmpl:       t,
		count:      1,
		loc:        LocVoid,
		slot:       storage.PaperdollNone,
		mana:       storage.NoMana,
		lastChange: Modified,
	}
	if t.Stackable && count > 1 {
		it.count = count
	}
	if t.Duration > 0 {
		it.mana = int32(min(t.Duration, math.MaxInt32)) //nolint:gosec // clamped to the int32 range
	}
	return it
}

// fromStorage returns the item of a stored item, normalized.
func fromStorage(si storage.Item, t *data.Item) *Item {
	si = si.Normalized()
	return &Item{
		objectID:   si.ObjectID,
		tmpl:       t,
		count:      si.Count,
		enchant:    si.Enchant,
		loc:        si.Loc,
		slot:       si.Slot,
		mana:       si.Mana,
		custom1:    si.CustomType1,
		custom2:    si.CustomType2,
		lastChange: Modified,
	}
}

// ObjectID returns the item's object ID.
func (it *Item) ObjectID() int32 { return it.objectID }

// ItemID returns the template ID.
func (it *Item) ItemID() int32 { return int32(it.tmpl.ID) } //nolint:gosec // item IDs are below 2^20

// Template returns the item template. It must not be changed.
func (it *Item) Template() *data.Item { return it.tmpl }

// Count returns how many the item is: the stack size, 1 for items that do
// not stack.
func (it *Item) Count() int64 { return it.count }

// Enchant returns the enchant level.
func (it *Item) Enchant() int32 { return it.enchant }

// Loc returns where the item is kept.
func (it *Item) Loc() storage.ItemLocation { return it.loc }

// Slot returns the paperdoll slot the item is worn in, or
// storage.PaperdollNone.
func (it *Item) Slot() storage.PaperdollSlot { return it.slot }

// Mana returns the mana a shadow item has left, in minutes, or
// storage.NoMana.
func (it *Item) Mana() int32 { return it.mana }

// LastChange returns the item's last change (Mobius getLastChange).
func (it *Item) LastChange() Change { return it.lastChange }

// Dirty reports whether the item changed since its container's tracker
// last handed out the save diff.
func (it *Item) Dirty() bool { return it.dirty }

// Container returns the container holding the item, or nil.
func (it *Item) Container() *Container { return it.c }

// Equipped reports whether the item is worn (Mobius isEquipped).
func (it *Item) Equipped() bool { return it.loc == storage.LocPaperdoll }

// Stackable reports whether items of the template stack.
func (it *Item) Stackable() bool { return it.tmpl.Stackable }

// Shadow reports whether the item has mana (Mobius isShadowItem).
func (it *Item) Shadow() bool { return it.mana >= 0 }

// Equipable reports whether the item can be put on with UseItem: it has a
// body part and is neither arrows nor a lure (Mobius Item.isEquipable).
func (it *Item) Equipable() bool {
	return it.tmpl.Part() != data.BodyPartNone && !isEtc(it.tmpl, data.EtcArrow) && !isEtc(it.tmpl, data.EtcLure)
}

// Depositable reports whether the item may go into a warehouse: the
// template allows it and it is not worn; the clan warehouse and freight
// (private false) also need it tradable and without mana (Mobius
// isDepositable).
func (it *Item) Depositable(private bool) bool {
	if !it.tmpl.Depositable || it.Equipped() {
		return false
	}
	return private || (it.tmpl.Tradable && !it.Shadow())
}

// Storage returns the item as the stores keep it.
func (it *Item) Storage() storage.Item {
	return storage.Item{
		ObjectID:    it.objectID,
		ItemID:      it.ItemID(),
		Count:       it.count,
		Enchant:     it.enchant,
		Slot:        it.slot,
		Loc:         it.loc,
		Mana:        it.mana,
		CustomType1: it.custom1,
		CustomType2: it.custom2,
	}
}

// SetEnchant sets the enchant level, at least 0 (Mobius setEnchantLevel).
// It does not refresh the stats of a worn item: unequip and equip it, or
// call the stats listener again.
func (it *Item) SetEnchant(level int32) {
	level = max(level, 0)
	if level == it.enchant {
		return
	}
	it.enchant = level
	it.lastChange = Modified
	it.touch(Modified)
}

// SetMana sets the mana of a shadow item.
func (it *Item) SetMana(mana int32) {
	if mana == it.mana {
		return
	}
	it.mana = mana
	it.lastChange = Modified
	it.touch(Modified)
}

// touch records a change of an item in its container.
func (it *Item) touch(ch Change) {
	if it.c != nil {
		it.c.touch(it, ch)
	}
}

// changeCount adds delta to the count, capped at max and not below 0
// (Mobius changeCount).
func (it *Item) changeCount(delta, maxCount int64) {
	if delta == 0 {
		return
	}
	if delta > 0 && it.count > maxCount-delta {
		it.count = maxCount
	} else {
		it.count += delta
	}
	it.count = max(it.count, 0)
}

// isEtc reports whether t is an etc item of type et.
func isEtc(t *data.Item, et data.EtcItemType) bool { return t.Etc != nil && t.Etc.Type == et }

// isWeapon reports whether t is a weapon of type wt.
func isWeapon(t *data.Item, wt data.WeaponType) bool { return t.Weapon != nil && t.Weapon.Type == wt }
