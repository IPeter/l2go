package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Mobius sources of the skill enums, relative to the project folder.
const (
	abnormalTypeFile   = javaRoot + "mechanics/skill/AbnormalType.java"
	abnormalVisualFile = javaRoot + "mechanics/skill/AbnormalVisualEffect.java"
	effectScopeFile    = javaRoot + "mechanics/skill/EffectScope.java"
	operateTypeFile    = javaRoot + "mechanics/skill/SkillOperateType.java"
	targetTypeFile     = javaRoot + "mechanics/skill/targets/TargetType.java"
	elementFile        = javaRoot + "mechanics/skill/enums/Element.java"
	traitTypeFile      = javaRoot + "mechanics/stats/TraitType.java"
	socialClassFile    = javaRoot + "entity/actor/enums/player/SocialClass.java"
	instanceTypeFile   = javaRoot + "entity/actor/enums/creature/InstanceType.java"
	categoryTypeFile   = javaRoot + "data/enums/CategoryType.java"
	documentBaseFile   = javaRoot + "util/DocumentBase.java"
	effectHandlersFile = "dist/game/data/scripts/handlers/EffectMasterHandler.java"
)

// skillEnumFile is the Go file the "skills" enum domain writes.
const skillEnumFile = "skill_enum_gen.go"

func init() { registerEnums("skills", genSkillEnums) }

// skillEnum is a generated enum of the skill data. Unlike enumSpec it may
// have more than 256 values (AbnormalType) and Go names that keep the Java
// spelling (InstanceType.WorldObject is InstanceWorldObject).
type skillEnum struct {
	typ    string // Go type: "TargetType"
	kind   string // for docs and errors: "target type"
	source string // the Mobius source, for the doc comment
	// value says what the values are; empty for the Mobius ordinal.
	value  string
	doc    string // more doc comment text, if any
	java   []string
	consts []string // Go constant names
	names  []string // names in the data files
}

// How a skillEnum turns Java constant names into Go names.
const (
	// goNameFull adds the prefix to the words in camel case, always (unlike
	// goConst): TARGET_PARTY is TargetTargetParty, next to PARTY as
	// TargetParty.
	goNameFull = iota
	// goNameVerbatim adds the prefix to the Java name as it is, for
	// CamelCase names: WorldObject is InstanceWorldObject.
	goNameVerbatim
)

// newSkillEnum builds a skillEnum whose data names are the Java names.
func newSkillEnum(typ, prefix, kind, source string, java []string, naming int) skillEnum {
	e := skillEnum{typ: typ, kind: kind, source: source, java: java, names: java}
	for _, j := range java {
		if naming == goNameVerbatim {
			e.consts = append(e.consts, prefix+j)
		} else {
			e.consts = append(e.consts, prefix+goConst("", j))
		}
	}
	return e
}

