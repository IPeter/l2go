package storagetest

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// baseTime is a creation time with millisecond precision, as stored.
var baseTime = time.Date(2026, time.November, 19, 12, 0, 0, 123_000_000, time.UTC)

// sampleCharacter returns a character with every field set to a distinct
// value, so a field that is stored under the wrong name or not at all shows up
// as a difference.
func sampleCharacter(objectID int32, account, name string, created time.Time) storage.Character {
	return storage.Character{
		ObjectID:    objectID,
		Account:     account,
		Name:        name,
		Race:        4,
		Sex:         1,
		ClassID:     54,
		BaseClassID: 53,
		HairStyle:   5,
		HairColor:   3,
		Face:        2,
		Level:       7,
		Exp:         1_234_567_890_123,
		SP:          4321,
		Karma:       11,
		PvPKills:    12,
		PKKills:     13,
		ClanID:      14,
		CurHP:       80.5,
		MaxHP:       81.25,
		CurMP:       30.5,
		MaxMP:       31.75,
		CurCP:       32.5,
		MaxCP:       33.125,
		X:           -71338,
		Y:           258271,
		Z:           -3104,
		Heading:     16384,
		AccessLevel: storage.AccessLevelAdmin,
		DeleteTime:  created.Add(7 * 24 * time.Hour),
		CreatedAt:   created,
		LastAccess:  created.Add(time.Hour),
		Items: []storage.Item{
			{ObjectID: objectID + 1, ItemID: 2369, Count: 1, Slot: storage.PaperdollRHand},
			{ObjectID: objectID + 2, ItemID: 1146, Count: 1, Enchant: 3, Slot: storage.PaperdollChest},
			{ObjectID: objectID + 3, ItemID: 57, Count: 5_000_000_000, Slot: storage.PaperdollNone},
		},
	}
}

// diffCharacter describes how got differs from want, or returns "" if they
// are equal. Times are compared with time.Equal, and a nil Items slice equals
// an empty one.
func diffCharacter(got, want storage.Character) string {
	var diffs []string
	for _, f := range []struct {
		name      string
		got, want time.Time
	}{
		{"DeleteTime", got.DeleteTime, want.DeleteTime},
		{"CreatedAt", got.CreatedAt, want.CreatedAt},
		{"LastAccess", got.LastAccess, want.LastAccess},
	} {
		if !f.got.Equal(f.want) {
			diffs = append(diffs, fmt.Sprintf("%s = %v, want %v", f.name, f.got, f.want))
		}
	}
	if !slices.Equal(got.Items, want.Items) {
		diffs = append(diffs, fmt.Sprintf("Items = %+v, want %+v", got.Items, want.Items))
	}
	got.DeleteTime, got.CreatedAt, got.LastAccess, got.Items = time.Time{}, time.Time{}, time.Time{}, nil
	want.DeleteTime, want.CreatedAt, want.LastAccess, want.Items = time.Time{}, time.Time{}, time.Time{}, nil
	if !reflect.DeepEqual(got, want) {
		diffs = append(diffs, fmt.Sprintf("other fields = %+v, want %+v", got, want))
	}
	return strings.Join(diffs, "; ")
}

