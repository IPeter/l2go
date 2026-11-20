package stats

import (
	"errors"
	"slices"
	"testing"
)

// tenTimes is a creature with a single formula: P.Atk × 10.
type tenTimes struct{}

func (tenTimes) hasFormula(s Stat) bool { return s == PAtk }

func (tenTimes) formula(_ Stat, v float64) float64 { return v * 10 }

func TestCalcStatOrder(t *testing.T) {
	o := Owner{Kind: OwnerEffect, ID: 1}
	never := CondFunc(func(Context) bool { return false })
	tests := []struct {
		name string
		stat Stat
		init float64
		mods []Modifier
		want float64
	}{
		{
			// Added as Add, Mul, Set; run as Set (0), Mul (20), Add (30):
			// 100 × 2 + 10 = 210.
			name: "set, then mul, then add",
			stat: SDef,
			mods: []Modifier{
				NewModifier(SDef, OpAdd, 10, o),
				NewModifier(SDef, OpMul, 2, o),
				NewModifier(SDef, OpSet, 100, o),
			},
			want: 210,
		},
		{
			name: "equal orders keep the insertion order: set, then add",
			stat: SDef,
			mods: []Modifier{
				{Stat: SDef, Op: OpSet, Order: 0, Value: 10},
				{Stat: SDef, Op: OpAdd, Order: 0, Value: 3},
			},
			want: 13, // 10 + 3
		},
		{
			name: "equal orders keep the insertion order: add, then set",
			stat: SDef,
			mods: []Modifier{
				{Stat: SDef, Op: OpAdd, Order: 0, Value: 3},
				{Stat: SDef, Op: OpSet, Order: 0, Value: 10},
			},
			want: 10, // the set replaces 0 + 3
		},
		{
			name: "the last set wins",
			stat: SDef,
			mods: []Modifier{NewModifier(SDef, OpSet, 5, o), NewModifier(SDef, OpSet, 7, o)},
			want: 7,
		},
		{
			// Order 0 runs before the formula; order 1 added later runs
			// after it, like a Mobius calculator where the formula was
			// added first: ((1 + 1) × 10 + 1) × 2 = 42.
			name: "formulas run after order 0 and before later order 1",
			stat: PAtk,
			init: 1,
			mods: []Modifier{
				{Stat: PAtk, Op: OpAdd, Order: 1, Value: 1},
				{Stat: PAtk, Op: OpMul, Order: 20, Value: 2},
				{Stat: PAtk, Op: OpAdd, Order: 0, Value: 1},
			},
			want: 42,
		},
		{
			name: "div before sub",
			stat: SDef,
			init: 100,
			mods: []Modifier{NewModifier(SDef, OpSub, 10, o), NewModifier(SDef, OpDiv, 4, o)},
			want: 15, // 100 / 4 - 10
		},
		{
			name: "a zero divisor is skipped",
			stat: SDef,
			init: 8,
			mods: []Modifier{NewModifier(SDef, OpDiv, 0, o)},
			want: 8,
		},
		{
			name: "share does nothing for players and NPCs",
			stat: SDef,
			init: 8,
			mods: []Modifier{NewModifier(SDef, OpShare, 0.5, o)},
			want: 8,
		},
		{
			name: "enchant bonuses add",
			stat: SDef,
			init: 8,
			mods: []Modifier{NewModifier(SDef, OpEnchant, 3, o), NewModifier(SDef, OpEnchantHP, 4, o)},
			want: 15,
		},
		{
			name: "a modifier whose condition fails is skipped",
			stat: SDef,
			mods: []Modifier{{Stat: SDef, Op: OpSet, Value: 5, Cond: never}, {Stat: SDef, Op: OpAdd, Value: 1, Cond: Always}},
			want: 1,
		},
		{
			name: "kept-positive stats become 1 at 0 or below",
			stat: PAtk,
			init: 1,
			mods: []Modifier{NewModifier(PAtk, OpSub, 50, o)},
			want: 1, // 1 × 10 - 50 = -40
		},
		{
			name: "a formula alone also keeps the stat positive",
			stat: PAtk,
			init: 0,
			want: 1, // 0 × 10
		},
		{
			name: "other stats may go negative",
			stat: REvas,
			mods: []Modifier{NewModifier(REvas, OpSub, 5, o)},
			want: -5,
		},
		{
			name: "without formula or modifier the base value stays, even 0",
			stat: SDef,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ms Modifiers
			if err := ms.Add(tt.mods...); err != nil {
				t.Fatalf("Add: %v", err)
			}
			if got := calcStat(tenTimes{}, ms.Of(tt.stat), tt.stat, tt.init, Context{}); got != tt.want {
				t.Errorf("calcStat(%s, %v) = %v, want %v", tt.stat, tt.init, got, tt.want)
			}
		})
	}
}

