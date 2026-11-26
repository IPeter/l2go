//go:build integration

package mongostore

import (
	"maps"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestCursedWeapons(t *testing.T) {
	storagetest.TestCursedWeaponStore(t, func(t *testing.T) (storagetest.CursedWeaponStores, func(error)) {
		s, _ := openTestStore(t)
		chars := s.Characters()
		return storagetest.CursedWeaponStores{Characters: chars, Batches: chars, CursedWeapons: chars}, s.fault.arm
	})
}

// TestCursedWeaponDocument checks the stored layout: Mobius's column names,
// the item ID as the key.
func TestCursedWeaponDocument(t *testing.T) {
	s, _ := openTestStore(t)
	chars := s.Characters()
	w := storage.CursedWeapon{ItemID: 8190, CharID: 0x10000001, PlayerKarma: 720, PlayerPKKills: 3, NbKills: 12,
		EndTime: 1_795_000_000_123, ItemObjectID: 0x10000005}
	if err := chars.WriteBatch(t.Context(), storage.Batch{CursedWeapons: []storage.CursedWeapon{w}}); err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}
	var got bson.M
	if err := s.db.Collection(cursedWeaponsCollection).FindOne(t.Context(), bson.D{}).Decode(&got); err != nil {
		t.Fatalf("find: %v", err)
	}
	want := bson.M{"_id": int32(8190), "charId": int32(0x10000001), "playerKarma": int32(720), "playerPkKills": int32(3),
		"nbKills": int32(12), "endTime": int64(1_795_000_000_123), "itemObjectId": int32(0x10000005)}
	if !maps.Equal(got, want) {
		t.Errorf("document = %v, want %v", got, want)
	}
}
