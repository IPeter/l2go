package storagetest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestSevenSignsStore runs the SevenSignsStore conformance tests. newStore
// must return an empty store for each call, and a function that makes the
// store's next write fail with the given error after it did all its work,
// just before it commits.
func TestSevenSignsStore(t *testing.T, newStore func(t *testing.T) (s storage.SevenSignsStore, failNext func(error))) {
	load := func(t *testing.T, s storage.SevenSignsStore) storage.SevenSigns {
		t.Helper()
		got, err := s.LoadSevenSigns(t.Context())
		if err != nil {
			t.Fatalf("LoadSevenSigns: %v", err)
		}
		return got
	}
	// equal compares two states, nil and empty lists alike.
	equal := func(a, b storage.SevenSigns) bool {
		norm := func(s storage.SevenSigns) storage.SevenSigns {
			if len(s.Players) == 0 {
				s.Players = nil
			}
			if len(s.Festival) == 0 {
				s.Festival = nil
			}
			if len(s.Status.AccumulatedBonus) == 0 {
				s.Status.AccumulatedBonus = nil
			}
			return s
		}
		return reflect.DeepEqual(norm(a), norm(b))
	}
	save := func(t *testing.T, s storage.SevenSignsStore, sv storage.SevenSignsSave) {
		t.Helper()
		if err := s.SaveSevenSigns(t.Context(), sv); err != nil {
			t.Fatalf("SaveSevenSigns: %v", err)
		}
	}
	status := storage.SevenSignsStatus{
		Cycle: 3, Period: 1, PreviousWinner: 2, DawnStoneScore: 1500, DuskStoneScore: 300,
		DawnFestivalScore: 120, DuskFestivalScore: 60, AvariceOwner: 2, GnosisOwner: 0, StrifeOwner: 1,
		AvariceDawnScore: 4, GnosisDawnScore: 1, StrifeDawnScore: 2, AvariceDuskScore: 0, GnosisDuskScore: 3, StrifeDuskScore: 5,
		FestivalCycle: 6, AccumulatedBonus: []int32{100, 0, 25, 0, 5}, Date: 1796040000123, NextPeriodChange: 1796126400000,
	}
	players := []storage.SevenSignsPlayer{
		{CharID: 0x10000001, Cabal: "dawn", Seal: 1, RedStones: 10, GreenStones: 2, BlueStones: 3, AncientAdena: 119, Contribution: 119},
		{CharID: 0x10000002, Cabal: "dusk", Seal: 3},
		{CharID: 0x10000003, Cabal: "", Seal: 0, RedStones: 1, AncientAdena: 10},
	}
	festival := []storage.SevenSignsFestivalScore{
		{Festival: 0, Cabal: "dawn", Cycle: 1, Date: 0, Score: 0, Members: ""},
		{Festival: 0, Cabal: "dusk", Cycle: 1, Date: 1796040000000, Score: 75, Members: "Alpha,Beta,Gamma,Delta,Eps"},
		{Festival: 4, Cabal: "dawn", Cycle: 1, Date: 1796040001000, Score: 160, Members: "Zed"},
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStore(t)
		if got := load(t, s); !equal(got, storage.SevenSigns{}) {
			t.Errorf("LoadSevenSigns = %+v, want the zero value", got)
		}
	})

	t.Run("SaveAndLoad", func(t *testing.T) {
		s, _ := newStore(t)
		save(t, s, storage.SevenSignsSave{Status: status, Players: players, ReplaceFestival: true, Festival: festival})
		want := storage.SevenSigns{HasStatus: true, Status: status, Players: players, Festival: festival}
		if got := load(t, s); !equal(got, want) {
			t.Errorf("LoadSevenSigns = %+v, want %+v", got, want)
		}
	})

	t.Run("PlayersUpsert", func(t *testing.T) {
		s, _ := newStore(t)
		save(t, s, storage.SevenSignsSave{Status: status, Players: players})
		changed := players[1]
		changed.RedStones, changed.AncientAdena, changed.Contribution = 5, 50, 50
		added := storage.SevenSignsPlayer{CharID: 0x10000000, Cabal: "dawn", Seal: 2}
		next := status
		next.DuskStoneScore = 350
		save(t, s, storage.SevenSignsSave{Status: next, Players: []storage.SevenSignsPlayer{changed, added}})
		want := storage.SevenSigns{HasStatus: true, Status: next,
			Players: []storage.SevenSignsPlayer{added, players[0], changed, players[2]}}
		if got := load(t, s); !equal(got, want) {
			t.Errorf("after an upsert = %+v, want %+v (sorted by CharID, the others kept)", got, want)
		}
	})

	t.Run("FestivalReplaceOnlyWhenAsked", func(t *testing.T) {
		s, _ := newStore(t)
		save(t, s, storage.SevenSignsSave{Status: status, ReplaceFestival: true, Festival: []storage.SevenSignsFestivalScore{festival[2], festival[1], festival[0]}})
		if got := load(t, s); !reflect.DeepEqual(got.Festival, festival) {
			t.Errorf("festival = %+v, want %+v (sorted by festival, cabal, cycle)", got.Festival, festival)
		}
		save(t, s, storage.SevenSignsSave{Status: status, Festival: festival[:1]})
		if got := load(t, s); !reflect.DeepEqual(got.Festival, festival) {
			t.Errorf("festival without ReplaceFestival = %+v, want it unchanged", got.Festival)
		}
		save(t, s, storage.SevenSignsSave{Status: status, ReplaceFestival: true})
		if got := load(t, s); len(got.Festival) != 0 {
			t.Errorf("festival after an empty replace = %+v, want none", got.Festival)
		}
	})

	t.Run("FailedWriteChangesNothing", func(t *testing.T) {
		s, failNext := newStore(t)
		save(t, s, storage.SevenSignsSave{Status: status, Players: players, ReplaceFestival: true, Festival: festival})
		before := load(t, s)
		boom := errors.New("boom")
		failNext(boom)
		next := status
		next.Cycle, next.Period = 4, 0
		changed := players[0]
		changed.Cabal = ""
		err := s.SaveSevenSigns(t.Context(), storage.SevenSignsSave{Status: next,
			Players:         []storage.SevenSignsPlayer{changed, {CharID: 0x10000009, Cabal: "dusk"}},
			ReplaceFestival: true})
		if !errors.Is(err, boom) {
			t.Fatalf("SaveSevenSigns = %v, want %v", err, boom)
		}
		if got := load(t, s); !equal(got, before) {
			t.Errorf("after a failed write = %+v, want %+v", got, before)
		}
	})
}
