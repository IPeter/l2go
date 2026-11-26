package main

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"strconv"
	"strings"
)

// The boss domains read Mobius's SQL install scripts
// (dist/db_installer/sql/game/*.sql). They hold one table each: a CREATE
// TABLE with the columns and their defaults, then INSERT statements. The
// parser below understands that much MySQL: comments (-- to the end of the
// line, # and /* */), quoted and backquoted names, numbers, strings and
// NULL. Anything else fails.

// sqlTable is a table of an SQL script: its columns and the rows the
// script inserts, in order.
type sqlTable struct {
	file    string
	name    string
	columns []sqlColumn
	rows    []sqlRow
}

// sqlColumn is a column of a CREATE TABLE.
type sqlColumn struct {
	name   string
	def    sqlValue
	hasDef bool
}

// sqlValue is a literal: a number, a string or NULL.
type sqlValue struct {
	text string
	null bool
}

// sqlRow is one inserted row: a value per column, defaults filled in.
type sqlRow struct {
	table *sqlTable
	line  int
	vals  map[string]sqlValue
}

// readSQLTable reads the SQL script name and returns the table it creates
// and fills, which must be table.
func readSQLTable(fsys fs.FS, name, table string) (*sqlTable, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	t, err := parseSQLTable(string(b), name, table)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	return t, nil
}

// SQL token kinds.
const (
	sqlWord   = 'w' // a bare word: keyword or name
	sqlName   = 'q' // a `quoted` name
	sqlString = 's' // a 'string'
	sqlNumber = 'n'
	sqlPunct  = 'p' // one character
)

type sqlToken struct {
	kind byte
	text string
	line int
}

// is reports whether the token is the keyword or punctuation s (keywords in
// any case).
func (t sqlToken) is(s string) bool {
	switch t.kind {
	case sqlWord:
		return strings.EqualFold(t.text, s)
	case sqlPunct:
		return t.text == s
	}
	return false
}

func (t sqlToken) String() string {
	if t.kind == sqlString {
		return "'" + t.text + "'"
	}
	return t.text
}

// sqlTokens splits an SQL script into tokens, without comments.
func sqlTokens(s string) ([]sqlToken, error) {
	var (
		toks []sqlToken
		line = 1
	)
	isDigit := func(c byte) bool { return c >= '0' && c <= '9' }
	isWord := func(c byte) bool {
		return c == '_' || c == '$' || isDigit(c) || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	}
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '#' || strings.HasPrefix(s[i:], "--"):
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case strings.HasPrefix(s[i:], "/*"):
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("line %d: unterminated comment", line)
			}
			line += strings.Count(s[i:i+2+end], "\n")
			i += end + 4
		case c == '`':
			end := strings.IndexByte(s[i+1:], '`')
			if end < 0 {
				return nil, fmt.Errorf("line %d: unterminated name", line)
			}
			toks = append(toks, sqlToken{sqlName, s[i+1 : i+1+end], line})
			i += end + 2
		case c == '\'' || c == '"':
			text, end, ok := sqlQuoted(s, i)
			if !ok {
				return nil, fmt.Errorf("line %d: unterminated string", line)
			}
			toks = append(toks, sqlToken{sqlString, text, line})
			line += strings.Count(s[i:end], "\n")
			i = end
		case isDigit(c) || ((c == '-' || c == '+' || c == '.') && i+1 < len(s) && (isDigit(s[i+1]) || s[i+1] == '.')):
			j := i + 1
			for j < len(s) && (isDigit(s[j]) || s[j] == '.' || s[j] == 'e' || s[j] == 'E' ||
				((s[j] == '-' || s[j] == '+') && (s[j-1] == 'e' || s[j-1] == 'E'))) {
				j++
			}
			toks = append(toks, sqlToken{sqlNumber, s[i:j], line})
			i = j
		case isWord(c):
			j := i + 1
			for j < len(s) && isWord(s[j]) {
				j++
			}
			toks = append(toks, sqlToken{sqlWord, s[i:j], line})
			i = j
		default:
			toks = append(toks, sqlToken{sqlPunct, s[i : i+1], line})
			i++
		}
	}
	return toks, nil
}

// sqlQuoted reads the string that starts with the quote at s[i]: a
// doubled quote or a backslash escapes the next character. It returns the
// text and the index after the closing quote.
func sqlQuoted(s string, i int) (text string, end int, ok bool) {
	q := s[i]
	var b strings.Builder
	for j := i + 1; j < len(s); j++ {
		switch {
		case s[j] == '\\' && j+1 < len(s):
			j++
			b.WriteByte(s[j])
		case s[j] == q && j+1 < len(s) && s[j+1] == q:
			j++
			b.WriteByte(q)
		case s[j] == q:
			return b.String(), j + 1, true
		default:
			b.WriteByte(s[j])
		}
	}
	return "", 0, false
}

// sqlParser reads the statements of a token list.
type sqlParser struct {
	toks []sqlToken
	pos  int
}

func (p *sqlParser) peek() (sqlToken, bool) {
	if p.pos >= len(p.toks) {
		return sqlToken{}, false
	}
	return p.toks[p.pos], true
}

