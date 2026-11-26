package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestSevenSigns(t *testing.T) {
	storagetest.TestSevenSignsStore(t, func(*testing.T) (storage.SevenSignsStore, func(error)) {
		s := NewSevenSigns()
		return s, s.FailNext
	})
}

// TestSevenSignsZeroValue checks that a zero SevenSigns works and that a
// load returns a copy.
func TestSevenSignsZeroValue(t *testing.T) {
	var s SevenSigns
	if err := s.SaveSevenSigns(t.Context(), storage.SevenSignsSave{Status: storage.SevenSignsStatus{Cycle: 1, AccumulatedBonus: []int32{1, 2, 3, 4, 5}},
		Players: []storage.SevenSignsPlayer{{CharID: 1, Cabal: "dawn"}}}); err != nil {
		t.Fatalf("SaveSevenSigns: %v", err)
	}
	got, err := s.LoadSevenSigns(t.Context())
	if err != nil || !got.HasStatus || got.Status.Cycle != 1 || len(got.Players) != 1 {
		t.Fatalf("LoadSevenSigns = %+v, %v", got, err)
	}
	got.Status.AccumulatedBonus[0] = 9
	if again, _ := s.LoadSevenSigns(t.Context()); again.Status.AccumulatedBonus[0] != 1 {
		t.Error("a loaded state shares its bonuses with the store")
	}
}
