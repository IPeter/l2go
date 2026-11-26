package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

// castlesCollection holds the castles (Plan 14 M11i, decision X4), one
// document per castle with Mobius's column names and its side tables as
// arrays:
//
//	{_id: <castleId>, taxPercent, treasury, siegeDate, regTimeOver, regTimeEnd,
//	 showNpcCrest, ticketBuyCount, functions: [...], doorUpgrades: [...],
//	 trapUpgrades: [...], siegeClans: [{clan_id, type}], hiredGuards: [...]}
//
// Batches write it (WriteBatch), in the transaction of the clans a siege
// changes.
const castlesCollection = "castles"

var _ storage.CastleStore = (*Characters)(nil)

type castleFunctionDoc struct {
	Type    int32 `bson:"type"`
	Level   int32 `bson:"lvl"`
	Lease   int32 `bson:"lease"`
	Rate    int64 `bson:"rate"`
	EndTime int64 `bson:"endTime"`
}

type castleDoorDoc struct {
	DoorID int32 `bson:"doorId"`
	Ratio  int32 `bson:"ratio"`
}

type castleTrapDoc struct {
	TowerIndex int32 `bson:"towerIndex"`
	Level      int32 `bson:"level"`
}

type siegeClanDoc struct {
	ClanID int32 `bson:"clan_id"`
	Type   int32 `bson:"type"`
}

type hiredGuardDoc struct {
	NpcID      int32 `bson:"npcId"`
	X          int32 `bson:"x"`
	Y          int32 `bson:"y"`
	Z          int32 `bson:"z"`
	Heading    int32 `bson:"heading"`
	TicketItem int32 `bson:"ticketItem"`
}

type castleDoc struct {
	ID             int32               `bson:"_id"`
	TaxPercent     int32               `bson:"taxPercent"`
	Treasury       int64               `bson:"treasury"`
	SiegeDate      int64               `bson:"siegeDate"`
	RegTimeOver    bool                `bson:"regTimeOver"`
	RegTimeEnd     int64               `bson:"regTimeEnd"`
	ShowNpcCrest   bool                `bson:"showNpcCrest"`
	TicketBuyCount int32               `bson:"ticketBuyCount"`
	Functions      []castleFunctionDoc `bson:"functions,omitempty"`
	DoorUpgrades   []castleDoorDoc     `bson:"doorUpgrades,omitempty"`
	TrapUpgrades   []castleTrapDoc     `bson:"trapUpgrades,omitempty"`
	SiegeClans     []siegeClanDoc      `bson:"siegeClans,omitempty"`
	HiredGuards    []hiredGuardDoc     `bson:"hiredGuards,omitempty"`
}

func toCastleDoc(c storage.Castle) castleDoc {
	d := castleDoc{ID: c.ID, TaxPercent: c.TaxPercent, Treasury: c.Treasury, SiegeDate: c.SiegeDate,
		RegTimeOver: c.RegTimeOver, RegTimeEnd: c.RegTimeEnd, ShowNpcCrest: c.ShowNpcCrest, TicketBuyCount: c.TicketBuyCount}
	for _, f := range c.Functions {
		d.Functions = append(d.Functions, castleFunctionDoc(f))
	}
	for _, u := range c.DoorUpgrades {
		d.DoorUpgrades = append(d.DoorUpgrades, castleDoorDoc(u))
	}
	for _, u := range c.TrapUpgrades {
		d.TrapUpgrades = append(d.TrapUpgrades, castleTrapDoc(u))
	}
	for _, sc := range c.SiegeClans {
		d.SiegeClans = append(d.SiegeClans, siegeClanDoc(sc))
	}
	for _, g := range c.HiredGuards {
		d.HiredGuards = append(d.HiredGuards, hiredGuardDoc(g))
	}
	return d
}

func (d castleDoc) state() storage.Castle {
	c := storage.Castle{ID: d.ID, TaxPercent: d.TaxPercent, Treasury: d.Treasury, SiegeDate: d.SiegeDate,
		RegTimeOver: d.RegTimeOver, RegTimeEnd: d.RegTimeEnd, ShowNpcCrest: d.ShowNpcCrest, TicketBuyCount: d.TicketBuyCount}
	for _, f := range d.Functions {
		c.Functions = append(c.Functions, storage.CastleFunction(f))
	}
	for _, u := range d.DoorUpgrades {
		c.DoorUpgrades = append(c.DoorUpgrades, storage.CastleDoorUpgrade(u))
	}
	for _, u := range d.TrapUpgrades {
		c.TrapUpgrades = append(c.TrapUpgrades, storage.CastleTrapUpgrade(u))
	}
	for _, sc := range d.SiegeClans {
		c.SiegeClans = append(c.SiegeClans, storage.SiegeClan(sc))
	}
	for _, g := range d.HiredGuards {
		c.HiredGuards = append(c.HiredGuards, storage.HiredGuard(g))
	}
	return c
}

func (s *Characters) castles() *mongo.Collection {
	return s.store.db.Collection(castlesCollection)
}

// LoadCastles implements storage.CastleStore.
func (s *Characters) LoadCastles(ctx context.Context) ([]storage.Castle, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	cur, err := s.castles().Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load castles: %w", err)
	}
	var docs []castleDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load castles: %w", err)
	}
	var out []storage.Castle
	for _, d := range docs {
		out = append(out, d.state())
	}
	return out, nil
}

// applyCastles writes the castle part of a batch inside its transaction.
func (s *Characters) applyCastles(ctx context.Context, b *storage.Batch) error {
	if len(b.Castles) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(b.Castles))
	for _, c := range b.Castles {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: c.ID}}).
			SetReplacement(toCastleDoc(c)).
			SetUpsert(true))
	}
	_, err := s.castles().BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
	return err
}
