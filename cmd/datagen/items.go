package main

import (
	"cmp"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// itemsDir holds the item files. Its custom/ folder is left out: Mobius
// loads it only with CustomItemsLoad, an extra L2Go keeps off.
const itemsDir = statsDir + "/items"

// Item IDs with a Mobius rule of their own.
const (
	adenaID        = 57
	ancientAdenaID = 5575
)

func init() { registerDomain("items", genItems) }

// genItems writes every item template.
func genItems(o options) error {
	items, err := readItems(o)
	if err != nil {
		return err
	}
	return addJSONLines(o.out, data.ItemsFile, items)
}

// readItems reads the item files directly in itemsDir, sorted by item ID.
func readItems(o options) ([]data.Item, error) {
	files, err := fs.Glob(o.src, itemsDir+"/*.xml")
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s: no item files found", itemsDir)
	}
	found := make(map[int]string) // item ID -> file
	var items []data.Item
	for _, file := range files {
		root, err := readXMLTree(o.src, file)
		if err != nil {
			return nil, err
		}
		if err := root.checkChildren("item"); err != nil {
			return nil, err
		}
		for _, n := range root.children {
			it, err := readItem(n)
			if err != nil {
				return nil, err
			}
			if prev, dup := found[it.ID]; dup {
				return nil, n.errorf("item %d is also defined in %s", it.ID, prev)
			}
			found[it.ID] = file
			items = append(items, it)
		}
	}
	slices.SortFunc(items, func(a, b data.Item) int { return cmp.Compare(a.ID, b.ID) })
	return items, nil
}

// The <set> names of the item files, by the Mobius class that reads them.
var (
	commonSets = []string{
		"icon", "displayId", "weight", "material", "equip_reuse_delay", "duration", "time",
		"auto_destroy_time", "bodypart", "price", "crystal_type", "crystal_count", "is_stackable",
		"is_sellable", "is_dropable", "is_destroyable", "is_tradable", "is_questitem",
		"is_depositable", "element_enabled", "enchant_enabled", "is_freightable",
		"allow_self_resurrection", "is_oly_restricted", "for_npc", "immediate_effect",
		"ex_immediate_effect", "default_action", "useSkillDisTime", "enchanted", "reuse_delay",
		"shared_reuse_group",
	}
	weaponSets = []string{
		"weapon_type", "is_magic_weapon", "soulshots", "spiritshots", "mp_consume", "damage_range",
		"reduced_soulshot", "reduced_mp_consume", "isForceEquip", "isAttackWeapon", "useWeaponSkillsOnly",
	}
	armorSets = []string{"armor_type"}
	etcSets   = []string{"etcitem_type", "handler", "extractableCountMin", "extractableCountMax", "blessed"}
	// Sets that Mobius never uses: attack_range is read into a weapon
	// field nothing reads, the others are not read at all.
	unusedSets = []string{"attack_range", "recipe_id", "item_type", "equip_condition", "use_condition", "is_premium"}
)

// itemSets holds the <set> values of one item and remembers which were
// used, so a name nothing reads is caught.
type itemSets struct {
	n    *xmlNode
	vals map[string]string
}

func (s itemSets) str(name, def string) string {
	if v, ok := s.vals[name]; ok {
		return v
	}
	return def
}

func (s itemSets) int(name string, def int) (int, error) {
	v, ok := s.vals[name]
	if !ok {
		return def, nil
	}
	i, err := parseJavaInt(v)
	if err != nil {
		return 0, s.n.errorf("set %s: %v", name, err)
	}
	return i, nil
}

func (s itemSets) bool(name string, def bool) (bool, error) {
	v, ok := s.vals[name]
	if !ok {
		return def, nil
	}
	b, err := parseJavaBool(v)
	if err != nil {
		return false, s.n.errorf("set %s: %v", name, err)
	}
	return b, nil
}

// enum reads an enum value with parse, or returns def.
func enumSet[T any](s itemSets, name string, def T, parse func(string) (T, bool)) (T, error) {
	v, ok := s.vals[name]
	if !ok {
		return def, nil
	}
	e, ok := parse(v)
	if !ok {
		return def, s.n.errorf("set %s: unknown value %q", name, v)
	}
	return e, nil
}

