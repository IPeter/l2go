package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"strconv"
	"strings"
)

// The world domains read the Mobius XML through mxDoc, a small element tree
// that remembers which elements and attributes the parser looked at. What
// it never looked at is reported as unknown, so a Mobius name the parser
// does not handle fails datagen instead of being dropped.

// mxDoc is a parsed Mobius XML file.
type mxDoc struct {
	path string // path in the Mobius project, for messages
	root *mxElem
	errs []error
}

// mxElem is an element of an mxDoc.
type mxElem struct {
	doc     *mxDoc
	name    string
	line    int
	attrs   []xml.Attr
	read    []bool // per attribute: looked at
	text    string // character data directly inside the element, trimmed
	kids    []*mxElem
	handled bool
}

// parseMX reads and parses the XML file name of fsys.
func parseMX(fsys fs.FS, name string) (*mxDoc, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	doc := &mxDoc{path: name}
	dec := xml.NewDecoder(bytes.NewReader(b))
	var (
		stack []*mxElem
		text  [][]byte // character data of each open element
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			t = t.Copy()
			line, _ := dec.InputPos()
			e := &mxElem{doc: doc, name: t.Name.Local, line: line, attrs: t.Attr, read: make([]bool, len(t.Attr))}
			if len(stack) == 0 {
				if doc.root != nil {
					return nil, fmt.Errorf("parse %s: more than one root element", name)
				}
				doc.root = e
				e.handled = true
			} else {
				parent := stack[len(stack)-1]
				parent.kids = append(parent.kids, e)
			}
			stack = append(stack, e)
			text = append(text, nil)
		case xml.EndElement:
			top := stack[len(stack)-1]
			top.text = strings.TrimSpace(string(text[len(text)-1]))
			stack = stack[:len(stack)-1]
			text = text[:len(text)-1]
		case xml.CharData:
			if len(text) > 0 {
				text[len(text)-1] = append(text[len(text)-1], t...)
			}
		}
	}
	if doc.root == nil {
		return nil, fmt.Errorf("parse %s: no root element", name)
	}
	return doc, nil
}

// errorf records a problem at the element.
func (e *mxElem) errorf(format string, args ...any) {
	e.doc.errs = append(e.doc.errs, fmt.Errorf("%s:%d: <%s>: %s", e.doc.path, e.line, e.name, fmt.Sprintf(format, args...)))
}

// children returns the child elements and marks them handled; the caller
// must handle each one or call unknown on it.
func (e *mxElem) children() []*mxElem {
	for _, c := range e.kids {
		c.handled = true
	}
	return e.kids
}

// unknown reports an element the parser does not expect here.
func (e *mxElem) unknown() { e.errorf("unknown element") }

// skip marks the element and everything in it as handled, for content that
// Mobius ignores too.
func (e *mxElem) skip() {
	e.handled = true
	for i := range e.read {
		e.read[i] = true
	}
	for _, c := range e.kids {
		c.skip()
	}
}

// ignore marks attributes as read without using them.
func (e *mxElem) ignore(names ...string) {
	for _, n := range names {
		e.attr(n)
	}
}

// attr returns an attribute's value and marks it read.
func (e *mxElem) attr(name string) (string, bool) {
	for i, a := range e.attrs {
		if a.Name.Space == "" && a.Name.Local == name {
			e.read[i] = true
			return a.Value, true
		}
	}
	return "", false
}

// str returns a string attribute, or def if it is missing.
func (e *mxElem) str(name, def string) string {
	if v, ok := e.attr(name); ok {
		return v
	}
	return def
}

// reqStr returns a string attribute that must be there.
func (e *mxElem) reqStr(name string) string {
	v, ok := e.attr(name)
	if !ok {
		e.errorf("missing attribute %s", name)
	}
	return v
}

// intAttr returns an integer attribute (Java Integer.decode syntax, as
// Mobius's parseInteger), or def if it is missing.
func (e *mxElem) intAttr(name string, def int) int {
	v, ok := e.attr(name)
	if !ok {
		return def
	}
	n, err := mxInt(v)
	if err != nil {
		e.errorf("attribute %s: %v", name, err)
	}
	return n
}

