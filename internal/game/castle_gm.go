package game

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// The castle GM commands (Mobius AdminSiege): //siege lists the castles and
// opens a castle's page; the page's buttons and the commands below take the
// castle's name first, as Mobius's do.

// castleGMCommands are the AdminSiege commands for castles.
var castleGMCommands = []string{
	"add_attacker", "add_defender", "add_guard", "list_siege_clans", "clear_siege_list", "move_defenders",
	"spawn_doors", "endsiege", "startsiege", "setsiegetime", "setcastle", "removecastle",
}

// registerCastleGM registers //siege and the castle commands.
func (s *Server) registerCastleGM() error {
	errs := []error{s.RegisterGMCommand("siege", storage.AccessLevelAdmin, func(p *Player, args []string) {
		s.gmCastle(p, "siege", args)
	})}
	for _, name := range castleGMCommands {
		errs = append(errs, s.RegisterGMCommand(name, storage.AccessLevelAdmin, func(p *Player, args []string) {
			s.gmCastle(p, name, args)
		}))
	}
	return errors.Join(errs...)
}

// gmCastle is AdminSiege.onCommand for the castles.
func (s *Server) gmCastle(p *Player, cmd string, args []string) {
	if len(args) == 0 {
		s.gmCastleList(p)
		return
	}
	c := s.CastleByName(args[0])
	if c == nil {
		p.SendMessage("Unknown castle " + args[0] + ".")
		return
	}
	target, _ := p.Target.(*Player)
	switch cmd {
	case "add_attacker":
		if target == nil {
			p.SendSystemMessage(siegeMsgInvalidTarget)
			return
		}
		s.RegisterAttacker(target, c, true)
	case "add_defender":
		if target == nil {
			p.SendSystemMessage(siegeMsgInvalidTarget)
			return
		}
		s.RegisterDefender(target, c, true)
	case "add_guard":
		id := 0
		if len(args) > 1 {
			id, _ = strconv.Atoi(args[1])
		}
		if id <= 0 {
			p.SendMessage("Usage: //add_guard castle npcId")
			return
		}
		// SiegeGuardManager.addSiegeGuard: a guard of the data kind, stored
		// with the hired ones (Mobius keeps it in castle_siege_guards).
		pos := p.Position()
		c.hired = append(c.hired, storage.HiredGuard{NpcID: int32(id), X: pos.X, Y: pos.Y, Z: pos.Z, Heading: pos.Heading}) //nolint:gosec // NPC IDs
		c.save()
	case "clear_siege_list":
		c.siege.reg.Clear()
		c.save()
	case "endsiege":
		s.EndSiege(c)
	case "list_siege_clans":
		s.ShowSiegeInfo(p, c)
	case "move_defenders":
		p.SendMessage("Not implemented yet.")
	case "setcastle":
		if target == nil || target.Clan() == nil {
			p.SendSystemMessage(siegeMsgInvalidTarget)
			return
		}
		s.setCastleOwner(c, target.Clan())
	case "removecastle":
		if c.Owner() == nil {
			p.SendMessage("Unable to remove castle.")
			return
		}
		s.setCastleOwner(c, nil)
	case "setsiegetime":
		s.gmSetSiegeTime(p, c, args[1:])
	case "spawn_doors":
		c.spawnDoors(false)
	case "startsiege":
		s.StartSiege(c)
	default:
		s.sendPage(p, s.pageFile(p, 0, "admin/castle.htm").replace("%castleName%", c.Name))
	}
}

// gmCastleList is showCastleSelectPage: three castles a row.
func (s *Server) gmCastleList(p *Player) {
	var b strings.Builder
	for i, c := range s.residence.castles {
		b.WriteString(`<td fixwidth=90><a action="bypass -h admin_siege ` + c.Name + `">` + c.Name + `</a></td>`)
		if i%3 == 2 {
			b.WriteString("</tr><tr>")
		}
	}
	pg := s.pageFile(p, 0, "admin/castles.htm").replace("%castles%", b.String())
	pg.replace("%siegableHalls%", "").replace("%clanhalls%", "").replace("%freeclanhalls%", "")
	s.sendPage(p, pg)
}

// gmSetSiegeTime is admin_setsiegetime: month, day, hour or min of the siege
// date, which must stay in the future; then the time page. Our extension:
// "now <minutes>" sets the siege that many minutes ahead (for demos).
func (s *Server) gmSetSiegeTime(p *Player, c *Castle, args []string) {
	if len(args) >= 2 {
		n, err := strconv.Atoi(args[1])
		if err != nil {
			p.SendMessage("Unable to change Siege Date")
			return
		}
		t := s.calendar.In(c.siegeDate)
		switch args[0] {
		case "month":
			t = time.Date(t.Year(), t.Month()+time.Month(n), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		case "day":
			t = time.Date(t.Year(), t.Month(), n, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		case "hour":
			t = time.Date(t.Year(), t.Month(), t.Day(), n, t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		case "min":
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), n, t.Second(), t.Nanosecond(), t.Location())
		case "now":
			t = s.Now().Add(time.Duration(n) * time.Minute)
		}
		switch {
		case t.Before(s.Now()):
			p.SendMessage("Unable to change Siege Date")
		case !t.Equal(c.siegeDate):
			s.SetSiegeDate(c, t)
			p.SendMessage("Castle siege time for castle " + c.Name + " has been changed.")
		}
	}
	date := s.calendar.In(c.siegeDate)
	pg := s.pageFile(p, 0, "admin/castlesiegetime.htm").replace("%castleName%", c.Name)
	pg.replace("%time%", date.Format("2006-01-02 15:04"))
	s.sendPage(p, pg)
}

// SetSiegeDate moves a castle's siege date and restarts its start task
// (Siege.saveSiegeDate). Loop-only.
func (s *Server) SetSiegeDate(c *Castle, t time.Time) {
	c.siegeDate = t
	c.save()
	s.scheduleSiegeStart(c, time.Second)
}
