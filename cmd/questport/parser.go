package main

import (
	"fmt"
	"strings"
)

// parseError is a syntax error or a construct outside the supported
// subset, with its position.
type parseError struct {
	file string
	line int
	msg  string
}

func (e *parseError) Error() string { return fmt.Sprintf("%s:%d: %s", e.file, e.line, e.msg) }

// parseJava parses one Java source file of the quest subset.
func parseJava(file, src string) (*compilationUnit, error) {
	toks, err := lex(file, src)
	if err != nil {
		return nil, err
	}
	p := &parser{file: file, toks: toks}
	cu := p.unit()
	if p.err != nil {
		return nil, p.err
	}
	return cu, nil
}

// parser is a recursive descent parser. It stops at the first error: every
// loop checks ok() and every helper does nothing once an error is set.
type parser struct {
	file string
	toks []token
	p    int
	err  error
}

func (p *parser) ok() bool { return p.err == nil }

func (p *parser) tok() token { return p.peek(0) }

func (p *parser) peek(n int) token {
	i := p.p + n
	if i >= len(p.toks) {
		i = len(p.toks) - 1
	}
	return p.toks[i]
}

// prev returns the last consumed token.
func (p *parser) prev() token {
	if p.p == 0 {
		return p.toks[0]
	}
	return p.toks[p.p-1]
}

func (p *parser) next() token {
	t := p.tok()
	if p.err == nil && p.p < len(p.toks)-1 {
		p.p++
	}
	return t
}

func (p *parser) failAt(line int, format string, args ...any) {
	if p.err == nil {
		p.err = &parseError{file: p.file, line: line, msg: fmt.Sprintf(format, args...)}
	}
}

func (p *parser) fail(format string, args ...any) { p.failAt(p.tok().line, format, args...) }

func (p *parser) isOp(op string) bool { return p.tok().is(tOp, op) }
func (p *parser) isKw(kw string) bool { return p.tok().is(tKeyword, kw) }
func (p *parser) isIdent() bool       { return p.tok().kind == tIdent }
func (p *parser) atEnd() bool         { return p.tok().kind == tEOF }
func (p *parser) acceptOp(op string) bool {
	if p.ok() && p.isOp(op) {
		p.next()
		return true
	}
	return false
}

func (p *parser) acceptKw(kw string) bool {
	if p.ok() && p.isKw(kw) {
		p.next()
		return true
	}
	return false
}

func (p *parser) expectOp(op string) {
	if !p.acceptOp(op) {
		p.fail("expected %q, found %s", op, describe(p.tok()))
	}
}

func (p *parser) expectKw(kw string) {
	if !p.acceptKw(kw) {
		p.fail("expected %q, found %s", kw, describe(p.tok()))
	}
}

func (p *parser) expectIdent() string {
	if p.ok() && p.isIdent() {
		return p.next().text
	}
	p.fail("expected a name, found %s", describe(p.tok()))
	return ""
}

func describe(t token) string {
	switch t.kind {
	case tEOF:
		return "end of file"
	case tString:
		return "string literal"
	}
	return fmt.Sprintf("%q", t.raw)
}

// adjacent reports whether the token at offset n+1 directly follows the
// token at offset n, without space.
func (p *parser) adjacent(n int) bool {
	a, b := p.peek(n), p.peek(n+1)
	return a.end == b.off
}

// --- Compilation unit and declarations ---

func (p *parser) unit() *compilationUnit {
	cu := &compilationUnit{file: p.file}
	for p.isOp("@") && p.ok() { // package annotations
		p.annotation()
	}
	if p.acceptKw("package") {
		cu.pkg = p.qualifiedName(false)
		p.expectOp(";")
	}
	for p.isKw("import") && p.ok() {
		p.next()
		p.acceptKw("static")
		cu.imports = append(cu.imports, p.qualifiedName(true))
		p.expectOp(";")
	}
	for p.ok() && !p.atEnd() {
		if p.acceptOp(";") {
			continue
		}
		lead := p.tok().lead
		mods := p.modifiers()
		if p.isKw("class") || p.isKw("interface") || p.isKw("enum") {
			cd := p.classDecl(mods)
			cd.comments = lead
			cu.types = append(cu.types, cd)
			continue
		}
		p.fail("expected a class declaration, found %s", describe(p.tok()))
	}
	return cu
}

