package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// gzipLimit is the size above which a JSON file is stored gzipped, as
// name.json.gz. internal/data reads both forms.
const gzipLimit = 10 << 20

// output collects the files of a run, keyed by slash-separated path inside
// the output folder.
type output struct {
	files map[string][]byte
	// gzipAbove is the size above which JSON files are gzipped: gzipLimit,
	// lower in tests.
	gzipAbove int
}

func newOutput() *output { return &output{files: make(map[string][]byte), gzipAbove: gzipLimit} }

// add stores a file. It fails on an invalid path or a path that is already
// taken, so two domains cannot overwrite each other.
func (o *output) add(name string, b []byte) error {
	if !fs.ValidPath(name) || name == "." {
		return fmt.Errorf("invalid output path %q", name)
	}
	other := ""
	switch {
	case strings.HasSuffix(name, ".json"):
		other = name + ".gz"
	case strings.HasSuffix(name, ".json.gz"):
		other = strings.TrimSuffix(name, ".gz")
	}
	_, dup := o.files[name]
	_, dupOther := o.files[other]
	if dup || (other != "" && dupOther) {
		return fmt.Errorf("output %s is generated twice", name)
	}
	o.files[name] = b
	return nil
}

// addJSON stores v as indented JSON (see prettyJSON). A name ending in
// .json that comes out bigger than gzipLimit is stored gzipped as
// name.json.gz.
func (o *output) addJSON(name string, v any) error {
	b, err := marshalCompact(v)
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	return o.addSized(name, prettyJSON(b))
}

// addJSONLines stores items as a JSON array with one compact element per
// line: small enough for big tables, and still diffs per record.
func addJSONLines[T any](o *output, name string, items []T) error {
	var buf bytes.Buffer
	buf.WriteString("[")
	for i, it := range items {
		b, err := marshalCompact(it)
		if err != nil {
			return fmt.Errorf("encode %s: element %d: %w", name, i, err)
		}
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
		buf.Write(b)
	}
	if len(items) > 0 {
		buf.WriteString("\n")
	}
	buf.WriteString("]\n")
	return o.addSized(name, buf.Bytes())
}

// addSized stores a JSON file, gzipped if it is over the limit.
func (o *output) addSized(name string, b []byte) error {
	if len(b) <= o.gzipAbove || !strings.HasSuffix(name, ".json") {
		return o.add(name, b)
	}
	z, err := gzipBytes(b)
	if err != nil {
		return fmt.Errorf("compress %s: %w", name, err)
	}
	return o.add(name+".gz", z)
}

// gzipBytes compresses b with a fixed header (no name, no time), so the
// same input gives the same bytes.
func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// marshalCompact returns v as compact JSON without HTML escaping, so names
// like "Squire's Sword" or "<Soul>" stay readable.
func marshalCompact(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// prettyJSON indents compact JSON by two spaces, like json.Indent, except
// that an array holding no objects or arrays (numbers, strings, booleans)
// stays on one line. Per-level tables then take one line per column.
func prettyJSON(compact []byte) []byte {
	var b bytes.Buffer
	depth := 0
	newline := func() {
		b.WriteByte('\n')
		for range depth {
			b.WriteString("  ")
		}
	}
	for i := 0; i < len(compact); i++ {
		c := compact[i]
		switch c {
		case '"':
			end := stringEnd(compact, i)
			b.Write(compact[i:end])
			i = end - 1
		case '{', '[':
			if i+1 < len(compact) && (compact[i+1] == '}' || compact[i+1] == ']') {
				b.Write(compact[i : i+2])
				i++
				continue
			}
			if c == '[' {
				if end, ok := scalarArrayEnd(compact, i); ok {
					b.Write(compact[i:end])
					i = end - 1
					continue
				}
			}
			b.WriteByte(c)
			depth++
			newline()
		case '}', ']':
			depth--
			newline()
			b.WriteByte(c)
		case ',':
			b.WriteByte(',')
			newline()
		case ':':
			b.WriteString(": ")
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('\n')
	return b.Bytes()
}

// stringEnd returns the index after the JSON string that starts at i.
func stringEnd(s []byte, i int) int {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '\\':
			j++
		case '"':
			return j + 1
		}
	}
	return len(s)
}

// scalarArrayEnd reports whether the array starting at i holds no objects
// or arrays, and returns the index after its closing bracket.
func scalarArrayEnd(s []byte, i int) (int, bool) {
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case '"':
			j = stringEnd(s, j) - 1
		case '{', '[':
			return 0, false
		case ']':
			return j + 1, true
		}
	}
	return 0, false
}

// writeFiles writes files into dir and reports each one to log. When a
// JSON file is written, its other form (gzipped or not) is removed, so a
// file that crossed gzipLimit does not stay behind twice.
func writeFiles(dir string, files map[string][]byte, log io.Writer) error {
	for _, name := range slices.Sorted(maps.Keys(files)) {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { //nolint:gosec // the data folder is meant to be readable
			return err
		}
		if err := os.WriteFile(p, files[name], 0o644); err != nil { //nolint:gosec // generated files are public
			return err
		}
		var other string
		switch {
		case strings.HasSuffix(name, ".json"):
			other = name + ".gz"
		case strings.HasSuffix(name, ".json.gz"):
			other = strings.TrimSuffix(name, ".gz")
		}
		if _, generated := files[other]; other != "" && !generated {
			err := os.Remove(filepath.Join(dir, filepath.FromSlash(other)))
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		fmt.Fprintf(log, "wrote %s\n", p)
	}
	return nil
}
