package stats

import (
	"math"
	"testing"
)

// TestPlayerViewStartingClasses checks the UserInfo numbers of new
// characters against values computed by hand from the Mobius formulas and
// data. Bonus values (statBonus.xml): STR 40 1.2, STR 22 0.63, DEX 30 1.1,
// DEX 21 1.01, CON 43 1.58, CON 27 0.98, INT 21 0.81, INT 41 1.21, WIT 11
// 0.64, WIT 20 1, MEN 25 1.28, MEN 39 1.48. Level modifier (level+89)/100:
// 0.9 at level 1, 1.09 at level 20. (int) truncates, Math.round rounds half
// up, crit is (int)(x + 0.5).
func TestPlayerViewStartingClasses(t *testing.T) {
	r := testRules(t, DefaultConfig())
	tests := []struct {
		name  string
		tmpl  *ClassTemplate
		level int
		gear  []equipped
		want  PlayerView // the floating-point fields are set from the rest below
		// speed is the running speed, base the template run speed;
		// pAtkSpd and basePAtkSpd give the attack speed multiplier.
		speed, base          float64
		pAtkSpd, basePAtkSpd float64
		radius, height       float64
	}{
		{
			name:  "Human Fighter level 1, nothing equipped",
			tmpl:  humanFighter(),
			level: 1,
			want: PlayerView{
				STR: 40, DEX: 30, CON: 43, INT: 21, WIT: 11, MEN: 25,
				MaxHP:          126,    // 80 × 1.58 = 126.4
				MaxMP:          38,     // 30 × 1.28 = 38.4
				MaxCP:          50,     // 32 × 1.58 = 50.56
				WeightLimit:    109020, // floor(1.58 × 69000)
				InventoryLimit: 80,
				PAtk:           4,   // 4 × 1.2 × 0.9 = 4.32
				PAtkSpd:        330, // round(300 × 1.1)
				PDef:           72,  // (31+18+12+7+8+3+1) × 0.9 = 80 × 0.9
				Evasion:        34,  // round(sqrt(30) × 6 + 1) = round(33.86)
				Accuracy:       34,  // same formula
				Crit:           44,  // (int)(4 × 1.1 × 10 + 0.5)
				MAtk:           3,   // 6 × 0.81² × 0.9² = 3.19
				MAtkSpd:        213, // 333 × 0.64 = 213.12
				MDef:           47,  // (9+9+5+5+13) × 1.28 × 0.9 = 47.23
				RunSpd:         115, // run 115 × 1.1 sent divided by the multiplier
				WalkSpd:        80,  // 80 × 1.1 / 1.1
				SwimRunSpd:     50,  // 50 × 1.1 / 1.1
				SwimWalkSpd:    50,
			},
			speed: 115 * dex30, base: 115,
			pAtkSpd: 330, basePAtkSpd: 300, // 1.1 × 330 / 300 = 1.21
			radius: 9, height: 23,
		},
		{
			name:  "Human Fighter level 1, starting gear",
			tmpl:  humanFighter(),
			level: 1,
			gear:  fighterGear,
			want: PlayerView{
				STR: 40, DEX: 30, CON: 43, INT: 21, WIT: 11, MEN: 25,
				MaxHP: 126, MaxMP: 38, MaxCP: 50,
				WeightLimit: 109020, InventoryLimit: 80,
				PAtk:     6,   // sword sets 6: 6 × 1.2 × 0.9 = 6.48
				PAtkSpd:  417, // sword sets 379: round(379 × 1.1) = round(416.9)
				PDef:     75,  // (80 + 33 + 20 - 31 - 18) × 0.9 = 84 × 0.9 = 75.6
				Evasion:  34,
				Accuracy: 34,
				Crit:     88, // sword sets 8: (int)(8 × 1.1 × 10 + 0.5)
				MAtk:     2,  // sword sets 5: 5 × 0.81² × 0.9² = 2.66
				MAtkSpd:  213,
				MDef:     47,
				RunSpd:   115, WalkSpd: 80, SwimRunSpd: 50, SwimWalkSpd: 50,
			},
			speed: 115 * dex30, base: 115,
			pAtkSpd: 417, basePAtkSpd: 300, // 1.529
			radius: 9, height: 23,
		},
		{
			name:  "Human Fighter level 20, starting gear",
			tmpl:  humanFighter(),
			level: 20,
			gear:  fighterGear,
			want: PlayerView{
				STR: 40, DEX: 30, CON: 43, INT: 21, WIT: 11, MEN: 25,
				MaxHP:       516, // 327 × 1.58 = 516.66
				MaxMP:       184, // 144 × 1.28 = 184.32
				MaxCP:       206, // 130.8 × 1.58 = 206.66
				WeightLimit: 109020, InventoryLimit: 80,
				PAtk:     7, // 6 × 1.2 × 1.09 = 7.85
				PAtkSpd:  417,
				PDef:     91, // 84 × 1.09 = 91.56
				Evasion:  53, // round(32.86 + 20)
				Accuracy: 53,
				Crit:     88,
				MAtk:     3, // 5 × 0.81² × 1.09² = 3.90
				MAtkSpd:  213,
				MDef:     57, // 41 × 1.28 × 1.09 = 57.20
				RunSpd:   115, WalkSpd: 80, SwimRunSpd: 50, SwimWalkSpd: 50,
			},
			speed: 115 * dex30, base: 115,
			pAtkSpd: 417, basePAtkSpd: 300,
			radius: 9, height: 23,
		},
		{
			name:  "Human Mystic level 1, nothing equipped",
			tmpl:  humanMystic(),
			level: 1,
			want: PlayerView{
				STR: 22, DEX: 21, CON: 27, INT: 41, WIT: 20, MEN: 39,
				MaxHP:          98,    // 101 × 0.98 = 98.98
				MaxMP:          59,    // 40 × 1.48 = 59.2
				MaxCP:          49,    // 50.5 × 0.98 = 49.49
				WeightLimit:    67620, // floor(0.98 × 69000)
				InventoryLimit: 80,
				PAtk:           1,   // 3 × 0.63 × 0.9 = 1.70
				PAtkSpd:        303, // round(300 × 1.01)
				PDef:           48,  // (15+8+12+7+8+3+1) × 0.9 = 54 × 0.9 = 48.6
				Evasion:        28,  // round(sqrt(21) × 6 + 1) = round(28.495)
				Accuracy:       28,
				Crit:           40,  // (int)(4 × 1.01 × 10 + 0.5) = (int)40.9
				MAtk:           7,   // 6 × 1.21² × 0.9² = 7.12
				MAtkSpd:        333, // 333 × 1
				MDef:           54,  // 41 × 1.48 × 0.9 = 54.61
				RunSpd:         120, WalkSpd: 78, SwimRunSpd: 50, SwimWalkSpd: 50,
			},
			speed: 120 * dex21, base: 120,
			pAtkSpd: 303, basePAtkSpd: 300, // 1.111
			radius: 7.5, height: f32(22.8), // 22.799999237060547
		},
		{
			name:  "Human Mystic level 1, starting gear",
			tmpl:  humanMystic(),
			level: 1,
			gear:  mysticGear,
			want: PlayerView{
				STR: 22, DEX: 21, CON: 27, INT: 41, WIT: 20, MEN: 39,
				MaxHP:       98,
				MaxMP:       102, // (40 + 19 + 10) × 1.48 = 102.12
				MaxCP:       49,
				WeightLimit: 67620, InventoryLimit: 80,
				PAtk:     2,   // wand sets 5: 5 × 0.63 × 0.9 = 2.84
				PAtkSpd:  383, // round(379 × 1.01) = round(382.79)
				PDef:     52,  // (54 + 17 + 10 - 15 - 8) × 0.9 = 58 × 0.9 = 52.2
				Evasion:  28,
				Accuracy: 33, // round(4.75 + 27.50 + 1) = round(33.25)
				Crit:     40, // wand sets 4
				MAtk:     8,  // wand sets 7: 7 × 1.21² × 0.9² = 8.30
				MAtkSpd:  333,
				MDef:     54,
				RunSpd:   120, WalkSpd: 78, SwimRunSpd: 50, SwimWalkSpd: 50,
			},
			speed: 120 * dex21, base: 120,
			pAtkSpd: 383, basePAtkSpd: 300,
			radius: 7.5, height: f32(22.8),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPlayer(t, r, tt.tmpl, tt.level, tt.gear...)
			want := tt.want
			want.MoveSpeed = tt.speed
			want.MoveMultiplier = moveMul(tt.speed, tt.base)
			want.AttackSpeedMultiplier = atkMul(tt.pAtkSpd, tt.basePAtkSpd)
			want.CollisionRadius, want.CollisionHeight = tt.radius, tt.height
			if got := p.View(); !got.Equal(want) {
				t.Errorf("View() =\n%+v\nwant\n%+v", got, want)
			}
		})
	}
}

