package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

// The boss collections (Plan 13, decision A8).
const (
	raidBossesCollection      = "raidbosses"
	grandBossesCollection     = "grandbosses"
	bossZonePlayersCollection = "grandboss_zone_players"
	raidPointsCollection      = "raid_points"
)

// Bosses returns the boss store backed by this database.
func (s *Store) Bosses() *Bosses { return &Bosses{store: s} }

// Bosses is a storage.BossStore on four collections, with Mobius's column
// names:
//
//	raidbosses:             {_id: <boss>, respawn_time, currentHP, currentMP}
//	grandbosses:            {_id: <boss>, loc_x, loc_y, loc_z, heading, respawn_time, currentHP, currentMP, status}
//	grandboss_zone_players: {_id: {zone, player_id}}
//	raid_points:            {_id: {charId, boss_id}, points}
//
// Every write runs in a transaction.
type Bosses struct{ store *Store }

var _ storage.BossStore = (*Bosses)(nil)

type raidBossDoc struct {
	BossID      int32   `bson:"_id"`
	RespawnTime int64   `bson:"respawn_time"`
	CurHP       float64 `bson:"currentHP"`
	CurMP       float64 `bson:"currentMP"`
}

type grandBossDoc struct {
	BossID      int32   `bson:"_id"`
	X           int32   `bson:"loc_x"`
	Y           int32   `bson:"loc_y"`
	Z           int32   `bson:"loc_z"`
	Heading     int32   `bson:"heading"`
	RespawnTime int64   `bson:"respawn_time"`
	CurHP       float64 `bson:"currentHP"`
	CurMP       float64 `bson:"currentMP"`
	Status      int32   `bson:"status"`
}

func toGrandBossDoc(g storage.GrandBossState) grandBossDoc {
	return grandBossDoc{BossID: g.BossID, X: g.X, Y: g.Y, Z: g.Z, Heading: g.Heading,
		RespawnTime: g.RespawnTime, CurHP: g.CurHP, CurMP: g.CurMP, Status: g.Status}
}

func (d grandBossDoc) state() storage.GrandBossState {
	return storage.GrandBossState{BossID: d.BossID, X: d.X, Y: d.Y, Z: d.Z, Heading: d.Heading,
		RespawnTime: d.RespawnTime, CurHP: d.CurHP, CurMP: d.CurMP, Status: d.Status}
}

type zonePlayerKey struct {
	Zone   int32 `bson:"zone"`
	Player int32 `bson:"player_id"`
}

type zonePlayerDoc struct {
	ID zonePlayerKey `bson:"_id"`
}

type raidPointsKey struct {
	CharID int32 `bson:"charId"`
	BossID int32 `bson:"boss_id"`
}

type raidPointsDoc struct {
	ID     raidPointsKey `bson:"_id"`
	Points int32         `bson:"points"`
}

func (b *Bosses) coll(name string) *mongo.Collection { return b.store.db.Collection(name) }

// findAll decodes every document of a collection, sorted by sort.
func findAll[T any](ctx context.Context, b *Bosses, coll string, sort bson.D) ([]T, error) {
	ctx, cancel := context.WithTimeout(ctx, b.store.timeout)
	defer cancel()
	cur, err := b.coll(coll).Find(ctx, bson.D{}, options.Find().SetSort(sort))
	if err != nil {
		return nil, err
	}
	var docs []T
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// writeTx runs fn in a write transaction under the store's timeout.
func (b *Bosses) writeTx(ctx context.Context, what string, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, b.store.timeout)
	defer cancel()
	if err := b.store.write(ctx, fn); err != nil {
		return fmt.Errorf("mongostore: %s: %w", what, err)
	}
	return nil
}

// bulk runs the models, if any.
func bulk(ctx context.Context, c *mongo.Collection, models []mongo.WriteModel) error {
	if len(models) == 0 {
		return nil
	}
	_, err := c.BulkWrite(ctx, models)
	return err
}

// LoadRaidBosses implements storage.BossStore.
func (b *Bosses) LoadRaidBosses(ctx context.Context) ([]storage.RaidBossState, error) {
	docs, err := findAll[raidBossDoc](ctx, b, raidBossesCollection, bson.D{{Key: "_id", Value: 1}})
	if err != nil {
		return nil, fmt.Errorf("mongostore: load raid bosses: %w", err)
	}
	var out []storage.RaidBossState
	for _, d := range docs {
		out = append(out, storage.RaidBossState{BossID: d.BossID, RespawnTime: d.RespawnTime, CurHP: d.CurHP, CurMP: d.CurMP})
	}
	return out, nil
}

// SaveRaidBosses implements storage.BossStore.
func (b *Bosses) SaveRaidBosses(ctx context.Context, bosses []storage.RaidBossState) error {
	var models []mongo.WriteModel
	for _, r := range bosses {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: r.BossID}}).
			SetReplacement(raidBossDoc{BossID: r.BossID, RespawnTime: r.RespawnTime, CurHP: r.CurHP, CurMP: r.CurMP}).
			SetUpsert(true))
	}
	return b.writeTx(ctx, "save raid bosses", func(ctx context.Context) error {
		return bulk(ctx, b.coll(raidBossesCollection), models)
	})
}

