package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

// cursedWeaponsCollection holds the held cursed weapons (Plan 14 M11m,
// decision X4), one document per weapon with Mobius's column names:
//
//	{_id: <itemId>, charId, playerKarma, playerPkKills, nbKills, endTime, itemObjectId}
//
// Batches write it (WriteBatch), in the transaction of the holder's save.
const cursedWeaponsCollection = "cursed_weapons"

var _ storage.CursedWeaponStore = (*Characters)(nil)

type cursedWeaponDoc struct {
	ItemID        int32 `bson:"_id"`
	CharID        int32 `bson:"charId"`
	PlayerKarma   int32 `bson:"playerKarma"`
	PlayerPKKills int32 `bson:"playerPkKills"`
	NbKills       int32 `bson:"nbKills"`
	EndTime       int64 `bson:"endTime"` // Unix milliseconds
	ItemObjectID  int32 `bson:"itemObjectId"`
}

func toCursedWeaponDoc(w storage.CursedWeapon) cursedWeaponDoc {
	return cursedWeaponDoc{ItemID: w.ItemID, CharID: w.CharID, PlayerKarma: w.PlayerKarma, PlayerPKKills: w.PlayerPKKills,
		NbKills: w.NbKills, EndTime: w.EndTime, ItemObjectID: w.ItemObjectID}
}

func (d cursedWeaponDoc) state() storage.CursedWeapon {
	return storage.CursedWeapon{ItemID: d.ItemID, CharID: d.CharID, PlayerKarma: d.PlayerKarma, PlayerPKKills: d.PlayerPKKills,
		NbKills: d.NbKills, EndTime: d.EndTime, ItemObjectID: d.ItemObjectID}
}

func (s *Characters) cursedWeapons() *mongo.Collection {
	return s.store.db.Collection(cursedWeaponsCollection)
}

// LoadCursedWeapons implements storage.CursedWeaponStore.
func (s *Characters) LoadCursedWeapons(ctx context.Context) ([]storage.CursedWeapon, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	cur, err := s.cursedWeapons().Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load cursed weapons: %w", err)
	}
	var docs []cursedWeaponDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load cursed weapons: %w", err)
	}
	var out []storage.CursedWeapon
	for _, d := range docs {
		out = append(out, d.state())
	}
	return out, nil
}

// ItemsByTemplate implements storage.CursedWeaponStore.
func (s *Characters) ItemsByTemplate(ctx context.Context, itemIDs []int32) ([]storage.OwnedItem, error) {
	if len(itemIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	filter := bson.D{{Key: "item_id", Value: bson.D{{Key: "$in", Value: itemIDs}}}}
	cur, err := s.items.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: find items by template: %w", err)
	}
	var docs []itemDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: find items by template: %w", err)
	}
	var out []storage.OwnedItem
	for i := range docs {
		out = append(out, storage.OwnedItem{Owner: docs[i].OwnerID, Item: docs[i].item()})
	}
	return out, nil
}

// applyCursedWeapons writes the cursed weapon part of a batch inside its
// transaction: the deletes, then the rows.
func (s *Characters) applyCursedWeapons(ctx context.Context, b *storage.Batch) error {
	var models []mongo.WriteModel
	for _, id := range b.DeletedCursedWeapons {
		models = append(models, mongo.NewDeleteOneModel().SetFilter(bson.D{{Key: "_id", Value: id}}))
	}
	for _, w := range b.CursedWeapons {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: w.ItemID}}).
			SetReplacement(toCursedWeaponDoc(w)).
			SetUpsert(true))
	}
	if len(models) == 0 {
		return nil
	}
	_, err := s.cursedWeapons().BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
	return err
}