func TestRunSpeedCap(t *testing.T) {
	haste := NewModifier(RunSpd, OpMul, 2, Owner{Kind: OwnerEffect, ID: 1})
	lowCap := DefaultConfig()
	lowCap.MaxRunSpeed = 120
	tests := []struct {
		name            string
		cfg             Config
		gm              bool
		mods            []Modifier
		run             float64
		runSpd, walkSpd int32
	}{
		{
			// 115 × 1.1 × 2 = 253 is capped at 250. The packet sends
			// 250 / (250/115) = 115 and the uncapped walk speed
			// 176 / (250/115) = 80.96.
			name: "a player is capped at MaxRunSpeed",
			cfg:  DefaultConfig(), mods: []Modifier{haste},
			run: 250, runSpd: 115, walkSpd: 81,
		},
		{
			name: "a GM is not capped",
			cfg:  DefaultConfig(), gm: true, mods: []Modifier{haste},
			run: 115 * dex30 * 2, runSpd: 115, walkSpd: 80,
		},
		{
			// 126.5 capped at 120; walk 88 / (120/115) = 84.33.
			name: "the cap comes from the config",
			cfg:  lowCap,
			run:  120, runSpd: 115, walkSpd: 84,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPlayer(t, testRules(t, tt.cfg), humanFighter(), 1)
			p.SetGM(tt.gm)
			if err := p.AddModifiers(tt.mods...); err != nil {
				t.Fatal(err)
			}
			if got := p.RunSpeed(); got != tt.run {
				t.Errorf("RunSpeed() = %v, want %v", got, tt.run)
			}
			v := p.View()
			if v.RunSpd != tt.runSpd || v.WalkSpd != tt.walkSpd {
				t.Errorf("View() run/walk = %d/%d, want %d/%d", v.RunSpd, v.WalkSpd, tt.runSpd, tt.walkSpd)
			}
			if v.MoveSpeed != tt.run || v.MoveMultiplier != moveMul(tt.run, 115) {
				t.Errorf("View() move speed %v multiplier %v, want %v and %v", v.MoveSpeed, v.MoveMultiplier, tt.run, moveMul(tt.run, 115))
			}
		})
	}
}

