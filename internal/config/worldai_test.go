package config

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// withWorldAINPCs returns n with the raid, minion and chaos keys at their
// defaults, for tests that spell out the older NPC keys.
func withWorldAINPCs(n NPCs) NPCs {
	d := DefaultNPCs()
	n.RaidMinionRespawnTime = d.RaidMinionRespawnTime
	n.CustomMinionsRespawnTime = d.CustomMinionsRespawnTime
	n.ForceDeleteMinions = d.ForceDeleteMinions
	n.DisableRaidCurse = d.DisableRaidCurse
	n.RaidChaosTime, n.GrandChaosTime, n.MinionChaosTime = d.RaidChaosTime, d.GrandChaosTime, d.MinionChaosTime
	n.RaidMinRespawnMultiplier, n.RaidMaxRespawnMultiplier = d.RaidMinRespawnMultiplier, d.RaidMaxRespawnMultiplier
	n.RaidHPRegenMultiplier, n.RaidMPRegenMultiplier = d.RaidHPRegenMultiplier, d.RaidMPRegenMultiplier
	n.RaidPDefenceMultiplier, n.RaidMDefenceMultiplier = d.RaidPDefenceMultiplier, d.RaidMDefenceMultiplier
	n.RaidPAttackMultiplier, n.RaidMAttackMultiplier = d.RaidPAttackMultiplier, d.RaidMAttackMultiplier
	return n
}

// The Mobius GeoEngine.ini, NPC.ini, GrandBoss.ini and General.ini values.
func TestDefaultWorldAI(t *testing.T) {
	gs := Default().GameServers[0]
	wantGeo := Geodata{
		Pathfinding: true, PathFindBuffers: "100x6;128x6;192x6;256x4;320x4;384x4;500x2",
		LowWeight: 0.5, MediumWeight: 2, HighWeight: 3, DiagonalWeight: 0.707,
		AvoidObstructedPathNodes: true, MaxPostfilterPasses: 3,
	}
	if gs.Geodata != wantGeo {
		t.Errorf("Geodata = %+v, want %+v", gs.Geodata, wantGeo)
	}
	n := gs.NPCs
	if n.RaidMinionRespawnTime != Duration(5*time.Minute) || n.ForceDeleteMinions || n.DisableRaidCurse ||
		n.RaidChaosTime != 30 || n.GrandChaosTime != 30 || n.MinionChaosTime != 30 ||
		n.RaidMinRespawnMultiplier != 1 || n.RaidMaxRespawnMultiplier != 1 ||
		n.RaidHPRegenMultiplier != 100 || n.RaidMPRegenMultiplier != 100 || n.RaidPDefenceMultiplier != 100 ||
		n.RaidMDefenceMultiplier != 100 || n.RaidPAttackMultiplier != 100 || n.RaidMAttackMultiplier != 100 {
		t.Errorf("NPCs = %+v", n)
	}
	times, err := n.MinionRespawnTimes()
	if err != nil || len(times) != 23 || times[22371] != 2*time.Minute || times[22450] != 30*time.Second {
		t.Errorf("MinionRespawnTimes = %v, %v; want Mobius's 23 entries", times, err)
	}
	if v, ok := times[25608]; !ok || v != 0 {
		t.Errorf("minion 25608 = %v, %t; want 0", v, ok)
	}
	b := gs.GrandBosses
	h := func(n int) Duration { return Duration(time.Duration(n) * time.Hour) }
	if b.AntharasWaitTime != Duration(20*time.Minute) || b.IntervalOfAntharasSpawn != h(264) || b.RandomOfAntharasSpawn != h(8) ||
		!b.AntharasRecognizeHero || b.ValakasWaitTime != Duration(20*time.Minute) || b.IntervalOfValakasSpawn != h(264) ||
		!b.ValakasRecognizeHero || b.IntervalOfBaiumSpawn != h(168) || !b.BaiumRecognizeHero ||
		b.IntervalOfCoreSpawn != h(60) || b.IntervalOfOrfenSpawn != h(48) || b.IntervalOfQueenAntSpawn != h(36) ||
		b.RandomOfQueenAntSpawn != h(8) || b.IntervalOfZakenSpawn != h(60) || b.RandomOfZakenSpawn != h(8) {
		t.Errorf("GrandBosses = %+v", b)
	}
	w := gs.World
	if w.MinNpcAnimation != Duration(5*time.Second) || w.MaxNpcAnimation != Duration(time.Minute) ||
		w.MinMonsterAnimation != Duration(5*time.Second) || w.MaxMonsterAnimation != Duration(time.Minute) ||
		!w.AllowBoat || w.BoatBroadcastRadius != 20000 {
		t.Errorf("World = %+v", w)
	}
}

