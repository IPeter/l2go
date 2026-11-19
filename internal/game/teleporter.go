package game

import (
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// sysMsgNotEnoughAdena is "You do not have enough adena." (Mobius
// YOU_DO_NOT_HAVE_ENOUGH_ADENA).
const sysMsgNotEnoughAdena serverpackets.SystemMessageID = 279

// teleporterCommand runs the commands Mobius Teleporter.onBypassFeedback
// handles itself: showNoblesSelect, showTeleports [list], teleport <list>
// <index> and chat <page>. It reports false for any other command, which
// goes to the bypass handlers.
func (s *Server) teleporterCommand(p *Player, n *Npc, command string) bool {
	// StringTokenizer(command, " "): tokens split on spaces only.
	tokens := strings.FieldsFunc(command, func(r rune) bool { return r == ' ' })
	if len(tokens) == 0 {
		return true
	}
	switch tokens[0] {
	case "showNoblesSelect":
		// Noblesse comes in Milestone 11: everyone gets the "not a noble"
		// page.
		s.teleporterPage(p, n, "teleporter/not_nobles.htm")
	case "showTeleports":
		list := data.TeleportNormal
		if len(tokens) > 1 {
			list = tokens[1]
		}
		s.showTeleports(p, n, list)
	case "teleport":
		if len(tokens) != 3 {
			s.log.Warn("game: teleport bypass without a list and a location", append(p.logArgs(), "bypass", command)...)
			return true
		}
		index := -1
		if bypassIsNumeric(tokens[2]) {
			if v, err := strconv.Atoi(tokens[2]); err == nil {
				index = v
			}
		}
		s.doTeleport(p, n, tokens[1], index)
	case "chat":
		value := 0
		if len(command) > 5 {
			if v, err := strconv.Atoi(command[5:]); err == nil {
				value = v
			}
		}
		s.showNpcPage(p, n, value)
	default:
		return false
	}
	return true
}

// teleporterPage shows a teleporter page with %objectId% and %npcname%
// filled (Mobius Teleporter.sendHtmlMessage).
func (s *Server) teleporterPage(p *Player, n *Npc, rel string) {
	s.sendPage(p, s.pageFile(p, n.ID(), rel).replaceObjectID(n.ID()).replace("%npcname%", n.Template.Name))
}

// teleportListNobles reports whether a list is for nobles only.
func teleportListNobles(l *data.TeleportList) bool {
	return l.Type == data.TeleportNoblesAdena || l.Type == data.TeleportNoblesToken
}

// showTeleports shows the teleport list name of n (Mobius
// TeleportHolder.showTeleportList): teleporter/teleports.htm with
// %locations% filled by one link per location, "<name> - <fee> <item>" when
// p pays a fee, each asking for confirmation (client message 811).
func (s *Server) showTeleports(p *Player, n *Npc, name string) {
	list := s.teleportList(p, n, name)
	if list == nil {
		return
	}
	prefix := "npc_" + strconv.Itoa(int(n.ID())) + "_teleport " + list.Name + " "
	var b strings.Builder
	for i, loc := range list.Locations {
		locName := teleportLocName(loc)
		text := locName
		if s.paysTeleportFee(p, list, loc) {
			if fee := s.teleportFee(p, list, loc); fee != 0 {
				text += " - " + strconv.Itoa(fee) + " " + s.feeItemName(loc.FeeItemID)
			}
		}
		b.WriteString(`<a action="bypass -h ` + prefix + strconv.Itoa(i) + `" msg="811;` + locName + `">` + text + `</a><br1>`)
	}
	s.sendPage(p, s.pageFile(p, n.ID(), "teleporter/teleports.htm").replace("%locations%", b.String()))
}

// teleportList returns n's teleport list name, or nil (logged) when there
// is none or it is closed to p: the noblesse lists stay closed until
// noblesse exists (decision N7).
func (s *Server) teleportList(p *Player, n *Npc, name string) *data.TeleportList {
	if s.worldData == nil {
		return nil
	}
	list, ok := s.worldData.TeleportList(n.Template.ID, name)
	if !ok {
		s.log.Warn("game: unknown teleport list", append(p.logArgs(), "npc", n.Template.ID, "list", name)...)
		return nil
	}
	if teleportListNobles(list) {
		s.log.Warn("game: noblesse teleport list asked for by a player who is not a noble", append(p.logArgs(), "npc", n.Template.ID, "list", name)...)
		return nil
	}
	return list
}

// teleportLocName is a location's link text; Mobius writes a missing name
// as Java writes null.
func teleportLocName(loc data.TeleportLocation) string {
	if loc.Name == "" {
		return "null"
	}
	return loc.Name
}

// doTeleport teleports p to location index of n's list name (Mobius
// TeleportHolder.doTeleport): a player with karma is sent away from the
// normal lists, the fee is taken (not enough gives Mobius's message), then
// the teleport runs. Sieges, which can block destinations, do not exist yet.
func (s *Server) doTeleport(p *Player, n *Npc, name string, index int) {
	list := s.teleportList(p, n, name)
	if list == nil {
		return
	}
	if index < 0 || index >= len(list.Locations) {
		s.log.Warn("game: unknown teleport location", append(p.logArgs(), "npc", n.Template.ID, "list", name, "location", index)...)
		return
	}
	loc := list.Locations[index]
	if list.Type == data.TeleportNormal && !karmaCanUseGK && p.Karma() > 0 {
		p.SendMessage("Go away, you're not welcome here.")
		return
	}
	if s.paysTeleportFee(p, list, loc) && !s.destroyItemByItemID(p, int32(loc.FeeItemID), int64(s.teleportFee(p, list, loc)), true) { //nolint:gosec // item IDs fit in int32
		if loc.FeeItemID == adenaID {
			p.SendSystemMessage(sysMsgNotEnoughAdena)
		} else {
			p.SendMessage("You do not have enough " + s.feeItemName(loc.FeeItemID) + ".")
		}
		return
	}
	//nolint:gosec // world coordinates fit in int32
	dest := world.Position{X: int32(loc.X), Y: int32(loc.Y), Z: int32(loc.Z)}
	if err := s.Teleport(p, dest); err != nil {
		s.log.Warn("game: teleport failed", append(p.logArgs(), "npc", n.Template.ID, "list", name, "location", index, "err", err)...)
	}
}

// paysTeleportFee reports whether p pays for loc (Mobius
// TeleportHolder.shouldPayFee): always on lists other than NORMAL; on
// NORMAL lists above the free teleport level (subclasses, which also pay,
// do not exist yet), when the location has a fee.
func (s *Server) paysTeleportFee(p *Player, list *data.TeleportList, loc data.TeleportLocation) bool {
	if list.Type != data.TeleportNormal {
		return true
	}
	return p.Level() > s.settings.NPCs.MaxFreeTeleportLevel && loc.FeeItemID != 0 && loc.FeeCount > 0
}

// teleportFee returns what p pays for loc (Mobius
// TeleportHolder.calculateFee): nothing up to the free teleport level on a
// NORMAL list, half from 20:00 on Mondays and Tuesdays (server time),
// otherwise the location's fee.
func (s *Server) teleportFee(p *Player, list *data.TeleportList, loc data.TeleportLocation) int {
	if list.Type == data.TeleportNormal {
		if p.Level() <= s.settings.NPCs.MaxFreeTeleportLevel {
			return 0
		}
		if now := s.Now(); now.Hour() >= 20 && (now.Weekday() == time.Monday || now.Weekday() == time.Tuesday) {
			return loc.FeeCount / 2
		}
	}
	return loc.FeeCount
}

// feeItemName returns the name of a fee item (Mobius
// TeleportHolder.getItemName), with Mobius's names for its pseudo items.
func (s *Server) feeItemName(id int) string {
	if t, ok := s.data.ItemTemplate(id); ok {
		return t.Name
	}
	switch id {
	case -100:
		return "Player Commendation Points"
	case -200:
		return "Clan Reputation Points"
	case -300:
		return "Fame"
	}
	return "Unknown item: " + strconv.Itoa(id)
}
