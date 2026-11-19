package stats

import "fmt"

// Item is an equipped item as the stat engine needs it. The strings use the
// datapack names (Mobius item XML).
type Item struct {
	// Type is the item class: "Weapon", "Armor" or "EtcItem".
	Type string
	// BodyPart is the bodypart set, for example "rhand", "lrhand", "chest",
	// "fullarmor" or "rear;lear".
	BodyPart string
	// WeaponType is weapon_type, for example "SWORD" or "BOW". Only "BOW"
	// changes anything here (the enchant bonus).
	WeaponType string
	// ArmorType is armor_type, for example "LIGHT" or "SHIELD"; empty means
	// "NONE".
	ArmorType string
	// Grade is crystal_type: "NONE", "D", "C", "B", "A" or "S"; empty
	// means "NONE".
	Grade string
	// Enchant is the enchant level of this item.
	Enchant int
	// Stats is the stats list of the item, in file order.
	Stats []ItemStat
}

// ItemStat is one stat entry of an item: <stat type="pAtk">6</stat>.
type ItemStat struct {
	// Stat is the Mobius XML stat name, for example "pAtk".
	Stat  string
	Value float64
}

// itemKind is what decides an item's enchant function (DocumentItem).
type itemKind uint8

const (
	itemOther  itemKind = iota // etc items: no enchant function
	itemWeapon                 // enchant raises P.Atk and M.Atk
	itemArmor                  // enchant raises P.Def
	itemShield                 // enchant on S.Def, which has no effect (FuncEnchant)
	itemJewel                  // earrings, rings, necklaces: enchant raises M.Def
)

// kind classifies the item like the Mobius Weapon, Armor and EtcItem
// classes: armor on a jewel body part is a jewel (Armor.set), armor of type
// SHIELD or of type NONE in the left hand is a shield.
func (it *Item) kind() (itemKind, error) {
	switch it.Type {
	case "Weapon":
		return itemWeapon, nil
	case "EtcItem":
		return itemOther, nil
	case "Armor":
		switch it.BodyPart {
		case "neck", "rear", "rbracelet", "lear", "rear;lear", "rfinger", "lfinger", "rfinger;lfinger":
			return itemJewel, nil
		}
		armorType := it.ArmorType
		if armorType == "" {
			armorType = "NONE"
		}
		if armorType == "SHIELD" || (armorType == "NONE" && it.BodyPart == "lhand") {
			return itemShield, nil
		}
		return itemArmor, nil
	}
	return 0, fmt.Errorf("stats: %w: unknown item type %q", ErrInvalid, it.Type)
}

// fullArmor reports whether the item covers chest and legs (BodyPart
// FULL_ARMOR, named "fullarmor" or "onepiece").
func (it *Item) fullArmor() bool { return it.BodyPart == "fullarmor" || it.BodyPart == "onepiece" }

// grade is a crystal grade (CrystalType).
type grade uint8

const (
	gradeNone grade = iota
	gradeD
	gradeC
	gradeB
	gradeA
	gradeS
)

func parseGrade(s string) (grade, error) {
	switch s {
	case "", "NONE":
		return gradeNone, nil
	case "D":
		return gradeD, nil
	case "C":
		return gradeC, nil
	case "B":
		return gradeB, nil
	case "A":
		return gradeA, nil
	case "S":
		return gradeS, nil
	}
	return 0, fmt.Errorf("stats: %w: unknown crystal grade %q", ErrInvalid, s)
}

// itemStatOp is the operation DocumentItem attaches for an item stat. The
// stats it does not list are skipped (Mobius logs "Unhandled stat type");
// no Interlude item has one.
func itemStatOp(s Stat) (Op, bool) {
	switch s {
	case PAtk, PAtkSpd, PAtkRange, CritRate, MAtk, SDef, RShld:
		return OpSet, true
	case AccCombat, REvas, RandomDamage, PDef, MDef, MaxMP,
		FirePower, WaterPower, WindPower, EarthPower, HolyPower, DarkPower,
		FireRes, WaterRes, WindRes, EarthRes, HolyRes, DarkRes:
		return OpAdd, true
	case MagicSuccRes:
		return OpMul, true
	}
	return 0, false
}

