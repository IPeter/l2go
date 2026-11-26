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

// The collections of the Olympiad (Plan 14, M11h, decision X4).
const (
	// olympiadCollection holds one document: Mobius olympiad_data with the
	// nobles, the month-end nobles, the heroes and the unclaimed passes, so
	// that a period's end is one write.
	olympiadCollection = "olympiad"
	// olympiadFightsCollection is Mobius olympiad_fights, one document per
	// validated match.
	olympiadFightsCollection = "olympiad_fights"
	// heroDiaryCollection is Mobius heroes_diary.
	heroDiaryCollection = "heroes_diary"
)

// olympiadID is the _id of the olympiad document.
const olympiadID = "olympiad"

// Olympiad returns the Olympiad store backed by this database.
func (s *Store) Olympiad() *Olympiad { return &Olympiad{store: s} }

// Olympiad is a storage.OlympiadStore with Mobius's column names:
//
//	olympiad: {_id: "olympiad", current_cycle, period, olympiad_end, validation_end, next_weekly_change,
//	           nobles: [{charId, class_id, char_name, olympiad_points, competitions_done, competitions_won,
//	                     competitions_lost, competitions_drawn}], nobles_eom: [...],
//	           heroes: [{charId, class_id, char_name, count, played, claimed, message}],
//	           unclaimed: [{charId, passes}]}
//	olympiad_fights: {charOneId, charTwoId, charOneName, charTwoName, charOneClass, charTwoClass, winner, start, time, classed}
//	heroes_diary: {charId, time, action, param}
type Olympiad struct {
	store *Store
}

var _ storage.OlympiadStore = (*Olympiad)(nil)

type olympiadDoc struct {
	ID               string         `bson:"_id"`
	Cycle            int32          `bson:"current_cycle"`
	Period           int32          `bson:"period"`
	OlympiadEnd      int64          `bson:"olympiad_end"`
	ValidationEnd    int64          `bson:"validation_end"`
	NextWeeklyChange int64          `bson:"next_weekly_change"`
	Nobles           []nobleDoc     `bson:"nobles"`
	NoblesEOM        []nobleDoc     `bson:"nobles_eom"`
	Heroes           []heroDoc      `bson:"heroes"`
	Unclaimed        []unclaimedDoc `bson:"unclaimed"`
}

type nobleDoc struct {
	CharID  int32  `bson:"charId"`
	ClassID int32  `bson:"class_id"`
	Name    string `bson:"char_name"`
	Points  int32  `bson:"olympiad_points"`
	Done    int32  `bson:"competitions_done"`
	Won     int32  `bson:"competitions_won"`
	Lost    int32  `bson:"competitions_lost"`
	Drawn   int32  `bson:"competitions_drawn"`
}

type heroDoc struct {
	CharID  int32  `bson:"charId"`
	ClassID int32  `bson:"class_id"`
	Name    string `bson:"char_name"`
	Count   int32  `bson:"count"`
	Played  bool   `bson:"played"`
	Claimed bool   `bson:"claimed"`
	Message string `bson:"message"`
}

type unclaimedDoc struct {
	CharID int32 `bson:"charId"`
	Passes int64 `bson:"passes"`
}

type fightDoc struct {
	CharOneID    int32  `bson:"charOneId"`
	CharTwoID    int32  `bson:"charTwoId"`
	CharOneName  string `bson:"charOneName"`
	CharTwoName  string `bson:"charTwoName"`
	CharOneClass int32  `bson:"charOneClass"`
	CharTwoClass int32  `bson:"charTwoClass"`
	Winner       int32  `bson:"winner"`
	Start        int64  `bson:"start"`
	Time         int64  `bson:"time"`
	Classed      bool   `bson:"classed"`
}

type diaryDoc struct {
	CharID int32 `bson:"charId"`
	Time   int64 `bson:"time"`
	Action int32 `bson:"action"`
	Param  int32 `bson:"param"`
}

func toNobleDocs(list []storage.OlympiadNoble) []nobleDoc {
	out := make([]nobleDoc, len(list))
	for i, n := range list {
		out[i] = nobleDoc{CharID: n.CharID, ClassID: n.ClassID, Name: n.Name, Points: n.Points, Done: n.Done, Won: n.Won, Lost: n.Lost, Drawn: n.Drawn}
	}
	return out
}

func fromNobleDocs(list []nobleDoc) []storage.OlympiadNoble {
	var out []storage.OlympiadNoble
	for _, n := range list {
		out = append(out, storage.OlympiadNoble{CharID: n.CharID, ClassID: n.ClassID, Name: n.Name, Points: n.Points, Done: n.Done, Won: n.Won, Lost: n.Lost, Drawn: n.Drawn})
	}
	return out
}

func toOlympiadDoc(o storage.Olympiad) olympiadDoc {
	d := olympiadDoc{
		ID: olympiadID, Cycle: o.Cycle, Period: o.Period,
		OlympiadEnd: o.OlympiadEnd, ValidationEnd: o.ValidationEnd, NextWeeklyChange: o.NextWeeklyChange,
		Nobles: toNobleDocs(o.Nobles), NoblesEOM: toNobleDocs(o.NoblesEOM),
		Heroes: []heroDoc{}, Unclaimed: []unclaimedDoc{},
	}
	for _, h := range o.Heroes {
		d.Heroes = append(d.Heroes, heroDoc{CharID: h.CharID, ClassID: h.ClassID, Name: h.Name, Count: h.Count, Played: h.Played, Claimed: h.Claimed, Message: h.Message})
	}
	for _, u := range o.Unclaimed {
		d.Unclaimed = append(d.Unclaimed, unclaimedDoc(u))
	}
	return d
}