func (p *parser) qualifiedName(allowStar bool) string {
	var b strings.Builder
	b.WriteString(p.expectIdent())
	for p.isOp(".") && p.ok() {
		p.next()
		if allowStar && p.acceptOp("*") {
			b.WriteString(".*")
			break
		}
		b.WriteString(".")
		b.WriteString(p.expectIdent())
	}
	return b.String()
}

var modifierKeywords = map[string]bool{
	"public": true, "protected": true, "private": true, "static": true, "final": true,
	"abstract": true, "native": true, "synchronized": true, "transient": true,
	"volatile": true, "strictfp": true, "default": true,
}

func (p *parser) annotation() string {
	p.expectOp("@")
	name := p.qualifiedName(false)
	if p.isOp("(") {
		p.skipBalanced("(", ")")
	}
	return name
}

// skipBalanced skips from an opening token to its matching closing token.
func (p *parser) skipBalanced(open, close string) {
	depth := 0
	for p.ok() && !p.atEnd() {
		t := p.next()
		if t.is(tOp, open) {
			depth++
		} else if t.is(tOp, close) {
			depth--
			if depth == 0 {
				return
			}
		}
	}
	p.fail("unbalanced %q", open)
}

func (p *parser) modifiers() modifiers {
	m := modifiers{flags: map[string]bool{}}
	for p.ok() {
		t := p.tok()
		if t.kind == tKeyword && modifierKeywords[t.text] {
			if t.text == "synchronized" && p.peek(1).is(tOp, "(") {
				break
			}
			m.flags[t.text] = true
			p.next()
			continue
		}
		if t.is(tOp, "@") && !p.peek(1).is(tKeyword, "interface") {
			m.annotations = append(m.annotations, p.annotation())
			continue
		}
		break
	}
	return m
}

func (p *parser) classDecl(mods modifiers) *classDecl {
	cd := &classDecl{line: p.tok().line, mods: mods}
	cd.kind = p.next().text
	cd.name = p.expectIdent()
	if p.isOp("<") {
		p.typeArgs()
	}
	if p.acceptKw("extends") {
		cd.extends = p.parseType(true)
		for p.acceptOp(",") {
			p.parseType(true)
		}
	}
	if p.acceptKw("implements") {
		p.parseType(true)
		for p.acceptOp(",") {
			p.parseType(true)
		}
	}
	p.expectOp("{")
	if cd.kind == "enum" {
		p.enumConstants(cd)
	}
	for p.ok() && !p.isOp("}") && !p.atEnd() {
		if m := p.member(cd.name); m != nil {
			cd.members = append(cd.members, m)
		}
	}
	cd.endLine = p.tok().line
	p.expectOp("}")
	return cd
}

func (p *parser) enumConstants(cd *classDecl) {
	for p.ok() && p.isIdent() {
		cd.enums = append(cd.enums, p.next().text)
		if p.isOp("(") {
			p.skipBalanced("(", ")")
		}
		if p.isOp("{") {
			p.fail("enum constants with bodies are not supported")
			return
		}
		if !p.acceptOp(",") {
			break
		}
	}
	p.acceptOp(";")
}

// member parses one class body member; nil for an empty declaration.
func (p *parser) member(className string) member {
	if p.acceptOp(";") {
		return nil
	}
	start := p.tok()
	if p.isKw("static") && p.peek(1).is(tOp, "{") {
		p.next()
		return &initBlock{line: start.line, static: true, body: p.block()}
	}
	if p.isOp("{") {
		return &initBlock{line: start.line, body: p.block()}
	}
	mods := p.modifiers()
	if p.isKw("class") || p.isKw("interface") || p.isKw("enum") {
		cd := p.classDecl(mods)
		cd.comments = start.lead
		return cd
	}
	if p.isOp("<") {
		p.typeArgs() // generic method
	}
	if p.isIdent() && p.tok().text == className && p.peek(1).is(tOp, "(") {
		md := &methodDecl{line: p.tok().line, mods: mods, name: p.next().text, ctor: true, comments: start.lead}
		p.methodRest(md)
		return md
	}
	typ := p.parseType(true)
	nameTok := p.tok()
	name := p.expectIdent()
	if p.isOp("(") {
		md := &methodDecl{line: nameTok.line, mods: mods, result: typ, name: name, comments: start.lead}
		p.methodRest(md)
		return md
	}
	fd := &fieldDecl{line: start.line, mods: mods, typ: typ, comments: start.lead}
	fd.vars = p.varDeclRest(name, nameTok.line)
	p.expectOp(";")
	fd.trail = p.prev().trail
	return fd
}

