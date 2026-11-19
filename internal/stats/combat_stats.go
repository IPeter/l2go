package stats

import "math"

// The getters of this file take a context, the other creature of a fight,
// for the modifier conditions (the target argument of Mobius calcStat). With
// an empty context they equal the getters the views use.

// EvasionFor returns the evasion against the creature in ctx, capped at
// Config.MaxEvasion (CreatureStat.getEvasionRate).
func (p *Player) EvasionFor(ctx Context) int {
	return min(int(javaRoundInt(p.CalcFor(REvas, 0, ctx))), p.rules.cfg.MaxEvasion)
}

// CritFor returns the critical rate against the creature in ctx: capped at
// Config.MaxPCritRate and rounded half up (CreatureStat.getCriticalHit).
func (p *Player) CritFor(ctx Context) int {
	v := math.Min(p.CalcFor(CritRate, float64(p.tmpl.CritRate), ctx), float64(p.rules.cfg.MaxPCritRate))
	return int(javaInt(v + .5))
}

// PAtkFor returns P.Atk against the creature in ctx, capped at
// Config.MaxPAtk (CreatureStat.getPAtk).
func (p *Player) PAtkFor(ctx Context) float64 {
	return math.Min(p.CalcFor(PAtk, float64(p.tmpl.PAtk), ctx), float64(p.rules.cfg.MaxPAtk))
}

// PDefFor returns P.Def against the creature in ctx (CreatureStat.getPDef).
func (p *Player) PDefFor(ctx Context) float64 {
	return p.CalcFor(PDef, float64(p.tmpl.basePDef()), ctx)
}

// ShieldDef returns the shield defence (CreatureStat.getShldDef): the sDef
// modifiers on 0, cut to an int.
func (p *Player) ShieldDef() int { return int(javaInt(p.Calc(SDef, 0))) }

// StatBonus returns the statBonus.xml multiplier of base stat s at the
// player's value of it (BaseStat.calcBonus); 1 for other stats.
func (p *Player) StatBonus(s Stat) float64 {
	switch s {
	case STR:
		return p.rules.Bonus(s, p.STR())
	case CON:
		return p.rules.Bonus(s, p.CON())
	case DEX:
		return p.rules.Bonus(s, p.DEX())
	case INT:
		return p.rules.Bonus(s, p.INT())
	case WIT:
		return p.rules.Bonus(s, p.WIT())
	case MEN:
		return p.rules.Bonus(s, p.MEN())
	}
	return 1
}

// MaxRecoverableHP returns the HP regeneration stops at: the
// maxRecoverableHp modifiers on MaxHP (CreatureStat.getMaxRecoverableHp).
func (p *Player) MaxRecoverableHP() int {
	return int(javaInt(p.Calc(MaxRecoverableHP, float64(p.MaxHP()))))
}

// MaxRecoverableMP returns the MP regeneration stops at.
func (p *Player) MaxRecoverableMP() int {
	return int(javaInt(p.Calc(MaxRecoverableMP, float64(p.MaxMP()))))
}

// MaxRecoverableCP returns the CP regeneration stops at.
func (p *Player) MaxRecoverableCP() int {
	return int(javaInt(p.Calc(MaxRecoverableCP, float64(p.MaxCP()))))
}

// EvasionFor returns the evasion against the creature in ctx, capped at
// Config.MaxEvasion.
func (n *Npc) EvasionFor(ctx Context) int {
	return min(int(javaRoundInt(n.CalcFor(REvas, 0, ctx))), n.rules.cfg.MaxEvasion)
}

// CritFor returns the critical rate against the creature in ctx, capped at
// Config.MaxPCritRate and rounded half up.
func (n *Npc) CritFor(ctx Context) int {
	v := math.Min(n.CalcFor(CritRate, float64(n.tmpl.CritRate), ctx), float64(n.rules.cfg.MaxPCritRate))
	return int(javaInt(v + .5))
}

// PAtkFor returns P.Atk against the creature in ctx, capped at
// Config.MaxPAtk. Raid multipliers are not applied yet.
func (n *Npc) PAtkFor(ctx Context) float64 {
	return math.Min(n.CalcFor(PAtk, float64(javaInt(n.tmpl.PAtk)), ctx), float64(n.rules.cfg.MaxPAtk))
}

// PDefFor returns P.Def against the creature in ctx.
func (n *Npc) PDefFor(ctx Context) float64 {
	return n.CalcFor(PDef, float64(javaInt(n.tmpl.PDef)), ctx)
}

// ShieldDef returns the shield defence: the sDef modifiers on 0. The shield
// value of the NPC data is not used, as in Mobius.
func (n *Npc) ShieldDef() int { return int(javaInt(n.Calc(SDef, 0))) }

// StatBonus returns the statBonus.xml multiplier of base stat s at the NPC's
// value of it; 1 for other stats.
func (n *Npc) StatBonus(s Stat) float64 {
	switch s {
	case STR:
		return n.rules.Bonus(s, n.STR())
	case CON:
		return n.rules.Bonus(s, n.CON())
	case DEX:
		return n.rules.Bonus(s, n.DEX())
	case INT:
		return n.rules.Bonus(s, n.INT())
	case WIT:
		return n.rules.Bonus(s, n.WIT())
	case MEN:
		return n.rules.Bonus(s, n.MEN())
	}
	return 1
}

// MaxRecoverableHP returns the HP regeneration stops at.
func (n *Npc) MaxRecoverableHP() int {
	return int(javaInt(n.Calc(MaxRecoverableHP, float64(n.MaxHP()))))
}

// MaxRecoverableMP returns the MP regeneration stops at.
func (n *Npc) MaxRecoverableMP() int {
	return int(javaInt(n.Calc(MaxRecoverableMP, float64(n.MaxMP()))))
}
