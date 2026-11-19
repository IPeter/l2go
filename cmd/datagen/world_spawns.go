package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// spawnXMLDir holds the spawn lists, in region folders (read recursively,
// as Mobius does).
const spawnXMLDir = "dist/game/data/spawns"

// readSpawns reads the spawn lists: the spawn blocks in file order, and the
// territories they define, sorted by name. Lists with enabled="false" are
// skipped. Entries of NPCs that are not in npcs (custom and fake player
// NPCs) are left out, as Mobius skips them; blocks left without entries
// are dropped.
func readSpawns(o options, npcs map[int]bool) ([]data.SpawnTerritory, []data.Spawn, error) {
	files, err := mxFilesUnder(o.src, spawnXMLDir)
	if err != nil {
		return nil, nil, err
	}
	var (
		terr     = territorySet{byName: make(map[string]int)}
		spawns   []data.Spawn
		dropped  = make(map[int]int) // NPC ID -> entries left out
		disabled []string
		errs     []error
	)
	for _, name := range files {
		rel := strings.TrimPrefix(name, spawnXMLDir+"/")
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, nil, err
		}
		list, err := mxRoot(doc)
		if err != nil {
			return nil, nil, err
		}
		// Mobius fails on a list without enabled.
		if enabled := list.boolAttr("enabled", false); !enabled {
			if _, ok := list.attr("enabled"); !ok {
				list.errorf("missing attribute enabled")
			}
			list.skip()
			disabled = append(disabled, rel)
		} else {
			for _, e := range list.children() {
				if e.name != "spawn" {
					e.unknown()
					continue
				}
				if s := parseSpawn(e, rel, &terr, npcs, dropped); len(s.Npcs) > 0 {
					spawns = append(spawns, s)
				}
			}
		}
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, nil, err
	}
	if len(disabled) > 0 {
		o.warnf("spawns: skipped disabled lists: %s", strings.Join(disabled, ", "))
	}
	if len(dropped) > 0 {
		ids := slices.Sorted(maps.Keys(dropped))
		total := 0
		for _, n := range dropped {
			total += n
		}
		o.warnf("spawns: left out %d entries of %d NPCs that are not in the NPC data (custom or fake player NPCs, which Mobius skips too): %s",
			total, len(ids), joinInts(ids, 20))
	}
	territories := terr.list
	slices.SortFunc(territories, func(a, b data.SpawnTerritory) int { return cmp.Compare(a.Name, b.Name) })
	return territories, spawns, nil
}

// territorySet collects the spawn territories of all files by name.
type territorySet struct {
	byName map[string]int
	list   []data.SpawnTerritory
}

// parseSpawn reads one <spawn> block, as Mobius's SpawnData does.
func parseSpawn(e *mxElem, file string, terr *territorySet, npcs map[int]bool, dropped map[int]int) data.Spawn {
	s := data.Spawn{File: file, Name: e.str("name", ""), Zone: e.str("zone", "")}
	var zones []string
	if v, ok := e.attr("zones"); ok {
		if s.Zone != "" {
			e.errorf("zone and zones are exclusive (Mobius ignores zones)")
		} else {
			zones = strings.Split(v, ";")
			s.Zone = zones[0]
		}
	}
	if v, ok := e.attr("selection"); ok {
		switch strings.ToLower(v) {
		case "all":
			s.Selection = data.SelectionAll
		case "random":
			s.Selection = data.SelectionRandom
		case "randomfill":
			s.Selection = data.SelectionRandomFill
		default:
			e.errorf("unknown selection %q", v)
		}
		s.MaximumNpc = e.intAttr("maximumNpc", 0)
		if s.Selection == data.SelectionRandom {
			s.MaximumNpc = 0 // Mobius ignores it for random
		}
		s.Zones = zones
	}
	sawNpc := false
	for _, c := range e.children() {
		switch c.name {
		case "AIData":
			if sawNpc {
				c.errorf("AIData after NPC entries (Mobius applies it only to the entries after it)")
			}
			for _, k := range c.children() {
				v := 0
				switch k.name {
				case "disableRandomAnimation", "disableRandomWalk":
					if k.textBool() {
						v = 1
					}
				default:
					v = k.textInt()
				}
				if s.AIData == nil {
					s.AIData = make(map[string]int)
				}
				s.AIData[k.name] = v
			}
		case "territory":
			name := c.str("name", s.Zone)
			if name == "" {
				c.errorf("territory without a name (no name attribute and no zone on its spawn)")
				c.skip()
				continue
			}
			if i, dup := terr.byName[name]; dup {
				c.errorf("territory %q is also defined in %s (Mobius keeps the first one it loads)", name, terr.list[i].File)
				c.skip()
				continue
			}
			terr.byName[name] = len(terr.list)
			terr.list = append(terr.list, data.SpawnTerritory{Name: name, File: file, Shape: parseSpawnShape(c)})
		case "banned_territory":
			i, ok := terr.byName[s.Zone]
			if !ok {
				c.errorf("banned_territory of %q before its territory (Mobius skips it)", s.Zone)
				c.skip()
				continue
			}
			terr.list[i].Banned = append(terr.list[i].Banned, parseSpawnShape(c))
		case "npc":
			sawNpc = true
			n := parseSpawnNpc(c, &s)
			if !npcs[n.NpcID] {
				dropped[n.NpcID]++
				continue
			}
			if s.Zone == "" && n.X == 0 && n.Y == 0 {
				c.errorf("NPC %d has neither coordinates nor a zone (Mobius skips it)", n.NpcID)
				continue
			}
			s.Npcs = append(s.Npcs, n)
		default:
			c.unknown()
		}
	}
	return s
}

