package storagetest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// ManorStores are the stores of one database the manor lives in (Plan 14
// M11k): the manor is read with Manor and written with Batches, together
// with the castles a change pays from, which Castles reads.
type ManorStores struct {
	Batches storage.BatchWriter
	Manor   storage.ManorStore
	Castles storage.CastleStore
}

// TestManorStore runs the manor conformance tests. newStores must return
// empty stores of one database for each call, and a function that makes the
// next write fail with the given error after it did all its work, just
// before it commits.
func TestManorStore(t *testing.T, newStores func(t *testing.T) (ManorStores, func(error))) {
	gludio := storage.ManorCastle{CastleID: 1,
		Production:     []storage.SeedProduction{{SeedID: 5016, Amount: 40, Price: 10, StartAmount: 100}},
		ProductionNext: []storage.SeedProduction{{SeedID: 5016, Amount: 100, Price: 12, StartAmount: 100}, {SeedID: 5650, Amount: 5, Price: 30, StartAmount: 5}},
		Procure:        []storage.CropProcure{{CropID: 5073, Amount: 3, Price: 100, StartAmount: 10, RewardType: 1}},
		ProcureNext:    []storage.CropProcure{{CropID: 5818, Amount: 20, Price: 200, StartAmount: 20, RewardType: 2}},
	}
	dion := storage.ManorCastle{CastleID: 2, Production: []storage.SeedProduction{{SeedID: 5221, Amount: 1, Price: 5, StartAmount: 1}}}
	state := storage.ManorState{Mode: 3, NextChange: 1_795_000_000_000}

	load := func(t *testing.T, s storage.ManorStore) (storage.ManorState, bool, []storage.ManorCastle) {
		t.Helper()
		st, ok, castles, err := s.LoadManor(t.Context())
		if err != nil {
			t.Fatalf("LoadManor: %v", err)
		}
		return st, ok, castles
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStores(t)
		if st, ok, castles := load(t, s.Manor); ok || castles != nil || st != (storage.ManorState{}) {
			t.Errorf("LoadManor = %+v, %v, %+v; want nothing", st, ok, castles)
		}
	})

	t.Run("Replace", func(t *testing.T) {
		s, _ := newStores(t)
		mustWrite(t, s.Batches, storage.Batch{ManorCastles: []storage.ManorCastle{dion, gludio}, ManorState: &state})
		st, ok, castles := load(t, s.Manor)
		if !ok || st != state || !reflect.DeepEqual(castles, []storage.ManorCastle{gludio, dion}) {
			t.Fatalf("LoadManor = %+v, %v, %+v", st, ok, castles)
		}
		emptied := storage.ManorCastle{CastleID: 1}
		next := storage.ManorState{Mode: 2, NextChange: 1_795_000_360_000}
		mustWrite(t, s.Batches, storage.Batch{ManorCastles: []storage.ManorCastle{emptied}})
		mustWrite(t, s.Batches, storage.Batch{ManorState: &next})
		st, ok, castles = load(t, s.Manor)
		if !ok || st != next || !reflect.DeepEqual(castles, []storage.ManorCastle{emptied, dion}) {
			t.Fatalf("LoadManor after the replace = %+v, %v, %+v", st, ok, castles)
		}
	})

	t.Run("WithCastles", func(t *testing.T) {
		s, fail := newStores(t)
		castle := storage.Castle{ID: 1, Treasury: 5000}
		b := storage.Batch{Castles: []storage.Castle{castle}, ManorCastles: []storage.ManorCastle{gludio}, ManorState: &state}
		// A failing transition writes neither the castle nor the manor.
		boom := errors.New("boom")
		fail(boom)
		if err := s.Batches.WriteBatch(t.Context(), b); !errors.Is(err, boom) {
			t.Fatalf("WriteBatch = %v, want %v", err, boom)
		}
		if _, ok, castles := load(t, s.Manor); ok || castles != nil {
			t.Fatalf("LoadManor after a failed batch = %v, %+v; want nothing", ok, castles)
		}
		mustWrite(t, s.Batches, b)
		got, err := s.Castles.LoadCastles(t.Context())
		if err != nil || len(got) != 1 || got[0].Treasury != 5000 {
			t.Fatalf("LoadCastles = %+v, %v", got, err)
		}
		if _, ok, castles := load(t, s.Manor); !ok || len(castles) != 1 {
			t.Fatalf("LoadManor = %v, %+v", ok, castles)
		}
	})
}
