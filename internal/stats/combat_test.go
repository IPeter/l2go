package stats

import (
	"errors"
	"fmt"
	"math"
	"testing"
)

// mobiusHitBonus is hitConditionBonus.xml of the Mobius Interlude datapack.
var mobiusHitBonus = HitConditionBonus{Front: 0, Side: 5, Back: 10, High: 3, Low: -3, Dark: -10, Rain: -3}

func testCombat(t *testing.T, cfg CombatConfig) *CombatRules {
	t.Helper()
	c, err := NewCombatRules(cfg, mobiusHitBonus)
	if err != nil {
		t.Fatalf("NewCombatRules: %v", err)
	}
	return c
}

// draw is one scripted random draw: Get(n) or Between(lo, hi), returning v.
type draw struct {
	between   bool
	n, lo, hi int
	v         int
}

func get(n, v int) draw          { return draw{n: n, v: v} }
func between(lo, hi, v int) draw { return draw{between: true, lo: lo, hi: hi, v: v} }

func (d draw) String() string {
	if d.between {
		return fmt.Sprintf("Between(%d, %d) = %d", d.lo, d.hi, d.v)
	}
	return fmt.Sprintf("Get(%d) = %d", d.n, d.v)
}

// scriptRand is a Rand that returns scripted draws. It has Mobius Rnd's
// semantics (no draw for Get(n <= 0) or Between(lo >= hi)) and fails the
// test when a draw is not the next scripted one, so it checks Mobius's draw
// order, kinds and bounds.
type scriptRand struct {
	t     *testing.T
	draws []draw
}

func script(t *testing.T, draws ...draw) *scriptRand { return &scriptRand{t: t, draws: draws} }

func (r *scriptRand) next(call string) draw {
	r.t.Helper()
	if len(r.draws) == 0 {
		r.t.Fatalf("unexpected draw %s: the script is used up", call)
	}
	d := r.draws[0]
	r.draws = r.draws[1:]
	return d
}

func (r *scriptRand) Get(n int) int {
	r.t.Helper()
	if n <= 0 {
		return 0
	}
	call := fmt.Sprintf("Get(%d)", n)
	d := r.next(call)
	if d.between || d.n != n || d.v < 0 || d.v >= n {
		r.t.Fatalf("draw %s, but the script has %v", call, d)
	}
	return d.v
}

func (r *scriptRand) Between(lo, hi int) int {
	r.t.Helper()
	if lo >= hi {
		return lo
	}
	call := fmt.Sprintf("Between(%d, %d)", lo, hi)
	d := r.next(call)
	if !d.between || d.lo != lo || d.hi != hi || d.v < lo || d.v > hi {
		r.t.Fatalf("draw %s, but the script has %v", call, d)
	}
	return d.v
}

// done fails the test when scripted draws were left unused.
func (r *scriptRand) done() {
	r.t.Helper()
	if len(r.draws) > 0 {
		r.t.Errorf("%d scripted draws not used: %v", len(r.draws), r.draws)
	}
}

// fake is a Combatant with fixed numbers, so formulas can be checked with
// simple arithmetic. calc replaces CalcFor for the stats it lists; the
// others return their base value, as a stat without modifiers does.
type fake struct {
	level          int
	acc, eva, crit int
	pAtk, pDef     float64
	sDef           int
	pAtkSpd        float64
	bonus          map[Stat]float64 // StatBonus; 1 when missing
	calc           map[Stat]func(init float64, ctx Context) float64
}

func (f *fake) Level() int { return f.level }
func (f *fake) CalcFor(s Stat, init float64, ctx Context) float64 {
	if fn := f.calc[s]; fn != nil {
		return fn(init, ctx)
	}
	return init
}
func (f *fake) Accuracy() int           { return f.acc }
func (f *fake) EvasionFor(Context) int  { return f.eva }
func (f *fake) CritFor(Context) int     { return f.crit }
func (f *fake) PAtkFor(Context) float64 { return f.pAtk }
func (f *fake) PDefFor(Context) float64 { return f.pDef }
func (f *fake) ShieldDef() int          { return f.sDef }
func (f *fake) PAtkSpd() float64        { return f.pAtkSpd }
func (f *fake) StatBonus(s Stat) float64 {
	if b, ok := f.bonus[s]; ok {
		return b
	}
	return 1
}

// set, times and plus are fake stat functions: a modifier that sets the
// value, multiplies it or adds to it.
func set(v float64) func(float64, Context) float64 {
	return func(float64, Context) float64 { return v }
}

func times(v float64) func(float64, Context) float64 {
	return func(init float64, _ Context) float64 { return init * v }
}

func plus(v float64) func(float64, Context) float64 {
	return func(init float64, _ Context) float64 { return init + v }
}

// The test fights happen around the origin. The target stands at (0, 0)
// facing east (heading 0); the attacker stands in front of it (east), behind
// it (west) or at its side (north).
var placements = map[Position][2]int{
	PosFront: {100, 0},
	PosBack:  {-100, 0},
	PosSide:  {0, 100},
}

// strikeOf returns a strike of attacker at target, the attacker placed at
// pos. The fighters are references "attacker" and "target" for conditions.
func strikeOf(attacker, target Fighter, pos Position) *Strike {
	xy := placements[pos]
	attacker.X, attacker.Y = xy[0], xy[1]
	target.X, target.Y, target.Heading = 0, 0, 0
	if attacker.Ref == nil {
		attacker.Ref = "attacker"
	}
	if target.Ref == nil {
		target.Ref = "target"
	}
	return &Strike{Attacker: attacker, Target: target}
}

