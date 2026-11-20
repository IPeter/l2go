package stats

import (
	"fmt"
	"math"
)

// The combat formulas port Mobius Formulas.java and the auto-attack code of
// Creature.java (doAttack, doAttackHitSimple/ByDual/ByBow, onHitTimer) as
// pure functions. The game layer describes both creatures (Fighter), the
// swing (Strike) and passes a random source; the formulas return numbers and
// change nothing. Every function documents its draws in Mobius's order, so a
// scripted source can check them. The rounding points are Mobius's: each
// (int) cast, Math.round and integer division is kept where Mobius has it.
//
// Class balance, champion, premium and vitality factors are left out: their
// Mobius defaults are 1 (Plan 04 W16). Raid multipliers of P.Atk and P.Def
// come with raids.

// Rand is the random source of the combat formulas. Its methods have the
// semantics of Mobius Rnd, including the cases in which they draw nothing.
type Rand interface {
	// Get returns a number in [0, n), or 0 without drawing when n <= 0
	// (Rnd.get(int)).
	Get(n int) int
	// Between returns a number in [lo, hi], or lo without drawing when
	// lo >= hi (Rnd.get(int, int)).
	Between(lo, hi int) int
}

// CombatConfig holds the Mobius settings of the combat formulas,
// regeneration and rewards. DefaultCombatConfig returns the Mobius defaults
// (Player.ini, NPC.ini and Rates.ini).
type CombatConfig struct {
	// RandomizeAutoAttackDamage multiplies auto-attack damage by a second,
	// independent random damage factor (RandomizeAutoAttackDamage, true).
	// The first factor always applies, so the default rolls twice.
	RandomizeAutoAttackDamage bool
	// ShieldBlocks is AltShieldBlocks (false): a blocked hit loses the
	// shield defence from its damage instead of adding it to P.Def.
	ShieldBlocks bool
	// PerfectShieldBlockRate is AltPerfectShieldBlockRate (10). A block is
	// perfect when Get(100) is above 100 minus it, so 10 means 9 %.
	PerfectShieldBlockRate int
	// DistanceBowDamage scales bow damage with the distance to the target
	// (DistanceBowDamage, false).
	DistanceBowDamage bool
	// CancelCastByHit and CancelBowByHit are AltGameCancelByHit ("cast"):
	// "cast" sets the first, "bow" the second, "all" both. They decide
	// which actions a hit can break (AttackBreak).
	CancelCastByHit, CancelBowByHit bool
	// HPRegenMultiplier, MPRegenMultiplier and CPRegenMultiplier scale the
	// regeneration of players, and the HP and MP ones also that of NPCs
	// that are not raids: Player.ini HpRegenMultiplier, MpRegenMultiplier
	// and CpRegenMultiplier divided by 100 (1).
	HPRegenMultiplier, MPRegenMultiplier, CPRegenMultiplier float64
	// RaidHPRegenMultiplier and RaidMPRegenMultiplier scale the
	// regeneration of raids: NPC.ini RaidHpRegenMultiplier and
	// RaidMpRegenMultiplier divided by 100 (1).
	RaidHPRegenMultiplier, RaidMPRegenMultiplier float64
	// RateXP and RateSP multiply the experience and SP of monsters
	// (Rates.ini RateXp and RateSp, 1). Mobius keeps them in Java floats,
	// so they are used at float32 precision.
	RateXP, RateSP float64
	// MonsterExpMaxLevelDifference is the level difference from which a
	// monster gives no experience and SP (MonsterExpMaxLevelDifference, 11).
	MonsterExpMaxLevelDifference int
	// MaxExpBonus and MaxSPBonus cap the bonus multipliers of the bonusExp
	// and bonusSp stats (MaxExpBonus and MaxSpBonus, 3.5); 0 or less means
	// no cap.
	MaxExpBonus, MaxSPBonus float64
}

// DefaultCombatConfig returns the Mobius defaults.
func DefaultCombatConfig() CombatConfig {
	return CombatConfig{
		RandomizeAutoAttackDamage:    true,
		ShieldBlocks:                 false,
		PerfectShieldBlockRate:       10,
		DistanceBowDamage:            false,
		CancelCastByHit:              true,
		CancelBowByHit:               false,
		HPRegenMultiplier:            1,
		MPRegenMultiplier:            1,
		CPRegenMultiplier:            1,
		RaidHPRegenMultiplier:        1,
		RaidMPRegenMultiplier:        1,
		RateXP:                       1,
		RateSP:                       1,
		MonsterExpMaxLevelDifference: 11,
		MaxExpBonus:                  3.5,
		MaxSPBonus:                   3.5,
	}
}

