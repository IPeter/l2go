package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// tokKind is the kind of a Java token.
type tokKind int

const (
	tEOF tokKind = iota
	tIdent
	tKeyword
	tInt    // int literal; text holds the decimal value
	tLong   // long literal (L suffix); text holds the decimal value
	tFloat  // float or double literal; text holds a Lua-readable number
	tChar   // char literal; text holds the decoded character
	tString // string literal; text holds the decoded value
	tOp     // operator or punctuation
)

func (k tokKind) String() string {
	switch k {
	case tEOF:
		return "end of file"
	case tIdent:
		return "identifier"
	case tKeyword:
		return "keyword"
	case tInt, tLong:
		return "integer"
	case tFloat:
		return "number"
	case tChar:
		return "char"
	case tString:
		return "string"
	}
	return "operator"
}

// comment is one source comment, kept so the Lua output can carry it.
type comment struct {
	text  string // without the comment markers, trimmed
	line  int
	block bool
}

// token is one lexical token.
type token struct {
	kind tokKind
	text string // see tokKind; the operator or name itself otherwise
	raw  string // source text
	line int
	col  int
	end  int // byte offset just after the token
	off  int // byte offset of the token

	lead  []comment // comments before the token on earlier lines
	trail []comment // comments right after the token on the same line
}

func (t token) is(kind tokKind, text string) bool { return t.kind == kind && t.text == text }

var javaKeywords = map[string]bool{
	"abstract": true, "assert": true, "boolean": true, "break": true, "byte": true,
	"case": true, "catch": true, "char": true, "class": true, "const": true,
	"continue": true, "default": true, "do": true, "double": true, "else": true,
	"enum": true, "extends": true, "final": true, "finally": true, "float": true,
	"for": true, "goto": true, "if": true, "implements": true, "import": true,
	"instanceof": true, "int": true, "interface": true, "long": true, "native": true,
	"new": true, "package": true, "private": true, "protected": true, "public": true,
	"return": true, "short": true, "static": true, "strictfp": true, "super": true,
	"switch": true, "synchronized": true, "this": true, "throw": true, "throws": true,
	"transient": true, "try": true, "void": true, "volatile": true, "while": true,
	"true": true, "false": true, "null": true,
}

// Operators, longest first. '>' is never joined with a following '>' so
// generic types like Map<Integer, List<Integer>> close cleanly; the parser
// joins adjacent '>' tokens into shift operators.
var javaOps = []string{
	"<<=", "...", "->", "::", "++", "--", "&&", "||", "==", "!=", "<=", ">=",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<",
	"(", ")", "{", "}", "[", "]", ";", ",", ".", "@", "=", ">", "<", "!", "~",
	"?", ":", "+", "-", "*", "/", "&", "|", "^", "%",
}

// lexError is a lexing failure with its position.
type lexError struct {
	file string
	line int
	msg  string
}

func (e *lexError) Error() string { return fmt.Sprintf("%s:%d: %s", e.file, e.line, e.msg) }

// lex splits Java source into tokens. Comments are attached to the tokens
// around them: a comment that starts on the line where the previous token
// ends trails that token, any other comment leads the next token.
func lex(file, src string) ([]token, error) {
	lx := &lexer{file: file, src: src, line: 1, col: 1}
	return lx.run()
}

type lexer struct {
	file      string
	src       string
	pos       int
	line, col int
	toks      []token
	pending   []comment // comments waiting for the next token
	lastLine  int       // line of the last token's end, 0 before the first
	lastIndex int       // index of the last token
}

func (lx *lexer) errf(format string, args ...any) error {
	return &lexError{file: lx.file, line: lx.line, msg: fmt.Sprintf(format, args...)}
}

func (lx *lexer) advance(n int) {
	for i := 0; i < n && lx.pos < len(lx.src); i++ {
		if lx.src[lx.pos] == '\n' {
			lx.line++
			lx.col = 1
		} else {
			lx.col++
		}
		lx.pos++
	}
}

func (lx *lexer) run() ([]token, error) {
	for {
		if err := lx.skipSpaceAndComments(); err != nil {
			return nil, err
		}
		if lx.pos >= len(lx.src) {
			lx.toks = append(lx.toks, token{kind: tEOF, line: lx.line, col: lx.col, off: lx.pos, end: lx.pos, lead: lx.pending})
			return lx.toks, nil
		}
		tok, err := lx.next()
		if err != nil {
			return nil, err
		}
		tok.lead = lx.pending
		lx.pending = nil
		lx.toks = append(lx.toks, tok)
		lx.lastLine = lx.line
		lx.lastIndex = len(lx.toks) - 1
	}
}

