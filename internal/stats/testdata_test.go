package stats

import "testing"

// Test data copied from the Mobius Interlude datapack
// (L2J_Mobius_CT_0_Interlude/dist/game/data/stats/).

// mobiusBonus is statBonus.xml: the bonus of each stat value 0 to 99.
var mobiusBonus = BonusTables{
	STR: [BaseStatValues]float64{
		0.29, 0.3, 0.31, 0.32, 0.34, 0.35, 0.36, 0.37, 0.39, 0.4, // 0-9
		0.42, 0.43, 0.45, 0.46, 0.48, 0.5, 0.51, 0.53, 0.55, 0.57, // 10-19
		0.59, 0.61, 0.63, 0.66, 0.68, 0.71, 0.73, 0.76, 0.78, 0.81, // 20-29
		0.84, 0.87, 0.9, 0.94, 0.97, 1.01, 1.04, 1.08, 1.12, 1.16, // 30-39
		1.2, 1.24, 1.29, 1.33, 1.38, 1.43, 1.48, 1.54, 1.59, 1.65, // 40-49
		1.71, 1.77, 1.83, 1.9, 1.97, 2.04, 2.11, 2.19, 2.27, 2.35, // 50-59
		2.43, 2.52, 2.61, 2.71, 2.8, 2.91, 3.01, 3.12, 3.23, 3.35, // 60-69
		3.47, 3.59, 3.72, 3.86, 3.99, 4.14, 4.29, 4.44, 4.6, 4.77, // 70-79
		4.94, 5.12, 5.3, 5.49, 5.69, 5.89, 6.11, 6.33, 6.55, 6.79, // 80-89
		7.03, 7.29, 7.55, 7.82, 8.1, 8.39, 8.7, 9.01, 9.33, 9.67, // 90-99
	},
	INT: [BaseStatValues]float64{
		0.54, 0.55, 0.56, 0.57, 0.58, 0.59, 0.61, 0.62, 0.63, 0.64, // 0-9
		0.65, 0.67, 0.68, 0.69, 0.71, 0.72, 0.74, 0.75, 0.77, 0.78, // 10-19
		0.8, 0.81, 0.83, 0.85, 0.86, 0.88, 0.9, 0.92, 0.94, 0.95, // 20-29
		0.97, 0.99, 1.01, 1.03, 1.05, 1.07, 1.1, 1.12, 1.14, 1.16, // 30-39
		1.19, 1.21, 1.23, 1.26, 1.28, 1.31, 1.34, 1.36, 1.39, 1.42, // 40-49
		1.45, 1.47, 1.5, 1.53, 1.57, 1.6, 1.63, 1.66, 1.69, 1.73, // 50-59
		1.76, 1.8, 1.83, 1.87, 1.91, 1.95, 1.99, 2.02, 2.07, 2.11, // 60-69
		2.15, 2.19, 2.24, 2.28, 2.33, 2.37, 2.42, 2.47, 2.52, 2.57, // 70-79
		2.62, 2.67, 2.73, 2.78, 2.84, 2.89, 2.95, 3.01, 3.07, 3.13, // 80-89
		3.19, 3.26, 3.32, 3.39, 3.46, 3.53, 3.6, 3.67, 3.74, 3.82, // 90-99
	},
	CON: [BaseStatValues]float64{
		0.45, 0.46, 0.47, 0.48, 0.5, 0.51, 0.53, 0.54, 0.56, 0.58, // 0-9
		0.59, 0.61, 0.63, 0.65, 0.67, 0.69, 0.71, 0.73, 0.75, 0.77, // 10-19
		0.8, 0.82, 0.85, 0.87, 0.9, 0.93, 0.95, 0.98, 1.01, 1.04, // 20-29
		1.07, 1.1, 1.14, 1.17, 1.21, 1.24, 1.28, 1.32, 1.36, 1.4, // 30-39
		1.44, 1.48, 1.53, 1.58, 1.62, 1.67, 1.72, 1.77, 1.83, 1.88, // 40-49
		1.94, 2, 2.06, 2.12, 2.18, 2.25, 2.31, 2.38, 2.45, 2.53, // 50-59
		2.6, 2.68, 2.76, 2.84, 2.93, 3.02, 3.11, 3.2, 3.3, 3.4, // 60-69
		3.5, 3.6, 3.71, 3.82, 3.94, 4.06, 4.18, 4.3, 4.43, 4.56, // 70-79
		4.7, 4.84, 4.99, 5.14, 5.29, 5.45, 5.61, 5.78, 5.96, 6.13, // 80-89
		6.32, 6.51, 6.7, 6.9, 7.11, 7.33, 7.54, 7.77, 8, 8.24, // 90-99
	},
	MEN: [BaseStatValues]float64{
		1, 1.01, 1.02, 1.03, 1.04, 1.05, 1.06, 1.07, 1.08, 1.09, // 0-9
		1.11, 1.12, 1.13, 1.14, 1.15, 1.16, 1.17, 1.19, 1.2, 1.21, // 10-19
		1.22, 1.23, 1.25, 1.26, 1.27, 1.28, 1.3, 1.31, 1.32, 1.34, // 20-29
		1.35, 1.36, 1.38, 1.39, 1.4, 1.42, 1.43, 1.45, 1.46, 1.48, // 30-39
		1.49, 1.5, 1.52, 1.53, 1.55, 1.57, 1.58, 1.6, 1.61, 1.63, // 40-49
		1.65, 1.66, 1.68, 1.7, 1.71, 1.73, 1.75, 1.76, 1.78, 1.8, // 50-59
		1.82, 1.84, 1.85, 1.87, 1.89, 1.91, 1.93, 1.95, 1.97, 1.99, // 60-69
		2.01, 2.03, 2.05, 2.07, 2.09, 2.11, 2.13, 2.15, 2.17, 2.2, // 70-79
		2.22, 2.24, 2.26, 2.29, 2.31, 2.33, 2.35, 2.38, 2.4, 2.43, // 80-89
		2.45, 2.47, 2.5, 2.52, 2.55, 2.58, 2.6, 2.63, 2.65, 2.68, // 90-99
	},
	DEX: [BaseStatValues]float64{
		0.84, 0.85, 0.86, 0.86, 0.87, 0.88, 0.89, 0.9, 0.9, 0.91, // 0-9
		0.92, 0.93, 0.94, 0.94, 0.95, 0.96, 0.97, 0.98, 0.99, 1, // 10-19
		1.01, 1.01, 1.02, 1.03, 1.04, 1.05, 1.06, 1.07, 1.08, 1.09, // 20-29
		1.1, 1.11, 1.12, 1.13, 1.14, 1.15, 1.16, 1.17, 1.18, 1.19, // 30-39
		1.2, 1.21, 1.22, 1.24, 1.25, 1.26, 1.27, 1.28, 1.29, 1.3, // 40-49
		1.32, 1.33, 1.34, 1.35, 1.36, 1.38, 1.39, 1.4, 1.41, 1.43, // 50-59
		1.44, 1.45, 1.47, 1.48, 1.49, 1.51, 1.52, 1.53, 1.55, 1.56, // 60-69
		1.57, 1.59, 1.6, 1.62, 1.63, 1.65, 1.66, 1.68, 1.69, 1.71, // 70-79
		1.72, 1.74, 1.75, 1.77, 1.78, 1.8, 1.82, 1.83, 1.85, 1.87, // 80-89
		1.88, 1.9, 1.92, 1.93, 1.95, 1.97, 1.99, 2, 2.02, 2.04, // 90-99
	},
	WIT: [BaseStatValues]float64{
		0.39, 0.4, 0.42, 0.44, 0.46, 0.48, 0.51, 0.53, 0.56, 0.58, // 0-9
		0.61, 0.64, 0.68, 0.71, 0.75, 0.78, 0.82, 0.86, 0.91, 0.95, // 10-19
		1, 1.05, 1.1, 1.16, 1.22, 1.28, 1.34, 1.41, 1.48, 1.55, // 20-29
		1.63, 1.71, 1.8, 1.89, 1.98, 2.08, 2.18, 2.29, 2.41, 2.53, // 30-39
		2.65, 2.79, 2.93, 3.07, 3.23, 3.39, 3.56, 3.73, 3.92, 4.12, // 40-49
		4.32, 4.54, 4.76, 5, 5.25, 5.52, 5.79, 6.08, 6.39, 6.7, // 50-59
		7.04, 7.39, 7.76, 8.15, 8.56, 8.99, 9.43, 9.91, 10.4, 10.92, // 60-69
		11.47, 12.04, 12.64, 13.27, 13.94, 14.64, 15.37, 16.14, 16.94, 17.79, // 70-79
		18.68, 19.61, 20.59, 21.62, 22.7, 23.84, 25.03, 26.28, 27.6, 28.98, // 80-89
		30.43, 31.95, 33.55, 35.22, 36.98, 38.83, 40.77, 42.81, 44.95, 47.2, // 90-99
	},
}

