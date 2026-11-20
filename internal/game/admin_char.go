package game

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/IPeter/l2go/internal/storage"
)

// The character tools of the admin panel (the Char menu, //admin6, and the
// character pages), with Mobius's names, messages and pages: AdminEditChar
// (//character_info, //current_player, //edit_character), AdminExpSp's page
// (//add_exp_sp_to_character, the page //add_exp_sp and //remove_exp_sp
// show again).
func init() { registerFeature("admin_char", setupAdminChar) }

// adminClientProtocol is the game protocol the character page shows (Mobius
// GameClient.getProtocolVersion); L2Go only lets Interlude clients in.
const adminClientProtocol = 746

// setupAdminChar registers the character tools.
func setupAdminChar(s *Server) error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("character_info", admin, s.gmCharacterInfo),
		s.RegisterGMCommand("current_player", admin, s.gmCurrentPlayer),
		s.RegisterGMCommand("edit_character", admin, s.gmEditCharacter),
		s.RegisterGMCommand("add_exp_sp_to_character", admin, s.gmAddExpSPToCharacter),
	)
}

// gmCurrentPlayer is //current_player (Mobius AdminEditChar): the GM's own
// character page.
func (s *Server) gmCurrentPlayer(p *Player, args []string) {
	if len(args) > 0 {
		return // Mobius matches the whole command
	}
	s.adminShowCharacterInfo(p, p)
}

// gmCharacterInfo is //character_info [name] (Mobius AdminEditChar): the
// character page of that player, who becomes the GM's target, or of the
// GM's target. A name nobody online has shows nothing; no player target
// gets "Invalid target.".
func (s *Server) gmCharacterInfo(p *Player, args []string) {
	if len(args) > 0 {
		t, _ := s.PlayerByName(args[0])
		s.adminShowCharacterInfo(p, t)
		return
	}
	if t, ok := gmTargetPlayer(p); ok {
		s.adminShowCharacterInfo(p, t)
		return
	}
	p.SendSystemMessage(gmSysMsgInvalidTarget)
}

// adminShowCharacterInfo is AdminEditChar.showCharacterInfo: t (made the
// GM's target) or, when nil, the GM's player target, on charinfo.htm.
func (s *Server) adminShowCharacterInfo(gm, t *Player) {
	if t == nil {
		var ok bool
		if t, ok = gmTargetPlayer(gm); !ok {
			return
		}
	} else if gm.Target != t {
		s.SetTarget(gm, t)
	}
	s.adminCharacterPage(gm, t, "charinfo.htm")
}

// gmEditCharacter is //edit_character [name] (Mobius AdminEditChar): the
// edit page (charedit.htm) of that player or of the GM's player target.
func (s *Server) gmEditCharacter(p *Player, args []string) {
	if len(args) > 0 {
		if t, ok := s.PlayerByName(args[0]); ok {
			s.adminCharacterPage(p, t, "charedit.htm")
		}
		return
	}
	if t, ok := gmTargetPlayer(p); ok {
		s.adminCharacterPage(p, t, "charedit.htm")
		return
	}
	p.SendSystemMessage(gmSysMsgInvalidTarget)
}