func TestCondSeesContext(t *testing.T) {
	type target struct{ name string }
	onlyGremlin := CondFunc(func(ctx Context) bool {
		tg, ok := ctx.Target.(target)
		return ok && tg.name == "Gremlin"
	})
	// REvas, not a keepsPositive stat: those would turn the unmet result 0 into 1.
	var ms Modifiers
	if err := ms.Add(Modifier{Stat: REvas, Op: OpAdd, Value: 7, Cond: onlyGremlin}); err != nil {
		t.Fatal(err)
	}
	if got := calcStat(tenTimes{}, ms.Of(REvas), REvas, 0, Context{Target: target{"Gremlin"}}); got != 7 {
		t.Errorf("with the Gremlin as target = %v, want 7", got)
	}
	if got := calcStat(tenTimes{}, ms.Of(REvas), REvas, 0, Context{}); got != 0 {
		t.Errorf("without target = %v, want 0", got)
	}
}

func TestModifiersRemoveOwner(t *testing.T) {
	item := ItemOwner(7)
	buff := Owner{Kind: OwnerEffect, ID: 7} // same ID, other kind
	var ms Modifiers
	err := ms.Add(
		Modifier{Stat: PDef, Op: OpAdd, Order: 0, Value: 33, Owner: item},
		NewModifier(PDef, OpMul, 1.1, buff),
		Modifier{Stat: PDef, Op: OpAdd, Order: 0, Value: 1, Owner: buff},
		Modifier{Stat: MaxMP, Op: OpAdd, Order: 0, Value: 19, Owner: item},
	)
	if err != nil {
		t.Fatal(err)
	}
	if ms.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", ms.Len())
	}
	if n := ms.RemoveOwner(item); n != 2 {
		t.Errorf("RemoveOwner(item) = %d, want 2", n)
	}
	if ms.Len() != 2 || len(ms.Of(MaxMP)) != 0 {
		t.Errorf("after removal: Len() = %d, maxMp modifiers %v", ms.Len(), ms.Of(MaxMP))
	}
	want := []Modifier{
		{Stat: PDef, Op: OpAdd, Order: 0, Value: 1, Owner: buff},
		{Stat: PDef, Op: OpMul, Order: OrderMul, Value: 1.1, Owner: buff},
	}
	if got := ms.Of(PDef); !slices.Equal(got, want) {
		t.Errorf("pDef modifiers = %+v, want %+v", got, want)
	}
	if n := ms.RemoveOwner(item); n != 0 {
		t.Errorf("second RemoveOwner(item) = %d, want 0", n)
	}
}

func TestModifiersAddRejectsInvalid(t *testing.T) {
	bad := []Modifier{
		{Stat: Stat(NumStats), Op: OpAdd},
		{Stat: PAtk},
		{Stat: PAtk, Op: OpEnchantHP + 1},
		{Stat: PAtk, Op: OpAdd, Order: -1},
	}
	for _, m := range bad {
		var ms Modifiers
		err := ms.Add(NewModifier(PAtk, OpAdd, 1, Owner{}), m)
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("Add(%+v) error = %v, want ErrInvalid", m, err)
		}
		if ms.Len() != 0 {
			t.Errorf("Add(%+v) added %d modifiers, want none", m, ms.Len())
		}
	}
}

func TestOps(t *testing.T) {
	orders := map[Op]int{
		OpSet: 0, OpEnchant: 0,
		OpMul: 20, OpDiv: 20,
		OpAdd: 30, OpSub: 30, OpShare: 30,
		OpEnchantHP: 40,
	}
	for op, want := range orders {
		if got := op.DefaultOrder(); got != want {
			t.Errorf("%s.DefaultOrder() = %d, want %d", op, got, want)
		}
		if got, ok := ParseOp(op.String()); !ok || got != op {
			t.Errorf("ParseOp(%q) = %v, %v", op.String(), got, ok)
		}
	}
	for name, want := range map[string]Op{"set": OpSet, "ENCHANTHP": OpEnchantHP, "Mul": OpMul} {
		if got, ok := ParseOp(name); !ok || got != want {
			t.Errorf("ParseOp(%q) = %v, %v; want %v", name, got, ok, want)
		}
	}
	if _, ok := ParseOp("pow"); ok {
		t.Error(`ParseOp("pow") succeeded`)
	}
	if Op(0).DefaultOrder() != -1 || Op(0).Valid() {
		t.Error("Op(0) is valid")
	}
}
