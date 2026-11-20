package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Limits of Lua 5.1 the output stays under: when a file would declare
// more top-level locals, or a function would capture more upvalues, the
// constants move into one table K.
const (
	maxTopLocals = 180
	maxUpvalues  = 55
)

// translateAll translates the named quests from dir into out.
func translateAll(dir string, names []string, out string) (*report, error) {
	rep := &report{}
	for _, name := range names {
		path := filepath.Join(dir, name, name+".java")
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		lua, res := translateQuest(path, string(src), name)
		rep.results = append(rep.results, res)
		if res.err != nil {
			continue
		}
		target := filepath.Join(out, name, name+".lua")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil { //nolint:gosec // the script folder is meant to be readable
			return nil, fmt.Errorf("create output folder: %w", err)
		}
		if err := os.WriteFile(target, []byte(lua), 0o644); err != nil { //nolint:gosec // generated scripts are public
			return nil, fmt.Errorf("write %s: %w", target, err)
		}
		res.out = target
	}
	return rep, nil
}

// translateQuest translates one quest file. On a parse error the result
// carries it and the Lua is empty.
func translateQuest(path, src, name string) (string, *questResult) {
	return translateScript(path, src, name, "quests/"+name)
}

// translateScript translates one Mobius script file; origin is its place
// under scripts/, for the header (e.g. "ai/others/Chests").
func translateScript(path, src, name, origin string) (string, *questResult) {
	res := &questResult{name: name, api: map[string]int{}}
	cu, err := parseJava(path, src)
	if err != nil {
		res.err = err
		return "", res
	}
	var cls *classDecl
	for _, c := range cu.types {
		if c.name == name {
			cls = c
		}
	}
	if cls == nil && len(cu.types) > 0 {
		cls = cu.types[0]
	}
	if cls == nil {
		res.err = fmt.Errorf("%s: no class found", path)
		return "", res
	}
	lines := strings.Split(src, "\n")
	if cls.extends != nil && cls.extends.name == "AbstractSagaQuest" {
		t := newTranslator(path, lines, cls, res, false)
		return t.saga(), res
	}
	t := newTranslator(path, lines, cls, res, false)
	t.origin = origin
	lua := t.quest()
	if t.needTable {
		// Too many locals or upvalues: translate again with the constants
		// in one table.
		res2 := &questResult{name: name, api: map[string]int{}}
		t2 := newTranslator(path, lines, cls, res2, true)
		t2.origin = origin
		lua = t2.quest()
		*res = *res2
	}
	return lua, res
}

// translator translates one quest class.
type translator struct {
	file  string
	lines []string // Java source lines
	cls   *classDecl
	res   *questResult
	w     *luaWriter

	fields   map[string]*fieldInfo
	fieldSeq []*fieldInfo
	methods  map[string][]*methodInfo
	inner    map[string]*innerClass
	idents   map[string]bool // every identifier in the file
	topNames map[string]bool // Lua names declared at the top level

	constTable bool // constants live in table K
	needTable  bool // limits exceeded: translate again with constTable
	topLocals  int

	fn        *funcCtx
	features  map[string]bool
	questID   string
	questDesc string
	extends   string
	origin    string // the file's place under Mobius scripts/, for the header
}

// fieldInfo is a field of the quest class or of an inner class.
type fieldInfo struct {
	name   string
	lua    string // how code reads it
	t      *jtype
	static bool
	final  bool
	early  bool // initialized where it is declared, before methods
	decl   *fieldDecl
	v      *varDecl
	ref    bool // holds a world object across runs: kept as its object ID
	refSeq bool // a collection of world objects kept across runs
	konst  *lx  // the value of a compile-time constant
}

// methodInfo is a method of the quest class or of an inner class.
type methodInfo struct {
	decl     *methodDecl
	lua      string
	params   []*jtype
	ret      *jtype
	varargs  bool
	callback string // Lua handler (on_talk, ...) when this is a callback
	called   bool   // called from the script's own code
}

// innerClass is a nested class, translated into a Lua table with a
// constructor function and methods.
type innerClass struct {
	decl    *classDecl
	lua     string
	fields  map[string]*fieldInfo
	fseq    []*fieldInfo
	methods map[string][]*methodInfo
	ctors   []*methodInfo
}

func newTranslator(path string, lines []string, cls *classDecl, res *questResult, constTable bool) *translator {
	t := &translator{
		file:       path,
		lines:      lines,
		cls:        cls,
		res:        res,
		w:          &luaWriter{},
		fields:     map[string]*fieldInfo{},
		methods:    map[string][]*methodInfo{},
		inner:      map[string]*innerClass{},
		idents:     map[string]bool{},
		topNames:   map[string]bool{},
		constTable: constTable,
		features:   map[string]bool{},
	}
	if cls.extends != nil {
		t.extends = cls.extends.name
	}
	collectIdents(cls, t.idents)
	return t
}

