package game

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/htmlutil"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// nonHTMLBypasses are the bypass prefixes Mobius accepts without a page
// that offered them (RequestBypassToServer._possibleNonHtmlCommands).
var nonHTMLBypasses = []string{
	"item_", "_bbs", "bbs", "_mail", "_friend", "_match", "_diary",
	"OlympiadArenaChange", "manor_menu_select", "report",
}

// notYetCommands are the bypass commands (lower case) of the Mobius bypass
// handlers whose features come in later milestones: trying clothes on,
// multisell, warehouses and freight, skill enchanting, augmenting, the
// blessing of protection, the lottery, festivals, observation, the rift and the rest
// (decision N6). They answer with the "not available yet" page; so does any
// other NPC command the server does not know yet, since the link cache
// already proved that a page offered it. The shops are in shop.go.
var notYetCommands = map[string]bool{
	"wear": true, "multisell": true, "exc_multisell": true,
	"depositp": true, "withdrawp": true, "withdrawsortedp": true,
	"depositc": true, "withdrawc": true, "withdrawsortedc": true,
	"package_deposit": true, "package_withdraw": true,
	"enchantskilllist": true, "augment": true,
	"giveblessing": true, "loto": true,
	"festival": true, "festivaldesc": true,
	"observe": true, "observesiege": true, "observeoracle": true,
	"enterrift": true, "changeriftroom": true, "exitrift": true,
	"territorystatus": true, "questlist": true, "findpvp": true,
	"tutorial_close": true, "voice": true, "npcviewmod": true,
}

// handleBypass handles RequestBypassToServer (0x21), a click on a "bypass"
// link (Mobius RequestBypassToServer.runImpl). The command must be one a
// page offered (the link cache), and the player must be within interaction
// range of the NPC that page came from; otherwise it is dropped. Then:
//
//   - "admin_<command>" runs the GM command;
//   - "npc_<objectId>_<command>" goes to that NPC, if it is within
//     interaction range (2D), followed by ActionFailed;
//   - other commands go to the bypass handlers (Chat, Link, Script,
//     player_help, the "not available yet" ones) with the page's NPC.
//
// An empty command disconnects the player, as in Mobius.
func (s *Server) handleBypass(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestBypassToServer(body)
	if err != nil {
		s.log.Debug("game: bad RequestBypassToServer", append(p.logArgs(), "err", err)...)
		return
	}
	command := req.Command
	if command == "" {
		s.log.Warn("game: empty bypass, disconnecting the player", p.logArgs()...)
		sess := p.Session()
		s.Leave(p, nil)
		if sess != nil {
			sess.CloseAfter(leaveWorldPacket, 0)
		}
		return
	}

	var origin world.ObjectID
	if needsHTMLValidation(command) {
		var ok bool
		origin, ok = p.HTML.Validate(command)
		if !ok {
			level := slog.LevelDebug
			if p.IsGM() {
				// A GM's dropped click is worth seeing: it shows what the
				// client sent for a button that did nothing.
				level = slog.LevelInfo
			}
			s.log.Log(context.Background(), level, "game: bypass not offered by a page, dropped", append(p.logArgs(), "bypass", command)...)
			return
		}
		if origin > 0 && !s.inInteractionRange(p, origin) {
			// Mobius drops it quietly: the player walked away with the
			// dialog open.
			return
		}
	}

	switch {
	case strings.HasPrefix(command, "admin_"):
		s.RunGMCommand(p, strings.TrimPrefix(command, "admin_"))
	case strings.HasPrefix(command, "npc_"):
		s.npcBypass(p, command)
		p.SendActionFailed()
	case strings.HasPrefix(command, "item_"):
		// No item has a dialog yet (Mobius item.onBypassFeedback).
		p.SendActionFailed()
	case strings.HasPrefix(command, "_") || strings.HasPrefix(command, "bbs"),
		strings.HasPrefix(command, "OlympiadArenaChange"),
		strings.HasPrefix(command, "manor_menu_select"),
		strings.HasPrefix(command, "report"),
		command == "come_here":
		// The community board, the Olympiad, the manor, bot reports and
		// GM NPC calls come later.
		s.log.Debug("game: bypass not handled yet", append(p.logArgs(), "bypass", command)...)
	default:
		var n *Npc
		if origin > 0 {
			n = s.Npc(origin)
		}
		s.runBypassHandler(p, n, command)
	}
}

// needsHTMLValidation reports whether a bypass must have been offered by a
// page.
func needsHTMLValidation(command string) bool {
	for _, prefix := range nonHTMLBypasses {
		if strings.HasPrefix(command, prefix) {
			return false
		}
	}
	return true
}

// inInteractionRange reports whether the object id is in the world within
// interaction range of p (Mobius LocationUtil.isInsideRangeOfObjectId with
// Npc.INTERACTION_DISTANCE, a 3D distance).
func (s *Server) inInteractionRange(p *Player, id world.ObjectID) bool {
	o, ok := s.world.Find(id)
	return ok && p.Position().InRange3D(o.Position(), s.npcInteractionRange())
}