func TestGMSpeed(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	p.SetGMSpeed(2)
	// The multiplier applies to the base speed: 230 × 1.1 = 253, capped
	// for a player who is not a GM.
	if got := p.RunSpeed(); got != 250 {
		t.Errorf("RunSpeed() with //gmspeed 2 = %v, want 250", got)
	}
	p.SetGM(true)
	if got, want := p.RunSpeed(), 230*dex30; got != want {
		t.Errorf("GM RunSpeed() with //gmspeed 2 = %v, want %v", got, want)
	}
	p.SetGMSpeed(0) // back to 1, like AdminGmSpeed
	if got, want := p.RunSpeed(), 115*dex30; got != want {
		t.Errorf("RunSpeed() after //gmspeed 0 = %v, want %v", got, want)
	}
}

// TestSwimming checks speeds in water. Mobius starts the run and walk speed
// from the swim speed, which already has the DEX bonus, and applies the
// bonus again: 50 × 1.1 × 1.1 = 60.5.
func TestSwimming(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	p.SetMotion(Motion{Running: true, InWater: true})
	swim := 50 * dex30
	if got := p.SwimRunSpeed(); got != swim {
		t.Errorf("SwimRunSpeed() = %v, want %v", got, swim)
	}
	if got := p.MoveSpeed(); got != swim {
		t.Errorf("MoveSpeed() = %v, want the swim speed %v", got, swim)
	}
	if got, want := p.RunSpeed(), swim*dex30; got != want {
		t.Errorf("RunSpeed() in water = %v, want %v", got, want)
	}
	v := p.View()
	// Multiplier 55 / 50 = 1.1: run 60.5 / 1.1, swim 55 / 1.1.
	if v.RunSpd != 55 || v.WalkSpd != 55 || v.SwimRunSpd != 50 || v.SwimWalkSpd != 50 {
		t.Errorf("speeds in water = %d/%d/%d/%d, want 55/55/50/50", v.RunSpd, v.WalkSpd, v.SwimRunSpd, v.SwimWalkSpd)
	}
	if want := moveMul(swim, 50); v.MoveMultiplier != want {
		t.Errorf("MoveMultiplier in water = %v, want %v", v.MoveMultiplier, want)
	}
}

