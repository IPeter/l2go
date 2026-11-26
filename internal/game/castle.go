package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// Castle is a castle on the loop (Mobius Castle): its data, its stored state
// and its siege. The owner is the clan whose CastleID names it. Loop-only.
type Castle struct {
	srv  *Server
	Data *data.Castle
	ID   int32
	Name string

	ownerID        int32
	taxPercent     int
	treasury       int64
	siegeDate      time.Time
	regTimeOver    bool // the lord can no longer choose the time (Mobius _isTimeRegistrationOver)
	regTimeEnd     time.Time
	showNpcCrest   bool
	ticketBuyCount int
	functions      map[int]residence.Function
	doorUpgrades   map[int32]int32
	trapUpgrades   map[int32]int32
	hired          []storage.HiredGuard

	// formerOwner is Mobius _formerOwner: the first owner that lost the
	// castle since the server started; never cleared (Mobius quirk).
	formerOwner     int32
	firstMidVictory bool

	siegeZone, castleZone, teleZone *data.Zone
	siege                           castleSiege
}

var _ Siegable = (*Castle)(nil)

// newCastle makes a castle of the data without an owner, with Mobius's
// defaults (castle.sql) and its zones.
func (s *Server) newCastle(d *data.Castle) *Castle {
	c := &Castle{srv: s, Data: d, ID: int32(d.ID), Name: d.Name, regTimeOver: true, //nolint:gosec // castle IDs are small
		functions: make(map[int]residence.Function), doorUpgrades: make(map[int32]int32), trapUpgrades: make(map[int32]int32)}
	if w := s.worldData; w != nil {
		for i := range w.Zones {
			z := &w.Zones[i]
			if zoneCastleID(z) != c.ID {
				continue
			}
			switch z.Type {
			case "SiegeZone":
				c.siegeZone = z
			case "CastleZone":
				c.castleZone = z
			case "ResidenceTeleportZone":
				c.teleZone = z
			}
		}
	}
	return c
}

// restore takes the stored state.
func (c *Castle) restore(st storage.Castle) {
	c.taxPercent = int(st.TaxPercent)
	c.treasury = st.Treasury
	c.siegeDate = msTime(st.SiegeDate)
	c.regTimeOver = st.RegTimeOver
	c.regTimeEnd = msTime(st.RegTimeEnd)
	c.showNpcCrest = st.ShowNpcCrest
	c.ticketBuyCount = int(st.TicketBuyCount)
	for _, f := range st.Functions {
		c.functions[int(f.Type)] = residence.Function{Type: int(f.Type), Level: int(f.Level), Lease: int(f.Lease), Rate: f.Rate, EndTime: f.EndTime}
	}
	for _, u := range st.DoorUpgrades {
		c.doorUpgrades[u.DoorID] = u.Ratio
	}
	for _, u := range st.TrapUpgrades {
		c.trapUpgrades[u.TowerIndex] = u.Level
	}
	c.hired = slices.Clone(st.HiredGuards)
	rows := make([]residence.Registration, 0, len(st.SiegeClans))
	for _, sc := range st.SiegeClans {
		rows = append(rows, residence.Registration{ClanID: sc.ClanID, Type: int(sc.Type)})
	}
	c.siege.stored = rows
}

