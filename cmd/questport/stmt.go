package main

import (
	"fmt"
	"strconv"
	"strings"
)

func (t *translator) stmtList(stmts []stmt) {
	for _, s := range stmts {
		t.stmt(s)
	}
}

// block translates a statement as the body of a Lua block.
func (t *translator) block(s stmt) {
	t.w.ind++
	t.pushScope()
	if b, ok := s.(*blockStmt); ok {
		t.w.comments(b.comments)
		t.stmtList(b.stmts)
	} else {
		t.stmt(s)
	}
	t.popScope()
	t.w.ind--
}

// asList returns the statements of a block, or the statement alone.
func asList(s stmt) []stmt {
	if b, ok := s.(*blockStmt); ok {
		return b.stmts
	}
	if s == nil {
		return nil
	}
	return []stmt{s}
}

func (t *translator) stmt(s stmt) {
	info := s.stmtInfo()
	if _, isBlock := s.(*blockStmt); !isBlock {
		t.w.comments(info.comments)
	}
	switch v := s.(type) {
	case *blockStmt:
		t.w.comments(v.comments)
		if declaresLocals(v.stmts) {
			t.w.line("do")
			t.block(v)
			t.w.line("end")
		} else {
			t.pushScope()
			t.stmtList(v.stmts)
			t.popScope()
		}
	case *localVarStmt:
		t.localVar(v)
	case *exprStmt:
		t.exprStmt(v)
	case *ifStmt:
		t.ifStmt(v)
	case *switchStmt:
		t.switchStmt(v)
	case *forStmt:
		t.forStmt(v)
	case *forEachStmt:
		t.forEach(v)
	case *whileStmt:
		t.whileStmt(v)
	case *doWhileStmt:
		t.doWhile(v)
	case *returnStmt:
		t.returnStmt(v)
	case *breakStmt:
		t.breakStmt(v)
	case *continueStmt:
		t.continueStmt(v)
	case *throwStmt:
		t.throwStmt(v)
	case *tryStmt:
		t.tryStmt(v)
	case *labeledStmt:
		m := t.todo(v.line, "labeled statement", "labeled statement "+v.label)
		t.flushTodoComments()
		t.w.line(m.code)
	case *syncStmt:
		t.stmt(v.body)
	case *emptyStmt:
	default:
		m := t.todo(s.stmtLine(), "statement", fmt.Sprintf("unsupported statement %T", s))
		t.flushTodoComments()
		t.w.line(m.code)
	}
	t.w.trail(info.trail)
}

// declaresLocals reports whether a statement list declares variables at
// its top level.
func declaresLocals(stmts []stmt) bool {
	for _, s := range stmts {
		if _, ok := s.(*localVarStmt); ok {
			return true
		}
	}
	return false
}

// emitLine writes a statement line after the hoisted statements and the
// todo comments of its expressions.
func (t *translator) emitLine(code string) {
	t.flushPre()
	t.flushTodoComments()
	t.w.line(code)
}

func (t *translator) localVar(v *localVarStmt) {
	for _, d := range v.vars {
		typ := typeFromRef(v.typ)
		for i := 0; i < d.dims; i++ {
			typ = arrayOf(typ)
		}
		if d.init == nil {
			lv := t.declare(d.name, typ)
			t.emitLine("local " + lv.lua)
			continue
		}
		if c, ok := unparen(d.init).(*condExpr); ok && !t.safeTernary(c) {
			lv := t.declare(d.name, typ)
			t.emitLine("local " + lv.lua)
			t.assignTernary(lv.lua, c, typ, false)
			continue
		}
		val := t.initValue(d.init, typ)
		if isUnknown(typ) {
			typ = val.t
		}
		lv := t.declare(d.name, typ)
		if !isUnknown(val.t) && val.t.kind == tkClass && typ.kind == tkClass && (len(typ.args) == 0) && len(val.t.args) > 0 {
			lv.t = val.t
		}
		t.emitLine("local " + lv.lua + " = " + val.code)
	}
}

