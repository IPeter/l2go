package game

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The character tools of the admin panel (the Char menu, //admin6, and the
// character pages), with Mobius's names, messages and pages:
//   - AdminEditChar: the character pages (//character_info,
//     //current_player, //edit_character), the lists (//show_characters,
//     //find_character, //find_ip, //find_account, //find_dualbox) and the
//     edits (//nokarma, //setpk, //setpvp, //setfame, //rec, //settitle,
//     //changename, //setsex, //setcolor, //settcolor, //setnoble,
//     //set_hp, //set_mp, //set_cp);
//   - AdminExpSp's page (//add_exp_sp_to_character, the page //add_exp_sp
//     and //remove_exp_sp show again), AdminAdmin's //sethero, AdminKick
//     and AdminMenu's //kick_menu, //ban_menu and //unban_menu, and
//     AdminRepairChar (//repair, //restore);
//   - AdminPunishment (//punishment and its pages, //ban_char, //ban_acc,
//     //ban_chat, //jail and their undoing), with Mobius's ban, chat ban and
//     jail handlers.
//
// Punishments are kept on the characters, not in a punishments table: a
// character ban is access level -1 (enter.go refuses the character), an
// account ban is access level -1 of the account on the login server (sent
// over the login link; the login server refuses the account), and chat bans
// and jail are player variables holding when they end. Bans are permanent;
// IP, HWID and party bans, and account chat bans and jail, are answered as
// not available.
func init() {
	registerFeature("admin_char", setupAdminChar)
	// The punishments' hooks run after the zones of the dialog feature
	// (features set up in name order), so they see where the player is.
	registerFeature("punishment", setupPunishments)
}

// AccountAccess changes the access level of an account on the login server
// (*link.Client). It may block on the network; the game calls it off the
// loop.
type AccountAccess interface {
	ChangeAccessLevel(account string, level int) error
}

const (
	// adminClientProtocol is the game protocol the character page shows
	// (Mobius GameClient.getProtocolVersion); L2Go only lets Interlude
	// clients in.
	adminClientProtocol = 746
	// adminListPageSize is how many players a page of //show_characters
	// lists, and adminFindLimit after how many //find_character and
	// //find_ip stop.
	adminListPageSize = 20
	adminFindLimit    = 20
	// adminMaxRecommendations is the most recommendations a character can
	// have (Mobius setRecomHave).
	adminMaxRecommendations = 255
	// adminMaxTitleLength is the longest title, in UTF-16 units; a longer
	// one is cut to adminCutTitleLength (Mobius Creature.setTitle).
	adminMaxTitleLength = 21
	adminCutTitleLength = 20

	// chatBanVar and jailVar are the player variables of a chat ban and of
	// jail: when they end, in Unix milliseconds, or 0 for never.
	chatBanVar = "l2go_chat_ban"
	jailVar    = "l2go_jail"
	// punishmentTeleportDelay is how long after jail starts or ends the
	// player is teleported (Mobius TeleportTask after 2 s).
	punishmentTeleportDelay = 2 * time.Second

	// adminRepairX, Y and Z are where //repair puts a character (Mobius
	// AdminRepairChar: Talking Island).
	adminRepairX = -84318
	adminRepairY = 244579
	adminRepairZ = -3730
)

// System messages of the character tools.
const (
	sysMsgDoesNotExist       serverpackets.SystemMessageID = 2   // "$s1 does not exist."
	sysMsgChattingProhibited serverpackets.SystemMessageID = 147 // "Chatting is currently prohibited."
	sysMsgChatEnabled        serverpackets.SystemMessageID = 345 // "Chat enabled."
	sysMsgChatDisabled       serverpackets.SystemMessageID = 346 // "Chat disabled."
)

// The jail's places (Mobius JailZone): where the jailed go in, and where
// they come out (Floran).
var (
	jailInLoc  = world.Position{X: -114356, Y: -249645, Z: -2984}
	jailOutLoc = world.Position{X: 17836, Y: 170178, Z: -3507}
)

// adminCharFeature is the state of the character tools: the login link for
// account bans, and per player what Mobius keeps in memory only.
type adminCharFeature struct {
	accounts AccountAccess
	players  map[*Player]*adminCharState
}

// adminCharState is what a player has of the character tools while it is
// in the game.
type adminCharState struct {
	entered time.Time // when it entered the world (Mobius uptime)
	// hero is the hero status (Mobius _hero): set at selection for a
	// claimed hero (hero.go) and by //sethero, which is not stored.
	hero bool
	// nameColor and titleColor are //setcolor's and //settcolor's colors
	// while set.
	nameColor, titleColor       int32
	hasNameColor, hasTitleColor bool
	// jailMove is the pending teleport into or out of jail, jailEnd and
	// chatBanEnd the end of a timed jail or chat ban.
	jailMove, jailEnd, chatBanEnd world.TimerID
	// inJail is whether the player was in the jail zone after its last
	// move, to tell when it leaves the jail.
	inJail bool
}

// adminState returns p's state of the character tools, made on first use.
func (s *Server) adminState(p *Player) *adminCharState {
	if s.adminChar.players == nil {
		s.adminChar.players = make(map[*Player]*adminCharState)
	}
	st := s.adminChar.players[p]
	if st == nil {
		st = &adminCharState{}
		s.adminChar.players[p] = st
	}
	return st
}

// IsHero reports whether the player is a hero (Mobius isHero): a claimed
// hero of the Olympiad (hero.go), or one //sethero made.
func (p *Player) IsHero() bool {
	st := p.srv.adminChar.players[p]
	return st != nil && st.hero
}

// nameColor and titleColor are the colors UserInfo and CharInfo show.
func (p *Player) nameColor() int32 {
	if st := p.srv.adminChar.players[p]; st != nil && st.hasNameColor {
		return st.nameColor
	}
	return defaultNameColor
}

func (p *Player) titleColor() int32 {
	if st := p.srv.adminChar.players[p]; st != nil && st.hasTitleColor {
		return st.titleColor
	}
	return defaultTitleColor
}

// rawVar reads a player variable without making the quest state (the info
// packets run before the quests are loaded).
func (p *Player) rawVar(name string) (string, bool) {
	if v := p.quests.vars; v != nil {
		return v.Get(name)
	}
	v, ok := p.char.Vars[name]
	return v, ok
}

// punishmentEnd reads a punishment variable: when it ends (0 for never) and
// whether it is in force at now.
func punishmentEnd(value string, ok bool, now time.Time) (end int64, active bool) {
	if !ok {
		return 0, false
	}
	end, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return end, end <= 0 || end > now.UnixMilli()
}

// chatBanned reports whether the player is chat banned (Mobius
// isChatBanned).
func (p *Player) chatBanned() bool {
	v, ok := p.rawVar(chatBanVar)
	_, active := punishmentEnd(v, ok, p.srv.Now())
	return active
}

// jailed reports whether the player is jailed (Mobius isJailed).
func (p *Player) jailed() bool {
	v, ok := p.rawVar(jailVar)
	_, active := punishmentEnd(v, ok, p.srv.Now())
	return active
}

// chatPunished is Mobius Say2's chat ban and jail checks: a chat banned
// player says nothing (a text starting with "." still goes, for the voiced
// commands), and a jailed one cannot whisper, shout, trade or use hero
// voice. It reports whether the message was refused.
func (s *Server) chatPunished(p *Player, typ serverpackets.ChatType, text string) bool {
	if p.chatBanned() && !strings.HasPrefix(text, ".") {
		switch typ { // Mobius BanChatChannels
		case serverpackets.ChatGeneral, serverpackets.ChatShout, serverpackets.ChatTrade,
			serverpackets.ChatHeroVoice, serverpackets.ChatWhisper:
			p.SendSystemMessage(sysMsgChattingProhibited)
		}
		return true
	}
	if p.jailed() { // Mobius JailDisableChat
		switch typ {
		case serverpackets.ChatWhisper, serverpackets.ChatShout, serverpackets.ChatTrade, serverpackets.ChatHeroVoice:
			p.SendMessage("You can not chat with players outside of the jail.")
			return true
		}
	}
	return false
}

// setupAdminChar registers the character tools.
func setupAdminChar(s *Server) error {
	admin := storage.AccessLevelAdmin
	errs := []error{
		s.RegisterGMCommand("character_info", admin, s.gmCharacterInfo),
		s.RegisterGMCommand("current_player", admin, s.gmCurrentPlayer),
		s.RegisterGMCommand("edit_character", admin, s.gmEditCharacter),
		s.RegisterGMCommand("add_exp_sp_to_character", admin, s.gmAddExpSPToCharacter),
		s.RegisterGMCommand("character_list", admin, s.gmCharacterList),
		s.RegisterGMCommand("show_characters", admin, s.gmShowCharacters),
		s.RegisterGMCommand("find_character", admin, s.gmFindCharacter),
		s.RegisterGMCommand("find_ip", admin, s.gmFindIP),
		s.RegisterGMCommand("find_account", admin, s.gmFindAccount),
		s.RegisterGMCommand("find_dualbox", admin, s.gmFindDualbox("")),
		// L2Go knows no client traces, so the strict search is by IP too.
		s.RegisterGMCommand("strict_find_dualbox", admin, s.gmFindDualbox("strict_")),
		s.RegisterGMCommand("nokarma", admin, s.gmNoKarma),
		s.RegisterGMCommand("setpk", admin, s.gmSetPK),
		s.RegisterGMCommand("setpvp", admin, s.gmSetPvP),
		s.RegisterGMCommand("setfame", admin, s.gmSetFame),
		s.RegisterGMCommand("rec", admin, s.gmRec),
		s.RegisterGMCommand("settitle", admin, s.gmSetTitle),
		s.RegisterGMCommand("changename", admin, s.gmChangeName),
		s.RegisterGMCommand("setsex", admin, s.gmSetSex),
		s.RegisterGMCommand("setcolor", admin, s.gmSetColor(false)),
		s.RegisterGMCommand("settcolor", admin, s.gmSetColor(true)),
		s.RegisterGMCommand("setnoble", admin, s.gmSetNoble),
		s.RegisterGMCommand("sethero", admin, s.gmSetHero),
		s.RegisterGMCommand("kick", admin, s.gmKick),
		s.RegisterGMCommand("kick_non_gm", admin, s.gmKickNonGM),
		s.RegisterGMCommand("kick_menu", admin, s.gmKickMenu),
		s.RegisterGMCommand("ban_menu", admin, s.gmPunishMenu("ban_char")),
		s.RegisterGMCommand("unban_menu", admin, s.gmPunishMenu("unban_char")),
		s.RegisterGMCommand("repair", admin, s.gmRepair),
		s.RegisterGMCommand("restore", admin, s.gmRepair),
		s.RegisterGMCommand("punishment", admin, s.gmPunishment),
		s.RegisterGMCommand("punishment_add", admin, s.gmPunishmentAdd),
		s.RegisterGMCommand("punishment_remove", admin, s.gmPunishmentRemove),
	}
	for _, v := range []struct {
		name, affect, typ string
		add               bool
	}{
		{"ban_char", "CHARACTER", "BAN", true},
		{"unban_char", "CHARACTER", "BAN", false},
		{"ban_acc", "ACCOUNT", "BAN", true},
		{"unban_acc", "ACCOUNT", "BAN", false},
		{"ban_hwid", "HWID", "BAN", true},
		{"unban_hwid", "HWID", "BAN", false},
		{"ban_chat", "CHARACTER", "CHAT_BAN", true},
		{"unban_chat", "CHARACTER", "CHAT_BAN", false},
		{"jail", "CHARACTER", "JAIL", true},
		{"unjail", "CHARACTER", "JAIL", false},
	} {
		errs = append(errs, s.RegisterGMCommand(v.name, admin, s.gmPunishShortcut(v.affect, v.typ, v.add)))
	}
	for _, vital := range []string{"hp", "mp", "cp"} {
		errs = append(errs, s.RegisterGMCommand("set_"+vital, admin, s.gmSetVital(vital)))
	}
	return errors.Join(errs...)
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

// --- The character lists ---

// adminPlayersByUptime returns the players in the game, the last to enter
// first (Mobius sorts them by uptime).
func (s *Server) adminPlayersByUptime() []*Player {
	var list []*Player
	for p := range s.Players() {
		list = append(list, p)
	}
	slices.SortStableFunc(list, func(a, b *Player) int {
		return s.adminState(b).entered.Compare(s.adminState(a).entered)
	})
	return list
}

// adminPlayerRow is a player's row of the lists: name (to its character
// page), class and level.
func adminPlayerRow(p *Player) string {
	return `<tr><td width=80><a action="bypass -h admin_character_info ` + p.Name() + `">` + p.Name() +
		`</a></td><td width=110>` + p.Class().Name + `</td><td width=40>` + strconv.Itoa(p.Level()) + `</td></tr>`
}

// gmCharacterList is //character_list (Mobius AdminEditChar): the first
// page of the character list.
func (s *Server) gmCharacterList(p *Player, _ []string) { s.adminListCharacters(p, 0) }

// gmShowCharacters is //show_characters <page> (Mobius AdminEditChar): a
// page of the character list; the pager sends "page=<n>".
func (s *Server) gmShowCharacters(p *Player, args []string) {
	page, err := strconv.Atoi(strings.TrimPrefix(strings.Join(args, " "), "page="))
	if err != nil {
		p.SendMessage("Usage: //show_characters <page_number>")
		return
	}
	s.adminListCharacters(p, page)
}

// adminListCharacters is AdminEditChar.listCharacters: charlist.htm with
// 20 players a page and Mobius's pager.
func (s *Server) adminListCharacters(gm *Player, page int) {
	players := s.adminPlayersByUptime()
	pages := (len(players) + adminListPageSize - 1) / adminListPageSize
	page = max(page, 0)
	pager := ""
	if pages > 1 {
		pager = `<table width=280 cellspacing=0><tr>` + adminPager("bypass -h admin_show_characters", page, pages-1) + `</tr></table>`
	}
	if page > pages {
		page = pages - 1
	}
	var b strings.Builder
	start := max(page*adminListPageSize, 0)
	for i := start; i < len(players) && i < start+adminListPageSize; i++ {
		b.WriteString(adminPlayerRow(players[i]))
	}
	pg := s.adminPageFile(gm, "charlist.htm")
	pg.replace("%pages%", pager)
	pg.replace("%players%", b.String())
	s.sendPage(gm, pg)
}

// adminPager is Mobius's NextPrevPageHandler in ButtonsStyle: << < Page > >>
// as buttons, or as text where they lead nowhere.
func adminPager(bypass string, current, last int) string {
	link := func(page int, name string, text bool) string {
		if text {
			return "<td>" + name + "</td>"
		}
		return fmt.Sprintf(`<td><button action="%s page=%d" value="%s" width="30" height="15" back="sek.cbui94" fore="sek.cbui92"></td>`, bypass, page, name)
	}
	const sep = "<td align=center> | </td>"
	return link(0, "<<", current-1 < 0) + sep +
		link(current-1, "<", current <= 0) + sep +
		fmt.Sprintf(`<td align="center">Page: %d/%d</td>`, current+1, last+1) + sep +
		link(current+1, ">", current >= last) + sep +
		link(last, ">>", current+1 > last)
}

// adminFoundEnd is the end of the "Found %number% character%end%" line.
func adminFoundEnd(found int, none, tooMany string) string {
	switch {
	case found == 0:
		return none
	case found > adminFindLimit:
		return tooMany
	case found == 1:
		return "."
	}
	return "s."
}

// gmFindCharacter is //find_character <part> (Mobius AdminEditChar): the
// players whose name has part, ignoring case, at most 21.
func (s *Server) gmFindCharacter(p *Player, args []string) {
	if len(args) == 0 {
		p.SendMessage("Usage: //find_character <character_name>")
		s.adminListCharacters(p, 0)
		return
	}
	part := strings.ToLower(strings.Join(args, " "))
	found := 0
	var b strings.Builder
	for _, t := range s.adminPlayersByUptime() {
		if strings.Contains(strings.ToLower(t.Name()), part) {
			found++
			b.WriteString(adminPlayerRow(t))
		}
		if found > adminFindLimit {
			break
		}
	}
	pg := s.adminPageFile(p, "charfind.htm")
	pg.replace("%results%", b.String())
	if found > adminFindLimit {
		pg.replace("%number%", " more than 20")
	}
	pg.replace("%number%", strconv.Itoa(found))
	pg.replace("%end%", adminFoundEnd(found, "s. Please try again.", "s.<br>Please refine your search to see all of the results."))
	s.sendPage(p, pg)
}

// gmFindIP is //find_ip <ipv4 | disconnected> (Mobius AdminEditChar): the
// players connected from that address, or those without a connection.
func (s *Server) gmFindIP(p *Player, args []string) {
	val := strings.Join(args, " ")
	disconnected := val == "disconnected"
	if !disconnected {
		if addr, err := netip.ParseAddr(val); err != nil || !addr.Is4() {
			p.SendMessage("Usage: //find_ip <www.xxx.yyy.zzz>")
			s.adminListCharacters(p, 0)
			return
		}
	}
	found := 0
	var b strings.Builder
	for _, t := range s.adminPlayersByUptime() {
		if t.session == nil {
			if !disconnected {
				continue
			}
		} else if disconnected || gmRemoteHost(t.session.RemoteAddr()) != val {
			continue
		}
		found++
		b.WriteString(adminPlayerRow(t))
		if found > adminFindLimit {
			break
		}
	}
	pg := s.adminPageFile(p, "ipfind.htm")
	pg.replace("%results%", b.String())
	if found > adminFindLimit {
		pg.replace("%number%", " more than "+strconv.Itoa(found))
	}
	pg.replace("%ip%", val)
	pg.replace("%number%", strconv.Itoa(found))
	pg.replace("%end%", adminFoundEnd(found, "s. Maybe they got d/c? :)",
		"s.<br>In order to avoid you a client crash I won't <br1>display results beyond the 20th character."))
	s.sendPage(p, pg)
}

// gmFindAccount is //find_account <player> (Mobius AdminEditChar): the
// account of an online player and its other characters, read off the loop.
func (s *Server) gmFindAccount(p *Player, args []string) {
	name := strings.Join(args, " ")
	t, ok := s.PlayerByName(name)
	if !ok {
		p.SendMessage("Usage: //find_account <player_name>")
		s.adminListCharacters(p, 0)
		return
	}
	account, self := t.Account(), t.ObjectID()
	store := s.chars
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
		defer cancel()
		chars, err := store.ListByAccount(ctx, account)
		s.loop.Post(func() {
			if err != nil {
				s.log.Error("game: cannot list an account's characters", "account", account, "err", err)
				p.SendMessage("The characters of the account could not be read.")
				return
			}
			var names []string
			for _, c := range chars {
				if c.ObjectID != self {
					names = append(names, c.Name)
				}
			}
			pg := s.adminPageFile(p, "accountinfo.htm")
			pg.replace("%account%", account)
			pg.replace("%player%", name)
			pg.replace("%characters%", strings.Join(names, "<br1>"))
			s.sendPage(p, pg)
		})
	}()
}

// gmFindDualbox is //find_dualbox [n] and //strict_find_dualbox [n] (Mobius
// AdminEditChar): the addresses with n (2 by default) or more players.
func (s *Server) gmFindDualbox(strict string) GMCommand {
	return func(p *Player, args []string) {
		multibox := 2
		if v, err := strconv.Atoi(strings.Join(args, " ")); err == nil {
			if v < 1 {
				p.SendMessage("Usage: //" + strict + "find_dualbox [number > 0]")
				return
			}
			multibox = v
		}
		counts := make(map[string]int)
		for _, t := range s.adminPlayersByUptime() {
			if t.session != nil {
				counts[gmRemoteHost(t.session.RemoteAddr())]++
			}
		}
		var ips []string
		for ip, n := range counts {
			if n >= multibox {
				ips = append(ips, ip)
			}
		}
		slices.SortFunc(ips, func(a, b string) int { return cmp.Or(cmp.Compare(counts[b], counts[a]), cmp.Compare(a, b)) })
		var b strings.Builder
		for _, ip := range ips {
			fmt.Fprintf(&b, `<a action="bypass -h admin_find_ip %s">%s (%d)</a><br1>`, ip, ip, counts[ip])
		}
		pg := s.adminPageFile(p, "dualbox.htm")
		pg.replace("%multibox%", strconv.Itoa(multibox))
		pg.replace("%results%", b.String())
		pg.replace("%strict%", strict)
		s.sendPage(p, pg)
	}
}

// --- The character edits ---

// adminIntArg parses the command's argument as one int (Mobius
// Integer.parseInt of the rest of the command).
func adminIntArg(args []string) (int32, bool) {
	v, err := strconv.ParseInt(strings.Join(args, " "), 10, 32)
	return int32(v), len(args) > 0 && err == nil
}

// gmNoKarma is //nokarma (Mobius AdminEditChar): //setkarma 0.
func (s *Server) gmNoKarma(p *Player, args []string) {
	if len(args) == 0 {
		s.gmSetKarma(p, []string{"0"})
	}
}

// gmSetCount is //setpk and //setpvp (Mobius AdminEditChar): the target
// player's PK or PvP count, then its UserInfo and CharInfo.
func (s *Server) gmSetCount(p *Player, args []string, what, usage string, set func(t *Player, n int32)) {
	n, ok := adminIntArg(args)
	if !ok {
		p.SendMessage(usage)
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	set(t, n)
	s.RequestSave(t)
	t.BroadcastUserInfo()
	t.SendMessage(fmt.Sprintf("A GM changed your %s count to %d", what, n))
	p.SendMessage(fmt.Sprintf("%s's %s count changed to %d", t.Name(), what, n))
}

// gmSetPK is //setpk <count>.
func (s *Server) gmSetPK(p *Player, args []string) {
	s.gmSetCount(p, args, "PK", "Usage: //setpk <pk_count>", func(t *Player, n int32) { t.char.PKKills = n })
}

// gmSetPvP is //setpvp <count>. (Mobius's PvP title colors are off by
// default.)
func (s *Server) gmSetPvP(p *Player, args []string) {
	s.gmSetCount(p, args, "PVP", "Usage: //setpvp <pvp_count>", func(t *Player, n int32) { t.char.PvPKills = n })
}

// gmSetFame is //setfame: Interlude has no fame, so the answer is Mobius's
// for a server with the fame system off.
func (s *Server) gmSetFame(p *Player, _ []string) {
	p.SendMessage("Fame system is not enabled on the server!")
}

// gmRec is //rec <count> (Mobius AdminEditChar): the recommendations the
// target player has, 0 to 255.
func (s *Server) gmRec(p *Player, args []string) {
	n, ok := adminIntArg(args)
	if !ok {
		p.SendMessage("Usage: //rec number")
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	t.char.RecomHave = min(max(n, 0), adminMaxRecommendations)
	s.RequestSave(t)
	t.BroadcastUserInfo()
	t.SendMessage(fmt.Sprintf("A GM changed your Recommend points to %d", n))
	p.SendMessage(fmt.Sprintf("%s's Recommend changed to %d", t.Name(), n))
}

// gmSetTitle is //settitle <title> (Mobius AdminEditChar): the target
// player's title, cut to 20 characters when longer than 21.
func (s *Server) gmSetTitle(p *Player, args []string) {
	if len(args) == 0 {
		p.SendMessage("You need to specify the new title.")
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		return
	}
	title := strings.Join(args, " ")
	if units := utf16.Encode([]rune(title)); len(units) > adminMaxTitleLength {
		title = string(utf16.Decode(units[:adminCutTitleLength]))
	}
	t.char.Title = title
	s.RequestSave(t)
	t.SendMessage("Your title has been changed by a GM")
	s.broadcastTitleInfo(t)
}

// broadcastTitleInfo is Mobius broadcastTitleInfo: UserInfo to the player
// and NicknameChanged to it and the players that see it.
func (s *Server) broadcastTitleInfo(p *Player) {
	p.UpdateUserInfo()
	body := serverpackets.NewNicknameChangedPacket(p.ObjectID(), p.Title())
	p.Send(body)
	if !p.InWorld() {
		return
	}
	for v := range s.world.VisiblePlayers(p) {
		if s.world.CanSee(v, p) {
			world.Send(v, body)
		}
	}
}

// gmChangeName is //changename <name> (Mobius AdminEditChar): the target
// player's new name, stored off the loop; a name another character has is
// refused.
func (s *Server) gmChangeName(p *Player, args []string) {
	if len(args) == 0 {
		p.SendMessage("Usage: //setname new_name_for_target")
		return
	}
	t, ok := gmTargetPlayer(p)
	if !ok {
		return
	}
	name, id, store := strings.Join(args, " "), t.ObjectID(), s.chars
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
		defer cancel()
		err := store.Rename(ctx, id, name)
		s.loop.Post(func() {
			switch {
			case errors.Is(err, storage.ErrExists):
				p.SendMessage("Warning, player " + name + " already exists")
				return
			case err != nil:
				s.log.Error("game: cannot rename a character", "objectId", id, "name", name, "err", err)
				p.SendMessage("The name could not be changed.")
				return
			}
			p.SendMessage("Changed name to " + name)
			if t.state != stateInGame {
				return
			}
			if old := storage.NameKey(t.char.Name); s.byName[old] == t {
				delete(s.byName, old)
			}
			t.char.Name = name
			s.byName[storage.NameKey(name)] = t
			t.SendMessage("Your name has been changed by a GM.")
			t.BroadcastUserInfo()
		})
	}()
}

// gmSetSex is //setsex (Mobius AdminEditChar): the target player becomes
// the other sex.
func (s *Server) gmSetSex(p *Player, _ []string) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		return
	}
	t.char.Sex = 1 - min(max(t.char.Sex, 0), 1)
	t.stats.SetFemale(t.char.Sex == 1)
	s.RequestSave(t)
	t.SendMessage("Your gender has been changed by a GM")
	t.BroadcastUserInfo()
}

