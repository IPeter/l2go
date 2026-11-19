package mongostore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

const (
	charactersCollection = "characters"
	itemsCollection      = "items"
)

// characterIndexes are the indexes of the characters collection.
//
// Names are unique ignoring case through name_lower, a stored copy of
// storage.NameKey(name) with a unique index. We use a stored field instead of
// a case-insensitive collation (strength 2) because:
//   - Lookups and the duplicate check are plain equality matches on a normal
//     index. With a collation, every query must repeat the same collation or
//     it silently skips the index and compares case-sensitively.
//   - memstore uses the same Go function, so both stores agree exactly on
//     which names are the same. ICU collation rules can differ from Go's.
//
// The account index also covers the sort order of ListByAccount.
func characterIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name_lower", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "account", Value: 1}, {Key: "created_at", Value: 1}, {Key: "_id", Value: 1}},
		},
	}
}

// itemIndexes are the indexes of the items collection (decision I1): an
// owner's items, also by where they are kept. The object ID is _id.
func itemIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{
		{Keys: bson.D{{Key: "owner_id", Value: 1}, {Key: "loc", Value: 1}}},
	}
}

// Characters returns the character store backed by this database.
func (s *Store) Characters() *Characters {
	return &Characters{
		store: s,
		chars: s.db.Collection(charactersCollection),
		items: s.db.Collection(itemsCollection),
	}
}

// Characters is a storage.CharacterStore and storage.CharacterSaver on two
// collections: "characters" with one document per character, and "items"
// with one document per item, the Mobius items table (decision I1):
//
//	{_id: <object ID>, owner_id, item_id, count, enchant_level, loc,
//	 loc_data, custom_type1, custom_type2, mana_left}
//
// Every write that touches both runs in one transaction. Reads that need
// both run in a transaction too, so they see a character and its items as
// one write left them.
type Characters struct {
	store *Store
	chars *mongo.Collection
	items *mongo.Collection
}

var (
	_ storage.CharacterStore = (*Characters)(nil)
	_ storage.CharacterSaver = (*Characters)(nil)
)

// characterDoc is the stored form of storage.Character without its items.
type characterDoc struct {
	ObjectID    int32      `bson:"_id"`
	Account     string     `bson:"account"`
	Name        string     `bson:"name"`
	NameLower   string     `bson:"name_lower"` // storage.NameKey(Name), unique
	Race        int32      `bson:"race"`
	Sex         int32      `bson:"sex"`
	ClassID     int32      `bson:"class_id"`
	BaseClassID int32      `bson:"base_class_id"`
	HairStyle   int32      `bson:"hair_style"`
	HairColor   int32      `bson:"hair_color"`
	Face        int32      `bson:"face"`
	Level       int32      `bson:"level"`
	Exp         int64      `bson:"exp"`
	SP          int32      `bson:"sp"`
	Karma       int32      `bson:"karma"`
	PvPKills    int32      `bson:"pvp_kills"`
	PKKills     int32      `bson:"pk_kills"`
	ClanID      int32      `bson:"clan_id"`
	CurHP       float64    `bson:"cur_hp"`
	MaxHP       float64    `bson:"max_hp"`
	CurMP       float64    `bson:"cur_mp"`
	MaxMP       float64    `bson:"max_mp"`
	CurCP       float64    `bson:"cur_cp"`
	MaxCP       float64    `bson:"max_cp"`
	X           int32      `bson:"x"`
	Y           int32      `bson:"y"`
	Z           int32      `bson:"z"`
	Heading     int32      `bson:"heading"`
	AccessLevel int8       `bson:"access_level"`
	DeleteTime  *time.Time `bson:"delete_time,omitempty"` // absent if not deleting
	CreatedAt   time.Time  `bson:"created_at"`
	LastAccess  *time.Time `bson:"last_access,omitempty"` // absent if never entered the world
}