// Every new key is read, a section left out keeps its defaults, and
// explicit false and 0 are kept.
func TestParseWorldAI(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1,
		"world": {"minNpcAnimation": "0s", "maxNpcAnimation": "0s", "minMonsterAnimation": "1s", "maxMonsterAnimation": "2s",
			"allowBoat": false, "boatBroadcastRadius": 0},
		"npcs": {"raidMinionRespawnTime": "1m", "customMinionsRespawnTime": "1,2", "forceDeleteMinions": true, "disableRaidCurse": true,
			"raidChaosTime": 0, "grandChaosTime": 1, "minionChaosTime": 2, "raidMinRespawnMultiplier": 0.5, "raidMaxRespawnMultiplier": 2,
			"raidHpRegenMultiplier": 10, "raidMpRegenMultiplier": 20, "raidPDefenceMultiplier": 30, "raidMDefenceMultiplier": 40,
			"raidPAttackMultiplier": 50, "raidMAttackMultiplier": 60},
		"geodata": {"dir": "/srv/geo", "pathfinding": false, "pathFindBuffers": "50x1", "lowWeight": 1, "mediumWeight": 4, "highWeight": 9,
			"advancedDiagonalStrategy": true, "diagonalWeight": 1.4, "avoidObstructedPathNodes": false, "maxPostfilterPasses": 0},
		"grandBosses": {"_comment": "g", "intervalOfQueenAntSpawn": "1h", "randomOfQueenAntSpawn": "0s", "baiumRecognizeHero": false}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	gs := cfg.GameServers[0]
	wantGeo := Geodata{Dir: "/srv/geo", PathFindBuffers: "50x1", LowWeight: 1, MediumWeight: 4, HighWeight: 9,
		AdvancedDiagonalStrategy: true, DiagonalWeight: 1.4}
	if gs.Geodata != wantGeo {
		t.Errorf("Geodata = %+v, want %+v", gs.Geodata, wantGeo)
	}
	wantNPCs := DefaultNPCs()
	wantNPCs.RaidMinionRespawnTime, wantNPCs.CustomMinionsRespawnTime = Duration(time.Minute), "1,2"
	wantNPCs.ForceDeleteMinions, wantNPCs.DisableRaidCurse = true, true
	wantNPCs.RaidChaosTime, wantNPCs.GrandChaosTime, wantNPCs.MinionChaosTime = 0, 1, 2
	wantNPCs.RaidMinRespawnMultiplier, wantNPCs.RaidMaxRespawnMultiplier = 0.5, 2
	wantNPCs.RaidHPRegenMultiplier, wantNPCs.RaidMPRegenMultiplier = 10, 20
	wantNPCs.RaidPDefenceMultiplier, wantNPCs.RaidMDefenceMultiplier = 30, 40
	wantNPCs.RaidPAttackMultiplier, wantNPCs.RaidMAttackMultiplier = 50, 60
	if gs.NPCs != wantNPCs {
		t.Errorf("NPCs = %+v\nwant %+v", gs.NPCs, wantNPCs)
	}
	wantWorld := DefaultWorld()
	wantWorld.MinNpcAnimation, wantWorld.MaxNpcAnimation = 0, 0
	wantWorld.MinMonsterAnimation, wantWorld.MaxMonsterAnimation = Duration(time.Second), Duration(2*time.Second)
	wantWorld.AllowBoat, wantWorld.BoatBroadcastRadius = false, 0
	if gs.World != wantWorld {
		t.Errorf("World = %+v\nwant %+v", gs.World, wantWorld)
	}
	wantBosses := DefaultGrandBosses()
	wantBosses.Comment = "g"
	wantBosses.IntervalOfQueenAntSpawn, wantBosses.RandomOfQueenAntSpawn = Duration(time.Hour), 0
	wantBosses.BaiumRecognizeHero = false
	if gs.GrandBosses != wantBosses {
		t.Errorf("GrandBosses = %+v\nwant %+v", gs.GrandBosses, wantBosses)
	}
	if errs := gs.validateWorldAI(); len(errs) != 0 {
		t.Errorf("validateWorldAI = %v", errs)
	}

	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if gs := cfg.GameServers[0]; gs.Geodata != DefaultGeodata() || gs.GrandBosses != DefaultGrandBosses() {
		t.Errorf("left out: Geodata %+v GrandBosses %+v, want the defaults", gs.Geodata, gs.GrandBosses)
	}
	if _, err := Parse([]byte(`{"gameservers": [{"id": 1, "grandBosses": {"intervalOfFrintezzaSpawn": "1h"}}]}`)); err == nil {
		t.Error("an unknown grandBosses key parsed")
	}
}

// The compose server reads the folder docker-compose.yml mounts at
// /opt/l2go/geodata, its working directory being /opt/l2go.
func TestComposeGeodata(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "deploy", "compose", "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := DefaultGeodata()
	want.Comment = cfg.GameServers[0].Geodata.Comment
	want.Dir = "geodata"
	if got := cfg.GameServers[0].Geodata; got != want {
		t.Errorf("compose geodata = %+v, want %+v", got, want)
	}
	if got := cfg.GameServers[0].GrandBosses; got.IntervalOfQueenAntSpawn != DefaultGrandBosses().IntervalOfQueenAntSpawn {
		t.Errorf("compose grandBosses = %+v", got)
	}
}

