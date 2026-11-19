package main

import (
	"cmp"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Mobius data files of the players domain.
const (
	statsDir          = "dist/game/data/stats"
	playersDir        = statsDir + "/players"
	classListFile     = playersDir + "/classList.xml"
	templatesDir      = playersDir + "/templates"
	equipmentFile     = playersDir + "/initialEquipment.xml"
	experienceFile    = playersDir + "/experience.xml"
	experienceLossXML = playersDir + "/experienceLoss.xml"
	karmaLossXML      = playersDir + "/karmaLoss.xml"
	statBonusXML      = statsDir + "/statBonus.xml"
	hitConditionXML   = statsDir + "/hitConditionBonus.xml"
)

func init() { registerDomain("players", genPlayers) }

// genPlayers writes the class templates of every player class, the initial
// equipment and the player tables.
//
// Race and parent of each class come from the Mobius PlayerClass enum, which
// the server code uses; classList.xml only gives the names (its parents
// differ from the enum for two classes, which is reported as a warning).
func genPlayers(o options) error {
	classes, _, err := readPlayerClasses(o)
	if err != nil {
		return err
	}
	names, err := readClassList(o, classes)
	if err != nil {
		return err
	}
	exp, err := readExperience(o)
	if err != nil {
		return err
	}
	templates, err := readClassTemplates(o, classes, names, exp.MaxLevel)
	if err != nil {
		return err
	}
	equipment, err := readEquipment(o)
	if err != nil {
		return err
	}
	bonus, err := readStatBonus(o)
	if err != nil {
		return err
	}
	expLoss, err := readLevelTable(o, experienceLossXML, "experience")
	if err != nil {
		return err
	}
	karmaLoss, err := readLevelTable(o, karmaLossXML, "modifier")
	if err != nil {
		return err
	}
	hit, err := readHitConditionBonus(o)
	if err != nil {
		return err
	}
	for _, f := range []struct {
		name string
		v    any
	}{
		{data.ClassesFile, templates},
		{data.EquipmentFile, equipment},
		{data.StatBonusFile, bonus},
		{data.ExperienceFile, exp},
		{data.ExperienceLossFile, expLoss},
		{data.KarmaLossFile, karmaLoss},
		{data.HitConditionBonusFile, hit},
	} {
		if err := o.out.addJSON(f.name, f.v); err != nil {
			return err
		}
	}
	return nil
}

// readClassList returns the name of each class from classList.xml. Every
// PlayerClass must be listed, and nothing else.
func readClassList(o options, classes []playerClassDef) (map[int]string, error) {
	root, err := readXMLTree(o.src, classListFile)
	if err != nil {
		return nil, err
	}
	if err := root.checkChildren("class"); err != nil {
		return nil, err
	}
	byID := make(map[int]playerClassDef, len(classes))
	for _, c := range classes {
		byID[c.id] = c
	}
	names := make(map[int]string, len(root.children))
	for _, n := range root.children {
		if err := n.checkAttrs("classId", "name", "parentClassId"); err != nil {
			return nil, err
		}
		id, err := n.intAttr("classId")
		if err != nil {
			return nil, err
		}
		name, _ := n.attr("name")
		if name == "" {
			return nil, n.errorf("class %d has no name", id)
		}
		if _, dup := names[id]; dup {
			return nil, n.errorf("class %d is listed twice", id)
		}
		c, ok := byID[id]
		if !ok {
			return nil, n.errorf("class %d is not in %s", id, playerClassFile)
		}
		parent := -1
		if _, ok := n.attr("parentClassId"); ok {
			if parent, err = n.intAttr("parentClassId"); err != nil {
				return nil, err
			}
		}
		if parent != c.parent {
			o.warnf("%s: class %d (%s) has parent %d, but PlayerClass.%s has %d; using %d", classListFile, id, name, parent, c.java, c.parent, c.parent)
		}
		names[id] = name
	}
	for _, c := range classes {
		if _, ok := names[c.id]; !ok {
			return nil, fmt.Errorf("%s: PlayerClass.%s (%d) is not listed", classListFile, c.java, c.id)
		}
	}
	return names, nil
}

// readClassTemplates reads every template under templatesDir. Each player
// class must have exactly one, with one lvlUpgainData level for each level
// 1 to levels.
func readClassTemplates(o options, classes []playerClassDef, names map[int]string, levels int) ([]data.ClassTemplate, error) {
	var files []string
	err := fs.WalkDir(o.src, templatesDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".xml") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", templatesDir, err)
	}
	byID := make(map[int]playerClassDef, len(classes))
	for _, c := range classes {
		byID[c.id] = c
	}
	seen := make(map[int]string)
	var templates []data.ClassTemplate
	for _, file := range files {
		t, err := readClassTemplate(o, file, levels)
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[t.ClassID]; dup {
			return nil, fmt.Errorf("%s: class %d is also defined in %s", file, t.ClassID, prev)
		}
		seen[t.ClassID] = file
		c, ok := byID[t.ClassID]
		if !ok {
			return nil, fmt.Errorf("%s: class %d is not in %s", file, t.ClassID, playerClassFile)
		}
		if err := fillClass(&t, c, names[t.ClassID]); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		templates = append(templates, t)
	}
	for _, c := range classes {
		if _, ok := seen[c.id]; !ok {
			return nil, fmt.Errorf("%s: PlayerClass.%s (%d) has no template", templatesDir, c.java, c.id)
		}
	}
	slices.SortFunc(templates, func(a, b data.ClassTemplate) int { return cmp.Compare(a.ClassID, b.ClassID) })
	return templates, nil
}

