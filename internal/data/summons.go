package data

// Summon data: the pets, the skills of pets and servitors, and the cubics,
// which cmd/datagen converts from Mobius stats/pets/*.xml (read by
// data/xml/PetDataTable.java into data/holders/PetData.java and
// PetLevelData.java), PetSkillData.xml (data/xml/PetSkillData.java) and
// stats/cubics/*.xml (data/xml/CubicData.java into
// entity/actor/templates/CubicTemplate.java and entity/cubic/*.java). They
// are loaded with LoadSummons.

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"slices"
)

// Paths of the summon data files, relative to the data folder.
const (
	PetsFile      = "pets/pets.json"
	PetSkillsFile = "pets/skills.json"
	CubicsFile    = "cubics/cubics.json"
)

// PetData is one pet of stats/pets: the pet's NPC, the item (collar) that
// summons it, its food and its stats per level (Mobius PetData).
type PetData struct {
	NpcID int `json:"npcId"`
	// ItemID is the item that summons the pet; -1 for none (the wyvern).
	ItemID int `json:"itemId"`
	// Index is the pet's index attribute, which Mobius ignores.
	Index int `json:"index"`
	// Food are the items the pet eats (the "food" set; Mobius keeps them
	// in a set), in file order without repeats.
	Food []int `json:"food"`
	// HungryLimit is hungry_limit (Mobius default 1): the meal percent
	// below which the pet is hungry.
	HungryLimit int `json:"hungryLimit"`
	// Load is the pet's weight limit (Mobius default 20000).
	Load int `json:"load"`
	// SyncLevel is sync_level 1: the pet's level follows its owner's.
	SyncLevel bool `json:"syncLevel,omitempty"`
	// Skills are the <skills> entries (Mobius PetData.getAvailableLevel
	// uses them), in file order.
	Skills []PetDataSkill `json:"skills,omitempty"`
	// Levels are the stats per level, sorted by level, contiguous.
	Levels []PetLevel `json:"levels"`
}

// PetDataSkill is a skill of a pet's <skills> element (Mobius
// PetData.PetSkillLearn). A SkillLevel of 0 means "by pet level".
type PetDataSkill struct {
	SkillID    int `json:"skillId"`
	SkillLevel int `json:"skillLevel"`
	MinLevel   int `json:"minLevel"`
}

// PetLevel holds a pet's stats at one level (Mobius PetLevelData). The
// stats Mobius keeps as Java floats hold the float32-rounded value.
type PetLevel struct {
	Level int `json:"level"`
	// MaxMeal is max_meal, the full food gauge.
	MaxMeal int `json:"maxMeal"`
	// Exp is the experience the level starts at ("exp", read as (long) of
	// a double).
	Exp int64 `json:"exp"`
	// OwnerExpTaken is get_exp_type: the percent of the owner's experience
	// the pet takes (-1 for the wyvern).
	OwnerExpTaken int `json:"ownerExpTaken"`
	// MealInBattle and MealInNormal are consume_meal_in_battle and
	// consume_meal_in_normal: the food used per tick.
	MealInBattle int     `json:"mealInBattle"`
	MealInNormal int     `json:"mealInNormal"`
	PAtk         float64 `json:"pAtk"`    // org_pattack
	PDef         float64 `json:"pDef"`    // org_pdefend
	MAtk         float64 `json:"mAtk"`    // org_mattack
	MDef         float64 `json:"mDef"`    // org_mdefend
	HP           float64 `json:"hp"`      // org_hp
	MP           float64 `json:"mp"`      // org_mp
	HPRegen      float64 `json:"hpRegen"` // org_hp_regen
	MPRegen      float64 `json:"mpRegen"` // org_mp_regen
	// SoulshotCount and SpiritshotCount are the shots used per attack or
	// cast.
	SoulshotCount   int `json:"soulshotCount"`
	SpiritshotCount int `json:"spiritshotCount"`
	// The speeds of the speed_on_ride set when the pet is ridden (0 when
	// absent).
	WalkOnRide     float64 `json:"walkOnRide,omitempty"`
	RunOnRide      float64 `json:"runOnRide,omitempty"`
	SlowSwimOnRide float64 `json:"slowSwimOnRide,omitempty"`
	FastSwimOnRide float64 `json:"fastSwimOnRide,omitempty"`
	SlowFlyOnRide  float64 `json:"slowFlyOnRide,omitempty"`
	FastFlyOnRide  float64 `json:"fastFlyOnRide,omitempty"`
}