// collectIdents records every identifier, so generated names avoid them.
func collectIdents(cls *classDecl, into map[string]bool) {
	into[cls.name] = true
	for _, m := range cls.members {
		switch v := m.(type) {
		case *fieldDecl:
			for _, d := range v.vars {
				into[d.name] = true
				if d.init != nil {
					collectExprIdents(d.init, into)
				}
			}
		case *methodDecl:
			into[v.name] = true
			for _, p := range v.params {
				into[p.name] = true
			}
			if v.body != nil {
				collectExprIdents(v.body, into)
			}
		case *initBlock:
			collectExprIdents(v.body, into)
		case *classDecl:
			collectIdents(v, into)
		}
	}
}

func collectExprIdents(n any, into map[string]bool) {
	inspect(n, func(x any) bool {
		switch v := x.(type) {
		case *identExpr:
			into[v.name] = true
		case *localVarStmt:
			for _, d := range v.vars {
				into[d.name] = true
			}
		case *forEachStmt:
			into[v.name] = true
		case *lambdaExpr:
			for _, p := range v.params {
				into[p] = true
			}
		}
		return true
	})
}

// --- Notes ---

// todo records an untranslatable construct and returns the J.todo marker
// that replaces it.
func (t *translator) todo(line int, key, reason string) lx {
	t.res.todos = append(t.res.todos, note{line: line, key: key, reason: reason})
	if t.fn != nil {
		t.fn.todoLines = append(t.fn.todoLines, line)
	}
	return lx{code: "J.todo(" + luaQuote(reason) + ")", prec: precAtom, t: tyUnknown, call: true, todo: true}
}

// review records a remark that does not block the translation.
func (t *translator) review(line int, key, reason string) {
	t.res.reviews = append(t.res.reviews, note{line: line, key: key, reason: reason})
}

// useAPI counts a Lua API function the output calls.
func (t *translator) useAPI(name string) {
	if name != "" {
		t.res.api[name]++
	}
}

// requires notes a feature the quest needs (q:requires).
func (t *translator) requires(feature string) {
	if feature != "" {
		t.features[feature] = true
	}
}

// javaLine returns the trimmed Java source line.
func (t *translator) javaLine(n int) string {
	if n < 1 || n > len(t.lines) {
		return ""
	}
	return strings.TrimSpace(strings.TrimRight(t.lines[n-1], "\r"))
}

// flushTodoComments writes the Java lines of the todos found while
// translating the current statement, as comments.
func (t *translator) flushTodoComments() {
	if t.fn == nil || len(t.fn.todoLines) == 0 {
		return
	}
	seen := map[int]bool{}
	for _, l := range t.fn.todoLines {
		if seen[l] {
			continue
		}
		seen[l] = true
		t.w.linef("-- TODO(questport) java:%d: %s", l, t.javaLine(l))
	}
	t.fn.todoLines = nil
}

// --- Functions and scopes ---

// funcCtx is the state of the Lua function being written.
type funcCtx struct {
	parent    *funcCtx
	scope     *scope
	inner     *innerClass // inside an inner class method: fields are self.x
	ret       *jtype
	temps     map[string]bool
	pre       []string // statements to write before the current one
	noHoist   int      // > 0: the current expression may not hoist statements
	todoLines []int
	targets   []*jumpTarget
	upvals    map[string]bool
	world     bool // a world call was translated
	ctor      bool // the quest constructor
	static    bool // a static method or initializer
}

// jumpTarget is a Java loop or switch that break/continue may leave.
type jumpTarget struct {
	loop      bool
	luaLoop   bool   // a Lua loop stands for it: break works directly
	contBreak bool   // continue is written as break (the body runs in repeat ... until true)
	brkFlag   string // break sets this flag before leaving the repeat block
}

type scope struct {
	parent *scope
	vars   map[string]*localVar
}

// localVar is a Java local variable or parameter.
type localVar struct {
	lua string
	t   *jtype
	// For a Map.Entry loop variable: the Lua names of key and value.
	entryKey, entryVal string
}

func (t *translator) pushFunc(ret *jtype) *funcCtx {
	fn := &funcCtx{
		parent: t.fn,
		scope:  &scope{vars: map[string]*localVar{}},
		ret:    ret,
		temps:  map[string]bool{},
		upvals: map[string]bool{"q": true, "J": true},
	}
	if t.fn != nil {
		fn.inner = t.fn.inner
		fn.static = t.fn.static
	}
	t.fn = fn
	return fn
}

// popFunc ends a function and checks its upvalue count.
func (t *translator) popFunc() {
	fn := t.fn
	if len(fn.upvals) > maxUpvalues && !t.constTable {
		t.needTable = true
	}
	t.fn = fn.parent
	if t.fn != nil {
		for u := range fn.upvals {
			t.fn.upvals[u] = true
		}
	}
}

func (t *translator) pushScope() {
	t.fn.scope = &scope{parent: t.fn.scope, vars: map[string]*localVar{}}
}
func (t *translator) popScope() { t.fn.scope = t.fn.scope.parent }

// declare adds a local variable and returns its Lua name.
func (t *translator) declare(name string, typ *jtype) *localVar {
	lua := luaName(name)
	if t.topNames[lua] || lua == "K" {
		lua += "_"
	}
	v := &localVar{lua: lua, t: typ}
	t.fn.scope.vars[name] = v
	t.fn.temps[lua] = true
	return v
}

