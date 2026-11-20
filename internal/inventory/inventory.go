package inventory

import (
	"math"
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
)

// Limits are the owner's limits an inventory checks; *stats.Player
// implements it.
type Limits interface {
	// InventoryLimit is the number of inventory slots
	// (Player.getInventoryLimit).
	InventoryLimit() int
	// WeightLimit is the maximum load (Creature.getMaxLoad).
	WeightLimit() int
}

// Wearer is the owner's state that equipItem checks.
type Wearer struct {
	GM        bool
	Hero      bool
	StoreMode bool // a private store or manufacture is open
}

// Listener is told when an item goes into or comes out of a paperdoll slot
// (Mobius PaperdollListener). Listeners run in the order they were added,
// after the slot changed; they may change the paperdoll themselves.
type Listener interface {
	Equipped(inv *Inventory, slot storage.PaperdollSlot, it *Item)
	Unequipped(inv *Inventory, slot storage.PaperdollSlot, it *Item)
}

// Inventory is a player's inventory (Mobius PlayerInventory with Inventory):
// the carried items, the paperdoll and its listeners, weight and limits.
// Items are at storage.LocInventory, or storage.LocPaperdoll while worn.
type Inventory struct {
	Container
	owner      Limits
	paperdoll  [storage.PaperdollTotalSlots]*Item
	listeners  []Listener
	wearedMask int
}

// NewInventory returns an empty inventory of owner. Its first listener is
// the bow listener (Mobius BowCrossRodListener); add the stats listener
// before loading items.
func NewInventory(owner Limits, opts Options) *Inventory {
	inv := &Inventory{Container: newContainer(storage.LocInventory, opts), owner: owner}
	inv.inv = inv
	inv.listeners = []Listener{bowListener{}}
	return inv
}

// AddListener adds a paperdoll listener after the others.
func (inv *Inventory) AddListener(l Listener) {
	if !slices.Contains(inv.listeners, l) {
		inv.listeners = append(inv.listeners, l)
	}
}

// RemoveListener removes a paperdoll listener.
func (inv *Inventory) RemoveListener(l Listener) {
	if i := slices.Index(inv.listeners, l); i >= 0 {
		inv.listeners = slices.Delete(inv.listeners, i, i+1)
	}
}

// Load fills the empty inventory with the stored items the character
// carries and equips the worn ones in slot order with the Mobius rules,
// telling the listeners (Mobius Inventory.restore). Hero items of a player
// who is neither a GM nor a hero come off. Only real changes end up in the
// save diff, and no updates are recorded. It returns the items it skipped
// for want of a template; they stay stored as they are.
func (inv *Inventory) Load(items []storage.Item, w Wearer) (skipped []storage.Item) {
	return inv.load(items, w, storage.Item.Carried)
}

// Paperdoll returns the item worn in slot, or nil.
func (inv *Inventory) Paperdoll(slot storage.PaperdollSlot) *Item {
	if !slot.Equipped() {
		return nil
	}
	return inv.paperdoll[slot]
}

// WearedMask returns the item type masks of the worn items or'ed together
// (Mobius getWearedMask).
func (inv *Inventory) WearedMask() int { return inv.wearedMask }

// Weight returns the load: weight × count of every item, capped at the
// int range (Mobius refreshWeight).
func (inv *Inventory) Weight() int64 {
	var w int64
	for _, it := range inv.items {
		w += int64(it.tmpl.Weight) * it.count
		if w >= math.MaxInt32 {
			return math.MaxInt32
		}
	}
	return w
}

// ValidateWeight reports whether weight more fits under the owner's
// maximum load (Mobius PlayerInventory.validateWeight; the GM diet mode is
// not there yet).
func (inv *Inventory) ValidateWeight(weight int64) bool {
	return inv.Weight()+weight <= int64(inv.owner.WeightLimit())
}

// ValidateCapacityFor reports whether the item fits: it takes a slot unless
// it stacks onto one the inventory holds or takes effect at once (Mobius
// PlayerInventory.validateCapacity(Item)).
func (inv *Inventory) ValidateCapacityFor(it *Item) bool {
	var slots int64
	if !it.Stackable() || (inv.Count(it.ItemID(), -1, true) <= 0 && !it.tmpl.ExImmediateEffect) {
		slots = 1
	}
	return inv.ValidateCapacity(slots)
}

