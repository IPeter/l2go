package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// The world fidelity and AI settings (Plan 13): geodata, raid bosses and
// minions, grand bosses, NPC animations and boats.

// Geodata configures the geodata and pathfinding (Mobius GeoEngine.ini).
type Geodata struct {
	Comment string `json:"_comment,omitempty"`
	// Dir is the geodata folder of L2J "X_Y.l2j" tiles (Mobius
	// GeoDataPath). Empty means no geodata: heights come from the client
	// and the spawns, and everything is walkable and visible.
	Dir string `json:"dir"`
	// Pathfinding turns pathfinding on: walks go around walls (A*), and
	// clicks on closed cells are refused (Mobius PathFinding > 0). It is
	// off anyway when no tile loads.
	Pathfinding bool `json:"pathfinding"`
	// PathFindBuffers are the A* search windows, "size x count" separated
	// by ";" (Mobius PathFindBuffers). Only the sizes count, in cells; the
	// largest bounds how far a path reaches.
	PathFindBuffers string `json:"pathFindBuffers"`
	// LowWeight, MediumWeight and HighWeight are the A* costs of a plain
	// step, a step beside an obstacle and a step into one (Mobius
	// LowWeight, MediumWeight, HighWeight).
	LowWeight    float64 `json:"lowWeight"`
	MediumWeight float64 `json:"mediumWeight"`
	HighWeight   float64 `json:"highWeight"`
	// AdvancedDiagonalStrategy lets paths step diagonally, at
	// DiagonalWeight (Mobius AdvancedDiagonalStrategy, DiagonalWeight).
	AdvancedDiagonalStrategy bool    `json:"advancedDiagonalStrategy"`
	DiagonalWeight           float64 `json:"diagonalWeight"`
	// AvoidObstructedPathNodes makes straight walks check the plain
	// movement flags too (Mobius AvoidObstructedPathNodes).
	AvoidObstructedPathNodes bool `json:"avoidObstructedPathNodes"`
	// MaxPostfilterPasses is how many times a player's path is smoothed;
	// NPC paths are smoothed once; 0 never (Mobius MaxPostfilterPasses).
	MaxPostfilterPasses int `json:"maxPostfilterPasses"`
}

// DefaultGeodata returns the Mobius GeoEngine.ini values, without a
// geodata folder.
func DefaultGeodata() Geodata {
	return Geodata{
		Pathfinding:              true,
		PathFindBuffers:          "100x6;128x6;192x6;256x4;320x4;384x4;500x2",
		LowWeight:                0.5,
		MediumWeight:             2,
		HighWeight:               3,
		DiagonalWeight:           0.707,
		AvoidObstructedPathNodes: true,
		MaxPostfilterPasses:      3,
	}
}

// BufferSizes returns the window sizes of PathFindBuffers, as Mobius reads
// them: each ";"-separated entry up to its "x".
func (g Geodata) BufferSizes() ([]int, error) {
	var sizes []int
	for entry := range strings.SplitSeq(g.PathFindBuffers, ";") {
		size, _, _ := strings.Cut(strings.TrimSpace(entry), "x")
		n, err := strconv.Atoi(strings.TrimSpace(size))
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("%q is not a size x count", entry)
		}
		sizes = append(sizes, n)
	}
	return sizes, nil
}

// MinionRespawnTimes returns CustomMinionsRespawnTime as minion NPC ID →
// respawn time: entries "id,seconds" separated by ";", as in Mobius. An
// empty string has none.
func (n NPCs) MinionRespawnTimes() (map[int]time.Duration, error) {
	out := map[int]time.Duration{}
	for entry := range strings.SplitSeq(n.CustomMinionsRespawnTime, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		idText, secText, ok := strings.Cut(entry, ",")
		id, err1 := strconv.Atoi(strings.TrimSpace(idText))
		sec, err2 := strconv.Atoi(strings.TrimSpace(secText))
		if !ok || err1 != nil || err2 != nil || id <= 0 || sec < 0 {
			return nil, fmt.Errorf("%q is not npcId,seconds", entry)
		}
		out[id] = time.Duration(sec) * time.Second
	}
	return out, nil
}

// DefaultCustomMinionsRespawnTime is Mobius NPC.ini CustomMinionsRespawnTime.
const DefaultCustomMinionsRespawnTime = "22450,30;22371,120;22543,0;25545,0;22424,30;22425,30;22426,30;22427,30;22428,30;" +
	"22429,30;22430,30;22432,30;22433,30;22434,30;22435,30;22436,30;22437,30;22438,30;25596,30;25605,0;25606,0;25607,0;25608,0"

