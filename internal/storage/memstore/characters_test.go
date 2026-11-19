package memstore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestCharacters(t *testing.T) {
	storagetest.TestCharacterStore(t, func(*testing.T) storage.CharacterStore {
		return memstore.NewCharacters()
	})
}

func TestIDs(t *testing.T) {
	storagetest.TestIDAllocator(t, func(*testing.T) storage.IDAllocator {
		return memstore.NewIDs()
	})
}

func TestCharactersZeroValue(t *testing.T) {
	var s memstore.Characters
	if err := s.Create(t.Context(), &storage.Character{ObjectID: 1, Name: "x"}); err != nil {
		t.Fatalf("Create on zero value: %v", err)
	}
	if _, err := s.Get(t.Context(), 1); err != nil {
		t.Errorf("Get: %v", err)
	}
}

func TestIDsZeroValue(t *testing.T) {
	var a memstore.IDs
	id, err := a.NextID(t.Context())
	if err != nil {
		t.Fatalf("NextID on zero value: %v", err)
	}
	if id != storage.FirstObjectID {
		t.Errorf("NextID = %#x, want %#x", id, storage.FirstObjectID)
	}
}

func TestCharactersCanceledContext(t *testing.T) {
	s := memstore.NewCharacters()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.Create(ctx, &storage.Character{ObjectID: 1, Name: "x"}); !errors.Is(err, context.Canceled) {
		t.Errorf("Create error = %v, want context.Canceled", err)
	}
	if _, err := s.ListByAccount(ctx, "a"); !errors.Is(err, context.Canceled) {
		t.Errorf("ListByAccount error = %v, want context.Canceled", err)
	}
	if _, err := s.CountByAccount(ctx, "a"); !errors.Is(err, context.Canceled) {
		t.Errorf("CountByAccount error = %v, want context.Canceled", err)
	}
	if _, err := s.NameExists(ctx, "x"); !errors.Is(err, context.Canceled) {
		t.Errorf("NameExists error = %v, want context.Canceled", err)
	}
	if _, err := s.Get(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Errorf("Get error = %v, want context.Canceled", err)
	}
}

func TestIDsCanceledContext(t *testing.T) {
	a := memstore.NewIDs()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := a.NextID(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("NextID error = %v, want context.Canceled", err)
	}
	// A canceled call must not use up IDs.
	id, err := a.NextID(t.Context())
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id != storage.FirstObjectID {
		t.Errorf("NextID = %#x, want %#x", id, storage.FirstObjectID)
	}
}