// ValidateCapacityByItemID reports whether one item of the template fits:
// it takes a slot unless it stacks onto one the inventory holds (Mobius
// PlayerInventory.validateCapacityByItemId(int)).
func (inv *Inventory) ValidateCapacityByItemID(itemID int32) bool {
	var slots int64
	if it := inv.ByItemID(itemID); it == nil || !it.Stackable() {
		slots = 1
	}
	return inv.ValidateCapacity(slots)
}

// Drop takes count of the item with the object ID out of the inventory to
// drop it (Mobius Inventory.dropItem(objectId, count)): from a bigger
// stack, a new item with a new object ID and the count, which is in no
// container; otherwise the item itself, taken off and removed. The
// returned item is at LocVoid.
func (inv *Inventory) Drop(objectID int32, count int64) (*Item, error) {
	it := inv.ByObjectID(objectID)
	if it == nil {
		return nil, ErrNotFound
	}
	if count <= 0 {
		return nil, ErrInvalidCount
	}
	if it.count > count {
		ids, err := inv.newIDs(1)
		if err != nil {
			return nil, err
		}
		it.changeCount(-count, math.MaxInt64)
		it.lastChange = Modified
		inv.touch(it, Modified)
		return NewItem(ids[0], it.tmpl, count), nil
	}
	return inv.DropItem(it)
}

// DropItem takes the whole item out of the inventory to drop it (Mobius
// Inventory.dropItem(Item)): it comes off, leaves the character and is at
// LocVoid.
func (inv *Inventory) DropItem(it *Item) (*Item, error) {
	if it == nil || !inv.remove(it) {
		return nil, ErrNotFound
	}
	it.loc, it.slot, it.lastChange = LocVoid, storage.PaperdollNone, Removed
	inv.tracker.leave(it)
	inv.record(it, Removed)
	return it, nil
}

// FormalWear reports whether the chest slot holds formal wear (Mobius
// Player.isWearingFormalWear).
func (inv *Inventory) FormalWear() bool {
	chest := inv.paperdoll[storage.PaperdollChest]
	return chest != nil && chest.tmpl.Part() == data.BodyPartAlldress
}

// FindArrowForBow returns the first arrows of the bow's grade, or nil
// (Mobius findArrowForBow).
func (inv *Inventory) FindArrowForBow(bow *data.Item) *Item {
	if bow == nil {
		return nil
	}
	for _, it := range inv.items {
		if isEtc(it.tmpl, data.EtcArrow) && it.tmpl.Grade == bow.Grade {
			return it
		}
	}
	return nil
}

// SetPaperdoll puts it (nil to empty) into slot and returns what was there
// (Mobius setPaperdollItem): the old item goes back to the inventory, the
// new one is worn in slot, each is marked modified, and the listeners are
// told, the old item first. it must be in this inventory. Equip and the
// Unequip methods apply the Mobius rules on top of it.
func (inv *Inventory) SetPaperdoll(slot storage.PaperdollSlot, it *Item) *Item {
	if !slot.Equipped() || (it != nil && it.c != &inv.Container) {
		return nil
	}
	return inv.setPaperdoll(slot, it)
}

func (inv *Inventory) setPaperdoll(slot storage.PaperdollSlot, it *Item) *Item {
	old := inv.paperdoll[slot]
	if old == it {
		return old
	}
	if old != nil {
		inv.paperdoll[slot] = nil
		old.loc, old.slot, old.lastChange = inv.base, storage.PaperdollNone, Modified
		inv.wearedMask = 0
		for _, pi := range inv.paperdoll {
			if pi != nil {
				inv.wearedMask |= pi.tmpl.Mask()
			}
		}
		for _, l := range slices.Clone(inv.listeners) {
			l.Unequipped(inv, slot, old)
		}
		inv.touch(old, Modified)
	}
	if it != nil {
		inv.paperdoll[slot] = it
		it.loc, it.slot, it.lastChange = storage.LocPaperdoll, slot, Modified
		inv.wearedMask |= it.tmpl.Mask()
		for _, l := range slices.Clone(inv.listeners) {
			l.Equipped(inv, slot, it)
		}
		inv.touch(it, Modified)
	}
	return old
}

// recorder collects the items a paperdoll change touched (Mobius
// ChangeRecorder).
type recorder struct{ items []*Item }

