package mongostore

import (
	"context"
	"errors"
	"fmt"
	"math"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

const (
	clansCollection     = "clans"
	crestsCollection    = "crests"
	relationsCollection = "character_relations"
	crestIDCounter      = "crest_id" // _id of the crest ID counter document in counters
)

// clanIndexes are the indexes of the clans collection (Plan 12, data
// model): clan names are unique ignoring case through nameKey, as character
// names are through name_lower; alliances are found by ID and by name.
func clanIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		{Keys: bson.D{{Key: "nameKey", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "allyId", Value: 1}}},
		{Keys: bson.D{{Key: "allyNameKey", Value: 1}}},
	}
}

// relationIndexes are the indexes of the character_relations collection
// (decision T4): one document per owner, target and relation, which also
// finds an owner's relations, and the relations to a target.
func relationIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "owner", Value: 1}, {Key: "target", Value: 1}, {Key: "relation", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "target", Value: 1}}},
	}
}

// clanDoc is the stored form of storage.Clan, with the field names of Plan
// 12's data model.
type clanDoc struct {
	ID           int32  `bson:"_id"`
	Name         string `bson:"name"`
	NameKey      string `bson:"nameKey"` // storage.NameKey(Name), unique
	Level        int32  `bson:"level"`
	Reputation   int32  `bson:"reputation"`
	LeaderID     int32  `bson:"leaderId"`
	NewLeaderID  int32  `bson:"newLeaderId"`
	CrestID      int32  `bson:"crestId"`
	CrestLargeID int32  `bson:"crestLargeId"`

	AllyID              int32          `bson:"allyId"`
	AllyName            string         `bson:"allyName"`
	AllyNameKey         string         `bson:"allyNameKey"` // storage.NameKey(AllyName)
	AllyCrestID         int32          `bson:"allyCrestId"`
	AllyPenalty         allyPenaltyDoc `bson:"allyPenalty"`
	MemberPenaltyExpiry int64          `bson:"memberPenaltyExpiry"`
	DissolvingExpiry    int64          `bson:"dissolvingExpiry"`

	CastleID  int32 `bson:"castleId"`
	HideoutID int32 `bson:"hideoutId"`

	Ranks        []clanRankDoc  `bson:"ranks,omitempty"`
	Units        []clanUnitDoc  `bson:"units,omitempty"`
	Skills       []clanSkillDoc `bson:"skills,omitempty"`
	Notice       clanNoticeDoc  `bson:"notice"`
	WarsDeclared []int32        `bson:"warsDeclared,omitempty"`
}

type allyPenaltyDoc struct {
	Expiry int64 `bson:"expiry"`
	Type   int32 `bson:"type"`
}

type clanRankDoc struct {
	Rank  int32 `bson:"rank"`
	Privs int32 `bson:"privs"`
}

type clanUnitDoc struct {
	Type     int32  `bson:"type"`
	Name     string `bson:"name"`
	LeaderID int32  `bson:"leaderId"`
}

type clanSkillDoc struct {
	SkillID int32 `bson:"skillId"`
	Level   int32 `bson:"level"`
	Unit    int32 `bson:"unit"`
}

type clanNoticeDoc struct {
	Text    string `bson:"text"`
	Enabled bool   `bson:"enabled"`
}

func toClanDoc(c *storage.Clan) clanDoc {
	return clanDoc{
		ID:                  c.ID,
		Name:                c.Name,
		NameKey:             storage.NameKey(c.Name),
		Level:               c.Level,
		Reputation:          c.Reputation,
		LeaderID:            c.LeaderID,
		NewLeaderID:         c.NewLeaderID,
		CrestID:             c.CrestID,
		CrestLargeID:        c.CrestLargeID,
		AllyID:              c.AllyID,
		AllyName:            c.AllyName,
		AllyNameKey:         storage.NameKey(c.AllyName),
		AllyCrestID:         c.AllyCrestID,
		AllyPenalty:         allyPenaltyDoc(c.AllyPenalty),
		MemberPenaltyExpiry: c.MemberPenaltyExpiry,
		DissolvingExpiry:    c.DissolvingExpiry,
		CastleID:            c.CastleID,
		HideoutID:           c.HideoutID,
		Ranks:               convertList(c.Ranks, func(r storage.ClanRank) clanRankDoc { return clanRankDoc(r) }),
		Units:               convertList(c.Units, func(u storage.ClanUnit) clanUnitDoc { return clanUnitDoc(u) }),
		Skills:              convertList(c.Skills, func(s storage.ClanSkill) clanSkillDoc { return clanSkillDoc(s) }),
		Notice:              clanNoticeDoc(c.Notice),
		WarsDeclared:        convertList(c.WarsDeclared, func(id int32) int32 { return id }),
	}
}

