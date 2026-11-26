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

const sampleEnchantSkillsXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/EnchantSkillTreeData.xsd">
	<skill id="1" level="101" name="Triple Slash" desc="+1 Power" baseLevel="37" minSkillLevel="37" sp="306000" exp="3060000" rate76="82" rate77="92" rate78="97" rate79="97" rate80="97" />
	<skill id="1" level="102" name="Triple Slash" desc="+2 Power" baseLevel="37" minSkillLevel="101" sp="315000" exp="3150000" rate76="80" rate77="90" rate78="95" rate79="95" rate80="95" />
	<skill id="1" level="141" name="Triple Slash" desc="+1 Cost" baseLevel="37" minSkillLevel="37" sp="306000" exp="3060000" rate76="82" rate77="92" rate78="97" rate79="97" rate80="97" />
</list>
`

func enchantSkillSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[enchantSkillTreeXML] = file(sampleEnchantSkillsXML)
	return m
}

// The enchant levels keep the XML order, without the name and desc.
func TestEnchantSkills(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(enchantSkillSample(t), false, []string{"skillenchant"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n" +
		`{"id":1,"level":101,"baseLevel":37,"minSkillLevel":37,"sp":306000,"exp":3060000,"rates":[82,92,97,97,97]},` + "\n" +
		`{"id":1,"level":102,"baseLevel":37,"minSkillLevel":101,"sp":315000,"exp":3150000,"rates":[80,90,95,95,95]},` + "\n" +
		`{"id":1,"level":141,"baseLevel":37,"minSkillLevel":37,"sp":306000,"exp":3060000,"rates":[82,92,97,97,97]}` + "\n]\n"
	if got := string(files[data.EnchantSkillTreeFile]); got != want {
		t.Errorf("%s =\n%s\nwant\n%s", data.EnchantSkillTreeFile, got, want)
	}
	tree, err := data.LoadEnchantSkillTreeFS(fstest.MapFS{data.EnchantSkillTreeFile: {Data: files[data.EnchantSkillTreeFile]}})
	if err != nil {
		t.Fatal(err)
	}
	if e := tree.Find(1, 102); e == nil || e.MinSkillLevel != 101 || e.Rate(77) != 90 {
		t.Errorf("Find(1, 102) = %+v", e)
	}
	if !strings.Contains(log.String(), "skillenchant: 3 enchant levels") {
		t.Errorf("log %q", log.String())
	}
}

func TestEnchantSkillsErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[enchantSkillTreeXML] = file(strings.Replace(sampleEnchantSkillsXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"skillenchant"}, []sampleCase{
		{"no file", func(*testing.T, fstest.MapFS) {}, "read " + enchantSkillTreeXML},
		{"twice", edit(`level="102"`, `level="101"`), "listed twice"},
		{"no rate", edit(` rate80="95"`, ``), "missing attribute rate80"},
		{"unknown attribute", edit(`desc="+1 Cost"`, `desc="+1 Cost" route="2"`), "unknown attribute route"},
		{"unknown element", edit(`<skill id="1" level="141"`, `<skills id="1" level="141"`), "unknown element"},
		{"bad rate", edit(`rate76="80"`, `rate76="101"`), "rate 101 out of 0-100"},
	})
}

// The committed enchant data is what datagen makes from Mobius.
func TestCommittedEnchantSkillsMatchMobius(t *testing.T) {
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
	files, err := generate(os.DirFS(dir), false, []string{"skillenchant"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(repoData, filepath.FromSlash(data.EnchantSkillTreeFile)))
	if err != nil {
		t.Fatal(err)
	}
	if got := files[data.EnchantSkillTreeFile]; !bytes.Equal(got, want) {
		t.Errorf("%s is out of date", data.EnchantSkillTreeFile)
	}
	tree, err := data.LoadEnchantSkillTreeFS(fstest.MapFS{data.EnchantSkillTreeFile: {Data: want}})
	if err != nil {
		t.Fatal(err)
	}
	// Mobius loads 14,551 enchant levels; Triple Slash +1 is the first.
	if n := len(tree.Skills); n != 14551 {
		t.Errorf("%d enchant levels, want 14551", n)
	}
	if e := tree.Find(1, 101); e == nil || e.BaseLevel != 37 || e.SP != 306000 || e.Exp != 3060000 || e.Rates != [5]int{82, 92, 97, 97, 97} {
		t.Errorf("Triple Slash +1 = %+v", e)
	}
}
