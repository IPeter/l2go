package data

import (
	"errors"
	"fmt"
	"io/fs"
)

func init() {
	registerDomain(domain{name: "items", load: loadItems})
}

// Item is an item template, Mobius ItemTemplate with its subclass Weapon,
// Armor or EtcItem, from dist/game/data/stats/items/*.xml (custom/ is left
// out). Values hold what Mobius uses: a value missing in the XML holds the
// Mobius default, and the Mobius rules that derive values (the item list
// types, the shield and shot item types, the reuse delay of bows, the stat
// functions of <stats>) are applied by cmd/datagen.
type Item struct {
	ID int `json:"id"`
	// DisplayID is the item ID the client shows. The loader sets it to ID
	// when the file has none.
	DisplayID int    `json:"displayId,omitempty"`
	Name      string `json:"name"`
	// Type is the kind of item: ItemTypeWeapon, ItemTypeArmor or
	// ItemTypeEtc. The matching one of Weapon, Armor and Etc is set.
	Type string `json:"type"`
	// BodyPart is the Mobius body part name of the item, as the XML writes
	// it ("rhand", "rear;lear", "onepiece"), or empty if it has none. Part
	// returns it as a BodyPart.
	BodyPart string `json:"bodyPart,omitempty"`
	// BodyPartMask is the Mobius body part bit mask that ItemList sends (0
	// if none).
	BodyPartMask int `json:"bodyPartMask,omitempty"`
	// Type1 and Type2 are the Mobius item list types that ItemList sends.
	// Type1: 0 weapon or accessory, 1 armor or shield, 4 other. Type2: 0
	// weapon, 1 armor or shield, 2 accessory, 3 quest item, 4 adena, 5
	// other.
	Type1 int `json:"type1"`
	Type2 int `json:"type2"`
	// Grade is the crystal type (CrystalNone if none); CrystalCount is how
	// many crystals of it the item gives when crystallized.
	Grade        CrystalType `json:"grade,omitempty"`
	CrystalCount int         `json:"crystalCount,omitempty"`
	Weight       int         `json:"weight,omitempty"`
	// Price is the reference price in adena.
	Price    int          `json:"price,omitempty"`
	Material MaterialType `json:"material"`
	Icon     string       `json:"icon,omitempty"`
	// DefaultAction is what using the item does (ActionNone if nothing).
	DefaultAction ActionType `json:"defaultAction"`
	// DefaultEnchant is the enchant level a new item of this template has.
	DefaultEnchant int `json:"defaultEnchant,omitempty"`

	Stackable             bool `json:"stackable,omitempty"`
	Tradable              bool `json:"tradable,omitempty"`
	Droppable             bool `json:"droppable,omitempty"`
	Destroyable           bool `json:"destroyable,omitempty"`
	Sellable              bool `json:"sellable,omitempty"`
	Depositable           bool `json:"depositable,omitempty"`
	Freightable           bool `json:"freightable,omitempty"`
	QuestItem             bool `json:"questItem,omitempty"`
	Enchantable           bool `json:"enchantable,omitempty"`
	Elementable           bool `json:"elementable,omitempty"`
	OlyRestricted         bool `json:"olyRestricted,omitempty"`
	ForNpc                bool `json:"forNpc,omitempty"`
	AllowSelfResurrection bool `json:"allowSelfResurrection,omitempty"`
	ImmediateEffect       bool `json:"immediateEffect,omitempty"`
	ExImmediateEffect     bool `json:"exImmediateEffect,omitempty"`
	// Hero, PvP and Common come from Mobius rules on item ID ranges
	// (ItemTemplate), kept here as data.
	Hero   bool `json:"hero,omitempty"`
	PvP    bool `json:"pvp,omitempty"`
	Common bool `json:"common,omitempty"`

	// Duration is the mana of a shadow item, in minutes; 0 if it has none
	// (Mobius -1).
	Duration int `json:"duration,omitempty"`
	// Time is the life of a limited-time item, in minutes; 0 if it has
	// none (Mobius -1).
	Time int `json:"time,omitempty"`
	// AutoDestroyTime is how long the item stays on the ground, in
	// milliseconds; 0 for the configured default.
	AutoDestroyTime int `json:"autoDestroyTime,omitempty"`
	// ReuseDelay is the reuse delay of the item (Mobius reuse_delay, 1500
	// for a bow without one); SharedReuseGroup groups items whose reuse is
	// shared.
	ReuseDelay       int `json:"reuseDelay,omitempty"`
	SharedReuseGroup int `json:"sharedReuseGroup,omitempty"`
	// EquipReuseDelay is the delay after equipping, in milliseconds.
	EquipReuseDelay int `json:"equipReuseDelay,omitempty"`
	UseSkillDisTime int `json:"useSkillDisTime,omitempty"`

	Weapon *WeaponInfo `json:"weapon,omitempty"`
	Armor  *ArmorInfo  `json:"armor,omitempty"`
	Etc    *EtcInfo    `json:"etc,omitempty"`

	// Stats are the stat functions the item gives while worn, in Mobius
	// order: those of <stats>, then the enchant functions Mobius adds for
	// armor and weapons that have <stats>.
	Stats []ItemStat `json:"stats,omitempty"`
	// Skills are the skills of <skills>, in XML order (used from M7).
	Skills []ItemSkill `json:"skills,omitempty"`
	// Conditions is the <conditions> element, kept as XML for M7.
	Conditions *XMLElement `json:"conditions,omitempty"`
}

