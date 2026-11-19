package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strconv"
	"strings"
)

// xmlNode is an XML element read by readXMLTree: its name, attributes,
// child elements and the text directly inside it, with the file and line
// for error messages. Namespace declarations and xsi: attributes are left
// out.
type xmlNode struct {
	name     string
	attrs    []xml.Attr
	children []*xmlNode
	text     string // trimmed
	file     string
	line     int
}

// readXMLTree parses the XML file name and returns its root element.
func readXMLTree(fsys fs.FS, name string) (*xmlNode, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	root, err := parseXMLTree(b, name)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	return root, nil
}

func parseXMLTree(b []byte, file string) (*xmlNode, error) {
	dec := xml.NewDecoder(bytes.NewReader(b))
	var (
		root  *xmlNode
		stack []*xmlNode
		texts []*strings.Builder
	)
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			line, _ := dec.InputPos()
			if t.Name.Space != "" {
				return nil, fmt.Errorf("line %d: element %s:%s has a namespace", line, t.Name.Space, t.Name.Local)
			}
			n := &xmlNode{name: t.Name.Local, file: file, line: line}
			for _, a := range t.Attr {
				if a.Name.Space != "" || a.Name.Local == "xmlns" {
					continue
				}
				n.attrs = append(n.attrs, xml.Attr{Name: xml.Name{Local: a.Name.Local}, Value: a.Value})
			}
			switch {
			case len(stack) > 0:
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, n)
			case root != nil:
				return nil, fmt.Errorf("line %d: a second root element", line)
			default:
				root = n
			}
			stack = append(stack, n)
			texts = append(texts, &strings.Builder{})
		case xml.EndElement:
			n := stack[len(stack)-1]
			n.text = strings.TrimSpace(texts[len(texts)-1].String())
			stack, texts = stack[:len(stack)-1], texts[:len(texts)-1]
		case xml.CharData:
			if len(texts) > 0 {
				texts[len(texts)-1].Write(t)
			}
		}
	}
	if root == nil {
		return nil, errors.New("no root element")
	}
	return root, nil
}

// errorf returns an error located at the element.
func (n *xmlNode) errorf(format string, args ...any) error {
	return fmt.Errorf("%s:%d: <%s>: %s", n.file, n.line, n.name, fmt.Sprintf(format, args...))
}

// attr returns the value of an attribute.
func (n *xmlNode) attr(name string) (string, bool) {
	for _, a := range n.attrs {
		if a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// checkAttrs fails on an attribute that is not allowed, or allowed twice.
func (n *xmlNode) checkAttrs(allowed ...string) error {
	seen := make(map[string]bool, len(n.attrs))
	for _, a := range n.attrs {
		if !slices.Contains(allowed, a.Name.Local) {
			return n.errorf("unknown attribute %q", a.Name.Local)
		}
		if seen[a.Name.Local] {
			return n.errorf("attribute %q given twice", a.Name.Local)
		}
		seen[a.Name.Local] = true
	}
	return nil
}

// checkChildren fails on a child element that is not allowed.
func (n *xmlNode) checkChildren(allowed ...string) error {
	for _, c := range n.children {
		if !slices.Contains(allowed, c.name) {
			return c.errorf("unknown element (want one of %s)", strings.Join(allowed, ", "))
		}
	}
	return nil
}

// leaf fails if the element has attributes or child elements, so only
// its text counts.
func (n *xmlNode) leaf() error {
	if len(n.attrs) > 0 || len(n.children) > 0 {
		return n.errorf("want only a value")
	}
	return nil
}

// intAttr returns a required int attribute.
func (n *xmlNode) intAttr(name string) (int, error) {
	s, ok := n.attr(name)
	if !ok {
		return 0, n.errorf("no %s attribute", name)
	}
	v, err := parseJavaInt(s)
	if err != nil {
		return 0, n.errorf("%s: %v", name, err)
	}
	return v, nil
}

// floatAttr returns a required float attribute.
func (n *xmlNode) floatAttr(name string) (float64, error) {
	s, ok := n.attr(name)
	if !ok {
		return 0, n.errorf("no %s attribute", name)
	}
	v, err := parseJavaFloat(s)
	if err != nil {
		return 0, n.errorf("%s: %v", name, err)
	}
	return v, nil
}

// intText returns the element's text as an int.
func (n *xmlNode) intText() (int, error) {
	if err := n.leaf(); err != nil {
		return 0, err
	}
	v, err := parseJavaInt(n.text)
	if err != nil {
		return 0, n.errorf("%v", err)
	}
	return v, nil
}

// floatText returns the element's text as a float.
func (n *xmlNode) floatText() (float64, error) {
	if err := n.leaf(); err != nil {
		return 0, err
	}
	v, err := parseJavaFloat(n.text)
	if err != nil {
		return 0, n.errorf("%v", err)
	}
	return v, nil
}

// parseJavaInt parses a decimal int as Java Integer.parseInt does.
func parseJavaInt(s string) (int, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is not an int", s)
	}
	return int(v), nil
}

// parseJavaLong parses a decimal long as Java Long.parseLong does.
func parseJavaLong(s string) (int64, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a long", s)
	}
	return v, nil
}

// parseJavaFloat parses a decimal number as Java Double.parseDouble does
// for the forms the data uses.
func parseJavaFloat(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number", s)
	}
	return v, nil
}

// parseJavaBool parses true or false, in any case. Java's
// Boolean.parseBoolean takes anything else as false; datagen refuses it.
func parseJavaBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%q is not true or false", s)
}
