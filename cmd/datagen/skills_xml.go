package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// skillsDir holds the skill files. Its custom/ folder is left out: Mobius
// loads it only with CustomSkillsLoad, an extra L2Go keeps off.
const skillsDir = statsDir + "/skills"

// skillUnusedValues are skill values (and enchant values) the files set
// that the Mobius Skill constructor never reads; datagen leaves them out.
var skillUnusedValues = []string{
	"aggroPoints", "attributePower", "fanRange", "flyRadius", "flyType", "inheritElementals", "inheritPercent",
	"itemConsumeSteps", "magicLvl", "offensive",
}

// effectBlocks are the effect blocks of a skill (and, after "enchantN", of
// an enchant route), by lower case element name. Mobius matches them in
// any case.
var effectBlocks = map[string]data.EffectScope{
	"effects":           data.ScopeGeneral,
	"starteffects":      data.ScopeStart,
	"channelingeffects": data.ScopeChanneling,
	"pveeffects":        data.ScopePVE,
	"pvpeffects":        data.ScopePVP,
	"endeffects":        data.ScopeEnd,
	"selfeffects":       data.ScopeSelf,
}

// isEffectBlock reports whether a lower case element name is an effect
// block.
func isEffectBlock(lname string) bool {
	_, ok := effectBlocks[lname]
	return ok
}

// conditionElems are the condition elements, lower case.
var conditionElems = []string{"and", "or", "not", "player", "target", "using", "game"}

// readSkills reads the skill files directly in skillsDir, sorted by ID.
func readSkills(o options) ([]data.SkillTemplate, error) {
	names, err := fs.Glob(o.src, skillsDir+"/*.xml")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: no skill files found", skillsDir)
	}
	known := make(map[string]bool)
	for _, n := range data.SkillValueNames() {
		known[n] = true
	}
	var (
		skills []data.SkillTemplate
		seen   = make(map[int]string) // skill ID -> file
		errs   []error
	)
	for _, name := range names {
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		list, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		for _, e := range list.children() {
			if e.name != "skill" {
				e.unknown()
				continue
			}
			t := parseSkill(e, known, o.warnf)
			if prev, dup := seen[t.ID]; dup {
				e.errorf("skill %d is also defined in %s", t.ID, prev)
				continue
			}
			seen[t.ID] = name
			skills = append(skills, t)
		}
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	slices.SortFunc(skills, func(a, b data.SkillTemplate) int { return cmp.Compare(a.ID, b.ID) })
	return skills, nil
}

// enchantGroupSize is the number of levels of an enchant group, Mobius
// DocumentSkill.getEnchantGroupSize.
func enchantGroupSize(group int) int {
	if group < 3 {
		return 30
	}
	return 15
}

// parseSkill reads one <skill> as Mobius DocumentSkill does. Problems are
// recorded on the document; warn reports what Mobius ignores.
func parseSkill(e *mxElem, known map[string]bool, warn func(format string, args ...any)) data.SkillTemplate {
	t := data.SkillTemplate{
		ID:     e.reqInt("id"),
		Name:   e.reqStr("name"),
		Levels: e.reqInt("levels"),
		Values: make(map[string]string),
	}
	for r := range data.MaxEnchantRoutes {
		attr := "enchantGroup" + strconv.Itoa(r+1)
		if _, ok := e.attr(attr); !ok {
			continue
		}
		for len(t.Enchants) <= r {
			t.Enchants = append(t.Enchants, data.EnchantRoute{})
		}
		t.Enchants[r].Levels = enchantGroupSize(e.intAttr(attr, 0))
	}
	if t.Levels <= 0 {
		e.errorf("skill %d: levels must be positive", t.ID)
	}

	// Where each value is set, to check that the enchant routes' values
	// come after the skill's: Mobius applies them in XML order.
	basePos := make(map[string]int)
	routePos := make([]map[string]int, len(t.Enchants))
	for i, c := range e.children() {
		lname := strings.ToLower(c.name)
		switch {
		case lname == "table":
			parseSkillTable(c, &t)
		case lname == "conditions":
			t.Conditions = append(t.Conditions, parseSkillConditions(c, warn))
		case isEffectBlock(lname):
			addEffectBlock(c, &t.Effects, effectBlocks[lname], t.ID)
		case strings.HasPrefix(lname, "enchant"):
			parseEnchantElem(c, &t, lname, i, routePos, known, warn)
		case lname == "set":
			c.errorf("skill %d: a skill-level <set> is not supported (the data has none)", t.ID)
			c.skip()
		default:
			v, ok := skillValue(c, t.ID)
			if !ok {
				continue
			}
			switch {
			case known[c.name]:
				if _, dup := t.Values[c.name]; dup {
					c.errorf("skill %d: value given twice", t.ID)
				}
				t.Values[c.name] = v
				basePos[c.name] = i
			case slices.Contains(skillUnusedValues, c.name):
				// Mobius stores it but never reads it.
			default:
				c.errorf("skill %d: unknown value (not read by the Mobius Skill constructor)", t.ID)
			}
		}
	}
	for r := range routePos {
		for name, pos := range routePos[r] {
			if bp, ok := basePos[name]; ok && bp > pos {
				e.errorf("skill %d: <%s> comes after <enchant%d name=%q>, which Mobius then ignores", t.ID, name, r+1, name)
			}
		}
	}
	return t
}

