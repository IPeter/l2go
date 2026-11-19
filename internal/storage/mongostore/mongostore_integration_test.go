//go:build integration

package mongostore_test

import (
	"context"
	"crypto/rand"
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
