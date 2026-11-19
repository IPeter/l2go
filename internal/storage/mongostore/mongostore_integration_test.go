//go:build integration

package mongostore_test

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/mongostore"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

// mongoURI returns the test deployment, by default the docker compose one: a
// one-node replica set whose member name, mongo, only resolves inside
// compose, so the host connects directly.
func mongoURI() string {
	if uri := os.Getenv("L2GO_MONGO_URI"); uri != "" {
		return uri
	}
	return "mongodb://127.0.0.1:27017/?directConnection=true"
}

// open opens database name and closes it when the test ends.
func open(t *testing.T, name string) *mongostore.Store {
	t.Helper()
	s, err := mongostore.Open(t.Context(), mongoURI(), name, mongostore.WithLogger(slog.New(slog.DiscardHandler)))
	if err != nil {
		t.Fatalf("Open: %v (is MongoDB running? try: docker compose up -d --wait mongo)", err)
	}
	t.Cleanup(func() {
		if err := s.Close(context.Background()); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return s
}

// openFresh opens a new, uniquely named database that is dropped at the end.
func openFresh(t *testing.T) (*mongostore.Store, string) {
	t.Helper()
	name := "l2go_test_" + rand.Text()[:12]
	s := open(t, name)
	t.Cleanup(func() {
		if err := s.Drop(context.Background()); err != nil {
			t.Errorf("Drop: %v", err)
		}
	})
	return s, name
}

func TestAccounts(t *testing.T) {
	storagetest.TestAccountStore(t, func(t *testing.T) storage.AccountStore {
		s, _ := openFresh(t)
		return s.Accounts()
	})
}

func TestCharacters(t *testing.T) {
	storagetest.TestCharacterStore(t, func(t *testing.T) storage.CharacterStore {
		s, _ := openFresh(t)
		return s.Characters()
	})
}

func TestCharacterSaver(t *testing.T) {
	storagetest.TestCharacterSaver(t, func(t *testing.T) storagetest.SaverStore {
		s, _ := openFresh(t)
		return s.Characters()
	})
}

func TestIDs(t *testing.T) {
	storagetest.TestIDAllocator(t, func(t *testing.T) storage.IDAllocator {
		s, _ := openFresh(t)
		return s.IDs()
	})
}

// TestCharactersSurviveReopen simulates a server restart: one connection
// stores a character and closes, a new one must find it and keep going.
func TestCharactersSurviveReopen(t *testing.T) {
	_, name := openFresh(t) // owns the database and drops it at the end
	want, used := firstRun(t, name)

	s := open(t, name)
	got, err := s.Characters().Get(t.Context(), want.ObjectID)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got.Name != want.Name || got.Account != want.Account || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("Get after reopen = %+v, want %+v", got, want)
	}
	if wantItems := storagetest.StoredItems(want.Items); !slices.Equal(got.Items, wantItems) {
		t.Errorf("Items after reopen = %+v, want %+v", got.Items, wantItems)
	}
	list, err := s.Characters().ListByAccount(t.Context(), want.Account)
	if err != nil {
		t.Fatalf("ListByAccount after reopen: %v", err)
	}
	if len(list) != 1 || list[0].ObjectID != want.ObjectID {
		t.Errorf("ListByAccount after reopen = %+v, want the stored character", list)
	}

	// The unique name index still applies.
	dup := storage.Character{ObjectID: 1, Account: "other", Name: "ELROND"}
	if err := s.Characters().Create(t.Context(), &dup); !errors.Is(err, storage.ErrExists) {
		t.Errorf("Create with a taken name after reopen: error = %v, want ErrExists", err)
	}

	// The ID counter continues where it stopped.
	next, err := s.IDs().NextIDs(t.Context(), 5)
	if err != nil {
		t.Fatalf("NextIDs after reopen: %v", err)
	}
	for _, id := range next {
		if id <= used {
			t.Errorf("NextIDs after reopen = %#x, want IDs after %#x", next, used)
			break
		}
	}
}

// firstRun opens database name, stores a character with IDs from the
// counter, and closes the connection. It returns the character and the
// highest ID it was given.
func firstRun(t *testing.T, name string) (storage.Character, int32) {
	t.Helper()
	s, err := mongostore.Open(t.Context(), mongoURI(), name)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := s.Close(context.Background()); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	ids, err := s.IDs().NextIDs(t.Context(), 3)
	if err != nil {
		t.Fatalf("NextIDs: %v", err)
	}
	c := storage.Character{
		ObjectID: ids[0],
		Account:  "alice",
		Name:     "Elrond",
		Level:    1,
		Items: []storage.Item{
			{ObjectID: ids[1], ItemID: 2369, Count: 1, Slot: storage.PaperdollRHand},
			{ObjectID: ids[2], ItemID: 57, Count: 100, Slot: storage.PaperdollNone},
		},
	}
	if err := s.Characters().Create(t.Context(), &c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return c, ids[2]
}

// TestCharacterChangesSurviveReopen saves a character's state, marks one
// character for deletion and deletes another, then checks all three after a
// new connection.
func TestCharacterChangesSurviveReopen(t *testing.T) {
	_, name := openFresh(t)
	deleteAt := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	lastAccess := time.Date(2026, time.November, 19, 18, 30, 0, 0, time.UTC)

	first := open(t, name)
	chars := first.Characters()
	for _, c := range []storage.Character{
		{ObjectID: 1, Account: "alice", Name: "Saved", Level: 1, X: 1, Y: 2, Z: 3},
		{ObjectID: 2, Account: "alice", Name: "Marked", Level: 1},
		{ObjectID: 3, Account: "alice", Name: "Gone", Level: 1},
	} {
		if err := chars.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create(%s): %v", c.Name, err)
		}
	}
	saved, err := chars.Get(t.Context(), 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	saved.X, saved.Y, saved.Z, saved.Heading = -71338, 258271, -3104, 16384
	saved.CurHP, saved.LastAccess = 42.5, lastAccess
	saved.Items = []storage.Item{{ObjectID: 10, ItemID: 57, Count: 1000, Slot: storage.PaperdollNone}}
	if err := chars.Update(t.Context(), &saved); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := chars.SetDeleteTime(t.Context(), 2, deleteAt); err != nil {
		t.Fatalf("SetDeleteTime: %v", err)
	}
	if err := chars.Delete(t.Context(), 3); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	second := open(t, name)
	list, err := second.Characters().ListByAccount(t.Context(), "alice")
	if err != nil {
		t.Fatalf("ListByAccount after reopen: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListByAccount after reopen = %+v, want Saved and Marked", list)
	}
	got := list[0]
	if got.X != -71338 || got.Y != 258271 || got.Z != -3104 || got.Heading != 16384 || got.CurHP != 42.5 ||
		!got.LastAccess.Equal(lastAccess) || len(got.Items) != 1 || got.Items[0].Count != 1000 {
		t.Errorf("saved character after reopen = %+v", got)
	}
	if !list[1].DeleteTime.Equal(deleteAt) {
		t.Errorf("delete time after reopen = %v, want %v", list[1].DeleteTime, deleteAt)
	}
	// The deleted name is free again under the unique index.
	again := storage.Character{ObjectID: 4, Account: "bob", Name: "GONE"}
	if err := second.Characters().Create(t.Context(), &again); err != nil {
		t.Errorf("Create with a deleted name: %v", err)
	}
}

func TestOpenTwiceKeepsIndex(t *testing.T) {
	_, name := openFresh(t)
	// Opening the same database again must not fail on the existing index.
	s := open(t, name)
	if err := s.Accounts().Create(t.Context(), storage.Account{Username: "eve"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestOpenUnreachable(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := mongostore.Open(ctx, "mongodb://127.0.0.1:1", "l2go_unreachable"); err == nil {
		t.Fatal("Open succeeded against an unreachable deployment")
	}
}