// gmSetColor is //setcolor <hex> and //settcolor <hex> (Mobius
// AdminEditChar): the target player's name or title color, until it leaves.
func (s *Server) gmSetColor(title bool) GMCommand {
	return func(p *Player, args []string) {
		const bad = "You need to specify a valid new color."
		if len(args) == 0 {
			p.SendMessage(bad)
			return
		}
		t, ok := gmTargetPlayer(p)
		if !ok {
			return
		}
		hex := strings.Join(args, " ")
		v, err := strconv.ParseInt(hex, 16, 32)
		if err != nil || strings.ContainsAny(hex[:1], "+-") {
			p.SendMessage(bad)
			return
		}
		st := s.adminState(t)
		if title {
			st.titleColor, st.hasTitleColor = int32(v), true
			t.SendMessage("Your title color has been changed by a GM")
		} else {
			st.nameColor, st.hasNameColor = int32(v), true
			t.SendMessage("Your name color has been changed by a GM")
		}
		t.BroadcastUserInfo()
	}
}

// gmSetNoble is //setnoble (Mobius AdminEditChar): the target player, or
// the GM without a target, gains or loses noblesse (SetNoble).
func (s *Server) gmSetNoble(p *Player, args []string) {
	if len(args) > 0 {
		return // Mobius matches the whole command
	}
	var t *Player
	if p.Target == nil {
		t = p
	} else if tp, ok := gmTargetPlayer(p); ok {
		t = tp
	}
	if t == nil {
		return
	}
	t.SetNoble(!t.IsNoble())
	if t != p {
		p.SendMessage("You've changed nobless status of: " + t.Name())
	}
	t.SendMessage("GM changed your nobless status!")
}