// genSkillEnums writes the enums of the skill data: abnormal types, target
// types, operate types, effect scopes, effect names (the effect handlers),
// condition kinds (the conditions DocumentBase reads), traits, abnormal
// visual effects, social classes, instance types, category types and the
// seed elements.
func genSkillEnums(o options) error {
	var b bytes.Buffer
	b.WriteString(goHeader())

	plain := []struct {
		file, java, typ, prefix, kind, source, doc string
		naming                                     int
	}{
		{abnormalTypeFile, "AbnormalType", "AbnormalType", "Abnormal", "abnormal type", "mechanics/skill/AbnormalType",
			"An effect replaces an active effect of the same type unless that one has a higher abnormal level.", goNameFull},
		{targetTypeFile, "TargetType", "TargetType", "Target", "target type", "mechanics/skill/targets/TargetType", "", goNameFull},
		{operateTypeFile, "SkillOperateType", "SkillOperateType", "Operate", "skill operate type", "mechanics/skill/SkillOperateType", "", goNameVerbatim},
		{effectScopeFile, "EffectScope", "EffectScope", "Scope", "effect scope", "mechanics/skill/EffectScope", "", goNameFull},
		{socialClassFile, "SocialClass", "SocialClass", "Social", "social class", "actor/enums/player/SocialClass",
			"It is the clan rank a pledge skill needs.", goNameFull},
		{categoryTypeFile, "CategoryType", "CategoryType", "Category", "category type", "data/enums/CategoryType", "", goNameFull},
		{elementFile, "Element", "SeedElement", "Seed", "seed element", "mechanics/skill/enums/Element",
			"The elementSeed condition names it.", goNameFull},
	}
	for _, p := range plain {
		consts, err := simpleEnum(o, p.file, p.java, 0)
		if err != nil {
			return err
		}
		e := newSkillEnum(p.typ, p.prefix, p.kind, p.source, constNames(consts), p.naming)
		e.doc = p.doc
		if err := writeSkillEnum(&b, e); err != nil {
			return err
		}
	}
	if err := writeTraitTypes(o, &b); err != nil {
		return err
	}
	if err := writeVisualEffects(o, &b); err != nil {
		return err
	}
	if err := writeInstanceTypes(o, &b); err != nil {
		return err
	}
	names, err := readEffectHandlers(o.src)
	if err != nil {
		return err
	}
	e := newSkillEnum("EffectName", "Effect", "effect", "handlers/EffectMasterHandler", names, goNameVerbatim)
	e.value = "the position in its EFFECTS array"
	e.doc = "Its name in the data files is the handler class name (\"MagicalDamage\")."
	if err := writeSkillEnum(&b, e); err != nil {
		return err
	}
	kinds, err := readConditionKinds(o.src)
	if err != nil {
		return err
	}
	if err := writeSkillEnum(&b, conditionKindEnum(kinds)); err != nil {
		return err
	}
	return addGo(o, skillEnumFile, b.Bytes())
}

func constNames(consts []javaConstant) []string {
	names := make([]string, len(consts))
	for i, c := range consts {
		names[i] = c.name
	}
	return names
}

// writeTraitTypes writes TraitType from TraitType.java, with the kind of
// each trait (its constructor argument).
func writeTraitTypes(o options, b *bytes.Buffer) error {
	consts, err := simpleEnum(o, traitTypeFile, "TraitType", 1)
	if err != nil {
		return err
	}
	e := newSkillEnum("TraitType", "Trait", "trait", "mechanics/stats/TraitType", constNames(consts), goNameFull)
	if err := writeSkillEnum(b, e); err != nil {
		return err
	}
	b.WriteString("var traitTypeKinds = [...]uint8{")
	for i, c := range consts {
		v, err := javaInt(c.args[0])
		if err != nil {
			return fmt.Errorf("%s: TraitType.%s: %w", traitTypeFile, c.name, err)
		}
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d", v)
	}
	b.WriteString(`}

// Kind returns the Mobius trait type of t (TraitType.getType): 1 for the
// weapon traits, 2 for the race weaknesses, 3 for the effect traits, 0 for
// the others.
func (t TraitType) Kind() int {
	if !t.Valid() {
		return 0
	}
	return int(traitTypeKinds[t])
}

`)
	return nil
}

// writeVisualEffects writes AbnormalVisualEffect from
// AbnormalVisualEffect.java with each effect's client mask and type.
func writeVisualEffects(o options, b *bytes.Buffer) error {
	consts, err := simpleEnum(o, abnormalVisualFile, "AbnormalVisualEffect", 2)
	if err != nil {
		return err
	}
	e := newSkillEnum("AbnormalVisualEffect", "Visual", "abnormal visual effect", "mechanics/skill/AbnormalVisualEffect", constNames(consts), goNameFull)
	e.doc = "The client shows it on the creature; Mask is its bit in the packets."
	if err := writeSkillEnum(b, e); err != nil {
		return err
	}
	b.WriteString("var abnormalVisualEffectInfos = [...]struct {\n\tmask uint32\n\ttyp  int\n}{\n")
	for _, c := range consts {
		// Java int literals may set the sign bit (0x80000000).
		mask, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(c.args[0]), "0x"), 16, 32)
		if err != nil || !strings.HasPrefix(strings.TrimSpace(c.args[0]), "0x") {
			return fmt.Errorf("%s: AbnormalVisualEffect.%s: mask %q is not a hex int literal", abnormalVisualFile, c.name, c.args[0])
		}
		typ, err := javaInt(c.args[1])
		if err != nil {
			return fmt.Errorf("%s: AbnormalVisualEffect.%s: %w", abnormalVisualFile, c.name, err)
		}
		fmt.Fprintf(b, "\t{%#x, %d}, // %s\n", mask, typ, c.name)
	}
	b.WriteString(`}

// Mask returns the client bit mask of a (0 for none).
func (a AbnormalVisualEffect) Mask() uint32 {
	if !a.Valid() {
		return 0
	}
	return abnormalVisualEffectInfos[a].mask
}

// IsSpecial reports whether a goes into the special effect mask.
func (a AbnormalVisualEffect) IsSpecial() bool {
	return a.Valid() && abnormalVisualEffectInfos[a].typ == 1
}

// IsEvent reports whether a goes into the event effect mask.
func (a AbnormalVisualEffect) IsEvent() bool {
	return a.Valid() && abnormalVisualEffectInfos[a].typ == 2
}

`)
	return nil
}

