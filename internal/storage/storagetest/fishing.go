package storagetest

import (
	"errors"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestFishingStore runs the FishingStore conformance tests. newStore must
// return an empty store for each call, and a function that makes the
// store's next write fail with the given error after it did all its work,
// just before it commits.
func TestFishingStore(t *testing.T, newStore func(t *testing.T) (s storage.FishingStore, failNext func(error))) {
	load := func(t *testing.T, s storage.FishingStore) storage.FishingChampionship {
		t.Helper()
		got, err := s.LoadFishingChampionship(t.Context())
		if err != nil {
			t.Fatalf("LoadFishingChampionship: %v", err)
		}
		return got
	}
	equal := func(a, b storage.FishingChampionship) bool {
		return a.End == b.End && slices.Equal(a.Fishers, b.Fishers)
	}
	week := storage.FishingChampionship{
		End: 1796000400123,
		Fishers: []storage.ChampionshipFisher{
			{Name: "Bob", Length: 85.25, Rewarded: 2},
			{Name: "Ann", Length: 70, Rewarded: 1},
			{Name: "Zed", Length: 61.999, Rewarded: 0},
			{Name: "Cid", Length: 88.5, Rewarded: 0},
		},
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStore(t)
		if got := load(t, s); got.End != 0 || len(got.Fishers) != 0 {
			t.Errorf("LoadFishingChampionship = %+v, want the zero value", got)
		}
	})

	t.Run("SaveReplaces", func(t *testing.T) {
		s, _ := newStore(t)
		if err := s.SaveFishingChampionship(t.Context(), week); err != nil {
			t.Fatalf("SaveFishingChampionship: %v", err)
		}
		if got := load(t, s); !equal(got, week) {
			t.Errorf("LoadFishingChampionship = %+v, want %+v (order kept)", got, week)
		}
		next := storage.FishingChampionship{End: week.End + 7*24*3600*1000, Fishers: week.Fishers[2:3]}
		if err := s.SaveFishingChampionship(t.Context(), next); err != nil {
			t.Fatalf("SaveFishingChampionship: %v", err)
		}
		if got := load(t, s); !equal(got, next) {
			t.Errorf("after a second save = %+v, want %+v", got, next)
		}
		if err := s.SaveFishingChampionship(t.Context(), storage.FishingChampionship{End: 5}); err != nil {
			t.Fatalf("SaveFishingChampionship: %v", err)
		}
		if got := load(t, s); got.End != 5 || len(got.Fishers) != 0 {
			t.Errorf("after saving no fishers = %+v", got)
		}
	})

	t.Run("FailedWriteChangesNothing", func(t *testing.T) {
		s, failNext := newStore(t)
		if err := s.SaveFishingChampionship(t.Context(), week); err != nil {
			t.Fatalf("SaveFishingChampionship: %v", err)
		}
		boom := errors.New("boom")
		failNext(boom)
		if err := s.SaveFishingChampionship(t.Context(), storage.FishingChampionship{End: 1}); !errors.Is(err, boom) {
			t.Fatalf("SaveFishingChampionship = %v, want %v", err, boom)
		}
		if got := load(t, s); !equal(got, week) {
			t.Errorf("after a failed write = %+v, want %+v", got, week)
		}
	})
}
