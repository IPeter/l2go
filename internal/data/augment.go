package data

// Augmentation data (Plan 14, M11d), loaded with LoadAugment:
//
//   - augment/refine.json (AugmentRefine): the tables Mobius keeps in Java
//     (AbstractRefinePacket: the life stones, the gemstones and their
//     counts; RequestRefineCancel: the removal fees) and the NPCs whose
//     dialogs offer augmentation (decision X10). cmd/datagen writes them
//     from its input file, which names the Java sources, and checks them
//     against those sources.
//   - augment/options.json ([]AugmentOption, one per line): the options of
//     dist/game/data/stats/augmentation/options (Mobius OptionData).
//   - augment/chances.json ([]AugmentChanceGroup, one per line): the
//     retail chances of retailchances.xml (Mobius AugmentationData with
//     RetailLikeAugmentation), in file order.

import (
	"cmp"
	"fmt"
	"io/fs"
	"math"
	"os"
	"slices"
)

// Paths of the augmentation files, relative to the data folder.
const (
	AugmentRefineFile  = "augment/refine.json"
	AugmentOptionsFile = "augment/options.json"
	AugmentChancesFile = "augment/chances.json"
)

// The life stone grades (Mobius AbstractRefinePacket.GRADE_*).
const (
	LifeStoneGradeNone = 0
	LifeStoneGradeMid  = 1
	LifeStoneGradeHigh = 2
	LifeStoneGradeTop  = 3
)

// The weapon kinds of the retail chances (retailchances.xml <weapon type>).
const (
	AugmentWeaponWarrior = "warrior"
	AugmentWeaponMage    = "mage"
)

// The kinds of option trigger skills (Mobius OptionSkillType).
const (
	AugmentTriggerAttack   = "attack"
	AugmentTriggerMagic    = "magic"
	AugmentTriggerCritical = "critical"
)

// MaxAugmentOptionID is the highest option ID an augmentation can carry:
// each half of the augmentation int holds one.
const MaxAugmentOptionID = math.MaxUint16

// AugmentRefine holds the augmentation tables Mobius keeps in Java.
type AugmentRefine struct {
	// Source names the Java files the tables come from.
	Source []string `json:"source"`
	// LifeStones are the life stones, sorted by item ID.
	LifeStones []LifeStone `json:"lifeStones"`
	// Gemstones are the gemstone item and count per item grade, C to S
	// (AbstractRefinePacket.getGemStoneId and getGemStoneCount; the count
	// does not depend on the life stone grade).
	Gemstones []AugmentGemstone `json:"gemstones"`
	// CancelFees are the fees of removing an augmentation, per item grade
	// and crystal count (RequestRefineCancel), sorted by grade, then by
	// threshold with the open-ended entry (Below 0) last.
	CancelFees []AugmentCancelFee `json:"cancelFees"`
	// Npcs are the NPC templates whose dialogs open the augmentation
	// windows, sorted: the refine and cancel packets need one of them
	// nearby (decision X10).
	Npcs []int `json:"npcs"`
}

// LifeStone is a life stone (AbstractRefinePacket.LifeStone).
type LifeStone struct {
	ItemID int `json:"itemId"`
	// Grade is LifeStoneGradeNone to LifeStoneGradeTop.
	Grade int `json:"grade"`
	// Level is the life stone level, 0-9.
	Level int `json:"level"`
	// PlayerLevel is the level a player needs to use it.
	PlayerLevel int `json:"playerLevel"`
}

// AugmentGemstone is the gemstone an item grade needs.
type AugmentGemstone struct {
	Grade  CrystalType `json:"grade"`
	ItemID int         `json:"itemId"`
	Count  int         `json:"count"`
}

// AugmentCancelFee is the fee of removing an augmentation from an item of
// the grade with fewer than Below crystals; Below 0 is any count.
type AugmentCancelFee struct {
	Grade CrystalType `json:"grade"`
	Below int         `json:"below,omitempty"`
	Fee   int64       `json:"fee"`
}

// AugmentOption is an augmentation option (Mobius Options): stat functions
// and skills a worn augmented item gives.
type AugmentOption struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
	// Funcs are the stat functions of <stats>, in XML order, with the
	// default order of their operation when the XML names none.
	Funcs []AugmentFunc `json:"funcs,omitempty"`
	// ActiveSkill and PassiveSkill are learned while the item is worn.
	ActiveSkill  *SkillRef `json:"activeSkill,omitempty"`
	PassiveSkill *SkillRef `json:"passiveSkill,omitempty"`
	// Triggers are the skills cast with a chance on a hit, a critical hit
	// or a skill (attack_skill, critical_skill, magic_skill), in XML order.
	Triggers []AugmentTrigger `json:"triggers,omitempty"`
}

