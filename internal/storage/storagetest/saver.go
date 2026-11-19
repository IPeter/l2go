package storagetest

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// SaverStore is a character store that also saves item changes. Both stores
// are one.
type SaverStore interface {
	storage.CharacterStore
	storage.CharacterSaver
}

// without returns items without the ones with these object IDs.
func without(items []storage.Item, ids ...int32) []storage.Item {
	return slices.DeleteFunc(slices.Clone(items), func(it storage.Item) bool { return slices.Contains(ids, it.ObjectID) })
}

// TestCharacterSaver runs the CharacterSaver conformance tests (decision
// I2). newStore must return an empty store for each call.
func TestCharacterSaver(t *testing.T, newStore func(t *testing.T) SaverStore) {
	t.Run("FieldsWithoutItems", func(t *testing.T) {
		s := newStore(t)
		orig := sampleCharacter(100, "alice", "Thorin", baseTime)
		mustCreate(t, s, orig)

		upd := changedCharacter(orig)
		arg := upd
		// The identity fields are kept as stored, and c.Items is not used.
		arg.Account, arg.Name, arg.Race, arg.CreatedAt = "mallory", "Smaug", 0, baseTime.Add(time.Hour)
		if err := s.Save(t.Context(), &arg, storage.ItemChanges{}); err != nil {
			t.Fatalf("Save: %v", err)
		}
		want := upd
		want.Items = orig.Items
		checkStored(t, s, want)
		if ok, err := s.NameExists(t.Context(), "Smaug"); err != nil || ok {
			t.Errorf("NameExists(Smaug) = %v, %v, want false", ok, err)
		}
	})

	t.Run("ClearsTimes", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Dwalin", baseTime)
		mustCreate(t, s, c)
		c.DeleteTime, c.LastAccess = time.Time{}, time.Time{}
		if err := s.Save(t.Context(), &c, storage.ItemChanges{}); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got := mustGet(t, s, 100)
		if !got.DeleteTime.IsZero() || !got.LastAccess.IsZero() {
			t.Errorf("DeleteTime %v, LastAccess %v; want both zero", got.DeleteTime, got.LastAccess)
		}
	})

	t.Run("Upserts", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Gimli", baseTime)
		other := sampleCharacter(200, "bob", "Gloin", baseTime)
		mustCreate(t, s, c, other)

		adena := c.Items[2]
		adena.Count = 9
		unworn := c.Items[0]
		unworn.Slot = storage.PaperdollNone
		bought := storage.Item{ObjectID: 106, ItemID: itemBoneArrow, Count: 50, Loc: storage.LocWarehouse, Mana: storage.NoMana}
		changes := storage.ItemChanges{Upserts: []storage.Item{adena, unworn, bought}}
		if err := s.Save(t.Context(), &c, changes); err != nil {
			t.Fatalf("Save: %v", err)
		}
		want := c
		want.Items = append(without(c.Items, adena.ObjectID, unworn.ObjectID), adena, unworn, bought)
		checkStored(t, s, want)
		checkStored(t, s, other)
	})

	t.Run("DuplicateUpserts", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Ori", baseTime)
		mustCreate(t, s, c)
		first := storage.Item{ObjectID: 106, ItemID: itemAdena, Count: 1, Slot: storage.PaperdollNone, Mana: storage.NoMana}
		last := first
		last.Count = 2
		if err := s.Save(t.Context(), &c, storage.ItemChanges{Upserts: []storage.Item{first, last}}); err != nil {
			t.Fatalf("Save: %v", err)
		}
		want := c
		want.Items = append(slices.Clone(c.Items), last)
		checkStored(t, s, want)
	})

	t.Run("Destroyed", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Nori", baseTime)
		other := sampleCharacter(200, "bob", "Dori", baseTime)
		mustCreate(t, s, c, other)
		// A destroyed item goes whoever owns it.
		if err := s.Save(t.Context(), &c, storage.ItemChanges{Destroyed: []int32{102, 104, 203}}); err != nil {
			t.Fatalf("Save: %v", err)
		}
		want := c
		want.Items = without(c.Items, 102, 104)
		checkStored(t, s, want)
		wantOther := other
		wantOther.Items = without(other.Items, 203)
		checkStored(t, s, wantOther)
		checkItemIDsFree(t, s, 900, 102, 104, 203)
	})

	t.Run("LeftGiverFirst", func(t *testing.T) {
		s := newStore(t)
		giver := sampleCharacter(100, "alice", "Bilbo", baseTime)
		receiver := sampleCharacter(200, "bob", "Frodo", baseTime)
		mustCreate(t, s, giver, receiver)
		gift := giver.Items[2]

		if err := s.Save(t.Context(), &giver, storage.ItemChanges{Left: []int32{gift.ObjectID}}); err != nil {
			t.Fatalf("Save(giver): %v", err)
		}
		wantGiver := giver
		wantGiver.Items = without(giver.Items, gift.ObjectID)
		checkStored(t, s, wantGiver)
		checkStored(t, s, receiver)

		if err := s.Save(t.Context(), &receiver, storage.ItemChanges{Upserts: []storage.Item{gift}}); err != nil {
			t.Fatalf("Save(receiver): %v", err)
		}
		wantReceiver := receiver
		wantReceiver.Items = append(slices.Clone(receiver.Items), gift)
		checkStored(t, s, wantReceiver)
		checkStored(t, s, wantGiver)
	})

	t.Run("LeftReceiverFirst", func(t *testing.T) {
		s := newStore(t)
		giver := sampleCharacter(100, "alice", "Bilbo", baseTime)
		receiver := sampleCharacter(200, "bob", "Frodo", baseTime)
		mustCreate(t, s, giver, receiver)
		gift := giver.Items[2]

		// The receiver's save comes first and takes the item over.
		if err := s.Save(t.Context(), &receiver, storage.ItemChanges{Upserts: []storage.Item{gift}}); err != nil {
			t.Fatalf("Save(receiver): %v", err)
		}
		wantReceiver := receiver
		wantReceiver.Items = append(slices.Clone(receiver.Items), gift)
		wantGiver := giver
		wantGiver.Items = without(giver.Items, gift.ObjectID)
		checkStored(t, s, wantReceiver)
		checkStored(t, s, wantGiver)

		// The giver's late save must not delete the receiver's item.
		if err := s.Save(t.Context(), &giver, storage.ItemChanges{Left: []int32{gift.ObjectID}}); err != nil {
			t.Fatalf("Save(giver): %v", err)
		}
		checkStored(t, s, wantReceiver)
		checkStored(t, s, wantGiver)
	})

	t.Run("UpsertWinsOverDeletes", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Bifur", baseTime)
		mustCreate(t, s, c)
		// The item left and came back before the save.
		back := c.Items[0]
		back.Slot = storage.PaperdollNone
		changes := storage.ItemChanges{
			Upserts:   []storage.Item{back},
			Destroyed: []int32{back.ObjectID},
			Left:      []int32{back.ObjectID},
		}
		if err := s.Save(t.Context(), &c, changes); err != nil {
			t.Fatalf("Save: %v", err)
		}
		want := c
		want.Items = append(without(c.Items, back.ObjectID), back)
		checkStored(t, s, want)
	})

	t.Run("UnknownDeletes", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Bofur", baseTime)
		mustCreate(t, s, c)
		if err := s.Save(t.Context(), &c, storage.ItemChanges{Destroyed: []int32{999}, Left: []int32{998}}); err != nil {
			t.Fatalf("Save: %v", err)
		}
		checkStored(t, s, c)
	})

	t.Run("NotFound", func(t *testing.T) {
		s := newStore(t)
		ghost := sampleCharacter(100, "alice", "Ghost", baseTime)
		err := s.Save(t.Context(), &ghost, storage.ItemChanges{Upserts: ghost.Items, Destroyed: []int32{1}})
		if !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("Save error = %v, want ErrNotFound", err)
		}
		if _, err := s.Get(t.Context(), 100); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get after a failed Save: error = %v, want ErrNotFound", err)
		}
		// Its items were not stored either.
		mustCreate(t, s, sampleCharacter(100, "alice", "Ghost", baseTime))
	})
}