// parseSpawnNpc reads one <npc> entry of a spawn block.
func parseSpawnNpc(e *mxElem, s *data.Spawn) data.SpawnNpc {
	n := data.SpawnNpc{
		NpcID:         e.reqInt("id"),
		Heading:       e.intAttr("heading", -1),
		Count:         e.intAttr("count", 1),
		RespawnDelay:  e.intAttr("respawnDelay", 0),
		RespawnRandom: e.intAttr("respawnRandom", 0),
		ChaseRange:    e.intAttr("chaseRange", 0),
	}
	// Mobius reads x, y and z together and leaves all three 0 when one is
	// missing; datagen wants all or none.
	_, hasX := e.attr("x")
	_, hasY := e.attr("y")
	_, hasZ := e.attr("z")
	switch {
	case hasX && hasY && hasZ:
		n.X, n.Y, n.Z = e.intAttr("x", 0), e.intAttr("y", 0), e.intAttr("z", 0)
	case hasX || hasY || hasZ:
		e.errorf("NPC %d: x, y and z go together", n.NpcID)
	}
	if v, ok := e.attr("periodOfDay"); ok {
		switch strings.ToLower(v) {
		case "day":
			n.PeriodOfDay = data.PeriodDay
		case "night":
			n.PeriodOfDay = data.PeriodNight
		default:
			e.errorf("unknown periodOfDay %q", v)
		}
		if s.Selection != "" {
			e.errorf("NPC %d: periodOfDay with an entry selection (Mobius skips the entry)", n.NpcID)
		}
	}
	return n
}

// parseSpawnShape reads a <territory> or <banned_territory>: nodes with x
// and y, an optional shape (NPoly by default) and, for a Cylinder, rad.
func parseSpawnShape(e *mxElem) data.Shape {
	minZ, maxZ := e.reqInt("minZ"), e.reqInt("maxZ")
	kind := e.str("shape", data.ShapeNPoly)
	rad := 0
	if strings.EqualFold(kind, data.ShapeCylinder) {
		rad = e.reqInt("rad")
	}
	var points []data.Point
	for _, c := range e.children() {
		if c.name != "node" {
			c.unknown()
			continue
		}
		points = append(points, data.Point{X: c.reqInt("x"), Y: c.reqInt("y")})
	}
	return mxShape(e, kind, points, minZ, maxZ, rad)
}

// mxShape builds a shape from a Mobius shape name (matched ignoring case,
// as Mobius does) and checks the point counts Mobius requires: at least 3
// for an NPoly, 2 for a Cuboid, 1 and a positive radius for a Cylinder.
// Mobius skips a shape that breaks them; datagen refuses it.
func mxShape(e *mxElem, kind string, points []data.Point, minZ, maxZ, rad int) data.Shape {
	s := data.Shape{Points: points, MinZ: minZ, MaxZ: maxZ}
	switch {
	case strings.EqualFold(kind, data.ShapeNPoly):
		s.Type = data.ShapeNPoly
		if len(points) < 3 {
			e.errorf("an NPoly needs at least 3 nodes (got %d)", len(points))
		}
	case strings.EqualFold(kind, data.ShapeCuboid):
		s.Type = data.ShapeCuboid
		if len(points) != 2 {
			e.errorf("a Cuboid needs exactly 2 nodes (got %d)", len(points))
		}
	case strings.EqualFold(kind, data.ShapeCylinder):
		s.Type = data.ShapeCylinder
		s.Radius = rad
		if len(points) != 1 || rad <= 0 {
			e.errorf("a Cylinder needs 1 node and a positive rad (got %d nodes, rad %d)", len(points), rad)
		}
	default:
		e.errorf("unknown shape %q", kind)
	}
	return s
}

// mxFilesUnder returns the .xml files below dir (recursively), sorted.
func mxFilesUnder(fsys fs.FS, dir string) ([]string, error) {
	var files []string
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(path.Ext(p), ".xml") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s: no XML files found", dir)
	}
	slices.Sort(files)
	return files, nil
}

// joinInts formats up to limit numbers, then how many more there are.
func joinInts(ids []int, limit int) string {
	var b strings.Builder
	for i, id := range ids {
		if i == limit {
			fmt.Fprintf(&b, " and %d more", len(ids)-limit)
			break
		}
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprint(&b, id)
	}
	return b.String()
}
