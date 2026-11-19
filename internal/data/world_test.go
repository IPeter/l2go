package data

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// worldNpc returns a plain NPC template of the given type.
func worldNpc(id int, typ, name string) NpcTemplate {
	return NpcTemplate{
		ID: id, DisplayID: id, Level: 1, Type: typ, Name: name, Race: RaceFairy, Sex: "MALE",
		Stats: NpcStats{
			Attack: NpcAttack{Type: WeaponSword, Critical: 4, Speed: 253, Range: 40, Width: 120},
			Speed:  NpcSpeed{Walk: 20, Run: 50, SwimWalk: 20, SwimRun: 50, FlyWalk: 20, FlyRun: 50},
		},
		Status:    NpcStatus{Attackable: true, Targetable: true, Talkable: true, ShowName: true, RandomWalk: true, RandomAnimation: true, CanMove: true},
		Collision: NpcCollision{Radius: 10, Height: 15},
		AI:        NpcAI{Type: "FIGHTER", IsAggressive: true, MinSkillChance: 7, MaxSkillChance: 15},
	}
}

// square returns an NPoly square of side 2*half around (x, y).
func square(x, y, half, minZ, maxZ int) Shape {
	return Shape{Type: ShapeNPoly, MinZ: minZ, MaxZ: maxZ, Points: []Point{
		{x - half, y - half}, {x + half, y - half}, {x + half, y + half}, {x - half, y + half},
	}}
}