func (p *parser) methodRest(md *methodDecl) {
	p.expectOp("(")
	for p.ok() && !p.isOp(")") {
		p.modifiers()
		pr := &param{typ: p.parseType(true)}
		if p.acceptOp("...") {
			pr.varargs = true
			pr.typ.dims++
		}
		pr.name = p.expectIdent()
		for p.isOp("[") && p.peek(1).is(tOp, "]") {
			p.next()
			p.next()
			pr.typ.dims++
		}
		md.params = append(md.params, pr)
		if !p.acceptOp(",") {
			break
		}
	}
	p.expectOp(")")
	for p.isOp("[") && p.peek(1).is(tOp, "]") && p.ok() {
		p.next()
		p.next()
	}
	if p.acceptKw("throws") {
		p.parseType(true)
		for p.acceptOp(",") {
			p.parseType(true)
		}
	}
	if p.acceptOp(";") {
		return
	}
	md.body = p.block()
	md.endLine = md.body.endLine
}

// varDeclRest parses the declarators after the first name of a field or
// local variable declaration, up to (not including) the ';'.
func (p *parser) varDeclRest(first string, line int) []*varDecl {
	var vars []*varDecl
	name := first
	for p.ok() {
		v := &varDecl{line: line, name: name}
		for p.isOp("[") && p.peek(1).is(tOp, "]") {
			p.next()
			p.next()
			v.dims++
		}
		if p.acceptOp("=") {
			if p.isOp("{") {
				v.init = p.arrayInit()
			} else {
				v.init = p.expression()
			}
		}
		vars = append(vars, v)
		if !p.acceptOp(",") {
			break
		}
		line = p.tok().line
		name = p.expectIdent()
	}
	return vars
}

// --- Types ---

var primitiveTypes = map[string]bool{
	"boolean": true, "byte": true, "char": true, "short": true, "int": true,
	"long": true, "float": true, "double": true, "void": true,
}

// parseType parses a type; withDims also takes trailing [] pairs.
func (p *parser) parseType(withDims bool) *typeRef {
	t := p.tok()
	tr := &typeRef{}
	switch {
	case t.kind == tKeyword && primitiveTypes[t.text]:
		p.next()
		tr.name, tr.qual = t.text, t.text
	case t.kind == tIdent:
		p.next()
		tr.name, tr.qual = t.text, t.text
		for p.ok() {
			if p.isOp("<") {
				tr.args = p.typeArgs()
				continue
			}
			if p.isOp(".") && p.peek(1).kind == tIdent {
				p.next()
				n := p.next().text
				tr.name = n
				tr.qual += "." + n
				continue
			}
			break
		}
	default:
		p.fail("expected a type, found %s", describe(t))
		return tr
	}
	if withDims {
		for p.isOp("[") && p.peek(1).is(tOp, "]") && p.ok() {
			p.next()
			p.next()
			tr.dims++
		}
	}
	return tr
}

func (p *parser) typeArgs() []*typeRef {
	p.expectOp("<")
	args := []*typeRef{}
	for p.ok() && !p.isOp(">") {
		if p.acceptOp("?") {
			if p.acceptKw("extends") || p.acceptKw("super") {
				args = append(args, p.parseType(true))
			} else {
				args = append(args, &typeRef{name: "Object", qual: "Object"})
			}
		} else {
			args = append(args, p.parseType(true))
		}
		if !p.acceptOp(",") {
			break
		}
	}
	p.expectOp(">")
	return args
}

// tryType parses a type speculatively; on failure it restores the
// position and returns nil.
func (p *parser) tryType() *typeRef {
	if p.err != nil {
		return nil
	}
	save := p.p
	t := p.parseType(true)
	if p.err != nil {
		p.err = nil
		p.p = save
		return nil
	}
	return t
}

// --- Statements ---