// readItem reads one <item> as Mobius DocumentItem, ItemTemplate and its
// subclasses do.
func readItem(n *xmlNode) (data.Item, error) {
	if err := n.checkAttrs("id", "type", "name"); err != nil {
		return data.Item{}, err
	}
	id, err := n.intAttr("id")
	if err != nil {
		return data.Item{}, err
	}
	kind, _ := n.attr("type")
	name, ok := n.attr("name")
	if !ok || name == "" {
		return data.Item{}, n.errorf("item %d has no name", id)
	}
	switch kind {
	case data.ItemTypeWeapon, data.ItemTypeArmor, data.ItemTypeEtc:
	default:
		return data.Item{}, n.errorf("item %d: unknown type %q", id, kind)
	}

	sets := itemSets{n: n, vals: make(map[string]string)}
	var stats, skills, conditions *xmlNode
	created := false // Mobius creates the item at <stats> or <conditions>
	for _, c := range n.children {
		switch c.name {
		case "set":
			if created {
				return data.Item{}, c.errorf("item %d: <set> after <stats> or <conditions>; Mobius does not load such an item", id)
			}
			if err := c.checkAttrs("name", "val"); err != nil {
				return data.Item{}, err
			}
			sn, _ := c.attr("name")
			sv, ok := c.attr("val")
			if !ok {
				return data.Item{}, c.errorf("item %d: set %q has no val", id, sn)
			}
			sn, sv = strings.TrimSpace(sn), strings.TrimSpace(sv)
			if _, dup := sets.vals[sn]; dup {
				return data.Item{}, c.errorf("item %d: set %q given twice", id, sn)
			}
			if sv != "" { // Mobius skips empty values
				sets.vals[sn] = sv
			}
		case "stats":
			if stats != nil {
				return data.Item{}, c.errorf("item %d: <stats> given twice", id)
			}
			stats, created = c, true
		case "skills":
			if skills != nil {
				return data.Item{}, c.errorf("item %d: <skills> given twice", id)
			}
			skills = c
		case "conditions":
			if conditions != nil {
				return data.Item{}, c.errorf("item %d: <conditions> given twice", id)
			}
			conditions, created = c, true
		default:
			return data.Item{}, c.errorf("item %d: unknown element", id)
		}
	}
	for sn := range sets.vals {
		if !slices.Contains(commonSets, sn) && !slices.Contains(weaponSets, sn) && !slices.Contains(armorSets, sn) &&
			!slices.Contains(etcSets, sn) && !slices.Contains(unusedSets, sn) {
			return data.Item{}, n.errorf("item %d: unknown set %q", id, sn)
		}
	}

	it := data.Item{ID: id, Name: name, Type: kind}
	if err := readCommonSets(sets, &it); err != nil {
		return it, err
	}
	switch kind {
	case data.ItemTypeWeapon:
		err = readWeaponSets(sets, &it)
	case data.ItemTypeArmor:
		err = readArmorSets(sets, &it)
	case data.ItemTypeEtc:
		err = readEtcSets(sets, &it)
	}
	if err != nil {
		return it, err
	}
	if stats != nil {
		if err := readItemStats(stats, &it); err != nil {
			return it, err
		}
	}
	if skills != nil {
		if err := readItemSkills(skills, &it); err != nil {
			return it, err
		}
	}
	if conditions != nil {
		if it.Conditions, err = readConditions(conditions); err != nil {
			return it, err
		}
	}
	return it, nil
}