func (c *CombatConfig) validate() error {
	for _, f := range []struct {
		name string
		v    float64
	}{
		{"HPRegenMultiplier", c.HPRegenMultiplier},
		{"MPRegenMultiplier", c.MPRegenMultiplier},
		{"CPRegenMultiplier", c.CPRegenMultiplier},
		{"RaidHPRegenMultiplier", c.RaidHPRegenMultiplier},
		{"RaidMPRegenMultiplier", c.RaidMPRegenMultiplier},
		{"RateXP", c.RateXP},
		{"RateSP", c.RateSP},
	} {
		if !(f.v >= 0) || math.IsInf(f.v, 0) {
			return fmt.Errorf("stats: %w: combat config %s is %v, want 0 or more", ErrInvalid, f.name, f.v)
		}
	}
	switch {
	case c.PerfectShieldBlockRate < 0 || c.PerfectShieldBlockRate > 100:
		return fmt.Errorf("stats: %w: combat config PerfectShieldBlockRate is %d, want 0 to 100", ErrInvalid, c.PerfectShieldBlockRate)
	case c.MonsterExpMaxLevelDifference < 0:
		return fmt.Errorf("stats: %w: combat config MonsterExpMaxLevelDifference is %d, want 0 or more", ErrInvalid, c.MonsterExpMaxLevelDifference)
	case math.IsNaN(c.MaxExpBonus) || math.IsNaN(c.MaxSPBonus):
		return fmt.Errorf("stats: %w: combat config MaxExpBonus %v, MaxSPBonus %v: want numbers", ErrInvalid, c.MaxExpBonus, c.MaxSPBonus)
	}
	return nil
}

// HitConditionBonus is hitConditionBonus.xml: the accuracy bonus, in
// percent, of attacking from a position or in a condition. It has the fields
// of data.HitConditionBonus, so the loaded data converts directly:
// stats.HitConditionBonus(d.HitConditionBonus). Mobius has no rain yet, so
// Rain is unused.
type HitConditionBonus struct {
	Front int
	Side  int
	Back  int
	High  int
	Low   int
	Dark  int
	Rain  int
}

// CombatRules are what every fight of a server shares: the combat config
// and the hit condition bonuses. They do not change after NewCombatRules.
// The formulas are their methods; the stat sets keep their own Rules (stat
// bonus tables, caps).
type CombatRules struct {
	cfg CombatConfig
	hit HitConditionBonus
}

// NewCombatRules checks the config and returns the combat rules.
func NewCombatRules(cfg CombatConfig, hit HitConditionBonus) (*CombatRules, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &CombatRules{cfg: cfg, hit: hit}, nil
}

// Config returns the config the rules were made with.
func (c *CombatRules) Config() CombatConfig { return c.cfg }

// HitBonus returns the hit condition bonuses the rules were made with.
func (c *CombatRules) HitBonus() HitConditionBonus { return c.hit }

// Combatant is the stat set of a creature in a fight. *Player and *Npc
// implement it. The context of the ...For methods carries the other creature
// for modifier conditions, as the target argument of Mobius calcStat does.
type Combatant interface {
	// Level returns the level.
	Level() int
	// CalcFor runs stat for the base value init (CreatureStat.calcStat).
	CalcFor(stat Stat, init float64, ctx Context) float64
	// Accuracy is CreatureStat.getAccuracy, which has no target.
	Accuracy() int
	// EvasionFor is CreatureStat.getEvasionRate(target).
	EvasionFor(ctx Context) int
	// CritFor is CreatureStat.getCriticalHit(target, null).
	CritFor(ctx Context) int
	// PAtkFor is CreatureStat.getPAtk(target).
	PAtkFor(ctx Context) float64
	// PDefFor is CreatureStat.getPDef(target).
	PDefFor(ctx Context) float64
	// ShieldDef is CreatureStat.getShldDef.
	ShieldDef() int
	// PAtkSpd is the attack speed (getPAtkSpd, capped for players).
	PAtkSpd() float64
	// StatBonus is BaseStat.calcBonus: the statBonus.xml multiplier of base
	// stat s (STR, INT, DEX, WIT, CON or MEN) at the creature's value.
	StatBonus(s Stat) float64
}

