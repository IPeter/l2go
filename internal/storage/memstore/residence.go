package memstore

import (
	"context"
	"maps"
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

var _ storage.CastleStore = (*Characters)(nil)

// LoadCastles implements storage.CastleStore.
func (s *Characters) LoadCastles(ctx context.Context) ([]storage.Castle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.castles) == 0 {
		return nil, nil
	}
	out := make([]storage.Castle, 0, len(s.castles))
	for _, id := range slices.Sorted(maps.Keys(s.castles)) {
		out = append(out, cloneCastle(s.castles[id]))
	}
	return out, nil
}

// cloneCastle copies a castle's lists, so the store never shares them.
func cloneCastle(c storage.Castle) storage.Castle {
	c.Functions = slices.Clone(c.Functions)
	c.DoorUpgrades = slices.Clone(c.DoorUpgrades)
	c.TrapUpgrades = slices.Clone(c.TrapUpgrades)
	c.SiegeClans = slices.Clone(c.SiegeClans)
	c.HiredGuards = slices.Clone(c.HiredGuards)
	return c
}

// applyCastles applies the castle part of a batch. The caller holds s.mu.
func (s *Characters) applyCastles(b *storage.Batch) {
	if len(b.Castles) > 0 && s.castles == nil {
		s.castles = make(map[int32]storage.Castle)
	}
	for _, c := range b.Castles {
		s.castles[c.ID] = cloneCastle(c)
	}
}