// validWorld returns a small, valid world. Each call returns fresh values.
func validWorld() WorldData {
	boss := worldNpc(25001, "RaidBoss", "Greyclaw Kutus")
	boss.Parameters.Minions = map[string][]NpcMinion{"Privates": {{NpcID: 20001, Count: 3, RespawnTime: 360}}}
	return WorldData{
		NpcTypes: []MobiusClass{
			{Name: "Attackable", Parent: "Npc"},
			{Name: "Creature", Parent: "WorldObject", Abstract: true},
			{Name: "Folk", Parent: "Npc", Usable: true},
			{Name: "Merchant", Parent: "Folk", Usable: true},
			{Name: "Monster", Parent: "Attackable", Usable: true},
			{Name: "Npc", Parent: "Creature"},
			{Name: "RaidBoss", Parent: "Monster", Usable: true},
			{Name: "Teleporter", Parent: "Merchant", Usable: true},
		},
		Npcs: []NpcTemplate{
			worldNpc(20001, "Monster", "Gremlin"),
			boss,
			worldNpc(30006, "Teleporter", "Roxxy"),
		},
		Territories: []SpawnTerritory{{
			Name: "gludio32_1725_01", File: "TalkingIsland/TalkingIslandMonsters.xml",
			Shape: Shape{Type: ShapeNPoly, MinZ: -3451, MaxZ: -3001, Points: []Point{
				{-75291, 251836}, {-70888, 254949}, {-74207, 257108}, {-77148, 253578},
			}},
		}},
		Spawns: []Spawn{{
			File: "TalkingIsland/TalkingIslandMonsters.xml", Zone: "gludio32_1725_01",
			Npcs: []SpawnNpc{{NpcID: 20001, Heading: -1, Count: 10, RespawnDelay: 25}},
		}, {
			File: "TalkingIsland/TalkingIslandNPCs.xml", Name: "TalkingIslandNPCs",
			Npcs: []SpawnNpc{{NpcID: 30006, X: -84108, Y: 244604, Z: -3728, Heading: 40960, Count: 1, RespawnDelay: 60}},
		}},
		Teleporters: []Teleporter{{NpcID: 30006, Lists: []TeleportList{{
			Name: "NORMAL", Type: TeleportNormal, File: "town/30006.xml",
			Locations: []TeleportLocation{{Name: "The Village of Gludin", X: -80749, Y: 149834, Z: -3043, FeeItemID: 57, FeeCount: 18000}},
		}}}},
		ZoneTypes: []MobiusClass{
			{Name: "ResidenceTeleportZone", Parent: "ZoneRespawn", Usable: true},
			{Name: "RespawnZone", Parent: "ZoneType", Usable: true},
			{Name: "WaterZone", Parent: "ZoneType", Usable: true},
			{Name: "ZoneRespawn", Parent: "ZoneType", Abstract: true},
			{Name: "ZoneType", Parent: "ListenersContainer", Abstract: true},
		},
		Zones: []Zone{{
			ID: 300001, Name: "11_23_water1", Type: "WaterZone", File: "water.xml",
			Shape: Shape{Type: ShapeCuboid, MinZ: -4779, MaxZ: -3779, Points: []Point{{-294912, 163839}, {-262144, 196607}}},
		}, {
			ID: 300002, Name: "kamaroka_23", Type: "RespawnZone", File: "respawn.xml",
			Shape:      square(0, 0, 100, -100, 100),
			RacePoints: map[Race]string{RaceHuman: "talking_island_town", RaceElf: "kamaroka"},
		}, {
			ID: 300003, Name: "gludio_castle_owner_restart_territory", Type: "ResidenceTeleportZone", File: "castle_teleport.xml",
			Shape:  square(-16730, 109360, 500, -1840, -1540),
			Params: []ZoneParam{{Name: "residenceId", Value: "1"}},
			Spawns: []ZoneSpawn{{X: -18099, Y: 109297, Z: -2475}},
		}},
		MapRegions: []MapRegion{{
			Name: "orc_town", Town: "Orc Town", LocID: 920, Castle: 9, BBS: 9, File: "orc_town.xml",
			RespawnPoints: []Location{{X: -44316, Y: -113136, Z: -80}},
			Cells:         []MapCell{{X: 18, Y: 14}},
		}, {
			Name: "talking_island_town", Town: "Talking Island Town", LocID: 910, Castle: 1, BBS: 1, File: "talking_island_town.xml",
			RespawnPoints:        []Location{{X: -83990, Y: 243336, Z: -3700}, {X: -83964, Y: 242680, Z: -3700}},
			ChaoticRespawnPoints: []Location{{X: -79411, Y: 240677, Z: -3450}},
			Cells:                []MapCell{{X: 17, Y: 24}, {X: 17, Y: 25}},
			BannedRaces:          map[Race]string{RaceElf: "orc_town"},
		}},
	}
}