// GrandBosses configures the grand bosses (Mobius GrandBoss.ini). A boss
// respawns its interval plus or minus a random whole number of hours up to
// its random time after it died.
type GrandBosses struct {
	Comment string `json:"_comment,omitempty"`
	// AntharasWaitTime is how long Antharas waits after the first player
	// entered his lair; the others are his respawn interval and random
	// part, and whether heroes are greeted (Mobius AntharasWaitTime,
	// IntervalOfAntharasSpawn, RandomOfAntharasSpawn, AntharasRecognizeHero).
	AntharasWaitTime        Duration `json:"antharasWaitTime"`
	IntervalOfAntharasSpawn Duration `json:"intervalOfAntharasSpawn"`
	RandomOfAntharasSpawn   Duration `json:"randomOfAntharasSpawn"`
	AntharasRecognizeHero   bool     `json:"antharasRecognizeHero"`
	// The same for Valakas.
	ValakasWaitTime        Duration `json:"valakasWaitTime"`
	IntervalOfValakasSpawn Duration `json:"intervalOfValakasSpawn"`
	RandomOfValakasSpawn   Duration `json:"randomOfValakasSpawn"`
	ValakasRecognizeHero   bool     `json:"valakasRecognizeHero"`
	// Baium's respawn and hero greeting.
	IntervalOfBaiumSpawn Duration `json:"intervalOfBaiumSpawn"`
	RandomOfBaiumSpawn   Duration `json:"randomOfBaiumSpawn"`
	BaiumRecognizeHero   bool     `json:"baiumRecognizeHero"`
	// The respawns of Core, Orfen, Queen Ant and Zaken.
	IntervalOfCoreSpawn     Duration `json:"intervalOfCoreSpawn"`
	RandomOfCoreSpawn       Duration `json:"randomOfCoreSpawn"`
	IntervalOfOrfenSpawn    Duration `json:"intervalOfOrfenSpawn"`
	RandomOfOrfenSpawn      Duration `json:"randomOfOrfenSpawn"`
	IntervalOfQueenAntSpawn Duration `json:"intervalOfQueenAntSpawn"`
	RandomOfQueenAntSpawn   Duration `json:"randomOfQueenAntSpawn"`
	IntervalOfZakenSpawn    Duration `json:"intervalOfZakenSpawn"`
	RandomOfZakenSpawn      Duration `json:"randomOfZakenSpawn"`
}

// DefaultGrandBosses returns the Mobius GrandBoss.ini values.
func DefaultGrandBosses() GrandBosses {
	h := func(n int) Duration { return Duration(time.Duration(n) * time.Hour) }
	return GrandBosses{
		AntharasWaitTime:        Duration(20 * time.Minute),
		IntervalOfAntharasSpawn: h(264),
		RandomOfAntharasSpawn:   h(8),
		AntharasRecognizeHero:   true,
		ValakasWaitTime:         Duration(20 * time.Minute),
		IntervalOfValakasSpawn:  h(264),
		RandomOfValakasSpawn:    h(8),
		ValakasRecognizeHero:    true,
		IntervalOfBaiumSpawn:    h(168),
		RandomOfBaiumSpawn:      h(8),
		BaiumRecognizeHero:      true,
		IntervalOfCoreSpawn:     h(60),
		RandomOfCoreSpawn:       h(8),
		IntervalOfOrfenSpawn:    h(48),
		RandomOfOrfenSpawn:      h(8),
		IntervalOfQueenAntSpawn: h(36),
		RandomOfQueenAntSpawn:   h(8),
		IntervalOfZakenSpawn:    h(60),
		RandomOfZakenSpawn:      h(8),
	}
}