var (
	_ Combatant = (*Player)(nil)
	_ Combatant = (*Npc)(nil)
)

// AttackShape is how a creature's auto-attack swings, decided by its attack
// type in Creature.doAttack.
type AttackShape uint8

// The attack shapes.
const (
	// ShapeSimple hits once at half the swing: swords, blunts, daggers,
	// etc, and the fists of NPCs (doAttackHitSimple).
	ShapeSimple AttackShape = iota
	// ShapeDual hits twice for half damage each, at a quarter and at half
	// the swing: dual swords, dual fists, dual daggers and the bare fists
	// of players (doAttackHitByDual).
	ShapeDual
	// ShapeBow hits once at the end of the swing, then waits for the bow
	// reuse (doAttackHitByBow).
	ShapeBow
	// ShapePole hits like ShapeSimple; it adds targets only when the
	// atkCountMax stat is above 1 (doAttackHitByPole).
	ShapePole
)

var shapeNames = [...]string{ShapeSimple: "simple", ShapeDual: "dual", ShapeBow: "bow", ShapePole: "pole"}

// String returns the name of the shape.
func (s AttackShape) String() string {
	if int(s) < len(shapeNames) {
		return shapeNames[s]
	}
	return fmt.Sprintf("AttackShape(%d)", uint8(s))
}

// AttackShapeOf returns the shape of an attack type, a Mobius weapon type
// name such as "SWORD", "BOW" or "DUALFIST" (the active weapon's type, or
// the template's attack type without one). Mobius sends FIST down the dual
// path for players only. Unknown types swing simply, as Mobius's default
// branch does.
func AttackShapeOf(weaponType string, player bool) AttackShape {
	switch weaponType {
	case "BOW":
		return ShapeBow
	case "POLE":
		return ShapePole
	case "DUAL", "DUALFIST", "DUALDAGGER":
		return ShapeDual
	case "FIST":
		if player {
			return ShapeDual
		}
	}
	return ShapeSimple
}

// Fighter is one creature in a fight: its stat set and what the formulas
// read besides its stats. The game layer fills one per side for each swing.
type Fighter struct {
	// Stats is the stat set; it must not be nil.
	Stats Combatant
	// Ref is what modifier conditions see as this creature when the other
	// one calculates a stat against it (Context.Target). It may be nil.
	Ref any
	// Playable is set for players, summons and pets (Mobius isPlayable):
	// PvP rules apply between two of them.
	Playable bool
	// Attackable is set for monsters (Mobius isAttackable): PvE rules apply
	// when they are hit.
	Attackable bool
	// Attack is the shape of the creature's auto-attack (AttackShapeOf).
	Attack AttackShape
	// Armed is Mobius getActiveWeaponItem() != null. Players are always
	// armed: without a weapon they hold their class fists (items 244-252),
	// which add no stats, so their random damage is 0. NPCs are armed when
	// their template's right-hand item is a weapon.
	Armed bool
	// ReuseDelay is the reuse_delay of the active weapon (bows), 0 without.
	ReuseDelay int
	// Shield is set when the left hand holds armor other than a sigil
	// (calcShldUse). Only players can block: Mobius reads an NPC's left-hand
	// item only when it is a weapon, so NPCs never block whatever their
	// data says. Leave it false for NPCs.
	Shield bool
	// X, Y and Z are the position, Heading the client heading (0-65535).
	X, Y, Z int
	Heading int
}

// bowWeapon reports whether the active weapon is a bow (Mobius
// getActiveWeaponItem() is a BOW). An unarmed NPC whose template attack
// type is BOW swings like a bow but does not count here.
func (f *Fighter) bowWeapon() bool { return f.Armed && f.Attack == ShapeBow }

