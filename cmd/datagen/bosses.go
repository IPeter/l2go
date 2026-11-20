package main

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/IPeter/l2go/internal/data"
)

// The boss sources: the SQL tables Mobius's RaidBossSpawnManager and
// GrandBossManager start from, and the boss zone list.
const (
	raidSpawnSQL = "dist/db_installer/sql/game/raidboss_spawnlist.sql"
	grandBossSQL = "dist/db_installer/sql/game/grandboss_data.sql"
	bossZoneXML  = "dist/game/data/zones/custom_boss.xml"
)

// The bosses domain converts the raid boss spawns, the grand boss seeds and
// the boss zone parameters, loaded by data.LoadBosses.
func init() { registerDomain("bosses", genBosses) }

// genBosses reads the boss data, checks it with data.NewBosses and against
// the NPC templates, and adds spawns/raidbosses.json,
// bosses/grandbosses.json and bosses/zones.json, one record per line.
func genBosses(o options) error {
	var (
		d   data.BossData
		err error
	)
	if d.RaidSpawns, err = readRaidSpawns(o); err != nil {
		return fmt.Errorf("raid spawns: %w", err)
	}
	if d.GrandBosses, err = readGrandBosses(o); err != nil {
		return fmt.Errorf("grand bosses: %w", err)
	}
	if d.Zones, err = readBossZones(o); err != nil {
		return fmt.Errorf("boss zones: %w", err)
	}
	if _, err := data.NewBosses(d); err != nil {
		return err
	}
	if err := checkBossNpcs(o, &d); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.RaidSpawnsFile, d.RaidSpawns); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.GrandBossesFile, d.GrandBosses); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.BossZonesFile, d.Zones); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "bosses: %d raid boss spawns, %d grand boss seeds, %d boss zones\n",
		len(d.RaidSpawns), len(d.GrandBosses), len(d.Zones))
	return nil
}

// readRaidSpawns reads the raidboss_spawnlist rows, sorted by boss ID (as
// Mobius selects them). Rows commented out are not read. A boss listed
// twice fails (INSERT IGNORE would keep the first).
func readRaidSpawns(o options) ([]data.RaidSpawn, error) {
	t, err := readSQLTable(o.src, raidSpawnSQL, "raidboss_spawnlist")
	if err != nil {
		return nil, err
	}
	var (
		spawns []data.RaidSpawn
		seen   = make(map[int]bool)
		errs   sqlErrors
	)
	for _, r := range t.rows {
		s := data.RaidSpawn{
			BossID:        errs.int(r, "boss_id"),
			X:             errs.int(r, "loc_x"),
			Y:             errs.int(r, "loc_y"),
			Z:             errs.int(r, "loc_z"),
			Heading:       errs.int(r, "heading"),
			RespawnDelay:  errs.int(r, "respawn_delay"),
			RespawnRandom: errs.int(r, "respawn_random"),
			HP:            errs.float(r, "currentHp"),
			MP:            errs.float(r, "currentMp"),
		}
		// A raid boss spawn holds one boss, and the seed has no respawn time.
		if n := errs.int(r, "amount"); n != 1 {
			errs.add(r.errorf("boss %d: amount %d, want 1", s.BossID, n))
		}
		if n := errs.long(r, "respawn_time"); n != 0 {
			errs.add(r.errorf("boss %d: respawn_time %d, want 0", s.BossID, n))
		}
		if seen[s.BossID] {
			errs.add(r.errorf("boss %d is listed twice (INSERT IGNORE keeps the first)", s.BossID))
			continue
		}
		seen[s.BossID] = true
		spawns = append(spawns, s)
	}
	if err := errs.err(); err != nil {
		return nil, err
	}
	if len(spawns) == 0 {
		return nil, fmt.Errorf("%s: no rows", raidSpawnSQL)
	}
	slices.SortFunc(spawns, func(a, b data.RaidSpawn) int { return cmp.Compare(a.BossID, b.BossID) })
	return spawns, nil
}

// readGrandBosses reads the grandboss_data rows, sorted by boss ID.
func readGrandBosses(o options) ([]data.GrandBoss, error) {
	t, err := readSQLTable(o.src, grandBossSQL, "grandboss_data")
	if err != nil {
		return nil, err
	}
	var (
		bosses []data.GrandBoss
		seen   = make(map[int]bool)
		errs   sqlErrors
	)
	for _, r := range t.rows {
		g := data.GrandBoss{
			BossID:      errs.int(r, "boss_id"),
			X:           errs.int(r, "loc_x"),
			Y:           errs.int(r, "loc_y"),
			Z:           errs.int(r, "loc_z"),
			Heading:     errs.int(r, "heading"),
			RespawnTime: errs.long(r, "respawn_time"),
			HP:          errs.float(r, "currentHP"),
			MP:          errs.float(r, "currentMP"),
			Status:      errs.int(r, "status"),
		}
		if seen[g.BossID] {
			errs.add(r.errorf("boss %d is listed twice (INSERT IGNORE keeps the first)", g.BossID))
			continue
		}
		seen[g.BossID] = true
		bosses = append(bosses, g)
	}
	if err := errs.err(); err != nil {
		return nil, err
	}
	if len(bosses) == 0 {
		return nil, fmt.Errorf("%s: no rows", grandBossSQL)
	}
	slices.SortFunc(bosses, func(a, b data.GrandBoss) int { return cmp.Compare(a.BossID, b.BossID) })
	return bosses, nil
}

