package game

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/residence"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// Castles and sieges (Plan 14 M11i): the castles of the castle data with
// their stored state, owned by the clan whose CastleID names them, and the
// siege engine (siege.go). The pure rules are internal/residence; this file
// holds the castles on the loop, their zones and the lookups the other
// features ask (which castle stands at a point, which a clan owns).
//
// Clan halls (M11j) reuse the siege engine through Siegable: a contestable
// hall registers itself with registerSiegable, and the zone, restart, relation
// and Die code ask every Siegable.

func init() { registerFeature("residence", setupResidence) }

// residenceLoadTimeout bounds reading the castles at startup.
const residenceLoadTimeout = 30 * time.Second

// Siegable is a residence with a siege (Mobius Siegable): a castle, or a
// contestable clan hall (M11j). Loop-only.
type Siegable interface {
	// ResidenceID is the castle or hall ID (Player.getSiegeSide).
	ResidenceID() int32
	// InProgress reports whether the siege is on.
	InProgress() bool
	// Clans is the siege's registry.
	Clans() *residence.Registry
	// Contains reports whether pos is in the siege's zone.
	Contains(pos world.Position) bool
}

// residenceFeature is the castles on the Server (Server.residence).
// Loop-only, except data and store, which New sets.
type residenceFeature struct {
	data  *data.Castles
	store storage.CastleStore

	castles []*Castle // by ID - 1

	// others are the other Siegables (clan halls, M11j).
	others []Siegable

	// sides are the siege states of the players in a siege (Mobius
	// _siegeState and _siegeSide), zoneFlags the siege zones whose flags a
	// player carries (SiegeZone.onEnter while active).
	sides     map[*Player]playerSiege
	zoneFlags map[*Player]map[int]bool

	cfg      residenceConfig
	tickets  []*groundTicket // mercenary tickets on the ground (castle_merc.go)
	sieging  bool            // a siege is in progress somewhere
	loadedOK bool
}

// residenceConfig is the siege config with the reputation points.
type residenceConfig struct {
	cycle, length, maxFlags, minLevel int

	maxAttackers, maxDefenders int

	attackerRespawn time.Duration

	points residence.ReputationPoints
}

// playerSiege is a player's siege state: 1 attacker, 2 defender (Mobius
// setSiegeState) and the residence it is registered on.
type playerSiege struct {
	state int
	side  int32
}

// setupResidence makes the castles from the data and their stored state,
// and starts the siege tasks.
func setupResidence(s *Server) error {
	f := &s.residence
	sc := s.gs.Siege
	f.cfg = residenceConfig{cycle: sc.SiegeCycle, length: sc.SiegeLength, maxFlags: sc.MaxFlags, minLevel: sc.SiegeClanMinLevel,
		maxAttackers: sc.AttackerMaxClans, maxDefenders: sc.DefenderMaxClans}
	f.cfg.attackerRespawn = time.Duration(sc.AttackerRespawn) * time.Millisecond
	f.cfg.points = residence.ReputationPoints{Take: sc.TakeCastlePoints, Lose: sc.LooseCastlePoints, Defend: sc.CastleDefendedPoints}
	if f.cfg.cycle <= 0 {
		f.cfg = residenceDefaults()
	}
	f.sides = make(map[*Player]playerSiege)
	f.zoneFlags = make(map[*Player]map[int]bool)
	s.OnPlayerEnter(s.residencePlayerEntered)
	s.OnPlayerLeave(s.residencePlayerLeft)
	s.OnDie(s.residenceDie)
	if err := s.registerSiegePackets(); err != nil { // siege_packets.go
		return err
	}
	if err := s.registerCastleGM(); err != nil { // castle_gm.go
		return err
	}
	if f.data == nil {
		return nil
	}
	var stored []storage.Castle
	if f.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), residenceLoadTimeout)
		defer cancel()
		var err error
		if stored, err = f.store.LoadCastles(ctx); err != nil {
			return err
		}
	}
	byID := make(map[int32]storage.Castle, len(stored))
	for _, c := range stored {
		byID[c.ID] = c
	}
	for i := range f.data.Castles {
		d := &f.data.Castles[i]
		c := s.newCastle(d)
		if st, ok := byID[int32(d.ID)]; ok { //nolint:gosec // castle IDs are small
			c.restore(st)
		}
		f.castles = append(f.castles, c)
	}
	// The owners come from the clans (Mobius clan_data.hasCastle).
	for _, cl := range s.Clans() {
		if c := s.CastleByID(cl.CastleID); c != nil {
			c.ownerID = cl.ID
		}
	}
	f.loadedOK = true
	// Once the loop runs: the siege tasks (Siege.startAutoTask) and the hired
	// mercenaries' tickets (MercTicketManager.load).
	s.loop.After(0, func() {
		for _, c := range f.castles {
			s.startCastleAutoTask(c)
		}
		s.restoreMercTickets()
	})
	return nil
}