// safeTernary reports whether c ? a : b can be written as "c and a or b":
// it translates the branch speculatively without writing anything.
func (t *translator) safeTernary(c *condExpr) bool {
	save := *t.res
	saveTodos := len(t.res.todos)
	savePre := len(t.fn.pre)
	saveLines := len(t.fn.todoLines)
	saveAPI := map[string]int{}
	for k, v := range t.res.api {
		saveAPI[k] = v
	}
	a := t.lazy(c.then)
	t.res.todos = t.res.todos[:saveTodos]
	t.res.reviews = save.reviews
	t.res.api = saveAPI
	t.fn.pre = t.fn.pre[:savePre]
	t.fn.todoLines = t.fn.todoLines[:saveLines]
	return a.safe
}

// assignTernary writes target = c ? a : b as an if statement (or return
// when ret is set), descending into nested ternaries in the else branch.
func (t *translator) assignTernary(target string, c *condExpr, typ *jtype, ret bool) {
	cond := t.expr(c.cond)
	t.emitLine("if " + cond.code + " then")
	for {
		t.w.ind++
		t.branchValue(target, c.then, typ, ret)
		t.w.ind--
		next, ok := unparen(c.els).(*condExpr)
		if !ok {
			t.w.line("else")
			t.w.ind++
			t.branchValue(target, c.els, typ, ret)
			t.w.ind--
			break
		}
		cond := t.expr(next.cond)
		if len(t.fn.pre) > 0 {
			t.w.line("else")
			t.w.ind++
			t.assignTernary(target, next, typ, ret)
			t.w.ind--
			break
		}
		t.flushTodoComments()
		t.w.line("elseif " + cond.code + " then")
		c = next
	}
	t.w.line("end")
}

func (t *translator) branchValue(target string, e expr, typ *jtype, ret bool) {
	v := t.expr(e)
	if ret {
		t.emitLine("return " + v.code)
		return
	}
	t.emitLine(target + " = " + v.code)
}

func (t *translator) exprStmt(v *exprStmt) {
	switch x := unparen(v.x).(type) {
	case *assignExpr:
		t.assignStmt(x)
		return
	case *unaryExpr:
		if x.op == "++" || x.op == "--" {
			lv := t.lvalue(x.x)
			if !lv.ok {
				m := t.todo(x.line, "increment target", "increment of an unsupported target")
				t.emitLine(m.code)
				return
			}
			op := "+"
			if x.op == "--" {
				op = "-"
			}
			t.emitLine(fmt.Sprintf("%s = %s %s 1", lv.code, lv.code, op))
			return
		}
	case *callExpr:
		r := t.call(x, true)
		t.writeExprStmt(r)
		return
	}
	r := t.expr(v.x)
	t.writeExprStmt(r)
}

func (t *translator) writeExprStmt(r lx) {
	if r.code == "" {
		t.flushPre()
		t.flushTodoComments()
		return
	}
	if r.call {
		t.emitLine(r.code)
		return
	}
	t.emitLine("local _ = " + r.code)
}

func (t *translator) assignStmt(x *assignExpr) {
	if id, ok := unparen(x.x).(*identExpr); ok && id.name == "_questItemIds" && x.op == "=" && t.lookupLocal(id.name) == nil && t.fields[id.name] == nil {
		// Mobius's public Quest._questItemIds: the registered quest items.
		val := t.expr(x.y)
		t.useAPI("q:register_quest_items")
		t.emitLine("q:register_quest_items(" + val.code + ")")
		return
	}
	lv := t.lvalue(x.x)
	if !lv.ok {
		m := t.todo(x.line, "assignment target", "assignment to an unsupported target")
		t.emitLine(m.code)
		return
	}
	if x.op == "=" {
		if c, ok := unparen(x.y).(*condExpr); ok && !lv.ref && !t.safeTernary(c) {
			t.assignTernary(lv.code, c, lv.t, false)
			return
		}
	}
	val := t.assignedValue(x, lv)
	t.emitLine(lv.code + " = " + val.code)
}

func (t *translator) ifStmt(v *ifStmt) {
	cond := t.expr(v.cond)
	t.emitLine("if " + cond.code + " then")
	t.ifRest(v)
	t.w.line("end")
}