func (p *sqlParser) next() (sqlToken, error) {
	t, ok := p.peek()
	if !ok {
		return t, errors.New("unexpected end of script")
	}
	p.pos++
	return t, nil
}

// accept consumes the next token if it is s.
func (p *sqlParser) accept(s string) bool {
	if t, ok := p.peek(); ok && t.is(s) {
		p.pos++
		return true
	}
	return false
}

func (p *sqlParser) expect(s string) error {
	t, err := p.next()
	if err != nil {
		return fmt.Errorf("want %s: %w", s, err)
	}
	if !t.is(s) {
		return fmt.Errorf("line %d: want %s, have %s", t.line, s, t)
	}
	return nil
}

// name reads a table or column name.
func (p *sqlParser) name() (string, error) {
	t, err := p.next()
	if err != nil {
		return "", err
	}
	if t.kind != sqlName && t.kind != sqlWord {
		return "", fmt.Errorf("line %d: want a name, have %s", t.line, t)
	}
	return t.text, nil
}

// value reads a literal.
func (p *sqlParser) value() (sqlValue, error) {
	t, err := p.next()
	if err != nil {
		return sqlValue{}, err
	}
	switch {
	case t.kind == sqlNumber, t.kind == sqlString:
		return sqlValue{text: t.text}, nil
	case t.is("NULL"):
		return sqlValue{null: true}, nil
	}
	return sqlValue{}, fmt.Errorf("line %d: want a value, have %s", t.line, t)
}

// parseSQLTable parses an SQL script that creates and fills table.
func parseSQLTable(script, file, table string) (*sqlTable, error) {
	toks, err := sqlTokens(script)
	if err != nil {
		return nil, err
	}
	p := &sqlParser{toks: toks}
	var t *sqlTable
	for {
		for p.accept(";") {
			// Skip empty statements.
		}
		first, ok := p.peek()
		if !ok {
			break
		}
		switch {
		case first.is("CREATE"):
			if t != nil {
				return nil, fmt.Errorf("line %d: a second CREATE TABLE", first.line)
			}
			if t, err = p.createTable(file); err != nil {
				return nil, err
			}
			if t.name != table {
				return nil, fmt.Errorf("line %d: creates table %s, want %s", first.line, t.name, table)
			}
		case first.is("DROP"):
			// DROP TABLE [IF EXISTS] name: nothing to read (castle_siege_guards).
			for {
				tok, ok := p.peek()
				if !ok || tok.is(";") {
					break
				}
				p.pos++
			}
		case first.is("INSERT"), first.is("REPLACE"):
			if t == nil {
				return nil, fmt.Errorf("line %d: INSERT before CREATE TABLE", first.line)
			}
			if err := p.insert(t); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("line %d: unsupported statement %s", first.line, first)
		}
		if tok, ok := p.peek(); ok && !tok.is(";") {
			return nil, fmt.Errorf("line %d: want ; after the statement, have %s", tok.line, tok)
		}
	}
	if t == nil {
		return nil, fmt.Errorf("no CREATE TABLE %s", table)
	}
	return t, nil
}

// createTable reads CREATE TABLE [IF NOT EXISTS] name (definitions)
// options: the column names and defaults. Keys are skipped; the table
// options after the closing parenthesis too.
func (p *sqlParser) createTable(file string) (*sqlTable, error) {
	if err := p.expect("CREATE"); err != nil {
		return nil, err
	}
	if err := p.expect("TABLE"); err != nil {
		return nil, err
	}
	if p.accept("IF") {
		if err := p.expect("NOT"); err != nil {
			return nil, err
		}
		if err := p.expect("EXISTS"); err != nil {
			return nil, err
		}
	}
	name, err := p.name()
	if err != nil {
		return nil, err
	}
	t := &sqlTable{file: file, name: name}
	if err := p.expect("("); err != nil {
		return nil, err
	}
	for {
		// One definition: up to a comma or the closing parenthesis at depth 0.
		var def []sqlToken
		depth := 0
		for {
			tok, err := p.next()
			if err != nil {
				return nil, err
			}
			if depth == 0 && (tok.is(",") || tok.is(")")) {
				p.pos--
				break
			}
			switch {
			case tok.is("("):
				depth++
			case tok.is(")"):
				depth--
			}
			def = append(def, tok)
		}
		if err := t.addColumn(def); err != nil {
			return nil, err
		}
		if p.accept(")") {
			break
		}
		if err := p.expect(","); err != nil {
			return nil, err
		}
	}
	// Table options up to the end of the statement.
	for {
		tok, ok := p.peek()
		if !ok || tok.is(";") {
			break
		}
		p.pos++
	}
	if len(t.columns) == 0 {
		return nil, fmt.Errorf("table %s has no columns", name)
	}
	return t, nil
}

