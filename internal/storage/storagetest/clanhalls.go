package storagetest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// ClanHallStores are the stores of one database the clan halls live in (Plan
// 14 M11j): halls and auctions are read with Halls and written with Batches,
// together with the clans and clan warehouse items a sale or a lease
// changes, which Clans reads.
type ClanHallStores struct {
	Batches storage.BatchWriter
	Halls   storage.ClanHallStore
	Clans   storage.ClanStore
}

// TestClanHallStore runs the clan hall conformance tests. newStores must
// return empty stores of one database for each call, and a function that
// makes the next write fail with the given error after it did all its work,
// just before it commits.
func TestClanHallStore(t *testing.T, newStores func(t *testing.T) (ClanHallStores, func(error))) {
	moonstone := storage.ClanHall{ID: 22, PaidUntil: 1_795_000_000_000, Paid: true,
		Functions: []storage.ClanHallFunction{{Type: 3, Level: 40, Lease: 800, Rate: 86_400_000, EndTime: 1_795_100_000_000}}}
	fortress := storage.ClanHall{ID: 21, NextSiege: 1_796_000_000_000, Attackers: []int32{0x10000010, 0x10000011},
		Vars: map[string]string{"flag.0x10000010": "35423"}}
	auction := storage.Auction{ID: 23, SellerName: "NPC", SellerClanName: "NPC Clan", ItemName: "Onyx Hall",
		StartingBid: 20_000_000, EndDate: 1_795_200_000_000,
		Bids: []storage.AuctionBid{{ClanID: 0x10000010, BidderName: "Alice", ClanName: "Lions", MaxBid: 21_000_000, TimeBid: 1_795_000_000_000}}}
	npcAuction := storage.Auction{ID: 24, SellerName: "NPC", SellerClanName: "NPC Clan", ItemName: "Topaz Hall", StartingBid: 20_000_000, EndDate: 1}

	loadHalls := func(t *testing.T, s storage.ClanHallStore) []storage.ClanHall {
		t.Helper()
		got, err := s.LoadClanHalls(t.Context())
		if err != nil {
			t.Fatalf("LoadClanHalls: %v", err)
		}
		return got
	}
	loadAuctions := func(t *testing.T, s storage.ClanHallStore) []storage.Auction {
		t.Helper()
		got, err := s.LoadAuctions(t.Context())
		if err != nil {
			t.Fatalf("LoadAuctions: %v", err)
		}
		return got
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStores(t)
		if got := loadHalls(t, s.Halls); got != nil {
			t.Errorf("LoadClanHalls = %+v, want nil", got)
		}
		if got := loadAuctions(t, s.Halls); got != nil {
			t.Errorf("LoadAuctions = %+v, want nil", got)
		}
	})

	t.Run("Replace", func(t *testing.T) {
		s, _ := newStores(t)
		mustWrite(t, s.Batches, storage.Batch{ClanHalls: []storage.ClanHall{moonstone, fortress}, Auctions: []storage.Auction{npcAuction, auction}})
		if got, want := loadHalls(t, s.Halls), []storage.ClanHall{fortress, moonstone}; !reflect.DeepEqual(got, want) {
			t.Fatalf("LoadClanHalls = %+v, want %+v", got, want)
		}
		if got, want := loadAuctions(t, s.Halls), []storage.Auction{auction, npcAuction}; !reflect.DeepEqual(got, want) {
			t.Fatalf("LoadAuctions = %+v, want %+v", got, want)
		}
		freed := moonstone
		freed.PaidUntil, freed.Paid, freed.Functions = 0, false, nil
		mustWrite(t, s.Batches, storage.Batch{ClanHalls: []storage.ClanHall{freed}, DeletedAuctions: []int32{23}})
		if got, want := loadHalls(t, s.Halls), []storage.ClanHall{fortress, freed}; !reflect.DeepEqual(got, want) {
			t.Fatalf("LoadClanHalls after the replace = %+v, want %+v", got, want)
		}
		if got, want := loadAuctions(t, s.Halls), []storage.Auction{npcAuction}; !reflect.DeepEqual(got, want) {
			t.Fatalf("LoadAuctions after the delete = %+v, want %+v", got, want)
		}
	})

	t.Run("WithClans", func(t *testing.T) {
		s, fail := newStores(t)
		owner := storage.Clan{ID: 0x10000010, Name: "Lions", Level: 4, HideoutID: 22}
		// A failing sale writes neither the hall, the auction nor the clan.
		boom := errors.New("boom")
		fail(boom)
		b := storage.Batch{Clans: []storage.Clan{owner}, ClanHalls: []storage.ClanHall{moonstone}, DeletedAuctions: []int32{22}}
		if err := s.Batches.WriteBatch(t.Context(), b); !errors.Is(err, boom) {
			t.Fatalf("WriteBatch = %v, want %v", err, boom)
		}
		if got := loadHalls(t, s.Halls); got != nil {
			t.Fatalf("LoadClanHalls after a failed batch = %+v, want nil", got)
		}
		mustWrite(t, s.Batches, b)
		clans, err := s.Clans.LoadClans(t.Context())
		if err != nil || len(clans) != 1 || clans[0].HideoutID != 22 {
			t.Fatalf("LoadClans = %+v, %v; want the owner of hall 22", clans, err)
		}
		if got := loadHalls(t, s.Halls); len(got) != 1 || got[0].ID != 22 {
			t.Fatalf("LoadClanHalls = %+v, want hall 22", got)
		}
	})
}