// readBossZones reads the parameters of the BossZones of custom_boss.xml
// (the only zone file with boss zones), sorted by zone ID, as
// BossZone.setParameter reads them. Their shapes are left to the world
// domain.
func readBossZones(o options) ([]data.BossZone, error) {
	doc, err := parseMX(o.src, bossZoneXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	zones := []data.BossZone{}
	if !root.boolAttr("enabled", true) {
		// Mobius skips a disabled list, as the world domain does.
		o.warnf("boss zones: %s is disabled", bossZoneXML)
		root.skip()
		return zones, nil
	}
	for _, e := range root.children() {
		if e.name != "zone" {
			e.unknown()
			continue
		}
		e.ignore("shape", "minZ", "maxZ", "rad")
		if typ := e.reqStr("type"); typ != "BossZone" {
			e.errorf("zone type %s, want BossZone", typ)
		}
		z := data.BossZone{ZoneID: e.reqInt("id"), Name: e.str("name", ""), Enabled: true}
		var oust [3]int
		hasOust := false
		for _, c := range e.children() {
			switch c.name {
			case "node":
				c.skip()
			case "stat":
				name, val := c.reqStr("name"), c.reqStr("val")
				integer := func() int {
					n, err := parseJavaInt(val) // Mobius: Integer.parseInt
					if err != nil {
						c.errorf("%s: %v", name, err)
					}
					return n
				}
				switch name {
				case "InvadeTime":
					z.InvadeTime = integer()
				case "oustX", "oustY", "oustZ":
					oust[name[4]-'X'] = integer()
					hasOust = true
				case "default_enabled":
					b, err := parseJavaBool(val)
					if err != nil {
						c.errorf("%s: %v", name, err)
					}
					z.Enabled = b
				default:
					c.errorf("unknown boss zone parameter %s", name)
				}
			default:
				c.unknown()
			}
		}
		if hasOust {
			z.Oust = &data.Location{X: oust[0], Y: oust[1], Z: oust[2]}
		}
		zones = append(zones, z)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	slices.SortFunc(zones, func(a, b data.BossZone) int { return cmp.Compare(a.ZoneID, b.ZoneID) })
	return zones, nil
}

// checkBossNpcs checks that every raid spawn names a RaidBoss NPC (Mobius
// casts the spawned NPC to RaidBoss) and every grand boss seed an NPC of
// the NPC data. The NPC warnings are left to the world domain.
func checkBossNpcs(o options, d *data.BossData) error {
	quiet := o
	quiet.log = io.Discard
	npcs, err := readNpcs(quiet)
	if err != nil {
		return fmt.Errorf("NPCs: %w", err)
	}
	byID := make(map[int]*data.NpcTemplate, len(npcs))
	for i := range npcs {
		byID[npcs[i].ID] = &npcs[i]
	}
	var p []error
	for _, r := range d.RaidSpawns {
		n, ok := byID[r.BossID]
		switch {
		case !ok:
			p = append(p, fmt.Errorf("%s: raid boss %d is not in the NPC data", raidSpawnSQL, r.BossID))
		case n.Type != "RaidBoss":
			p = append(p, fmt.Errorf("%s: raid boss %d (%s) is a %s, not a RaidBoss", raidSpawnSQL, r.BossID, n.Name, n.Type))
		}
	}
	for _, g := range d.GrandBosses {
		if _, ok := byID[g.BossID]; !ok {
			p = append(p, fmt.Errorf("%s: grand boss %d is not in the NPC data", grandBossSQL, g.BossID))
		}
	}
	return errors.Join(p...)
}

// sqlErrors collects the errors of reading SQL rows.
type sqlErrors []error

func (e *sqlErrors) add(err error) {
	if err != nil {
		*e = append(*e, err)
	}
}

func (e *sqlErrors) int(r sqlRow, col string) int {
	v, err := r.int(col)
	e.add(err)
	return v
}

func (e *sqlErrors) long(r sqlRow, col string) int64 {
	v, err := r.long(col)
	e.add(err)
	return v
}

func (e *sqlErrors) float(r sqlRow, col string) float64 {
	v, err := r.float(col)
	e.add(err)
	return v
}

func (e *sqlErrors) err() error { return errors.Join(*e...) }
