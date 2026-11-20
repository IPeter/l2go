package data

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func validBosses() BossData {
	return BossData{
		RaidSpawns: []RaidSpawn{
			{BossID: 25001, X: 1, Y: 2, Z: 3, RespawnDelay: 129600, RespawnRandom: 86400, HP: 100, MP: 10},
			{BossID: 25004, RespawnDelay: 100, RespawnRandom: 300},
		},
		GrandBosses: []GrandBoss{{BossID: 29001, HP: 10}, {BossID: 29020}},
		Zones:       []BossZone{{ZoneID: 12012, Enabled: false}, {ZoneID: 12015, InvadeTime: 1800000, Oust: &Location{X: 1, Y: 2, Z: 3}, Enabled: true}},
	}
}

func TestNewBosses(t *testing.T) {
	b, err := NewBosses(validBosses())
	if err != nil {
		t.Fatal(err)
	}
	r, ok := b.RaidSpawn(25001)
	if !ok || r.HP != 100 {
		t.Fatalf("RaidSpawn(25001) = %+v, %v", r, ok)
	}
	if lo, hi := r.RespawnWindow(); lo != 12*time.Hour || hi != 60*time.Hour {
		t.Errorf("RespawnWindow = %v, %v; want 12h, 60h", lo, hi)
	}
	// delay - random below 1 s is raised to 1 s.
	if r, _ := b.RaidSpawn(25004); r != nil {
		if lo, hi := r.RespawnWindow(); lo != time.Second || hi != 400*time.Second {
			t.Errorf("RespawnWindow = %v, %v; want 1s, 400s", lo, hi)
		}
	}
	if lo, hi := (&RaidSpawn{}).RespawnWindow(); lo != 0 || hi != 0 {
		t.Errorf("RespawnWindow without delay = %v, %v", lo, hi)
	}
	if _, ok := b.RaidSpawn(25002); ok {
		t.Error("RaidSpawn(25002) found")
	}
	if g, ok := b.GrandBoss(29001); !ok || g.HP != 10 {
		t.Errorf("GrandBoss(29001) = %+v, %v", g, ok)
	}
	if _, ok := b.GrandBoss(29002); ok {
		t.Error("GrandBoss(29002) found")
	}
	z, ok := b.Zone(12015)
	if !ok {
		t.Fatal("Zone(12015) not found")
	}
	if p, ok := z.OustPoint(); !ok || p != (Location{X: 1, Y: 2, Z: 3}) {
		t.Errorf("OustPoint = %v, %v", p, ok)
	}
	if z, _ := b.Zone(12012); z == nil {
		t.Error("Zone(12012) not found")
	} else if _, ok := z.OustPoint(); ok {
		t.Error("zone 12012 has an oust point")
	}
	// Mobius uses the oust point only when no coordinate is 0.
	if _, ok := (&BossZone{Oust: &Location{X: 1, Y: 0, Z: 3}}).OustPoint(); ok {
		t.Error("an oust point with a zero coordinate is used")
	}
}

func TestNewBossesErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(d *BossData)
		want   string
	}{
		{"raid order", func(d *BossData) { d.RaidSpawns[1].BossID = 25001 }, "raid boss 25001: boss IDs are not sorted and unique"},
		{"raid ID", func(d *BossData) { d.RaidSpawns[0].BossID = 0 }, "raid boss 0: ID must be positive"},
		{"raid delay", func(d *BossData) { d.RaidSpawns[0].RespawnRandom = -1 }, "negative respawn delay or random"},
		{"raid HP", func(d *BossData) { d.RaidSpawns[0].MP = -1 }, "negative HP or MP"},
		{"grand order", func(d *BossData) { d.GrandBosses[0].BossID = 29021 }, "grand boss 29020: boss IDs are not sorted and unique"},
		{"grand status", func(d *BossData) { d.GrandBosses[0].Status = -1 }, "negative respawn time, HP, MP or status"},
		{"zone order", func(d *BossData) { d.Zones[1].ZoneID = 12012 }, "zone 12012: zone IDs are not sorted and unique"},
		{"zone invade", func(d *BossData) { d.Zones[0].InvadeTime = -5 }, "negative invade time"},
	} {
		d := validBosses()
		tt.change(&d)
		_, err := NewBosses(d)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error mentioning %q", tt.name, err, tt.want)
		}
	}
}