func TestSwampAndFlying(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	p.SetMotion(Motion{Running: true, Swamp: 0.5, Flying: true})
	run := 115 * dex30 * 0.5
	if got := p.RunSpeed(); got != run {
		t.Errorf("RunSpeed() in a swamp = %v, want %v", got, run)
	}
	v := p.View()
	if v.FlyRunSpd != v.RunSpd || v.FlyWalkSpd != v.WalkSpd || v.FlyRunSpd == 0 {
		t.Errorf("fly speeds %d/%d, want the run and walk speeds %d/%d", v.FlyRunSpd, v.FlyWalkSpd, v.RunSpd, v.WalkSpd)
	}
}

// TestBaseStatClamp checks that base stats of 100 and more use the last
// bonus instead of failing like Mobius's array lookup.
func TestBaseStatClamp(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	o := Owner{Kind: OwnerEffect, ID: 1}
	if err := p.AddModifiers(NewModifier(STR, OpAdd, 200, o), NewModifier(CON, OpAdd, 100, o)); err != nil {
		t.Fatal(err)
	}
	if p.STR() != 240 || p.CON() != 143 {
		t.Fatalf("STR, CON = %d, %d; want 240, 143", p.STR(), p.CON())
	}
	// STR bonus[99] = 9.67: 4 × 9.67 × 0.9 = 34.81.
	if got := javaInt(p.PAtk()); got != 34 {
		t.Errorf("P.Atk with STR 240 = %d, want 34", got)
	}
	// CON bonus[99] = 8.24: 80 × 8.24 = 659.2.
	if got := p.MaxHP(); got != 659 {
		t.Errorf("max HP with CON 143 = %d, want 659", got)
	}

	// A base stat never drops below 1 (CreatureStat.calcStat).
	p.RemoveOwner(o)
	if err := p.AddModifiers(NewModifier(STR, OpSub, 100, o)); err != nil {
		t.Fatal(err)
	}
	if got := p.STR(); got != 1 {
		t.Errorf("STR 40 - 100 = %d, want 1", got)
	}
}

func TestStatCaps(t *testing.T) {
	r := testRules(t, DefaultConfig())
	o := Owner{Kind: OwnerEffect, ID: 1}
	p := newTestPlayer(t, r, humanFighter(), 1)
	err := p.AddModifiers(
		NewModifier(PAtkSpd, OpMul, 10, o),  // 330 × 10 = 3300
		NewModifier(MAtkSpd, OpMul, 10, o),  // 213.12 × 10 = 2131.2
		NewModifier(REvas, OpAdd, 300, o),   // 333.86
		NewModifier(CritRate, OpMul, 20, o), // 44 × 20 = 880
	)
	if err != nil {
		t.Fatal(err)
	}
	v := p.View()
	if v.PAtkSpd != 1500 || v.MAtkSpd != 1999 || v.Evasion != 250 || v.Crit != 500 {
		t.Errorf("capped atk/cast speed, evasion, crit = %d, %d, %d, %d; want 1500, 1999, 250, 500", v.PAtkSpd, v.MAtkSpd, v.Evasion, v.Crit)
	}
	// 1.1 × 1500 / 300 = 5.5.
	if v.AttackSpeedMultiplier != 5.5 {
		t.Errorf("attack speed multiplier = %v, want 5.5", v.AttackSpeedMultiplier)
	}
	// GMs have no attack speed cap; the casting speed cap applies to all.
	p.SetGM(true)
	if v := p.View(); v.PAtkSpd != 3300 || v.MAtkSpd != 1999 {
		t.Errorf("GM atk/cast speed = %d, %d; want 3300, 1999", v.PAtkSpd, v.MAtkSpd)
	}
}

func TestInventoryLimit(t *testing.T) {
	r := testRules(t, DefaultConfig())
	dwarf := humanFighter()
	dwarf.Race = RaceDwarf
	p := newTestPlayer(t, r, dwarf, 1)
	if got := p.InventoryLimit(); got != 100 {
		t.Errorf("dwarf inventory = %d, want 100", got)
	}
	if err := p.AddModifiers(NewModifier(InventoryLimit, OpAdd, 12, Owner{Kind: OwnerSkill, ID: 1})); err != nil {
		t.Fatal(err)
	}
	if got := p.InventoryLimit(); got != 112 {
		t.Errorf("dwarf inventory with +12 = %d, want 112", got)
	}
	p.SetGM(true)
	if got := p.InventoryLimit(); got != 262 {
		t.Errorf("GM inventory with +12 = %d, want 262", got)
	}
}

