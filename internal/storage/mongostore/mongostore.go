// Package mongostore implements the storage interfaces on MongoDB.
//
// Collections:
//   - accounts: one document per login account, unique on username.
//   - characters: one document per character, unique on name_lower,
//     indexed on account (see Characters).
//   - items: one document per item with its owner, indexed on owner_id and
//     loc (see Characters).
//   - counters: the object ID counter (see IDs).
//
// A character and its items are written in one transaction, so the
// deployment must be a replica set (a one-node one is enough, as in docker
// compose) or a sharded cluster. Open refuses a standalone server.
package mongostore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

// DefaultTimeout bounds every database call, and every transaction.
const DefaultTimeout = 5 * time.Second

const accountsCollection = "accounts"

// Store is a MongoDB connection with access to one database.
type Store struct {
	client  *mongo.Client
	db      *mongo.Database
	timeout time.Duration
	log     *slog.Logger

	// fault, when armed, fails the next write transaction just before it
	// commits. Tests in this package use it to check that a failed write
	// leaves nothing behind.
	fault faultHook
}

// Option changes how Open sets up a Store.
type Option func(*Store)

// WithLogger sets the logger for what the store reports, such as the item
// migration in Open. The default is slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(s *Store) {
		if l != nil {
			s.log = l
		}
	}
}

// Open connects to the MongoDB deployment at uri, checks that it answers and
// can run transactions, prepares the indexes in database dbName, and moves
// items that older versions kept inside the character documents into the
// items collection (see migrateEmbeddedItems).
//
// From the host, the docker compose deployment needs directConnection=true
// in uri: its replica set names its member mongo:27017, which only resolves
// inside compose.
func Open(ctx context.Context, uri, dbName string, opts ...Option) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetServerSelectionTimeout(DefaultTimeout))
	if err != nil {
		return nil, fmt.Errorf("mongostore: connect: %w", err)
	}
	s := &Store{client: client, db: client.Database(dbName), timeout: DefaultTimeout, log: slog.Default()}
	for _, opt := range opts {
		opt(s)
	}
	if err := s.setUp(ctx); err != nil {
		_ = s.Close(ctx)
		return nil, err
	}
	return s, nil
}

// setUp runs the checks and preparations of Open.
func (s *Store) setUp(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := s.client.Ping(pingCtx, nil); err != nil {
		return fmt.Errorf("mongostore: ping: %w", err)
	}
	if err := s.checkTransactions(pingCtx); err != nil {
		return err
	}
	if err := s.ensureIndexes(ctx); err != nil {
		return err
	}
	chars, items, err := s.migrateEmbeddedItems(ctx)
	if err != nil {
		return err
	}
	if chars > 0 {
		s.log.Info("storage: moved the items stored inside characters into the items collection",
			"db", s.db.Name(), "characters", chars, "items", items)
	} else {
		s.log.Debug("storage: no items stored inside characters to move", "db", s.db.Name())
	}
	return nil
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

// helloReply is the part of the hello command's reply that tells what kind
// of deployment the connection reached.
type helloReply struct {
	SetName           string `bson:"setName"`           // the replica set's name, on a member of an initiated set
	IsReplicaSet      bool   `bson:"isreplicaset"`      // a member started with --replSet whose set is not initiated yet
	Msg               string `bson:"msg"`               // "isdbgrid" from a mongos
	IsWritablePrimary bool   `bson:"isWritablePrimary"` // the server takes writes
}

// checkTransactions returns an error that says what to do unless the
// deployment can run the transactions the character store needs.
func (s *Store) checkTransactions(ctx context.Context) error {
	var hello helloReply
	if err := s.client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		return fmt.Errorf("mongostore: hello: %w", err)
	}
	return hello.transactionsError()
}

// transactionsError returns nil if the server that sent the reply can run
// transactions, which takes the primary of a replica set or a mongos, and
// otherwise an error that says what to do.
func (h helloReply) transactionsError() error {
	switch {
	case h.Msg == "isdbgrid":
		return nil
	case h.SetName != "" && h.IsWritablePrimary:
		return nil
	case h.SetName != "":
		return fmt.Errorf("mongostore: connected to a member of replica set %q that is not the primary; "+
			"connect to the primary, or leave directConnection out of the URI", h.SetName)
	case h.IsReplicaSet:
		return errors.New("mongostore: the MongoDB replica set is not initiated yet: run rs.initiate() once " +
			"(the docker compose mongo healthcheck does it)")
	default:
		return errors.New("mongostore: MongoDB runs as a standalone server, but L2Go stores a character and its " +
			"items in one transaction, which needs a replica set: start mongod with --replSet rs0 and run " +
			"rs.initiate() once (docker compose does both); a standalone's data stays usable")
	}
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
	_, err = s.db.Collection(charactersCollection).Indexes().CreateMany(ctx, characterIndexes())
	if err != nil {
		return fmt.Errorf("mongostore: create characters indexes: %w", err)
	}
	_, err = s.db.Collection(itemsCollection).Indexes().CreateMany(ctx, itemIndexes())
	if err != nil {
		return fmt.Errorf("mongostore: create items indexes: %w", err)
	}
	return nil
}

// write runs fn in a transaction that commits only if fn returns nil.
//
// The driver runs fn again after a transient error, such as a write
// conflict with another transaction, so fn must start over on every call and
// return the driver's errors unchanged: the driver reads their labels to
// decide about retrying. Map errors after write returns.
func (s *Store) write(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.transact(ctx, func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}
		return s.fault.take()
	})
}

// read runs fn in a transaction, so all its reads see the same snapshot: a
// character and its items as one save left them. The same rules as for
// write apply to fn.
func (s *Store) read(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.transact(ctx, fn)
}

func (s *Store) transact(ctx context.Context, fn func(ctx context.Context) error) error {
	sess, err := s.client.StartSession()
	if err != nil {
		return fmt.Errorf("mongostore: start a session: %w", err)
	}
	defer sess.EndSession(ctx)
	_, err = sess.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	})
	return err
}

// faultHook makes the next write transaction fail before it commits.
type faultHook struct {
	mu  sync.Mutex
	err error
}

// arm makes the next write transaction fail with err.
func (h *faultHook) arm(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.err = err
}

// take returns the armed error, if any, and disarms the hook.
func (h *faultHook) take() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	err := h.err
	h.err = nil
	return err
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
