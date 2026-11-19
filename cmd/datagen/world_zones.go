package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// zoneXMLDir holds the zone lists. Only the files directly in it are read,
// as Mobius does; that includes the custom_*.xml files, which Mobius loads
// unconditionally (the town zones are in custom_town.xml).
const zoneXMLDir = "dist/game/data/zones"

// npcSpawnTerritory is the zone type Mobius treats as a spawn territory
// instead of a zone.
const npcSpawnTerritory = "NpcSpawnTerritory"

// readZoneTypes reads the zone type list from the Mobius Java sources.
// NpcSpawnTerritory lives in the same package but is not a zone type.
func readZoneTypes(o options) ([]data.MobiusClass, error) {
	classes, err := readMobiusClasses(o.src, zoneTypeJavaDir, zoneBaseJavaDir)
	if err != nil {
		return nil, err
	}
	for i := range classes {
		if classes[i].Name == npcSpawnTerritory {
			classes[i].Usable = false
		}
	}
	return classes, nil
}

// readZones reads the zones, sorted by ID. Zones without an ID get one from
// data.FirstDynamicZoneID on, in file name order (Mobius numbers them in
// its load order, which depends on the file system). Lists with
// enabled="false" are skipped, like tax.xml.
func readZones(o options, types []data.MobiusClass) ([]data.Zone, error) {
	names, err := fs.Glob(o.src, zoneXMLDir+"/*.xml")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: no zone files found", zoneXMLDir)
	}
	classes := make(map[string]data.MobiusClass, len(types))
	for _, c := range types {
		classes[c.Name] = c
	}
	is := func(name, class string) bool {
		for range len(classes) + 1 {
			if name == class {
				return true
			}
			c, ok := classes[name]
			if !ok {
				return false
			}
			name = c.Parent
		}
		return false
	}
	var (
		zones    []data.Zone
		byID     = make(map[int]string) // zone ID -> file
		nextID   = data.FirstDynamicZoneID
		disabled []string
		errs     []error
	)
	for _, name := range names {
		file := path.Base(name)
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		list, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		if !list.boolAttr("enabled", true) {
			list.skip()
			disabled = append(disabled, file)
		} else {
			for _, e := range list.children() {
				if e.name != "zone" {
					e.unknown()
					continue
				}
				z, ok := parseZone(e, file, &nextID, classes, is)
				if !ok {
					continue
				}
				if prev, dup := byID[z.ID]; dup {
					e.errorf("zone ID %d is also used in %s (Mobius overrides the first)", z.ID, prev)
					continue
				}
				byID[z.ID] = file
				zones = append(zones, z)
			}
		}
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if len(disabled) > 0 {
		o.warnf("zones: skipped disabled lists: %s", strings.Join(disabled, ", "))
	}
	slices.SortFunc(zones, func(a, b data.Zone) int { return cmp.Compare(a.ID, b.ID) })
	return zones, nil
}

// parseZone reads one <zone>, as Mobius's ZoneManager does.
func parseZone(e *mxElem, file string, nextID *int, classes map[string]data.MobiusClass, is func(name, class string) bool) (data.Zone, bool) {
	typ := e.reqStr("type")
	if typ == npcSpawnTerritory {
		e.errorf("NpcSpawnTerritory zones are not supported (the Interlude data has none)")
		e.skip()
		return data.Zone{}, false
	}
	z := data.Zone{Type: typ, File: file, Name: e.str("name", "")}
	if v, ok := e.attr("id"); ok {
		id, err := parseJavaInt(v) // Mobius: Integer.parseInt
		if err != nil {
			e.errorf("attribute id: %v", err)
		}
		z.ID = id
	} else {
		z.ID = *nextID
		*nextID++
	}
	if c, ok := classes[typ]; !ok || !c.Usable {
		e.errorf("unknown zone type %q (Mobius skips the zone)", typ)
	}
	minZ, maxZ := e.reqInt("minZ"), e.reqInt("maxZ")
	kind := e.reqStr("shape")
	rad := 0
	if strings.EqualFold(kind, data.ShapeCylinder) {
		if v, ok := e.attr("rad"); ok {
			r, err := parseJavaInt(v) // Mobius: Integer.parseInt
			if err != nil {
				e.errorf("attribute rad: %v", err)
			}
			rad = r
		}
	}
	var points []data.Point
	for _, c := range e.children() {
		switch c.name {
		case "node":
			points = append(points, data.Point{X: c.reqInt("X"), Y: c.reqInt("Y")})
		case "stat":
			z.Params = append(z.Params, data.ZoneParam{Name: c.reqStr("name"), Value: c.reqStr("val")})
		case "spawn":
			if !is(typ, "ZoneRespawn") {
				c.errorf("spawn point on a %s, which Mobius ignores", typ)
				c.skip()
				continue
			}
			s := data.ZoneSpawn{X: zoneSpawnInt(c, "X"), Y: zoneSpawnInt(c, "Y"), Z: zoneSpawnInt(c, "Z")}
			switch t := c.str("type", ""); {
			case t == data.ZoneSpawnNormal, t == data.ZoneSpawnOther, t == data.ZoneSpawnChaotic, t == data.ZoneSpawnBanish:
				s.Type = t
			case t == data.ZoneSpawnChallenger && is(typ, "SiegableHallZone"):
				s.Type = t // SiegableHallZone.parseLoc reads these
			default:
				c.errorf("unknown spawn point type %q for a %s (Mobius ignores the point)", t, typ)
				continue
			}
			z.Spawns = append(z.Spawns, s)
		case "race":
			if !is(typ, "RespawnZone") {
				c.errorf("race point on a %s, which Mobius ignores", typ)
				c.skip()
				continue
			}
			name, point := c.reqStr("name"), c.reqStr("point")
			race, ok := data.ParseRace(name) // Mobius: Race.valueOf
			if !ok {
				c.errorf("unknown race %q", name)
				continue
			}
			if _, dup := z.RacePoints[race]; dup {
				c.errorf("race %s is given twice (Mobius keeps the last)", name)
				continue
			}
			if z.RacePoints == nil {
				z.RacePoints = make(map[data.Race]string)
			}
			z.RacePoints[race] = point
		default:
			c.unknown()
		}
	}
	z.Shape = mxShape(e, kind, points, minZ, maxZ, rad)
	return z, true
}

// zoneSpawnInt reads a coordinate of a zone <spawn>, which Mobius parses
// with Integer.parseInt and requires.
func zoneSpawnInt(e *mxElem, name string) int {
	v, ok := e.attr(name)
	if !ok {
		e.errorf("missing attribute %s", name)
		return 0
	}
	n, err := parseJavaInt(v)
	if err != nil {
		e.errorf("attribute %s: %v", name, err)
	}
	return n
}