func (t *translator) lookupLocal(name string) *localVar {
	if t.fn == nil {
		return nil
	}
	for s := t.fn.scope; s != nil; s = s.parent {
		if v, ok := s.vars[name]; ok {
			return v
		}
	}
	// Locals of enclosing functions (lambdas, inline functions).
	for f := t.fn.parent; f != nil; f = f.parent {
		for s := f.scope; s != nil; s = s.parent {
			if v, ok := s.vars[name]; ok {
				return v
			}
		}
	}
	return nil
}

// temp returns a fresh local name based on base.
func (t *translator) temp(base string) string {
	for i := 1; ; i++ {
		name := base
		if i > 1 {
			name = base + strconv.Itoa(i)
		}
		if t.idents[name] || t.topNames[name] || luaGlobals[name] || luaKeywords[name] {
			continue
		}
		used := false
		for f := t.fn; f != nil; f = f.parent {
			if f.temps[name] {
				used = true
				break
			}
		}
		if used {
			continue
		}
		t.fn.temps[name] = true
		return name
	}
}

// upval notes a top-level name used by the current function.
func (t *translator) upval(name string) {
	if t.fn != nil {
		t.fn.upvals[name] = true
	}
}

// hoist adds a statement to write before the current one. It fails where
// hoisting would change the evaluation order (conditions of loops, the
// right side of and/or, ternary branches).
func (t *translator) hoist(code string) bool {
	if t.fn == nil || t.fn.noHoist > 0 {
		return false
	}
	t.fn.pre = append(t.fn.pre, code)
	return true
}

// flushPre writes the hoisted statements.
func (t *translator) flushPre() {
	if t.fn == nil {
		return
	}
	for _, p := range t.fn.pre {
		t.w.line(p)
	}
	t.fn.pre = nil
}

// noHoisting runs f where expressions must not hoist statements.
func (t *translator) noHoisting(f func() lx) lx {
	if t.fn == nil {
		return f()
	}
	t.fn.noHoist++
	defer func() { t.fn.noHoist-- }()
	return f()
}

// --- The quest file ---

// callbacks maps Mobius callback methods to the Lua handler setters.
var callbacks = map[string]string{
	"onTalk":              "on_talk",
	"onFirstTalk":         "on_first_talk",
	"onEvent":             "on_event",
	"onKill":              "on_kill",
	"onAttack":            "on_attack",
	"onSpawn":             "on_spawn",
	"onSkillSee":          "on_skill_see",
	"onSpellFinished":     "on_spell_finished",
	"onAggroRangeEnter":   "on_aggro_range_enter",
	"onEnterZone":         "on_enter_zone",
	"onExitZone":          "on_exit_zone",
	"onItemTalk":          "on_item_talk",
	"onDeath":             "on_death",
	"onEnterWorld":        "on_enter_world",
	"actionForEachPlayer": "on_action_for_each_player",
	"onFactionCall":       "on_faction_call",
	"onCreatureSee":       "on_creature_see",
	"onMoveFinished":      "on_move_finished",
	"onNpcHate":           "on_npc_hate",
	"onCanSeeMe":          "on_can_see_me",
	"checkPartyMember":    "on_check_party_member",
	"onTeleport":          "on_teleport",
}

// quest translates a Quest or Script subclass.
func (t *translator) quest() string {
	cls := t.cls
	var ctor *methodDecl
	var inits []member // fields and initializer blocks, in order
	for _, m := range cls.members {
		switch v := m.(type) {
		case *methodDecl:
			if v.ctor {
				if ctor == nil {
					ctor = v
				} else {
					t.todo(v.line, "second constructor", "second constructor")
				}
				continue
			}
			if isScriptMain(v) {
				// main(String[]) of the AI scripts only makes the script; the
				// loader does that.
				continue
			}
			t.addMethod(t.methods, v, true)
		case *fieldDecl, *initBlock:
			inits = append(inits, m)
		case *classDecl:
			t.addInner(v)
		}
	}
	t.findCalledCallbacks()
	t.collectFields(inits)

	// Constructor: super(id, description) and the rest.
	var ctorBody []stmt
	if ctor != nil && ctor.body != nil {
		ctorBody = ctor.body.stmts
		if len(ctorBody) > 0 {
			if es, ok := ctorBody[0].(*exprStmt); ok {
				if c, ok := es.x.(*callExpr); ok && c.x == nil && c.name == "super" {
					t.superArgs(c)
					ctorBody = ctorBody[1:]
				}
			}
		}
	}
	if t.extends == "Script" {
		t.questID, t.questDesc = "-1", `""`
	}
	if t.questID == "" {
		t.questID = "-1"
		t.questDesc = `""`
		if t.extends != "Quest" && t.extends != "Script" {
			t.todo(cls.line, "class extends "+t.extends, "the class extends "+t.extends)
		}
	}

	w := t.w
	origin := t.origin
	if origin == "" {
		origin = "quests/" + cls.name
	}
	w.linef("-- Translated by questport from %s (Mobius CT_0 Interlude); owned by L2Go from here on.", origin)
	w.comments(cls.comments)
	w.line(`local J = require("lib.java")`)
	w.line("")
	w.linef("local q = quest.new(%s, %s, %s)", t.questID, luaQuote(cls.name), t.questDesc)
	requiresAt := len(w.lines)
	t.topLocals = 2
	t.topNames["q"], t.topNames["J"] = true, true
	if t.constTable {
		w.line("local K = {}")
		t.topNames["K"] = true
		t.topLocals++
	}

	t.emitFields(inits, ctorBody)

	// on_start and handlers.
	t.emitHandlers()

	if len(t.features) > 0 {
		feats := sortedKeys(t.features)
		quoted := make([]string, len(feats))
		for i, f := range feats {
			quoted[i] = luaQuote(f)
		}
		line := "q:requires(" + strings.Join(quoted, ", ") + ")"
		w.lines = slices.Insert(w.lines, requiresAt, line)
		t.res.requires = feats
		t.useAPI("q:requires")
	}
	if t.topLocals > maxTopLocals && !t.constTable {
		t.needTable = true
	}
	return w.String()
}

