package data

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
)

// SevenSignsFile is the Seven Signs data file (Plan 14, M11l, decision X6),
// relative to the data folder: the Dimensional Rift's rooms
// (sql/dimensional_rift.sql) with their spawns (DimensionalRift.xml), the
// Festival of Darkness spawn tables (the FESTIVAL_*_SPAWNS initializers of
// SevenSignsFestival.java) and the random spawns of the Seven Signs NPCs
// (sql/random_spawn.sql and random_spawn_loc.sql).
const SevenSignsFile = "sevensigns/sevensigns.json"

// Rift room types (Mobius DimensionalRiftManager): 0 is the waiting room
// (room 0, the peace zone) and the rift as a whole (room 1); 1-6 are the
// recruit, soldier, officer, captain, commander and hero areas.
const (
	RiftTypeWaiting = 0
	RiftTypeHero    = 6
)

// FestivalCount is the number of festivals per cabal
// (SevenSignsFestival.FESTIVAL_COUNT).
const FestivalCount = 5

// SevenSigns is the Seven Signs data. It must not be modified after
// NewSevenSigns or LoadSevenSigns returns, and is then safe for concurrent
// use.
type SevenSigns struct {
	Rift       []RiftRoom  `json:"rift"`
	Festival   Festival    `json:"festival"`
	AutoSpawns []AutoSpawn `json:"autoSpawns"`
}

// RiftRoom is a dimensional_rift row with the room's spawns: the box the
// room covers, where a party teleports in, and whether it is the boss room.
type RiftRoom struct {
	Type  int  `json:"type"`
	Room  int  `json:"room"`
	XMin  int  `json:"xMin"`
	XMax  int  `json:"xMax"`
	YMin  int  `json:"yMin"`
	YMax  int  `json:"yMax"`
	ZMin  int  `json:"zMin"`
	ZMax  int  `json:"zMax"`
	TeleX int  `json:"teleX"`
	TeleY int  `json:"teleY"`
	TeleZ int  `json:"teleZ"`
	Boss  bool `json:"boss"`
	// Spawns are the room's <spawn> lines, in file order.
	Spawns []RiftSpawn `json:"spawns,omitempty"`
}

// Contains reports whether the point is inside the room's box
// (DimensionalRiftRoom.checkIfInZone: java.awt.Polygon's insideness on x
// and y, which takes the minimum edges and leaves out the maximum ones, and
// z inclusive).
func (r *RiftRoom) Contains(x, y, z int) bool {
	return x >= r.XMin && x < r.XMax && y >= r.YMin && y < r.YMax && z >= r.ZMin && z <= r.ZMax
}

// RiftSpawnMargin is how far inside the box the room's monsters spawn
// (DimensionalRiftRoom: getRandomX and getRandomY pick from xMin+128 to
// xMax-128 and yMin+128 to yMax-128).
const RiftSpawnMargin = 128

// RiftSpawn is a <spawn> of a rift room: Count monsters NpcID that respawn
// after Delay seconds.
type RiftSpawn struct {
	NpcID int `json:"npcId"`
	Count int `json:"count"`
	Delay int `json:"delay"`
}

// FestivalSpawn is a row of a festival spawn table: a place, a heading (-1
// means a random one, as in Mobius) and an NPC (0 for the player spawns).
type FestivalSpawn struct {
	X       int `json:"x"`
	Y       int `json:"y"`
	Z       int `json:"z"`
	Heading int `json:"heading"`
	NpcID   int `json:"npcId,omitempty"`
}