func (d *clanDoc) clan() storage.Clan {
	return storage.Clan{
		ID:                  d.ID,
		Name:                d.Name,
		Level:               d.Level,
		Reputation:          d.Reputation,
		LeaderID:            d.LeaderID,
		NewLeaderID:         d.NewLeaderID,
		CrestID:             d.CrestID,
		CrestLargeID:        d.CrestLargeID,
		AllyID:              d.AllyID,
		AllyName:            d.AllyName,
		AllyCrestID:         d.AllyCrestID,
		AllyPenalty:         storage.AllyPenalty(d.AllyPenalty),
		MemberPenaltyExpiry: d.MemberPenaltyExpiry,
		DissolvingExpiry:    d.DissolvingExpiry,
		CastleID:            d.CastleID,
		HideoutID:           d.HideoutID,
		Ranks:               convertList(d.Ranks, func(r clanRankDoc) storage.ClanRank { return storage.ClanRank(r) }),
		Units:               convertList(d.Units, func(u clanUnitDoc) storage.ClanUnit { return storage.ClanUnit(u) }),
		Skills:              convertList(d.Skills, func(s clanSkillDoc) storage.ClanSkill { return storage.ClanSkill(s) }),
		Notice:              storage.ClanNotice(d.Notice),
		WarsDeclared:        convertList(d.WarsDeclared, func(id int32) int32 { return id }),
	}
}

// crestDoc is the stored form of storage.Crest (decision T3).
type crestDoc struct {
	ID   int32  `bson:"_id"`
	Type int32  `bson:"type"`
	Data []byte `bson:"data"`
}

// relationDoc is the stored form of storage.Relation (decision T4), the
// Mobius character_friends row. The _id is the server's.
type relationDoc struct {
	Owner    int32 `bson:"owner"`
	Target   int32 `bson:"target"`
	Relation int32 `bson:"relation"`
}

func (d relationDoc) relation() storage.Relation {
	return storage.Relation{Owner: d.Owner, Target: d.Target, Type: d.Relation}
}

func relationFilter(r storage.Relation) bson.D {
	return bson.D{{Key: "owner", Value: r.Owner}, {Key: "target", Value: r.Target}, {Key: "relation", Value: r.Type}}
}

// patchUpdate returns the update that writes the fields of p, or nil if it
// selects none. The clan fields are removed while zero, as the character
// documents keep them.
func patchUpdate(p storage.CharacterPatch) bson.D {
	var c storage.Character
	p.Apply(&c)
	doc := toCharacterDoc(&c)
	var set, unset bson.D
	if p.Fields&storage.PatchClanID != 0 {
		set = append(set, bson.E{Key: "clan_id", Value: doc.ClanID})
	}
	if p.Fields&storage.PatchTitle != 0 {
		set = append(set, bson.E{Key: "title", Value: doc.Title})
	}
	if p.Fields&storage.PatchKarma != 0 {
		set = append(set, bson.E{Key: "karma", Value: doc.Karma})
	}
	if p.Fields&storage.PatchPKKills != 0 {
		set = append(set, bson.E{Key: "pk_kills", Value: doc.PKKills})
	}
	for _, f := range clanFieldValues(&doc) {
		switch {
		case p.Fields&f.patch == 0:
		case f.zero:
			unset = append(unset, bson.E{Key: f.key, Value: ""})
		default:
			set = append(set, bson.E{Key: f.key, Value: f.value})
		}
	}
	var update bson.D
	if len(set) > 0 {
		update = append(update, bson.E{Key: "$set", Value: set})
	}
	if len(unset) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unset})
	}
	return update
}

// batchWrite is a storage.Batch prepared for a transaction.
type batchWrite struct {
	saves        []saveWrite
	owners       []ownerWrite
	deletedClans []int32
	clans        []mongo.WriteModel
	allyCrests   []storage.AllyCrest
	deadCrests   []int32
	crests       []mongo.WriteModel
	relations    []mongo.WriteModel // removes, then adds
	patches      []patchWrite
}

type ownerWrite struct {
	owner           int32
	destroyed, left []int32
	upserts         []mongo.WriteModel
}

