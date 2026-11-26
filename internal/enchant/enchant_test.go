package enchant

import (
	"math"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// The tests run on the committed data: the enchant file datagen made from
// Mobius and the item templates.
var loadTestData = sync.OnceValues(func() (*data.Data, error) { return data.Load(filepath.Join("..", "..", "data")) })

func testItems(t *testing.T) *data.Data {
	t.Helper()
	d, err := loadTestData()
	if err != nil {
		t.Fatalf("load the item data: %v", err)
	}
	return d
}

func testTable(t *testing.T, blacklist ...int) *Table {
	t.Helper()
	d, err := Load(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	tbl, skipped := NewTable(d, testItems(t), blacklist)
	if len(skipped) != 0 {
		t.Fatalf("NewTable skipped scrolls %v", skipped)
	}
	return tbl
}

func tmpl(t *testing.T, id int) *data.Item {
	t.Helper()
	it, ok := testItems(t).ItemTemplate(id)
	if !ok {
		t.Fatalf("no item %d", id)
	}
	return it
}

// Items of the tests (Mobius item IDs).
const (
	saber           = 123  // D, rhand, fighter, 743 crystals
	twoHandedSword  = 124  // D, lrhand, fighter, 1758 crystals
	boneStaff       = 178  // D, lrhand, magic, 743 crystals
	spiritSword     = 131  // C, rhand, 1148 crystals
	towerShield     = 103  // C, lhand shield, 103 crystals
	necklaceOfSeal  = 119  // C, neck, 226 crystals
	compositeArmor  = 60   // C, onepiece (full armor), 576 crystals
	longSword       = 2    // no grade
	keshanberk      = 142  // B, rhand
	swordOfMiracles = 151  // A, rhand, magic
	yablonskis      = 165  // S, rhand
	weaponD         = 955  // Scroll: Enchant Weapon (Grade D)
	armorD          = 956  // Scroll: Enchant Armor (Grade D)
	weaponC         = 951  // Scroll: Enchant Weapon (Grade C)
	armorC          = 952  // Scroll: Enchant Armor (Grade C)
	blessedWeaponD  = 6575 // Blessed Scroll: Enchant Weapon (Grade D)
	blessedArmorC   = 6574 // Blessed Scroll: Enchant Armor (Grade C)
	crystalWeaponD  = 957  // Crystal Scroll: Enchant Weapon (Grade D)
	crystalArmorC   = 954  // Crystal Scroll: Enchant Armor (Grade C)
)

// TestMobiusData checks the committed data against EnchantItemData.xml and
// EnchantItemGroups.xml: 30 scrolls of 6 kinds and grades D to S, the
// derived maximum +16 for every kind of item.
func TestMobiusData(t *testing.T) {
	tbl := testTable(t)
	ids := tbl.Scrolls()
	if len(ids) != 30 {
		t.Fatalf("%d scrolls, want 30", len(ids))
	}
	type kind struct {
		weapon bool
		kind   Kind
		bonus  float64
		max    int
	}
	count := map[kind]int{}
	for _, id := range ids {
		s := tbl.Scroll(id)
		count[kind{s.Weapon, s.Kind, s.BonusRate, s.MaxEnchant}]++
	}
	// Five grades each of the weapon and armor scrolls, the blessed ones and
	// the crystal ones, which never fail.
	for _, k := range []kind{
		{true, KindNormal, 0, 16},
		{false, KindNormal, 0, 16},
		{true, KindBlessed, 0, 16},
		{false, KindBlessed, 0, 16},
		{true, KindNormal, 100, 65535},
		{false, KindNormal, 100, 65535},
	} {
		if count[k] != 5 {
			t.Errorf("%+v: %d scrolls, want 5 (all: %v)", k, count[k], count)
		}
	}
	if s := tbl.Scroll(weaponD); s.Grade != data.CrystalD || !s.Weapon || s.Kind != KindNormal {
		t.Errorf("Scroll(955) = %+v", s)
	}
	if tbl.Scroll(saber) != nil {
		t.Error("a sword is a scroll")
	}
	if tbl.MaxWeapon() != 16 || tbl.MaxArmor() != 16 || tbl.MaxAccessory() != 16 {
		t.Errorf("max weapon %d, armor %d, accessory %d; want 16 each", tbl.MaxWeapon(), tbl.MaxArmor(), tbl.MaxAccessory())
	}
}

// TestValid checks EnchantScroll.isValid for every scroll kind and item
// kind: the scroll's type and grade, the location, the blacklist and the
// scroll's maximum level.
func TestValid(t *testing.T) {
	tbl := testTable(t, saber)
	tests := []struct {
		name    string
		scroll  int
		item    int
		enchant int
		carried bool
		want    bool
	}{
		{"weapon scroll, D weapon", weaponD, twoHandedSword, 0, true, true},
		{"weapon scroll, magic weapon", weaponD, boneStaff, 5, true, true},
		{"weapon scroll, C weapon", weaponD, spiritSword, 0, true, false},
		{"weapon scroll, C armor", weaponC, compositeArmor, 0, true, false},
		{"armor scroll, weapon", armorC, spiritSword, 0, true, false},
		{"armor scroll, full armor", armorC, compositeArmor, 0, true, true},
		{"armor scroll, shield", armorC, towerShield, 0, true, true},
		{"armor scroll, accessory", armorC, necklaceOfSeal, 0, true, true},
		{"blessed armor scroll, accessory", blessedArmorC, necklaceOfSeal, 0, true, true},
		{"crystal armor scroll, accessory", crystalArmorC, necklaceOfSeal, 0, true, true},
		{"no-grade item", weaponD, longSword, 0, true, false},
		{"blacklisted", weaponD, saber, 0, true, false},
		{"in the warehouse", weaponD, twoHandedSword, 0, false, false},
		{"+15 below the maximum", weaponD, twoHandedSword, 15, true, true},
		{"+16 is the scroll's maximum", weaponD, twoHandedSword, 16, true, false},
		{"blessed at +16", blessedWeaponD, twoHandedSword, 16, true, false},
		{"crystal scroll at +16 (no maximum)", crystalWeaponD, twoHandedSword, 16, true, true},
		{"crystal scroll at +200", crystalWeaponD, twoHandedSword, 200, true, true},
	}
	for _, tt := range tests {
		got := tbl.Valid(tbl.Scroll(tt.scroll), Target{Template: tmpl(t, tt.item), Enchant: tt.enchant, Carried: tt.carried})
		if got != tt.want {
			t.Errorf("%s: Valid = %v, want %v", tt.name, got, tt.want)
		}
	}
	if tbl.Valid(nil, Target{Template: tmpl(t, saber), Carried: true}) {
		t.Error("Valid without a scroll")
	}
}

// TestChance checks the rate groups: weapons and armor 100 % to +2, full
// armor to +3, then 66 % to +15 and 0 from +16; a level above every range
// takes the last range's chance.
func TestChance(t *testing.T) {
	tbl := testTable(t)
	type row struct {
		scroll, item int
		chances      [18]float64 // +0 to +17
	}
	normal := [18]float64{100, 100, 100, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 0, 0}
	full := [18]float64{100, 100, 100, 100, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 66, 0, 0}
	for _, r := range []row{
		{weaponD, twoHandedSword, normal},
		{weaponD, boneStaff, normal}, // mage weapon group
		{weaponC, spiritSword, normal},
		{armorC, towerShield, normal},
		{armorC, necklaceOfSeal, normal},
		{armorC, compositeArmor, full},
		{crystalArmorC, compositeArmor, full}, // the bonus is not part of the chance
	} {
		for lvl, want := range r.chances {
			got := tbl.Chance(tbl.Scroll(r.scroll), Target{Template: tmpl(t, r.item), Enchant: lvl, Carried: true})
			if got != want {
				t.Errorf("scroll %d, item %d at +%d: chance %v, want %v", r.scroll, r.item, lvl, got, want)
			}
		}
	}
}

// TestChanceGroupsAndBinding checks the parts of the rate lookup the
// Interlude data does not use: bound scrolls, alternative scroll groups,
// item ID filters, a missing rate and a level above every range.
func TestChanceGroupsAndBinding(t *testing.T) {
	items := testItems(t)
	magic := true
	d := Data{
		Scrolls: []Scroll{
			{ID: weaponC, Grade: data.CrystalC, MaxEnchant: 16, ScrollGroup: 1, Items: []ScrollItem{{ID: spiritSword, ScrollGroup: 2}}},
			{ID: weaponD, Grade: data.CrystalD, MaxEnchant: 16},
		},
		RateGroups: []RateGroup{
			{Name: "LOW", Chances: []RangeChance{{0, 3, 50}, {4, 5, 10}}},
			{Name: "HIGH", Chances: []RangeChance{{0, 65535, 90}}},
			{Name: "EMPTY", Chances: []RangeChance{}},
		},
		ScrollGroups: []ScrollGroup{
			{ID: 0, Rates: []RateItem{{Group: "LOW", ItemIDs: []int{twoHandedSword}}, {Group: "HIGH", MagicWeapon: &magic}}},
			{ID: 1, Rates: []RateItem{{Group: "LOW"}}},
			{ID: 2, Rates: []RateItem{{Group: "EMPTY"}}},
		},
	}
	if err := d.Check(); err != nil {
		t.Fatalf("Check: %v", err)
	}
	tbl, skipped := NewTable(d, items, nil)
	if len(skipped) != 0 {
		t.Fatalf("skipped %v", skipped)
	}
	wd, wc := tbl.Scroll(weaponD), tbl.Scroll(weaponC)
	at := func(id, lvl int) Target { return Target{Template: tmpl(t, id), Enchant: lvl, Carried: true} }
	tests := []struct {
		name string
		s    *ScrollInfo
		it   Target
		want float64
	}{
		{"item ID filter", wd, at(twoHandedSword, 2), 50},
		{"above every range: the last one", wd, at(twoHandedSword, 9), 10},
		{"magic filter", wd, at(boneStaff, 9), 90},
		{"no rate takes the item", wd, at(saber, 0), -1},
		{"bound item with its own group, no chances", wc, at(spiritSword, 0), -1},
	}
	for _, tt := range tests {
		if got := tbl.Chance(tt.s, tt.it); got != tt.want {
			t.Errorf("%s: chance %v, want %v", tt.name, got, tt.want)
		}
	}
	// A bound scroll takes only its items.
	if tbl.Valid(wc, at(131+1, 0)) {
		t.Error("the bound scroll took another C weapon")
	}
	if !tbl.Valid(wc, at(spiritSword, 0)) {
		t.Error("the bound scroll refused its item")
	}
	// The roll of a missing chance is an error without a draw.
	if r := tbl.Roll(wd, at(saber, 0), rnd.NewScripted(t)); r != ResultError {
		t.Errorf("Roll without a rate = %v, want error", r)
	}
}

// TestUnboundScrollRefusesBoundItems: an unbound scroll does not take an
// item another scroll is bound to (EnchantScroll.isValid).
func TestUnboundScrollRefusesBoundItems(t *testing.T) {
	d := Data{
		Scrolls: []Scroll{
			{ID: weaponD, Grade: data.CrystalD, MaxEnchant: 16},
			{ID: blessedWeaponD, Grade: data.CrystalD, MaxEnchant: 16, Items: []ScrollItem{{ID: saber}}},
		},
		RateGroups:   []RateGroup{{Name: "ANY", Chances: []RangeChance{{0, 65535, 100}}}},
		ScrollGroups: []ScrollGroup{{ID: 0, Rates: []RateItem{{Group: "ANY"}}}},
	}
	tbl, _ := NewTable(d, testItems(t), nil)
	if tbl.Valid(tbl.Scroll(weaponD), Target{Template: tmpl(t, saber), Carried: true}) {
		t.Error("the unbound scroll took the bound item")
	}
	if !tbl.Valid(tbl.Scroll(weaponD), Target{Template: tmpl(t, twoHandedSword), Carried: true}) {
		t.Error("the unbound scroll refused a free item")
	}
	if !tbl.Valid(tbl.Scroll(blessedWeaponD), Target{Template: tmpl(t, saber), Carried: true}) {
		t.Error("the bound scroll refused its item")
	}
}

// TestNewTableSkipsBadScrolls: a scroll whose item is unknown or not an
// enchant scroll is left out.
func TestNewTableSkipsBadScrolls(t *testing.T) {
	d := Data{
		Scrolls:      []Scroll{{ID: saber}, {ID: weaponD, Grade: data.CrystalD}, {ID: 999999}},
		RateGroups:   []RateGroup{},
		ScrollGroups: []ScrollGroup{{ID: 0}},
	}
	tbl, skipped := NewTable(d, testItems(t), nil)
	if !slices.Equal(skipped, []int{saber, 999999}) {
		t.Errorf("skipped %v, want [%d 999999]", skipped, saber)
	}
	if tbl.Scroll(weaponD) == nil || tbl.Scroll(saber) != nil {
		t.Error("the wrong scrolls were kept")
	}
	// Without rate groups the maximum levels are +1 (Mobius adds one to 0).
	if tbl.MaxWeapon() != 1 || tbl.MaxArmor() != 1 || tbl.MaxAccessory() != 1 {
		t.Errorf("max %d %d %d, want 1", tbl.MaxWeapon(), tbl.MaxArmor(), tbl.MaxAccessory())
	}
}

// TestMaxLevels checks the names EnchantItemGroupsData sorts its groups by.
func TestMaxLevels(t *testing.T) {
	d := Data{RateGroups: []RateGroup{
		{Name: "MY_WEAPON", Chances: []RangeChance{{0, 20, 50}, {21, 30, 0}}},
		{Name: "RING_GROUP", Chances: []RangeChance{{0, 7, 50}}},
		{Name: "BODY", Chances: []RangeChance{{0, 9, 50}}},
	}}
	if w, a, acc := d.maxLevels(); w != 21 || a != 10 || acc != 8 {
		t.Errorf("maxLevels = %d, %d, %d; want 21, 10, 8", w, a, acc)
	}
}

// TestRollDraws pins the draw: one Float64, success when 100 × it is below
// the chance plus the bonus, capped at 100.
func TestRollDraws(t *testing.T) {
	tbl := testTable(t)
	at := func(lvl int) Target { return Target{Template: tmpl(t, twoHandedSword), Enchant: lvl, Carried: true} }
	tests := []struct {
		name   string
		scroll int
		lvl    int
		draw   float64
		want   Result
	}{
		{"+0 always", weaponD, 0, 0.999999, ResultSuccess},
		{"+3 just below 66", weaponD, 3, 0.6599, ResultSuccess},
		{"+3 at 66", weaponD, 3, 0.66, ResultFailure},
		{"+15 high draw", weaponD, 15, 0.99, ResultFailure},
		{"blessed +5 fails", blessedWeaponD, 5, 0.7, ResultFailure},
		{"crystal +10 never fails", crystalWeaponD, 10, 0.999999, ResultSuccess},
		{"crystal +16: chance 0 + 100", crystalWeaponD, 16, 0.999999, ResultSuccess},
	}
	for _, tt := range tests {
		src := rnd.NewScripted(t, rnd.Float64(tt.draw))
		if got := tbl.Roll(tbl.Scroll(tt.scroll), at(tt.lvl), src); got != tt.want {
			t.Errorf("%s: Roll = %v, want %v", tt.name, got, tt.want)
		}
	}
	// An invalid scroll is an error without a draw.
	if got := tbl.Roll(tbl.Scroll(weaponD), at(16), rnd.NewScripted(t)); got != ResultError {
		t.Errorf("Roll at the maximum = %v, want error", got)
	}
	// A crystal scroll at +16 succeeds without a gain.
	if tbl.Gains(tbl.Scroll(crystalWeaponD), at(16)) || !tbl.Gains(tbl.Scroll(crystalWeaponD), at(15)) {
		t.Error("Gains: a crystal scroll gains at +16 or not at +15")
	}
}

// TestRollDistribution rolls 100,000 times per rate with a seeded source:
// the success share is the chance within 0.5 %.
func TestRollDistribution(t *testing.T) {
	tbl := testTable(t)
	const tries = 100_000
	tests := []struct {
		name   string
		scroll int
		item   int
		lvl    int
		want   float64
	}{
		{"weapon +0", weaponD, twoHandedSword, 0, 1},
		{"weapon +3", weaponD, twoHandedSword, 3, 0.66},
		{"weapon +15", weaponD, boneStaff, 15, 0.66},
		{"full armor +3", armorC, compositeArmor, 3, 1},
		{"full armor +4", armorC, compositeArmor, 4, 0.66},
		{"accessory +8", blessedArmorC, necklaceOfSeal, 8, 0.66},
		{"crystal +9", crystalArmorC, compositeArmor, 9, 1},
		{"crystal +16", crystalWeaponD, twoHandedSword, 16, 1},
	}
	for i, tt := range tests {
		src := rnd.New(uint64(i) + 11)
		s := tbl.Scroll(tt.scroll)
		it := Target{Template: tmpl(t, tt.item), Enchant: tt.lvl, Carried: true}
		won := 0
		for range tries {
			switch tbl.Roll(s, it, src) {
			case ResultSuccess:
				won++
			case ResultError:
				t.Fatalf("%s: error", tt.name)
			}
		}
		if got := float64(won) / tries; math.Abs(got-tt.want) > 0.005 {
			t.Errorf("%s: success rate %.4f, want %.2f", tt.name, got, tt.want)
		}
	}
}

// TestCrystalCount checks ItemTemplate.getCrystalCount and the crystals of
// a failed enchant, worked out by hand from the Mobius formulas.
func TestCrystalCount(t *testing.T) {
	tests := []struct {
		name          string
		item, lvl     int
		count, failed int
		crystal       int
	}{
		// D weapon: bonus 90 per level, 90 × (2 × level − 3) above +3.
		{"D weapon +0", saber, 0, 743, 371, 1458},
		{"D weapon +3", saber, 3, 743 + 270, 1013 - 372, 1458},
		{"D weapon +4", saber, 4, 743 + 450, 1193 - 372, 1458},
		{"D weapon +10", saber, 10, 743 + 90*17, 743 + 90*17 - 372, 1458},
		// C armor: bonus 6 per level, 6 × (3 × level − 6) above +3.
		{"C full armor +0", compositeArmor, 0, 576, 576 - 288, 1459},
		{"C full armor +2", compositeArmor, 2, 588, 588 - 288, 1459},
		{"C full armor +5", compositeArmor, 5, 576 + 54, 576 + 54 - 288, 1459},
		{"C accessory +4", necklaceOfSeal, 4, 226 + 36, 226 + 36 - 113, 1459},
		{"C shield +1", towerShield, 1, 109, 109 - 52, 1459},
		// S weapon: bonus 250.
		{"S weapon +6", yablonskis, 6, 1765 + 250*9, 1765 + 250*9 - 883, 1462},
		// A weapon: bonus 145; B weapon: bonus 67.
		{"A weapon +1", swordOfMiracles, 1, 2160 + 145, 2160 + 145 - 1080, 1461},
		{"B weapon +4", keshanberk, 4, 1157 + 67*5, 1157 + 67*5 - 579, 1460},
		// No grade: no crystals.
		{"no grade", longSword, 5, 0, 0, 0},
	}
	for _, tt := range tests {
		tp := tmpl(t, tt.item)
		if got := CrystalCount(tp, tt.lvl); got != tt.count {
			t.Errorf("%s: CrystalCount = %d, want %d", tt.name, got, tt.count)
		}
		id, n := FailureCrystals(tp, tt.lvl)
		if id != tt.crystal || n != tt.failed {
			t.Errorf("%s: FailureCrystals = %d × %d, want %d × %d", tt.name, n, id, tt.failed, tt.crystal)
		}
	}
	// At least one crystal.
	one := &data.Item{Type2: Type2Weapon, Grade: data.CrystalD, CrystalCount: 1}
	if id, n := FailureCrystals(one, 0); id != 1458 || n != 1 {
		t.Errorf("FailureCrystals of a 1-crystal item = %d × %d, want 1 × 1458", n, id)
	}
	// Etc items keep their count.
	etc := &data.Item{Type2: 5, Grade: data.CrystalD, CrystalCount: 7}
	if got := CrystalCount(etc, 9); got != 7 {
		t.Errorf("CrystalCount of an etc item = %d, want 7", got)
	}
}

// TestOverEnchant checks the derived maximum and the over-enchant test.
func TestOverEnchant(t *testing.T) {
	tbl := testTable(t)
	tests := []struct {
		item int
		lvl  int
		over bool
	}{
		{saber, 16, false},
		{saber, 17, true},
		{compositeArmor, 17, true},
		{necklaceOfSeal, 16, false},
		{necklaceOfSeal, 17, true},
		{towerShield, 30, true},
		{weaponD, 300, false}, // a scroll is no equipment
	}
	for _, tt := range tests {
		tp := tmpl(t, tt.item)
		if got := tbl.OverEnchanted(tp, tt.lvl); got != tt.over {
			t.Errorf("item %d +%d: OverEnchanted = %v, want %v", tt.item, tt.lvl, got, tt.over)
		}
		if tbl.MaxEnchant(tp) != 16 {
			t.Errorf("item %d: MaxEnchant %d, want 16", tt.item, tbl.MaxEnchant(tp))
		}
	}
}

// TestCrystallizeLevel: Crystallize levels 2, 3, 4, 5 unlock C, B, A, S.
func TestCrystallizeLevel(t *testing.T) {
	grades := []data.CrystalType{data.CrystalNone, data.CrystalD, data.CrystalC, data.CrystalB, data.CrystalA, data.CrystalS}
	for lvl := 0; lvl <= 5; lvl++ {
		var got []bool
		for _, g := range grades {
			got = append(got, CanCrystallize(g, lvl))
		}
		want := make([]bool, len(grades))
		for i := range want {
			want[i] = lvl > 0 && lvl >= max(i, 1)
		}
		if !slices.Equal(got, want) {
			t.Errorf("level %d: %v, want %v", lvl, got, want)
		}
	}
}

// TestAnnounced: +7 and +15 for weapons, +6 for armor.
func TestAnnounced(t *testing.T) {
	sword, armor := tmpl(t, saber), tmpl(t, compositeArmor)
	jewel := tmpl(t, necklaceOfSeal)
	var got []string
	for lvl := 1; lvl <= 16; lvl++ {
		for name, tp := range map[string]*data.Item{"weapon": sword, "armor": armor, "jewel": jewel} {
			if Announced(tp, lvl) {
				got = append(got, name+"+"+string(rune('0'+lvl/10))+string(rune('0'+lvl%10)))
			}
		}
	}
	slices.Sort(got)
	if want := []string{"armor+06", "jewel+06", "weapon+07", "weapon+15"}; !slices.Equal(got, want) {
		t.Errorf("announced %v, want %v", got, want)
	}
}

// TestCheck reports broken data.
func TestCheck(t *testing.T) {
	d := Data{
		Scrolls: []Scroll{
			{ID: 5, ScrollGroup: 3},
			{ID: 4, MaxEnchant: -1, Items: []ScrollItem{{ID: 1, ScrollGroup: 9}}},
		},
		RateGroups: []RateGroup{
			{Name: "A", Chances: []RangeChance{{5, 1, 10}}},
			{Name: "A"},
			{},
		},
		ScrollGroups: []ScrollGroup{{ID: 2, Rates: []RateItem{{Group: "B"}}}, {ID: 1}},
	}
	err := d.Check()
	if err == nil {
		t.Fatal("Check passed broken data")
	}
	for _, want := range []string{
		"no name", "listed twice", "bad range", "not sorted and unique", "unknown rate group",
		"unknown scroll group 3", "bad grade, maxEnchant", "unknown scroll group 9",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Check error %q does not mention %q", err, want)
		}
	}
}

// TestKindAndResultNames covers the String methods.
func TestKindAndResultNames(t *testing.T) {
	if KindNormal.String() != "normal" || KindBlessed.String() != "blessed" || KindSafe.String() != "safe" {
		t.Error("Kind names")
	}
	if ResultError.String() != "error" || ResultSuccess.String() != "success" || ResultFailure.String() != "failure" {
		t.Error("Result names")
	}
}