func (r *recorder) add(it *Item) {
	if !slices.Contains(r.items, it) {
		r.items = append(r.items, it)
	}
}

func (r *recorder) Equipped(_ *Inventory, _ storage.PaperdollSlot, it *Item)   { r.add(it) }
func (r *recorder) Unequipped(_ *Inventory, _ storage.PaperdollSlot, it *Item) { r.add(it) }

// recordChanges runs f with a recorder and returns the items it touched.
func (inv *Inventory) recordChanges(f func()) []*Item {
	r := &recorder{}
	inv.listeners = append(inv.listeners, r)
	defer inv.RemoveListener(r)
	f()
	return r.items
}

// Equip puts the item on with the Mobius rules and returns every item that
// went on or came off, each once (Mobius equipItemAndRecord). Nothing
// happens in store mode, for a hero item of a player who is neither a GM
// nor a hero, or for an item not in this inventory or already worn (Mobius
// would put a worn ring into the other slot too; its callers never ask).
// Whether the item went on shows in it.Equipped().
func (inv *Inventory) Equip(it *Item, w Wearer) []*Item {
	if it == nil || it.c != &inv.Container || it.Equipped() {
		return nil
	}
	return inv.recordChanges(func() { inv.equipItem(it, w) })
}

// equipItem is Mobius Inventory.equipItem.
func (inv *Inventory) equipItem(it *Item, w Wearer) {
	part := it.tmpl.Part()
	if w.StoreMode || (!w.GM && !w.Hero && it.tmpl.Hero) {
		return
	}
	if inv.FormalWear() {
		switch part {
		case data.BodyPartLRHand, data.BodyPartLHand, data.BodyPartRHand:
			inv.UnequipBodyPart(data.BodyPartAlldress)
		case data.BodyPartLegs, data.BodyPartFeet, data.BodyPartGloves, data.BodyPartHead:
			return
		}
	}
	pd := &inv.paperdoll
	set := inv.setPaperdoll
	switch part {
	case data.BodyPartLRHand:
		set(storage.PaperdollLHand, nil)
		set(storage.PaperdollRHand, it)
	case data.BodyPartLHand:
		rh := pd[storage.PaperdollRHand]
		if rh != nil && rh.tmpl.Part() == data.BodyPartLRHand {
			ammo := (isWeapon(rh.tmpl, data.WeaponBow) && isEtc(it.tmpl, data.EtcArrow)) ||
				(isWeapon(rh.tmpl, data.WeaponFishingrod) && isEtc(it.tmpl, data.EtcLure))
			if !ammo {
				set(storage.PaperdollRHand, nil)
			}
		}
		set(storage.PaperdollLHand, it)
	case data.BodyPartRHand:
		set(storage.PaperdollRHand, it)
	case data.BodyPartLEar, data.BodyPartREar, data.BodyPartLREar:
		switch {
		case pd[storage.PaperdollLEar] == nil:
			set(storage.PaperdollLEar, it)
		case pd[storage.PaperdollREar] == nil:
			set(storage.PaperdollREar, it)
		default:
			set(storage.PaperdollLEar, it)
		}
	case data.BodyPartLFinger, data.BodyPartRFinger, data.BodyPartLRFinger:
		switch {
		case pd[storage.PaperdollLFinger] == nil:
			set(storage.PaperdollLFinger, it)
		case pd[storage.PaperdollRFinger] == nil:
			set(storage.PaperdollRFinger, it)
		default:
			set(storage.PaperdollLFinger, it)
		}
	case data.BodyPartNeck:
		set(storage.PaperdollNeck, it)
	case data.BodyPartFullArmor:
		set(storage.PaperdollLegs, nil)
		set(storage.PaperdollChest, it)
	case data.BodyPartChest:
		set(storage.PaperdollChest, it)
	case data.BodyPartLegs:
		if chest := pd[storage.PaperdollChest]; chest != nil && chest.tmpl.Part() == data.BodyPartFullArmor {
			set(storage.PaperdollChest, nil)
		}
		set(storage.PaperdollLegs, it)
	case data.BodyPartFeet:
		set(storage.PaperdollFeet, it)
	case data.BodyPartGloves:
		set(storage.PaperdollGloves, it)
	case data.BodyPartHead:
		set(storage.PaperdollHead, it)
	case data.BodyPartHair:
		if hair := pd[storage.PaperdollHair]; hair != nil && hair.tmpl.Part() == data.BodyPartHairall {
			set(storage.PaperdollHair2, nil)
		} else {
			set(storage.PaperdollHair, nil)
		}
		set(storage.PaperdollHair, it)
	case data.BodyPartHair2:
		if hair := pd[storage.PaperdollHair]; hair != nil && hair.tmpl.Part() == data.BodyPartHairall {
			set(storage.PaperdollHair, nil)
		} else {
			set(storage.PaperdollHair2, nil)
		}
		set(storage.PaperdollHair2, it)
	case data.BodyPartHairall:
		set(storage.PaperdollHair2, nil)
		set(storage.PaperdollHair, it)
	case data.BodyPartUnderwear:
		set(storage.PaperdollUnder, it)
	case data.BodyPartBack:
		set(storage.PaperdollCloak, it)
	case data.BodyPartAlldress:
		set(storage.PaperdollLegs, nil)
		set(storage.PaperdollLHand, nil)
		set(storage.PaperdollRHand, nil)
		set(storage.PaperdollHead, nil)
		set(storage.PaperdollFeet, nil)
		set(storage.PaperdollGloves, nil)
		set(storage.PaperdollChest, it)
	}
	// Other body parts (none, the pet slots) cannot be worn: Mobius logs
	// them and does nothing.
}

