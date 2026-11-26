package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// olympiadJava is an OlympiadManager.java with the stadiums of
// input/olympiad.json, and one in a comment that must not count.
func olympiadJava(t *testing.T) string {
	t.Helper()
	var in olympiadTables
	if err := json.Unmarshal(olympiadInput, &in); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	b.WriteString("class OlympiadManager implements Runnable\n{\n\tprotected static final OlympiadStadium[] STADIUMS =\n\t{\n")
	b.WriteString("\t\t// new OlympiadStadium(1, 2, 3),\n")
	for i, s := range in.Stadiums {
		sep := ","
		if i == len(in.Stadiums)-1 {
			sep = ""
		}
		fmt.Fprintf(&b, "\t\tnew OlympiadStadium(%d, %d, %d)%s\n", s.X, s.Y, s.Z, sep)
	}
	b.WriteString("\t};\n}\n")
	return b.String()
}

func olympiadSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[olympiadManagerFile] = file(olympiadJava(t))
	return m
}

func TestOlympiadStadiums(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(olympiadSample(t), false, []string{"olympiad"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	got := string(files[data.OlympiadStadiumsFile])
	if !strings.HasPrefix(got, "[\n"+`{"x":-20814,"y":-21189,"z":-3030},`+"\n") || strings.Count(got, "\n") != 24 {
		t.Errorf("%s =\n%s", data.OlympiadStadiumsFile, got)
	}
	list, err := data.LoadOlympiadStadiumsFS(fstest.MapFS{data.OlympiadStadiumsFile: {Data: files[data.OlympiadStadiumsFile]}})
	if err != nil || len(list) != 22 {
		t.Fatalf("load: %d stadiums, %v", len(list), err)
	}
	if !strings.Contains(log.String(), "olympiad: 22 stadiums") {
		t.Errorf("log %q", log.String())
	}
}

func TestOlympiadStadiumsErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[olympiadManagerFile] = file(strings.ReplaceAll(olympiadJava(t), old, new))
		}
	}
	expectErrors(t, []string{"olympiad"}, []sampleCase{
		{"no source", func(*testing.T, fstest.MapFS) {}, "read " + olympiadManagerFile},
		{"a stadium moved", edit("new OlympiadStadium(-20814, -21189, -3030)", "new OlympiadStadium(-20814, -21189, -3000)"), "input/olympiad.json has"},
		{"a stadium more", edit("\t};", "\t\t, new OlympiadStadium(5, 6, 7)\n\t};"), "input/olympiad.json has"},
		{"no table", edit("new OlympiadStadium", "new Stadium"), "no stadium"},
	})
}

// The committed stadiums are what datagen makes from Mobius.
func TestCommittedOlympiadMatchesMobius(t *testing.T) { committed(t, repoData, false, "olympiad") }
