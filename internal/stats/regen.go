package stats

import (
	"fmt"
	"math"
	"time"
)

// RegenPeriod is how often creatures regenerate (Formulas.getRegeneratePeriod;
// doors every 300 s).
//
// What a tick does is the status layer's (CreatureStatus, PlayerStatus):
//   - A player regenerates CP, then HP, then MP. Each grows by its regen
//     only while it is below its MaxRecoverable value, and is capped at its
//     maximum, not at the recoverable value.
//   - An NPC regenerates while its HP or MP is below its MaxRecoverable
//     value; then both grow, each capped at its maximum.
//   - The timer starts when a value drops below its maximum and stops when
//     all are full.
const RegenPeriod = 3 * time.Second

// Activity is what a player is doing, which scales its regeneration
// (Formulas.calcHpRegen's movement bonus).
type Activity uint8

// The activities.
const (
	Standing Activity = iota // not moving: ×1.1
	Sitting                  // ×1.5
	Walking                  // moving without running: ×1
	Running                  // moving and running: ×0.7
)

var activityNames = [...]string{Standing: "standing", Sitting: "sitting", Walking: "walking", Running: "running"}

// String returns the name of the activity.
func (a Activity) String() string {
	if int(a) < len(activityNames) {
		return activityNames[a]
	}
	return fmt.Sprintf("Activity(%d)", uint8(a))
}

// factor returns the regeneration multiplier of the activity. Mobius
// multiplies nothing for walking; ×1 is the same number.
func (a Activity) factor() float64 {
	switch a {
	case Standing:
		return 1.1
	case Sitting:
		return 1.5
	case Running:
		return 0.7
	}
	return 1
}

// RegenZone holds what zones and residences add to a player's
// regeneration. The zero value adds nothing.
type RegenZone struct {
	// HPMul and MPMul are multipliers in Mobius order. HP: the festival
	// modifier or else the siege one, then the clan hall and the castle
	// restore functions. MP: the festival modifier, then the clan hall and
	// the castle. They apply after the config multiplier and before the
	// activity.
	HPMul, MPMul []float64
	// HPBonus and MPBonus are added at the end (Mother Tree zones).
	HPBonus, MPBonus float64
}

// Regen is what a creature regenerates per RegenPeriod.
type Regen struct {
	HP, MP, CP float64
}

// PlayerRegen returns what a player regenerates per tick
// (Formulas.calcHpRegen, calcMpRegen and calcCpRegen):
//
//   - HP: the class's hpRegen of the level × (level modifier × CON bonus),
//     at least 1, through the regHp modifiers, × the HP multiplier, + the
//     zone bonus. The multiplier is HPRegenMultiplier × the zone
//     multipliers × the activity.
//   - MP: alike with mpRegen and the MEN bonus.
//   - CP: (cpRegen × level modifier) × CON bonus, at least 1, through the
//     regCp modifiers, × CPRegenMultiplier × the activity. Note the other
//     grouping of the products, as in Mobius.
func (c *CombatRules) PlayerRegen(p *Player, act Activity, zone RegenZone) Regen {
	g := p.tmpl.gain(p.level)
	lm := p.LevelMod()
	con := p.StatBonus(CON)
	f := act.factor()

	hpMul := c.cfg.HPRegenMultiplier
	for _, m := range zone.HPMul {
		hpMul = mul(hpMul, m)
	}
	hpMul = mul(hpMul, f)
	hpInit := mul(g.HPRegen, mul(lm, con))

	mpMul := c.cfg.MPRegenMultiplier
	for _, m := range zone.MPMul {
		mpMul = mul(mpMul, m)
	}
	mpMul = mul(mpMul, f)
	mpInit := mul(g.MPRegen, mul(lm, p.StatBonus(MEN)))

	cpInit := mul(mul(g.CPRegen, lm), con)

	return Regen{
		HP: mul(p.Calc(RegHP, math.Max(1, hpInit)), hpMul) + zone.HPBonus,
		MP: mul(p.Calc(RegMP, math.Max(1, mpInit)), mpMul) + zone.MPBonus,
		CP: mul(p.Calc(RegCP, math.Max(1, cpInit)), mul(c.cfg.CPRegenMultiplier, f)),
	}
}

// NpcRegen returns what an NPC regenerates per tick: the template's vitals
// hpRegen and mpRegen (NpcTemplate.HPRegen and MPRegen, Java floats), at
// least 1, through the regHp and regMp modifiers, × the raid multipliers for
// raids. Other NPCs use the player multipliers HPRegenMultiplier and
// MPRegenMultiplier, as in Mobius. No level modifier, stat bonus or activity
// applies. CP is 0.
func (c *CombatRules) NpcRegen(n *Npc, raid bool) Regen {
	hpMul, mpMul := c.cfg.HPRegenMultiplier, c.cfg.MPRegenMultiplier
	if raid {
		hpMul, mpMul = c.cfg.RaidHPRegenMultiplier, c.cfg.RaidMPRegenMultiplier
	}
	return Regen{
		HP: mul(n.Calc(RegHP, math.Max(1, f32(n.tmpl.HPRegen))), hpMul),
		MP: mul(n.Calc(RegMP, math.Max(1, f32(n.tmpl.MPRegen))), mpMul),
	}
}