// itemDoc is the stored form of storage.Item, with the column names of the
// Mobius items table.
type itemDoc struct {
	ObjectID    int32  `bson:"_id"`
	OwnerID     int32  `bson:"owner_id"`
	ItemID      int32  `bson:"item_id"`
	Count       int64  `bson:"count"`
	Enchant     int32  `bson:"enchant_level"`
	Loc         string `bson:"loc"`      // a storage.ItemLocation
	LocData     int32  `bson:"loc_data"` // the paperdoll slot of a worn item, else 0 as in Mobius
	CustomType1 int32  `bson:"custom_type1"`
	CustomType2 int32  `bson:"custom_type2"`
	ManaLeft    int32  `bson:"mana_left"` // storage.NoMana if none
}

func toCharacterDoc(c *storage.Character) characterDoc {
	return characterDoc{
		ObjectID:    c.ObjectID,
		Account:     c.Account,
		Name:        c.Name,
		NameLower:   storage.NameKey(c.Name),
		Race:        c.Race,
		Sex:         c.Sex,
		ClassID:     c.ClassID,
		BaseClassID: c.BaseClassID,
		HairStyle:   c.HairStyle,
		HairColor:   c.HairColor,
		Face:        c.Face,
		Level:       c.Level,
		Exp:         c.Exp,
		SP:          c.SP,
		Karma:       c.Karma,
		PvPKills:    c.PvPKills,
		PKKills:     c.PKKills,
		ClanID:      c.ClanID,
		CurHP:       c.CurHP,
		MaxHP:       c.MaxHP,
		CurMP:       c.CurMP,
		MaxMP:       c.MaxMP,
		CurCP:       c.CurCP,
		MaxCP:       c.MaxCP,
		X:           c.X,
		Y:           c.Y,
		Z:           c.Z,
		Heading:     c.Heading,
		AccessLevel: int8(c.AccessLevel),
		DeleteTime:  optionalTime(c.DeleteTime),
		CreatedAt:   c.CreatedAt,
		LastAccess:  optionalTime(c.LastAccess),
	}
}

// character returns the character with items, which are already in object
// ID order. A character without items gets a nil Items slice.
func (d *characterDoc) character(items []storage.Item) storage.Character {
	return storage.Character{
		ObjectID:    d.ObjectID,
		Account:     d.Account,
		Name:        d.Name,
		Race:        d.Race,
		Sex:         d.Sex,
		ClassID:     d.ClassID,
		BaseClassID: d.BaseClassID,
		HairStyle:   d.HairStyle,
		HairColor:   d.HairColor,
		Face:        d.Face,
		Level:       d.Level,
		Exp:         d.Exp,
		SP:          d.SP,
		Karma:       d.Karma,
		PvPKills:    d.PvPKills,
		PKKills:     d.PKKills,
		ClanID:      d.ClanID,
		CurHP:       d.CurHP,
		MaxHP:       d.MaxHP,
		CurMP:       d.CurMP,
		MaxMP:       d.MaxMP,
		CurCP:       d.CurCP,
		MaxCP:       d.MaxCP,
		X:           d.X,
		Y:           d.Y,
		Z:           d.Z,
		Heading:     d.Heading,
		AccessLevel: storage.AccessLevel(d.AccessLevel),
		DeleteTime:  timeOrZero(d.DeleteTime),
		CreatedAt:   d.CreatedAt,
		LastAccess:  timeOrZero(d.LastAccess),
		Items:       items,
	}
}

// toItemDoc returns it, normalized, as owner's item document.
func toItemDoc(owner int32, it storage.Item) itemDoc {
	it = it.Normalized()
	d := itemDoc{
		ObjectID:    it.ObjectID,
		OwnerID:     owner,
		ItemID:      it.ItemID,
		Count:       it.Count,
		Enchant:     it.Enchant,
		Loc:         string(it.Loc),
		CustomType1: it.CustomType1,
		CustomType2: it.CustomType2,
		ManaLeft:    it.Mana,
	}
	if it.Loc == storage.LocPaperdoll {
		d.LocData = int32(it.Slot)
	}
	return d
}

