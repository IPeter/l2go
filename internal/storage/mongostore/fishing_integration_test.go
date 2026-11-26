//go:build integration

package mongostore

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestFishing(t *testing.T) {
	storagetest.TestFishingStore(t, func(t *testing.T) (storage.FishingStore, func(error)) {
		s, _ := openTestStore(t)
		return s.Fishing(), s.fault.arm
	})
}

// TestFishingDocument checks the stored layout of decision X4: one
// document with Mobius's names.
func TestFishingDocument(t *testing.T) {
	s, _ := openTestStore(t)
	ctx := t.Context()
	if err := s.Fishing().SaveFishingChampionship(ctx, storage.FishingChampionship{End: 1796000400123,
		Fishers: []storage.ChampionshipFisher{{Name: "Bob", Length: 85.25, Rewarded: 1}}}); err != nil {
		t.Fatalf("SaveFishingChampionship: %v", err)
	}
	var got struct {
		ID      string `bson:"_id"`
		End     int64  `bson:"fishChampionshipEnd"`
		Fishers []struct {
			Name     string  `bson:"player_name"`
			Length   float64 `bson:"fish_length"`
			Rewarded int32   `bson:"rewarded"`
		} `bson:"fishers"`
	}
	if err := s.db.Collection(fishingChampionshipCollection).FindOne(ctx, bson.D{}).Decode(&got); err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ID != fishingChampionshipID || got.End != 1796000400123 || len(got.Fishers) != 1 || got.Fishers[0].Name != "Bob" || got.Fishers[0].Rewarded != 1 {
		t.Errorf("document = %+v", got)
	}
}
