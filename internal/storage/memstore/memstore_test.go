package memstore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestAccounts(t *testing.T) {
	storagetest.TestAccountStore(t, func(*testing.T) storage.AccountStore {
		return memstore.NewAccounts()
	})
}

func TestAccountsZeroValue(t *testing.T) {
	var s memstore.Accounts
	if err := s.Create(t.Context(), storage.Account{Username: "x"}); err != nil {
		t.Fatalf("Create on zero value: %v", err)
	}
}

func TestAccountsCanceledContext(t *testing.T) {
	s := memstore.NewAccounts()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.Create(ctx, storage.Account{Username: "x"}); !errors.Is(err, context.Canceled) {
		t.Errorf("Create error = %v, want context.Canceled", err)
	}
	if _, err := s.FindByUsername(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Errorf("FindByUsername error = %v, want context.Canceled", err)
	}
}