type patchWrite struct {
	objectID int32
	update   bson.D
}

func newBatchWrite(b *storage.Batch) batchWrite {
	w := batchWrite{deletedClans: b.DeletedClans, allyCrests: b.AllyCrests, deadCrests: b.DeletedCrests}
	for i := range b.Saves {
		w.saves = append(w.saves, newSaveWrite(&b.Saves[i].Character, b.Saves[i].Changes))
	}
	for _, o := range b.Items {
		w.owners = append(w.owners, ownerWrite{owner: o.Owner, destroyed: o.Destroyed, left: o.Left, upserts: itemUpserts(o.Owner, o.Upserts)})
	}
	for i := range b.Clans {
		w.clans = append(w.clans, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: b.Clans[i].ID}}).
			SetReplacement(toClanDoc(&b.Clans[i])).
			SetUpsert(true))
	}
	for _, c := range b.Crests {
		w.crests = append(w.crests, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: c.ID}}).
			SetReplacement(crestDoc(c)).
			SetUpsert(true))
	}
	for _, r := range b.Relations.Removed {
		w.relations = append(w.relations, mongo.NewDeleteOneModel().SetFilter(relationFilter(r)))
	}
	for _, r := range b.Relations.Added {
		w.relations = append(w.relations, mongo.NewReplaceOneModel().
			SetFilter(relationFilter(r)).
			SetReplacement(relationDoc{Owner: r.Owner, Target: r.Target, Relation: r.Type}).
			SetUpsert(true))
	}
	for _, p := range b.Patches {
		if update := patchUpdate(p); update != nil {
			w.patches = append(w.patches, patchWrite{objectID: p.ObjectID, update: update})
		}
	}
	return w
}

// WriteBatch implements storage.BatchWriter: everything in one transaction,
// in the order storage.Batch gives. The unique nameKey index refuses a clan
// name another clan has.
func (s *Characters) WriteBatch(ctx context.Context, b storage.Batch) error {
	w := newBatchWrite(&b)
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	err := s.store.write(ctx, func(ctx context.Context) error {
		for _, sw := range w.saves {
			if err := s.applySave(ctx, sw); err != nil {
				return err
			}
		}
		for _, o := range w.owners {
			if err := s.applyItems(ctx, o.owner, o.destroyed, o.left, o.upserts); err != nil {
				return err
			}
		}
		if err := s.applyClans(ctx, &w); err != nil {
			return err
		}
		if len(w.deadCrests) > 0 {
			if _, err := s.crests.DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: w.deadCrests}}}}); err != nil {
				return err
			}
		}
		if len(w.crests) > 0 {
			if _, err := s.crests.BulkWrite(ctx, w.crests); err != nil {
				return err
			}
		}
		if len(w.relations) > 0 {
			if _, err := s.relations.BulkWrite(ctx, w.relations); err != nil {
				return err
			}
		}
		for _, p := range w.patches {
			// A patch to a character that does not exist is skipped.
			if _, err := s.chars.UpdateOne(ctx, bson.D{{Key: "_id", Value: p.objectID}}, p.update); err != nil {
				return err
			}
		}
		if err := s.applyCursedWeapons(ctx, &b); err != nil { // cursedweapons.go
			return err
		}
		if err := s.applyCastles(ctx, &b); err != nil { // residence.go
			return err
		}
		if err := s.applyClanHalls(ctx, &b); err != nil { // clanhalls.go
			return err
		}
		return s.applyManor(ctx, &b) // manor.go
	})
	if mongo.IsDuplicateKeyError(err) {
		return storage.ErrExists
	}
	return writeError("write batch", err)
}

// applyClans writes the clan part of a batch inside a transaction: the
// deleted clans with their warehouse items and the wars on them, the clan
// documents, and the alliance crests.
func (s *Characters) applyClans(ctx context.Context, w *batchWrite) error {
	if len(w.deletedClans) > 0 {
		in := bson.D{{Key: "$in", Value: w.deletedClans}}
		if _, err := s.clans.DeleteMany(ctx, bson.D{{Key: "_id", Value: in}}); err != nil {
			return err
		}
		if _, err := s.items.DeleteMany(ctx, bson.D{{Key: "owner_id", Value: in}}); err != nil {
			return err
		}
		_, err := s.clans.UpdateMany(ctx, bson.D{{Key: "warsDeclared", Value: in}},
			bson.D{{Key: "$pull", Value: bson.D{{Key: "warsDeclared", Value: in}}}})
		if err != nil {
			return err
		}
	}
	if len(w.clans) > 0 {
		if _, err := s.clans.BulkWrite(ctx, w.clans); err != nil {
			return err
		}
	}
	for _, a := range w.allyCrests {
		if a.AllyID == 0 {
			continue
		}
		_, err := s.clans.UpdateMany(ctx, bson.D{{Key: "allyId", Value: a.AllyID}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "allyCrestId", Value: a.CrestID}}}})
		if err != nil {
			return err
		}
	}
	return nil
}