func TestDisableCP(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisableCP = true
	p := newTestPlayer(t, testRules(t, cfg), humanFighter(), 1)
	if got := p.MaxCP(); got != 0 {
		t.Errorf("MaxCP() with DisableCP = %d, want 0", got)
	}
}

func TestEquipment(t *testing.T) {
	r := testRules(t, DefaultConfig())

	t.Run("a new item replaces the one in the slot", func(t *testing.T) {
		p := newTestPlayer(t, r, humanFighter(), 1, fighterGear...)
		if err := p.Equip(SlotRHand, ItemOwner(9), apprenticesWand); err != nil {
			t.Fatal(err)
		}
		// Wand: 5 × 1.2 × 0.9 = 5.4; accuracy 4.75 + 32.86 + 1 = 38.61.
		if got := javaInt(p.PAtk()); got != 5 {
			t.Errorf("P.Atk = %d, want 5", got)
		}
		if got := p.Accuracy(); got != 39 {
			t.Errorf("accuracy = %d, want 39", got)
		}
		// 7 wand stats + shirt + pants, nothing of the sword.
		if got := p.mods.Len(); got != 9 {
			t.Errorf("%d modifiers, want 9", got)
		}
	})

	t.Run("unequip removes the item's modifiers", func(t *testing.T) {
		p := newTestPlayer(t, r, humanFighter(), 1)
		before := p.View()
		for _, g := range fighterGear {
			if err := p.Equip(g.slot, ItemOwner(g.id), g.item); err != nil {
				t.Fatal(err)
			}
		}
		for _, g := range fighterGear {
			p.Unequip(g.slot)
		}
		if after := p.View(); !after.Equal(before) || p.mods.Len() != 0 {
			t.Errorf("after unequipping everything:\n%+v\nwant\n%+v (%d modifiers left)", after, before, p.mods.Len())
		}
	})

	t.Run("an item moved to another slot leaves the first", func(t *testing.T) {
		p := newTestPlayer(t, r, humanFighter(), 1)
		ring := Item{Type: "Armor", BodyPart: "rfinger;lfinger", Stats: []ItemStat{{"mDef", 10}}}
		if err := p.Equip(SlotLFinger, ItemOwner(5), ring); err != nil {
			t.Fatal(err)
		}
		if err := p.Equip(SlotRFinger, ItemOwner(5), ring); err != nil {
			t.Fatal(err)
		}
		if p.Equipped(SlotLFinger) || !p.Equipped(SlotRFinger) || p.mods.Len() != 1 {
			t.Errorf("left %v, right %v, %d modifiers; want false, true, 1", p.Equipped(SlotLFinger), p.Equipped(SlotRFinger), p.mods.Len())
		}
		// (41 + 10 - 5) × 1.28 × 0.9 = 52.99.
		if got := javaInt(p.MDef()); got != 52 {
			t.Errorf("M.Def = %d, want 52", got)
		}
	})

	t.Run("full armor also replaces the legs", func(t *testing.T) {
		p := newTestPlayer(t, r, humanFighter(), 1)
		plate := Item{Type: "Armor", BodyPart: "fullarmor", ArmorType: "HEAVY", Stats: []ItemStat{{"pDef", 100}}}
		if err := p.Equip(SlotChest, ItemOwner(5), plate); err != nil {
			t.Fatal(err)
		}
		// (80 + 100 - 31 - 18) × 0.9 = 117.9.
		if got := javaInt(p.PDef()); got != 117 {
			t.Errorf("P.Def with full armor = %d, want 117", got)
		}
		plate.BodyPart = "chest"
		if err := p.Equip(SlotChest, ItemOwner(5), plate); err != nil {
			t.Fatal(err)
		}
		// (80 + 100 - 31) × 0.9 = 134.1.
		if got := javaInt(p.PDef()); got != 134 {
			t.Errorf("P.Def with a chest piece = %d, want 134", got)
		}
	})

	t.Run("a weapon decides magic crit and reach", func(t *testing.T) {
		p := newTestPlayer(t, r, humanFighter(), 1)
		if got := p.Calc(MCritRate, 1); got != 1 {
			t.Errorf("magic crit without weapon = %v, want 1 (no WIT bonus)", got)
		}
		if got := p.PhysicalAttackRange(); got != 0 {
			t.Errorf("reach without weapon = %d, want 0: the fists are the active weapon", got)
		}
		if err := p.Equip(SlotRHand, ItemOwner(1), squiresSword); err != nil {
			t.Fatal(err)
		}
		if got := p.Calc(MCritRate, 1); got != 0.64 {
			t.Errorf("magic crit with weapon = %v, want 0.64 (WIT 11)", got)
		}
		if got := p.PhysicalAttackRange(); got != 40 {
			t.Errorf("reach with the sword = %d, want 40", got)
		}
	})

	t.Run("bad input", func(t *testing.T) {
		p := newTestPlayer(t, r, humanFighter(), 1)
		if err := p.Equip(NumSlots, ItemOwner(1), squiresSword); err == nil {
			t.Error("Equip into slot NumSlots succeeded")
		}
		if err := p.Equip(SlotRHand, Owner{}, squiresSword); err == nil {
			t.Error("Equip without owner succeeded")
		}
		if err := p.Equip(SlotRHand, ItemOwner(1), Item{Type: "Weapon", Stats: []ItemStat{{"bogus", 1}}}); err == nil {
			t.Error("Equip of an item with an unknown stat succeeded")
		}
		if p.Equipped(SlotRHand) || p.mods.Len() != 0 {
			t.Error("a failed Equip changed the player")
		}
	})
}

