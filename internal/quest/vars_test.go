package quest

import (
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
)

func TestVarsGetters(t *testing.T) {
	v := NewVars(map[string]string{
		"int": "42", "neg": "-7", "long": "1795000000000", "word": "abc",
		"t": "true", "T": "TRUE", "f": "false", "yes": "yes", "empty": "",
	})
	if s, ok := v.Get("word"); !ok || s != "abc" || !v.Has("word") || v.Has("none") {
		t.Errorf("Get(word) = %q, %v", s, ok)
	}
	if v.String("word", "d") != "abc" || v.String("none", "d") != "d" || v.String("empty", "d") != "" {
		t.Error("String returns the wrong values")
	}
	for _, tt := range []struct {
		name string
		def  int32
		want int32
		err  bool
	}{
		{"int", 0, 42, false}, {"neg", 0, -7, false}, {"none", 5, 5, false},
		{"long", 0, 0, true}, {"word", 3, 3, true}, {"empty", 0, 0, true},
	} {
		got, err := v.Int(tt.name, tt.def)
		var ve *VarError
		if got != tt.want || (err != nil) != tt.err || (err != nil && !errors.As(err, &ve)) {
			t.Errorf("Int(%s, %d) = %d, %v; want %d, error %v", tt.name, tt.def, got, err, tt.want, tt.err)
		}
	}
	if n, err := v.Long("long", 0); n != 1795000000000 || err != nil {
		t.Errorf("Long(long) = %d, %v", n, err)
	}
	if n, err := v.Long("none", -1); n != -1 || err != nil {
		t.Errorf("Long(none) = %d, %v", n, err)
	}
	if _, err := v.Long("word", 0); err == nil {
		t.Error("Long(word) gave no error")
	}
	for name, want := range map[string]bool{"t": true, "T": true, "f": false, "yes": false, "int": false} {
		if got := v.Bool(name, !want); got != want {
			t.Errorf("Bool(%s) = %v, want %v", name, got, want)
		}
	}
	if !v.Bool("none", true) || v.Bool("none", false) {
		t.Error("Bool of a missing variable is not the default")
	}
	if v.Dirty() {
		t.Error("NewVars has changes")
	}
}

func TestVarsChanges(t *testing.T) {
	var v Vars // the zero value works
	v.Set("a", "1")
	v.SetInt("b", -2)
	v.SetLong("c", 1795000000000)
	v.SetBool("d", true)
	v.Remove("none")
	want := map[string]string{"a": "1", "b": "-2", "c": "1795000000000", "d": "true"}
	if got := v.Map(); !maps.Equal(got, want) {
		t.Errorf("Map() = %v, want %v", got, want)
	}
	ch := v.TakeChanges()
	if !maps.Equal(ch.Set, want) || len(ch.Deleted) != 0 || v.Dirty() {
		t.Errorf("TakeChanges = %+v, dirty %v", ch, v.Dirty())
	}

	v.Remove("a")
	v.Set("b", "3")
	v.Set("e", "x")
	v.Remove("e") // never saved: deleted anyway, harmless
	ch = v.TakeChanges()
	if !maps.Equal(ch.Set, map[string]string{"b": "3"}) || !slices.Equal(ch.Deleted, []string{"a", "e"}) {
		t.Errorf("TakeChanges = %+v", ch)
	}

	// A failed save: the next one writes the variables as they are then.
	v.SaveFailed(ch)
	v.Set("a", "again")
	ch = v.TakeChanges()
	if !maps.Equal(ch.Set, map[string]string{"a": "again", "b": "3"}) || !slices.Equal(ch.Deleted, []string{"e"}) {
		t.Errorf("TakeChanges after SaveFailed = %+v", ch)
	}

	// Map is a copy, nil when empty.
	m := v.Map()
	m["a"] = "changed"
	if s, _ := v.Get("a"); s != "again" {
		t.Error("Map shares the variables")
	}
	for _, name := range []string{"a", "b", "c", "d"} {
		v.Remove(name)
	}
	if v.Map() != nil {
		t.Errorf("Map() of no variables = %v, want nil", v.Map())
	}
}

func TestVarsWithStore(t *testing.T) {
	// Global variables through the store and back.
	ctx := t.Context()
	var store storage.GlobalVariableStore = memstore.NewGlobalVariables()
	g := NewVars(nil)
	g.SetLong("DAILY_TASK_RESET", 1795000000000)
	if err := store.SaveGlobalVariables(ctx, g.TakeChanges()); err != nil {
		t.Fatalf("SaveGlobalVariables: %v", err)
	}
	loaded, err := store.LoadGlobalVariables(ctx)
	if err != nil {
		t.Fatalf("LoadGlobalVariables: %v", err)
	}
	if n, err := NewVars(loaded).Long("DAILY_TASK_RESET", 0); n != 1795000000000 || err != nil {
		t.Errorf("after the round trip: %d, %v", n, err)
	}
}
