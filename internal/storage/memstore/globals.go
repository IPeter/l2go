package memstore

import (
	"context"
	"maps"
	"sync"

	"github.com/IPeter/l2go/internal/storage"
)

// GlobalVariables is an in-memory storage.GlobalVariableStore. The zero
// value is ready to use and it is safe for concurrent use.
type GlobalVariables struct {
	mu   sync.Mutex
	vars map[string]string

	// failNext, if set, fails the next SaveGlobalVariables before it changes
	// anything. Tests in this package use it to run the shared failed-write
	// tests.
	failNext error
}

var _ storage.GlobalVariableStore = (*GlobalVariables)(nil)

// NewGlobalVariables returns an empty store.
func NewGlobalVariables() *GlobalVariables {
	return &GlobalVariables{}
}

// LoadGlobalVariables implements storage.GlobalVariableStore.
func (s *GlobalVariables) LoadGlobalVariables(ctx context.Context) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneVars(s.vars), nil
}

// SaveGlobalVariables implements storage.GlobalVariableStore.
func (s *GlobalVariables) SaveGlobalVariables(ctx context.Context, changes storage.VarChanges) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.failNext; err != nil {
		s.failNext = nil
		return err
	}
	if s.vars == nil {
		s.vars = make(map[string]string)
	}
	for _, name := range changes.Deleted {
		delete(s.vars, name)
	}
	maps.Copy(s.vars, changes.Set)
	return nil
}
