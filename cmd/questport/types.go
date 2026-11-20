package main

import "strings"

// typeKind classifies static Java types.
type typeKind int

const (
	tkUnknown typeKind = iota
	tkVoid
	tkNull
	tkPrim
	tkClass
	tkArray
)

// jtype is a static Java type as far as the translator tracks it.
type jtype struct {
	kind typeKind
	name string   // primitive or simple class name
	args []*jtype // generic arguments
	elem *jtype   // array element type
}

var (
	tyUnknown = &jtype{kind: tkUnknown, name: "?"}
	tyVoid    = &jtype{kind: tkVoid, name: "void"}
	tyNull    = &jtype{kind: tkNull, name: "null"}
	tyInt     = &jtype{kind: tkPrim, name: "int"}
	tyLong    = &jtype{kind: tkPrim, name: "long"}
	tyDouble  = &jtype{kind: tkPrim, name: "double"}
	tyBool    = &jtype{kind: tkPrim, name: "boolean"}
	tyChar    = &jtype{kind: tkPrim, name: "char"}
	tyString  = &jtype{kind: tkClass, name: "String"}
)

func classType(name string, args ...*jtype) *jtype {
	return &jtype{kind: tkClass, name: name, args: args}
}

func arrayOf(elem *jtype) *jtype { return &jtype{kind: tkArray, name: elem.name + "[]", elem: elem} }

func (t *jtype) String() string {
	if t == nil {
		return "?"
	}
	switch t.kind {
	case tkArray:
		return t.elem.String() + "[]"
	case tkClass:
		if len(t.args) > 0 {
			parts := make([]string, len(t.args))
			for i, a := range t.args {
				parts[i] = a.String()
			}
			return t.name + "<" + strings.Join(parts, ",") + ">"
		}
	}
	return t.name
}

// arg returns generic argument i, or unknown.
func (t *jtype) arg(i int) *jtype {
	if t == nil || i >= len(t.args) || t.args[i] == nil {
		return tyUnknown
	}
	return t.args[i]
}

func isUnknown(t *jtype) bool { return t == nil || t.kind == tkUnknown }

func isIntegral(t *jtype) bool {
	if t == nil {
		return false
	}
	switch t.name {
	case "int", "long", "short", "byte", "Integer", "Long", "Short", "Byte":
		return t.kind == tkPrim || t.kind == tkClass
	}
	return false
}

func isFloating(t *jtype) bool {
	if t == nil {
		return false
	}
	switch t.name {
	case "double", "float", "Double", "Float":
		return t.kind == tkPrim || t.kind == tkClass
	}
	return false
}

func isNumeric(t *jtype) bool { return isIntegral(t) || isFloating(t) }

func isBool(t *jtype) bool {
	return t != nil && (t.name == "boolean" || t.name == "Boolean") && t.kind != tkArray
}

func isString(t *jtype) bool { return t != nil && t.kind == tkClass && t.name == "String" }

func isChar(t *jtype) bool {
	return t != nil && (t.name == "char" || t.name == "Character") && t.kind != tkArray
}

func isArray(t *jtype) bool { return t != nil && t.kind == tkArray }

var mapTypes = map[string]bool{
	"Map": true, "HashMap": true, "ConcurrentHashMap": true, "LinkedHashMap": true,
	"TreeMap": true, "EnumMap": true, "ConcurrentSkipListMap": true,
}

var listTypes = map[string]bool{
	"List": true, "ArrayList": true, "LinkedList": true, "CopyOnWriteArrayList": true,
	"Collection": true, "Iterable": true, "Queue": true, "Deque": true, "ArrayDeque": true,
	"Vector": true, "Stack": true,
}

var setTypes = map[string]bool{
	"Set": true, "HashSet": true, "LinkedHashSet": true, "TreeSet": true,
	"ConcurrentSkipListSet": true, "EnumSet": true,
}

func isMap(t *jtype) bool  { return t != nil && t.kind == tkClass && mapTypes[t.name] }
func isList(t *jtype) bool { return t != nil && t.kind == tkClass && listTypes[t.name] }
func isSet(t *jtype) bool  { return t != nil && t.kind == tkClass && setTypes[t.name] }

// isSequence reports whether t is a Lua array in the output: Java arrays
// and lists.
func isSequence(t *jtype) bool { return isArray(t) || isList(t) }

// elemType is the element type of an array, list or set.
func elemType(t *jtype) *jtype {
	switch {
	case isArray(t):
		return t.elem
	case isList(t), isSet(t):
		return t.arg(0)
	}
	return tyUnknown
}

