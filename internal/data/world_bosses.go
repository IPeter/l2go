package data

// Boss data: the raid boss spawns, the grand boss seeds and the boss zone
// parameters that cmd/datagen converts from Mobius raidboss_spawnlist.sql,
// grandboss_data.sql and zones/custom_boss.xml. They are loaded with
// LoadBosses. The live boss state (respawn times, HP and MP, grand boss
// status) is kept by the game server in its database, seeded from here.

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"time"
)

// Paths of the boss data files, relative to the data folder.
const (
	RaidSpawnsFile  = "spawns/raidbosses.json"
	GrandBossesFile = "bosses/grandbosses.json"
	BossZonesFile   = "bosses/zones.json"
)

// RaidSpawn is one row of Mobius's raidboss_spawnlist table: where a raid
// boss spawns and how long it takes to come back.
type RaidSpawn struct {
	BossID  int `json:"bossId"`
	X       int `json:"x"`
	Y       int `json:"y"`
	Z       int `json:"z"`
	Heading int `json:"heading"`
	// RespawnDelay and RespawnRandom are in seconds: the boss returns
	// RespawnDelay ± RespawnRandom after its death (see RespawnWindow).
	RespawnDelay  int `json:"respawnDelay"`
	RespawnRandom int `json:"respawnRandom"`
	// HP and MP are the boss's current HP and MP the table starts with
	// (currentHp, currentMp): Mobius spawns the boss with them when it has
	// no respawn time stored.
	HP float64 `json:"hp"`
	MP float64 `json:"mp"`
}

// RespawnWindow returns the shortest and longest respawn time after a
// death, as Mobius Spawn.setRespawnDelay computes them: delay ∓ random,
// at least one second each; both zero when the delay is zero (no respawn).
// Mobius picks uniformly between them, after multiplying both with
// RaidMinRespawnMultiplier and RaidMaxRespawnMultiplier.
func (r *RaidSpawn) RespawnWindow() (lo, hi time.Duration) {
	if r.RespawnDelay == 0 {
		return 0, 0
	}
	lo = time.Duration(max(1, r.RespawnDelay-r.RespawnRandom)) * time.Second
	hi = time.Duration(max(1, r.RespawnDelay+r.RespawnRandom)) * time.Second
	return lo, hi
}

// GrandBoss is one row Mobius's grandboss_data table is seeded with: the
// boss's position, its HP and MP, its respawn time and status. The game
// server copies these into its database once (as INSERT IGNORE does); the
// boss scripts change them from then on. Status means what the boss's
// script says (0 is alive or dormant in all of them; dead is 1 for Queen
// Ant, Core, Orfen and Zaken, 2 for Dr. Chaos and Sailren, 3 for the
// others).
type GrandBoss struct {
	BossID  int `json:"bossId"`
	X       int `json:"x"`
	Y       int `json:"y"`
	Z       int `json:"z"`
	Heading int `json:"heading"`
	// RespawnTime is the epoch time in milliseconds the boss respawns at
	// (0 in every seed).
	RespawnTime int64   `json:"respawnTime"`
	HP          float64 `json:"hp"`
	MP          float64 `json:"mp"`
	Status      int     `json:"status"`
}

// BossZone holds the parameters of one BossZone of custom_boss.xml. Its
// shape is in the zone data (ZonesFile), under the same zone ID.
type BossZone struct {
	ZoneID int    `json:"zoneId"`
	Name   string `json:"name"`
	// InvadeTime is how long, in milliseconds, a player who logged out in
	// the zone may come back in (0: not at all).
	InvadeTime int `json:"invadeTime,omitempty"`
	// Oust is where players are sent out of the zone; Mobius uses it only
	// when all three coordinates are non-zero, the nearest town otherwise
	// (see OustPoint).
	Oust *Location `json:"oust,omitempty"`
	// Enabled is default_enabled (default true).
	Enabled bool `json:"enabled"`
}

// OustPoint returns the oust point, if the zone has one Mobius uses.
func (z *BossZone) OustPoint() (Location, bool) {
	if z.Oust == nil || z.Oust.X == 0 || z.Oust.Y == 0 || z.Oust.Z == 0 {
		return Location{}, false
	}
	return *z.Oust, true
}

// BossData is the content of the boss data files.
type BossData struct {
	RaidSpawns  []RaidSpawn // RaidSpawnsFile, sorted by boss ID
	GrandBosses []GrandBoss // GrandBossesFile, sorted by boss ID
	Zones       []BossZone  // BossZonesFile, sorted by zone ID
}

// Bosses is the checked boss data. It must not be modified after
// NewBosses or LoadBosses returns, and is then safe for concurrent use; the
// pointers its lookups return point into it.
type Bosses struct {
	BossData
}

// LoadBosses reads and checks the boss data files in dir.
func LoadBosses(dir string) (*Bosses, error) {
	b, err := LoadBossesFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return b, nil
}