func TestNewWorldLookups(t *testing.T) {
	w, err := NewWorld(validWorld())
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := w.Npc(30006); !ok || n.Name != "Roxxy" {
		t.Errorf("Npc(30006) = %+v, %v", n, ok)
	}
	if _, ok := w.Npc(1); ok {
		t.Error("Npc(1) found")
	}
	for id, want := range map[int]bool{20001: true, 30006: true, 25001: false, 1: false} {
		if got := w.Spawnable(id); got != want {
			t.Errorf("Spawnable(%d) = %v, want %v", id, got, want)
		}
	}

	types := []struct {
		typ, class string
		want       bool
	}{
		{"Teleporter", "Teleporter", true},
		{"Teleporter", "Folk", true},
		{"RaidBoss", "Monster", true},
		{"RaidBoss", "Creature", true},
		{"RaidBoss", "WorldObject", true}, // the chain's last parent counts
		{"Monster", "Folk", false},
		{"Ghost", "Npc", false},
	}
	for _, tt := range types {
		if got := w.NpcTypeIs(tt.typ, tt.class); got != tt.want {
			t.Errorf("NpcTypeIs(%q, %q) = %v, want %v", tt.typ, tt.class, got, tt.want)
		}
	}
	if !w.ZoneTypeIs("ResidenceTeleportZone", "ZoneRespawn") || w.ZoneTypeIs("WaterZone", "ZoneRespawn") {
		t.Error("ZoneTypeIs does not follow the zone classes")
	}

	if tr, ok := w.Territory("gludio32_1725_01"); !ok || len(tr.Shape.Points) != 4 {
		t.Errorf("Territory(gludio32_1725_01) = %+v, %v", tr, ok)
	}
	if _, ok := w.Territory("nowhere"); ok {
		t.Error("Territory(nowhere) found")
	}

	if lists := w.TeleportLists(30006); len(lists) != 1 {
		t.Errorf("TeleportLists(30006) = %+v", lists)
	}
	if lists := w.TeleportLists(20001); lists != nil {
		t.Errorf("TeleportLists(20001) = %+v, want nil", lists)
	}
	l, ok := w.TeleportList(30006, "NORMAL")
	if !ok || l.Locations[0].FeeCount != 18000 || l.Locations[0].FeeItemID != DefaultTeleportFeeItem {
		t.Errorf("TeleportList(30006, NORMAL) = %+v, %v", l, ok)
	}
	if _, ok := w.TeleportList(30006, "NOBLES_TOKEN"); ok {
		t.Error("TeleportList(30006, NOBLES_TOKEN) found")
	}
	if _, ok := w.TeleportList(1, "NORMAL"); ok {
		t.Error("TeleportList(1, NORMAL) found")
	}

	z, ok := w.Zone(300003)
	if !ok {
		t.Fatal("Zone(300003) not found")
	}
	if v, ok := z.Param("residenceId"); !ok || v != "1" {
		t.Errorf("Param(residenceId) = %q, %v", v, ok)
	}
	if _, ok := z.Param("townId"); ok {
		t.Error("Param(townId) found")
	}
	if _, ok := w.Zone(1); ok {
		t.Error("Zone(1) found")
	}
}

func TestMapRegionLookups(t *testing.T) {
	w, err := NewWorld(validWorld())
	if err != nil {
		t.Fatal(err)
	}
	// Roxxy's spot in Talking Island village, cell 17_25.
	r, ok := w.MapRegionAt(-84108, 244604)
	if !ok || r.Name != "talking_island_town" {
		t.Fatalf("MapRegionAt(Talking Island) = %+v, %v", r, ok)
	}
	if got := w.ClosestTownName(-84108, 244604); got != "Talking Island Town" {
		t.Errorf("ClosestTownName = %q", got)
	}
	if w.MapRegionLocID(-84108, 244604) != 910 || w.AreaCastle(-84108, 244604) != 1 || w.BBS(-84108, 244604) != 1 {
		t.Error("Talking Island: wrong location ID, castle or BBS")
	}
	if r, ok := w.MapRegionAt(-44316, -113136); !ok || r.Name != "orc_town" || w.BBS(-44316, -113136) != 9 {
		t.Errorf("MapRegionAt(Orc Town) = %+v, %v", r, ok)
	}

	// Points without a region: inside the grid but unclaimed, and off it.
	for _, pt := range [][2]int{{0, 0}, {1 << 29, 0}, {0, -(1 << 29)}} {
		if r, ok := w.MapRegionAt(pt[0], pt[1]); ok {
			t.Errorf("MapRegionAt(%v) = %s", pt, r.Name)
		}
		if got := w.ClosestTownName(pt[0], pt[1]); got != DefaultTownName {
			t.Errorf("ClosestTownName(%v) = %q, want %q", pt, got, DefaultTownName)
		}
		if w.MapRegionLocID(pt[0], pt[1]) != 0 || w.AreaCastle(pt[0], pt[1]) != 0 {
			t.Errorf("%v: location ID and castle should be 0", pt)
		}
		if got := w.BBS(pt[0], pt[1]); got != 1 {
			t.Errorf("BBS(%v) = %d, want the default region's 1", pt, got)
		}
	}

	if got := w.DefaultRegion().Name; got != DefaultRespawnRegion {
		t.Errorf("DefaultRegion = %s", got)
	}
	if got := w.RespawnRegion("orc_town").Name; got != "orc_town" {
		t.Errorf("RespawnRegion(orc_town) = %s", got)
	}
	// kamaroka is named by a respawn zone but does not exist.
	if got := w.RespawnRegion("kamaroka").Name; got != DefaultRespawnRegion {
		t.Errorf("RespawnRegion(kamaroka) = %s, want %s", got, DefaultRespawnRegion)
	}
}

