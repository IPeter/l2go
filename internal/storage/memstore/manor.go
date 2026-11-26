package memstore

import (
	"context"
	"maps"
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

var _ storage.ManorStore = (*Characters)(nil)

// manorState is the stored manor: its mode and the castles' lists.
type manorState struct {
	state   *storage.ManorState
	castles map[int32]storage.ManorCastle // by castle ID
}

// LoadManor implements storage.ManorStore.
func (s *Characters) LoadManor(ctx context.Context) (storage.ManorState, bool, []storage.ManorCastle, error) {
	if err := ctx.Err(); err != nil {
		return storage.ManorState{}, false, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var st storage.ManorState
	ok := s.manor.state != nil
	if ok {
		st = *s.manor.state
	}
	var out []storage.ManorCastle
	for _, id := range slices.Sorted(maps.Keys(s.manor.castles)) {
		out = append(out, cloneManorCastle(s.manor.castles[id]))
	}
	return st, ok, out, nil
}

// cloneManorCastle copies a castle's lists, so the store never shares them.
func cloneManorCastle(c storage.ManorCastle) storage.ManorCastle {
	c.Production, c.ProductionNext = slices.Clone(c.Production), slices.Clone(c.ProductionNext)
	c.Procure, c.ProcureNext = slices.Clone(c.Procure), slices.Clone(c.ProcureNext)
	return c
}

// applyManor applies the manor part of a batch. The caller holds s.mu.
func (s *Characters) applyManor(b *storage.Batch) {
	if len(b.ManorCastles) > 0 && s.manor.castles == nil {
		s.manor.castles = make(map[int32]storage.ManorCastle)
	}
	for _, c := range b.ManorCastles {
		s.manor.castles[c.CastleID] = cloneManorCastle(c)
	}
	if b.ManorState != nil {
		st := *b.ManorState
		s.manor.state = &st
	}
}