// document returns the castle's stored state.
func (c *Castle) document() storage.Castle {
	st := storage.Castle{ID: c.ID, TaxPercent: int32(c.taxPercent), Treasury: c.treasury, //nolint:gosec // tax 0-100
		SiegeDate: timeMs(c.siegeDate), RegTimeOver: c.regTimeOver, RegTimeEnd: timeMs(c.regTimeEnd),
		ShowNpcCrest: c.showNpcCrest, TicketBuyCount: int32(c.ticketBuyCount)} //nolint:gosec // small
	for _, t := range sortedKeys(c.functions) {
		f := c.functions[t]
		st.Functions = append(st.Functions, storage.CastleFunction{Type: int32(f.Type), Level: int32(f.Level), //nolint:gosec // small
			Lease: int32(f.Lease), Rate: f.Rate, EndTime: f.EndTime}) //nolint:gosec // small
	}
	for _, id := range sortedKeys(c.doorUpgrades) {
		st.DoorUpgrades = append(st.DoorUpgrades, storage.CastleDoorUpgrade{DoorID: id, Ratio: c.doorUpgrades[id]})
	}
	for _, i := range sortedKeys(c.trapUpgrades) {
		st.TrapUpgrades = append(st.TrapUpgrades, storage.CastleTrapUpgrade{TowerIndex: i, Level: c.trapUpgrades[i]})
	}
	for _, r := range c.siege.rows() {
		st.SiegeClans = append(st.SiegeClans, storage.SiegeClan{ClanID: r.ClanID, Type: int32(r.Type)}) //nolint:gosec // 0-2
	}
	st.HiredGuards = slices.Clone(c.hired)
	return st
}

func sortedKeys[K int | int32, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func msTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

func timeMs(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// save writes the castle, with the clans given, in one transaction (X4).
func (c *Castle) save(clans ...*Clan) {
	s := c.srv
	b := storage.Batch{Castles: []storage.Castle{c.document()}}
	for _, cl := range clans {
		if cl != nil {
			b.Clans = append(b.Clans, cl.document())
		}
	}
	s.SaveJoint(nil, b)
}

// ResidenceID implements Siegable.
func (c *Castle) ResidenceID() int32 { return c.ID }

// InProgress implements Siegable.
func (c *Castle) InProgress() bool { return c.siege.inProgress }

// Clans implements Siegable.
func (c *Castle) Clans() *residence.Registry { return &c.siege.reg }

// Contains reports whether pos is in the castle's siege zone (Mobius
// Castle.checkIfInZone).
func (c *Castle) Contains(pos world.Position) bool {
	return c.siegeZone != nil && c.siegeZone.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z))
}

// OwnerID returns the owner clan's ID, 0 for none.
func (c *Castle) OwnerID() int32 { return c.ownerID }

// Owner returns the owner clan, or nil.
func (c *Castle) Owner() *Clan { return c.srv.Clan(c.ownerID) }

// TaxPercent returns the tax the lord set.
func (c *Castle) TaxPercent() int { return c.taxPercent }

// TaxRate returns the tax as a fraction (Castle.getTaxRate).
func (c *Castle) TaxRate() float64 { return residence.TaxRate(c.taxPercent) }

// Treasury returns the castle's treasury.
func (c *Castle) Treasury() int64 { return c.treasury }

// SiegeDate returns the next siege's start.
func (c *Castle) SiegeDate() time.Time { return c.siegeDate }

// SetTaxPercent is Castle.setTaxPercent, saved.
func (c *Castle) SetTaxPercent(p int) {
	c.taxPercent = p
	c.save()
}

// treasuryState is the castle as residence.TreasuryShares sees it.
func (c *Castle) treasuryState() residence.TreasuryState {
	return residence.TreasuryState{ID: int(c.ID), Name: c.Name, OwnerID: c.ownerID, TaxRate: c.TaxRate(), Treasury: c.treasury}
}

// AddToTreasury is Castle.addToTreasury: tax income, of which Rune and Aden
// take their cut. Every castle that receives a share is saved.
func (c *Castle) AddToTreasury(amount int64) {
	s := c.srv
	byName := func(n string) (residence.TreasuryState, bool) {
		if o := s.CastleByName(n); o != nil {
			return o.treasuryState(), true
		}
		return residence.TreasuryState{}, false
	}
	shares := residence.TreasuryShares(c.treasuryState(), amount, byName)
	for _, id := range sortedKeys(shares) {
		if o := s.CastleByID(int32(id)); o != nil { //nolint:gosec // castle IDs
			o.AddToTreasuryNoTax(shares[id])
		}
	}
}

