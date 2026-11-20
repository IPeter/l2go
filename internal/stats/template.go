package stats

import (
	"fmt"
	"math"
)

// BaseStats holds a value for each base stat: a template's base values, or
// a bonus such as the henna or armor set bonus.
type BaseStats struct {
	STR, CON, DEX, INT, WIT, MEN int
}

// RaceDwarf is the race ID (Mobius Race ordinal) of dwarves, who get a
// bigger inventory.
const RaceDwarf = 4

// ArmorDefs are a class's P.Def per armor slot (basePDef in the player
// template). The base P.Def is their sum; each worn armor slot takes its
// own part away (FuncPDefMod), so the armor's P.Def replaces it.
type ArmorDefs struct {
	Chest, Legs, Head, Feet, Gloves, Underwear, Cloak int
}

// JewelDefs are a class's M.Def per jewel slot (baseMDef in the player
// template).
//
// Mobius reads rfinger for both finger slots: the base M.Def is
// REar + LEar + 2*RFinger + Neck (PlayerTemplateData), and a ring in the left
// finger takes RFinger away (PlayerTemplate). LFinger is therefore unused;
// it equals RFinger in every Interlude template.
type JewelDefs struct {
	REar, LEar, RFinger, LFinger, Neck int
}

// Collision is a collision radius and height. Mobius keeps them in float
// fields, so they are used at float32 precision.
type Collision struct {
	Radius, Height float64
}

// LevelGain holds a class's base values at one level (lvlUpgainData).
// Mobius keeps HP, MP and CP in float arrays, so they are used at float32
// precision. The regeneration rates are kept for later.
type LevelGain struct {
	HP, MP, CP                float64
	HPRegen, MPRegen, CPRegen float64
}

// ClassTemplate holds the base values of a player class (Mobius
// PlayerTemplate, from stats/players/templates).
type ClassTemplate struct {
	ClassID int
	// Race is the Mobius Race ordinal: 0 Human, 1 Elf, 2 Dark Elf, 3 Orc,
	// 4 Dwarf.
	Race int
	Base BaseStats
	// PAtk and MAtk are basePAtk and baseMAtk.
	PAtk, MAtk int
	// PAtkSpd is basePAtkSpd; 0 means the Mobius default 300.
	PAtkSpd int
	// MAtkSpd is baseMAtkSpd, which the player XML does not set; 0 means
	// the Mobius default 333.
	MAtkSpd int
	// CritRate is baseCritRate.
	CritRate int
	// AttackRange is baseAtkRange. Mobius never uses it for players: they
	// always have an active weapon (PhysicalAttackRange).
	AttackRange int
	PDef        ArmorDefs
	MDef        JewelDefs
	// RunSpeed and WalkSpeed are baseMoveSpd run and walk.
	RunSpeed, WalkSpeed float64
	// SwimRunSpeed and SwimWalkSpeed are baseMoveSpd fastSwim and
	// slowSwim; 0 means the run and walk speed, as in Mobius.
	SwimRunSpeed, SwimWalkSpeed float64
	// Male and Female are collisionMale and collisionFemale.
	Male, Female Collision
	// Levels holds the per-level values: Levels[0] is level 1. Its length
	// is the highest level (80 in Interlude).
	Levels []LevelGain
}

// Validate checks that the template can be used: it has levels and
// positive run and walk speeds.
func (t *ClassTemplate) Validate() error {
	switch {
	case t == nil:
		return fmt.Errorf("stats: %w: no class template", ErrInvalid)
	case len(t.Levels) == 0:
		return fmt.Errorf("stats: %w: class %d has no level data", ErrInvalid, t.ClassID)
	case !(t.RunSpeed > 0) || !(t.WalkSpeed > 0) || math.IsInf(t.RunSpeed, 0) || math.IsInf(t.WalkSpeed, 0):
		return fmt.Errorf("stats: %w: class %d has run speed %v and walk speed %v, want positive numbers", ErrInvalid, t.ClassID, t.RunSpeed, t.WalkSpeed)
	}
	return nil
}

// MaxLevel returns the highest level the template has data for.
func (t *ClassTemplate) MaxLevel() int { return len(t.Levels) }

