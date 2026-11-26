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

const sampleHennaListXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../xsd/hennaList.xsd">
	<henna dyeId="2" dyeName="dye_s1d3_d" dyeItemId="4446" dyeLevel="28">
		<stats str="1" con="0" dex="-3" int="0" men="0" wit="0" />
		<wear count="10" fee="37000" />
		<cancel count="5" fee="7400" />
		<classId>11</classId> <!-- Human Wizard -->
		<classId>1</classId> <!-- Warrior -->
		<classId>11</classId>
	</henna>
	<henna dyeId="1" dyeName="dye_s1c3_d" dyeItemId="4445" dyeLevel="28">
		<stats str="1" con="-3" />
		<wear count="10" fee="37000" />
		<cancel count="5" fee="7400" />
		<classId>1</classId>
	</henna>
</list>
`

func hennaSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[hennaListXML] = file(sampleHennaListXML)
	return m
}

// The dyes come sorted by dye ID, their classes as a sorted set; missing
// stats are 0 and left out.
func TestHennas(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(hennaSample(t), false, []string{"hennas"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n" +
		`{"dyeId":1,"dyeItemId":4445,"str":1,"con":-3,"wearCount":10,"wearFee":37000,"cancelCount":5,"cancelFee":7400,"classes":[1]},` + "\n" +
		`{"dyeId":2,"dyeItemId":4446,"str":1,"dex":-3,"wearCount":10,"wearFee":37000,"cancelCount":5,"cancelFee":7400,"classes":[1,11]}` + "\n]\n"
	if got := string(files[data.HennasFile]); got != want {
		t.Errorf("%s =\n%s\nwant\n%s", data.HennasFile, got, want)
	}
	h, err := data.LoadHennasFS(fstest.MapFS{data.HennasFile: {Data: files[data.HennasFile]}})
	if err != nil {
		t.Fatal(err)
	}
	if got := h.ForClass(11); len(got) != 1 || got[0].DyeID != 2 {
		t.Errorf("ForClass(11) = %+v", got)
	}
	if !strings.Contains(log.String(), "hennas: 2 dyes") {
		t.Errorf("log %q", log.String())
	}
}

func TestHennasErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[hennaListXML] = file(strings.Replace(sampleHennaListXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"hennas"}, []sampleCase{
		{"no file", func(*testing.T, fstest.MapFS) {}, "read " + hennaListXML},
		{"twice", edit(`dyeId="1"`, `dyeId="2"`), "dye 2 is listed twice"},
		{"unknown class", edit(`<classId>1</classId>
	</henna>`, `<classId>999</classId>
	</henna>`), "unknown class 999"},
		{"no wear", edit(`<wear count="10" fee="37000" />`, ``), "has no <wear>"},
		{"bad fee", edit(`fee="7400"`, `fee="lots"`), "is not an int"},
		{"unknown element", edit(`<stats str="1" con="-3" />`, `<stat str="1" />`), "unknown element"},
		{"unknown stat", edit(`<stats str="1" con="-3" />`, `<stats luc="1" />`), `unknown attribute "luc"`},
		{"negative count", edit(`<cancel count="5" fee="7400" />
		<classId>1</classId>`, `<cancel count="-5" fee="7400" />
		<classId>1</classId>`), "a negative count or fee"},
	})
}

// The committed hennas are what datagen makes from Mobius: 180 dyes. On a
// difference the generated file is shown. A worktree under .worktrees/ finds
// the clone of its main checkout.
func TestCommittedHennasMatchMobius(t *testing.T) {
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
	files, err := generate(os.DirFS(dir), false, []string{"hennas"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(repoData, filepath.FromSlash(data.HennasFile)))
	if err != nil {
		t.Fatal(err)
	}
	if got := files[data.HennasFile]; !bytes.Equal(got, want) {
		t.Errorf("%s is out of date; datagen makes:\n%s", data.HennasFile, got)
	}
	h, err := data.LoadHennasFS(fstest.MapFS{data.HennasFile: {Data: want}})
	if err != nil {
		t.Fatal(err)
	}
	// The demo dye: Dye of STR (Str+1 Con-3), 10 dyes and 37,000 adena to
	// draw, 5 dyes back for 7,400 adena.
	d := h.Henna(1)
	if len(h.List) != 180 || d == nil || d.DyeItemID != 4445 || d.STR != 1 || d.CON != -3 ||
		d.WearCount != 10 || d.WearFee != 37000 || d.CancelCount != 5 || d.CancelFee != 7400 {
		t.Errorf("%d dyes, dye 1 = %+v", len(h.List), d)
	}
}