// Bonuses used in the hand computations of the tests. They are variables,
// not constants, so expressions with them use float64 arithmetic like
// Java's doubles (Go evaluates constant expressions exactly).
var (
	dex21 = mobiusBonus.DEX[21] // 1.01
	dex30 = mobiusBonus.DEX[30] // 1.1
)

// humanFighter is class 0 (StartingClass/HumanFighter.xml), levels 1-20.
func humanFighter() *ClassTemplate {
	return &ClassTemplate{
		ClassID:     0,
		Race:        0,
		Base:        BaseStats{STR: 40, CON: 43, DEX: 30, INT: 21, WIT: 11, MEN: 25},
		PAtk:        4,
		MAtk:        6,
		PAtkSpd:     300,
		CritRate:    4,
		AttackRange: 20,
		PDef:        ArmorDefs{Chest: 31, Legs: 18, Head: 12, Feet: 7, Gloves: 8, Underwear: 3, Cloak: 1},
		MDef:        JewelDefs{REar: 9, LEar: 9, RFinger: 5, LFinger: 5, Neck: 13},
		RunSpeed:    115, WalkSpeed: 80, SwimRunSpeed: 50, SwimWalkSpeed: 50,
		Male:   Collision{Radius: 9.0, Height: 23},
		Female: Collision{Radius: 8.0, Height: 23.5},
		Levels: []LevelGain{
			{HP: 80.0, MP: 30.0, CP: 32.0, HPRegen: 2.0, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 91.83, MP: 35.46, CP: 36.732, HPRegen: 2.05, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 103.79, MP: 40.98, CP: 41.516, HPRegen: 2.1, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 115.88, MP: 46.56, CP: 46.352, HPRegen: 2.15, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 128.1, MP: 52.2, CP: 51.24, HPRegen: 2.2, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 140.45, MP: 57.9, CP: 56.18, HPRegen: 2.25, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 152.93, MP: 63.66, CP: 61.172, HPRegen: 2.3, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 165.54, MP: 69.48, CP: 66.216, HPRegen: 2.35, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 178.28, MP: 75.36, CP: 71.312, HPRegen: 2.4, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 191.15, MP: 81.3, CP: 76.46, HPRegen: 2.45, MPRegen: 0.9, CPRegen: 2.0},
			{HP: 204.15, MP: 87.3, CP: 81.66, HPRegen: 2.5, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 217.28, MP: 93.36, CP: 86.912, HPRegen: 2.6, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 230.54, MP: 99.48, CP: 92.216, HPRegen: 2.7, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 243.93, MP: 105.66, CP: 97.572, HPRegen: 2.8, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 257.45, MP: 111.9, CP: 102.98, HPRegen: 2.9, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 271.1, MP: 118.2, CP: 108.44, HPRegen: 3.0, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 284.88, MP: 124.56, CP: 113.952, HPRegen: 3.1, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 298.79, MP: 130.98, CP: 119.516, HPRegen: 3.2, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 312.83, MP: 137.46, CP: 125.132, HPRegen: 3.3, MPRegen: 1.2, CPRegen: 2.5},
			{HP: 327.0, MP: 144.0, CP: 130.8, HPRegen: 3.4, MPRegen: 1.2, CPRegen: 2.5},
		},
	}
}

