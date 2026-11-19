package data

// World data: the NPC templates, spawns, teleporters, zones and map regions
// that cmd/datagen converts from the L2J Mobius Interlude datapack. It is
// loaded with LoadWorld, separately from the player data of Load.

import (
	"fmt"
	"io/fs"
	"os"
)

// Paths of the world data files, relative to the data folder. cmd/datagen
// stores a file bigger than 10 MB gzipped, under its name plus ".gz"; the
// loader reads either form (but refuses to guess when both exist).
const (
	NpcTypesFile    = "npcs/types.json"
	NpcsFile        = "npcs/npcs.json"
	TerritoriesFile = "spawns/territories.json"
	SpawnsFile      = "spawns/spawns.json"
	TeleportersFile = "teleporters/teleporters.json"
	ZoneTypesFile   = "zones/types.json"
	ZonesFile       = "zones/zones.json"
	MapRegionsFile  = "mapregions/mapregions.json"
)

// WorldData is the content of the world data files, as decoded from JSON.
type WorldData struct {
	NpcTypes    []MobiusClass    // NpcTypesFile, sorted by name
	Npcs        []NpcTemplate    // NpcsFile, sorted by ID
	Territories []SpawnTerritory // TerritoriesFile, sorted by name
	Spawns      []Spawn          // SpawnsFile, in file order
	Teleporters []Teleporter     // TeleportersFile, sorted by NPC ID
	ZoneTypes   []MobiusClass    // ZoneTypesFile, sorted by name
	Zones       []Zone           // ZonesFile, sorted by ID
	MapRegions  []MapRegion      // MapRegionsFile, sorted by name
}

// worldFiles pairs each world file with the field it decodes into.
func (d *WorldData) worldFiles() []struct {
	name string
	v    any
} {
	return []struct {
		name string
		v    any
	}{
		{NpcTypesFile, &d.NpcTypes},
		{NpcsFile, &d.Npcs},
		{TerritoriesFile, &d.Territories},
		{SpawnsFile, &d.Spawns},
		{TeleportersFile, &d.Teleporters},
		{ZoneTypesFile, &d.ZoneTypes},
		{ZonesFile, &d.Zones},
		{MapRegionsFile, &d.MapRegions},
	}
}

// World is the checked world data with its lookups. The exported slices
// keep the order of the files. A World must not be modified after NewWorld
// or LoadWorld returns, and is then safe for concurrent use; the pointers
// its lookups return point into it and must not be modified either.
type World struct {
	WorldData

	npcTypes    classTree
	zoneTypes   classTree
	npcs        map[int]int    // NPC ID -> index in Npcs
	territories map[string]int // name -> index in Territories
	teleporters map[int]int    // NPC ID -> index in Teleporters
	zones       map[int]int    // zone ID -> index in Zones
	regions     map[string]int // name -> index in MapRegions
	// regionGrid holds, per map cell, the index+1 of its region in
	// MapRegions (0: no region), like Mobius's spatial index.
	regionGrid [MapRegionGridSize][MapRegionGridSize]int
}

// LoadWorld reads and checks the world data files in dir.
func LoadWorld(dir string) (*World, error) {
	w, err := LoadWorldFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return w, nil
}

// LoadWorldFS reads and checks the world data files in fsys. Each file may
// be stored as name.json or name.json.gz (see readJSON).
func LoadWorldFS(fsys fs.FS) (*World, error) {
	var d WorldData
	for _, f := range d.worldFiles() {
		if err := readJSON(fsys, f.name, f.v); err != nil {
			return nil, err
		}
	}
	return NewWorld(d)
}