func (lx *lexer) skipSpaceAndComments() error {
	for lx.pos < len(lx.src) {
		c := lx.src[lx.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f':
			lx.advance(1)
		case strings.HasPrefix(lx.src[lx.pos:], "//"):
			line := lx.line
			end := strings.IndexByte(lx.src[lx.pos:], '\n')
			if end < 0 {
				end = len(lx.src) - lx.pos
			}
			text := strings.TrimSpace(lx.src[lx.pos+2 : lx.pos+end])
			lx.advance(end)
			lx.addComment(comment{text: text, line: line})
		case strings.HasPrefix(lx.src[lx.pos:], "/*"):
			line := lx.line
			end := strings.Index(lx.src[lx.pos+2:], "*/")
			if end < 0 {
				return lx.errf("unterminated block comment")
			}
			body := lx.src[lx.pos+2 : lx.pos+2+end]
			lx.advance(end + 4)
			lx.addComment(comment{text: cleanBlockComment(body), line: line, block: true})
		default:
			return nil
		}
	}
	return nil
}

// addComment attaches c to the previous token when it starts on that
// token's line, otherwise keeps it for the next token.
func (lx *lexer) addComment(c comment) {
	if len(lx.toks) > 0 && c.line == lx.lastLine && !c.block {
		t := &lx.toks[lx.lastIndex]
		t.trail = append(t.trail, c)
		return
	}
	lx.pending = append(lx.pending, c)
}

