package data

import (
	"os"
	"path/filepath"
	"testing"
)

// worldRepoData is the data/ folder of the repository, relative to this
// package.
const worldRepoData = "../../data"

// loadRepoWorld loads the committed world data.
func loadRepoWorld(t *testing.T) *World {
	t.Helper()
	if _, err := os.Stat(filepath.Join(worldRepoData, NpcTypesFile)); err != nil {
		t.Skipf("no world data in %s: %v", worldRepoData, err)
	}
	w, err := LoadWorld(worldRepoData)
	if err != nil {
		t.Fatalf("LoadWorld(%s): %v", worldRepoData, err)
	}
	return w
}

// Spot checks of the committed world data against the Mobius XML it was
// generated from (the expected values are copied from the XML files named
// in each check).
func TestRepoWorldSize(t *testing.T) {
	w := loadRepoWorld(t)
	// Lower bounds, so a datapack update does not break the test; the
	// Interlude datapack of 2026-11 has 6519 templates (not custom), 3753
	// spawn blocks, 141 teleporters, 2411 zones and 42 map regions.
	for name, n := range map[string][2]int{
		"NPC templates": {len(w.Npcs), 6000},
		"spawn blocks":  {len(w.Spawns), 3000},
		"territories":   {len(w.Territories), 3000},
		"teleporters":   {len(w.Teleporters), 100},
		"zones":         {len(w.Zones), 2000},
		"map regions":   {len(w.MapRegions), 40},
	} {
		if n[0] < n[1] {
			t.Errorf("%d %s, want at least %d", n[0], name, n[1])
		}
	}
}

// stats/npcs/30000-30099.xml, npc 30006, and spawns/TalkingIsland/TalkingIslandNPCs.xml.
func TestRepoWorldGatekeeper(t *testing.T) {
	w := loadRepoWorld(t)
	n, ok := w.Npc(30006)
	if !ok {
		t.Fatal("NPC 30006 (Roxxy) not found")
	}
	if n.Name != "Roxxy" || n.Title != "Gatekeeper" || n.Type != "Teleporter" || n.Level != 70 ||
		n.Race != RaceHuman || n.Sex != "FEMALE" || n.DisplayID != 30006 {
		t.Errorf("Roxxy = %q %q, type %s, level %d, %v %s, display %d", n.Name, n.Title, n.Type, n.Level, n.Race, n.Sex, n.DisplayID)
	}
	if n.Status.Attackable || !n.Status.Undying || !n.Status.Targetable || !n.Status.Talkable {
		t.Errorf("Roxxy status = %+v, want not attackable, undying, targetable, talkable", n.Status)
	}
	if n.Collision.Radius != 8 || n.Collision.Height != 23 {
		t.Errorf("Roxxy collision = %+v, want radius 8, height 23", n.Collision)
	}
	if n.Stats.Vitals.HP != 2444.46819 || n.Stats.Speed.Walk != 50 || n.Stats.Speed.Run != 120 {
		t.Errorf("Roxxy HP %v, walk %v, run %v; want 2444.46819, 50, 120", n.Stats.Vitals.HP, n.Stats.Speed.Walk, n.Stats.Speed.Run)
	}
	if !w.NpcTypeIs(n.Type, "Folk") || w.NpcTypeIs(n.Type, "Monster") || !w.Spawnable(30006) {
		t.Error("Teleporter should be a spawnable Folk and not a Monster")
	}

	found := false
	for _, s := range w.Spawns {
		for _, e := range s.Npcs {
			if e.NpcID == 30006 && s.File == "TalkingIsland/TalkingIslandNPCs.xml" {
				found = true
				if e.X != -84108 || e.Y != 244604 || e.Z != -3728 || e.Heading != 40960 || e.RespawnDelay != 60 || e.Count != 1 || s.Zone != "" {
					t.Errorf("Roxxy spawn = %+v (zone %q)", e, s.Zone)
				}
			}
		}
	}
	if !found {
		t.Error("Roxxy is not spawned in TalkingIsland/TalkingIslandNPCs.xml")
	}
}

