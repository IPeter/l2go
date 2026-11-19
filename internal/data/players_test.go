package data

import (
	"slices"
	"sync"
	"testing"
)

// repository loads the repository's data/ folder once.
var repository = sync.OnceValues(func() (*Data, error) { return Load(repoData) })

func loadRepository(t *testing.T) *Data {
	t.Helper()
	d, err := repository()
	if err != nil {
		t.Fatalf("Load(%s): %v", repoData, err)
	}
	return d
}

func TestRepositoryClasses(t *testing.T) {
	d := loadRepository(t)
	if len(d.Classes) != NumPlayerClasses {
		t.Errorf("%d class templates, want %d", len(d.Classes), NumPlayerClasses)
	}
	for _, c := range d.Classes {
		if c.MaxLevel() != 80 {
			t.Errorf("class %d: %d levels, want 80", c.ClassID, c.MaxLevel())
		}
	}

	var ids []int
	for _, tpl := range d.Templates {
		ids = append(ids, tpl.ClassID)
		if tpl.Race < 0 || tpl.Race > 4 {
			t.Errorf("class %d: race %d, want 0-4", tpl.ClassID, tpl.Race)
		}
		if len(tpl.CreationPoints) == 0 {
			t.Errorf("class %d: no creation points", tpl.ClassID)
		}
		if len(d.InitialItems(tpl.ClassID)) == 0 {
			t.Errorf("class %d: no initial equipment", tpl.ClassID)
		}
	}
	if want := []int{0, 10, 18, 25, 31, 38, 44, 49, 53}; !slices.Equal(ids, want) {
		t.Fatalf("starting classes = %v, want %v", ids, want)
	}

	hf, ok := d.Template(0)
	if !ok {
		t.Fatal("Template(0) not found")
	}
	if hf.ClassName != "Human Fighter" || hf.Race != 0 || hf.RaceName != "Human" {
		t.Errorf("class 0 = %q, race %d %q; want Human Fighter, race 0 Human", hf.ClassName, hf.Race, hf.RaceName)
	}
	if want := (BaseStats{STR: 40, CON: 43, DEX: 30, INT: 21, WIT: 11, MEN: 25}); hf.Stats != want {
		t.Errorf("Human Fighter stats = %+v, want %+v", hf.Stats, want)
	}
	if hf.Level1 != (Vitals{HP: 80, MP: 30, CP: 32}) {
		t.Errorf("Human Fighter level 1 = %+v, want HP 80, MP 30, CP 32", hf.Level1)
	}
	if de, _ := d.Template(31); de.RaceName != "Dark Elf" || de.Race != 2 {
		t.Errorf("class 31: race %d %q, want 2 Dark Elf", de.Race, de.RaceName)
	}
}

// f32 rounds like Mobius, which keeps HP, MP and CP as float.
func f32(v float64) float64 { return float64(float32(v)) }

