//go:build integration

package mongostore

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestOlympiad(t *testing.T) {
	storagetest.TestOlympiadStore(t, func(t *testing.T) (storage.OlympiadStore, func(error)) {
		s, _ := openTestStore(t)
		return s.Olympiad(), s.fault.arm
	})
}

// TestOlympiadDocument checks the stored layout of decision X4: one
// document with Mobius's names.
func TestOlympiadDocument(t *testing.T) {
	s, _ := openTestStore(t)
	ctx := t.Context()
	if err := s.Olympiad().SaveOlympiad(ctx, storage.Olympiad{Cycle: 2, OlympiadEnd: 1796040000123,
		Nobles: []storage.OlympiadNoble{{CharID: 7, ClassID: 88, Name: "Duelist", Points: 21}},
		Heroes: []storage.Hero{{CharID: 7, ClassID: 88, Name: "Duelist", Count: 1, Played: true}}}); err != nil {
		t.Fatalf("SaveOlympiad: %v", err)
	}
	var got struct {
		ID     string `bson:"_id"`
		Cycle  int32  `bson:"current_cycle"`
		End    int64  `bson:"olympiad_end"`
		Nobles []struct {
			CharID int32 `bson:"charId"`
			Points int32 `bson:"olympiad_points"`
		} `bson:"nobles"`
		Heroes []struct {
			Played bool `bson:"played"`
		} `bson:"heroes"`
	}
	if err := s.db.Collection(olympiadCollection).FindOne(ctx, bson.D{}).Decode(&got); err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.ID != olympiadID || got.Cycle != 2 || got.End != 1796040000123 || len(got.Nobles) != 1 || got.Nobles[0].Points != 21 || !got.Heroes[0].Played {
		t.Errorf("document = %+v", got)
	}
}
