//go:build integration

package mongostore_test

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/mongostore"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

// mongoURI returns the test deployment, by default the docker compose one.
func mongoURI() string {
	if uri := os.Getenv("L2GO_MONGO_URI"); uri != "" {
		return uri
	}
	return "mongodb://127.0.0.1:27017"
}

// open opens database name and closes it when the test ends.
func open(t *testing.T, name string) *mongostore.Store {
	t.Helper()
	s, err := mongostore.Open(t.Context(), mongoURI(), name)
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
	if len(got.Items) != len(want.Items) {
		t.Fatalf("Items after reopen = %+v, want %+v", got.Items, want.Items)
	}
	for i := range want.Items {
		if got.Items[i] != want.Items[i] {
			t.Errorf("Items[%d] after reopen = %+v, want %+v", i, got.Items[i], want.Items[i])
		}
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