// UnequipSlot empties the paperdoll slot and returns the items that came
// off, each once (Mobius unEquipItemInSlotAndRecord).
func (inv *Inventory) UnequipSlot(slot storage.PaperdollSlot) []*Item {
	if !slot.Equipped() {
		return nil
	}
	return inv.recordChanges(func() { inv.setPaperdoll(slot, nil) })
}

// UnequipBodyPart takes off what is worn for the body part and returns the
// items that came off, each once (Mobius unEquipItemInBodySlotAndRecord):
// two-handed weapons from the right hand, full armor and formal wear from
// the chest, hairall from hair (and hair itself first).
func (inv *Inventory) UnequipBodyPart(part data.BodyPart) []*Item {
	return inv.recordChanges(func() {
		if part == data.BodyPartHairall {
			inv.setPaperdoll(storage.PaperdollHair, nil)
		}
		if slot := PaperdollIndex(part); slot.Equipped() {
			inv.setPaperdoll(slot, nil)
		}
	})
}

// UnequipItem takes off a worn item of this inventory by its body part and
// returns the items that came off (Mobius useEquippableItem for a worn
// item: BodyPart.fromItem, then unEquipItemInBodySlotAndRecord).
func (inv *Inventory) UnequipItem(it *Item) []*Item {
	if it == nil || it.c != &inv.Container || !it.Equipped() {
		return nil
	}
	return inv.UnequipBodyPart(BodyPartOf(it))
}

// PaperdollIndex returns the paperdoll slot of a body part (Mobius
// BodyPart.getPaperdollIndex): lrhand is the right hand, fullarmor and
// alldress the chest, hairall the hair slot. The ear and finger pairs have
// none (PaperdollNone), and neither has a body part without a slot.
func PaperdollIndex(part data.BodyPart) storage.PaperdollSlot {
	switch part {
	case data.BodyPartLRHand:
		return storage.PaperdollRHand
	case data.BodyPartFullArmor, data.BodyPartAlldress:
		return storage.PaperdollChest
	case data.BodyPartHairall:
		return storage.PaperdollHair
	}
	slot := storage.PaperdollSlot(part.PaperdollSlot()) //nolint:gosec // -1 to 17
	if !slot.Equipped() {
		return storage.PaperdollNone
	}
	return slot
}

// BodyPartOf returns the body part a worn item takes (Mobius
// BodyPart.fromItem): the template's for the chest slot, otherwise the
// body part of the slot. Items not worn, and the face slot, have none.
func BodyPartOf(it *Item) data.BodyPart {
	if it.slot == storage.PaperdollChest {
		return it.tmpl.Part()
	}
	if !it.slot.Equipped() {
		return data.BodyPartNone
	}
	for p := data.BodyPart(0); p < data.NumBodyParts; p++ {
		if p.PaperdollSlot() == int(it.slot) {
			return p
		}
	}
	return data.BodyPartNone
}
