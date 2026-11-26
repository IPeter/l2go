package storagetest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestOlympiadStore runs the OlympiadStore conformance tests. newStore must
// return an empty store for each call, and a function that makes the
// store's next write fail with the given error after it did all its work,
// just before it commits.
func TestOlympiadStore(t *testing.T, newStore func(t *testing.T) (s storage.OlympiadStore, failNext func(error))) {
	load := func(t *testing.T, s storage.OlympiadStore) storage.Olympiad {
		t.Helper()
		got, err := s.LoadOlympiad(t.Context())
		if err != nil {
			t.Fatalf("LoadOlympiad: %v", err)
		}
		return got
	}
	// equal compares two states, nil and empty lists alike.
	equal := func(a, b storage.Olympiad) bool {
		norm := func(o storage.Olympiad) storage.Olympiad {
			if len(o.Nobles) == 0 {
				o.Nobles = nil
			}
			if len(o.NoblesEOM) == 0 {
				o.NoblesEOM = nil
			}
			if len(o.Heroes) == 0 {
				o.Heroes = nil
			}
			if len(o.Unclaimed) == 0 {
				o.Unclaimed = nil
			}
			return o
		}
		return reflect.DeepEqual(norm(a), norm(b))
	}
	state := storage.Olympiad{
		Cycle: 3, Period: 1, OlympiadEnd: 1796040000123, ValidationEnd: 1796126400123, NextWeeklyChange: 1795500000000,
		Nobles: []storage.OlympiadNoble{
			{CharID: 0x10000002, ClassID: 88, Name: "Duelist", Points: 40, Done: 12, Won: 7, Lost: 4, Drawn: 1},
			{CharID: 0x10000001, ClassID: 93, Name: "Adventurer", Points: 18},
		},
		NoblesEOM: []storage.OlympiadNoble{{CharID: 0x10000002, ClassID: 88, Name: "Duelist", Points: 33, Done: 10, Won: 6, Lost: 4}},
		Heroes: []storage.Hero{
			{CharID: 0x10000002, ClassID: 88, Name: "Duelist", Count: 2, Played: true, Claimed: true, Message: "Fear me"},
			{CharID: 0x10000009, ClassID: 93, Name: "Old", Count: 1},
		},
		Unclaimed: []storage.UnclaimedPasses{{CharID: 0x10000002, Passes: 100000}},
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStore(t)
		if got := load(t, s); !equal(got, storage.Olympiad{}) {
			t.Errorf("LoadOlympiad = %+v, want the zero value", got)
		}
		if f, err := s.OlympiadFights(t.Context(), 1, 1<<62); err != nil || len(f) != 0 {
			t.Errorf("OlympiadFights = %v, %v", f, err)
		}
		if d, err := s.HeroDiary(t.Context(), 1); err != nil || len(d) != 0 {
			t.Errorf("HeroDiary = %v, %v", d, err)
		}
	})

	t.Run("SaveReplaces", func(t *testing.T) {
		s, _ := newStore(t)
		if err := s.SaveOlympiad(t.Context(), state); err != nil {
			t.Fatalf("SaveOlympiad: %v", err)
		}
		if got := load(t, s); !equal(got, state) {
			t.Errorf("LoadOlympiad = %+v, want %+v (order kept)", got, state)
		}
		next := storage.Olympiad{Cycle: 4, OlympiadEnd: 1798718400000, NextWeeklyChange: 1796731200000, NoblesEOM: state.NoblesEOM, Heroes: state.Heroes[:1]}
		if err := s.SaveOlympiad(t.Context(), next); err != nil {
			t.Fatalf("SaveOlympiad: %v", err)
		}
		if got := load(t, s); !equal(got, next) {
			t.Errorf("after a second save = %+v, want %+v", got, next)
		}
	})

	t.Run("FailedWriteChangesNothing", func(t *testing.T) {
		s, failNext := newStore(t)
		if err := s.SaveOlympiad(t.Context(), state); err != nil {
			t.Fatalf("SaveOlympiad: %v", err)
		}
		boom := errors.New("boom")
		failNext(boom)
		if err := s.SaveOlympiad(t.Context(), storage.Olympiad{Cycle: 9}); !errors.Is(err, boom) {
			t.Fatalf("SaveOlympiad = %v, want %v", err, boom)
		}
		if got := load(t, s); !equal(got, state) {
			t.Errorf("after a failed write = %+v, want %+v", got, state)
		}
	})

	t.Run("Fights", func(t *testing.T) {
		s, _ := newStore(t)
		fights := []storage.OlympiadFight{
			{CharOneID: 1, CharTwoID: 2, CharOneName: "A", CharTwoName: "B", CharOneClass: 88, CharTwoClass: 89, Winner: 1, Start: 3000, Time: 61000, Classed: true},
			{CharOneID: 3, CharTwoID: 1, CharOneName: "C", CharTwoName: "A", CharOneClass: 90, CharTwoClass: 88, Winner: 0, Start: 1000, Time: 360000},
			{CharOneID: 2, CharTwoID: 3, CharOneName: "B", CharTwoName: "C", CharOneClass: 89, CharTwoClass: 90, Winner: 2, Start: 2000, Time: 5000},
			{CharOneID: 1, CharTwoID: 3, CharOneName: "A", CharTwoName: "C", CharOneClass: 88, CharTwoClass: 90, Winner: 2, Start: 9000, Time: 5000},
		}
		for _, f := range fights {
			if err := s.AddOlympiadFight(t.Context(), f); err != nil {
				t.Fatalf("AddOlympiadFight: %v", err)
			}
		}
		got, err := s.OlympiadFights(t.Context(), 1, 9000)
		if err != nil {
			t.Fatal(err)
		}
		if want := []storage.OlympiadFight{fights[1], fights[0]}; !reflect.DeepEqual(got, want) {
			t.Errorf("OlympiadFights(1, before 9000) = %+v, want %+v (oldest first)", got, want)
		}
	})

	t.Run("Diary", func(t *testing.T) {
		s, failNext := newStore(t)
		entries := []storage.HeroDiaryEntry{
			{CharID: 1, Time: 5000, Action: 2},
			{CharID: 2, Time: 1000, Action: 1, Param: 29001},
			{CharID: 1, Time: 2000, Action: 1, Param: 25001},
		}
		for _, e := range entries {
			if err := s.AddHeroDiary(t.Context(), e); err != nil {
				t.Fatalf("AddHeroDiary: %v", err)
			}
		}
		boom := errors.New("boom")
		failNext(boom)
		if err := s.AddHeroDiary(t.Context(), storage.HeroDiaryEntry{CharID: 1, Time: 1}); !errors.Is(err, boom) {
			t.Fatalf("AddHeroDiary = %v, want %v", err, boom)
		}
		got, err := s.HeroDiary(t.Context(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if want := []storage.HeroDiaryEntry{entries[2], entries[0]}; !reflect.DeepEqual(got, want) {
			t.Errorf("HeroDiary(1) = %+v, want %+v", got, want)
		}
	})
}
