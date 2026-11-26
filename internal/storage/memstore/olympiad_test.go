package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestOlympiad(t *testing.T) {
	storagetest.TestOlympiadStore(t, func(*testing.T) (storage.OlympiadStore, func(error)) {
		s := NewOlympiad()
		return s, s.FailNext
	})
}

// TestOlympiadZeroValue checks that a zero Olympiad works and that a load
// returns a copy.
func TestOlympiadZeroValue(t *testing.T) {
	var s Olympiad
	if err := s.SaveOlympiad(t.Context(), storage.Olympiad{Cycle: 1, Nobles: []storage.OlympiadNoble{{CharID: 1}}}); err != nil {
		t.Fatalf("SaveOlympiad: %v", err)
	}
	got, err := s.LoadOlympiad(t.Context())
	if err != nil || got.Cycle != 1 {
		t.Fatalf("LoadOlympiad = %+v, %v", got, err)
	}
	got.Nobles[0].CharID = 9
	if again, _ := s.LoadOlympiad(t.Context()); again.Nobles[0].CharID != 1 {
		t.Error("a loaded state shares its lists with the store")
	}
}
