package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

// The clan halls (Plan 14 M11j, decision X4): one document per hall in
// clanhalls, with Mobius's column names and its side tables as arrays, and
// one per auction in auctions, with its bids:
//
//	{_id: <hallId>, paidUntil, paid, functions: [{type, lvl, lease, rate,
//	 endTime}], nextSiege, attackers: [<clanId>], vars: {name: value}}
//	{_id: <auctionId>, sellerId, sellerName, sellerClanName, itemName,
//	 startingBid, currentBid, endDate, bids: [{bidderId, bidderName,
//	 clan_name, maxBid, time_bid}]}
//
// Batches write them (WriteBatch), in the transaction of the clans and
// warehouse items a sale, lease or siege changes.
const (
	clanHallsCollection = "clanhalls"
	auctionsCollection  = "auctions"
)

var _ storage.ClanHallStore = (*Characters)(nil)

type clanHallDoc struct {
	ID        int32               `bson:"_id"`
	PaidUntil int64               `bson:"paidUntil"`
	Paid      bool                `bson:"paid"`
	Functions []castleFunctionDoc `bson:"functions,omitempty"`
	NextSiege int64               `bson:"nextSiege"`
	Attackers []int32             `bson:"attackers,omitempty"`
	Vars      map[string]string   `bson:"vars,omitempty"`
}

type auctionBidDoc struct {
	ClanID     int32  `bson:"bidderId"`
	BidderName string `bson:"bidderName"`
	ClanName   string `bson:"clan_name"`
	MaxBid     int64  `bson:"maxBid"`
	TimeBid    int64  `bson:"time_bid"`
}

type auctionDoc struct {
	ID             int32           `bson:"_id"`
	SellerID       int32           `bson:"sellerId"`
	SellerName     string          `bson:"sellerName"`
	SellerClanName string          `bson:"sellerClanName"`
	ItemName       string          `bson:"itemName"`
	StartingBid    int64           `bson:"startingBid"`
	CurrentBid     int64           `bson:"currentBid"`
	EndDate        int64           `bson:"endDate"`
	Bids           []auctionBidDoc `bson:"bids,omitempty"`
}

func toClanHallDoc(h storage.ClanHall) clanHallDoc {
	d := clanHallDoc{ID: h.ID, PaidUntil: h.PaidUntil, Paid: h.Paid, NextSiege: h.NextSiege, Attackers: h.Attackers, Vars: h.Vars}
	for _, f := range h.Functions {
		d.Functions = append(d.Functions, castleFunctionDoc(f))
	}
	return d
}

func (d clanHallDoc) state() storage.ClanHall {
	h := storage.ClanHall{ID: d.ID, PaidUntil: d.PaidUntil, Paid: d.Paid, NextSiege: d.NextSiege, Attackers: d.Attackers, Vars: d.Vars}
	for _, f := range d.Functions {
		h.Functions = append(h.Functions, storage.ClanHallFunction(f))
	}
	return h
}

func toAuctionDoc(a storage.Auction) auctionDoc {
	d := auctionDoc{ID: a.ID, SellerID: a.SellerID, SellerName: a.SellerName, SellerClanName: a.SellerClanName,
		ItemName: a.ItemName, StartingBid: a.StartingBid, CurrentBid: a.CurrentBid, EndDate: a.EndDate}
	for _, b := range a.Bids {
		d.Bids = append(d.Bids, auctionBidDoc(b))
	}
	return d
}

func (d auctionDoc) state() storage.Auction {
	a := storage.Auction{ID: d.ID, SellerID: d.SellerID, SellerName: d.SellerName, SellerClanName: d.SellerClanName,
		ItemName: d.ItemName, StartingBid: d.StartingBid, CurrentBid: d.CurrentBid, EndDate: d.EndDate}
	for _, b := range d.Bids {
		a.Bids = append(a.Bids, storage.AuctionBid(b))
	}
	return a
}

// LoadClanHalls implements storage.ClanHallStore.
func (s *Characters) LoadClanHalls(ctx context.Context) ([]storage.ClanHall, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	cur, err := s.store.db.Collection(clanHallsCollection).Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load clan halls: %w", err)
	}
	var docs []clanHallDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load clan halls: %w", err)
	}
	var out []storage.ClanHall
	for _, d := range docs {
		out = append(out, d.state())
	}
	return out, nil
}

// LoadAuctions implements storage.ClanHallStore.
func (s *Characters) LoadAuctions(ctx context.Context) ([]storage.Auction, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	cur, err := s.store.db.Collection(auctionsCollection).Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load auctions: %w", err)
	}
	var docs []auctionDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load auctions: %w", err)
	}
	var out []storage.Auction
	for _, d := range docs {
		out = append(out, d.state())
	}
	return out, nil
}

// applyClanHalls writes the clan hall part of a batch inside its
// transaction.
func (s *Characters) applyClanHalls(ctx context.Context, b *storage.Batch) error {
	if len(b.ClanHalls) > 0 {
		models := make([]mongo.WriteModel, 0, len(b.ClanHalls))
		for _, h := range b.ClanHalls {
			models = append(models, mongo.NewReplaceOneModel().SetFilter(bson.D{{Key: "_id", Value: h.ID}}).
				SetReplacement(toClanHallDoc(h)).SetUpsert(true))
		}
		if _, err := s.store.db.Collection(clanHallsCollection).BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true)); err != nil {
			return err
		}
	}
	if len(b.DeletedAuctions) == 0 && len(b.Auctions) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(b.DeletedAuctions)+len(b.Auctions))
	for _, id := range b.DeletedAuctions {
		models = append(models, mongo.NewDeleteOneModel().SetFilter(bson.D{{Key: "_id", Value: id}}))
	}
	for _, a := range b.Auctions {
		models = append(models, mongo.NewReplaceOneModel().SetFilter(bson.D{{Key: "_id", Value: a.ID}}).
			SetReplacement(toAuctionDoc(a)).SetUpsert(true))
	}
	_, err := s.store.db.Collection(auctionsCollection).BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
	return err
}