// superArgs reads the quest ID and description from super(...).
func (t *translator) superArgs(c *callExpr) {
	if len(c.args) >= 1 {
		if v, ok := t.constValue(c.args[0]); ok {
			t.questID = v
		} else {
			t.questID = "-1"
			t.todo(c.line, "quest id", "quest id is not a constant")
		}
	}
	if len(c.args) >= 2 {
		if lit, ok := unparen(c.args[1]).(*literalExpr); ok && lit.kind == litString {
			t.questDesc = luaQuote(lit.value)
		} else if v, ok := t.constValue(c.args[1]); ok {
			t.questDesc = v
		} else {
			t.questDesc = `""`
		}
	} else {
		t.questDesc = `""`
	}
}

// constValue evaluates a literal or a reference to a constant field.
func (t *translator) constValue(e expr) (string, bool) {
	switch v := unparen(e).(type) {
	case *literalExpr:
		switch v.kind {
		case litInt, litLong, litFloat:
			return v.value, true
		case litString:
			return luaQuote(v.value), true
		}
	case *identExpr:
		if f := t.fields[v.name]; f != nil && f.v != nil && f.v.init != nil && f.final {
			return t.constValue(f.v.init)
		}
	case *unaryExpr:
		if v.op == "-" {
			if s, ok := t.constValue(v.x); ok {
				return "-" + s, true
			}
		}
	}
	return "", false
}

// addMethod registers a method of the quest class (top) or inner class.
func (t *translator) addMethod(into map[string][]*methodInfo, md *methodDecl, top bool) *methodInfo {
	mi := &methodInfo{decl: md, ret: typeFromRef(md.result)}
	for _, p := range md.params {
		mi.params = append(mi.params, typeFromRef(p.typ))
		if p.varargs {
			mi.varargs = true
		}
	}
	if top && !md.mods.has("static") {
		if cb, ok := callbacks[md.name]; ok {
			mi.callback = cb
			if md.name == "checkPartyMember" && len(md.params) > 0 && md.params[0].typ.name == "QuestState" {
				mi.callback = "on_check_party_member_state"
			}
		}
	}
	lua := luaName(md.name)
	if n := len(into[md.name]); n > 0 {
		lua = fmt.Sprintf("%s_%d", lua, n+1)
	}
	if top {
		if _, isField := t.fieldDeclared(md.name); isField {
			lua += "_fn"
		}
	}
	mi.lua = lua
	into[md.name] = append(into[md.name], mi)
	return mi
}

// fieldDeclared reports whether the class declares a field of this name.
func (t *translator) fieldDeclared(name string) (*fieldDecl, bool) {
	for _, m := range t.cls.members {
		if fd, ok := m.(*fieldDecl); ok {
			for _, v := range fd.vars {
				if v.name == name {
					return fd, true
				}
			}
		}
	}
	return nil, false
}

// findCalledCallbacks marks callbacks the script calls itself; they become
// named local functions.
func (t *translator) findCalledCallbacks() {
	inspect(classBodies(t.cls), func(x any) bool {
		c, ok := x.(*callExpr)
		if !ok {
			return true
		}
		if c.x != nil {
			if _, isThis := c.x.(*thisExpr); !isThis {
				return true
			}
		}
		for _, mi := range t.methods[c.name] {
			if mi.callback != "" && len(mi.params) == len(c.args) {
				mi.called = true
			}
		}
		return true
	})
}

// classBodies returns every method and initializer body of the class.
func classBodies(cls *classDecl) []stmt {
	var out []stmt
	for _, m := range cls.members {
		switch v := m.(type) {
		case *methodDecl:
			if v.body != nil {
				out = append(out, v.body)
			}
		case *initBlock:
			out = append(out, v.body)
		case *classDecl:
			out = append(out, classBodies(v)...)
		case *fieldDecl:
			for _, d := range v.vars {
				if d.init != nil {
					out = append(out, &exprStmt{x: d.init})
				}
			}
		}
	}
	return out
}