// skillValue returns the text of a value element such as <mpConsume>,
// which takes no attributes or children.
func skillValue(c *mxElem, id int) (string, bool) {
	if len(c.attrs) > 0 || len(c.kids) > 0 {
		c.errorf("skill %d: a value element takes only text", id)
		c.skip()
		return "", false
	}
	if c.text == "" {
		c.errorf("skill %d: empty value", id)
		return "", false
	}
	return c.text, true
}

// parseSkillTable reads a <table name="#x">values</table>.
func parseSkillTable(c *mxElem, t *data.SkillTemplate) {
	name := c.reqStr("name")
	if !strings.HasPrefix(name, "#") {
		c.errorf("skill %d: table name %q must start with #", t.ID, name)
		return
	}
	values := strings.Fields(c.text)
	if len(values) == 0 || len(c.kids) > 0 {
		c.errorf("skill %d: table %s: want values only", t.ID, name)
		return
	}
	if t.Tables == nil {
		t.Tables = make(map[string][]string)
	}
	if _, dup := t.Tables[name]; dup {
		c.errorf("skill %d: table %s given twice", t.ID, name)
	}
	t.Tables[name] = values
}

// parseEnchantElem reads an element of an enchant route: <enchantN
// name="x">value</enchantN>, <enchantNconditions> or an effect block such
// as <enchantNeffects>.
func parseEnchantElem(c *mxElem, t *data.SkillTemplate, lname string, pos int, routePos []map[string]int, known map[string]bool, warn func(string, ...any)) {
	rest := strings.TrimPrefix(lname, "enchant")
	r, err := strconv.Atoi(rest[:min(1, len(rest))])
	if err != nil || r < 1 || r > data.MaxEnchantRoutes {
		c.errorf("skill %d: unknown enchant element", t.ID)
		c.skip()
		return
	}
	if r > len(t.Enchants) || t.Enchants[r-1].Levels == 0 {
		warn("skills: %s:%d: skill %d: <%s> without enchantGroup%d, which Mobius ignores", c.doc.path, c.line, t.ID, c.name, r)
		c.skip()
		return
	}
	route := &t.Enchants[r-1]
	block := rest[1:]
	switch {
	case block == "":
		name := strings.TrimSpace(c.reqStr("name"))
		v, ok := c.attr("val")
		if !ok {
			v = c.text
		}
		v = strings.TrimSpace(v)
		if v == "" || len(c.kids) > 0 {
			c.errorf("skill %d: enchant value %q: want a value", t.ID, name)
			return
		}
		if slices.Contains(skillUnusedValues, name) {
			return // Mobius stores it but never reads it
		}
		if !known[name] {
			c.errorf("skill %d: enchant value %q is not read by the Mobius Skill constructor", t.ID, name)
			return
		}
		if route.Values == nil {
			route.Values = make(map[string]string)
		}
		if _, dup := route.Values[name]; dup {
			c.errorf("skill %d: enchant value %q given twice", t.ID, name)
		}
		route.Values[name] = v
		if routePos[r-1] == nil {
			routePos[r-1] = make(map[string]int)
		}
		routePos[r-1][name] = pos
	case block == "conditions":
		route.Conditions = append(route.Conditions, parseSkillConditions(c, warn))
	case isEffectBlock(block):
		addEffectBlock(c, &route.Effects, effectBlocks[block], t.ID)
	default:
		c.errorf("skill %d: unknown enchant element", t.ID)
		c.skip()
	}
}

