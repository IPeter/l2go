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

func TestSocialStores(t *testing.T) {
	storagetest.TestSocialStores(t, func(*testing.T) (storagetest.SocialStores, func(error)) {
		s := NewCharacters()
		stores := storagetest.SocialStores{Characters: s, Batches: s, Clans: s.Clans(), Crests: s.Crests(), Relations: s.Relations()}
		return stores, func(err error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.failNext = err
		}
	})
}

// TestSocialZeroValue checks the clan, crest and relation views of a zero
// Characters.
func TestSocialZeroValue(t *testing.T) {
	var s Characters
	if list, err := s.Clans().LoadClans(t.Context()); err != nil || list != nil {
		t.Errorf("LoadClans on zero value = %+v, %v", list, err)
	}
	if id, err := s.Crests().NextCrestID(t.Context()); err != nil || id != 1 {
		t.Errorf("NextCrestID on zero value = %d, %v; want 1", id, err)
	}
	if list, err := s.Relations().Relations(t.Context(), 1); err != nil || list != nil {
		t.Errorf("Relations on zero value = %+v, %v", list, err)
	}
	var z Characters
	b := storage.Batch{Relations: storage.RelationChanges{Added: storage.FriendRelations(1, 2)}}
	if err := z.WriteBatch(t.Context(), b); err != nil {
		t.Fatalf("WriteBatch on zero value: %v", err)
	}
}

func TestGlobalVariables(t *testing.T) {
	storagetest.TestGlobalVariableStore(t, func(*testing.T) (storage.GlobalVariableStore, func(error)) {
		s := NewGlobalVariables()
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
