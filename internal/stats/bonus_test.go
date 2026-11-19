package stats

import (
	"errors"
	"testing"
)

func TestBonus(t *testing.T) {
	b := &mobiusBonus
	tests := []struct {
		name  string
		stat  Stat
		value int
		want  float64
	}{
		{"STR 40", STR, 40, 1.2},
		{"CON 43", CON, 43, 1.58},
		{"DEX 30", DEX, 30, 1.1},
		{"INT 21", INT, 21, 0.81},
		{"WIT 11", WIT, 11, 0.64},
		{"MEN 25", MEN, 25, 1.28},
		{"value 0", STR, 0, 0.29},
		{"value 99", STR, 99, 9.67},
		// Mobius fails on 100 and up (array index); the table end is used.
		{"value 100 uses 99", STR, 100, 9.67},
		{"value 1000 uses 99", WIT, 1000, 47.2},
		{"negative values use 0", CON, -3, 0.45},
		{"not a base stat", RunSpd, 30, 1}, // BaseStat.NONE
	}
	for _, tt := range tests {
		if got := b.Bonus(tt.stat, tt.value); got != tt.want {
			t.Errorf("%s: Bonus(%s, %d) = %v, want %v", tt.name, tt.stat, tt.value, got, tt.want)
		}
	}
}

func TestNewRules(t *testing.T) {
	if _, err := NewRules(DefaultConfig(), mobiusBonus); err != nil {
		t.Fatalf("NewRules with the Mobius data: %v", err)
	}

	missing := mobiusBonus
	missing.MEN[57] = 0
	if _, err := NewRules(DefaultConfig(), missing); !errors.Is(err, ErrInvalid) {
		t.Errorf("NewRules with a zero bonus: error = %v, want ErrInvalid", err)
	}

	cfg := DefaultConfig()
	cfg.MaxRunSpeed = 0
	if _, err := NewRules(cfg, mobiusBonus); !errors.Is(err, ErrInvalid) {
		t.Errorf("NewRules with MaxRunSpeed 0: error = %v, want ErrInvalid", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	// Player.ini and NPC.ini defaults.
	if c.MaxRunSpeed != 250 || c.RunSpeedBoost != 0 || c.MaxPAtkSpeed != 1500 || c.MaxMAtkSpeed != 1999 ||
		c.MaxPCritRate != 500 || c.MaxEvasion != 250 || c.WeightLimitMultiplier != 1 ||
		c.InventoryNoDwarf != 80 || c.InventoryDwarf != 100 || c.InventoryGM != 250 ||
		c.DisableCP || !c.NpcSpeedsWithoutStatBonus {
		t.Errorf("DefaultConfig() = %+v", c)
	}
}

func TestClassTemplate(t *testing.T) {
	hf := humanFighter()
	// basePDef = 31 + 18 + 12 + 7 + 8 + 3 + 1.
	if got := hf.basePDef(); got != 80 {
		t.Errorf("Human Fighter base P.Def = %d, want 80", got)
	}
	// Mobius counts rfinger twice and ignores lfinger: 9 + 9 + 5 + 5 + 13.
	odd := humanFighter()
	odd.MDef.LFinger = 9
	if got := odd.baseMDef(); got != 41 {
		t.Errorf("base M.Def with lfinger 9 = %d, want 41", got)
	}
	if hf.pAtkSpd() != 300 || hf.mAtkSpd() != 333 {
		t.Errorf("attack speeds = %d, %d; want 300 and the default 333", hf.pAtkSpd(), hf.mAtkSpd())
	}

	noSwim := humanFighter()
	noSwim.SwimRunSpeed, noSwim.SwimWalkSpeed = 0, 0
	if noSwim.swimRunSpeed() != 115 || noSwim.swimWalkSpeed() != 80 {
		t.Errorf("swim speeds without data = %v, %v; want run and walk", noSwim.swimRunSpeed(), noSwim.swimWalkSpeed())
	}

	for name, tmpl := range map[string]*ClassTemplate{
		"nil":       nil,
		"no levels": {RunSpeed: 1, WalkSpeed: 1},
		"no speed":  {Levels: []LevelGain{{HP: 1}}},
	} {
		if err := tmpl.Validate(); !errors.Is(err, ErrInvalid) {
			t.Errorf("Validate(%s) = %v, want ErrInvalid", name, err)
		}
	}
	if _, err := NewPlayer(nil, hf, 1); !errors.Is(err, ErrInvalid) {
		t.Errorf("NewPlayer without rules: error = %v, want ErrInvalid", err)
	}
}

func TestNpcTemplateDefaults(t *testing.T) {
	n := gremlin()
	if n.swimRunSpeed() != 50 || n.swimWalkSpeed() != 20 {
		t.Errorf("Gremlin swim speeds = %v, %v; want the ground speeds 50 and 20", n.swimRunSpeed(), n.swimWalkSpeed())
	}
	if n.mAtkSpd() != 333 {
		t.Errorf("Gremlin casting speed = %d, want the default 333", n.mAtkSpd())
	}
	still := gremlin()
	still.RunSpeed, still.WalkSpeed = 0, -1
	// NpcData turns ground speeds of 0 or less into 0.1.
	if still.runSpeed() != 0.1 || still.walkSpeed() != 0.1 {
		t.Errorf("speeds of 0 and -1 = %v, %v; want 0.1", still.runSpeed(), still.walkSpeed())
	}
}
