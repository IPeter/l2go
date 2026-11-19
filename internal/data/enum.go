package data

import "fmt"

// The enums in the *_gen.go files mirror Mobius Java enums (Stat,
// BodyPart, WeaponType, PlayerClass, ...). They are generated from the
// Mobius sources by cmd/datagen -enums and committed:
//
//	go generate ./internal/data
//
// cmd/datagen imports this package, so the generated files must compile
// for it to run; if a broken generator damages them, restore them from git
// first.
//go:generate go run ../../cmd/datagen -enums -mobius ../../L2J_Mobius/L2J_Mobius_CT_0_Interlude -out .

// enumTable holds the names of a generated enum whose values are 0 to
// len(names)-1.
type enumTable struct {
	kind  string // what the values are, for errors: "weapon type"
	names []string
	index map[string]int
}

func newEnumTable(kind string, names ...string) *enumTable {
	t := &enumTable{kind: kind, names: names, index: make(map[string]int, len(names))}
	for i, n := range names {
		t.index[n] = i
	}
	return t
}

func (t *enumTable) valid(v int) bool { return v >= 0 && v < len(t.names) }

func (t *enumTable) name(v int) string {
	if !t.valid(v) {
		return fmt.Sprintf("%s(%d)", t.kind, v)
	}
	return t.names[v]
}

func (t *enumTable) parse(s string) (int, bool) {
	v, ok := t.index[s]
	return v, ok
}

func (t *enumTable) marshal(v int) ([]byte, error) {
	if !t.valid(v) {
		return nil, fmt.Errorf("data: invalid %s %d", t.kind, v)
	}
	return []byte(t.names[v]), nil
}

func (t *enumTable) unmarshal(text []byte) (int, error) {
	v, ok := t.index[string(text)]
	if !ok {
		return 0, fmt.Errorf("data: unknown %s %q", t.kind, text)
	}
	return v, nil
}
