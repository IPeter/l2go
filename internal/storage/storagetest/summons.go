package storagetest

import (
	"errors"
	"reflect"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestSummonStore runs the SummonStore conformance tests. newStore must
// return an empty store for each call, and a function that makes the
// store's next write fail with the given error.
func TestSummonStore(t *testing.T, newStore func(t *testing.T) (s storage.SummonStore, failNext func(error))) {
	wolf := storage.Pet{ControlID: 268480001, OwnerID: 268470001, Name: "Rex", Level: 15, CurHP: 400.5, CurMP: 160,
		Exp: 9000, SP: 12, Fed: 200, Restore: true,
		Effects: []storage.SavedSkill{{SkillID: 1068, SkillLevel: 3, RemainingTime: 1100}}}

	t.Run("pets", func(t *testing.T) {
		s, _ := newStore(t)
		if _, err := s.LoadPet(t.Context(), wolf.ControlID); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("LoadPet of a missing pet: %v", err)
		}
		if err := s.SavePet(t.Context(), wolf); err != nil {
			t.Fatalf("SavePet: %v", err)
		}
		got, err := s.LoadPet(t.Context(), wolf.ControlID)
		if err != nil || !reflect.DeepEqual(got, wolf) {
			t.Fatalf("LoadPet = %+v, %v; want %+v", got, err, wolf)
		}
		// A save replaces the pet.
		hungry := wolf
		hungry.Fed, hungry.Effects, hungry.Restore = 10, nil, false
		if err := s.SavePet(t.Context(), hungry); err != nil {
			t.Fatalf("SavePet: %v", err)
		}
		if got, err := s.LoadPet(t.Context(), wolf.ControlID); err != nil || !reflect.DeepEqual(got, hungry) {
			t.Fatalf("LoadPet after the update = %+v, %v", got, err)
		}
		if err := s.DeletePet(t.Context(), wolf.ControlID); err != nil {
			t.Fatalf("DeletePet: %v", err)
		}
		if err := s.DeletePet(t.Context(), wolf.ControlID); err != nil {
			t.Fatalf("DeletePet of a missing pet: %v", err)
		}
		if _, err := s.LoadPet(t.Context(), wolf.ControlID); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("LoadPet after the delete: %v", err)
		}
	})

	t.Run("owner summons", func(t *testing.T) {
		s, _ := newStore(t)
		got, err := s.LoadOwnerSummons(t.Context(), 7)
		if err != nil || !reflect.DeepEqual(got, storage.OwnerSummons{OwnerID: 7}) {
			t.Fatalf("LoadOwnerSummons of nothing = %+v, %v", got, err)
		}
		o := storage.OwnerSummons{OwnerID: 7,
			Servitor: &storage.SavedServitor{SkillID: 1111, CurHP: 300, CurMP: 100, Time: 1200000},
			Effects: []storage.ServitorEffects{{ClassIndex: 0, SummonSkillID: 1111,
				Effects: []storage.SavedSkill{{SkillID: 1040, SkillLevel: 2, RemainingTime: 600}}}},
			RestorePet: 268480001}
		if err := s.SaveOwnerSummons(t.Context(), o); err != nil {
			t.Fatalf("SaveOwnerSummons: %v", err)
		}
		if got, err := s.LoadOwnerSummons(t.Context(), 7); err != nil || !reflect.DeepEqual(got, o) {
			t.Fatalf("LoadOwnerSummons = %+v, %v; want %+v", got, err, o)
		}
		if err := s.SaveOwnerSummons(t.Context(), storage.OwnerSummons{OwnerID: 7}); err != nil {
			t.Fatalf("SaveOwnerSummons of nothing: %v", err)
		}
		if got, err := s.LoadOwnerSummons(t.Context(), 7); err != nil || !reflect.DeepEqual(got, storage.OwnerSummons{OwnerID: 7}) {
			t.Fatalf("LoadOwnerSummons after the clear = %+v, %v", got, err)
		}
	})

	t.Run("failed write", func(t *testing.T) {
		s, failNext := newStore(t)
		boom := errors.New("boom")
		failNext(boom)
		if err := s.SavePet(t.Context(), wolf); !errors.Is(err, boom) {
			t.Fatalf("SavePet with a failure armed: %v", err)
		}
		if _, err := s.LoadPet(t.Context(), wolf.ControlID); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("a failed write stored the pet: %v", err)
		}
	})
}