// item returns the stored item, normalized. loc_data counts only for a worn
// item; a slot out of range, which L2Go never writes, makes it a carried one.
func (d *itemDoc) item() storage.Item {
	it := storage.Item{
		ObjectID:    d.ObjectID,
		ItemID:      d.ItemID,
		Count:       d.Count,
		Enchant:     d.Enchant,
		Slot:        storage.PaperdollNone,
		Loc:         storage.ItemLocation(d.Loc),
		Mana:        d.ManaLeft,
		CustomType1: d.CustomType1,
		CustomType2: d.CustomType2,
	}
	if it.Loc == storage.LocPaperdoll && d.LocData >= 0 && d.LocData < storage.PaperdollTotalSlots {
		it.Slot = storage.PaperdollSlot(d.LocData) //nolint:gosec // range checked above
	}
	return it.Normalized()
}

// itemUpserts returns the writes that store items with owner as their owner:
// a replace by object ID that inserts the item if it is missing, and takes
// it from another owner if it has one.
func itemUpserts(owner int32, items []storage.Item) []mongo.WriteModel {
	models := make([]mongo.WriteModel, len(items))
	for i, it := range items {
		models[i] = mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: it.ObjectID}}).
			SetReplacement(toItemDoc(owner, it)).
			SetUpsert(true)
	}
	return models
}

// optionalTime returns nil for the zero time, so the field is left out.
func optionalTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func timeOrZero(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// Create implements storage.CharacterStore. The character and its items are
// inserted in one transaction.
func (s *Characters) Create(ctx context.Context, c *storage.Character) error {
	if c.CreatedAt.IsZero() {
		// MongoDB keeps milliseconds; truncate so c matches what Get returns.
		c.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	}
	doc := toCharacterDoc(c)
	items := make([]itemDoc, len(c.Items))
	for i, it := range c.Items {
		items[i] = toItemDoc(c.ObjectID, it)
	}
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	err := s.store.write(ctx, func(ctx context.Context) error {
		if _, err := s.chars.InsertOne(ctx, doc); err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		_, err := s.items.InsertMany(ctx, items)
		return err
	})
	if mongo.IsDuplicateKeyError(err) {
		return storage.ErrExists
	}
	if err != nil {
		return fmt.Errorf("mongostore: create character: %w", err)
	}
	return nil
}

// findItems returns the items that match filter by owner, each owner's in
// object ID order.
func (s *Characters) findItems(ctx context.Context, filter bson.D) (map[int32][]storage.Item, error) {
	cur, err := s.items.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var docs []itemDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	byOwner := make(map[int32][]storage.Item)
	for i := range docs {
		byOwner[docs[i].OwnerID] = append(byOwner[docs[i].OwnerID], docs[i].item())
	}
	return byOwner, nil
}

// ListByAccount implements storage.CharacterStore.
func (s *Characters) ListByAccount(ctx context.Context, account string) ([]storage.Character, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	var list []storage.Character
	err := s.store.read(ctx, func(ctx context.Context) error {
		list = nil
		opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}})
		cur, err := s.chars.Find(ctx, bson.D{{Key: "account", Value: account}}, opts)
		if err != nil {
			return err
		}
		var docs []characterDoc
		if err := cur.All(ctx, &docs); err != nil {
			return err
		}
		if len(docs) == 0 {
			return nil
		}
		ids := make([]int32, len(docs))
		for i := range docs {
			ids[i] = docs[i].ObjectID
		}
		items, err := s.findItems(ctx, bson.D{{Key: "owner_id", Value: bson.D{{Key: "$in", Value: ids}}}})
		if err != nil {
			return err
		}
		for i := range docs {
			list = append(list, docs[i].character(items[docs[i].ObjectID]))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("mongostore: list characters: %w", err)
	}
	return list, nil
}

