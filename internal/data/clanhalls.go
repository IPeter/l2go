package data

import (
	"fmt"
	"io/fs"
	"os"
)

// Paths of the clan hall data files (Plan 14, M11j, decision X6), relative to
// the data folder: the halls of sql/clanhall.sql and sql/siegable_clanhall.sql,
// the siege guards of sql/clanhall_siege_guards.sql, and the NPC auctions of
// ClanHallAuctionManager.ITEM_INIT_DATA.
const (
	ClanHallsFile        = "clanhalls/clanhalls.json"
	ClanHallGuardsFile   = "clanhalls/guards.json"
	ClanHallAuctionsFile = "clanhalls/auctions.json"
)

// ClanHall is one clan hall: a sql/clanhall.sql row (an auctionable hall) or
// a sql/siegable_clanhall.sql row (a contestable hall).
type ClanHall struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Location string `json:"location"`
	// Lease is the weekly lease of an auctionable hall; Grade its grade
	// (1-3).
	Lease int `json:"lease,omitempty"`
	Grade int `json:"grade,omitempty"`
	// Siegable marks a contestable hall: SiegeLength is its siege's length
	// in milliseconds, Schedule its schedule_config "DD;MM;YY;HH;mm" (days,
	// months and years to add, then the hour and minute).
	Siegable    bool   `json:"siegable,omitempty"`
	SiegeLength int64  `json:"siegeLength,omitempty"`
	Schedule    [5]int `json:"schedule,omitzero"`
}

// ClanHallGuard is a clanhall_siege_guards row: a guard that stands during a
// contestable hall's siege.
type ClanHallGuard struct {
	HallID       int  `json:"hallId"`
	NpcID        int  `json:"npcId"`
	X            int  `json:"x"`
	Y            int  `json:"y"`
	Z            int  `json:"z"`
	Heading      int  `json:"heading"`
	RespawnDelay int  `json:"respawnDelay"` // seconds; 0 never respawns
	SiegeBoss    bool `json:"siegeBoss,omitempty"`
}

// ClanHallAuction is an NPC auction a free hall starts with (a row of
// ClanHallAuctionManager.ITEM_INIT_DATA): the hall, its name, the starting
// bid and the end date in Unix milliseconds (always past: the first auction
// ends a week after the server starts).
type ClanHallAuction struct {
	HallID      int    `json:"hallId"`
	ItemName    string `json:"itemName"`
	StartingBid int64  `json:"startingBid"`
	EndDate     int64  `json:"endDate"`
}

// ClanHallData is the content of the clan hall data files.
type ClanHallData struct {
	Halls    []ClanHall        // ClanHallsFile, by ID
	Guards   []ClanHallGuard   // ClanHallGuardsFile, in the SQL's order
	Auctions []ClanHallAuction // ClanHallAuctionsFile, by hall
}

// ClanHalls is the checked clan hall data. It must not be modified after
// NewClanHalls or LoadClanHalls returns, and is then safe for concurrent use.
type ClanHalls struct {
	ClanHallData
	byID     map[int]int
	auctions map[int]int
}

// LoadClanHalls reads and checks the clan hall data files in dir.
func LoadClanHalls(dir string) (*ClanHalls, error) {
	c, err := LoadClanHallsFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return c, nil
}

// LoadClanHallsFS reads and checks the clan hall data files in fsys.
func LoadClanHallsFS(fsys fs.FS) (*ClanHalls, error) {
	var d ClanHallData
	if err := readJSON(fsys, ClanHallsFile, &d.Halls); err != nil {
		return nil, err
	}
	if err := readJSON(fsys, ClanHallGuardsFile, &d.Guards); err != nil {
		return nil, err
	}
	if err := readJSON(fsys, ClanHallAuctionsFile, &d.Auctions); err != nil {
		return nil, err
	}
	return NewClanHalls(d)
}

// NewClanHalls checks the clan hall data: unique rising IDs with names; an
// auctionable hall has a lease, a contestable one a siege length; guards and
// auctions of known halls.
func NewClanHalls(d ClanHallData) (*ClanHalls, error) {
	var p problems
	c := &ClanHalls{ClanHallData: d, byID: make(map[int]int), auctions: make(map[int]int)}
	last := 0
	for i, h := range d.Halls {
		where := fmt.Sprintf("%s: hall %d", ClanHallsFile, h.ID)
		if h.ID <= last || h.Name == "" {
			p.addf("%s: IDs must rise and halls have names", where)
		}
		last = h.ID
		if h.Siegable && h.SiegeLength <= 0 {
			p.addf("%s: a contestable hall without a siege length", where)
		}
		if !h.Siegable && h.Lease < 0 {
			p.addf("%s: negative lease", where)
		}
		c.byID[h.ID] = i
	}
	for _, g := range d.Guards {
		if _, ok := c.byID[g.HallID]; !ok || g.NpcID <= 0 {
			p.addf("%s: guard of unknown hall %d or without an NPC", ClanHallGuardsFile, g.HallID)
		}
	}
	for i, a := range d.Auctions {
		if _, ok := c.byID[a.HallID]; !ok {
			p.addf("%s: auction of unknown hall %d", ClanHallAuctionsFile, a.HallID)
		}
		if _, dup := c.auctions[a.HallID]; dup {
			p.addf("%s: hall %d has two auctions", ClanHallAuctionsFile, a.HallID)
		}
		c.auctions[a.HallID] = i
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return c, nil
}

// Hall returns the hall with the ID.
func (c *ClanHalls) Hall(id int) (*ClanHall, bool) {
	i, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	return &c.Halls[i], true
}

// Auction returns the NPC auction of a hall (ClanHallAuctionManager.initNPC).
func (c *ClanHalls) Auction(hallID int) (*ClanHallAuction, bool) {
	i, ok := c.auctions[hallID]
	if !ok {
		return nil, false
	}
	return &c.Auctions[i], true
}

// GuardsOf returns the siege guards of a hall, in the data's order.
func (c *ClanHalls) GuardsOf(hallID int) []ClanHallGuard {
	var out []ClanHallGuard
	for _, g := range c.Guards {
		if g.HallID == hallID {
			out = append(out, g)
		}
	}
	return out
}
