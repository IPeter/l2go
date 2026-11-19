package stats

import (
	"fmt"
	"math"
)

// Slot is a paperdoll slot. The values are Mobius's Inventory.PAPERDOLL_*
// (and storage.PaperdollSlot).
type Slot uint8

// The paperdoll slots.
const (
	SlotUnder Slot = iota
	SlotLEar
	SlotREar
	SlotNeck
	SlotLFinger
	SlotRFinger
	SlotHead
	SlotRHand
	SlotLHand
	SlotGloves
	SlotChest
	SlotLegs
	SlotFeet
	SlotCloak
	SlotFace
	SlotHair
	SlotHair2
	NumSlots
)

// Motion is the movement state that picks and scales the speeds.
type Motion struct {
	// Running is set while running instead of walking. Mobius loads
	// players running (GameClient: "running is default").
	Running bool
	// InWater is set inside a water zone: movement uses the swim speeds.
	InWater bool
	// Flying fills the fly speed fields of the info packets.
	Flying bool
	// Swamp is the move bonus of the swamp zone a player is in (SwampZone,
	// usually 0.5), or 0 outside one. Only players use it.
	Swamp float64
}

// slotState is what the formulas need to know about a paperdoll slot.
type slotState struct {
	used      bool
	fullArmor bool
	owner     Owner
}

// Player is the stat set of a player character: class template, level,
// equipped items and other modifiers. Every getter computes its value from
// the current state, as Mobius does on each read; getters change nothing.
// Not safe for concurrent use.
type Player struct {
	rules    *Rules
	tmpl     *ClassTemplate
	level    int
	female   bool
	gm       bool
	gmSpeed  float64
	motion   Motion
	henna    BaseStats
	armorSet BaseStats
	slots    [NumSlots]slotState
	mods     Modifiers
	version  uint64
}

