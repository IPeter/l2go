package scripting

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// typesFile is the LuaLS definitions file shipped with the scripts.
var typesFile = filepath.Join("..", "..", "scripts", "types", "l2go.lua")

func readTypesFile(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(typesFile)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

// apiFunctions returns "module.function" for every Go function scripts can
// call through the API modules.
func apiFunctions() []string {
	var names []string
	for _, m := range (&vm{}).modules() {
		for f := range m.funcs {
			names = append(names, m.name+"."+f)
		}
	}
	slices.Sort(names)
	return names
}

// apiMethods returns "Class:method" for every method of the userdata
// classes.
func apiMethods() []string {
	var names []string
	for _, typ := range (&vm{}).types() {
		for m := range typ.methods {
			names = append(names, typ.name+":"+m)
		}
	}
	slices.Sort(names)
	return names
}

// declared returns the names the declaration pattern decl captures (two
// groups, joined with sep), sorted.
func declared(lines []string, decl *regexp.Regexp, sep string) []string {
	var names []string
	for _, line := range lines {
		if m := decl.FindStringSubmatch(line); m != nil {
			names = append(names, m[1]+sep+m[2])
		}
	}
	slices.Sort(names)
	return names
}

func sameNames(t *testing.T, what string, api, documented []string) {
	t.Helper()
	for _, f := range api {
		if !slices.Contains(documented, f) {
			t.Errorf("%s %s is registered in Go but missing from %s", what, f, typesFile)
		}
	}
	for _, f := range documented {
		if !slices.Contains(api, f) {
			t.Errorf("%s %s is in %s but not registered in Go", what, f, typesFile)
		}
	}
}

// Every function and method registered in Go is documented in the types
// file, and the types file documents nothing that does not exist.
func TestTypesFileListsAPI(t *testing.T) {
	lines := readTypesFile(t)
	if len(lines) == 0 || lines[0] != "---@meta" {
		t.Errorf("%s must start with ---@meta", typesFile)
	}
	funcDecl := regexp.MustCompile(`^function ([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\(`)
	methodDecl := regexp.MustCompile(`^function ([A-Za-z_][A-Za-z0-9_]*):([A-Za-z_][A-Za-z0-9_]*)\(`)
	sameNames(t, "function", apiFunctions(), declared(lines, funcDecl, "."))
	sameNames(t, "method", apiMethods(), declared(lines, methodDecl, ":"))

	for _, m := range (&vm{}).modules() {
		if !slices.Contains(lines, m.name+" = {}") {
			t.Errorf("%s does not declare the %s table", typesFile, m.name)
		}
	}
	for _, typ := range (&vm{}).types() {
		if !slices.Contains(lines, "---@class "+typ.name) {
			t.Errorf("%s does not declare the class %s", typesFile, typ.name)
		}
	}
}

// luaName turns an event name into its LuaLS type prefix:
// "character_create" is "CharacterCreate".
func luaName(event string) string {
	var b strings.Builder
	for part := range strings.SplitSeq(event, "_") {
		if part != "" {
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return b.String()
}

// block returns the "---" comment lines after the line that equals header.
func block(lines []string, header string) ([]string, bool) {
	i := slices.Index(lines, header)
	if i < 0 {
		return nil, false
	}
	var out []string
	for _, line := range lines[i+1:] {
		if !strings.HasPrefix(line, "---") {
			break
		}
		out = append(out, line)
	}
	return out, true
}

// aliasValues returns the quoted values of an ---@alias block.
func aliasValues(lines []string) []string {
	value := regexp.MustCompile(`^---\| "([a-z_]+)"`)
	var out []string
	for _, line := range lines {
		if m := value.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

// The event table fields and the reject reasons of every event are
// documented.
func TestTypesFileDescribesEvents(t *testing.T) {
	lines := readTypesFile(t)
	field := regexp.MustCompile(`^---@field ([a-z_]+) `)

	// Samples with every optional field set.
	samples := map[string]Event{
		EventCharacterCreate: CharacterCreate{}.Event(),
		EventChat:            ChatMessage{Player: hero(), Type: 2, Text: "hi", Target: "Friend"}.Event(),
	}
	for _, spec := range eventSpecs {
		name := luaName(spec.name)

		ev, ok := samples[spec.name]
		if !ok {
			t.Errorf("no sample event for %s in this test", spec.name)
			continue
		}
		fields, ok := block(lines, "---@class "+name+"Event")
		if !ok {
			t.Errorf("%s has no ---@class %sEvent", typesFile, name)
		}
		var documented []string
		for _, line := range fields {
			if m := field.FindStringSubmatch(line); m != nil {
				documented = append(documented, m[1])
			}
		}
		for key := range ev.Data {
			if !slices.Contains(documented, key) {
				t.Errorf("%sEvent: field %s is missing from %s", name, key, typesFile)
			}
		}
		for _, key := range documented {
			if _, ok := ev.Data[key]; !ok {
				t.Errorf("%sEvent: field %s is documented but not sent", name, key)
			}
		}

		if len(spec.reasons) == 0 {
			continue
		}
		alias, ok := block(lines, "---@alias "+name+"Reason")
		if !ok {
			t.Errorf("%s has no ---@alias %sReason", typesFile, name)
		}
		if reasons := aliasValues(alias); !slices.Equal(reasons, spec.reasons) {
			t.Errorf("%sReason = %v, want %v", name, reasons, spec.reasons)
		}
	}
}

// The ChatType alias lists every chat type name, in ID order.
func TestTypesFileListsChatTypes(t *testing.T) {
	alias, ok := block(readTypesFile(t), "---@alias ChatType")
	if !ok {
		t.Fatalf("%s has no ---@alias ChatType", typesFile)
	}
	want := append(slices.Clone(chatTypes), chatTypeUnknown)
	if got := aliasValues(alias); !slices.Equal(got, want) {
		t.Errorf("ChatType = %v, want %v", got, want)
	}
}

func TestReasons(t *testing.T) {
	r := Reasons(EventCharacterCreate)
	want := []string{
		ReasonCreationFailed, ReasonTooManyCharacters, ReasonNameExists,
		ReasonNameTooLong, ReasonIncorrectName, ReasonCreationNotAllowed,
	}
	if !slices.Equal(r, want) {
		t.Errorf("Reasons = %v, want %v", r, want)
	}
	r[0] = "changed"
	if Reasons(EventCharacterCreate)[0] != ReasonCreationFailed {
		t.Error("Reasons returns the internal slice")
	}
	if Reasons(EventChat) != nil {
		t.Error("chat events have reject reasons")
	}
	if Reasons("no_such_event") != nil {
		t.Error("Reasons of an unknown event is not nil")
	}
}
