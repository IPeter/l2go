package mongostore

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
)

// The recipe books go through BSON as the character document stores them;
// no database is needed.
func TestRecipeBookDocRoundTrip(t *testing.T) {
	in := storage.RecipeBook{
		Common:  []int32{900, 851},
		Dwarven: map[int32][]int32{0: {1, 5, 3}, 2: {77}, 1: {}},
	}
	c := storage.Character{ObjectID: 1, Name: "x", Recipes: in}
	raw, err := bson.Marshal(toCharacterDoc(&c))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back characterDoc
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	want := storage.RecipeBook{Common: []int32{900, 851}, Dwarven: map[int32][]int32{0: {1, 5, 3}, 2: {77}}}
	if got := back.character(nil, nil).Recipes; !reflect.DeepEqual(got, want) {
		t.Errorf("read back %+v, want %+v", got, want)
	}
	var m bson.M
	if err := bson.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	books, ok := m["recipes"].(bson.D)
	if !ok {
		t.Fatalf("recipes = %#v", m["recipes"])
	}
	var dwarven any
	for _, e := range books {
		if e.Key == "dwarven" {
			dwarven = e.Value
		}
	}
	if dw, ok := dwarven.(bson.A); !ok || len(dw) != 2 {
		t.Errorf("dwarven = %#v, want the books of class indexes 0 and 2", dwarven)
	}

	// Empty books leave the field out, and read back as the zero value.
	c.Recipes = storage.RecipeBook{Dwarven: map[int32][]int32{0: nil}}
	raw, err = bson.Marshal(toCharacterDoc(&c))
	if err != nil {
		t.Fatal(err)
	}
	m = nil
	if err := bson.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["recipes"]; ok {
		t.Errorf("empty books stored: %v", m["recipes"])
	}
	back = characterDoc{}
	if err := bson.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if got := back.character(nil, nil).Recipes; !reflect.DeepEqual(got, storage.RecipeBook{}) {
		t.Errorf("empty books read back as %+v", got)
	}
	if u := characterUpdate(&c); !hasUnset(u, "recipes") {
		t.Errorf("an update of empty books does not unset them: %v", u)
	}
}

// hasUnset reports whether an update document unsets the key.
func hasUnset(update bson.D, key string) bool {
	for _, e := range update {
		if e.Key != "$unset" {
			continue
		}
		if d, ok := e.Value.(bson.D); ok {
			for _, f := range d {
				if f.Key == key {
					return true
				}
			}
		}
	}
	return false
}