// Strike is one auto-attack swing of Attacker at Target, with everything
// the rolls read. Build one per swing; the formulas do not change it.
type Strike struct {
	Attacker, Target Fighter
	// Soulshot is set when the attacker's soulshot is charged
	// (Attack.hasSoulshot); it doubles P.Atk.
	Soulshot bool
	// Night is set at game-time night, which changes the hit chance by the
	// Dark bonus.
	Night bool

	trait    float64
	hasTrait bool
}

// SetTrait sets the attacker's trait bonus against the target,
// Formulas.calcAttackTraitBonus (see AttackTraitBonus), which multiplies the
// damage. A strike without one deals damage as if no traits applied (×1).
func (s *Strike) SetTrait(bonus float64) { s.trait, s.hasTrait = bonus, true }

func (s *Strike) traitBonus() float64 {
	if s.hasTrait {
		return s.trait
	}
	return 1
}

// position returns where the attacker stands relative to the target.
func (s *Strike) position() Position {
	a, t := &s.Attacker, &s.Target
	return PositionOf(a.X, a.Y, t.X, t.Y, t.Heading)
}

// Position is where a creature stands relative to the way another one
// faces (Mobius Position).
type Position uint8

// The positions.
const (
	PosFront Position = iota
	PosSide
	PosBack
)

var positionNames = [...]string{PosFront: "front", PosSide: "side", PosBack: "back"}

// String returns the name of the position.
func (p Position) String() string {
	if int(p) < len(positionNames) {
		return positionNames[p]
	}
	return fmt.Sprintf("Position(%d)", uint8(p))
}

// radToDeg is Java's Math.toDegrees factor (RADIANS_TO_DEGREES).
const radToDeg = 57.29577951308232

// headingUnits is the heading units per degree LocationUtil uses.
const headingUnits = 182.044444444

// angleFrom is LocationUtil.calculateAngleFrom: the angle in degrees, 0 to
// 360, from (x1, y1) towards (x2, y2). Go's math.Atan2 may differ from
// Java's in the last bit; that matters only exactly on a boundary.
func angleFrom(x1, y1, x2, y2 int) float64 {
	a := mul(math.Atan2(float64(y2-y1), float64(x2-x1)), radToDeg)
	if a < 0 {
		a += 360
	}
	return a
}

// HeadingTo is LocationUtil.calculateHeadingFrom: the client heading, 0 to
// 65535, from (fromX, fromY) towards (toX, toY). The cast truncates, so due
// north (90°) is 16383, not 16384.
func HeadingTo(fromX, fromY, toX, toY int) int {
	return int(javaInt(angleFrom(fromX, fromY, toX, toY) * headingUnits))
}

// PositionOf is Position.getPosition: where a creature at (fromX, fromY)
// stands relative to one at (toX, toY) that faces toHeading. With h the
// difference of toHeading and the heading from the first to the second:
// side for h in 0x2000-0x6000 or 0xA000-0xE000, front for the rest of
// 0x2000-0xE000 (around 0x8000: they face each other), back otherwise.
func PositionOf(fromX, fromY, toX, toY, toHeading int) Position {
	h := toHeading - HeadingTo(fromX, fromY, toX, toY)
	if h < 0 {
		h = -h
	}
	switch {
	case (h >= 0x2000 && h <= 0x6000) || uint32(h-0xA000) <= 0x4000: //nolint:gosec // unsigned range check, as Mobius's toUnsignedLong
		return PosSide
	case uint32(h-0x2000) <= 0xC000: //nolint:gosec // as above
		return PosFront
	}
	return PosBack
}

// distance3D is WorldObject.calculateDistance3D.
func distance3D(a, b *Fighter) float64 {
	dx, dy, dz := float64(a.X-b.X), float64(a.Y-b.Y), float64(a.Z-b.Z)
	return math.Sqrt(mul(dx, dx) + mul(dy, dy) + mul(dz, dz))
}

// javaLong is Java's (long) cast of a double: toward zero, NaN to 0, and
// saturating at the long range.
func javaLong(x float64) int64 {
	switch {
	case x != x:
		return 0
	case x >= math.MaxInt64: // 2^63 as a float64
		return math.MaxInt64
	case x <= math.MinInt64:
		return math.MinInt64
	}
	return int64(x)
}

// javaClamp is Java 21's Math.clamp(double, double, double) for lo < hi.
func javaClamp(v, lo, hi float64) float64 { return math.Min(hi, math.Max(v, lo)) }