// gmSetHero is //sethero (Mobius AdminAdmin): the target player, or the GM
// when the target is no player, gains or loses the hero status, until it
// leaves.
func (s *Server) gmSetHero(p *Player, _ []string) {
	if p.Target == nil {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	t := p
	if tp, ok := gmTargetPlayer(p); ok {
		t = tp
	}
	s.setHero(t, !t.IsHero())
	t.BroadcastUserInfo()
}

// setHero is Mobius setHero: the hero skills are given (not stored) on the
// base class, or taken, and the skill list sent.
func (s *Server) setHero(p *Player, hero bool) {
	if d, c := s.skills.data, s.skills.catalog; d != nil && c != nil {
		for _, l := range d.OtherSkillTree(data.TreeHero) {
			if !hero || p.BaseClassID() != p.ClassID() {
				s.RemoveSkill(p, l.SkillID, false, true)
				continue
			}
			if sk, ok := c.Skill(l.SkillID, l.Level); ok {
				s.AddSkill(p, sk, false)
			}
		}
	}
	s.adminState(p).hero = hero
	p.SendSkillList()
}

// gmSetVital is //set_hp, //set_mp and //set_cp <value> (Mobius
// AdminEditChar): the target creature's current HP, MP or CP; a player gets
// its UserInfo.
func (s *Server) gmSetVital(vital string) GMCommand {
	return func(p *Player, args []string) {
		var t Creature
		switch c := p.Target.(type) {
		case *Player:
			t = c
		case *Npc:
			t = c
		}
		if t == nil {
			p.SendSystemMessage(gmSysMsgInvalidTarget)
			return
		}
		v, err := 0.0, errors.New("no value")
		if len(args) > 0 {
			v, err = strconv.ParseFloat(args[0], 64)
		}
		if err != nil || math.IsNaN(v) {
			p.SendMessage("Usage: //set_" + vital + " 1000")
			return
		}
		switch c := t.(type) {
		case *Player:
			switch vital {
			case "hp":
				c.SetCurrentHP(v)
			case "mp":
				c.SetCurrentMP(v)
			default:
				c.SetCurrentCP(v)
			}
			c.UpdateUserInfo()
		case *Npc:
			switch vital {
			case "hp":
				c.SetCurrentHP(v)
			case "mp":
				c.SetCurrentMP(v)
			} // an NPC has no CP
		}
	}
}

// --- Kicks ---

// adminKick takes t out of the game the way Mobius Disconnection does: it
// is saved and its connection closes after final.
func (s *Server) adminKick(t *Player, final []byte) {
	sess := t.session
	s.log.Info("game: player kicked by a GM", t.logArgs()...)
	s.leave(t, true, nil)
	if sess != nil {
		sess.CloseAfter(final, 0)
	}
}

// gmKick is //kick <name> (Mobius AdminKick).
func (s *Server) gmKick(p *Player, args []string) {
	if len(args) == 0 {
		return
	}
	if t, ok := s.PlayerByName(args[0]); ok {
		s.adminKick(t, leaveWorldPacket)
		p.SendMessage("You kicked " + t.Name() + " from the game.")
	}
}

// gmKickNonGM is //kick_non_gm (Mobius AdminKick): every player below GM
// level is kicked.
func (s *Server) gmKickNonGM(p *Player, _ []string) {
	var kick []*Player
	for t := range s.Players() {
		if !t.IsGM() {
			kick = append(kick, t)
		}
	}
	for _, t := range kick {
		s.adminKick(t, leaveWorldPacket)
	}
	p.SendMessage(fmt.Sprintf("Kicked %d players.", len(kick)))
}

// gmKickMenu is //kick_menu <name> (Mobius AdminMenu): //kick with an
// answer either way, then the character page.
func (s *Server) gmKickMenu(p *Player, args []string) {
	if len(args) > 0 {
		if t, ok := s.PlayerByName(args[0]); ok {
			s.adminKick(t, leaveWorldPacket)
			p.SendMessage("You kicked " + t.Name() + " from the game.")
		} else {
			p.SendMessage("Player " + args[0] + " was not found in the game.")
		}
	}
	s.ShowAdminPage(p, "charmanage.htm")
}

// gmPunishMenu is //ban_menu and //unban_menu <name> (Mobius AdminMenu):
// //ban_char or //unban_char, then the character page.
func (s *Server) gmPunishMenu(cmd string) GMCommand {
	return func(p *Player, args []string) {
		page := func() { s.ShowAdminPage(p, "charmanage.htm") }
		if len(args) == 0 {
			page()
			return
		}
		s.adminPunish(p, args[0], "CHARACTER", "BAN", 0, "Banned by admin", cmd == "ban_char", page)
	}
}

// --- Repair ---

// gmRepair is //repair <name> and //restore <name> (Mobius
// AdminRepairChar): an offline character goes to Talking Island, loses its
// shortcuts, and all its items, worn or kept, go to its inventory. Mobius
// says nothing; L2Go tells the GM what happened.
func (s *Server) gmRepair(p *Player, args []string) {
	if len(args) != 1 {
		return
	}
	name := args[0]
	if _, online := s.PlayerByName(name); online {
		p.SendMessage("The character " + name + " is online; log it out to repair it.")
		return
	}
	s.adminWithCharacter(p, name, func() { p.SendMessage("No character " + name + ".") }, func(t punishTarget) bool {
		if t.c == nil {
			p.SendMessage("The character " + name + " is online; log it out to repair it.")
			return false
		}
		c := t.c
		c.X, c.Y, c.Z = adminRepairX, adminRepairY, adminRepairZ
		c.Shortcuts = nil
		for i := range c.Items {
			c.Items[i].Loc, c.Items[i].Slot = storage.LocInventory, storage.PaperdollNone
		}
		return true
	}, func() { p.SendMessage("Character " + name + " repaired.") })
}

// --- Punishments ---

// The punishment types and the affects of the punishment pages (Mobius
// PunishmentType and PunishmentAffect).
var (
	punishmentTypes   = []string{"BAN", "CHAT_BAN", "PARTY_BAN", "JAIL"}
	punishmentAffects = []string{"ACCOUNT", "CHARACTER", "IP", "HWID"}
)

// punishmentName returns the name of list s names, ignoring case, or "".
func punishmentName(list []string, s string) string {
	for _, n := range list {
		if strings.EqualFold(n, s) {
			return n
		}
	}
	return ""
}

// punishTarget is a character a punishment changes: online (p) or loaded
// from the store (c).
type punishTarget struct {
	p *Player
	c *storage.Character
}

func (t punishTarget) varValue(name string) (string, bool) {
	if t.p != nil {
		return t.p.rawVar(name)
	}
	v, ok := t.c.Vars[name]
	return v, ok
}

func (t punishTarget) setVar(name, value string) {
	if t.p != nil {
		t.p.SetVar(name, value)
		return
	}
	if t.c.Vars == nil {
		t.c.Vars = make(map[string]string)
	}
	t.c.Vars[name] = value
}

func (t punishTarget) removeVar(s *Server, name string) {
	if t.p != nil {
		s.questsOf(t.p).vars.Remove(name)
		s.scheduleQuestSave(t.p)
		return
	}
	delete(t.c.Vars, name)
}

func (t punishTarget) access() storage.AccessLevel {
	if t.p != nil {
		return t.p.char.AccessLevel
	}
	return t.c.AccessLevel
}

func (t punishTarget) setAccess(l storage.AccessLevel) {
	if t.p != nil {
		t.p.char.AccessLevel = l
		return
	}
	t.c.AccessLevel = l
}

// has reports whether the character has the punishment at now, and when it
// ends (0 for never).
func (t punishTarget) has(typ string, now time.Time) (end int64, ok bool) {
	switch typ {
	case "BAN":
		return 0, t.access() < storage.AccessLevelPlayer
	case "CHAT_BAN":
		v, set := t.varValue(chatBanVar)
		return punishmentEnd(v, set, now)
	case "JAIL":
		v, set := t.varValue(jailVar)
		return punishmentEnd(v, set, now)
	}
	return 0, false
}

// adminWithCharacter runs edit on the character called name: at once when
// it is online, otherwise on the loop once it is loaded off the loop (after
// its pending saves), and when edit then reports true the character is
// written back off the loop, after which saved (if not nil) runs on the
// loop. Without such a character notFound runs (nil answers "$s1 does not
// exist.").
func (s *Server) adminWithCharacter(gm *Player, name string, notFound func(), edit func(t punishTarget) bool, saved func()) {
	if p, ok := s.PlayerByName(name); ok {
		edit(punishTarget{p: p})
		return
	}
	if notFound == nil {
		notFound = func() { gm.SendSystemMessage(sysMsgDoesNotExist, serverpackets.TextParam(name)) }
	}
	store, saver := s.chars, s.saver
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
		defer cancel()
		c, err := store.GetByName(ctx, name)
		if err == nil {
			if err = saver.wait(ctx, c.ObjectID); err == nil {
				c, err = store.Get(ctx, c.ObjectID)
			}
		}
		s.loop.Post(func() {
			switch {
			case errors.Is(err, storage.ErrNotFound):
				notFound()
				return
			case err != nil:
				s.log.Error("game: cannot load a character for a GM", "name", name, "err", err)
				gm.SendMessage("The character " + name + " could not be loaded.")
				return
			}
			if p, ok := s.PlayerByName(name); ok { // logged in meanwhile
				edit(punishTarget{p: p})
				return
			}
			if !edit(punishTarget{c: &c}) {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
				defer cancel()
				if err := store.Update(ctx, &c); err != nil {
					s.log.Error("game: cannot store a character a GM changed", "name", name, "err", err)
					s.loop.Post(func() { gm.SendMessage("The character " + name + " could not be saved.") })
					return
				}
				if saved != nil {
					s.loop.Post(saved)
				}
			}()
		})
	}()
}

