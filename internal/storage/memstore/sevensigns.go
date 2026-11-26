package memstore

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/IPeter/l2go/internal/storage"
)

// SevenSigns is an in-memory storage.SevenSignsStore. The zero value is
// ready to use and it is safe for concurrent use.
type SevenSigns struct {
	mu        sync.Mutex
	hasStatus bool
	status    storage.SevenSignsStatus
	players   map[int32]storage.SevenSignsPlayer
	festival  []storage.SevenSignsFestivalScore

	// failNext, if set, fails the next write before it changes anything.
	failNext error
}

var _ storage.SevenSignsStore = (*SevenSigns)(nil)

// NewSevenSigns returns an empty store.
func NewSevenSigns() *SevenSigns { return &SevenSigns{} }

// FailNext makes the next write fail with err before it changes anything.
// For tests of the callers.
func (s *SevenSigns) FailNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

func cloneSevenSignsStatus(st storage.SevenSignsStatus) storage.SevenSignsStatus {
	st.AccumulatedBonus = slices.Clone(st.AccumulatedBonus)
	return st
}

// LoadSevenSigns implements storage.SevenSignsStore.
func (s *SevenSigns) LoadSevenSigns(ctx context.Context) (storage.SevenSigns, error) {
	if err := ctx.Err(); err != nil {
		return storage.SevenSigns{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := storage.SevenSigns{HasStatus: s.hasStatus, Status: cloneSevenSignsStatus(s.status)}
	for _, p := range s.players {
		out.Players = append(out.Players, p)
	}
	slices.SortFunc(out.Players, func(a, b storage.SevenSignsPlayer) int { return cmp.Compare(a.CharID, b.CharID) })
	out.Festival = slices.Clone(s.festival)
	slices.SortStableFunc(out.Festival, func(a, b storage.SevenSignsFestivalScore) int {
		return cmp.Or(cmp.Compare(a.Festival, b.Festival), cmp.Compare(a.Cabal, b.Cabal), cmp.Compare(a.Cycle, b.Cycle))
	})
	return out, nil
}

// SaveSevenSigns implements storage.SevenSignsStore.
func (s *SevenSigns) SaveSevenSigns(ctx context.Context, sv storage.SevenSignsSave) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failNext; err != nil {
		s.failNext = nil
		return err
	}
	s.hasStatus, s.status = true, cloneSevenSignsStatus(sv.Status)
	if s.players == nil {
		s.players = make(map[int32]storage.SevenSignsPlayer)
	}
	for _, p := range sv.Players {
		s.players[p.CharID] = p
	}
	if sv.ReplaceFestival {
		s.festival = slices.Clone(sv.Festival)
	}
	return nil
}