// AddToTreasuryNoTax is Castle.addToTreasuryNoTax: false when the castle has
// no owner or too little for a negative amount. Saved.
func (c *Castle) AddToTreasuryNoTax(amount int64) bool {
	v, ok := residence.AddNoTax(c.ownerID > 0, c.treasury, amount)
	if !ok {
		return false
	}
	c.treasury = v
	c.save()
	return true
}

// ----- zones -----

// spawnLoc is a random point of the castle zone's spawns of a kind
// (ZoneRespawn.getSpawnLoc and its chaotic, other and banish twins).
func (c *Castle) spawnLoc(kind string) (data.Location, bool) {
	if c.castleZone == nil {
		return data.Location{}, false
	}
	var pts []data.Location
	for _, sp := range c.castleZone.Spawns {
		if sp.Type == kind {
			pts = append(pts, data.Location{X: sp.X, Y: sp.Y, Z: sp.Z})
		}
	}
	if len(pts) == 0 {
		return data.Location{}, false
	}
	if c.srv.combatSettings().RandomRespawnInTown {
		return pts[c.srv.rnd.Get(len(pts))], true
	}
	return pts[0], true
}

// playersInZone returns the players in the game inside the siege zone.
func (c *Castle) playersInZone() []*Player {
	var out []*Player
	for p := range c.srv.Players() {
		if p.state == stateInGame && p.InWorld() && c.Contains(p.Position()) {
			out = append(out, p)
		}
	}
	return out
}

// doors returns the castle's doors: the doors in its siege zone (Mobius
// Door.getCastle).
func (c *Castle) doors() []*Door {
	var out []*Door
	for _, d := range c.srv.Doors() {
		if c.Contains(d.Position()) {
			out = append(out, d)
		}
	}
	return out
}

// spawnDoors is Castle.spawnDoor: a broken door is revived at full HP (half
// when weak, (int) max HP / 2), and every door closes. A door already closed
// is shown again too, so that its viewers see whether it is an enemy now
// (DoorStatusUpdate; Mobius sends nothing for it).
func (c *Castle) spawnDoors(weak bool) {
	s := c.srv
	for _, d := range c.doors() {
		if d.IsDead() {
			s.reviveDoor(d) // door_combat.go
			hp := float64(d.maxHP)
			if weak {
				hp = float64(d.maxHP / 2)
			}
			s.setDoorHP(d, hp, true)
		}
		if d.open {
			s.closeDoor(d)
		} else {
			s.broadcastDoor(d)
		}
	}
}

// ----- owner -----

// setOwner is Castle.setOwner (clan nil is removeOwner): the old owner loses
// the castle (and becomes the former owner the first time), the new one gets
// it, both are saved with the castle in one transaction, and a siege in
// progress goes to its mid victory.
func (s *Server) setCastleOwner(c *Castle, clan *Clan) {
	var changed []*Clan
	if old := c.Owner(); old != nil && (clan == nil || clan.ID != old.ID) {
		if c.formerOwner == 0 {
			c.formerOwner = old.ID
		}
		old.CastleID = 0
		if q, ok := s.onlinePlayer(old.LeaderID); ok && q.mount.typ == serverpackets.MountWyvern {
			s.dismount(q) // mount.go: the old lord gets off his wyvern
		}
		changed = append(changed, old)
		s.broadcastClanInfo(old)
	}
	if clan == nil {
		c.ownerID = 0
		c.functions = make(map[int]residence.Function)
		s.manorCastleRemoved(c) // manor.go: Mobius resetManorData
	} else {
		c.ownerID = clan.ID
		clan.CastleID = c.ID
		changed = append(changed, clan)
		s.broadcastClanInfo(clan)
		broadcastToClan(clan, serverpackets.NewPlaySoundPacket(serverpackets.PlaySound{Type: 1, Sound: "Siege_Victory"}))
	}
	c.showNpcCrest = false
	c.save(changed...)
	if c.siege.inProgress {
		s.midVictory(c)
	}
}
