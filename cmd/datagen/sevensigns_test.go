package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

func TestJavaArrayInit(t *testing.T) {
	src := stripJavaComments(`class X {
	static final int[][][] T =
	{
		{
			/* first */
			{ -1, 2, 0x10, -1, 5 },
			{ 3, 4, 5, 6, 7 }, // trailing comma
		},
		{ { 8, 9, 10, 11, 12 } }
	};
	static final int[][] FLAT = { { 1, 2, 3, 4 } };
}`)
	tree, err := javaArrayInit(src, "T")
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.list) != 2 || len(tree.list[0].list) != 2 {
		t.Fatalf("tree %+v", tree)
	}
	got, err := festivalSpawnList(tree.list[0], true)
	want := []data.FestivalSpawn{{X: -1, Y: 2, Z: 16, Heading: -1, NpcID: 5}, {X: 3, Y: 4, Z: 5, Heading: 6, NpcID: 7}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("spawns = %+v, %v", got, err)
	}
	flat, err := javaArrayInit(src, "FLAT")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := festivalSpawnList(flat, true); err == nil {
		t.Error("4 ints where 5 are wanted is no error")
	}
	if s, err := festivalSpawnList(flat, false); err != nil || s[0] != (data.FestivalSpawn{X: 1, Y: 2, Z: 3, Heading: 4}) {
		t.Errorf("player spawns = %+v, %v", s, err)
	}
	for _, bad := range []string{"NONE", "BROKEN"} {
		if _, err := javaArrayInit(" BROKEN = { 1 2 };", bad); err == nil {
			t.Errorf("%s: no error", bad)
		}
	}
}