// readCommonSets reads what Mobius ItemTemplate.set reads.
func readCommonSets(s itemSets, it *data.Item) error {
	var err error
	ints := []struct {
		dst  *int
		name string
		def  int
	}{
		{&it.Weight, "weight", 0},
		{&it.Price, "price", 0},
		{&it.CrystalCount, "crystal_count", 0},
		{&it.UseSkillDisTime, "useSkillDisTime", 0},
		{&it.DefaultEnchant, "enchanted", 0},
		{&it.ReuseDelay, "reuse_delay", 0},
		{&it.SharedReuseGroup, "shared_reuse_group", 0},
		{&it.EquipReuseDelay, "equip_reuse_delay", 0},
		{&it.AutoDestroyTime, "auto_destroy_time", 0},
	}
	for _, f := range ints {
		if *f.dst, err = s.int(f.name, f.def); err != nil {
			return err
		}
	}
	it.EquipReuseDelay *= 1000 // seconds in the XML
	it.AutoDestroyTime *= 1000
	if it.AutoDestroyTime < 0 {
		it.AutoDestroyTime = 0 // 0 or less: the configured default
	}

	// Mobius uses -1 for "none"; the data uses 0, which no item has.
	for _, f := range []struct {
		dst  *int
		name string
	}{{&it.Duration, "duration"}, {&it.Time, "time"}} {
		v, err := s.int(f.name, -1)
		if err != nil {
			return err
		}
		if v == 0 || v < -1 {
			return s.n.errorf("item %d: set %s %d: want a positive number of minutes", it.ID, f.name, v)
		}
		*f.dst = max(v, 0)
	}

	if displayID, err := s.int("displayId", it.ID); err != nil {
		return err
	} else if displayID != it.ID {
		it.DisplayID = displayID
	}

	bools := []struct {
		dst  *bool
		name string
		def  bool
	}{
		{&it.Stackable, "is_stackable", false},
		{&it.Sellable, "is_sellable", true},
		{&it.Droppable, "is_dropable", true},
		{&it.Destroyable, "is_destroyable", true},
		{&it.Tradable, "is_tradable", true},
		{&it.QuestItem, "is_questitem", false},
		{&it.Depositable, "is_depositable", true},
		{&it.Elementable, "element_enabled", false},
		{&it.Enchantable, "enchant_enabled", false},
		{&it.Freightable, "is_freightable", false},
		{&it.AllowSelfResurrection, "allow_self_resurrection", false},
		{&it.OlyRestricted, "is_oly_restricted", false},
		{&it.ForNpc, "for_npc", false},
		{&it.ImmediateEffect, "immediate_effect", false},
		{&it.ExImmediateEffect, "ex_immediate_effect", false},
	}
	for _, f := range bools {
		if *f.dst, err = s.bool(f.name, f.def); err != nil {
			return err
		}
	}

	it.Icon = s.str("icon", "")
	if it.Material, err = enumSet(s, "material", data.MaterialSteel, data.ParseMaterialType); err != nil {
		return err
	}
	if it.Grade, err = enumSet(s, "crystal_type", data.CrystalNone, data.ParseCrystalType); err != nil {
		return err
	}
	if it.DefaultAction, err = enumSet(s, "default_action", data.ActionNone, data.ParseActionType); err != nil {
		return err
	}
	bodyPart := s.str("bodypart", "none")
	part, ok := data.BodyPartByName(bodyPart)
	if !ok {
		return s.n.errorf("item %d: unknown body part %q", it.ID, bodyPart)
	}
	if part != data.BodyPartNone {
		it.BodyPart, it.BodyPartMask = bodyPart, part.Mask()
	}

	id := it.ID
	it.Common = id >= 11605 && id <= 12361
	it.Hero = (id >= 6611 && id <= 6621) || (id >= 9388 && id <= 9390) || id == 6842
	it.PvP = (id >= 10667 && id <= 10835) || (id >= 12852 && id <= 12977) || (id >= 14363 && id <= 14525) ||
		id == 14528 || id == 14529 || id == 14558 || (id >= 15913 && id <= 16024) || (id >= 16134 && id <= 16147) ||
		id == 16149 || id == 16151 || id == 16153 || id == 16155 || id == 16157 || id == 16159 ||
		(id >= 16168 && id <= 16176) || (id >= 16179 && id <= 16220)
	return nil
}