// superTypes lists the class hierarchy used to find mappings, nearest
// first. Only the classes quests touch are listed.
var superTypes = map[string][]string{
	"Player":        {"Playable", "Creature", "WorldObject"},
	"Playable":      {"Creature", "WorldObject"},
	"Summon":        {"Playable", "Creature", "WorldObject"},
	"Pet":           {"Summon", "Playable", "Creature", "WorldObject"},
	"Servitor":      {"Summon", "Playable", "Creature", "WorldObject"},
	"Npc":           {"Creature", "WorldObject"},
	"Attackable":    {"Npc", "Creature", "WorldObject"},
	"Monster":       {"Attackable", "Npc", "Creature", "WorldObject"},
	"RaidBoss":      {"Monster", "Attackable", "Npc", "Creature", "WorldObject"},
	"GrandBoss":     {"Monster", "Attackable", "Npc", "Creature", "WorldObject"},
	"Chest":         {"Monster", "Attackable", "Npc", "Creature", "WorldObject"},
	"Folk":          {"Npc", "Creature", "WorldObject"},
	"Merchant":      {"Folk", "Npc", "Creature", "WorldObject"},
	"Creature":      {"WorldObject"},
	"Door":          {"Creature", "WorldObject"},
	"FeedableBeast": {"Monster", "Attackable", "Npc", "Creature", "WorldObject"},
	"TamedBeast":    {"FeedableBeast", "Monster", "Attackable", "Npc", "Creature", "WorldObject"},
	"Integer":       {"Number"},
	"Long":          {"Number"},
}

// typeChain returns t's name followed by its supertypes.
func typeChain(t *jtype) []string {
	if t == nil {
		return nil
	}
	switch {
	case isMap(t):
		return []string{t.name, "Map"}
	case isList(t):
		return []string{t.name, "List"}
	case isSet(t):
		return []string{t.name, "Set"}
	case isString(t):
		return []string{"String"}
	case isArray(t):
		return []string{"Array"}
	}
	return append([]string{t.name}, superTypes[t.name]...)
}

// isA reports whether t is the class name or one of its subclasses.
func isA(t *jtype, name string) bool {
	for _, n := range typeChain(t) {
		if n == name {
			return true
		}
	}
	return false
}

// refTypes are world objects and handles that the Lua API passes as
// references valid for one run only.
var refTypes = map[string]bool{
	"Player": true, "Playable": true, "Summon": true, "Pet": true, "Servitor": true,
	"Npc": true, "Attackable": true, "Monster": true, "RaidBoss": true, "GrandBoss": true,
	"Chest": true, "Folk": true, "Merchant": true, "Creature": true, "WorldObject": true,
	"Door": true, "QuestState": true, "Item": true, "FeedableBeast": true, "TamedBeast": true,
}

func isRef(t *jtype) bool { return t != nil && t.kind == tkClass && refTypes[t.name] }

// typeFromRef converts a source type.
func typeFromRef(tr *typeRef) *jtype {
	if tr == nil {
		return tyVoid
	}
	var base *jtype
	switch tr.name {
	case "int", "short", "byte":
		base = tyInt
	case "long":
		base = tyLong
	case "double", "float":
		base = tyDouble
	case "boolean":
		base = tyBool
	case "char":
		base = tyChar
	case "void":
		base = tyVoid
	case "String":
		base = tyString
	case "var":
		base = tyUnknown
	default:
		args := make([]*jtype, len(tr.args))
		for i, a := range tr.args {
			args[i] = typeFromRef(a)
		}
		base = classType(tr.name, args...)
	}
	for i := 0; i < tr.dims; i++ {
		base = arrayOf(base)
	}
	return base
}

// binaryNumeric is Java's binary numeric promotion.
func binaryNumeric(a, b *jtype) *jtype {
	switch {
	case isFloating(a) || isFloating(b):
		return tyDouble
	case a != nil && (a.name == "long" || a.name == "Long"), b != nil && (b.name == "long" || b.name == "Long"):
		return tyLong
	case isUnknown(a) || isUnknown(b):
		if isIntegral(a) || isIntegral(b) {
			return tyUnknown
		}
		return tyUnknown
	}
	return tyInt
}

// enumTables maps Mobius enum and constant classes to the global Lua
// tables that hold their values.
var enumTables = map[string]string{
	"State":             "State",
	"QuestSound":        "QuestSound",
	"QuestType":         "QuestType",
	"PlayerClass":       "ClassId",
	"Race":              "Race",
	"ChatType":          "ChatType",
	"CategoryType":      "Category",
	"SystemMessageId":   "SystemMessageId",
	"Sex":               "Sex",
	"Intention":         "Intention",
	"SkillType":         "SkillType",
	"WeaponType":        "WeaponType",
	"AbnormalType":      "AbnormalType",
	"ItemProcessType":   "ItemProcessType",
	"TeleportWhereType": "TeleportWhereType",
	"Stat":              "Stat",
}

// enumValueType is the static type of a value of the enum class.
func enumValueType(class string) *jtype {
	switch class {
	case "State":
		return tyInt
	case "QuestSound":
		return classType("QuestSound")
	}
	return classType(class)
}
