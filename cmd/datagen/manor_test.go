package main

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// sampleSeedsXML is the start of Mobius's Seeds.xml: two castles.
const sampleSeedsXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/Seeds.xsd">
	<castle id="1">
		<crop id="5073" seedId="5016" mature_Id="5103" reward1="1864" reward2="1878" alternative="false" level="10" limit_seed="8100" limit_crops="9000" />
		<crop id="5818" seedId="5650" mature_Id="5838" reward1="1864" reward2="1878" alternative="true" level="10" limit_seed="22500" limit_crops="2250" />
	</castle>
	<castle id="2">
		<crop id="5073" seedId="5221" mature_Id="5103" reward1="1864" reward2="1878" alternative="false" level="10" limit_seed="8100" limit_crops="9000" />
	</castle>
</list>`

func seedSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[seedsXML] = file(sampleSeedsXML)
	return m
}

func TestSeeds(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(seedSample(t), false, []string{"manor"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n" +
		`{"castleId":1,"seedId":5016,"cropId":5073,"matureId":5103,"reward1":1864,"reward2":1878,"level":10,"limitSeeds":8100,"limitCrops":9000},` + "\n" +
		`{"castleId":1,"seedId":5650,"cropId":5818,"matureId":5838,"reward1":1864,"reward2":1878,"alternative":true,"level":10,"limitSeeds":22500,"limitCrops":2250},` + "\n" +
		`{"castleId":2,"seedId":5221,"cropId":5073,"matureId":5103,"reward1":1864,"reward2":1878,"level":10,"limitSeeds":8100,"limitCrops":9000}` + "\n]\n"
	if got := string(files[data.SeedsFile]); got != want {
		t.Errorf("%s =\n%s\nwant\n%s", data.SeedsFile, got, want)
	}
	d, err := data.LoadDomains(fstest.MapFS{data.SeedsFile: {Data: files[data.SeedsFile]}}, "manor")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Seeds) != 3 || d.Seeds[1].SeedID != 5650 || !d.Seeds[1].Alternative {
		t.Errorf("Seeds = %+v", d.Seeds)
	}
	if s := "manor: 3 seeds in 2 castles"; !strings.Contains(log.String(), s) {
		t.Errorf("log %q does not mention %q", log.String(), s)
	}
}

func TestSeedsErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[seedsXML] = file(strings.Replace(sampleSeedsXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"manor"}, []sampleCase{
		{"no file", func(*testing.T, fstest.MapFS) {}, "read " + seedsXML},
		{"bad castle", edit(`<castle id="1">`, `<castle id="x">`), `id: "x" is not an int`},
		{"bad level", edit(`level="10"`, `level="ten"`), `level: "ten" is not an int`},
		{"no mature", edit(` mature_Id="5103"`, ``), "no mature_Id attribute"},
		{"bad alternative", edit(`alternative="false"`, `alternative="no"`), `alternative: "no" is not true or false`},
		{"unknown attribute", edit(`level="10"`, `level="10" price="5"`), `unknown attribute "price"`},
		{"unknown element", edit(`<crop id="5818"`, `<seed id="5818"`), "unknown element"},
		{"same seed", edit(`seedId="5221"`, `seedId="5016"`), "seed listed twice"},
	})
}

// The committed seeds are what datagen makes from Mobius.
func TestCommittedSeedsMatchMobius(t *testing.T) {
	committed(t, repoData, false, "manor")
}