// writeInstanceTypes writes InstanceType from InstanceType.java with the
// parent of each type, declared before it.
func writeInstanceTypes(o options, b *bytes.Buffer) error {
	consts, err := simpleEnum(o, instanceTypeFile, "InstanceType", 1)
	if err != nil {
		return err
	}
	names := constNames(consts)
	parents := make([]int, len(consts))
	for i, c := range consts {
		parents[i] = -1
		if p := strings.TrimSpace(c.args[0]); p != "null" {
			j := slices.Index(names[:i], p)
			if j < 0 {
				return fmt.Errorf("%s: InstanceType.%s: parent %q is not an InstanceType declared before it", instanceTypeFile, c.name, p)
			}
			parents[i] = j
		}
	}
	e := newSkillEnum("InstanceType", "Instance", "instance type", "actor/enums/creature/InstanceType", names, goNameVerbatim)
	e.doc = "The npcType condition names it; Is follows the parents like a Java instanceof."
	if err := writeSkillEnum(b, e); err != nil {
		return err
	}
	b.WriteString("var instanceTypeParents = [...]int8{")
	for i, p := range parents {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%d", p)
	}
	b.WriteString(`}

// Parent returns the type i extends; false for WorldObject.
func (i InstanceType) Parent() (InstanceType, bool) {
	if !i.Valid() || instanceTypeParents[i] < 0 {
		return 0, false
	}
	return InstanceType(instanceTypeParents[i]), true
}

// Is reports whether i is other or extends it (Mobius InstanceType.isType).
func (i InstanceType) Is(other InstanceType) bool {
	for i.Valid() {
		if i == other {
			return true
		}
		p, ok := i.Parent()
		if !ok {
			return false
		}
		i = p
	}
	return false
}

`)
	return nil
}

var effectClassRe = regexp.MustCompile(`^([A-Z][A-Za-z0-9]*)\.class$`)

// readEffectHandlers returns the effect names, the simple class names in
// the EFFECTS array of EffectMasterHandler.java (null entries are disabled
// handlers), in array order.
func readEffectHandlers(fsys fs.FS) ([]string, error) {
	src, err := fs.ReadFile(fsys, effectHandlersFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", effectHandlersFile, err)
	}
	code := stripJavaComments(string(src))
	loc := regexp.MustCompile(`\bEFFECTS\s*=\s*\{`).FindStringIndex(code)
	if loc == nil {
		return nil, fmt.Errorf("%s: the EFFECTS array was not found", effectHandlersFile)
	}
	end, err := closing(code, loc[1]-1)
	if err != nil {
		return nil, fmt.Errorf("%s: EFFECTS: %w", effectHandlersFile, err)
	}
	var names []string
	for _, item := range strings.Split(code[loc[1]:end], ",") {
		item = strings.TrimSpace(item)
		if item == "" || item == "null" {
			continue
		}
		m := effectClassRe.FindStringSubmatch(item)
		if m == nil {
			return nil, fmt.Errorf("%s: EFFECTS: cannot read %q (want Name.class)", effectHandlersFile, item)
		}
		if slices.Contains(names, m[1]) {
			return nil, fmt.Errorf("%s: EFFECTS: %s is listed twice", effectHandlersFile, m[1])
		}
		names = append(names, m[1])
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: EFFECTS is empty", effectHandlersFile)
	}
	return names, nil
}