// ifRest writes the branches of an if statement after its "if ... then"
// line, without the final "end".
func (t *translator) ifRest(v *ifStmt) {
	t.block(v.then)
	els := v.els
	for els != nil {
		ei, ok := els.(*ifStmt)
		if !ok || len(ei.comments) > 0 {
			t.w.line("else")
			t.block(els)
			return
		}
		cond := t.expr(ei.cond)
		if len(t.fn.pre) > 0 {
			// The condition needs statements first: nest it.
			t.w.line("else")
			t.w.ind++
			t.emitLine("if " + cond.code + " then")
			t.ifRest(ei)
			t.w.line("end")
			t.w.ind--
			return
		}
		t.flushTodoComments()
		t.w.line("elseif " + cond.code + " then")
		t.block(ei.then)
		els = ei.els
	}
}

func (t *translator) returnStmt(v *returnStmt) {
	if v.x == nil {
		t.emitLine("return")
		return
	}
	if c, ok := unparen(v.x).(*condExpr); ok && !t.safeTernary(c) {
		t.assignTernary("", c, t.fn.ret, true)
		return
	}
	r := t.expr(v.x)
	t.emitLine("return " + r.code)
}

func (t *translator) throwStmt(v *throwStmt) {
	msg := `"exception"`
	if ne, ok := unparen(v.x).(*newExpr); ok && len(ne.args) == 1 {
		m := t.expr(ne.args[0])
		msg = m.code
	}
	t.review(v.line, "throw", "throw becomes error()")
	t.emitLine("error(" + msg + ")")
}

func (t *translator) tryStmt(v *tryStmt) {
	jumps := false
	inspect(v.body, func(x any) bool {
		switch x.(type) {
		case *returnStmt, *breakStmt, *continueStmt:
			jumps = true
		case *lambdaExpr:
			return false
		}
		return !jumps
	})
	if jumps {
		m := t.todo(v.line, "try with jumps", "try block with return, break or continue")
		t.emitLine(m.code)
		return
	}
	ok, errName := t.temp("ok"), t.temp("err")
	t.emitLine(fmt.Sprintf("local %s, %s = pcall(function()", ok, errName))
	t.pushFunc(t.fn.ret)
	t.block(v.body)
	t.popFunc()
	t.w.line("end)")
	if len(v.catches) > 0 {
		t.w.line("if not " + ok + " then")
		t.w.ind++
		c := v.catches[0]
		t.pushScope()
		if mentionsName(c.body, c.name) {
			lv := t.declare(c.name, classType("Exception"))
			t.w.line("local " + lv.lua + " = " + errName)
		}
		t.stmtList(c.body.stmts)
		t.popScope()
		t.w.ind--
		t.w.line("end")
		if len(v.catches) > 1 {
			t.review(v.line, "several catch clauses", "only the first catch clause is kept")
		}
	}
	if v.finally != nil {
		t.stmtList(v.finally.stmts)
	}
}

// --- Switch ---

func (t *translator) switchStmt(v *switchStmt) {
	tag := t.expr(v.tag)
	t.flushPre()
	t.flushTodoComments()
	tagCode := tag.code
	if !tag.pure || tag.prec != precAtom || strings.ContainsAny(tag.code, "(:[") {
		name := t.temp("sw")
		t.w.line("local " + name + " = " + tag.code)
		tagCode = name
	}
	enumTable := ""
	if tag.t != nil {
		for class, table := range enumTables {
			if tag.t.name == class && class != "State" {
				enumTable = table
			}
		}
	}

	// Each case's statements, with the fall-through into the following
	// cases copied in.
	n := len(v.cases)
	bodies := make([][]stmt, n)
	for i := n - 1; i >= 0; i-- {
		body := v.cases[i].body
		if !endsAbruptly(body) && i+1 < n {
			merged := append(append([]stmt{}, body...), bodies[i+1]...)
			if len(body) > 0 {
				t.review(v.cases[i].line, "switch fall-through", "case falls through: the next case's statements are copied")
			}
			body = merged
		}
		bodies[i] = body
	}

	first := true
	dflt := -1
	for i, c := range v.cases {
		if c.dflt {
			dflt = i
			continue
		}
		var conds []string
		for _, l := range c.labels {
			conds = append(conds, tagCode+" == "+paren(t.caseLabel(l, enumTable), precCmp+1))
		}
		t.flushTodoComments()
		if first {
			t.w.line("if " + strings.Join(conds, " or ") + " then")
		} else {
			t.w.line("elseif " + strings.Join(conds, " or ") + " then")
		}
		first = false
		t.caseBody(bodies[i], c)
	}
	if dflt >= 0 {
		if first {
			t.w.line("do")
		} else {
			t.w.line("else")
		}
		t.caseBody(bodies[dflt], v.cases[dflt])
		first = false
	}
	if !first {
		t.w.line("end")
	}
}

