package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestFailedWrites(t *testing.T) {
	storagetest.TestFailedWrites(t, func(*testing.T) (storagetest.SaverStore, func(error)) {
		s := NewCharacters()
		return s, func(err error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.failNext = err
		}
	})
}

// TestIDsEndBelowRuntimeRange checks that persistent IDs never reach the
// runtime range of NPCs (decision W3).
func TestIDsEndBelowRuntimeRange(t *testing.T) {
	last := storage.FirstRuntimeObjectID - 1
	a := &IDs{used: int64(last-storage.FirstObjectID) - 1} // two IDs left
	ids, err := a.NextIDs(t.Context(), 2)
	if err != nil {
		t.Fatalf("NextIDs(2) with two IDs left: %v", err)
	}
	if ids[0] != last-1 || ids[1] != last {
		t.Errorf("NextIDs(2) = %#x, want %#x and %#x", ids, last-1, last)
	}
	if ids, err := a.NextID(t.Context()); err == nil {
		t.Errorf("NextID past the range = %#x, want an error", ids)
	}
}
