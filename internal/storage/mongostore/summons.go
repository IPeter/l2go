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

// The summon collections (Plan 14 M11f).
const (
	petsCollection    = "pets"
	summonsCollection = "summons"
)

// Summons returns the summon store backed by this database.
func (s *Store) Summons() *Summons { return &Summons{store: s} }

// Summons is a storage.SummonStore on two collections, with Mobius's column
// names:
//
//	pets:    {_id: <collar object ID>, ownerId, name, level, curHp, curMp, exp, sp, fed, restore, [skills_save]}
//	summons: {_id: <owner>, [summon: {summonSkillId, curHp, curMp, time}], [skills_save: [{class_index, summonSkillId, skills}]], [restore_pet]}
//
// Every write runs in a transaction.
type Summons struct{ store *Store }

var _ storage.SummonStore = (*Summons)(nil)

type petDoc struct {
	ControlID int32           `bson:"_id"`
	OwnerID   int32           `bson:"ownerId"`
	Name      string          `bson:"name"`
	Level     int32           `bson:"level"`
	CurHP     float64         `bson:"curHp"`
	CurMP     float64         `bson:"curMp"`
	Exp       int64           `bson:"exp"`
	SP        int64           `bson:"sp"`
	Fed       int32           `bson:"fed"`
	Restore   bool            `bson:"restore"`
	Effects   []savedSkillDoc `bson:"skills_save,omitempty"`
}

type savedServitorDoc struct {
	SkillID int32 `bson:"summonSkillId"`
	CurHP   int32 `bson:"curHp"`
	CurMP   int32 `bson:"curMp"`
	Time    int32 `bson:"time"`
}

type servitorEffectsDoc struct {
	ClassIndex    int32           `bson:"class_index"`
	SummonSkillID int32           `bson:"summonSkillId"`
	Effects       []savedSkillDoc `bson:"skills,omitempty"`
}

type ownerSummonsDoc struct {
	OwnerID    int32                `bson:"_id"`
	Servitor   *savedServitorDoc    `bson:"summon,omitempty"`
	Effects    []servitorEffectsDoc `bson:"skills_save,omitempty"`
	RestorePet int32                `bson:"restore_pet,omitempty"`
}

func toPetDoc(p storage.Pet) petDoc {
	return petDoc{ControlID: p.ControlID, OwnerID: p.OwnerID, Name: p.Name, Level: p.Level, CurHP: p.CurHP, CurMP: p.CurMP,
		Exp: p.Exp, SP: p.SP, Fed: p.Fed, Restore: p.Restore, Effects: toSavedSkillDocs(p.Effects)}
}

func (d petDoc) pet() storage.Pet {
	return storage.Pet{ControlID: d.ControlID, OwnerID: d.OwnerID, Name: d.Name, Level: d.Level, CurHP: d.CurHP, CurMP: d.CurMP,
		Exp: d.Exp, SP: d.SP, Fed: d.Fed, Restore: d.Restore, Effects: fromSavedSkillDocs(d.Effects)}
}

func toOwnerSummonsDoc(o storage.OwnerSummons) ownerSummonsDoc {
	d := ownerSummonsDoc{OwnerID: o.OwnerID, RestorePet: o.RestorePet}
	if sv := o.Servitor; sv != nil {
		d.Servitor = &savedServitorDoc{SkillID: sv.SkillID, CurHP: sv.CurHP, CurMP: sv.CurMP, Time: sv.Time}
	}
	for _, e := range o.Effects {
		d.Effects = append(d.Effects, servitorEffectsDoc{ClassIndex: e.ClassIndex, SummonSkillID: e.SummonSkillID, Effects: toSavedSkillDocs(e.Effects)})
	}
	return d
}

func (d ownerSummonsDoc) state() storage.OwnerSummons {
	o := storage.OwnerSummons{OwnerID: d.OwnerID, RestorePet: d.RestorePet}
	if sv := d.Servitor; sv != nil {
		o.Servitor = &storage.SavedServitor{SkillID: sv.SkillID, CurHP: sv.CurHP, CurMP: sv.CurMP, Time: sv.Time}
	}
	for _, e := range d.Effects {
		o.Effects = append(o.Effects, storage.ServitorEffects{ClassIndex: e.ClassIndex, SummonSkillID: e.SummonSkillID, Effects: fromSavedSkillDocs(e.Effects)})
	}
	return o
}

func (s *Summons) coll(name string) *mongo.Collection { return s.store.db.Collection(name) }

// writeTx runs fn in a write transaction under the store's timeout.
func (s *Summons) writeTx(ctx context.Context, what string, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	if err := s.store.write(ctx, fn); err != nil {
		return fmt.Errorf("mongostore: %s: %w", what, err)
	}
	return nil
}

// LoadPet implements storage.SummonStore.
func (s *Summons) LoadPet(ctx context.Context, controlID int32) (storage.Pet, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	var d petDoc
	err := s.coll(petsCollection).FindOne(ctx, bson.D{{Key: "_id", Value: controlID}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return storage.Pet{}, fmt.Errorf("mongostore: pet %d: %w", controlID, storage.ErrNotFound)
	}
	if err != nil {
		return storage.Pet{}, fmt.Errorf("mongostore: load pet %d: %w", controlID, err)
	}
	return d.pet(), nil
}

// SavePet implements storage.SummonStore.
func (s *Summons) SavePet(ctx context.Context, p storage.Pet) error {
	doc := toPetDoc(p)
	return s.writeTx(ctx, "save pet", func(ctx context.Context) error {
		_, err := s.coll(petsCollection).ReplaceOne(ctx, bson.D{{Key: "_id", Value: p.ControlID}}, doc, options.Replace().SetUpsert(true))
		return err
	})
}

// DeletePet implements storage.SummonStore.
func (s *Summons) DeletePet(ctx context.Context, controlID int32) error {
	return s.writeTx(ctx, "delete pet", func(ctx context.Context) error {
		_, err := s.coll(petsCollection).DeleteOne(ctx, bson.D{{Key: "_id", Value: controlID}})
		return err
	})
}

// LoadOwnerSummons implements storage.SummonStore.
func (s *Summons) LoadOwnerSummons(ctx context.Context, ownerID int32) (storage.OwnerSummons, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	var d ownerSummonsDoc
	err := s.coll(summonsCollection).FindOne(ctx, bson.D{{Key: "_id", Value: ownerID}}).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return storage.OwnerSummons{OwnerID: ownerID}, nil
	}
	if err != nil {
		return storage.OwnerSummons{}, fmt.Errorf("mongostore: load the summons of %d: %w", ownerID, err)
	}
	return d.state(), nil
}

// SaveOwnerSummons implements storage.SummonStore.
func (s *Summons) SaveOwnerSummons(ctx context.Context, o storage.OwnerSummons) error {
	empty := o.Servitor == nil && len(o.Effects) == 0 && o.RestorePet == 0
	doc := toOwnerSummonsDoc(o)
	return s.writeTx(ctx, "save summons", func(ctx context.Context) error {
		filter := bson.D{{Key: "_id", Value: o.OwnerID}}
		if empty {
			_, err := s.coll(summonsCollection).DeleteOne(ctx, filter)
			return err
		}
		_, err := s.coll(summonsCollection).ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(true))
		return err
	})
}