// NewPlayer returns the stat set of a male player of class t at level, with
// nothing equipped, walking on land. level is brought into 1..t.MaxLevel().
func NewPlayer(r *Rules, t *ClassTemplate, level int) (*Player, error) {
	if r == nil {
		return nil, fmt.Errorf("stats: %w: no rules", ErrInvalid)
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return &Player{rules: r, tmpl: t, level: t.clampLevel(level), gmSpeed: 1}, nil
}

// Version returns a counter that changes whenever the stat set changes. Two
// equal versions mean equal stats, unless a modifier's condition depends on
// state outside the stat set.
func (p *Player) Version() uint64 { return p.version }

func (p *Player) changed() { p.version++ }

// Template returns the class template.
func (p *Player) Template() *ClassTemplate { return p.tmpl }

// SetTemplate changes the class template (class change). The level is kept,
// brought into the new template's range.
func (p *Player) SetTemplate(t *ClassTemplate) error {
	if err := t.Validate(); err != nil {
		return err
	}
	p.tmpl = t
	p.level = t.clampLevel(p.level)
	p.changed()
	return nil
}

// Level returns the level.
func (p *Player) Level() int { return p.level }

// SetLevel sets the level, brought into 1..Template().MaxLevel().
func (p *Player) SetLevel(level int) {
	if level = p.tmpl.clampLevel(level); level != p.level {
		p.level = level
		p.changed()
	}
}

// SetFemale sets the sex, which picks the collision size.
func (p *Player) SetFemale(female bool) {
	if female != p.female {
		p.female = female
		p.changed()
	}
}

// SetGM marks the player as a GM: no run speed and attack speed caps, and
// the GM inventory size.
func (p *Player) SetGM(gm bool) {
	if gm != p.gm {
		p.gm = gm
		p.changed()
	}
}

// SetGMSpeed sets the //gmspeed multiplier of the base movement speeds. A
// value of 0 or less means 1, as AdminGmSpeed sets it.
func (p *Player) SetGMSpeed(mult float64) {
	if !(mult > 0) {
		mult = 1
	}
	if mult != p.gmSpeed {
		p.gmSpeed = mult
		p.changed()
	}
}

// Motion returns the movement state.
func (p *Player) Motion() Motion { return p.motion }

// SetMotion sets the movement state.
func (p *Player) SetMotion(m Motion) {
	if m != p.motion {
		p.motion = m
		p.changed()
	}
}

// SetHenna sets the base stat bonus of the player's dyes (FuncHenna).
func (p *Player) SetHenna(b BaseStats) {
	if b != p.henna {
		p.henna = b
		p.changed()
	}
}

// SetArmorSetBonus sets the base stat bonus of a complete armor set, or
// zero when the worn set is not complete (FuncArmorSet).
func (p *Player) SetArmorSetBonus(b BaseStats) {
	if b != p.armorSet {
		p.armorSet = b
		p.changed()
	}
}

// Equip puts an item into slot: the modifiers of what was there are
// removed, and the item's (ItemModifiers) are added with owner, which must
// identify this item (ItemOwner). If owner was in another slot, that slot
// is emptied. Call it again when the item's enchant level changes. The
// paperdoll rules (two-handed weapons, full armor) are the caller's.
func (p *Player) Equip(slot Slot, owner Owner, it Item) error {
	if slot >= NumSlots {
		return fmt.Errorf("stats: %w: unknown paperdoll slot %d", ErrInvalid, slot)
	}
	if owner == (Owner{}) {
		return fmt.Errorf("stats: %w: equipped item without an owner", ErrInvalid)
	}
	mods, err := ItemModifiers(owner, it)
	if err != nil {
		return err
	}
	for s := range p.slots {
		if p.slots[s].used && p.slots[s].owner == owner {
			p.slots[s] = slotState{}
		}
	}
	if prev := p.slots[slot]; prev.used {
		p.mods.RemoveOwner(prev.owner)
	}
	p.mods.RemoveOwner(owner)
	if err := p.mods.Add(mods...); err != nil {
		return err // cannot happen: ItemModifiers builds valid modifiers
	}
	p.slots[slot] = slotState{used: true, fullArmor: it.fullArmor(), owner: owner}
	p.changed()
	return nil
}

// Unequip empties slot and removes the modifiers of its item.
func (p *Player) Unequip(slot Slot) {
	if slot >= NumSlots || !p.slots[slot].used {
		return
	}
	p.mods.RemoveOwner(p.slots[slot].owner)
	p.slots[slot] = slotState{}
	p.changed()
}

// Equipped reports whether slot holds an item.
func (p *Player) Equipped(slot Slot) bool { return slot < NumSlots && p.slots[slot].used }

// AddModifiers adds modifiers, for example of a passive skill or a buff.
// If one of them is invalid, none is added.
func (p *Player) AddModifiers(mods ...Modifier) error {
	if err := p.mods.Add(mods...); err != nil {
		return err
	}
	if len(mods) > 0 {
		p.changed()
	}
	return nil
}

// RemoveOwner removes every modifier of owner and returns how many it
// removed. Use Unequip for items.
func (p *Player) RemoveOwner(owner Owner) int {
	n := p.mods.RemoveOwner(owner)
	if n > 0 {
		p.changed()
	}
	return n
}

// Calc returns stat for the base value init: init through the stat's
// modifiers and formulas (CreatureStat.calcStat with no target and skill).
func (p *Player) Calc(stat Stat, init float64) float64 {
	return p.CalcFor(stat, init, Context{})
}

// CalcFor is Calc with a target and skill for the modifier conditions.
func (p *Player) CalcFor(stat Stat, init float64, ctx Context) float64 {
	return calcStat(p, p.mods.Of(stat), stat, init, ctx)
}

// playerFormulas are the stats Formulas.addFuncsToNewCharacter gives a
// player a formula function for.
var playerFormulas = [NumStats]bool{
	MaxHP: true, MaxCP: true, MaxMP: true,
	PAtk: true, MAtk: true, PDef: true, MDef: true,
	CritRate: true, MCritRate: true, AccCombat: true, REvas: true,
	PAtkSpd: true, MAtkSpd: true, RunSpd: true,
	STR: true, DEX: true, INT: true, MEN: true, CON: true, WIT: true, // FuncHenna, FuncArmorSet
}

func (p *Player) hasFormula(s Stat) bool { return s.Valid() && playerFormulas[s] }

// formula runs the player formula functions of s on v (all at order 1).
func (p *Player) formula(s Stat, v float64) float64 {
	b := &p.rules.bonus
	switch s {
	case MaxHP, MaxCP: // FuncMaxHpMul, FuncMaxCpMul
		return mul(v, b.Bonus(CON, p.CON()))
	case MaxMP: // FuncMaxMpMul
		return mul(v, b.Bonus(MEN, p.MEN()))
	case PAtk: // FuncPAtkMod
		return mul(mul(v, b.Bonus(STR, p.STR())), p.LevelMod())
	case MAtk: // FuncMAtkMod
		return mAtkMod(v, b.Bonus(INT, p.INT()), p.LevelMod())
	case PDef: // FuncPDefMod: worn armor replaces the slot's base P.Def
		d := &p.tmpl.PDef
		if p.slots[SlotChest].used {
			v -= float64(d.Chest)
		}
		if p.slots[SlotLegs].used || (p.slots[SlotChest].used && p.slots[SlotChest].fullArmor) {
			v -= float64(d.Legs)
		}
		if p.slots[SlotHead].used {
			v -= float64(d.Head)
		}
		if p.slots[SlotFeet].used {
			v -= float64(d.Feet)
		}
		if p.slots[SlotGloves].used {
			v -= float64(d.Gloves)
		}
		if p.slots[SlotUnder].used {
			v -= float64(d.Underwear)
		}
		if p.slots[SlotCloak].used {
			v -= float64(d.Cloak)
		}
		return mul(v, p.LevelMod())
	case MDef: // FuncMDefMod: worn jewels replace the slot's base M.Def
		d := &p.tmpl.MDef
		if p.slots[SlotLFinger].used {
			v -= float64(d.RFinger) // Mobius uses rfinger for both fingers
		}
		if p.slots[SlotRFinger].used {
			v -= float64(d.RFinger)
		}
		if p.slots[SlotLEar].used {
			v -= float64(d.LEar)
		}
		if p.slots[SlotREar].used {
			v -= float64(d.REar)
		}
		if p.slots[SlotNeck].used {
			v -= float64(d.Neck)
		}
		return mul(mul(v, b.Bonus(MEN, p.MEN())), p.LevelMod())
	case CritRate: // FuncAtkCritical
		return mul(mul(v, b.Bonus(DEX, p.DEX())), 10)
	case MCritRate: // FuncMAtkCritical: players get the WIT bonus only with a weapon
		if p.slots[SlotRHand].used {
			return mul(v, b.Bonus(WIT, p.WIT()))
		}
		return v
	case AccCombat: // FuncAtkAccuracy
		return accuracy(v, p.DEX(), p.level)
	case REvas: // FuncAtkEvasion, player branch
		v += mul(math.Sqrt(float64(p.DEX())), 6) + float64(p.level)
		diff := float64(p.level - 69)
		if p.level >= 78 {
			diff = mul(diff, 1.2)
		}
		if p.level >= 70 {
			v += diff
		}
		return v
	case PAtkSpd: // FuncPAtkSpeed
		return mul(v, b.Bonus(DEX, p.DEX()))
	case MAtkSpd: // FuncMAtkSpeed
		return mul(v, b.Bonus(WIT, p.WIT()))
	case RunSpd: // FuncMoveSpeed
		return mul(v, b.Bonus(DEX, p.DEX()))
	case STR: // FuncHenna, then FuncArmorSet
		return v + float64(p.henna.STR) + float64(p.armorSet.STR)
	case CON:
		return v + float64(p.henna.CON) + float64(p.armorSet.CON)
	case DEX:
		return v + float64(p.henna.DEX) + float64(p.armorSet.DEX)
	case INT:
		return v + float64(p.henna.INT) + float64(p.armorSet.INT)
	case WIT:
		return v + float64(p.henna.WIT) + float64(p.armorSet.WIT)
	case MEN:
		return v + float64(p.henna.MEN) + float64(p.armorSet.MEN)
	}
	return v
}

// mAtkMod is FuncMAtkMod: v × INT bonus² × level modifier², in Mobius's
// order of operations.
func mAtkMod(v, intBonus, levelMod float64) float64 {
	return mul(mul(v, mul(intBonus, intBonus)), mul(levelMod, levelMod))
}

// accuracy is FuncAtkAccuracy: sqrt(DEX)×6 + level, with extras above
// levels 69 and 77.
func accuracy(v float64, dex, level int) float64 {
	v = v + mul(math.Sqrt(float64(dex)), 6) + float64(level)
	if level > 77 {
		v += float64(level - 76)
	}
	if level > 69 {
		v += float64(level - 69)
	}
	return v
}

// LevelMod returns the level modifier (level + 89) / 100.
func (p *Player) LevelMod() float64 { return float64(p.level+89) / 100 }

func (p *Player) baseStat(s Stat, base int) int { return int(javaInt(p.Calc(s, float64(base)))) }

// STR returns STR with henna, armor set and modifiers. The others are alike.
func (p *Player) STR() int { return p.baseStat(STR, p.tmpl.Base.STR) }
func (p *Player) CON() int { return p.baseStat(CON, p.tmpl.Base.CON) }
func (p *Player) DEX() int { return p.baseStat(DEX, p.tmpl.Base.DEX) }
func (p *Player) INT() int { return p.baseStat(INT, p.tmpl.Base.INT) }
func (p *Player) WIT() int { return p.baseStat(WIT, p.tmpl.Base.WIT) }
func (p *Player) MEN() int { return p.baseStat(MEN, p.tmpl.Base.MEN) }

// MaxHP returns the maximum HP (PlayerStat.getMaxHp).
func (p *Player) MaxHP() int {
	return int(javaInt(p.Calc(MaxHP, f32(p.tmpl.gain(p.level).HP))))
}

// MaxMP returns the maximum MP.
func (p *Player) MaxMP() int {
	return int(javaInt(p.Calc(MaxMP, f32(p.tmpl.gain(p.level).MP))))
}

// MaxCP returns the maximum CP, 0 with Config.DisableCP.
func (p *Player) MaxCP() int {
	if p.rules.cfg.DisableCP {
		return 0
	}
	return int(javaInt(p.Calc(MaxCP, f32(p.tmpl.gain(p.level).CP))))
}

// PAtk returns P.Atk, capped at Config.MaxPAtk (CreatureStat.getPAtk).
func (p *Player) PAtk() float64 {
	return math.Min(p.Calc(PAtk, float64(p.tmpl.PAtk)), float64(p.rules.cfg.MaxPAtk))
}

// MAtk returns M.Atk, capped at Config.MaxMAtk.
func (p *Player) MAtk() float64 {
	return math.Min(p.Calc(MAtk, float64(p.tmpl.MAtk)), float64(p.rules.cfg.MaxMAtk))
}

// PDef returns P.Def.
func (p *Player) PDef() float64 { return p.Calc(PDef, float64(p.tmpl.basePDef())) }

// MDef returns M.Def.
func (p *Player) MDef() float64 { return p.Calc(MDef, float64(p.tmpl.baseMDef())) }

// PAtkSpd returns the attack speed: rounded (CreatureStat.getPAtkSpd) and
// capped at Config.MaxPAtkSpeed for non-GMs (PlayerStat.getPAtkSpd).
func (p *Player) PAtkSpd() float64 {
	v := float64(javaRound(p.Calc(PAtkSpd, float64(p.tmpl.pAtkSpd()))))
	if limit := float64(p.rules.cfg.MaxPAtkSpeed); v > limit && !p.gm {
		return limit
	}
	return v
}

// MAtkSpd returns the casting speed, capped at Config.MaxMAtkSpeed.
func (p *Player) MAtkSpd() int {
	return int(javaInt(math.Min(p.Calc(MAtkSpd, float64(p.tmpl.mAtkSpd())), float64(p.rules.cfg.MaxMAtkSpeed))))
}

// Accuracy returns the accuracy (CreatureStat.getAccuracy).
func (p *Player) Accuracy() int { return int(javaRoundInt(p.Calc(AccCombat, 0))) }

// Evasion returns the evasion, capped at Config.MaxEvasion.
func (p *Player) Evasion() int {
	return min(int(javaRoundInt(p.Calc(REvas, 0))), p.rules.cfg.MaxEvasion)
}

// Crit returns the critical rate, capped at Config.MaxPCritRate and
// rounded half up (CreatureStat.getCriticalHit).
func (p *Player) Crit() int {
	v := math.Min(p.Calc(CritRate, float64(p.tmpl.CritRate)), float64(p.rules.cfg.MaxPCritRate))
	return int(javaInt(v + .5))
}

// PhysicalAttackRange returns the attack reach: the weapon's pAtkRange, or
// the template's range without a weapon.
func (p *Player) PhysicalAttackRange() int {
	base := p.tmpl.AttackRange
	if p.slots[SlotRHand].used {
		base = 0
	}
	return int(javaInt(p.Calc(PAtkRange, float64(base))))
}

// WeightLimit returns the maximum load: CON bonus × 69000 × the config
// multiplier, rounded down, then the weightLimit modifiers
// (Creature.getMaxLoad).
func (p *Player) WeightLimit() int {
	base := math.Floor(mul(mul(p.rules.bonus.Bonus(CON, p.CON()), 69000), p.rules.cfg.WeightLimitMultiplier))
	return int(javaInt(p.Calc(WeightLimit, base)))
}

// InventoryLimit returns the number of inventory slots
// (Player.getInventoryLimit).
func (p *Player) InventoryLimit() int {
	c := &p.rules.cfg
	limit := c.InventoryNoDwarf
	switch {
	case p.gm:
		limit = c.InventoryGM
	case p.tmpl.Race == RaceDwarf:
		limit = c.InventoryDwarf
	}
	return limit + int(javaInt(p.Calc(InventoryLimit, 0)))
}

// moveSpeed is the creature part of every movement speed: the base speed
// times the //gmspeed multiplier, through the runSpd chain; 0 for a base of
// 0 or less.
func (p *Player) moveSpeed(base float64) float64 {
	if base <= 0 {
		return 0
	}
	return p.Calc(RunSpd, mul(base, p.gmSpeed))
}

// limitSpeed adds what PlayableStat and PlayerStat put on top of the
// creature speed: the swamp bonus, Config.RunSpeedBoost and, for non-GMs,
// the Config.MaxRunSpeed cap. Mounts are not supported yet.
func (p *Player) limitSpeed(v float64) float64 {
	if p.motion.Swamp > 0 {
		v = mul(v, p.motion.Swamp)
	}
	v += float64(p.rules.cfg.RunSpeedBoost)
	if limit := float64(p.rules.cfg.MaxRunSpeed); v > limit && !p.gm {
		return limit
	}
	return v
}

// RunSpeed returns the run speed (PlayerStat.getRunSpeed). In water Mobius
// starts from the swim run speed, which already went through the runSpd
// chain, and runs the chain again; that is kept.
func (p *Player) RunSpeed() float64 {
	base := p.tmpl.RunSpeed
	if p.motion.InWater {
		base = p.SwimRunSpeed()
	}
	return p.limitSpeed(p.moveSpeed(base))
}

// WalkSpeed returns the walk speed, like RunSpeed.
func (p *Player) WalkSpeed() float64 {
	base := p.tmpl.WalkSpeed
	if p.motion.InWater {
		base = p.SwimWalkSpeed()
	}
	return p.limitSpeed(p.moveSpeed(base))
}

// SwimRunSpeed returns the fast swim speed (CreatureStat.getSwimRunSpeed;
// no cap).
func (p *Player) SwimRunSpeed() float64 { return p.moveSpeed(p.tmpl.swimRunSpeed()) }

// SwimWalkSpeed returns the slow swim speed.
func (p *Player) SwimWalkSpeed() float64 { return p.moveSpeed(p.tmpl.swimWalkSpeed()) }

// MoveSpeed returns the current speed in game units per second, for the
// movement simulation (CreatureStat.getMoveSpeed).
func (p *Player) MoveSpeed() float64 {
	switch {
	case p.motion.InWater && p.motion.Running:
		return p.SwimRunSpeed()
	case p.motion.InWater:
		return p.SwimWalkSpeed()
	case p.motion.Running:
		return p.RunSpeed()
	}
	return p.WalkSpeed()
}

// baseMoveSpeed is the template speed of the current movement type.
func (p *Player) baseMoveSpeed() float64 {
	switch {
	case p.motion.InWater && p.motion.Running:
		return p.tmpl.swimRunSpeed()
	case p.motion.InWater:
		return p.tmpl.swimWalkSpeed()
	case p.motion.Running:
		return p.tmpl.RunSpeed
	}
	return p.tmpl.WalkSpeed
}

// MoveMultiplier returns the current speed over the template speed of the
// current movement type, computed as Mobius does:
// getMoveSpeed() * (1. / baseSpeed).
func (p *Player) MoveMultiplier() float64 {
	return mul(p.MoveSpeed(), 1/p.baseMoveSpeed())
}

// AttackSpeedMultiplier returns 1.1 × attack speed / template attack speed
// as a Java float (CreatureStat.getAttackSpeedMultiplier).
func (p *Player) AttackSpeedMultiplier() float32 {
	return float32(mul(1.1, p.PAtkSpd()) / float64(p.tmpl.pAtkSpd()))
}

// collision returns the collision size of the player's sex.
func (p *Player) collision() Collision {
	if p.female {
		return p.tmpl.Female
	}
	return p.tmpl.Male
}

// CollisionRadius returns the collision radius at Java float precision.
func (p *Player) CollisionRadius() float64 { return f32(p.collision().Radius) }

// CollisionHeight returns the collision height at Java float precision.
func (p *Player) CollisionHeight() float64 { return f32(p.collision().Height) }

// View returns every stat number UserInfo and CharInfo show, computed as
// those Mobius packets compute them.
func (p *Player) View() PlayerView {
	mm := p.MoveMultiplier()
	v := PlayerView{
		STR:                   int32(p.STR()),
		DEX:                   int32(p.DEX()),
		CON:                   int32(p.CON()),
		INT:                   int32(p.INT()),
		WIT:                   int32(p.WIT()),
		MEN:                   int32(p.MEN()),
		MaxHP:                 int32(p.MaxHP()),
		MaxMP:                 int32(p.MaxMP()),
		MaxCP:                 int32(p.MaxCP()),
		WeightLimit:           int32(p.WeightLimit()),
		InventoryLimit:        int32(p.InventoryLimit()),
		PAtk:                  javaInt(p.PAtk()),
		PAtkSpd:               javaInt(p.PAtkSpd()),
		PDef:                  javaInt(p.PDef()),
		Evasion:               int32(p.Evasion()),
		Accuracy:              int32(p.Accuracy()),
		Crit:                  int32(p.Crit()),
		MAtk:                  javaInt(p.MAtk()),
		MAtkSpd:               int32(p.MAtkSpd()),
		MDef:                  javaInt(p.MDef()),
		RunSpd:                javaRoundInt(p.RunSpeed() / mm),
		WalkSpd:               javaRoundInt(p.WalkSpeed() / mm),
		SwimRunSpd:            javaRoundInt(p.SwimRunSpeed() / mm),
		SwimWalkSpd:           javaRoundInt(p.SwimWalkSpeed() / mm),
		MoveMultiplier:        mm,
		AttackSpeedMultiplier: p.AttackSpeedMultiplier(),
		CollisionRadius:       p.CollisionRadius(),
		CollisionHeight:       p.CollisionHeight(),
		MoveSpeed:             p.MoveSpeed(),
	}
	if p.motion.Flying {
		v.FlyRunSpd, v.FlyWalkSpd = v.RunSpd, v.WalkSpd
	}
	return v
}