// NewWorld checks the world data and builds its lookups. The slices are
// used as they are, not copied. Every part is checked, then the references
// between parts: spawns, minions and teleporters name existing NPCs, spawns
// existing territories, banned races existing map regions. (Respawn zones
// may name missing regions: Mobius then uses the default one.) The rules of
// each part are documented on its check function.
//
// All problems are reported together in one error wrapping ErrInvalid
// (with a long list cut short).
func NewWorld(d WorldData) (*World, error) {
	w := &World{
		WorldData:   d,
		npcs:        make(map[int]int, len(d.Npcs)),
		territories: make(map[string]int, len(d.Territories)),
		teleporters: make(map[int]int, len(d.Teleporters)),
		zones:       make(map[int]int, len(d.Zones)),
		regions:     make(map[string]int, len(d.MapRegions)),
	}
	var p problems
	w.npcTypes = newClassTree(&p, NpcTypesFile, d.NpcTypes)
	w.zoneTypes = newClassTree(&p, ZoneTypesFile, d.ZoneTypes)
	w.checkNpcs(&p)
	w.checkTerritories(&p)
	w.checkSpawns(&p)
	w.checkTeleporters(&p)
	w.checkMapRegions(&p)
	w.checkZones(&p)
	if err := p.err(); err != nil {
		return nil, err
	}
	return w, nil
}

// MobiusClass is a Java class of Mobius that the data names by its simple
// name: an NPC type (NpcTemplate.Type, a class of entity/actor/instance) or
// a zone type (Zone.Type, a class of entity/zone/type). The list holds the
// classes they extend too, so code can ask "is it a Monster" the way Mobius
// uses instanceof (see World.NpcTypeIs and World.ZoneTypeIs).
type MobiusClass struct {
	Name string `json:"name"`
	// Parent is the class it extends. The chain ends at an empty parent or
	// at a parent that is not in the list (such as WorldObject).
	Parent   string `json:"parent,omitempty"`
	Abstract bool   `json:"abstract,omitempty"`
	// Usable is set on the classes that data may name: the non-abstract
	// classes of the package Mobius instantiates them from by name.
	Usable bool `json:"usable,omitempty"`
}

// classTree indexes a class list by name.
type classTree map[string]*MobiusClass

// newClassTree checks a class list (unique names, no inheritance cycle)
// and indexes it.
func newClassTree(p *problems, file string, classes []MobiusClass) classTree {
	t := make(classTree, len(classes))
	for i := range classes {
		c := &classes[i]
		if c.Name == "" {
			p.addf("%s: class %d has no name", file, i)
			continue
		}
		if _, dup := t[c.Name]; dup {
			p.addf("%s: class %s: duplicate name", file, c.Name)
			continue
		}
		t[c.Name] = c
	}
	if len(t) == 0 {
		p.addf("%s: no classes", file)
	}
	for i := range classes {
		name, steps := classes[i].Name, 0
		for c, ok := t[name]; ok && c.Parent != ""; c, ok = t[c.Parent] {
			if steps++; steps > len(t) {
				p.addf("%s: class %s: inheritance cycle", file, name)
				break
			}
		}
	}
	return t
}

// is reports whether class name is class or extends it.
func (t classTree) is(name, class string) bool {
	for range len(t) + 1 {
		if name == class {
			return true
		}
		c, ok := t[name]
		if !ok {
			return false
		}
		name = c.Parent
	}
	return false
}

// usable reports whether data may name class name.
func (t classTree) usable(name string) bool {
	c, ok := t[name]
	return ok && c.Usable
}

// NpcTypeIs reports whether the NPC type typ (an NpcTemplate.Type) is class
// or extends it, like a Java instanceof on the Mobius NPC object: for
// example NpcTypeIs("RaidBoss", "Monster") and NpcTypeIs("Teleporter",
// "Folk") are true.
func (w *World) NpcTypeIs(typ, class string) bool { return w.npcTypes.is(typ, class) }

// ZoneTypeIs reports whether the zone type typ (a Zone.Type) is class or
// extends it, for example ZoneTypeIs("CastleZone", "ZoneRespawn").
func (w *World) ZoneTypeIs(typ, class string) bool { return w.zoneTypes.is(typ, class) }

// setOf returns a set holding the given strings.
func setOf(values ...string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, v := range values {
		m[v] = true
	}
	return m
}