// humanMystic is class 10 (StartingClass/HumanMystic.xml), level 1 only.
func humanMystic() *ClassTemplate {
	return &ClassTemplate{
		ClassID:     10,
		Race:        0,
		Base:        BaseStats{STR: 22, CON: 27, DEX: 21, INT: 41, WIT: 20, MEN: 39},
		PAtk:        3,
		MAtk:        6,
		PAtkSpd:     300,
		CritRate:    4,
		AttackRange: 20,
		PDef:        ArmorDefs{Chest: 15, Legs: 8, Head: 12, Feet: 7, Gloves: 8, Underwear: 3, Cloak: 1},
		MDef:        JewelDefs{REar: 9, LEar: 9, RFinger: 5, LFinger: 5, Neck: 13},
		RunSpeed:    120, WalkSpeed: 78, SwimRunSpeed: 50, SwimWalkSpeed: 50,
		Male:   Collision{Radius: 7.5, Height: 22.8},
		Female: Collision{Radius: 6.5, Height: 22.5},
		Levels: []LevelGain{
			{HP: 101.0, MP: 40.0, CP: 50.5, HPRegen: 2.0, MPRegen: 0.9, CPRegen: 2.0},
		},
	}
}

// gremlin is NPC 20001 (npcs/20000-20099.xml).
func gremlin() *NpcTemplate {
	return &NpcTemplate{
		Level:       1,
		Base:        BaseStats{STR: 40, INT: 21, DEX: 30, WIT: 20, CON: 43, MEN: 10},
		HP:          39.74519,
		MP:          40,
		HPRegen:     2,
		MPRegen:     0.9,
		PAtk:        8.47458,
		MAtk:        5.78704,
		PDef:        44.44444,
		MDef:        29.5916164000214,
		CritRate:    4,
		PAtkSpd:     253,
		AttackRange: 40,
		RunSpeed:    50,
		WalkSpeed:   20,
		Collision:   Collision{Radius: 10, Height: 15},
	}
}