// SeedGrandBosses implements storage.BossStore.
func (b *Bosses) SeedGrandBosses(ctx context.Context, seeds []storage.GrandBossState) error {
	var models []mongo.WriteModel
	for _, g := range seeds {
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.D{{Key: "_id", Value: g.BossID}}).
			SetUpdate(bson.D{{Key: "$setOnInsert", Value: toGrandBossDoc(g)}}).
			SetUpsert(true))
	}
	return b.writeTx(ctx, "seed grand bosses", func(ctx context.Context) error {
		return bulk(ctx, b.coll(grandBossesCollection), models)
	})
}

// LoadGrandBosses implements storage.BossStore.
func (b *Bosses) LoadGrandBosses(ctx context.Context) ([]storage.GrandBossState, error) {
	docs, err := findAll[grandBossDoc](ctx, b, grandBossesCollection, bson.D{{Key: "_id", Value: 1}})
	if err != nil {
		return nil, fmt.Errorf("mongostore: load grand bosses: %w", err)
	}
	var out []storage.GrandBossState
	for _, d := range docs {
		out = append(out, d.state())
	}
	return out, nil
}

// SaveGrandBossStatus implements storage.BossStore.
func (b *Bosses) SaveGrandBossStatus(ctx context.Context, bossID, status int32) error {
	return b.writeTx(ctx, "save grand boss status", func(ctx context.Context) error {
		_, err := b.coll(grandBossesCollection).UpdateOne(ctx,
			bson.D{{Key: "_id", Value: bossID}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}}}})
		return err
	})
}

// SaveGrandBosses implements storage.BossStore.
func (b *Bosses) SaveGrandBosses(ctx context.Context, bosses []storage.GrandBossState) error {
	var models []mongo.WriteModel
	for _, g := range bosses {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: g.BossID}}).
			SetReplacement(toGrandBossDoc(g)).
			SetUpsert(true))
	}
	return b.writeTx(ctx, "save grand bosses", func(ctx context.Context) error {
		return bulk(ctx, b.coll(grandBossesCollection), models)
	})
}

// LoadBossZonePlayers implements storage.BossStore.
func (b *Bosses) LoadBossZonePlayers(ctx context.Context) ([]storage.BossZonePlayer, error) {
	docs, err := findAll[zonePlayerDoc](ctx, b, bossZonePlayersCollection, bson.D{{Key: "_id.zone", Value: 1}, {Key: "_id.player_id", Value: 1}})
	if err != nil {
		return nil, fmt.Errorf("mongostore: load boss zone players: %w", err)
	}
	var out []storage.BossZonePlayer
	for _, d := range docs {
		out = append(out, storage.BossZonePlayer{Zone: d.ID.Zone, Player: d.ID.Player})
	}
	return out, nil
}

// SaveBossZonePlayers implements storage.BossStore.
func (b *Bosses) SaveBossZonePlayers(ctx context.Context, rows []storage.BossZonePlayer) error {
	seen := make(map[zonePlayerKey]bool, len(rows))
	var docs []any
	for _, r := range rows {
		k := zonePlayerKey{Zone: r.Zone, Player: r.Player}
		if !seen[k] {
			seen[k] = true
			docs = append(docs, zonePlayerDoc{ID: k})
		}
	}
	return b.writeTx(ctx, "save boss zone players", func(ctx context.Context) error {
		c := b.coll(bossZonePlayersCollection)
		if _, err := c.DeleteMany(ctx, bson.D{}); err != nil {
			return err
		}
		if len(docs) == 0 {
			return nil
		}
		_, err := c.InsertMany(ctx, docs)
		return err
	})
}

// LoadRaidPoints implements storage.BossStore.
func (b *Bosses) LoadRaidPoints(ctx context.Context) ([]storage.RaidPoints, error) {
	docs, err := findAll[raidPointsDoc](ctx, b, raidPointsCollection, bson.D{{Key: "_id.charId", Value: 1}, {Key: "_id.boss_id", Value: 1}})
	if err != nil {
		return nil, fmt.Errorf("mongostore: load raid points: %w", err)
	}
	var out []storage.RaidPoints
	for _, d := range docs {
		out = append(out, storage.RaidPoints{CharID: d.ID.CharID, BossID: d.ID.BossID, Points: d.Points})
	}
	return out, nil
}

// SaveRaidPoints implements storage.BossStore.
func (b *Bosses) SaveRaidPoints(ctx context.Context, rows []storage.RaidPoints) error {
	var models []mongo.WriteModel
	for _, r := range rows {
		k := raidPointsKey{CharID: r.CharID, BossID: r.BossID}
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: k}}).
			SetReplacement(raidPointsDoc{ID: k, Points: r.Points}).
			SetUpsert(true))
	}
	return b.writeTx(ctx, "save raid points", func(ctx context.Context) error {
		return bulk(ctx, b.coll(raidPointsCollection), models)
	})
}

// ClearRaidPoints implements storage.BossStore.
func (b *Bosses) ClearRaidPoints(ctx context.Context) error {
	return b.writeTx(ctx, "clear raid points", func(ctx context.Context) error {
		_, err := b.coll(raidPointsCollection).DeleteMany(ctx, bson.D{})
		return err
	})
}
