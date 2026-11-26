package subclass

import (
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/data"
)

var loadCategories = sync.OnceValues(func() (*data.Categories, error) {
	return data.LoadCategories(filepath.Join("..", "..", "data"))
})

func testCategories(t *testing.T) *data.Categories {
	t.Helper()
	c, err := loadCategories()
	if err != nil {
		t.Fatalf("load the categories: %v", err)
	}
	return c
}

func ids(cs []data.PlayerClass) []int {
	out := make([]int, len(cs))
	for i, c := range cs {
		out[i] = int(c)
	}
	return out
}

// The main subclasses are the 31 second classes but the Overlord and the
// Warsmith.
func TestMainSubclasses(t *testing.T) {
	got := ids(mainSubclasses(testCategories(t)))
	want := []int{2, 3, 5, 6, 8, 9, 12, 13, 14, 16, 17, 20, 21, 23, 24, 27, 28, 30, 33, 34, 36, 37, 40, 41, 43, 46, 48, 52, 55}
	if !slices.Equal(got, want) {
		t.Errorf("main subclasses %v, want %v", got, want)
	}
}

func TestIsOfType(t *testing.T) {
	tests := []struct {
		class data.PlayerClass
		typ   TeachType
		want  bool
	}{
		{data.ClassPaladin, TeachFighter, true},
		{data.ClassPaladin, TeachMystic, false},
		{data.ClassBishop, TeachPriest, true},
		{data.ClassBishop, TeachMystic, false},
		{data.ClassBishop, TeachFighter, false},
		{data.ClassSorcerer, TeachMystic, true},
		{data.ClassSorcerer, TeachPriest, false},
		{data.ClassShillienElder, TeachPriest, true},
		{data.ClassEvaSaint, TeachPriest, true},
	}
	for _, tt := range tests {
		if got := IsOfType(tt.class, tt.typ); got != tt.want {
			t.Errorf("IsOfType(%v, %d) = %v, want %v", tt.class, tt.typ, got, tt.want)
		}
	}
	if !EqualsOrChildOf(data.PlayerClass(88), data.ClassGladiator) || !EqualsOrChildOf(data.PlayerClass(88), data.PlayerClass(0)) || EqualsOrChildOf(data.ClassGladiator, data.PlayerClass(88)) {
		t.Error("EqualsOrChildOf: a Duelist comes from the Gladiator and the Human Fighter, not the other way")
	}
}

// Every rule of getSubclasses and getAvailableSubClasses.
func TestAvailable(t *testing.T) {
	cats := testCategories(t)
	cfg := DefaultConfig()
	human := func(base int, subs ...Sub) *Owner {
		return &Owner{Race: data.RaceHuman, BaseClass: base, ActiveClass: base, Level: 76, Subs: subs}
	}
	humanFighter := Master{Race: data.RaceHuman, Teach: TeachFighter}
	humanMystic := Master{Race: data.RaceHuman, Teach: TeachMystic}
	humanPriest := Master{Race: data.RaceHuman, Teach: TeachPriest}
	tests := []struct {
		name   string
		owner  *Owner
		master Master
		cfg    Config
		want   []int
	}{
		{"gladiator at a human fighter master", human(2), humanFighter, cfg, []int{3, 5, 6, 8, 9, 20, 21, 23, 24}},
		{"gladiator at a mystic master", human(2), humanMystic, cfg, []int{12, 13, 14, 27, 28}},
		{"paladin at a priest master", human(5), humanPriest, cfg, []int{16, 17, 30}},
		// The Paladin excludes the other knights.
		{"paladin at a fighter master", human(5), humanFighter, cfg, []int{2, 3, 8, 9, 21, 23, 24}},
		{"gladiator at a dark elf master", human(2), Master{Race: data.RaceDarkElf}, cfg, []int{33, 34, 36, 37, 40, 41, 43}},
		{"gladiator at an orc master", human(2), Master{Race: data.RaceOrc}, cfg, []int{46, 48, 52}},
		{"gladiator at a dwarf master", human(2), Master{Race: data.RaceDwarf}, cfg, []int{55}},
		// An elf gets no dark elf class, even everywhere.
		{"elf everywhere", &Owner{Race: data.RaceElf, BaseClass: 20, ActiveClass: 20, Level: 76}, humanFighter, Config{Everywhere: true, MaxSubclass: 3},
			[]int{2, 3, 8, 9, 12, 13, 14, 16, 17, 21, 23, 24, 27, 28, 30, 46, 48, 52, 55}},
		// A sorcerer subclass goes, and with a subclass the base class too;
		// the active class (the sorcerer) and what it comes from go as well.
		{"with a sorcerer subclass", &Owner{Race: data.RaceHuman, BaseClass: 2, ActiveClass: 12, Level: 75, Subs: []Sub{{1, 12, 75}}}, humanMystic, cfg, []int{13, 14, 27, 28}},
		// A third-class subclass takes its second class out.
		{"with an archmage subclass", human(2, Sub{1, 94, 76}), humanMystic, cfg, []int{13, 14, 27, 28}},
		// A first class has no list.
		{"warrior", human(1), humanFighter, cfg, nil},
		// A third-class base class lists through the third-class category;
		// a Phoenix Knight's (90) Paladin parent is not excluded there.
		{"phoenix knight", human(90), humanFighter, cfg, []int{2, 3, 6, 8, 9, 20, 21, 23, 24}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Available(cats, tt.owner, tt.master, tt.cfg)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("got %v, want none", ids(got))
				}
				return
			}
			if !slices.Equal(ids(got), tt.want) {
				t.Errorf("got %v, want %v", ids(got), tt.want)
			}
		})
	}
}

