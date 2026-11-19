package data

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

func init() {
	registerDomain(domain{name: "players", load: loadPlayers, link: linkPlayers})
}

// BaseStats are the base attributes of a class.
type BaseStats struct {
	STR int `json:"str"`
	CON int `json:"con"`
	DEX int `json:"dex"`
	INT int `json:"int"`
	WIT int `json:"wit"`
	MEN int `json:"men"`
}

// Vitals are maximum HP, MP and CP.
type Vitals struct {
	HP float64 `json:"hp"`
	MP float64 `json:"mp"`
	CP float64 `json:"cp"`
}

// Location is a point in the world.
type Location struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

// Template describes a starting class: what character creation offers and
// what a new character starts with. It is a view of the class's
// ClassTemplate.
type Template struct {
	ClassID   int    `json:"classId"`
	ClassName string `json:"className"`
	// Race is the race ID the client uses (Mobius Race ordinal): 0 Human,
	// 1 Elf, 2 Dark Elf, 3 Orc, 4 Dwarf.
	Race     int       `json:"race"`
	RaceName string    `json:"raceName"`
	Stats    BaseStats `json:"baseStats"`
	// Level1 holds the maximum HP, MP and CP at level 1.
	Level1 Vitals `json:"level1"`
	// CreationPoints are the possible start locations; a new character
	// gets one of them at random.
	CreationPoints []Location `json:"creationPoints"`
}

// EquipmentItem is one item a new character receives.
type EquipmentItem struct {
	ItemID   int   `json:"itemId"`
	Count    int64 `json:"count"`
	Equipped bool  `json:"equipped"`
}

// Equipment is the initial equipment of a starting class.
type Equipment struct {
	ClassID int             `json:"classId"`
	Items   []EquipmentItem `json:"items"`
}

// ClassTemplate is a player class template, Mobius PlayerTemplate, from
// dist/game/data/stats/players/templates/**. Values hold what Mobius uses:
// the XML has every field but mAtkSpd and mCritRate, which take the Mobius
// defaults (CreatureTemplate). The XML's baseCanPenetrate and
// baseDamRange are left out; Mobius does not use them.
type ClassTemplate struct {
	ClassID int `json:"classId"`
	// Name is the class name from classList.xml, such as "Human Fighter".
	Name string `json:"name"`
	Race Race   `json:"race"`
	// ParentID is the class this one transfers from, -1 for a starting
	// class. Race and parent come from the Mobius PlayerClass enum.
	ParentID int       `json:"parentId"`
	Stats    BaseStats `json:"baseStats"`

	PAtk      int `json:"pAtk"`
	MAtk      int `json:"mAtk"`
	PAtkSpd   int `json:"pAtkSpd"`
	MAtkSpd   int `json:"mAtkSpd"`
	CritRate  int `json:"critRate"`
	MCritRate int `json:"mCritRate"`
	// AttackType is the weapon type of the bare hands.
	AttackType   WeaponType `json:"attackType"`
	AttackRange  int        `json:"attackRange"`
	RandomDamage int        `json:"randomDamage"`
	MoveSpeed    MoveSpeeds `json:"moveSpeed"`
	Breath       int        `json:"breath"`
	SafeFall     int        `json:"safeFall"`
	// PDef and MDef are the defence of the empty armor and jewel slots.
	PDef            ArmorDefence `json:"pDef"`
	MDef            JewelDefence `json:"mDef"`
	CollisionMale   Collision    `json:"collisionMale"`
	CollisionFemale Collision    `json:"collisionFemale"`
	// CreationPoints are the start locations of a new character.
	CreationPoints []Location  `json:"creationPoints"`
	Levels         ClassLevels `json:"levels"`
}

// MoveSpeeds are the base movement speeds.
type MoveSpeeds struct {
	Walk     float64 `json:"walk"`
	Run      float64 `json:"run"`
	SlowSwim float64 `json:"slowSwim"`
	FastSwim float64 `json:"fastSwim"`
}

// ArmorDefence is the base P.Def of each armor slot while it is empty.
type ArmorDefence struct {
	Chest     int `json:"chest"`
	Legs      int `json:"legs"`
	Head      int `json:"head"`
	Feet      int `json:"feet"`
	Gloves    int `json:"gloves"`
	Underwear int `json:"underwear"`
	Cloak     int `json:"cloak"`
}

// JewelDefence is the base M.Def of each jewel slot while it is empty.
type JewelDefence struct {
	REar    int `json:"rear"`
	LEar    int `json:"lear"`
	RFinger int `json:"rfinger"`
	LFinger int `json:"lfinger"`
	Neck    int `json:"neck"`
}