// TestCharacterStore runs the CharacterStore conformance tests. newStore must
// return an empty store for each call.
func TestCharacterStore(t *testing.T, newStore func(t *testing.T) storage.CharacterStore) {
	t.Run("CreateThenGet", func(t *testing.T) {
		s := newStore(t)
		want := sampleCharacter(100, "alice", "Gimli", baseTime)
		c := want
		c.Items = slices.Clone(want.Items)
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if d := diffCharacter(c, want); d != "" {
			t.Errorf("Create changed its argument: %s", d)
		}
		got, err := s.Get(t.Context(), 100)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if d := diffCharacter(got, want); d != "" {
			t.Errorf("Get: %s", d)
		}
	})

	t.Run("CreateSetsCreatedAt", func(t *testing.T) {
		s := newStore(t)
		before := time.Now().Add(-time.Second)
		c := sampleCharacter(100, "alice", "Gimli", time.Time{})
		c.DeleteTime, c.LastAccess = time.Time{}, time.Time{}
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if c.CreatedAt.Before(before) || c.CreatedAt.After(time.Now().Add(time.Second)) {
			t.Errorf("CreatedAt = %v, want about now", c.CreatedAt)
		}
		got, err := s.Get(t.Context(), 100)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if d := diffCharacter(got, c); d != "" {
			t.Errorf("Get: %s", d)
		}
		if !got.DeleteTime.IsZero() || !got.LastAccess.IsZero() {
			t.Errorf("DeleteTime = %v, LastAccess = %v, want both zero", got.DeleteTime, got.LastAccess)
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		s := newStore(t)
		if _, err := s.Get(t.Context(), 12345); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get error = %v, want ErrNotFound", err)
		}
	})

	t.Run("ListAndCount", func(t *testing.T) {
		s := newStore(t)
		newest := sampleCharacter(300, "alice", "Newest", baseTime.Add(2*time.Hour))
		oldest := sampleCharacter(100, "alice", "Oldest", baseTime)
		middle := sampleCharacter(200, "alice", "Middle", baseTime.Add(time.Hour))
		other := sampleCharacter(400, "bob", "Other", baseTime)
		// Created out of order: the list must follow the creation time.
		for _, c := range []storage.Character{newest, oldest, other, middle} {
			if err := s.Create(t.Context(), &c); err != nil {
				t.Fatalf("Create(%s): %v", c.Name, err)
			}
		}

		got, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		want := []storage.Character{oldest, middle, newest}
		if len(got) != len(want) {
			t.Fatalf("ListByAccount returned %d characters, want %d", len(got), len(want))
		}
		for i := range want {
			if d := diffCharacter(got[i], want[i]); d != "" {
				t.Errorf("ListByAccount[%d]: %s", i, d)
			}
		}

		for account, want := range map[string]int{"alice": 3, "bob": 1, "nobody": 0, "ALICE": 0} {
			n, err := s.CountByAccount(t.Context(), account)
			if err != nil {
				t.Fatalf("CountByAccount(%s): %v", account, err)
			}
			if n != want {
				t.Errorf("CountByAccount(%s) = %d, want %d", account, n, want)
			}
		}

		none, err := s.ListByAccount(t.Context(), "nobody")
		if err != nil {
			t.Fatalf("ListByAccount(nobody): %v", err)
		}
		if len(none) != 0 {
			t.Errorf("ListByAccount(nobody) = %+v, want empty", none)
		}
	})

	t.Run("NameExistsIgnoresCase", func(t *testing.T) {
		s := newStore(t)
		if ok, err := s.NameExists(t.Context(), "Gandalf"); err != nil || ok {
			t.Fatalf("NameExists on empty store = %v, %v, want false, nil", ok, err)
		}
		c := sampleCharacter(100, "alice", "Gandalf", baseTime)
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		for name, want := range map[string]bool{"Gandalf": true, "gandalf": true, "GANDALF": true, "Gandal": false, "Gandalf2": false} {
			ok, err := s.NameExists(t.Context(), name)
			if err != nil {
				t.Fatalf("NameExists(%s): %v", name, err)
			}
			if ok != want {
				t.Errorf("NameExists(%s) = %v, want %v", name, ok, want)
			}
		}
	})

	t.Run("DuplicateNameIgnoringCase", func(t *testing.T) {
		s := newStore(t)
		first := sampleCharacter(100, "alice", "Frodo", baseTime)
		if err := s.Create(t.Context(), &first); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		second := sampleCharacter(200, "bob", "fRODO", baseTime)
		if err := s.Create(t.Context(), &second); !errors.Is(err, storage.ErrExists) {
			t.Fatalf("second Create error = %v, want ErrExists", err)
		}
		if _, err := s.Get(t.Context(), 200); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get(rejected character) error = %v, want ErrNotFound", err)
		}
		if n, err := s.CountByAccount(t.Context(), "bob"); err != nil || n != 0 {
			t.Errorf("CountByAccount(bob) = %d, %v, want 0, nil", n, err)
		}
		got, err := s.Get(t.Context(), 100)
		if err != nil {
			t.Fatalf("Get(first): %v", err)
		}
		if got.Name != "Frodo" {
			t.Errorf("Name = %q, want the first character kept", got.Name)
		}
	})

	t.Run("DuplicateObjectID", func(t *testing.T) {
		s := newStore(t)
		first := sampleCharacter(100, "alice", "Merry", baseTime)
		if err := s.Create(t.Context(), &first); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		second := sampleCharacter(100, "alice", "Pippin", baseTime)
		if err := s.Create(t.Context(), &second); !errors.Is(err, storage.ErrExists) {
			t.Fatalf("second Create error = %v, want ErrExists", err)
		}
		if ok, err := s.NameExists(t.Context(), "Pippin"); err != nil || ok {
			t.Errorf("NameExists(rejected name) = %v, %v, want false, nil", ok, err)
		}
	})

	t.Run("ConcurrentCreateSameName", func(t *testing.T) {
		s := newStore(t)
		const n = 8
		names := []string{"Samwise", "SAMWISE", "samwise"}
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			created int
		)
		for i := range n {
			wg.Go(func() {
				// Different object IDs, so only the name can collide.
				c := sampleCharacter(int32(100*(i+1)), "alice", names[i%len(names)], baseTime)
				err := s.Create(t.Context(), &c)
				switch {
				case err == nil:
					mu.Lock()
					created++
					mu.Unlock()
				case !errors.Is(err, storage.ErrExists):
					t.Errorf("Create: %v", err)
				}
			})
		}
		wg.Wait()
		if created != 1 {
			t.Errorf("%d creates succeeded, want exactly 1", created)
		}
		if got, err := s.CountByAccount(t.Context(), "alice"); err != nil || got != 1 {
			t.Errorf("CountByAccount = %d, %v, want 1, nil", got, err)
		}
	})

	t.Run("ItemsRoundTrip", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(1000, "alice", "Legolas", baseTime)
		c.Items = nil
		for slot := range storage.PaperdollTotalSlots {
			c.Items = append(c.Items, storage.Item{
				ObjectID: 1001 + int32(slot),
				ItemID:   2000 + int32(slot),
				Count:    1,
				Enchant:  int32(slot % 4),
				Slot:     storage.PaperdollSlot(slot),
			})
		}
		c.Items = append(c.Items,
			storage.Item{ObjectID: 1100, ItemID: 57, Count: 2_000_000_000_000, Slot: storage.PaperdollNone},
			storage.Item{ObjectID: 1101, ItemID: 1835, Count: 500, Slot: storage.PaperdollNone},
		)
		want := slices.Clone(c.Items)
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := s.Get(t.Context(), 1000)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if !slices.Equal(got.Items, want) {
			t.Errorf("Get items = %+v, want %+v", got.Items, want)
		}
		list, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		if len(list) != 1 || !slices.Equal(list[0].Items, want) {
			t.Errorf("ListByAccount = %+v, want one character with items %+v", list, want)
		}
	})

	t.Run("NoItems", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Boromir", baseTime)
		c.Items = nil
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		got, err := s.Get(t.Context(), 100)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if len(got.Items) != 0 {
			t.Errorf("Items = %+v, want none", got.Items)
		}
	})

	t.Run("ResultsAreCopies", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Aragorn", baseTime)
		want := sampleCharacter(100, "alice", "Aragorn", baseTime)
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		c.Items[0].Count = 99
		got, err := s.Get(t.Context(), 100)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		got.Items[1].Count = 98
		list, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		list[0].Items[2].Count = 97
		got, err = s.Get(t.Context(), 100)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if d := diffCharacter(got, want); d != "" {
			t.Errorf("stored character changed through a caller's slice: %s", d)
		}
	})
}