// residenceDefaults are Mobius's siege settings, for a config without the
// section.
func residenceDefaults() residenceConfig {
	return residenceConfig{cycle: 2, length: 120, maxFlags: 1, minLevel: 4, maxAttackers: 500, maxDefenders: 500,
		points: residence.DefaultReputationPoints}
}

// Castles returns the castles in ID order. Loop-only.
func (s *Server) Castles() []*Castle { return s.residence.castles }

// CastleByID returns the castle with the ID, or nil. Loop-only.
func (s *Server) CastleByID(id int32) *Castle {
	if id <= 0 || int(id) > len(s.residence.castles) {
		return nil
	}
	return s.residence.castles[id-1]
}

// CastleByName returns the castle with the name, ignoring case, or nil.
func (s *Server) CastleByName(name string) *Castle {
	for _, c := range s.residence.castles {
		if strings.EqualFold(c.Name, strings.TrimSpace(name)) {
			return c
		}
	}
	return nil
}

// CastleByOwner returns the castle the clan owns, or nil (Mobius
// getCastleByOwner). Loop-only.
func (s *Server) CastleByOwner(c *Clan) *Castle {
	if c == nil {
		return nil
	}
	for _, cs := range s.residence.castles {
		if cs.ownerID == c.ID {
			return cs
		}
	}
	return nil
}

// CastleAt returns the castle whose siege zone holds pos (Mobius
// CastleManager.getCastle(x, y, z)), or nil. Loop-only.
func (s *Server) CastleAt(pos world.Position) *Castle {
	for _, c := range s.residence.castles {
		if c.Contains(pos) {
			return c
		}
	}
	return nil
}

// siegeAt returns the siege in progress whose zone holds pos, castles first
// (SiegeManager.getSiege, then the clan halls), or nil.
func (s *Server) siegeAt(pos world.Position) Siegable {
	for _, c := range s.residence.castles {
		if c.siege.inProgress && c.Contains(pos) {
			return c
		}
	}
	for _, o := range s.residence.others {
		if o.InProgress() && o.Contains(pos) {
			return o
		}
	}
	return nil
}

// registerSiegable adds a contestable clan hall's siege (M11j). Loop-only.
func (s *Server) registerSiegable(sg Siegable) { s.residence.others = append(s.residence.others, sg) }

// siegableByID returns the castle or hall with the residence ID, or nil.
func (s *Server) siegableByID(id int32) Siegable {
	if c := s.CastleByID(id); c != nil {
		return c
	}
	for _, o := range s.residence.others {
		if o.ResidenceID() == id {
			return o
		}
	}
	return nil
}

// broadcastSysMsg sends a system message to every player in the game (Mobius
// World.broadcastToAllOnlinePlayers).
func (s *Server) broadcastSysMsg(id int, params ...serverpackets.SystemMessageParam) {
	body := clanSysMsg(serverpackets.SystemMessageID(id), params...) //nolint:gosec // message IDs fit
	for p := range s.Players() {
		if p.state == stateInGame {
			p.Send(body)
		}
	}
}

// castleParam is the castle name parameter of a system message.
func castleParam(c *Castle) serverpackets.SystemMessageParam {
	return serverpackets.CastleNameParam(c.ID)
}

// zoneCastleID returns the castleId parameter of a zone, 0 for none.
func zoneCastleID(z *data.Zone) int32 {
	for _, key := range []string{"castleId", "residenceId"} {
		if v, ok := z.Param(key); ok {
			if id, err := strconv.Atoi(v); err == nil {
				return int32(id) //nolint:gosec // small
			}
		}
	}
	return 0
}

// errNoCastle is what castle commands answer for an unknown castle.
var errNoCastle = errors.New("no such castle")
