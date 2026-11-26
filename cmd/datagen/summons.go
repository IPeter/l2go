package main

import (
	"bytes"
	"cmp"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The summon sources: the pet folder PetDataTable reads, PetSkillData.xml
// and the cubic folder CubicData reads (with its subfolders).
const (
	petXMLDir      = "dist/game/data/stats/pets"
	petSkillXML    = "dist/game/data/PetSkillData.xml"
	cubicXMLDir    = "dist/game/data/stats/cubics"
	petDefaultLoad = 20000 // PetData._load
	petDefaultHL   = 1     // PetData._hungryLimit
)

// The summons domain converts the pets, the pet and servitor skills and the
// cubics, loaded by data.LoadSummons.
func init() { registerDomain("summons", genSummons) }

// genSummons reads the summon data, checks it with data.NewSummons and
// adds pets/pets.json (each pet on a line, followed by its levels one per
// line), pets/skills.json and cubics/cubics.json (one record per line).
func genSummons(o options) error {
	var (
		d   data.SummonData
		err error
	)
	if d.Pets, err = readPets(o); err != nil {
		return fmt.Errorf("pets: %w", err)
	}
	if d.PetSkills, err = readPetSkills(o); err != nil {
		return fmt.Errorf("pet skills: %w", err)
	}
	if d.Cubics, err = readCubics(o); err != nil {
		return fmt.Errorf("cubics: %w", err)
	}
	if _, err := data.NewSummons(d); err != nil {
		return err
	}
	if err := addPets(o.out, d.Pets); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.PetSkillsFile, d.PetSkills); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.CubicsFile, d.Cubics); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "summons: %d pets, %d pet skill rows, %d cubic levels\n", len(d.Pets), len(d.PetSkills), len(d.Cubics))
	return nil
}

// summonXMLFiles returns the .xml files of a folder, sorted; with
// recursive, those of its subfolders too.
func summonXMLFiles(fsys fs.FS, dir string, recursive bool) ([]string, error) {
	var files []string
	err := fs.WalkDir(fsys, dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			if p != dir && !recursive {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(p), ".xml") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s: no XML files", dir)
	}
	slices.Sort(files)
	return files, nil
}

// readPets reads the pet files as PetDataTable does, sorted by NPC ID. A pet
// listed twice fails (Mobius keeps the last).
func readPets(o options) ([]data.PetData, error) {
	files, err := summonXMLFiles(o.src, petXMLDir, false)
	if err != nil {
		return nil, err
	}
	var pets []data.PetData
	for _, name := range files {
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		if doc.root.name != "pets" {
			return nil, fmt.Errorf("%s: root element is <%s>, want <pets>", name, doc.root.name)
		}
		for _, e := range doc.root.children() {
			if e.name != "pet" {
				e.unknown()
				continue
			}
			pets = append(pets, readPet(e))
		}
		if err := doc.check(); err != nil {
			return nil, err
		}
	}
	slices.SortFunc(pets, func(a, b data.PetData) int { return cmp.Compare(a.NpcID, b.NpcID) })
	for i := 1; i < len(pets); i++ {
		if pets[i].NpcID == pets[i-1].NpcID {
			return nil, fmt.Errorf("%s: pet %d is listed twice", petXMLDir, pets[i].NpcID)
		}
	}
	return pets, nil
}

