package mongostore

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/IPeter/l2go/internal/storage"
)

// The community board's collections (Plan 14 M11o, decision X4), with
// Mobius's column names:
//
//	bbs_mail:      {_id: <id>, receiver_id, sender_id, sender_name, location, recipients, subject, message, sent_date, is_unread}
//	bbs_favorites: {_id: <favId>, playerId, favTitle, favBypass, favAddDate}
//
// Dates are Unix milliseconds; sender_name is not in Mobius.
const (
	bbsMailCollection      = "bbs_mail"
	bbsFavoritesCollection = "bbs_favorites"
)

// BBS returns the community board store backed by this database.
func (s *Store) BBS() *BBS {
	return &BBS{store: s, mails: s.db.Collection(bbsMailCollection), favorites: s.db.Collection(bbsFavoritesCollection)}
}

// BBS is a storage.BBSStore on the "bbs_mail" and "bbs_favorites"
// collections.
type BBS struct {
	store     *Store
	mails     *mongo.Collection
	favorites *mongo.Collection
}

var _ storage.BBSStore = (*BBS)(nil)

type bbsMailDoc struct {
	ID         int32  `bson:"_id"`
	ReceiverID int32  `bson:"receiver_id"`
	SenderID   int32  `bson:"sender_id"`
	SenderName string `bson:"sender_name"`
	Location   string `bson:"location"`
	Recipients string `bson:"recipients"`
	Subject    string `bson:"subject"`
	Message    string `bson:"message"`
	SentDate   int64  `bson:"sent_date"`
	Unread     bool   `bson:"is_unread"`
}

type bbsFavoriteDoc struct {
	ID       int32  `bson:"_id"`
	PlayerID int32  `bson:"playerId"`
	Title    string `bson:"favTitle"`
	Bypass   string `bson:"favBypass"`
	AddDate  int64  `bson:"favAddDate"`
}

// LoadBBSMails implements storage.BBSStore.
func (b *BBS) LoadBBSMails(ctx context.Context) ([]storage.BBSMail, error) {
	ctx, cancel := context.WithTimeout(ctx, b.store.timeout)
	defer cancel()
	cur, err := b.mails.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load bbs mails: %w", err)
	}
	var docs []bbsMailDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load bbs mails: %w", err)
	}
	var out []storage.BBSMail
	for _, d := range docs {
		out = append(out, storage.BBSMail(d))
	}
	return out, nil
}

// SaveBBSMails implements storage.BBSStore.
func (b *BBS) SaveBBSMails(ctx context.Context, mails []storage.BBSMail) error {
	if len(mails) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(mails))
	for _, m := range mails {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: m.ID}}).
			SetReplacement(bbsMailDoc(m)).
			SetUpsert(true))
	}
	ctx, cancel := context.WithTimeout(ctx, b.store.timeout)
	defer cancel()
	err := b.store.write(ctx, func(ctx context.Context) error {
		_, err := b.mails.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(true))
		return err
	})
	if err != nil {
		return fmt.Errorf("mongostore: save bbs mails: %w", err)
	}
	return nil
}

// DeleteBBSMail implements storage.BBSStore.
func (b *BBS) DeleteBBSMail(ctx context.Context, id int32) error {
	ctx, cancel := context.WithTimeout(ctx, b.store.timeout)
	defer cancel()
	err := b.store.write(ctx, func(ctx context.Context) error {
		_, err := b.mails.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
		return err
	})
	if err != nil {
		return fmt.Errorf("mongostore: delete bbs mail %d: %w", id, err)
	}
	return nil
}

// LoadBBSFavorites implements storage.BBSStore.
func (b *BBS) LoadBBSFavorites(ctx context.Context) ([]storage.BBSFavorite, error) {
	ctx, cancel := context.WithTimeout(ctx, b.store.timeout)
	defer cancel()
	cur, err := b.favorites.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("mongostore: load bbs favorites: %w", err)
	}
	var docs []bbsFavoriteDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load bbs favorites: %w", err)
	}
	var out []storage.BBSFavorite
	for _, d := range docs {
		out = append(out, storage.BBSFavorite(d))
	}
	return out, nil
}

// SaveBBSFavorite implements storage.BBSStore.
func (b *BBS) SaveBBSFavorite(ctx context.Context, f storage.BBSFavorite) error {
	ctx, cancel := context.WithTimeout(ctx, b.store.timeout)
	defer cancel()
	err := b.store.write(ctx, func(ctx context.Context) error {
		_, err := b.favorites.ReplaceOne(ctx, bson.D{{Key: "_id", Value: f.ID}}, bbsFavoriteDoc(f), options.Replace().SetUpsert(true))
		return err
	})
	if err != nil {
		return fmt.Errorf("mongostore: save bbs favorite %d: %w", f.ID, err)
	}
	return nil
}

// DeleteBBSFavorite implements storage.BBSStore.
func (b *BBS) DeleteBBSFavorite(ctx context.Context, playerID, id int32) error {
	ctx, cancel := context.WithTimeout(ctx, b.store.timeout)
	defer cancel()
	err := b.store.write(ctx, func(ctx context.Context) error {
		_, err := b.favorites.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}, {Key: "playerId", Value: playerID}})
		return err
	})
	if err != nil {
		return fmt.Errorf("mongostore: delete bbs favorite %d: %w", id, err)
	}
	return nil
}