// gmPunishment is //punishment (Mobius AdminPunishment): the punishment
// panel; "info <key> <affect>" the punishments of a character, an account,
// an IP or a HWID; "player [name]" the panel of a player or the target.
func (s *Server) gmPunishment(p *Player, args []string) {
	if len(args) == 0 {
		content, ok := s.htmlFile(p, "admin/punishments/punishment.htm")
		if !ok {
			s.log.Warn("game: missing admin page", "page", htmlPrefix+"admin/punishments/punishment.htm")
			return
		}
		pg := newPage(0, content)
		pg.replace("%punishments%", strings.Join(punishmentTypes, ";"))
		pg.replace("%affects%", strings.Join(punishmentAffects, ";"))
		s.sendPage(p, pg)
		return
	}
	switch args[0] {
	case "info":
		if len(args) < 3 {
			p.SendMessage("Not enough data specified!")
			return
		}
		affect := punishmentName(punishmentAffects, args[2])
		if affect == "" {
			p.SendMessage("Incorrect value specified for affect type!")
			return
		}
		s.adminPunishmentInfo(p, args[1], affect)
	case "player":
		var t *Player
		if len(args) > 1 {
			t, _ = s.PlayerByName(args[1])
		}
		if t == nil {
			var ok bool
			if t, ok = gmTargetPlayer(p); !ok {
				p.SendMessage("You must target player!")
				return
			}
		}
		content, ok := s.htmlFile(p, "admin/punishments/punishment-player.htm")
		if !ok {
			s.log.Warn("game: missing admin page", "page", htmlPrefix+"admin/punishments/punishment-player.htm")
			return
		}
		ip := "N/A"
		if t.session != nil {
			ip = gmRemoteHost(t.session.RemoteAddr())
		}
		pg := newPage(0, content)
		pg.replace("%player_name%", t.Name())
		pg.replace("%punishments%", strings.Join(punishmentTypes, ";"))
		pg.replace("%acc%", t.Account())
		pg.replace("%char%", t.Name())
		pg.replace("%ip%", ip)
		pg.replace("%hwid%", "Unknown")
		s.sendPage(p, pg)
	}
}