// WeaponInfo holds the values of a weapon, Mobius Weapon. The weapon's
// reach and random damage are its pAtkRange and randomDamage stats.
type WeaponInfo struct {
	Type        WeaponType `json:"type"`
	Magic       bool       `json:"magic,omitempty"`
	SoulShots   int        `json:"soulShots,omitempty"`
	SpiritShots int        `json:"spiritShots,omitempty"`
	MPConsume   int        `json:"mpConsume,omitempty"`
	// ReducedSoulShot is the soulshot count used instead of SoulShots, with
	// ReducedSoulShotChance percent; the same for MP.
	ReducedSoulShot        int `json:"reducedSoulShot,omitempty"`
	ReducedSoulShotChance  int `json:"reducedSoulShotChance,omitempty"`
	ReducedMPConsume       int `json:"reducedMpConsume,omitempty"`
	ReducedMPConsumeChance int `json:"reducedMpConsumeChance,omitempty"`
	// AttackRadius and AttackAngle are the fan of a hit, from the XML's
	// damage_range (Mobius defaults 40 and 0).
	AttackRadius int `json:"attackRadius"`
	AttackAngle  int `json:"attackAngle"`
	// AttackWeapon is false for weapons that cannot attack (Mobius
	// isAttackWeapon).
	AttackWeapon     bool `json:"attackWeapon"`
	ForceEquip       bool `json:"forceEquip,omitempty"`
	WeaponSkillsOnly bool `json:"weaponSkillsOnly,omitempty"`
}

// ArmorInfo holds the values of an armor, Mobius Armor. Shields are armors
// of type ArmorShield; jewels have Type2 2.
type ArmorInfo struct {
	Type ArmorType `json:"type"`
}

// EtcInfo holds the values of an other item, Mobius EtcItem.
type EtcInfo struct {
	// Type is the etc item type; shots are EtcShot (Mobius sets it from
	// the default action).
	Type EtcItemType `json:"type"`
	// Handler is the Mobius item handler that runs when the item is used.
	Handler string `json:"handler,omitempty"`
	// Blessed is set for blessed shots (Mobius EtcItem.isBlessed).
	Blessed             bool `json:"blessed,omitempty"`
	ExtractableCountMin int  `json:"extractableCountMin,omitempty"`
	ExtractableCountMax int  `json:"extractableCountMax,omitempty"`
}

