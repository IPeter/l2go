package social

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxPartyMembers is the party size limit (RequestJoinParty: a party of 9 is
// full).
const MaxPartyMembers = 9

// partyBonus is Party.BONUS_EXP_SP: the exp/SP multiplier by the number of
// valid members, 1 to 9.
var partyBonus = [...]float64{1.0, 1.10, 1.20, 1.30, 1.40, 1.50, 2.0, 2.10, 2.20}

// CutoffMethod is Mobius PartyExpType (PartyXpCutoffMethod): which members
// of a party share a kill's exp and SP.
type CutoffMethod int

// The cutoff methods. Mobius ships "level" (Player.ini, though its comment
// says the default is "highfive").
const (
	// CutoffLevel drops members more than CutoffLevel levels below the
	// highest rewarded member.
	CutoffLevel CutoffMethod = iota
	// CutoffPercentage drops members whose level² is under CutoffPercent %
	// of the sum of the members' level².
	CutoffPercentage
	// CutoffAuto drops members whose level² is under the sum of level²
	// divided by the member count squared.
	CutoffAuto
	// CutoffHighFive keeps everyone and scales each member's share by the
	// gap table (Gaps, GapPercents).
	CutoffHighFive
	// CutoffNone keeps everyone.
	CutoffNone
)

var cutoffNames = [...]string{"level", "percentage", "auto", "highfive", "none"}

// String returns the config name of m.
func (m CutoffMethod) String() string {
	if m < 0 || int(m) >= len(cutoffNames) {
		return "CutoffMethod(" + strconv.Itoa(int(m)) + ")"
	}
	return cutoffNames[m]
}

// ParseCutoffMethod reads a PartyXpCutoffMethod value, in any case, as
// Mobius does (Enum.valueOf of the upper-cased string).
func ParseCutoffMethod(s string) (CutoffMethod, error) {
	for i, n := range cutoffNames {
		if strings.EqualFold(strings.TrimSpace(s), n) {
			return CutoffMethod(i), nil
		}
	}
	return 0, fmt.Errorf("social: unknown party exp cutoff method %q (want level, percentage, auto, highfive or none)", s)
}

// PartyExp is the party exp configuration (Player.ini 791–821, Rates.ini).
type PartyExp struct {
	// Method is PartyXpCutoffMethod.
	Method CutoffMethod
	// CutoffLevel is PartyXpCutoffLevel (method level).
	CutoffLevel int
	// CutoffPercent is PartyXpCutoffPercent (method percentage).
	CutoffPercent float64
	// Gaps and GapPercents are PartyXpCutoffGaps and
	// PartyXpCutoffGapPercent (method highfive): a member whose level is
	// Gaps[i][0] to Gaps[i][1] below the highest gets GapPercents[i] % of
	// the share. Both have the same length.
	Gaps        [][2]int
	GapPercents []int
	// RateXP and RateSP are RatePartyXp and RatePartySp. Mobius keeps them
	// in Java floats; they are rounded to float32 where used.
	RateXP, RateSP float64
}

// DefaultPartyExp returns Mobius's shipped values: method level, 20 levels,
// 3 %, gaps 0–9/10–14/15–99 at 100/30/0 %, rates 1.
func DefaultPartyExp() PartyExp {
	return PartyExp{
		Method:        CutoffLevel,
		CutoffLevel:   20,
		CutoffPercent: 3.0,
		Gaps:          [][2]int{{0, 9}, {10, 14}, {15, 99}},
		GapPercents:   []int{100, 30, 0},
		RateXP:        1,
		RateSP:        1,
	}
}

// ParseCutoffGaps reads PartyXpCutoffGaps ("0,9;10,14;15,99") and
// PartyXpCutoffGapPercent ("100;30;0"). Mobius indexes the percents by the
// gap, so the two lists must have the same length.
func ParseCutoffGaps(gaps, percents string) ([][2]int, []int, error) {
	var g [][2]int
	for part := range strings.SplitSeq(gaps, ";") {
		lo, hi, ok := strings.Cut(part, ",")
		if !ok {
			return nil, nil, fmt.Errorf("social: party exp gap %q is not \"min,max\"", part)
		}
		a, err1 := strconv.Atoi(strings.TrimSpace(lo))
		b, err2 := strconv.Atoi(strings.TrimSpace(hi))
		if err1 != nil || err2 != nil {
			return nil, nil, fmt.Errorf("social: party exp gap %q is not two numbers", part)
		}
		g = append(g, [2]int{a, b})
	}
	var p []int
	for part := range strings.SplitSeq(percents, ";") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, nil, fmt.Errorf("social: party exp gap percent %q is not a number", part)
		}
		p = append(p, n)
	}
	if len(g) != len(p) {
		return nil, nil, fmt.Errorf("social: %d party exp gaps but %d gap percents", len(g), len(p))
	}
	return g, p, nil
}