// readWeaponSets reads what Mobius Weapon.set reads.
func readWeaponSets(s itemSets, it *data.Item) error {
	typ, ok := data.ParseWeaponType(strings.ToUpper(s.str("weapon_type", "none")))
	if !ok {
		return s.n.errorf("item %d: unknown weapon type %q", it.ID, s.str("weapon_type", ""))
	}
	w := &data.WeaponInfo{Type: typ, AttackRadius: 40}
	var err error
	if w.Magic, err = s.bool("is_magic_weapon", false); err != nil {
		return err
	}
	for _, f := range []struct {
		dst  *int
		name string
	}{{&w.SoulShots, "soulshots"}, {&w.SpiritShots, "spiritshots"}, {&w.MPConsume, "mp_consume"}} {
		if *f.dst, err = s.int(f.name, 0); err != nil {
			return err
		}
	}
	// damage_range is "?;?;radius;angle"; Mobius reads the last two.
	if v, ok := s.vals["damage_range"]; ok {
		parts := strings.Split(v, ";")
		if len(parts) != 4 {
			return s.n.errorf("item %d: damage_range %q: want 4 numbers", it.ID, v)
		}
		if w.AttackRadius, err = parseJavaInt(parts[2]); err != nil {
			return s.n.errorf("item %d: damage_range: %v", it.ID, err)
		}
		if w.AttackAngle, err = parseJavaInt(parts[3]); err != nil {
			return s.n.errorf("item %d: damage_range: %v", it.ID, err)
		}
	}
	// reduced_soulshot and reduced_mp_consume are "chance,count".
	for _, f := range []struct {
		name          string
		chance, count *int
	}{
		{"reduced_soulshot", &w.ReducedSoulShotChance, &w.ReducedSoulShot},
		{"reduced_mp_consume", &w.ReducedMPConsumeChance, &w.ReducedMPConsume},
	} {
		v, ok := s.vals[f.name]
		if !ok {
			continue
		}
		chance, count, found := strings.Cut(v, ",")
		if !found {
			return s.n.errorf("item %d: %s %q: want chance,count", it.ID, f.name, v)
		}
		if *f.chance, err = parseJavaInt(chance); err != nil {
			return s.n.errorf("item %d: %s: %v", it.ID, f.name, err)
		}
		if *f.count, err = parseJavaInt(count); err != nil {
			return s.n.errorf("item %d: %s: %v", it.ID, f.name, err)
		}
	}
	if w.ForceEquip, err = s.bool("isForceEquip", false); err != nil {
		return err
	}
	if w.AttackWeapon, err = s.bool("isAttackWeapon", true); err != nil {
		return err
	}
	if w.WeaponSkillsOnly, err = s.bool("useWeaponSkillsOnly", false); err != nil {
		return err
	}
	if it.ReuseDelay == 0 && typ.Ranged() {
		it.ReuseDelay = 1500
	}
	it.Weapon = w
	it.Type1, it.Type2 = 0, 0
	return nil
}

// readArmorSets reads what Mobius Armor.set reads.
func readArmorSets(s itemSets, it *data.Item) error {
	typ, err := enumSet(s, "armor_type", data.ArmorNone, data.ParseArmorType)
	if err != nil {
		return err
	}
	switch it.Part() {
	case data.BodyPartNeck, data.BodyPartLEar, data.BodyPartREar, data.BodyPartLREar,
		data.BodyPartLFinger, data.BodyPartRFinger, data.BodyPartLRFinger:
		it.Type1, it.Type2 = 0, 2
	default:
		if typ == data.ArmorNone && it.Part() == data.BodyPartLHand {
			typ = data.ArmorShield
		}
		it.Type1, it.Type2 = 1, 1
	}
	it.Armor = &data.ArmorInfo{Type: typ}
	return nil
}

// readEtcSets reads what Mobius EtcItem.set reads.
func readEtcSets(s itemSets, it *data.Item) error {
	typ, err := enumSet(s, "etcitem_type", data.EtcNone, data.ParseEtcItemType)
	if err != nil {
		return err
	}
	switch it.DefaultAction {
	case data.ActionSoulshot, data.ActionSummonSoulshot, data.ActionSummonSpiritshot, data.ActionSpiritshot:
		typ = data.EtcShot
	}
	e := &data.EtcInfo{Type: typ, Handler: s.str("handler", "")}
	if e.ExtractableCountMin, err = s.int("extractableCountMin", 0); err != nil {
		return err
	}
	if e.ExtractableCountMax, err = s.int("extractableCountMax", 0); err != nil {
		return err
	}
	if e.Blessed, err = s.bool("blessed", false); err != nil {
		return err
	}
	if (it.DefaultAction == data.ActionSpiritshot || it.DefaultAction == data.ActionSoulshot) && strings.Contains(it.Name, "Blessed") {
		e.Blessed = true
	}
	it.Type1, it.Type2 = 4, 5
	switch {
	case it.QuestItem:
		it.Type2 = 3
	case it.ID == adenaID || it.ID == ancientAdenaID:
		it.Type2 = 4
	}
	it.Etc = e
	return nil
}

