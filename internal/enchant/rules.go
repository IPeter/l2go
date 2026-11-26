package enchant

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// The item list types of Mobius ItemTemplate.
const (
	Type2Weapon      = 0
	Type2ShieldArmor = 1
	Type2Accessory   = 2
)

// Templates looks up item templates; *data.Data implements it.
type Templates interface {
	ItemTemplate(id int) (*data.Item, bool)
}

// Kind is what a failed enchant does to the item, from the scroll item's
// etc item type (Mobius EnchantScroll).
type Kind uint8

// The scroll kinds.
const (
	// KindNormal scrolls crystallize the item when they fail
	// (SCRL_ENCHANT_*; crystal scrolls are normal scrolls with a bonus rate
	// of 100).
	KindNormal Kind = iota
	// KindBlessed scrolls reset the item to +0 (BLESS_SCRL_ENCHANT_*).
	KindBlessed
	// KindSafe scrolls keep the item's level (ANCIENT_CRYSTAL_ENCHANT_*).
	KindSafe
)

// String returns the kind's name.
func (k Kind) String() string {
	switch k {
	case KindBlessed:
		return "blessed"
	case KindSafe:
		return "safe"
	}
	return "normal"
}

// ScrollInfo is a scroll with what its item template says about it.
type ScrollInfo struct {
	Scroll
	// Weapon is set for weapon scrolls (*_WP), else the scroll is for armor
	// and accessories.
	Weapon bool
	Kind   Kind

	items map[int]int // item ID -> scroll group, nil when not bound
}

// scrollKinds are the etc item types a scroll's item may have
// (AbstractEnchantItem.ENCHANT_TYPES); SCRL_INC_ENCHANT_PROP_* are allowed
// and are armor scrolls of the normal kind, as in Mobius.
var scrollKinds = map[data.EtcItemType]struct {
	weapon bool
	kind   Kind
}{
	data.EtcAncientCrystalEnchantAm: {false, KindSafe},
	data.EtcAncientCrystalEnchantWp: {true, KindSafe},
	data.EtcBlessScrlEnchantAm:      {false, KindBlessed},
	data.EtcBlessScrlEnchantWp:      {true, KindBlessed},
	data.EtcScrlEnchantAm:           {false, KindNormal},
	data.EtcScrlEnchantWp:           {true, KindNormal},
	data.EtcScrlIncEnchantPropAm:    {false, KindNormal},
	data.EtcScrlIncEnchantPropWp:    {false, KindNormal},
}

// Table is the enchant data ready for the rules. It is not changed after
// NewTable and is safe for concurrent use.
type Table struct {
	data      Data
	scrolls   map[int]*ScrollInfo
	bound     map[int]bool // item IDs some scroll is bound to
	blacklist map[int]bool

	maxWeapon, maxArmor, maxAccessory int
}

// NewTable builds the rules of d, which must pass Check. Scrolls whose item
// the templates do not have, or whose item is not an enchant scroll, are
// left out and returned, as Mobius EnchantItemData leaves them out with a
// warning. blacklist is EnchantBlackList: items that cannot be enchanted.
func NewTable(d Data, items Templates, blacklist []int) (*Table, []int) {
	t := &Table{
		data:      d,
		scrolls:   make(map[int]*ScrollInfo, len(d.Scrolls)),
		bound:     make(map[int]bool),
		blacklist: make(map[int]bool, len(blacklist)),
	}
	for _, id := range blacklist {
		t.blacklist[id] = true
	}
	var skipped []int
	for _, s := range d.Scrolls {
		tmpl, ok := items.ItemTemplate(s.ID)
		if !ok || tmpl.Etc == nil {
			skipped = append(skipped, s.ID)
			continue
		}
		k, ok := scrollKinds[tmpl.Etc.Type]
		if !ok {
			skipped = append(skipped, s.ID)
			continue
		}
		info := &ScrollInfo{Scroll: s, Weapon: k.weapon, Kind: k.kind}
		if len(s.Items) > 0 {
			info.items = make(map[int]int, len(s.Items))
			for _, it := range s.Items {
				info.items[it.ID] = it.ScrollGroup
				t.bound[it.ID] = true
			}
		}
		t.scrolls[s.ID] = info
	}
	t.maxWeapon, t.maxArmor, t.maxAccessory = d.maxLevels()
	return t, skipped
}

// Scroll returns the scroll of an item, nil if the item is no enchant
// scroll (EnchantItemData.getEnchantScroll).
func (t *Table) Scroll(itemID int) *ScrollInfo {
	if t == nil {
		return nil
	}
	return t.scrolls[itemID]
}

