package main

import (
	"fmt"
	"strings"
)

// Lua operator precedence, lowest first.
const (
	precOr     = 1
	precAnd    = 2
	precCmp    = 3
	precConcat = 4
	precAdd    = 5
	precMul    = 6
	precUnary  = 7
	precPow    = 8
	precAtom   = 10
)

// lx is a translated Lua expression with what the translator knows
// about it.
type lx struct {
	code string
	prec int    // precedence of the outermost Lua operator
	t    *jtype // static Java type
	safe bool   // never nil or false, so "c and x or y" is exact
	call bool   // a Lua function call, valid as a statement
	pure bool   // no side effects and cheap: a name, literal or field
	todo bool   // contains a J.todo marker
	base  string // for views (player.getVariables()): the object they belong to
	refID string // for fields keeping a world object: the stored object ID

	isType  string // the expression names a type (a class used as qualifier)
	packet  string // a server packet built with new, see mapping.go
	pargs   []lx   // the packet's constructor arguments
	nilLit  bool   // the null literal
	strLit  bool   // a string literal
	intLit  bool   // an integer literal
	intVal  int64  // its value
	entryOf string // for Map.Entry loop variables: the key name; value name in base
}

func atom(code string, t *jtype) lx { return lx{code: code, prec: precAtom, t: t} }

// paren wraps e in parentheses when its precedence is below min.
func paren(e lx, min int) string {
	if e.prec < min {
		return "(" + e.code + ")"
	}
	return e.code
}

// asAtom returns e usable before ':', '.' or '['.
func asAtom(e lx) string {
	if e.prec < precAtom || strings.HasPrefix(e.code, "{") || strings.HasPrefix(e.code, "\"") {
		return "(" + e.code + ")"
	}
	return e.code
}

// luaQuote returns s as a Lua string literal.
func luaQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 || c == 0x7f {
				fmt.Fprintf(&b, `\%03d`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

var luaKeywords = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true, "end": true,
	"false": true, "for": true, "function": true, "goto": true, "if": true, "in": true,
	"local": true, "nil": true, "not": true, "or": true, "repeat": true, "return": true,
	"then": true, "true": true, "until": true, "while": true,
}

// luaGlobals are names the generated code uses; Java names that clash get
// a trailing underscore.
var luaGlobals = map[string]bool{
	"J": true, "q": true, "K": true, "self": true, "quest": true, "script": true, "saga": true,
	"rnd": true, "State": true, "QuestSound": true, "QuestType": true, "ClassId": true,
	"Race": true, "Category": true, "ChatType": true, "Paperdoll": true,
	"SystemMessageId": true, "Config": true, "Sex": true, "Intention": true,
	"world": true, "globals": true, "time": true, "util": true, "skills": true, "items": true,
	"math": true, "string": true, "table": true, "ipairs": true, "pairs": true,
	"unpack": true, "tostring": true, "tonumber": true, "type": true, "select": true,
	"next": true, "error": true, "pcall": true, "setmetatable": true, "getmetatable": true,
	"rawget": true, "rawset": true, "rawequal": true, "log": true, "html": true,
	"events": true, "print": true, "assert": true, "require": true, "_G": true,
}

// luaName makes a Java identifier safe as a Lua name.
func luaName(name string) string {
	name = strings.ReplaceAll(name, "$", "_")
	if luaKeywords[name] || luaGlobals[name] {
		return name + "_"
	}
	return name
}

// luaWriter collects indented Lua lines.
type luaWriter struct {
	lines []string
	ind   int
}

func (w *luaWriter) line(s string) {
	if s == "" {
		w.lines = append(w.lines, "")
		return
	}
	w.lines = append(w.lines, strings.Repeat("  ", w.ind)+s)
}

func (w *luaWriter) linef(format string, args ...any) { w.line(fmt.Sprintf(format, args...)) }

// blank adds an empty line unless the last line is already empty or opens
// a block.
func (w *luaWriter) blank() {
	if len(w.lines) == 0 || w.lines[len(w.lines)-1] == "" {
		return
	}
	w.lines = append(w.lines, "")
}

// comments writes source comments as Lua comment lines.
func (w *luaWriter) comments(cs []comment) {
	for _, c := range cs {
		if skipComment(c.text) {
			continue
		}
		for _, l := range strings.Split(c.text, "\n") {
			l = strings.TrimRight(l, " \t")
			if l == "" {
				w.line("--")
			} else {
				w.line("-- " + l)
			}
		}
	}
}

// trail appends same-line comments to the last line.
func (w *luaWriter) trail(cs []comment) {
	if len(w.lines) == 0 {
		return
	}
	var parts []string
	for _, c := range cs {
		if skipComment(c.text) || strings.Contains(c.text, "\n") {
			continue
		}
		parts = append(parts, c.text)
	}
	if len(parts) == 0 {
		return
	}
	last := len(w.lines) - 1
	w.lines[last] += " -- " + strings.Join(parts, " ")
}

func skipComment(text string) bool {
	return strings.HasPrefix(text, "@formatter") || strings.TrimSpace(text) == ""
}

func (w *luaWriter) String() string {
	return strings.Join(w.lines, "\n") + "\n"
}

// snake converts camelCase to snake_case.
func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 && s[i-1] != '_' {
				prev := rune(s[i-1])
				nextLower := i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z'
				if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || nextLower {
					b.WriteByte('_')
				}
			}
			b.WriteRune(r - 'A' + 'a')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