func TestBufferSizes(t *testing.T) {
	tests := []struct {
		in   string
		want []int
		bad  bool
	}{
		{"100x6;128x6;192x6;256x4;320x4;384x4;500x2", []int{100, 128, 192, 256, 320, 384, 500}, false},
		{"500x2;100x6", []int{500, 100}, false},
		{" 64 ; 128x1 ", []int{64, 128}, false},
		{"", nil, true},
		{"100x6;", nil, true},
		{"x6", nil, true},
		{"0x1", nil, true},
		{"-5x1", nil, true},
	}
	for _, tc := range tests {
		got, err := Geodata{PathFindBuffers: tc.in}.BufferSizes()
		if (err != nil) != tc.bad || !slices.Equal(got, tc.want) {
			t.Errorf("BufferSizes(%q) = %v, %v; want %v (error %t)", tc.in, got, err, tc.want, tc.bad)
		}
	}
}

func TestMinionRespawnTimes(t *testing.T) {
	tests := []struct {
		in   string
		want map[int]time.Duration
		bad  bool
	}{
		{"", map[int]time.Duration{}, false},
		{"1,30; 2,0 ;", map[int]time.Duration{1: 30 * time.Second, 2: 0}, false},
		{"1", nil, true},
		{"1,x", nil, true},
		{"0,5", nil, true},
		{"1,-5", nil, true},
	}
	for _, tc := range tests {
		got, err := NPCs{CustomMinionsRespawnTime: tc.in}.MinionRespawnTimes()
		if (err != nil) != tc.bad || (!tc.bad && !maps.Equal(got, tc.want)) {
			t.Errorf("MinionRespawnTimes(%q) = %v, %v; want %v (error %t)", tc.in, got, err, tc.want, tc.bad)
		}
	}
}

func TestValidateWorldAI(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*GameServer)
		want   string // substring of the error; empty means valid
	}{
		{"defaults", func(*GameServer) {}, ""},
		{"pathfinding off", func(g *GameServer) { g.Geodata.Pathfinding = false }, ""},
		{"bad buffers", func(g *GameServer) { g.Geodata.PathFindBuffers = "big" }, "geodata.pathFindBuffers"},
		{"zero low weight", func(g *GameServer) { g.Geodata.LowWeight = 0 }, "geodata.lowWeight"},
		{"negative medium weight", func(g *GameServer) { g.Geodata.MediumWeight = -1 }, "geodata.mediumWeight"},
		{"zero high weight", func(g *GameServer) { g.Geodata.HighWeight = 0 }, "geodata.highWeight"},
		{"zero diagonal weight", func(g *GameServer) { g.Geodata.DiagonalWeight = 0 }, "geodata.diagonalWeight"},
		{"negative passes", func(g *GameServer) { g.Geodata.MaxPostfilterPasses = -1 }, "geodata.maxPostfilterPasses"},
		{"negative minion respawn", func(g *GameServer) { g.NPCs.RaidMinionRespawnTime = -1 }, "npcs.raidMinionRespawnTime"},
		{"bad custom minions", func(g *GameServer) { g.NPCs.CustomMinionsRespawnTime = "1;2" }, "npcs.customMinionsRespawnTime"},
		{"negative chaos", func(g *GameServer) { g.NPCs.GrandChaosTime = -1 }, "npcs.grandChaosTime"},
		{"negative respawn multiplier", func(g *GameServer) { g.NPCs.RaidMaxRespawnMultiplier = -1 }, "npcs.raidMaxRespawnMultiplier"},
		{"negative raid attack", func(g *GameServer) { g.NPCs.RaidMAttackMultiplier = -1 }, "npcs.raidMAttackMultiplier"},
		{"negative wait", func(g *GameServer) { g.GrandBosses.ValakasWaitTime = -1 }, "grandBosses.valakasWaitTime"},
		{"part of an hour", func(g *GameServer) { g.GrandBosses.IntervalOfCoreSpawn = Duration(90 * time.Minute) }, "grandBosses.intervalOfCoreSpawn"},
		{"random above interval", func(g *GameServer) { g.GrandBosses.RandomOfZakenSpawn = Duration(61 * time.Hour) }, "grandBosses.randomOfZakenSpawn"},
		{"animations off", func(g *GameServer) { g.World.MinNpcAnimation, g.World.MaxNpcAnimation = 0, 0 }, ""},
		{"min above max", func(g *GameServer) { g.World.MinMonsterAnimation = Duration(2 * time.Minute) }, "world.minMonsterAnimation"},
		{"negative animation", func(g *GameServer) { g.World.MinNpcAnimation = -1 }, "world.minNpcAnimation"},
		{"negative boat radius", func(g *GameServer) { g.World.BoatBroadcastRadius = -1 }, "world.boatBroadcastRadius"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(&cfg.GameServers[0])
			err := cfg.Validate()
			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate() = %v, want nil", err)
			case tt.want != "" && err == nil:
				t.Fatalf("Validate() = nil, want error containing %q", tt.want)
			case tt.want != "" && !strings.Contains(err.Error(), "gameservers[0]."+tt.want):
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.want)
			}
		})
	}
}
