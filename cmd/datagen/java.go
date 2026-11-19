package main

import (
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
)

// Mobius Java sources that datagen reads, relative to the project folder.
const (
	javaRoot          = "java/org/l2jmobius/gameserver/"
	statFile          = javaRoot + "mechanics/stats/Stat.java"
	bodyPartFile      = javaRoot + "entity/item/enums/BodyPart.java"
	itemSkillTypeFile = javaRoot + "entity/item/enums/ItemSkillType.java"
	weaponTypeFile    = javaRoot + "entity/item/type/WeaponType.java"
	armorTypeFile     = javaRoot + "entity/item/type/ArmorType.java"
	etcItemTypeFile   = javaRoot + "entity/item/type/EtcItemType.java"
	crystalTypeFile   = javaRoot + "entity/item/type/CrystalType.java"
	actionTypeFile    = javaRoot + "entity/item/type/ActionType.java"
	materialTypeFile  = javaRoot + "entity/item/type/MaterialType.java"
	inventoryFile     = javaRoot + "entity/itemcontainer/Inventory.java"
	raceFile          = javaRoot + "entity/actor/enums/creature/Race.java"
	playerClassFile   = javaRoot + "entity/actor/enums/player/PlayerClass.java"
)

// javaConstant is one constant of a Java enum: its name and the source
// text of its constructor arguments.
type javaConstant struct {
	name string
	args []string
}