// fillClass sets the identity of a template from the PlayerClass enum and
// checks it against the generated enums of internal/data.
func fillClass(t *data.ClassTemplate, c playerClassDef, name string) error {
	race, ok := data.ParseRace(c.race)
	pc := data.PlayerClass(c.id)
	parent, hasParent := pc.Parent()
	if !ok || !pc.Valid() || pc.Race() != race || hasParent != (c.parent >= 0) || (hasParent && int(parent) != c.parent) {
		return fmt.Errorf("class %d differs from the generated enums of internal/data; run go generate ./internal/data", c.id)
	}
	t.Name = name
	t.Race = race
	t.ParentID = c.parent
	return nil
}

// Mobius defaults of CreatureTemplate and PlayerTemplate for values a
// template may leave out.
const (
	defaultPAtkSpd     = 300
	defaultMAtkSpd     = 333
	defaultAttackRange = 40
	defaultCritRate    = 4
	defaultMCritRate   = 5
	defaultBreath      = 100
	defaultSafeFall    = 333
	defaultRunSpeed    = 120
	defaultWalkSpeed   = 50
)

// readClassTemplate reads one template file, as PlayerTemplateData does.
func readClassTemplate(o options, file string, levels int) (data.ClassTemplate, error) {
	root, err := readXMLTree(o.src, file)
	if err != nil {
		return data.ClassTemplate{}, err
	}
	if err := root.checkChildren("classId", "staticData", "lvlUpgainData"); err != nil {
		return data.ClassTemplate{}, err
	}
	t := data.ClassTemplate{
		ClassID: -1,
		PAtkSpd: defaultPAtkSpd, MAtkSpd: defaultMAtkSpd, AttackRange: defaultAttackRange,
		CritRate: defaultCritRate, MCritRate: defaultMCritRate, AttackType: data.WeaponFist,
		Breath: defaultBreath, SafeFall: defaultSafeFall,
		MoveSpeed: data.MoveSpeeds{Walk: -1, Run: -1, SlowSwim: -1, FastSwim: -1},
	}
	var haveStatic, haveLevels bool
	for _, n := range root.children {
		switch n.name {
		case "classId":
			if t.ClassID >= 0 {
				return t, n.errorf("given twice")
			}
			if t.ClassID, err = n.intText(); err != nil {
				return t, err
			}
		case "staticData":
			if haveStatic {
				return t, n.errorf("given twice")
			}
			haveStatic = true
			if err := readStaticData(n, &t); err != nil {
				return t, err
			}
		case "lvlUpgainData":
			if haveLevels {
				return t, n.errorf("given twice")
			}
			haveLevels = true
			if err := readLevels(n, &t, levels); err != nil {
				return t, err
			}
		}
	}
	switch {
	case t.ClassID < 0:
		return t, fmt.Errorf("%s: no classId", file)
	case !haveStatic:
		return t, fmt.Errorf("%s: no staticData", file)
	case !haveLevels:
		return t, fmt.Errorf("%s: no lvlUpgainData", file)
	}
	m := &t.MoveSpeed
	if m.Run < 0 {
		m.Run = defaultRunSpeed
	}
	if m.Walk < 0 {
		m.Walk = defaultWalkSpeed
	}
	if m.FastSwim < 0 {
		m.FastSwim = m.Run
	}
	if m.SlowSwim < 0 {
		m.SlowSwim = m.Walk
	}
	if t.MDef.LFinger != t.MDef.RFinger {
		// Mobius uses the right ring's value for both rings.
		return t, fmt.Errorf("%s: baseMDef lfinger %d differs from rfinger %d, which Mobius uses for both", file, t.MDef.LFinger, t.MDef.RFinger)
	}
	return t, nil
}

