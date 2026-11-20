// Package storagetest holds conformance tests that every storage
// implementation must pass.
package storagetest

import (
	"errors"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestAccountStore runs the AccountStore conformance tests. newStore must
// return an empty store for each call.
func TestAccountStore(t *testing.T, newStore func(t *testing.T) storage.AccountStore) {
	t.Run("CreateThenFind", func(t *testing.T) {
		s := newStore(t)
		want := storage.Account{Username: "alice", Password: "hash", AccessLevel: storage.AccessLevelAdmin, LastServer: 3}
		if err := s.Create(t.Context(), want); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := s.FindByUsername(t.Context(), "alice")
		if err != nil {
			t.Fatalf("FindByUsername: %v", err)
		}
		if got.ID == "" {
			t.Error("ID is empty, want an ID assigned by the store")
		}
		got.ID = ""
		if got != want {
			t.Errorf("FindByUsername = %+v, want %+v", got, want)
		}
	})

	t.Run("LastServer", func(t *testing.T) {
		s := newStore(t)
		if err := s.Create(t.Context(), storage.Account{Username: "erin"}); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := s.FindByUsername(t.Context(), "erin")
		if err != nil {
			t.Fatalf("FindByUsername: %v", err)
		}
		if got.LastServer != storage.DefaultLastServer {
			t.Errorf("new account LastServer = %d, want %d", got.LastServer, storage.DefaultLastServer)
		}
		if err := s.SetLastServer(t.Context(), "erin", 7); err != nil {
			t.Fatalf("SetLastServer: %v", err)
		}
		got, err = s.FindByUsername(t.Context(), "erin")
		if err != nil {
			t.Fatalf("FindByUsername: %v", err)
		}
		if got.Username != "erin" || got.LastServer != 7 {
			t.Errorf("after SetLastServer: %+v, want erin with LastServer 7", got)
		}
		if err := s.SetLastServer(t.Context(), "nobody", 7); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("SetLastServer(nobody) error = %v, want ErrNotFound", err)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		s := newStore(t)
		_, err := s.FindByUsername(t.Context(), "nobody")
		if !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("FindByUsername error = %v, want ErrNotFound", err)
		}
	})

	t.Run("DuplicateUsername", func(t *testing.T) {
		s := newStore(t)
		acc := storage.Account{Username: "bob", Password: "one"}
		if err := s.Create(t.Context(), acc); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		acc.Password = "two"
		if err := s.Create(t.Context(), acc); !errors.Is(err, storage.ErrExists) {
			t.Fatalf("second Create error = %v, want ErrExists", err)
		}
		got, err := s.FindByUsername(t.Context(), "bob")
		if err != nil {
			t.Fatalf("FindByUsername: %v", err)
		}
		if got.Password != "one" {
			t.Errorf("Password = %q, want the first account kept", got.Password)
		}
	})

	t.Run("CaseSensitive", func(t *testing.T) {
		s := newStore(t)
		if err := s.Create(t.Context(), storage.Account{Username: "Carol"}); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, err := s.FindByUsername(t.Context(), "carol"); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("FindByUsername(carol) error = %v, want ErrNotFound", err)
		}
	})

	t.Run("ConcurrentCreate", func(t *testing.T) {
		s := newStore(t)
		const n = 8
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			created int
		)
		for range n {
			wg.Go(func() {
				err := s.Create(t.Context(), storage.Account{Username: "dave"})
				switch {
				case err == nil:
					mu.Lock()
					created++
					mu.Unlock()
				case !errors.Is(err, storage.ErrExists):
					t.Errorf("Create: %v", err)
				}
			})
		}
		wg.Wait()
		if created != 1 {
			t.Errorf("%d creates succeeded, want exactly 1", created)
		}
	})
}