// conditionElements are the condition elements DocumentBase.parseCondition
// handles, in its order: the logic ones, then those whose attributes are
// the conditions.
var conditionElements = []string{"and", "or", "not", "player", "target", "using", "game"}

// conditionMethods are the DocumentBase methods that read the attributes
// of a condition element.
var conditionMethods = map[string]string{
	"player": "parsePlayerCondition",
	"target": "parseTargetCondition",
	"using":  "parseUsingCondition",
	"game":   "parseGameCondition",
}

var (
	caseLabelRe = regexp.MustCompile(`case\s+"([a-z_]+)"\s*:`)
	gameAttrRe  = regexp.MustCompile(`"([a-z_]+)"\.equalsIgnoreCase\(\s*a\.getNodeName\(\)\s*\)`)
)

// readConditionKinds returns the condition kinds of DocumentBase.java:
// and, or and not, then "element.attribute" for each attribute its parse
// methods handle ("player.level"), in source order. The attribute names are
// the lower case labels DocumentBase matches the XML names against.
func readConditionKinds(fsys fs.FS) ([]string, error) {
	src, err := fs.ReadFile(fsys, documentBaseFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", documentBaseFile, err)
	}
	code := stripJavaComments(string(src))
	body, err := javaMethodBody(code, "parseCondition")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", documentBaseFile, err)
	}
	var elems []string
	for _, m := range caseLabelRe.FindAllStringSubmatch(body, -1) {
		elems = append(elems, m[1])
	}
	if !slices.Equal(elems, conditionElements) {
		return nil, fmt.Errorf("%s: parseCondition handles %v; datagen knows %v", documentBaseFile, elems, conditionElements)
	}
	kinds := []string{"and", "or", "not"}
	for _, elem := range conditionElements[3:] {
		body, err := javaMethodBody(code, conditionMethods[elem])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", documentBaseFile, err)
		}
		re := caseLabelRe
		if elem == "game" {
			re = gameAttrRe
		}
		matches := re.FindAllStringSubmatch(body, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("%s: %s handles no attributes", documentBaseFile, conditionMethods[elem])
		}
		for _, m := range matches {
			k := elem + "." + m[1]
			if slices.Contains(kinds, k) {
				return nil, fmt.Errorf("%s: %s handles %q twice", documentBaseFile, conditionMethods[elem], m[1])
			}
			kinds = append(kinds, k)
		}
	}
	return kinds, nil
}

