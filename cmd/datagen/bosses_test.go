package main

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/IPeter/l2go/internal/data"
)

const sampleRaidSQL = "CREATE TABLE IF NOT EXISTS `raidboss_spawnlist` (\n" +
	"  `boss_id` smallint(5) unsigned NOT NULL,\n" +
	"  `amount` tinyint(1) unsigned NOT NULL DEFAULT '1',\n" +
	"  `loc_x` mediumint(6) NOT NULL,\n" +
	"  `loc_y` mediumint(6) NOT NULL,\n" +
	"  `loc_z` mediumint(6) NOT NULL,\n" +
	"  `heading` mediumint(6) NOT NULL DEFAULT '0',\n" +
	"  `respawn_delay` mediumint(6) unsigned NOT NULL DEFAULT '129600', -- 36 Hours\n" +
	"  `respawn_random` mediumint(6) unsigned NOT NULL DEFAULT '86400', -- (24 Hours)\n" +
	"  `respawn_time` bigint(13) unsigned NOT NULL DEFAULT '0',\n" +
	"  `currentHp` decimal(8,0) DEFAULT NULL,\n" +
	"  `currentMp` decimal(8,0) DEFAULT NULL,\n" +
	"  PRIMARY KEY (`boss_id`)\n" +
	") DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;\n" +
	"\n" +
	"INSERT IGNORE INTO `raidboss_spawnlist` (`boss_id`,`loc_x`,`loc_y`,`loc_z`,`heading`,`respawn_delay`,`respawn_random`,`currentHp`,`currentMp`) VALUES\n" +
	"(25004,-94208,100240,-3520,0,129600,86400,168366,731), -- Turek Mercenary Captain (30)\n" +
	"-- (25283,184410,-10111,-5488,0,129600,86400,1639146,3793), -- Lilith (80) (Spawn by Seven Signs)\n" +
	"(25001,-54416,146480,-2887,0,129600,86400,95986,514); -- Greyclaw Kutus (23) {a;b}\n" +
	"INSERT INTO raidboss_spawnlist (boss_id, loc_x, loc_y, loc_z) VALUES (25007, 1, 2, 3);\n"

const sampleGrandBossSQL = "CREATE TABLE IF NOT EXISTS `grandboss_data` (\n" +
	"  `boss_id` smallint(5) unsigned NOT NULL,\n" +
	"  `loc_x` mediumint(6) NOT NULL,\n" +
	"  `loc_y` mediumint(6) NOT NULL,\n" +
	"  `loc_z` mediumint(6) NOT NULL,\n" +
	"  `heading` mediumint(6) NOT NULL DEFAULT '0',\n" +
	"  `respawn_time` bigint(13) unsigned NOT NULL DEFAULT '0',\n" +
	"  `currentHP` decimal(30,15) NOT NULL,\n" +
	"  `currentMP` decimal(30,15) NOT NULL,\n" +
	"  `status` tinyint(1) unsigned NOT NULL DEFAULT '0',\n" +
	"  PRIMARY KEY (`boss_id`)\n" +
	") DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;\n" +
	"\n" +
	"INSERT IGNORE INTO `grandboss_data` (`boss_id`,`loc_x`,`loc_y`,`loc_z`,`heading`,`currentHP`,`currentMP`) VALUES\n" +
	"(29020, 116033, 17447, 10107, -25348, 4068372, 39960), -- Baium (75)\n" +
	"(29001, -21610, 181594, -5734, 0, 229898.48, 667.776); -- Queen Ant (40)\n" +
	"\n" +
	"-- Dr. Chaos\n" +
	"INSERT IGNORE INTO grandboss_data VALUES (25512, 96320, -110912, -3328, 8191, 0, 0, 0, 0);\n"

const sampleBossZoneXML = `<?xml version="1.0" encoding="UTF-8"?>
<list enabled="true" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../xsd/zones.xsd">
	<zone name="Queen Ant Boss" id="12012" type="BossZone" shape="NPoly" minZ="-5947" maxZ="-5547">
		<stat name="default_enabled" val="false" />
		<node X="-21490" Y="178613" />
		<node X="-22258" Y="178961" />
		<node X="-22890" Y="182205" />
	</zone>
	<zone id="12015" name="Ice Fairy Sirra Boss" type="BossZone" shape="Cuboid" minZ="-4000" maxZ="-500">
		<stat name="oustX" val="115844" />
		<stat name="oustY" val="-125602" />
		<stat name="oustZ" val="-3352" />
		<stat name="InvadeTime" val="1800000" />
		<node X="101744" Y="-130016" />
		<node X="111376" Y="-130096" />
	</zone>
</list>
`

