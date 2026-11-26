package game

import (
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

// mergeCursedWeapons adds the cursed weapon part of b, written after a, to
// a (mergeBatch): a stored row replaces the stored or deleted one of its
// weapon, a delete drops the stored one.
func mergeCursedWeapons(a, b *storage.Batch) {
	for _, id := range b.DeletedCursedWeapons {
		a.CursedWeapons = slices.DeleteFunc(a.CursedWeapons, func(w storage.CursedWeapon) bool { return w.ItemID == id })
		if !slices.Contains(a.DeletedCursedWeapons, id) {
			a.DeletedCursedWeapons = append(a.DeletedCursedWeapons, id)
		}
	}
	for _, w := range b.CursedWeapons {
		if i := slices.IndexFunc(a.CursedWeapons, func(x storage.CursedWeapon) bool { return x.ItemID == w.ItemID }); i >= 0 {
			a.CursedWeapons[i] = w
		} else {
			a.CursedWeapons = append(a.CursedWeapons, w)
		}
	}
}