// itemStatOps is the function DocumentItem attaches for each stat of
// <stats>. Mobius ignores any other stat with a warning; datagen refuses it.
var itemStatOps = map[data.Stat]data.StatOp{
	data.StatPowerAttack:      data.OpSet,
	data.StatPowerAttackSpeed: data.OpSet,
	data.StatPowerAttackRange: data.OpSet,
	data.StatCriticalRate:     data.OpSet,
	data.StatMagicAttack:      data.OpSet,
	data.StatShieldDefence:    data.OpSet,
	data.StatShieldRate:       data.OpSet,
	data.StatAccuracyCombat:   data.OpAdd,
	data.StatEvasionRate:      data.OpAdd,
	data.StatRandomDamage:     data.OpAdd,
	data.StatPowerDefence:     data.OpAdd,
	data.StatMagicDefence:     data.OpAdd,
	data.StatMaxMP:            data.OpAdd,
	data.StatFirePower:        data.OpAdd,
	data.StatWaterPower:       data.OpAdd,
	data.StatWindPower:        data.OpAdd,
	data.StatEarthPower:       data.OpAdd,
	data.StatHolyPower:        data.OpAdd,
	data.StatDarkPower:        data.OpAdd,
	data.StatFireRes:          data.OpAdd,
	data.StatWaterRes:         data.OpAdd,
	data.StatWindRes:          data.OpAdd,
	data.StatEarthRes:         data.OpAdd,
	data.StatHolyRes:          data.OpAdd,
	data.StatDarkRes:          data.OpAdd,
	data.StatMagicSuccessRes:  data.OpMul,
}

// readItemStats reads <stats> as DocumentItem does: one function per stat,
// all at order 0, then the enchant functions of armor and weapons.
func readItemStats(n *xmlNode, it *data.Item) error {
	if len(n.attrs) > 0 {
		return n.errorf("item %d: unexpected attributes", it.ID)
	}
	if err := n.checkChildren("stat"); err != nil {
		return err
	}
	for _, c := range n.children {
		if err := c.checkAttrs("type"); err != nil {
			return err
		}
		name, _ := c.attr("type")
		stat, ok := data.ParseStat(name)
		if !ok {
			return c.errorf("item %d: unknown stat %q", it.ID, name)
		}
		op, ok := itemStatOps[stat]
		if !ok {
			return c.errorf("item %d: Mobius does not handle stat %q in <stats>", it.ID, name)
		}
		if len(c.children) > 0 {
			return c.errorf("item %d: want only a value", it.ID)
		}
		v, err := parseJavaFloat(c.text)
		if err != nil {
			return c.errorf("item %d: %v", it.ID, err)
		}
		it.Stats = append(it.Stats, data.ItemStat{Stat: stat, Op: op, Value: v})
	}
	enchant := func(s data.Stat) { it.Stats = append(it.Stats, data.ItemStat{Stat: s, Op: data.OpEnchant}) }
	switch {
	case it.Armor != nil && it.Armor.Type == data.ArmorShield:
		enchant(data.StatShieldDefence)
	case it.Armor != nil && it.Type1 == 0 && it.Type2 == 2:
		enchant(data.StatMagicDefence)
	case it.Armor != nil:
		enchant(data.StatPowerDefence)
	case it.Weapon != nil:
		enchant(data.StatPowerAttack)
		enchant(data.StatMagicAttack)
	}
	return nil
}