// adminCharacterPage is AdminEditChar.gatherCharacterInfo: the admin page
// file with t's numbers filled in.
func (s *Server) adminCharacterPage(gm, t *Player, file string) {
	ip := "N/A"
	if t.session == nil {
		gm.SendMessage("Client is null.")
	} else {
		ip = gmRemoteHost(t.session.RemoteAddr())
	}
	v := t.stats.View()
	load := t.inv.Load()
	percent := 0.0
	if v.WeightLimit != 0 {
		percent = float64(float32(load) / float32(v.WeightLimit) * 100)
	}
	baseClass := t.Class().Name
	if c, ok := s.data.ClassTemplate(t.BaseClassID()); ok {
		baseClass = c.Name
	}
	protocol := "NULL"
	if t.session != nil {
		protocol = strconv.Itoa(adminClientProtocol)
	}
	inst := ""
	if id := t.InstanceID(); id != 0 {
		inst = fmt.Sprintf(`<tr><td>InstanceId:</td><td><a action="bypass -h admin_instance_spawns %d">%d</a></td></tr>`, id, id)
	}
	pos := t.Position()
	runSpeed := int(float64(v.RunSpd) * v.MoveMultiplier)
	pg := s.adminPageFile(gm, file)
	for _, r := range [][2]string{
		{"%name%", t.Name()},
		{"%level%", strconv.Itoa(t.Level())},
		{"%clan%", "null"}, // clans do not exist yet; Java prints the null
		{"%xp%", strconv.FormatInt(t.char.Exp, 10)},
		{"%sp%", strconv.Itoa(int(t.char.SP))},
		{"%class%", t.Class().Name},
		{"%ordinal%", strconv.Itoa(t.ClassID())},
		{"%classid%", strconv.Itoa(t.ClassID())},
		{"%baseclass%", baseClass},
		{"%x%", strconv.Itoa(int(pos.X))},
		{"%y%", strconv.Itoa(int(pos.Y))},
		{"%z%", strconv.Itoa(int(pos.Z))},
		{"%heading%", strconv.Itoa(int(pos.Heading))},
		{"%currenthp%", strconv.Itoa(int(t.CurHP))},
		{"%maxhp%", strconv.Itoa(int(v.MaxHP))},
		{"%karma%", strconv.Itoa(int(t.char.Karma))},
		{"%currentmp%", strconv.Itoa(int(t.CurMP))},
		{"%maxmp%", strconv.Itoa(int(v.MaxMP))},
		{"%pvpflag%", strconv.Itoa(int(t.PvPFlag))},
		{"%currentcp%", strconv.Itoa(int(t.CurCP))},
		{"%maxcp%", strconv.Itoa(int(v.MaxCP))},
		{"%pvpkills%", strconv.Itoa(int(t.char.PvPKills))},
		{"%pkkills%", strconv.Itoa(int(t.char.PKKills))},
		{"%currentload%", strconv.Itoa(int(load))},
		{"%maxload%", strconv.Itoa(int(v.WeightLimit))},
		{"%percent%", fmt.Sprintf("%.2f", percent)},
		{"%patk%", strconv.Itoa(int(v.PAtk))},
		{"%matk%", strconv.Itoa(int(v.MAtk))},
		{"%pdef%", strconv.Itoa(int(v.PDef))},
		{"%mdef%", strconv.Itoa(int(v.MDef))},
		{"%accuracy%", strconv.Itoa(int(v.Accuracy))},
		{"%evasion%", strconv.Itoa(int(v.Evasion))},
		{"%critical%", strconv.Itoa(int(v.Crit))},
		{"%runspeed%", strconv.Itoa(runSpeed)},
		{"%patkspd%", strconv.Itoa(int(v.PAtkSpd))},
		{"%matkspd%", strconv.Itoa(int(v.MAtkSpd))},
		{"%access%", adminAccessName(t.AccessLevel())},
		{"%account%", t.Account()},
		{"%ip%", ip},
		{"%protocol%", protocol},
		{"%hwid%", "N/A"},
		{"%ai%", adminIntention(t)},
		{"%autoplay%", ""}, // Interlude has no auto play
		{"%inst%", inst},
		{"%noblesse%", adminYesNo(t.IsNoble())},
	} {
		pg.replace(r[0], r[1])
	}
	s.sendPage(gm, pg)
}

// adminPageFile makes a page of the admin folder, with AdminHtml's text for
// a missing file.
func (s *Server) adminPageFile(p *Player, rel string) *htmlPage {
	full := adminHTMLFolder + rel
	content, ok := s.htmlFile(p, full)
	if !ok {
		s.log.Warn("game: missing admin page", "page", htmlPrefix+full)
		content = adminMissingPage(htmlPrefix + full)
	}
	return newPage(0, content)
}

// adminAccessName is the access level as the character page shows it: the
// level and its name (Mobius AccessLevel getLevel and getName).
func adminAccessName(l storage.AccessLevel) string {
	name := "User"
	switch {
	case l < storage.AccessLevelPlayer:
		name = "Banned"
	case l >= storage.AccessLevelAdmin:
		name = "Admin"
	}
	return strconv.Itoa(int(l)) + " (" + name + ")"
}

// adminIntention is the AI intention the character page shows: L2Go players
// have no AI object, so it is told from what the player does.
func adminIntention(p *Player) string {
	switch {
	case p.dead:
		return "IDLE"
	case p.Move != nil:
		return "MOVE_TO"
	case p.Target != nil && p.srv.InCombat(p):
		return "ATTACK"
	}
	return "IDLE"
}

func adminYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

// gmAddExpSPToCharacter is //add_exp_sp_to_character (Mobius AdminExpSp):
// the exp/SP page of the GM's target. Mobius matches the command as
// "admin_add_exp_sp" too, so it first answers the //add_exp_sp usage, as
// //add_exp_sp without numbers does.
func (s *Server) gmAddExpSPToCharacter(p *Player, _ []string) {
	if _, ok := gmTargetPlayer(p); !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
	}
	p.SendMessage("Usage: //add_exp_sp exp sp")
	s.adminExpSPPage(p)
}

// adminExpSPPage is AdminExpSp.addExpSp: the exp/SP page (expsp.htm) of the
// GM's player target, or "Invalid target.".
func (s *Server) adminExpSPPage(p *Player) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	pg := s.adminPageFile(p, "expsp.htm")
	pg.replace("%name%", t.Name())
	pg.replace("%level%", strconv.Itoa(t.Level()))
	pg.replace("%xp%", strconv.FormatInt(t.char.Exp, 10))
	pg.replace("%sp%", strconv.Itoa(int(t.char.SP)))
	pg.replace("%class%", t.Class().Name)
	s.sendPage(p, pg)
}
