package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// expr translates an expression.
func (t *translator) expr(e expr) lx {
	switch v := e.(type) {
	case *parenExpr:
		return t.expr(v.x)
	case *literalExpr:
		return literal(v)
	case *identExpr:
		return t.ident(v)
	case *fieldExpr:
		return t.fieldAccess(v)
	case *callExpr:
		return t.call(v, false)
	case *indexExpr:
		arr := t.expr(v.x)
		idx := t.indexPlusOne(v.index)
		return lx{code: asAtom(arr) + "[" + idx.code + "]", prec: precAtom, t: elemType(arr.t), pure: arr.pure && idx.pure, todo: arr.todo || idx.todo}
	case *newExpr:
		return t.newObject(v)
	case *newArrayExpr:
		return t.newArray(v)
	case *arrayInitExpr:
		return t.arrayInit(v, tyUnknown)
	case *unaryExpr:
		return t.unary(v)
	case *binaryExpr:
		return t.binary(v)
	case *assignExpr:
		return t.assignValue(v)
	case *condExpr:
		return t.ternary(v)
	case *castExpr:
		return t.cast(v)
	case *instanceOfExpr:
		x := t.expr(v.x)
		t.useAPI("J.instance_of")
		return lx{code: fmt.Sprintf("J.instance_of(%s, %s)", x.code, luaQuote(v.typ.name)), prec: precAtom, t: tyBool, call: true}
	case *lambdaExpr:
		return t.todo(v.line, "lambda", "lambda expression")
	case *methodRefExpr:
		return t.todo(v.line, "method reference", "method reference ::"+v.name)
	case *thisExpr:
		if t.fn != nil && t.fn.inner != nil && !t.fn.static {
			return lx{code: "self", prec: precAtom, t: classType(t.fn.inner.decl.name), pure: true, safe: true}
		}
		return t.todo(v.line, "this as a value", "the quest object used as a value")
	case *superExpr:
		return t.todo(v.line, "super as a value", "super used as a value")
	case *classLitExpr:
		return t.todo(v.line, "class literal", "class literal "+v.typ.name+".class")
	}
	return t.todo(e.exprLine(), "expression", fmt.Sprintf("unsupported expression %T", e))
}

func literal(v *literalExpr) lx {
	switch v.kind {
	case litInt, litLong:
		n, _ := strconv.ParseInt(v.value, 10, 64)
		typ := tyInt
		if v.kind == litLong {
			typ = tyLong
		}
		return lx{code: v.value, prec: precAtom, t: typ, safe: true, pure: true, intLit: true, intVal: n}
	case litFloat:
		return lx{code: v.value, prec: precAtom, t: tyDouble, safe: true, pure: true}
	case litString:
		return lx{code: luaQuote(v.value), prec: precAtom, t: tyString, safe: true, pure: true, strLit: true}
	case litChar:
		return lx{code: luaQuote(v.value), prec: precAtom, t: tyChar, safe: true, pure: true, strLit: true}
	case litBool:
		return lx{code: v.value, prec: precAtom, t: tyBool, safe: v.value == "true", pure: true}
	}
	return lx{code: "nil", prec: precAtom, t: tyNull, pure: true, nilLit: true}
}

