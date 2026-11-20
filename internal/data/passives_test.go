package data

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadPassivesFS(t *testing.T) {
	const js = `[
{"id":4276,"name":"Bow Weapon Weak Point","levels":[{"defenceTraits":[{"trait":"BOW","value":-11}]},{"defenceTraits":[{"trait":"BOW","value":-25}]}]},
{"id":4408,"name":"HP Increase (12x)","levels":[{"funcs":[{"stat":"maxHp","op":"mul","order":20,"value":1.1}]}],"skipped":["Heal"]}
]
`
	p, err := LoadPassivesFS(fstest.MapFS{PassivesFile: {Data: []byte(js)}})
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := p.Skill(4408); !ok || s.Name != "HP Increase (12x)" || !slices.Equal(s.Skipped, []string{"Heal"}) {
		t.Errorf("Skill(4408) = %+v, %v", s, ok)
	}
	want := PassiveFunc{Stat: StatMaxHP, Op: OpMul, Order: 20, Value: 1.1}
	if l, ok := p.Level(4408, 1); !ok || !slices.Equal(l.Funcs, []PassiveFunc{want}) {
		t.Errorf("Level(4408, 1) = %+v, %v", l, ok)
	}
	// Levels as Mobius SkillData.getSkill finds them: above the highest
	// gives the highest.
	for _, tt := range []struct {
		level int
		value float32
		ok    bool
	}{{1, -11, true}, {2, -25, true}, {9, -25, true}, {0, 0, false}} {
		l, ok := p.Level(4276, tt.level)
		if ok != tt.ok || (ok && l.DefenceTraits[0].Value != tt.value) {
			t.Errorf("Level(4276, %d) = %+v, %v; want value %v, %v", tt.level, l, ok, tt.value, tt.ok)
		}
	}
	if _, ok := p.Level(1, 1); ok {
		t.Error("Level of a skill that is not in the data was found")
	}
}

func TestLoadPassivesMissing(t *testing.T) {
	if _, err := LoadPassives(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadPassives of an empty folder: %v", err)
	}
}

func TestNewPassivesErrors(t *testing.T) {
	level := func(l PassiveLevel) []PassiveLevel { return []PassiveLevel{l} }
	ok := PassiveSkill{ID: 1, Levels: level(PassiveLevel{})}
	for _, tt := range []struct {
		name   string
		skills []PassiveSkill
		want   string
	}{
		{"empty", nil, "no skills"},
		{"no levels", []PassiveSkill{{ID: 2}}, "must have levels"},
		{"duplicate", []PassiveSkill{ok, ok}, "duplicate skill ID"},
		{"unsorted", []PassiveSkill{{ID: 3, Levels: ok.Levels}, ok}, "not sorted by ID"},
		{"bad stat", []PassiveSkill{{ID: 1, Levels: level(PassiveLevel{Funcs: []PassiveFunc{{Stat: Stat(250), Op: OpMul}}})}}, "invalid function"},
		{"NaN", []PassiveSkill{{ID: 1, Levels: level(PassiveLevel{Funcs: []PassiveFunc{{Stat: StatMaxHP, Op: OpMul, Value: math.NaN()}}})}}, "invalid function"},
		{"bad trait", []PassiveSkill{{ID: 1, Levels: level(PassiveLevel{AttackTraits: []PassiveTrait{{Trait: TraitType(250)}}})}}, "invalid trait"},
		{"unsorted traits", []PassiveSkill{{ID: 1, Levels: level(PassiveLevel{DefenceTraits: []PassiveTrait{{Trait: TraitBow}, {Trait: TraitSword}}})}}, "traits are not sorted"},
		{"unsorted skipped", []PassiveSkill{{ID: 1, Levels: ok.Levels, Skipped: []string{"b", "a"}}}, "skipped effects are not sorted"},
	} {
		_, err := NewPassives(tt.skills)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error mentioning %q", tt.name, err, tt.want)
		}
	}
	// A trait two effects name is kept twice.
	twice := []PassiveSkill{{ID: 1, Levels: level(PassiveLevel{DefenceTraits: []PassiveTrait{{Trait: TraitBow, Value: 5}, {Trait: TraitBow, Value: 6}}})}}
	if _, err := NewPassives(twice); err != nil {
		t.Errorf("a trait named twice: %v", err)
	}
}

// The trait multiplier is Mobius's (value + 100) / 100 in float arithmetic.
func TestPassiveTraitMultiplier(t *testing.T) {
	for _, tt := range []struct {
		value, mult float32
		invul       bool
	}{
		{-11, 0.89, false},
		{1, 1.01, false}, // the neutral entries of 4416
		{0.85, 1.0085, false},
		{90, 1.9, false},
		{100, 2, true},
	} {
		tr := PassiveTrait{Trait: TraitBow, Value: tt.value}
		if got := tr.Multiplier(); got != tt.mult || tr.Invulnerable() != tt.invul {
			t.Errorf("value %v: multiplier %v, invulnerable %v; want %v, %v", tt.value, got, tr.Invulnerable(), tt.mult, tt.invul)
		}
	}
}

