package augment

import (
	"math"
	"path/filepath"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/stats"
)

var loadTables = sync.OnceValues(func() (*Tables, error) {
	d, err := data.LoadAugment(filepath.Join("..", "..", "data"))
	if err != nil {
		return nil, err
	}
	return NewTables(d), nil
})

// testTables returns the tables of the committed data/augment files.
func testTables(t *testing.T) *Tables {
	t.Helper()
	tb, err := loadTables()
	if err != nil {
		t.Fatalf("load the augmentation data: %v", err)
	}
	return tb
}

// Item templates of the tests (made up, with Mobius-like values).
var (
	cSword = &data.Item{ID: 2499, Name: "Elven Long Sword", Type: data.ItemTypeWeapon, BodyPart: "rhand", Grade: data.CrystalC, CrystalCount: 1501,
		Weapon: &data.WeaponInfo{Type: data.WeaponSword}}
	cStaff = &data.Item{ID: 2503, Name: "Yaksa Mace", Type: data.ItemTypeWeapon, BodyPart: "rhand", Grade: data.CrystalC, CrystalCount: 1501,
		Weapon: &data.WeaponInfo{Type: data.WeaponBlunt, Magic: true}}
	dSword = &data.Item{ID: 69, Name: "Bastard Sword", Type: data.ItemTypeWeapon, BodyPart: "rhand", Grade: data.CrystalD,
		Weapon: &data.WeaponInfo{Type: data.WeaponSword}}
	cRod = &data.Item{ID: 6530, Type: data.ItemTypeWeapon, BodyPart: "lrhand", Grade: data.CrystalC,
		Weapon: &data.WeaponInfo{Type: data.WeaponFishingrod}}
	cNone = &data.Item{ID: 9999, Type: data.ItemTypeWeapon, BodyPart: "rhand", Grade: data.CrystalC,
		Weapon: &data.WeaponInfo{Type: data.WeaponNone}}
	bRing     = &data.Item{ID: 868, Type: data.ItemTypeArmor, BodyPart: "rfinger;lfinger", Grade: data.CrystalB, CrystalCount: 100, Armor: &data.ArmorInfo{}}
	aEarring  = &data.Item{ID: 860, Type: data.ItemTypeArmor, BodyPart: "rear;lear", Grade: data.CrystalA, Armor: &data.ArmorInfo{}}
	sNecklace = &data.Item{ID: 6326, Type: data.ItemTypeArmor, BodyPart: "neck", Grade: data.CrystalS, Armor: &data.ArmorInfo{}}
	cChest    = &data.Item{ID: 354, Type: data.ItemTypeArmor, BodyPart: "chest", Grade: data.CrystalC, Armor: &data.ArmorInfo{}}
	cEtc      = &data.Item{ID: 1, Type: data.ItemTypeEtc, Grade: data.CrystalC, Etc: &data.EtcInfo{}}
	lifeStone = &data.Item{ID: 8723, Type: data.ItemTypeEtc, Etc: &data.EtcInfo{}}
	topStone  = &data.Item{ID: 8762, Type: data.ItemTypeEtc, Etc: &data.EtcInfo{}}
	gemD      = &data.Item{ID: 2130, Type: data.ItemTypeEtc, Stackable: true, Etc: &data.EtcInfo{}}
	gemC      = &data.Item{ID: 2131, Type: data.ItemTypeEtc, Stackable: true, Etc: &data.EtcInfo{}}
)

func with(t *data.Item, change func(c *data.Item)) *data.Item {
	c := *t
	change(&c)
	return &c
}

func carried(t *data.Item) Item { return Item{Template: t, Owned: true, Carried: true} }

func TestMake(t *testing.T) {
	a := Make(91, 14578)
	if a != 14578<<16+91 || a.Stat12() != 91 || a.Stat34() != 14578 || a.Options() != [2]int{91, 14578} {
		t.Errorf("Make(91, 14578) = %d (%d, %d)", a, a.Stat12(), a.Stat34())
	}
	if a := Make(5, 0); a.Stat34() != 0 || a.Stat12() != 5 {
		t.Errorf("Make(5, 0) = %d", a)
	}
}