// errInjected is the failure TestFailedWrites makes the stores report.
var errInjected = errors.New("storagetest: injected failure")

// TestFailedWrites checks that a write that fails just before it commits
// leaves nothing behind: not the character's fields, not its items, not the
// items it took from another character. newStore must return an empty store
// and a function that makes the store's next Create, Update, Save or Delete
// fail with the given error after it did all its work, just before it
// commits; the failure is used up by that write.
func TestFailedWrites(t *testing.T, newStore func(t *testing.T) (s SaverStore, failNext func(error))) {
	setUp := func(t *testing.T) (s SaverStore, failNext func(error), a, b storage.Character) {
		t.Helper()
		s, failNext = newStore(t)
		a = sampleCharacter(100, "alice", "Fili", baseTime)
		b = sampleCharacter(200, "bob", "Kili", baseTime)
		mustCreate(t, s, a, b)
		return s, failNext, a, b
	}

	t.Run("Create", func(t *testing.T) {
		s, failNext, a, b := setUp(t)
		c := sampleCharacter(300, "carol", "Oin", baseTime)
		failNext(errInjected)
		if err := s.Create(t.Context(), &c); !errors.Is(err, errInjected) {
			t.Fatalf("Create error = %v, want the injected failure", err)
		}
		if _, err := s.Get(t.Context(), 300); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get after a failed Create: error = %v, want ErrNotFound", err)
		}
		if ok, err := s.NameExists(t.Context(), "Oin"); err != nil || ok {
			t.Errorf("NameExists after a failed Create = %v, %v, want false", ok, err)
		}
		checkStored(t, s, a)
		checkStored(t, s, b)
		// The failure is used up, and nothing is in the way of the retry.
		mustCreate(t, s, sampleCharacter(300, "carol", "Oin", baseTime))
	})

	t.Run("Update", func(t *testing.T) {
		s, failNext, a, b := setUp(t)
		upd := changedCharacter(a)
		upd.Items = append(upd.Items, b.Items[2]) // and take b's adena
		failNext(errInjected)
		if err := s.Update(t.Context(), &upd); !errors.Is(err, errInjected) {
			t.Fatalf("Update error = %v, want the injected failure", err)
		}
		checkStored(t, s, a)
		checkStored(t, s, b)
		if err := s.Update(t.Context(), &upd); err != nil {
			t.Fatalf("Update after the failure: %v", err)
		}
		checkStored(t, s, upd)
	})

	t.Run("Save", func(t *testing.T) {
		s, failNext, a, b := setUp(t)
		upd := changedCharacter(a)
		changes := storage.ItemChanges{
			Upserts: []storage.Item{
				b.Items[2], // taken from b
				{ObjectID: 106, ItemID: itemAdena, Count: 3, Slot: storage.PaperdollNone, Mana: storage.NoMana},
			},
			Destroyed: []int32{102, 204},
			Left:      []int32{103},
		}
		failNext(errInjected)
		if err := s.Save(t.Context(), &upd, changes); !errors.Is(err, errInjected) {
			t.Fatalf("Save error = %v, want the injected failure", err)
		}
		checkStored(t, s, a)
		checkStored(t, s, b)
	})

	t.Run("Delete", func(t *testing.T) {
		s, failNext, a, b := setUp(t)
		failNext(errInjected)
		if err := s.Delete(t.Context(), a.ObjectID); !errors.Is(err, errInjected) {
			t.Fatalf("Delete error = %v, want the injected failure", err)
		}
		checkStored(t, s, a)
		checkStored(t, s, b)
		if ok, err := s.NameExists(t.Context(), a.Name); err != nil || !ok {
			t.Errorf("NameExists after a failed Delete = %v, %v, want true", ok, err)
		}
	})
}