// mobiusSevenSigns runs the domain on the local Mobius clone (skipped
// without it) and returns the loaded data.
func mobiusSevenSigns(t *testing.T) (*data.SevenSigns, string) {
	t.Helper()
	dir := mobiusDir
	if _, err := os.Stat(dir); err != nil {
		// A git worktree in .worktrees/<name> finds the clone of the main
		// checkout.
		dir = "../../../../L2J_Mobius/L2J_Mobius_CT_0_Interlude"
		if _, err := os.Stat(dir); err != nil {
			t.Skipf("no Mobius clone at %s", mobiusDir)
		}
	}
	var log bytes.Buffer
	files, err := generate(os.DirFS(dir), false, []string{"sevensigns"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	d, err := data.LoadSevenSignsFS(fstest.MapFS{data.SevenSignsFile: {Data: files[data.SevenSignsFile]}})
	if err != nil {
		t.Fatal(err)
	}
	return d, log.String()
}

func TestSevenSignsFromMobius(t *testing.T) {
	d, log := mobiusSevenSigns(t)
	// 2 rooms of type 0 (the waiting room and the whole rift), 9 in each of
	// the six areas.
	if len(d.Rift) != 56 || len(d.RiftRooms(0)) != 2 || len(d.RiftRooms(1)) != 9 || len(d.RiftRooms(6)) != 9 {
		t.Errorf("%d rooms; type 0: %d, 1: %d, 6: %d", len(d.Rift), len(d.RiftRooms(0)), len(d.RiftRooms(1)), len(d.RiftRooms(6)))
	}
	// (1, 1, -112368, -111584, -182832, -182064, -6816, -6240, -111976, -182448, -6752, 0)
	r, ok := d.RiftRoom(1, 1)
	want := data.RiftRoom{Type: 1, Room: 1, XMin: -112368, XMax: -111584, YMin: -182832, YMax: -182064, ZMin: -6816, ZMax: -6240,
		TeleX: -111976, TeleY: -182448, TeleZ: -6752,
		Spawns: []data.RiftSpawn{{NpcID: 21659, Count: 2, Delay: 90}, {NpcID: 21661, Count: 4, Delay: 80}, {NpcID: 21660, Count: 2, Delay: 90}}}
	if !ok || !reflect.DeepEqual(*r, want) {
		t.Errorf("room 1/1 = %+v", r)
	}
	if b, ok := d.RiftRoom(6, 9); !ok || !b.Boss || len(b.Spawns) == 0 {
		t.Errorf("room 6/9 = %+v", b)
	}
	if w, ok := d.RiftRoom(0, 0); !ok || w.TeleX != -114790 || len(w.Spawns) != 0 {
		t.Errorf("waiting room = %+v", w)
	}
	f := d.Festival
	if f.DawnWitch[0] != (data.FestivalSpawn{X: -79183, Y: 113052, Z: -4891, Heading: 0, NpcID: 31132}) {
		t.Errorf("dawn witch 0 = %+v", f.DawnWitch[0])
	}
	if f.DuskPlayer[4] != (data.FestivalSpawn{X: -79954, Y: 84697, Z: -5151}) {
		t.Errorf("dusk player 4 = %+v", f.DuskPlayer[4])
	}
	if len(f.DawnChest[0]) != 12 || f.DawnChest[0][0] != (data.FestivalSpawn{X: -78999, Y: 112957, Z: -4927, Heading: -1, NpcID: 18109}) {
		t.Errorf("dawn chests 0 = %d, %+v", len(f.DawnChest[0]), f.DawnChest[0][0])
	}
	if f.DawnPrimary[0][0] != (data.FestivalSpawn{X: -78537, Y: 113839, Z: -4895, Heading: -1, NpcID: 18009}) {
		t.Errorf("dawn primary 0/0 = %+v", f.DawnPrimary[0][0])
	}
	// The Seven Signs groups of random_spawn: 1 merchant, 1 blacksmith, the
	// orators, preachers, crests, Lilith and Anakim; not the quest NPCs.
	merch := d.AutoSpawnsOf(31113)
	if len(merch) != 1 || merch[0].GroupID != 11 || len(merch[0].Locations) != 8 || !merch[0].BroadcastSpawn || !merch[0].RandomSpawn ||
		merch[0].RespawnDelay != -1 || merch[0].Locations[0] != (data.AutoSpawnLoc{X: -52172, Y: 78884, Z: -4741, Heading: -1}) {
		t.Errorf("merchant of mammon = %+v", merch)
	}
	if l := d.AutoSpawnsOf(25283); len(l) != 1 || l[0].RespawnDelay != 86400 || l[0].DespawnDelay != 0 || l[0].RandomSpawn {
		t.Errorf("lilith = %+v", l)
	}
	if len(d.AutoSpawnsOf(32014)) != 0 {
		t.Error("a quest NPC is among the Seven Signs spawns")
	}
	if !strings.Contains(log, "sevensigns: 56 rift rooms") {
		t.Errorf("log %q", log)
	}
}

// sevenSignsSample is a cut-down Mobius with the sevensigns sources: the
// waiting room and rooms 1 and 9 of each area, one spawn list, the festival
// tables with one row each, and two random spawn groups (one of a quest
// NPC).
func sevenSignsSample() fstest.MapFS {
	var sql strings.Builder
	sql.WriteString("DROP TABLE IF EXISTS `dimensional_rift`;\nCREATE TABLE `dimensional_rift` (\n" +
		"  `type` tinyint(1) unsigned NOT NULL,\n  `room_id` tinyint(1) unsigned NOT NULL,\n" +
		"  `xMin` mediumint(6) NOT NULL,\n  `xMax` mediumint(6) NOT NULL,\n  `yMin` mediumint(6) NOT NULL,\n" +
		"  `yMax` mediumint(6) NOT NULL,\n  `zMin` mediumint(6) NOT NULL,\n  `zMax` mediumint(6) NOT NULL,\n" +
		"  `xT` mediumint(6) NOT NULL,\n  `yT` mediumint(6) NOT NULL,\n  `zT` mediumint(6) NOT NULL,\n" +
		"  `boss` tinyint(1) unsigned NOT NULL DEFAULT '0',\n  PRIMARY KEY (`type`,`room_id`)\n) DEFAULT CHARSET=latin1;\n\n" +
		"INSERT INTO `dimensional_rift` VALUES\n" +
		"(0, 0, -115440, -114160, -182160, -178048, -6816, -6240, -114790, -180576, -6752, 0),\t-- waiting room\n" +
		"(0, 1, -120576, -109024, -187328, -172880, -6816, -6240, 0, 0, 0, 0)")
	for typ := 6; typ >= 1; typ-- { // out of order: datagen sorts
		fmt.Fprintf(&sql, ",\n(%d, 9, -110688, -109136, -181968, -181072, -6816, -6240, -109912, -181520, -6752, 1)", typ)
		fmt.Fprintf(&sql, ",\n(%d, 1, -112368, -111584, -182832, -182064, -6816, -6240, -111976, -182448, -6752, 0)", typ)
	}
	sql.WriteString(";\n")
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<rift xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/DimensionalRift.xsd">
	<area type="1">
		<room id="1">
			<!-- Otherworldy Invader Soldier -->
			<spawn mobId="21659" count="2" delay="90" />
			<spawn mobId="21661" count="4" delay="80" />
		</room>
	</area>
	<area type="6">
		<room id="9">
			<spawn mobId="25339" count="1" delay="300" />
		</room>
	</area>
</rift>
`
	var java strings.Builder
	java.WriteString("public class SevenSignsFestival\n{\n")
	for _, name := range []string{"DAWN_PLAYER", "DUSK_PLAYER"} {
		fmt.Fprintf(&java, "\tpublic static final int[][] FESTIVAL_%s_SPAWNS =\n\t{\n", name)
		for i := range 5 {
			fmt.Fprintf(&java, "\t\t{ -79187, %d, -4895, 0 }, // %d\n", 113186+i, i)
		}
		java.WriteString("\t};\n")
	}
	for _, name := range []string{"DAWN_WITCH", "DUSK_WITCH"} {
		fmt.Fprintf(&java, "\tprotected static final int[][] FESTIVAL_%s_SPAWNS =\n\t{\n", name)
		for i := range 5 {
			fmt.Fprintf(&java, "\t\t{ -79183, 113052, -4891, 0, %d },\n", 31132+i)
		}
		java.WriteString("\t};\n")
	}
	for _, name := range []string{"DAWN_PRIMARY", "DUSK_PRIMARY", "DAWN_SECONDARY", "DUSK_SECONDARY", "DAWN_CHEST", "DUSK_CHEST"} {
		fmt.Fprintf(&java, "\tprotected static final int[][][] FESTIVAL_%s_SPAWNS =\n\t{\n", name)
		for i := range 5 {
			fmt.Fprintf(&java, "\t\t{\n\t\t\t/* Level %d */\n\t\t\t{ -78999, 112957, -4927, -1, 18109 },\n\t\t\t{ -79153, 112873, -4927, -1, %d },\n\t\t},\n", i, 18109+i)
		}
		java.WriteString("\t};\n")
	}
	java.WriteString("}\n")
	spawn := "DROP TABLE IF EXISTS `random_spawn`;\nCREATE TABLE `random_spawn` (\n" +
		"  `groupId` tinyint(3) unsigned NOT NULL,\n  `npcId` smallint(5) unsigned NOT NULL,\n" +
		"  `count` tinyint(1) unsigned NOT NULL DEFAULT '1',\n  `initialDelay` int(8) NOT NULL DEFAULT '-1',\n" +
		"  `respawnDelay` int(8) NOT NULL DEFAULT '-1',\n  `despawnDelay` int(8) NOT NULL DEFAULT '-1',\n" +
		"  `broadcastSpawn` enum('true','false') NOT NULL DEFAULT 'false',\n  `randomSpawn` enum('true','false') NOT NULL DEFAULT 'true',\n" +
		"  PRIMARY KEY (`groupId`)\n) DEFAULT CHARSET=utf8mb4;\n\nINSERT INTO `random_spawn` VALUES\n" +
		"(110,25283,1,-1,86400,0,'false','false'), -- Lilith (80)\n" +
		"(11,31113,1,-1,-1,-1,'true','true'), -- Merchant of Mammon\n" +
		"(133,32014,1,-1,3600000,0,'false','true'); -- Ivan (Runaway Youth quest)\n"
	loc := "DROP TABLE IF EXISTS `random_spawn_loc`;\nCREATE TABLE `random_spawn_loc` (\n" +
		"  `groupId` tinyint(3) unsigned NOT NULL,\n  `x` mediumint(6) NOT NULL,\n  `y` mediumint(6) NOT NULL,\n" +
		"  `z` mediumint(6) NOT NULL,\n  `heading` mediumint(6) NOT NULL DEFAULT '0',\n" +
		"  PRIMARY KEY (`groupId`,`x`,`y`,`z`,`heading`)\n) DEFAULT CHARSET=latin1;\n\nINSERT INTO `random_spawn_loc` VALUES\n" +
		"-- Merchant of Mammon\n(11,-52172,78884,-4741,-1), -- Necropolis of Devotion\n(11,-41350,209876,-5087,-1),\n" +
		"(110,185000,-13000,-5480,0),\n(133,1,2,3,0);\n"
	return fstest.MapFS{
		dimensionalRiftSQL: file(sql.String()),
		dimensionalRiftXML: file(xml),
		festivalJava:       file(java.String()),
		randomSpawnSQL:     file(spawn),
		randomSpawnLocSQL:  file(loc),
	}
}

func TestSevenSignsSample(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(sevenSignsSample(), false, []string{"sevensigns"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	d, err := data.LoadSevenSignsFS(fstest.MapFS{data.SevenSignsFile: {Data: files[data.SevenSignsFile]}})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Rift) != 14 || d.Rift[0].Type != 0 || !reflect.DeepEqual(d.Rift[2], data.RiftRoom{Type: 1, Room: 1, XMin: -112368, XMax: -111584,
		YMin: -182832, YMax: -182064, ZMin: -6816, ZMax: -6240, TeleX: -111976, TeleY: -182448, TeleZ: -6752,
		Spawns: d.Rift[2].Spawns}) {
		t.Errorf("rooms = %+v", d.Rift)
	}
	if r, ok := d.RiftRoom(1, 1); !ok || !reflect.DeepEqual(r.Spawns, []data.RiftSpawn{{NpcID: 21659, Count: 2, Delay: 90}, {NpcID: 21661, Count: 4, Delay: 80}}) {
		t.Errorf("room 1/1 = %+v", r)
	}
	if r, ok := d.RiftRoom(6, 9); !ok || !r.Boss || len(r.Spawns) != 1 {
		t.Errorf("room 6/9 = %+v", r)
	}
	if rooms := d.RiftRooms(3); len(rooms) != 2 || rooms[0].Room != 1 || rooms[1].Room != 9 {
		t.Errorf("RiftRooms(3) = %+v", rooms)
	}
	f := d.Festival
	if f.DuskWitch[4] != (data.FestivalSpawn{X: -79183, Y: 113052, Z: -4891, NpcID: 31136}) || f.DawnPlayer[2].Y != 113188 || f.DawnPlayer[2].NpcID != 0 {
		t.Errorf("witch 4 %+v, player 2 %+v", f.DuskWitch[4], f.DawnPlayer[2])
	}
	if len(f.DuskChest) != 5 || len(f.DuskChest[3]) != 2 || f.DuskChest[3][1] != (data.FestivalSpawn{X: -79153, Y: 112873, Z: -4927, Heading: -1, NpcID: 18112}) {
		t.Errorf("dusk chests = %+v", f.DuskChest)
	}
	if len(d.AutoSpawns) != 2 || d.AutoSpawns[0].GroupID != 11 || len(d.AutoSpawns[0].Locations) != 2 || !d.AutoSpawns[0].BroadcastSpawn ||
		d.AutoSpawns[1].NpcID != 25283 || d.AutoSpawns[1].RandomSpawn || d.AutoSpawns[1].RespawnDelay != 86400 {
		t.Errorf("auto spawns = %+v", d.AutoSpawns)
	}
	if !strings.Contains(log.String(), "sevensigns: 14 rift rooms with 3 spawn lines, 20 festival chests, 2 auto spawn groups") {
		t.Errorf("log %q", log.String())
	}
}

func TestSevenSignsErrors(t *testing.T) {
	for _, c := range []struct {
		name, file, old, new, want string
	}{
		{"unknown room", dimensionalRiftXML, `<room id="1">`, `<room id="12">`, "room 1/12 is not in"},
		{"unknown attribute", dimensionalRiftXML, `delay="90"`, `delay="90" boss="1"`, "boss"},
		{"no festival table", festivalJava, "FESTIVAL_DUSK_CHEST_SPAWNS", "FESTIVAL_DUSK_BOXES", "no array FESTIVAL_DUSK_CHEST_SPAWNS"},
		{"a short witch row", festivalJava, "{ -79183, 113052, -4891, 0, 31132 }", "{ -79183, 113052, -4891, 0 }", "FESTIVAL_DAWN_WITCH_SPAWNS: row 0: want 5 ints"},
		{"no location", randomSpawnLocSQL, "(110,185000,-13000,-5480,0),\n", "", "group 110 has no location"},
		{"a rift type missing", dimensionalRiftSQL, ",\n(4, 9,", ",\n(9, 9,", "bad type or box"},
		{"no source", randomSpawnSQL, "CREATE TABLE `random_spawn`", "CREATE TABLE `other`", "creates table other"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := sevenSignsSample()
			text := string(m[c.file].Data)
			if !strings.Contains(text, c.old) {
				t.Fatalf("the sample has no %q", c.old)
			}
			m[c.file] = file(strings.Replace(text, c.old, c.new, 1))
			_, err := generate(m, false, []string{"sevensigns"}, io.Discard)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %v, want it to mention %q", err, c.want)
			}
		})
	}
}

// The committed Seven Signs data is what datagen makes from Mobius.
func TestCommittedSevenSignsMatchesMobius(t *testing.T) { committed(t, repoData, false, "sevensigns") }