// ExpMember is a rewarded party member as distributeXpAndSp sees it.
// Attackable.calculateRewards passes the party's living members that stand
// within AltPartyRange of the monster (3D distance < range), whether or not
// they hit it, in party order.
type ExpMember struct {
	// Level is the member's level.
	Level int
	// SameInstance is true when the member is in the monster's instance.
	// Outside instances (M11) it is always true.
	SameInstance bool
	// Dead members get nothing (they may have died since the list was
	// made), but still count for the cutoff and the bonus.
	Dead bool
	// ServitorPenalty is the exp multiplier of the member's servitor
	// (Servitor.getExpMultiplier), or 0 without a servitor (no penalty).
	ServitorPenalty float32
}

// PartyShare is one member's part of a party kill before the member's own
// stats.
type PartyShare struct {
	// Get is true when the member is rewarded: valid and alive. Mobius
	// calls addExpAndSp(0, 0) for living members cut off, which adds
	// nothing.
	Get bool
	// Exp and SP are xpReward × preCalculation and spReward ×
	// preCalculation, the values Mobius passes through the member's
	// EXPSP_RATE stat.
	Exp, SP float64
}

// MemberExp is what a member's PlayerStat.addExpAndSp receives.
type MemberExp struct {
	// Get is true when addExpAndSp is called with these values.
	Get bool
	// Exp and SP go into addExpAndSp (which applies the exp/SP bonus
	// stats and rounds them).
	Exp, SP float64
}

// PartyDamageMul is Attackable.calculateRewards' partyMul: the party's part
// of the damage when others hit the monster too, else 1. Mobius multiplies
// the party's exp and SP by it after calculateExpAndSp has already given
// them the share partyDmg / totalDamage, so a party that did half the
// damage gets a quarter; kept as in Mobius.
func PartyDamageMul(partyDamage, totalDamage int64) float64 {
	if partyDamage < totalDamage {
		return float64(partyDamage) / float64(totalDamage)
	}
	return 1
}

// TopLevel is the party level calculateRewards uses for the kill: the
// highest level among the rewarded members, or 0 for none.
func TopLevel(members []ExpMember) int {
	top := 0
	for _, m := range members {
		top = max(top, m.Level)
	}
	return top
}

// Valid is Party.getValidMembers: which of members share the kill.
func (c PartyExp) Valid(members []ExpMember, topLevel int) []bool {
	valid := make([]bool, len(members))
	switch c.Method {
	case CutoffLevel:
		for i, m := range members {
			valid[i] = m.SameInstance && topLevel-m.Level <= c.CutoffLevel
		}
	case CutoffPercentage:
		sum := 0
		for _, m := range members {
			if m.SameInstance {
				sum += m.Level * m.Level
			}
		}
		for i, m := range members {
			sq := m.Level * m.Level
			valid[i] = m.SameInstance && float64(sq*100) >= float64(float64(sum)*c.CutoffPercent)
		}
	case CutoffAuto:
		if len(members) < 2 {
			// Mobius returns the rewarded list itself, instance or not.
			for i := range valid {
				valid[i] = true
			}
			break
		}
		sum := 0
		for _, m := range members {
			sum += m.Level * m.Level // every member, any instance
		}
		limit := sum / (len(members) * len(members))
		for i, m := range members {
			valid[i] = m.SameInstance && m.Level*m.Level >= limit
		}
	default: // CutoffHighFive, CutoffNone
		for i, m := range members {
			valid[i] = m.SameInstance
		}
	}
	return valid
}