// sampleBossNpcXML holds the boss templates the boss sample names.
const sampleBossNpcXML = `<?xml version="1.0" encoding="UTF-8"?>
<list>
	<npc id="25001" level="23" type="RaidBoss" name="Greyclaw Kutus" />
	<npc id="25004" level="30" type="RaidBoss" name="Turek Mercenary Captain" />
	<npc id="25007" level="42" type="RaidBoss" name="Retreat Spider Cletu" />
	<npc id="25512" level="52" type="Monster" name="Gigantic Chaos Golem" />
	<npc id="29001" level="40" type="GrandBoss" name="Queen Ant" />
	<npc id="29020" level="75" type="GrandBoss" name="Baium" />
</list>
`

const bossNpcFile = npcXMLDir + "/25000-25099.xml"

func bossSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[raidSpawnSQL] = file(sampleRaidSQL)
	m[grandBossSQL] = file(sampleGrandBossSQL)
	m[bossZoneXML] = file(sampleBossZoneXML)
	m[bossNpcFile] = file(sampleBossNpcXML)
	return m
}

func TestBosses(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(bossSample(t), false, []string{"bosses"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		data.RaidSpawnsFile: "[\n" +
			`{"bossId":25001,"x":-54416,"y":146480,"z":-2887,"heading":0,"respawnDelay":129600,"respawnRandom":86400,"hp":95986,"mp":514},` + "\n" +
			`{"bossId":25004,"x":-94208,"y":100240,"z":-3520,"heading":0,"respawnDelay":129600,"respawnRandom":86400,"hp":168366,"mp":731},` + "\n" +
			// The column defaults; NULL HP and MP read as 0.
			`{"bossId":25007,"x":1,"y":2,"z":3,"heading":0,"respawnDelay":129600,"respawnRandom":86400,"hp":0,"mp":0}` + "\n]\n",
		data.GrandBossesFile: "[\n" +
			`{"bossId":25512,"x":96320,"y":-110912,"z":-3328,"heading":8191,"respawnTime":0,"hp":0,"mp":0,"status":0},` + "\n" +
			`{"bossId":29001,"x":-21610,"y":181594,"z":-5734,"heading":0,"respawnTime":0,"hp":229898.48,"mp":667.776,"status":0},` + "\n" +
			`{"bossId":29020,"x":116033,"y":17447,"z":10107,"heading":-25348,"respawnTime":0,"hp":4068372,"mp":39960,"status":0}` + "\n]\n",
		data.BossZonesFile: "[\n" +
			`{"zoneId":12012,"name":"Queen Ant Boss","enabled":false},` + "\n" +
			`{"zoneId":12015,"name":"Ice Fairy Sirra Boss","invadeTime":1800000,"oust":{"x":115844,"y":-125602,"z":-3352},"enabled":true}` + "\n]\n",
	} {
		if got := string(files[name]); got != want {
			t.Errorf("%s =\n%s\nwant\n%s", name, got, want)
		}
	}
	fsys := fstest.MapFS{}
	for _, name := range []string{data.RaidSpawnsFile, data.GrandBossesFile, data.BossZonesFile} {
		fsys[name] = &fstest.MapFile{Data: files[name]}
	}
	b, err := data.LoadBossesFS(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := b.RaidSpawn(25001); !ok || r.X != -54416 {
		t.Errorf("RaidSpawn(25001) = %+v, %v", r, ok)
	} else if lo, hi := r.RespawnWindow(); lo != 12*time.Hour || hi != 60*time.Hour {
		t.Errorf("RespawnWindow = %v, %v; want 12h, 60h", lo, hi)
	}
	if s := "bosses: 3 raid boss spawns, 3 grand boss seeds, 2 boss zones"; !strings.Contains(log.String(), s) {
		t.Errorf("log %q does not mention %q", log.String(), s)
	}
}

func TestBossesErrors(t *testing.T) {
	edit := func(name, old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			for k, v := range bossSample(t) {
				m[k] = v
			}
			src := string(m[name].Data)
			if !strings.Contains(src, old) {
				t.Fatalf("%s does not contain %q", name, old)
			}
			m[name] = file(strings.Replace(src, old, new, 1))
		}
	}
	raid := func(old, new string) func(*testing.T, fstest.MapFS) { return edit(raidSpawnSQL, old, new) }
	expectErrors(t, []string{"bosses"}, []sampleCase{
		{"no raid file", func(*testing.T, fstest.MapFS) {}, "read " + raidSpawnSQL},
		{"raid twice", raid("(25001,", "(25004,"), "boss 25004 is listed twice"},
		{"raid not an NPC", raid("(25007,", "(25008,"), "raid boss 25008 is not in the NPC data"},
		{"raid not a RaidBoss", raid("(25007,", "(25512,"), "raid boss 25512 (Gigantic Chaos Golem) is a Monster, not a RaidBoss"},
		{"raid amount", raid("DEFAULT '1'", "DEFAULT '2'"), "amount 2, want 1"},
		{"raid column", raid("(boss_id, loc_x", "(boss_id, loc_q"), "unknown column loc_q"},
		{"raid value count", raid("(25007, 1, 2, 3)", "(25007, 1, 2)"), "3 values for 4 columns"},
		{"raid bad value", raid("(25007, 1, 2, 3)", "(25007, 1, 2, x)"), "want a value, have x"},
		{"raid not int", raid("(25007, 1, 2, 3)", "(25007, 1, 2, 3.5)"), `column loc_z: "3.5" is not an integer`},
		{"raid no default", raid("(boss_id, loc_x, loc_y, loc_z) VALUES (25007, 1, 2, 3)", "(boss_id, loc_x, loc_y) VALUES (25007, 1, 2)"), "column loc_z has no value and no default"},
		{"raid other statement", raid("INSERT INTO raidboss_spawnlist", "DELETE FROM raidboss_spawnlist; INSERT INTO raidboss_spawnlist"), "unsupported statement DELETE"},
		{"grand boss not an NPC", edit(grandBossSQL, "(29020,", "(29021,"), "grand boss 29021 is not in the NPC data"},
		{"grand boss other table", edit(grandBossSQL, "INSERT IGNORE INTO grandboss_data VALUES", "INSERT IGNORE INTO grandboss_list VALUES"), "insert into grandboss_list, want grandboss_data"},
		{"zone type", edit(bossZoneXML, `id="12012" type="BossZone"`, `id="12012" type="ArenaZone"`), "zone type ArenaZone, want BossZone"},
		{"zone parameter", edit(bossZoneXML, `name="InvadeTime"`, `name="invadeTime"`), "unknown boss zone parameter invadeTime"},
		{"zone bad value", edit(bossZoneXML, `val="1800000"`, `val="30m"`), `InvadeTime: "30m" is not an int`},
	})
}

func TestBossZonesDisabled(t *testing.T) {
	m := bossSample(t)
	m[bossZoneXML] = file(strings.Replace(sampleBossZoneXML, `enabled="true"`, `enabled="false"`, 1))
	var log bytes.Buffer
	files, err := generate(m, false, []string{"bosses"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(files[data.BossZonesFile]); got != "[]\n" {
		t.Errorf("%s = %q, want no zones", data.BossZonesFile, got)
	}
	if s := "warning: boss zones: " + bossZoneXML + " is disabled"; !strings.Contains(log.String(), s) {
		t.Errorf("log %q does not mention %q", log.String(), s)
	}
}

func TestSQLTokens(t *testing.T) {
	toks, err := sqlTokens("INSERT `a b` ('it''s', 'x\\'y', -1.5e3, +2) # note\n/* multi\nline */ -- end\nNULL;")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tok := range toks {
		got = append(got, string(tok.kind)+":"+tok.text)
	}
	want := "w:INSERT q:a b p:( s:it's p:, s:x'y p:, n:-1.5e3 p:, n:+2 p:) w:NULL p:;"
	if strings.Join(got, " ") != want {
		t.Errorf("tokens = %s\nwant     %s", strings.Join(got, " "), want)
	}
	if last := toks[len(toks)-1]; last.line != 4 {
		t.Errorf("last token on line %d, want 4", last.line)
	}
	for _, bad := range []string{"'open", "`open", "/* open"} {
		if _, err := sqlTokens(bad); err == nil {
			t.Errorf("sqlTokens(%q) succeeded", bad)
		}
	}
}

// The committed boss data is what datagen makes from Mobius.
func TestCommittedBossesMatchMobius(t *testing.T) { committed(t, repoData, false, "bosses") }
