package game

import (
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/bbs"
)

// The region board (Mobius RegionBoard): "_bbsloc" lists the nine regions
// with their castle's owner, alliance and tax; "_bbsloc;<i>" shows one with
// its lord, next siege and clan halls. Writing is a TODO in Mobius.

// regionCastle is what the region board shows of a castle.
type regionCastle struct {
	name       string
	ownerID    int32 // the owning clan, 0 for none
	taxPercent int
	siegeDate  time.Time
}

// regionCastles returns the castles in ID order (Mobius
// CastleManager.getCastles) from the castles of M11i (residence.go).
// Without castle data it gives the nine castles of Mobius's castle.sql with
// no owner, no tax and no siege date, so the board still opens.
func (s *Server) regionCastles() []regionCastle {
	var out []regionCastle
	for _, c := range s.Castles() {
		date := c.SiegeDate()
		if date.IsZero() {
			date = time.UnixMilli(0) // a castle.sql siegeDate of 0
		}
		out = append(out, regionCastle{name: c.Name, ownerID: c.OwnerID(), taxPercent: c.TaxPercent(), siegeDate: date})
	}
	if len(out) > 0 {
		return out
	}
	for _, name := range regionFallbackCastles {
		out = append(out, regionCastle{name: name, siegeDate: time.UnixMilli(0)})
	}
	return out
}

// regionFallbackCastles are the castles of Mobius's castle.sql, by ID.
var regionFallbackCastles = [9]string{"Gludio", "Dion", "Giran", "Oren", "Aden", "Innadril", "Goddard", "Rune", "Schuttgart"}

// regionHall is a clan hall of Mobius's clanhall table: its ID, name and
// location (the town's castle name).
type regionHall struct {
	id             int32
	name, location string
}

// regionHallData are the rows of Mobius's clanhall.sql, which the region
// page lists on a server without clan hall data (as regionFallbackCastles),
// owned by the clan whose HideoutID names them (Mobius clanhall.ownerId).
var regionHallData = []regionHall{
	{22, "Moonstone Hall", "Gludio"}, {23, "Onyx Hall", "Gludio"}, {24, "Topaz Hall", "Gludio"}, {25, "Ruby Hall", "Gludio"},
	{26, "Crystal Hall", "Gludin"}, {27, "Onyx Hall", "Gludin"}, {28, "Sapphire Hall", "Gludin"}, {29, "Moonstone Hall", "Gludin"},
	{30, "Emerald Hall", "Gludin"},
	{31, "The Atramental Barracks", "Dion"}, {32, "The Scarlet Barracks", "Dion"}, {33, "The Viridian Barracks", "Dion"},
	{36, "The Golden Chamber", "Aden"}, {37, "The Silver Chamber", "Aden"}, {38, "The Mithril Chamber", "Aden"},
	{39, "Silver Manor", "Aden"}, {40, "Gold Manor", "Aden"}, {41, "The Bronze Chamber", "Aden"},
	{42, "The Golden Chamber", "Giran"}, {43, "The Silver Chamber", "Giran"}, {44, "The Mithril Chamber", "Giran"},
	{45, "The Bronze Chamber", "Giran"}, {46, "Silver Manor", "Giran"},
	{47, "Moonstone Hall", "Goddard"}, {48, "Onyx Hall", "Goddard"}, {49, "Emerald Hall", "Goddard"}, {50, "Sapphire Hall", "Goddard"},
	{51, "Mont Chamber", "Rune"}, {52, "Astaire Chamber", "Rune"}, {53, "Aria Chamber", "Rune"}, {54, "Yiana Chamber", "Rune"},
	{55, "Roien Chamber", "Rune"}, {56, "Luna Chamber", "Rune"}, {57, "Traban Chamber", "Rune"},
	{58, "Eisen Hall", "Schuttgart"}, {59, "Heavy Metal Hall", "Schuttgart"}, {60, "Molten Ore Hall", "Schuttgart"},
	{61, "Titan Hall", "Schuttgart"},
}