// npcBypass handles "npc_<objectId>_<command>": the command goes to the NPC
// when it exists and p is within its interaction range on the X/Y plane.
func (s *Server) npcBypass(p *Player, command string) {
	end := -1
	if len(command) > 5 {
		if i := strings.IndexByte(command[5:], '_'); i >= 0 {
			end = 5 + i
		}
	}
	idText := command[len("npc_"):]
	if end > 0 {
		idText = command[len("npc_"):end]
	}
	if end < 0 || !bypassIsNumeric(idText) {
		return
	}
	id, err := strconv.ParseInt(idText, 10, 32)
	if err != nil {
		return
	}
	n := s.Npc(world.ObjectID(id))
	if n == nil || n.Template == nil || p.Position().Distance2D(n.Position()) > float64(s.npcInteractionRange()) {
		return
	}
	s.npcCommand(p, n, command[end+1:])
}

// bypassIsNumeric reports whether s is a non-empty string of ASCII digits (Mobius
// StringUtil.isNumeric).
func bypassIsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// npcCommand runs a command of n's page (Mobius Npc.onBypassFeedback and
// the overrides of its subclasses).
func (s *Server) npcCommand(p *Player, n *Npc, command string) {
	switch {
	case s.npcIs(n, "Teleporter"):
		if s.teleporterCommand(p, n, command) {
			return
		}
	case s.npcIs(n, "VillageMaster"):
		if s.villageMasterCommand(p, n, command) { // clan_vm.go
			return
		}
	case s.npcIs(n, "Doorman"):
		if strings.HasPrefix(command, "Chat") {
			s.showFirstPage(p, n)
			return
		}
		if strings.HasPrefix(command, "open_doors") || strings.HasPrefix(command, "close_doors") || strings.HasPrefix(command, "tele") {
			// Only the owners of a castle or clan hall may use them, and
			// nobody owns one yet; a plain doorman's doors come with doors.
			if !s.npcIs(n, "CastleDoorman") && !s.npcIs(n, "ClanHallDoorman") {
				s.showNotYet(p, n)
			}
			return
		}
	}
	// Mobius Npc.onBypassFeedback runs the handlers only when the player
	// can interact with the NPC: standing, in its instance and in range.
	if !s.npcCanInteract(p, n) {
		return
	}
	s.runBypassHandler(p, n, command)
}

// runBypassHandler runs the bypass handler of command's first word (Mobius
// BypassHandler, matched without regard to case) for n, which may be nil.
func (s *Server) runBypassHandler(p *Player, n *Npc, command string) {
	word, _, _ := strings.Cut(command, " ")
	switch word = strings.ToLower(word); {
	case word == "chat":
		s.bypassChat(p, n, command)
	case word == "link":
		s.bypassLink(p, n, command)
	case word == "script":
		s.bypassScript(p, n, command)
	case word == "player_help":
		s.bypassPlayerHelp(p, command)
	case n != nil && (word == "multisell" || word == "exc_multisell"):
		s.bypassMultisell(p, n, word, command)
	case n != nil && word == "depositp":
		s.bypassDepositP(p)
	case n != nil && word == "withdrawp":
		s.bypassWithdrawP(p)
	case n != nil && word == "skilllist":
		s.bypassSkillList(p, n) // skill_learn.go
	case n != nil && word == "supportmagic":
		s.bypassSupportMagic(p, n) // newbie.go
	case shopCommands[word]:
		s.shopBypass(p, n, word, command) // shop.go
	case n != nil:
		if !notYetCommands[word] {
			s.log.Debug("game: unknown NPC bypass, answered as not available yet", append(p.logArgs(), "npc", n.Template.ID, "bypass", command)...)
		}
		s.showNotYet(p, n)
	default:
		s.log.Debug("game: bypass without a handler", append(p.logArgs(), "bypass", command)...)
	}
}

// bypassChat handles "Chat <page>" (Mobius ChatLink): page N of the NPC's
// dialog; "Chat 0" of an NPC with a first-talk script runs the script.
func (s *Server) bypassChat(p *Player, n *Npc, command string) {
	if n == nil {
		return
	}
	value := 0
	if len(command) > 5 {
		if v, err := strconv.Atoi(command[5:]); err == nil {
			value = v
		}
	}
	if value == 0 && s.scripts != nil && s.scripts.HasFirstTalk(n.Template.ID) {
		s.scriptFirstTalk(p, n)
		return
	}
	s.showNpcPage(p, n, value)
}

