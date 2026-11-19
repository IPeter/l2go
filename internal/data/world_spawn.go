package data

import "fmt"

// Spawn periods of day (SpawnNpc.PeriodOfDay).
const (
	PeriodDay   = "day"
	PeriodNight = "night"
)

// Spawn selections of a spawn group (Spawn.Selection), unused by the
// Interlude spawn lists.
const (
	SelectionAll        = "all"
	SelectionRandom     = "random"
	SelectionRandomFill = "randomFill"
)

// Spawn is one <spawn> block of the Mobius spawn lists
// (dist/game/data/spawns/**/*.xml): NPC entries that share a territory, a
// name and AI data.
type Spawn struct {
	// File is the source file below dist/game/data/spawns, with its region
	// folder (for example "TalkingIsland/TalkingIslandNPCs.xml").
	File string `json:"file"`
	// Name is the spawn's name; Mobius keys AIData by it.
	Name string `json:"name,omitempty"`
	// Zone names the SpawnTerritory the NPCs spawn in, at a random point.
	// Without it every NPC entry has fixed coordinates.
	Zone string `json:"zone,omitempty"`
	// Zones, Selection and MaximumNpc describe a spawn group, which picks
	// entries (all, random or randomFill) and spawns one slot per
	// territory; Zones is set only with a Selection. Unused in Interlude.
	Zones      []string `json:"zones,omitempty"`
	Selection  string   `json:"selection,omitempty"`
	MaximumNpc int      `json:"maximumNpc,omitempty"`
	// AIData overrides AI values for the spawned NPCs, as Mobius stores
	// them: disableRandomAnimation and disableRandomWalk as 1 or 0, the
	// others (aggroRange, fleeX, fleeY, fleeZ) as numbers. Kept raw.
	AIData map[string]int `json:"aiData,omitempty"`
	Npcs   []SpawnNpc     `json:"npcs"`
}

// SpawnNpc is one <npc> entry of a spawn block. Entries of NPCs that do not
// exist (such as custom or fake-player NPCs) are left out by cmd/datagen,
// as Mobius skips them.
type SpawnNpc struct {
	NpcID int `json:"npcId"`
	// X, Y, Z is the fixed spawn point. Mobius treats X and Y both 0 as "no
	// point"; a spawn with a Zone uses a random point of the territory
	// instead, even when a point is given.
	X int `json:"x,omitempty"`
	Y int `json:"y,omitempty"`
	Z int `json:"z,omitempty"`
	// Heading is the fixed heading, or -1 for a random one (Mobius uses
	// Rnd.get(61794)); territory spawns always get a random heading. Some
	// fixed headings are negative (such as -16384); Mobius uses them as
	// they are.
	Heading int `json:"heading,omitempty"`
	// Count is how many NPCs this entry keeps alive.
	Count int `json:"count"`
	// RespawnDelay and RespawnRandom are in seconds. Mobius respawns after
	// a uniform time in [delay-random, delay+random] (each end at least 1
	// second); a delay of 0 means no respawn.
	RespawnDelay  int `json:"respawnDelay,omitempty"`
	RespawnRandom int `json:"respawnRandom,omitempty"`
	ChaseRange    int `json:"chaseRange,omitempty"`
	// PeriodOfDay is PeriodDay or PeriodNight for NPCs that only exist at
	// that time of the game day (Mobius DayNightSpawnManager), or empty.
	PeriodOfDay string `json:"periodOfDay,omitempty"`
}

// SpawnTerritory is a named area that spawns pick random points in
// (Mobius NpcSpawnTerritory). Mobius keeps the first territory of a name;
// cmd/datagen refuses duplicates.
type SpawnTerritory struct {
	Name  string `json:"name"`
	File  string `json:"file"`
	Shape Shape  `json:"shape"`
	// Banned are areas inside the territory where NPCs must not spawn.
	Banned []Shape `json:"banned,omitempty"`
}