// MinLevel returns the pet's lowest level.
func (p *PetData) MinLevel() int {
	if len(p.Levels) == 0 {
		return 0
	}
	return p.Levels[0].Level
}

// MaxLevel returns the pet's highest level.
func (p *PetData) MaxLevel() int {
	if len(p.Levels) == 0 {
		return 0
	}
	return p.Levels[len(p.Levels)-1].Level
}

// Level returns the stats of a level. A level above the highest gives the
// highest (as Mobius PetDataTable.getPetLevelData); one below the lowest
// is not found.
func (p *PetData) Level(lvl int) (*PetLevel, bool) {
	if len(p.Levels) == 0 {
		return nil, false
	}
	lvl = min(lvl, p.MaxLevel())
	i := lvl - p.MinLevel()
	if i < 0 {
		return nil, false
	}
	return &p.Levels[i], true
}

// PetSkill is one row of PetSkillData.xml: a skill of a pet or servitor
// NPC. A SkillLevel of 0 means "by pet level" (Mobius
// PetSkillData.getAvailableLevel).
type PetSkill struct {
	NpcID      int `json:"npcId"`
	SkillID    int `json:"skillId"`
	SkillLevel int `json:"skillLevel"`
}

// The cubic target types (Mobius CubicTargetType).
const (
	CubicTargetTarget  = "TARGET"
	CubicTargetBySkill = "BY_SKILL"
	CubicTargetHeal    = "HEAL"
	CubicTargetMaster  = "MASTER"
)

// The cubic skill selection modes (Mobius CubicSelectionMode).
const (
	CubicSelectRandom   = "RANDOM"
	CubicSelectWeighted = "WEIGHTED"
	CubicSelectOrdered  = "ORDERED"
)

// The kinds of cubic condition (CubicCondition.Kind), named like their
// XML elements.
const (
	// CubicCondHP is HpCondition: the target's HP percent is greater or
	// lesser than Percent.
	CubicCondHP = "hp"
	// CubicCondRange is RangeCondition: the owner is at most Range away
	// from the target (2D).
	CubicCondRange = "range"
	// CubicCondHealthPercent is HealthCondition: the target's HP percent is
	// strictly between Min and Max.
	CubicCondHealthPercent = "healthPercent"
)

// The comparisons of an hp condition (Mobius HpConditionType).
const (
	CubicHPGreater = "GREATER"
	CubicHPLesser  = "LESSER"
)

var (
	cubicTargetTypes    = []string{CubicTargetTarget, CubicTargetBySkill, CubicTargetHeal, CubicTargetMaster}
	cubicSelectionModes = []string{CubicSelectRandom, CubicSelectWeighted, CubicSelectOrdered}
)

// CubicTemplate is one <cubic> of stats/cubics: a cubic at one level
// (Mobius CubicTemplate).
type CubicTemplate struct {
	ID    int `json:"id"`
	Level int `json:"level"`
	Slot  int `json:"slot"`
	// Duration is how long the cubic stays, in seconds.
	Duration int `json:"duration"`
	// Delay is the time between its actions, in seconds.
	Delay int `json:"delay"`
	// MaxCount is how many times it acts; UseUp is 1 when it leaves after
	// that.
	MaxCount int `json:"maxCount"`
	UseUp    int `json:"useUp"`
	// Power is the cubic's P. Atk and M. Atk (as an int in Mobius; default
	// 0).
	Power float64 `json:"power"`
	// SelectionMode is how a skill is picked (CubicSelect*, default
	// RANDOM).
	SelectionMode string `json:"selectionMode"`
	// TargetType is how the target is picked (CubicTarget*, default
	// TARGET).
	TargetType string `json:"targetType"`
	// Conditions must all hold for the cubic to act.
	Conditions []CubicCondition `json:"conditions,omitempty"`
	Skills     []CubicSkill     `json:"skills"`
}

