package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrettyJSON(t *testing.T) {
	in := `{"a":1,"b":[1,2.5,"x,y"],"c":[{"d":"[{,:}]","e":[]},{}],"f":{},"g":"q\"{"}`
	want := `{
  "a": 1,
  "b": [1,2.5,"x,y"],
  "c": [
    {
      "d": "[{,:}]",
      "e": []
    },
    {}
  ],
  "f": {},
  "g": "q\"{"
}
`
	if got := string(prettyJSON([]byte(in))); got != want {
		t.Errorf("prettyJSON =\n%s\nwant\n%s", got, want)
	}
}

func TestAddJSON(t *testing.T) {
	o := newOutput()
	v := map[string]any{"name": "Squire's Sword <S>", "levels": []float64{80, 91.83}}
	if err := o.addJSON("players/x.json", v); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"levels\": [80,91.83],\n  \"name\": \"Squire's Sword <S>\"\n}\n"
	if got := string(o.files["players/x.json"]); got != want {
		t.Errorf("addJSON wrote %q, want %q", got, want)
	}
	if err := o.addJSON("players/y.json", func() {}); err == nil {
		t.Error("addJSON of a func succeeded")
	}
}

func TestAddJSONLines(t *testing.T) {
	type rec struct {
		ID   int    `json:"id"`
		Name string `json:"name,omitempty"`
	}
	o := newOutput()
	if err := addJSONLines(o, "items/a.json", []rec{{1, "a&b"}, {2, ""}}); err != nil {
		t.Fatal(err)
	}
	if got, want := string(o.files["items/a.json"]), "[\n{\"id\":1,\"name\":\"a&b\"},\n{\"id\":2}\n]\n"; got != want {
		t.Errorf("addJSONLines wrote %q, want %q", got, want)
	}
	if err := addJSONLines(o, "items/b.json", []rec{}); err != nil {
		t.Fatal(err)
	}
	if got := string(o.files["items/b.json"]); got != "[]\n" {
		t.Errorf("empty list: %q", got)
	}
	if err := addJSONLines(o, "items/c.json", []any{func() {}}); err == nil {
		t.Error("addJSONLines of a func succeeded")
	}
}

func TestOutputGzip(t *testing.T) {
	o := newOutput()
	o.gzipAbove = 16
	if err := o.addJSON("npcs/big.json", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}); err != nil {
		t.Fatal(err)
	}
	if err := o.addJSON("npcs/small.json", 1); err != nil {
		t.Fatal(err)
	}
	if _, ok := o.files["npcs/big.json"]; ok {
		t.Error("a file over the limit is stored plain")
	}
	z, ok := o.files["npcs/big.json.gz"]
	if !ok {
		t.Fatal("a file over the limit is not stored gzipped")
	}
	zr, err := gzip.NewReader(bytes.NewReader(z))
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "[1,2,3,4,5,6,7,8,9,10]\n" {
		t.Errorf("gzipped content %q", b)
	}
	if string(o.files["npcs/small.json"]) != "1\n" {
		t.Errorf("small file %q", o.files["npcs/small.json"])
	}
	// The same input gives the same bytes.
	again, err := gzipBytes(b)
	if err != nil || !bytes.Equal(again, z) {
		t.Errorf("gzip output is not stable: %v", err)
	}

	for _, name := range []string{"npcs/big.json", "npcs/big.json.gz", "npcs/small.json", "npcs/small.json.gz", "../x.json", "/abs.json", ".", ""} {
		if err := o.add(name, nil); err == nil {
			t.Errorf("add(%q) succeeded", name)
		}
	}
}

func TestWriteFiles(t *testing.T) {
	dir := t.TempDir()
	for _, stale := range []string{"a/b.json.gz", "c/d.json", "e/f.htm.gz"} {
		p := filepath.Join(dir, filepath.FromSlash(stale))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string][]byte{"a/b.json": []byte("b"), "c/d.json.gz": []byte("d"), "e/f.htm": []byte("f")}
	var log bytes.Buffer
	if err := writeFiles(dir, files, &log); err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
		if !strings.Contains(log.String(), "wrote "+filepath.Join(dir, filepath.FromSlash(name))) {
			t.Errorf("log %q does not mention %s", log.String(), name)
		}
	}
	for _, gone := range []string{"a/b.json.gz", "c/d.json"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(gone))); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("stale %s: %v, want it removed", gone, err)
		}
	}
	// Only JSON files have a gzipped form.
	if _, err := os.Stat(filepath.Join(dir, "e", "f.htm.gz")); err != nil {
		t.Errorf("e/f.htm.gz: %v, want it kept", err)
	}

	// A file in the way of a folder fails.
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFiles(dir, map[string][]byte{"blocked/x.json": nil}, io.Discard); err == nil {
		t.Error("writeFiles under a file succeeded")
	}
}