// ItemStat is a stat function an item gives while it is worn, Mobius
// FuncTemplate: Op applies Value to Stat.
type ItemStat struct {
	Stat Stat   `json:"stat"`
	Op   StatOp `json:"op"`
	// Order is the Mobius evaluation order. Items always use 0: Mobius
	// DocumentItem passes 0 instead of the function's default order.
	Order int     `json:"order"`
	Value float64 `json:"value"`
}

// ItemSkill is a skill an item gives, from its <skills>.
type ItemSkill struct {
	ID    int `json:"id"`
	Level int `json:"level"`
	// Type is when the skill applies: ItemSkillNormal while the item is
	// worn (or when it is used), ItemSkillOnEnchant4 from enchant +4,
	// ItemSkillOnUnequip when it is taken off, ItemSkillOnCriticalSkill and
	// ItemSkillOnMagicSkill as weapon special abilities.
	Type ItemSkillType `json:"type,omitempty"`
	// Chance is the chance in percent of the critical and magic skills.
	Chance int `json:"chance,omitempty"`
}

// XMLElement is an XML element kept as data for a later milestone: its
// name, attributes and child elements.
type XMLElement struct {
	Name     string            `json:"name"`
	Attrs    map[string]string `json:"attrs,omitempty"`
	Children []XMLElement      `json:"children,omitempty"`
}

// StatOp is a stat function, Mobius mechanics/stats/functions/
// StatFunction. Its name in the data is the Mobius XML name ("add").
type StatOp uint8

// The stat functions.
const (
	OpAdd StatOp = iota
	OpDiv
	OpEnchant
	OpEnchantHP
	OpMul
	OpSet
	OpShare
	OpSub
)

var (
	statOpTable  = newEnumTable("stat function", "add", "div", "enchant", "enchanthp", "mul", "set", "share", "sub")
	statOpOrders = [...]int{30, 20, 0, 40, 20, 0, 30, 30}
)

// String returns the Mobius XML name of o.
func (o StatOp) String() string { return statOpTable.name(int(o)) }

// Valid reports whether o is a known StatOp.
func (o StatOp) Valid() bool { return statOpTable.valid(int(o)) }

// DefaultOrder returns the Mobius evaluation order of o when the data
// gives none: 0 for set and enchant, 20 for mul and div, 30 for add, sub
// and share, 40 for enchanthp.
func (o StatOp) DefaultOrder() int {
	if !o.Valid() {
		return 0
	}
	return statOpOrders[o]
}

// MarshalText implements encoding.TextMarshaler.
func (o StatOp) MarshalText() ([]byte, error) { return statOpTable.marshal(int(o)) }

// UnmarshalText implements encoding.TextUnmarshaler. Unknown names are
// errors.
func (o *StatOp) UnmarshalText(text []byte) error {
	v, err := statOpTable.unmarshal(text)
	if err != nil {
		return err
	}
	*o = StatOp(v) //nolint:gosec // an index into the 8 stat functions
	return nil
}

// ParseStatOp returns the StatOp with the Mobius XML name.
func ParseStatOp(name string) (StatOp, bool) {
	v, ok := statOpTable.parse(name)
	return StatOp(v), ok //nolint:gosec // an index into the 8 stat functions
}

// Mask returns the Mobius item type mask of a weapon type, 1 << ordinal.
func (w WeaponType) Mask() int { return 1 << w }

// Ranged reports whether the weapon type shoots (Mobius isRanged: bows).
func (w WeaponType) Ranged() bool { return w == WeaponBow }

// Mask returns the Mobius item type mask of an armor type.
func (a ArmorType) Mask() int { return 1 << (int(a) + NumWeaponTypes) }

// Part returns the body part of the item, BodyPartNone if it has none.
func (it *Item) Part() BodyPart {
	p, ok := BodyPartByName(it.BodyPart)
	if !ok {
		return BodyPartNone
	}
	return p
}

