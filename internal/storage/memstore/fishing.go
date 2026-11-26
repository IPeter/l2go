package memstore

import (
	"context"
	"slices"
	"sync"

	"github.com/IPeter/l2go/internal/storage"
)

// Fishing is an in-memory storage.FishingStore. The zero value is ready to
// use and it is safe for concurrent use.
type Fishing struct {
	mu    sync.Mutex
	champ storage.FishingChampionship

	// failNext, if set, fails the next write before it changes anything.
	failNext error
}

var _ storage.FishingStore = (*Fishing)(nil)

// NewFishing returns an empty store.
func NewFishing() *Fishing { return &Fishing{} }

// FailNext makes the next write fail with err before it changes anything.
// For tests of the callers.
func (s *Fishing) FailNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

// LoadFishingChampionship implements storage.FishingStore.
func (s *Fishing) LoadFishingChampionship(ctx context.Context) (storage.FishingChampionship, error) {
	if err := ctx.Err(); err != nil {
		return storage.FishingChampionship{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return storage.FishingChampionship{End: s.champ.End, Fishers: slices.Clone(s.champ.Fishers)}, nil
}

// SaveFishingChampionship implements storage.FishingStore.
func (s *Fishing) SaveFishingChampionship(ctx context.Context, c storage.FishingChampionship) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failNext; err != nil {
		s.failNext = nil
		return err
	}
	s.champ = storage.FishingChampionship{End: c.End, Fishers: slices.Clone(c.Fishers)}
	return nil
}
