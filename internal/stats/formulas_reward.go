package stats

import "math"

// MonsterReward is what a monster's template gives for a kill.
type MonsterReward struct {
	// Level is the monster's level.
	Level int
	// Exp and SP are the template's acquire exp and sp.
	Exp, SP float64
}

// ExpSpShare is Attackable.calculateExpAndSp: the experience and SP for
// damage out of total, for a rewarded player (or party) of level.
//
// Nothing unless the level difference (level − monster level) is under
// MonsterExpMaxLevelDifference either way. Otherwise the template exp × RateXP
// cut to a long (Npc.getExpReward) and sp × RateSP cut to an int
// (getSpReward), each × damage / total in integer division, at least 0.
// Above level 84 a monster 3 to 10 levels higher gives 97 % down to 3 %;
// Interlude never gets there. total counts the damage of every rewarded
// attacker: players who did more than 1 damage and stand within the party
// range. A total of 0 gives nothing, where Mobius fails and rewards nobody.
func (c *CombatRules) ExpSpShare(m MonsterReward, level int, damage, total int64) (exp, sp float64) {
	diff := level - m.Level
	limit := c.cfg.MonsterExpMaxLevelDifference
	if diff >= limit || diff <= -limit || total == 0 {
		return 0, 0
	}
	expReward := javaLong(mul(m.Exp, f32(c.cfg.RateXP)))
	spReward := int64(javaInt(mul(m.SP, f32(c.cfg.RateSP))))
	exp = float64(max(0, expReward*damage/total))
	sp = float64(max(0, spReward*damage/total))
	if level > 84 && diff <= -3 {
		f := highLevelPenalty(diff)
		exp, sp = mul(exp, f), mul(sp, f)
	}
	return exp, sp
}

// highLevelPenalty is calculateExpAndSp's table for players above level 84
// who kill a monster 3 or more levels higher.
func highLevelPenalty(diff int) float64 {
	switch diff {
	case -3:
		return 0.97
	case -4:
		return 0.67
	case -5:
		return 0.42
	case -6:
		return 0.25
	case -7:
		return 0.15
	case -8:
		return 0.09
	case -9:
		return 0.05
	case -10:
		return 0.03
	}
	return 1
}

// Reward is the experience and SP a kill adds to a player.
type Reward struct {
	// Exp and SP are added to the player (PlayerStat.addExpAndSp's finalExp
	// and finalSp, both Math.round).
	Exp, SP int64
	// ShownExp and ShownSP are what "You have earned N experience and M
	// SP." prints: Mobius prints the values before the rounding, cut, so
	// with a bonus they can be 1 below Exp and SP. The messages for exp or
	// SP alone print Exp or SP.
	ShownExp, ShownSP int64
}

// SoloReward is Attackable.calculateRewards for a player without a party,
// followed by PlayerStat.addExpAndSp. The share (ExpSpShare) goes through
// the player's expSpRate modifiers: Math.round for the experience, an (int)
// cast for the SP. addExpAndSp then multiplies each by its bonus,
// 1 + (bonusExp / 100 − 1 when above 1), at least 1 and at most MaxExpBonus
// (bonusSp and MaxSPBonus alike), and rounds them. Vitality, premium,
// champions, servitors, pets and overhit are not applied; the caller checks
// that the player is alive and near.
func (c *CombatRules) SoloReward(p *Player, m MonsterReward, damage, total int64) Reward {
	exp, sp := c.ExpSpShare(m, p.Level(), damage, total)
	addExp := javaRound(p.Calc(RExp, exp))
	addSP := javaInt(p.Calc(RExp, sp))
	toExp := mul(float64(addExp), bonusMultiplier(p, BonusExp, c.cfg.MaxExpBonus))
	toSP := mul(float64(addSP), bonusMultiplier(p, BonusSP, c.cfg.MaxSPBonus))
	return Reward{
		Exp:      javaRound(toExp),
		SP:       javaRound(toSP),
		ShownExp: javaLong(toExp),
		ShownSP:  javaLong(toSP),
	}
}

// bonusMultiplier is PlayerStat.getExpBonusMultiplier (stat BonusExp) or
// getSpBonusMultiplier (BonusSP) without vitality.
func bonusMultiplier(p *Player, s Stat, limit float64) float64 {
	bonus := 1.0
	if b := 1 + p.Calc(s, 0)/100; b > 1 {
		bonus += b - 1
	}
	bonus = math.Max(bonus, 1)
	if limit > 0 {
		bonus = math.Min(bonus, limit)
	}
	return bonus
}