// cleanBlockComment strips the leading '*' of Javadoc lines.
func cleanBlockComment(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		l = strings.TrimPrefix(l, "*")
		l = strings.TrimSpace(l)
		if l == "" && (len(out) == 0) {
			continue
		}
		out = append(out, l)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

func (lx *lexer) next() (token, error) {
	start, line, col := lx.pos, lx.line, lx.col
	c := lx.src[lx.pos]
	mk := func(kind tokKind, text string) token {
		return token{kind: kind, text: text, raw: lx.src[start:lx.pos], line: line, col: col, off: start, end: lx.pos}
	}
	switch {
	case isIdentStart(c):
		for lx.pos < len(lx.src) {
			r, size := utf8.DecodeRuneInString(lx.src[lx.pos:])
			if r != '_' && r != '$' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				break
			}
			lx.advance(size)
		}
		word := lx.src[start:lx.pos]
		if javaKeywords[word] {
			return mk(tKeyword, word), nil
		}
		return mk(tIdent, word), nil
	case c >= '0' && c <= '9', c == '.' && lx.pos+1 < len(lx.src) && lx.src[lx.pos+1] >= '0' && lx.src[lx.pos+1] <= '9':
		kind, text, err := lx.number()
		if err != nil {
			return token{}, err
		}
		return mk(kind, text), nil
	case c == '"':
		if strings.HasPrefix(lx.src[lx.pos:], `"""`) {
			return token{}, lx.errf("text blocks are not supported")
		}
		s, err := lx.quoted('"')
		if err != nil {
			return token{}, err
		}
		return mk(tString, s), nil
	case c == '\'':
		s, err := lx.quoted('\'')
		if err != nil {
			return token{}, err
		}
		return mk(tChar, s), nil
	}
	for _, op := range javaOps {
		if strings.HasPrefix(lx.src[lx.pos:], op) {
			lx.advance(len(op))
			return mk(tOp, op), nil
		}
	}
	r, _ := utf8.DecodeRuneInString(lx.src[lx.pos:])
	return token{}, lx.errf("unexpected character %q", r)
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

// number scans a numeric literal and returns its kind and a normalized
// value: decimal digits for integers, a Lua-readable number for floats.
func (lx *lexer) number() (tokKind, string, error) {
	start := lx.pos
	s := lx.src
	isDigit := func(c byte, hex bool) bool {
		if c >= '0' && c <= '9' || c == '_' {
			return true
		}
		return hex && (c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F')
	}
	// Hexadecimal and binary.
	if strings.HasPrefix(s[lx.pos:], "0x") || strings.HasPrefix(s[lx.pos:], "0X") ||
		strings.HasPrefix(s[lx.pos:], "0b") || strings.HasPrefix(s[lx.pos:], "0B") {
		base := 16
		if s[lx.pos+1] == 'b' || s[lx.pos+1] == 'B' {
			base = 2
		}
		lx.advance(2)
		digStart := lx.pos
		for lx.pos < len(s) && isDigit(s[lx.pos], base == 16) {
			lx.advance(1)
		}
		digits := strings.ReplaceAll(s[digStart:lx.pos], "_", "")
		kind := tInt
		if lx.pos < len(s) && (s[lx.pos] == 'l' || s[lx.pos] == 'L') {
			kind = tLong
			lx.advance(1)
		}
		v, err := strconv.ParseUint(digits, base, 64)
		if err != nil {
			return 0, "", lx.errf("bad number %q", s[start:lx.pos])
		}
		if kind == tInt && v <= 0xFFFFFFFF && v > 0x7FFFFFFF {
			return kind, strconv.FormatInt(int64(int32(uint32(v))), 10), nil //nolint:gosec // same bits: a Java int hex literal
		}
		return kind, strconv.FormatUint(v, 10), nil
	}
	for lx.pos < len(s) && isDigit(s[lx.pos], false) {
		lx.advance(1)
	}
	isFloat := false
	if lx.pos < len(s) && s[lx.pos] == '.' && lx.pos+1 < len(s) && s[lx.pos+1] >= '0' && s[lx.pos+1] <= '9' {
		isFloat = true
		lx.advance(1)
		for lx.pos < len(s) && isDigit(s[lx.pos], false) {
			lx.advance(1)
		}
	} else if lx.pos < len(s) && s[lx.pos] == '.' && (lx.pos+1 >= len(s) || !isIdentStart(s[lx.pos+1])) {
		// "1." is a double literal.
		isFloat = true
		lx.advance(1)
	}
	if lx.pos < len(s) && (s[lx.pos] == 'e' || s[lx.pos] == 'E') {
		isFloat = true
		lx.advance(1)
		if lx.pos < len(s) && (s[lx.pos] == '+' || s[lx.pos] == '-') {
			lx.advance(1)
		}
		for lx.pos < len(s) && isDigit(s[lx.pos], false) {
			lx.advance(1)
		}
	}
	body := strings.ReplaceAll(s[start:lx.pos], "_", "")
	if lx.pos < len(s) {
		switch s[lx.pos] {
		case 'f', 'F', 'd', 'D':
			lx.advance(1)
			isFloat = true
		case 'l', 'L':
			lx.advance(1)
			v, err := strconv.ParseInt(body, 10, 64)
			if err != nil {
				return 0, "", lx.errf("bad number %q", s[start:lx.pos])
			}
			return tLong, strconv.FormatInt(v, 10), nil
		}
	}
	if isFloat {
		f, err := strconv.ParseFloat(body, 64)
		if err != nil {
			return 0, "", lx.errf("bad number %q", s[start:lx.pos])
		}
		return tFloat, strconv.FormatFloat(f, 'g', -1, 64), nil
	}
	// A leading zero means octal in Java.
	if len(body) > 1 && body[0] == '0' {
		v, err := strconv.ParseInt(body[1:], 8, 64)
		if err != nil {
			return 0, "", lx.errf("bad octal number %q", body)
		}
		return tInt, strconv.FormatInt(v, 10), nil
	}
	return tInt, body, nil
}

// quoted scans a string or char literal and decodes its escapes.
func (lx *lexer) quoted(q byte) (string, error) {
	lx.advance(1)
	var b strings.Builder
	for {
		if lx.pos >= len(lx.src) || lx.src[lx.pos] == '\n' {
			return "", lx.errf("unterminated literal")
		}
		c := lx.src[lx.pos]
		if c == q {
			lx.advance(1)
			return b.String(), nil
		}
		if c != '\\' {
			r, size := utf8.DecodeRuneInString(lx.src[lx.pos:])
			b.WriteRune(r)
			lx.advance(size)
			continue
		}
		if lx.pos+1 >= len(lx.src) {
			return "", lx.errf("unterminated escape")
		}
		e := lx.src[lx.pos+1]
		lx.advance(2)
		switch e {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 's':
			b.WriteByte(' ')
		case '0', '1', '2', '3', '4', '5', '6', '7':
			v := int(e - '0')
			for i := 0; i < 2 && lx.pos < len(lx.src) && lx.src[lx.pos] >= '0' && lx.src[lx.pos] <= '7'; i++ {
				v = v*8 + int(lx.src[lx.pos]-'0')
				lx.advance(1)
			}
			b.WriteRune(rune(v)) //nolint:gosec // at most three octal digits
		case 'u':
			for lx.pos < len(lx.src) && lx.src[lx.pos] == 'u' {
				lx.advance(1)
			}
			if lx.pos+4 > len(lx.src) {
				return "", lx.errf("bad unicode escape")
			}
			v, err := strconv.ParseUint(lx.src[lx.pos:lx.pos+4], 16, 32)
			if err != nil {
				return "", lx.errf("bad unicode escape")
			}
			lx.advance(4)
			b.WriteRune(rune(v)) //nolint:gosec // four hex digits
		default:
			b.WriteByte(e)
		}
	}
}
