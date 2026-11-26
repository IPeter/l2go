package memstore

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/IPeter/l2go/internal/storage"
)

// Summons is an in-memory storage.SummonStore. The zero value is ready to
// use and it is safe for concurrent use.
type Summons struct {
	mu     sync.Mutex
	pets   map[int32]storage.Pet
	owners map[int32]storage.OwnerSummons

	// failNext, if set, fails the next write before it changes anything.
	failNext error
}

var _ storage.SummonStore = (*Summons)(nil)

// NewSummons returns an empty store.
func NewSummons() *Summons { return &Summons{} }

// FailNext makes the next write fail with err before it changes anything.
// For tests of the callers.
func (s *Summons) FailNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = err
}

func (s *Summons) write(ctx context.Context, fn func()) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failNext; err != nil {
		s.failNext = nil
		return err
	}
	fn()
	return nil
}

func clonePet(p storage.Pet) storage.Pet {
	p.Effects = slices.Clone(p.Effects)
	if len(p.Effects) == 0 {
		p.Effects = nil
	}
	return p
}

func cloneOwnerSummons(o storage.OwnerSummons) storage.OwnerSummons {
	if o.Servitor != nil {
		sv := *o.Servitor
		o.Servitor = &sv
	}
	if len(o.Effects) == 0 {
		o.Effects = nil
		return o
	}
	effects := make([]storage.ServitorEffects, len(o.Effects))
	for i, e := range o.Effects {
		e.Effects = slices.Clone(e.Effects)
		if len(e.Effects) == 0 {
			e.Effects = nil
		}
		effects[i] = e
	}
	o.Effects = effects
	return o
}

// LoadPet implements storage.SummonStore.
func (s *Summons) LoadPet(ctx context.Context, controlID int32) (storage.Pet, error) {
	if err := ctx.Err(); err != nil {
		return storage.Pet{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pets[controlID]
	if !ok {
		return storage.Pet{}, fmt.Errorf("memstore: pet %d: %w", controlID, storage.ErrNotFound)
	}
	return clonePet(p), nil
}

// SavePet implements storage.SummonStore.
func (s *Summons) SavePet(ctx context.Context, p storage.Pet) error {
	p = clonePet(p)
	return s.write(ctx, func() {
		if s.pets == nil {
			s.pets = make(map[int32]storage.Pet)
		}
		s.pets[p.ControlID] = p
	})
}

// DeletePet implements storage.SummonStore.
func (s *Summons) DeletePet(ctx context.Context, controlID int32) error {
	return s.write(ctx, func() { delete(s.pets, controlID) })
}

// LoadOwnerSummons implements storage.SummonStore.
func (s *Summons) LoadOwnerSummons(ctx context.Context, ownerID int32) (storage.OwnerSummons, error) {
	if err := ctx.Err(); err != nil {
		return storage.OwnerSummons{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.owners[ownerID]
	if !ok {
		return storage.OwnerSummons{OwnerID: ownerID}, nil
	}
	return cloneOwnerSummons(o), nil
}

// SaveOwnerSummons implements storage.SummonStore.
func (s *Summons) SaveOwnerSummons(ctx context.Context, o storage.OwnerSummons) error {
	o = cloneOwnerSummons(o)
	return s.write(ctx, func() {
		if o.Servitor == nil && len(o.Effects) == 0 && o.RestorePet == 0 {
			delete(s.owners, o.OwnerID)
			return
		}
		if s.owners == nil {
			s.owners = make(map[int32]storage.OwnerSummons)
		}
		s.owners[o.OwnerID] = o
	})
}
