package stats

import (
	"fmt"
	"math"
)

// Npc is the stat set of an NPC: its template and the modifiers on it
// (none until skills and buffs exist). Like Mobius's shared standard NPC
// calculator, an Npc without modifiers holds no per-stat lists. Getters
// change nothing. Not safe for concurrent use.
type Npc struct {
	rules   *Rules
	tmpl    *NpcTemplate
	gmSpeed float64
	motion  Motion
	mods    Modifiers
	version uint64
}

// NewNpc returns the stat set of an NPC of template t, walking on land.
func NewNpc(r *Rules, t *NpcTemplate) (*Npc, error) {
	if r == nil {
		return nil, fmt.Errorf("stats: %w: no rules", ErrInvalid)
	}
	if t == nil {
		return nil, fmt.Errorf("stats: %w: no NPC template", ErrInvalid)
	}
	return &Npc{rules: r, tmpl: t, gmSpeed: 1}, nil
}

// Version returns a counter that changes whenever the stat set changes.
func (n *Npc) Version() uint64 { return n.version }

func (n *Npc) changed() { n.version++ }

// Template returns the NPC template.
func (n *Npc) Template() *NpcTemplate { return n.tmpl }

// Level returns the NPC level, which is the template's (NpcStat.getLevel).
func (n *Npc) Level() int { return n.tmpl.Level }

// Motion returns the movement state.
func (n *Npc) Motion() Motion { return n.motion }

// SetMotion sets the movement state. Motion.Swamp does not apply to NPCs.
func (n *Npc) SetMotion(m Motion) {
	if m != n.motion {
		n.motion = m
		n.changed()
	}
}

// SetGMSpeed sets the //gmspeed multiplier of the base movement speeds; 0
// or less means 1.
func (n *Npc) SetGMSpeed(mult float64) {
	if !(mult > 0) {
		mult = 1
	}
	if mult != n.gmSpeed {
		n.gmSpeed = mult
		n.changed()
	}
}

// AddModifiers adds modifiers. If one of them is invalid, none is added.
func (n *Npc) AddModifiers(mods ...Modifier) error {
	if err := n.mods.Add(mods...); err != nil {
		return err
	}
	if len(mods) > 0 {
		n.changed()
	}
	return nil
}

// RemoveOwner removes every modifier of owner and returns how many it
// removed.
func (n *Npc) RemoveOwner(owner Owner) int {
	c := n.mods.RemoveOwner(owner)
	if c > 0 {
		n.changed()
	}
	return c
}

// Calc returns stat for the base value init (CreatureStat.calcStat with no
// target and skill).
func (n *Npc) Calc(stat Stat, init float64) float64 { return n.CalcFor(stat, init, Context{}) }

// CalcFor is Calc with a target and skill for the modifier conditions.
func (n *Npc) CalcFor(stat Stat, init float64, ctx Context) float64 {
	return calcStat(n, n.mods.Of(stat), stat, init, ctx)
}

// npcFormulas are the stats of the standard NPC calculator
// (Formulas.getStdNPCCalculators). Casting and movement speed join them
// only when Config.NpcSpeedsWithoutStatBonus is off.
var npcFormulas = [NumStats]bool{
	MaxHP: true, MaxMP: true, PAtk: true, MAtk: true, PDef: true, MDef: true,
	CritRate: true, MCritRate: true, AccCombat: true, REvas: true, PAtkSpd: true,
}

func (n *Npc) hasFormula(s Stat) bool {
	if s == MAtkSpd || s == RunSpd {
		return !n.rules.cfg.NpcSpeedsWithoutStatBonus
	}
	return s.Valid() && npcFormulas[s]
}