func (p *parser) block() *blockStmt {
	b := &blockStmt{}
	b.line = p.tok().line
	b.comments = p.tok().lead
	p.expectOp("{")
	for p.ok() && !p.isOp("}") && !p.atEnd() {
		b.stmts = append(b.stmts, p.statement())
	}
	b.endLine = p.tok().line
	p.expectOp("}")
	b.trail = p.prev().trail
	return b
}

func (p *parser) statement() stmt {
	start := p.tok()
	s := p.statementInner()
	if s == nil {
		s = &emptyStmt{}
	}
	info := s.stmtInfo()
	info.line = start.line
	if info.comments == nil {
		info.comments = start.lead
	}
	last := p.prev()
	info.endLine = last.line
	if _, isBlock := s.(*blockStmt); !isBlock {
		info.trail = last.trail
	}
	return s
}

func (p *parser) statementInner() stmt {
	t := p.tok()
	switch {
	case t.is(tOp, "{"):
		return p.block()
	case t.is(tOp, ";"):
		p.next()
		return &emptyStmt{}
	case t.is(tKeyword, "if"):
		p.next()
		s := &ifStmt{}
		p.expectOp("(")
		s.cond = p.expression()
		p.expectOp(")")
		s.then = p.statement()
		if p.acceptKw("else") {
			s.els = p.statement()
		}
		return s
	case t.is(tKeyword, "for"):
		return p.forStatement()
	case t.is(tKeyword, "while"):
		p.next()
		s := &whileStmt{}
		p.expectOp("(")
		s.cond = p.expression()
		p.expectOp(")")
		s.body = p.statement()
		return s
	case t.is(tKeyword, "do"):
		p.next()
		s := &doWhileStmt{}
		s.body = p.statement()
		p.expectKw("while")
		p.expectOp("(")
		s.cond = p.expression()
		p.expectOp(")")
		p.expectOp(";")
		return s
	case t.is(tKeyword, "switch"):
		return p.switchStatement()
	case t.is(tKeyword, "return"):
		p.next()
		s := &returnStmt{}
		if !p.isOp(";") {
			s.x = p.expression()
		}
		p.expectOp(";")
		return s
	case t.is(tKeyword, "break"):
		p.next()
		s := &breakStmt{}
		if p.isIdent() {
			s.label = p.next().text
		}
		p.expectOp(";")
		return s
	case t.is(tKeyword, "continue"):
		p.next()
		s := &continueStmt{}
		if p.isIdent() {
			s.label = p.next().text
		}
		p.expectOp(";")
		return s
	case t.is(tKeyword, "throw"):
		p.next()
		s := &throwStmt{x: p.expression()}
		p.expectOp(";")
		return s
	case t.is(tKeyword, "try"):
		return p.tryStatement()
	case t.is(tKeyword, "synchronized"):
		p.next()
		s := &syncStmt{}
		p.expectOp("(")
		s.lock = p.expression()
		p.expectOp(")")
		s.body = p.block()
		return s
	case t.is(tKeyword, "class"), t.is(tKeyword, "interface"), t.is(tKeyword, "enum"):
		p.fail("local classes are not supported")
		return nil
	case t.kind == tIdent && p.peek(1).is(tOp, ":"):
		p.next()
		p.next()
		return &labeledStmt{label: t.text, body: p.statement()}
	case t.is(tKeyword, "final") || t.is(tOp, "@") || t.is(tKeyword, "abstract") || t.is(tKeyword, "static"):
		mods := p.modifiers()
		if p.isKw("class") || mods.has("abstract") || mods.has("static") {
			p.fail("local classes are not supported")
			return nil
		}
		typ := p.parseType(true)
		s := p.localVarRest(typ)
		p.expectOp(";")
		return s
	}
	if typ := p.tryLocalVarType(); typ != nil {
		s := p.localVarRest(typ)
		p.expectOp(";")
		return s
	}
	s := &exprStmt{x: p.expression()}
	p.expectOp(";")
	return s
}

// tryLocalVarType returns the type of a local variable declaration that
// starts here, or nil (position unchanged) when this is an expression.
func (p *parser) tryLocalVarType() *typeRef {
	t := p.tok()
	startsType := t.kind == tIdent || (t.kind == tKeyword && primitiveTypes[t.text] && t.text != "void")
	if !startsType {
		return nil
	}
	save := p.p
	typ := p.tryType()
	if typ != nil && p.isIdent() {
		n := p.peek(1)
		if n.is(tOp, "=") || n.is(tOp, ";") || n.is(tOp, ",") || n.is(tOp, "[") || n.is(tOp, ":") {
			return typ
		}
	}
	p.p = save
	return nil
}