// stats/npcs/20000-20099.xml, npc 20001.
func TestRepoWorldGremlin(t *testing.T) {
	w := loadRepoWorld(t)
	n, ok := w.Npc(20001)
	if !ok {
		t.Fatal("NPC 20001 (Gremlin) not found")
	}
	if n.Name != "Gremlin" || n.Type != "Monster" || n.Level != 1 || n.Race != RaceFairy || n.Sex != "MALE" {
		t.Errorf("Gremlin = %q, type %s, level %d, %v %s", n.Name, n.Type, n.Level, n.Race, n.Sex)
	}
	s := n.Stats
	if s.STR != 40 || s.INT != 21 || s.DEX != 30 || s.WIT != 20 || s.CON != 43 || s.MEN != 10 {
		t.Errorf("Gremlin base stats = %+v", s)
	}
	if want := (NpcVitals{HP: 39.74519, HPRegen: 2, MP: 40, MPRegen: 0.9}); s.Vitals != want {
		t.Errorf("Gremlin vitals = %+v, want %+v", s.Vitals, want)
	}
	wantAttack := NpcAttack{Physical: 8.47458, Magical: 5.78704, Random: 30, Critical: 4, Accuracy: 4.75,
		Speed: 253, Type: WeaponSword, Range: 40, Distance: 80, Width: 120}
	if s.Attack != wantAttack {
		t.Errorf("Gremlin attack = %+v, want %+v", s.Attack, wantAttack)
	}
	if want := (NpcDefence{Physical: 44.44444, Magical: 29.5916164000214}); s.Defence != want {
		t.Errorf("Gremlin defence = %+v, want %+v", s.Defence, want)
	}
	if want := (NpcSpeed{Walk: 20, Run: 50, SwimWalk: 20, SwimRun: 50, FlyWalk: 20, FlyRun: 50}); s.Speed != want {
		t.Errorf("Gremlin speed = %+v, want %+v", s.Speed, want)
	}
	if s.HitTime != 370 || n.Acquire.Exp != 29 || n.Acquire.SP != 2 {
		t.Errorf("Gremlin hit time %d, exp %v, sp %v; want 370, 29, 2", s.HitTime, n.Acquire.Exp, n.Acquire.SP)
	}
	if n.Collision.Radius != 10 || n.Collision.Height != 15 {
		t.Errorf("Gremlin collision = %+v", n.Collision)
	}
	if n.Status.Undying || !n.Status.Attackable || !n.Status.CanBeSown || !n.Status.RandomWalk {
		t.Errorf("Gremlin status = %+v", n.Status)
	}
	if n.AI.AggroRange != 1000 || n.AI.ClanHelpRange != 300 || n.AI.IsAggressive || n.AI.Type != "FIGHTER" {
		t.Errorf("Gremlin AI = %+v", n.AI)
	}
	if len(n.Skills) != 1 || n.Skills[0] != (NpcSkill{ID: 4416, Level: 13}) {
		t.Errorf("Gremlin skills = %+v", n.Skills)
	}
	if !w.NpcTypeIs(n.Type, "Attackable") || !w.Spawnable(20001) {
		t.Error("Monster should be a spawnable Attackable")
	}
}

// spawns/TalkingIsland/TalkingIslandMonsters.xml, zone gludio32_1725_01.
func TestRepoWorldTerritorySpawn(t *testing.T) {
	w := loadRepoWorld(t)
	terr, ok := w.Territory("gludio32_1725_01")
	if !ok {
		t.Fatal("territory gludio32_1725_01 not found")
	}
	want := Shape{Type: ShapeNPoly, MinZ: -3451, MaxZ: -3001, Points: []Point{
		{-75291, 251836}, {-70888, 254949}, {-74207, 257108}, {-77148, 253578},
	}}
	if terr.File != "TalkingIsland/TalkingIslandMonsters.xml" || terr.Shape.String() != want.String() || len(terr.Banned) != 0 {
		t.Errorf("territory = %+v", terr)
	}
	for i, p := range want.Points {
		if terr.Shape.Points[i] != p {
			t.Errorf("point %d = %v, want %v", i, terr.Shape.Points[i], p)
		}
	}
	found := false
	for _, s := range w.Spawns {
		if s.Zone != "gludio32_1725_01" {
			continue
		}
		for _, e := range s.Npcs {
			if e.NpcID == 20481 { // Bearded Keltir
				found = true
				if e.Count != 10 || e.RespawnDelay != 25 || e.Heading != -1 || e.X != 0 || e.Y != 0 {
					t.Errorf("Bearded Keltir entry = %+v", e)
				}
			}
		}
	}
	if !found {
		t.Error("no Bearded Keltir spawn in gludio32_1725_01")
	}
	if loc := terr.RandomPoint(func(n int) int { return n / 2 }); loc.Z != -3226 {
		t.Errorf("territory spawn Z = %d, want -3226", loc.Z)
	}
}

// teleporters/town/30006.xml.
func TestRepoWorldTeleport(t *testing.T) {
	w := loadRepoWorld(t)
	l, ok := w.TeleportList(30006, "NORMAL")
	if !ok {
		t.Fatal("Roxxy has no NORMAL teleport list")
	}
	want := TeleportLocation{Name: "The Village of Gludin", X: -80749, Y: 149834, Z: -3043, FeeItemID: 57, FeeCount: 18000}
	if l.Type != TeleportNormal || len(l.Locations) != 9 || l.Locations[0].Name != want.Name ||
		l.Locations[0].X != want.X || l.Locations[0].Y != want.Y || l.Locations[0].Z != want.Z ||
		l.Locations[0].FeeItemID != want.FeeItemID || l.Locations[0].FeeCount != want.FeeCount {
		t.Errorf("Roxxy NORMAL list = %+v; want 9 locations starting with %+v", l, want)
	}
	tok, ok := w.TeleportList(30006, "NOBLES_TOKEN")
	if !ok || tok.Type != TeleportNoblesToken || tok.Locations[0].Name != "Gludin Arena" ||
		tok.Locations[0].FeeItemID != 6651 || tok.Locations[0].FeeCount != 1 {
		t.Errorf("Roxxy NOBLES_TOKEN list = %+v, %v", tok, ok)
	}
	if len(w.TeleportLists(30006)) != 3 {
		t.Errorf("Roxxy has %d lists, want 3", len(w.TeleportLists(30006)))
	}
}

