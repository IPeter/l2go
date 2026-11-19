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

func TestCharacterSaver(t *testing.T) {
	storagetest.TestCharacterSaver(t, func(*testing.T) storagetest.SaverStore {
		return memstore.NewCharacters()
	})
}

// TestItemsWithoutLocation checks the callers that only know about the
// paperdoll, as before items had a location: an item without Loc is worn
// or carried by its slot, and comes back with the location filled in.
func TestItemsWithoutLocation(t *testing.T) {
	s := memstore.NewCharacters()
	c := storage.Character{ObjectID: 1, Name: "Old", Items: []storage.Item{
		{ObjectID: 3, ItemID: 57, Count: 100, Slot: storage.PaperdollNone},
		{ObjectID: 2, ItemID: 2369, Count: 1, Slot: storage.PaperdollRHand},
	}}
	if err := s.Create(t.Context(), &c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(t.Context(), 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := []storage.Item{
		{ObjectID: 2, ItemID: 2369, Count: 1, Slot: storage.PaperdollRHand, Loc: storage.LocPaperdoll},
		{ObjectID: 3, ItemID: 57, Count: 100, Slot: storage.PaperdollNone, Loc: storage.LocInventory},
	}
	if len(got.Items) != len(want) || got.Items[0] != want[0] || got.Items[1] != want[1] {
		t.Errorf("Items = %+v, want %+v", got.Items, want)
	}
}

func TestIDs(t *testing.T) {
	storagetest.TestIDAllocator(t, func(*testing.T) storage.IDAllocator {
		return memstore.NewIDs()
	})
}

func TestCharactersZeroValue(t *testing.T) {
	var empty memstore.Characters
	if err := empty.Update(t.Context(), &storage.Character{ObjectID: 1}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Update on zero value: error = %v, want ErrNotFound", err)
	}
	if err := empty.Save(t.Context(), &storage.Character{ObjectID: 1}, storage.ItemChanges{Destroyed: []int32{2}}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Save on zero value: error = %v, want ErrNotFound", err)
	}
	if err := empty.Delete(t.Context(), 1); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Delete on zero value: error = %v, want ErrNotFound", err)
	}
	if list, err := empty.ListByAccount(t.Context(), ""); err != nil || len(list) != 0 {
		t.Errorf("ListByAccount on zero value = %+v, %v, want none", list, err)
	}

	var s memstore.Characters
	c := storage.Character{ObjectID: 1, Name: "x", Items: []storage.Item{{ObjectID: 2, ItemID: 57, Count: 1, Slot: storage.PaperdollNone}}}
	if err := s.Create(t.Context(), &c); err != nil {
		t.Fatalf("Create on zero value: %v", err)
	}
	if got, err := s.Get(t.Context(), 1); err != nil || len(got.Items) != 1 {
		t.Errorf("Get = %+v, %v, want the character with its item", got, err)
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
	if err := s.Save(ctx, &storage.Character{ObjectID: 1}, storage.ItemChanges{}); !errors.Is(err, context.Canceled) {
		t.Errorf("Save error = %v, want context.Canceled", err)
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
