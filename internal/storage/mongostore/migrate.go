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

// legacyItemDoc is an item as L2Go kept it inside its character document,
// in an array field "items", before items got a collection of their own in
// Milestone 5.
type legacyItemDoc struct {
	ObjectID int32 `bson:"object_id"`
	ItemID   int32 `bson:"item_id"`
	Count    int64 `bson:"count"`
	Enchant  int32 `bson:"enchant"`
	Slot     int32 `bson:"slot"` // the paperdoll slot, -1 if not worn
}

// item returns the item as the items collection stores it. Items inside a
// character were always carried and had no mana or custom types.
func (l *legacyItemDoc) item() storage.Item {
	slot := storage.PaperdollNone
	if l.Slot >= 0 && l.Slot < storage.PaperdollTotalSlots {
		slot = storage.PaperdollSlot(l.Slot) //nolint:gosec // range checked above
	}
	return storage.Item{
		ObjectID: l.ObjectID,
		ItemID:   l.ItemID,
		Count:    l.Count,
		Enchant:  l.Enchant,
		Slot:     slot,
		Mana:     storage.NoMana,
	}.Normalized()
}

// embeddedItems matches the characters that still keep their items inside.
func embeddedItems() bson.E {
	return bson.E{Key: "items", Value: bson.D{{Key: "$exists", Value: true}}}
}

// migrateEmbeddedItems moves the items that L2Go versions before Milestone 5
// kept inside each character document into the items collection (decision
// I1), and returns how many characters and items it moved.
//
// Each character moves in a transaction of its own, which inserts its items
// and removes its items field together. A crash leaves every character
// either moved or untouched, and running the migration again moves only what
// is left: after one complete run it finds nothing to do. Nothing is ever
// overwritten: an item whose object ID is already in the items collection
// stops the migration with an error, and that character keeps its items
// inside until the conflict is resolved by hand.
func (s *Store) migrateEmbeddedItems(ctx context.Context) (chars, items int, err error) {
	ids, err := s.charactersWithEmbeddedItems(ctx)
	if err != nil {
		return 0, 0, err
	}
	for _, id := range ids {
		moved, n, err := s.migrateCharacter(ctx, id)
		if mongo.IsDuplicateKeyError(err) {
			err = fmt.Errorf("an item's object ID is already in the items collection, so the character keeps "+
				"its items inside until that is resolved: %w", err)
		}
		if err != nil {
			return chars, items, fmt.Errorf("mongostore: move the items of character %d into the items collection: %w", id, err)
		}
		if moved {
			chars++
			items += n
		}
	}
	return chars, items, nil
}

// charactersWithEmbeddedItems returns the object IDs of the characters that
// still have an items field, in object ID order.
func (s *Store) charactersWithEmbeddedItems(ctx context.Context) ([]int32, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	opts := options.Find().
		SetProjection(bson.D{{Key: "_id", Value: 1}}).
		SetSort(bson.D{{Key: "_id", Value: 1}})
	cur, err := s.db.Collection(charactersCollection).Find(ctx, bson.D{embeddedItems()}, opts)
	if err != nil {
		return nil, fmt.Errorf("mongostore: find characters that keep their items inside: %w", err)
	}
	var docs []struct {
		ID int32 `bson:"_id"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: find characters that keep their items inside: %w", err)
	}
	ids := make([]int32, len(docs))
	for i, d := range docs {
		ids[i] = d.ID
	}
	return ids, nil
}

// migrateCharacter moves the items of one character in one transaction. It
// reports false if the character has no items field (any more), and the
// number of items it moved.
func (s *Store) migrateCharacter(ctx context.Context, id int32) (moved bool, n int, err error) {
	chars := s.db.Collection(charactersCollection)
	itemColl := s.db.Collection(itemsCollection)
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	err = s.write(ctx, func(ctx context.Context) error {
		moved, n = false, 0
		var doc struct {
			Items []legacyItemDoc `bson:"items"`
		}
		err := chars.FindOne(ctx,
			bson.D{{Key: "_id", Value: id}, embeddedItems()},
			options.FindOne().SetProjection(bson.D{{Key: "items", Value: 1}}),
		).Decode(&doc)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil // moved meanwhile, by another server starting up
		}
		if err != nil {
			return err
		}
		if len(doc.Items) > 0 {
			docs := make([]itemDoc, len(doc.Items))
			for i := range doc.Items {
				docs[i] = toItemDoc(id, doc.Items[i].item())
			}
			if _, err := itemColl.InsertMany(ctx, docs); err != nil {
				return err
			}
		}
		_, err = chars.UpdateOne(ctx,
			bson.D{{Key: "_id", Value: id}},
			bson.D{{Key: "$unset", Value: bson.D{{Key: "items", Value: ""}}}})
		if err != nil {
			return err
		}
		moved, n = true, len(doc.Items)
		return nil
	})
	return moved, n, err
}