// addEffectBlock reads an effect block into blocks[scope]. A block without
// effects is kept (as an empty list): an enchant route's empty block still
// replaces the skill's.
func addEffectBlock(c *mxElem, blocks *map[data.EffectScope][]data.EffectDef, scope data.EffectScope, id int) {
	if *blocks == nil {
		*blocks = make(map[data.EffectScope][]data.EffectDef)
	}
	if _, dup := (*blocks)[scope]; dup {
		c.errorf("skill %d: <%s> given twice", id, c.name)
	}
	if c.text != "" {
		c.errorf("skill %d: unexpected text %q", id, c.text)
	}
	effects := []data.EffectDef{}
	for _, k := range c.children() {
		if k.name != "effect" {
			// Mobius drops stat functions here (they need an effect) and
			// reads a <conditions> only as the very first node.
			k.errorf("skill %d: only <effect> elements are read here", id)
			k.skip()
			continue
		}
		effects = append(effects, parseEffect(k, id))
	}
	(*blocks)[scope] = effects
}

// parseSkillConditions reads a <conditions> block: its message and its
// condition element. Mobius reads only the first one; warn reports others.
func parseSkillConditions(c *mxElem, warn func(string, ...any)) data.SkillConditions {
	sc := data.SkillConditions{}
	msg, hasMsg := c.attr("msg")
	_, hasMsgID := c.attr("msgId")
	_, hasAddName := c.attr("addName")
	switch {
	case hasMsg:
		sc.Msg = msg // Mobius ignores msgId and addName then
	case hasMsgID:
		sc.MsgID = c.intAttr("msgId", 0)
		sc.AddName = hasAddName && sc.MsgID > 0 // any addName value counts
	}
	kids := c.children()
	if len(kids) == 0 {
		c.errorf("no condition element")
		return sc
	}
	for _, k := range kids[1:] {
		warn("skills: %s:%d: <%s> after the first condition of <%s>, which Mobius ignores", c.doc.path, k.line, k.name, c.name)
		k.skip()
	}
	if c.text != "" {
		c.errorf("unexpected text %q", c.text)
	}
	sc.Cond = parseConditionDef(kids[0])
	return sc
}

// parseConditionDef reads a condition element as Mobius
// DocumentBase.parseCondition does: and, or and not over their element
// children, or a player, target, using or game element whose attributes
// are the conditions (several are joined with and, in the attribute order
// of the Java DOM: sorted by name).
func parseConditionDef(c *mxElem) data.ConditionDef {
	lname := strings.ToLower(c.name)
	if c.text != "" {
		c.errorf("unexpected text %q", c.text)
	}
	switch lname {
	case "and", "or", "not":
		d := data.ConditionDef{Kind: map[string]data.ConditionKind{"and": data.CondAnd, "or": data.CondOr, "not": data.CondNot}[lname]}
		kids := c.children()
		if len(kids) == 0 || (lname == "not" && len(kids) != 1) {
			c.errorf("%d operands", len(kids))
		}
		for _, k := range kids {
			if strings.EqualFold(k.name, "value") {
				k.errorf("a <value> in a logic condition")
				k.skip()
				continue
			}
			d.Conds = append(d.Conds, parseConditionDef(k))
		}
		return d
	case "player", "target", "using", "game":
		for _, k := range c.children() {
			k.errorf("a condition element has no children")
			k.skip()
		}
		idx := make([]int, len(c.attrs))
		for i := range idx {
			idx[i] = i
		}
		slices.SortFunc(idx, func(a, b int) int { return strings.Compare(c.attrs[a].Name.Local, c.attrs[b].Name.Local) })
		var leaves []data.ConditionDef
		for _, i := range idx {
			a := c.attrs[i]
			if a.Name.Space != "" {
				continue
			}
			c.read[i] = true
			kind, ok := data.ParseConditionKind(lname + "." + strings.ToLower(a.Name.Local))
			if !ok {
				c.errorf("unknown condition attribute %q", a.Name.Local)
				continue
			}
			leaves = append(leaves, data.ConditionDef{Kind: kind, Value: a.Value})
		}
		switch len(leaves) {
		case 0:
			c.errorf("no condition attribute")
			return data.ConditionDef{Kind: data.CondAnd}
		case 1:
			return leaves[0]
		}
		return data.ConditionDef{Kind: data.CondAnd, Conds: leaves}
	}
	c.errorf("unknown condition element")
	c.skip()
	return data.ConditionDef{Kind: data.CondAnd}
}