// caseLabel translates a case label; enum constants are qualified.
func (t *translator) caseLabel(l expr, enumTable string) lx {
	if id, ok := unparen(l).(*identExpr); ok && enumTable != "" && t.lookupLocal(id.name) == nil && t.fields[id.name] == nil {
		return lx{code: enumTable + "." + id.name, prec: precAtom, t: tyUnknown}
	}
	return t.expr(l)
}

func (t *translator) caseBody(stmts []stmt, c *switchCase) {
	stmts = stripTailBreak(stmts)
	brks, _ := loopJumps(&blockStmt{stmts: stmts})
	t.w.ind++
	t.pushScope()
	t.w.comments(c.comments)
	c.comments = nil
	// "case X: { ... }": the branch is already a scope.
	if len(stmts) == 1 {
		if b, ok := stmts[0].(*blockStmt); ok {
			t.w.comments(b.comments)
			stmts = b.stmts
		}
	}
	if len(brks) > 0 {
		t.w.line("repeat")
		t.w.ind++
		t.fn.targets = append(t.fn.targets, &jumpTarget{luaLoop: true})
		t.stmtList(stmts)
		t.fn.targets = t.fn.targets[:len(t.fn.targets)-1]
		t.w.ind--
		t.w.line("until true")
	} else {
		t.fn.targets = append(t.fn.targets, &jumpTarget{})
		t.stmtList(stmts)
		t.fn.targets = t.fn.targets[:len(t.fn.targets)-1]
	}
	t.popScope()
	t.w.ind--
}

// stripTailBreak removes the break statements that end a case: at the
// end, at the end of a final block, or ending both branches of a final if.
func stripTailBreak(stmts []stmt) []stmt {
	if len(stmts) == 0 {
		return stmts
	}
	last := stmts[len(stmts)-1]
	head := stmts[:len(stmts)-1]
	switch v := last.(type) {
	case *breakStmt:
		if v.label == "" {
			return head
		}
	case *blockStmt:
		nb := *v
		nb.stmts = stripTailBreak(v.stmts)
		return append(append([]stmt{}, head...), &nb)
	case *ifStmt:
		ni := *v
		ni.then = stripTailBreakStmt(v.then)
		if v.els != nil {
			ni.els = stripTailBreakStmt(v.els)
		}
		return append(append([]stmt{}, head...), &ni)
	}
	return stmts
}

func stripTailBreakStmt(s stmt) stmt {
	switch v := s.(type) {
	case *breakStmt:
		if v.label == "" {
			return &blockStmt{stmtBase: v.stmtBase}
		}
	case *blockStmt:
		nb := *v
		nb.stmts = stripTailBreak(v.stmts)
		return &nb
	case *ifStmt:
		ni := *v
		ni.then = stripTailBreakStmt(v.then)
		if v.els != nil {
			ni.els = stripTailBreakStmt(v.els)
		}
		return &ni
	}
	return s
}

// --- Loops ---

func (t *translator) breakStmt(v *breakStmt) {
	if v.label != "" {
		m := t.todo(v.line, "labeled break", "break "+v.label)
		t.emitLine(m.code)
		return
	}
	if len(t.fn.targets) == 0 {
		m := t.todo(v.line, "break outside loop", "break outside a loop or switch")
		t.emitLine(m.code)
		return
	}
	tgt := t.fn.targets[len(t.fn.targets)-1]
	switch {
	case tgt.loop && tgt.brkFlag != "":
		t.emitLine(tgt.brkFlag + " = true")
		t.w.line("break")
	case tgt.luaLoop:
		t.emitLine("break")
	default:
		// The end of a case: nothing to do.
	}
}

