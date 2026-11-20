// Package memstore is an in-memory storage implementation for tests and
// local runs without a database.
package memstore

import (
	"context"
	"strconv"
	"sync"

	"github.com/IPeter/l2go/internal/storage"
)

// Accounts is an in-memory storage.AccountStore. The zero value is ready to
// use and it is safe for concurrent use.
type Accounts struct {
	mu     sync.Mutex
	nextID int
	byName map[string]storage.Account
}

var _ storage.AccountStore = (*Accounts)(nil)

// NewAccounts returns an empty store.
func NewAccounts() *Accounts {
	return &Accounts{}
}

// FindByUsername implements storage.AccountStore.
func (s *Accounts) FindByUsername(ctx context.Context, username string) (storage.Account, error) {
	if err := ctx.Err(); err != nil {
		return storage.Account{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.byName[username]
	if !ok {
		return storage.Account{}, storage.ErrNotFound
	}
	return acc, nil
}

// Create implements storage.AccountStore.
func (s *Accounts) Create(ctx context.Context, acc storage.Account) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byName[acc.Username]; ok {
		return storage.ErrExists
	}
	if s.byName == nil {
		s.byName = make(map[string]storage.Account)
	}
	s.nextID++
	acc.ID = strconv.Itoa(s.nextID)
	if acc.LastServer == 0 {
		acc.LastServer = storage.DefaultLastServer
	}
	s.byName[acc.Username] = acc
	return nil
}

// SetLastServer implements storage.AccountStore.
func (s *Accounts) SetLastServer(ctx context.Context, username string, serverID int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.byName[username]
	if !ok {
		return storage.ErrNotFound
	}
	acc.LastServer = serverID
	s.byName[username] = acc
	return nil
}

// SetAccessLevel implements storage.AccountStore.
func (s *Accounts) SetAccessLevel(ctx context.Context, username string, level storage.AccessLevel) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.byName[username]
	if !ok {
		return storage.ErrNotFound
	}
	acc.AccessLevel = level
	s.byName[username] = acc
	return nil
}