// Clans returns the clan store backed by this database.
func (s *Store) Clans() *Clans {
	return &Clans{chars: s.Characters()}
}

// Clans is a storage.ClanStore on the "clans" collection, one document per
// clan (decision T1, Plan 12's data model), with the members read from
// "characters" and the warehouses from "items". Clans are written with
// Characters.WriteBatch.
type Clans struct {
	chars *Characters
}

var _ storage.ClanStore = (*Clans)(nil)

// LoadClans implements storage.ClanStore.
func (c *Clans) LoadClans(ctx context.Context) ([]storage.Clan, error) {
	ctx, cancel := context.WithTimeout(ctx, c.chars.store.timeout)
	defer cancel()
	cur, err := c.chars.clans.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load clans: %w", err)
	}
	var docs []clanDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load clans: %w", err)
	}
	return convertList(docs, func(d clanDoc) storage.Clan { return d.clan() }), nil
}

// GetClan implements storage.ClanStore.
func (c *Clans) GetClan(ctx context.Context, id int32) (storage.Clan, error) {
	ctx, cancel := context.WithTimeout(ctx, c.chars.store.timeout)
	defer cancel()
	var doc clanDoc
	err := c.chars.clans.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return storage.Clan{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.Clan{}, fmt.Errorf("mongostore: get clan: %w", err)
	}
	return doc.clan(), nil
}

// clanMemberDoc is the projection of a character document that ClanMembers
// reads.
type clanMemberDoc struct {
	ObjectID         int32  `bson:"_id"`
	ClanID           int32  `bson:"clan_id"`
	Name             string `bson:"name"`
	Level            int32  `bson:"level"`
	ClassID          int32  `bson:"class_id"`
	Race             int32  `bson:"race"`
	Sex              int32  `bson:"sex"`
	Title            string `bson:"title"`
	PowerGrade       int32  `bson:"power_grade"`
	PledgeType       int32  `bson:"pledge_type"`
	Apprentice       int32  `bson:"apprentice"`
	Sponsor          int32  `bson:"sponsor"`
	LvlJoinedAcademy int32  `bson:"lvl_joined_academy"`
	WantsPeace       bool   `bson:"wants_peace"`
}

// clanMemberProjection selects the fields of clanMemberDoc.
var clanMemberProjection = bson.D{
	{Key: "clan_id", Value: 1}, {Key: "name", Value: 1}, {Key: "level", Value: 1}, {Key: "class_id", Value: 1},
	{Key: "race", Value: 1}, {Key: "sex", Value: 1}, {Key: "title", Value: 1}, {Key: "power_grade", Value: 1},
	{Key: "pledge_type", Value: 1}, {Key: "apprentice", Value: 1}, {Key: "sponsor", Value: 1},
	{Key: "lvl_joined_academy", Value: 1}, {Key: "wants_peace", Value: 1},
}

// ClanMembers implements storage.ClanStore with a projection on the clan_id
// index. Clan IDs are object IDs, so every clan's ID is above 0.
func (c *Clans) ClanMembers(ctx context.Context, clanID int32) ([]storage.ClanMember, error) {
	filter := bson.D{{Key: "clan_id", Value: clanID}}
	if clanID == 0 {
		filter = bson.D{{Key: "clan_id", Value: bson.D{{Key: "$gt", Value: 0}}}}
	}
	ctx, cancel := context.WithTimeout(ctx, c.chars.store.timeout)
	defer cancel()
	opts := options.Find().SetProjection(clanMemberProjection).SetSort(bson.D{{Key: "clan_id", Value: 1}, {Key: "_id", Value: 1}})
	cur, err := c.chars.chars.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mongostore: find clan members: %w", err)
	}
	var docs []clanMemberDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: find clan members: %w", err)
	}
	return convertList(docs, func(d clanMemberDoc) storage.ClanMember { return storage.ClanMember(d) }), nil
}