// addColumn adds the column a definition of CREATE TABLE defines, if it
// defines one (not a key).
func (t *sqlTable) addColumn(def []sqlToken) error {
	if len(def) == 0 {
		return errors.New("empty definition in CREATE TABLE")
	}
	first := def[0]
	if first.kind == sqlWord {
		for _, kw := range []string{"PRIMARY", "KEY", "UNIQUE", "INDEX", "CONSTRAINT", "FOREIGN", "FULLTEXT"} {
			if first.is(kw) {
				return nil
			}
		}
	}
	if first.kind != sqlName && first.kind != sqlWord {
		return fmt.Errorf("line %d: want a column name, have %s", first.line, first)
	}
	if t.column(first.text) >= 0 {
		return fmt.Errorf("line %d: column %s is defined twice", first.line, first.text)
	}
	c := sqlColumn{name: first.text}
	for i := 1; i < len(def); i++ {
		if !def[i].is("DEFAULT") {
			continue
		}
		if i+1 == len(def) {
			return fmt.Errorf("line %d: DEFAULT without a value", def[i].line)
		}
		v := def[i+1]
		switch {
		case v.kind == sqlNumber, v.kind == sqlString:
			c.def = sqlValue{text: v.text}
		case v.is("NULL"):
			c.def = sqlValue{null: true}
		default:
			return fmt.Errorf("line %d: unsupported default %s", v.line, v)
		}
		c.hasDef = true
	}
	t.columns = append(t.columns, c)
	return nil
}

// column returns the index of a column, or -1.
func (t *sqlTable) column(name string) int {
	for i, c := range t.columns {
		if c.name == name {
			return i
		}
	}
	return -1
}

// insert reads INSERT [IGNORE] [INTO] name [(columns)] VALUES (row), ...
// into t.
func (p *sqlParser) insert(t *sqlTable) error {
	// REPLACE INTO inserts the same way into an empty table.
	if !p.accept("REPLACE") {
		if err := p.expect("INSERT"); err != nil {
			return err
		}
	}
	p.accept("IGNORE")
	p.accept("INTO")
	start, _ := p.peek()
	name, err := p.name()
	if err != nil {
		return err
	}
	if name != t.name {
		return fmt.Errorf("line %d: insert into %s, want %s", start.line, name, t.name)
	}
	var cols []string
	if p.accept("(") {
		for {
			c, err := p.name()
			if err != nil {
				return err
			}
			if t.column(c) < 0 {
				return fmt.Errorf("line %d: unknown column %s", start.line, c)
			}
			cols = append(cols, c)
			if p.accept(")") {
				break
			}
			if err := p.expect(","); err != nil {
				return err
			}
		}
	} else {
		for _, c := range t.columns {
			cols = append(cols, c.name)
		}
	}
	if !p.accept("VALUES") && !p.accept("VALUE") {
		return fmt.Errorf("line %d: want VALUES", start.line)
	}
	for {
		open, _ := p.peek()
		if err := p.expect("("); err != nil {
			return err
		}
		row := sqlRow{table: t, line: open.line, vals: make(map[string]sqlValue, len(t.columns))}
		for i := 0; ; i++ {
			v, err := p.value()
			if err != nil {
				return err
			}
			if i >= len(cols) {
				return fmt.Errorf("line %d: more values than columns", open.line)
			}
			row.vals[cols[i]] = v
			if p.accept(")") {
				if i+1 != len(cols) {
					return fmt.Errorf("line %d: %d values for %d columns", open.line, i+1, len(cols))
				}
				break
			}
			if err := p.expect(","); err != nil {
				return err
			}
		}
		for _, c := range t.columns {
			if _, ok := row.vals[c.name]; !ok && c.hasDef {
				row.vals[c.name] = c.def
			}
		}
		t.rows = append(t.rows, row)
		if !p.accept(",") {
			return nil
		}
	}
}

// errorf returns an error located at the row.
func (r sqlRow) errorf(format string, args ...any) error {
	return fmt.Errorf("%s:%d: %s: %s", r.table.file, r.line, r.table.name, fmt.Sprintf(format, args...))
}

// get returns a column's value; a column the row does not set and that
// has no default is an error.
func (r sqlRow) get(col string) (sqlValue, error) {
	v, ok := r.vals[col]
	if !ok {
		if r.table.column(col) < 0 {
			return v, r.errorf("no column %s", col)
		}
		return v, r.errorf("column %s has no value and no default", col)
	}
	return v, nil
}

// long returns an integer column as JDBC getLong does (NULL is 0).
func (r sqlRow) long(col string) (int64, error) {
	v, err := r.get(col)
	if err != nil || v.null {
		return 0, err
	}
	n, err := strconv.ParseInt(v.text, 10, 64)
	if err != nil {
		return 0, r.errorf("column %s: %q is not an integer", col, v.text)
	}
	return n, nil
}

// int returns an integer column as JDBC getInt does (NULL is 0).
func (r sqlRow) int(col string) (int, error) {
	n, err := r.long(col)
	if err != nil {
		return 0, err
	}
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, r.errorf("column %s: %d is out of the int range", col, n)
	}
	return int(n), nil
}

// float returns a decimal column as JDBC getDouble does (NULL is 0).
func (r sqlRow) float(col string) (float64, error) {
	v, err := r.get(col)
	if err != nil || v.null {
		return 0, err
	}
	f, err := mxFloat(v.text)
	if err != nil {
		return 0, r.errorf("column %s: %v", col, err)
	}
	return f, nil
}
