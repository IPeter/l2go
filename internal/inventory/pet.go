package inventory

import "github.com/IPeter/l2go/internal/storage"

// NewPetInventory returns an empty pet inventory (Mobius PetInventory):
// items at storage.LocPet, worn ones at storage.LocPetEquip. limit returns
// its slot count (Mobius MaximumSlotsForPet, 12). Share the owner's
// Tracker: a pet's items stay with the player as their owner
// (PetInventory.getOwnerId).
func NewPetInventory(limit func() int, opts Options) *Container {
	c := newContainer(storage.LocPet, opts)
	c.capacity = limit
	return &c
}

// LoadPet fills the empty pet inventory with the stored items at LocPet and
// LocPetEquip, keeping the worn ones worn, and returns those it skipped for
// want of a template.
func (c *Container) LoadPet(items []storage.Item) (skipped []storage.Item) {
	return c.load(items, Wearer{}, func(si storage.Item) bool {
		return si.Loc == storage.LocPet || si.Loc == storage.LocPetEquip
	})
}

// PetEquipped reports a pet's worn item (Mobius ItemLocation.PET_EQUIP).
func (it *Item) PetEquipped() bool { return it.loc == storage.LocPetEquip }

// SetPetEquipped puts an item of the pet inventory on the pet (PET_EQUIP)
// or takes it off (PET), recording the change for the save and the
// updates. Items of other containers are left alone.
func (c *Container) SetPetEquipped(it *Item, worn bool) {
	if it == nil || it.c != c || c.base != storage.LocPet || it.PetEquipped() == worn {
		return
	}
	it.loc = storage.LocPet
	if worn {
		it.loc = storage.LocPetEquip
	}
	it.lastChange = Modified
	c.touch(it, Modified)
}