// ClanItems implements storage.ClanStore.
func (c *Clans) ClanItems(ctx context.Context, clanID int32) ([]storage.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, c.chars.store.timeout)
	defer cancel()
	items, err := c.chars.findItems(ctx, bson.D{{Key: "owner_id", Value: clanID}})
	if err != nil {
		return nil, fmt.Errorf("mongostore: find clan items: %w", err)
	}
	return items[clanID], nil
}

// Crests returns the crest store backed by this database.
func (s *Store) Crests() *Crests {
	return &Crests{store: s, coll: s.db.Collection(crestsCollection), counters: s.db.Collection(countersCollection)}
}

// Crests is a storage.CrestStore on the "crests" collection (decision T3):
//
//	{_id: <crest ID>, type: <storage.CrestPledge, ...>, data: <binary>}
//
// with its own counter document in "counters", {_id: "crest_id", used},
// whose used is the last crest ID handed out. Crests are written with
// Characters.WriteBatch.
type Crests struct {
	store    *Store
	coll     *mongo.Collection
	counters *mongo.Collection
}

var _ storage.CrestStore = (*Crests)(nil)

var errCrestIDsExhausted = errors.New("mongostore: crest IDs exhausted")

// NextCrestID implements storage.CrestStore with one atomic $inc.
func (c *Crests) NextCrestID(ctx context.Context) (int32, error) {
	ctx, cancel := context.WithTimeout(ctx, c.store.timeout)
	defer cancel()
	filter := bson.D{{Key: "_id", Value: crestIDCounter}}
	update := bson.D{{Key: "$inc", Value: bson.D{{Key: "used", Value: int64(1)}}}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var doc counterDoc
	err := c.counters.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if mongo.IsDuplicateKeyError(err) {
		// Two first calls raced to insert the counter; see IDs.NextIDs.
		err = c.counters.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	}
	if err != nil {
		return 0, fmt.Errorf("mongostore: reserve a crest ID: %w", err)
	}
	if doc.Used > math.MaxInt32 {
		return 0, errCrestIDsExhausted
	}
	return int32(doc.Used), nil //nolint:gosec // range checked above
}

// LoadCrests implements storage.CrestStore.
func (c *Crests) LoadCrests(ctx context.Context) ([]storage.Crest, error) {
	ctx, cancel := context.WithTimeout(ctx, c.store.timeout)
	defer cancel()
	cur, err := c.coll.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load crests: %w", err)
	}
	var docs []crestDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load crests: %w", err)
	}
	return convertList(docs, func(d crestDoc) storage.Crest { return storage.Crest(d) }), nil
}

// Relations returns the relation store backed by this database.
func (s *Store) Relations() *Relations {
	return &Relations{store: s, coll: s.db.Collection(relationsCollection)}
}

// Relations is a storage.RelationStore on the "character_relations"
// collection (decision T4), the Mobius character_friends table:
//
//	{owner: <object ID>, target: <object ID>, relation: <0 friend, 1 block>}
//
// Relations are written with Characters.WriteBatch, and Characters.Delete
// removes a character's relations both ways.
type Relations struct {
	store *Store
	coll  *mongo.Collection
}

var _ storage.RelationStore = (*Relations)(nil)

// Relations implements storage.RelationStore.
func (r *Relations) Relations(ctx context.Context, owner int32) ([]storage.Relation, error) {
	return r.find(ctx, bson.D{{Key: "owner", Value: owner}}, bson.D{{Key: "relation", Value: 1}, {Key: "target", Value: 1}})
}

// RelationsTo implements storage.RelationStore.
func (r *Relations) RelationsTo(ctx context.Context, target int32) ([]storage.Relation, error) {
	return r.find(ctx, bson.D{{Key: "target", Value: target}}, bson.D{{Key: "relation", Value: 1}, {Key: "owner", Value: 1}})
}

func (r *Relations) find(ctx context.Context, filter, sort bson.D) ([]storage.Relation, error) {
	ctx, cancel := context.WithTimeout(ctx, r.store.timeout)
	defer cancel()
	cur, err := r.coll.Find(ctx, filter, options.Find().SetSort(sort))
	if err != nil {
		return nil, fmt.Errorf("mongostore: find relations: %w", err)
	}
	var docs []relationDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: find relations: %w", err)
	}
	return convertList(docs, relationDoc.relation), nil
}
