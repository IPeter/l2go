package main

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

const sampleDoorXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/Doors.xsd">
	<door id="23180002" name="hubris_3F_002" open_method="0" height="355" baseHpMax="150000" hp_showable="false" basePDef="476" baseMDef="383" pos="113771,16849,-1979" node1="113728,16788" node2="113833,16893" node3="113815,16910" node4="113710,16806" nodeZ="-2156" default_status="close" master_open_event="act_open" master_close_event="act_close" />
	<door id="23180001" name="hubris_3F_001" open_method="2" close_time="120" open_time="120" random_time="30" height="355" baseHpMax="150000" hp_showable="false" basePDef="476" baseMDef="383" pos="113876,16952,-1979" node1="113832,16890" node2="113937,16996" node3="113920,17013" node4="113814,16908" nodeZ="-2156" default_status="close" child_id_event="23180002" />
	<door id="17220001" name="gludin_agit_003_003" open_method="1" height="450" baseHpMax="316500" basePDef="644" baseMDef="518" pos="-84495,155208,-3150" node1="-84494,155208" node2="-84494,155246" node3="-84497,155246" node4="-84496,155208" nodeZ="-3192" clanhall_id="28" />
	<door id="19160010" name="sanctuary_gate01" open_method="12" level="3" close_time="10" height="512" pos="-16118,-44160,-10384" node1="-16652,-44231" node2="-16080,-44235" node3="-16080,-44081" node4="-16654,-44082" nodeZ="-10877" group="sanctuary_parent" default_status="open" is_wall="true" />
</list>
`

const sampleStaticObjectXML = `<?xml version="1.0" encoding="UTF-8"?>
<list>
	<object id="17250001" name="town_map_talking_1" type="0" x="-82873" y="244808" z="-3717" texture="town_map_talking_t00" map_x="365" map_y="152" />
	<object id="17220015" name="pvp_signboard_gl_1" type="2" x="-88696" y="142315" z="-3621" />
</list>
`

const sampleFenceXML = `<?xml version="1.0" encoding="UTF-8"?>
<list>
	<fence name="demo" x="-114552" y="-251256" z="-2992" width="100" length="100" height="3" state="CLOSED" />
	<fence name="other" x="1" y="2" z="3" width="10" length="20" height="1" />
