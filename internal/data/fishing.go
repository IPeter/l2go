package data

import (
	"fmt"
	"io/fs"
	"os"
)

// Paths of the fishing data files (Plan 14, M11g), relative to the data
// folder. The first three come from dist/game/data/stats/fishing/, the lure
// table from the lure switches of Mobius Player.java (decision X6).
const (
	FishesFile          = "fishing/fishes.json"
	FishingMonstersFile = "fishing/monsters.json"
	FishingRodsFile     = "fishing/rods.json"
	FishingLuresFile    = "fishing/lures.json"
)

// The fish grades (Mobius Fish.getGradeId of the fishGrade attribute).
const (
	FishGradeEasy   = 0 // fish_easy: caught with the beginners' lures
	FishGradeNormal = 1 // fish_normal
	FishGradeHard   = 2 // fish_hard: caught with the luminous lures
)

// The fish groups (Mobius Fish.getGroupId of the fishGroup attribute): what
// the lures prefer and the client's fish window shows.
const (
	FishGroupWide      = 0  // wide (fat fish)
	FishGroupSwift     = 1  // swift (nimble fish)
	FishGroupUgly      = 2  // ugly
	FishGroupBox       = 3  // fish_box (treasure chests)
	FishGroupEasyWide  = 4  // easy_wide
	FishGroupEasySwift = 5  // easy_swift
	FishGroupEasyUgly  = 6  // easy_ugly
	FishGroupHardWide  = 7  // hard_wide
	FishGroupHardSwift = 8  // hard_swift
	FishGroupHardUgly  = 9  // hard_ugly
	FishGroupHS        = 10 // hs_fish (the Old Boxes; no lure picks it)
)

// Fish is a <fish> of fishes.xml, Mobius Fish. Group and Grade are Mobius's
// numbers of the named attributes. BiteRate, MaxLength, LengthRate,
// GutsCheckProbability and CheatingProb are kept although Mobius does not
// use them.
type Fish struct {
	FishID   int     `json:"fishId"`
	ItemID   int     `json:"itemId"`
	ItemName string  `json:"itemName"`
	Group    int     `json:"group"`
	Level    int     `json:"level"`
	BiteRate float64 `json:"biteRate"`
	// Guts is the chance in percent that the fish bites at each check.
	Guts       float64 `json:"guts"`
	HP         int     `json:"hp"`
	MaxLength  int     `json:"maxLength"`
	LengthRate float64 `json:"lengthRate"`
	// HPRegen is what the fish regains each second while it fights.
	HPRegen float64 `json:"hpRegen"`
	// StartCombatTime is how long, in seconds after the first 10, the fish
	// may take to bite.
	StartCombatTime int `json:"startCombatTime"`
	// CombatDuration is the length of the fight in seconds.
	CombatDuration int `json:"combatDuration"`
	// GutsCheckTime times the lure's check delay is the period of the bite
	// checks, in milliseconds.
	GutsCheckTime        int     `json:"gutsCheckTime"`
	GutsCheckProbability float64 `json:"gutsCheckProbability"`
	CheatingProb         float64 `json:"cheatingProb"`
	Grade                int     `json:"grade"`
}

// FishingRod is a <fishingRod> of fishingRods.xml, Mobius FishingRod.
type FishingRod struct {
	ID     int     `json:"id"`
	ItemID int     `json:"itemId"`
	Level  int     `json:"level"`
	Name   string  `json:"name"`
	Damage float64 `json:"damage"`
}

// FishingMonster is a <fishingMonster> of fishingMonsters.xml, Mobius
// FishingMonster: a player of MinLevel to MaxLevel who lands a fish pulls
// up the monster NpcID instead, with Probability percent (Rnd.get(100) <=
// Probability).
type FishingMonster struct {
	MinLevel    int `json:"minLevel"`
	MaxLevel    int `json:"maxLevel"`
	NpcID       int `json:"npcId"`
	Probability int `json:"probability"`
}

// Lure is what Mobius Player.java decides by the lure's item ID
// (getRandomFishGrade, getRandomFishGroup, startLookingForFishTask) and
// Item.isNightLure. Mobius keeps it in Java code; datagen writes it from a
// table that names that source (decision X6).
type Lure struct {
	ItemID int `json:"itemId"`
	// Grade is the grade of the fish it catches (FishGradeEasy...).
	Grade int `json:"grade"`
	// Night lures catch only at night, the others only by day.
	Night bool `json:"night,omitempty"`
	// CheckDelay times the fish's GutsCheckTime is the bite check period in
	// milliseconds (133, 100 or 66).
	CheckDelay int `json:"checkDelay"`
	// Groups pick the fish group from check = Rnd.get(100): the first entry
	// with check <= UpTo wins. The last entry has UpTo 99.
	Groups []LureGroup `json:"groups"`
}

// LureGroup is one branch of a lure's group switch.
type LureGroup struct {
	UpTo  int `json:"upTo"`
	Group int `json:"group"`
}

// FishingData is the content of the fishing data files.
type FishingData struct {
	Fishes   []Fish           // FishesFile, in file order (by fish ID)
	Monsters []FishingMonster // FishingMonstersFile, in file order
	Rods     []FishingRod     // FishingRodsFile, sorted by rod ID
	Lures    []Lure           // FishingLuresFile, sorted by item ID
}

// Fishing is the checked fishing data. It must not be modified after
// NewFishing or LoadFishing returns, and is then safe for concurrent use;
// the pointers its lookups return point into it.
type Fishing struct {
	FishingData
	rods  map[int]int // item ID → index in Rods
	lures map[int]int // item ID → index in Lures
}

