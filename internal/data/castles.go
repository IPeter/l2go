package data

import (
	"fmt"
	"io/fs"
	"os"
	"time"
)

// Paths of the castle data files (Plan 14, M11i, decision X6), relative to
// the data folder: the castles of sql/castle.sql with their schedule
// (cfg/SiegeSchedule.xml), towers and mercenary limit (cfg/Siege.ini); the
// siege guards of sql/castle_siege_guards.sql; the mercenary tickets of the
// MercTicketManager tables.
const (
	CastlesFile     = "castles/castles.json"
	SiegeGuardsFile = "castles/guards.json"
	MercTicketsFile = "castles/merctickets.json"
)

// Castle is one castle: a sql/castle.sql row with its schedule, towers and
// mercenary limit.
type Castle struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// SiegeEnabled, SiegeDay (0 Sunday … 6 Saturday), SiegeHour and
	// MaxConcurrent are the castle's cfg/SiegeSchedule.xml line.
	SiegeEnabled  bool `json:"siegeEnabled"`
	SiegeDay      int  `json:"siegeDay"`
	SiegeHour     int  `json:"siegeHour"`
	MaxConcurrent int  `json:"maxConcurrent"`
	// ControlTowers and FlameTowers are the <Name>ControlTowerN and
	// <Name>FlameTowerN lines of cfg/Siege.ini, in N order.
	ControlTowers []TowerSpawn `json:"controlTowers"`
	FlameTowers   []TowerSpawn `json:"flameTowers"`
	// MaxMercenaries is <Name>MaxMercenaries (MercTicketManager's table for
	// a castle without the line).
	MaxMercenaries int `json:"maxMercenaries"`
}

// SiegeWeekday returns the schedule's weekday.
func (c *Castle) SiegeWeekday() time.Weekday { return time.Weekday(c.SiegeDay) }

// TowerSpawn is a control or flame tower of a castle (Mobius TowerSpawn).
// Zones are the trap zones a flame tower switches (flame towers only).
type TowerSpawn struct {
	NpcID int   `json:"npcId"`
	X     int   `json:"x"`
	Y     int   `json:"y"`
	Z     int   `json:"z"`
	Zones []int `json:"zones,omitempty"`
}

// SiegeGuard is a castle_siege_guards row of the data (isHired 0): a guard
// that defends a castle without an owner.
type SiegeGuard struct {
	CastleID     int `json:"castleId"`
	ID           int `json:"id"`
	NpcID        int `json:"npcId"`
	X            int `json:"x"`
	Y            int `json:"y"`
	Z            int `json:"z"`
	Heading      int `json:"heading"`
	RespawnDelay int `json:"respawnDelay"` // seconds; 0 never respawns
}

// MercTicket is a mercenary posting ticket (MercTicketManager ITEM_IDS,
// NPC_IDS and MAX_MERC_PER_TYPE at the same index): the item places the
// mercenary NpcID in its castle. Index is the position in the castle's 52
// ticket types.
type MercTicket struct {
	ItemID     int `json:"itemId"`
	NpcID      int `json:"npcId"`
	CastleID   int `json:"castleId"`
	Index      int `json:"index"`
	MaxPerType int `json:"maxPerType"`
}

// CastleData is the content of the castle data files.
type CastleData struct {
	Castles []Castle     // CastlesFile, by ID
	Guards  []SiegeGuard // SiegeGuardsFile, by castle and ID
	Tickets []MercTicket // MercTicketsFile, in Mobius's table order
}

// Castles is the checked castle data. It must not be modified after
// NewCastles or LoadCastles returns, and is then safe for concurrent use.
type Castles struct {
	CastleData
	byID    map[int]int
	tickets map[int]int // item ID → index in Tickets
}

// LoadCastles reads and checks the castle data files in dir.
func LoadCastles(dir string) (*Castles, error) {
	c, err := LoadCastlesFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return c, nil
}

// LoadCastlesFS reads and checks the castle data files in fsys.
func LoadCastlesFS(fsys fs.FS) (*Castles, error) {
	var d CastleData
	if err := readJSON(fsys, CastlesFile, &d.Castles); err != nil {
		return nil, err
	}
	if err := readJSON(fsys, SiegeGuardsFile, &d.Guards); err != nil {
		return nil, err
	}
	if err := readJSON(fsys, MercTicketsFile, &d.Tickets); err != nil {
		return nil, err
	}
	return NewCastles(d)
}

// NewCastles checks the castle data: castle IDs rising from 1 with names,
// a weekday and hour, towers with NPCs; guards of known castles with NPCs;
// tickets of known castles with unique item IDs.
func NewCastles(d CastleData) (*Castles, error) {
	var p problems
	c := &Castles{CastleData: d, byID: make(map[int]int), tickets: make(map[int]int)}
	for i, x := range d.Castles {
		where := fmt.Sprintf("%s: castle %d", CastlesFile, x.ID)
		if x.ID != i+1 || x.Name == "" {
			p.addf("%s: IDs must rise from 1 and castles have names", where)
		}
		if x.SiegeDay < 0 || x.SiegeDay > 6 || x.SiegeHour < 0 || x.SiegeHour > 23 || x.MaxConcurrent < 0 {
			p.addf("%s: bad schedule", where)
		}
		for _, t := range append(append([]TowerSpawn(nil), x.ControlTowers...), x.FlameTowers...) {
			if t.NpcID <= 0 {
				p.addf("%s: a tower without an NPC", where)
			}
		}
		c.byID[x.ID] = i
	}
	for _, g := range d.Guards {
		if _, ok := c.byID[g.CastleID]; !ok || g.NpcID <= 0 {
			p.addf("%s: guard %d: unknown castle %d or no NPC", SiegeGuardsFile, g.ID, g.CastleID)
		}
	}
	for i, t := range d.Tickets {
		if _, ok := c.byID[t.CastleID]; !ok || t.NpcID <= 0 || t.ItemID <= 0 {
			p.addf("%s: ticket %d: unknown castle %d or no NPC", MercTicketsFile, t.ItemID, t.CastleID)
		}
		if _, dup := c.tickets[t.ItemID]; dup {
			p.addf("%s: ticket %d is listed twice", MercTicketsFile, t.ItemID)
		}
		c.tickets[t.ItemID] = i
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return c, nil
}

// Castle returns the castle with the ID.
func (c *Castles) Castle(id int) (*Castle, bool) {
	i, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	return &c.Castles[i], true
}

// Ticket returns the mercenary ticket of an item.
func (c *Castles) Ticket(itemID int) (*MercTicket, bool) {
	i, ok := c.tickets[itemID]
	if !ok {
		return nil, false
	}
	return &c.Tickets[i], true
}

// GuardsOf returns the data's siege guards of a castle.
func (c *Castles) GuardsOf(castleID int) []SiegeGuard {
	var out []SiegeGuard
	for _, g := range c.Guards {
		if g.CastleID == castleID {
			out = append(out, g)
		}
	}
	return out
}