func (t *translator) addInner(cd *classDecl) {
	ic := &innerClass{decl: cd, lua: luaName(cd.name), fields: map[string]*fieldInfo{}, methods: map[string][]*methodInfo{}}
	if cd.kind != "class" {
		t.todo(cd.line, "inner "+cd.kind, "inner "+cd.kind+" "+cd.name)
	}
	for _, m := range cd.members {
		switch v := m.(type) {
		case *fieldDecl:
			for _, d := range v.vars {
				typ := typeFromRef(v.typ)
				for i := 0; i < d.dims; i++ {
					typ = arrayOf(typ)
				}
				f := &fieldInfo{name: d.name, lua: "self." + luaName(d.name), t: typ, static: v.mods.has("static"), final: v.mods.has("final"), decl: v, v: d}
				if f.static {
					f.lua = ic.lua + "." + luaName(d.name)
				}
				ic.fields[d.name] = f
				ic.fseq = append(ic.fseq, f)
			}
		case *methodDecl:
			if v.ctor {
				mi := &methodInfo{decl: v, lua: "new"}
				for _, p := range v.params {
					mi.params = append(mi.params, typeFromRef(p.typ))
				}
				if n := len(ic.ctors); n > 0 {
					mi.lua = fmt.Sprintf("new%d", n+1)
				}
				ic.ctors = append(ic.ctors, mi)
				continue
			}
			t.addMethod(ic.methods, v, false)
		case *classDecl:
			t.todo(v.line, "nested inner class", "class nested in an inner class")
		case *initBlock:
			t.todo(v.line, "inner class initializer", "initializer block in an inner class")
		}
	}
	t.inner[cd.name] = ic
	t.topNames[ic.lua] = true
}

// collectFields builds the field table and decides which fields are
// initialized early (pure initializers) or late.
func (t *translator) collectFields(inits []member) {
	for _, m := range inits {
		fd, ok := m.(*fieldDecl)
		if !ok {
			continue
		}
		for _, d := range fd.vars {
			typ := typeFromRef(fd.typ)
			for i := 0; i < d.dims; i++ {
				typ = arrayOf(typ)
			}
			f := &fieldInfo{
				name:   d.name,
				lua:    luaName(d.name),
				t:      typ,
				static: fd.mods.has("static"),
				final:  fd.mods.has("final"),
				decl:   fd,
				v:      d,
			}
			if t.inner[f.lua] != nil || f.lua == "K" {
				f.lua += "_"
			}
			if isRef(typ) && (!f.final || !f.static) {
				f.ref = true
			}
			if (isMap(typ) || isSequence(typ) || isSet(typ)) && containsRefType(typ) {
				f.refSeq = true
			}
			f.early = d.init == nil || t.pureInit(d.init)
			if t.constTable && f.early && f.final && f.static {
				f.lua = "K." + f.lua
			} else {
				t.topNames[f.lua] = true
			}
			t.fields[d.name] = f
			t.fieldSeq = append(t.fieldSeq, f)
		}
	}
}

// containsRefType reports whether a collection type holds world objects.
func containsRefType(t *jtype) bool {
	if t == nil {
		return false
	}
	if isRef(t) {
		return true
	}
	if t.kind == tkArray {
		return containsRefType(t.elem)
	}
	for _, a := range t.args {
		if containsRefType(a) {
			return true
		}
	}
	return false
}

// pureInit reports whether a field initializer can run before the methods
// and inner classes exist: literals, arrays, collections, holders and
// constants of earlier fields only.
func (t *translator) pureInit(e expr) bool {
	pure := true
	inspect(e, func(x any) bool {
		if !pure {
			return false
		}
		switch v := x.(type) {
		case *callExpr:
			switch v.name {
			case "getSimpleName", "asList", "of":
			default:
				pure = false
			}
		case *newExpr:
			if t.inner[v.typ.name] != nil || v.anon {
				pure = false
			}
		case *identExpr:
			if f := t.fields[v.name]; f != nil && !f.early {
				pure = false
			}
		case *assignExpr, *lambdaExpr:
			pure = false
		case *unaryExpr:
			if v.op == "++" || v.op == "--" {
				pure = false
			}
		}
		return pure
	})
	return pure
}