// TestTargetValid is the item part of AbstractRefinePacket.isValid(Player,
// Item).
func TestTargetValid(t *testing.T) {
	cfg := DefaultConfig()
	pvpCfg := cfg
	pvpCfg.AllowPvP = true
	tests := []struct {
		name string
		cfg  Config
		it   Item
		want bool
	}{
		{"C-grade sword", cfg, carried(cSword), true},
		{"C-grade sword worn", cfg, Item{Template: cSword, Owned: true, Worn: true}, true},
		{"magic weapon", cfg, carried(cStaff), true},
		{"B-grade ring", cfg, carried(bRing), true},
		{"A-grade earring", cfg, carried(aEarring), true},
		{"S-grade necklace", cfg, carried(sNecklace), true},
		{"not owned", cfg, Item{Template: cSword, Carried: true}, false},
		{"in the warehouse", cfg, Item{Template: cSword, Owned: true}, false},
		{"augmented", cfg, Item{Template: cSword, Owned: true, Carried: true, Augmentation: Make(1, 2)}, false},
		{"hero", cfg, carried(with(cSword, func(c *data.Item) { c.Hero = true })), false},
		{"shadow", cfg, Item{Template: cSword, Owned: true, Carried: true, Shadow: true}, false},
		{"common", cfg, carried(with(cSword, func(c *data.Item) { c.Common = true })), false},
		{"etc item", cfg, carried(cEtc), false},
		{"time-limited", cfg, carried(with(cSword, func(c *data.Item) { c.Time = 60 })), false},
		{"PvP", cfg, carried(with(cSword, func(c *data.Item) { c.PvP = true })), false},
		{"PvP allowed", pvpCfg, carried(with(cSword, func(c *data.Item) { c.PvP = true })), true},
		{"D grade", cfg, carried(dSword), false},
		{"no grade", cfg, carried(with(cSword, func(c *data.Item) { c.Grade = data.CrystalNone })), false},
		{"fishing rod", cfg, carried(cRod), false},
		{"weapon type none", cfg, carried(cNone), false},
		{"chest armor", cfg, carried(cChest), false},
		{"blacklisted jewel", cfg, carried(with(sNecklace, func(c *data.Item) { c.ID = 6656 })), false},
		{"blacklisted 8191", cfg, carried(with(sNecklace, func(c *data.Item) { c.ID = 8191 })), false},
		{"no template", cfg, Item{Owned: true, Carried: true}, false},
	}
	for _, tt := range tests {
		if got := tt.cfg.TargetValid(tt.it); got != tt.want {
			t.Errorf("%s: TargetValid = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestCheckPlayer is AbstractRefinePacket.isValid(Player), in its order.
func TestCheckPlayer(t *testing.T) {
	tests := []struct {
		name string
		p    Player
		msg  int
		ok   bool
	}{
		{"fine", Player{Level: 80}, 0, true},
		{"store", Player{StoreMode: true, Trading: true, Dead: true}, MsgStore, false},
		{"trade", Player{Trading: true, Dead: true}, MsgTrade, false},
		{"dead", Player{Dead: true, Sitting: true}, MsgDead, false},
		{"paralyzed", Player{Paralyzed: true, Fishing: true}, MsgParalyzed, false},
		{"fishing", Player{Fishing: true, Sitting: true}, MsgFishing, false},
		{"sitting", Player{Sitting: true, Enchanting: true}, MsgSitting, false},
		{"cursed weapon", Player{CursedWeapon: true}, 0, false},
		{"enchanting", Player{Enchanting: true}, 0, false},
		{"processing a transaction", Player{ProcessingTransaction: true}, 0, false},
	}
	for _, tt := range tests {
		msg, ok := CheckPlayer(tt.p)
		if msg != tt.msg || ok != tt.ok {
			t.Errorf("%s: CheckPlayer = %d, %v; want %d, %v", tt.name, msg, ok, tt.msg, tt.ok)
		}
	}
}

func TestTables(t *testing.T) {
	tb := testTables(t)
	d := tb.Data()
	if len(d.Refine.LifeStones) != 40 {
		t.Errorf("%d life stones, want 40", len(d.Refine.LifeStones))
	}
	for _, tt := range []struct {
		id, grade, level, player int
	}{
		{8723, data.LifeStoneGradeNone, 0, 46},
		{8732, data.LifeStoneGradeNone, 9, 76},
		{8733, data.LifeStoneGradeMid, 0, 46},
		{8747, data.LifeStoneGradeHigh, 4, 58},
		{8762, data.LifeStoneGradeTop, 9, 76},
	} {
		ls, ok := tb.LifeStone(tt.id)
		if !ok || ls.Grade != tt.grade || ls.Level != tt.level || ls.PlayerLevel != tt.player {
			t.Errorf("LifeStone(%d) = %+v, %v", tt.id, ls, ok)
		}
	}
	if _, ok := tb.LifeStone(8722); ok {
		t.Error("8722 is a life stone")
	}
	for _, tt := range []struct {
		grade     data.CrystalType
		gem, need int
	}{
		{data.CrystalC, 2130, 20}, {data.CrystalB, 2130, 30}, {data.CrystalA, 2131, 20}, {data.CrystalS, 2131, 25}, {data.CrystalD, 0, 0},
	} {
		for grade := range 4 { // the life stone grade does not count
			if id, n := tb.GemstoneID(tt.grade), tb.GemstoneCount(tt.grade, grade); id != tt.gem || n != tt.need {
				t.Errorf("gemstones of %s with a grade %d stone: %d × %d, want %d × %d", tt.grade, grade, n, id, tt.need, tt.gem)
			}
		}
	}
	for _, id := range []int{30298, 30300, 30458, 30471, 30678, 30688, 30846, 30898, 31271, 31316, 31583, 31960} {
		if !tb.Blacksmith(id) {
			t.Errorf("NPC %d does not offer augmentation", id)
		}
	}
	if tb.Blacksmith(30001) {
		t.Error("the merchant 30001 offers augmentation")
	}
}

// TestCancelFee: 95,000 to 480,000 adena by grade and crystal count
// (RequestRefineCancel).
func TestCancelFee(t *testing.T) {
	tb := testTables(t)
	tests := []struct {
		grade    data.CrystalType
		crystals int
		fee      int64
		ok       bool
	}{
		{data.CrystalC, 0, 95000, true},
		{data.CrystalC, 1719, 95000, true},
		{data.CrystalC, 1720, 150000, true},
		{data.CrystalC, 2451, 150000, true},
		{data.CrystalC, 2452, 210000, true},
		{data.CrystalB, 1745, 240000, true},
		{data.CrystalB, 1746, 270000, true},
		{data.CrystalA, 2159, 330000, true},
		{data.CrystalA, 2160, 390000, true},
		{data.CrystalA, 2823, 390000, true},
		{data.CrystalA, 2824, 420000, true},
		{data.CrystalS, 99999, 480000, true},
		{data.CrystalD, 100, 0, false},
		{data.CrystalNone, 0, 0, false},
	}
	for _, tt := range tests {
		fee, ok := tb.CancelFee(tt.grade, tt.crystals)
		if fee != tt.fee || ok != tt.ok {
			t.Errorf("CancelFee(%s, %d) = %d, %v; want %d, %v", tt.grade, tt.crystals, fee, ok, tt.fee, tt.ok)
		}
	}
}

// TestCheck walks the checks of the confirm and refine packets.
func TestCheck(t *testing.T) {
	tb := testTables(t)
	cfg := DefaultConfig()
	stone, top := carried(lifeStone), carried(topStone)
	gems := carried(gemD)
	worn := Item{Template: lifeStone, Owned: true, Worn: true}
	tests := []struct {
		name  string
		req   Request
		valid bool
		msg   int
	}{
		{"target alone", Request{Player: Player{Level: 46}, Target: carried(cSword)}, true, 0},
		{"sitting", Request{Player: Player{Level: 46, Sitting: true}, Target: carried(cSword)}, false, MsgSitting},
		{"bad target", Request{Player: Player{Level: 46}, Target: carried(dSword), Refiner: &stone}, false, 0},
		{"with a life stone", Request{Player: Player{Level: 46}, Target: carried(cSword), Refiner: &stone}, true, 0},
		{"level too low", Request{Player: Player{Level: 45}, Target: carried(cSword), Refiner: &stone}, false, 0},
		{"level 76 stone at 75", Request{Player: Player{Level: 75}, Target: carried(cSword), Refiner: &top}, false, 0},
		{"level 76 stone at 76", Request{Player: Player{Level: 76}, Target: carried(cSword), Refiner: &top}, true, 0},
		{"refiner not a life stone", Request{Player: Player{Level: 80}, Target: carried(cSword), Refiner: &gems}, false, 0},
		{"refiner worn", Request{Player: Player{Level: 80}, Target: carried(cSword), Refiner: &worn}, false, 0},
		{"gemstones", Request{Player: Player{Level: 46}, Target: carried(cSword), Refiner: &stone, Gemstone: &gems, GemCount: 20}, true, 0},
		{"more gemstones", Request{Player: Player{Level: 46}, Target: carried(cSword), Refiner: &stone, Gemstone: &gems, GemCount: 999}, true, 0},
		{"too few gemstones", Request{Player: Player{Level: 46}, Target: carried(cSword), Refiner: &stone, Gemstone: &gems, GemCount: 19}, false, 0},
		{"wrong gemstone", Request{Player: Player{Level: 46}, Target: carried(cSword), Refiner: &stone, Gemstone: ptr(carried(gemC)), GemCount: 20}, false, 0},
		{"B ring needs 30", Request{Player: Player{Level: 46}, Target: carried(bRing), Refiner: &stone, Gemstone: &gems, GemCount: 29}, false, 0},
		{"A earring takes Gemstone C", Request{Player: Player{Level: 46}, Target: carried(aEarring), Refiner: &stone, Gemstone: ptr(carried(gemC)), GemCount: 20}, true, 0},
	}
	for _, tt := range tests {
		r := tb.Check(cfg, tt.req)
		if r.Valid != tt.valid || r.PlayerMsg != tt.msg {
			t.Errorf("%s: Check = %+v, want valid %v, message %d", tt.name, r, tt.valid, tt.msg)
		}
		if r.Valid && tt.req.Refiner != nil && r.LifeStone.ItemID != tt.req.Refiner.Template.ID {
			t.Errorf("%s: life stone %+v", tt.name, r.LifeStone)
		}
	}
}

func ptr[T any](v T) *T { return &v }

// TestCheckCancel: the checks of RequestConfirmCancelItem and
// RequestRefineCancel and their different orders.
func TestCheckCancel(t *testing.T) {
	tb := testTables(t)
	cfg := DefaultConfig()
	aug := func(tmpl *data.Item) Item {
		return Item{Template: tmpl, Owned: true, Carried: true, Augmentation: Make(1, 2)}
	}
	pvp := with(cSword, func(c *data.Item) { c.PvP = true })
	tests := []struct {
		name          string
		p             Player
		it            Item
		confirm, real CancelCheck
	}{
		{"C sword", Player{}, aug(cSword), CancelCheck{Valid: true, Fee: 95000}, CancelCheck{Valid: true, Fee: 95000}},
		{"S necklace", Player{}, aug(sNecklace), CancelCheck{Valid: true, Fee: 480000}, CancelCheck{Valid: true, Fee: 480000}},
		{"not augmented", Player{}, carried(cSword), CancelCheck{Msg: MsgRemoveNotAugmented}, CancelCheck{Msg: MsgRemoveNotAugmented}},
		{"PvP", Player{}, aug(pvp), CancelCheck{Msg: MsgNotSuitable}, CancelCheck{}},
		{"D grade", Player{}, aug(dSword), CancelCheck{}, CancelCheck{}},
		// The confirmation checks the player first; the removal the item.
		{"dead, not augmented", Player{Dead: true}, carried(cSword), CancelCheck{PlayerMsg: MsgDead}, CancelCheck{Msg: MsgRemoveNotAugmented}},
		{"dead", Player{Dead: true}, aug(cSword), CancelCheck{PlayerMsg: MsgDead}, CancelCheck{PlayerMsg: MsgDead}},
	}
	for _, tt := range tests {
		if got := tb.CheckCancel(cfg, tt.p, tt.it); got != tt.confirm {
			t.Errorf("%s: CheckCancel = %+v, want %+v", tt.name, got, tt.confirm)
		}
		if got := tb.CheckRefineCancel(cfg, tt.p, tt.it); got != tt.real {
			t.Errorf("%s: CheckRefineCancel = %+v, want %+v", tt.name, got, tt.real)
		}
	}
	if !cfg.Destroyable(aug(cSword)) || cfg.Tradable(aug(cSword)) || !cfg.Tradable(carried(cSword)) {
		t.Error("augmented items: want destroyable, not tradable")
	}
}

// TestGenerateScripted pins the draws and the selection loop of
// generateRandomAugmentation on the first life stone of the warrior
// chances: Rnd.get(10000) for the low option, Rnd.get(100) for the grade,
// Rnd.get(10000) for the high option.
func TestGenerateScripted(t *testing.T) {
	tb := testTables(t)
	cfg := DefaultConfig()
	// Options 1-90 have 1.1 each and 91 has 1.0 (stone 8723, variation 1):
	// s goes 10000, 9890, 9780, ...; the entries taken while s > r.
	// r = 9999: only the first (s = 10000 > 9999). r = 0: all, 91 wins.
	// r = 9000: s after k entries is 10000 - 110k, taken while > 9000:
	// k = 0..9 (s = 9010 > 9000), so entry 10 is the last.
	tests := []struct {
		name              string
		r12, grade, r34   int
		stat12, stat34    int
		magic             bool
		stone, stoneGrade int
	}{
		{"first", 9999, 0, 9999, 1, 7281, false, 8723, 0},
		{"last", 0, 0, 0, 91, 7293, false, 8723, 0},
		// The yellow options have 7.69: r = 9000 takes two (10000, 9231).
		{"tenth", 9000, 54, 9000, 10, 7282, false, 8723, 0},
		// Grade roll 55-89: the blue category, 17 options of 5.88 (the last
		// 5.92): r = 9999 takes the first.
		{"blue", 9999, 55, 9999, 1, 8191, false, 8723, 0},
		{"blue last", 9999, 89, 0, 1, 8207, false, 8723, 0},
		// 90-96 purple, 97-99 red.
		{"purple", 9999, 90, 9999, 1, 14578, false, 8723, 0},
		{"red", 9999, 97, 9999, 1, 14685, false, 8723, 0},
		// Magic weapons take the mage chances.
		{"mage", 9999, 0, 9999, 3641, 10921, true, 8723, 0},
	}
	for _, tt := range tests {
		r := rnd.NewScripted(t, rnd.Get(10000, tt.r12), rnd.Get(100, tt.grade), rnd.Get(10000, tt.r34))
		a := tb.Generate(r, cfg, tt.stone, tt.stoneGrade, tt.magic)
		if a.Stat12() != tt.stat12 || a.Stat34() != tt.stat34 {
			t.Errorf("%s: Generate = (%d, %d), want (%d, %d)", tt.name, a.Stat12(), a.Stat34(), tt.stat12, tt.stat34)
		}
		r.Done()
	}
}

// TestGenerateDistribution is a seeded distribution test (decision X7):
// the categories of the high option come out 55/35/7/3, for warriors and
// mages and every life stone grade, and every option generated is one of
// the stone's chances.
func TestGenerateDistribution(t *testing.T) {
	tb := testTables(t)
	cfg := DefaultConfig()
	const n = 200_000
	for _, tt := range []struct {
		stone int
		magic bool
	}{{8723, false}, {8740, true}, {8752, false}, {8762, true}} {
		category := map[int]int{}
		allowed12 := map[int]bool{}
		for _, g := range tb.Data().Chances {
			if g.Stone != tt.stone || (g.Weapon == data.AugmentWeaponMage) != tt.magic {
				continue
			}
			for _, c := range g.Augments {
				if g.Variation == 1 {
					allowed12[c.ID] = true
				} else {
					category[c.ID] = g.Category
				}
			}
		}
		ls, _ := tb.LifeStone(tt.stone)
		src := rnd.New(uint64(tt.stone)) //nolint:gosec // a positive item ID as the seed
		counts := map[int]int{}
		for range n {
			a := tb.Generate(src, cfg, tt.stone, ls.Grade, tt.magic)
			if !allowed12[a.Stat12()] {
				t.Fatalf("stone %d: low option %d is not one of the stone's", tt.stone, a.Stat12())
			}
			c, ok := category[a.Stat34()]
			if !ok {
				t.Fatalf("stone %d: high option %d is not one of the stone's", tt.stone, a.Stat34())
			}
			counts[c]++
		}
		for _, c := range Categories {
			got := float64(counts[c]) / n * 100
			// 4.5 standard deviations of a binomial share.
			tol := 4.5 * math.Sqrt(float64(c)*(100-float64(c))/n)
			if math.Abs(got-float64(c)) > tol {
				t.Errorf("stone %d (magic %v): category %d came %.2f %%, want %d ± %.2f", tt.stone, tt.magic, c, got, c, tol)
			}
		}
	}
}

// TestGenerateGradeChances: the grade roll follows the configured chances
// of the life stone grade.
func TestGenerateGradeChances(t *testing.T) {
	tb := testTables(t)
	cfg := DefaultConfig()
	cfg.GradeChances[data.LifeStoneGradeTop] = [4]int{0, 0, 0, 100}
	r := rnd.NewScripted(t, rnd.Get(10000, 0), rnd.Get(100, 0), rnd.Get(10000, 9999))
	a := tb.Generate(r, cfg, 8762, data.LifeStoneGradeTop, false)
	found := false
	for _, g := range tb.Data().Chances {
		if g.Stone == 8762 && g.Weapon == data.AugmentWeaponWarrior && g.Category == 3 && g.Augments[0].ID == a.Stat34() {
			found = true
		}
	}
	if !found {
		t.Errorf("a top grade roll of 0 with 0/0/0/100 gave option %d, want the first red one", a.Stat34())
	}
}

// TestOptions: every option of the data converts to modifiers that come
// off again (apply/remove symmetry), and its skills and triggers are well
// formed.
func TestOptions(t *testing.T) {
	tb := testTables(t)
	var ms stats.Modifiers
	for i := range tb.Data().Options {
		o := &tb.Data().Options[i]
		mods, err := Modifiers(o)
		if err != nil {
			t.Fatal(err)
		}
		if err := ms.Add(mods...); err != nil {
			t.Fatalf("option %d: %v", o.ID, err)
		}
		if n := ms.RemoveOwner(OptionOwner(o.ID)); n != len(mods) {
			t.Fatalf("option %d: %d modifiers came off, want %d", o.ID, n, len(mods))
		}
		if ms.Len() != 0 {
			t.Fatalf("option %d: %d modifiers left", o.ID, ms.Len())
		}
		for _, tr := range o.Triggers {
			if !FiresOnHit(tr, false) && !FiresOnHit(tr, true) && !FiresOnSkill(tr, true, false) && !FiresOnSkill(tr, false, true) {
				t.Errorf("option %d: trigger %+v never fires", o.ID, tr)
			}
		}
	}
	opts, missing := tb.Options(Make(1, 14578))
	if len(opts) != 2 || opts[0].ID != 1 || opts[1].ID != 14578 || len(missing) != 0 {
		t.Errorf("Options(1, 14578) = %v, missing %v", opts, missing)
	}
	if opts, missing := tb.Options(Make(1, 0)); len(opts) != 1 || len(missing) != 1 || missing[0] != 0 {
		t.Errorf("Options(1, 0) = %v, missing %v", opts, missing)
	}
	o, _ := tb.Data().Option(1)
	if mods, _ := Modifiers(o); len(mods) != 1 || mods[0].Stat != stats.PDef || mods[0].Op != stats.OpAdd || mods[0].Order != stats.OrderAdd || mods[0].Value != 15.45 {
		t.Errorf("option 1: %+v, want P. Def. + 15.45 at order 30", mods)
	}
}

func TestTriggers(t *testing.T) {
	attack := data.AugmentTrigger{Type: data.AugmentTriggerAttack, Chance: 1}
	crit := data.AugmentTrigger{Type: data.AugmentTriggerCritical, Chance: 5}
	magic := data.AugmentTrigger{Type: data.AugmentTriggerMagic, Chance: 2.5}
	for _, tt := range []struct {
		tr                   data.AugmentTrigger
		hit, critHit, mg, ph bool
	}{
		{attack, true, false, false, true},
		{crit, false, true, false, false},
		{magic, false, false, true, false},
	} {
		if FiresOnHit(tt.tr, false) != tt.hit || FiresOnHit(tt.tr, true) != tt.critHit ||
			FiresOnSkill(tt.tr, true, false) != tt.mg || FiresOnSkill(tt.tr, false, true) != tt.ph {
			t.Errorf("%s: wrong firing", tt.tr.Type)
		}
	}
	r := rnd.NewScripted(t, rnd.Get(100, 2), rnd.Get(100, 3))
	if !Rolled(r, magic) || Rolled(r, magic) {
		t.Error("Rnd.get(100) < 2.5: 2 fires, 3 does not")
	}
}
