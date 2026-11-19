package stats

import (
	"math"
	"testing"
)

// TestPlayerRegen follows Formulas.calcHpRegen, calcMpRegen and
// calcCpRegen for level 1 characters (regen per level from the class XML:
// HP 2.0, MP 0.9, CP 2.0; level modifier 0.9):
//
//   - Human Fighter, CON 43 (1.58), MEN 25 (1.28): HP 2.0 × (0.9 × 1.58)
//     = 2.844, MP 0.9 × (0.9 × 1.28) = 1.0368, CP (2.0 × 0.9) × 1.58 = 2.844.
//   - Human Mystic, CON 27 (0.98), MEN 39 (1.48): HP 2.0 × (0.9 × 0.98)
//     = 1.764, MP 0.9 × (0.9 × 1.48) = 1.1988, CP 1.764.
//
// Each is then multiplied by 1.1 standing, 1.5 sitting, 1 walking, 0.7
// running. want is the product in Mobius's order; approx shows it.
func TestPlayerRegen(t *testing.T) {
	r := testRules(t, DefaultConfig())
	c := testCombat(t, DefaultCombatConfig())
	lm := 0.9
	type base struct{ hp, mp, cp float64 }
	fighter := base{
		hp: mul(2.0, mul(lm, mobiusBonus.CON[43])),
		mp: mul(0.9, mul(lm, mobiusBonus.MEN[25])),
		cp: mul(mul(2.0, lm), mobiusBonus.CON[43]),
	}
	mystic := base{
		hp: mul(2.0, mul(lm, mobiusBonus.CON[27])),
		mp: mul(0.9, mul(lm, mobiusBonus.MEN[39])),
		cp: mul(mul(2.0, lm), mobiusBonus.CON[27]),
	}
	tests := []struct {
		name   string
		tmpl   *ClassTemplate
		b      base
		act    Activity
		f      float64
		approx Regen
	}{
		{"fighter standing", humanFighter(), fighter, Standing, 1.1, Regen{3.1284, 1.14048, 3.1284}},
		{"fighter sitting", humanFighter(), fighter, Sitting, 1.5, Regen{4.266, 1.5552, 4.266}},
		{"fighter walking", humanFighter(), fighter, Walking, 1, Regen{2.844, 1.0368, 2.844}},
		{"fighter running", humanFighter(), fighter, Running, 0.7, Regen{1.9908, 0.72576, 1.9908}},
		{"mystic standing", humanMystic(), mystic, Standing, 1.1, Regen{1.9404, 1.31868, 1.9404}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPlayer(t, r, tt.tmpl, 1)
			got := c.PlayerRegen(p, tt.act, RegenZone{})
			want := Regen{HP: mul(tt.b.hp, tt.f), MP: mul(tt.b.mp, tt.f), CP: mul(tt.b.cp, tt.f)}
			if got != want {
				t.Errorf("PlayerRegen = %+v, want %+v", got, want)
			}
			if !near(got.HP, tt.approx.HP) || !near(got.MP, tt.approx.MP) || !near(got.CP, tt.approx.CP) {
				t.Errorf("PlayerRegen = %+v, want about %+v", got, tt.approx)
			}
		})
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestPlayerRegenModifiers(t *testing.T) {
	r := testRules(t, DefaultConfig())
	hp := mul(2.0, mul(0.9, mobiusBonus.CON[43])) // 2.844
	mp := mul(0.9, mul(0.9, mobiusBonus.MEN[25])) // 1.0368

	t.Run("at least 1 before the modifiers", func(t *testing.T) {
		weak := humanFighter()
		weak.Levels[0].HPRegen = 0.5 // 0.5 × 1.422 = 0.711
		p := newTestPlayer(t, r, weak, 1)
		if got := testCombat(t, DefaultCombatConfig()).PlayerRegen(p, Standing, RegenZone{}).HP; got != 1.1 {
			t.Errorf("HP regen = %v, want max(1, 0.711) × 1.1 = 1.1", got)
		}
	})

	t.Run("config multiplier", func(t *testing.T) {
		cfg := DefaultCombatConfig()
		cfg.HPRegenMultiplier = 0.5 // HpRegenMultiplier = 50
		p := newTestPlayer(t, r, humanFighter(), 1)
		got := testCombat(t, cfg).PlayerRegen(p, Standing, RegenZone{})
		if want := mul(hp, mul(0.5, 1.1)); got.HP != want { // 2.844 × 0.55 = 1.5642
			t.Errorf("HP regen = %v, want %v", got.HP, want)
		}
	})

	t.Run("zone multipliers and bonuses", func(t *testing.T) {
		p := newTestPlayer(t, r, humanFighter(), 1)
		zone := RegenZone{HPMul: []float64{1.5}, MPMul: []float64{2}, HPBonus: 3, MPBonus: 1}
		got := testCombat(t, DefaultCombatConfig()).PlayerRegen(p, Standing, zone)
		// 2.844 × (1 × 1.5 × 1.1) + 3 = 7.6926; 1.0368 × (1 × 2 × 1.1) + 1
		// = 3.28096. CP has no zone effects: 3.1284.
		wantHP := mul(hp, mul(mul(1, 1.5), 1.1)) + 3
		wantMP := mul(mp, mul(mul(1, 2), 1.1)) + 1
		if got.HP != wantHP || got.MP != wantMP || !near(got.CP, 3.1284) {
			t.Errorf("PlayerRegen in a zone = %+v, want HP %v, MP %v, CP 3.1284", got, wantHP, wantMP)
		}
	})

	t.Run("regHp modifiers", func(t *testing.T) {
		p := newTestPlayer(t, r, humanFighter(), 1)
		if err := p.AddModifiers(NewModifier(RegHP, OpMul, 2, Owner{Kind: OwnerEffect, ID: 1})); err != nil {
			t.Fatal(err)
		}
		got := testCombat(t, DefaultCombatConfig()).PlayerRegen(p, Standing, RegenZone{})
		if want := mul(mul(hp, 2), 1.1); got.HP != want { // 5.688 × 1.1 = 6.2568
			t.Errorf("HP regen with regHp ×2 = %v, want %v", got.HP, want)
		}
	})
}

// TestNpcRegen: the Gremlin's vitals have hpRegen 2 and mpRegen 0.9. NPCs get
// max(1, regen) through the player multipliers (raids: the raid ones), with
// no level modifier, stat bonus or activity.
func TestNpcRegen(t *testing.T) {
	r := testRules(t, DefaultConfig())
	n, err := NewNpc(r, gremlin())
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultCombatConfig()
	cfg.RaidHPRegenMultiplier, cfg.RaidMPRegenMultiplier = 0.5, 3
	c := testCombat(t, cfg)
	if got := c.NpcRegen(n, 2, 0.9, false); got != (Regen{HP: 2, MP: 1}) {
		t.Errorf("NpcRegen = %+v, want HP 2, MP max(1, 0.9) = 1", got)
	}
	if got := c.NpcRegen(n, 2, 0.9, true); got != (Regen{HP: 1, MP: 3}) {
		t.Errorf("raid NpcRegen = %+v, want HP 2 × 0.5, MP 1 × 3", got)
	}
	cfg.HPRegenMultiplier = 2
	if got := testCombat(t, cfg).NpcRegen(n, 2, 0.9, false); got.HP != 4 {
		t.Errorf("NpcRegen with HpRegenMultiplier 200 = %+v, want HP 4", got)
	}
	// Mobius keeps the template values in floats: 7.7f = 7.69999980926513671875.
	if got := c.NpcRegen(n, 7.7, 0, false); got.HP != 7.69999980926513671875 {
		t.Errorf("NpcRegen of 7.7 = %v, want 7.69999980926513671875", got.HP)
	}
}

func TestMaxRecoverable(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	if p.MaxRecoverableHP() != 126 || p.MaxRecoverableMP() != 38 || p.MaxRecoverableCP() != 50 {
		t.Errorf("max recoverable HP/MP/CP = %d/%d/%d, want the maximums 126/38/50",
			p.MaxRecoverableHP(), p.MaxRecoverableMP(), p.MaxRecoverableCP())
	}
	if err := p.AddModifiers(NewModifier(MaxRecoverableHP, OpMul, 0.5, Owner{Kind: OwnerEffect, ID: 1})); err != nil {
		t.Fatal(err)
	}
	if got := p.MaxRecoverableHP(); got != 63 {
		t.Errorf("max recoverable HP with ×0.5 = %d, want 63", got)
	}
	n, err := NewNpc(r, gremlin())
	if err != nil {
		t.Fatal(err)
	}
	if n.MaxRecoverableHP() != 62 || n.MaxRecoverableMP() != 44 {
		t.Errorf("gremlin max recoverable HP/MP = %d/%d, want 62/44", n.MaxRecoverableHP(), n.MaxRecoverableMP())
	}
}
