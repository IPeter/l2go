// Package mongostore implements the storage interfaces on MongoDB.
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

// DefaultTimeout bounds every database call.
const DefaultTimeout = 5 * time.Second

const accountsCollection = "accounts"

// Store is a MongoDB connection with access to one database.
type Store struct {
	client  *mongo.Client
	db      *mongo.Database
	timeout time.Duration
}

// Open connects to the MongoDB deployment at uri, checks that it answers,
// and prepares the indexes in database dbName.
func Open(ctx context.Context, uri, dbName string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(DefaultTimeout))
	if err != nil {
		return nil, fmt.Errorf("mongostore: connect: %w", err)
	}
	s := &Store{client: client, db: client.Database(dbName), timeout: DefaultTimeout}

	pingCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = s.Close(ctx)
		return nil, fmt.Errorf("mongostore: ping: %w", err)
	}
	if err := s.ensureIndexes(ctx); err != nil {
		_ = s.Close(ctx)
		return nil, err
	}
	return s, nil
}

// Close disconnects from the deployment.
func (s *Store) Close(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := s.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongostore: disconnect: %w", err)
	}
	return nil
}

// Drop deletes the whole database. It is meant for tests.
func (s *Store) Drop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.db.Drop(ctx)
}

func (s *Store) ensureIndexes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	_, err := s.db.Collection(accountsCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("mongostore: create accounts index: %w", err)
	}
	return nil
}

// Accounts returns the account store backed by this database.
func (s *Store) Accounts() *Accounts {
	return &Accounts{coll: s.db.Collection(accountsCollection), timeout: s.timeout}
}

// Accounts is a storage.AccountStore on the "accounts" collection.
type Accounts struct {
	coll    *mongo.Collection
	timeout time.Duration
}

var _ storage.AccountStore = (*Accounts)(nil)

// accountDoc is the stored form. Field names match the documents written by
// earlier L2Go versions.
type accountDoc struct {
	ID          bson.ObjectID `bson:"_id"`
	Username    string        `bson:"username"`
	Password    string        `bson:"password"`
	AccessLevel int8          `bson:"access_level"`
}

// FindByUsername implements storage.AccountStore.
func (a *Accounts) FindByUsername(ctx context.Context, username string) (storage.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	var doc accountDoc
	err := a.coll.FindOne(ctx, bson.D{{Key: "username", Value: username}}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return storage.Account{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.Account{}, fmt.Errorf("mongostore: find account: %w", err)
	}
	return storage.Account{
		ID:          doc.ID.Hex(),
		Username:    doc.Username,
		Password:    doc.Password,
		AccessLevel: storage.AccessLevel(doc.AccessLevel),
	}, nil
}

// Create implements storage.AccountStore.
func (a *Accounts) Create(ctx context.Context, acc storage.Account) error {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	_, err := a.coll.InsertOne(ctx, accountDoc{
		ID:          bson.NewObjectID(),
		Username:    acc.Username,
		Password:    acc.Password,
		AccessLevel: int8(acc.AccessLevel),
	})
	if mongo.IsDuplicateKeyError(err) {
		return storage.ErrExists
	}
	if err != nil {
		return fmt.Errorf("mongostore: create account: %w", err)
	}
	return nil
}
