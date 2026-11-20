package main

import (
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The Mobius sources of the script constants, relative to the project
// folder (see data.ConstantTableNames).
const (
	questSoundFile      = javaRoot + "mechanics/script/QuestSound.java"
	questStateFile      = javaRoot + "mechanics/script/State.java"
	questTypeFile       = javaRoot + "mechanics/script/QuestType.java"
	chatTypeFile        = javaRoot + "network/enums/ChatType.java"
	sexFile             = javaRoot + "entity/actor/enums/player/Sex.java"
	systemMessageIDFile = javaRoot + "network/SystemMessageId.java"
)

// paperdollPrefix starts the Inventory slot constants; the Paperdoll table
// drops it (PAPERDOLL_RHAND is Paperdoll.RHAND).
const paperdollPrefix = "PAPERDOLL_"

// The constants domain writes the global tables of the Lua scripts, loaded
// by data.LoadConstants.
func init() { registerDomain("constants", genConstants) }

// genConstants reads the constant tables and adds constants/constants.json.
// ClassId, Race and Category come from the generated enums of
// internal/data (themselves generated from Mobius by the -enums mode), so
// the scripts and the Go code use the same values.
func genConstants(o options) error {
	readers := []func(options) (data.ConstantTable, error){
		enumConstants(data.ConstCategory, "data/enums/CategoryType", data.NumCategoryTypes,
			func(i int) string { return data.CategoryType(i).String() }),
		readChatTypes,
		classConstants,
		readPaperdoll,
		readQuestSounds,
		readQuestTypes,
		enumConstants(data.ConstRace, "entity/actor/enums/creature/Race", data.NumRaces,
			func(i int) string { return data.Race(i).String() }),
		readSexes,
		readQuestStates,
		readSystemMessageIDs,
	}
	tables := make([]data.ConstantTable, 0, len(readers))
	for _, read := range readers {
		t, err := read(o)
		if err != nil {
			return fmt.Errorf("constants: %w", err)
		}
		tables = append(tables, t)
	}
	if _, err := data.NewConstants(tables); err != nil {
		return err
	}
	if err := o.out.addJSON(data.ConstantsFile, tables); err != nil {
		return err
	}
	parts := make([]string, len(tables))
	for i, t := range tables {
		parts[i] = fmt.Sprintf("%s %d", t.Name, t.Len())
	}
	fmt.Fprintf(o.log, "constants: %s\n", strings.Join(parts, ", "))
	return nil
}

// enumConstants returns a reader of a table holding the n values of a
// generated enum, named by name(i).
func enumConstants(table, source string, n int, name func(int) string) func(options) (data.ConstantTable, error) {
	return func(options) (data.ConstantTable, error) {
		t := data.ConstantTable{Name: table, Source: source, Ints: make(map[string]int, n)}
		for i := range n {
			t.Ints[name(i)] = i
		}
		return t, nil
	}
}

// classConstants is the ClassId table: the class IDs of the generated
// PlayerClass enum.
func classConstants(options) (data.ConstantTable, error) {
	t := data.ConstantTable{Name: data.ConstClassID, Source: "entity/actor/enums/player/PlayerClass", Ints: make(map[string]int)}
	for _, c := range data.PlayerClasses() {
		t.Ints[c.String()] = int(c)
	}
	return t, nil
}

// readChatTypes reads ChatType: each constant's client ID, its only
// argument.
func readChatTypes(o options) (data.ConstantTable, error) {
	consts, err := simpleEnum(o, chatTypeFile, "ChatType", 1)
	if err != nil {
		return data.ConstantTable{}, err
	}
	t := data.ConstantTable{Name: data.ConstChatType, Source: "network/enums/ChatType", Ints: make(map[string]int, len(consts))}
	for _, c := range consts {
		v, err := javaInt(c.args[0])
		if err != nil {
			return data.ConstantTable{}, fmt.Errorf("%s: ChatType.%s: %w", chatTypeFile, c.name, err)
		}
		t.Ints[c.name] = v
	}
	return t, nil
}

// ordinalConstants reads an enum without arguments as a table of its
// ordinals.
func ordinalConstants(o options, file, enum, table, source string) (data.ConstantTable, error) {
	consts, err := simpleEnum(o, file, enum, 0)
	if err != nil {
		return data.ConstantTable{}, err
	}
	t := data.ConstantTable{Name: table, Source: source, Ints: make(map[string]int, len(consts))}
	for i, c := range consts {
		t.Ints[c.name] = i
	}
	return t, nil
}

func readQuestTypes(o options) (data.ConstantTable, error) {
	return ordinalConstants(o, questTypeFile, "QuestType", data.ConstQuestType, "mechanics/script/QuestType")
}

func readSexes(o options) (data.ConstantTable, error) {
	return ordinalConstants(o, sexFile, "Sex", data.ConstSex, "entity/actor/enums/player/Sex")
}

// readQuestSounds reads QuestSound: each constant's argument is
// new PlaySound("name").
func readQuestSounds(o options) (data.ConstantTable, error) {
	consts, err := simpleEnum(o, questSoundFile, "QuestSound", 1)
	if err != nil {
		return data.ConstantTable{}, err
	}
	t := data.ConstantTable{Name: data.ConstQuestSound, Source: "mechanics/script/QuestSound", Strings: make(map[string]string, len(consts))}
	for _, c := range consts {
		arg, ok := strings.CutPrefix(strings.TrimSpace(c.args[0]), "new PlaySound(")
		arg, ok2 := strings.CutSuffix(arg, ")")
		if !ok || !ok2 {
			return data.ConstantTable{}, fmt.Errorf("%s: QuestSound.%s: argument %q is not new PlaySound(\"name\")", questSoundFile, c.name, c.args[0])
		}
		name, err := javaString(arg)
		if err != nil {
			return data.ConstantTable{}, fmt.Errorf("%s: QuestSound.%s: %w", questSoundFile, c.name, err)
		}
		t.Strings[c.name] = name
	}
	return t, nil
}

var javaByteConstRe = regexp.MustCompile(`(?m)^\s*public\s+static\s+final\s+byte\s+([A-Z][A-Z0-9_]*)\s*=\s*(-?\d+)\s*;`)

// readQuestStates reads the byte constants of State: CREATED, STARTED and
// COMPLETED.
func readQuestStates(o options) (data.ConstantTable, error) {
	src, err := fs.ReadFile(o.src, questStateFile)
	if err != nil {
		return data.ConstantTable{}, fmt.Errorf("read %s: %w", questStateFile, err)
	}
	t := data.ConstantTable{Name: data.ConstState, Source: "mechanics/script/State", Ints: make(map[string]int)}
	for _, m := range javaByteConstRe.FindAllStringSubmatch(stripJavaComments(string(src)), -1) {
		v, err := strconv.Atoi(m[2])
		if err != nil {
			return data.ConstantTable{}, fmt.Errorf("%s: %s: %w", questStateFile, m[1], err)
		}
		t.Ints[m[1]] = v
	}
	for _, name := range []string{"CREATED", "STARTED", "COMPLETED"} {
		if _, ok := t.Ints[name]; !ok {
			return data.ConstantTable{}, fmt.Errorf("%s: State.%s not found", questStateFile, name)
		}
	}
	return t, nil
}

// readPaperdoll reads the PAPERDOLL_* slots of Inventory, without
// PAPERDOLL_TOTALSLOTS (data.PaperdollSlots).
func readPaperdoll(o options) (data.ConstantTable, error) {
	consts, err := readJavaIntConstants(o.src, inventoryFile)
	if err != nil {
		return data.ConstantTable{}, err
	}
	t := data.ConstantTable{Name: data.ConstPaperdoll, Source: "entity/itemcontainer/Inventory", Ints: make(map[string]int)}
	for name, v := range consts {
		slot, ok := strings.CutPrefix(name, paperdollPrefix)
		if !ok || slot == "TOTALSLOTS" {
			continue
		}
		t.Ints[slot] = v
	}
	if _, ok := t.Ints["RHAND"]; !ok {
		return data.ConstantTable{}, fmt.Errorf("%s: %sRHAND not found", inventoryFile, paperdollPrefix)
	}
	return t, nil
}

var (
	// clientStringRe matches one system message declaration:
	// @ClientString(id = 0, message = "...") public static SystemMessageId NAME;
	clientStringRe    = regexp.MustCompile(`@ClientString\s*\(\s*id\s*=\s*(\d+)\s*,\s*message\s*=\s*"(?:[^"\\\n]|\\.)*"\s*\)\s*public\s+static\s+SystemMessageId\s+([A-Za-z_][A-Za-z0-9_]*)\s*;`)
	clientStringAnyRe = regexp.MustCompile(`@ClientString\b`)
)

// readSystemMessageIDs reads the system message IDs of SystemMessageId,
// without their text (the client has it). Every @ClientString must be
// read.
func readSystemMessageIDs(o options) (data.ConstantTable, error) {
	b, err := fs.ReadFile(o.src, systemMessageIDFile)
	if err != nil {
		return data.ConstantTable{}, fmt.Errorf("read %s: %w", systemMessageIDFile, err)
	}
	src := stripJavaComments(string(b))
	t := data.ConstantTable{Name: data.ConstSystemMessageID, Source: "network/SystemMessageId", Ints: make(map[string]int)}
	matches := clientStringRe.FindAllStringSubmatch(src, -1)
	for _, m := range matches {
		id, err := strconv.Atoi(m[1])
		if err != nil {
			return data.ConstantTable{}, fmt.Errorf("%s: %s: %w", systemMessageIDFile, m[2], err)
		}
		if _, dup := t.Ints[m[2]]; dup {
			return data.ConstantTable{}, fmt.Errorf("%s: SystemMessageId.%s is declared twice", systemMessageIDFile, m[2])
		}
		t.Ints[m[2]] = id
	}
	if all := len(clientStringAnyRe.FindAllStringIndex(src, -1)); all != len(matches) {
		return data.ConstantTable{}, fmt.Errorf("%s: read %d of %d @ClientString declarations", systemMessageIDFile, len(matches), all)
	}
	return t, nil
}