// statFuncOps are the stat function elements, lower case (Mobius matches
// them in any case).
var statFuncOps = map[string]bool{"add": true, "sub": true, "mul": true, "div": true, "set": true, "share": true, "enchant": true, "enchanthp": true}

// parseEffect reads one <effect> as Mobius DocumentBase.attachEffect does:
// its name and ticks, its parameters (the attributes of <param> elements,
// then the elements holding only a value), its stat functions and the
// items of RestorationRandom.
func parseEffect(e *mxElem, id int) data.EffectDef {
	name := e.reqStr("name")
	def := data.EffectDef{Name: name}
	// A #table name is checked level by level when the data is loaded.
	en, ok := data.ParseEffectName(name)
	if !ok && !strings.HasPrefix(name, "#") {
		e.errorf("skill %d: unknown effect %q", id, name)
	}
	if v, ok := e.attr("ticks"); ok {
		def.Ticks = v
	}
	e.ignore("abnormalTime") // read into the effect's set, used by nothing
	if e.text != "" {
		e.errorf("skill %d: unexpected text %q", id, e.text)
	}
	params := make(map[string]string)
	setParam := func(k *mxElem, param, value string) {
		if _, dup := params[param]; dup {
			k.errorf("skill %d: effect %s: parameter %q given twice", id, name, param)
		}
		params[param] = value
	}
	var valueElems []*mxElem
	for _, k := range e.children() {
		lname := strings.ToLower(k.name)
		switch {
		case k.name == "param":
			if len(k.kids) > 0 || k.text != "" {
				k.errorf("skill %d: <param> takes attributes only", id)
			}
			for i, a := range k.attrs {
				if a.Name.Space == "" {
					k.read[i] = true
					setParam(k, a.Name.Local, a.Value)
				}
			}
		case statFuncOps[lname]:
			def.Funcs = append(def.Funcs, parseStatFuncs(k, lname, id)...)
		case k.name == "items" && en == data.EffectRestorationRandom:
			if def.Items != nil {
				k.errorf("skill %d: <items> given twice", id)
			}
			def.Items = parseRestorationItems(k, id)
		case lname == "param" || lname == "conditions" || lname == "effect" || slices.Contains(conditionElems, lname):
			// Mobius ignores these here (an effect's own condition is
			// read but never used), or fails on a nested effect.
			k.errorf("skill %d: <%s> is not supported in an effect", id, k.name)
			k.skip()
		default:
			if len(k.attrs) > 0 || len(k.kids) > 0 {
				k.errorf("skill %d: a parameter element takes only text", id)
				k.skip()
				continue
			}
			valueElems = append(valueElems, k)
		}
	}
	for _, k := range valueElems {
		setParam(k, k.name, k.text)
	}
	if len(params) > 0 {
		def.Params = params
	}
	return def
}

// parseStatFuncs reads a stat function element: <add stat="pAtk"
// val="10"/>, with the value as text or in a <value> child and an optional
// condition child, or the form without stat and val whose children name
// the stats (<enchant><pDef>0</pDef></enchant>).
func parseStatFuncs(k *mxElem, lname string, id int) []data.StatFuncDef {
	op, _ := data.ParseStatOp(lname)
	stat, hasStat := k.attr("stat")
	val, hasVal := k.attr("val")
	if !hasStat && !hasVal {
		if k.text != "" {
			k.errorf("skill %d: unexpected text %q", id, k.text)
		}
		var funcs []data.StatFuncDef
		for _, sk := range k.children() {
			f := data.StatFuncDef{Op: op, Order: op.DefaultOrder(), Value: sk.text}
			f.Stat = parseFuncStat(sk, sk.name, id)
			f.Cond = funcCondition(sk, id)
			funcs = append(funcs, f)
		}
		if len(funcs) == 0 {
			k.errorf("skill %d: a stat function without a stat", id)
		}
		return funcs
	}
	if !hasStat {
		k.errorf("skill %d: no stat attribute (Mobius drops the function)", id)
		k.skip()
		return nil
	}
	f := data.StatFuncDef{Op: op, Order: op.DefaultOrder(), Stat: parseFuncStat(k, stat, id)}
	if v, ok := k.attr("order"); ok {
		n, err := parseJavaInt(v)
		if err != nil {
			k.errorf("skill %d: order: %v", id, err)
		}
		if n >= 0 {
			f.Order = n
		}
	}
	switch {
	case hasVal:
		f.Value = val
	default:
		for _, v := range k.kids {
			if strings.EqualFold(v.name, "value") {
				if f.Value != "" {
					v.errorf("skill %d: a second <value>", id)
				}
				f.Value = v.text
			}
		}
		if f.Value == "" {
			f.Value = k.text
		}
	}
	f.Cond = funcCondition(k, id)
	return []data.StatFuncDef{f}
}

