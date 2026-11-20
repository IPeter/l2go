//go:build integration

package mongostore

import (
	"maps"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

func TestBosses(t *testing.T) {
	storagetest.TestBossStore(t, func(t *testing.T) (storage.BossStore, func(error)) {
		s, _ := openTestStore(t)
		return s.Bosses(), s.fault.arm
	})
}

// TestBossDocuments checks the stored layout of decision A8: Mobius's
// column names, one document per boss, zone player and raid points row.
func TestBossDocuments(t *testing.T) {
	s, _ := openTestStore(t)
	b := s.Bosses()
	ctx := t.Context()
	if err := b.SaveRaidBosses(ctx, []storage.RaidBossState{{BossID: 25001, RespawnTime: 1795000000000, CurHP: 95986, CurMP: 514}}); err != nil {
		t.Fatalf("SaveRaidBosses: %v", err)
	}
	if err := b.SeedGrandBosses(ctx, []storage.GrandBossState{{BossID: 29001, X: -21610, Y: 181594, Z: -5734, CurHP: 229898.48, CurMP: 667.776}}); err != nil {
		t.Fatalf("SeedGrandBosses: %v", err)
	}
	if err := b.SaveBossZonePlayers(ctx, []storage.BossZonePlayer{{Zone: 12010, Player: 0x10000001}}); err != nil {
		t.Fatalf("SaveBossZonePlayers: %v", err)
	}
	if err := b.SaveRaidPoints(ctx, []storage.RaidPoints{{CharID: 0x10000001, BossID: 25001, Points: 12}}); err != nil {
		t.Fatalf("SaveRaidPoints: %v", err)
	}
	for _, c := range []struct {
		coll string
		want bson.M
	}{
		{raidBossesCollection, bson.M{"_id": int32(25001), "respawn_time": int64(1795000000000), "currentHP": 95986.0, "currentMP": 514.0}},
		{grandBossesCollection, bson.M{"_id": int32(29001), "loc_x": int32(-21610), "loc_y": int32(181594), "loc_z": int32(-5734),
			"heading": int32(0), "respawn_time": int64(0), "currentHP": 229898.48, "currentMP": 667.776, "status": int32(0)}},
	} {
		var got bson.M
		if err := s.db.Collection(c.coll).FindOne(ctx, bson.D{}).Decode(&got); err != nil {
			t.Fatalf("find in %s: %v", c.coll, err)
		}
		if !maps.Equal(got, c.want) {
			t.Errorf("%s document = %v, want %v", c.coll, got, c.want)
		}
	}
	var zone struct {
		ID struct {
			Zone   int32 `bson:"zone"`
			Player int32 `bson:"player_id"`
		} `bson:"_id"`
	}
	if err := s.db.Collection(bossZonePlayersCollection).FindOne(ctx, bson.D{}).Decode(&zone); err != nil || zone.ID.Zone != 12010 || zone.ID.Player != 0x10000001 {
		t.Errorf("zone player document = %+v, %v", zone, err)
	}
	var pts struct {
		ID struct {
			CharID int32 `bson:"charId"`
			BossID int32 `bson:"boss_id"`
		} `bson:"_id"`
		Points int32 `bson:"points"`
	}
	if err := s.db.Collection(raidPointsCollection).FindOne(ctx, bson.D{}).Decode(&pts); err != nil || pts.ID.CharID != 0x10000001 || pts.ID.BossID != 25001 || pts.Points != 12 {
		t.Errorf("raid points document = %+v, %v", pts, err)
	}
}