func (p *parser) localVarRest(typ *typeRef) *localVarStmt {
	s := &localVarStmt{typ: typ}
	line := p.tok().line
	name := p.expectIdent()
	s.vars = p.varDeclRest(name, line)
	return s
}

func (p *parser) forStatement() stmt {
	p.next() // for
	p.expectOp("(")
	save := p.p
	p.modifiers()
	if typ := p.tryType(); typ != nil && p.isIdent() && p.peek(1).is(tOp, ":") {
		s := &forEachStmt{varType: typ, name: p.next().text}
		p.next() // :
		s.iter = p.expression()
		p.expectOp(")")
		s.body = p.statement()
		return s
	}
	p.p = save
	s := &forStmt{}
	if !p.isOp(";") {
		line := p.tok().line
		mods := p.modifiers()
		var typ *typeRef
		if mods.has("final") || len(mods.annotations) > 0 {
			typ = p.parseType(true)
		} else {
			typ = p.tryLocalVarType()
		}
		if typ != nil {
			init := p.localVarRest(typ)
			init.line = line
			s.init = append(s.init, init)
		} else {
			for p.ok() {
				x := p.expression()
				es := &exprStmt{x: x}
				es.line = x.exprLine()
				s.init = append(s.init, es)
				if !p.acceptOp(",") {
					break
				}
			}
		}
	}
	p.expectOp(";")
	if !p.isOp(";") {
		s.cond = p.expression()
	}
	p.expectOp(";")
	for p.ok() && !p.isOp(")") {
		s.update = append(s.update, p.expression())
		if !p.acceptOp(",") {
			break
		}
	}
	p.expectOp(")")
	s.body = p.statement()
	return s
}

func (p *parser) switchStatement() stmt {
	p.next() // switch
	s := &switchStmt{}
	p.expectOp("(")
	s.tag = p.expression()
	p.expectOp(")")
	p.expectOp("{")
	for p.ok() && !p.isOp("}") && !p.atEnd() {
		c := &switchCase{line: p.tok().line, comments: p.tok().lead}
		for p.ok() && (p.isKw("case") || p.isKw("default")) {
			if p.acceptKw("default") {
				c.dflt = true
			} else {
				p.next() // case
				for p.ok() {
					c.labels = append(c.labels, p.ternary())
					if !p.acceptOp(",") {
						break
					}
				}
			}
			if p.isOp("->") {
				p.fail("arrow switch cases are not supported")
				return s
			}
			p.expectOp(":")
		}
		if len(c.labels) == 0 && !c.dflt {
			p.fail("expected case or default, found %s", describe(p.tok()))
			return s
		}
		for p.ok() && !p.isKw("case") && !p.isKw("default") && !p.isOp("}") && !p.atEnd() {
			c.body = append(c.body, p.statement())
		}
		s.cases = append(s.cases, c)
	}
	p.expectOp("}")
	return s
}

func (p *parser) tryStatement() stmt {
	p.next() // try
	s := &tryStmt{}
	if p.isOp("(") {
		p.fail("try-with-resources is not supported")
		return s
	}
	s.body = p.block()
	for p.ok() && p.isKw("catch") {
		p.next()
		p.expectOp("(")
		p.modifiers()
		c := &catchClause{types: []*typeRef{p.parseType(true)}}
		for p.acceptOp("|") {
			c.types = append(c.types, p.parseType(true))
		}
		c.name = p.expectIdent()
		p.expectOp(")")
		c.body = p.block()
		s.catches = append(s.catches, c)
	}
	if p.acceptKw("finally") {
		s.finally = p.block()
	}
	if len(s.catches) == 0 && s.finally == nil {
		p.fail("try without catch or finally")
	}
	return s
}

// --- Expressions ---

func (p *parser) expression() expr {
	lhs := p.ternary()
	if op, n := p.assignOp(); op != "" {
		line := p.tok().line
		for i := 0; i < n; i++ {
			p.next()
		}
		rhs := p.expression()
		return &assignExpr{exprBase: exprBase{line}, op: op, x: lhs, y: rhs}
	}
	return lhs
}

