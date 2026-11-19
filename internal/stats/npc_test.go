package stats

import (
	"errors"
	"testing"
)

// TestGremlin checks NPC 20001 against the Mobius formulas. The NPC XML
// holds bases divided by the stat bonus; the standard NPC calculator
// multiplies them back. Bonuses: STR 40 1.2, DEX 30 1.1, CON 43 1.58, INT 21
// 0.81, WIT 20 1, MEN 10 1.11; level 1 modifier 0.9.
func TestGremlin(t *testing.T) {
	r := testRules(t, DefaultConfig())
	n, err := NewNpc(r, gremlin())
	if err != nil {
		t.Fatal(err)
	}
	ints := []struct {
		name      string
		got, want int
	}{
		{"max HP: (float)39.74519 × 1.58 = 62.80", n.MaxHP(), 62},
		{"max MP: 40 × 1.11 = 44.4", n.MaxMP(), 44},
		{"P.Atk: (int)8.47458 × 1.2 × 0.9 = 8.64", int(javaInt(n.PAtk())), 8},
		{"M.Atk: (int)5.78704 × 0.81² × 0.9² = 2.66", int(javaInt(n.MAtk())), 2},
		{"P.Def: (int)44.44444 × 0.9 = 39.6", int(javaInt(n.PDef())), 39},
		{"M.Def: (int)29.59 × 1.11 × 0.9 = 28.97", int(javaInt(n.MDef())), 28},
		{"accuracy: round(sqrt(30) × 6 + 1)", n.Accuracy(), 34},
		{"evasion: round(sqrt(30) × 6 + 1)", n.Evasion(), 34},
		{"crit: (int)(4 × 1.1 × 10 + 0.5)", n.Crit(), 44},
		{"reach: the template range", n.PhysicalAttackRange(), 40},
	}
	for _, c := range ints {
		if c.got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, c.got, c.want)
		}
	}

	// NpcSpeedsWithoutStatBonus (default on): speeds and casting speed as
	// defined; attack speed round(253 × 1.1) = 278. Walking by default.
	want := NpcView{
		MaxHP:                 62,
		PAtkSpd:               278,
		MAtkSpd:               333,
		RunSpd:                50,
		WalkSpd:               20,
		SwimRunSpd:            50, // no swim speed in the data: the ground speed
		SwimWalkSpd:           20,
		MoveMultiplier:        moveMul(20, 20),
		AttackSpeedMultiplier: atkMul(278, 253),
		CollisionRadius:       10,
		CollisionHeight:       15,
		MoveSpeed:             20,
	}
	if got := n.View(); !got.Equal(want) {
		t.Errorf("View() =\n%+v\nwant\n%+v", got, want)
	}

	n.SetMotion(Motion{Running: true})
	if got := n.View(); got.MoveSpeed != 50 || got.MoveMultiplier != moveMul(50, 50) || got.RunSpd != 50 || got.WalkSpd != 20 {
		t.Errorf("running View() = %+v", got)
	}
}

func TestNpcSpeedsWithStatBonus(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NpcSpeedsWithoutStatBonus = false
	n, err := NewNpc(testRules(t, cfg), gremlin())
	if err != nil {
		t.Fatal(err)
	}
	// With the option off, movement gets the DEX bonus (and casting the
	// WIT bonus, which is 1 at WIT 20).
	walk := 20 * dex30
	if got := n.WalkSpeed(); got != walk {
		t.Errorf("WalkSpeed() = %v, want %v", got, walk)
	}
	if got := n.RunSpeed(); got != 50*dex30 {
		t.Errorf("RunSpeed() = %v, want %v", got, 50*dex30)
	}
	v := n.View()
	if v.MAtkSpd != 333 || v.RunSpd != 50 || v.WalkSpd != 20 || v.MoveSpeed != walk || v.MoveMultiplier != moveMul(walk, 20) {
		t.Errorf("View() = %+v", v)
	}
}

func TestNpcModifiersAndVersion(t *testing.T) {
	r := testRules(t, DefaultConfig())
	n, err := NewNpc(r, gremlin())
	if err != nil {
		t.Fatal(err)
	}
	before, ver := n.View(), n.Version()
	slow := Owner{Kind: OwnerEffect, ID: 3}
	if err := n.AddModifiers(NewModifier(RunSpd, OpMul, 0.5, slow)); err != nil {
		t.Fatal(err)
	}
	if n.Version() == ver {
		t.Error("AddModifiers did not change the version")
	}
	if got := n.WalkSpeed(); got != 10 {
		t.Errorf("slowed WalkSpeed() = %v, want 10", got)
	}
	if n.RemoveOwner(slow) != 1 || !n.View().Equal(before) {
		t.Error("removing the slow did not restore the view")
	}
	n.SetGMSpeed(2)
	if got := n.RunSpeed(); got != 100 {
		t.Errorf("RunSpeed() with //gmspeed 2 = %v, want 100", got)
	}
	if _, err := NewNpc(r, nil); !errors.Is(err, ErrInvalid) {
		t.Errorf("NewNpc(nil) error = %v, want ErrInvalid", err)
	}
	if _, err := NewNpc(nil, gremlin()); !errors.Is(err, ErrInvalid) {
		t.Errorf("NewNpc without rules: error = %v, want ErrInvalid", err)
	}
}