// emitFields writes fields, inner classes, methods, initializers and the
// constructor, in an order that keeps Java's semantics: pure fields first,
// then forward declarations, inner classes, methods, then the remaining
// initializers in source order and the constructor body.
func (t *translator) emitFields(inits []member, ctorBody []stmt) {
	w := t.w
	t.pushFunc(tyVoid)
	t.fn.static = true
	// 1. Early fields.
	wroteField := false
	for _, f := range t.fieldSeq {
		if !f.early {
			continue
		}
		if !wroteField {
			w.blank()
		}
		wroteField = true
		t.emitFieldDecl(f, true)
	}
	t.popFunc()

	// 2. Forward declarations.
	var forward []string
	for _, f := range t.fieldSeq {
		if !f.early {
			forward = append(forward, f.lua)
		}
	}
	var methodSeq []*methodInfo
	for _, m := range t.cls.members {
		md, ok := m.(*methodDecl)
		if !ok || md.ctor {
			continue
		}
		for _, mi := range t.methods[md.name] {
			if mi.decl == md && (mi.callback == "" || mi.called) {
				methodSeq = append(methodSeq, mi)
			}
		}
	}
	for _, mi := range methodSeq {
		t.topNames[mi.lua] = true
	}
	forwardMethods := t.needForwardMethods(methodSeq)
	if forwardMethods {
		for _, mi := range methodSeq {
			forward = append(forward, mi.lua)
		}
	}
	if len(forward) > 0 {
		w.blank()
		for i := 0; i < len(forward); i += 8 {
			end := min(i+8, len(forward))
			w.line("local " + strings.Join(forward[i:end], ", "))
		}
	}
	t.topLocals += len(forward)
	if !forwardMethods {
		t.topLocals += len(methodSeq)
	}

	// 3. Inner classes.
	for _, m := range t.cls.members {
		if cd, ok := m.(*classDecl); ok {
			if ic := t.inner[cd.name]; ic != nil {
				w.blank()
				t.emitInner(ic)
				t.topLocals++
			}
		}
	}

	// 4. Methods.
	for _, mi := range methodSeq {
		w.blank()
		t.emitMethod(mi, forwardMethods)
	}

	// 5. Late initializers in source order.
	t.pushFunc(tyVoid)
	t.fn.static = true
	wrote := false
	for _, m := range inits {
		switch v := m.(type) {
		case *fieldDecl:
			for _, d := range v.vars {
				f := t.fields[d.name]
				if f == nil || f.early {
					continue
				}
				if !wrote {
					w.blank()
					wrote = true
				}
				t.emitFieldDecl(f, false)
			}
		case *initBlock:
			if !wrote {
				w.blank()
				wrote = true
			}
			w.comments(v.body.comments)
			t.stmtList(v.body.stmts)
		}
	}
	t.popFunc()

	// 6. Constructor.
	t.emitConstructor(ctorBody)
}

// needForwardMethods reports whether some method is used before its
// definition: by an earlier method, an inner class or a late initializer.
func (t *translator) needForwardMethods(seq []*methodInfo) bool {
	if len(seq) == 0 {
		return false
	}
	pos := map[string]int{}
	for i, mi := range seq {
		pos[mi.decl.name] = i
	}
	calls := func(n any) map[string]bool {
		out := map[string]bool{}
		inspect(n, func(x any) bool {
			if c, ok := x.(*callExpr); ok && (c.x == nil || isThis(c.x)) {
				out[c.name] = true
			}
			if r, ok := x.(*methodRefExpr); ok {
				out[r.name] = true
			}
			return true
		})
		return out
	}
	for i, mi := range seq {
		for name := range calls(mi.decl.body) {
			if j, ok := pos[name]; ok && j >= i {
				return true
			}
		}
	}
	for _, m := range t.cls.members {
		switch v := m.(type) {
		case *classDecl:
			for name := range calls(classBodies(v)) {
				if _, ok := pos[name]; ok {
					return true
				}
			}
		case *initBlock:
			for name := range calls(v.body) {
				if _, ok := pos[name]; ok {
					return true
				}
			}
		case *fieldDecl:
			for _, d := range v.vars {
				if d.init == nil {
					continue
				}
				for name := range calls(d.init) {
					if _, ok := pos[name]; ok {
						return true
					}
				}
			}
		}
	}
	return false
}

func isThis(e expr) bool {
	_, ok := unparen(e).(*thisExpr)
	return ok
}

// emitFieldDecl writes one field. decl: write "local" (early fields);
// otherwise assign the forward-declared name.
func (t *translator) emitFieldDecl(f *fieldInfo, decl bool) {
	w := t.w
	w.comments(f.decl.comments)
	f.decl.comments = nil
	var val lx
	if f.v.init != nil {
		val = t.initValue(f.v.init, f.t)
	} else {
		val = defaultValue(f.t)
	}
	if f.ref && f.v.init != nil && !val.nilLit {
		val = t.refStore(val)
	}
	if f.refSeq {
		t.review(f.v.line, "world objects kept in a collection", fmt.Sprintf("field %s keeps world objects across events; store object IDs", f.name))
	}
	t.flushPre()
	t.flushTodoComments()
	switch {
	case strings.HasPrefix(f.lua, "K."):
		w.linef("%s = %s", f.lua, val.code)
	case decl:
		if val.nilLit && f.v.init == nil {
			w.linef("local %s", f.lua)
		} else {
			w.linef("local %s = %s", f.lua, val.code)
		}
		t.topLocals++
	default:
		w.linef("%s = %s", f.lua, val.code)
	}
	if len(f.decl.vars) == 1 || f.v == f.decl.vars[len(f.decl.vars)-1] {
		w.trail(f.decl.trail)
	}
	if v, ok := t.constValue(f.v.init); ok && f.final && f.v.init != nil {
		k := atom(v, f.t)
		f.konst = &k
	}
}

// initValue translates a field or variable initializer of type typ.
func (t *translator) initValue(e expr, typ *jtype) lx {
	if ai, ok := e.(*arrayInitExpr); ok {
		return t.arrayInit(ai, elemType(typ))
	}
	v := t.expr(e)
	return t.coerce(v, typ)
}

