package storagetest

import (
	"errors"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// CursedWeaponStores are the stores of one database the cursed weapon state
// lives in (Plan 14 M11m): the rows are read with CursedWeapons and written
// with Batches, together with the holders' characters and items.
type CursedWeaponStores struct {
	Characters    SaverStore
	Batches       storage.BatchWriter
	CursedWeapons storage.CursedWeaponStore
}

// TestCursedWeaponStore runs the cursed weapon conformance tests. newStores
// must return empty stores of one database for each call, and a function
// that makes the next write fail with the given error after it did all its
// work, just before it commits.
func TestCursedWeaponStore(t *testing.T, newStores func(t *testing.T) (CursedWeaponStores, func(error))) {
	zariche := storage.CursedWeapon{ItemID: 8190, CharID: 0x10000001, PlayerKarma: 720, PlayerPKKills: 3, NbKills: 12,
		EndTime: 1_795_000_000_123, ItemObjectID: 0x10000005}
	akamanah := storage.CursedWeapon{ItemID: 8689, CharID: 0x10000002, EndTime: 1_795_000_000_456}

	load := func(t *testing.T, s storage.CursedWeaponStore) []storage.CursedWeapon {
		t.Helper()
		got, err := s.LoadCursedWeapons(t.Context())
		if err != nil {
			t.Fatalf("LoadCursedWeapons: %v", err)
		}
		return got
	}
	create := func(t *testing.T, s SaverStore, id int32, name string, items ...storage.Item) {
		t.Helper()
		c := storage.Character{ObjectID: id, Account: name, Name: name, Level: 1, Karma: 5, PKKills: 1, Items: items}
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}
	weapon := func(objectID, itemID int32) storage.Item {
		return storage.Item{ObjectID: objectID, ItemID: itemID, Count: 1, Slot: storage.PaperdollRHand, Mana: storage.NoMana}
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStores(t)
		if got := load(t, s.CursedWeapons); got != nil {
			t.Errorf("LoadCursedWeapons = %+v, want nil", got)
		}
		got, err := s.CursedWeapons.ItemsByTemplate(t.Context(), []int32{8190, 8689})
		if err != nil || got != nil {
			t.Errorf("ItemsByTemplate = %+v, %v; want nil", got, err)
		}
		mustWrite(t, s.Batches, storage.Batch{DeletedCursedWeapons: []int32{8190}})
	})

	t.Run("Rows", func(t *testing.T) {
		s, _ := newStores(t)
		mustWrite(t, s.Batches, storage.Batch{CursedWeapons: []storage.CursedWeapon{zariche, akamanah}})
		if got, want := load(t, s.CursedWeapons), []storage.CursedWeapon{zariche, akamanah}; !slices.Equal(got, want) {
			t.Fatalf("LoadCursedWeapons = %+v, want %+v", got, want)
		}
		// A row is replaced whole.
		killed := zariche
		killed.NbKills, killed.EndTime = 13, zariche.EndTime-180_000
		mustWrite(t, s.Batches, storage.Batch{CursedWeapons: []storage.CursedWeapon{killed}})
		if got, want := load(t, s.CursedWeapons), []storage.CursedWeapon{killed, akamanah}; !slices.Equal(got, want) {
			t.Errorf("after a replacement: %+v, want %+v", got, want)
		}
		// Deletes go first: a batch can delete and store the same weapon.
		moved := akamanah
		moved.CharID = 0x10000009
		mustWrite(t, s.Batches, storage.Batch{DeletedCursedWeapons: []int32{8190, 8689}, CursedWeapons: []storage.CursedWeapon{moved}})
		if got, want := load(t, s.CursedWeapons), []storage.CursedWeapon{moved}; !slices.Equal(got, want) {
			t.Errorf("after deletes: %+v, want %+v", got, want)
		}
	})

	t.Run("ItemsByTemplate", func(t *testing.T) {
		s, _ := newStores(t)
		create(t, s.Characters, 0x10000001, "holder", weapon(0x10000005, 8190),
			storage.Item{ObjectID: 0x10000006, ItemID: 57, Count: 100, Mana: storage.NoMana})
		create(t, s.Characters, 0x10000002, "stray", weapon(0x10000003, 8689))
		got, err := s.CursedWeapons.ItemsByTemplate(t.Context(), []int32{8190, 8689})
		if err != nil {
			t.Fatalf("ItemsByTemplate: %v", err)
		}
		if len(got) != 2 || got[0].Owner != 0x10000002 || got[0].Item.ObjectID != 0x10000003 || got[0].Item.ItemID != 8689 ||
			got[1].Owner != 0x10000001 || got[1].Item.ObjectID != 0x10000005 || got[1].Item.Slot != storage.PaperdollRHand {
			t.Errorf("ItemsByTemplate = %+v", got)
		}
		if got, err := s.CursedWeapons.ItemsByTemplate(t.Context(), []int32{8190}); err != nil || len(got) != 1 || got[0].Owner != 0x10000001 {
			t.Errorf("ItemsByTemplate(8190) = %+v, %v", got, err)
		}
	})

	// The end of an offline holder's weapon: its copy goes, its karma and
	// PK count come back and the row goes, in one batch.
	t.Run("OfflineEnd", func(t *testing.T) {
		s, _ := newStores(t)
		create(t, s.Characters, 0x10000001, "holder", weapon(0x10000005, 8190))
		mustWrite(t, s.Batches, storage.Batch{CursedWeapons: []storage.CursedWeapon{zariche}})
		mustWrite(t, s.Batches, storage.Batch{
			Items: []storage.OwnerItemChanges{{Owner: 0x10000001, Destroyed: []int32{0x10000005}}},
			Patches: []storage.CharacterPatch{{ObjectID: 0x10000001, Fields: storage.PatchKarma | storage.PatchPKKills,
				Values: storage.PatchValues{Karma: 720, PKKills: 3}}},
			DeletedCursedWeapons: []int32{8190},
		})
		c, err := s.Characters.Get(t.Context(), 0x10000001)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if c.Karma != 720 || c.PKKills != 3 || len(c.Items) != 0 {
			t.Errorf("after the end: karma %d, PKs %d, items %+v; want 720, 3, none", c.Karma, c.PKKills, c.Items)
		}
		if got := load(t, s.CursedWeapons); got != nil {
			t.Errorf("rows after the end: %+v", got)
		}
	})

	t.Run("FailedWrite", func(t *testing.T) {
		s, failNext := newStores(t)
		create(t, s.Characters, 0x10000001, "holder", weapon(0x10000005, 8190))
		mustWrite(t, s.Batches, storage.Batch{CursedWeapons: []storage.CursedWeapon{zariche}})
		failNext(errInjected)
		err := s.Batches.WriteBatch(t.Context(), storage.Batch{
			Items: []storage.OwnerItemChanges{{Owner: 0x10000001, Destroyed: []int32{0x10000005}}},
			Patches: []storage.CharacterPatch{{ObjectID: 0x10000001, Fields: storage.PatchKarma,
				Values: storage.PatchValues{Karma: 0}}},
			DeletedCursedWeapons: []int32{8190},
			CursedWeapons:        []storage.CursedWeapon{akamanah},
		})
		if !errors.Is(err, errInjected) {
			t.Fatalf("WriteBatch error = %v, want the injected failure", err)
		}
		if got, want := load(t, s.CursedWeapons), []storage.CursedWeapon{zariche}; !slices.Equal(got, want) {
			t.Errorf("rows after a failed write = %+v, want %+v", got, want)
		}
		c, err := s.Characters.Get(t.Context(), 0x10000001)
		if err != nil || c.Karma != 5 || len(c.Items) != 1 {
			t.Errorf("character after a failed write = %+v, %v", c, err)
		}
	})
}