// assignOp returns the assignment operator here and its token count.
func (p *parser) assignOp() (string, int) {
	t := p.tok()
	if t.kind != tOp {
		return "", 0
	}
	switch t.text {
	case "=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=":
		return t.text, 1
	case ">":
		if p.adjacent(0) && p.peek(1).is(tOp, ">=") {
			return ">>=", 2
		}
		if p.adjacent(0) && p.peek(1).is(tOp, ">") && p.adjacent(1) && p.peek(2).is(tOp, ">=") {
			return ">>>=", 3
		}
	}
	return "", 0
}

func (p *parser) ternary() expr {
	c := p.binary(0)
	if p.isOp("?") && p.ok() {
		line := p.tok().line
		p.next()
		a := p.ternary()
		p.expectOp(":")
		b := p.ternary()
		return &condExpr{exprBase: exprBase{line}, cond: c, then: a, els: b}
	}
	return c
}

var binaryPrec = map[string]int{
	"||": 1, "&&": 2, "|": 3, "^": 4, "&": 5,
	"==": 6, "!=": 6,
	"<": 7, ">": 7, "<=": 7, ">=": 7, "instanceof": 7,
	"<<": 8, ">>": 8, ">>>": 8,
	"+": 9, "-": 9,
	"*": 10, "/": 10, "%": 10,
}

// binaryOp returns the binary operator here and its token count.
func (p *parser) binaryOp() (string, int) {
	t := p.tok()
	if t.is(tKeyword, "instanceof") {
		return "instanceof", 1
	}
	if t.kind != tOp {
		return "", 0
	}
	if t.text == ">" && p.adjacent(0) && p.peek(1).is(tOp, ">") {
		if p.adjacent(1) && p.peek(2).is(tOp, ">") {
			if p.adjacent(2) && p.peek(3).is(tOp, ">=") {
				return "", 0
			}
			return ">>>", 3
		}
		if p.adjacent(1) && p.peek(2).is(tOp, ">=") {
			return "", 0 // >>>=
		}
		return ">>", 2
	}
	if t.text == ">" && p.adjacent(0) && p.peek(1).is(tOp, ">=") {
		return "", 0 // >>=
	}
	if _, ok := binaryPrec[t.text]; ok {
		return t.text, 1
	}
	return "", 0
}

func (p *parser) binary(minPrec int) expr {
	x := p.unary()
	for p.ok() {
		op, n := p.binaryOp()
		prec, ok := binaryPrec[op]
		if op == "" || !ok || prec <= minPrec {
			break
		}
		line := p.tok().line
		for i := 0; i < n; i++ {
			p.next()
		}
		if op == "instanceof" {
			p.acceptKw("final")
			typ := p.parseType(true)
			if p.isIdent() {
				p.fail("pattern matching instanceof is not supported")
				return x
			}
			x = &instanceOfExpr{exprBase: exprBase{line}, x: x, typ: typ}
			continue
		}
		y := p.binary(prec)
		x = &binaryExpr{exprBase: exprBase{line}, op: op, x: x, y: y}
	}
	return x
}

func (p *parser) unary() expr {
	t := p.tok()
	if t.kind == tOp {
		switch t.text {
		case "+", "-", "!", "~", "++", "--":
			p.next()
			x := p.unary()
			return &unaryExpr{exprBase: exprBase{t.line}, op: t.text, x: x}
		case "(":
			if c := p.tryCast(); c != nil {
				return c
			}
		}
	}
	return p.postfix(p.primary())
}

// tryCast parses a cast at '(' or returns nil with the position unchanged.
func (p *parser) tryCast() expr {
	if p.err != nil {
		return nil
	}
	save := p.p
	line := p.tok().line
	p.next() // (
	t := p.tok()
	if t.kind == tKeyword && primitiveTypes[t.text] {
		typ := p.parseType(true)
		if p.acceptOp(")") {
			return &castExpr{exprBase: exprBase{line}, typ: typ, x: p.unary()}
		}
		p.err = nil
		p.p = save
		return nil
	}
	if t.kind == tIdent {
		typ := p.tryType()
		if typ != nil && p.isOp(")") && castFollows(p.peek(1)) {
			p.next() // )
			return &castExpr{exprBase: exprBase{line}, typ: typ, x: p.unary()}
		}
	}
	p.p = save
	return nil
}