// Collision is a collision radius and height.
type Collision struct {
	Radius float64 `json:"radius"`
	Height float64 `json:"height"`
}

// ClassLevels holds the per-level values of a class, from its
// lvlUpgainData: index 0 is level 1.
type ClassLevels struct {
	HP      []float64 `json:"hp"`
	MP      []float64 `json:"mp"`
	CP      []float64 `json:"cp"`
	HPRegen []float64 `json:"hpRegen"`
	MPRegen []float64 `json:"mpRegen"`
	CPRegen []float64 `json:"cpRegen"`
}

// LevelStats are the base vitals of a class at one level.
type LevelStats struct {
	// HP, MP and CP are rounded to float32, as Mobius keeps them, so the
	// stat formulas give Mobius's numbers.
	HP, MP, CP float64
	// HPRegen, MPRegen and CPRegen are the base regeneration values.
	HPRegen, MPRegen, CPRegen float64
}

// Class returns the PlayerClass of the template.
func (c *ClassTemplate) Class() PlayerClass { return PlayerClass(c.ClassID) }

// Parent returns the class this one transfers from; false for a starting
// class.
func (c *ClassTemplate) Parent() (PlayerClass, bool) {
	if c.ParentID < 0 {
		return 0, false
	}
	return PlayerClass(c.ParentID), true
}

// IsStarting reports whether the class is one that character creation
// offers.
func (c *ClassTemplate) IsStarting() bool { return c.ParentID < 0 }

// MaxLevel returns the highest level the template has values for.
func (c *ClassTemplate) MaxLevel() int { return len(c.Levels.HP) }

// Level returns the base vitals at level; false if the template has none
// for it.
func (c *ClassTemplate) Level(level int) (LevelStats, bool) {
	l := c.Levels
	i := level - 1
	if i < 0 || i >= len(l.HP) || i >= len(l.MP) || i >= len(l.CP) ||
		i >= len(l.HPRegen) || i >= len(l.MPRegen) || i >= len(l.CPRegen) {
		return LevelStats{}, false
	}
	return LevelStats{
		HP:      float64(float32(l.HP[i])),
		MP:      float64(float32(l.MP[i])),
		CP:      float64(float32(l.CP[i])),
		HPRegen: l.HPRegen[i],
		MPRegen: l.MPRegen[i],
		CPRegen: l.CPRegen[i],
	}, true
}

// CollisionOf returns the collision size of a male or female character.
func (c *ClassTemplate) CollisionOf(female bool) Collision {
	if female {
		return c.CollisionFemale
	}
	return c.CollisionMale
}

// BasePDef returns the base P.Def with every armor slot empty, Mobius
// basePDef.
func (c *ClassTemplate) BasePDef() int {
	d := c.PDef
	return d.Chest + d.Legs + d.Head + d.Feet + d.Gloves + d.Underwear + d.Cloak
}

// BaseMDef returns the base M.Def with every jewel slot empty, Mobius
// baseMDef. Mobius adds the right ring twice; datagen makes sure both rings
// are equal, so this is the same sum.
func (c *ClassTemplate) BaseMDef() int {
	d := c.MDef
	return d.REar + d.LEar + d.RFinger + d.LFinger + d.Neck
}

// Paperdoll slot numbers, as Mobius Inventory and storage.PaperdollSlot
// number them, for SlotDef.
const (
	slotUnder   = 0
	slotLEar    = 1
	slotREar    = 2
	slotNeck    = 3
	slotLFinger = 4
	slotRFinger = 5
	slotHead    = 6
	slotGloves  = 9
	slotChest   = 10
	slotLegs    = 11
	slotFeet    = 12
	slotCloak   = 13
)

// SlotDef returns the base defence of a paperdoll slot (Mobius
// getBaseDefBySlot): P.Def for the armor slots, M.Def for the jewel slots,
// 0 for the others. The stat formulas take it off when an item is worn
// there.
func (c *ClassTemplate) SlotDef(slot int) int {
	switch slot {
	case slotUnder:
		return c.PDef.Underwear
	case slotLEar:
		return c.MDef.LEar
	case slotREar:
		return c.MDef.REar
	case slotNeck:
		return c.MDef.Neck
	case slotLFinger, slotRFinger: // Mobius uses the right ring for both
		return c.MDef.RFinger
	case slotHead:
		return c.PDef.Head
	case slotGloves:
		return c.PDef.Gloves
	case slotChest:
		return c.PDef.Chest
	case slotLegs:
		return c.PDef.Legs
	case slotFeet:
		return c.PDef.Feet
	case slotCloak:
		return c.PDef.Cloak
	}
	return 0
}