// readStaticData reads the staticData element of a template.
func readStaticData(sd *xmlNode, t *data.ClassTemplate) error {
	ints := map[string]*int{
		"baseSTR": &t.Stats.STR, "baseCON": &t.Stats.CON, "baseDEX": &t.Stats.DEX,
		"baseINT": &t.Stats.INT, "baseWIT": &t.Stats.WIT, "baseMEN": &t.Stats.MEN,
		"basePAtk": &t.PAtk, "baseMAtk": &t.MAtk, "basePAtkSpd": &t.PAtkSpd, "baseMAtkSpd": &t.MAtkSpd,
		"baseCritRate": &t.CritRate, "baseMCritRate": &t.MCritRate,
		"baseAtkRange": &t.AttackRange, "baseRndDam": &t.RandomDamage,
		"baseBreath": &t.Breath, "baseSafeFall": &t.SafeFall,
	}
	groups := map[string]map[string]*int{
		"basePDef": {
			"chest": &t.PDef.Chest, "legs": &t.PDef.Legs, "head": &t.PDef.Head, "feet": &t.PDef.Feet,
			"gloves": &t.PDef.Gloves, "underwear": &t.PDef.Underwear, "cloak": &t.PDef.Cloak,
		},
		"baseMDef": {
			"rear": &t.MDef.REar, "lear": &t.MDef.LEar, "rfinger": &t.MDef.RFinger,
			"lfinger": &t.MDef.LFinger, "neck": &t.MDef.Neck,
		},
	}
	floats := map[string]map[string]*float64{
		"baseMoveSpd": {
			"walk": &t.MoveSpeed.Walk, "run": &t.MoveSpeed.Run,
			"slowSwim": &t.MoveSpeed.SlowSwim, "fastSwim": &t.MoveSpeed.FastSwim,
		},
		"collisionMale":   {"radius": &t.CollisionMale.Radius, "height": &t.CollisionMale.Height},
		"collisionFemale": {"radius": &t.CollisionFemale.Radius, "height": &t.CollisionFemale.Height},
	}
	// Read by PlayerTemplateData but not used by Mobius.
	ignored := map[string][]string{
		"baseCanPenetrate": nil,
		"baseDamRange":     {"verticalDirection", "horizontalDirection", "distance", "width"},
	}
	seen := make(map[string]bool)
	for _, n := range sd.children {
		if seen[n.name] {
			return n.errorf("given twice")
		}
		seen[n.name] = true
		var err error
		switch {
		case ints[n.name] != nil:
			*ints[n.name], err = n.intText()
		case n.name == "baseAtkType":
			if err = n.leaf(); err == nil {
				var ok bool
				if t.AttackType, ok = data.ParseWeaponType(n.text); !ok {
					err = n.errorf("unknown weapon type %q", n.text)
				}
			}
		case n.name == "creationPoints":
			err = readCreationPoints(n, t)
		case groups[n.name] != nil:
			err = readGroup(n, func(c *xmlNode) error {
				p := groups[n.name][c.name]
				if p == nil {
					return c.errorf("unknown element")
				}
				v, err := c.intText()
				*p = v
				return err
			})
		case floats[n.name] != nil:
			err = readGroup(n, func(c *xmlNode) error {
				p := floats[n.name][c.name]
				if p == nil {
					return c.errorf("unknown element")
				}
				v, err := c.floatText()
				*p = v
				return err
			})
		default:
			sub, ok := ignored[n.name]
			if !ok {
				return n.errorf("unknown element")
			}
			if len(n.attrs) > 0 {
				return n.errorf("unexpected attributes")
			}
			if err = n.checkChildren(sub...); err == nil && len(sub) == 0 && len(n.children) > 0 {
				err = n.errorf("want only a value")
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// readGroup calls read for each child of n, which may have no attributes
// and no child given twice.
func readGroup(n *xmlNode, read func(c *xmlNode) error) error {
	if len(n.attrs) > 0 {
		return n.errorf("unexpected attributes")
	}
	seen := make(map[string]bool)
	for _, c := range n.children {
		if seen[c.name] {
			return c.errorf("given twice")
		}
		seen[c.name] = true
		if err := read(c); err != nil {
			return err
		}
	}
	return nil
}

func readCreationPoints(n *xmlNode, t *data.ClassTemplate) error {
	if err := n.checkChildren("node"); err != nil {
		return err
	}
	for _, c := range n.children {
		if err := c.checkAttrs("x", "y", "z"); err != nil {
			return err
		}
		var loc data.Location
		var err error
		if loc.X, err = c.intAttr("x"); err != nil {
			return err
		}
		if loc.Y, err = c.intAttr("y"); err != nil {
			return err
		}
		if loc.Z, err = c.intAttr("z"); err != nil {
			return err
		}
		t.CreationPoints = append(t.CreationPoints, loc)
	}
	return nil
}

// readLevels reads lvlUpgainData: one level element for each level 1 to
// levels, in any order (Mobius indexes them by val), each with hp, mp, cp
// and their regeneration.
func readLevels(n *xmlNode, t *data.ClassTemplate, levels int) error {
	if err := n.checkChildren("level"); err != nil {
		return err
	}
	l := &t.Levels
	columns := map[string]*[]float64{
		"hp": &l.HP, "mp": &l.MP, "cp": &l.CP,
		"hpRegen": &l.HPRegen, "mpRegen": &l.MPRegen, "cpRegen": &l.CPRegen,
	}
	byLevel := make(map[int]map[string]float64, len(n.children))
	for _, lv := range n.children {
		if err := lv.checkAttrs("val"); err != nil {
			return err
		}
		val, err := lv.intAttr("val")
		if err != nil {
			return err
		}
		if _, dup := byLevel[val]; dup {
			return lv.errorf("level %d given twice", val)
		}
		values := make(map[string]float64, len(columns))
		for _, c := range lv.children {
			if _, known := columns[c.name]; !known {
				return c.errorf("unknown element")
			}
			if _, dup := values[c.name]; dup {
				return c.errorf("given twice")
			}
			if values[c.name], err = c.floatText(); err != nil {
				return err
			}
		}
		for _, name := range slices.Sorted(maps.Keys(columns)) {
			if _, ok := values[name]; !ok {
				return lv.errorf("level %d has no %s", val, name)
			}
		}
		byLevel[val] = values
	}
	if len(byLevel) != levels {
		return n.errorf("%d levels, want 1-%d (the maxLevel of %s)", len(byLevel), levels, experienceFile)
	}
	for level := 1; level <= levels; level++ {
		values, ok := byLevel[level]
		if !ok {
			return n.errorf("level %d is missing", level)
		}
		for name, col := range columns {
			*col = append(*col, values[name])
		}
	}
	return nil
}

// byLevel orders the elements of a per-level list by their level
// attribute, which must take each value 1 to len(nodes) once, in any
// order (Mobius indexes them by level).
func byLevel(parent *xmlNode, nodes []*xmlNode, attr string) ([]*xmlNode, error) {
	ordered := make([]*xmlNode, len(nodes))
	for _, n := range nodes {
		level, err := n.intAttr(attr)
		if err != nil {
			return nil, err
		}
		if level < 1 || level > len(nodes) {
			return nil, n.errorf("level %d: levels must be 1 to %d without gaps", level, len(nodes))
		}
		if ordered[level-1] != nil {
			return nil, n.errorf("level %d given twice", level)
		}
		ordered[level-1] = n
	}
	if len(ordered) == 0 {
		return nil, parent.errorf("no levels")
	}
	return ordered, nil
}

// readEquipment reads initialEquipment.xml, sorted by class and item ID.
func readEquipment(o options) ([]data.Equipment, error) {
	root, err := readXMLTree(o.src, equipmentFile)
	if err != nil {
		return nil, err
	}
	if err := root.checkChildren("equipment"); err != nil {
		return nil, err
	}
	var equipment []data.Equipment
	for _, n := range root.children {
		if err := n.checkAttrs("classId"); err != nil {
			return nil, err
		}
		if err := n.checkChildren("item"); err != nil {
			return nil, err
		}
		classID, err := n.intAttr("classId")
		if err != nil {
			return nil, err
		}
		e := data.Equipment{ClassID: classID, Items: make([]data.EquipmentItem, 0, len(n.children))}
		for _, it := range n.children {
			if err := it.checkAttrs("id", "count", "equipped"); err != nil {
				return nil, err
			}
			id, err := it.intAttr("id")
			if err != nil {
				return nil, err
			}
			s, ok := it.attr("count")
			if !ok {
				return nil, it.errorf("class %d: item %d has no count", classID, id)
			}
			count, err := parseJavaLong(s)
			if err != nil {
				return nil, it.errorf("count: %v", err)
			}
			var equipped bool
			if s, ok := it.attr("equipped"); ok {
				if equipped, err = parseJavaBool(s); err != nil {
					return nil, it.errorf("equipped: %v", err)
				}
			}
			e.Items = append(e.Items, data.EquipmentItem{ItemID: id, Count: count, Equipped: equipped})
		}
		slices.SortStableFunc(e.Items, func(a, b data.EquipmentItem) int { return cmp.Compare(a.ItemID, b.ItemID) })
		equipment = append(equipment, e)
	}
	slices.SortStableFunc(equipment, func(a, b data.Equipment) int { return cmp.Compare(a.ClassID, b.ClassID) })
	return equipment, nil
}

// readStatBonus reads statBonus.xml: for each base attribute, one bonus per
// value 0 to 99.
func readStatBonus(o options) (data.StatBonusTable, error) {
	var t data.StatBonusTable
	root, err := readXMLTree(o.src, statBonusXML)
	if err != nil {
		return t, err
	}
	columns := map[string]*[]float64{
		"STR": &t.STR, "INT": &t.INT, "DEX": &t.DEX, "WIT": &t.WIT, "CON": &t.CON, "MEN": &t.MEN,
	}
	for _, n := range root.children {
		dst, ok := columns[n.name]
		if !ok {
			return t, n.errorf("unknown element (want STR, INT, DEX, WIT, CON or MEN)")
		}
		if *dst != nil {
			return t, n.errorf("given twice")
		}
		if err := n.checkChildren("stat"); err != nil {
			return t, err
		}
		values := make([]float64, data.StatBonusValues)
		set := make([]bool, data.StatBonusValues)
		for _, s := range n.children {
			if err := s.checkAttrs("value", "bonus"); err != nil {
				return t, err
			}
			v, err := s.intAttr("value")
			if err != nil {
				return t, err
			}
			if v < 0 || v >= data.StatBonusValues {
				return t, s.errorf("value %d is not in 0-%d", v, data.StatBonusValues-1)
			}
			if set[v] {
				return t, s.errorf("value %d given twice", v)
			}
			set[v] = true
			if values[v], err = s.floatAttr("bonus"); err != nil {
				return t, err
			}
		}
		if i := slices.Index(set, false); i >= 0 {
			return t, n.errorf("no bonus for value %d", i)
		}
		*dst = values
	}
	for _, name := range slices.Sorted(maps.Keys(columns)) {
		if *columns[name] == nil {
			return t, fmt.Errorf("%s: no %s table", statBonusXML, name)
		}
	}
	return t, nil
}

// readExperience reads experience.xml.
func readExperience(o options) (data.ExperienceTable, error) {
	var t data.ExperienceTable
	root, err := readXMLTree(o.src, experienceFile)
	if err != nil {
		return t, err
	}
	if err := root.checkAttrs("maxLevel", "maxPetLevel"); err != nil {
		return t, err
	}
	if t.MaxLevel, err = root.intAttr("maxLevel"); err != nil {
		return t, err
	}
	if t.MaxPetLevel, err = root.intAttr("maxPetLevel"); err != nil {
		return t, err
	}
	if err := root.checkChildren("experience"); err != nil {
		return t, err
	}
	rows, err := byLevel(root, root.children, "level")
	if err != nil {
		return t, err
	}
	for _, n := range rows {
		if err := n.checkAttrs("level", "tolevel"); err != nil {
			return t, err
		}
		s, ok := n.attr("tolevel")
		if !ok {
			return t, n.errorf("no tolevel attribute")
		}
		exp, err := parseJavaLong(s)
		if err != nil {
			return t, n.errorf("tolevel: %v", err)
		}
		t.Exp = append(t.Exp, exp)
	}
	if t.MaxLevel < 1 || t.MaxLevel >= len(t.Exp) {
		return t, fmt.Errorf("%s: maxLevel %d needs levels 1 to %d, have %d", experienceFile, t.MaxLevel, t.MaxLevel+1, len(t.Exp))
	}
	return t, nil
}

// readLevelTable reads a list of <elem level="n" val="v"/> for levels 1 to
// n.
func readLevelTable(o options, file, elem string) (data.LevelTable, error) {
	root, err := readXMLTree(o.src, file)
	if err != nil {
		return nil, err
	}
	if err := root.checkChildren(elem); err != nil {
		return nil, err
	}
	rows, err := byLevel(root, root.children, "level")
	if err != nil {
		return nil, err
	}
	t := make(data.LevelTable, 0, len(rows))
	for _, n := range rows {
		if err := n.checkAttrs("level", "val"); err != nil {
			return nil, err
		}
		v, err := n.floatAttr("val")
		if err != nil {
			return nil, err
		}
		t = append(t, v)
	}
	return t, nil
}

// readHitConditionBonus reads hitConditionBonus.xml. A missing condition
// is 0, as in Mobius.
func readHitConditionBonus(o options) (data.HitConditionBonus, error) {
	var h data.HitConditionBonus
	root, err := readXMLTree(o.src, hitConditionXML)
	if err != nil {
		return h, err
	}
	fields := map[string]*int{
		"front": &h.Front, "side": &h.Side, "back": &h.Back, "high": &h.High,
		"low": &h.Low, "dark": &h.Dark, "rain": &h.Rain,
	}
	seen := make(map[string]bool)
	for _, n := range root.children {
		p, ok := fields[n.name]
		if !ok {
			return h, n.errorf("unknown element (want %s)", strings.Join(slices.Sorted(maps.Keys(fields)), ", "))
		}
		if seen[n.name] {
			return h, n.errorf("given twice")
		}
		seen[n.name] = true
		if err := n.checkAttrs("val"); err != nil {
			return h, err
		}
		if *p, err = n.intAttr("val"); err != nil {
			return h, err
		}
	}
	return h, nil
}