// RandomPoint returns a random spawn point in the territory, as Mobius's
// NpcSpawnTerritory.getRandomPoint does without geodata: with banned areas
// it tries 100 points, then 100 more, for one outside every banned area;
// failing that, and without banned areas, it takes the shape's next random
// point. See Shape.RandomPoint for the point and its Z, which is the middle
// of the territory's Z band (decision N2).
func (t *SpawnTerritory) RandomPoint(rnd RandInt) Location {
	if len(t.Banned) > 0 {
		// Mobius runs this search twice: once with a geodata height check
		// that never fails without geodata, once without it.
		for range 2 {
			for range 100 {
				loc := t.Shape.RandomPoint(rnd)
				if !t.inBanned(loc) {
					return loc
				}
			}
		}
	}
	// The plain loop: without geodata its height check never fails, so the
	// first point is taken.
	return t.Shape.RandomPoint(rnd)
}

func (t *SpawnTerritory) inBanned(loc Location) bool {
	for _, b := range t.Banned {
		if b.Contains(loc.X, loc.Y, loc.Z) {
			return true
		}
	}
	return false
}

// Territory returns the spawn territory with the given name.
func (w *World) Territory(name string) (*SpawnTerritory, bool) {
	i, ok := w.territories[name]
	if !ok {
		return nil, false
	}
	return &w.Territories[i], true
}

// checkTerritories checks the spawn territories (unique names, shapes
// Mobius accepts) and indexes them.
func (w *World) checkTerritories(p *problems) {
	for i := range w.Territories {
		t := &w.Territories[i]
		where := fmt.Sprintf("%s: territory %q (%s)", TerritoriesFile, t.Name, t.File)
		if t.Name == "" {
			p.addf("%s: no name", where)
			continue
		}
		if _, dup := w.territories[t.Name]; dup {
			p.addf("%s: duplicate territory name", where)
			continue
		}
		w.territories[t.Name] = i
		t.Shape.check(p, where)
		for j, b := range t.Banned {
			b.check(p, fmt.Sprintf("%s: banned area %d", where, j))
		}
	}
}

// checkSpawns checks the spawn blocks:
//
//   - each names existing territories, has NPC entries, and a valid
//     selection if any;
//   - each entry names an existing NPC, has a positive count, no negative
//     delays or chase range, a valid period of day, and a point (X and Y
//     not both 0) when the block has no zone.
func (w *World) checkSpawns(p *problems) {
	if len(w.Spawns) == 0 {
		p.addf("%s: no spawns", SpawnsFile)
	}
	for i := range w.Spawns {
		s := &w.Spawns[i]
		where := fmt.Sprintf("%s: spawn %d (%s", SpawnsFile, i, s.File)
		if s.Name != "" {
			where += " " + s.Name
		}
		where += ")"
		if s.Zone != "" {
			if _, ok := w.territories[s.Zone]; !ok {
				p.addf("%s: territory %q does not exist in %s", where, s.Zone, TerritoriesFile)
			}
		}
		for _, z := range s.Zones {
			if _, ok := w.territories[z]; !ok {
				p.addf("%s: territory %q does not exist in %s", where, z, TerritoriesFile)
			}
		}
		switch s.Selection {
		case "", SelectionAll, SelectionRandom, SelectionRandomFill:
		default:
			p.addf("%s: unknown selection %q", where, s.Selection)
		}
		if len(s.Zones) > 0 && s.Selection == "" {
			p.addf("%s: zones without a selection", where)
		}
		if len(s.Npcs) == 0 {
			p.addf("%s: no NPC entries", where)
		}
		for _, n := range s.Npcs {
			at := fmt.Sprintf("%s: NPC %d", where, n.NpcID)
			if _, ok := w.npcs[n.NpcID]; !ok {
				p.addf("%s: does not exist in %s", at, NpcsFile)
			}
			if n.Count <= 0 {
				p.addf("%s: count must be positive (got %d)", at, n.Count)
			}
			if n.RespawnDelay < 0 || n.RespawnRandom < 0 || n.ChaseRange < 0 {
				p.addf("%s: respawn delay, respawn random and chase range must not be negative", at)
			}
			switch n.PeriodOfDay {
			case "", PeriodDay, PeriodNight:
			default:
				p.addf("%s: unknown period of day %q", at, n.PeriodOfDay)
			}
			if s.Zone == "" && n.X == 0 && n.Y == 0 {
				p.addf("%s: neither a point nor a territory", at)
			}
		}
	}
}