// CubicSkill is a skill a cubic uses (Mobius CubicSkill).
type CubicSkill struct {
	ID    int `json:"id"`
	Level int `json:"level"`
	// TriggerRate is the chance in percent to use it (default 100).
	TriggerRate int `json:"triggerRate"`
	// TargetType is the "target" attribute (CubicTarget*, default TARGET).
	TargetType string `json:"targetType"`
	// TargetDebuff: only on a target with a debuff that can be dispelled.
	TargetDebuff bool `json:"targetDebuff,omitempty"`
	Priority     bool `json:"priority,omitempty"`
	// Conditions must all hold for the skill to be used.
	Conditions []CubicCondition `json:"conditions,omitempty"`
}

// CubicCondition is one condition of a cubic or cubic skill. Kind says
// which fields count: CubicCondHP uses HPType and Percent, CubicCondRange
// Range, CubicCondHealthPercent Min and Max.
type CubicCondition struct {
	Kind    string `json:"kind"`
	HPType  string `json:"hpType,omitempty"`
	Percent int    `json:"percent,omitempty"`
	Range   int    `json:"range,omitempty"`
	Min     int    `json:"min,omitempty"`
	Max     int    `json:"max,omitempty"`
}

// SummonData is the content of the summon data files.
type SummonData struct {
	Pets      []PetData       // PetsFile, sorted by NPC ID
	PetSkills []PetSkill      // PetSkillsFile, in file order
	Cubics    []CubicTemplate // CubicsFile, sorted by ID and level
}

// Summons is the checked summon data. It must not be modified after
// NewSummons or LoadSummons returns, and is then safe for concurrent use;
// the pointers and slices its lookups return point into it. The PetSkills
// method hides the field of the same name; all rows are
// s.SummonData.PetSkills.
type Summons struct {
	SummonData

	petByItem map[int]int        // item ID -> index in Pets
	petSkills map[int][]PetSkill // NPC ID -> its rows of PetSkills
}

// LoadSummons reads and checks the summon data files in dir.
func LoadSummons(dir string) (*Summons, error) {
	s, err := LoadSummonsFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return s, nil
}

// LoadSummonsFS reads and checks the summon data files in fsys.
func LoadSummonsFS(fsys fs.FS) (*Summons, error) {
	var d SummonData
	for _, f := range []struct {
		name string
		v    any
	}{
		{PetsFile, &d.Pets},
		{PetSkillsFile, &d.PetSkills},
		{CubicsFile, &d.Cubics},
	} {
		if err := readJSON(fsys, f.name, f.v); err != nil {
			return nil, err
		}
	}
	return NewSummons(d)
}