// AugmentFunc is a stat function of an option: Op applies Value to Stat at
// Order.
type AugmentFunc struct {
	Stat  Stat    `json:"stat"`
	Op    StatOp  `json:"op"`
	Order int     `json:"order"`
	Value float64 `json:"value"`
}

// AugmentTrigger is a skill an option casts with a chance (Mobius
// OptionSkillHolder).
type AugmentTrigger struct {
	// Type is AugmentTriggerAttack, AugmentTriggerMagic or
	// AugmentTriggerCritical.
	Type  string `json:"type"`
	ID    int    `json:"id"`
	Level int    `json:"level"`
	// Chance is in percent (Rnd.get(100) < chance).
	Chance float64 `json:"chance"`
}

// AugmentChanceGroup is one <category> of retailchances.xml: the options a
// life stone gives a weapon kind for one half of the augmentation
// (variation 1 the low half, 2 the high half), with the category's
// probability.
type AugmentChanceGroup struct {
	// Weapon is AugmentWeaponWarrior or AugmentWeaponMage.
	Weapon    string `json:"weapon"`
	Stone     int    `json:"stone"`
	Variation int    `json:"variation"`
	// Category is the <category probability>: 100 for variation 1; 55, 35,
	// 7 or 3 for the grades of variation 2.
	Category int             `json:"category"`
	Augments []AugmentChance `json:"augments"`
}

// AugmentChance is an option with its chance in percent, a Java float as
// Mobius parses it.
type AugmentChance struct {
	ID     int     `json:"id"`
	Chance float32 `json:"chance"`
}

// AugmentData is the content of the augmentation files.
type AugmentData struct {
	Refine  AugmentRefine
	Options []AugmentOption
	Chances []AugmentChanceGroup
}

// Augment is the checked augmentation data. It must not be modified after
// NewAugment or LoadAugment returns, and is then safe for concurrent use.
type Augment struct {
	AugmentData

	options map[int]int // option ID -> index in Options
	stones  map[int]int // item ID -> index in Refine.LifeStones
}

// LoadAugment reads and checks the augmentation files in dir.
func LoadAugment(dir string) (*Augment, error) {
	a, err := LoadAugmentFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return a, nil
}

// LoadAugmentFS reads and checks the augmentation files in fsys.
func LoadAugmentFS(fsys fs.FS) (*Augment, error) {
	var d AugmentData
	if err := readJSON(fsys, AugmentRefineFile, &d.Refine); err != nil {
		return nil, err
	}
	if err := readJSON(fsys, AugmentOptionsFile, &d.Options); err != nil {
		return nil, err
	}
	if err := readJSON(fsys, AugmentChancesFile, &d.Chances); err != nil {
		return nil, err
	}
	return NewAugment(d)
}