// isValidNewSubClass: the second class of a third-class base, the subclasses
// already taken, the excluded sets and the elf races.
func TestValidNew(t *testing.T) {
	cats := testCategories(t)
	duelist := &Owner{Race: data.RaceHuman, BaseClass: 88, ActiveClass: 88, Level: 78}
	tests := []struct {
		name  string
		owner *Owner
		class int
		want  bool
	}{
		{"paladin for a duelist", duelist, 5, true},
		{"gladiator for a duelist", duelist, 2, false},
		{"overlord", duelist, 51, false},
		{"warsmith", duelist, 57, false},
		{"a third class", duelist, 90, false},
		{"taken", &Owner{Race: data.RaceHuman, BaseClass: 88, ActiveClass: 88, Subs: []Sub{{1, 12, 76}}}, 12, false},
		{"its third class taken", &Owner{Race: data.RaceHuman, BaseClass: 88, ActiveClass: 88, Subs: []Sub{{1, 94, 76}}}, 12, false},
		{"dark elf for an elf", &Owner{Race: data.RaceElf, BaseClass: 20, ActiveClass: 20}, 33, false},
		{"temple knight excludes the paladin", &Owner{Race: data.RaceElf, BaseClass: 20, ActiveClass: 20}, 5, false},
		{"elf to human", &Owner{Race: data.RaceElf, BaseClass: 20, ActiveClass: 20}, 2, true},
		{"the active subclass", &Owner{Race: data.RaceHuman, BaseClass: 2, ActiveClass: 14, Subs: []Sub{{1, 14, 76}}}, 14, false},
	}
	for _, tt := range tests {
		if got := ValidNew(cats, tt.owner, tt.class); got != tt.want {
			t.Errorf("%s: ValidNew(%d) = %v, want %v", tt.name, tt.class, got, tt.want)
		}
	}
}

func TestCanAddAndQuests(t *testing.T) {
	cfg := DefaultConfig()
	tests := []struct {
		name  string
		owner Owner
		want  bool
	}{
		{"level 74", Owner{Level: 74}, false},
		{"level 75", Owner{Level: 75}, true},
		{"a subclass below 75", Owner{Level: 80, Subs: []Sub{{1, 12, 74}}}, false},
		{"two at 75", Owner{Level: 75, Subs: []Sub{{1, 12, 75}, {2, 13, 79}}}, true},
		{"three", Owner{Level: 80, Subs: []Sub{{1, 12, 80}, {2, 13, 80}, {3, 14, 80}}}, false},
	}
	for _, tt := range tests {
		if got := CanAdd(&tt.owner, cfg); got != tt.want {
			t.Errorf("%s: CanAdd = %v, want %v", tt.name, got, tt.want)
		}
	}
	done := map[string]bool{}
	completed := func(q string) bool { return done[q] }
	if QuestsDone(false, completed) || !QuestsDone(true, completed) {
		t.Error("no quests: only a noble passes")
	}
	done[QuestFatesWhisper] = true
	if QuestsDone(false, completed) {
		t.Error("Fate's Whisper alone passed")
	}
	done[QuestMimirsElixir] = true
	if !QuestsDone(false, completed) {
		t.Error("both quests did not pass")
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want Command
	}{
		{"Subclass 0", Command{Choice: 0}},
		{"Subclass 1", Command{Choice: 1}},
		{"Subclass 4 12", Command{Choice: 4, One: 12}},
		{"Subclass 5 0", Command{Choice: 5}},
		{"Subclass 6 3", Command{Choice: 6, One: 3}},
		{"Subclass 7 1 12", Command{Choice: 7, One: 1, Two: 12}},
		{"Subclass 7 2 100", Command{Choice: 7, One: 2, Two: 100}},
		{"Subclass x", Command{Bad: true}},
		{"Subclass 4 x", Command{Choice: 4, Bad: true}},
		{"Subclass", Command{Bad: true}},
	}
	for _, tt := range tests {
		if got := ParseCommand(tt.cmd); got != tt.want {
			t.Errorf("ParseCommand(%q) = %+v, want %+v", tt.cmd, got, tt.want)
		}
	}
}

func TestLevels(t *testing.T) {
	cfg := DefaultConfig()
	if got := cfg.MaxSubLevel(80); got != 79 {
		t.Errorf("MaxSubLevel(80) = %d, want 79", got)
	}
	if got := cfg.MaxSubLevel(86); got != 80 {
		t.Errorf("MaxSubLevel(86) = %d, want 80", got)
	}
	for _, tt := range []struct{ in, want int }{{30, 40}, {40, 40}, {60, 60}, {85, 79}} {
		if got := cfg.ClampLevel(tt.in, 80); got != tt.want {
			t.Errorf("ClampLevel(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