// regionHalls returns the auctionable clan halls of a location with their
// owners (Mobius RegionBoard.getClanHallsByLocation), from the clan halls of
// M11j (clanhall.go), or regionHallData without them.
func (s *Server) regionHalls(location string) []bbs.Hall {
	var out []bbs.Hall
	if halls := s.ClanHalls(); len(halls) > 0 {
		for _, h := range halls {
			if h.siege != nil || h.Data.Location != location {
				continue
			}
			hall := bbs.Hall{ID: h.ID, Name: h.Name}
			if c := h.Owner(); c != nil {
				hall.OwnerID, hall.OwnerName, hall.OwnerLeader = c.ID, c.Name, c.LeaderName()
			}
			out = append(out, hall)
		}
		return out
	}
	for _, h := range regionHallData {
		if h.location != location {
			continue
		}
		hall := bbs.Hall{ID: h.id, Name: h.name}
		for _, c := range s.Clans() {
			if c.HideoutID == h.id {
				hall.OwnerID, hall.OwnerName, hall.OwnerLeader = c.ID, c.Name, c.LeaderName()
				break
			}
		}
		out = append(out, hall)
	}
	return out
}

// regionOwner returns the clan link, alliance and lord of a castle's owner.
func (s *Server) regionOwner(c regionCastle) (link, ally, lord string) {
	clan := s.Clan(c.ownerID)
	if clan == nil {
		return "NPC", "None", "NPC"
	}
	ally = "None"
	if clan.AllyName != "" {
		ally = clan.AllyName
	}
	return bbs.ClanLink(clan.ID, clan.Name), ally, clan.LeaderName()
}

// regionBoard handles "_bbsloc" and "_bbsloc;<region>".
func (s *Server) regionBoard(p *Player, command string) {
	castles := s.regionCastles()
	switch {
	case command == "_bbsloc":
		s.rememberBoardPage(p, "Region", command)
		row, ok := s.boardFile(p, "region/region_list.html")
		if !ok {
			return
		}
		if len(castles) < len(bbs.RegionNames) {
			s.log.Warn("game: the region board needs nine castles", "castles", len(castles))
			return
		}
		var list strings.Builder
		for i, region := range bbs.RegionNames {
			link, ally, _ := s.regionOwner(castles[i])
			r := strings.ReplaceAll(row, "%region_id%", strconv.Itoa(i))
			r = strings.ReplaceAll(r, "%region_name%", strconv.Itoa(region))
			r = strings.ReplaceAll(r, "%region_owning_clan%", link)
			r = strings.ReplaceAll(r, "%region_owning_clan_alliance%", ally)
			r = strings.ReplaceAll(r, "%region_tax_rate%", bbs.TaxText(castles[i].taxPercent))
			list.WriteString(r)
		}
		if html, ok := s.boardFile(p, "region/region.html"); ok {
			s.sendBoard(p, strings.ReplaceAll(html, "%region_list%", list.String()))
		}
	case strings.HasPrefix(command, "_bbsloc;"):
		s.rememberBoardPage(p, "Region>", command)
		text := strings.ReplaceAll(command, "_bbsloc;", "")
		id, ok := parseJavaInt(text)
		if !bbs.IsNumeric(text) || !ok {
			s.log.Warn("game: invalid region bypass", append(p.logArgs(), "bypass", command)...)
			return
		}
		if id < 0 || int(id) >= len(castles) {
			s.log.Warn("game: invalid region ID", append(p.logArgs(), "region", id)...)
			return
		}
		c := castles[id]
		html, ok := s.boardFile(p, "region/region_show.html")
		if !ok {
			return
		}
		link, ally, lord := s.regionOwner(c)
		html = strings.ReplaceAll(html, "%regionName%", c.name)
		html = strings.ReplaceAll(html, "%tax%", bbs.TaxText(c.taxPercent))
		html = strings.ReplaceAll(html, "%lord%", lord)
		html = strings.ReplaceAll(html, "%clanName%", link)
		html = strings.ReplaceAll(html, "%allyName%", ally)
		html = strings.ReplaceAll(html, "%siegeDate%", s.boardDate(bbs.SiegeDateFormat)(c.siegeDate))
		html = strings.ReplaceAll(html, "%hallsList%", bbs.HallsList(s.regionHalls(c.name)))
		s.sendBoard(p, html)
	}
}