func TestLoadBossesFS(t *testing.T) {
	fsys := fstest.MapFS{
		RaidSpawnsFile:  {Data: []byte(`[{"bossId":25001,"x":0,"y":0,"z":0,"heading":0,"respawnDelay":1,"respawnRandom":0,"hp":1,"mp":1}]`)},
		GrandBossesFile: {Data: []byte(`[]`)},
		BossZonesFile:   {Data: []byte(`[]`)},
	}
	if b, err := LoadBossesFS(fsys); err != nil || len(b.RaidSpawns) != 1 {
		t.Fatalf("LoadBossesFS: %v", err)
	}
	delete(fsys, BossZonesFile)
	if _, err := LoadBossesFS(fsys); err == nil || !strings.Contains(err.Error(), BossZonesFile) {
		t.Errorf("missing zones: %v", err)
	}
	if _, err := LoadBosses(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadBosses of an empty folder: %v", err)
	}
}

// Spot checks of the committed boss data against Mobius
// raidboss_spawnlist.sql, grandboss_data.sql and zones/custom_boss.xml, and
// its references into the world data.
func TestRepoBosses(t *testing.T) {
	if _, err := os.Stat(filepath.Join(repoData, RaidSpawnsFile)); err != nil {
		t.Skipf("no boss data in %s: %v", repoData, err)
	}
	b, err := LoadBosses(repoData)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.RaidSpawns) != 196 || len(b.GrandBosses) != 9 || len(b.Zones) != 10 {
		t.Errorf("%d raid spawns, %d grand bosses, %d boss zones; want 196, 9, 10", len(b.RaidSpawns), len(b.GrandBosses), len(b.Zones))
	}
	// (25001,-54416,146480,-2887,0,129600,86400,95986,514), -- Greyclaw Kutus (23)
	if r, ok := b.RaidSpawn(25001); !ok || *r != (RaidSpawn{BossID: 25001, X: -54416, Y: 146480, Z: -2887,
		RespawnDelay: 129600, RespawnRandom: 86400, HP: 95986, MP: 514}) {
		t.Errorf("RaidSpawn(25001) = %+v, %v", r, ok)
	}
	// (29040,189400,-105702,-782,0,604800,0,520605,4140), -- Wings of Flame Ixion (84)
	if r, ok := b.RaidSpawn(29040); !ok {
		t.Error("RaidSpawn(29040) not found")
	} else if lo, hi := r.RespawnWindow(); lo != 7*24*time.Hour || hi != lo {
		t.Errorf("Ixion respawn window = %v, %v; want 7 days", lo, hi)
	}
	// Commented out in the SQL: Lilith (25283) and the Dimensional Rift Anakazels.
	for _, id := range []int{25283, 25333} {
		if _, ok := b.RaidSpawn(id); ok {
			t.Errorf("raid boss %d is spawned", id)
		}
	}
	// (29020, 116033, 17447, 10107, -25348, 4068372, 39960), -- Baium (75)
	if g, ok := b.GrandBoss(29020); !ok || *g != (GrandBoss{BossID: 29020, X: 116033, Y: 17447, Z: 10107, Heading: -25348, HP: 4068372, MP: 39960}) {
		t.Errorf("GrandBoss(29020) = %+v, %v", g, ok)
	}
	// INSERT IGNORE INTO grandboss_data VALUES (25512, 96320, -110912, -3328, 8191, 0, 0, 0, 0);
	if g, ok := b.GrandBoss(25512); !ok || g.Heading != 8191 || g.X != 96320 {
		t.Errorf("GrandBoss(25512) = %+v, %v", g, ok)
	}
	if z, ok := b.Zone(12015); !ok || z.InvadeTime != 0 || !z.Enabled {
		t.Errorf("Zone(12015) = %+v, %v", z, ok)
	} else if p, ok := z.OustPoint(); !ok || p != (Location{X: 115844, Y: -125602, Z: -3352}) {
		t.Errorf("Sirra oust point = %v, %v", p, ok)
	}
	if z, ok := b.Zone(12016); !ok || z.InvadeTime != 1800000 || z.Enabled || z.Oust != nil || z.Name != "Lair of Zaken" {
		t.Errorf("Zone(12016) = %+v, %v", z, ok)
	}

	if err := b.CheckWorld(loadRepoWorld(t)); err != nil {
		t.Errorf("CheckWorld: %v", err)
	}
}

func TestBossesCheckWorld(t *testing.T) {
	b, err := NewBosses(validBosses())
	if err != nil {
		t.Fatal(err)
	}
	w := &World{npcs: map[int]int{}, zones: map[int]int{}}
	err = b.CheckWorld(w)
	for _, want := range []string{"raid boss 25001: no such NPC", "grand boss 29001: no such NPC", "zone 12012: no such zone"} {
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), want) {
			t.Errorf("CheckWorld: %v, want an error mentioning %q", err, want)
		}
	}
}