// linkPages are the pages "Link <page>" may open (Mobius Link.VALID_LINKS).
var linkPages = map[string]bool{
	"adventurer_guildsman/AboutHighLevelGuilds.htm": true,
	"adventurer_guildsman/AboutNewLifeCrystals.htm": true,
	"clanHallDoorman/evolve.htm":                    true,
	"common/augmentation_01.htm":                    true,
	"common/augmentation_02.htm":                    true,
	"common/crafting_01.htm":                        true,
	"common/duals_01.htm":                           true,
	"common/duals_02.htm":                           true,
	"common/duals_03.htm":                           true,
	"common/g_cube_warehouse001.htm":                true,
	"common/weapon_sa_01.htm":                       true,
	"common/welcomeback002.htm":                     true,
	"common/welcomeback003.htm":                     true,
	"default/BlessingOfProtection.htm":              true,
	"default/SupportMagic.htm":                      true,
	"fisherman/fishing_championship.htm":            true,
	"fortress/foreman.htm":                          true,
	"olympiad/hero_main2.htm":                       true,
	"seven_signs/blkmrkt_1.htm":                     true,
	"seven_signs/blkmrkt_2.htm":                     true,
	"seven_signs/mammblack_1a.htm":                  true,
	"seven_signs/mammblack_1b.htm":                  true,
	"seven_signs/mammblack_1c.htm":                  true,
	"seven_signs/mammblack_2a.htm":                  true,
	"seven_signs/mammblack_2b.htm":                  true,
	"seven_signs/mammmerch_1.htm":                   true,
	"seven_signs/mammmerch_1a.htm":                  true,
	"seven_signs/mammmerch_1b.htm":                  true,
	"warehouse/clanwh.htm":                          true,
	"warehouse/privatewh.htm":                       true,
}

// bypassLink handles "Link <page>" (Mobius Link): one of the pages of
// linkPages, with %objectId% filled. Any other page gets a page without
// text, as Mobius sends (it then fails on it, which clears the NPC page's
// links).
func (s *Server) bypassLink(p *Player, n *Npc, command string) {
	page := bypassTrim(command[len("Link"):])
	if page == "" || strings.Contains(page, "..") {
		s.log.Warn("game: bad Link bypass", append(p.logArgs(), "bypass", command)...)
		return
	}
	var npc world.ObjectID
	if n != nil {
		npc = n.ID()
	}
	var content string
	ok := false
	if linkPages[page] {
		content, ok = s.htmlFile(p, page)
	}
	if strings.HasPrefix(page, "teleporter/") {
		// Mobius's precaution: teleporter pages only at a teleporter.
		if t, isNpc := p.Target.(*Npc); !isNpc || !s.npcIs(t, "Teleporter") {
			ok = false
		}
	}
	if !ok {
		s.log.Warn("game: Link to a page that is not allowed", append(p.logArgs(), "page", page)...)
		p.HTML.Clear(HTMLScopeNpc)
		p.Send(emptyNpcHTML(npc))
		return
	}
	s.sendPage(p, newPage(npc, strings.ReplaceAll(content, "%objectId%", strconv.Itoa(int(npc)))))
}

// emptyNpcHTML is an NpcHtmlMessage without text.
func emptyNpcHTML(npc world.ObjectID) []byte {
	return serverpackets.NewNpcHtmlMessagePacket(serverpackets.NpcHtmlMessage{NpcObjectID: int32(npc)})
}

// bypassPlayerHelp handles "player_help <page>[#<itemId>]" (Mobius
// PlayerHelp): a page of the help folder, not bound to an NPC; with an item
// ID it is an item page.
func (s *Server) bypassPlayerHelp(p *Player, command string) {
	if len(command) < 13 {
		return
	}
	arg := command[12:]
	if strings.Contains(arg, "..") {
		return
	}
	fields := strings.Fields(arg)
	if len(fields) == 0 {
		return
	}
	file, item, hasItem := strings.Cut(fields[0], "#")
	var itemID int64
	if hasItem {
		if before, _, more := strings.Cut(item, "#"); more {
			item = before
		}
		v, err := strconv.ParseInt(item, 10, 32)
		if err != nil {
			return
		}
		itemID = v
	}
	pg := s.pageFile(p, 0, "help/"+file)
	pg.itemID = int32(itemID)
	s.sendPage(p, pg)
}

// handleLinkHtml handles RequestLinkHtml (0x20), a click on a "link" link
// (Mobius RequestLinkHtml): the page, relative to the HTML folder, opens if
// a page offered the link and the player is within interaction range of
// that page's NPC.
func (s *Server) handleLinkHtml(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestLinkHtml(body)
	if err != nil {
		return
	}
	link := req.Link
	switch {
	case link == "":
		s.log.Warn("game: empty html link", p.logArgs()...)
		return
	case strings.Contains(link, ".."):
		s.log.Warn("game: invalid html link", append(p.logArgs(), "link", link)...)
		return
	}
	origin, ok := p.HTML.Validate(htmlutil.LinkPrefix + link)
	if !ok {
		s.log.Warn("game: html link not offered by a page", append(p.logArgs(), "link", link)...)
		return
	}
	if origin > 0 && !s.inInteractionRange(p, origin) {
		return
	}
	s.sendPage(p, s.pageFile(p, origin, link))
}