func TestPlayerLevelAndTemplate(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 0)
	if p.Level() != 1 {
		t.Errorf("level 0 became %d, want 1", p.Level())
	}
	p.SetLevel(99)
	if p.Level() != 20 {
		t.Errorf("level 99 became %d, want the template's highest, 20", p.Level())
	}
	if err := p.SetTemplate(humanMystic()); err != nil {
		t.Fatal(err)
	}
	if p.Level() != 1 || p.STR() != 22 {
		t.Errorf("after changing to a 1-level template: level %d, STR %d; want 1, 22", p.Level(), p.STR())
	}
	if err := p.SetTemplate(nil); err == nil {
		t.Error("SetTemplate(nil) succeeded")
	}
	p.SetFemale(true)
	if p.CollisionRadius() != 6.5 || p.CollisionHeight() != 22.5 {
		t.Errorf("female collision = %v/%v, want 6.5/22.5", p.CollisionRadius(), p.CollisionHeight())
	}
}

func TestPlayerVersionAndEqual(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	before, ver := p.View(), p.Version()
	if !before.Equal(p.View()) {
		t.Fatal("two views of the same state differ")
	}

	// Setting what is already set changes nothing.
	p.SetMotion(Motion{Running: true})
	p.SetLevel(1)
	p.SetGMSpeed(1)
	p.SetHenna(BaseStats{})
	if p.Version() != ver {
		t.Errorf("version changed from %d to %d without a change", ver, p.Version())
	}

	p.SetHenna(BaseStats{STR: 5, CON: -1})
	if p.Version() == ver {
		t.Error("SetHenna did not change the version")
	}
	if p.STR() != 45 || p.CON() != 42 {
		t.Errorf("STR, CON with henna = %d, %d; want 45, 42", p.STR(), p.CON())
	}
	p.SetHenna(BaseStats{})
	if !p.View().Equal(before) {
		t.Error("view differs after removing the henna")
	}

	p.SetMotion(Motion{}) // walking
	walking := p.View()
	if walking.Equal(before) {
		t.Error("walking and running views are equal")
	}
	// Walking: 80 × 1.1 at multiplier 88 / 80.
	if want := 80 * dex30; walking.MoveSpeed != want || walking.MoveMultiplier != moveMul(want, 80) {
		t.Errorf("walking speed %v multiplier %v, want %v and %v", walking.MoveSpeed, walking.MoveMultiplier, want, moveMul(want, 80))
	}
	if walking.RunSpd != 115 || walking.WalkSpd != 80 {
		t.Errorf("walking run/walk = %d/%d, want 115/80", walking.RunSpd, walking.WalkSpd)
	}
}

func TestViewEqual(t *testing.T) {
	a := PlayerView{MaxHP: 1, MoveMultiplier: math.NaN()}
	if !a.Equal(a) {
		t.Error("a view with a NaN is not equal to itself")
	}
	b := a
	b.MaxHP = 2
	c := a
	c.MoveMultiplier = 1
	if a.Equal(b) || a.Equal(c) {
		t.Error("views with different numbers are equal")
	}

	n := NpcView{MaxHP: 1, AttackSpeedMultiplier: 1.5}
	m := n
	m.CollisionRadius = 10
	if !n.Equal(n) || n.Equal(m) {
		t.Error("NpcView.Equal is wrong")
	}
}
