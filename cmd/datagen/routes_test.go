package main

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// The walkers name NPCs of the sample datapack (30006 Roxxy, 20481
// Bearded Keltir) and one that is not in it (80000).
const sampleRouteXML = `<?xml version="1.0" encoding="UTF-8"?>
<!-- Possible "repeatStyle" are: -->
<routes xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/Routes.xsd">
	<route name="FPC_Giran_Evi" repeat="true" repeatStyle="cycle">
		<target id="80000" spawnX="83485" spawnY="147998" spawnZ="-3407" />
		<point X="83485" Y="147998" Z="-3407" delay="10" run="true" />
	</route>
	<route name="scribe_leandro" repeat="false" repeatStyle="Back">
		<target id="30006" spawnX="-82428" spawnY="245203" spawnZ="-3712" />
		<target id="20481" spawnX="1" spawnY="2" spawnZ="3" />
		<point X="-82428" Y="245204" Z="-3712" delay="0" run="false" />
		<point string="Where has he gone?" X="-83980" Y="242776" Z="-3728" delay="10" run="false" />
	</route>
</routes>
`

func routeSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[routeXML] = file(sampleRouteXML)
	return m
}

func TestRoutes(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(routeSample(t), false, []string{"routes"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n" +
		`{"name":"FPC_Giran_Evi","repeat":true,"repeatStyle":"cycle","targets":[],"nodes":[{"x":83485,"y":147998,"z":-3407,"delay":10,"run":true}]},` + "\n" +
		`{"name":"scribe_leandro","repeat":false,"repeatStyle":"back","targets":[{"npcId":30006,"x":-82428,"y":245203,"z":-3712},{"npcId":20481,"x":1,"y":2,"z":3}],` +
		`"nodes":[{"x":-82428,"y":245204,"z":-3712,"delay":0},{"x":-83980,"y":242776,"z":-3728,"delay":10,"chat":"Where has he gone?"}]}` + "\n]\n"
	if got := string(files[data.RoutesFile]); got != want {
		t.Errorf("%s =\n%s\nwant\n%s", data.RoutesFile, got, want)
	}
	r, err := data.LoadRoutesFS(fstest.MapFS{data.RoutesFile: {Data: files[data.RoutesFile]}})
	if err != nil {
		t.Fatal(err)
	}
	if route, ok := r.RouteAt(30006, -82428, 245203, -3712); !ok || route.Name != "scribe_leandro" || route.Repeats() {
		t.Errorf("RouteAt(30006) = %+v, %v", route, ok)
	}
	for _, s := range []string{
		"routes: 2 walker routes with 3 nodes and 2 walkers",
		"warning: routes: left out walkers that are not in the NPC data (Mobius skips them): 80000 (FPC_Giran_Evi)",
	} {
		if !strings.Contains(log.String(), s) {
			t.Errorf("log %q does not mention %q", log.String(), s)
		}
	}
}

func TestRoutesErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			if !strings.Contains(sampleRouteXML, old) {
				t.Fatalf("the sample does not contain %q", old)
			}
			m[routeXML] = file(strings.Replace(sampleRouteXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"routes"}, []sampleCase{
		{"no file", func(*testing.T, fstest.MapFS) {}, "read " + routeXML},
		{"twice", edit(`"FPC_Giran_Evi"`, `"scribe_leandro"`), `route "scribe_leandro" is listed twice`},
		{"repeat style", edit(`repeatStyle="cycle"`, `repeatStyle="loop"`), `unknown repeatStyle "loop"`},
		{"no repeat", edit(`repeat="false" `, ""), "missing attribute repeat"},
		{"no run", edit(`delay="10" run="true"`, `delay="10"`), "missing attribute run"},
		{"npc string", edit(`string="Where has he gone?"`, `npcString="WHERE"`), "unknown attribute npcString"},
		{"bad target", edit(`spawnX="1"`, `spawnX="0x1"`), `attribute spawnX: "0x1" is not an int`},
		{"no nodes", edit(`<point X="83485" Y="147998" Z="-3407" delay="10" run="true" />`, ""), `route "FPC_Giran_Evi": no nodes`},
		{"root", func(_ *testing.T, m fstest.MapFS) {
			m[routeXML] = file(strings.ReplaceAll(sampleRouteXML, "routes", "list"))
		}, "root element is <list>, want <routes>"},
	})
}

// The committed routes are what datagen makes from Mobius.
func TestCommittedRoutesMatchMobius(t *testing.T) { committed(t, repoData, false, "routes") }