// castFollows reports whether t can start the operand of a reference cast.
func castFollows(t token) bool {
	switch t.kind {
	case tIdent, tInt, tLong, tFloat, tChar, tString:
		return true
	case tKeyword:
		switch t.text {
		case "this", "new", "super", "true", "false", "null":
			return true
		}
		return primitiveTypes[t.text]
	case tOp:
		return t.text == "(" || t.text == "!" || t.text == "~"
	}
	return false
}

func (p *parser) postfix(x expr) expr {
	for p.ok() {
		t := p.tok()
		switch {
		case t.is(tOp, "."):
			p.next()
			if p.isOp("<") {
				p.typeArgs()
			}
			switch {
			case p.isKw("class"):
				p.next()
				x = &classLitExpr{exprBase: exprBase{t.line}, typ: exprToType(x)}
				continue
			case p.isKw("this"):
				p.next()
				x = &thisExpr{exprBase: exprBase{t.line}}
				continue
			case p.isKw("new"):
				p.fail("qualified inner class creation is not supported")
				return x
			}
			name := p.expectIdent()
			if p.isOp("(") {
				x = &callExpr{exprBase: exprBase{t.line}, x: x, name: name, args: p.arguments()}
			} else {
				x = &fieldExpr{exprBase: exprBase{t.line}, x: x, name: name}
			}
		case t.is(tOp, "["):
			p.next()
			idx := p.expression()
			p.expectOp("]")
			x = &indexExpr{exprBase: exprBase{t.line}, x: x, index: idx}
		case t.is(tOp, "++"), t.is(tOp, "--"):
			p.next()
			x = &unaryExpr{exprBase: exprBase{t.line}, op: t.text, x: x, postfix: true}
		case t.is(tOp, "::"):
			p.next()
			name := "new"
			if !p.acceptKw("new") {
				name = p.expectIdent()
			}
			x = &methodRefExpr{exprBase: exprBase{t.line}, x: x, name: name}
		default:
			return x
		}
	}
	return x
}

// exprToType turns a (qualified) name expression into a type.
func exprToType(x expr) *typeRef {
	switch v := x.(type) {
	case *identExpr:
		return &typeRef{name: v.name, qual: v.name}
	case *fieldExpr:
		inner := exprToType(v.x)
		return &typeRef{name: v.name, qual: inner.qual + "." + v.name}
	}
	return &typeRef{name: "?", qual: "?"}
}

func (p *parser) arguments() []expr {
	p.expectOp("(")
	var args []expr
	for p.ok() && !p.isOp(")") {
		args = append(args, p.expression())
		if !p.acceptOp(",") {
			break
		}
	}
	p.expectOp(")")
	return args
}

func (p *parser) primary() expr {
	t := p.tok()
	base := exprBase{t.line}
	switch t.kind {
	case tInt:
		p.next()
		return &literalExpr{exprBase: base, kind: litInt, value: t.text}
	case tLong:
		p.next()
		return &literalExpr{exprBase: base, kind: litLong, value: t.text}
	case tFloat:
		p.next()
		return &literalExpr{exprBase: base, kind: litFloat, value: t.text}
	case tChar:
		p.next()
		return &literalExpr{exprBase: base, kind: litChar, value: t.text}
	case tString:
		p.next()
		return &literalExpr{exprBase: base, kind: litString, value: t.text}
	case tKeyword:
		switch t.text {
		case "true", "false":
			p.next()
			return &literalExpr{exprBase: base, kind: litBool, value: t.text}
		case "null":
			p.next()
			return &literalExpr{exprBase: base, kind: litNull, value: "null"}
		case "this":
			p.next()
			if p.isOp("(") {
				p.fail("constructor chaining with this(...) is not supported")
			}
			return &thisExpr{exprBase: base}
		case "super":
			p.next()
			if p.isOp("(") {
				// super(...) constructor call: kept as a call named "super".
				return &callExpr{exprBase: base, name: "super", args: p.arguments()}
			}
			return &superExpr{exprBase: base}
		case "new":
			return p.newExpr()
		case "switch":
			p.fail("switch expressions are not supported")
			return &literalExpr{exprBase: base, kind: litNull}
		}
		if primitiveTypes[t.text] {
			typ := p.parseType(true)
			if p.isOp(".") && p.peek(1).is(tKeyword, "class") {
				p.next()
				p.next()
				return &classLitExpr{exprBase: base, typ: typ}
			}
			if p.isOp("::") {
				p.next()
				p.acceptKw("new")
				return &methodRefExpr{exprBase: base, x: &identExpr{exprBase: base, name: typ.name}, name: "new"}
			}
			p.fail("unexpected type %s in an expression", typ.name)
			return &literalExpr{exprBase: base, kind: litNull}
		}
	case tIdent:
		if p.peek(1).is(tOp, "->") {
			return p.lambda()
		}
		p.next()
		if p.isOp("(") {
			return &callExpr{exprBase: base, name: t.text, args: p.arguments()}
		}
		return &identExpr{exprBase: base, name: t.text}
	case tOp:
		switch t.text {
		case "(":
			if p.isLambdaParen() {
				return p.lambda()
			}
			p.next()
			x := p.expression()
			p.expectOp(")")
			return &parenExpr{exprBase: base, x: x}
		case "{":
			return p.arrayInit()
		}
	}
	p.fail("unexpected %s", describe(t))
	return &literalExpr{exprBase: base, kind: litNull}
}