// javaMethodBody returns the body of the method name (without braces).
func javaMethodBody(code, name string) (string, error) {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*\([^)]*\)\s*\{`)
	loc := re.FindStringIndex(code)
	if loc == nil {
		return "", fmt.Errorf("method %s not found", name)
	}
	end, err := closing(code, loc[1]-1)
	if err != nil {
		return "", fmt.Errorf("method %s: %w", name, err)
	}
	return code[loc[1]:end], nil
}

// conditionKindEnum builds ConditionKind from the kinds: "player.level" is
// CondPlayerLevel.
func conditionKindEnum(kinds []string) skillEnum {
	e := skillEnum{
		typ: "ConditionKind", kind: "condition", source: "util/DocumentBase (the condition elements and attributes it reads)",
		value: "its position in DocumentBase",
		doc:   "Its name in the data files is \"and\", \"or\", \"not\" or the element and the lower case attribute (\"player.level\").",
		java:  kinds,
		names: kinds,
	}
	for _, k := range kinds {
		var java strings.Builder
		for _, part := range strings.Split(k, ".") {
			if java.Len() > 0 {
				java.WriteString("_")
			}
			java.WriteString(strings.ToUpper(part))
		}
		e.consts = append(e.consts, "Cond"+goConst("", java.String()))
	}
	return e
}

// writeSkillEnum writes the type, constants, name table and methods of e,
// as writeEnum does, in a uint8 or, for more than 256 values, a uint16.
func writeSkillEnum(b *bytes.Buffer, e skillEnum) error {
	n := len(e.names)
	if n == 0 || len(e.java) != n || len(e.consts) != n {
		return fmt.Errorf("enum %s: %d names, %d Java names, %d Go names", e.typ, n, len(e.java), len(e.consts))
	}
	if n > 1<<16 {
		return fmt.Errorf("enum %s has %d values, more than a uint16 holds", e.typ, n)
	}
	seen := make(map[string]bool, n)
	goNames := make(map[string]bool, n)
	for i, name := range e.names {
		if seen[name] {
			return fmt.Errorf("enum %s: name %q is used twice", e.typ, name)
		}
		seen[name] = true
		if goNames[e.consts[i]] {
			return fmt.Errorf("enum %s: two constants map to the Go name %s", e.typ, e.consts[i])
		}
		goNames[e.consts[i]] = true
	}
	underlying := "uint8"
	if n > 256 {
		underlying = "uint16"
	}
	table := lowerFirst(e.typ) + "Table"
	value := e.value
	if value == "" {
		value = "the Mobius ordinal"
	}
	article := "a"
	if strings.ContainsRune("aeiou", rune(e.kind[0])) {
		article = "an"
	}
	doc := fmt.Sprintf("%s is %s %s, Mobius %s. The value is %s.", e.typ, article, e.kind, e.source, value)
	if e.doc != "" {
		doc += " " + e.doc
	}
	b.WriteString(wrapComment(doc))
	fmt.Fprintf(b, "type %s %s\n\n// The %s values.\nconst (\n", e.typ, underlying, e.typ)
	for i, c := range e.consts {
		comment := ""
		if e.names[i] != e.java[i] {
			comment = " // " + e.names[i]
		}
		fmt.Fprintf(b, "\t%s %s = %d%s\n", c, e.typ, i, comment)
	}
	fmt.Fprintf(b, ")\n\n// Num%s is the number of %s values.\nconst Num%s = %d\n\n", plural(e.typ), e.typ, plural(e.typ), n)
	fmt.Fprintf(b, "var %s = newEnumTable(%q,\n", table, e.kind)
	for _, name := range e.names {
		fmt.Fprintf(b, "\t%s,\n", strconv.Quote(name))
	}
	b.WriteString(")\n\n")
	r := strings.ToLower(e.typ[:1])
	fmt.Fprintf(b, `// String returns the name of %[1]s as the data files write it.
func (%[1]s %[2]s) String() string { return %[3]s.name(int(%[1]s)) }

// Valid reports whether %[1]s is a known %[2]s.
func (%[1]s %[2]s) Valid() bool { return %[3]s.valid(int(%[1]s)) }

// MarshalText implements encoding.TextMarshaler.
func (%[1]s %[2]s) MarshalText() ([]byte, error) { return %[3]s.marshal(int(%[1]s)) }

// UnmarshalText implements encoding.TextUnmarshaler. Unknown names are
// errors.
func (%[1]s *%[2]s) UnmarshalText(text []byte) error {
	v, err := %[3]s.unmarshal(text)
	if err != nil {
		return err
	}
	*%[1]s = %[2]s(v)
	return nil
}

// Parse%[2]s returns the %[2]s with the given name.
func Parse%[2]s(name string) (%[2]s, bool) {
	v, ok := %[3]s.parse(name)
	return %[2]s(v), ok
}

`, r, e.typ, table)
	return nil
}

// wrapComment returns text as // comment lines of at most 77 characters.
func wrapComment(text string) string {
	var b, line strings.Builder
	for _, w := range strings.Fields(text) {
		if line.Len() > 0 && line.Len()+1+len(w) > 74 {
			b.WriteString("// " + line.String() + "\n")
			line.Reset()
		}
		if line.Len() > 0 {
			line.WriteString(" ")
		}
		line.WriteString(w)
	}
	if line.Len() > 0 {
		b.WriteString("// " + line.String() + "\n")
	}
	return b.String()
}
