package storagetest

import (
	"errors"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// TestBBSStore runs the BBSStore conformance tests (Plan 14 M11o). newStore
// must return an empty store for each call, and a function that makes the
// store's next write fail with the given error after it did all its work,
// just before it commits.
func TestBBSStore(t *testing.T, newStore func(t *testing.T) (s storage.BBSStore, failNext func(error))) {
	inbox := storage.BBSMail{ID: 1, ReceiverID: 0x10000002, SenderID: 0x10000001, SenderName: "Alice", Location: "inbox",
		Recipients: "Bob;Cid", Subject: "Hi", Message: "Line<br1>two", SentDate: 1_796_000_000_123, Unread: true}
	inbox2 := inbox
	inbox2.ID, inbox2.ReceiverID = 2, 0x10000003
	sent := inbox
	sent.ID, sent.ReceiverID, sent.Location, sent.Unread = 3, 0x10000001, "sentbox", false

	loadMails := func(t *testing.T, s storage.BBSStore) []storage.BBSMail {
		t.Helper()
		got, err := s.LoadBBSMails(t.Context())
		if err != nil {
			t.Fatalf("LoadBBSMails: %v", err)
		}
		return got
	}
	loadFavs := func(t *testing.T, s storage.BBSStore) []storage.BBSFavorite {
		t.Helper()
		got, err := s.LoadBBSFavorites(t.Context())
		if err != nil {
			t.Fatalf("LoadBBSFavorites: %v", err)
		}
		return got
	}
	must := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("Empty", func(t *testing.T) {
		s, _ := newStore(t)
		if got := loadMails(t, s); got != nil {
			t.Errorf("LoadBBSMails = %+v, want nil", got)
		}
		if got := loadFavs(t, s); got != nil {
			t.Errorf("LoadBBSFavorites = %+v, want nil", got)
		}
		must(t, s.DeleteBBSMail(t.Context(), 5))
		must(t, s.DeleteBBSFavorite(t.Context(), 1, 5))
	})

	t.Run("Mails", func(t *testing.T) {
		s, _ := newStore(t)
		must(t, s.SaveBBSMails(t.Context(), []storage.BBSMail{sent, inbox, inbox2}))
		if got, want := loadMails(t, s), []storage.BBSMail{inbox, inbox2, sent}; !slices.Equal(got, want) {
			t.Fatalf("LoadBBSMails = %+v, want %+v (by ID)", got, want)
		}
		read := inbox
		read.Unread, read.Location = false, "archive"
		must(t, s.SaveBBSMails(t.Context(), []storage.BBSMail{read}))
		must(t, s.DeleteBBSMail(t.Context(), 2))
		if got, want := loadMails(t, s), []storage.BBSMail{read, sent}; !slices.Equal(got, want) {
			t.Errorf("after a change and a delete: %+v, want %+v", got, want)
		}
	})

	t.Run("Favorites", func(t *testing.T) {
		s, _ := newStore(t)
		home := storage.BBSFavorite{ID: 1, PlayerID: 7, Title: "Home", Bypass: "_bbshome", AddDate: 1_796_000_000_000}
		region := storage.BBSFavorite{ID: 2, PlayerID: 7, Title: "Region>", Bypass: "_bbsloc;3", AddDate: 1_796_000_001_000}
		other := storage.BBSFavorite{ID: 3, PlayerID: 8, Title: "Clan", Bypass: "_bbsclan", AddDate: 1_796_000_002_000}
		for _, f := range []storage.BBSFavorite{region, other, home} {
			must(t, s.SaveBBSFavorite(t.Context(), f))
		}
		if got, want := loadFavs(t, s), []storage.BBSFavorite{home, region, other}; !slices.Equal(got, want) {
			t.Fatalf("LoadBBSFavorites = %+v, want %+v", got, want)
		}
		// Another player's favorite stays.
		must(t, s.DeleteBBSFavorite(t.Context(), 7, 3))
		must(t, s.DeleteBBSFavorite(t.Context(), 7, 1))
		if got, want := loadFavs(t, s), []storage.BBSFavorite{region, other}; !slices.Equal(got, want) {
			t.Errorf("after deletes: %+v, want %+v", got, want)
		}
	})

	t.Run("FailedWriteChangesNothing", func(t *testing.T) {
		s, failNext := newStore(t)
		must(t, s.SaveBBSMails(t.Context(), []storage.BBSMail{inbox}))
		boom := errors.New("boom")
		failNext(boom)
		if err := s.SaveBBSMails(t.Context(), []storage.BBSMail{inbox2, sent}); !errors.Is(err, boom) {
			t.Fatalf("SaveBBSMails = %v, want %v", err, boom)
		}
		if got := loadMails(t, s); !slices.Equal(got, []storage.BBSMail{inbox}) {
			t.Errorf("after a failed write: %+v", got)
		}
		failNext(boom)
		if err := s.DeleteBBSMail(t.Context(), 1); !errors.Is(err, boom) {
			t.Fatalf("DeleteBBSMail = %v, want %v", err, boom)
		}
		failNext(boom)
		if err := s.SaveBBSFavorite(t.Context(), storage.BBSFavorite{ID: 1, PlayerID: 7}); !errors.Is(err, boom) {
			t.Fatalf("SaveBBSFavorite = %v, want %v", err, boom)
		}
		if len(loadMails(t, s)) != 1 || loadFavs(t, s) != nil {
			t.Error("a failed write changed the store")
		}
	})
}