// Bonus is Party.getBaseExpSpBonus times the party rate (getExpBonus,
// getSpBonus): the multiplier for n valid members. A lone member gets 1
// without the rate.
func Bonus(n int, rate float64) float64 {
	i := n - 1
	if i < 1 {
		return 1
	}
	i = min(i, len(partyBonus)-1)
	return float64(partyBonus[i] * f32(rate))
}

// Shares is Party.distributeXpAndSp up to the member stats: exp and sp are
// the party's reward (calculateExpAndSp for the party level, times
// PartyDamageMul), topLevel is TopLevel(members). The valid members' count
// picks the bonus; each valid living member gets bonus × reward ×
// level² / Σ valid level², times its servitor penalty.
func (c PartyExp) Shares(members []ExpMember, topLevel int, exp, sp float64) []PartyShare {
	valid := c.Valid(members, topLevel)
	n, sqSum := 0, 0
	for i, m := range members {
		if valid[i] {
			n++
			sqSum += m.Level * m.Level
		}
	}
	xpReward := float64(exp * Bonus(n, c.RateXP))
	spReward := float64(sp * Bonus(n, c.RateSP))
	shares := make([]PartyShare, len(members))
	for i, m := range members {
		if m.Dead || !valid[i] {
			continue
		}
		penalty := float32(1)
		if m.ServitorPenalty != 0 {
			penalty = m.ServitorPenalty
		}
		pre := float64(float64(m.Level*m.Level)/float64(sqSum)) * float64(penalty)
		shares[i] = PartyShare{
			Get: true,
			Exp: float64(xpReward * pre),
			SP:  float64(spReward * pre),
		}
	}
	return shares
}

// Final is the rest of distributeXpAndSp for one rewarded member
// (calculateExpSpPartyCutoff): ratedExp and ratedSP are the member's share
// after its EXPSP_RATE stat. The exp is Math.round-ed; with the highfive
// method both are scaled by the member's gap percent. ok is false when
// highfive finds no gap for the member: Mobius then adds nothing.
// Mobius's class balance multipliers (1 by default) and premium rates (off)
// are left out.
func (c PartyExp) Final(level, topLevel int, ratedExp, ratedSP float64) (exp, sp float64, ok bool) {
	addExp := float64(javaRound(ratedExp))
	if c.Method != CutoffHighFive {
		return addExp, ratedSP, true
	}
	diff := topLevel - level
	for i, g := range c.Gaps {
		if diff >= g[0] && diff <= g[1] {
			p := float64(c.GapPercents[i])
			return float64(addExp*p) / 100, float64(ratedSP*p) / 100, true
		}
	}
	return 0, 0, false
}

// Distribute is Shares followed by Final for every member. rate is the
// member's EXPSP_RATE stat (Creature.calcStat(Stat.EXPSP_RATE, v)) for
// member i; nil applies none. The result is aligned with members.
func (c PartyExp) Distribute(members []ExpMember, topLevel int, exp, sp float64, rate func(i int, v float64) float64) []MemberExp {
	shares := c.Shares(members, topLevel, exp, sp)
	out := make([]MemberExp, len(members))
	for i, s := range shares {
		if !s.Get {
			continue
		}
		e, p := s.Exp, s.SP
		if rate != nil {
			e, p = rate(i, e), rate(i, p)
		}
		fe, fp, ok := c.Final(members[i].Level, topLevel, e, p)
		out[i] = MemberExp{Get: ok, Exp: fe, SP: fp}
	}
	return out
}