// formula runs the NPC formula functions of s on v: the same functions as
// for players, through their non-player branches.
func (n *Npc) formula(s Stat, v float64) float64 {
	b := &n.rules.bonus
	switch s {
	case MaxHP: // FuncMaxHpMul
		return mul(v, b.Bonus(CON, n.CON()))
	case MaxMP: // FuncMaxMpMul
		return mul(v, b.Bonus(MEN, n.MEN()))
	case PAtk: // FuncPAtkMod
		return mul(mul(v, b.Bonus(STR, n.STR())), n.LevelMod())
	case MAtk: // FuncMAtkMod
		return mAtkMod(v, b.Bonus(INT, n.INT()), n.LevelMod())
	case PDef: // FuncPDefMod: no armor slots for NPCs
		return mul(v, n.LevelMod())
	case MDef: // FuncMDefMod: no jewel slots for NPCs
		return mul(mul(v, b.Bonus(MEN, n.MEN())), n.LevelMod())
	case CritRate: // FuncAtkCritical
		return mul(mul(v, b.Bonus(DEX, n.DEX())), 10)
	case MCritRate: // FuncMAtkCritical: NPCs always get the WIT bonus
		return mul(v, b.Bonus(WIT, n.WIT()))
	case AccCombat: // FuncAtkAccuracy
		return accuracy(v, n.DEX(), n.tmpl.Level)
	case REvas: // FuncAtkEvasion, non-player branch
		level := n.tmpl.Level
		v += mul(math.Sqrt(float64(n.DEX())), 6) + float64(level)
		if level > 69 {
			v += float64((level - 69) + 2)
		}
		return v
	case PAtkSpd: // FuncPAtkSpeed
		return mul(v, b.Bonus(DEX, n.DEX()))
	case MAtkSpd: // FuncMAtkSpeed
		return mul(v, b.Bonus(WIT, n.WIT()))
	case RunSpd: // FuncMoveSpeed
		return mul(v, b.Bonus(DEX, n.DEX()))
	}
	return v
}

// LevelMod returns the level modifier (level + 89) / 100.
func (n *Npc) LevelMod() float64 { return float64(n.tmpl.Level+89) / 100 }

func (n *Npc) baseStat(s Stat, base int) int { return int(javaInt(n.Calc(s, float64(base)))) }

// STR returns STR with modifiers. The others are alike.
func (n *Npc) STR() int { return n.baseStat(STR, n.tmpl.Base.STR) }
func (n *Npc) CON() int { return n.baseStat(CON, n.tmpl.Base.CON) }
func (n *Npc) DEX() int { return n.baseStat(DEX, n.tmpl.Base.DEX) }
func (n *Npc) INT() int { return n.baseStat(INT, n.tmpl.Base.INT) }
func (n *Npc) WIT() int { return n.baseStat(WIT, n.tmpl.Base.WIT) }
func (n *Npc) MEN() int { return n.baseStat(MEN, n.tmpl.Base.MEN) }

// MaxHP returns the maximum HP (CreatureStat.getMaxHp): the template HP at
// float precision times the CON bonus.
func (n *Npc) MaxHP() int { return int(javaInt(n.Calc(MaxHP, f32(n.tmpl.HP)))) }

// MaxMP returns the maximum MP.
func (n *Npc) MaxMP() int { return int(javaInt(n.Calc(MaxMP, f32(n.tmpl.MP)))) }

// PAtk returns P.Atk, capped at Config.MaxPAtk. Raid and champion
// multipliers are not applied yet.
func (n *Npc) PAtk() float64 {
	return math.Min(n.Calc(PAtk, float64(javaInt(n.tmpl.PAtk))), float64(n.rules.cfg.MaxPAtk))
}

// MAtk returns M.Atk, capped at Config.MaxMAtk.
func (n *Npc) MAtk() float64 {
	return math.Min(n.Calc(MAtk, float64(javaInt(n.tmpl.MAtk))), float64(n.rules.cfg.MaxMAtk))
}

// PDef returns P.Def.
func (n *Npc) PDef() float64 { return n.Calc(PDef, float64(javaInt(n.tmpl.PDef))) }

// MDef returns M.Def.
func (n *Npc) MDef() float64 { return n.Calc(MDef, float64(javaInt(n.tmpl.MDef))) }

// PAtkSpd returns the attack speed, rounded; NPCs have no cap.
func (n *Npc) PAtkSpd() float64 {
	return float64(javaRound(n.Calc(PAtkSpd, float64(n.tmpl.pAtkSpd()))))
}

// MAtkSpd returns the casting speed, capped at Config.MaxMAtkSpeed.
func (n *Npc) MAtkSpd() int {
	return int(javaInt(math.Min(n.Calc(MAtkSpd, float64(n.tmpl.mAtkSpd())), float64(n.rules.cfg.MaxMAtkSpeed))))
}

// Accuracy returns the accuracy.
func (n *Npc) Accuracy() int { return int(javaRoundInt(n.Calc(AccCombat, 0))) }

// Evasion returns the evasion, capped at Config.MaxEvasion.
func (n *Npc) Evasion() int {
	return min(int(javaRoundInt(n.Calc(REvas, 0))), n.rules.cfg.MaxEvasion)
}

// Crit returns the critical rate, capped at Config.MaxPCritRate.
func (n *Npc) Crit() int {
	v := math.Min(n.Calc(CritRate, float64(n.tmpl.CritRate)), float64(n.rules.cfg.MaxPCritRate))
	return int(javaInt(v + .5))
}

