package main

// The Java syntax tree for the quest subset. Every node knows its source
// line; statements and members keep the comments around them.

// typeRef is a type as written in the source.
type typeRef struct {
	name string     // last simple name: "int", "Map", "Entry", "Player"
	qual string     // the dotted name as written: "Map.Entry"
	args []*typeRef // generic arguments; nil for none, empty for "<>"
	dims int        // array dimensions
}

// modifiers are the modifiers and annotations of a declaration.
type modifiers struct {
	flags       map[string]bool // public, private, static, final, ...
	annotations []string
}

func (m modifiers) has(flag string) bool { return m.flags[flag] }

// compilationUnit is one parsed .java file.
type compilationUnit struct {
	file    string
	pkg     string
	imports []string
	types   []*classDecl
}

// member is a class body member.
type member interface{ memberLine() int }

type classDecl struct {
	line     int
	endLine  int
	mods     modifiers
	kind     string // class, interface, enum
	name     string
	extends  *typeRef
	members  []member
	enums    []string // enum constants, in order
	comments []comment
}

type fieldDecl struct {
	line     int
	mods     modifiers
	typ      *typeRef
	vars     []*varDecl
	comments []comment
	trail    []comment
}

type varDecl struct {
	line int
	name string
	dims int
	init expr // nil without initializer
}

type param struct {
	typ     *typeRef
	name    string
	varargs bool
}

type methodDecl struct {
	line     int
	endLine  int
	mods     modifiers
	result   *typeRef // nil for constructors
	name     string
	params   []*param
	body     *blockStmt // nil for abstract methods
	ctor     bool
	comments []comment
}

type initBlock struct {
	line   int
	static bool
	body   *blockStmt
}

func (d *classDecl) memberLine() int  { return d.line }
func (d *fieldDecl) memberLine() int  { return d.line }
func (d *methodDecl) memberLine() int { return d.line }
func (d *initBlock) memberLine() int  { return d.line }

// stmt is a statement.
type stmt interface {
	stmtLine() int
	stmtInfo() *stmtBase
}

// stmtBase holds what every statement has.
type stmtBase struct {
	line     int
	endLine  int
	comments []comment // comment lines before the statement
	trail    []comment // comments after the statement on its last line
}

func (s *stmtBase) stmtLine() int       { return s.line }
func (s *stmtBase) stmtInfo() *stmtBase { return s }

type blockStmt struct {
	stmtBase
	stmts []stmt
}

type localVarStmt struct {
	stmtBase
	typ  *typeRef
	vars []*varDecl
}

type exprStmt struct {
	stmtBase
	x expr
}

type ifStmt struct {
	stmtBase
	cond expr
	then stmt
	els  stmt // nil without else
}

type switchCase struct {
	line     int
	labels   []expr // empty with isDefault and no other labels
	dflt     bool
	body     []stmt
	comments []comment
}

type switchStmt struct {
	stmtBase
	tag   expr
	cases []*switchCase
}

type forStmt struct {
	stmtBase
	init   []stmt // local variable or expression statements
	cond   expr   // nil: forever
	update []expr
	body   stmt
}

type forEachStmt struct {
	stmtBase
	varType *typeRef
	name    string
	iter    expr
	body    stmt
}

type whileStmt struct {
	stmtBase
	cond expr
	body stmt
}

type doWhileStmt struct {
	stmtBase
	body stmt
	cond expr
}

type returnStmt struct {
	stmtBase
	x expr // nil for a bare return
}

type breakStmt struct {
	stmtBase
	label string
}

type continueStmt struct {
	stmtBase
	label string
}

type throwStmt struct {
	stmtBase
	x expr
}

type catchClause struct {
	types []*typeRef
	name  string
	body  *blockStmt
}

type tryStmt struct {
	stmtBase
	resources []stmt
	body      *blockStmt
	catches   []*catchClause
	finally   *blockStmt
}

type labeledStmt struct {
	stmtBase
	label string
	body  stmt
}

type emptyStmt struct {
	stmtBase
}

type syncStmt struct {
	stmtBase
	lock expr
	body *blockStmt
}

// expr is an expression.
type expr interface{ exprLine() int }

type exprBase struct{ line int }

func (e *exprBase) exprLine() int { return e.line }

type identExpr struct {
	exprBase
	name string
}

type litKind int

const (
	litInt litKind = iota
	litLong
	litFloat
	litChar
	litString
	litBool
	litNull
)

type literalExpr struct {
	exprBase
	kind  litKind
	value string // decoded string or normalized number, "true"/"false"
}

type fieldExpr struct {
	exprBase
	x    expr
	name string
}

type callExpr struct {
	exprBase
	x    expr // nil for unqualified calls
	name string
	args []expr
}

type indexExpr struct {
	exprBase
	x, index expr
}

type newExpr struct {
	exprBase
	typ  *typeRef
	args []expr
	body []member // anonymous class body, unsupported by the translator
	anon bool
}

type newArrayExpr struct {
	exprBase
	elem  *typeRef // element type without the dimensions
	dims  []expr   // given dimension sizes
	extra int      // dimensions without a size
	init  *arrayInitExpr
}

type arrayInitExpr struct {
	exprBase
	elems []expr
}

type unaryExpr struct {
	exprBase
	op      string
	x       expr
	postfix bool
}

type binaryExpr struct {
	exprBase
	op   string
	x, y expr
}

type assignExpr struct {
	exprBase
	op   string // "=", "+=", ...
	x, y expr
}

type condExpr struct {
	exprBase
	cond, then, els expr
}

type castExpr struct {
	exprBase
	typ *typeRef
	x   expr
}

type instanceOfExpr struct {
	exprBase
	x   expr
	typ *typeRef
}

type lambdaExpr struct {
	exprBase
	params []string
	body   expr       // expression body, or nil
	block  *blockStmt // block body, or nil
}

type methodRefExpr struct {
	exprBase
	x    expr
	name string
}

type thisExpr struct{ exprBase }

type superExpr struct{ exprBase }

type classLitExpr struct {
	exprBase
	typ *typeRef
}

// parenExpr keeps explicit parentheses; the translator looks through them.
type parenExpr struct {
	exprBase
	x expr
}

// unparen strips parentheses.
func unparen(e expr) expr {
	for {
		p, ok := e.(*parenExpr)
		if !ok {
			return e
		}
		e = p.x
	}
}
