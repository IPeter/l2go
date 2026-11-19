package main

import (
	"bytes"
	"errors"
	"flag"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

var updateWorld = flag.Bool("update-world", false, "rewrite the world golden files in testdata/golden")

const (
	worldSample = "testdata/mobius"
	worldGolden = "testdata/golden"
	// The local Mobius clone and the committed data, relative to this
	// package.
	worldMobius   = "../../L2J_Mobius/L2J_Mobius_CT_0_Interlude"
	worldRepoData = "../../data"
)

// worldOutputs are the files of the world domain (none of the sample's is
// big enough to be gzipped).
var worldOutputs = []string{
	data.NpcTypesFile, data.NpcsFile, data.TerritoriesFile, data.SpawnsFile,
	data.TeleportersFile, data.ZoneTypesFile, data.ZonesFile, data.MapRegionsFile,
}

// runWorld runs the world domain on fsys.
func runWorld(fsys fs.FS) (files map[string][]byte, log string, err error) {
	var buf bytes.Buffer
	out := newOutput()
	err = genWorld(options{src: fsys, out: out, log: &buf})
	return out.files, buf.String(), err
}

// The sample in testdata/mobius holds cut-down Mobius files plus a few
// marked "Not Mobius data" for the cases Interlude does not use. The
// golden files were checked by hand against it: Mobius defaults applied,
// custom and fake player NPCs left out, disabled lists skipped, zones
// without an ID numbered from 300001 in file order, cell 19_16 kept by
// rune_town only.
func TestWorldGolden(t *testing.T) {
	files, log, err := runWorld(os.DirFS(worldSample))
	if err != nil {
		t.Fatalf("genWorld: %v", err)
	}
	if got := slices.Sorted(maps.Keys(files)); !slices.Equal(got, slices.Sorted(slices.Values(worldOutputs))) {
		t.Errorf("genWorld wrote %v, want %v", got, worldOutputs)
	}
	for _, name := range worldOutputs {
		golden := filepath.Join(worldGolden, filepath.FromSlash(name))
		if *updateWorld {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, files[name], 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(files[name], want) {
			t.Errorf("%s differs from %s\n--- got ---\n%s\n--- want ---\n%s", name, golden, files[name], want)
		}
	}
	for _, want := range []string{
		"npcs: left out 1 fake player NPCs",
		"spawns: skipped disabled lists: Others/Disabled.xml",
		"spawns: left out 1 entries of 1 NPCs that are not in the NPC data (custom or fake player NPCs, which Mobius skips too): 31688",
		"teleporters: left out the lists of NPCs that are not in the NPC data: 50000",
		"zones: skipped disabled lists: tax.xml",
		"cell 19_16 is listed by orc_town and rune_town; Mobius's grid gives it to rune_town",
		"world: 6 NPC templates, 5 spawn blocks with 7 entries, 2 territories, 3 teleporters with 9 locations, 7 zones, 4 map regions",
	} {
		if !strings.Contains(log, want) {
			t.Errorf("log does not mention %q:\n%s", want, log)
		}
	}
}

// The generated files load, and the loader sees what the sample says.
func TestWorldLoads(t *testing.T) {
	files, _, err := runWorld(os.DirFS(worldSample))
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	for name, b := range files {
		fsys[name] = &fstest.MapFile{Data: b}
	}
	w, err := data.LoadWorldFS(fsys)
	if err != nil {
		t.Fatalf("LoadWorldFS: %v", err)
	}
	if _, ok := w.Npc(50000); ok {
		t.Error("custom NPC 50000 is in the data")
	}
	if _, ok := w.Npc(20091); ok {
		t.Error("fake player NPC 20091 is in the data")
	}
	boss, ok := w.Npc(20090)
	if !ok {
		t.Fatal("NPC 20090 not found")
	}
	if w.Spawnable(20090) || !w.NpcTypeIs(boss.Type, "Monster") {
		t.Error("the raid boss should be a Monster that the spawn lists do not spawn")
	}
	// The sample boss: walk 0 becomes 0.1, which keeps canMove on.
	if boss.Stats.Speed.Walk != 0.1 || !boss.Status.CanMove || boss.Level != 85 || boss.Race != data.RaceNone {
		t.Errorf("sample boss = speed %+v, status %+v, level %d, race %v", boss.Stats.Speed, boss.Status, boss.Level, boss.Race)
	}
	if guard, _ := w.Npc(30099); guard.Status.RandomWalk || !guard.Status.Undying {
		t.Errorf("sample guard status = %+v, want no random walk, undying", guard.Status)
	}
	if r, ok := w.MapRegionAt(-10000, -40000); !ok || r.Name != "rune_town" {
		t.Errorf("cell 19_16 belongs to %+v, %v; want rune_town", r, ok)
	}
	if z, ok := w.Zone(300001); !ok || z.Name != "gludio_castle_owner_restart_territory" || len(z.Spawns) != 1 {
		t.Errorf("Zone(300001) = %+v, %v", z, ok)
	}
	terr, ok := w.Territory("sample_square")
	if !ok || len(terr.Banned) != 1 {
		t.Fatalf("Territory(sample_square) = %+v, %v", terr, ok)
	}
	// (-3500 + -3401) / 2, rounded toward zero as in Java.
	if loc := terr.RandomPoint(func(n int) int { return 0 }); loc.Z != -3450 {
		t.Errorf("sample_square spawn Z = %d, want -3450", loc.Z)
	}
	if got := w.TeleportLists(30099); len(got) != 2 || got[1].Locations[0].CastleIDs[1] != 2 {
		t.Errorf("TeleportLists(30099) = %+v", got)
	}
}

func TestWorldIsStable(t *testing.T) {
	a, _, err := runWorld(os.DirFS(worldSample))
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		b, _, err := runWorld(os.DirFS(worldSample))
		if err != nil {
			t.Fatal(err)
		}
		if !maps.EqualFunc(a, b, bytes.Equal) {
			t.Fatal("two runs on the same input gave different output")
		}
	}
}

// worldSampleFS returns the sample as an in-memory file system a test can
// change.
func worldSampleFS(t *testing.T) fstest.MapFS {
	t.Helper()
	root := os.DirFS(worldSample)
	m := fstest.MapFS{}
	err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(root, p)
		if err != nil {
			return err
		}
		m[p] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// xmlList wraps elements in a <list>.
func xmlList(attrs, body string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n<list " + attrs + ">\n" + body + "\n</list>\n")}
}

func TestWorldErrors(t *testing.T) {
	const (
		npcFile      = npcXMLDir + "/90000-90099.xml"
		spawnFile    = spawnXMLDir + "/Bad/Bad.xml"
		teleFile     = teleporterXMLDir + "/town/30006b.xml"
		zoneFile     = zoneXMLDir + "/zz.xml"
		regionFile   = mapRegionXMLDir + "/zz.xml"
		instanceFile = npcTypeJavaDir + "/Foo.java"
	)
	triangle := `<node x="0" y="0" /><node x="10" y="0" /><node x="0" y="10" />`
	zoneTriangle := `<node X="0" Y="0" /><node X="10" Y="0" /><node X="0" Y="10" />`
	tests := []struct {
		name   string
		change func(fstest.MapFS)
		want   string
	}{
		{"unknown NPC element", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="90000"><wings /></npc>`)
		}, "<wings>: unknown element"},
		{"unknown NPC attribute", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="90000" colour="red" />`)
		}, "unknown attribute colour"},
		{"unknown list element", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<monster id="90000" />`)
		}, "<monster>: unknown element"},
		{"NPC defined twice", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="20001" />`)
		}, "NPC 20001 is also defined in"},
		{"unknown race", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="90000"><race>MARTIAN</race></npc>`)
		}, `unknown race "MARTIAN"`},
		{"unknown attack type", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="90000"><stats><attack type="LASER" /></stats></npc>`)
		}, `unknown attack type "LASER"`},
		{"unknown NPC type", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="90000" type="Ghost" />`)
		}, `unknown NPC type "Ghost"`},
		{"invalid integer", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="9x" />`)
		}, `attribute id: invalid integer "9x"`},
		{"invalid boolean", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="90000"><status attackable="yes" /></npc>`)
		}, `attribute attackable: "yes" is not true or false`},
		{"invalid zone ID", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" id="0x10" type="PeaceZone" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`</zone>`)
		}, `attribute id: "0x10" is not an int`},
		{"walk without ground", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="90000"><stats><speed><walk swim="5" /></speed></stats></npc>`)
		}, "missing attribute ground"},
		{"group in a spoil list", func(m fstest.MapFS) {
			m[npcFile] = xmlList("", `<npc id="90000"><dropLists><spoil><group chance="1" /></spoil></dropLists></npc>`)
		}, "<group>: unknown element"},
		{"broken XML", func(m fstest.MapFS) {
			m[npcFile] = &fstest.MapFile{Data: []byte(`<list><npc id="90000">`)}
		}, "parse " + npcFile},
		{"no NPC files", func(m fstest.MapFS) {
			for p := range m {
				if strings.HasPrefix(p, npcXMLDir+"/") && !strings.Contains(p, "/custom/") {
					delete(m, p)
				}
			}
		}, "no NPC files found"},

		{"polygon of 2 nodes", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn zone="bad"><territory minZ="0" maxZ="10"><node x="0" y="0" /><node x="1" y="1" /></territory><npc id="20001" /></spawn>`)
		}, "an NPoly needs at least 3 nodes (got 2)"},
		{"unknown territory shape", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn zone="bad"><territory shape="Star" minZ="0" maxZ="10">`+triangle+`</territory><npc id="20001" /></spawn>`)
		}, `unknown shape "Star"`},
		{"territory defined twice", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn zone="gludio32_1725_01"><territory minZ="0" maxZ="10">`+triangle+`</territory><npc id="20001" /></spawn>`)
		}, `territory "gludio32_1725_01" is also defined in`},
		{"banned area before its territory", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn zone="bad"><banned_territory minZ="0" maxZ="10">`+triangle+`</banned_territory><npc id="20001" /></spawn>`)
		}, `banned_territory of "bad" before its territory`},
		{"AIData after NPCs", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn name="s"><npc id="20001" x="1" y="1" z="1" /><AIData><aggroRange>5</aggroRange></AIData></spawn>`)
		}, "AIData after NPC entries"},
		{"x without y", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn><npc id="20001" x="1" z="1" /></spawn>`)
		}, "x, y and z go together"},
		{"unknown period of day", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn><npc id="20001" x="1" y="1" z="1" periodOfDay="dusk" /></spawn>`)
		}, `unknown periodOfDay "dusk"`},
		{"NPC without a place", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn><npc id="20001" /></spawn>`)
		}, "NPC 20001 has neither coordinates nor a zone"},
		{"spawn list without enabled", func(m fstest.MapFS) {
			m[spawnFile] = xmlList("", `<spawn><npc id="20001" x="1" y="1" z="1" /></spawn>`)
		}, "missing attribute enabled"},
		{"unknown territory", func(m fstest.MapFS) {
			m[spawnFile] = xmlList(`enabled="true"`, `<spawn zone="nowhere"><npc id="20001" /></spawn>`)
		}, `territory "nowhere" does not exist`},

		{"list twice in a file", func(m fstest.MapFS) {
			m[teleFile] = xmlList("", `<npc id="30099"><teleport type="NORMAL" /><teleport type="NORMAL" /></npc>`)
		}, `duplicate teleport list "NORMAL"`},
		{"list in two files", func(m fstest.MapFS) {
			m[teleFile] = xmlList("", `<npc id="30006"><teleport type="NORMAL" /></npc>`)
		}, `teleport list "NORMAL" is also defined in town/30006.xml`},
		{"list without type", func(m fstest.MapFS) {
			m[teleFile] = xmlList("", `<npc id="30099"><teleport /></npc>`)
		}, "missing attribute type"},
		{"bad castle ID", func(m fstest.MapFS) {
			m[teleFile] = xmlList("", `<npc id="30099"><teleport type="OTHER"><location x="1" y="1" z="1" castleId="1;x" /></teleport></npc>`)
		}, `castleId: invalid integer "x"`},
		{"unknown list type", func(m fstest.MapFS) {
			m[teleFile] = xmlList("", `<npc id="30099"><teleport type="VIP" /></npc>`)
		}, `unknown type "VIP"`},

		{"unknown zone type", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="FooZone" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`</zone>`)
		}, `unknown zone type "FooZone"`},
		{"abstract zone type", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="ResidenceZone" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`</zone>`)
		}, `unknown zone type "ResidenceZone"`},
		{"zone ID used twice", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" id="11022" type="PeaceZone" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`</zone>`)
		}, "zone ID 11022 is also used in custom_town.xml"},
		{"spawn point on a water zone", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="WaterZone" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`<spawn X="1" Y="2" Z="3" /></zone>`)
		}, "spawn point on a WaterZone, which Mobius ignores"},
		{"challenger point on a castle zone", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="ResidenceTeleportZone" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`<spawn X="1" Y="2" Z="3" type="challenger" /></zone>`)
		}, `unknown spawn point type "challenger" for a ResidenceTeleportZone`},
		{"race point on a peace zone", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="PeaceZone" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`<race name="HUMAN" point="orc_town" /></zone>`)
		}, "race point on a PeaceZone, which Mobius ignores"},
		{"unknown race of a respawn zone", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="RespawnZone" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`<race name="MARTIAN" point="orc_town" /></zone>`)
		}, `unknown race "MARTIAN"`},
		{"spawn territory zone", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="NpcSpawnTerritory" shape="NPoly" minZ="0" maxZ="1">`+zoneTriangle+`</zone>`)
		}, "NpcSpawnTerritory zones are not supported"},
		{"cuboid of 3 nodes", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="PeaceZone" shape="Cuboid" minZ="0" maxZ="1">`+zoneTriangle+`</zone>`)
		}, "a Cuboid needs exactly 2 nodes (got 3)"},
		{"cylinder without rad", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="PeaceZone" shape="Cylinder" minZ="0" maxZ="1"><node X="0" Y="0" /></zone>`)
		}, "a Cylinder needs 1 node and a positive rad"},
		{"zone without shape", func(m fstest.MapFS) {
			m[zoneFile] = xmlList(`enabled="true"`, `<zone name="x" type="PeaceZone" minZ="0" maxZ="1">`+zoneTriangle+`</zone>`)
		}, "missing attribute shape"},

		{"region defined twice", func(m fstest.MapFS) {
			m[regionFile] = xmlList(`enabled="true"`, `<region name="orc_town" town="Orc Town" castle="9" locId="920" bbs="9"><respawnPoint X="1" Y="1" Z="1" /></region>`)
		}, `region "orc_town" is also defined in orc_town.xml`},
		{"unknown region element", func(m fstest.MapFS) {
			m[regionFile] = xmlList(`enabled="true"`, `<region name="x" town="X" castle="0" locId="1" bbs="0"><airport /></region>`)
		}, "<airport>: unknown element"},
		{"region without town", func(m fstest.MapFS) {
			m[regionFile] = xmlList(`enabled="true"`, `<region name="x" castle="0" locId="1" bbs="0"><respawnPoint X="1" Y="1" Z="1" /></region>`)
		}, "missing attribute town"},

		{"Java class in the wrong file", func(m fstest.MapFS) {
			m[instanceFile] = &fstest.MapFile{Data: []byte("public class Bar extends Npc\n{\n}\n")}
		}, "declares class Bar, want Foo"},
		{"no NPC classes", func(m fstest.MapFS) {
			for p := range m {
				if strings.HasPrefix(p, npcTypeJavaDir+"/") {
					delete(m, p)
				}
			}
			// A file without a class (an interface) is skipped.
			m[npcTypeJavaDir+"/Named.java"] = &fstest.MapFile{Data: []byte("public interface Named\n{\n}\n")}
		}, "no classes found"},
		{"no NPC class folder", func(m fstest.MapFS) {
			for p := range m {
				if strings.HasPrefix(p, npcTypeJavaDir+"/") {
					delete(m, p)
				}
			}
		}, "read " + npcTypeJavaDir},
	}
	for _, tt := range tests {
		m := worldSampleFS(t)
		tt.change(m)
		_, _, err := runWorld(m)
		if err == nil {
			t.Errorf("%s: genWorld succeeded", tt.name)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}

	// Rules checked by internal/data keep its sentinel.
	m := worldSampleFS(t)
	m[npcFile] = xmlList("", `<npc id="90000" type="Ghost" />`)
	if _, _, err := runWorld(m); !errors.Is(err, data.ErrInvalid) {
		t.Errorf("unknown NPC type: genWorld = %v, want data.ErrInvalid", err)
	}
}

// Mobius keeps the last value of a parameter given twice (the real data
// does that for NPCs 20997 and 21000).
func TestWorldRepeatedParameter(t *testing.T) {
	m := worldSampleFS(t)
	m[npcXMLDir+"/90000-90099.xml"] = xmlList("", `<npc id="90000"><parameters>`+
		`<skill name="DDMagic" id="4034" level="5" /><param name="DDMagic" value="1" /><skill name="DDMagic" id="4001" level="5" />`+
		`</parameters></npc>`)
	files, log, err := runWorld(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log, "NPC 90000: DDMagic; NPC 90000: DDMagic") {
		t.Errorf("log does not report the repeated parameter:\n%s", log)
	}
	want := `"parameters":{"skills":{"DDMagic":{"id":4001,"level":5}}}`
	if !bytes.Contains(files[data.NpcsFile], []byte(want)) {
		t.Errorf("%s does not contain %s", data.NpcsFile, want)
	}
}

func TestHTMLCopy(t *testing.T) {
	var log bytes.Buffer
	out := newOutput()
	if err := genHTML(options{src: os.DirFS(worldSample), out: out, log: &log}); err != nil {
		t.Fatalf("genHTML: %v", err)
	}
	want := map[string]string{ // output path -> source path
		"html/npcdefault.htm":                                   htmlSrcDir + "/npcdefault.htm",
		"html/teleporter/30006.htm":                             htmlSrcDir + "/teleporter/30006.htm",
		"html/admin/documentation.txt":                          htmlSrcDir + "/admin/documentation.txt",
		"html/scripts/quests/Q00001_LettersOfLove/30048-01.htm": scriptsSrcDir + "/quests/Q00001_LettersOfLove/30048-01.htm",
		"html/scripts/ai/others/Sample/sample.HTML":             scriptsSrcDir + "/ai/others/Sample/sample.HTML",
	}
	if got := slices.Sorted(maps.Keys(out.files)); !slices.Equal(got, slices.Sorted(maps.Keys(want))) {
		t.Errorf("copied %v, want %v", got, slices.Sorted(maps.Keys(want)))
	}
	for dst, src := range want {
		b, err := os.ReadFile(filepath.Join(worldSample, filepath.FromSlash(src)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out.files[dst], b) {
			t.Errorf("%s is not a copy of %s", dst, src)
		}
	}
	if want := "html: 3 pages from dist/game/data/html, 2 from dist/game/data/scripts"; !strings.Contains(log.String(), want) {
		t.Errorf("log %q does not mention %q", log.String(), want)
	}

	// Both folders must have pages.
	m := worldSampleFS(t)
	for p := range m {
		if strings.HasPrefix(p, scriptsSrcDir+"/") && isHTMLFile(p) {
			delete(m, p)
		}
	}
	if err := genHTML(options{src: m, out: newOutput(), log: &log}); err == nil || !strings.Contains(err.Error(), "want some in both") {
		t.Errorf("no script pages: genHTML = %v", err)
	}
	m = worldSampleFS(t)
	for p := range m {
		if strings.HasPrefix(p, htmlSrcDir+"/") {
			delete(m, p)
		}
	}
	if err := genHTML(options{src: m, out: newOutput(), log: &log}); err == nil || !strings.Contains(err.Error(), "copy "+htmlSrcDir) {
		t.Errorf("no html folder: genHTML = %v", err)
	}
}

// The world and html domains are registered and run through generate,
// which a plain "go run ./cmd/datagen" (make data) uses.
func TestWorldDomainsRegistered(t *testing.T) {
	ds, err := selectDomains(false, []string{"world", "html"})
	if err != nil || len(ds) != 2 {
		t.Fatalf("selectDomains(world, html) = %v, %v", ds, err)
	}
	files, err := generate(os.DirFS(worldSample), false, []string{"world", "html"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, name := range append(slices.Clone(worldOutputs), "html/npcdefault.htm", "html/scripts/quests/Q00001_LettersOfLove/30048-01.htm") {
		if _, ok := files[name]; !ok {
			t.Errorf("generate did not make %s", name)
		}
	}
	if len(files) != len(worldOutputs)+5 {
		t.Errorf("generate made %d files, want %d", len(files), len(worldOutputs)+5)
	}
}

func TestReadMobiusClasses(t *testing.T) {
	classes, err := readMobiusClasses(os.DirFS(worldSample), npcTypeJavaDir, npcBaseJavaDir)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]data.MobiusClass)
	for _, c := range classes {
		byName[c.Name] = c
	}
	if c := byName["RaidBoss"]; c.Parent != "Monster" || !c.Usable {
		t.Errorf("RaidBoss = %+v", c)
	}
	if c := byName["Npc"]; c.Parent != "Creature" || c.Usable {
		t.Errorf("Npc = %+v, want a base class that data cannot name", c)
	}
	if c := byName["Creature"]; !c.Abstract || c.Usable {
		t.Errorf("Creature = %+v", c)
	}
	for _, absent := range []string{"Ghost", "Player"} {
		if _, ok := byName[absent]; ok {
			t.Errorf("%s is in the list (commented out, or not an NPC ancestor)", absent)
		}
	}
	if !slices.IsSortedFunc(classes, func(a, b data.MobiusClass) int { return strings.Compare(a.Name, b.Name) }) {
		t.Error("classes are not sorted by name")
	}
}

func TestMxInt(t *testing.T) {
	good := map[string]int{"0": 0, "10": 10, "-5": -5, "+7": 7, "0x1F": 31, "0X1f": 31, "#ff": 255, "-#10": -16, "010": 8, " 12 ": 12, "2147483647": 2147483647}
	for in, want := range good {
		if got, err := mxInt(in); err != nil || got != want {
			t.Errorf("mxInt(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "-", "abc", "1.5", "08", "--1", "0x", "2147483648", "1_000"} {
		if got, err := mxInt(in); err == nil {
			t.Errorf("mxInt(%q) = %d, want an error", in, got)
		}
	}
}

func TestMxFloat(t *testing.T) {
	for in, want := range map[string]float64{"1": 1, "0.9": 0.9, "-3.5": -3.5, "29.5916164000214": 29.5916164000214, "1e3": 1000} {
		if got, err := mxFloat(in); err != nil || got != want {
			t.Errorf("mxFloat(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "NaN", "Infinity", "1,5"} {
		if _, err := mxFloat(in); err == nil {
			t.Errorf("mxFloat(%q) succeeded", in)
		}
	}
}

func TestJavaStringHash(t *testing.T) {
	for in, want := range map[string]int32{"": 0, "a": 97, "abc": 96354, "hello": 99162322, "Aa": 2112, "BB": 2112} {
		if got := javaStringHash(in); got != want {
			t.Errorf("javaStringHash(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestJavaHashMapOrder(t *testing.T) {
	// With 16 buckets: "hello" goes to bucket 11, "abc" to 3, and "Aa" and
	// "BB" (same hash) to 0, in insertion order.
	got, err := javaHashMapOrder([]string{"hello", "abc", "Aa", "BB"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{2, 3, 1, 0}; !slices.Equal(got, want) {
		t.Errorf("javaHashMapOrder = %v, want %v", got, want)
	}

	// Strings made of "Aa" and "BB" blocks all share a hash. Java grows
	// the table at the 9th and 10th, then makes a tree at the 11th, whose
	// order datagen does not reproduce.
	var same []string
	for mask := range 16 {
		var b strings.Builder
		for bit := range 4 {
			if mask&(1<<bit) != 0 {
				b.WriteString("BB")
			} else {
				b.WriteString("Aa")
			}
		}
		same = append(same, b.String())
	}
	if order, err := javaHashMapOrder(same[:10]); err != nil || !slices.Equal(order, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}) {
		t.Errorf("10 colliding keys: %v, %v; want insertion order", order, err)
	}
	if _, err := javaHashMapOrder(same); err == nil {
		t.Error("16 colliding keys: no error")
	}
}

// The committed world data must be what datagen makes from the local
// Mobius clone. Skipped when the clone is not there (it is not in git).
func TestCommittedWorldMatchesMobius(t *testing.T) {
	if _, err := os.Stat(worldMobius); err != nil {
		t.Skipf("no Mobius clone at %s", worldMobius)
	}
	files, _, err := runWorld(os.DirFS(worldMobius))
	if err != nil {
		t.Fatalf("genWorld: %v", err)
	}
	for name, b := range files {
		want, err := os.ReadFile(filepath.Join(worldRepoData, filepath.FromSlash(name)))
		if err != nil {
			t.Errorf("%v; run: go run ./cmd/datagen world", err)
			continue
		}
		if !bytes.Equal(b, want) {
			t.Errorf("data/%s is out of date; run: go run ./cmd/datagen world", name)
		}
	}
}