// adminPunishmentInfo is //punishment info: punishment-info.htm with the
// punishments of key. L2Go knows those of characters; an account's ban is
// on the login server, so accounts show none.
func (s *Server) adminPunishmentInfo(gm *Player, key, affect string) {
	show := func(rows string) {
		content, ok := s.htmlFile(gm, "admin/punishments/punishment-info.htm")
		if !ok {
			s.log.Warn("game: missing admin page", "page", htmlPrefix+"admin/punishments/punishment-info.htm")
			return
		}
		pg := newPage(0, content)
		pg.replace("%player_name%", key)
		pg.replace("%punishments%", rows)
		pg.replace("%affects%", strings.Join(punishmentAffects, ";"))
		pg.replace("%affect_type%", affect)
		s.sendPage(gm, pg)
	}
	if affect != "CHARACTER" {
		show("")
		return
	}
	s.adminWithCharacter(gm, key, nil, func(t punishTarget) bool {
		var b strings.Builder
		for _, typ := range punishmentTypes {
			end, ok := t.has(typ, s.Now())
			if !ok {
				continue
			}
			expire := "never"
			if end > 0 {
				expire = time.UnixMilli(end).Local().Format("02/01/2006 15:04:05")
			}
			fmt.Fprintf(&b, `<tr><td><font color="LEVEL">%s</font></td><td>%s</td><td><a action="bypass -h admin_punishment_remove %s %s %s">Remove</a></td></tr>`,
				typ, expire, key, affect, typ)
		}
		show(b.String())
		return false
	}, nil)
}

