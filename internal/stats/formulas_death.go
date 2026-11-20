package stats

import "math"

// KillerKind is what killed a player, as the death penalties tell killers
// apart (Mobius checks isRaid, then isMonster, then isPlayable).
type KillerKind uint8

// The killer kinds.
const (
	KillerOther    KillerKind = iota // none of the others: a guard, another NPC
	KillerRaid                       // a raid boss or a raid minion (isRaid)
	KillerMonster                    // any other monster (isMonster)
	KillerPlayable                   // a player, summon or pet (isPlayable)
)

// expLostStat and penaltyStat are the stats that reduce the death
// penalties for the killer kind; false for KillerOther.
func (k KillerKind) expLostStat() (Stat, bool) {
	switch k {
	case KillerRaid:
		return ReduceExpLostByRaid, true
	case KillerMonster:
		return ReduceExpLostByMob, true
	case KillerPlayable:
		return ReduceExpLostByPvp, true
	}
	return 0, false
}

func (k KillerKind) penaltyStat() (Stat, bool) {
	switch k {
	case KillerRaid:
		return ReduceDeathPenaltyByRaid, true
	case KillerMonster:
		return ReduceDeathPenaltyByMob, true
	case KillerPlayable:
		return ReduceDeathPenaltyByPvp, true
	}
	return 0, false
}

// DeathLoss is what Player.calculateDeathExpPenalty reads besides the
// player's stats.
type DeathLoss struct {
	// Span is the experience of the player's level: the exp of level + 1
	// minus that of level, at the maximum level the span of the level
	// below it; 0 during an event (no loss).
	Span int64
	// Percent is experienceLoss.xml's percent for the player's level.
	Percent float64
	// Killer is what killed the player.
	Killer KillerKind
	// Karma is set when the player has karma.
	Karma bool
	// Reduced is set for a Festival of Darkness participant or a death in a
	// clan war: the loss is a quarter.
	Reduced bool
}

// DeathExpLoss is Player.calculateDeathExpPenalty: the experience a player
// loses on death. The percent goes through the player's reduceExpLostBy*
// stat for the killer kind and, with karma, × RateKarmaExpLost. The loss is
// Math.round(span × percent / 100), at most a tenth of the span (integer
// division), then a quarter (integer division) when reduced. A level 10
// death at x1 loses round(22972 × 8.875 / 100) = 2039.
//
// Whether to call it is the caller's (Player.doDie): Delevel on, not Lucky
// (levels 1-9 until M7), outside PvP zones unless in a siege zone, not on an
// event. The caller keeps the exp before death and removes the loss, which
// may lower the level.
func (c *CombatRules) DeathExpLoss(p *Player, d DeathLoss) int64 {
	percent := d.Percent
	if s, ok := d.Killer.expLostStat(); ok {
		percent = mul(percent, p.Calc(s, 1))
	}
	if d.Karma {
		percent = mul(percent, f32(c.cfg.RateKarmaExpLost))
	}
	if d.Span <= 0 {
		return 0
	}
	lost := javaRound(mul(float64(d.Span), percent) / 100)
	lost = min(lost, d.Span/10)
	if d.Reduced {
		lost /= 4
	}
	return lost
}

// DeathPenaltyRoll is the roll of Player.calculateDeathPenaltyBuffLevel:
// whether a death raises the Death Penalty level (skill 5076, at most 15).
// It draws Between(1, 100) once and succeeds when the draw is at most
// DeathPenaltyChance × the player's reduceDeathPenaltyBy* stat for the
// killer kind, and the killer is not playable or the player has karma.
//
// The checks before it draw nothing and are the caller's: a killer, and no
// resurrection-special effect, Lucky, event, PvP or siege zone, or GM.
func (c *CombatRules) DeathPenaltyRoll(rnd Rand, p *Player, killer KillerKind, karma bool) bool {
	percent := 1.0
	if s, ok := killer.penaltyStat(); ok {
		percent = mul(percent, p.Calc(s, 1))
	}
	roll := rnd.Between(1, 100)
	return float64(roll) <= mul(float64(c.cfg.DeathPenaltyChance), percent) && (killer != KillerPlayable || karma)
}

// KarmaGain is Formulas.calculateKarmaGain: the karma a player with pkCount
// PKs (before this kill) gets for killing a karma-free, unflagged player or
// summon. (pk × 0.5 + 1) × 60 × 12 below 99 PKs (720 for the first),
// (pk × 0.125 + 37.75) × 60 × 12 below 180, else 43200. For a summon Mobius
// first computes its own value and returns 10800 when that is above 10800;
// otherwise the player formula overwrites it, as here.
func KarmaGain(pkCount int, summon bool) int {
	result := 43200
	pk := float64(pkCount)
	if summon {
		result = int(javaInt(mul(mul(mul(pk, 0.375)+1, 60), 4))) - 150
		if result > 10800 {
			return 10800
		}
	}
	switch {
	case pkCount < 99:
		result = int(javaInt(mul(mul(mul(pk, 0.5)+1, 60), 12)))
	case pkCount < 180:
		result = int(javaInt(mul(mul(mul(pk, 0.125)+37.75, 60), 12)))
	}
	return result
}

// KarmaLost is Formulas.calculateKarmaLost: the karma that exp burns for a
// player whose level has the karmaLoss.xml modifier lossMod. For gained
// experience (exp > 0): |exp / RateKarmaLost| / lossMod / 30, where Mobius
// divides the long by a Java float, so the quotient is a float; RateKarmaLost
// -1 means RateXP. For exp of 0 or less: |exp| / lossMod / 30. Cut to an
// int. PlayerStat.addExp applies it only to players with karma, outside
// PvP zones and without a cursed weapon.
func (c *CombatRules) KarmaLost(exp int64, lossMod float64) int {
	if exp > 0 {
		rate := c.cfg.RateKarmaLost
		if rate == -1 {
			rate = c.cfg.RateXP
		}
		q := float32(exp) / float32(rate)
		q = float32(math.Abs(float64(q)))
		return int(javaInt(float64(q) / lossMod / 30))
	}
	if exp < 0 {
		exp = -exp
	}
	return int(javaInt(float64(exp) / lossMod / 30))
}

// KarmaAfterDeath is the karma a player keeps after dying with karma
// (Player.doDie): 0 below 200, otherwise karma − karma / 4 (integer
// division).
func KarmaAfterDeath(karma int) int {
	if karma <= 0 {
		return karma
	}
	if karma < 200 {
		return 0
	}
	return karma - karma/4
}

// HateRatio is Attackable.getHateRatio: 1 + percent / 100. percent is the
// sum of the NPC template's SetHateGroupRatio, SetHateOccupationRatio and
// SetHateRaceRatio parameters whose group (category), class or race match
// the attacking player; 0 (ratio 1) for attackers that are not players. An
// Orc Archer (SetHateRace HUMAN, SetHateRaceRatio 10) hates humans × 1.1.
func HateRatio(percent int) float64 { return 1 + float64(percent)/100.0 }

// DamageHate is the hate a hit adds to a monster's aggro list
// (Attackable.addDamage): damage × 100 / (level + 7) in long integer
// division, × the hate ratio, cut to a long and then to an int. level is the
// monster's. Attackable.addDamageHate (the AI's) then counts a hate of 0 as
// 1.
func DamageHate(damage, level int, ratio float64) int {
	h := int64(damage) * 100 / int64(level+7)
	return int(int32(javaLong(mul(float64(h), ratio)))) //nolint:gosec // Java (int) cast of a long keeps the low bits
}
