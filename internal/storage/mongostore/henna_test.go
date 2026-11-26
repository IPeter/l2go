package mongostore

import (
	"maps"
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
)

// The dyes are stored as Mobius character_hennas rows, by class index and
// slot, and read back as the map; empty classes and slots are not stored.
func TestHennaDocRoundTrip(t *testing.T) {
	c := storage.Character{
		ObjectID: 7,
		Name:     "Dyed",
		Hennas: map[int32]storage.HennaSlots{
			0: {1, 0, 180},
			2: {0, 5, 0},
			3: {}, // nothing worn: left out
		},
	}
	d := toCharacterDoc(&c)
	want := []hennaDoc{
		{ClassIndex: 0, Slot: 1, SymbolID: 1},
		{ClassIndex: 0, Slot: 3, SymbolID: 180},
		{ClassIndex: 2, Slot: 2, SymbolID: 5},
	}
	if !slices.Equal(d.Hennas, want) {
		t.Errorf("rows = %+v, want %+v", d.Hennas, want)
	}
	raw, err := bson.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var back characterDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	got := back.character(nil, nil).Hennas
	wantMap := map[int32]storage.HennaSlots{0: {1, 0, 180}, 2: {0, 5, 0}}
	if !maps.Equal(got, wantMap) {
		t.Errorf("read back %v, want %v", got, wantMap)
	}

	// None: the field is absent and read back as nil.
	none := toCharacterDoc(&storage.Character{ObjectID: 8, Hennas: map[int32]storage.HennaSlots{0: {}}})
	if none.Hennas != nil {
		t.Errorf("rows of no dyes = %+v, want nil", none.Hennas)
	}
	if got := none.character(nil, nil).Hennas; got != nil {
		t.Errorf("read back %v, want nil", got)
	}

	// Rows Mobius would skip are skipped.
	bad := characterDoc{Hennas: []hennaDoc{{Slot: 0, SymbolID: 3}, {Slot: 4, SymbolID: 3}, {Slot: 2, SymbolID: 0}}}
	if got := bad.character(nil, nil).Hennas; got != nil {
		t.Errorf("bad rows read back as %v, want nil", got)
	}

	// An update writes the rows, or removes the field.
	if !hasKey(characterUpdate(&c), "$set", "hennas") || !hasKey(characterUpdate(&storage.Character{}), "$unset", "hennas") {
		t.Error("characterUpdate does not write the hennas")
	}
}

func hasKey(update bson.D, op, key string) bool {
	for _, e := range update {
		if e.Key != op {
			continue
		}
		fields, _ := e.Value.(bson.D)
		for _, f := range fields {
			if f.Key == key {
				return true
			}
		}
	}
	return false
}
