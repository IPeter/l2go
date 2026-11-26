package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

// manorCollection holds the manor (Plan 14 M11k, decision X4): one document
// per castle with Mobius's castle_manor_production and castle_manor_procure
// rows of the running and the next period as arrays, and the manor's mode
// in the document with _id 0:
//
//	{_id: <castleId>, production: [{seed_id, amount, price, start_amount}],
//	 productionNext: [...], procure: [{crop_id, amount, price, start_amount,
//	 reward_type}], procureNext: [...]}
//	{_id: 0, mode, nextChange}
//
// Batches write it (WriteBatch), in the transaction of the castles and the
// clan warehouse items a manor change moves.
const manorCollection = "manor"

// manorStateID is the _id of the mode's document.
const manorStateID = 0

var _ storage.ManorStore = (*Characters)(nil)

type seedProductionDoc struct {
	SeedID      int32 `bson:"seed_id"`
	Amount      int64 `bson:"amount"`
	Price       int64 `bson:"price"`
	StartAmount int64 `bson:"start_amount"`
}

type cropProcureDoc struct {
	CropID      int32 `bson:"crop_id"`
	Amount      int64 `bson:"amount"`
	Price       int64 `bson:"price"`
	StartAmount int64 `bson:"start_amount"`
	RewardType  int32 `bson:"reward_type"`
}

// manorDoc is a castle's document, or the mode's (_id 0).
type manorDoc struct {
	ID             int32               `bson:"_id"`
	Mode           int32               `bson:"mode,omitempty"`
	NextChange     int64               `bson:"nextChange,omitempty"`
	Production     []seedProductionDoc `bson:"production,omitempty"`
	ProductionNext []seedProductionDoc `bson:"productionNext,omitempty"`
	Procure        []cropProcureDoc    `bson:"procure,omitempty"`
	ProcureNext    []cropProcureDoc    `bson:"procureNext,omitempty"`
}

func toManorDoc(c storage.ManorCastle) manorDoc {
	d := manorDoc{ID: c.CastleID}
	conv := func(list []storage.SeedProduction) []seedProductionDoc {
		var out []seedProductionDoc
		for _, p := range list {
			out = append(out, seedProductionDoc(p))
		}
		return out
	}
	convCrops := func(list []storage.CropProcure) []cropProcureDoc {
		var out []cropProcureDoc
		for _, p := range list {
			out = append(out, cropProcureDoc(p))
		}
		return out
	}
	d.Production, d.ProductionNext = conv(c.Production), conv(c.ProductionNext)
	d.Procure, d.ProcureNext = convCrops(c.Procure), convCrops(c.ProcureNext)
	return d
}

func (d manorDoc) castle() storage.ManorCastle {
	c := storage.ManorCastle{CastleID: d.ID}
	conv := func(list []seedProductionDoc) []storage.SeedProduction {
		var out []storage.SeedProduction
		for _, p := range list {
			out = append(out, storage.SeedProduction(p))
		}
		return out
	}
	convCrops := func(list []cropProcureDoc) []storage.CropProcure {
		var out []storage.CropProcure
		for _, p := range list {
			out = append(out, storage.CropProcure(p))
		}
		return out
	}
	c.Production, c.ProductionNext = conv(d.Production), conv(d.ProductionNext)
	c.Procure, c.ProcureNext = convCrops(d.Procure), convCrops(d.ProcureNext)
	return c
}

func (s *Characters) manor() *mongo.Collection {
	return s.store.db.Collection(manorCollection)
}

// LoadManor implements storage.ManorStore.
func (s *Characters) LoadManor(ctx context.Context) (storage.ManorState, bool, []storage.ManorCastle, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	cur, err := s.manor().Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return storage.ManorState{}, false, nil, fmt.Errorf("mongostore: load the manor: %w", err)
	}
	var docs []manorDoc
	if err := cur.All(ctx, &docs); err != nil {
		return storage.ManorState{}, false, nil, fmt.Errorf("mongostore: load the manor: %w", err)
	}
	var (
		st      storage.ManorState
		ok      bool
		castles []storage.ManorCastle
	)
	for _, d := range docs {
		if d.ID == manorStateID {
			st, ok = storage.ManorState{Mode: d.Mode, NextChange: d.NextChange}, true
			continue
		}
		castles = append(castles, d.castle())
	}
	return st, ok, castles, nil
}

// applyManor writes the manor part of a batch inside its transaction.
func (s *Characters) applyManor(ctx context.Context, b *storage.Batch) error {
	models := make([]mongo.WriteModel, 0, len(b.ManorCastles)+1)
	replace := func(d manorDoc) {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: d.ID}}).
			SetReplacement(d).
			SetUpsert(true))
	}
	for _, c := range b.ManorCastles {
		replace(toManorDoc(c))
	}
	if st := b.ManorState; st != nil {
		replace(manorDoc{ID: manorStateID, Mode: st.Mode, NextChange: st.NextChange})
	}
	if len(models) == 0 {
		return nil
	}
	_, err := s.manor().BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
	return err
}