// gmPunishmentAdd is //punishment_add <key> <affect> <type> <minutes>
// <reason> (Mobius AdminPunishment): 0 or -1 minutes is for ever.
func (s *Server) gmPunishmentAdd(p *Player, args []string) {
	if len(args) < 5 {
		p.SendMessage("Please fill all the fields!")
		return
	}
	minutes, err := strconv.ParseInt(args[3], 10, 32)
	if err != nil || (minutes < 0 && minutes != -1) || strings.HasPrefix(args[3], "+") {
		p.SendMessage("Incorrect value specified for expiration time!")
		return
	}
	affect, typ := punishmentName(punishmentAffects, args[1]), punishmentName(punishmentTypes, args[2])
	if affect == "" || typ == "" {
		p.SendMessage("Incorrect value specified for affect/punishment type!")
		return
	}
	var end int64
	if minutes > 0 {
		end = s.Now().Add(time.Duration(minutes) * time.Minute).UnixMilli()
	}
	reason := strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(strings.Join(args[4:], " "))
	s.adminPunish(p, args[0], affect, typ, end, reason, true, nil)
}

// gmPunishmentRemove is //punishment_remove <key> <affect> <type> (Mobius
// AdminPunishment).
func (s *Server) gmPunishmentRemove(p *Player, args []string) {
	if len(args) < 3 {
		p.SendMessage("Not enough data specified!")
		return
	}
	affect, typ := punishmentName(punishmentAffects, args[1]), punishmentName(punishmentTypes, args[2])
	if affect == "" || typ == "" {
		p.SendMessage("Incorrect value specified for affect/punishment type!")
		return
	}
	s.adminPunish(p, args[0], affect, typ, 0, "", false, nil)
}

