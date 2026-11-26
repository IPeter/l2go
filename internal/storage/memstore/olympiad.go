package memstore

import (
	"cmp"
	"context"
	"slices"
	"sync"

	"github.com/IPeter/l2go/internal/storage"
)

// Olympiad is an in-memory storage.OlympiadStore. The zero value is ready
// to use and it is safe for concurrent use.
type Olympiad struct {
	mu     sync.Mutex
	state  storage.Olympiad
	fights []storage.OlympiadFight
	diary  []storage.HeroDiaryEntry

	// failNext, if set, fails the next write before it changes anything.
	failNext error
}

var _ storage.OlympiadStore = (*Olympiad)(nil)

// NewOlympiad returns an empty store.
func NewOlympiad() *Olympiad { return &Olympiad{} }

// FailNext makes the next write fail with err before it changes anything.
// For tests of the callers.
func (s *Olympiad) FailNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

func (s *Olympiad) takeFault() error {
	err := s.failNext
	s.failNext = nil
	return err
}

// cloneOlympiad copies a state, its lists included.
func cloneOlympiad(o storage.Olympiad) storage.Olympiad {
	o.Nobles = slices.Clone(o.Nobles)
	o.NoblesEOM = slices.Clone(o.NoblesEOM)
	o.Heroes = slices.Clone(o.Heroes)
	o.Unclaimed = slices.Clone(o.Unclaimed)
	return o
}

// LoadOlympiad implements storage.OlympiadStore.
func (s *Olympiad) LoadOlympiad(ctx context.Context) (storage.Olympiad, error) {
	if err := ctx.Err(); err != nil {
		return storage.Olympiad{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneOlympiad(s.state), nil
}

// SaveOlympiad implements storage.OlympiadStore.
func (s *Olympiad) SaveOlympiad(ctx context.Context, o storage.Olympiad) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.takeFault(); err != nil {
		return err
	}
	s.state = cloneOlympiad(o)
	return nil
}

// AddOlympiadFight implements storage.OlympiadStore.
func (s *Olympiad) AddOlympiadFight(ctx context.Context, f storage.OlympiadFight) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.takeFault(); err != nil {
		return err
	}
	s.fights = append(s.fights, f)
	return nil
}

// OlympiadFights implements storage.OlympiadStore.
func (s *Olympiad) OlympiadFights(ctx context.Context, charID int32, before int64) ([]storage.OlympiadFight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []storage.OlympiadFight
	for _, f := range s.fights {
		if (f.CharOneID == charID || f.CharTwoID == charID) && f.Start < before {
			out = append(out, f)
		}
	}
	slices.SortStableFunc(out, func(a, b storage.OlympiadFight) int { return cmp.Compare(a.Start, b.Start) })
	return out, nil
}

// AddHeroDiary implements storage.OlympiadStore.
func (s *Olympiad) AddHeroDiary(ctx context.Context, e storage.HeroDiaryEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.takeFault(); err != nil {
		return err
	}
	s.diary = append(s.diary, e)
	return nil
}

// HeroDiary implements storage.OlympiadStore.
func (s *Olympiad) HeroDiary(ctx context.Context, charID int32) ([]storage.HeroDiaryEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []storage.HeroDiaryEntry
	for _, e := range s.diary {
		if e.CharID == charID {
			out = append(out, e)
		}
	}
	slices.SortStableFunc(out, func(a, b storage.HeroDiaryEntry) int { return cmp.Compare(a.Time, b.Time) })
	return out, nil
}
