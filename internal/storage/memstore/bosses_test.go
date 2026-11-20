package memstore

import (
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestBosses(t *testing.T) {
	storagetest.TestBossStore(t, func(*testing.T) (storage.BossStore, func(error)) {
		s := NewBosses()
		return s, s.FailNext
	})
}

// TestBossesZeroValue checks that a zero Bosses works.
func TestBossesZeroValue(t *testing.T) {
	var s Bosses
	if err := s.SaveRaidPoints(t.Context(), []storage.RaidPoints{{CharID: 1, BossID: 2, Points: 3}}); err != nil {
		t.Fatalf("SaveRaidPoints: %v", err)
	}
	if got, err := s.LoadRaidPoints(t.Context()); err != nil || len(got) != 1 {
		t.Errorf("LoadRaidPoints = %+v, %v", got, err)
	}
}