// reqInt returns an integer attribute that must be there.
func (e *mxElem) reqInt(name string) int {
	if _, ok := e.attr(name); !ok {
		e.errorf("missing attribute %s", name)
		return 0
	}
	return e.intAttr(name, 0)
}

// floatAttr returns a decimal attribute, or def if it is missing.
func (e *mxElem) floatAttr(name string, def float64) float64 {
	v, ok := e.attr(name)
	if !ok {
		return def
	}
	f, err := mxFloat(v)
	if err != nil {
		e.errorf("attribute %s: %v", name, err)
	}
	return f
}

// reqFloat returns a decimal attribute that must be there.
func (e *mxElem) reqFloat(name string) float64 {
	if _, ok := e.attr(name); !ok {
		e.errorf("missing attribute %s", name)
		return 0
	}
	return e.floatAttr(name, 0)
}

// boolAttr returns a boolean attribute (true or false in any case; Java's
// parseBoolean reads anything else as false, datagen refuses it), or def
// if it is missing.
func (e *mxElem) boolAttr(name string, def bool) bool {
	v, ok := e.attr(name)
	if !ok {
		return def
	}
	b, err := parseJavaBool(v)
	if err != nil {
		e.errorf("attribute %s: %v", name, err)
	}
	return b
}

// textInt returns the element's text as an integer.
func (e *mxElem) textInt() int {
	n, err := mxInt(e.text)
	if err != nil {
		e.errorf("text: %v", err)
	}
	return n
}

// textFloat returns the element's text as a decimal.
func (e *mxElem) textFloat() float64 {
	f, err := mxFloat(e.text)
	if err != nil {
		e.errorf("text: %v", err)
	}
	return f
}

// textBool returns the element's text as a boolean (see boolAttr).
func (e *mxElem) textBool() bool {
	b, err := parseJavaBool(e.text)
	if err != nil {
		e.errorf("text: %v", err)
	}
	return b
}

// check returns the recorded problems plus every element nobody handled
// and every attribute nobody read (namespace attributes aside).
func (d *mxDoc) check() error {
	var walk func(e *mxElem)
	walk = func(e *mxElem) {
		if !e.handled {
			e.errorf("unknown element")
			return
		}
		for i, a := range e.attrs {
			if !e.read[i] && a.Name.Space == "" && a.Name.Local != "xmlns" {
				e.errorf("unknown attribute %s", a.Name.Local)
			}
		}
		for _, c := range e.kids {
			walk(c)
		}
	}
	walk(d.root)
	return errors.Join(d.errs...)
}

// mxInt parses an integer like Java's Integer.decode: an optional sign,
// then decimal digits, 0x, 0X or # and hex digits, or 0 and octal digits.
func mxInt(s string) (int, error) {
	t, sign := strings.TrimSpace(s), ""
	if t != "" && (t[0] == '-' || t[0] == '+') {
		sign, t = t[:1], t[1:]
	}
	base := 10
	switch {
	case strings.HasPrefix(t, "0x"), strings.HasPrefix(t, "0X"):
		base, t = 16, t[2:]
	case strings.HasPrefix(t, "#"):
		base, t = 16, t[1:]
	case len(t) > 1 && t[0] == '0':
		base, t = 8, t[1:]
	}
	if t == "" || t[0] == '-' || t[0] == '+' {
		return 0, fmt.Errorf("invalid integer %q", s)
	}
	n, err := strconv.ParseInt(sign+t, base, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", s)
	}
	return int(n), nil
}

// mxFloat parses a decimal number, refusing NaN and infinities.
func mxFloat(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid number %q", s)
	}
	return f, nil
}

// mxRoot returns the root element of a world data file, which must be a
// <list>, with its namespace attributes already accepted.
func mxRoot(doc *mxDoc) (*mxElem, error) {
	if doc.root.name != "list" {
		return nil, fmt.Errorf("%s: root element is <%s>, want <list>", doc.path, doc.root.name)
	}
	return doc.root, nil
}