// Equipable reports whether the item can be worn: it has a body part and
// is not an etc item (Mobius isEquipable).
func (it *Item) Equipable() bool {
	return it.Part() != BodyPartNone && it.Type != ItemTypeEtc
}

// Crystallizable reports whether the item gives crystals.
func (it *Item) Crystallizable() bool { return it.Grade != CrystalNone && it.CrystalCount > 0 }

// Mask returns the Mobius item type mask (getItemMask): a bit per weapon
// type and per armor type, 0 for etc items.
func (it *Item) Mask() int {
	switch {
	case it.Weapon != nil:
		return it.Weapon.Type.Mask()
	case it.Armor != nil:
		return it.Armor.Type.Mask()
	}
	return 0
}

// StatValue returns the value of the first stat function of the item for
// s that is not an enchant function, such as the pAtkRange of a weapon.
func (it *Item) StatValue(s Stat) (float64, bool) {
	for _, st := range it.Stats {
		if st.Stat == s && st.Op != OpEnchant {
			return st.Value, true
		}
	}
	return 0, false
}

// maxItemID bounds item IDs, so the item index stays small.
const maxItemID = 1 << 20

// Item returns a copy of the item template with the given ID.
func (d *Data) Item(id int) (Item, bool) {
	it, ok := d.ItemTemplate(id)
	if !ok {
		return Item{}, false
	}
	return *it, true
}

// ItemTemplate returns the item template with the given ID.
func (d *Data) ItemTemplate(id int) (*Item, bool) {
	if id < 0 || id >= len(d.items) || d.items[id] == 0 {
		return nil, false
	}
	return &d.Items[d.items[id]-1], true
}

// loadItems reads the items domain.
func loadItems(fsys fs.FS, d *Data) error {
	if err := readJSON(fsys, ItemsFile, &d.Items); err != nil {
		return err
	}
	var p problems
	d.indexItems(&p)
	if err := p.err(); err != nil {
		return err
	}
	return nil
}

// indexItems checks the items and indexes them. It checks that:
//
//   - there are items, IDs are positive, below 2^20 and unique, and each
//     item has a name;
//   - the kind is known and matches the one of Weapon, Armor and Etc that
//     is set, and the item list types fit the kind;
//   - the body part is a Mobius body part and the mask is its mask;
//   - enums are valid, numbers are not negative, and stat functions and
//     skills are sane.
func (d *Data) indexItems(p *problems) {
	if len(d.Items) == 0 {
		p.addf("%s: no items", ItemsFile)
	}
	maxID := 0
	for _, it := range d.Items {
		if it.ID > maxID && it.ID < maxItemID {
			maxID = it.ID
		}
	}
	d.items = make([]int32, maxID+1)
	for i := range d.Items {
		it := &d.Items[i]
		where := fmt.Sprintf("%s: item %d (%q)", ItemsFile, it.ID, it.Name)
		if it.ID <= 0 || it.ID >= maxItemID {
			p.addf("%s: item ID must be in 1-%d", where, maxItemID-1)
			continue
		}
		if d.items[it.ID] != 0 {
			p.addf("%s: duplicate item ID", where)
			continue
		}
		d.items[it.ID] = int32(i + 1) //nolint:gosec // fewer than 2^31 items
		if it.DisplayID == 0 {
			it.DisplayID = it.ID
		}
		it.check(p, where)
	}
}