// Festival holds the festival spawn tables, each indexed by festival 0-4
// (the level ranges up to 31, 42, 53, 64 and none).
type Festival struct {
	DawnPlayer    []FestivalSpawn   `json:"dawnPlayer"`
	DuskPlayer    []FestivalSpawn   `json:"duskPlayer"`
	DawnWitch     []FestivalSpawn   `json:"dawnWitch"`
	DuskWitch     []FestivalSpawn   `json:"duskWitch"`
	DawnPrimary   [][]FestivalSpawn `json:"dawnPrimary"`
	DuskPrimary   [][]FestivalSpawn `json:"duskPrimary"`
	DawnSecondary [][]FestivalSpawn `json:"dawnSecondary"`
	DuskSecondary [][]FestivalSpawn `json:"duskSecondary"`
	DawnChest     [][]FestivalSpawn `json:"dawnChest"`
	DuskChest     [][]FestivalSpawn `json:"duskChest"`
}

// AutoSpawn is a random_spawn group of a Seven Signs NPC
// (AutoSpawnHandler): Count NPCs at one of the Locations (a random one with
// RandomSpawn). The delays are Mobius's values as stored; -1 means the
// handler's default (30,000 ms initial, 3,600,000 ms respawn and despawn,
// AutoSpawnHandler.registerSpawn). The initial and despawn delays are
// milliseconds; with a despawn delay above 0 the respawn delay is the
// milliseconds between moves (scheduleAtFixedRate), with 0 it is the
// spawned NPC's own respawn delay in seconds (Spawn.setRespawnDelay).
type AutoSpawn struct {
	GroupID        int            `json:"groupId"`
	NpcID          int            `json:"npcId"`
	Count          int            `json:"count"`
	InitialDelay   int            `json:"initialDelay"`
	RespawnDelay   int            `json:"respawnDelay"`
	DespawnDelay   int            `json:"despawnDelay"`
	BroadcastSpawn bool           `json:"broadcastSpawn"`
	RandomSpawn    bool           `json:"randomSpawn"`
	Locations      []AutoSpawnLoc `json:"locations"`
}

// AutoSpawnLoc is a random_spawn_loc row; Heading -1 is a random heading.
type AutoSpawnLoc struct {
	X       int `json:"x"`
	Y       int `json:"y"`
	Z       int `json:"z"`
	Heading int `json:"heading"`
}

// SevenSignsNpcIDs are the NPCs SevenSigns.spawnSevenSignsNPC drives: the
// Merchant and Blacksmith of Mammon, Lilith and Anakim, the Crests of Dawn
// and Dusk, the Orators of Revelations, the Preachers of Doom and the Black
// Marketeer of Mammon.
var SevenSignsNpcIDs = []int{31113, 31126, 25283, 25286, 31170, 31171, 31094, 31093, 31092}

// LoadSevenSigns reads and checks the Seven Signs data file in dir.
func LoadSevenSigns(dir string) (*SevenSigns, error) {
	d, err := LoadSevenSignsFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return d, nil
}

// LoadSevenSignsFS reads and checks the Seven Signs data file in fsys.
func LoadSevenSignsFS(fsys fs.FS) (*SevenSigns, error) {
	var d SevenSigns
	if err := readJSON(fsys, SevenSignsFile, &d); err != nil {
		return nil, err
	}
	if err := d.Check(); err != nil {
		return nil, err
	}
	return &d, nil
}