func (t *translator) continueStmt(v *continueStmt) {
	if v.label != "" {
		m := t.todo(v.line, "labeled continue", "continue "+v.label)
		t.emitLine(m.code)
		return
	}
	for i := len(t.fn.targets) - 1; i >= 0; i-- {
		tgt := t.fn.targets[i]
		if !tgt.loop {
			if tgt.luaLoop {
				m := t.todo(v.line, "continue inside switch", "continue inside a switch inside a loop")
				t.emitLine(m.code)
				return
			}
			continue
		}
		if tgt.contBreak {
			t.emitLine("break")
			return
		}
		break
	}
	m := t.todo(v.line, "continue", "continue that could not be rewritten")
	t.emitLine(m.code)
}

// loopBody writes a loop body; update runs at the end of every iteration
// (also after continue). It handles continue by moving the rest of the
// body into an else branch, or by running the body in repeat ... until
// true.
func (t *translator) loopBody(body stmt, update func()) {
	breaks, conts := loopJumps(body)
	stmts := asList(body)
	tgt := &jumpTarget{loop: true, luaLoop: true}
	t.w.ind++
	t.pushScope()
	if b, ok := body.(*blockStmt); ok {
		t.w.comments(b.comments)
	}
	if len(conts) > 0 {
		if rewritten, ok := rewriteContinues(stmts); ok {
			stmts = rewritten
			conts = nil
		}
	}
	t.fn.targets = append(t.fn.targets, tgt)
	if len(conts) == 0 {
		t.stmtList(stmts)
	} else {
		flag := ""
		if len(breaks) > 0 {
			flag = t.temp("brk")
			t.w.line("local " + flag + " = false")
			tgt.brkFlag = flag
		}
		tgt.contBreak = true
		t.w.line("repeat")
		t.w.ind++
		t.stmtList(stmts)
		t.w.ind--
		t.w.line("until true")
		if flag != "" {
			t.w.line("if " + flag + " then")
			t.w.line("  break")
			t.w.line("end")
		}
	}
	t.fn.targets = t.fn.targets[:len(t.fn.targets)-1]
	if update != nil {
		update()
	}
	t.popScope()
	t.w.ind--
}

// rewriteContinues rewrites "if (c) { ...; continue; } rest" at the top
// level of a loop body into "if (c) { ... } else { rest }".
func rewriteContinues(stmts []stmt) ([]stmt, bool) {
	for i, s := range stmts {
		_, conts := loopJumps(s)
		if len(conts) == 0 {
			continue
		}
		ifs, ok := s.(*ifStmt)
		if !ok {
			return nil, false
		}
		then := asList(ifs.then)
		if len(then) == 0 {
			return nil, false
		}
		if _, isCont := then[len(then)-1].(*continueStmt); !isCont {
			return nil, false
		}
		head := then[:len(then)-1]
		if _, c := loopJumps(&blockStmt{stmts: head}); len(c) > 0 {
			return nil, false
		}
		var elseStmts []stmt
		if ifs.els != nil {
			if _, c := loopJumps(ifs.els); len(c) > 0 {
				return nil, false
			}
			elseStmts = append(elseStmts, asList(ifs.els)...)
		}
		rest, ok := rewriteContinues(stmts[i+1:])
		if !ok {
			return nil, false
		}
		elseStmts = append(elseStmts, rest...)
		ni := &ifStmt{stmtBase: ifs.stmtBase, cond: ifs.cond}
		if len(head) == 0 && ifs.els == nil {
			ni.cond = &unaryExpr{exprBase: exprBase{ifs.line}, op: "!", x: &parenExpr{exprBase: exprBase{ifs.line}, x: ifs.cond}}
			ni.then = &blockStmt{stmts: elseStmts}
		} else {
			ni.then = &blockStmt{stmts: head}
			if len(elseStmts) > 0 {
				ni.els = &blockStmt{stmts: elseStmts}
			}
		}
		out := append(append([]stmt{}, stmts[:i]...), ni)
		return out, true
	}
	return stmts, true
}

func (t *translator) whileStmt(v *whileStmt) {
	var cond lx
	cond = t.noHoisting(func() lx { return t.expr(v.cond) })
	t.emitLine("while " + cond.code + " do")
	t.loopBody(v.body, nil)
	t.w.line("end")
}

