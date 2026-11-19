package memstore_test

import (
	"context"
	"errors"
	"testing"
	"time"

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
	if err := s.Update(ctx, &storage.Character{ObjectID: 1}); !errors.Is(err, context.Canceled) {
		t.Errorf("Update error = %v, want context.Canceled", err)
	}
	if err := s.Delete(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Errorf("Delete error = %v, want context.Canceled", err)
	}
	if err := s.SetDeleteTime(ctx, 1, time.Now()); !errors.Is(err, context.Canceled) {
		t.Errorf("SetDeleteTime error = %v, want context.Canceled", err)
	}
}

func TestCharacterDeleteExpired(t *testing.T) {
	now := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		at       time.Time
		deleting bool
		expired  bool
	}{
		{"not deleting", time.Time{}, false, false},
		{"in the future", now.Add(time.Millisecond), true, false},
		{"exactly now", now, true, false}, // Mobius: now > deletetime
		{"just passed", now.Add(-time.Millisecond), true, true},
		{"in the past", now.Add(-time.Hour), true, true},
	}
	for _, tt := range tests {
		c := storage.Character{DeleteTime: tt.at}
		if got := c.Deleting(); got != tt.deleting {
			t.Errorf("%s: Deleting() = %v, want %v", tt.name, got, tt.deleting)
		}
		if got := c.DeleteExpired(now); got != tt.expired {
			t.Errorf("%s: DeleteExpired() = %v, want %v", tt.name, got, tt.expired)
		}
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