// readPet reads one <pet>.
func readPet(e *mxElem) data.PetData {
	p := data.PetData{
		NpcID:       e.reqInt("id"),
		ItemID:      e.reqInt("itemId"),
		Index:       e.intAttr("index", 0),
		Food:        []int{},
		HungryLimit: petDefaultHL,
		Load:        petDefaultLoad,
	}
	for _, c := range e.children() {
		switch c.name {
		case "set":
			switch name := c.reqStr("name"); name {
			case "food":
				for _, s := range strings.Split(c.reqStr("val"), ";") {
					f, err := parseJavaInt(s) // Integer.parseInt
					if err != nil {
						c.errorf("food: %v", err)
					}
					if !slices.Contains(p.Food, f) {
						p.Food = append(p.Food, f)
					}
				}
			case "load":
				p.Load = c.reqInt("val")
			case "hungry_limit":
				p.HungryLimit = c.reqInt("val")
			case "sync_level":
				p.SyncLevel = c.reqInt("val") == 1
			case "evolve":
				c.skip() // Mobius ignores it
			default:
				c.errorf("unknown pet set %s", name)
			}
		case "skills":
			for _, s := range c.children() {
				if s.name != "skill" {
					s.unknown()
					continue
				}
				p.Skills = append(p.Skills, data.PetDataSkill{
					SkillID:    s.reqInt("skillId"),
					SkillLevel: s.reqInt("skillLevel"),
					MinLevel:   s.reqInt("minLevel"),
				})
			}
		case "stats":
			for _, s := range c.children() {
				if s.name != "stat" {
					s.unknown()
					continue
				}
				p.Levels = append(p.Levels, readPetLevel(s))
			}
		default:
			c.unknown()
		}
	}
	slices.SortFunc(p.Levels, func(a, b data.PetLevel) int { return cmp.Compare(a.Level, b.Level) })
	for i := 1; i < len(p.Levels); i++ {
		if p.Levels[i].Level == p.Levels[i-1].Level {
			e.errorf("pet %d: level %d is listed twice", p.NpcID, p.Levels[i].Level)
		}
	}
	return p
}

// readPetLevel reads one <stat level="n"> as PetLevelData reads its StatSet.
func readPetLevel(s *mxElem) data.PetLevel {
	l := data.PetLevel{}
	if lvl, err := parseJavaInt(s.reqStr("level")); err != nil { // Integer.parseInt
		s.errorf("level: %v", err)
	} else {
		l.Level = lvl
	}
	vals := make(map[string]string)
	for _, c := range s.children() {
		if c.name != "set" {
			c.unknown()
			continue
		}
		name := c.reqStr("name")
		if name == "speed_on_ride" {
			c.ignore("val") // Mobius reads only the speeds
			l.WalkOnRide = c.reqFloat("walk")
			l.RunOnRide = c.reqFloat("run")
			l.SlowSwimOnRide = c.reqFloat("slowSwim")
			l.FastSwimOnRide = c.reqFloat("fastSwim")
			l.SlowFlyOnRide = c.floatAttr("slowFly", 0)
			l.FastFlyOnRide = c.floatAttr("fastFly", 0)
			continue
		}
		if _, dup := vals[name]; dup {
			c.errorf("%s is set twice", name)
		}
		vals[name] = c.reqStr("val")
	}
	get := func(name string) (string, bool) {
		v, ok := vals[name]
		if !ok {
			s.errorf("level %d: no %s", l.Level, name)
		}
		delete(vals, name)
		return strings.TrimSpace(v), ok
	}
	getInt := func(name string, bits int) int { // Integer.parseInt, Short.parseShort
		v, ok := get(name)
		if !ok {
			return 0
		}
		n, err := strconv.ParseInt(v, 10, bits)
		if err != nil {
			s.errorf("level %d: %s: %q is not an integer", l.Level, name, v)
		}
		return int(n)
	}
	getFloat := func(name string) float64 { // Float.parseFloat
		v, ok := get(name)
		if !ok {
			return 0
		}
		f, err := strconv.ParseFloat(v, 32)
		if err != nil {
			s.errorf("level %d: %s: %q is not a number", l.Level, name, v)
		}
		return float64(float32(f))
	}
	l.MaxMeal = getInt("max_meal", 32)
	if v, ok := get("exp"); ok { // (long) Double.parseDouble
		f, err := mxFloat(v)
		if err != nil {
			s.errorf("level %d: exp: %v", l.Level, err)
		}
		l.Exp = int64(f)
	}
	l.OwnerExpTaken = getInt("get_exp_type", 32)
	l.MealInBattle = getInt("consume_meal_in_battle", 32)
	l.MealInNormal = getInt("consume_meal_in_normal", 32)
	l.PAtk = getFloat("org_pattack")
	l.PDef = getFloat("org_pdefend")
	l.MAtk = getFloat("org_mattack")
	l.MDef = getFloat("org_mdefend")
	l.HP = getFloat("org_hp")
	l.MP = getFloat("org_mp")
	l.HPRegen = getFloat("org_hp_regen")
	l.MPRegen = getFloat("org_mp_regen")
	l.SoulshotCount = getInt("soulshot_count", 16)
	l.SpiritshotCount = getInt("spiritshot_count", 16)
	for _, name := range slices.Sorted(maps.Keys(vals)) {
		s.errorf("level %d: unknown set %s", l.Level, name)
	}
	return l
}

