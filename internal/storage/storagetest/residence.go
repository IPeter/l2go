package storagetest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// CastleStores are the stores of one database the castles live in (Plan 14
// M11i): castles are read with Castles and written with Batches, together
// with the clans a siege changes, which Clans reads.
type CastleStores struct {
	Batches storage.BatchWriter
	Castles storage.CastleStore
	Clans   storage.ClanStore
}

// TestCastleStore runs the castle conformance tests. newStores must return
// empty stores of one database for each call, and a function that makes the
// next write fail with the given error after it did all its work, just
// before it commits.
func TestCastleStore(t *testing.T, newStores func(t *testing.T) (CastleStores, func(error))) {
	gludio := storage.Castle{ID: 1, TaxPercent: 10, Treasury: 123_456, SiegeDate: 1_795_000_000_000, RegTimeOver: true,
		RegTimeEnd: 1_794_000_000_000, ShowNpcCrest: true, TicketBuyCount: 2,
		Functions:    []storage.CastleFunction{{Type: 1, Level: 2, Lease: 7000, Rate: 86_400_000, EndTime: 1_795_100_000_000}},
		DoorUpgrades: []storage.CastleDoorUpgrade{{DoorID: 19210001, Ratio: 2}},
		TrapUpgrades: []storage.CastleTrapUpgrade{{TowerIndex: 1, Level: 3}},
		SiegeClans:   []storage.SiegeClan{{ClanID: 0x10000010, Type: 1}, {ClanID: 0x10000011, Type: 2}},
		HiredGuards:  []storage.HiredGuard{{NpcID: 35010, X: -18000, Y: 110000, Z: -2500, Heading: 100, TicketItem: 3960}},
	}
	dion := storage.Castle{ID: 2, SiegeDate: 1_795_000_000_000}

	load := func(t *testing.T, s storage.CastleStore) []storage.Castle {
		t.Helper()
		got, err := s.LoadCastles(t.Context())
		if err != nil {
			t.Fatalf("LoadCastles: %v", err)
		}
		return got
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStores(t)
		if got := load(t, s.Castles); got != nil {
			t.Errorf("LoadCastles = %+v, want nil", got)
		}
	})

	t.Run("Replace", func(t *testing.T) {
		s, _ := newStores(t)
		mustWrite(t, s.Batches, storage.Batch{Castles: []storage.Castle{dion, gludio}})
		if got, want := load(t, s.Castles), []storage.Castle{gludio, dion}; !reflect.DeepEqual(got, want) {
			t.Fatalf("LoadCastles = %+v, want %+v", got, want)
		}
		ended := gludio
		ended.SiegeClans, ended.HiredGuards, ended.Treasury = nil, nil, 0
		mustWrite(t, s.Batches, storage.Batch{Castles: []storage.Castle{ended}})
		if got, want := load(t, s.Castles), []storage.Castle{ended, dion}; !reflect.DeepEqual(got, want) {
			t.Fatalf("LoadCastles after the replace = %+v, want %+v", got, want)
		}
	})

	t.Run("WithClans", func(t *testing.T) {
		s, fail := newStores(t)
		owner := storage.Clan{ID: 0x10000010, Name: "Lords", Level: 5, Reputation: 1500, CastleID: 1}
		// A failing transition writes neither the castle nor the clan.
		boom := errors.New("boom")
		fail(boom)
		if err := s.Batches.WriteBatch(t.Context(), storage.Batch{Clans: []storage.Clan{owner}, Castles: []storage.Castle{gludio}}); !errors.Is(err, boom) {
			t.Fatalf("WriteBatch = %v, want %v", err, boom)
		}
		if got := load(t, s.Castles); got != nil {
			t.Fatalf("LoadCastles after a failed batch = %+v, want nil", got)
		}
		mustWrite(t, s.Batches, storage.Batch{Clans: []storage.Clan{owner}, Castles: []storage.Castle{gludio}})
		clans, err := s.Clans.LoadClans(t.Context())
		if err != nil || len(clans) != 1 || clans[0].CastleID != 1 {
			t.Fatalf("LoadClans = %+v, %v; want the owner of castle 1", clans, err)
		}
		if got := load(t, s.Castles); len(got) != 1 || got[0].ID != 1 {
			t.Fatalf("LoadCastles = %+v, want castle 1", got)
		}
	})
}
