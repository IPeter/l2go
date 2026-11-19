package stats

import (
	"errors"
	"slices"
	"testing"
)

func TestItemModifiersStartingItems(t *testing.T) {
	owner := ItemOwner(268435457)
	tests := []struct {
		name string
		item Item
		want []Modifier
	}{
		{
			// DocumentItem: weapon stats are Set at order 0 and replace
			// the base values; randomDamage is an Add. Enchant level 0
			// adds no enchant modifier.
			name: "Squire's Sword",
			item: squiresSword,
			want: []Modifier{
				{Stat: PAtk, Op: OpSet, Value: 6, Owner: owner},
				{Stat: MAtk, Op: OpSet, Value: 5, Owner: owner},
				{Stat: CritRate, Op: OpSet, Value: 8, Owner: owner},
				{Stat: PAtkSpd, Op: OpSet, Value: 379, Owner: owner},
				{Stat: RandomDamage, Op: OpAdd, Value: 10, Owner: owner},
				{Stat: PAtkRange, Op: OpSet, Value: 40, Owner: owner},
			},
		},
		{
			name: "Squire's Shirt",
			item: squiresShirt,
			want: []Modifier{{Stat: PDef, Op: OpAdd, Value: 33, Owner: owner}},
		},
		{
			// accCombat is an Add at order 0, before the accuracy formula.
			name: "Apprentice's Wand",
			item: apprenticesWand,
			want: []Modifier{
				{Stat: PAtk, Op: OpSet, Value: 5, Owner: owner},
				{Stat: MAtk, Op: OpSet, Value: 7, Owner: owner},
				{Stat: CritRate, Op: OpSet, Value: 4, Owner: owner},
				{Stat: AccCombat, Op: OpAdd, Value: 4.75, Owner: owner},
				{Stat: PAtkSpd, Op: OpSet, Value: 379, Owner: owner},
				{Stat: RandomDamage, Op: OpAdd, Value: 20, Owner: owner},
				{Stat: PAtkRange, Op: OpSet, Value: 40, Owner: owner},
			},
		},
		{
			// maxMp is an Add at order 0, so the MEN bonus multiplies it.
			name: "Apprentice's Tunic",
			item: apprenticesTunic,
			want: []Modifier{
				{Stat: PDef, Op: OpAdd, Value: 17, Owner: owner},
				{Stat: MaxMP, Op: OpAdd, Value: 19, Owner: owner},
			},
		},
		{
			// Enchant +5 on a no-grade sword: P.Atk and M.Atk both
			// 2×3 + 4×2 = 14, after the stats.
			name: "Squire's Sword +5",
			item: withEnchant(squiresSword, 5),
			want: []Modifier{
				{Stat: PAtk, Op: OpSet, Value: 6, Owner: owner},
				{Stat: MAtk, Op: OpSet, Value: 5, Owner: owner},
				{Stat: CritRate, Op: OpSet, Value: 8, Owner: owner},
				{Stat: PAtkSpd, Op: OpSet, Value: 379, Owner: owner},
				{Stat: RandomDamage, Op: OpAdd, Value: 10, Owner: owner},
				{Stat: PAtkRange, Op: OpSet, Value: 40, Owner: owner},
				{Stat: PAtk, Op: OpEnchant, Value: 14, Owner: owner},
				{Stat: MAtk, Op: OpEnchant, Value: 14, Owner: owner},
			},
		},
		{
			// Armor enchant +4: P.Def 3 + 3×1 = 6.
			name: "Squire's Shirt +4",
			item: withEnchant(squiresShirt, 4),
			want: []Modifier{
				{Stat: PDef, Op: OpAdd, Value: 33, Owner: owner},
				{Stat: PDef, Op: OpEnchant, Value: 6, Owner: owner},
			},
		},
		{
			// Shields get FuncEnchant on sDef, which adds nothing.
			name: "shield +5",
			item: Item{Type: "Armor", BodyPart: "lhand", ArmorType: "SHIELD", Grade: "D", Enchant: 5,
				Stats: []ItemStat{{"sDef", 79}, {"rShld", 20}, {"rEvas", -8}}},
			want: []Modifier{
				{Stat: SDef, Op: OpSet, Value: 79, Owner: owner},
				{Stat: RShld, Op: OpSet, Value: 20, Owner: owner},
				{Stat: REvas, Op: OpAdd, Value: -8, Owner: owner},
			},
		},
		{
			// Jewel enchant +4: M.Def 3 + 3×1 = 6.
			name: "necklace +4",
			item: Item{Type: "Armor", BodyPart: "neck", Grade: "C", Enchant: 4, Stats: []ItemStat{{"mDef", 37}}},
			want: []Modifier{
				{Stat: MDef, Op: OpAdd, Value: 37, Owner: owner},
				{Stat: MDef, Op: OpEnchant, Value: 6, Owner: owner},
			},
		},
		{
			// Enchant functions come with the stats element.
			name: "enchanted item without stats",
			item: Item{Type: "Armor", BodyPart: "hair", Enchant: 5},
			want: nil,
		},
		{
			// magicSuccRes is a Mul; stats DocumentItem does not handle
			// are skipped.
			name: "Mul and skipped stats",
			item: Item{Type: "Armor", BodyPart: "head", Stats: []ItemStat{{"magicSuccRes", 1.1}, {"STR", 2}, {"maxHp", 100}}},
			want: []Modifier{{Stat: MagicSuccRes, Op: OpMul, Value: 1.1, Owner: owner}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ItemModifiers(owner, tt.item)
			if err != nil {
				t.Fatalf("ItemModifiers: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ItemModifiers =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}

func withEnchant(it Item, level int) Item {
	it.Enchant = level
	return it
}

func TestItemModifiersErrors(t *testing.T) {
	for name, it := range map[string]Item{
		"unknown stat":  {Type: "Weapon", Stats: []ItemStat{{"pAttack", 1}}},
		"unknown grade": {Type: "Weapon", Grade: "S80"},
		"unknown type":  {Type: "Pet"},
	} {
		if _, err := ItemModifiers(ItemOwner(1), it); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: error = %v, want ErrInvalid", name, err)
		}
	}
}

func TestItemKind(t *testing.T) {
	tests := []struct {
		name string
		item Item
		want itemKind
	}{
		{"weapon", Item{Type: "Weapon", BodyPart: "lrhand"}, itemWeapon},
		{"etc item", Item{Type: "EtcItem"}, itemOther},
		{"body armor", Item{Type: "Armor", BodyPart: "chest", ArmorType: "HEAVY"}, itemArmor},
		{"earrings", Item{Type: "Armor", BodyPart: "rear;lear"}, itemJewel},
		{"bracelet is an ear", Item{Type: "Armor", BodyPart: "rbracelet"}, itemJewel},
		{"rings", Item{Type: "Armor", BodyPart: "rfinger;lfinger"}, itemJewel},
		{"necklace", Item{Type: "Armor", BodyPart: "neck"}, itemJewel},
		{"shield", Item{Type: "Armor", BodyPart: "lhand", ArmorType: "SHIELD"}, itemShield},
		{"left hand armor of type NONE is a shield", Item{Type: "Armor", BodyPart: "lhand"}, itemShield},
		{"sigil is armor", Item{Type: "Armor", BodyPart: "lhand", ArmorType: "SIGIL"}, itemArmor},
	}
	for _, tt := range tests {
		got, err := tt.item.kind()
		if err != nil || got != tt.want {
			t.Errorf("%s: kind() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// TestEnchantBonus checks FuncEnchant: the first 3 levels count once, every
// level above 3 counts double (triple for defence).
func TestEnchantBonus(t *testing.T) {
	tests := []struct {
		name      string
		stat      Stat
		kind      itemKind
		grade     grade
		bow, twoH bool
		enchant   int
		want      int
	}{
		{"not enchanted", PAtk, itemWeapon, gradeD, false, false, 0, 0},
		{"no-grade sword +5: 2×3 + 4×2", PAtk, itemWeapon, gradeNone, false, false, 5, 14},
		{"D bow +4: 4×3 + 8×1", PAtk, itemWeapon, gradeD, true, true, 4, 20},
		{"D two-handed sword +4: 2×3 + 4×1", PAtk, itemWeapon, gradeD, false, true, 4, 10},
		{"C one-handed +3: 3×3", PAtk, itemWeapon, gradeC, false, false, 3, 9},
		{"B two-handed +4: 4×3 + 8×1", PAtk, itemWeapon, gradeB, false, true, 4, 20},
		{"B bow +5: 6×3 + 12×2", PAtk, itemWeapon, gradeB, true, true, 5, 42},
		{"A one-handed +6: 4×3 + 8×3", PAtk, itemWeapon, gradeA, false, false, 6, 36},
		{"A two-handed +6: 5×3 + 10×3", PAtk, itemWeapon, gradeA, false, true, 6, 45},
		{"A bow +4: 8×3 + 16×1", PAtk, itemWeapon, gradeA, true, true, 4, 40},
		{"S one-handed +4: 5×3 + 10×1", PAtk, itemWeapon, gradeS, false, false, 4, 25},
		{"S two-handed +4: 6×3 + 12×1", PAtk, itemWeapon, gradeS, false, true, 4, 30},
		{"S bow +4: 10×3 + 20×1", PAtk, itemWeapon, gradeS, true, true, 4, 50},
		{"S M.Atk +4: 4×3 + 8×1", MAtk, itemWeapon, gradeS, false, false, 4, 20},
		{"A M.Atk +4: 3×3 + 6×1", MAtk, itemWeapon, gradeA, false, false, 4, 15},
		{"D M.Atk +2: 2×2", MAtk, itemWeapon, gradeD, false, false, 2, 4},
		{"armor P.Def +5: 3 + 3×2", PDef, itemArmor, gradeC, false, false, 5, 9},
		{"jewel M.Def +4: 3 + 3×1", MDef, itemJewel, gradeD, false, false, 4, 6},
		{"shield S.Def +5: nothing", SDef, itemShield, gradeD, false, false, 5, 0},
	}
	for _, tt := range tests {
		if got := enchantBonus(tt.stat, tt.kind, tt.grade, tt.bow, tt.twoH, tt.enchant); got != tt.want {
			t.Errorf("%s: enchantBonus = %d, want %d", tt.name, got, tt.want)
		}
	}
}