func TestNewCombatRules(t *testing.T) {
	c, err := NewCombatRules(DefaultCombatConfig(), mobiusHitBonus)
	if err != nil {
		t.Fatal(err)
	}
	if c.Config() != DefaultCombatConfig() || c.HitBonus() != mobiusHitBonus {
		t.Error("the rules do not keep their config and bonuses")
	}
	bad := []func(*CombatConfig){
		func(c *CombatConfig) { c.HPRegenMultiplier = -1 },
		func(c *CombatConfig) { c.RaidMPRegenMultiplier = math.Inf(1) },
		func(c *CombatConfig) { c.RateXP = math.NaN() },
		func(c *CombatConfig) { c.PerfectShieldBlockRate = 101 },
		func(c *CombatConfig) { c.PerfectShieldBlockRate = -1 },
		func(c *CombatConfig) { c.MonsterExpMaxLevelDifference = -1 },
		func(c *CombatConfig) { c.MaxSPBonus = math.NaN() },
	}
	for i, change := range bad {
		cfg := DefaultCombatConfig()
		change(&cfg)
		if _, err := NewCombatRules(cfg, mobiusHitBonus); !errors.Is(err, ErrInvalid) {
			t.Errorf("bad config %d: error = %v, want ErrInvalid", i, err)
		}
	}
}

func TestAttackShapeOf(t *testing.T) {
	tests := []struct {
		weapon string
		player bool
		want   AttackShape
	}{
		{"SWORD", true, ShapeSimple},
		{"BLUNT", false, ShapeSimple},
		{"DAGGER", true, ShapeSimple},
		{"ETC", true, ShapeSimple},
		{"FIST", true, ShapeDual},    // Mobius sends player fists down the dual path
		{"FIST", false, ShapeSimple}, // but not NPC fists
		{"DUAL", false, ShapeDual},
		{"DUALFIST", true, ShapeDual},
		{"DUALDAGGER", true, ShapeDual},
		{"BOW", false, ShapeBow},
		{"POLE", true, ShapePole},
		{"FISHINGROD", true, ShapeSimple},
		{"", false, ShapeSimple},
	}
	for _, tt := range tests {
		if got := AttackShapeOf(tt.weapon, tt.player); got != tt.want {
			t.Errorf("AttackShapeOf(%q, %v) = %v, want %v", tt.weapon, tt.player, got, tt.want)
		}
	}
}

// TestHeadingAndPosition checks LocationUtil.calculateHeadingFrom, which
// truncates degrees × 182.044444444, and Position.getPosition.
func TestHeadingAndPosition(t *testing.T) {
	headings := []struct {
		x, y int
		want int
	}{
		{100, 0, 0},        // 0°
		{0, 100, 16383},    // 90° × 182.044444444 = 16383.99999996
		{-100, 0, 32767},   // 180°: 32767.99999992
		{0, -100, 49151},   // −90° + 360: 49151.99999988
		{100, 100, 8191},   // 45°: 8191.99999998
		{100, -100, 57343}, // 315°: 57343.99999986
	}
	for _, h := range headings {
		if got := HeadingTo(0, 0, h.x, h.y); got != h.want {
			t.Errorf("HeadingTo(0, 0, %d, %d) = %d, want %d", h.x, h.y, got, h.want)
		}
	}

	// The target stands at the origin facing east (heading 0) unless the
	// case says otherwise. h is |target heading − heading to the target|.
	positions := []struct {
		name       string
		x, y, head int
		want       Position
	}{
		{"east of it: h = 32767, they face each other", 100, 0, 0, PosFront},
		{"west of it: h = 0", -100, 0, 0, PosBack},
		{"north of it: h = 49151 = 0xBFFF", 0, 100, 0, PosSide},
		{"south of it: h = 16383 = 0x3FFF", 0, -100, 0, PosSide},
		{"north-east of it: h = 40959 = 0x9FFF", 100, 100, 0, PosFront},
		{"north-west of it: h = 57343 = 0xDFFF", -100, 100, 0, PosSide},
		{"west, target facing west: h = 32768", -100, 0, 32768, PosFront},
		{"east, target facing west: h = |32768 − 32767| = 1", 100, 0, 32768, PosBack},
		{"h = 0xE001 is back again", -100, 0, 0xE001, PosBack},
	}
	for _, p := range positions {
		if got := PositionOf(p.x, p.y, 0, 0, p.head); got != p.want {
			t.Errorf("%s: PositionOf = %v, want %v", p.name, got, p.want)
		}
	}
}

func TestStringers(t *testing.T) {
	if PosBack.String() != "back" || Position(9).String() != "Position(9)" {
		t.Error("Position.String")
	}
	if ShapeDual.String() != "dual" || AttackShape(9).String() != "AttackShape(9)" {
		t.Error("AttackShape.String")
	}
	if ShieldPerfect.String() != "perfect" || ShieldResult(9).String() != "ShieldResult(9)" {
		t.Error("ShieldResult.String")
	}
	if Sitting.String() != "sitting" || Activity(9).String() != "Activity(9)" {
		t.Error("Activity.String")
	}
}

// TestJavaCasts checks the (long) cast and Math.clamp helpers.
func TestJavaCasts(t *testing.T) {
	longs := []struct {
		x    float64
		want int64
	}{
		{38.5, 38},
		{-2.9, -2},
		{math.NaN(), 0},
		{math.Inf(1), math.MaxInt64},
		{math.Inf(-1), math.MinInt64},
		{1e19, math.MaxInt64},
	}
	for _, l := range longs {
		if got := javaLong(l.x); got != l.want {
			t.Errorf("javaLong(%v) = %d, want %d", l.x, got, l.want)
		}
	}
	if javaClamp(0.5, 1, 99) != 1 || javaClamp(120, 1, 99) != 99 || javaClamp(42, 1, 99) != 42 {
		t.Error("javaClamp")
	}
}
