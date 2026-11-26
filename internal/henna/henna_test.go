package henna

import (
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// Classes of the tests (Mobius PlayerClass).
const (
	classFighter   = 0  // base class, level 0
	classWarrior   = 1  // first class, level 1
	classGladiator = 2  // second class, level 2
	classDuelist   = 88 // third class, level 3
	classMystic    = 10 // base class
	classWizard    = 11 // first class
	classCleric    = 15 // first class
	classBishop    = 16 // second class
)

var loadTable = sync.OnceValues(func() (*data.Hennas, error) {
	return data.LoadHennas(filepath.Join("..", "..", "data"))
})

func table(t *testing.T) *data.Hennas {
	t.Helper()
	h, err := loadTable()
	if err != nil {
		t.Fatalf("load the hennas: %v", err)
	}
	return h
}

func dye(t *testing.T, id int) *data.Henna {
	t.Helper()
	h := table(t).Henna(id)
	if h == nil {
		t.Fatalf("no dye %d", id)
	}
	return h
}

// fake is a dye with only stats, for the cap tests.
func fake(id, str, con, dex, intel, men, wit int) *data.Henna {
	return &data.Henna{DyeID: id, STR: str, CON: con, DEX: dex, INT: intel, MEN: men, WIT: wit}
}

func TestSlots(t *testing.T) {
	tests := []struct {
		name  string
		class int
		want  int
	}{
		{"base fighter", classFighter, 0},
		{"base mystic", classMystic, 0},
		{"first class", classWarrior, 2},
		{"first class mystic", classWizard, 2},
		{"second class", classGladiator, 3},
		{"third class", classDuelist, 3},
	}
	for _, tt := range tests {
		level := data.PlayerClass(tt.class).Level()
		if got := Slots(level); got != tt.want {
			t.Errorf("%s (class %d, level %d): Slots = %d, want %d", tt.name, tt.class, level, got, tt.want)
		}
	}
}

func TestEmptySlots(t *testing.T) {
	one := Worn{dye(t, 1)}
	two := Worn{dye(t, 1), dye(t, 2)}
	gap := Worn{nil, dye(t, 2)}
	full := Worn{dye(t, 1), dye(t, 2), dye(t, 3)}
	tests := []struct {
		name  string
		w     Worn
		level int
		want  int
	}{
		{"base class", Worn{}, 0, 0},
		{"first class, none", Worn{}, 1, 2},
		{"first class, one", one, 1, 1},
		{"first class, two", two, 1, 0},
		{"first class, one in slot 2", gap, 1, 1},
		{"second class, two", two, 2, 1},
		{"third class, three", full, 3, 0},
		// A second class with 3 dyes that became... never below 0.
		{"three on a first class", full, 1, 0},
		{"one on a base class", one, 0, 0},
	}
	for _, tt := range tests {
		if got := tt.w.EmptySlots(tt.level); got != tt.want {
			t.Errorf("%s: EmptySlots = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// Each stat sum is capped at +5 on the positive side only, slot by slot
// (Mobius recalcHennaStats).
func TestBonus(t *testing.T) {
	tests := []struct {
		name string
		w    Worn
		want stats.BaseStats
	}{
		{"none", Worn{}, stats.BaseStats{}},
		{"dye of STR", Worn{fake(1, 1, -3, 0, 0, 0, 0)}, stats.BaseStats{STR: 1, CON: -3}},
		{"sums", Worn{fake(1, 1, -3, 0, 0, 0, 0), fake(2, 2, 0, -1, 0, 0, 0)}, stats.BaseStats{STR: 3, CON: -3, DEX: -1}},
		{"exactly 5", Worn{fake(1, 3, 0, 0, 0, 0, 0), fake(2, 2, 0, 0, 0, 0, 0)}, stats.BaseStats{STR: 5}},
		{"capped at 5", Worn{fake(1, 4, 0, 0, 0, 0, 0), fake(2, 4, 0, 0, 0, 0, 0), fake(3, 4, 0, 0, 0, 0, 0)}, stats.BaseStats{STR: 5}},
		{"minuses not capped", Worn{fake(1, 0, -4, 0, 0, 0, 0), fake(2, 0, -4, 0, 0, 0, 0), fake(3, 0, -4, 0, 0, 0, 0)}, stats.BaseStats{CON: -12}},
		// The cap applies as the slots add up: 4, then +4 is cut to +1, then
		// -3 takes it to 2.
		{"cut, then a minus", Worn{fake(1, 0, 0, 0, 4, 0, 0), fake(2, 0, 0, 0, 4, 0, 0), fake(3, 0, 0, 0, -3, 0, 0)}, stats.BaseStats{INT: 2}},
		// A minus first leaves room: -3, +4 = 1, +4 = 5.
		{"a minus first", Worn{fake(1, 0, 0, 0, 0, -3, 0), fake(2, 0, 0, 0, 0, 4, 0), fake(3, 0, 0, 0, 0, 4, 0)}, stats.BaseStats{MEN: 5}},
		{"every stat", Worn{fake(1, 6, 6, 6, 6, 6, 6)}, stats.BaseStats{STR: 5, CON: 5, DEX: 5, INT: 5, MEN: 5, WIT: 5}},
		{"empty middle slot", Worn{fake(1, 0, 0, 0, 0, 0, 2), nil, fake(3, 0, 0, 0, 0, 0, 2)}, stats.BaseStats{WIT: 4}},
	}
	for _, tt := range tests {
		if got := tt.w.Bonus(); got != tt.want {
			t.Errorf("%s: Bonus = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestAddRemoveFind(t *testing.T) {
	var w Worn
	for i, id := range []int{1, 2, 3} {
		if slot := w.Add(dye(t, id)); slot != i+1 {
			t.Errorf("Add(%d) = slot %d, want %d", id, slot, i+1)
		}
	}
	if slot := w.Add(dye(t, 4)); slot != 0 {
		t.Errorf("Add to a full set = slot %d, want 0", slot)
	}
	if got := w.Find(2); got != 2 {
		t.Errorf("Find(2) = %d, want 2", got)
	}
	if got := w.Find(4); got != 0 {
		t.Errorf("Find(4) = %d, want 0", got)
	}
	if h := w.Remove(2); h == nil || h.DyeID != 2 {
		t.Errorf("Remove(2) = %+v", h)
	}
	if h := w.Remove(2); h != nil {
		t.Errorf("Remove(2) again = %+v", h)
	}
	for _, slot := range []int{0, 4, -1} {
		if h := w.Remove(slot); h != nil {
			t.Errorf("Remove(%d) = %+v", slot, h)
		}
	}
	if got := w.IDs(); got != [MaxSlots]int32{1, 0, 3} {
		t.Errorf("IDs = %v", got)
	}
	// The next dye fills the gap (Mobius addHenna takes the first null).
	if slot := w.Add(dye(t, 5)); slot != 2 {
		t.Errorf("Add into the gap = slot %d, want 2", slot)
	}
	if got := w.List(); len(got) != 3 || got[1].DyeID != 5 {
		t.Errorf("List = %+v", got)
	}
}

func TestRestore(t *testing.T) {
	w := Restore(table(t), [MaxSlots]int32{0, 1, 99999})
	if w[0] != nil || w[1] == nil || w[1].DyeID != 1 || w[2] != nil {
		t.Errorf("Restore = %+v", w)
	}
	if got := w.IDs(); got != [MaxSlots]int32{0, 1, 0} {
		t.Errorf("IDs = %v", got)
	}
}

func TestCheckEquip(t *testing.T) {
	strDye := dye(t, 1) // Str+1 Con-3: 10 dyes, 37,000 adena; fighters and the first mystics
	gladiatorDye := &data.Henna{DyeID: 99, WearCount: 10, WearFee: 37000, Classes: []int{classGladiator}}
	ok := Player{ClassID: classWarrior, ClassLevel: 1, DyeCount: 10, Adena: 37000}
	with := func(f func(p *Player)) Player {
		p := ok
		f(&p)
		return p
	}
	tests := []struct {
		name      string
		w         Worn
		h         *data.Henna
		p         Player
		want      EquipResult
		forbidden bool
	}{
		{"ok", Worn{}, strDye, ok, EquipOK, false},
		{"more than enough", Worn{}, strDye, with(func(p *Player) { p.DyeCount, p.Adena = 50, 1e9 }), EquipOK, false},
		{"base class", Worn{}, strDye, with(func(p *Player) { p.ClassID, p.ClassLevel = classFighter, 0 }), EquipNoSlot, false},
		{"slots full", Worn{dye(t, 2), dye(t, 3)}, strDye, ok, EquipNoSlot, false},
		{"third slot after the second class", Worn{dye(t, 2), dye(t, 3)}, gladiatorDye, with(func(p *Player) { p.ClassID, p.ClassLevel = classGladiator, 2 }), EquipOK, false},
		{"no slot before an unknown dye", Worn{dye(t, 2), dye(t, 3)}, nil, ok, EquipNoSlot, false},
		{"unknown dye", Worn{}, nil, ok, EquipUnknown, false},
		{"too few dyes", Worn{}, strDye, with(func(p *Player) { p.DyeCount = 9 }), EquipRefused, false},
		{"too little adena", Worn{}, strDye, with(func(p *Player) { p.Adena = 36999 }), EquipRefused, false},
		// Dye 1 is not for Bishops.
		{"class not allowed", Worn{}, strDye, with(func(p *Player) { p.ClassID, p.ClassLevel = classBishop, 2 }), EquipRefused, true},
		{"class not allowed, no dyes", Worn{}, strDye, with(func(p *Player) { p.ClassID, p.ClassLevel, p.DyeCount = classBishop, 2, 0 }), EquipRefused, true},
		{"the same dye twice", Worn{strDye}, strDye, ok, EquipOK, false},
	}
	for _, tt := range tests {
		res, forbidden := CheckEquip(tt.w, tt.h, tt.p)
		if res != tt.want || forbidden != tt.forbidden {
			t.Errorf("%s: CheckEquip = %d, %v; want %d, %v", tt.name, res, forbidden, tt.want, tt.forbidden)
		}
	}
}

func TestCheckRemove(t *testing.T) {
	w := Worn{dye(t, 1), nil, dye(t, 2)}
	tests := []struct {
		name  string
		dyeID int
		adena int64
		slot  int
		want  RemoveResult
	}{
		{"ok", 1, 7400, 1, RemoveOK},
		{"slot 3", 2, 10000, 3, RemoveOK},
		{"not worn", 3, 10000, 0, RemoveNotWorn},
		{"unknown", 0, 10000, 0, RemoveNotWorn},
		{"too little adena", 1, 7399, 1, RemoveNoAdena},
	}
	for _, tt := range tests {
		slot, res := CheckRemove(w, tt.dyeID, tt.adena)
		if slot != tt.slot || res != tt.want {
			t.Errorf("%s: CheckRemove = %d, %d; want %d, %d", tt.name, slot, res, tt.slot, tt.want)
		}
	}
}

// Every dye's classes: dye 1 (Str+1 Con-3) is for the first fighter and
// mystic classes, not for their bases; a class change to a class not listed
// takes it off.
func TestAllowedClasses(t *testing.T) {
	strDye := dye(t, 1)
	for _, c := range []int{classWarrior, classWizard, classCleric} {
		if !strDye.AllowedClass(c) {
			t.Errorf("dye 1 refuses class %d", c)
		}
	}
	for _, c := range []int{classFighter, classMystic, classGladiator, classBishop} {
		if strDye.AllowedClass(c) {
			t.Errorf("dye 1 allows class %d", c)
		}
	}
	w := Worn{strDye, nil, dye(t, 2)}
	if got := w.Disallowed(classWarrior); len(got) != 0 {
		t.Errorf("Disallowed(Warrior) = %v", got)
	}
	if got := w.Disallowed(classGladiator); !slices.Equal(got, []int{1, 3}) {
		t.Errorf("Disallowed(Gladiator) = %v, want [1 3]", got)
	}
	// The table lists a class's dyes by dye ID.
	list := table(t).ForClass(classWarrior)
	if len(list) == 0 || list[0].DyeID != 1 || !slices.IsSortedFunc(list, func(a, b *data.Henna) int { return a.DyeID - b.DyeID }) {
		t.Errorf("ForClass(Warrior) starts with %+v", list[:min(len(list), 1)])
	}
	if got := table(t).ForClass(classFighter); len(got) != 0 {
		t.Errorf("ForClass(Fighter) = %d dyes, want none", len(got))
	}
}

func TestSymbolMakers(t *testing.T) {
	if !IsSymbolMaker(31046) || !IsSymbolMaker(31953) || IsSymbolMaker(30001) {
		t.Error("IsSymbolMaker is wrong")
	}
	if len(SymbolMakers) != 11 {
		t.Errorf("%d symbol makers, want 11", len(SymbolMakers))
	}
}