// Spot checks against dist/game/data/stats/players/templates.
func TestRepositoryClassValues(t *testing.T) {
	d := loadRepository(t)
	tests := []struct {
		class, level int
		hp, mp, cp   float64
	}{
		{0, 1, 80, 30, 32},             // StartingClass/HumanFighter.xml
		{0, 20, 327, 144, 130.8},       //
		{10, 1, 101, 40, 50.5},         // StartingClass/HumanMystic.xml
		{10, 20, 424, 192, 212},        //
		{1, 1, 80, 30, 64},             // 1stClass/Warrior.xml: more CP
		{0, 80, 1415.1, 646.2, 566.04}, // the last level
	}
	for _, tt := range tests {
		c, ok := d.ClassTemplate(tt.class)
		if !ok {
			t.Fatalf("ClassTemplate(%d) not found", tt.class)
		}
		got, ok := c.Level(tt.level)
		if !ok {
			t.Fatalf("class %d: no level %d", tt.class, tt.level)
		}
		if got.HP != f32(tt.hp) || got.MP != f32(tt.mp) || got.CP != f32(tt.cp) {
			t.Errorf("class %d level %d: HP %v MP %v CP %v, want %v %v %v", tt.class, tt.level, got.HP, got.MP, got.CP, tt.hp, tt.mp, tt.cp)
		}
		if raw := c.Levels.CP[tt.level-1]; raw != tt.cp {
			t.Errorf("class %d level %d: raw CP %v, want the XML's %v", tt.class, tt.level, raw, tt.cp)
		}
	}

	hf, _ := d.ClassTemplate(0)
	if l, _ := hf.Level(1); l.HPRegen != 2 || l.MPRegen != 0.9 || l.CPRegen != 2 {
		t.Errorf("Human Fighter level 1 regen = %+v, want 2, 0.9, 2", l)
	}
	if hf.PAtk != 4 || hf.MAtk != 6 || hf.PAtkSpd != 300 || hf.MAtkSpd != 333 || hf.CritRate != 4 || hf.MCritRate != 5 ||
		hf.AttackType != WeaponFist || hf.AttackRange != 20 || hf.RandomDamage != 10 || hf.Breath != 100 || hf.SafeFall != 250 {
		t.Errorf("Human Fighter combat values = %+v", hf)
	}
	if hf.MoveSpeed != (MoveSpeeds{Walk: 80, Run: 115, SlowSwim: 50, FastSwim: 50}) {
		t.Errorf("Human Fighter speeds = %+v", hf.MoveSpeed)
	}
	if hf.PDef != (ArmorDefence{Chest: 31, Legs: 18, Head: 12, Feet: 7, Gloves: 8, Underwear: 3, Cloak: 1}) || hf.BasePDef() != 80 {
		t.Errorf("Human Fighter P.Def slots = %+v, total %d; want 80", hf.PDef, hf.BasePDef())
	}
	if hf.MDef != (JewelDefence{REar: 9, LEar: 9, RFinger: 5, LFinger: 5, Neck: 13}) || hf.BaseMDef() != 41 {
		t.Errorf("Human Fighter M.Def slots = %+v, total %d; want 41", hf.MDef, hf.BaseMDef())
	}
	if hf.CollisionOf(false) != (Collision{Radius: 9, Height: 23}) || hf.CollisionOf(true) != (Collision{Radius: 8, Height: 23.5}) {
		t.Errorf("Human Fighter collisions = %+v, %+v", hf.CollisionMale, hf.CollisionFemale)
	}

	// Class identity comes from the Mobius PlayerClass enum.
	for _, tt := range []struct {
		class, parent, level int
		race                 Race
	}{
		{0, -1, 0, RaceHuman},
		{1, 0, 1, RaceHuman},
		{34, 32, 2, RaceDarkElf},  // classList.xml says 33; the enum wins
		{104, 28, 3, RaceElf},     // classList.xml says 26; the enum wins
		{118, 57, 3, RaceDwarf},   // Maestro
		{49, -1, 0, RaceOrc},      // Orc Mystic
		{116, 52, 3, RaceOrc},     // Doomcryer
		{105, 30, 3, RaceElf},     // Eva's Saint
		{112, 43, 3, RaceDarkElf}, // Shillien Saint
	} {
		c, ok := d.ClassTemplate(tt.class)
		if !ok {
			t.Errorf("ClassTemplate(%d) not found", tt.class)
			continue
		}
		if c.ParentID != tt.parent || c.Class().Level() != tt.level || c.Race != tt.race || c.IsStarting() != (tt.parent < 0) {
			t.Errorf("class %d: parent %d, level %d, race %s; want %d, %d, %s", tt.class, c.ParentID, c.Class().Level(), c.Race, tt.parent, tt.level, tt.race)
		}
	}
	if c, _ := d.ClassTemplate(105); c.Name != "Eva's Saint" {
		t.Errorf("class 105 name %q", c.Name)
	}
}

// Spot checks against statBonus.xml, experience.xml, experienceLoss.xml,
// karmaLoss.xml and hitConditionBonus.xml.
func TestRepositoryTables(t *testing.T) {
	d := loadRepository(t)
	for _, tt := range []struct {
		stat  BaseStat
		value int
		want  float64
	}{
		{BaseSTR, 40, 1.2}, {BaseSTR, 0, 0.29}, {BaseSTR, 99, 9.67},
		{BaseINT, 21, 0.81}, {BaseCON, 43, 1.58}, {BaseMEN, 43, 1.53},
		{BaseDEX, 21, 1.01}, {BaseWIT, 43, 3.07},
	} {
		if got, ok := d.StatBonus(tt.stat, tt.value); !ok || got != tt.want {
			t.Errorf("StatBonus(%s, %d) = %v, %v; want %v", tt.stat, tt.value, got, ok, tt.want)
		}
	}
	if _, ok := d.StatBonus(BaseSTR, 100); ok {
		t.Error("StatBonus(STR, 100) found")
	}

	if d.Experience.MaxLevel != 80 || d.Experience.MaxPetLevel != 80 || len(d.Experience.Exp) != 81 {
		t.Errorf("experience: max %d, pet %d, %d levels; want 80, 80, 81", d.Experience.MaxLevel, d.Experience.MaxPetLevel, len(d.Experience.Exp))
	}
	for level, want := range map[int]int64{1: 0, 2: 68, 20: 835854, 80: 4200000000, 81: 6300000000} {
		if got, ok := d.ExpForLevel(level); !ok || got != want {
			t.Errorf("ExpForLevel(%d) = %d, %v; want %d", level, got, ok, want)
		}
	}
	if v, ok := d.ExperienceLoss.At(1); !ok || v != 10 || len(d.ExperienceLoss) != 80 {
		t.Errorf("experience loss: level 1 %v, %d levels; want 10, 80", v, len(d.ExperienceLoss))
	}
	if v, ok := d.KarmaLoss.At(1); !ok || v != 0.772184315 || len(d.KarmaLoss) != 80 {
		t.Errorf("karma loss: level 1 %v, %d levels; want 0.772184315, 80", v, len(d.KarmaLoss))
	}
	want := HitConditionBonus{Front: 0, Side: 5, Back: 10, High: 3, Low: -3, Dark: -10, Rain: -3}
	if d.HitConditionBonus != want {
		t.Errorf("hit condition bonus = %+v, want %+v", d.HitConditionBonus, want)
	}
}

