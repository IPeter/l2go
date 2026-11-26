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

// The collections of the Seven Signs (Plan 14, M11l, decision X4), named
// after Mobius's tables.
const (
	// sevenSignsCollection is Mobius seven_signs, one document per
	// character (_id the character's object ID).
	sevenSignsCollection = "seven_signs"
	// sevenSignsStatusCollection is Mobius seven_signs_status, one document.
	sevenSignsStatusCollection = "seven_signs_status"
	// sevenSignsFestivalCollection is Mobius seven_signs_festival, one
	// document per festival, cabal and cycle.
	sevenSignsFestivalCollection = "seven_signs_festival"
)

// sevenSignsStatusID is the _id of the status document (Mobius id 0).
const sevenSignsStatusID = 0

// sevenSignsBonuses is the number of accumulated_bonus columns.
const sevenSignsBonuses = 5

// SevenSigns returns the Seven Signs store backed by this database.
func (s *Store) SevenSigns() *SevenSigns { return &SevenSigns{store: s} }

// SevenSigns is a storage.SevenSignsStore with Mobius's column names:
//
//	seven_signs: {_id: charId, charId, cabal, seal, red_stones, green_stones, blue_stones,
//	              ancient_adena_amount, contribution_score}
//	seven_signs_status: {_id: 0, current_cycle, active_period, previous_winner, dawn_stone_score,
//	              dawn_festival_score, dusk_stone_score, dusk_festival_score, avarice_owner, gnosis_owner,
//	              strife_owner, avarice_dawn_score, ..., strife_dusk_score, festival_cycle,
//	              accumulated_bonus0..4, date, next_period_change}
//	seven_signs_festival: {festivalId, cabal, cycle, date, score, members}
type SevenSigns struct {
	store *Store
}

var _ storage.SevenSignsStore = (*SevenSigns)(nil)

type sevenSignsPlayerDoc struct {
	ID           int32  `bson:"_id"`
	CharID       int32  `bson:"charId"`
	Cabal        string `bson:"cabal"`
	Seal         int32  `bson:"seal"`
	RedStones    int32  `bson:"red_stones"`
	GreenStones  int32  `bson:"green_stones"`
	BlueStones   int32  `bson:"blue_stones"`
	AncientAdena int64  `bson:"ancient_adena_amount"`
	Contribution int64  `bson:"contribution_score"`
}

type sevenSignsStatusDoc struct {
	ID                int32   `bson:"_id"`
	Cycle             int32   `bson:"current_cycle"`
	Period            int32   `bson:"active_period"`
	PreviousWinner    int32   `bson:"previous_winner"`
	DawnStoneScore    float64 `bson:"dawn_stone_score"`
	DawnFestivalScore int32   `bson:"dawn_festival_score"`
	DuskStoneScore    float64 `bson:"dusk_stone_score"`
	DuskFestivalScore int32   `bson:"dusk_festival_score"`
	AvariceOwner      int32   `bson:"avarice_owner"`
	GnosisOwner       int32   `bson:"gnosis_owner"`
	StrifeOwner       int32   `bson:"strife_owner"`
	AvariceDawnScore  int32   `bson:"avarice_dawn_score"`
	GnosisDawnScore   int32   `bson:"gnosis_dawn_score"`
	StrifeDawnScore   int32   `bson:"strife_dawn_score"`
	AvariceDuskScore  int32   `bson:"avarice_dusk_score"`
	GnosisDuskScore   int32   `bson:"gnosis_dusk_score"`
	StrifeDuskScore   int32   `bson:"strife_dusk_score"`
	FestivalCycle     int32   `bson:"festival_cycle"`
	Bonus0            int32   `bson:"accumulated_bonus0"`
	Bonus1            int32   `bson:"accumulated_bonus1"`
	Bonus2            int32   `bson:"accumulated_bonus2"`
	Bonus3            int32   `bson:"accumulated_bonus3"`
	Bonus4            int32   `bson:"accumulated_bonus4"`
	Date              int64   `bson:"date"`
	NextPeriodChange  int64   `bson:"next_period_change"`
}

type sevenSignsFestivalDoc struct {
	Festival int32  `bson:"festivalId"`
	Cabal    string `bson:"cabal"`
	Cycle    int32  `bson:"cycle"`
	Date     int64  `bson:"date"`
	Score    int32  `bson:"score"`
	Members  string `bson:"members"`
}

func toSevenSignsPlayerDoc(p storage.SevenSignsPlayer) sevenSignsPlayerDoc {
	return sevenSignsPlayerDoc{ID: p.CharID, CharID: p.CharID, Cabal: p.Cabal, Seal: p.Seal, RedStones: p.RedStones,
		GreenStones: p.GreenStones, BlueStones: p.BlueStones, AncientAdena: p.AncientAdena, Contribution: p.Contribution}
}

func (d sevenSignsPlayerDoc) player() storage.SevenSignsPlayer {
	return storage.SevenSignsPlayer{CharID: d.CharID, Cabal: d.Cabal, Seal: d.Seal, RedStones: d.RedStones,
		GreenStones: d.GreenStones, BlueStones: d.BlueStones, AncientAdena: d.AncientAdena, Contribution: d.Contribution}
}