// gmPunishShortcut is //ban_char, //ban_acc, //ban_hwid, //ban_chat, //jail
// <key> and their undoing (Mobius AdminPunishment): //punishment_add or
// //punishment_remove for ever with Mobius's reason. L2Go also takes the
// minutes of a chat ban or jail as a second argument.
func (s *Server) gmPunishShortcut(affect, typ string, add bool) GMCommand {
	reason := map[string]string{"BAN": "Banned by admin", "CHAT_BAN": "Chat banned by admin", "JAIL": "Jailed by admin"}[typ]
	return func(p *Player, args []string) {
		if len(args) == 0 {
			return
		}
		var end int64
		if add && typ != "BAN" && len(args) > 1 {
			if m, err := strconv.ParseInt(args[1], 10, 32); err == nil && m > 0 {
				end = s.Now().Add(time.Duration(m) * time.Minute).UnixMilli()
			}
		}
		s.adminPunish(p, args[0], affect, typ, end, reason, add, nil)
	}
}

// adminPunish starts (add) or stops a punishment of type typ on key, the
// way Mobius PunishmentManager and its handlers do, tells the GM and then
// runs after (nil shows the punishment's info page). end is when it ends,
// in Unix milliseconds, 0 for never.
func (s *Server) adminPunish(gm *Player, key, affect, typ string, end int64, reason string, add bool, after func()) {
	if after == nil {
		after = func() { s.adminPunishmentInfo(gm, key, affect) }
	}
	verb := "stopped"
	if add {
		verb = "applied"
	}
	message := fmt.Sprintf("Punishment %s have been %s to: %s %s!", typ, verb, affect, key)
	switch {
	case typ == "BAN" && add && end > 0:
		gm.SendMessage("Bans for a time are not available; use 0 minutes for a permanent ban.")
	case affect == "CHARACTER" && typ != "PARTY_BAN":
		s.adminWithCharacter(gm, key, nil, func(t punishTarget) bool {
			if _, has := t.has(typ, s.Now()); has == add {
				if add {
					gm.SendMessage("Target is already affected by that punishment.")
				} else {
					gm.SendMessage("Target is not affected by that punishment!")
				}
				return false
			}
			s.punishCharacter(t, typ, end, reason, gm.Name(), add)
			gm.SendMessage(message)
			if t.c == nil {
				after()
			}
			return t.c != nil // an offline character's page waits for the write
		}, after)
	case affect == "ACCOUNT" && typ == "BAN":
		s.adminBanAccount(gm, strings.ToLower(key), add, func() {
			gm.SendMessage(message)
			after()
		})
	default:
		gm.SendMessage(fmt.Sprintf("The punishment %s of %s is not available.", typ, affect))
	}
}

// punishCharacter starts or stops a punishment of a character, and of the
// player when it is online (Mobius BanHandler, ChatBanHandler and
// JailHandler).
func (s *Server) punishCharacter(t punishTarget, typ string, end int64, reason, by string, add bool) {
	switch typ {
	case "BAN":
		if !add {
			t.setAccess(storage.AccessLevelPlayer)
			return
		}
		t.setAccess(storage.AccessLevelBanned)
		if t.p != nil {
			s.adminKick(t.p, serverClosePacket)
		}
	case "CHAT_BAN":
		if !add {
			t.removeVar(s, chatBanVar)
			if t.p != nil {
				s.chatBanLift(t.p)
			}
			return
		}
		t.setVar(chatBanVar, strconv.FormatInt(end, 10))
		if t.p != nil {
			s.chatBanStart(t.p, end)
		}
	case "JAIL":
		if !add {
			t.removeVar(s, jailVar)
			if t.p != nil {
				s.jailRelease(t.p)
			}
			return
		}
		t.setVar(jailVar, strconv.FormatInt(end, 10))
		if t.p != nil {
			s.jailStart(t.p, end, reason, by, true)
		}
	}
}

// adminBanAccount bans (access level -1) or unbans (0) an account on the
// login server, off the loop; a banned account's player is kicked (Mobius
// BanHandler). done runs when the login server was told.
func (s *Server) adminBanAccount(gm *Player, account string, ban bool, done func()) {
	accounts := s.adminChar.accounts
	if accounts == nil {
		gm.SendMessage("Account bans are not available: the game server has no link to the login server.")
		return
	}
	level := int(storage.AccessLevelPlayer)
	if ban {
		level = int(storage.AccessLevelBanned)
	}
	go func() {
		err := accounts.ChangeAccessLevel(account, level)
		s.loop.Post(func() {
			if err != nil {
				s.log.Warn("game: cannot change an account's access level", "account", account, "level", level, "err", err)
				gm.SendMessage("The login server could not be told: " + err.Error())
				return
			}
			s.log.Info("game: account access level changed by a GM", append(gm.logArgs(), "target", account, "level", level)...)
			if t := s.byAccount[account]; ban && t != nil {
				s.adminKick(t, serverClosePacket)
			}
			done()
		})
	}()
}

// punishmentTimer runs fire when a punishment of p ends at end (Unix
// milliseconds; 0 never ends), replacing the timer in slot.
func (s *Server) punishmentTimer(p *Player, slot *world.TimerID, end int64, fire func()) {
	if *slot != 0 {
		s.loop.Cancel(*slot)
		*slot = 0
	}
	if end <= 0 {
		return
	}
	d := max(time.Duration(end-s.Now().UnixMilli())*time.Millisecond, 0)
	*slot = s.loop.After(d, func() {
		*slot = 0
		if p.state == stateInGame {
			fire()
		}
	})
}

// chatBanStart is ChatBanHandler.applyToPlayer: the player is told for how
// long, and its status icons show the chat block.
func (s *Server) chatBanStart(p *Player, end int64) {
	delay := (end-s.Now().UnixMilli())/1000 + 1
	if end > 0 && delay > 0 {
		hours, minutes := delay/3600, (delay%3600)/60
		msg := "You've been chat banned for "
		if hours > 0 {
			msg += strconv.FormatInt(hours, 10) + " hour" + adminPlural(hours, "'s.", ".")
			if minutes > 0 {
				msg += " and "
			}
		}
		if minutes > 0 || hours == 0 {
			msg += strconv.FormatInt(minutes, 10) + " minute" + adminPlural(minutes, "'s.", ".")
		}
		p.SendSystemMessage(sysMsgChatDisabled)
		p.SendMessage(msg)
	} else {
		p.SendMessage("You've been chat banned forever.")
	}
	p.Send(etcStatusPacket(p))
	s.punishmentTimer(p, &s.adminState(p).chatBanEnd, end, func() { s.punishmentExpired(p, chatBanVar, s.chatBanLift) })
}