func (t *translator) doWhile(v *doWhileStmt) {
	_, conts := loopJumps(v.body)
	if len(conts) > 0 {
		m := t.todo(v.line, "continue in do-while", "continue inside do ... while")
		t.emitLine(m.code)
		return
	}
	t.emitLine("repeat")
	t.loopBody(v.body, nil)
	cond := t.noHoisting(func() lx { return t.expr(v.cond) })
	t.flushTodoComments()
	t.w.line("until not " + paren(cond, precUnary))
}

func (t *translator) forStmt(v *forStmt) {
	if nf, ok := t.numericFor(v); ok {
		t.flushPre()
		t.flushTodoComments()
		t.w.line(nf)
		t.loopBody(v.body, nil)
		t.popScope()
		t.w.line("end")
		return
	}
	// General form: init; while cond do body; update end.
	t.w.line("do")
	t.w.ind++
	t.pushScope()
	for _, s := range v.init {
		t.stmt(s)
	}
	cond := lx{code: "true", prec: precAtom}
	if v.cond != nil {
		cond = t.noHoisting(func() lx { return t.expr(v.cond) })
	}
	t.emitLine("while " + cond.code + " do")
	t.loopBody(v.body, func() {
		for _, u := range v.update {
			t.stmt(&exprStmt{stmtBase: stmtBase{line: u.exprLine()}, x: u})
		}
	})
	t.w.line("end")
	t.popScope()
	t.w.ind--
	t.w.line("end")
}

// numericFor recognizes for (int i = a; i < b; i++) loops whose variable
// and bound the body does not change, and returns the Lua "for" line. On
// success the loop variable is declared in a new scope the caller pops.
func (t *translator) numericFor(v *forStmt) (string, bool) {
	if len(v.init) != 1 || v.cond == nil || len(v.update) != 1 {
		return "", false
	}
	lv, ok := v.init[0].(*localVarStmt)
	if !ok || len(lv.vars) != 1 || lv.vars[0].init == nil || lv.vars[0].dims != 0 {
		return "", false
	}
	typ := typeFromRef(lv.typ)
	if !isIntegral(typ) {
		return "", false
	}
	name := lv.vars[0].name
	cond, ok := unparen(v.cond).(*binaryExpr)
	if !ok {
		return "", false
	}
	if id, ok := unparen(cond.x).(*identExpr); !ok || id.name != name {
		return "", false
	}
	var step int64
	switch u := unparen(v.update[0]).(type) {
	case *unaryExpr:
		if id, ok := unparen(u.x).(*identExpr); !ok || id.name != name {
			return "", false
		}
		step = 1
		if u.op == "--" {
			step = -1
		}
	case *assignExpr:
		if id, ok := unparen(u.x).(*identExpr); !ok || id.name != name {
			return "", false
		}
		lit, ok := unparen(u.y).(*literalExpr)
		if !ok || lit.kind != litInt {
			return "", false
		}
		n, _ := strconv.ParseInt(lit.value, 10, 64)
		switch u.op {
		case "+=":
			step = n
		case "-=":
			step = -n
		default:
			return "", false
		}
	default:
		return "", false
	}
	if step == 0 || assignsName(v.body, name) || !stableBound(cond.y, v.body) {
		return "", false
	}
	up := cond.op == "<" || cond.op == "<="
	down := cond.op == ">" || cond.op == ">="
	if (up && step < 0) || (down && step > 0) || (!up && !down) {
		return "", false
	}
	start := t.expr(lv.vars[0].init)
	limit := t.expr(cond.y)
	var limitCode string
	switch cond.op {
	case "<":
		limitCode = addConst(limit, -1)
	case ">":
		limitCode = addConst(limit, 1)
	default:
		limitCode = limit.code
	}
	t.pushScope()
	loopVar := t.declare(name, typ)
	line := fmt.Sprintf("for %s = %s, %s", loopVar.lua, start.code, limitCode)
	if step != 1 {
		line += ", " + strconv.FormatInt(step, 10)
	}
	return line + " do", true
}

// addConst returns the code of x + n, folding integer literals.
func addConst(x lx, n int64) string {
	if x.intLit {
		return strconv.FormatInt(x.intVal+n, 10)
	}
	if n < 0 {
		return paren(x, precAdd) + " - " + strconv.FormatInt(-n, 10)
	}
	return paren(x, precAdd) + " + " + strconv.FormatInt(n, 10)
}