func (d olympiadDoc) state() storage.Olympiad {
	o := storage.Olympiad{
		Cycle: d.Cycle, Period: d.Period,
		OlympiadEnd: d.OlympiadEnd, ValidationEnd: d.ValidationEnd, NextWeeklyChange: d.NextWeeklyChange,
		Nobles: fromNobleDocs(d.Nobles), NoblesEOM: fromNobleDocs(d.NoblesEOM),
	}
	for _, h := range d.Heroes {
		o.Heroes = append(o.Heroes, storage.Hero{CharID: h.CharID, ClassID: h.ClassID, Name: h.Name, Count: h.Count, Played: h.Played, Claimed: h.Claimed, Message: h.Message})
	}
	for _, u := range d.Unclaimed {
		o.Unclaimed = append(o.Unclaimed, storage.UnclaimedPasses(u))
	}
	return o
}

func (o *Olympiad) coll(name string) *mongo.Collection { return o.store.db.Collection(name) }

// write runs fn as a write under the store's timeout.
func (o *Olympiad) write(ctx context.Context, what string, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, o.store.timeout)
	defer cancel()
	if err := o.store.write(ctx, fn); err != nil {
		return fmt.Errorf("mongostore: %s: %w", what, err)
	}
	return nil
}

// LoadOlympiad implements storage.OlympiadStore.
func (o *Olympiad) LoadOlympiad(ctx context.Context) (storage.Olympiad, error) {
	ctx, cancel := context.WithTimeout(ctx, o.store.timeout)
	defer cancel()
	var doc olympiadDoc
	err := o.coll(olympiadCollection).FindOne(ctx, bson.D{{Key: "_id", Value: olympiadID}}).Decode(&doc)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return storage.Olympiad{}, nil
	case err != nil:
		return storage.Olympiad{}, fmt.Errorf("mongostore: load the olympiad: %w", err)
	}
	return doc.state(), nil
}

// SaveOlympiad implements storage.OlympiadStore.
func (o *Olympiad) SaveOlympiad(ctx context.Context, st storage.Olympiad) error {
	doc := toOlympiadDoc(st)
	return o.write(ctx, "save the olympiad", func(ctx context.Context) error {
		_, err := o.coll(olympiadCollection).ReplaceOne(ctx, bson.D{{Key: "_id", Value: olympiadID}}, doc, options.Replace().SetUpsert(true))
		return err
	})
}

// AddOlympiadFight implements storage.OlympiadStore.
func (o *Olympiad) AddOlympiadFight(ctx context.Context, f storage.OlympiadFight) error {
	doc := fightDoc(f)
	return o.write(ctx, "add an olympiad fight", func(ctx context.Context) error {
		_, err := o.coll(olympiadFightsCollection).InsertOne(ctx, doc)
		return err
	})
}

// OlympiadFights implements storage.OlympiadStore.
func (o *Olympiad) OlympiadFights(ctx context.Context, charID int32, before int64) ([]storage.OlympiadFight, error) {
	ctx, cancel := context.WithTimeout(ctx, o.store.timeout)
	defer cancel()
	filter := bson.D{
		{Key: "$or", Value: bson.A{bson.D{{Key: "charOneId", Value: charID}}, bson.D{{Key: "charTwoId", Value: charID}}}},
		{Key: "start", Value: bson.D{{Key: "$lt", Value: before}}},
	}
	cur, err := o.coll(olympiadFightsCollection).Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "start", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load the olympiad fights of %d: %w", charID, err)
	}
	var docs []fightDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load the olympiad fights of %d: %w", charID, err)
	}
	out := make([]storage.OlympiadFight, len(docs))
	for i, d := range docs {
		out[i] = storage.OlympiadFight(d)
	}
	return out, nil
}

// AddHeroDiary implements storage.OlympiadStore.
func (o *Olympiad) AddHeroDiary(ctx context.Context, e storage.HeroDiaryEntry) error {
	doc := diaryDoc(e)
	return o.write(ctx, "add a hero diary entry", func(ctx context.Context) error {
		_, err := o.coll(heroDiaryCollection).InsertOne(ctx, doc)
		return err
	})
}

// HeroDiary implements storage.OlympiadStore.
func (o *Olympiad) HeroDiary(ctx context.Context, charID int32) ([]storage.HeroDiaryEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, o.store.timeout)
	defer cancel()
	cur, err := o.coll(heroDiaryCollection).Find(ctx, bson.D{{Key: "charId", Value: charID}},
		options.Find().SetSort(bson.D{{Key: "time", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load the hero diary of %d: %w", charID, err)
	}
	var docs []diaryDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load the hero diary of %d: %w", charID, err)
	}
	out := make([]storage.HeroDiaryEntry, len(docs))
	for i, d := range docs {
		out[i] = storage.HeroDiaryEntry(d)
	}
	return out, nil
}
