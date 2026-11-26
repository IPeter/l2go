package storage

import "context"

// Cursed weapon state (Plan 14 M11m, decision X4): a Mobius cursed_weapons
// row for each weapon a player holds. A weapon on the ground or out of the
// world has no row. Rows are written with a Batch (BatchWriter), in the same
// transaction as the holder's character and items, so a pickup, a kill, a
// death or the end of a weapon is stored whole or not at all.

// CursedWeapon is a held cursed weapon (a Mobius cursed_weapons row).
type CursedWeapon struct {
	ItemID int32 // the weapon's item ID, the key
	CharID int32 // the holder
	// PlayerKarma and PlayerPKKills are the holder's karma and PK count from
	// before the weapon, given back when it goes.
	PlayerKarma   int32
	PlayerPKKills int32
	NbKills       int32 // the holder's kills with the weapon
	EndTime       int64 // when the weapon ends, Unix milliseconds
	// ItemObjectID is the object ID of the holder's copy of the weapon (not
	// in Mobius, which deletes by owner and item ID); 0 when unknown.
	ItemObjectID int32
}

// OwnedItem is an item and its owner, as a lookup by item ID finds it.
type OwnedItem struct {
	Owner int32
	Item  Item
}

// CursedWeaponStore reads the cursed weapon state. Both stores implement it
// on their character store, next to BatchWriter, which writes it
// (Batch.CursedWeapons and Batch.DeletedCursedWeapons).
type CursedWeaponStore interface {
	// LoadCursedWeapons returns the stored rows sorted by item ID, nil for
	// none.
	LoadCursedWeapons(ctx context.Context) ([]CursedWeapon, error)
	// ItemsByTemplate returns every stored item with one of the item IDs,
	// whoever owns it and wherever it is kept, sorted by object ID, nil for
	// none (Mobius CursedWeaponsManager.controlPlayers: SELECT owner_id
	// FROM items WHERE item_id=?).
	ItemsByTemplate(ctx context.Context, itemIDs []int32) ([]OwnedItem, error)
}
