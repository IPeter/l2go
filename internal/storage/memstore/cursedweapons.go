package memstore

import (
	"cmp"
	"context"
	"maps"
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

var _ storage.CursedWeaponStore = (*Characters)(nil)

// LoadCursedWeapons implements storage.CursedWeaponStore.
func (s *Characters) LoadCursedWeapons(ctx context.Context) ([]storage.CursedWeapon, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cursedWeapons) == 0 {
		return nil, nil
	}
	out := make([]storage.CursedWeapon, 0, len(s.cursedWeapons))
	for _, id := range slices.Sorted(maps.Keys(s.cursedWeapons)) {
		out = append(out, s.cursedWeapons[id])
	}
	return out, nil
}

// ItemsByTemplate implements storage.CursedWeaponStore.
func (s *Characters) ItemsByTemplate(ctx context.Context, itemIDs []int32) ([]storage.OwnedItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []storage.OwnedItem
	for _, e := range s.items {
		if slices.Contains(itemIDs, e.item.ItemID) {
			out = append(out, storage.OwnedItem{Owner: e.owner, Item: e.item})
		}
	}
	slices.SortFunc(out, func(a, b storage.OwnedItem) int { return cmp.Compare(a.Item.ObjectID, b.Item.ObjectID) })
	return out, nil
}

// applyCursedWeapons applies the cursed weapon part of a batch. The caller
// holds s.mu.
func (s *Characters) applyCursedWeapons(b *storage.Batch) {
	for _, id := range b.DeletedCursedWeapons {
		delete(s.cursedWeapons, id)
	}
	if len(b.CursedWeapons) > 0 && s.cursedWeapons == nil {
		s.cursedWeapons = make(map[int32]storage.CursedWeapon)
	}
	for _, w := range b.CursedWeapons {
		s.cursedWeapons[w.ItemID] = w
	}
}
