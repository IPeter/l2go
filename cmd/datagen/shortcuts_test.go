package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

const sampleShortcutsXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../../xsd/initialShortcuts.xsd">
	<shortcuts>
		<page pageId="0">
			<slot slotId="0" shortcutType="ACTION" shortcutId="2" /> <!-- Attack -->
			<slot slotId="11" shortcutType="ITEM" shortcutId="5588" />
		</page>
	</shortcuts>
	<shortcuts classId="10">
		<page pageId="0">
			<slot slotId="1" shortcutType="SKILL" shortcutId="1177" shortcutLevel="1" />
		</page>
	</shortcuts>
	<shortcuts>
		<page pageId="1">
			<slot slotId="0" shortcutType="MACRO" shortcutId="10000" />
		</page>
	</shortcuts>
	<macros>
		<macro macroId="10000" icon="1" name="Macro name" description="This is a test." acronym="HI!" enabled="false">
			<command type="TEXT">/loc</command>
		</macro>
	</macros>
</list>
`

func shortcutSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[initialShortcutsXML] = file(sampleShortcutsXML)
	return m
}

// Every class's shortcuts come first, in XML order, then each class's.
func TestShortcuts(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(shortcutSample(t), false, []string{"shortcuts"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n" +
		`{"classId":-1,"page":0,"slot":0,"type":"ACTION","id":2},` + "\n" +
		`{"classId":-1,"page":0,"slot":11,"type":"ITEM","id":5588},` + "\n" +
		`{"classId":-1,"page":1,"slot":0,"type":"MACRO","id":10000},` + "\n" +
		`{"classId":10,"page":0,"slot":1,"type":"SKILL","id":1177,"level":1}` + "\n]\n"
	if got := string(files[data.InitialShortcutsFile]); got != want {
		t.Errorf("%s =\n%s\nwant\n%s", data.InitialShortcutsFile, got, want)
	}
	s, err := data.LoadInitialShortcutsFS(fstest.MapFS{data.InitialShortcutsFile: {Data: files[data.InitialShortcutsFile]}})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.For(10); len(got) != 4 || got[3].ID != 1177 {
		t.Errorf("For(10) = %+v", got)
	}
	if got := s.For(0); len(got) != 3 {
		t.Errorf("For(0) = %+v", got)
	}
	if !strings.Contains(log.String(), "shortcuts: 4 initial shortcuts") {
		t.Errorf("log %q", log.String())
	}
}

func TestShortcutsErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[initialShortcutsXML] = file(strings.Replace(sampleShortcutsXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"shortcuts"}, []sampleCase{
		{"no file", func(*testing.T, fstest.MapFS) {}, "read " + initialShortcutsXML},
		{"twice", edit(`<macros>`, `<shortcuts classId="10"></shortcuts><macros>`), "class 10 is listed twice"},
		{"bad type", edit(`"ACTION"`, `"DANCE"`), `unknown type "DANCE"`},
		{"bad page", edit(`pageId="1"`, `pageId="11"`), "page out of 0-10"},
		{"unknown element", edit(`<slot slotId="0" shortcutType="MACRO"`, `<slots slotId="0" shortcutType="MACRO"`), "unknown element"},
	})
}

// The committed initial shortcuts are what datagen makes from Mobius. On a
// difference the generated file is shown.
// A worktree under .worktrees/ finds the clone of its main checkout.
func TestCommittedShortcutsMatchMobius(t *testing.T) {
	dir := ""
	for _, d := range []string{mobiusDir, filepath.Join("..", "..", "..", "..", mobiusDir[len("../../"):])} {
		if _, err := os.Stat(d); err == nil {
			dir = d
			break
		}
	}
	if dir == "" {
		t.Skipf("no Mobius clone at %s", mobiusDir)
	}
	files, err := generate(os.DirFS(dir), false, []string{"shortcuts"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(repoData, filepath.FromSlash(data.InitialShortcutsFile)))
	if err != nil {
		t.Fatal(err)
	}
	if got := files[data.InitialShortcutsFile]; !bytes.Equal(got, want) {
		t.Errorf("%s is out of date; datagen makes:\n%s", data.InitialShortcutsFile, got)
	}
}
