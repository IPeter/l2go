package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The world domains: "world" converts the NPC templates, spawns,
// teleporters, zones and map regions (loaded by data.LoadWorld), "html"
// copies the dialog pages.
func init() {
	registerDomain("world", genWorld)
	registerDomain("html", genHTML)
}

// genWorld reads the world data from the Mobius project, checks it with
// data.NewWorld (references between the parts included) and adds its files:
//
//   - npcs/types.json and zones/types.json: the NPC and zone classes, from
//     the Java sources;
//   - npcs/npcs.json: the NPC templates (not custom/), one per line;
//   - spawns/territories.json and spawns/spawns.json, one per line;
//   - teleporters/teleporters.json;
//   - zones/zones.json, one per line;
//   - mapregions/mapregions.json.
//
// Files over the size limit are stored gzipped (see output.addSized).
func genWorld(o options) error {
	d, err := readWorld(o)
	if err != nil {
		return err
	}
	if _, err := data.NewWorld(*d); err != nil {
		return err
	}
	if err := o.out.addJSON(data.NpcTypesFile, d.NpcTypes); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.NpcsFile, d.Npcs); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.TerritoriesFile, d.Territories); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.SpawnsFile, d.Spawns); err != nil {
		return err
	}
	if err := o.out.addJSON(data.TeleportersFile, d.Teleporters); err != nil {
		return err
	}
	if err := o.out.addJSON(data.ZoneTypesFile, d.ZoneTypes); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.ZonesFile, d.Zones); err != nil {
		return err
	}
	if err := o.out.addJSON(data.MapRegionsFile, d.MapRegions); err != nil {
		return err
	}
	entries, locations := 0, 0
	for _, s := range d.Spawns {
		entries += len(s.Npcs)
	}
	for _, t := range d.Teleporters {
		for _, l := range t.Lists {
			locations += len(l.Locations)
		}
	}
	fmt.Fprintf(o.log, "world: %d NPC templates, %d spawn blocks with %d entries, %d territories, %d teleporters with %d locations, %d zones, %d map regions\n",
		len(d.Npcs), len(d.Spawns), entries, len(d.Territories), len(d.Teleporters), locations, len(d.Zones), len(d.MapRegions))
	return nil
}

// readWorld reads all parts of the world data.
func readWorld(o options) (*data.WorldData, error) {
	var (
		d   data.WorldData
		err error
	)
	if d.NpcTypes, err = readNpcTypes(o); err != nil {
		return nil, fmt.Errorf("NPC types: %w", err)
	}
	if d.Npcs, err = readNpcs(o); err != nil {
		return nil, fmt.Errorf("NPCs: %w", err)
	}
	npcs := make(map[int]bool, len(d.Npcs))
	for _, n := range d.Npcs {
		npcs[n.ID] = true
	}
	if d.Territories, d.Spawns, err = readSpawns(o, npcs); err != nil {
		return nil, fmt.Errorf("spawns: %w", err)
	}
	if d.Teleporters, err = readTeleporters(o, npcs); err != nil {
		return nil, fmt.Errorf("teleporters: %w", err)
	}
	if d.ZoneTypes, err = readZoneTypes(o); err != nil {
		return nil, fmt.Errorf("zone types: %w", err)
	}
	if d.Zones, err = readZones(o, d.ZoneTypes); err != nil {
		return nil, fmt.Errorf("zones: %w", err)
	}
	if d.MapRegions, err = readMapRegions(o); err != nil {
		return nil, fmt.Errorf("map regions: %w", err)
	}
	warnMissingRegions(o, &d)
	return &d, nil
}

// warnMissingRegions reports the map regions that respawn zones name but
// that do not exist; Mobius uses its default respawn region for them.
func warnMissingRegions(o options, d *data.WorldData) {
	regions := make(map[string]bool, len(d.MapRegions))
	for _, r := range d.MapRegions {
		regions[r.Name] = true
	}
	missing := make(map[string][]string) // region -> zones naming it
	for _, z := range d.Zones {
		for _, region := range z.RacePoints {
			if !regions[region] && !slices.Contains(missing[region], z.Name) {
				missing[region] = append(missing[region], z.Name)
			}
		}
	}
	for _, region := range slices.Sorted(maps.Keys(missing)) {
		o.warnf("zones: map region %q does not exist; Mobius uses %s for the respawn zones naming it: %s",
			region, data.DefaultRespawnRegion, strings.Join(missing[region], ", "))
	}
}