// readItemSkills reads <skills>. Mobius keeps one skill of each special
// type and drops broken ones; datagen refuses what Mobius would drop.
func readItemSkills(n *xmlNode, it *data.Item) error {
	if len(n.attrs) > 0 {
		return n.errorf("item %d: unexpected attributes", it.ID)
	}
	if err := n.checkChildren("skill"); err != nil {
		return err
	}
	special := make(map[data.ItemSkillType]bool)
	for _, c := range n.children {
		if err := c.checkAttrs("id", "level", "type", "type_chance"); err != nil {
			return err
		}
		sk := data.ItemSkill{}
		var err error
		if sk.ID, err = c.intAttr("id"); err != nil {
			return err
		}
		if sk.Level, err = c.intAttr("level"); err != nil {
			return err
		}
		if v, ok := c.attr("type"); ok {
			if sk.Type, ok = data.ParseItemSkillType(v); !ok {
				return c.errorf("item %d: unknown skill type %q", it.ID, v)
			}
		}
		chance := 100
		if _, ok := c.attr("type_chance"); ok {
			if chance, err = c.intAttr("type_chance"); err != nil {
				return err
			}
		}
		if sk.ID <= 0 || sk.Level <= 0 {
			return c.errorf("item %d: skill %d level %d: Mobius ignores it", it.ID, sk.ID, sk.Level)
		}
		switch sk.Type {
		case data.ItemSkillNormal:
			if _, ok := c.attr("type_chance"); ok {
				return c.errorf("item %d: type_chance on a skill of type NORMAL", it.ID)
			}
		case data.ItemSkillOnCriticalSkill, data.ItemSkillOnMagicSkill:
			if it.Weapon == nil {
				return c.errorf("item %d: %s skill on a non-weapon, which Mobius ignores", it.ID, sk.Type)
			}
			if chance <= 0 {
				return c.errorf("item %d: chance %d: Mobius ignores the skill", it.ID, chance)
			}
			sk.Chance = chance
		case data.ItemSkillOnEnchant4:
			if it.Etc != nil {
				return c.errorf("item %d: %s skill on an etc item, which Mobius ignores", it.ID, sk.Type)
			}
		}
		if sk.Type != data.ItemSkillNormal {
			if special[sk.Type] {
				return c.errorf("item %d: two %s skills; Mobius keeps only the last", it.ID, sk.Type)
			}
			special[sk.Type] = true
			if sk.Type != data.ItemSkillOnCriticalSkill && sk.Type != data.ItemSkillOnMagicSkill {
				if _, ok := c.attr("type_chance"); ok {
					return c.errorf("item %d: type_chance on a %s skill", it.ID, sk.Type)
				}
			}
		}
		it.Skills = append(it.Skills, sk)
	}
	return nil
}

// readConditions keeps <conditions> as an XMLElement tree. Mobius reads
// its first child element only, so exactly one is allowed.
func readConditions(n *xmlNode) (*data.XMLElement, error) {
	if err := n.checkAttrs("msg", "msgId", "addName"); err != nil {
		return nil, err
	}
	if len(n.children) != 1 {
		return nil, n.errorf("want exactly one condition element, have %d", len(n.children))
	}
	e, err := xmlElement(n)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// xmlElement converts an element without text into an XMLElement.
func xmlElement(n *xmlNode) (data.XMLElement, error) {
	if n.text != "" {
		return data.XMLElement{}, n.errorf("unexpected text %q", n.text)
	}
	e := data.XMLElement{Name: n.name}
	if len(n.attrs) > 0 {
		e.Attrs = make(map[string]string, len(n.attrs))
		for _, a := range n.attrs {
			if _, dup := e.Attrs[a.Name.Local]; dup {
				return data.XMLElement{}, n.errorf("attribute %q given twice", a.Name.Local)
			}
			e.Attrs[a.Name.Local] = a.Value
		}
	}
	for _, c := range n.children {
		ce, err := xmlElement(c)
		if err != nil {
			return data.XMLElement{}, err
		}
		e.Children = append(e.Children, ce)
	}
	return e, nil
}

// knownSetNames returns every <set> name datagen accepts, for tests.
func knownSetNames() []string {
	names := slices.Concat(commonSets, weaponSets, armorSets, etcSets, unusedSets)
	slices.Sort(names)
	return names
}