// zones/water.xml, zone 11_23_water1 (no ID in the XML, so its ID is a
// dynamic one).
func TestRepoWorldWaterZone(t *testing.T) {
	w := loadRepoWorld(t)
	var z *Zone
	for i := range w.Zones {
		if w.Zones[i].Name == "11_23_water1" {
			z = &w.Zones[i]
		}
	}
	if z == nil {
		t.Fatal("zone 11_23_water1 not found")
	}
	if z.Type != "WaterZone" || z.File != "water.xml" || z.ID < FirstDynamicZoneID || z.Shape.Type != ShapeCuboid ||
		z.Shape.MinZ != -4779 || z.Shape.MaxZ != -3779 {
		t.Errorf("zone = %+v", z)
	}
	if !z.Shape.Contains(-280000, 180000, -4000) {
		t.Error("the middle of 11_23_water1 is not in it")
	}
	if z.Shape.Contains(-262144, 180000, -4000) || z.Shape.Contains(-280000, 180000, -3778) {
		t.Error("11_23_water1 contains points beyond its east edge or above its top")
	}
	if got, ok := w.Zone(z.ID); !ok || got.Name != z.Name {
		t.Errorf("Zone(%d) = %+v, %v", z.ID, got, ok)
	}
}

// zones/custom_town.xml, zone 11022, and zones/peace.xml.
func TestRepoWorldTownZone(t *testing.T) {
	w := loadRepoWorld(t)
	z, ok := w.Zone(11022)
	if !ok || z.Name != "Talking Island" || z.Type != "TownZone" {
		t.Fatalf("Zone(11022) = %+v, %v", z, ok)
	}
	if v, _ := z.Param("townId"); v != "2" {
		t.Errorf("townId = %q, want 2", v)
	}
	// Roxxy stands in the town and in its peace zone.
	if !z.Shape.Contains(-84108, 244604, -3728) {
		t.Error("Roxxy's spot is not in the Talking Island town zone")
	}
	inPeace := false
	for _, pz := range w.Zones {
		if pz.Type == "PeaceZone" && pz.Shape.Contains(-84108, 244604, -3728) {
			inPeace = true
		}
	}
	if !inPeace {
		t.Error("Roxxy's spot is in no peace zone")
	}
}

// mapregion/*.xml.
func TestRepoWorldMapRegions(t *testing.T) {
	w := loadRepoWorld(t)
	tests := []struct {
		x, y   int
		region string
		town   string
		locID  int
	}{
		{-84108, 244604, "talking_island_town", "Talking Island Town", 910}, // cell 17_25
		{-12694, 122776, "gludio_castle_town", "Gludio Castle Town", 912},   // cell 19_21
		// Cell 19_16 is listed by orc_town and rune_town; Mobius's grid
		// gives it to rune_town.
		{-10000, -40000, "rune_town", "Rune Town", 1537},
	}
	for _, tt := range tests {
		r, ok := w.MapRegionAt(tt.x, tt.y)
		if !ok || r.Name != tt.region || r.Town != tt.town || r.LocID != tt.locID {
			t.Errorf("MapRegionAt(%d, %d) = %+v, %v; want %s", tt.x, tt.y, r, ok, tt.region)
			continue
		}
		if got := w.ClosestTownName(tt.x, tt.y); got != tt.town {
			t.Errorf("ClosestTownName(%d, %d) = %q, want %q", tt.x, tt.y, got, tt.town)
		}
	}
	ti := w.DefaultRegion()
	if ti.Name != "talking_island_town" || ti.Castle != 1 || ti.BBS != 1 || len(ti.RespawnPoints) != 14 ||
		ti.RespawnPoints[0] != (Location{X: -83990, Y: 243336, Z: -3700}) {
		t.Errorf("talking_island_town = %+v", ti)
	}
	if got := w.ClosestTownName(1<<29, 0); got != DefaultTownName {
		t.Errorf("ClosestTownName off the map = %q", got)
	}
	// Respawn zones may name regions that do not exist.
	if got := w.RespawnRegion("kamaroka").Name; got != DefaultRespawnRegion {
		t.Errorf("RespawnRegion(kamaroka) = %s", got)
	}
}