func (it *Item) check(p *problems, where string) {
	if it.Name == "" {
		p.addf("%s: no name", where)
	}
	types := [2]int{it.Type1, it.Type2}
	switch it.Type {
	case ItemTypeWeapon:
		if it.Weapon == nil || it.Armor != nil || it.Etc != nil {
			p.addf("%s: a weapon must have weapon values only", where)
		} else if !it.Weapon.Type.Valid() {
			p.addf("%s: invalid weapon type %d", where, it.Weapon.Type)
		}
		if types != [2]int{0, 0} {
			p.addf("%s: item list types %v, want [0 0] for a weapon", where, types)
		}
	case ItemTypeArmor:
		if it.Armor == nil || it.Weapon != nil || it.Etc != nil {
			p.addf("%s: an armor must have armor values only", where)
		} else if !it.Armor.Type.Valid() {
			p.addf("%s: invalid armor type %d", where, it.Armor.Type)
		}
		if types != [2]int{0, 2} && types != [2]int{1, 1} {
			p.addf("%s: item list types %v, want [0 2] or [1 1] for an armor", where, types)
		}
	case ItemTypeEtc:
		if it.Etc == nil || it.Weapon != nil || it.Armor != nil {
			p.addf("%s: an etc item must have etc values only", where)
		} else if !it.Etc.Type.Valid() {
			p.addf("%s: invalid etc item type %d", where, it.Etc.Type)
		}
		if it.Type1 != 4 || it.Type2 < 3 || it.Type2 > 5 {
			p.addf("%s: item list types %v, want 4 and 3-5 for an etc item", where, types)
		}
	default:
		p.addf("%s: unknown type %q (want %s, %s or %s)", where, it.Type, ItemTypeWeapon, ItemTypeArmor, ItemTypeEtc)
	}

	if it.BodyPart != "" {
		part, ok := BodyPartByName(it.BodyPart)
		switch {
		case !ok:
			p.addf("%s: unknown body part %q", where, it.BodyPart)
		case it.BodyPartMask != part.Mask():
			p.addf("%s: body part mask %#x, want %#x for %s", where, it.BodyPartMask, part.Mask(), it.BodyPart)
		}
	} else if it.BodyPartMask != 0 {
		p.addf("%s: body part mask %#x without a body part", where, it.BodyPartMask)
	}

	if !it.Grade.Valid() || !it.Material.Valid() || !it.DefaultAction.Valid() {
		p.addf("%s: invalid grade, material or default action", where)
	}
	if min(it.CrystalCount, it.Weight, it.Price, it.Duration, it.Time, it.AutoDestroyTime,
		it.ReuseDelay, it.SharedReuseGroup, it.EquipReuseDelay, it.UseSkillDisTime, it.DefaultEnchant) < 0 {
		p.addf("%s: numbers must not be negative", where)
	}
	if w := it.Weapon; w != nil && min(w.SoulShots, w.SpiritShots, w.MPConsume, w.ReducedSoulShot,
		w.ReducedSoulShotChance, w.ReducedMPConsume, w.ReducedMPConsumeChance, w.AttackRadius, w.AttackAngle) < 0 {
		p.addf("%s: weapon numbers must not be negative", where)
	}
	if e := it.Etc; e != nil && e.ExtractableCountMin > e.ExtractableCountMax {
		p.addf("%s: extractable count min %d is above max %d", where, e.ExtractableCountMin, e.ExtractableCountMax)
	}
	for _, st := range it.Stats {
		if !st.Stat.Valid() || !st.Op.Valid() {
			p.addf("%s: invalid stat function %+v", where, st)
			break
		}
	}
	for _, sk := range it.Skills {
		if sk.ID <= 0 || sk.Level <= 0 || !sk.Type.Valid() || sk.Chance < 0 {
			p.addf("%s: invalid skill %+v", where, sk)
			break
		}
	}
	if err := it.Conditions.check(); err != nil {
		p.addf("%s: conditions: %v", where, err)
	}
}

// check reports an element without a name in the tree.
func (e *XMLElement) check() error {
	if e == nil {
		return nil
	}
	if e.Name == "" {
		return errors.New("element without a name")
	}
	for i := range e.Children {
		if err := e.Children[i].check(); err != nil {
			return err
		}
	}
	return nil
}
