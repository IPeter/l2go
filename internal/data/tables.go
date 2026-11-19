package data

import "fmt"

// BaseStat is a base attribute, Mobius mechanics/stats/BaseStat.
type BaseStat int

// The base attributes.
const (
	BaseSTR BaseStat = iota
	BaseINT
	BaseDEX
	BaseWIT
	BaseCON
	BaseMEN
)

// String returns the attribute name, such as STR.
func (s BaseStat) String() string {
	switch s {
	case BaseSTR:
		return "STR"
	case BaseINT:
		return "INT"
	case BaseDEX:
		return "DEX"
	case BaseWIT:
		return "WIT"
	case BaseCON:
		return "CON"
	case BaseMEN:
		return "MEN"
	}
	return fmt.Sprintf("BaseStat(%d)", int(s))
}

// StatBonusValues is the number of attribute values a stat bonus table
// covers, 0 to 99 (Mobius BaseStat.MAX_STAT_VALUE).
const StatBonusValues = 100

// StatBonusTable is Mobius statBonus.xml: the bonus multiplier of each
// base attribute value. Index i holds the bonus of value i.
type StatBonusTable struct {
	STR []float64 `json:"STR"`
	INT []float64 `json:"INT"`
	DEX []float64 `json:"DEX"`
	WIT []float64 `json:"WIT"`
	CON []float64 `json:"CON"`
	MEN []float64 `json:"MEN"`
}

// column returns the table of attribute s.
func (t *StatBonusTable) column(s BaseStat) []float64 {
	switch s {
	case BaseSTR:
		return t.STR
	case BaseINT:
		return t.INT
	case BaseDEX:
		return t.DEX
	case BaseWIT:
		return t.WIT
	case BaseCON:
		return t.CON
	case BaseMEN:
		return t.MEN
	}
	return nil
}

// Bonus returns the bonus of attribute s at value; false if the value is
// out of the table (Mobius would fail on it too).
func (t *StatBonusTable) Bonus(s BaseStat, value int) (float64, bool) {
	col := t.column(s)
	if value < 0 || value >= len(col) {
		return 0, false
	}
	return col[value], true
}

// StatBonus returns the bonus of attribute s at value, Mobius
// BaseStat.calcBonus; false if the value is out of the table.
func (d *Data) StatBonus(s BaseStat, value int) (float64, bool) {
	return d.StatBonuses.Bonus(s, value)
}

func (t *StatBonusTable) check(p *problems) {
	for s := BaseSTR; s <= BaseMEN; s++ {
		col := t.column(s)
		if len(col) != StatBonusValues {
			p.addf("%s: %s: %d values, want %d", StatBonusFile, s, len(col), StatBonusValues)
			continue
		}
		for v, bonus := range col {
			if bonus <= 0 {
				p.addf("%s: %s %d: bonus %v must be positive", StatBonusFile, s, v, bonus)
			}
		}
	}
}

// ExperienceTable is Mobius experience.xml.
type ExperienceTable struct {
	// MaxLevel and MaxPetLevel are the table's maxLevel and maxPetLevel.
	// Mobius adds one and caps the result with its MaximumPlayerLevel
	// setting (80), so by default players stop at level 79 there.
	MaxLevel    int `json:"maxLevel"`
	MaxPetLevel int `json:"maxPetLevel"`
	// Exp holds the total experience needed for each level: index 0 is
	// level 1 (0 experience). The table goes one level past MaxLevel.
	Exp []int64 `json:"exp"`
}

// ForLevel returns the total experience needed to reach level.
func (t *ExperienceTable) ForLevel(level int) (int64, bool) {
	if level < 1 || level > len(t.Exp) {
		return 0, false
	}
	return t.Exp[level-1], true
}

// ExpForLevel returns the total experience needed to reach level, Mobius
// ExperienceData.getExpForLevel without its configuration cap.
func (d *Data) ExpForLevel(level int) (int64, bool) { return d.Experience.ForLevel(level) }

func (t *ExperienceTable) check(p *problems) {
	switch {
	case len(t.Exp) == 0:
		p.addf("%s: no levels", ExperienceFile)
		return
	case t.Exp[0] != 0:
		p.addf("%s: level 1 needs %d experience, want 0", ExperienceFile, t.Exp[0])
	}
	for i := 1; i < len(t.Exp); i++ {
		if t.Exp[i] <= t.Exp[i-1] {
			p.addf("%s: level %d needs %d experience, not more than level %d (%d)", ExperienceFile, i+1, t.Exp[i], i, t.Exp[i-1])
		}
	}
	if t.MaxLevel < 1 || t.MaxLevel >= len(t.Exp) {
		p.addf("%s: maxLevel %d: the table must go from level 1 to one past it (it has %d levels)", ExperienceFile, t.MaxLevel, len(t.Exp))
	}
	if t.MaxPetLevel < 1 || t.MaxPetLevel >= len(t.Exp) {
		p.addf("%s: maxPetLevel %d: the table must go from level 1 to one past it (it has %d levels)", ExperienceFile, t.MaxPetLevel, len(t.Exp))
	}
}

// LevelTable is a value per level: index 0 is level 1. It holds Mobius
// experienceLoss.xml (the percent of experience lost on death) and
// karmaLoss.xml (the karma loss modifier).
type LevelTable []float64

// At returns the value of level.
func (t LevelTable) At(level int) (float64, bool) {
	if level < 1 || level > len(t) {
		return 0, false
	}
	return t[level-1], true
}

func (t LevelTable) check(p *problems, file string) {
	if len(t) == 0 {
		p.addf("%s: no levels", file)
	}
	for i, v := range t {
		if v < 0 {
			p.addf("%s: level %d: value %v must not be negative", file, i+1, v)
		}
	}
}

// HitConditionBonus is Mobius hitConditionBonus.xml: the accuracy bonus,
// in percent, of attacking from a position or in a condition.
type HitConditionBonus struct {
	Front int `json:"front"`
	Side  int `json:"side"`
	Back  int `json:"back"`
	High  int `json:"high"`
	Low   int `json:"low"`
	Dark  int `json:"dark"`
	Rain  int `json:"rain"`
}
