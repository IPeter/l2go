package mongostore

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

const (
	countersCollection = "counters"
	objectIDCounter    = "object_id" // _id of the object ID counter document
)

// IDs returns the object ID allocator backed by this database.
func (s *Store) IDs() *IDs {
	return &IDs{coll: s.db.Collection(countersCollection), timeout: s.timeout}
}

// IDs is a storage.IDAllocator on one document of the "counters" collection:
//
//	{_id: "object_id", used: <how many IDs were handed out>}
//
// NextIDs reserves n IDs with one atomic $inc of used, so concurrent callers
// and servers never get the same ID. The reserved IDs are
// storage.FirstObjectID+used-n up to storage.FirstObjectID+used-1. The first
// call creates the document (upsert).
type IDs struct {
	coll    *mongo.Collection
	timeout time.Duration
}

var _ storage.IDAllocator = (*IDs)(nil)

type counterDoc struct {
	Used int64 `bson:"used"`
}

var errIDsExhausted = errors.New("mongostore: object IDs exhausted")

// NextID implements storage.IDAllocator.
func (a *IDs) NextID(ctx context.Context) (int32, error) {
	ids, err := a.NextIDs(ctx, 1)
	if err != nil {
		return 0, err
	}
	return ids[0], nil
}

// NextIDs implements storage.IDAllocator.
func (a *IDs) NextIDs(ctx context.Context, n int) ([]int32, error) {
	if n < 1 {
		return nil, fmt.Errorf("mongostore: NextIDs(%d): n must be at least 1", n)
	}
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	filter := bson.D{{Key: "_id", Value: objectIDCounter}}
	update := bson.D{{Key: "$inc", Value: bson.D{{Key: "used", Value: int64(n)}}}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var doc counterDoc
	err := a.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if mongo.IsDuplicateKeyError(err) {
		// Two first calls raced to insert the counter and the other one won.
		// The server normally retries this itself; the document exists now,
		// so one retry is enough.
		err = a.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	}
	if err != nil {
		return nil, fmt.Errorf("mongostore: reserve object IDs: %w", err)
	}
	first := int64(storage.FirstObjectID) + doc.Used - int64(n)
	if first+int64(n)-1 > math.MaxInt32 {
		return nil, errIDsExhausted
	}
	ids := make([]int32, n)
	for i := range ids {
		ids[i] = int32(first + int64(i)) //nolint:gosec // range checked against math.MaxInt32 above
	}
	return ids, nil
}