// Scrolls returns the scroll item IDs, sorted.
func (t *Table) Scrolls() []int {
	ids := make([]int, 0, len(t.scrolls))
	for id := range t.scrolls {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// Enchantable reports whether items of the template can be enchanted at
// all: the template allows it and it is not in EnchantBlackList (Mobius
// ItemTemplate.isEnchantable).
func (t *Table) Enchantable(tmpl *data.Item) bool {
	return tmpl.Enchantable && (t == nil || !t.blacklist[tmpl.ID])
}

// Target is an item to enchant.
type Target struct {
	Template *data.Item
	// Enchant is its enchant level.
	Enchant int
	// Carried is set when it is in the inventory or worn (Mobius
	// Item.isEnchantable looks at the location).
	Carried bool
}

// Valid reports whether the scroll can be used on the item (Mobius
// EnchantScroll.isValid with AbstractEnchantItem.isValid): a bound scroll
// takes only its items, and an unbound one no item another scroll is bound
// to; the item must be carried and enchantable; a weapon scroll takes
// weapons, an armor scroll armor, shields and accessories; the level must
// be below the scroll's maximum (unless 0), and the grades must match.
func (t *Table) Valid(s *ScrollInfo, it Target) bool {
	if s == nil || it.Template == nil {
		return false
	}
	id := it.Template.ID
	if s.items != nil {
		if _, ok := s.items[id]; !ok {
			return false
		}
	} else if t.bound[id] && !t.boundOnlyTo(s.ID, id) {
		return false
	}
	switch {
	case !it.Carried || !t.Enchantable(it.Template):
		return false
	case !validType(s.Weapon, it.Template.Type2):
		return false
	case s.MaxEnchant != 0 && it.Enchant >= s.MaxEnchant:
		return false
	}
	return s.Grade == it.Template.Grade
}

// boundOnlyTo reports whether scroll is the only scroll bound to the item
// (Mobius skips the scroll itself when it looks for other scrolls).
func (t *Table) boundOnlyTo(scroll, itemID int) bool {
	for id, s := range t.scrolls {
		if id == scroll || s.items == nil {
			continue
		}
		if _, ok := s.items[itemID]; ok {
			return false
		}
	}
	return true
}

// validType is AbstractEnchantItem.isValidItemType.
func validType(weapon bool, type2 int) bool {
	switch type2 {
	case Type2Weapon:
		return weapon
	case Type2ShieldArmor, Type2Accessory:
		return !weapon
	}
	return false
}

// Chance returns the chance in percent of the scroll's rate group for the
// item at its level, without the bonus rate, or -1 when the scroll group or
// the item's rate group is missing (Mobius EnchantScroll.getChance). A
// level no range covers takes the last range's chance.
func (t *Table) Chance(s *ScrollInfo, it Target) float64 {
	if s == nil || it.Template == nil {
		return -1
	}
	groupID := s.ScrollGroup
	if g, ok := s.items[it.Template.ID]; ok {
		groupID = g
	}
	sg := t.data.scrollGroup(groupID)
	if sg == nil {
		return -1
	}
	for _, r := range sg.Rates {
		if !r.accepts(it.Template) {
			continue
		}
		g := t.data.rateGroup(r.Group)
		if g == nil || len(g.Chances) == 0 {
			return -1
		}
		for _, c := range g.Chances {
			if c.Min <= it.Enchant && it.Enchant <= c.Max {
				return c.Chance
			}
		}
		return g.Chances[len(g.Chances)-1].Chance
	}
	return -1
}

// accepts is EnchantRateItem.validate.
func (r *RateItem) accepts(tmpl *data.Item) bool {
	switch {
	case len(r.ItemIDs) > 0 && !slices.Contains(r.ItemIDs, tmpl.ID):
		return false
	case r.Slots != 0 && tmpl.BodyPartMask&r.Slots == 0:
		return false
	case r.MagicWeapon != nil && magicWeapon(tmpl) != *r.MagicWeapon:
		return false
	}
	return true
}

// magicWeapon is ItemTemplate.isMagicWeapon: only weapons can be.
func magicWeapon(tmpl *data.Item) bool { return tmpl.Weapon != nil && tmpl.Weapon.Magic }

// Result is the outcome of an enchant (Mobius EnchantResultType).
type Result uint8

// The results.
const (
	// ResultError: the scroll is not valid for the item, or it has no
	// chance (-1); nothing changes but the scroll.
	ResultError Result = iota
	ResultSuccess
	ResultFailure
)

// String returns the result's name.
func (r Result) String() string {
	switch r {
	case ResultSuccess:
		return "success"
	case ResultFailure:
		return "failure"
	}
	return "error"
}

// Roll is EnchantScroll.calculateSuccess: an invalid scroll or a missing
// chance is ResultError without a draw; otherwise one draw of
// r.Float64(), and the enchant succeeds when 100 × it is below the chance
// plus the bonus rate, capped at 100.
func (t *Table) Roll(s *ScrollInfo, it Target, r rnd.Source) Result {
	if !t.Valid(s, it) {
		return ResultError
	}
	chance := t.Chance(s, it)
	if chance == -1 {
		return ResultError
	}
	final := min(chance+s.BonusRate, 100)
	if 100*r.Float64() < final {
		return ResultSuccess
	}
	return ResultFailure
}

// Gains reports whether a success raises the item's level: only when the
// rate group's own chance is above 0, so a crystal scroll at +16 succeeds
// without a gain (RequestEnchantItem).
func (t *Table) Gains(s *ScrollInfo, it Target) bool { return t.Chance(s, it) > 0 }

// CrystalCount is ItemTemplate.getCrystalCount(enchantLevel): the crystals
// an item of the template at the level gives when crystallized. Above +3,
// armor and accessories add their grade's armor bonus × (3 × level − 6) and
// weapons the weapon bonus × (2 × level − 3); from +1 to +3 the bonus ×
// level; other items and +0 the template's count.
func CrystalCount(tmpl *data.Item, enchant int) int {
	base := tmpl.CrystalCount
	if enchant <= 0 {
		return base
	}
	var bonus int
	switch tmpl.Type2 {
	case Type2ShieldArmor, Type2Accessory:
		bonus = tmpl.Grade.EnchantBonusArmor()
		if enchant > 3 {
			return base + bonus*(3*enchant-6)
		}
	case Type2Weapon:
		bonus = tmpl.Grade.EnchantBonusWeapon()
		if enchant > 3 {
			return base + bonus*(2*enchant-3)
		}
	default:
		return base
	}
	return base + bonus*enchant
}

// FailureCrystals returns the crystals a failed normal enchant gives for an
// item of the template at the level (RequestEnchantItem): the crystal item
// ID (0 when the item gives none) and CrystalCount − (count + 1) / 2, at
// least 1.
func FailureCrystals(tmpl *data.Item, enchant int) (crystalID, count int) {
	crystalID = tmpl.Grade.CrystalItemID()
	if crystalID == 0 || !tmpl.Crystallizable() {
		return 0, 0
	}
	count = CrystalCount(tmpl, enchant) - (tmpl.CrystalCount+1)/2
	return crystalID, max(count, 1)
}

// Announced reports whether reaching the level is announced to the players
// around with a firework (RequestEnchantItem): +7 and +15 for weapons, +6
// for armor and accessories (whose second level, 0, cannot be reached).
func Announced(tmpl *data.Item, level int) bool {
	if tmpl.Armor != nil {
		return level == 6 || level == 0
	}
	return level == 7 || level == 15
}

// MaxWeapon, MaxArmor and MaxAccessory are the derived maximum levels of
// Mobius EnchantItemGroupsData: the highest level a rate group of the kind
// can succeed from, plus one (16 with the Interlude data).
func (t *Table) MaxWeapon() int    { return t.maxWeapon }
func (t *Table) MaxArmor() int     { return t.maxArmor }
func (t *Table) MaxAccessory() int { return t.maxAccessory }

// MaxEnchant returns the derived maximum level of items of the template:
// weapons, accessories, else armor (AdminCreateItem.getMaxEnchant).
func (t *Table) MaxEnchant(tmpl *data.Item) int {
	switch {
	case tmpl.Weapon != nil:
		return t.maxWeapon
	case tmpl.Type2 == Type2Accessory:
		return t.maxAccessory
	}
	return t.maxArmor
}

// OverEnchanted reports whether an item of the template at the level is
// above its derived maximum (the OverEnchantProtection test of Mobius
// EnterWorld and UseItem): weapons, accessories and other armor; other
// items never are.
func (t *Table) OverEnchanted(tmpl *data.Item, enchant int) bool {
	switch {
	case tmpl.Weapon != nil:
		return enchant > t.maxWeapon
	case tmpl.Type2 == Type2Accessory:
		return enchant > t.maxAccessory
	case tmpl.Armor != nil:
		return enchant > t.maxArmor
	}
	return false
}

// CrystallizeLevel returns the Crystallize (248) level needed to
// crystallize items of the grade (RequestCrystallizeItem): 1 for none and
// D, then 2 for C up to 5 for S.
func CrystallizeLevel(g data.CrystalType) int {
	switch g {
	case data.CrystalC:
		return 2
	case data.CrystalB:
		return 3
	case data.CrystalA:
		return 4
	case data.CrystalS:
		return 5
	}
	return 1
}

// CanCrystallize reports whether a player with Crystallize at skillLevel (0
// without the skill) may crystallize items of the grade.
func CanCrystallize(g data.CrystalType, skillLevel int) bool {
	return skillLevel > 0 && skillLevel >= CrystallizeLevel(g)
}