// NewAugment checks the augmentation data and indexes it. The slices are
// used as they are.
func NewAugment(d AugmentData) (*Augment, error) {
	a := &Augment{AugmentData: d, options: make(map[int]int, len(d.Options)), stones: make(map[int]int, len(d.Refine.LifeStones))}
	var p problems
	a.checkRefine(&p)
	a.checkOptions(&p)
	a.checkChances(&p)
	if err := p.err(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *Augment) checkRefine(p *problems) {
	r := &a.Refine
	if len(r.Source) == 0 {
		p.addf("%s: no source", AugmentRefineFile)
	}
	if len(r.LifeStones) == 0 {
		p.addf("%s: no life stones", AugmentRefineFile)
	}
	for i, ls := range r.LifeStones {
		where := fmt.Sprintf("%s: life stone %d", AugmentRefineFile, ls.ItemID)
		if _, dup := a.stones[ls.ItemID]; dup {
			p.addf("%s: listed twice", where)
			continue
		}
		a.stones[ls.ItemID] = i
		if i > 0 && r.LifeStones[i-1].ItemID > ls.ItemID {
			p.addf("%s: life stones are not sorted by item ID", where)
		}
		if ls.ItemID <= 0 || ls.Grade < LifeStoneGradeNone || ls.Grade > LifeStoneGradeTop || ls.Level < 0 || ls.Level > 9 || ls.PlayerLevel <= 0 {
			p.addf("%s: invalid %+v", where, ls)
		}
	}
	seen := map[CrystalType]bool{}
	for _, g := range r.Gemstones {
		if g.Grade < CrystalC || !g.Grade.Valid() || seen[g.Grade] || g.ItemID <= 0 || g.Count <= 0 {
			p.addf("%s: invalid gemstone %+v", AugmentRefineFile, g)
		}
		seen[g.Grade] = true
	}
	for _, grade := range []CrystalType{CrystalC, CrystalB, CrystalA, CrystalS} {
		if !seen[grade] {
			p.addf("%s: no gemstone for grade %s", AugmentRefineFile, grade)
		}
		var fees []AugmentCancelFee
		for _, f := range r.CancelFees {
			if f.Grade == grade {
				fees = append(fees, f)
			}
		}
		if len(fees) == 0 || fees[len(fees)-1].Below != 0 {
			p.addf("%s: grade %s needs cancel fees ending with one without a threshold", AugmentRefineFile, grade)
		}
		for i, f := range fees {
			if f.Fee <= 0 || f.Below < 0 || (i < len(fees)-1 && (f.Below == 0 || (i > 0 && fees[i-1].Below >= f.Below))) {
				p.addf("%s: invalid cancel fee %+v", AugmentRefineFile, f)
			}
		}
	}
	for _, f := range r.CancelFees {
		if f.Grade < CrystalC || !f.Grade.Valid() {
			p.addf("%s: cancel fee of grade %s", AugmentRefineFile, f.Grade)
		}
	}
	if !slices.IsSortedFunc(r.CancelFees, func(x, y AugmentCancelFee) int { return cmp.Compare(x.Grade, y.Grade) }) {
		p.addf("%s: cancel fees are not sorted by grade", AugmentRefineFile)
	}
	if !slices.IsSorted(r.Npcs) || len(slices.Compact(slices.Clone(r.Npcs))) != len(r.Npcs) {
		p.addf("%s: NPCs are not sorted and unique", AugmentRefineFile)
	}
}

func (a *Augment) checkOptions(p *problems) {
	if len(a.Options) == 0 {
		p.addf("%s: no options", AugmentOptionsFile)
	}
	for i := range a.Options {
		o := &a.Options[i]
		where := fmt.Sprintf("%s: option %d", AugmentOptionsFile, o.ID)
		if _, dup := a.options[o.ID]; dup {
			p.addf("%s: duplicate ID", where)
			continue
		}
		a.options[o.ID] = i
		if i > 0 && a.Options[i-1].ID > o.ID {
			p.addf("%s: options are not sorted by ID", where)
		}
		if o.ID < 0 || o.ID > MaxAugmentOptionID {
			p.addf("%s: ID out of range", where)
		}
		for _, f := range o.Funcs {
			if !f.Stat.Valid() || !f.Op.Valid() || f.Order < 0 || math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
				p.addf("%s: invalid function %+v", where, f)
			}
		}
		for _, sk := range []*SkillRef{o.ActiveSkill, o.PassiveSkill} {
			if sk != nil && (sk.ID <= 0 || sk.Level <= 0) {
				p.addf("%s: invalid skill %+v", where, *sk)
			}
		}
		for _, t := range o.Triggers {
			switch t.Type {
			case AugmentTriggerAttack, AugmentTriggerMagic, AugmentTriggerCritical:
			default:
				p.addf("%s: unknown trigger type %q", where, t.Type)
			}
			if t.ID <= 0 || t.Level <= 0 || math.IsNaN(t.Chance) || math.IsInf(t.Chance, 0) {
				p.addf("%s: invalid trigger %+v", where, t)
			}
		}
	}
}

func (a *Augment) checkChances(p *problems) {
	for _, g := range a.Chances {
		where := fmt.Sprintf("%s: %s stone %d variation %d category %d", AugmentChancesFile, g.Weapon, g.Stone, g.Variation, g.Category)
		if g.Weapon != AugmentWeaponWarrior && g.Weapon != AugmentWeaponMage {
			p.addf("%s: unknown weapon kind", where)
		}
		if _, ok := a.stones[g.Stone]; !ok {
			p.addf("%s: not a life stone", where)
		}
		if g.Variation <= 0 || g.Category <= 0 || len(g.Augments) == 0 {
			p.addf("%s: invalid group", where)
		}
		for _, c := range g.Augments {
			v := float64(c.Chance)
			if _, ok := a.options[c.ID]; !ok || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
				p.addf("%s: invalid augment %+v (an unknown option or a bad chance)", where, c)
			}
		}
	}
}

// Option returns an option.
func (a *Augment) Option(id int) (*AugmentOption, bool) {
	i, ok := a.options[id]
	if !ok {
		return nil, false
	}
	return &a.Options[i], true
}

// LifeStone returns the life stone of an item template.
func (a *Augment) LifeStone(itemID int) (LifeStone, bool) {
	i, ok := a.stones[itemID]
	if !ok {
		return LifeStone{}, false
	}
	return a.Refine.LifeStones[i], true
}