// ClassTemplate returns the template of a class.
func (d *Data) ClassTemplate(classID int) (*ClassTemplate, bool) {
	i, ok := d.classes[classID]
	if !ok {
		return nil, false
	}
	return &d.Classes[i], true
}

// Template returns the template of a starting class.
func (d *Data) Template(classID int) (Template, bool) {
	i, ok := d.templates[classID]
	if !ok {
		return Template{}, false
	}
	return d.Templates[i], true
}

// InitialItems returns the items a new character of the class receives, or
// nil if the class has none. The slice must not be modified.
func (d *Data) InitialItems(classID int) []EquipmentItem {
	i, ok := d.equipment[classID]
	if !ok {
		return nil
	}
	return d.InitialEquipment[i].Items
}

// loadPlayers reads the players domain: class templates, initial
// equipment and the player tables.
func loadPlayers(fsys fs.FS, d *Data) error {
	var errs []error
	for _, f := range []struct {
		name string
		v    any
	}{
		{ClassesFile, &d.Classes},
		{EquipmentFile, &d.InitialEquipment},
		{StatBonusFile, &d.StatBonuses},
		{ExperienceFile, &d.Experience},
		{ExperienceLossFile, &d.ExperienceLoss},
		{KarmaLossFile, &d.KarmaLoss},
		{HitConditionBonusFile, &d.HitConditionBonus},
	} {
		if err := readJSON(fsys, f.name, f.v); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	var p problems
	d.StatBonuses.check(&p)
	d.Experience.check(&p)
	d.ExperienceLoss.check(&p, ExperienceLossFile)
	d.KarmaLoss.check(&p, KarmaLossFile)
	d.indexClasses(&p)
	d.indexEquipment(&p)
	return p.err()
}

// indexClasses checks the class templates, indexes them and builds the
// starting templates. It checks that:
//
//   - there is at least one class, and class IDs are unique;
//   - each class is a Mobius PlayerClass with the same race and parent;
//   - names, base stats, combat values, speeds and collision sizes are
//     set, and slot defences are not negative;
//   - each class has creation points and one value per level for every
//     level of the experience table (maxLevel), with positive HP, MP and
//     CP and regeneration that is not negative.
func (d *Data) indexClasses(p *problems) {
	d.classes = make(map[int]int, len(d.Classes))
	d.templates = make(map[int]int)
	d.Templates = nil
	if len(d.Classes) == 0 {
		p.addf("%s: no class templates", ClassesFile)
	}
	for i := range d.Classes {
		c := &d.Classes[i]
		where := fmt.Sprintf("%s: class %d (%q)", ClassesFile, c.ClassID, c.Name)
		if _, dup := d.classes[c.ClassID]; dup {
			p.addf("%s: duplicate class ID", where)
			continue
		}
		d.classes[c.ClassID] = i
		c.check(p, where, d.Experience.MaxLevel)
		if c.IsStarting() && len(c.Levels.HP) > 0 && len(c.Levels.MP) > 0 && len(c.Levels.CP) > 0 {
			d.Templates = append(d.Templates, Template{
				ClassID:        c.ClassID,
				ClassName:      c.Name,
				Race:           int(c.Race),
				RaceName:       raceName(c.Race),
				Stats:          c.Stats,
				Level1:         Vitals{HP: c.Levels.HP[0], MP: c.Levels.MP[0], CP: c.Levels.CP[0]},
				CreationPoints: c.CreationPoints,
			})
		}
	}
	slices.SortFunc(d.Templates, func(a, b Template) int { return cmp.Compare(a.ClassID, b.ClassID) })
	for i, t := range d.Templates {
		d.templates[t.ClassID] = i
	}
}

func (c *ClassTemplate) check(p *problems, where string, levels int) {
	class := c.Class()
	switch {
	case !class.Valid():
		p.addf("%s: not a Mobius player class", where)
	case c.Race != class.Race():
		p.addf("%s: race %s, but the Mobius class has %s", where, c.Race, class.Race())
	default:
		parent, ok := class.Parent()
		want := -1
		if ok {
			want = int(parent)
		}
		if c.ParentID != want {
			p.addf("%s: parent %d, but the Mobius class has %d", where, c.ParentID, want)
		}
	}
	if c.Name == "" {
		p.addf("%s: no name", where)
	}
	s := c.Stats
	if s.STR <= 0 || s.CON <= 0 || s.DEX <= 0 || s.INT <= 0 || s.WIT <= 0 || s.MEN <= 0 {
		p.addf("%s: base stats must be positive (got %+v)", where, s)
	}
	if c.PAtk <= 0 || c.MAtk <= 0 || c.PAtkSpd <= 0 || c.MAtkSpd <= 0 || c.AttackRange <= 0 {
		p.addf("%s: P.Atk, M.Atk, their speeds and the attack range must be positive", where)
	}
	if c.CritRate < 0 || c.MCritRate < 0 || c.RandomDamage < 0 || c.Breath < 0 || c.SafeFall < 0 {
		p.addf("%s: crit rates, random damage, breath and safe fall must not be negative", where)
	}
	if !c.AttackType.Valid() {
		p.addf("%s: invalid attack type %d", where, c.AttackType)
	}
	m := c.MoveSpeed
	if m.Walk <= 0 || m.Run <= 0 || m.SlowSwim <= 0 || m.FastSwim <= 0 {
		p.addf("%s: move speeds must be positive (got %+v)", where, m)
	}
	pd, md := c.PDef, c.MDef
	if min(pd.Chest, pd.Legs, pd.Head, pd.Feet, pd.Gloves, pd.Underwear, pd.Cloak) < 0 ||
		min(md.REar, md.LEar, md.RFinger, md.LFinger, md.Neck) < 0 {
		p.addf("%s: slot defences must not be negative", where)
	}
	for _, col := range []Collision{c.CollisionMale, c.CollisionFemale} {
		if col.Radius <= 0 || col.Height <= 0 {
			p.addf("%s: collision sizes must be positive (got %+v)", where, col)
			break
		}
	}
	if len(c.CreationPoints) == 0 {
		p.addf("%s: no creation points", where)
	}
	l := c.Levels
	for _, col := range []struct {
		name     string
		values   []float64
		positive bool
	}{
		{"hp", l.HP, true}, {"mp", l.MP, true}, {"cp", l.CP, true},
		{"hpRegen", l.HPRegen, false}, {"mpRegen", l.MPRegen, false}, {"cpRegen", l.CPRegen, false},
	} {
		if levels > 0 && len(col.values) != levels {
			p.addf("%s: levels.%s has %d values, want one per level 1-%d", where, col.name, len(col.values), levels)
		}
		for i, v := range col.values {
			if v < 0 || (col.positive && v == 0) {
				p.addf("%s: level %d %s %v is out of range", where, i+1, col.name, v)
				break
			}
		}
	}
}

// indexEquipment checks the initial equipment and indexes it: each list
// belongs to a starting class, class IDs are unique, and each list has
// items with positive counts. linkPlayers checks the items themselves.
func (d *Data) indexEquipment(p *problems) {
	d.equipment = make(map[int]int, len(d.InitialEquipment))
	for i, e := range d.InitialEquipment {
		where := fmt.Sprintf("%s: class %d", EquipmentFile, e.ClassID)
		if _, dup := d.equipment[e.ClassID]; dup {
			p.addf("%s: duplicate class ID", where)
			continue
		}
		d.equipment[e.ClassID] = i
		if _, ok := d.templates[e.ClassID]; !ok {
			p.addf("%s: no starting class template for this class in %s", where, ClassesFile)
		}
		if len(e.Items) == 0 {
			p.addf("%s: no items", where)
		}
		for _, ei := range e.Items {
			if ei.Count <= 0 {
				p.addf("%s: item %d: count must be positive (got %d)", where, ei.ItemID, ei.Count)
			}
		}
	}
}

// linkPlayers checks that the initial equipment names existing items and
// that the ones marked equipped can be worn.
func linkPlayers(d *Data) error {
	if !d.Loaded("items") {
		return nil
	}
	var p problems
	for _, e := range d.InitialEquipment {
		where := fmt.Sprintf("%s: class %d", EquipmentFile, e.ClassID)
		for _, ei := range e.Items {
			it, ok := d.ItemTemplate(ei.ItemID)
			if !ok {
				p.addf("%s: item %d does not exist in %s", where, ei.ItemID, ItemsFile)
				continue
			}
			if ei.Equipped && !it.Equipable() {
				p.addf("%s: item %d (%q) is marked equipped but has no body part to wear it on", where, ei.ItemID, it.Name)
			}
		}
	}
	return p.err()
}

// raceName turns a race such as DARK_ELF into "Dark Elf".
func raceName(r Race) string {
	words := strings.Split(strings.ToLower(r.String()), "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