// LoadBossesFS reads and checks the boss data files in fsys.
func LoadBossesFS(fsys fs.FS) (*Bosses, error) {
	var d BossData
	for _, f := range []struct {
		name string
		v    any
	}{
		{RaidSpawnsFile, &d.RaidSpawns},
		{GrandBossesFile, &d.GrandBosses},
		{BossZonesFile, &d.Zones},
	} {
		if err := readJSON(fsys, f.name, f.v); err != nil {
			return nil, err
		}
	}
	return NewBosses(d)
}

// NewBosses checks the boss data. The slices are used as they are. Raid
// spawns and grand bosses: boss IDs positive, unique and sorted; delays,
// HP and MP not negative. Boss zones: IDs unique and sorted, invade times
// not negative.
func NewBosses(d BossData) (*Bosses, error) {
	var p problems
	for i, r := range d.RaidSpawns {
		where := fmt.Sprintf("%s: raid boss %d", RaidSpawnsFile, r.BossID)
		if r.BossID <= 0 {
			p.addf("%s: ID must be positive", where)
		}
		if i > 0 && d.RaidSpawns[i-1].BossID >= r.BossID {
			p.addf("%s: boss IDs are not sorted and unique", where)
		}
		if r.RespawnDelay < 0 || r.RespawnRandom < 0 {
			p.addf("%s: negative respawn delay or random", where)
		}
		if r.HP < 0 || r.MP < 0 {
			p.addf("%s: negative HP or MP", where)
		}
	}
	for i, g := range d.GrandBosses {
		where := fmt.Sprintf("%s: grand boss %d", GrandBossesFile, g.BossID)
		if g.BossID <= 0 {
			p.addf("%s: ID must be positive", where)
		}
		if i > 0 && d.GrandBosses[i-1].BossID >= g.BossID {
			p.addf("%s: boss IDs are not sorted and unique", where)
		}
		if g.RespawnTime < 0 || g.HP < 0 || g.MP < 0 || g.Status < 0 {
			p.addf("%s: negative respawn time, HP, MP or status", where)
		}
	}
	for i, z := range d.Zones {
		where := fmt.Sprintf("%s: zone %d", BossZonesFile, z.ZoneID)
		if i > 0 && d.Zones[i-1].ZoneID >= z.ZoneID {
			p.addf("%s: zone IDs are not sorted and unique", where)
		}
		if z.InvadeTime < 0 {
			p.addf("%s: negative invade time", where)
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return &Bosses{BossData: d}, nil
}

// RaidSpawn returns the spawn of a raid boss.
func (b *Bosses) RaidSpawn(bossID int) (*RaidSpawn, bool) {
	i, ok := slices.BinarySearchFunc(b.RaidSpawns, bossID, func(r RaidSpawn, id int) int { return cmp.Compare(r.BossID, id) })
	if !ok {
		return nil, false
	}
	return &b.RaidSpawns[i], true
}

// GrandBoss returns the seed of a grand boss.
func (b *Bosses) GrandBoss(bossID int) (*GrandBoss, bool) {
	i, ok := slices.BinarySearchFunc(b.GrandBosses, bossID, func(g GrandBoss, id int) int { return cmp.Compare(g.BossID, id) })
	if !ok {
		return nil, false
	}
	return &b.GrandBosses[i], true
}

// Zone returns the parameters of a boss zone.
func (b *Bosses) Zone(zoneID int) (*BossZone, bool) {
	i, ok := slices.BinarySearchFunc(b.Zones, zoneID, func(z BossZone, id int) int { return cmp.Compare(z.ZoneID, id) })
	if !ok {
		return nil, false
	}
	return &b.Zones[i], true
}

// CheckWorld checks the references of the boss data into the world data:
// raid spawns name RaidBoss NPCs, grand boss seeds existing NPCs, boss zones
// BossZone zones.
func (b *Bosses) CheckWorld(w *World) error {
	var p problems
	for _, r := range b.RaidSpawns {
		n, ok := w.Npc(r.BossID)
		switch {
		case !ok:
			p.addf("%s: raid boss %d: no such NPC", RaidSpawnsFile, r.BossID)
		case !w.NpcTypeIs(n.Type, "RaidBoss"):
			p.addf("%s: raid boss %d: NPC type %s is not a RaidBoss", RaidSpawnsFile, r.BossID, n.Type)
		}
	}
	for _, g := range b.GrandBosses {
		if _, ok := w.Npc(g.BossID); !ok {
			p.addf("%s: grand boss %d: no such NPC", GrandBossesFile, g.BossID)
		}
	}
	for _, z := range b.Zones {
		zone, ok := w.Zone(z.ZoneID)
		switch {
		case !ok:
			p.addf("%s: zone %d: no such zone", BossZonesFile, z.ZoneID)
		case !w.ZoneTypeIs(zone.Type, "BossZone"):
			p.addf("%s: zone %d: type %s is not a BossZone", BossZonesFile, z.ZoneID, zone.Type)
		}
	}
	return p.err()
}