// defaultValue is the Java default of a field of type t.
func defaultValue(t *jtype) lx {
	switch {
	case isBool(t) && t.kind == tkPrim:
		return lx{code: "false", prec: precAtom, t: tyBool, pure: true}
	case isNumeric(t) && t.kind == tkPrim:
		return lx{code: "0", prec: precAtom, t: t, pure: true, safe: true, intLit: true}
	}
	return lx{code: "nil", prec: precAtom, t: tyNull, pure: true, nilLit: true}
}

// emitConstructor writes the constructor body: registrations at the top
// level, world side effects (spawns, timers, global variables) in
// q:on_start.
func (t *translator) emitConstructor(body []stmt) {
	if len(body) == 0 {
		return
	}
	w := t.w
	main := w
	start := &luaWriter{ind: 1}
	t.pushFunc(tyVoid)
	t.fn.ctor = true
	inStart := false
	w.blank()
	for _, s := range body {
		sub := &luaWriter{ind: main.ind}
		t.w = sub
		t.fn.world = false
		t.stmt(s)
		isWorld := t.fn.world
		t.w = main
		if isWorld && !inStart {
			inStart = true
		}
		if inStart && !isRegistration(s) {
			for _, l := range sub.lines {
				if l == "" {
					start.lines = append(start.lines, "")
				} else {
					start.lines = append(start.lines, "  "+l)
				}
			}
			continue
		}
		main.lines = append(main.lines, sub.lines...)
	}
	t.popFunc()
	if len(start.lines) > 0 {
		w.blank()
		w.line("q:on_start(function()")
		w.lines = append(w.lines, start.lines...)
		w.line("end)")
		t.useAPI("q:on_start")
	}
}

// isRegistration reports whether s is a plain registration call.
func isRegistration(s stmt) bool {
	es, ok := s.(*exprStmt)
	if !ok {
		return false
	}
	c, ok := es.x.(*callExpr)
	if !ok || (c.x != nil && !isThis(c.x)) {
		return false
	}
	return strings.HasPrefix(c.name, "add") && strings.HasSuffix(c.name, "Id") ||
		c.name == "addStartNpc" || c.name == "registerQuestItems" || c.name == "setOnEnterWorld"
}

// emitInner writes an inner class as a Lua table with a constructor.
func (t *translator) emitInner(ic *innerClass) {
	w := t.w
	w.comments(ic.decl.comments)
	w.linef("local %s = {}", ic.lua)
	w.linef("%s.__index = %s", ic.lua, ic.lua)
	// Static fields.
	for _, f := range ic.fseq {
		if !f.static {
			continue
		}
		t.pushFunc(tyVoid)
		val := defaultValue(f.t)
		if f.v.init != nil {
			val = t.initValue(f.v.init, f.t)
		}
		t.flushPre()
		t.flushTodoComments()
		w.linef("%s = %s", f.lua, val.code)
		t.popFunc()
	}
	ctors := ic.ctors
	if len(ctors) == 0 {
		ctors = []*methodInfo{{lua: "new"}}
	}
	for _, mi := range ctors {
		w.blank()
		t.pushFunc(tyVoid)
		t.fn.inner = ic
		t.fn.static = false
		var params []string
		if mi.decl != nil {
			w.comments(mi.decl.comments)
			for _, p := range mi.decl.params {
				params = append(params, t.declare(p.name, typeFromRef(p.typ)).lua)
			}
		}
		w.linef("function %s.%s(%s)", ic.lua, mi.lua, strings.Join(params, ", "))
		w.ind++
		w.linef("local self = setmetatable({}, %s)", ic.lua)
		t.fn.temps["self"] = true
		for _, f := range ic.fseq {
			if f.static {
				continue
			}
			if f.v.init != nil {
				val := t.initValue(f.v.init, f.t)
				t.flushPre()
				t.flushTodoComments()
				w.linef("%s = %s", f.lua, val.code)
			} else if d := defaultValue(f.t); !d.nilLit {
				w.linef("%s = %s", f.lua, d.code)
			}
		}
		if mi.decl != nil && mi.decl.body != nil {
			t.stmtList(mi.decl.body.stmts)
		}
		w.line("return self")
		w.ind--
		w.line("end")
		t.popFunc()
	}
	for _, m := range ic.decl.members {
		md, ok := m.(*methodDecl)
		if !ok || md.ctor {
			continue
		}
		for _, mi := range ic.methods[md.name] {
			if mi.decl != md {
				continue
			}
			w.blank()
			t.pushFunc(mi.ret)
			t.fn.inner = ic
			t.fn.static = md.mods.has("static")
			w.comments(md.comments)
			var params []string
			for _, p := range md.params {
				params = append(params, t.declare(p.name, typeFromRef(p.typ)).lua)
			}
			sep := ":"
			if t.fn.static {
				sep = "."
				t.fn.inner = nil
			}
			w.linef("function %s%s%s(%s)", ic.lua, sep, mi.lua, strings.Join(params, ", "))
			w.ind++
			if md.body != nil {
				t.stmtList(md.body.stmts)
			}
			w.ind--
			w.line("end")
			t.popFunc()
		}
	}
}