// validateWorldAI checks the geodata and grand boss sections and the raid,
// minion, animation and boat keys of npcs and world.
func (g GameServer) validateWorldAI() []error {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}
	nonNegDur := func(name string, d Duration) {
		if d < 0 {
			add("%s: %v is negative", name, time.Duration(d))
		}
	}
	nonNeg := func(name string, v int) {
		if v < 0 {
			add("%s: %d is negative", name, v)
		}
	}
	positiveF := func(name string, v float64) {
		if !(v > 0) || math.IsInf(v, 0) {
			add("%s: %v is not a positive number", name, v)
		}
	}
	nonNegF := func(name string, v float64) {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			add("%s: %v is not a finite number of 0 or more", name, v)
		}
	}
	hours := func(name string, d Duration) {
		if d < 0 || time.Duration(d)%time.Hour != 0 {
			add("%s: %v is not a whole number of hours", name, time.Duration(d))
		}
	}

	geo := g.Geodata
	if _, err := geo.BufferSizes(); err != nil {
		add("geodata.pathFindBuffers: %w", err)
	}
	positiveF("geodata.lowWeight", geo.LowWeight)
	positiveF("geodata.mediumWeight", geo.MediumWeight)
	positiveF("geodata.highWeight", geo.HighWeight)
	positiveF("geodata.diagonalWeight", geo.DiagonalWeight)
	nonNeg("geodata.maxPostfilterPasses", geo.MaxPostfilterPasses)

	n := g.NPCs
	nonNegDur("npcs.raidMinionRespawnTime", n.RaidMinionRespawnTime)
	if _, err := n.MinionRespawnTimes(); err != nil {
		add("npcs.customMinionsRespawnTime: %w", err)
	}
	nonNeg("npcs.raidChaosTime", n.RaidChaosTime)
	nonNeg("npcs.grandChaosTime", n.GrandChaosTime)
	nonNeg("npcs.minionChaosTime", n.MinionChaosTime)
	nonNegF("npcs.raidMinRespawnMultiplier", n.RaidMinRespawnMultiplier)
	nonNegF("npcs.raidMaxRespawnMultiplier", n.RaidMaxRespawnMultiplier)
	nonNegF("npcs.raidHpRegenMultiplier", n.RaidHPRegenMultiplier)
	nonNegF("npcs.raidMpRegenMultiplier", n.RaidMPRegenMultiplier)
	nonNegF("npcs.raidPDefenceMultiplier", n.RaidPDefenceMultiplier)
	nonNegF("npcs.raidMDefenceMultiplier", n.RaidMDefenceMultiplier)
	nonNegF("npcs.raidPAttackMultiplier", n.RaidPAttackMultiplier)
	nonNegF("npcs.raidMAttackMultiplier", n.RaidMAttackMultiplier)

	b := g.GrandBosses
	nonNegDur("grandBosses.antharasWaitTime", b.AntharasWaitTime)
	nonNegDur("grandBosses.valakasWaitTime", b.ValakasWaitTime)
	for _, r := range []struct {
		name             string
		interval, random Duration
	}{
		{"Antharas", b.IntervalOfAntharasSpawn, b.RandomOfAntharasSpawn},
		{"Valakas", b.IntervalOfValakasSpawn, b.RandomOfValakasSpawn},
		{"Baium", b.IntervalOfBaiumSpawn, b.RandomOfBaiumSpawn},
		{"Core", b.IntervalOfCoreSpawn, b.RandomOfCoreSpawn},
		{"Orfen", b.IntervalOfOrfenSpawn, b.RandomOfOrfenSpawn},
		{"QueenAnt", b.IntervalOfQueenAntSpawn, b.RandomOfQueenAntSpawn},
		{"Zaken", b.IntervalOfZakenSpawn, b.RandomOfZakenSpawn},
	} {
		hours("grandBosses.intervalOf"+r.name+"Spawn", r.interval)
		hours("grandBosses.randomOf"+r.name+"Spawn", r.random)
		if r.random > r.interval {
			add("grandBosses.randomOf%sSpawn: %v is above intervalOf%sSpawn %v (a respawn time would be in the past)",
				r.name, time.Duration(r.random), r.name, time.Duration(r.interval))
		}
	}

	w := g.World
	for _, a := range []struct {
		name     string
		min, max Duration
	}{
		{"Npc", w.MinNpcAnimation, w.MaxNpcAnimation},
		{"Monster", w.MinMonsterAnimation, w.MaxMonsterAnimation},
	} {
		nonNegDur("world.min"+a.name+"Animation", a.min)
		nonNegDur("world.max"+a.name+"Animation", a.max)
		if a.min > a.max {
			add("world.min%sAnimation: %v is above max%sAnimation %v", a.name, time.Duration(a.min), a.name, time.Duration(a.max))
		}
	}
	nonNeg("world.boatBroadcastRadius", w.BoatBroadcastRadius)
	return errs
}