func toSevenSignsStatusDoc(s storage.SevenSignsStatus) sevenSignsStatusDoc {
	var b [sevenSignsBonuses]int32
	copy(b[:], s.AccumulatedBonus)
	return sevenSignsStatusDoc{
		ID: sevenSignsStatusID, Cycle: s.Cycle, Period: s.Period, PreviousWinner: s.PreviousWinner,
		DawnStoneScore: s.DawnStoneScore, DawnFestivalScore: s.DawnFestivalScore,
		DuskStoneScore: s.DuskStoneScore, DuskFestivalScore: s.DuskFestivalScore,
		AvariceOwner: s.AvariceOwner, GnosisOwner: s.GnosisOwner, StrifeOwner: s.StrifeOwner,
		AvariceDawnScore: s.AvariceDawnScore, GnosisDawnScore: s.GnosisDawnScore, StrifeDawnScore: s.StrifeDawnScore,
		AvariceDuskScore: s.AvariceDuskScore, GnosisDuskScore: s.GnosisDuskScore, StrifeDuskScore: s.StrifeDuskScore,
		FestivalCycle: s.FestivalCycle, Bonus0: b[0], Bonus1: b[1], Bonus2: b[2], Bonus3: b[3], Bonus4: b[4],
		Date: s.Date, NextPeriodChange: s.NextPeriodChange,
	}
}

func (d sevenSignsStatusDoc) status() storage.SevenSignsStatus {
	return storage.SevenSignsStatus{
		Cycle: d.Cycle, Period: d.Period, PreviousWinner: d.PreviousWinner,
		DawnStoneScore: d.DawnStoneScore, DawnFestivalScore: d.DawnFestivalScore,
		DuskStoneScore: d.DuskStoneScore, DuskFestivalScore: d.DuskFestivalScore,
		AvariceOwner: d.AvariceOwner, GnosisOwner: d.GnosisOwner, StrifeOwner: d.StrifeOwner,
		AvariceDawnScore: d.AvariceDawnScore, GnosisDawnScore: d.GnosisDawnScore, StrifeDawnScore: d.StrifeDawnScore,
		AvariceDuskScore: d.AvariceDuskScore, GnosisDuskScore: d.GnosisDuskScore, StrifeDuskScore: d.StrifeDuskScore,
		FestivalCycle:    d.FestivalCycle,
		AccumulatedBonus: []int32{d.Bonus0, d.Bonus1, d.Bonus2, d.Bonus3, d.Bonus4},
		Date:             d.Date, NextPeriodChange: d.NextPeriodChange,
	}
}

func (s *SevenSigns) coll(name string) *mongo.Collection { return s.store.db.Collection(name) }

// LoadSevenSigns implements storage.SevenSignsStore.
func (s *SevenSigns) LoadSevenSigns(ctx context.Context) (storage.SevenSigns, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	var out storage.SevenSigns
	err := s.store.read(ctx, func(ctx context.Context) error {
		out = storage.SevenSigns{}
		var st sevenSignsStatusDoc
		err := s.coll(sevenSignsStatusCollection).FindOne(ctx, bson.D{{Key: "_id", Value: sevenSignsStatusID}}).Decode(&st)
		switch {
		case errors.Is(err, mongo.ErrNoDocuments):
		case err != nil:
			return err
		default:
			out.HasStatus, out.Status = true, st.status()
		}
		cur, err := s.coll(sevenSignsCollection).Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
		if err != nil {
			return err
		}
		var players []sevenSignsPlayerDoc
		if err := cur.All(ctx, &players); err != nil {
			return err
		}
		for _, p := range players {
			out.Players = append(out.Players, p.player())
		}
		cur, err = s.coll(sevenSignsFestivalCollection).Find(ctx, bson.D{},
			options.Find().SetSort(bson.D{{Key: "festivalId", Value: 1}, {Key: "cabal", Value: 1}, {Key: "cycle", Value: 1}}))
		if err != nil {
			return err
		}
		var fest []sevenSignsFestivalDoc
		if err := cur.All(ctx, &fest); err != nil {
			return err
		}
		for _, f := range fest {
			out.Festival = append(out.Festival, storage.SevenSignsFestivalScore(f))
		}
		return nil
	})
	if err != nil {
		return storage.SevenSigns{}, fmt.Errorf("mongostore: load the seven signs: %w", err)
	}
	return out, nil
}

// SaveSevenSigns implements storage.SevenSignsStore: the status, the
// players and, with ReplaceFestival, the festival rows in one transaction.
func (s *SevenSigns) SaveSevenSigns(ctx context.Context, sv storage.SevenSignsSave) error {
	status := toSevenSignsStatusDoc(sv.Status)
	players := make([]sevenSignsPlayerDoc, len(sv.Players))
	for i, p := range sv.Players {
		players[i] = toSevenSignsPlayerDoc(p)
	}
	fest := make([]any, len(sv.Festival))
	for i, f := range sv.Festival {
		fest[i] = sevenSignsFestivalDoc(f)
	}
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	err := s.store.write(ctx, func(ctx context.Context) error {
		upsert := options.Replace().SetUpsert(true)
		if _, err := s.coll(sevenSignsStatusCollection).ReplaceOne(ctx, bson.D{{Key: "_id", Value: sevenSignsStatusID}}, status, upsert); err != nil {
			return err
		}
		for _, p := range players {
			if _, err := s.coll(sevenSignsCollection).ReplaceOne(ctx, bson.D{{Key: "_id", Value: p.ID}}, p, upsert); err != nil {
				return err
			}
		}
		if !sv.ReplaceFestival {
			return nil
		}
		if _, err := s.coll(sevenSignsFestivalCollection).DeleteMany(ctx, bson.D{}); err != nil {
			return err
		}
		if len(fest) > 0 {
			if _, err := s.coll(sevenSignsFestivalCollection).InsertMany(ctx, fest); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("mongostore: save the seven signs: %w", err)
	}
	return nil
}