// emitMethod writes a helper method (or a callback the script calls
// itself) as a local function.
func (t *translator) emitMethod(mi *methodInfo, forward bool) {
	w := t.w
	md := mi.decl
	w.comments(md.comments)
	t.pushFunc(mi.ret)
	t.fn.static = md.mods.has("static")
	var params []string
	for _, p := range md.params {
		params = append(params, t.declare(p.name, typeFromRef(p.typ)).lua)
	}
	if mi.varargs && len(params) > 0 {
		// Java varargs arrive as one table.
		t.review(md.line, "varargs method", "varargs method "+md.name+": callers pass a table")
	}
	if md.body == nil {
		t.todo(md.line, "abstract method", "abstract method "+md.name)
	}
	if forward {
		w.linef("function %s(%s)", mi.lua, strings.Join(params, ", "))
	} else {
		w.linef("local function %s(%s)", mi.lua, strings.Join(params, ", "))
	}
	w.ind++
	if md.body != nil {
		t.stmtList(md.body.stmts)
	}
	w.ind--
	w.line("end")
	t.popFunc()
}

// emitHandlers writes the q:on_* registrations.
func (t *translator) emitHandlers() {
	w := t.w
	seen := map[string]*methodDecl{}
	for _, m := range t.cls.members {
		md, ok := m.(*methodDecl)
		if !ok || md.ctor {
			continue
		}
		var mi *methodInfo
		for _, x := range t.methods[md.name] {
			if x.decl == md {
				mi = x
			}
		}
		if mi == nil || mi.callback == "" {
			continue
		}
		if prev := seen[mi.callback]; prev != nil {
			t.todo(md.line, "second "+md.name+" overload", fmt.Sprintf("second %s overload (also line %d)", md.name, prev.line))
			continue
		}
		seen[mi.callback] = md
		t.useAPI("q:" + mi.callback)
		w.blank()
		if mi.called {
			w.linef("q:%s(%s)", mi.callback, mi.lua)
			continue
		}
		w.comments(md.comments)
		t.pushFunc(mi.ret)
		var params []string
		for _, p := range md.params {
			params = append(params, t.declare(p.name, typeFromRef(p.typ)).lua)
		}
		w.linef("q:%s(function(%s)", mi.callback, strings.Join(params, ", "))
		w.ind++
		if md.body != nil {
			t.stmtList(md.body.stmts)
		}
		w.ind--
		w.line("end)")
		t.popFunc()
	}
}

// --- Sagas ---

// saga translates a subclass of AbstractSagaQuest into saga.define data.
func (t *translator) saga() string {
	t.res.saga = true
	w := t.w
	var ctor *methodDecl
	for _, m := range t.cls.members {
		switch v := m.(type) {
		case *methodDecl:
			if v.ctor {
				ctor = v
			} else {
				t.todo(v.line, "saga method", "saga subclass method "+v.name)
			}
		case *fieldDecl, *initBlock, *classDecl:
			t.todo(m.memberLine(), "saga member", "saga subclass member")
		}
	}
	w.linef("-- Translated by questport from quests/%s (Mobius CT_0 Interlude); owned by L2Go from here on.", t.cls.name)
	w.line("-- A saga: the quest logic is lib/saga.lua (Mobius AbstractSagaQuest); this file holds its data.")
	w.line(`local saga = require("lib.saga")`)
	w.line("")
	t.pushFunc(tyVoid)
	t.fn.static = true
	var entries []string
	if ctor != nil && ctor.body != nil {
		for _, s := range ctor.body.stmts {
			es, ok := s.(*exprStmt)
			if !ok {
				t.todo(s.stmtLine(), "saga statement", "saga constructor statement")
				continue
			}
			switch x := es.x.(type) {
			case *callExpr:
				switch {
				case x.x == nil && x.name == "super":
					t.superArgs(x)
				case x.x == nil && x.name == "registerNPCs":
				default:
					entries = append(entries, "  _todo = "+t.todo(x.line, "saga call "+x.name, "saga constructor call "+x.name).code+",")
				}
			case *assignExpr:
				name := ""
				switch lhs := unparen(x.x).(type) {
				case *identExpr:
					name = lhs.name
				case *fieldExpr:
					name = lhs.name
				}
				if name == "" || x.op != "=" {
					t.todo(x.line, "saga statement", "saga constructor statement")
					continue
				}
				key := snake(strings.TrimPrefix(name, "_"))
				var val lx
				if na, ok := unparen(x.y).(*newArrayExpr); ok && na.init != nil {
					val = t.arrayInit(na.init, typeFromRef(na.elem))
				} else {
					val = t.expr(x.y)
				}
				entries = append(entries, fmt.Sprintf("  %s = %s,", key, val.code))
			default:
				t.todo(s.stmtLine(), "saga statement", "saga constructor statement")
			}
		}
	}
	t.popFunc()
	w.line("saga.define({")
	w.linef("  id = %s,", t.questID)
	w.linef("  name = %s,", luaQuote(t.cls.name))
	w.linef("  description = %s,", t.questDesc)
	w.lines = append(w.lines, entries...)
	w.line("})")
	t.useAPI("saga.define")
	return w.String()
}