// Check checks the data: rift rooms of types 0-6 each with rooms, sorted
// and unique, with a box and known room numbers; five festivals in every
// spawn table with NPCs where they need them; auto spawns of the Seven
// Signs NPCs with locations.
func (d *SevenSigns) Check() error {
	var p problems
	seen := map[[2]int]bool{}
	types := map[int]bool{}
	for i, r := range d.Rift {
		where := fmt.Sprintf("%s: rift room %d/%d", SevenSignsFile, r.Type, r.Room)
		if i > 0 {
			prev := d.Rift[i-1]
			if prev.Type > r.Type || (prev.Type == r.Type && prev.Room >= r.Room) {
				p.addf("%s: rooms must be sorted by type and room, without repeats", where)
			}
		}
		if seen[[2]int{r.Type, r.Room}] {
			continue
		}
		seen[[2]int{r.Type, r.Room}] = true
		types[r.Type] = true
		if r.Type < RiftTypeWaiting || r.Type > RiftTypeHero || r.XMin >= r.XMax || r.YMin >= r.YMax || r.ZMin > r.ZMax {
			p.addf("%s: bad type or box", where)
		}
		for _, s := range r.Spawns {
			if s.NpcID <= 0 || s.Count <= 0 || s.Delay < 0 {
				p.addf("%s: bad spawn %+v", where, s)
			}
		}
	}
	for t := RiftTypeWaiting; t <= RiftTypeHero; t++ {
		if !types[t] {
			p.addf("%s: no rift rooms of type %d", SevenSignsFile, t)
		}
	}
	f := d.Festival
	for _, l := range []struct {
		name  string
		list  []FestivalSpawn
		npcID bool
	}{{"dawnPlayer", f.DawnPlayer, false}, {"duskPlayer", f.DuskPlayer, false}, {"dawnWitch", f.DawnWitch, true}, {"duskWitch", f.DuskWitch, true}} {
		if len(l.list) != FestivalCount {
			p.addf("%s: festival %s has %d spawns, want %d", SevenSignsFile, l.name, len(l.list), FestivalCount)
		}
		for _, s := range l.list {
			if l.npcID != (s.NpcID > 0) {
				p.addf("%s: festival %s: bad NPC in %+v", SevenSignsFile, l.name, s)
			}
		}
	}
	for _, l := range []struct {
		name  string
		lists [][]FestivalSpawn
	}{{"dawnPrimary", f.DawnPrimary}, {"duskPrimary", f.DuskPrimary}, {"dawnSecondary", f.DawnSecondary}, {"duskSecondary", f.DuskSecondary},
		{"dawnChest", f.DawnChest}, {"duskChest", f.DuskChest}} {
		if len(l.lists) != FestivalCount {
			p.addf("%s: festival %s has %d festivals, want %d", SevenSignsFile, l.name, len(l.lists), FestivalCount)
		}
		for i, list := range l.lists {
			if len(list) == 0 {
				p.addf("%s: festival %s %d has no spawns", SevenSignsFile, l.name, i)
			}
			for _, s := range list {
				if s.NpcID <= 0 {
					p.addf("%s: festival %s %d: a spawn without an NPC", SevenSignsFile, l.name, i)
				}
			}
		}
	}
	groups := map[int]bool{}
	for _, a := range d.AutoSpawns {
		where := fmt.Sprintf("%s: auto spawn group %d", SevenSignsFile, a.GroupID)
		if groups[a.GroupID] {
			p.addf("%s: listed twice", where)
		}
		groups[a.GroupID] = true
		if !slices.Contains(SevenSignsNpcIDs, a.NpcID) || a.Count <= 0 || len(a.Locations) == 0 {
			p.addf("%s: NPC %d is not a Seven Signs NPC, or no count or locations", where, a.NpcID)
		}
	}
	return p.err()
}

// RiftRoom returns the rift room of a type.
func (d *SevenSigns) RiftRoom(typ, room int) (*RiftRoom, bool) {
	for i := range d.Rift {
		if r := &d.Rift[i]; r.Type == typ && r.Room == room {
			return r, true
		}
	}
	return nil, false
}

// RiftRooms returns the rooms of a rift type, sorted by room.
func (d *SevenSigns) RiftRooms(typ int) []*RiftRoom {
	var out []*RiftRoom
	for i := range d.Rift {
		if d.Rift[i].Type == typ {
			out = append(out, &d.Rift[i])
		}
	}
	slices.SortFunc(out, func(a, b *RiftRoom) int { return a.Room - b.Room })
	return out
}

// AutoSpawnsOf returns the auto spawn groups of an NPC, in group order.
func (d *SevenSigns) AutoSpawnsOf(npcID int) []*AutoSpawn {
	var out []*AutoSpawn
	for i := range d.AutoSpawns {
		if d.AutoSpawns[i].NpcID == npcID {
			out = append(out, &d.AutoSpawns[i])
		}
	}
	return out
}
