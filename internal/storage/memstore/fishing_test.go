package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestFishing(t *testing.T) {
	storagetest.TestFishingStore(t, func(*testing.T) (storage.FishingStore, func(error)) {
		s := NewFishing()
		return s, s.FailNext
	})
}

// TestFishingZeroValue checks that a zero Fishing works.
func TestFishingZeroValue(t *testing.T) {
	var s Fishing
	if err := s.SaveFishingChampionship(t.Context(), storage.FishingChampionship{End: 1}); err != nil {
		t.Fatalf("SaveFishingChampionship: %v", err)
	}
	if got, err := s.LoadFishingChampionship(t.Context()); err != nil || got.End != 1 {
		t.Errorf("LoadFishingChampionship = %+v, %v", got, err)
	}
}