// CountByAccount implements storage.CharacterStore.
func (s *Characters) CountByAccount(ctx context.Context, account string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	n, err := s.chars.CountDocuments(ctx, bson.D{{Key: "account", Value: account}})
	if err != nil {
		return 0, fmt.Errorf("mongostore: count characters: %w", err)
	}
	return int(n), nil
}

// NameExists implements storage.CharacterStore.
func (s *Characters) NameExists(ctx context.Context, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	n, err := s.chars.CountDocuments(ctx,
		bson.D{{Key: "name_lower", Value: storage.NameKey(name)}},
		options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("mongostore: check character name: %w", err)
	}
	return n > 0, nil
}

// Get implements storage.CharacterStore.
func (s *Characters) Get(ctx context.Context, objectID int32) (storage.Character, error) {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	var c storage.Character
	err := s.store.read(ctx, func(ctx context.Context) error {
		var doc characterDoc
		if err := s.chars.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}).Decode(&doc); err != nil {
			return err
		}
		items, err := s.findItems(ctx, bson.D{{Key: "owner_id", Value: objectID}})
		if err != nil {
			return err
		}
		c = doc.character(items[objectID])
		return nil
	})
	if errors.Is(err, mongo.ErrNoDocuments) {
		return storage.Character{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.Character{}, fmt.Errorf("mongostore: get character: %w", err)
	}
	return c, nil
}

// characterUpdate returns the update that stores the fields Update saves.
// The identity fields (_id, account, name, name_lower, race, created_at) are
// not written.
func characterUpdate(c *storage.Character) bson.D {
	doc := toCharacterDoc(c)
	set := bson.D{
		{Key: "sex", Value: doc.Sex},
		{Key: "class_id", Value: doc.ClassID},
		{Key: "base_class_id", Value: doc.BaseClassID},
		{Key: "hair_style", Value: doc.HairStyle},
		{Key: "hair_color", Value: doc.HairColor},
		{Key: "face", Value: doc.Face},
		{Key: "level", Value: doc.Level},
		{Key: "exp", Value: doc.Exp},
		{Key: "sp", Value: doc.SP},
		{Key: "karma", Value: doc.Karma},
		{Key: "pvp_kills", Value: doc.PvPKills},
		{Key: "pk_kills", Value: doc.PKKills},
		{Key: "clan_id", Value: doc.ClanID},
		{Key: "cur_hp", Value: doc.CurHP},
		{Key: "max_hp", Value: doc.MaxHP},
		{Key: "cur_mp", Value: doc.CurMP},
		{Key: "max_mp", Value: doc.MaxMP},
		{Key: "cur_cp", Value: doc.CurCP},
		{Key: "max_cp", Value: doc.MaxCP},
		{Key: "x", Value: doc.X},
		{Key: "y", Value: doc.Y},
		{Key: "z", Value: doc.Z},
		{Key: "heading", Value: doc.Heading},
		{Key: "access_level", Value: doc.AccessLevel},
	}
	var unset bson.D
	for _, f := range []struct {
		key string
		t   *time.Time
	}{
		{"delete_time", doc.DeleteTime},
		{"last_access", doc.LastAccess},
	} {
		if f.t != nil {
			set = append(set, bson.E{Key: f.key, Value: *f.t})
		} else {
			unset = append(unset, bson.E{Key: f.key, Value: ""})
		}
	}
	update := bson.D{{Key: "$set", Value: set}}
	if len(unset) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unset})
	}
	return update
}

