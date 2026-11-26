package mongostore

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

// fishingChampionshipCollection holds the fishing championship (Plan 14,
// M11g, decision X4).
const fishingChampionshipCollection = "fishing_championship"

// fishingChampionshipID is the _id of the one document of the collection.
const fishingChampionshipID = "championship"

// Fishing returns the fishing championship store backed by this database.
func (s *Store) Fishing() *Fishing {
	return &Fishing{store: s, coll: s.db.Collection(fishingChampionshipCollection)}
}

// Fishing is a storage.FishingStore on the "fishing_championship"
// collection: one document holds the end of the running week (Mobius's
// global variable fishChampionshipEnd) and the fishers with Mobius's column
// names, so a save replaces everything at once:
//
//	{_id: "championship", fishChampionshipEnd: <ms>, fishers: [{player_name, fish_length, rewarded}]}
type Fishing struct {
	store *Store
	coll  *mongo.Collection
}

var _ storage.FishingStore = (*Fishing)(nil)

type fishingChampionshipDoc struct {
	ID      string             `bson:"_id"`
	End     int64              `bson:"fishChampionshipEnd"`
	Fishers []championshipFish `bson:"fishers"`
}

type championshipFish struct {
	Name     string  `bson:"player_name"`
	Length   float64 `bson:"fish_length"`
	Rewarded int32   `bson:"rewarded"`
}

// LoadFishingChampionship implements storage.FishingStore.
func (f *Fishing) LoadFishingChampionship(ctx context.Context) (storage.FishingChampionship, error) {
	ctx, cancel := context.WithTimeout(ctx, f.store.timeout)
	defer cancel()
	var doc fishingChampionshipDoc
	err := f.coll.FindOne(ctx, bson.D{{Key: "_id", Value: fishingChampionshipID}}).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return storage.FishingChampionship{}, nil
	case err != nil:
		return storage.FishingChampionship{}, fmt.Errorf("mongostore: load the fishing championship: %w", err)
	}
	out := storage.FishingChampionship{End: doc.End}
	for _, x := range doc.Fishers {
		out.Fishers = append(out.Fishers, storage.ChampionshipFisher{Name: x.Name, Length: x.Length, Rewarded: x.Rewarded})
	}
	return out, nil
}

// SaveFishingChampionship implements storage.FishingStore.
func (f *Fishing) SaveFishingChampionship(ctx context.Context, c storage.FishingChampionship) error {
	doc := fishingChampionshipDoc{ID: fishingChampionshipID, End: c.End, Fishers: []championshipFish{}}
	for _, x := range c.Fishers {
		doc.Fishers = append(doc.Fishers, championshipFish{Name: x.Name, Length: x.Length, Rewarded: x.Rewarded})
	}
	ctx, cancel := context.WithTimeout(ctx, f.store.timeout)
	defer cancel()
	err := f.store.write(ctx, func(ctx context.Context) error {
		_, err := f.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: fishingChampionshipID}}, doc, options.Replace().SetUpsert(true))
		return err
	})
	if err != nil {
		return fmt.Errorf("mongostore: save the fishing championship: %w", err)
	}
	return nil
}
