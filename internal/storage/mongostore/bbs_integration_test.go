//go:build integration

package mongostore

import (
	"maps"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestBBS(t *testing.T) {
	storagetest.TestBBSStore(t, func(t *testing.T) (storage.BBSStore, func(error)) {
		s, _ := openTestStore(t)
		return s.BBS(), s.fault.arm
	})
}

// TestBBSDocuments checks the stored layouts: Mobius's column names, the ID
// as the key.
func TestBBSDocuments(t *testing.T) {
	s, _ := openTestStore(t)
	ctx := t.Context()
	b := s.BBS()
	mail := storage.BBSMail{ID: 4, ReceiverID: 2, SenderID: 1, SenderName: "Alice", Location: "inbox", Recipients: "Bob",
		Subject: "Hi", Message: "Text", SentDate: 1_796_000_000_123, Unread: true}
	if err := b.SaveBBSMails(ctx, []storage.BBSMail{mail}); err != nil {
		t.Fatalf("SaveBBSMails: %v", err)
	}
	var got bson.M
	if err := s.db.Collection(bbsMailCollection).FindOne(ctx, bson.D{}).Decode(&got); err != nil {
		t.Fatalf("find mail: %v", err)
	}
	want := bson.M{"_id": int32(4), "receiver_id": int32(2), "sender_id": int32(1), "sender_name": "Alice", "location": "inbox",
		"recipients": "Bob", "subject": "Hi", "message": "Text", "sent_date": int64(1_796_000_000_123), "is_unread": true}
	if !maps.Equal(got, want) {
		t.Errorf("mail document = %v, want %v", got, want)
	}
	fav := storage.BBSFavorite{ID: 9, PlayerID: 7, Title: "Home", Bypass: "_bbshome", AddDate: 1_796_000_000_000}
	if err := b.SaveBBSFavorite(ctx, fav); err != nil {
		t.Fatalf("SaveBBSFavorite: %v", err)
	}
	got = nil
	if err := s.db.Collection(bbsFavoritesCollection).FindOne(ctx, bson.D{}).Decode(&got); err != nil {
		t.Fatalf("find favorite: %v", err)
	}
	want = bson.M{"_id": int32(9), "playerId": int32(7), "favTitle": "Home", "favBypass": "_bbshome", "favAddDate": int64(1_796_000_000_000)}
	if !maps.Equal(got, want) {
		t.Errorf("favorite document = %v, want %v", got, want)
	}
}