// isLambdaParen reports whether the '(' here opens lambda parameters.
func (p *parser) isLambdaParen() bool {
	depth := 0
	for i := p.p; i < len(p.toks); i++ {
		t := p.toks[i]
		if t.is(tOp, "(") {
			depth++
		} else if t.is(tOp, ")") {
			depth--
			if depth == 0 {
				return i+1 < len(p.toks) && p.toks[i+1].is(tOp, "->")
			}
		} else if t.kind == tEOF || t.is(tOp, ";") || t.is(tOp, "{") {
			return false
		}
	}
	return false
}

func (p *parser) lambda() expr {
	l := &lambdaExpr{exprBase: exprBase{p.tok().line}}
	if p.isIdent() {
		l.params = []string{p.next().text}
	} else {
		p.expectOp("(")
		for p.ok() && !p.isOp(")") {
			p.modifiers()
			// Either "name" or "Type name".
			if p.isIdent() && (p.peek(1).is(tOp, ",") || p.peek(1).is(tOp, ")")) {
				l.params = append(l.params, p.next().text)
			} else {
				p.parseType(true)
				l.params = append(l.params, p.expectIdent())
			}
			if !p.acceptOp(",") {
				break
			}
		}
		p.expectOp(")")
	}
	p.expectOp("->")
	if p.isOp("{") {
		l.block = p.block()
	} else {
		l.body = p.expression()
	}
	return l
}

func (p *parser) newExpr() expr {
	line := p.tok().line
	p.next() // new
	typ := p.parseType(false)
	if p.isOp("[") {
		na := &newArrayExpr{exprBase: exprBase{line}, elem: typ}
		for p.ok() && p.isOp("[") {
			p.next()
			if p.acceptOp("]") {
				na.extra++
				continue
			}
			if na.extra > 0 {
				p.fail("array dimension after an empty dimension")
				return na
			}
			na.dims = append(na.dims, p.expression())
			p.expectOp("]")
		}
		if p.isOp("{") {
			na.init = p.arrayInit()
		}
		return na
	}
	ne := &newExpr{exprBase: exprBase{line}, typ: typ, args: p.arguments()}
	if p.isOp("{") {
		ne.anon = true
		p.expectOp("{")
		for p.ok() && !p.isOp("}") && !p.atEnd() {
			if m := p.member(""); m != nil {
				ne.body = append(ne.body, m)
			}
		}
		p.expectOp("}")
	}
	return ne
}

func (p *parser) arrayInit() *arrayInitExpr {
	ai := &arrayInitExpr{exprBase: exprBase{p.tok().line}}
	p.expectOp("{")
	for p.ok() && !p.isOp("}") {
		if p.isOp("{") {
			ai.elems = append(ai.elems, p.arrayInit())
		} else {
			ai.elems = append(ai.elems, p.expression())
		}
		if !p.acceptOp(",") {
			break
		}
	}
	p.expectOp("}")
	return ai
}