// TestIDAllocator runs the IDAllocator conformance tests. newAllocator must
// return a fresh allocator, one that has handed out no IDs, for each call.
func TestIDAllocator(t *testing.T, newAllocator func(t *testing.T) storage.IDAllocator) {
	t.Run("StartsAtFirstObjectID", func(t *testing.T) {
		a := newAllocator(t)
		first, err := a.NextID(t.Context())
		if err != nil {
			t.Fatalf("NextID: %v", err)
		}
		if first != storage.FirstObjectID {
			t.Errorf("first NextID = %#x, want %#x", first, storage.FirstObjectID)
		}
		ids, err := a.NextIDs(t.Context(), 3)
		if err != nil {
			t.Fatalf("NextIDs: %v", err)
		}
		if len(ids) != 3 {
			t.Fatalf("NextIDs(3) returned %d IDs", len(ids))
		}
		seen := map[int32]bool{first: true}
		for _, id := range ids {
			if id < storage.FirstObjectID || seen[id] {
				t.Errorf("NextIDs(3) = %#x after %#x, want 3 new IDs, none below %#x", ids, first, storage.FirstObjectID)
				break
			}
			seen[id] = true
		}
	})

	t.Run("InvalidCount", func(t *testing.T) {
		a := newAllocator(t)
		for _, n := range []int{0, -1} {
			if ids, err := a.NextIDs(t.Context(), n); err == nil {
				t.Errorf("NextIDs(%d) = %v, want an error", n, ids)
			}
		}
	})

	t.Run("ConcurrentUnique", func(t *testing.T) {
		a := newAllocator(t)
		const (
			workers = 16
			calls   = 25
		)
		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			seen = make(map[int32]bool)
			dups []int32
		)
		for w := range workers {
			wg.Go(func() {
				for i := range calls {
					var ids []int32
					if (w+i)%3 == 0 {
						id, err := a.NextID(t.Context())
						if err != nil {
							t.Errorf("NextID: %v", err)
							return
						}
						ids = []int32{id}
					} else {
						n := 1 + (w+i)%7
						var err error
						ids, err = a.NextIDs(t.Context(), n)
						if err != nil {
							t.Errorf("NextIDs(%d): %v", n, err)
							return
						}
						if len(ids) != n {
							t.Errorf("NextIDs(%d) returned %d IDs", n, len(ids))
						}
					}
					mu.Lock()
					for _, id := range ids {
						if seen[id] {
							dups = append(dups, id)
						}
						seen[id] = true
					}
					mu.Unlock()
				}
			})
		}
		wg.Wait()
		if len(dups) > 0 {
			t.Errorf("IDs handed out more than once: %#x", dups)
		}
		lowest := int32(-1)
		for id := range seen {
			if lowest == -1 || id < lowest {
				lowest = id
			}
		}
		if lowest != storage.FirstObjectID {
			t.Errorf("lowest ID = %#x, want %#x", lowest, storage.FirstObjectID)
		}
	})
}