// isTypeName reports whether an unresolved name looks like a class name.
func isTypeName(name string) bool {
	if name == "" || !unicode.IsUpper(rune(name[0])) {
		return false
	}
	for _, r := range name {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func (t *translator) ident(v *identExpr) lx {
	if l := t.lookupLocal(v.name); l != nil {
		return lx{code: l.lua, prec: precAtom, t: l.t, pure: true, safe: isNumeric(l.t) && l.t.kind == tkPrim, entryOf: l.entryKey, base: l.entryVal}
	}
	if t.fn != nil && t.fn.inner != nil {
		if f := t.fn.inner.fields[v.name]; f != nil {
			return lx{code: f.lua, prec: precAtom, t: f.t, pure: true}
		}
	}
	if f := t.fields[v.name]; f != nil {
		return t.fieldRead(f)
	}
	if ic := t.inner[v.name]; ic != nil {
		return lx{code: ic.lua, prec: precAtom, t: tyUnknown, isType: v.name}
	}
	if f, ok := inheritedFields[v.name]; ok {
		t.useAPI(f.lua)
		return lx{code: f.lua, prec: precAtom, t: f.t, pure: true, safe: true}
	}
	if isTypeName(v.name) {
		return lx{code: v.name, prec: precAtom, t: tyUnknown, isType: v.name}
	}
	return t.todo(v.line, "unknown name", "unknown name "+v.name)
}

// fieldRead reads a field of the quest class.
func (t *translator) fieldRead(f *fieldInfo) lx {
	if strings.HasPrefix(f.lua, "K.") {
		t.upval("K")
	} else {
		t.upval(f.lua)
	}
	r := lx{code: f.lua, prec: precAtom, t: f.t, pure: true}
	if (isNumeric(f.t) && f.t.kind == tkPrim) || (f.final && f.v != nil && f.v.init != nil && !isNullLit(f.v.init)) {
		r.safe = true
	}
	if f.konst != nil {
		r.intLit = f.konst.intLit
		r.intVal = f.konst.intVal
	}
	if f.ref {
		t.useAPI("J.obj")
		r.code = "J.obj(" + f.lua + ")"
		r.call = true
		r.pure = true
		r.refID = f.lua
	}
	return r
}

func isNullLit(e expr) bool {
	l, ok := unparen(e).(*literalExpr)
	return ok && l.kind == litNull
}

// staticField translates Class.NAME.
func (t *translator) staticField(class, name string, line int) lx {
	if ic := t.inner[class]; ic != nil {
		if f := ic.fields[name]; f != nil && f.static {
			return lx{code: f.lua, prec: precAtom, t: f.t, pure: true}
		}
	}
	switch class {
	case "Inventory":
		if strings.HasPrefix(name, "PAPERDOLL_") {
			return lx{code: "Paperdoll." + strings.TrimPrefix(name, "PAPERDOLL_"), prec: precAtom, t: tyInt, pure: true, safe: true}
		}
	case "PlayerVariables":
		// Mobius names its variable keys after the constants.
		return lx{code: luaQuote(name), prec: precAtom, t: tyString, pure: true, safe: true, strLit: true}
	case "SevenSigns":
		// CABAL_DAWN, SEAL_AVARICE...: the numbers of seven_signs.
		t.requires("seven_signs")
		t.useAPI("seven_signs.*")
		return lx{code: "seven_signs." + name, prec: precAtom, t: tyInt, pure: true, safe: true}
	case "Integer":
		switch name {
		case "MAX_VALUE":
			return lx{code: "2147483647", prec: precAtom, t: tyInt, pure: true, safe: true}
		case "MIN_VALUE":
			return lx{code: "-2147483648", prec: precUnary, t: tyInt, pure: true, safe: true}
		}
	case "Long":
		switch name {
		case "MAX_VALUE":
			return lx{code: "9223372036854775807", prec: precAtom, t: tyLong, pure: true, safe: true}
		case "MIN_VALUE":
			return lx{code: "-9223372036854775808", prec: precUnary, t: tyLong, pure: true, safe: true}
		}
	}
	if name == "STATIC_PACKET" && packetClasses[class] {
		// A packet without arguments, consumed by sendPacket.
		return t.packetValue(line, class, nil)
	}
	if strings.HasSuffix(class, "Config") {
		typ, ok := configFields[name]
		if !ok {
			return t.todo(line, "config "+class+"."+name, "config value "+class+"."+name)
		}
		t.useAPI("Config." + name)
		return lx{code: "Config." + name, prec: precAtom, t: typ, pure: true, safe: isNumeric(typ)}
	}
	if table, ok := enumTables[class]; ok {
		t.useAPI(table + ".*")
		return lx{code: table + "." + name, prec: precAtom, t: enumValueType(class), pure: true, safe: true}
	}
	return t.todo(line, "static field "+class+"."+name, "static field "+class+"."+name)
}

// inheritedFields are protected fields of Mobius's Quest class.
var inheritedFields = map[string]struct {
	lua string
	t   *jtype
}{
	"DF_REWARD_35": {"quest.DF_REWARD_35", classType("Map", tyInt, tyInt)},
	"DF_REWARD_37": {"quest.DF_REWARD_37", classType("Map", tyInt, tyInt)},
	"DF_REWARD_39": {"quest.DF_REWARD_39", classType("Map", tyInt, tyInt)},
}

// configFields are the Mobius config values quests read, with types.
var configFields = map[string]*jtype{
	"ALT_PARTY_RANGE":                      tyInt,
	"DISABLE_TUTORIAL":                     tyBool,
	"ALT_VILLAGES_REPEATABLE_QUEST_REWARD": tyBool,
	"RAID_MIN_RESPAWN_MULTIPLIER":          tyDouble,
	"RAID_MAX_RESPAWN_MULTIPLIER":          tyDouble,
	"RATE_QUEST_DROP":                      tyDouble,
	"RATE_QUEST_REWARD":                    tyDouble,
	"RATE_QUEST_REWARD_XP":                 tyDouble,
	"RATE_QUEST_REWARD_SP":                 tyDouble,
	"RATE_QUEST_REWARD_ADENA":              tyDouble,
	"ALT_GAME_NEW_CHAR_ALWAYS_IS_NEWBIE":   tyBool,
	"ALT_PARTY_RANGE2":                     tyInt,
}

func (t *translator) fieldAccess(v *fieldExpr) lx {
	x := unparen(v.x)
	if isThis(x) {
		if t.fn != nil && t.fn.inner != nil {
			if f := t.fn.inner.fields[v.name]; f != nil {
				return lx{code: f.lua, prec: precAtom, t: f.t, pure: true}
			}
		}
		if f := t.fields[v.name]; f != nil {
			return t.fieldRead(f)
		}
		return t.todo(v.line, "inherited field", "inherited field this."+v.name)
	}
	if id, ok := x.(*identExpr); ok && t.lookupLocal(id.name) == nil && t.fields[id.name] == nil && !t.innerField(id.name) {
		if isTypeName(id.name) || t.inner[id.name] != nil {
			return t.staticField(id.name, v.name, v.line)
		}
	}
	recv := t.expr(x)
	if recv.isType != "" {
		return t.staticField(recv.isType, v.name, v.line)
	}
	if isArray(recv.t) && v.name == "length" {
		return lx{code: "#" + asAtom(recv), prec: precUnary, t: tyInt, pure: recv.pure, safe: true}
	}
	if recv.t != nil {
		if ic := t.inner[recv.t.name]; ic != nil {
			if f := ic.fields[v.name]; f != nil {
				return lx{code: asAtom(recv) + "." + luaName(v.name), prec: precAtom, t: f.t, pure: recv.pure}
			}
		}
		if hf, ok := holderFields[recv.t.name+"."+v.name]; ok {
			return lx{code: asAtom(recv) + "." + hf.lua, prec: precAtom, t: hf.t, pure: recv.pure}
		}
	}
	if recv.todo {
		return recv
	}
	return t.todo(v.line, "field "+recv.t.String()+"."+v.name, "field ."+v.name+" of "+recv.t.String())
}

// innerField reports whether name is a field of the inner class whose
// method is being translated.
func (t *translator) innerField(name string) bool {
	return t.fn != nil && t.fn.inner != nil && t.fn.inner.fields[name] != nil
}

// indexPlusOne translates a 0-based Java index into the 1-based Lua one,
// folding constants: A[0] -> A[1], A[i - 1] -> A[i], A[i] -> A[i + 1].
func (t *translator) indexPlusOne(e expr) lx {
	e = unparen(e)
	if b, ok := e.(*binaryExpr); ok && (b.op == "+" || b.op == "-") {
		if lit, ok := unparen(b.y).(*literalExpr); ok && (lit.kind == litInt || lit.kind == litLong) {
			n, _ := strconv.ParseInt(lit.value, 10, 64)
			x := t.expr(b.x)
			if isIntegral(x.t) || isUnknown(x.t) {
				if b.op == "-" {
					n = -n
				}
				n++
				switch {
				case n == 0:
					return x
				case n > 0:
					return lx{code: paren(x, precAdd) + " + " + strconv.FormatInt(n, 10), prec: precAdd, t: tyInt, pure: x.pure}
				default:
					return lx{code: paren(x, precAdd) + " - " + strconv.FormatInt(-n, 10), prec: precAdd, t: tyInt, pure: x.pure}
				}
			}
		}
	}
	x := t.expr(e)
	if x.intLit {
		return lx{code: strconv.FormatInt(x.intVal+1, 10), prec: precAtom, t: tyInt, pure: true, intLit: true, intVal: x.intVal + 1}
	}
	return lx{code: paren(x, precAdd) + " + 1", prec: precAdd, t: tyInt, pure: x.pure, todo: x.todo}
}

// newArray translates new T[n] and new T[]{...}.
func (t *translator) newArray(v *newArrayExpr) lx {
	elem := typeFromRef(v.elem)
	if v.init != nil {
		inner := elem
		for i := 0; i < v.extra-1; i++ {
			inner = arrayOf(inner)
		}
		return t.arrayInit(v.init, inner)
	}
	full := elem
	for i := 0; i < len(v.dims)+v.extra; i++ {
		full = arrayOf(full)
	}
	def := defaultValue(elem)
	if v.extra > 0 || def.nilLit {
		if len(v.dims) == 1 && v.extra == 0 {
			// Objects start as null: a Lua table with holes has no length.
			t.review(v.line, "array of objects", "new "+elem.name+"[n]: elements start as nil, so #array is 0 until filled")
			return lx{code: "{}", prec: precAtom, t: full, safe: true}
		}
		return t.todo(v.line, "array of arrays", "array creation without all sizes")
	}
	var dims []string
	for _, d := range v.dims {
		dims = append(dims, t.expr(d).code)
	}
	t.useAPI("J.array")
	return lx{code: fmt.Sprintf("J.array(%s, %s)", def.code, strings.Join(dims, ", ")), prec: precAtom, t: full, call: true, safe: true}
}

// arrayInit translates {a, b, c} with elements of type elem.
func (t *translator) arrayInit(ai *arrayInitExpr, elem *jtype) lx {
	parts := make([]string, len(ai.elems))
	total := 0
	multiline := false
	for i, e := range ai.elems {
		var v lx
		if sub, ok := e.(*arrayInitExpr); ok {
			v = t.arrayInit(sub, elemType(elem))
		} else {
			v = t.expr(e)
		}
		parts[i] = v.code
		total += len(v.code) + 2
		if strings.Contains(v.code, "\n") {
			multiline = true
		}
	}
	typ := arrayOf(elem)
	if total <= 100 && !multiline {
		return lx{code: "{" + strings.Join(parts, ", ") + "}", prec: precAtom, t: typ, safe: true}
	}
	// One element per line; the writer indents the first line only, so
	// the following lines carry their own indentation.
	ind := strings.Repeat("  ", t.w.ind+1)
	var b strings.Builder
	b.WriteString("{\n")
	for _, p := range parts {
		p = strings.ReplaceAll(p, "\n", "\n  ")
		b.WriteString(ind + p + ",\n")
	}
	b.WriteString(strings.Repeat("  ", t.w.ind) + "}")
	return lx{code: b.String(), prec: precAtom, t: typ, safe: true}
}

func (t *translator) unary(v *unaryExpr) lx {
	switch v.op {
	case "+":
		return t.expr(v.x)
	case "-":
		x := t.expr(v.x)
		if x.intLit {
			return lx{code: "-" + x.code, prec: precUnary, t: x.t, safe: true, pure: true, intLit: true, intVal: -x.intVal}
		}
		code := paren(x, precUnary)
		if strings.HasPrefix(code, "-") {
			code = "(" + code + ")"
		}
		return lx{code: "-" + code, prec: precUnary, t: x.t, safe: true, pure: x.pure}
	case "!":
		x := t.expr(v.x)
		return lx{code: "not " + paren(x, precUnary), prec: precUnary, t: tyBool, pure: x.pure, todo: x.todo}
	case "~":
		x := t.expr(v.x)
		t.useAPI("J.bnot")
		return lx{code: "J.bnot(" + x.code + ")", prec: precAtom, t: x.t, call: true, safe: true}
	case "++", "--":
		return t.incDecValue(v)
	}
	return t.todo(v.line, "operator "+v.op, "operator "+v.op)
}

// lazy translates an expression that Java may not evaluate (right side of
// && and ||, ternary branches): it must not hoist statements.
func (t *translator) lazy(e expr) lx {
	return t.noHoisting(func() lx { return t.expr(e) })
}

func (t *translator) binary(v *binaryExpr) lx {
	switch v.op {
	case "&&", "||":
		a := t.expr(v.x)
		b := t.lazy(v.y)
		op, prec := "and", precAnd
		if v.op == "||" {
			op, prec = "or", precOr
		}
		return lx{code: paren(a, prec) + " " + op + " " + paren(b, prec+1), prec: prec, t: tyBool, pure: a.pure && b.pure, todo: a.todo || b.todo}
	}
	a := t.expr(v.x)
	b := t.expr(v.y)
	return t.binop(v.op, a, b, v.line)
}

// binop applies a Java binary operator (not && or ||) to translated
// operands.
func (t *translator) binop(op string, a, b lx, line int) lx {
	pure := a.pure && b.pure
	todo := a.todo || b.todo
	switch op {
	case "==", "!=":
		// A field keeping a world object is null exactly when its stored
		// object ID is.
		if b.nilLit && a.refID != "" {
			a = atom(a.refID, a.t)
		}
		if a.nilLit && b.refID != "" {
			b = atom(b.refID, b.t)
		}
		lop := "=="
		if op == "!=" {
			lop = "~="
		}
		return lx{code: paren(a, precCmp+1) + " " + lop + " " + paren(b, precCmp+1), prec: precCmp, t: tyBool, pure: pure, todo: todo}
	case "<", ">", "<=", ">=":
		return lx{code: paren(a, precCmp+1) + " " + op + " " + paren(b, precCmp+1), prec: precCmp, t: tyBool, pure: pure, todo: todo}
	case "+":
		if isString(a.t) || isString(b.t) || (isChar(a.t) && isChar(b.t) && false) {
			return t.concat(a, b)
		}
		return arith("+", precAdd, a, b)
	case "-":
		return arith("-", precAdd, a, b)
	case "*":
		return arith("*", precMul, a, b)
	case "/":
		if isIntegral(a.t) && isIntegral(b.t) || (isIntegral(a.t) && isUnknown(b.t)) || (isUnknown(a.t) && isIntegral(b.t)) {
			t.useAPI("J.idiv")
			return lx{code: "J.idiv(" + a.code + ", " + b.code + ")", prec: precAtom, t: binaryNumeric(a.t, b.t), call: true, safe: true, pure: pure, todo: todo}
		}
		return arith("/", precMul, a, b)
	case "%":
		t.useAPI("math.fmod")
		return lx{code: "math.fmod(" + a.code + ", " + b.code + ")", prec: precAtom, t: binaryNumeric(a.t, b.t), call: true, safe: true, pure: pure, todo: todo}
	case "&", "|", "^":
		if isBool(a.t) || isBool(b.t) {
			switch op {
			case "&":
				return lx{code: paren(a, precAnd) + " and " + paren(b, precAnd+1), prec: precAnd, t: tyBool, todo: todo}
			case "|":
				return lx{code: paren(a, precOr) + " or " + paren(b, precOr+1), prec: precOr, t: tyBool, todo: todo}
			default:
				return lx{code: paren(a, precCmp+1) + " ~= " + paren(b, precCmp+1), prec: precCmp, t: tyBool, todo: todo}
			}
		}
		fn := map[string]string{"&": "J.band", "|": "J.bor", "^": "J.bxor"}[op]
		t.useAPI(fn)
		return lx{code: fn + "(" + a.code + ", " + b.code + ")", prec: precAtom, t: binaryNumeric(a.t, b.t), call: true, safe: true, pure: pure, todo: todo}
	case "<<", ">>", ">>>":
		fn := map[string]string{"<<": "J.shl", ">>": "J.shr", ">>>": "J.ushr"}[op]
		t.useAPI(fn)
		return lx{code: fn + "(" + a.code + ", " + b.code + ")", prec: precAtom, t: a.t, call: true, safe: true, pure: pure, todo: todo}
	}
	return t.todo(line, "operator "+op, "operator "+op)
}

func arith(op string, prec int, a, b lx) lx {
	return lx{
		code: paren(a, prec) + " " + op + " " + paren(b, prec+1),
		prec: prec,
		t:    binaryNumeric(a.t, b.t),
		safe: true,
		pure: a.pure && b.pure,
		todo: a.todo || b.todo,
	}
}

// strOperand converts a string concatenation operand: Lua's .. takes
// strings and numbers; Java prints doubles as 1.0 and booleans as words.
func (t *translator) strOperand(x lx) lx {
	switch {
	case x.nilLit:
		return lx{code: `"null"`, prec: precAtom, t: tyString, safe: true, pure: true}
	case isString(x.t) || isChar(x.t) || isIntegral(x.t):
		return x
	case isFloating(x.t):
		t.useAPI("J.str")
		return lx{code: "J.str(" + x.code + ")", prec: precAtom, t: tyString, call: true, safe: true, pure: x.pure}
	}
	return lx{code: "tostring(" + x.code + ")", prec: precAtom, t: tyString, call: true, safe: true, pure: x.pure}
}

func (t *translator) concat(a, b lx) lx {
	a, b = t.strOperand(a), t.strOperand(b)
	// "" + x is Java's way to write a string conversion.
	if a.code == `""` {
		if isString(b.t) {
			return b
		}
		return lx{code: "tostring(" + b.code + ")", prec: precAtom, t: tyString, call: true, safe: true, pure: b.pure}
	}
	side := func(x lx) string {
		if x.prec == precConcat || x.prec == precAtom {
			return x.code
		}
		return "(" + x.code + ")"
	}
	return lx{code: side(a) + " .. " + side(b), prec: precConcat, t: tyString, safe: true, pure: a.pure && b.pure, todo: a.todo || b.todo}
}

// ternary translates c ? a : b as "c and a or b" when a is never nil or
// false, otherwise as an inline function that evaluates one branch.
func (t *translator) ternary(v *condExpr) lx {
	c := t.expr(v.cond)
	a := t.lazy(v.then)
	b := t.lazy(v.els)
	typ := a.t
	if isUnknown(typ) || typ.kind == tkNull {
		typ = b.t
	}
	if isNumeric(a.t) && isNumeric(b.t) {
		typ = binaryNumeric(a.t, b.t)
	}
	todo := a.todo || b.todo || c.todo
	if a.safe {
		return lx{
			code: paren(c, precAnd) + " and " + paren(a, precAnd+1) + " or " + paren(b, precOr),
			prec: precOr, t: typ, safe: b.safe, pure: a.pure && b.pure && c.pure, todo: todo,
		}
	}
	code := fmt.Sprintf("(function() if %s then return %s end return %s end)()", c.code, a.code, b.code)
	return lx{code: code, prec: precAtom, t: typ, call: true, todo: todo}
}

func (t *translator) cast(v *castExpr) lx {
	x := t.expr(v.x)
	target := typeFromRef(v.typ)
	switch {
	case target.kind == tkPrim && isIntegral(target):
		if isFloating(x.t) {
			t.useAPI("J.int")
			return lx{code: "J.int(" + x.code + ")", prec: precAtom, t: target, call: true, safe: true, pure: x.pure, todo: x.todo}
		}
		x.t = target
		return x
	case target.kind == tkPrim && isFloating(target):
		x.t = target
		return x
	case target.kind == tkPrim && target.name == "char":
		return t.todo(v.line, "cast to char", "cast to char")
	}
	x.t = target
	return x
}

// coerce gives a value the declared type of its variable.
func (t *translator) coerce(v lx, typ *jtype) lx {
	if typ != nil && !isUnknown(typ) && typ.kind != tkVoid {
		if isIntegral(typ) && typ.kind == tkPrim && isFloating(v.t) {
			// Only compound assignments narrow implicitly in Java.
			return v
		}
		if !(isUnknown(v.t) || v.t.kind == tkNull) && isA(v.t, typ.name) && typ.kind == tkClass && len(typ.args) == 0 && len(v.t.args) > 0 {
			return v
		}
		v.t = typ
	}
	return v
}

// --- Assignments and increments used as values ---

// lvalue is an assignable Java expression translated to Lua.
type lvalue struct {
	code string // the Lua target
	t    *jtype
	ref  bool // a field that keeps an object ID
	ok   bool
}

func (t *translator) lvalue(e expr) lvalue {
	switch v := unparen(e).(type) {
	case *identExpr:
		if l := t.lookupLocal(v.name); l != nil {
			return lvalue{code: l.lua, t: l.t, ok: true}
		}
		if t.fn != nil && t.fn.inner != nil {
			if f := t.fn.inner.fields[v.name]; f != nil {
				return lvalue{code: f.lua, t: f.t, ok: true}
			}
		}
		if f := t.fields[v.name]; f != nil {
			t.upval(f.lua)
			return lvalue{code: f.lua, t: f.t, ref: f.ref, ok: true}
		}
	case *fieldExpr:
		if isThis(v.x) {
			return t.lvalue(&identExpr{exprBase: v.exprBase, name: v.name})
		}
		recv := t.expr(v.x)
		if recv.t != nil {
			if ic := t.inner[recv.t.name]; ic != nil {
				if f := ic.fields[v.name]; f != nil {
					return lvalue{code: asAtom(recv) + "." + luaName(v.name), t: f.t, ok: true}
				}
			}
		}
		if recv.isType != "" {
			if ic := t.inner[recv.isType]; ic != nil {
				if f := ic.fields[v.name]; f != nil {
					return lvalue{code: f.lua, t: f.t, ok: true}
				}
			}
		}
	case *indexExpr:
		arr := t.expr(v.x)
		idx := t.indexPlusOne(v.index)
		if !idx.pure {
			if tmp := t.temp("i"); t.hoist("local " + tmp + " = " + idx.code) {
				idx = atom(tmp, tyInt)
			}
		}
		return lvalue{code: asAtom(arr) + "[" + idx.code + "]", t: elemType(arr.t), ok: true}
	}
	return lvalue{}
}

// refStore converts a world object stored in a field into its object ID.
func (t *translator) refStore(v lx) lx {
	if v.nilLit {
		return v
	}
	t.useAPI("J.oid")
	return lx{code: "J.oid(" + v.code + ")", prec: precAtom, t: v.t, call: true}
}

// assignValue translates an assignment used as a value: the assignment is
// hoisted before the statement and the target is read back.
func (t *translator) assignValue(v *assignExpr) lx {
	lv := t.lvalue(v.x)
	if !lv.ok {
		return t.todo(v.line, "assignment target", "assignment to an unsupported target")
	}
	val := t.assignedValue(v, lv)
	if !t.hoist(lv.code + " = " + val.code) {
		return t.todo(v.line, "assignment in condition", "assignment inside a condition")
	}
	r := lx{code: lv.code, prec: precAtom, t: lv.t}
	if lv.ref {
		r.code = "J.obj(" + lv.code + ")"
	}
	return r
}

// assignedValue computes the new value of an assignment target.
func (t *translator) assignedValue(v *assignExpr, lv lvalue) lx {
	if v.op == "=" {
		val := t.expr(v.y)
		if lv.ref {
			val = t.refStore(val)
		}
		return val
	}
	op := strings.TrimSuffix(v.op, "=")
	cur := lx{code: lv.code, prec: precAtom, t: lv.t, pure: true}
	rhs := t.expr(v.y)
	res := t.binop(op, cur, rhs, v.line)
	// Compound assignments narrow back to the target type.
	if isIntegral(lv.t) && isFloating(res.t) {
		t.useAPI("J.int")
		res = lx{code: "J.int(" + res.code + ")", prec: precAtom, t: lv.t, call: true}
	}
	return res
}

// incDecValue translates ++x, x++, --x, x-- used as values.
func (t *translator) incDecValue(v *unaryExpr) lx {
	lv := t.lvalue(v.x)
	if !lv.ok {
		return t.todo(v.line, "increment target", "increment of an unsupported target")
	}
	op := "+"
	if v.op == "--" {
		op = "-"
	}
	update := fmt.Sprintf("%s = %s %s 1", lv.code, lv.code, op)
	if !v.postfix {
		if !t.hoist(update) {
			return t.todo(v.line, "increment in condition", "increment inside a condition")
		}
		return lx{code: lv.code, prec: precAtom, t: lv.t, safe: true}
	}
	tmp := t.temp("old")
	if !t.hoist("local " + tmp + " = " + lv.code) {
		return t.todo(v.line, "increment in condition", "increment inside a condition")
	}
	t.hoist(update)
	return lx{code: tmp, prec: precAtom, t: lv.t, safe: true, pure: true}
}