// chatBanLift is ChatBanHandler.removeFromPlayer.
func (s *Server) chatBanLift(p *Player) {
	st := s.adminState(p)
	s.loop.Cancel(st.chatBanEnd)
	st.chatBanEnd = 0
	p.SendSystemMessage(sysMsgChatEnabled)
	p.Send(etcStatusPacket(p))
}

// punishmentExpired ends a timed punishment whose time is up.
func (s *Server) punishmentExpired(p *Player, name string, lift func(*Player)) {
	v, ok := p.rawVar(name)
	if _, active := punishmentEnd(v, ok, s.Now()); ok && !active {
		punishTarget{p: p}.removeVar(s, name)
		lift(p)
	}
}

func adminPlural(n int64, many, one string) string {
	if n > 1 {
		return many
	}
	return one
}

// jailStart is JailHandler.applyToPlayer: the player goes to jail 2 s
// later and sees jail_in.htm; told is whether it is told for how long.
func (s *Server) jailStart(p *Player, end int64, reason, by string, told bool) {
	st := s.adminState(p)
	s.jailTeleport(p, st, jailInLoc)
	content, ok := s.htmlFile(p, "jail_in.htm")
	if ok {
		content = strings.NewReplacer("%reason%", reason, "%punishedBy%", by).Replace(content)
	} else {
		content = "<html><body>You have been put in jail by an admin.</body></html>"
	}
	s.sendPage(p, newPage(0, content))
	if told {
		if d := end - s.Now().UnixMilli(); end > 0 && d > 0 {
			p.SendMessage("You've been jailed for " + formatDuration((d+999)/1000*1000) + "!")
		} else {
			p.SendMessage("You've been jailed forever.")
		}
	}
	s.punishmentTimer(p, &st.jailEnd, end, func() { s.punishmentExpired(p, jailVar, s.jailRelease) })
}

// jailRelease is JailHandler.removeFromPlayer: the player leaves the jail 2
// s later and sees jail_out.htm.
func (s *Server) jailRelease(p *Player) {
	st := s.adminState(p)
	s.loop.Cancel(st.jailEnd)
	st.jailEnd = 0
	s.jailTeleport(p, st, jailOutLoc)
	content, ok := s.htmlFile(p, "jail_out.htm")
	if !ok {
		content = "<html><body>You are free for now, respect server rules!</body></html>"
	}
	s.sendPage(p, newPage(0, content))
}

// jailTeleport teleports p to pos 2 s later (Mobius TeleportTask), in place
// of a teleport still pending.
func (s *Server) jailTeleport(p *Player, st *adminCharState, pos world.Position) {
	if st.jailMove != 0 {
		s.loop.Cancel(st.jailMove)
	}
	st.jailMove = s.loop.After(punishmentTeleportDelay, func() {
		st.jailMove = 0
		if p.state != stateInGame {
			return
		}
		if err := s.Teleport(p, pos); err != nil {
			s.log.Warn("game: jail teleport failed", append(p.logArgs(), "err", err)...)
		}
	})
}

// formatDuration is Mobius TimeUtil.formatDuration: "1 day, 2 hours, 3
// minutes, 4 seconds".
func formatDuration(ms int64) string {
	if ms < 1 {
		return "0 milliseconds"
	}
	var parts []string
	for _, u := range []struct {
		name string
		size int64
	}{
		{"day", 24 * 60 * 60 * 1000},
		{"hour", 60 * 60 * 1000},
		{"minute", 60 * 1000},
		{"second", 1000},
		{"millisecond", 1},
	} {
		if n := ms / u.size; n > 0 {
			parts = append(parts, strconv.FormatInt(n, 10)+" "+u.name+adminPlural(n, "s", ""))
		}
		ms %= u.size
	}
	return strings.Join(parts, ", ")
}

// setupPunishments adds the punishments' hooks: Mobius's login checks of
// the chat ban and jail (JailHandler.onPlayerLogin), the jail's walls
// (JailZone.onExit) and the end of the player's state.
func setupPunishments(s *Server) error {
	s.OnPlayerEnter(s.punishmentsEnter)
	s.OnPlayerMoved(func(p *Player, _, _ world.Position) { s.jailWalls(p) })
	s.OnPlayerLeave(func(p *Player) {
		if st := s.adminChar.players[p]; st != nil {
			for _, id := range []world.TimerID{st.jailMove, st.jailEnd, st.chatBanEnd} {
				if id != 0 {
					s.loop.Cancel(id)
				}
			}
			delete(s.adminChar.players, p)
		}
	})
	return nil
}

// punishmentsEnter checks a player's punishments as it enters the world:
// those whose time is up go, a timed one gets its end, a jailed player
// outside the jail goes in, and a free one inside comes out.
func (s *Server) punishmentsEnter(p *Player) {
	st := s.adminState(p)
	st.entered = s.Now()
	now := s.Now()
	if v, ok := p.rawVar(chatBanVar); ok {
		if end, active := punishmentEnd(v, ok, now); !active {
			punishTarget{p: p}.removeVar(s, chatBanVar)
		} else {
			s.punishmentTimer(p, &st.chatBanEnd, end, func() { s.punishmentExpired(p, chatBanVar, s.chatBanLift) })
		}
	}
	v, ok := p.rawVar(jailVar)
	end, jailed := punishmentEnd(v, ok, now)
	if ok && !jailed {
		punishTarget{p: p}.removeVar(s, jailVar)
	}
	inJail := p.Zones.Inside(ZoneJail)
	st.inJail = inJail
	switch {
	case jailed && !inJail:
		s.jailStart(p, end, "", "", false)
	case jailed:
		if d := end - now.UnixMilli(); end > 0 && d > 0 {
			p.SendMessage("Jail time remaining: " + formatDuration((d+999)/1000*1000) + ".")
		} else {
			p.SendMessage("You are jailed forever.")
		}
		s.punishmentTimer(p, &st.jailEnd, end, func() { s.punishmentExpired(p, jailVar, s.jailRelease) })
	case inJail && !p.IsGM():
		s.jailRelease(p)
	}
}

// jailWalls is JailZone.onExit: a jailed player that left the jail goes
// back 2 s later.
func (s *Server) jailWalls(p *Player) {
	if p.state != stateInGame {
		return
	}
	st := s.adminState(p)
	wasIn := st.inJail
	st.inJail = p.Zones.Inside(ZoneJail)
	if !wasIn || st.inJail || !p.jailed() || st.jailMove != 0 {
		return
	}
	p.SendMessage("You cannot cheat your way out of here. You must wait until your jail time is over.")
	s.jailTeleport(p, st, jailInLoc)
}
