package game

import (
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/social"
)

// The castle NPCs written in Java in Mobius: the castle gatekeeper's
// condition (Teleporter.validateCondition on castle ground) and the castle
// doorman (CastleDoorman). The chamberlain, siege manager and the others are
// Lua scripts (scripts/npcs/).

// castleTeleporterPage is Teleporter.showChatWindow on castle ground: the
// owner clan's members get the gatekeeper's page, others "-no", and
// everyone "-busy" during the siege. It reports false off castle ground.
func (s *Server) castleTeleporterPage(p *Player, n *Npc) bool {
	c := s.CastleAt(n.Position())
	if c == nil {
		return false
	}
	id := n.ID()
	switch {
	case c.siege.inProgress:
		s.sendPage(p, s.pageFile(p, id, "teleporter/castleteleporter-busy.htm").replaceObjectID(id).replace("%npcname%", n.Template.Name))
	case p.ClanID() != 0 && p.ClanID() == c.ownerID:
		s.showNpcPage(p, n, 0)
	default:
		s.sendPage(p, s.pageFile(p, id, "teleporter/castleteleporter-no.htm").replaceObjectID(id).replace("%npcname%", n.Template.Name))
	}
	return true
}

// doormanCastleOwner is CastleDoorman.isOwnerClan.
func (s *Server) doormanCastleOwner(p *Player, n *Npc) (*Castle, bool) {
	c := s.npcCastle(n)
	return c, c != nil && p.Clan() != nil && p.HasClanAccess(social.AccessCastleOpenDoor) && p.ClanID() == c.ownerID
}

// castleDoormanPage is Doorman.showChatWindow for a castle doorman.
func (s *Server) castleDoormanPage(p *Player, n *Npc) {
	id := n.ID()
	page := "doorman/" + strconv.Itoa(n.Template.ID)
	c, owner := s.doormanCastleOwner(p, n)
	switch {
	case !owner:
		page += "-no"
	case c.siege.inProgress:
		page += "-busy"
	}
	p.SendActionFailed()
	s.sendPage(p, s.pageFile(p, id, page+".htm").replaceObjectID(id))
}

// castleDoormanBypass is Doorman.onBypassFeedback's open_doors and
// close_doors for a castle doorman ("open_doors 1, 19210001, 19210002"):
// the owner clan opens and closes its castle's doors outside a siege.
func (s *Server) castleDoormanBypass(p *Player, n *Npc, command string) {
	open := strings.HasPrefix(command, "open_doors")
	if !open && !strings.HasPrefix(command, "close_doors") {
		return
	}
	c, owner := s.doormanCastleOwner(p, n)
	if !owner {
		return
	}
	if c.siege.inProgress {
		id := n.ID()
		s.sendPage(p, s.pageFile(p, id, "doorman/"+strconv.Itoa(n.Template.ID)+"-busy.htm").replaceObjectID(id))
		return
	}
	fields := strings.FieldsFunc(command, func(r rune) bool { return r == ' ' || r == ',' })
	if len(fields) < 2 {
		return
	}
	for _, f := range fields[2:] {
		doorID, err := strconv.Atoi(f)
		if err != nil {
			continue
		}
		d, ok := s.Door(doorID)
		if !ok || !c.Contains(d.Position()) {
			continue
		}
		if open {
			s.openDoor(d)
		} else {
			s.closeDoor(d)
		}
	}
}