// clampLevel brings level into 1..MaxLevel. Mobius never has a player level
// outside its level tables (PlayerStat.setLevel caps it).
func (t *ClassTemplate) clampLevel(level int) int {
	return min(max(level, 1), len(t.Levels))
}

// gain returns the values of level, which must be in 1..MaxLevel.
func (t *ClassTemplate) gain(level int) *LevelGain { return &t.Levels[level-1] }

// basePDef is the base P.Def: the sum of the armor slots
// (PlayerTemplateData).
func (t *ClassTemplate) basePDef() int {
	d := t.PDef
	return d.Chest + d.Legs + d.Head + d.Feet + d.Gloves + d.Underwear + d.Cloak
}

// baseMDef is the base M.Def, with rfinger counted twice like
// PlayerTemplateData does.
func (t *ClassTemplate) baseMDef() int {
	d := t.MDef
	return d.REar + d.LEar + d.RFinger + d.RFinger + d.Neck
}

func (t *ClassTemplate) pAtkSpd() int { return orDefault(t.PAtkSpd, 300) }
func (t *ClassTemplate) mAtkSpd() int { return orDefault(t.MAtkSpd, 333) }

func (t *ClassTemplate) swimRunSpeed() float64 {
	if t.SwimRunSpeed > 0 {
		return t.SwimRunSpeed
	}
	return t.RunSpeed
}

func (t *ClassTemplate) swimWalkSpeed() float64 {
	if t.SwimWalkSpeed > 0 {
		return t.SwimWalkSpeed
	}
	return t.WalkSpeed
}

// NpcTemplate holds the base values of an NPC (the stats element of the NPC
// XML, as Mobius NpcData reads it). Values are given as the XML has them;
// the Mobius storage types are applied when they are used.
type NpcTemplate struct {
	Level int
	Base  BaseStats
	// HP and MP are vitals hp and mp. Mobius data holds them divided by
	// the CON and MEN bonus; the formulas multiply it back. They are used
	// at float32 precision (Mobius float fields).
	HP, MP float64
	// HPRegen and MPRegen are vitals hpRegen and mpRegen, the regeneration
	// per tick before modifiers (NpcRegen). Mobius keeps them in float
	// fields, so they are used at float32 precision.
	HPRegen, MPRegen float64
	// PAtk and MAtk are attack physical and magical; PDef and MDef are
	// defence physical and magical. Mobius keeps them in int fields, so the
	// fractions are dropped.
	PAtk, MAtk, PDef, MDef float64
	// CritRate is attack critical.
	CritRate int
	// PAtkSpd is attack attackSpeed; 0 means the Mobius default 300.
	PAtkSpd int
	// MAtkSpd is not in the NPC XML; 0 means the Mobius default 333.
	MAtkSpd int
	// AttackRange is attack range.
	AttackRange int
	// RunSpeed and WalkSpeed are speed run and walk ground. Mobius uses
	// 0.1 for a value of 0 or less (NpcData).
	RunSpeed, WalkSpeed float64
	// SwimRunSpeed and SwimWalkSpeed are speed run and walk swim, which no
	// Interlude NPC sets; 0 means the ground speed, as in Mobius.
	SwimRunSpeed, SwimWalkSpeed float64
	// Collision is collision radius and height normal.
	Collision Collision
}

func (t *NpcTemplate) pAtkSpd() int { return orDefault(t.PAtkSpd, 300) }
func (t *NpcTemplate) mAtkSpd() int { return orDefault(t.MAtkSpd, 333) }

func (t *NpcTemplate) runSpeed() float64  { return npcGroundSpeed(t.RunSpeed) }
func (t *NpcTemplate) walkSpeed() float64 { return npcGroundSpeed(t.WalkSpeed) }

func (t *NpcTemplate) swimRunSpeed() float64 {
	if t.SwimRunSpeed > 0 {
		return t.SwimRunSpeed
	}
	return t.runSpeed()
}

func (t *NpcTemplate) swimWalkSpeed() float64 {
	if t.SwimWalkSpeed > 0 {
		return t.SwimWalkSpeed
	}
	return t.walkSpeed()
}

// npcGroundSpeed is NpcData's rule for ground speeds.
func npcGroundSpeed(v float64) float64 {
	if v <= 0 {
		return 0.1
	}
	return v
}

func orDefault(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}