// updateFields applies a characterUpdate inside a transaction. It returns
// storage.ErrNotFound if there is no such character.
func (s *Characters) updateFields(ctx context.Context, objectID int32, update bson.D) error {
	res, err := s.chars.UpdateOne(ctx, bson.D{{Key: "_id", Value: objectID}}, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// upsertItems runs itemUpserts inside a transaction.
func (s *Characters) upsertItems(ctx context.Context, models []mongo.WriteModel) error {
	if len(models) == 0 {
		return nil
	}
	_, err := s.items.BulkWrite(ctx, models)
	return err
}

// Update implements storage.CharacterStore: in one transaction it stores the
// character's fields, deletes the items it owns that c.Items does not list,
// and upserts c.Items.
func (s *Characters) Update(ctx context.Context, c *storage.Character) error {
	update := characterUpdate(c)
	keep := make([]int32, len(c.Items)) // never nil: $nin needs an array
	for i, it := range c.Items {
		keep[i] = it.ObjectID
	}
	upserts := itemUpserts(c.ObjectID, c.Items)
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	err := s.store.write(ctx, func(ctx context.Context) error {
		if err := s.updateFields(ctx, c.ObjectID, update); err != nil {
			return err
		}
		_, err := s.items.DeleteMany(ctx, bson.D{
			{Key: "owner_id", Value: c.ObjectID},
			{Key: "_id", Value: bson.D{{Key: "$nin", Value: keep}}},
		})
		if err != nil {
			return err
		}
		return s.upsertItems(ctx, upserts)
	})
	return writeError("update character", err)
}

// Save implements storage.CharacterSaver: in one transaction it stores the
// character's fields, deletes the destroyed items, deletes the items that
// left while the character still owns them, and upserts the changed ones.
func (s *Characters) Save(ctx context.Context, c *storage.Character, changes storage.ItemChanges) error {
	update := characterUpdate(c)
	upserts := itemUpserts(c.ObjectID, changes.Upserts)
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	err := s.store.write(ctx, func(ctx context.Context) error {
		if err := s.updateFields(ctx, c.ObjectID, update); err != nil {
			return err
		}
		if len(changes.Destroyed) > 0 {
			_, err := s.items.DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: changes.Destroyed}}}})
			if err != nil {
				return err
			}
		}
		if len(changes.Left) > 0 {
			_, err := s.items.DeleteMany(ctx, bson.D{
				{Key: "_id", Value: bson.D{{Key: "$in", Value: changes.Left}}},
				{Key: "owner_id", Value: c.ObjectID},
			})
			if err != nil {
				return err
			}
		}
		return s.upsertItems(ctx, upserts)
	})
	return writeError("save character", err)
}

// Delete implements storage.CharacterStore: the character and all its items
// go in one transaction.
func (s *Characters) Delete(ctx context.Context, objectID int32) error {
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	err := s.store.write(ctx, func(ctx context.Context) error {
		res, err := s.chars.DeleteOne(ctx, bson.D{{Key: "_id", Value: objectID}})
		if err != nil {
			return err
		}
		if res.DeletedCount == 0 {
			return storage.ErrNotFound
		}
		_, err = s.items.DeleteMany(ctx, bson.D{{Key: "owner_id", Value: objectID}})
		return err
	})
	return writeError("delete character", err)
}

// writeError returns the error of a write transaction: storage.ErrNotFound
// as it is, anything else wrapped.
func writeError(what string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, storage.ErrNotFound):
		return storage.ErrNotFound
	default:
		return fmt.Errorf("mongostore: %s: %w", what, err)
	}
}

// SetDeleteTime implements storage.CharacterStore. The zero time removes
// the delete_time field.
func (s *Characters) SetDeleteTime(ctx context.Context, objectID int32, t time.Time) error {
	update := bson.D{{Key: "$unset", Value: bson.D{{Key: "delete_time", Value: ""}}}}
	if !t.IsZero() {
		update = bson.D{{Key: "$set", Value: bson.D{{Key: "delete_time", Value: t}}}}
	}
	ctx, cancel := context.WithTimeout(ctx, s.store.timeout)
	defer cancel()
	res, err := s.chars.UpdateOne(ctx, bson.D{{Key: "_id", Value: objectID}}, update)
	if err != nil {
		return fmt.Errorf("mongostore: set character delete time: %w", err)
	}
	if res.MatchedCount == 0 {
		return storage.ErrNotFound
	}
	return nil
}