// loadRepoPassives loads the committed passive skill data.
func loadRepoPassives(t *testing.T) *Passives {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repoData, PassivesFile)); err != nil {
		t.Skipf("no passive skill data in %s: %v", repoData, err)
	}
	p, err := LoadPassives(repoData)
	if err != nil {
		t.Fatalf("LoadPassives(%s): %v", repoData, err)
	}
	return p
}

// Spot checks of the committed passives against the Mobius XML
// (stats/skills/04200-04299.xml, 04400-04499.xml and 05000-05099.xml).
func TestRepoPassives(t *testing.T) {
	p := loadRepoPassives(t)
	if len(p.Skills) < 50 {
		t.Errorf("%d passive skills, want at least 50", len(p.Skills))
	}

	// 4408 level 10: #maxhp 2, for maxHp and regHp.
	l, ok := p.Level(4408, 10)
	wantHP := []PassiveFunc{
		{Stat: StatMaxHP, Op: OpMul, Order: 20, Value: 2},
		{Stat: StatRegenerateHPRate, Op: OpMul, Order: 20, Value: 2},
	}
	if !ok || !slices.Equal(l.Funcs, wantHP) {
		t.Errorf("4408 level 10 = %+v, %v; want %v", l, ok, wantHP)
	}

	// 4416 level 4 (Animals): #pDefBows and #pDefAnimals 0.85, the other
	// tables 1, #pDefShocks 0 (left out).
	l, ok = p.Level(4416, 4)
	if !ok || len(l.DefenceTraits) != 11 {
		t.Fatalf("4416 level 4 = %+v, %v; want 11 defence traits", l, ok)
	}
	for _, tr := range l.DefenceTraits {
		want := float32(1)
		if tr.Trait == TraitBow || tr.Trait == TraitAnimalWeakness {
			want = 0.85
		}
		if tr.Trait == TraitShock || tr.Value != want {
			t.Errorf("4416 level 4: %v = %v, want %v", tr.Trait, tr.Value, want)
		}
		if tr.Trait == TraitBow && tr.Multiplier() != 1.0085 {
			t.Errorf("4416 level 4 bow multiplier %v, want 1.0085", tr.Multiplier())
		}
	}

	// 4276 level 1: BOW -11, a multiplier of 0.89.
	if l, ok := p.Level(4276, 1); !ok || !slices.Equal(l.DefenceTraits, []PassiveTrait{{Trait: TraitBow, Value: -11}}) || l.DefenceTraits[0].Multiplier() != 0.89 {
		t.Errorf("4276 level 1 = %+v, %v", l, ok)
	}

	// 5076 level 1.
	mul := func(s Stat, v float64) PassiveFunc { return PassiveFunc{Stat: s, Op: OpMul, Order: 20, Value: v} }
	sub := func(s Stat, v float64) PassiveFunc { return PassiveFunc{Stat: s, Op: OpSub, Order: 30, Value: v} }
	want5076 := []PassiveFunc{
		mul(StatPowerAttack, 0.9), mul(StatPowerAttackSpeed, 0.9), mul(StatMagicAttack, 0.9), mul(StatMagicAttackSpeed, 0.9),
		sub(StatAccuracyCombat, 5), sub(StatEvasionRate, 5),
		mul(StatPowerDefence, 0.91), mul(StatMagicDefence, 0.91), mul(StatMoveSpeed, 0.95),
	}
	if l, ok := p.Level(SkillDeathPenalty, 1); !ok || !slices.Equal(l.Funcs, want5076) {
		t.Errorf("5076 level 1 = %+v, %v; want %v", l, ok, want5076)
	}
	if s, ok := p.Skill(SkillDeathPenalty); !ok || len(s.Levels) != 15 {
		t.Errorf("5076 has %d levels, want 15", len(s.Levels))
	}

	// 4270 (Plan 08 I5): regHp and regMp x0.5, runSpd x1, 0.5, 0.5, 0.
	for level, spd := range []float64{1, 0.5, 0.5, 0} {
		want := []PassiveFunc{mul(StatRegenerateHPRate, 0.5), mul(StatRegenerateMPRate, 0.5), mul(StatMoveSpeed, spd)}
		if l, ok := p.Level(SkillWeightPenalty, level+1); !ok || !slices.Equal(l.Funcs, want) {
			t.Errorf("4270 level %d = %+v, %v; want %v", level+1, l, ok, want)
		}
	}
}