// parseFuncStat returns the stat of a stat function.
func parseFuncStat(k *mxElem, name string, id int) data.Stat {
	s, ok := data.ParseStat(name)
	if !ok {
		k.errorf("skill %d: unknown stat %q", id, name)
	}
	return s
}

// funcCondition reads the condition of a stat function: its child element
// that is not a <value> (Mobius reads the first; datagen allows one).
func funcCondition(k *mxElem, id int) *data.ConditionDef {
	var cond *data.ConditionDef
	for _, c := range k.children() {
		if strings.EqualFold(c.name, "value") {
			if len(c.kids) > 0 || len(c.attrs) > 0 {
				c.errorf("skill %d: <value> takes only text", id)
				c.skip()
			}
			continue
		}
		if !slices.Contains(conditionElems, strings.ToLower(c.name)) {
			c.errorf("skill %d: not a condition (Mobius ignores it)", id)
			c.skip()
			continue
		}
		if cond != nil {
			c.errorf("skill %d: a second condition, which Mobius ignores", id)
			c.skip()
			continue
		}
		d := parseConditionDef(c)
		cond = &d
	}
	return cond
}

// parseRestorationItems reads the <items> of RestorationRandom: item groups
// for every level, or <value level="n"> elements holding a level's groups.
func parseRestorationItems(k *mxElem, id int) *data.RestorationGroups {
	kids := k.children()
	if len(kids) == 0 {
		k.errorf("skill %d: no items", id)
		return nil
	}
	items := &data.RestorationGroups{}
	if kids[0].name != "value" {
		items.All = parseItemGroups(kids, id)
		return items
	}
	items.ByLevel = make(map[int][]data.RestorationGroup)
	for _, v := range kids {
		if v.name != "value" {
			v.errorf("skill %d: Mobius ignores this next to <value> elements", id)
			v.skip()
			continue
		}
		level := v.intAttr("level", 1)
		if _, dup := items.ByLevel[level]; dup {
			v.errorf("skill %d: items of level %d given twice", id, level)
		}
		items.ByLevel[level] = parseItemGroups(v.children(), id)
	}
	return items
}

// parseItemGroups reads <item chance="x"> groups of <item id count
// minEnchant maxEnchant> items.
func parseItemGroups(kids []*mxElem, id int) []data.RestorationGroup {
	var groups []data.RestorationGroup
	for _, g := range kids {
		if g.name != "item" {
			g.errorf("skill %d: want an item group", id)
			g.skip()
			continue
		}
		group := data.RestorationGroup{Chance: g.floatAttr("chance", 100)}
		for _, it := range g.children() {
			if it.name != "item" {
				it.errorf("skill %d: want an item", id)
				it.skip()
				continue
			}
			gi := data.RestorationItem{
				ID:         it.reqInt("id"),
				Count:      1,
				MinEnchant: it.intAttr("minEnchant", 0),
				MaxEnchant: it.intAttr("maxEnchant", 0),
			}
			if v, ok := it.attr("count"); ok {
				n, err := parseJavaLong(v)
				if err != nil {
					it.errorf("skill %d: count: %v", id, err)
				}
				gi.Count = n
			}
			if gi.ID <= 0 {
				it.errorf("skill %d: item ID must be positive (Mobius skips it)", id)
			}
			group.Items = append(group.Items, gi)
		}
		if len(group.Items) == 0 {
			g.errorf("skill %d: an item group without items", id)
		}
		groups = append(groups, group)
	}
	return groups
}