// addPets stores the pets as a JSON array with each pet's header on a line
// and each of its levels on a line of their own.
func addPets(out *output, pets []data.PetData) error {
	var buf bytes.Buffer
	buf.WriteString("[")
	for i, p := range pets {
		levels := p.Levels
		p.Levels = nil
		head, err := marshalCompact(p)
		if err != nil {
			return fmt.Errorf("encode %s: pet %d: %w", data.PetsFile, p.NpcID, err)
		}
		const tail = `"levels":null}`
		if !bytes.HasSuffix(head, []byte(tail)) {
			return fmt.Errorf("encode %s: pet %d: levels are not the last field", data.PetsFile, p.NpcID)
		}
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
		buf.Write(head[:len(head)-len("null}")])
		buf.WriteString("[")
		for j, l := range levels {
			b, err := marshalCompact(l)
			if err != nil {
				return fmt.Errorf("encode %s: pet %d level %d: %w", data.PetsFile, p.NpcID, l.Level, err)
			}
			if j > 0 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
			buf.Write(b)
		}
		if len(levels) > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("]}")
	}
	if len(pets) > 0 {
		buf.WriteString("\n")
	}
	buf.WriteString("]\n")
	return out.addSized(data.PetsFile, buf.Bytes())
}

// readPetSkills reads the rows of PetSkillData.xml in file order. Mobius
// drops the rows whose skill it does not know and keeps one row per NPC,
// skill and level; the game server applies those rules.
func readPetSkills(o options) ([]data.PetSkill, error) {
	doc, err := parseMX(o.src, petSkillXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	rows := []data.PetSkill{}
	for _, e := range root.children() {
		if e.name != "skill" {
			e.unknown()
			continue
		}
		rows = append(rows, data.PetSkill{
			NpcID:      e.reqInt("npcId"),
			SkillID:    e.reqInt("skillId"),
			SkillLevel: e.reqInt("skillLevel"),
		})
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	return rows, nil
}

// readCubics reads the cubic files as CubicData does, sorted by ID and
// level. A cubic level listed twice fails (Mobius keeps the last).
func readCubics(o options) ([]data.CubicTemplate, error) {
	files, err := summonXMLFiles(o.src, cubicXMLDir, true)
	if err != nil {
		return nil, err
	}
	var cubics []data.CubicTemplate
	for _, name := range files {
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		root, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		for _, e := range root.children() {
			if e.name != "cubic" {
				e.unknown()
				continue
			}
			cubics = append(cubics, readCubic(e))
		}
		if err := doc.check(); err != nil {
			return nil, err
		}
	}
	cmpCubic := func(a, b data.CubicTemplate) int {
		return cmp.Or(cmp.Compare(a.ID, b.ID), cmp.Compare(a.Level, b.Level))
	}
	slices.SortFunc(cubics, cmpCubic)
	for i := 1; i < len(cubics); i++ {
		if cmpCubic(cubics[i-1], cubics[i]) == 0 {
			return nil, fmt.Errorf("%s: cubic %d level %d is listed twice", cubicXMLDir, cubics[i].ID, cubics[i].Level)
		}
	}
	return cubics, nil
}

// cubicInt reads a required attribute as StatSet.getInt does
// (Integer.parseInt).
func cubicInt(e *mxElem, name string) int {
	v, ok := e.attr(name)
	if !ok {
		e.errorf("missing attribute %s", name)
		return 0
	}
	n, err := parseJavaInt(v)
	if err != nil {
		e.errorf("attribute %s: %v", name, err)
	}
	return n
}

// cubicEnum reads an attribute as StatSet.getEnum does: one of the values,
// exactly, or def when it is missing.
func cubicEnum(e *mxElem, name, def string, values ...string) string {
	v, ok := e.attr(name)
	if !ok {
		return def
	}
	if !slices.Contains(values, v) {
		e.errorf("attribute %s: %q is not one of %s", name, v, strings.Join(values, ", "))
	}
	return v
}

var (
	cubicTargets    = []string{data.CubicTargetTarget, data.CubicTargetBySkill, data.CubicTargetHeal, data.CubicTargetMaster}
	cubicSelections = []string{data.CubicSelectRandom, data.CubicSelectWeighted, data.CubicSelectOrdered}
)

// readCubic reads one <cubic> as CubicTemplate and CubicData.parseTemplate
// do.
func readCubic(e *mxElem) data.CubicTemplate {
	c := data.CubicTemplate{
		ID:            cubicInt(e, "id"),
		Level:         cubicInt(e, "level"),
		Slot:          cubicInt(e, "slot"),
		Duration:      cubicInt(e, "duration"),
		Delay:         cubicInt(e, "delay"),
		MaxCount:      cubicInt(e, "maxCount"),
		UseUp:         cubicInt(e, "useUp"),
		Power:         e.floatAttr("power", 0),
		TargetType:    cubicEnum(e, "targetType", data.CubicTargetTarget, cubicTargets...),
		SelectionMode: cubicEnum(e, "selectionMode", data.CubicSelectRandom, cubicSelections...),
		Skills:        []data.CubicSkill{},
	}
	for _, k := range e.children() {
		switch k.name {
		case "conditions":
			c.Conditions = append(c.Conditions, readCubicConditions(k)...)
		case "skills":
			for _, s := range k.children() {
				if s.name != "skill" {
					s.unknown()
					continue
				}
				c.Skills = append(c.Skills, readCubicSkill(s))
			}
		default:
			k.unknown()
		}
	}
	return c
}

// readCubicSkill reads one cubic <skill> as CubicSkill does.
func readCubicSkill(s *mxElem) data.CubicSkill {
	sk := data.CubicSkill{
		ID:          cubicInt(s, "id"),
		Level:       cubicInt(s, "level"),
		TriggerRate: 100,
		TargetType:  cubicEnum(s, "target", data.CubicTargetTarget, cubicTargets...),
	}
	if _, ok := s.attr("triggerRate"); ok {
		sk.TriggerRate = cubicInt(s, "triggerRate")
	}
	sk.TargetDebuff = s.boolAttr("targetDebuff", false)
	sk.Priority = s.boolAttr("priority", false)
	for _, k := range s.children() {
		if k.name != "conditions" {
			k.unknown()
			continue
		}
		sk.Conditions = append(sk.Conditions, readCubicConditions(k)...)
	}
	return sk
}

// readCubicConditions reads a <conditions> element as
// CubicData.parseConditions does. Mobius warns about an unknown condition
// and skips it; datagen fails.
func readCubicConditions(e *mxElem) []data.CubicCondition {
	var conds []data.CubicCondition
	for _, k := range e.children() {
		switch k.name {
		case data.CubicCondHP:
			conds = append(conds, data.CubicCondition{
				Kind:    data.CubicCondHP,
				HPType:  cubicEnum(k, "type", "", data.CubicHPGreater, data.CubicHPLesser),
				Percent: k.reqInt("percent"),
			})
			if _, ok := k.attr("type"); !ok {
				k.errorf("missing attribute type")
			}
		case data.CubicCondRange:
			conds = append(conds, data.CubicCondition{Kind: data.CubicCondRange, Range: k.reqInt("value")})
		case data.CubicCondHealthPercent:
			conds = append(conds, data.CubicCondition{Kind: data.CubicCondHealthPercent, Min: k.reqInt("min"), Max: k.reqInt("max")})
		default:
			k.unknown()
		}
	}
	return conds
}