// NewSummons checks the summon data. The slices are used as they are.
// Pets: NPC IDs positive, unique and sorted; item IDs positive and unique,
// or -1; food positive and not repeated; limits and stats not negative
// (OwnerExpTaken aside);
// levels from 1 up, ascending and contiguous. Pet skills: IDs positive,
// levels not negative. Cubics: (ID, level) positive, unique and sorted;
// known selection modes, target types and conditions.
func NewSummons(d SummonData) (*Summons, error) {
	var p problems
	s := &Summons{
		SummonData: d,
		petByItem:  make(map[int]int, len(d.Pets)),
		petSkills:  make(map[int][]PetSkill),
	}
	for i := range d.Pets {
		pet := &d.Pets[i]
		where := fmt.Sprintf("%s: pet %d", PetsFile, pet.NpcID)
		if pet.NpcID <= 0 {
			p.addf("%s: NPC ID must be positive", where)
		}
		if i > 0 && d.Pets[i-1].NpcID >= pet.NpcID {
			p.addf("%s: NPC IDs are not sorted and unique", where)
		}
		switch {
		case pet.ItemID == -1:
		case pet.ItemID <= 0:
			p.addf("%s: item ID %d must be positive or -1", where, pet.ItemID)
		default:
			if j, dup := s.petByItem[pet.ItemID]; dup {
				p.addf("%s: item %d also summons pet %d", where, pet.ItemID, d.Pets[j].NpcID)
			} else {
				s.petByItem[pet.ItemID] = i
			}
		}
		for j, f := range pet.Food {
			if f <= 0 || slices.Contains(pet.Food[:j], f) {
				p.addf("%s: food %d is not positive or repeated", where, f)
			}
		}
		if pet.HungryLimit < 0 || pet.Load < 0 {
			p.addf("%s: negative hungry limit or load", where)
		}
		for _, sk := range pet.Skills {
			if sk.SkillID <= 0 || sk.SkillLevel < 0 || sk.MinLevel < 0 {
				p.addf("%s: skill %d level %d min level %d: bad ID or negative level", where, sk.SkillID, sk.SkillLevel, sk.MinLevel)
			}
		}
		if len(pet.Levels) == 0 {
			p.addf("%s: no levels", where)
		}
		for j, l := range pet.Levels {
			if l.Level <= 0 {
				p.addf("%s: level %d must be positive", where, l.Level)
			}
			if j > 0 && l.Level != pet.Levels[j-1].Level+1 {
				p.addf("%s: level %d does not follow level %d", where, l.Level, pet.Levels[j-1].Level)
			}
			if l.MaxMeal < 0 || l.Exp < 0 || l.MealInBattle < 0 || l.MealInNormal < 0 ||
				l.SoulshotCount < 0 || l.SpiritshotCount < 0 {
				p.addf("%s: level %d: negative meal, experience or shot count", where, l.Level)
			}
			if min(l.PAtk, l.PDef, l.MAtk, l.MDef, l.HP, l.MP, l.HPRegen, l.MPRegen,
				l.WalkOnRide, l.RunOnRide, l.SlowSwimOnRide, l.FastSwimOnRide, l.SlowFlyOnRide, l.FastFlyOnRide) < 0 {
				p.addf("%s: level %d: negative stat or speed", where, l.Level)
			}
		}
	}
	for i, ps := range d.PetSkills {
		if ps.NpcID <= 0 || ps.SkillID <= 0 || ps.SkillLevel < 0 {
			p.addf("%s: row %d (NPC %d, skill %d level %d): bad ID or negative level", PetSkillsFile, i+1, ps.NpcID, ps.SkillID, ps.SkillLevel)
		}
		s.petSkills[ps.NpcID] = append(s.petSkills[ps.NpcID], ps)
	}
	for i := range d.Cubics {
		c := &d.Cubics[i]
		where := fmt.Sprintf("%s: cubic %d level %d", CubicsFile, c.ID, c.Level)
		if c.ID <= 0 || c.Level <= 0 {
			p.addf("%s: ID and level must be positive", where)
		}
		if i > 0 && compareCubics(&d.Cubics[i-1], c) >= 0 {
			p.addf("%s: cubics are not sorted and unique by ID and level", where)
		}
		if c.Duration < 0 || c.Delay < 0 || c.MaxCount < 0 || c.UseUp < 0 || c.Power < 0 {
			p.addf("%s: negative duration, delay, count, use-up or power", where)
		}
		if !slices.Contains(cubicSelectionModes, c.SelectionMode) {
			p.addf("%s: unknown selection mode %q", where, c.SelectionMode)
		}
		if !slices.Contains(cubicTargetTypes, c.TargetType) {
			p.addf("%s: unknown target type %q", where, c.TargetType)
		}
		checkCubicConditions(&p, where, c.Conditions)
		for _, sk := range c.Skills {
			w := fmt.Sprintf("%s: skill %d level %d", where, sk.ID, sk.Level)
			if sk.ID <= 0 || sk.Level <= 0 {
				p.addf("%s: ID and level must be positive", w)
			}
			if sk.TriggerRate < 0 {
				p.addf("%s: negative trigger rate", w)
			}
			if !slices.Contains(cubicTargetTypes, sk.TargetType) {
				p.addf("%s: unknown target type %q", w, sk.TargetType)
			}
			checkCubicConditions(&p, w, sk.Conditions)
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return s, nil
}

func compareCubics(a, b *CubicTemplate) int {
	return cmp.Or(cmp.Compare(a.ID, b.ID), cmp.Compare(a.Level, b.Level))
}

// checkCubicConditions checks the kinds of the conditions and that each
// sets only the fields of its kind.
func checkCubicConditions(p *problems, where string, conds []CubicCondition) {
	for _, c := range conds {
		var ok bool
		switch c.Kind {
		case CubicCondHP:
			ok = (c.HPType == CubicHPGreater || c.HPType == CubicHPLesser) && c.Range == 0 && c.Min == 0 && c.Max == 0
		case CubicCondRange:
			ok = c.HPType == "" && c.Percent == 0 && c.Min == 0 && c.Max == 0 && c.Range >= 0
		case CubicCondHealthPercent:
			ok = c.HPType == "" && c.Percent == 0 && c.Range == 0
		}
		if !ok {
			p.addf("%s: bad condition %+v", where, c)
		}
	}
}

// Pet returns the pet of an NPC ID.
func (s *Summons) Pet(npcID int) (*PetData, bool) {
	i, ok := slices.BinarySearchFunc(s.Pets, npcID, func(p PetData, id int) int { return cmp.Compare(p.NpcID, id) })
	if !ok {
		return nil, false
	}
	return &s.Pets[i], true
}

// PetByItem returns the pet an item summons (Mobius
// PetDataTable.getPetDataByItemId). Item -1 (no item) finds nothing.
func (s *Summons) PetByItem(itemID int) (*PetData, bool) {
	i, ok := s.petByItem[itemID]
	if !ok {
		return nil, false
	}
	return &s.Pets[i], true
}

// IsPetItem reports whether an item summons a pet.
func (s *Summons) IsPetItem(itemID int) bool {
	_, ok := s.petByItem[itemID]
	return ok
}

// PetSkills returns the PetSkillData rows of an NPC, in file order. The
// slice must not be modified.
func (s *Summons) PetSkills(npcID int) []PetSkill { return s.petSkills[npcID] }

// Cubic returns a cubic at a level.
func (s *Summons) Cubic(id, level int) (*CubicTemplate, bool) {
	key := CubicTemplate{ID: id, Level: level}
	i, ok := slices.BinarySearchFunc(s.Cubics, &key, func(c CubicTemplate, k *CubicTemplate) int { return compareCubics(&c, k) })
	if !ok {
		return nil, false
	}
	return &s.Cubics[i], true
}

// CheckWorld checks the references of the summon data into the world data:
// every pet is an NPC template. The NPCs of the pet skills are not checked:
// PetSkillData.xml names a few that do not exist (14261-14264,
// 14445-14448, 14629-14632), which Mobius never looks up.
func (s *Summons) CheckWorld(w *World) error {
	var p problems
	for _, pet := range s.Pets {
		if _, ok := w.Npc(pet.NpcID); !ok {
			p.addf("%s: pet %d: no such NPC", PetsFile, pet.NpcID)
		}
	}
	return p.err()
}