// Starting items (items/*.xml).
var (
	squiresSword = Item{ // 2369
		Type: "Weapon", BodyPart: "rhand", WeaponType: "SWORD",
		Stats: []ItemStat{{"pAtk", 6}, {"mAtk", 5}, {"critRate", 8}, {"pAtkSpd", 379}, {"randomDamage", 10}, {"pAtkRange", 40}},
	}
	squiresShirt = Item{ // 1146
		Type: "Armor", BodyPart: "chest", ArmorType: "LIGHT",
		Stats: []ItemStat{{"pDef", 33}},
	}
	squiresPants = Item{ // 1147
		Type: "Armor", BodyPart: "legs", ArmorType: "LIGHT",
		Stats: []ItemStat{{"pDef", 20}},
	}
	apprenticesWand = Item{ // 6
		Type: "Weapon", BodyPart: "rhand", WeaponType: "BLUNT",
		Stats: []ItemStat{{"pAtk", 5}, {"mAtk", 7}, {"critRate", 4}, {"accCombat", 4.75}, {"pAtkSpd", 379}, {"randomDamage", 20}, {"pAtkRange", 40}},
	}
	apprenticesTunic = Item{ // 425
		Type: "Armor", BodyPart: "chest", ArmorType: "MAGIC",
		Stats: []ItemStat{{"pDef", 17}, {"maxMp", 19}},
	}
	apprenticesStockings = Item{ // 461
		Type: "Armor", BodyPart: "legs", ArmorType: "MAGIC",
		Stats: []ItemStat{{"pDef", 10}, {"maxMp", 10}},
	}
)

// equipped is an item to put on a test player.
type equipped struct {
	slot Slot
	id   int32 // object ID
	item Item
}

// fighterGear and mysticGear are the equipped starting items of classes 0
// and 10 (initialEquipment.xml).
var (
	fighterGear = []equipped{
		{SlotRHand, 1, squiresSword},
		{SlotChest, 2, squiresShirt},
		{SlotLegs, 3, squiresPants},
	}
	mysticGear = []equipped{
		{SlotRHand, 1, apprenticesWand},
		{SlotChest, 2, apprenticesTunic},
		{SlotLegs, 3, apprenticesStockings},
	}
)

func testRules(t *testing.T, cfg Config) *Rules {
	t.Helper()
	r, err := NewRules(cfg, mobiusBonus)
	if err != nil {
		t.Fatalf("NewRules: %v", err)
	}
	return r
}

// newTestPlayer returns a running player of class tmpl at level with gear
// equipped. Mobius loads players running.
func newTestPlayer(t *testing.T, r *Rules, tmpl *ClassTemplate, level int, gear ...equipped) *Player {
	t.Helper()
	p, err := NewPlayer(r, tmpl, level)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	for _, g := range gear {
		if err := p.Equip(g.slot, ItemOwner(g.id), g.item); err != nil {
			t.Fatalf("Equip(%d): %v", g.slot, err)
		}
	}
	p.SetMotion(Motion{Running: true})
	return p
}

// moveMul is Mobius's movement multiplier, getMoveSpeed() * (1. / baseSpeed).
func moveMul(speed, base float64) float64 { return speed * (1 / base) }

// atkMul is Mobius's attack speed multiplier,
// (float) ((1.1 * getPAtkSpd()) / basePAtkSpd).
func atkMul(pAtkSpd, base float64) float32 { return float32(1.1 * pAtkSpd / base) }
