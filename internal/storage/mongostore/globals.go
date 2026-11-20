package mongostore

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/IPeter/l2go/internal/storage"
)

const globalVariablesCollection = "global_variables"

// GlobalVariables returns the global variable store backed by this database.
func (s *Store) GlobalVariables() *GlobalVariables {
	return &GlobalVariables{store: s, coll: s.db.Collection(globalVariablesCollection)}
}

// GlobalVariables is a storage.GlobalVariableStore on the "global_variables"
// collection, one document per variable (decision Q15), the Mobius
// global_variables table:
//
//	{_id: <name>, value: <string>}
type GlobalVariables struct {
	store *Store
	coll  *mongo.Collection
}

var _ storage.GlobalVariableStore = (*GlobalVariables)(nil)

// globalVariableDoc is the stored form of one global variable.
type globalVariableDoc struct {
	Name  string `bson:"_id"`
	Value string `bson:"value"`
}

// LoadGlobalVariables implements storage.GlobalVariableStore.
func (g *GlobalVariables) LoadGlobalVariables(ctx context.Context) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, g.store.timeout)
	defer cancel()
	cur, err := g.coll.Find(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("mongostore: load global variables: %w", err)
	}
	var docs []globalVariableDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongostore: load global variables: %w", err)
	}
	if len(docs) == 0 {
		return nil, nil
	}
	vars := make(map[string]string, len(docs))
	for _, d := range docs {
		vars[d.Name] = d.Value
	}
	return vars, nil
}

// SaveGlobalVariables implements storage.GlobalVariableStore. The deletes
// and the upserts run in one transaction.
func (g *GlobalVariables) SaveGlobalVariables(ctx context.Context, changes storage.VarChanges) error {
	var deleted []string
	for _, name := range changes.Deleted {
		if _, set := changes.Set[name]; !set {
			deleted = append(deleted, name)
		}
	}
	var models []mongo.WriteModel
	for _, name := range slices.Sorted(maps.Keys(changes.Set)) {
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.D{{Key: "_id", Value: name}}).
			SetReplacement(globalVariableDoc{Name: name, Value: changes.Set[name]}).
			SetUpsert(true))
	}
	ctx, cancel := context.WithTimeout(ctx, g.store.timeout)
	defer cancel()
	err := g.store.write(ctx, func(ctx context.Context) error {
		if len(deleted) > 0 {
			if _, err := g.coll.DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: deleted}}}}); err != nil {
				return err
			}
		}
		if len(models) == 0 {
			return nil
		}
		_, err := g.coll.BulkWrite(ctx, models)
		return err
	})
	if err != nil {
		return fmt.Errorf("mongostore: save global variables: %w", err)
	}
	return nil
}
