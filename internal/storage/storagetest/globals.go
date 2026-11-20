package storagetest

import (
	"errors"
	"maps"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestGlobalVariableStore runs the GlobalVariableStore conformance tests.
// newStore must return an empty store for each call, and a function that
// makes the store's next SaveGlobalVariables fail with the given error after
// it did all its work, just before it commits.
func TestGlobalVariableStore(t *testing.T, newStore func(t *testing.T) (s storage.GlobalVariableStore, failNext func(error))) {
	load := func(t *testing.T, s storage.GlobalVariableStore) map[string]string {
		t.Helper()
		vars, err := s.LoadGlobalVariables(t.Context())
		if err != nil {
			t.Fatalf("LoadGlobalVariables: %v", err)
		}
		return vars
	}
	save := func(t *testing.T, s storage.GlobalVariableStore, ch storage.VarChanges) {
		t.Helper()
		if err := s.SaveGlobalVariables(t.Context(), ch); err != nil {
			t.Fatalf("SaveGlobalVariables: %v", err)
		}
	}
	initial := map[string]string{
		"DAILY_TASK_RESET":     "1795000000000",
		"Q00620_lastSpawnTime": "0",
		"empty":                "",
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStore(t)
		if vars := load(t, s); vars != nil {
			t.Errorf("LoadGlobalVariables on an empty store = %v, want nil", vars)
		}
		save(t, s, storage.VarChanges{})
		save(t, s, storage.VarChanges{Deleted: []string{"nothing"}})
		if vars := load(t, s); vars != nil {
			t.Errorf("LoadGlobalVariables = %v, want nil", vars)
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		s, _ := newStore(t)
		save(t, s, storage.VarChanges{Set: initial})
		if vars := load(t, s); !maps.Equal(vars, initial) {
			t.Errorf("LoadGlobalVariables = %v, want %v", vars, initial)
		}
	})

	t.Run("SetAndDelete", func(t *testing.T) {
		s, _ := newStore(t)
		save(t, s, storage.VarChanges{Set: initial})
		save(t, s, storage.VarChanges{
			Set:     map[string]string{"Q00620_lastSpawnTime": "1795000123456", "new": "x"},
			Deleted: []string{"empty", "new", "never"}, // Set wins for "new"
		})
		want := map[string]string{"DAILY_TASK_RESET": "1795000000000", "Q00620_lastSpawnTime": "1795000123456", "new": "x"}
		if vars := load(t, s); !maps.Equal(vars, want) {
			t.Errorf("LoadGlobalVariables = %v, want %v", vars, want)
		}
	})

	t.Run("ResultsAreCopies", func(t *testing.T) {
		s, _ := newStore(t)
		set := maps.Clone(initial)
		save(t, s, storage.VarChanges{Set: set})
		set["DAILY_TASK_RESET"] = "changed"
		got := load(t, s)
		got["empty"] = "changed"
		if vars := load(t, s); !maps.Equal(vars, initial) {
			t.Errorf("stored variables changed through a caller's map: %v", vars)
		}
	})

	t.Run("FailedWrite", func(t *testing.T) {
		s, failNext := newStore(t)
		save(t, s, storage.VarChanges{Set: initial})
		failNext(errInjected)
		ch := storage.VarChanges{Set: map[string]string{"DAILY_TASK_RESET": "2"}, Deleted: []string{"empty"}}
		if err := s.SaveGlobalVariables(t.Context(), ch); !errors.Is(err, errInjected) {
			t.Fatalf("SaveGlobalVariables error = %v, want the injected failure", err)
		}
		if vars := load(t, s); !maps.Equal(vars, initial) {
			t.Errorf("after a failed write: %v, want %v", vars, initial)
		}
		save(t, s, ch)
		want := map[string]string{"DAILY_TASK_RESET": "2", "Q00620_lastSpawnTime": "0"}
		if vars := load(t, s); !maps.Equal(vars, want) {
			t.Errorf("after the retry: %v, want %v", vars, want)
		}
	})
}
