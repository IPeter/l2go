//go:build integration

package mongostore

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/storagetest"
)

// These tests need the package's internals: the fault hook, the migration
// and the raw collections. They run against the same deployment as the
// tests in mongostore_integration_test.go (make integration).

// testURI returns the test deployment, by default the docker compose one,
// reached from the host with a direct connection.
func testURI() string {
	if uri := os.Getenv("L2GO_MONGO_URI"); uri != "" {
		return uri
	}
	return "mongodb://127.0.0.1:27017/?directConnection=true"
}

// openNamed opens database name, quietly unless opts set a logger, and
// closes it when the test ends.
func openNamed(t *testing.T, name string, opts ...Option) *Store {
	t.Helper()
	opts = append([]Option{WithLogger(slog.New(slog.DiscardHandler))}, opts...)
	s, err := Open(t.Context(), testURI(), name, opts...)
	if err != nil {
		t.Fatalf("Open: %v (is MongoDB running? try: docker compose up -d --wait mongo)", err)
	}
	t.Cleanup(func() {
		if err := s.Close(context.Background()); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return s
}

// openTestStore opens a new, uniquely named database that is dropped when
// the test ends.
func openTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	name := "l2go_test_" + rand.Text()[:12]
	s := openNamed(t, name)
	t.Cleanup(func() {
		if err := s.Drop(context.Background()); err != nil {
			t.Errorf("Drop: %v", err)
		}
	})
	return s, name
}

func TestFailedWrites(t *testing.T) {
	storagetest.TestFailedWrites(t, func(t *testing.T) (storagetest.SaverStore, func(error)) {
		s, _ := openTestStore(t)
		return s.Characters(), s.fault.arm
	})
}

func TestGlobalVariables(t *testing.T) {
	storagetest.TestGlobalVariableStore(t, func(t *testing.T) (storage.GlobalVariableStore, func(error)) {
		s, _ := openTestStore(t)
		return s.GlobalVariables(), s.fault.arm
	})
}

// TestQuestDocuments checks the stored layout of decision Q4: one document
// per character and quest with the L2J rows in vars, the unique
// {char, quest} index, and the character variables in the character.
func TestQuestDocuments(t *testing.T) {
	s, _ := openTestStore(t)
	c := storage.Character{ObjectID: 10, Account: "alice", Name: "Layout",
		Vars: map[string]string{"NEWBIE_SHOTS_RECEIVED": "1"},
		Quests: []storage.Quest{
			{Name: "Q00401_PathOfTheWarrior", Vars: map[string]string{"<state>": "Started", "cond": "5"}},
		},
	}
	if err := s.Characters().Create(t.Context(), &c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	var got struct {
		Char    int32             `bson:"char"`
		Quest   string            `bson:"quest"`
		Vars    map[string]string `bson:"vars"`
		Updated time.Time         `bson:"updated"`
	}
	if err := s.db.Collection(questsCollection).FindOne(t.Context(), bson.D{{Key: "char", Value: int32(10)}}).Decode(&got); err != nil {
		t.Fatalf("find quest: %v", err)
	}
	if got.Char != 10 || got.Quest != "Q00401_PathOfTheWarrior" || !maps.Equal(got.Vars, c.Quests[0].Vars) || got.Updated.IsZero() {
		t.Errorf("quest document = %+v", got)
	}
	var raw struct {
		Vars map[string]string `bson:"vars"`
	}
	if err := s.db.Collection(charactersCollection).FindOne(t.Context(), bson.D{{Key: "_id", Value: int32(10)}}).Decode(&raw); err != nil {
		t.Fatalf("find character: %v", err)
	}
	if !maps.Equal(raw.Vars, c.Vars) {
		t.Errorf("character vars = %v, want %v", raw.Vars, c.Vars)
	}
	// The unique index refuses a second document of the same quest.
	_, err := s.db.Collection(questsCollection).InsertOne(t.Context(), bson.D{
		{Key: "char", Value: int32(10)}, {Key: "quest", Value: "Q00401_PathOfTheWarrior"},
	})
	if err == nil {
		t.Error("a second document of the same character and quest was inserted, want a duplicate key error")
	}
}

func TestDeploymentRunsTransactions(t *testing.T) {
	s, _ := openTestStore(t)
	if err := s.checkTransactions(t.Context()); err != nil {
		t.Fatalf("checkTransactions: %v", err)
	}
}

// TestItemDocuments checks the stored layout of decision I1: one document
// per item with the Mobius column names, and the {owner_id, loc} index.
func TestItemDocuments(t *testing.T) {
	s, _ := openTestStore(t)
	c := storage.Character{ObjectID: 10, Account: "alice", Name: "Layout", Items: []storage.Item{
		{ObjectID: 11, ItemID: 2369, Count: 1, Enchant: 2, Slot: storage.PaperdollRHand, Mana: storage.NoMana},
		{ObjectID: 12, ItemID: 1835, Count: 300, Loc: storage.LocWarehouse, Mana: 7, CustomType1: 1, CustomType2: 2},
	}}
	if err := s.Characters().Create(t.Context(), &c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, want := range []bson.M{
		{"_id": int32(11), "owner_id": int32(10), "item_id": int32(2369), "count": int64(1), "enchant_level": int32(2),
			"loc": "PAPERDOLL", "loc_data": int32(7), "custom_type1": int32(0), "custom_type2": int32(0), "mana_left": int32(-1)},
		{"_id": int32(12), "owner_id": int32(10), "item_id": int32(1835), "count": int64(300), "enchant_level": int32(0),
			"loc": "WAREHOUSE", "loc_data": int32(0), "custom_type1": int32(1), "custom_type2": int32(2), "mana_left": int32(7)},
	} {
		var got bson.M
		if err := s.db.Collection(itemsCollection).FindOne(t.Context(), bson.D{{Key: "_id", Value: want["_id"]}}).Decode(&got); err != nil {
			t.Fatalf("find item %v: %v", want["_id"], err)
		}
		if !maps.Equal(got, want) {
			t.Errorf("item document = %v, want %v", got, want)
		}
	}
	var raw bson.M
	if err := s.db.Collection(charactersCollection).FindOne(t.Context(), bson.D{{Key: "_id", Value: int32(10)}}).Decode(&raw); err != nil {
		t.Fatalf("find character: %v", err)
	}
	if _, ok := raw["items"]; ok {
		t.Errorf("character document has an items field: %v", raw)
	}

	specs, err := s.db.Collection(itemsCollection).Indexes().ListSpecifications(t.Context())
	if err != nil {
		t.Fatalf("list item indexes: %v", err)
	}
	var indexes [][]string
	for _, sp := range specs {
		elems, err := sp.KeysDocument.Elements()
		if err != nil {
			t.Fatalf("index %s keys: %v", sp.Name, err)
		}
		var keys []string
		for _, e := range elems {
			keys = append(keys, e.Key())
		}
		indexes = append(indexes, keys)
	}
	if !slices.ContainsFunc(indexes, func(keys []string) bool { return slices.Equal(keys, []string{"owner_id", "loc"}) }) {
		t.Errorf("item indexes on %v, want one on owner_id and loc", indexes)
	}
}

// legacyCharacter returns a character document as L2Go stored it before
// Milestone 5: its items inside, unless items is nil.
func legacyCharacter(id int32, name string, items []legacyItemDoc) bson.D {
	doc := bson.D{
		{Key: "_id", Value: id},
		{Key: "account", Value: "alice"},
		{Key: "name", Value: name},
		{Key: "name_lower", Value: storage.NameKey(name)},
		{Key: "race", Value: int32(1)},
		{Key: "class_id", Value: int32(18)},
		{Key: "level", Value: int32(7)},
		{Key: "x", Value: int32(-71338)},
		{Key: "y", Value: int32(258271)},
		{Key: "z", Value: int32(-3104)},
		{Key: "created_at", Value: time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC)},
	}
	if items != nil {
		doc = append(doc, bson.E{Key: "items", Value: items})
	}
	return doc
}

// Items of the legacy characters.
var (
	legacySword  = legacyItemDoc{ObjectID: 11, ItemID: 2369, Count: 1, Slot: int32(storage.PaperdollRHand)}
	legacyAdena  = legacyItemDoc{ObjectID: 12, ItemID: 57, Count: 1500, Slot: -1}
	legacyShirt  = legacyItemDoc{ObjectID: 13, ItemID: 1146, Count: 1, Enchant: 3, Slot: int32(storage.PaperdollChest)}
	legacyArrows = legacyItemDoc{ObjectID: 21, ItemID: 1835, Count: 7, Slot: -1}
)

// insertLegacy stores the documents as an older L2Go version left them.
func insertLegacy(t *testing.T, s *Store, docs ...bson.D) {
	t.Helper()
	if _, err := s.db.Collection(charactersCollection).InsertMany(t.Context(), docs); err != nil {
		t.Fatalf("insert legacy characters: %v", err)
	}
}

// embeddedCount returns how many characters still keep items inside.
func embeddedCount(t *testing.T, s *Store) int64 {
	t.Helper()
	n, err := s.db.Collection(charactersCollection).CountDocuments(t.Context(), bson.D{embeddedItems()})
	if err != nil {
		t.Fatalf("count characters with items inside: %v", err)
	}
	return n
}

// legacyItemsOf returns the items character id keeps inside.
func legacyItemsOf(t *testing.T, s *Store, id int32) []legacyItemDoc {
	t.Helper()
	var doc struct {
		Items []legacyItemDoc `bson:"items"`
	}
	if err := s.db.Collection(charactersCollection).FindOne(t.Context(), bson.D{{Key: "_id", Value: id}}).Decode(&doc); err != nil {
		t.Fatalf("find character %d: %v", id, err)
	}
	return doc.Items
}

func TestMigrateEmbeddedItems(t *testing.T) {
	s, name := openTestStore(t)
	insertLegacy(t, s,
		legacyCharacter(10, "Old", []legacyItemDoc{legacyAdena, legacySword, legacyShirt}),
		legacyCharacter(20, "Older", []legacyItemDoc{legacyArrows}),
		legacyCharacter(30, "Empty", []legacyItemDoc{}),
		legacyCharacter(40, "Current", nil),
	)
	current := storage.Item{ObjectID: 41, ItemID: 57, Count: 3, Slot: storage.PaperdollNone, Mana: storage.NoMana}
	if _, err := s.db.Collection(itemsCollection).InsertOne(t.Context(), toItemDoc(40, current)); err != nil {
		t.Fatalf("insert an item of a current character: %v", err)
	}

	chars, items, err := s.migrateEmbeddedItems(t.Context())
	if err != nil {
		t.Fatalf("migrateEmbeddedItems: %v", err)
	}
	if chars != 3 || items != 4 {
		t.Errorf("moved %d characters and %d items, want 3 and 4", chars, items)
	}
	if n := embeddedCount(t, s); n != 0 {
		t.Errorf("%d characters keep items inside, want 0", n)
	}

	// Every character loads with the items it had, in object ID order.
	for id, want := range map[int32][]storage.Item{
		10: {
			{ObjectID: 11, ItemID: 2369, Count: 1, Slot: storage.PaperdollRHand, Loc: storage.LocPaperdoll, Mana: storage.NoMana},
			{ObjectID: 12, ItemID: 57, Count: 1500, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: storage.NoMana},
			{ObjectID: 13, ItemID: 1146, Count: 1, Enchant: 3, Slot: storage.PaperdollChest, Loc: storage.LocPaperdoll, Mana: storage.NoMana},
		},
		20: {{ObjectID: 21, ItemID: 1835, Count: 7, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: storage.NoMana}},
		30: nil,
		40: {current.Normalized()},
	} {
		c, err := s.Characters().Get(t.Context(), id)
		if err != nil {
			t.Fatalf("Get(%d): %v", id, err)
		}
		if !slices.Equal(c.Items, want) {
			t.Errorf("character %d items = %+v, want %+v", id, c.Items, want)
		}
		if id == 10 && (c.Name != "Old" || c.Level != 7 || c.ClassID != 18 || c.X != -71338) {
			t.Errorf("character 10 = %+v, want its fields kept", c)
		}
	}

	// Running it again finds nothing, and so does opening the database.
	chars, items, err = s.migrateEmbeddedItems(t.Context())
	if err != nil || chars != 0 || items != 0 {
		t.Errorf("second run moved %d characters and %d items, err %v; want nothing", chars, items, err)
	}
	openNamed(t, name)
}

func TestOpenMigratesAndLogs(t *testing.T) {
	s, name := openTestStore(t)
	insertLegacy(t, s, legacyCharacter(10, "Old", []legacyItemDoc{legacyAdena, legacySword}))

	var logs bytes.Buffer
	reopened := openNamed(t, name, WithLogger(slog.New(slog.NewTextHandler(&logs, nil))))
	if out := logs.String(); !strings.Contains(out, "items collection") || !strings.Contains(out, "characters=1") || !strings.Contains(out, "items=2") {
		t.Errorf("Open logged %q, want the counts of the migration", out)
	}
	c, err := reopened.Characters().Get(t.Context(), 10)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(c.Items) != 2 {
		t.Errorf("items after Open = %+v, want 2", c.Items)
	}

	logs.Reset()
	openNamed(t, name, WithLogger(slog.New(slog.NewTextHandler(&logs, nil))))
	if out := logs.String(); strings.Contains(out, "items collection") {
		t.Errorf("a second Open logged %q, want no migration", out)
	}
}

// TestMigrateConflictOverwritesNothing gives a legacy item an object ID that
// the items collection already has: the migration must stop and change
// nothing.
func TestMigrateConflictOverwritesNothing(t *testing.T) {
	s, _ := openTestStore(t)
	taken := storage.Item{ObjectID: legacySword.ObjectID, ItemID: 9, Count: 1, Slot: storage.PaperdollNone, Mana: storage.NoMana}
	if _, err := s.db.Collection(itemsCollection).InsertOne(t.Context(), toItemDoc(20, taken)); err != nil {
		t.Fatalf("insert item: %v", err)
	}
	insertLegacy(t, s, legacyCharacter(10, "Old", []legacyItemDoc{legacyAdena, legacySword}))

	if _, _, err := s.migrateEmbeddedItems(t.Context()); err == nil {
		t.Fatal("migrateEmbeddedItems succeeded despite the conflict")
	}
	if items := legacyItemsOf(t, s, 10); len(items) != 2 {
		t.Errorf("character 10 keeps %+v inside, want both items", items)
	}
	n, err := s.db.Collection(itemsCollection).CountDocuments(t.Context(), bson.D{{Key: "_id", Value: legacyAdena.ObjectID}})
	if err != nil || n != 0 {
		t.Errorf("the other item was moved (%d, %v), want nothing moved", n, err)
	}
	var owner struct {
		OwnerID int32 `bson:"owner_id"`
	}
	if err := s.db.Collection(itemsCollection).FindOne(t.Context(), bson.D{{Key: "_id", Value: taken.ObjectID}}).Decode(&owner); err != nil || owner.OwnerID != 20 {
		t.Errorf("the existing item's owner = %d, %v, want 20", owner.OwnerID, err)
	}
}

// TestMigrateFailureLeavesCharacterUntouched fails a character's
// transaction just before it commits.
func TestMigrateFailureLeavesCharacterUntouched(t *testing.T) {
	s, _ := openTestStore(t)
	insertLegacy(t, s, legacyCharacter(10, "Old", []legacyItemDoc{legacyAdena, legacySword}))
	injected := errors.New("injected failure")
	s.fault.arm(injected)
	if _, _, err := s.migrateEmbeddedItems(t.Context()); !errors.Is(err, injected) {
		t.Fatalf("migrateEmbeddedItems error = %v, want the injected failure", err)
	}
	if items := legacyItemsOf(t, s, 10); len(items) != 2 {
		t.Errorf("character 10 keeps %+v inside, want both items", items)
	}
	if n, err := s.db.Collection(itemsCollection).CountDocuments(t.Context(), bson.D{}); err != nil || n != 0 {
		t.Errorf("items collection has %d documents (%v), want none", n, err)
	}

	chars, items, err := s.migrateEmbeddedItems(t.Context())
	if err != nil || chars != 1 || items != 2 {
		t.Errorf("retry moved %d characters and %d items, err %v; want 1 and 2", chars, items, err)
	}
}

// TestMigrateConcurrently runs the migration from two connections at once,
// as when two servers start on one database: each character moves exactly
// once.
func TestMigrateConcurrently(t *testing.T) {
	s, name := openTestStore(t)
	const characters, perCharacter = 20, 3
	var docs []bson.D
	for i := range int32(characters) {
		id := 1000 * (i + 1)
		var items []legacyItemDoc
		for j := range int32(perCharacter) {
			items = append(items, legacyItemDoc{ObjectID: id + 1 + j, ItemID: 57, Count: int64(j + 1), Slot: -1})
		}
		docs = append(docs, legacyCharacter(id, "Legacy"+string(rune('A'+i)), items))
	}
	insertLegacy(t, s, docs...)
	other := openNamed(t, name)

	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		chars, items int
	)
	for _, st := range []*Store{s, other} {
		wg.Go(func() {
			c, n, err := st.migrateEmbeddedItems(t.Context())
			if err != nil {
				t.Errorf("migrateEmbeddedItems: %v", err)
			}
			mu.Lock()
			chars, items = chars+c, items+n
			mu.Unlock()
		})
	}
	wg.Wait()
	if chars != characters || items != characters*perCharacter {
		t.Errorf("moved %d characters and %d items in all, want %d and %d", chars, items, characters, characters*perCharacter)
	}
	if n, err := s.db.Collection(itemsCollection).CountDocuments(t.Context(), bson.D{}); err != nil || n != characters*perCharacter {
		t.Errorf("items collection has %d documents (%v), want %d", n, err, characters*perCharacter)
	}
	if n := embeddedCount(t, s); n != 0 {
		t.Errorf("%d characters keep items inside, want 0", n)
	}
}