// PhysicalAttackRange returns the attack reach: NPC weapons add no stats, so
// it starts from the template range.
func (n *Npc) PhysicalAttackRange() int {
	return int(javaInt(n.Calc(PAtkRange, float64(n.tmpl.AttackRange))))
}

// moveSpeed is CreatureStat's movement speed: base × //gmspeed through the
// runSpd chain, 0 for a base of 0 or less.
func (n *Npc) moveSpeed(base float64) float64 {
	if base <= 0 {
		return 0
	}
	return n.Calc(RunSpd, mul(base, n.gmSpeed))
}

// RunSpeed returns the run speed (CreatureStat.getRunSpeed; no cap for
// NPCs). In water it starts from the swim run speed, as in Mobius.
func (n *Npc) RunSpeed() float64 {
	base := n.tmpl.runSpeed()
	if n.motion.InWater {
		base = n.SwimRunSpeed()
	}
	return n.moveSpeed(base)
}

// WalkSpeed returns the walk speed.
func (n *Npc) WalkSpeed() float64 {
	base := n.tmpl.walkSpeed()
	if n.motion.InWater {
		base = n.SwimWalkSpeed()
	}
	return n.moveSpeed(base)
}

// SwimRunSpeed returns the fast swim speed.
func (n *Npc) SwimRunSpeed() float64 { return n.moveSpeed(n.tmpl.swimRunSpeed()) }

// SwimWalkSpeed returns the slow swim speed.
func (n *Npc) SwimWalkSpeed() float64 { return n.moveSpeed(n.tmpl.swimWalkSpeed()) }

// MoveSpeed returns the current speed in game units per second.
func (n *Npc) MoveSpeed() float64 {
	switch {
	case n.motion.InWater && n.motion.Running:
		return n.SwimRunSpeed()
	case n.motion.InWater:
		return n.SwimWalkSpeed()
	case n.motion.Running:
		return n.RunSpeed()
	}
	return n.WalkSpeed()
}

func (n *Npc) baseMoveSpeed() float64 {
	switch {
	case n.motion.InWater && n.motion.Running:
		return n.tmpl.swimRunSpeed()
	case n.motion.InWater:
		return n.tmpl.swimWalkSpeed()
	case n.motion.Running:
		return n.tmpl.runSpeed()
	}
	return n.tmpl.walkSpeed()
}

// MoveMultiplier returns getMoveSpeed() * (1. / baseSpeed), as Mobius
// computes it.
func (n *Npc) MoveMultiplier() float64 { return mul(n.MoveSpeed(), 1/n.baseMoveSpeed()) }

// AttackSpeedMultiplier returns 1.1 × attack speed / template attack speed
// as a Java float.
func (n *Npc) AttackSpeedMultiplier() float32 {
	return float32(mul(1.1, n.PAtkSpd()) / float64(n.tmpl.pAtkSpd()))
}

// CollisionRadius returns the collision radius at Java float precision.
func (n *Npc) CollisionRadius() float64 { return f32(n.tmpl.Collision.Radius) }

// CollisionHeight returns the collision height at Java float precision.
func (n *Npc) CollisionHeight() float64 { return f32(n.tmpl.Collision.Height) }

// View returns the stat numbers NpcInfo shows (AbstractNpcInfo), plus the
// maximum HP for the HP bar and the current speed.
func (n *Npc) View() NpcView {
	mm := n.MoveMultiplier()
	v := NpcView{
		MaxHP:                 int32(n.MaxHP()),
		PAtkSpd:               javaInt(n.PAtkSpd()),
		MAtkSpd:               int32(n.MAtkSpd()),
		RunSpd:                javaRoundInt(n.RunSpeed() / mm),
		WalkSpd:               javaRoundInt(n.WalkSpeed() / mm),
		SwimRunSpd:            javaRoundInt(n.SwimRunSpeed() / mm),
		SwimWalkSpd:           javaRoundInt(n.SwimWalkSpeed() / mm),
		MoveMultiplier:        mm,
		AttackSpeedMultiplier: n.AttackSpeedMultiplier(),
		CollisionRadius:       n.CollisionRadius(),
		CollisionHeight:       n.CollisionHeight(),
		MoveSpeed:             n.MoveSpeed(),
	}
	if n.motion.Flying {
		v.FlyRunSpd, v.FlyWalkSpd = v.RunSpd, v.WalkSpd
	}
	return v
}
