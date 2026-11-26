package inventory

import "github.com/IPeter/l2go/internal/storage"

// NewClanWarehouse returns an empty clan warehouse (Mobius ClanWarehouse,
// location CLANWH). limit returns its slot count (WarehouseSlotsClan, 200).
// It has a tracker of its own: the clan is the owner of its items, and its
// changes go into a joint save with the clan ID as owner
// (storage.OwnerItemChanges).
func NewClanWarehouse(limit func() int, opts Options) *Container {
	c := newContainer(storage.LocClanWarehouse, opts)
	c.capacity = limit
	return &c
}
