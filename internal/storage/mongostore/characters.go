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

const charactersCollection = "characters"

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

// Characters returns the character store backed by this database.
func (s *Store) Characters() *Characters {
	return &Characters{coll: s.db.Collection(charactersCollection), timeout: s.timeout}
}

// Characters is a storage.CharacterStore on the "characters" collection.
// Each character is one document with its items embedded, so a character and
// its starting items are written in one atomic insert. A standalone MongoDB
// has no multi-document transactions. Items may move to their own collection
// later.
type Characters struct {
	coll    *mongo.Collection
	timeout time.Duration
}

var _ storage.CharacterStore = (*Characters)(nil)

// characterDoc is the stored form of storage.Character.
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
	Items       []itemDoc  `bson:"items"`
}

// itemDoc is the stored form of storage.Item.
type itemDoc struct {
	ObjectID int32 `bson:"object_id"`
	ItemID   int32 `bson:"item_id"`
	Count    int64 `bson:"count"`
	Enchant  int32 `bson:"enchant"`
	Slot     int8  `bson:"slot"` // -1 if not equipped
}

func toCharacterDoc(c *storage.Character) characterDoc {
	items := make([]itemDoc, len(c.Items))
	for i, it := range c.Items {
		items[i] = itemDoc{
			ObjectID: it.ObjectID,
			ItemID:   it.ItemID,
			Count:    it.Count,
			Enchant:  it.Enchant,
			Slot:     int8(it.Slot),
		}
	}
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
		Items:       items,
	}
}

func (d *characterDoc) character() storage.Character {
	var items []storage.Item
	if len(d.Items) > 0 {
		items = make([]storage.Item, len(d.Items))
		for i, it := range d.Items {
			items[i] = storage.Item{
				ObjectID: it.ObjectID,
				ItemID:   it.ItemID,
				Count:    it.Count,
				Enchant:  it.Enchant,
				Slot:     storage.PaperdollSlot(it.Slot),
			}
		}
	}
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

// Create implements storage.CharacterStore.
func (s *Characters) Create(ctx context.Context, c *storage.Character) error {
	if c.CreatedAt.IsZero() {
		// MongoDB keeps milliseconds; truncate so c matches what Get returns.
		c.CreatedAt = time.Now().UTC().Truncate(time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	_, err := s.coll.InsertOne(ctx, toCharacterDoc(c))
	if mongo.IsDuplicateKeyError(err) {
		return storage.ErrExists
	}
	if err != nil {
		return fmt.Errorf("mongostore: create character: %w", err)
	}
	return nil
}

// ListByAccount implements storage.CharacterStore.
func (s *Characters) ListByAccount(ctx context.Context, account string) ([]storage.Character, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}})
	cur, err := s.coll.Find(ctx, bson.D{{Key: "account", Value: account}}, opts)
	if err != nil {
		return nil, fmt.Errorf("mongostore: list characters: %w", err)
	}
	var docs []characterDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: list characters: %w", err)
	}
	var list []storage.Character
	for i := range docs {
		list = append(list, docs[i].character())
	}
	return list, nil
}

// CountByAccount implements storage.CharacterStore.
func (s *Characters) CountByAccount(ctx context.Context, account string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	n, err := s.coll.CountDocuments(ctx, bson.D{{Key: "account", Value: account}})
	if err != nil {
		return 0, fmt.Errorf("mongostore: count characters: %w", err)
	}
	return int(n), nil
}

// NameExists implements storage.CharacterStore.
func (s *Characters) NameExists(ctx context.Context, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	n, err := s.coll.CountDocuments(ctx,
		bson.D{{Key: "name_lower", Value: storage.NameKey(name)}},
		options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("mongostore: check character name: %w", err)
	}
	return n > 0, nil
}

// Get implements storage.CharacterStore.
func (s *Characters) Get(ctx context.Context, objectID int32) (storage.Character, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var doc characterDoc
	err := s.coll.FindOne(ctx, bson.D{{Key: "_id", Value: objectID}}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return storage.Character{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.Character{}, fmt.Errorf("mongostore: get character: %w", err)
	}
	return doc.character(), nil
}
