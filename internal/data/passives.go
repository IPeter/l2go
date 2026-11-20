package data

// Passive skill data (Plan 09 decision F10): the parts of the passive
// skills that change a creature's stats without the skill system, loaded
// with LoadPassives. It holds the passive skills NPC templates name (race,
// HP and attack multipliers, weaknesses and resistances), Death Penalty
// (5076) and Weight Penalty (4270). Per level it keeps the stat functions of
// the Buff effects and the DefenceTrait and AttackTrait values; other
// effects are left out and named per skill.

import (
	"cmp"
	"fmt"
	"io/fs"
	"math"
	"os"
	"slices"
)

// PassivesFile is the path of the passive skill file, relative to the data
// folder.
const PassivesFile = "skills/passives.json"

// Skills the server applies itself.
const (
	// SkillWeightPenalty is applied at the weight penalty level (1-4).
	SkillWeightPenalty = 4270
	// SkillDeathPenalty is applied at the death penalty level (1-15).
	SkillDeathPenalty = 5076
)

// PassiveSkill is a passive skill's stat changes, per level.
type PassiveSkill struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Levels are levels 1 to len(Levels), in order.
	Levels []PassiveLevel `json:"levels"`
	// Skipped names the effects of the skill this file leaves out (an
	// effect name, or "Buff with a conditional function"), sorted; the
	// skill system (M7) handles them.
	Skipped []string `json:"skipped,omitempty"`
}

// PassiveLevel is what one level of a passive skill applies while the
// creature has it, as Mobius applies a passive's effects in addSkill.
type PassiveLevel struct {
	// Funcs are the stat functions of the level's Buff effects, in XML
	// order.
	Funcs []PassiveFunc `json:"funcs,omitempty"`
	// DefenceTraits are the DefenceTrait values, sorted by trait; each
	// applies on its own (a trait two effects name is listed twice). Mobius
	// leaves out a value of 0, as this file does.
	DefenceTraits []PassiveTrait `json:"defenceTraits,omitempty"`
	// AttackTraits are the AttackTrait values, likewise.
	AttackTraits []PassiveTrait `json:"attackTraits,omitempty"`
}

// PassiveFunc is a stat function: Op applies Value to Stat at Order.
type PassiveFunc struct {
	Stat  Stat    `json:"stat"`
	Op    StatOp  `json:"op"`
	Order int     `json:"order"`
	Value float64 `json:"value"`
}

// PassiveTrait is a trait value as the skill XML writes it (a Java float),
// a percent: -11 makes a creature 11 % weaker against the trait. The 0.85
// and 1 entries of skill 4416, written as factors, are read as percents
// too, as Mobius reads them (see Multiplier).
type PassiveTrait struct {
	Trait TraitType `json:"trait"`
	Value float32   `json:"value"`
}

// Multiplier returns the factor Mobius keeps for the trait: (Value + 100) /
// 100 in float arithmetic. So -11 is 0.89 and the neutral 1 of skill 4416
// is 1.01, as in Mobius.
func (t PassiveTrait) Multiplier() float32 { return (t.Value + 100) / 100 }

// Invulnerable reports whether a defence trait makes the creature immune
// to the trait instead of multiplying (Mobius DefenceTrait: a multiplier of
// 2 or more).
func (t PassiveTrait) Invulnerable() bool { return t.Multiplier() >= 2 }

// Passives is the checked passive skill data. It must not be modified after
// NewPassives or LoadPassives returns, and is then safe for concurrent use.
type Passives struct {
	// Skills are sorted by ID.
	Skills []PassiveSkill

	byID map[int]int // skill ID -> index in Skills
}

// LoadPassives reads and checks the passive skill file in dir.
func LoadPassives(dir string) (*Passives, error) {
	p, err := LoadPassivesFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return p, nil
}

// LoadPassivesFS reads and checks the passive skill file in fsys.
func LoadPassivesFS(fsys fs.FS) (*Passives, error) {
	var skills []PassiveSkill
	if err := readJSON(fsys, PassivesFile, &skills); err != nil {
		return nil, err
	}
	return NewPassives(skills)
}

// NewPassives checks the skills and indexes them. The slice is used as it
// is. Skill IDs are positive, unique and sorted, every skill has levels,
// and every stat, operation, trait and value is valid.
func NewPassives(skills []PassiveSkill) (*Passives, error) {
	ps := &Passives{Skills: skills, byID: make(map[int]int, len(skills))}
	var p problems
	if len(skills) == 0 {
		p.addf("%s: no skills", PassivesFile)
	}
	for i := range skills {
		s := &skills[i]
		where := fmt.Sprintf("%s: skill %d (%q)", PassivesFile, s.ID, s.Name)
		if _, dup := ps.byID[s.ID]; dup {
			p.addf("%s: duplicate skill ID", where)
			continue
		}
		ps.byID[s.ID] = i
		if i > 0 && skills[i-1].ID > s.ID {
			p.addf("%s: skills are not sorted by ID", where)
		}
		if s.ID <= 0 || len(s.Levels) == 0 {
			p.addf("%s: the ID must be positive and the skill must have levels", where)
		}
		for j, l := range s.Levels {
			for _, f := range l.Funcs {
				if !f.Stat.Valid() || !f.Op.Valid() || math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
					p.addf("%s: level %d: invalid function %+v", where, j+1, f)
				}
			}
			for _, traits := range [][]PassiveTrait{l.DefenceTraits, l.AttackTraits} {
				for k, t := range traits {
					v := float64(t.Value)
					if !t.Trait.Valid() || math.IsNaN(v) || math.IsInf(v, 0) {
						p.addf("%s: level %d: invalid trait %+v", where, j+1, t)
					}
					if k > 0 && cmp.Compare(traits[k-1].Trait, t.Trait) > 0 {
						p.addf("%s: level %d: traits are not sorted", where, j+1)
					}
				}
			}
		}
		if !slices.IsSorted(s.Skipped) {
			p.addf("%s: skipped effects are not sorted", where)
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return ps, nil
}

// Skill returns a passive skill.
func (ps *Passives) Skill(id int) (*PassiveSkill, bool) {
	i, ok := ps.byID[id]
	if !ok {
		return nil, false
	}
	return &ps.Skills[i], true
}

// Level returns one level of a passive skill as Mobius SkillData.getSkill
// finds it: a level above the highest gives the highest. The bool is false
// if the skill is not in the data or the level is below 1.
func (ps *Passives) Level(id, level int) (*PassiveLevel, bool) {
	s, ok := ps.Skill(id)
	if !ok || level < 1 {
		return nil, false
	}
	return &s.Levels[min(level, len(s.Levels))-1], true
}