// stableBound reports whether a loop bound keeps its value while the body
// runs: literals and names the body does not assign, array lengths and
// collection sizes the body does not change.
func stableBound(e expr, body stmt) bool {
	stable := true
	inspect(e, func(x any) bool {
		switch v := x.(type) {
		case *identExpr:
			if assignsName(body, v.name) || collectionChanged(body, v.name) {
				stable = false
			}
		case *callExpr:
			if v.name != "size" && v.name != "length" {
				stable = false
			}
		case *assignExpr, *unaryExpr:
			if u, ok := v.(*unaryExpr); !ok || u.op == "++" || u.op == "--" {
				stable = false
			}
		}
		return stable
	})
	return stable
}

// collectionChanged reports whether the body adds to or removes from the
// collection held by name.
func collectionChanged(body stmt, name string) bool {
	changed := false
	inspect(body, func(x any) bool {
		if c, ok := x.(*callExpr); ok {
			if id, ok := unparen(c.x).(*identExpr); ok && id.name == name {
				switch c.name {
				case "add", "remove", "clear", "addAll", "removeAll", "put", "removeIf":
					changed = true
				}
			}
		}
		return !changed
	})
	return changed
}

func (t *translator) forEach(v *forEachStmt) {
	elem := typeFromRef(v.varType)
	iter := unparen(v.iter)
	if c, ok := iter.(*callExpr); ok && c.x != nil && len(c.args) == 0 && (c.name == "keySet" || c.name == "values" || c.name == "entrySet") {
		m := t.expr(c.x)
		if isMap(m.t) || isUnknown(m.t) {
			t.mapLoop(v, c.name, m, elem)
			return
		}
	}
	it := t.expr(iter)
	t.flushPre()
	t.flushTodoComments()
	t.pushScope()
	if isUnknown(elem) {
		elem = elemType(it.t)
	}
	lv := t.declare(v.name, elem)
	if isSet(it.t) {
		t.useAPI("J.spairs")
		t.w.linef("for %s in J.spairs(%s) do", lv.lua, it.code)
	} else {
		if !isSequence(it.t) && !isUnknown(it.t) && !it.todo {
			t.review(v.line, "loop over "+it.t.String(), "for-each over "+it.t.String()+" iterates it as an array")
		}
		t.w.linef("for _, %s in ipairs(%s) do", lv.lua, it.code)
	}
	t.loopBody(v.body, nil)
	t.popScope()
	t.w.line("end")
}

// mapLoop writes a for-each over map.keySet(), values() or entrySet() in
// sorted key order.
func (t *translator) mapLoop(v *forEachStmt, kind string, m lx, elem *jtype) {
	t.flushPre()
	t.flushTodoComments()
	t.useAPI("J.spairs")
	brks, _ := loopJumps(v.body)
	returns := false
	inspect(v.body, func(x any) bool {
		if _, ok := x.(*returnStmt); ok {
			returns = true
		}
		return !returns
	})
	if len(brks) > 0 || returns {
		t.review(v.line, "map order", "the loop stops at the first match: Mobius iterates the map in hash order, the Lua in key order")
	}
	t.pushScope()
	switch kind {
	case "keySet":
		if isUnknown(elem) {
			elem = m.t.arg(0)
		}
		lv := t.declare(v.name, elem)
		t.w.linef("for %s in J.spairs(%s) do", lv.lua, m.code)
	case "values":
		if isUnknown(elem) {
			elem = m.t.arg(1)
		}
		lv := t.declare(v.name, elem)
		t.w.linef("for _, %s in J.spairs(%s) do", lv.lua, m.code)
	case "entrySet":
		lv := t.declare(v.name, classType("Entry", m.t.arg(0), m.t.arg(1)))
		key := t.temp(lv.lua + "_key")
		val := t.temp(lv.lua + "_value")
		lv.entryKey, lv.entryVal = key, val
		t.w.linef("for %s, %s in J.spairs(%s) do", key, val, m.code)
	}
	t.loopBody(v.body, nil)
	t.popScope()
	t.w.line("end")
}