</list>
`

func objectSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[doorXML] = file(sampleDoorXML)
	m[staticObjectXML] = file(sampleStaticObjectXML)
	m[fenceXML] = file(sampleFenceXML)
	return m
}

func TestObjects(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(objectSample(t), false, []string{"objects"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	doors := string(files[data.DoorsFile])
	for _, want := range []string{
		// Sorted by ID; defaults filled in.
		"[\n" + `{"id":17220001,"name":"gludin_agit_003_003","x":-84495,"y":155208,"z":-3150,"nodes":[{"x":-84494,"y":155208},{"x":-84494,"y":155246},{"x":-84497,"y":155246},{"x":-84496,"y":155208}],"nodeZ":-3192,"height":450,"hp":316500,"pDef":644,"mDef":518,"openMethod":1,"showHp":true,"targetable":true,"checkCollision":true,"clanHallId":28},`,
		`{"id":19160010,"name":"sanctuary_gate01","x":-16118,"y":-44160,"z":-10384,"nodes":[{"x":-16652,"y":-44231},{"x":-16080,"y":-44235},{"x":-16080,"y":-44081},{"x":-16654,"y":-44082}],"nodeZ":-10877,"height":512,"hp":1,"pDef":0,"mDef":0,"level":3,"openMethod":12,"defaultOpen":true,"showHp":true,"wall":true,"group":"sanctuary_parent","targetable":true,"closeTime":10,"checkCollision":true},`,
		`"openMethod":2,"showHp":false,"childId":23180002,"targetable":true,"closeTime":120,"openTime":120,"randomTime":30,"checkCollision":true},`,
		`"showHp":false,"masterOpen":"act_open","masterClose":"act_close","targetable":true,"checkCollision":true}` + "\n]\n",
	} {
		if !strings.Contains(doors, want) {
			t.Errorf("%s does not contain\n%s\nhave\n%s", data.DoorsFile, want, doors)
		}
	}
	wantStatic := "[\n" +
		`{"id":17220015,"name":"pvp_signboard_gl_1","type":2,"x":-88696,"y":142315,"z":-3621},` + "\n" +
		`{"id":17250001,"name":"town_map_talking_1","type":0,"x":-82873,"y":244808,"z":-3717,"texture":"town_map_talking_t00","mapX":365,"mapY":152}` + "\n]\n"
	if got := string(files[data.StaticObjectsFile]); got != wantStatic {
		t.Errorf("%s =\n%s\nwant\n%s", data.StaticObjectsFile, got, wantStatic)
	}
	wantFences := "[\n" +
		`{"name":"demo","x":-114552,"y":-251256,"z":-2992,"width":100,"length":100,"height":3,"state":"CLOSED"},` + "\n" +
		`{"name":"other","x":1,"y":2,"z":3,"width":10,"length":20,"height":1,"state":"CLOSED"}` + "\n]\n"
	if got := string(files[data.FencesFile]); got != wantFences {
		t.Errorf("%s =\n%s\nwant\n%s", data.FencesFile, got, wantFences)
	}

	fsys := fstest.MapFS{}
	for _, name := range []string{data.DoorsFile, data.StaticObjectsFile, data.FencesFile} {
		fsys[name] = &fstest.MapFile{Data: files[name]}
	}
	objs, err := data.LoadObjectsFS(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := objs.Door(23180001); !ok || !d.OpensBy(data.DoorOpenByTime) || d.ChildID != 23180002 {
		t.Errorf("Door(23180001) = %+v, %v", d, ok)
	}
	if g := objs.DoorGroup("sanctuary_parent"); len(g) != 1 || g[0] != 19160010 {
		t.Errorf("DoorGroup(sanctuary_parent) = %v", g)
	}
	if s := "objects: 4 doors (1 open at start, 1 open by time, 1 groups), 2 static objects, 2 fences"; !strings.Contains(log.String(), s) {
		t.Errorf("log %q does not mention %q", log.String(), s)
	}
}

func TestObjectsErrors(t *testing.T) {
	edit := func(name, old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[doorXML] = file(sampleDoorXML)
			m[staticObjectXML] = file(sampleStaticObjectXML)
			m[fenceXML] = file(sampleFenceXML)
			src := string(m[name].Data)
			if !strings.Contains(src, old) {
				t.Fatalf("%s does not contain %q", name, old)
			}
			m[name] = file(strings.Replace(src, old, new, 1))
		}
	}
	door := func(old, new string) func(*testing.T, fstest.MapFS) { return edit(doorXML, old, new) }
	expectErrors(t, []string{"objects"}, []sampleCase{
		{"no doors", func(*testing.T, fstest.MapFS) {}, "read " + doorXML},
		{"twice", door(`id="17220001"`, `id="23180002"`), "door 23180002 is listed twice"},
		{"unknown attribute", door(`clanhall_id="28"`, `clanhall_id="28" baseSTR="5"`), "unknown attribute baseSTR"},
		{"bad pos", door(`pos="-84495,155208,-3150"`, `pos="-84495,155208"`), `attribute pos: "-84495,155208" does not have 3 values`},
		{"bad node", door(`node1="-84494,155208"`, `node1="-84494,x"`), `attribute node1: "x" is not an int`},
		{"no nodeZ", door(`nodeZ="-3192" `, ""), "missing attribute nodeZ"},
		{"bad status", door(`default_status="open"`, `default_status="opened"`), `"opened" is not open or close`},
		{"bad action", door(`master_open_event="act_open"`, `master_open_event="act_flip"`), `unknown action "act_flip"`},
		{"zero close time", door(`close_time="10"`, `close_time="0"`), "close_time: 0 is not positive"},
		{"timed without open time", door(`open_time="120" `, ""), "missing attribute open_time"},
		{"open time on untimed door", door(`clanhall_id="28"`, `clanhall_id="28" open_time="5"`), "open_time on a door that does not open by time"},
		{"missing child", door(`child_id_event="23180002"`, `child_id_event="23180009"`), "child door 23180009 does not exist"},
		{"bad open method", door(`open_method="12"`, `open_method="64"`), "unknown open method bits 64"},
		{"static twice", edit(staticObjectXML, `id="17220015"`, `id="17250001"`), "static object 17250001 is listed twice"},
		{"static type", edit(staticObjectXML, `type="2"`, `type="7"`), "unknown type 7"},
		{"fence state", edit(fenceXML, `state="CLOSED"`, `state="closed"`), `unknown fence state "closed"`},
		{"fence size", edit(fenceXML, `width="10"`, `width="0"`), "size 0x20x1 must be positive"},
	})
}

// The committed objects are what datagen makes from Mobius.
func TestCommittedObjectsMatchMobius(t *testing.T) { committed(t, repoData, false, "objects") }