// readJavaEnum returns the constants of `enum name` in the Java file, in
// declaration order (the order gives the Mobius ordinals).
func readJavaEnum(fsys fs.FS, file, name string) ([]javaConstant, error) {
	src, err := fs.ReadFile(fsys, file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	consts, err := parseJavaEnum(stripJavaComments(string(src)), name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return consts, nil
}

var javaIdentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*`)

// parseJavaEnum reads the constants of `enum name` from Java source
// without comments.
func parseJavaEnum(src, name string) ([]javaConstant, error) {
	header := regexp.MustCompile(`\benum\s+` + regexp.QuoteMeta(name) + `\b[^{;]*\{`)
	loc := header.FindStringIndex(src)
	if loc == nil {
		return nil, fmt.Errorf("enum %s not found", name)
	}
	parts, err := splitJava(src[loc[1]:], true)
	if err != nil {
		return nil, fmt.Errorf("enum %s: %w", name, err)
	}
	var consts []javaConstant
	seen := make(map[string]bool)
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			if i == len(parts)-1 {
				continue // after a trailing comma, or no constants at all
			}
			return nil, fmt.Errorf("enum %s: empty constant", name)
		}
		c, err := parseJavaConstant(part)
		if err != nil {
			return nil, fmt.Errorf("enum %s: %w", name, err)
		}
		if seen[c.name] {
			return nil, fmt.Errorf("enum %s: constant %s declared twice", name, c.name)
		}
		seen[c.name] = true
		consts = append(consts, c)
	}
	if len(consts) == 0 {
		return nil, fmt.Errorf("enum %s has no constants", name)
	}
	return consts, nil
}

// parseJavaConstant reads `NAME`, `NAME(args)` or `NAME(args) { body }`.
func parseJavaConstant(s string) (javaConstant, error) {
	name := javaIdentRe.FindString(s)
	if name == "" {
		return javaConstant{}, fmt.Errorf("cannot read enum constant %q", s)
	}
	c := javaConstant{name: name}
	rest := strings.TrimSpace(s[len(name):])
	if strings.HasPrefix(rest, "(") {
		end, err := closing(rest, 0)
		if err != nil {
			return javaConstant{}, fmt.Errorf("constant %s: %w", name, err)
		}
		args, err := splitJava(rest[1:end]+")", false)
		if err != nil {
			return javaConstant{}, fmt.Errorf("constant %s: %w", name, err)
		}
		for _, a := range args {
			if a = strings.TrimSpace(a); a != "" {
				c.args = append(c.args, a)
			} else if len(args) > 1 {
				return javaConstant{}, fmt.Errorf("constant %s: empty argument", name)
			}
		}
		rest = strings.TrimSpace(rest[end+1:])
	}
	if strings.HasPrefix(rest, "{") {
		end, err := closing(rest, 0)
		if err != nil {
			return javaConstant{}, fmt.Errorf("constant %s: %w", name, err)
		}
		rest = strings.TrimSpace(rest[end+1:])
	}
	if rest != "" {
		return javaConstant{}, fmt.Errorf("cannot read enum constant %q", s)
	}
	return c, nil
}

// closing returns the index of the bracket that closes the one at s[i].
func closing(s string, i int) (int, error) {
	depth := 0
	for j := i; j < len(s); j++ {
		switch s[j] {
		case '"', '\'':
			j = javaLiteralEnd(s, j) - 1
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
			if depth == 0 {
				return j, nil
			}
		}
	}
	return 0, fmt.Errorf("unbalanced brackets in %q", s)
}

// splitJava splits s on the commas outside brackets and literals, up to
// the end of a list: for enum constants (constants true) the first ';' or
// unmatched '}', for arguments the unmatched ')'.
func splitJava(s string, constants bool) ([]string, error) {
	var parts []string
	depth, start := 0, 0
	for j := 0; j < len(s); j++ {
		switch c := s[j]; c {
		case '"', '\'':
			j = javaLiteralEnd(s, j) - 1
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			if depth == 0 {
				if (constants && c == '}') || (!constants && c == ')') {
					return append(parts, s[start:j]), nil
				}
				return nil, fmt.Errorf("unexpected %q", c)
			}
			depth--
		case ';':
			if depth == 0 && constants {
				return append(parts, s[start:j]), nil
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:j])
				start = j + 1
			}
		}
	}
	return nil, fmt.Errorf("list is not closed")
}

// javaLiteralEnd returns the index after the string or char literal that
// starts at s[i].
func javaLiteralEnd(s string, i int) int {
	q := s[i]
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case q:
			return j + 1
		case '\n':
			return j
		}
	}
	return len(s)
}

// stripJavaComments replaces the comments in Java source with spaces,
// keeping string and char literals intact.
func stripJavaComments(src string) string {
	var b strings.Builder
	for i := 0; i < len(src); i++ {
		switch {
		case src[i] == '"' || src[i] == '\'':
			end := javaLiteralEnd(src, i)
			b.WriteString(src[i:end])
			i = end - 1
		case strings.HasPrefix(src[i:], "//"):
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				return b.String()
			}
			b.WriteByte(' ')
			i += end - 1 // the newline is copied next
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return b.String()
			}
			b.WriteByte(' ')
			i += 2 + end + 1
		default:
			b.WriteByte(src[i])
		}
	}
	return b.String()
}

// javaInt parses a Java int literal (decimal or hex, maybe negative).
func javaInt(arg string) (int, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(arg), 0, 32)
	if err != nil {
		return 0, fmt.Errorf("not an int literal: %q", arg)
	}
	return int(v), nil
}

// javaString parses a Java string literal.
func javaString(arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if len(arg) < 2 || arg[0] != '"' || arg[len(arg)-1] != '"' {
		return "", fmt.Errorf("not a string literal: %q", arg)
	}
	s, err := strconv.Unquote(arg)
	if err != nil {
		return "", fmt.Errorf("not a string literal: %q", arg)
	}
	return s, nil
}

// javaBool parses a Java boolean literal.
func javaBool(arg string) (bool, error) {
	switch strings.TrimSpace(arg) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("not a boolean literal: %q", arg)
}

// javaRef returns NAME from `Class.NAME` for the given class.
func javaRef(arg, class string) (string, error) {
	name, ok := strings.CutPrefix(strings.TrimSpace(arg), class+".")
	if !ok || name == "" || javaIdentRe.FindString(name) != name {
		return "", fmt.Errorf("not a %s constant: %q", class, arg)
	}
	return name, nil
}

var javaIntConstRe = regexp.MustCompile(`(?m)^\s*public\s+static\s+final\s+int\s+([A-Z][A-Z0-9_]*)\s*=\s*(-?\d+)\s*;`)

// readJavaIntConstants returns the `public static final int` constants
// with literal values of a Java file.
func readJavaIntConstants(fsys fs.FS, file string) (map[string]int, error) {
	src, err := fs.ReadFile(fsys, file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file, err)
	}
	consts := make(map[string]int)
	for _, m := range javaIntConstRe.FindAllStringSubmatch(stripJavaComments(string(src)), -1) {
		v, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, m[1], err)
		}
		consts[m[1]] = v
	}
	return consts, nil
}