func TestMapRegionCell(t *testing.T) {
	tests := []struct {
		x, y int
		want MapCell
	}{
		{0, 0, MapCell{20, 18}},
		{32767, 32767, MapCell{20, 18}},
		{32768, 32768, MapCell{21, 19}},
		{-1, -1, MapCell{19, 17}}, // the shift rounds down
		{-32768, -32768, MapCell{19, 17}},
		{-32769, -32769, MapCell{18, 16}},
		{-84108, 244604, MapCell{17, 25}}, // Talking Island village
		{-12694, 122776, MapCell{19, 21}}, // Gludio
	}
	for _, tt := range tests {
		if got := MapRegionCell(tt.x, tt.y); got != tt.want {
			t.Errorf("MapRegionCell(%d, %d) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestNewWorldValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(d *WorldData)
		want   string
	}{
		{"no NPCs", func(d *WorldData) { d.Npcs = nil }, "npcs/npcs.json: no NPC templates"},
		{"duplicate NPC", func(d *WorldData) { d.Npcs = append(d.Npcs, d.Npcs[0]) }, `NPC 20001 ("Gremlin"): duplicate NPC ID`},
		{"bad NPC ID", func(d *WorldData) { d.Npcs = append(d.Npcs, worldNpc(-5, "Monster", "Bad")) }, "NPC ID must be positive"},
		{"bad display ID", func(d *WorldData) { d.Npcs[0].DisplayID = 0 }, "display ID must be positive"},
		{"level out of range", func(d *WorldData) { d.Npcs[0].Level = 200 }, "level 200 is out of range 0-127"},
		{"unknown NPC type", func(d *WorldData) { d.Npcs[0].Type = "Ghost" }, `unknown NPC type "Ghost"`},
		{"abstract NPC type", func(d *WorldData) { d.Npcs[0].Type = "Creature" }, `unknown NPC type "Creature"`},
		{"base NPC type", func(d *WorldData) { d.Npcs[0].Type = "Npc" }, `unknown NPC type "Npc"`},
		{"unknown race", func(d *WorldData) { d.Npcs[0].Race = Race(99) }, "unknown race"},
		{"unknown attack type", func(d *WorldData) { d.Npcs[0].Stats.Attack.Type = WeaponType(99) }, "unknown attack type"},
		{"unknown sex", func(d *WorldData) { d.Npcs[0].Sex = "BOTH" }, `unknown sex "BOTH"`},
		{"unknown AI type", func(d *WorldData) { d.Npcs[0].AI.Type = "SMART" }, `unknown AI type "SMART"`},
		{"unknown attribute", func(d *WorldData) { d.Npcs[0].Stats.Attribute.AttackType = "ICE" }, `unknown attribute type "ICE"`},
		{"negative collision", func(d *WorldData) { d.Npcs[0].Collision.Radius = -1 }, "collision sizes must not be negative"},
		{"negative speed", func(d *WorldData) { d.Npcs[0].Stats.Speed.SwimRun = -1 }, "speeds must not be negative"},
		{"bad skill", func(d *WorldData) { d.Npcs[0].Skills = []NpcSkill{{ID: 0, Level: 1}} }, "skill 0 level 1: ID and level must be positive"},
		{"bad parameter skill", func(d *WorldData) {
			d.Npcs[0].Parameters.Skills = map[string]NpcSkill{"DDMagic": {ID: 4001}}
		}, "parameter skill DDMagic: ID and level must be positive"},
		{"negative drop amount", func(d *WorldData) {
			d.Npcs[0].Drops.Items = []NpcDropItem{{ItemID: 57, Min: -1, Max: 1, Chance: 10}}
		}, "drop: item 57: bad numbers"},
		{"spoil without item", func(d *WorldData) {
			d.Npcs[0].Drops.Spoil = []NpcDropItem{{Min: 1, Max: 1, Chance: 10}}
		}, "spoil: item 0: bad numbers"},
		{"empty drop group", func(d *WorldData) { d.Npcs[0].Drops.Groups = []NpcDropGroup{{Chance: 10}} }, "drop group 0: needs items"},
		{"unknown minion", func(d *WorldData) { d.Npcs[1].Parameters.Minions["Privates"][0].NpcID = 99999 }, "minions Privates: NPC 99999 does not exist"},
		{"minion without count", func(d *WorldData) { d.Npcs[1].Parameters.Minions["Privates"][0].Count = 0 }, "needs a positive count"},

		{"class without name", func(d *WorldData) { d.NpcTypes = append(d.NpcTypes, MobiusClass{Parent: "Npc"}) }, "npcs/types.json: class 8 has no name"},
		{"duplicate class", func(d *WorldData) { d.ZoneTypes = append(d.ZoneTypes, d.ZoneTypes[0]) }, "zones/types.json: class ResidenceTeleportZone: duplicate name"},
		{"class cycle", func(d *WorldData) { d.NpcTypes[1].Parent = "Monster" }, "inheritance cycle"},
		{"no classes", func(d *WorldData) { d.ZoneTypes = nil }, "zones/types.json: no classes"},

		{"territory without name", func(d *WorldData) { d.Territories[0].Name = "" }, "no name"},
		{"duplicate territory", func(d *WorldData) { d.Territories = append(d.Territories, d.Territories[0]) }, "duplicate territory name"},
		{"polygon of 2 points", func(d *WorldData) {
			d.Territories[0].Shape.Points = d.Territories[0].Shape.Points[:2]
		}, "an NPoly needs at least 3 points (got 2)"},
		{"cuboid of 3 points", func(d *WorldData) {
			d.Zones[0].Shape.Points = append(d.Zones[0].Shape.Points, Point{0, 0})
		}, "a Cuboid needs exactly 2 points (got 3)"},
		{"cylinder without radius", func(d *WorldData) {
			d.Zones[0].Shape = Shape{Type: ShapeCylinder, Points: []Point{{0, 0}}}
		}, "a Cylinder needs 1 point and a positive radius"},
		{"radius on a polygon", func(d *WorldData) { d.Territories[0].Shape.Radius = 5 }, "only a Cylinder has a radius"},
		{"unknown shape", func(d *WorldData) { d.Zones[0].Shape.Type = "Circle" }, `unknown shape type "Circle"`},
		{"bad banned area", func(d *WorldData) {
			d.Territories[0].Banned = []Shape{{Type: ShapeNPoly, Points: []Point{{0, 0}}}}
		}, "banned area 0: an NPoly needs at least 3 points (got 1)"},

		{"no spawns", func(d *WorldData) { d.Spawns = nil }, "spawns/spawns.json: no spawns"},
		{"unknown territory", func(d *WorldData) { d.Spawns[0].Zone = "nowhere" }, `territory "nowhere" does not exist`},
		{"unknown group territory", func(d *WorldData) {
			d.Spawns[0].Zones, d.Spawns[0].Selection = []string{"nowhere"}, SelectionRandom
		}, `territory "nowhere" does not exist`},
		{"unknown spawn NPC", func(d *WorldData) { d.Spawns[1].Npcs[0].NpcID = 99999 }, "NPC 99999: does not exist in npcs/npcs.json"},
		{"zero count", func(d *WorldData) { d.Spawns[0].Npcs[0].Count = 0 }, "count must be positive (got 0)"},
		{"negative delay", func(d *WorldData) { d.Spawns[0].Npcs[0].RespawnRandom = -1 }, "must not be negative"},
		{"unknown period", func(d *WorldData) { d.Spawns[0].Npcs[0].PeriodOfDay = "dusk" }, `unknown period of day "dusk"`},
		{"no point", func(d *WorldData) { d.Spawns[1].Npcs[0].X, d.Spawns[1].Npcs[0].Y = 0, 0 }, "neither a point nor a territory"},
		{"unknown selection", func(d *WorldData) { d.Spawns[0].Selection = "some" }, `unknown selection "some"`},
		{"zones without selection", func(d *WorldData) { d.Spawns[0].Zones = []string{"gludio32_1725_01"} }, "zones without a selection"},
		{"spawn without entries", func(d *WorldData) { d.Spawns[0].Npcs = nil }, "no NPC entries"},

		{"duplicate teleporter", func(d *WorldData) { d.Teleporters = append(d.Teleporters, d.Teleporters[0]) }, "teleporters/teleporters.json: NPC 30006: duplicate NPC ID"},
		{"teleporter of unknown NPC", func(d *WorldData) { d.Teleporters[0].NpcID = 99999 }, "teleporters/teleporters.json: NPC 99999: does not exist"},
		{"no teleport lists", func(d *WorldData) { d.Teleporters[0].Lists = nil }, "no teleport lists"},
		{"list without name", func(d *WorldData) { d.Teleporters[0].Lists[0].Name = "" }, `list "" (town/30006.xml): no name`},
		{"duplicate list", func(d *WorldData) {
			d.Teleporters[0].Lists = append(d.Teleporters[0].Lists, d.Teleporters[0].Lists[0])
		}, "duplicate list name"},
		{"unknown list type", func(d *WorldData) { d.Teleporters[0].Lists[0].Type = "VIP" }, `unknown type "VIP"`},
		{"negative fee", func(d *WorldData) { d.Teleporters[0].Lists[0].Locations[0].FeeCount = -1 }, "fee -1 is negative"},
		{"bad castle", func(d *WorldData) { d.Teleporters[0].Lists[0].Locations[0].CastleIDs = []int{0} }, "castle ID 0 must be positive"},

		{"no zones", func(d *WorldData) { d.Zones = nil }, "zones/zones.json: no zones"},
		{"duplicate zone", func(d *WorldData) { d.Zones = append(d.Zones, d.Zones[0]) }, "duplicate zone ID"},
		{"zero zone ID", func(d *WorldData) { d.Zones[0].ID = 0 }, "zone ID must be positive"},
		{"unknown zone type", func(d *WorldData) { d.Zones[0].Type = "TaxZone" }, `unknown zone type "TaxZone"`},
		{"abstract zone type", func(d *WorldData) { d.Zones[0].Type = "ZoneRespawn" }, `unknown zone type "ZoneRespawn"`},
		{"parameter without name", func(d *WorldData) { d.Zones[0].Params = []ZoneParam{{Value: "1"}} }, "parameter without a name"},
		{"spawn points on a water zone", func(d *WorldData) {
			d.Zones[0].Spawns = []ZoneSpawn{{X: 1, Y: 2, Z: 3}}
		}, "spawn points on a WaterZone, which is not a ZoneRespawn"},
		{"unknown spawn point type", func(d *WorldData) { d.Zones[2].Spawns[0].Type = "start" }, `unknown spawn point type "start"`},
		{"challenger outside a hall", func(d *WorldData) { d.Zones[2].Spawns[0].Type = ZoneSpawnChallenger }, `unknown spawn point type "challenger"`},
		{"race points on a water zone", func(d *WorldData) {
			d.Zones[0].RacePoints = map[Race]string{RaceOrc: "orc_town"}
		}, "race points on a WaterZone, which is not a RespawnZone"},
		{"unknown race point race", func(d *WorldData) { d.Zones[1].RacePoints[Race(99)] = "orc_town" }, "needs a known race"},
		{"race point without region", func(d *WorldData) { d.Zones[1].RacePoints[RaceOrc] = "" }, "needs a known race and a map region"},

		{"no default region", func(d *WorldData) { d.MapRegions[1].Name = "talking_island" }, `the default respawn region "talking_island_town" does not exist`},
		{"duplicate region", func(d *WorldData) { d.MapRegions = append(d.MapRegions, d.MapRegions[0]) }, "duplicate region name"},
		{"region without name", func(d *WorldData) { d.MapRegions[0].Name = "" }, "no name"},
		{"region without town", func(d *WorldData) { d.MapRegions[0].Town = "" }, "no town"},
		{"bad location ID", func(d *WorldData) { d.MapRegions[0].LocID = 0 }, "location ID must be positive"},
		{"castle out of range", func(d *WorldData) { d.MapRegions[0].Castle = 10 }, "castle 10 is out of range 0-9"},
		{"no respawn point", func(d *WorldData) { d.MapRegions[0].RespawnPoints = nil }, "no normal respawn point"},
		{"cell off the grid", func(d *WorldData) { d.MapRegions[0].Cells = []MapCell{{32, 0}} }, "cell 32_0 is outside the 32x32 grid"},
		{"shared cell", func(d *WorldData) {
			d.MapRegions[0].Cells = append(d.MapRegions[0].Cells, MapCell{17, 25})
		}, `cell 17_25 also belongs to region "orc_town"`},
		{"banned race to a missing region", func(d *WorldData) { d.MapRegions[1].BannedRaces[RaceElf] = "elf_town" }, `region "elf_town" does not exist`},
		{"banned unknown race", func(d *WorldData) { d.MapRegions[1].BannedRaces[Race(99)] = "orc_town" }, "unknown race"},
	}
	for _, tt := range tests {
		d := validWorld()
		tt.mutate(&d)
		_, err := NewWorld(d)
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: NewWorld = %v, want ErrInvalid", tt.name, err)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}
}

func TestNewWorldReportsAllProblems(t *testing.T) {
	d := validWorld()
	d.Spawns[0].Npcs[0].NpcID = 99999
	d.Zones[0].Type = "TaxZone"
	d.MapRegions[0].Town = ""
	_, err := NewWorld(d)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewWorld = %v, want ErrInvalid", err)
	}
	for _, want := range []string{"NPC 99999", `unknown zone type "TaxZone"`, "no town"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// worldFS returns the world d as JSON files.
func worldFS(t *testing.T, d WorldData) fstest.MapFS {
	t.Helper()
	fsys := fstest.MapFS{}
	for _, f := range d.worldFiles() {
		b, err := json.Marshal(f.v)
		if err != nil {
			t.Fatal(err)
		}
		fsys[f.name] = &fstest.MapFile{Data: b}
	}
	return fsys
}

func gzipped(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLoadWorldFS(t *testing.T) {
	w, err := LoadWorldFS(worldFS(t, validWorld()))
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := w.Npc(20001); !ok || n.Race != RaceFairy || n.Stats.Attack.Type != WeaponSword {
		t.Errorf("Npc(20001) = %+v, %v", n, ok)
	}
	if z, ok := w.Zone(300002); !ok || z.RacePoints[RaceElf] != "kamaroka" {
		t.Errorf("Zone(300002) = %+v, %v", z, ok)
	}

	// A file over the size limit is stored gzipped.
	fsys := worldFS(t, validWorld())
	fsys[NpcsFile+".gz"] = &fstest.MapFile{Data: gzipped(t, fsys[NpcsFile].Data)}
	delete(fsys, NpcsFile)
	if w, err := LoadWorldFS(fsys); err != nil {
		t.Errorf("gzipped %s: %v", NpcsFile, err)
	} else if len(w.Npcs) != 3 {
		t.Errorf("gzipped %s: %d NPCs, want 3", NpcsFile, len(w.Npcs))
	}
}

func TestLoadWorldFSErrors(t *testing.T) {
	spawnsGz := gzipped(t, worldFS(t, validWorld())[SpawnsFile].Data)
	tests := []struct {
		name   string
		change func(fstest.MapFS)
		want   string
	}{
		{"missing file", func(m fstest.MapFS) { delete(m, ZonesFile) }, "data: read zones/zones.json"},
		{"both forms", func(m fstest.MapFS) {
			m[SpawnsFile+".gz"] = &fstest.MapFile{Data: spawnsGz}
		}, "remove the stale one"},
		{"broken gzip", func(m fstest.MapFS) {
			m[ZonesFile+".gz"] = &fstest.MapFile{Data: []byte("not gzip")}
			delete(m, ZonesFile)
		}, "zones/zones.json.gz"},
		{"bad JSON", func(m fstest.MapFS) { m[TeleportersFile] = &fstest.MapFile{Data: []byte(`[{"npcId": 1,`)} }, "data: parse teleporters/teleporters.json"},
		{"unknown field", func(m fstest.MapFS) {
			m[MapRegionsFile] = &fstest.MapFile{Data: []byte(`[{"name": "x", "colour": "red"}]`)}
		}, `unknown field "colour"`},
		{"unknown race", func(m fstest.MapFS) {
			m[NpcsFile] = &fstest.MapFile{Data: []byte(`[{"id": 1, "race": "MARTIAN"}]`)}
		}, `unknown race "MARTIAN"`},
		{"trailing data", func(m fstest.MapFS) {
			m[NpcTypesFile] = &fstest.MapFile{Data: append(m[NpcTypesFile].Data, []byte(" []")...)}
		}, "unexpected data after the JSON value"},
		{"invalid data", func(m fstest.MapFS) { m[SpawnsFile] = &fstest.MapFile{Data: []byte(`[]`)} }, "no spawns"},
	}
	for _, tt := range tests {
		fsys := worldFS(t, validWorld())
		tt.change(fsys)
		_, err := LoadWorldFS(fsys)
		if err == nil {
			t.Errorf("%s: LoadWorldFS succeeded", tt.name)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}

	fsys := worldFS(t, validWorld())
	delete(fsys, TerritoriesFile)
	if _, err := LoadWorldFS(fsys); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: error %v does not wrap fs.ErrNotExist", err)
	}
}

func TestLoadWorldMissingFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nothing-here")
	_, err := LoadWorld(dir)
	if err == nil {
		t.Fatal("LoadWorld of a missing folder succeeded")
	}
	if !strings.Contains(err.Error(), dir) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %q: want it to name %s and wrap fs.ErrNotExist", err, dir)
	}
}

// The JSON names of the enums are the Mobius names, map keys included.
func TestWorldJSONNames(t *testing.T) {
	d := validWorld()
	b, err := json.Marshal(d.Zones[1])
	if err != nil {
		t.Fatal(err)
	}
	if want := `"racePoints":{"ELF":"kamaroka","HUMAN":"talking_island_town"}`; !strings.Contains(string(b), want) {
		t.Errorf("zone JSON %s does not contain %s", b, want)
	}
	b, err = json.Marshal(d.Npcs[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"race":"FAIRY"`, `"type":"SWORD"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("NPC JSON %s does not contain %s", b, want)
		}
	}
	// Empty optional groups are left out.
	for _, absent := range []string{`"equipment"`, `"acquire"`, `"shots"`, `"parameters"`, `"drops"`, `"attribute"`} {
		if strings.Contains(string(b), absent) {
			t.Errorf("NPC JSON %s contains %s", b, absent)
		}
	}
}