// ItemModifiers returns the modifiers an item adds while it is equipped, as
// Mobius DocumentItem attaches them. All of them are at order 0, before the
// formula functions:
//
//   - Set (replacing the base value): pAtk, pAtkSpd, pAtkRange, critRate,
//     mAtk, sDef, rShld;
//   - Add: accCombat, rEvas, randomDamage, pDef, mDef, maxMp, and the
//     element powers and resistances;
//   - Mul: magicSuccRes;
//   - any other stat is skipped, as Mobius does.
//
// Then the enchant bonus (FuncEnchant, computed from the item's enchant
// level): P.Atk and M.Atk for weapons, M.Def for jewels, P.Def for other
// armor, nothing for shields and for items without stats. A zero bonus adds
// no modifier.
//
// Mobius also gives graded armor an EnchantHp function on max HP; its
// Interlude table (enchantHPBonus.xml) is all zeros, so it is left out.
//
// Unknown stat names, item types and grades are errors.
func ItemModifiers(owner Owner, it Item) ([]Modifier, error) {
	kind, err := it.kind()
	if err != nil {
		return nil, err
	}
	g, err := parseGrade(it.Grade)
	if err != nil {
		return nil, err
	}
	mods := make([]Modifier, 0, len(it.Stats)+2)
	for _, is := range it.Stats {
		s, ok := ParseStat(is.Stat)
		if !ok {
			return nil, fmt.Errorf("stats: %w: unknown item stat %q", ErrInvalid, is.Stat)
		}
		op, ok := itemStatOp(s)
		if !ok {
			continue
		}
		mods = append(mods, Modifier{Stat: s, Op: op, Order: OrderSet, Value: is.Value, Owner: owner})
	}

	// DocumentItem attaches the enchant functions while it reads the stats
	// element, so items without stats get none.
	var enchanted []Stat
	switch {
	case len(it.Stats) == 0:
	case kind == itemWeapon:
		enchanted = []Stat{PAtk, MAtk}
	case kind == itemArmor:
		enchanted = []Stat{PDef}
	case kind == itemJewel:
		enchanted = []Stat{MDef}
	}
	for _, s := range enchanted {
		if b := enchantBonus(s, kind, g, it.WeaponType == "BOW", it.BodyPart == "lrhand", it.Enchant); b != 0 {
			mods = append(mods, Modifier{Stat: s, Op: OpEnchant, Order: OrderSet, Value: float64(b), Owner: owner})
		}
	}
	return mods, nil
}

// enchantBonus is FuncEnchant: what an item enchanted to level enchant adds
// to stat. The first 3 levels count once, every level above 3 counts double
// (triple for defence). The Olympiad enchant limit is not applied (there is
// no Olympiad yet).
//
// Mobius adds the parts one after the other (value + enchant + 3*over for
// defence); a precomputed sum gives the same double for the integer-valued
// defence totals of the data.
func enchantBonus(stat Stat, kind itemKind, g grade, bow, twoHanded bool, enchant int) int {
	if enchant <= 0 {
		return 0
	}
	over := 0
	if enchant > 3 {
		over = enchant - 3
		enchant = 3
	}
	if stat == MDef || stat == PDef {
		return enchant + 3*over
	}
	if stat == MAtk {
		switch g {
		case gradeS:
			return 4*enchant + 8*over
		case gradeA, gradeB, gradeC:
			return 3*enchant + 6*over
		default: // D and NONE
			return 2*enchant + 4*over
		}
	}
	if kind != itemWeapon {
		return 0
	}
	switch g {
	case gradeS:
		switch {
		case twoHanded && bow:
			return 10*enchant + 20*over
		case twoHanded:
			return 6*enchant + 12*over
		}
		return 5*enchant + 10*over
	case gradeA:
		switch {
		case twoHanded && bow:
			return 8*enchant + 16*over
		case twoHanded:
			return 5*enchant + 10*over
		}
		return 4*enchant + 8*over
	case gradeB, gradeC:
		switch {
		case twoHanded && bow:
			return 6*enchant + 12*over
		case twoHanded:
			return 4*enchant + 8*over
		}
		return 3*enchant + 6*over
	}
	// D and NONE: bows by 4, everything else by 2, whatever the hands.
	if bow {
		return 4*enchant + 8*over
	}
	return 2*enchant + 4*over
}