// LoadFishing reads and checks the fishing data files in dir.
func LoadFishing(dir string) (*Fishing, error) {
	f, err := LoadFishingFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return f, nil
}

// LoadFishingFS reads and checks the fishing data files in fsys.
func LoadFishingFS(fsys fs.FS) (*Fishing, error) {
	var d FishingData
	for _, f := range []struct {
		name string
		v    any
	}{
		{FishesFile, &d.Fishes},
		{FishingMonstersFile, &d.Monsters},
		{FishingRodsFile, &d.Rods},
		{FishingLuresFile, &d.Lures},
	} {
		if err := readJSON(fsys, f.name, f.v); err != nil {
			return nil, err
		}
	}
	return NewFishing(d)
}

// NewFishing checks the fishing data. The slices are used as they are.
// Fishes: IDs positive and unique, item IDs positive, a known grade and
// group, a level of 1-27, HP and the fight and check times positive. Rods:
// IDs sorted and unique, one rod per item. Monsters: level ranges that do
// not overlap. Lures: item IDs sorted and unique, a known grade, a positive
// check delay, groups in rising order ending at 99.
func NewFishing(d FishingData) (*Fishing, error) {
	var p problems
	f := &Fishing{FishingData: d, rods: make(map[int]int), lures: make(map[int]int)}
	seen := make(map[int]bool, len(d.Fishes))
	for _, x := range d.Fishes {
		where := fmt.Sprintf("%s: fish %d", FishesFile, x.FishID)
		switch {
		case x.FishID <= 0 || seen[x.FishID]:
			p.addf("%s: the ID must be positive and unique", where)
		case x.ItemID <= 0:
			p.addf("%s: item ID %d must be positive", where, x.ItemID)
		case x.Grade < FishGradeEasy || x.Grade > FishGradeHard:
			p.addf("%s: unknown grade %d", where, x.Grade)
		case x.Group < FishGroupWide || x.Group > FishGroupHS:
			p.addf("%s: unknown group %d", where, x.Group)
		case x.Level < 1 || x.Level > 27:
			p.addf("%s: level %d is not 1-27", where, x.Level)
		case x.HP <= 0 || x.CombatDuration <= 0 || x.GutsCheckTime <= 0 || x.StartCombatTime < 0:
			p.addf("%s: HP, combat duration and guts check time must be positive", where)
		}
		seen[x.FishID] = true
	}
	for i, r := range d.Rods {
		where := fmt.Sprintf("%s: rod %d", FishingRodsFile, r.ID)
		if i > 0 && d.Rods[i-1].ID >= r.ID {
			p.addf("%s: rod IDs are not sorted and unique", where)
		}
		if _, dup := f.rods[r.ItemID]; dup || r.ItemID <= 0 {
			p.addf("%s: item %d is not positive or has two rods", where, r.ItemID)
		}
		f.rods[r.ItemID] = i
	}
	for i, m := range d.Monsters {
		where := fmt.Sprintf("%s: monster %d", FishingMonstersFile, m.NpcID)
		if m.NpcID <= 0 || m.MinLevel > m.MaxLevel || m.Probability < 0 {
			p.addf("%s: needs an NPC, a level range and a probability that is not negative", where)
		}
		for _, o := range d.Monsters[:i] {
			if m.MinLevel <= o.MaxLevel && o.MinLevel <= m.MaxLevel {
				p.addf("%s: levels %d-%d overlap monster %d", where, m.MinLevel, m.MaxLevel, o.NpcID)
			}
		}
	}
	for i, l := range d.Lures {
		where := fmt.Sprintf("%s: lure %d", FishingLuresFile, l.ItemID)
		if i > 0 && d.Lures[i-1].ItemID >= l.ItemID {
			p.addf("%s: item IDs are not sorted and unique", where)
		}
		f.lures[l.ItemID] = i
		if l.Grade < FishGradeEasy || l.Grade > FishGradeHard || l.CheckDelay <= 0 {
			p.addf("%s: unknown grade %d or check delay %d", where, l.Grade, l.CheckDelay)
		}
		last := -1
		for _, g := range l.Groups {
			if g.UpTo <= last || g.Group < FishGroupWide || g.Group > FishGroupHS {
				p.addf("%s: groups must rise and name known groups", where)
			}
			last = g.UpTo
		}
		if last != 99 {
			p.addf("%s: the groups must end at 99", where)
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return f, nil
}

// Rod returns the fishing rod of a weapon item (Mobius
// FishingRodsData.getFishingRod), false for none.
func (f *Fishing) Rod(itemID int) (*FishingRod, bool) {
	i, ok := f.rods[itemID]
	if !ok {
		return nil, false
	}
	return &f.Rods[i], true
}

// Lure returns the lure of an item, false for an item that is no lure.
func (f *Fishing) Lure(itemID int) (*Lure, bool) {
	i, ok := f.lures[itemID]
	if !ok {
		return nil, false
	}
	return &f.Lures[i], true
}

// Monster returns the fishing monster of a player level (Mobius
// FishingMonstersData.getFishingMonster), false for none (levels below 6).
func (f *Fishing) Monster(level int) (*FishingMonster, bool) {
	for i := range f.Monsters {
		if m := &f.Monsters[i]; level >= m.MinLevel && level <= m.MaxLevel {
			return m, true
		}
	}
	return nil, false
}