func TestClassTemplateLevel(t *testing.T) {
	c := testClass(0, "Human Fighter", RaceHuman, -1)
	for _, level := range []int{0, 4, -1} {
		if _, ok := c.Level(level); ok {
			t.Errorf("Level(%d) found", level)
		}
	}
	got, ok := c.Level(2)
	if !ok {
		t.Fatal("Level(2) not found")
	}
	// 91.83 has no exact float32 value: Mobius keeps the float32 one.
	if got.HP != float64(float32(91.83)) || got.HP == 91.83 || got.HPRegen != 2.05 {
		t.Errorf("Level(2) = %+v", got)
	}
	short := testClass(0, "Human Fighter", RaceHuman, -1)
	short.Levels.MPRegen = short.Levels.MPRegen[:1]
	if _, ok := short.Level(2); ok {
		t.Error("Level(2) found with a short mpRegen column")
	}
	if c.MaxLevel() != 3 {
		t.Errorf("MaxLevel() = %d, want 3", c.MaxLevel())
	}
}

func TestClassTemplateSlots(t *testing.T) {
	c := testClass(0, "Human Fighter", RaceHuman, -1)
	c.MDef.LFinger = 6 // Mobius reads the right ring for both
	for slot, want := range map[int]int{
		0: 3, 1: 9, 2: 9, 3: 13, 4: 5, 5: 5, 6: 12, 7: 0, 8: 0, 9: 8,
		10: 31, 11: 18, 12: 7, 13: 1, 14: 0, 15: 0, 16: 0, -1: 0,
	} {
		if got := c.SlotDef(slot); got != want {
			t.Errorf("SlotDef(%d) = %d, want %d", slot, got, want)
		}
	}
	if p, ok := c.Parent(); ok {
		t.Errorf("Parent() = %d, true for a starting class", p)
	}
	w := testClass(1, "Warrior", RaceHuman, 0)
	if p, ok := w.Parent(); !ok || p != ClassFighter || w.IsStarting() || w.Class() != ClassWarrior {
		t.Errorf("Warrior: parent %v %v, starting %v", p, ok, w.IsStarting())
	}
}

func TestTables(t *testing.T) {
	tb := validTables()
	if v, ok := tb.bonus.Bonus(BaseSTR, 50); !ok || v != 1.5 {
		t.Errorf("Bonus(STR, 50) = %v, %v; want 1.5", v, ok)
	}
	if v, ok := tb.bonus.Bonus(BaseMEN, 0); !ok || v != 6 {
		t.Errorf("Bonus(MEN, 0) = %v, %v; want 6", v, ok)
	}
	for _, tt := range []struct {
		stat  BaseStat
		value int
	}{{BaseSTR, -1}, {BaseMEN, 100}, {BaseStat(9), 5}} {
		if _, ok := tb.bonus.Bonus(tt.stat, tt.value); ok {
			t.Errorf("Bonus(%v, %d) found", tt.stat, tt.value)
		}
	}
	if BaseWIT.String() != "WIT" || BaseStat(9).String() != "BaseStat(9)" {
		t.Errorf("BaseStat names: %s, %s", BaseWIT, BaseStat(9))
	}
	if v, ok := tb.exp.ForLevel(4); !ok || v != 1168 {
		t.Errorf("ForLevel(4) = %d, %v", v, ok)
	}
	for _, level := range []int{0, 5} {
		if _, ok := tb.exp.ForLevel(level); ok {
			t.Errorf("ForLevel(%d) found", level)
		}
	}
	if v, ok := tb.expLoss.At(3); !ok || v != 9.75 {
		t.Errorf("At(3) = %v, %v", v, ok)
	}
	if _, ok := tb.expLoss.At(4); ok {
		t.Error("At(4) found")
	}
}
