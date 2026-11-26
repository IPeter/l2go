package game

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The admin panel's world and server tools, with Mobius's messages and
// pages:
//   - AdminShop's admin_buy, the GM shop (//gmshop, its menu page, is in
//     admin.go; the gmstore pages below html/admin/gmstore open with //html
//     and offer "admin_buy <listId>" buttons);
//   - AdminAnnouncements: //announce, //announce_crit, //announce_screen and
//     the announcement list (//announces). The announcements are kept in
//     memory, not in a table, so they last until the server stops. Mobius
//     shows their pages on the community board, which the server does not
//     have yet; here they are dialog pages of their own;
//   - AdminSearch (//search), AdminTeleport's //recall_npc, AdminSpawn's
//     //list_spawns and //list_positions, AdminServerInfo (//serverinfo,
//     with the Go runtime's numbers in place of the Java VM's), AdminReload
//     (//reload_menu, //reload html and //reload quest; the other data do
//     not reload yet) and AdminCreateItem's //give_item_to_all and
//     //create_coin.
func init() { registerFeature("admin_world", setupAdminWorld) }

const (
	// adminSearchPageLimit is how many items a page of //search lists, and
	// adminSearchMaxPages how many page links it shows at most (Mobius
	// AdminSearch).
	adminSearchPageLimit = 15
	adminSearchMaxPages  = 100
	// adminScreenAnnounceTime is how long //announce_screen's text stays,
	// in milliseconds (Mobius broadcastToAllOnlinePlayersOnScreen).
	adminScreenAnnounceTime = 10000
	// adminAnnouncesPageLimit is how many announcements a page of the list
	// shows (Mobius AdminAnnouncements).
	adminAnnouncesPageLimit = 10
	// adminAnnounceMinDelay is the shortest delay of an auto announcement.
	adminAnnounceMinDelay = 10
	// adminReloadPageSize is how many links a page of //reload_menu shows
	// (Mobius AdminReload ITEMS_PER_PAGE).
	adminReloadPageSize = 20
	// adminReloadUsage and adminReloadWarning are AdminReload's usage and
	// the warning after each reload.
	adminReloadUsage   = "Usage: //reload <type> [parameters]"
	adminReloadWarning = "WARNING: There are several known issues regarding this feature. Reloading server data during runtime is STRONGLY NOT RECOMMENDED for live servers, just for developing environments."
	// adminAnnounceSyntax is the answer to a bad //announces add or edit.
	adminAnnounceSyntax = "Syntax: //announces add <type> <delay> <repeat> <text>"
)

// The announcement types (Mobius AnnouncementType), by name.
const (
	announceNormal       = "NORMAL"
	announceCritical     = "CRITICAL"
	announceEvent        = "EVENT"
	announceAutoNormal   = "AUTO_NORMAL"
	announceAutoCritical = "AUTO_CRITICAL"
)

// announceTypes are the announcement types in Mobius's order.
var announceTypes = []string{announceNormal, announceCritical, announceEvent, announceAutoNormal, announceAutoCritical}

// adminWorldFeature is the state of the world tools.
type adminWorldFeature struct {
	// started is when the server was made, for //serverinfo's uptime.
	started time.Time
	// announcements are the announcements in ID order, lastID the ID of the
	// last one added.
	announcements []*announcement
	lastID        int
}

// announcement is an announcement of //announces (Mobius Announcement and
// AutoAnnouncement).
type announcement struct {
	id                   int
	typ, content, author string
	// isAuto is set for an auto announcement (Mobius AutoAnnouncement): it
	// comes after initial, then after each delay, repeat times (-1 for
	// ever); left is how many are still to come (Mobius _currentState) and
	// timer the next one, or 0. initialSec and delaySec are the delays as
	// given, for the pages.
	isAuto               bool
	initial, delay       time.Duration
	initialSec, delaySec int
	repeat, left         int
	timer                world.TimerID
}

// adminCoins are the coins of //create_coin (Mobius getCoinId).
var adminCoins = map[string]int{
	"adena":         57,
	"ancientadena":  5575,
	"festivaladena": 6673,
	"blueeva":       4355,
	"goldeinhasad":  4356,
	"silvershilen":  4357,
	"bloodypaagrio": 4358,
}

// adminReloadMenu are the links of //reload_menu (Mobius RELOAD_COMMANDS).
var adminReloadMenu = [][2]string{
	{"Access", "admin_reload access"},
	{"Armor Sets", "admin_reload sets"},
	{"BuyList", "admin_reload buylist"},
	{"Category", "admin_reload category"},
	{"Class List", "admin_reload classlist"},
	{"Config", "admin_reload config"},
	{"Crests", "admin_reload crest"},
	{"Cubic", "admin_reload cubic"},
	{"Cursed Weapons", "admin_reload cw"},
	{"Doors", "admin_reload door"},
	{"Dynamic Exp Rate", "admin_reload dynamicexprate"},
	{"Effects", "admin_reload effect"},
	{"Ench Item HP Bonus", "admin_reload enchantitemhpbonus"},
	{"Ench Skill Tree", "admin_reload enchantskilltree"},
	{"Enchant", "admin_reload enchant"},
	{"Experience", "admin_reload experience"},
	{"Experience Loss", "admin_reload experienceloss"},
	{"Fake Player Chat", "admin_reload fakeplayerchat"},
	{"Fence", "admin_reload fence"},
	{"Fish", "admin_reload fish"},
	{"Fishing Monsters", "admin_reload fishingmonsters"},
	{"Fishing Rods", "admin_reload fishingrods"},
	{"Handler", "admin_reload handler"},
	{"Henna", "admin_reload henna"},
	{"Hit Condition Bonus", "admin_reload hitconditionbonus"},
	{"HTM (path/x.htm)", "admin_reload htm $value"},
	{"HTML (path/x.htm)", "admin_reload html $value"},
	{"Initial Equipment", "admin_reload initialequipment"},
	{"Initial Shortcut", "admin_reload initialshortcut"},
	{"Instance", "admin_reload instance"},
	{"Item", "admin_reload item"},
	{"Karma Loss", "admin_reload karmaloss"},
	{"LevelUp Crystal", "admin_reload levelupcrystal"},
	{"Localisations", "admin_reload localisations"},
	{"Map Region", "admin_reload mapregion"},
	{"Multisell", "admin_reload multisell"},
	{"NPC", "admin_reload npc $value"},
	{"Options", "admin_reload options"},
	{"Pet Data", "admin_reload petdata"},
	{"Pet Skill", "admin_reload petskill"},
	{"Player Template", "admin_reload playertemplate"},
	{"Quest", "admin_reload quest $value"},
	{"Recipe", "admin_reload recipe"},
	{"Siege Schedule", "admin_reload siegeschedule"},
	{"Skill", "admin_reload skill"},
	{"Skill Learn", "admin_reload skilllearn"},
	{"Skill Tree", "admin_reload skilltree"},
	{"Spawn", "admin_reload spawn"},
	{"Static Objects", "admin_reload staticobject"},
	{"Teleport", "admin_reload teleport"},
	{"Walker", "admin_reload walker"},
	{"Zone", "admin_reload zone"},
}

// setupAdminWorld registers the GM commands of the world and server tools.
func setupAdminWorld(s *Server) error {
	s.adminWorld.started = s.Clock().Now()
	s.OnPlayerEnter(s.showAnnouncements)
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("buy", admin, s.gmBuy),
		s.RegisterGMCommand("announce", admin, s.gmAnnounce(false, false)),
		s.RegisterGMCommand("announce_crit", admin, s.gmAnnounce(true, false)),
		s.RegisterGMCommand("announce_screen", admin, s.gmAnnounce(false, true)),
		s.RegisterGMCommand("announces", admin, s.gmAnnounces),
		s.RegisterGMCommand("search", admin, s.gmSearch),
		s.RegisterGMCommand("recall_npc", admin, s.gmRecallNpc),
		s.RegisterGMCommand("list_spawns", admin, s.gmListSpawns(false)),
		s.RegisterGMCommand("list_positions", admin, s.gmListSpawns(true)),
		s.RegisterGMCommand("serverinfo", admin, s.gmServerInfo),
		s.RegisterGMCommand("reload", admin, s.gmReload),
		s.RegisterGMCommand("reload_menu", admin, func(p *Player, _ []string) { s.showReloadMenu(p, 0) }),
		s.RegisterGMCommand("reload_menu_page", admin, s.gmReloadMenuPage),
		s.RegisterGMCommand("give_item_to_all", admin, s.gmGiveItemToAll),
		s.RegisterGMCommand("create_coin", admin, s.gmCreateCoin),
	)
}

// gmBuy is //buy <listId> (Mobius AdminShop.handleBuyRequest): the buy
// window of that buylist opens with the GM's adena and no tax, bound to no
// merchant, then ActionFailed. Buying goes through RequestBuyItem, which
// lets GMs buy from any list without a merchant, untaxed, and take the
// GM-only free products (OnlyGMItemsFree). Mobius only logs a list that
// does not exist; the GM is also told here, since some of the gmstore
// buttons name lists that Mobius does not have either.
func (s *Server) gmBuy(p *Player, args []string) {
	if len(args) == 0 {
		p.SendMessage("Please specify buylist.")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		s.log.Warn("game: admin buylist failed", append(p.logArgs(), "list", args[0])...)
		id = -1
	}
	if l := s.buyList(id); l != nil {
		p.Send(serverpackets.NewBuyListPacket(s.buyListPacket(p, l, 0)))
	} else {
		s.log.Warn("game: no buylist with that id", append(p.logArgs(), "list", args[0])...)
		p.SendMessage("Buylist " + args[0] + " does not exist.")
	}
	p.SendActionFailed()
}

// --- Announcements ---

// sendToOnline sends body to every player in the game (Mobius
// World.broadcastToAllOnlinePlayers).
func (s *Server) sendToOnline(body []byte) {
	for v := range s.Players() {
		v.Send(body)
	}
}

// announceToAll sends text to every player as an announcement, a critical
// one in its own color (Mobius broadcastToAllOnlinePlayers(text,
// isCritical)).
func (s *Server) announceToAll(text string, critical bool) {
	typ := serverpackets.ChatAnnouncement
	if critical {
		typ = serverpackets.ChatCriticalAnnounce
	}
	s.sendToOnline(serverpackets.NewCreatureSayPacket(serverpackets.CreatureSay{Type: typ, Text: text}))
}

// gmAnnounce is //announce <text>, //announce_crit <text> and
// //announce_screen <text> (Mobius AdminAnnouncements): every player gets
// the text as an announcement, a critical one, or on the screen for 10
// seconds; then the GM menu shows. Mobius's GMShowAnnouncerName is off, so
// the GM's name is not added.
func (s *Server) gmAnnounce(critical, screen bool) GMCommand {
	return func(p *Player, args []string) {
		if len(args) == 0 {
			gmSysMessage(p, "Syntax: //announce <text to announce here>")
			return
		}
		text := strings.Join(args, " ")
		if screen {
			s.sendToOnline(serverpackets.NewExShowScreenMessagePacket(serverpackets.ScreenText(text, adminScreenAnnounceTime)))
		} else {
			s.announceToAll(text, critical)
		}
		s.ShowAdminPage(p, "gm_menu.htm")
	}
}

// showAnnouncements sends an entering player the normal, critical and event
// announcements, with its own name as the sender (Mobius
// AnnouncementsTable.showAnnouncements).
func (s *Server) showAnnouncements(p *Player) {
	for _, typ := range []string{announceNormal, announceCritical, announceEvent} {
		chat := serverpackets.ChatAnnouncement
		if typ == announceCritical {
			chat = serverpackets.ChatCriticalAnnounce
		}
		for _, a := range s.adminWorld.announcements {
			if a.typ == typ {
				p.Send(serverpackets.NewCreatureSayPacket(serverpackets.CreatureSay{Type: chat, SenderName: p.Name(), Text: a.content}))
			}
		}
	}
}

// findAnnounceType is AnnouncementType.findByName: the type with the name,
// ignoring case, or NORMAL.
func (s *Server) findAnnounceType(name string) string {
	for _, t := range announceTypes {
		if strings.EqualFold(t, name) {
			return t
		}
	}
	s.log.Warn("game: unknown announcement type", "type", name)
	return announceNormal
}

// autoAnnounceType reports whether typ is an auto announcement type.
func autoAnnounceType(typ string) bool {
	return typ == announceAutoNormal || typ == announceAutoCritical
}

// announcementByID returns the announcement with the ID, or nil.
func (s *Server) announcementByID(id int) *announcement {
	for _, a := range s.adminWorld.announcements {
		if a.id == id {
			return a
		}
	}
	return nil
}

// restartAnnouncement starts an auto announcement again: it comes after its
// initial delay, then after each delay, repeat times (Mobius restartMe).
func (s *Server) restartAnnouncement(a *announcement) {
	s.stopAnnouncement(a)
	a.left = a.repeat
	a.timer = s.loop.After(a.initial, func() { s.runAnnouncement(a) })
}

// stopAnnouncement stops the timer of an auto announcement.
func (s *Server) stopAnnouncement(a *announcement) {
	if a.timer != 0 {
		s.loop.Cancel(a.timer)
		a.timer = 0
	}
}

// runAnnouncement is AutoAnnouncement.run: each line of the text goes to
// every player, then the next one is due after the delay while repeats are
// left.
func (s *Server) runAnnouncement(a *announcement) {
	a.timer = 0
	if a.left != -1 && a.left <= 0 {
		return
	}
	for line := range strings.SplitSeq(a.content, "\n") {
		s.announceToAll(strings.TrimSuffix(line, "\r"), a.typ == announceAutoCritical)
	}
	if a.left != -1 {
		a.left--
	}
	if a.left != 0 {
		a.timer = s.loop.After(a.delay, func() { s.runAnnouncement(a) })
	}
}

// announceNumber reads the number args[i] of //announces add and edit:
// digits only (StringUtil.isNumeric) that fit an int.
func announceNumber(args []string, i int) (int, bool) {
	if i >= len(args) || !adminNumeric(args[i]) {
		return 0, false
	}
	return javaParseInt(args[i])
}

// gmAnnounces is //announces <list|add|edit|remove|restart|show> (Mobius
// AdminAnnouncements): the announcement list and its pages.
func (s *Server) gmAnnounces(p *Player, args []string) {
	if len(args) == 0 {
		return
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "add":
		s.announcesAdd(p, rest)
	case "edit":
		s.announcesEdit(p, rest)
	case "remove":
		if len(rest) == 0 || !adminNumeric(rest[0]) {
			gmSysMessage(p, "Syntax: //announces remove <announcement id>")
			return
		}
		id, _ := javaParseInt(rest[0])
		if a := s.announcementByID(id); a != nil {
			s.stopAnnouncement(a)
			s.adminWorld.announcements = slices.DeleteFunc(s.adminWorld.announcements, func(o *announcement) bool { return o == a })
			gmSysMessage(p, "Announcement has been successfully removed!")
		} else {
			gmSysMessage(p, "Announcement does not exist!")
		}
		s.announcesList(p, 0)
	case "restart":
		s.announcesRestart(p, rest)
	case "show":
		if len(rest) == 0 || !adminNumeric(rest[0]) {
			gmSysMessage(p, "Syntax: //announces show <announcement id>")
			return
		}
		id, _ := javaParseInt(rest[0])
		if a := s.announcementByID(id); a != nil {
			s.sendPage(p, newPage(0, announceShowPage(a)))
			return
		}
		gmSysMessage(p, "Announcement does not exist!")
		s.announcesList(p, 0)
	case "list":
		page := 0
		if len(rest) > 0 {
			if v := strings.ReplaceAll(rest[0], "page=", ""); adminNumeric(v) {
				page, _ = javaParseInt(v)
			}
		}
		s.announcesList(p, page)
	}
}

// announcesAdd is //announces add [<type> <initial> <delay> <repeat>
// <text>]: without arguments the form, else a new announcement; delays are
// in seconds, a repeat of 0 is for ever.
func (s *Server) announcesAdd(p *Player, args []string) {
	if len(args) == 0 {
		s.sendPage(p, newPage(0, announceAddPage))
		return
	}
	typ := s.findAnnounceType(args[0])
	initial, ok := announceNumber(args, 1)
	if !ok {
		gmSysMessage(p, adminAnnounceSyntax)
		return
	}
	delay, ok := announceNumber(args, 2)
	if !ok {
		gmSysMessage(p, adminAnnounceSyntax)
		return
	}
	if delay < adminAnnounceMinDelay && autoAnnounceType(typ) {
		gmSysMessage(p, "Delay cannot be less then 10 seconds!")
		return
	}
	repeat, ok := announceNumber(args, 3)
	if !ok || len(args) < 5 {
		gmSysMessage(p, adminAnnounceSyntax)
		return
	}
	if repeat == 0 {
		repeat = -1
	}
	s.adminWorld.lastID++
	a := &announcement{id: s.adminWorld.lastID, typ: typ, content: strings.Join(args[4:], " "), author: p.Name(), repeat: -1}
	if autoAnnounceType(typ) {
		a.isAuto = true
		a.setTimes(initial, delay, repeat)
		s.restartAnnouncement(a)
	}
	s.adminWorld.announcements = append(s.adminWorld.announcements, a)
	gmSysMessage(p, "Announcement has been successfully added!")
	s.announcesList(p, 0)
}

// setTimes sets the delays, in seconds, and the repeat of an auto
// announcement.
func (a *announcement) setTimes(initial, delay, repeat int) {
	a.initialSec, a.delaySec, a.repeat = initial, delay, repeat
	a.initial = time.Duration(initial) * time.Second
	a.delay = time.Duration(delay) * time.Second
}

// announcesEdit is //announces edit <id> [<type> <initial> <delay> <repeat>
// [text]]: without more arguments the form, else the announcement changes
// (its text only when one is given). A normal announcement stays normal or
// critical, an auto one auto.
func (s *Server) announcesEdit(p *Player, args []string) {
	if len(args) == 0 || !adminNumeric(args[0]) {
		gmSysMessage(p, "Syntax: //announces edit <id>")
		return
	}
	id, _ := javaParseInt(args[0])
	a := s.announcementByID(id)
	if a == nil {
		gmSysMessage(p, "Announcement does not exist!")
		return
	}
	if len(args) == 1 {
		s.sendPage(p, newPage(0, announceEditPage(a)))
		return
	}
	typ := s.findAnnounceType(args[1])
	switch a.typ {
	case announceAutoNormal, announceAutoCritical:
		if !autoAnnounceType(typ) {
			gmSysMessage(p, "Announce type can be changed only to AUTO_NORMAL or AUTO_CRITICAL!")
			return
		}
	case announceNormal, announceCritical:
		if typ != announceNormal && typ != announceCritical {
			gmSysMessage(p, "Announce type can be changed only to NORMAL or CRITICAL!")
			return
		}
	}
	initial, ok := announceNumber(args, 2)
	if !ok {
		gmSysMessage(p, adminAnnounceSyntax)
		return
	}
	delay, ok := announceNumber(args, 3)
	if !ok {
		gmSysMessage(p, adminAnnounceSyntax)
		return
	}
	if delay < adminAnnounceMinDelay && autoAnnounceType(typ) {
		gmSysMessage(p, "Delay cannot be less then 10 seconds!")
		return
	}
	repeat, ok := announceNumber(args, 4)
	if !ok {
		gmSysMessage(p, adminAnnounceSyntax)
		return
	}
	if repeat == 0 {
		repeat = -1
	}
	if content := strings.Join(args[5:], " "); content != "" {
		a.content = content
	}
	a.typ, a.author = typ, p.Name()
	if a.isAuto {
		a.setTimes(initial, delay, repeat)
	}
	gmSysMessage(p, "Announcement has been successfully edited!")
	s.announcesList(p, 0)
}

// announcesRestart is //announces restart [id]: every auto announcement, or
// the one, starts again.
func (s *Server) announcesRestart(p *Player, args []string) {
	if len(args) == 0 {
		for _, a := range s.adminWorld.announcements {
			if a.isAuto {
				s.restartAnnouncement(a)
			}
		}
		gmSysMessage(p, "Auto announcements has been successfully restarted!")
		return
	}
	if !adminNumeric(args[0]) {
		gmSysMessage(p, "Syntax: //announces show <announcement id>")
		return
	}
	id, _ := javaParseInt(args[0])
	switch a := s.announcementByID(id); {
	case a == nil:
		gmSysMessage(p, "Announcement does not exist!")
	case a.isAuto:
		s.restartAnnouncement(a)
		gmSysMessage(p, "Auto announcement has been successfully restarted")
	default:
		gmSysMessage(p, "This option has effect only on auto announcements!")
	}
}

// adminButton is a small button of the announcement pages.
func adminButton(value, action string) string {
	return `<button value="` + value + `" action="` + action + `" width=60 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2">`
}

// announceHeader is the top of the announcement pages: Main, the title and
// Back to the server menu.
const announceHeader = `<html><title>Announcements</title><body><table width=270><tr>` +
	`<td width=45><button value="Main" action="bypass admin_admin" width=45 height=15 back="sek.cbui94" fore="sek.cbui92"></td>` +
	`<td width=180><center>Announcements</center></td>` +
	`<td width=45><button value="Back" action="bypass admin_admin4" width=45 height=15 back="sek.cbui94" fore="sek.cbui92"></td>` +
	`</tr></table><br>`

// announceAddPage is the form of //announces add (Mobius
// announces-add.htm, as a dialog page).
const announceAddPage = announceHeader + `<center>Add announcement:</center><br>` +
	`<table width=270>` +
	`<tr><td width=110>Type</td><td width=160><combobox width=150 height=17 var="typeBox" list="NORMAL;CRITICAL;AUTO_NORMAL;AUTO_CRITICAL"></td></tr>` +
	`<tr><td>Initial delay (sec)</td><td><edit var="autoann_initdelay" width=150 height=12 type=number></td></tr>` +
	`<tr><td>Delay (sec)</td><td><edit var="autoann_delay" width=150 height=12 type=number></td></tr>` +
	`<tr><td>Repeat (0: for ever)</td><td><edit var="autoann_repeat" width=150 height=12 type=number></td></tr>` +
	`</table>Text:<br1><multiedit var="autoann_memo" width=260 height=60><br>` +
	`<font color="LEVEL">Fill every number (0 for a normal announcement).</font><br>` +
	`<center><table><tr>` +
	`<td>` + `<button value="Add" action="bypass admin_announces add $typeBox $autoann_initdelay $autoann_delay $autoann_repeat $autoann_memo" width=60 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2">` + `</td>` +
	`<td>` + `<button value="Back" action="bypass admin_announces list" width=60 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2">` + `</td>` +
	`</tr></table></center></body></html>`

// announceTimes returns the initial delay, delay and repeat the pages show.
func announceTimes(a *announcement) (initial, delay, repeat string) {
	if !a.isAuto {
		return "0", "0", "0"
	}
	return strconv.Itoa(a.initialSec), strconv.Itoa(a.delaySec), strconv.Itoa(a.repeat)
}

// announceShowPage is the page of //announces show (Mobius
// announces-show.htm, as a dialog page).
func announceShowPage(a *announcement) string {
	initial, delay, repeat := announceTimes(a)
	id := strconv.Itoa(a.id)
	return announceHeader + `<center>Announcement: ` + id + `</center><br><table width=270>` +
		`<tr><td width=90>Type</td><td width=180>` + a.typ + `</td></tr>` +
		`<tr><td>Initial Delay</td><td>` + initial + `</td></tr>` +
		`<tr><td>Delay</td><td>` + delay + `</td></tr>` +
		`<tr><td>Repeat:</td><td>` + repeat + `</td></tr>` +
		`<tr><td>Author:</td><td>` + a.author + `</td></tr>` +
		`<tr><td>Text:</td><td>` + a.content + `</td></tr></table><br>` +
		`<center><table><tr>` +
		`<td>` + adminButton("Remove", "bypass admin_announces remove "+id) + `</td>` +
		`<td>` + adminButton("Restart", "bypass admin_announces restart "+id) + `</td>` +
		`<td>` + adminButton("Back", "bypass admin_announces list") + `</td>` +
		`</tr></table></center></body></html>`
}

// announceEditPage is the form of //announces edit <id> (Mobius
// announces-edit.htm, as a dialog page): the new values, with the current
// ones beside.
func announceEditPage(a *announcement) string {
	initial, delay, repeat := announceTimes(a)
	id := strconv.Itoa(a.id)
	return announceHeader + `<center>Edit announcement: ` + id + `</center><br><table width=270>` +
		`<tr><td width=80>Type</td><td width=110><combobox width=105 height=17 var="typeBox" list="NORMAL;CRITICAL;AUTO_NORMAL;AUTO_CRITICAL"></td><td width=80>` + a.typ + `</td></tr>` +
		`<tr><td>Initial delay</td><td><edit var="autoann_initdelay" width=105 height=12 type=number></td><td>` + initial + `</td></tr>` +
		`<tr><td>Delay</td><td><edit var="autoann_delay" width=105 height=12 type=number></td><td>` + delay + `</td></tr>` +
		`<tr><td>Repeat</td><td><edit var="autoann_repeat" width=105 height=12 type=number></td><td>` + repeat + `</td></tr>` +
		`</table>Text: ` + a.content + `<br1><multiedit var="autoann_memo" width=260 height=60><br>` +
		`<center><table><tr>` +
		`<td>` + adminButton("Edit", "bypass admin_announces edit "+id+" $typeBox $autoann_initdelay $autoann_delay $autoann_repeat $autoann_memo") + `</td>` +
		`<td>` + adminButton("Back", "bypass admin_announces list") + `</td>` +
		`</tr></table></center></body></html>`
}

// announcesList shows a page of the announcement list (Mobius
// AdminAnnouncements list): 10 to a page, each with its buttons.
func (s *Server) announcesList(p *Player, page int) {
	list := s.adminWorld.announcements
	pages := (len(list) + adminAnnouncesPageLimit - 1) / adminAnnouncesPageLimit
	page = min(max(page, 0), max(pages-1, 0))
	var b strings.Builder
	b.WriteString(announceHeader)
	b.WriteString(`<table width=270><tr><td width=30>Id</td><td width=110>Type</td><td width=130>Author</td></tr></table>`)
	b.WriteString(`<img src="L2UI.SquareGray" width=270 height=1><br>`)
	start := page * adminAnnouncesPageLimit
	for _, a := range list[start:min(start+adminAnnouncesPageLimit, len(list))] {
		id := strconv.Itoa(a.id)
		b.WriteString(`<table width=270 bgcolor=000000><tr><td width=30>` + id + `</td><td width=110>` + a.typ + `</td><td width=130>` + a.author + `</td></tr></table>`)
		b.WriteString(`<table width=270><tr><td>` + adminButton("Show", "bypass admin_announces show "+id) + `</td>`)
		if a.typ != announceEvent {
			b.WriteString(`<td>` + adminButton("Edit", "bypass admin_announces edit "+id) + `</td>`)
		}
		b.WriteString(`<td>` + adminButton("Remove", "bypass admin_announces remove "+id) + `</td>`)
		if autoAnnounceType(a.typ) {
			b.WriteString(`<td>` + adminButton("Restart", "bypass admin_announces restart "+id) + `</td>`)
		}
		b.WriteString(`</tr></table>`)
	}
	if len(list) == 0 {
		b.WriteString(`<center>There are no announcements.</center>`)
	}
	if pages > 1 {
		b.WriteString(`<table width=270 cellspacing=0><tr>` + adminPager("bypass admin_announces list", page, pages-1) + `</tr></table>`)
	}
	b.WriteString(`<br><center><table><tr><td>` + adminButton("Add", "bypass admin_announces add") + `</td><td>` +
		adminButton("Restart", "bypass admin_announces restart") + `</td></tr></table></center></body></html>`)
	s.sendPage(p, newPage(0, b.String()))
}

// --- Item tools ---

// gmSearch is //search [word [page]] (Mobius AdminSearch): the item search
// page, and the items whose name has the word, 15 to a page, each with a
// button that creates it. Mobius writes the result rows with stray
// backslashes before some quotes; here they are left out.
func (s *Server) gmSearch(p *Player, args []string) {
	if len(args) == 0 {
		s.sendPage(p, s.pageFile(p, 0, "admin/search.htm").replace("%items%", "").replace("%pages%", ""))
		return
	}
	text, page := args[0], 1
	if len(args) > 1 {
		if v, ok := javaParseInt(args[1]); ok {
			page = v
		}
	}
	lower := strings.ToLower(text)
	var items []*data.Item
	for i := range s.data.Items {
		if it := &s.data.Items[i]; strings.Contains(strings.ToLower(it.Name), lower) {
			items = append(items, it)
		}
	}
	slices.SortFunc(items, func(a, b *data.Item) int { return a.ID - b.ID })
	pg := s.pageFile(p, 0, "admin/search.htm")
	if len(items) == 0 {
		s.sendPage(p, pg.replace("%items%", "<tr><td>No items found with word "+text+".</td></tr>").replace("%pages%", ""))
		return
	}
	pages := (len(items) + adminSearchPageLimit - 1) / adminSearchPageLimit
	page = min(max(page, 1), pages)
	var b strings.Builder
	for _, it := range items[(page-1)*adminSearchPageLimit : min(page*adminSearchPageLimit, len(items))] {
		stack := `<font name="hs8" color=ff0000 size=-1>ST</font>`
		if it.Stackable {
			stack = `<font name="hs8" color=008000 size=-1>ST</font>`
		}
		fmt.Fprintf(&b, `<tr><td align=center><button value="Add" action="bypass admin_create_item %d" width=32 height=15 back="sek.cbui94" fore="sek.cbui94"></td>`+
			`<td><img src="%s" width=32 height=32></td><td>%d</td><td>%s</td><td align=center>%s</td><td>%s</td></tr>`,
			it.ID, it.Icon, it.DisplayID, stack, adminItemGrade(it.Grade), adminFontedWord(text, it.Name))
	}
	pg.replace("%items%", b.String())
	b.Reset()
	for nr := 1; nr <= min(adminSearchMaxPages, pages); nr++ {
		if nr == page {
			b.WriteString(strconv.Itoa(nr) + "&nbsp;")
		} else {
			fmt.Fprintf(&b, `<a action="bypass -h admin_search %s %d">%d</a>&nbsp;`, text, nr, nr)
		}
	}
	s.sendPage(p, pg.replace("%pages%", b.String()))
}

// adminItemGrade is AdminSearch.getItemGrade: the grade as a small font.
func adminItemGrade(g data.CrystalType) string {
	name := map[data.CrystalType]string{
		data.CrystalNone: "NG", data.CrystalD: "D", data.CrystalC: "C",
		data.CrystalB: "B", data.CrystalA: "A", data.CrystalS: "S",
	}[g]
	if name == "" {
		return strconv.Itoa(int(g))
	}
	return `<font name="hs8" color=ae9977 size=-1>` + name + `</font>`
}

// adminFontedWord is AdminSearch.getFontedWord: the searched word of the
// name in the LEVEL color.
func adminFontedWord(text, name string) string {
	i := strings.Index(strings.ToLower(name), strings.ToLower(text))
	if i < 0 || i+len(text) > len(name) {
		return name
	}
	return name[:i] + `<font color="LEVEL">` + name[i:i+len(text)] + `</font>` + name[i+len(text):]
}

// gmGiveItemToAll is //give_item_to_all <id> [count] (Mobius
// AdminCreateItem): every other player in the game with a client gets the
// items and is told.
func (s *Server) gmGiveItemToAll(p *Player, args []string) {
	id, count, ok := 0, 0, true
	switch len(args) {
	case 2:
		if id, ok = javaParseInt(args[0]); ok {
			count, ok = javaParseInt(args[1])
		}
	case 1:
		id, ok = javaParseInt(args[0])
		count = 1
	}
	if !ok {
		gmSysMessage(p, "Specify a valid number.")
		return
	}
	t, found := s.data.ItemTemplate(id)
	if !found {
		gmSysMessage(p, "This item doesn't exist.")
		return
	}
	if count > 10 && !t.Stackable {
		gmSysMessage(p, "This item does not stack - Creation aborted.")
		return
	}
	if count <= 0 {
		gmSysMessage(p, "Specify a valid number.")
		return
	}
	var players []*Player
	for v := range s.Players() {
		if v != p && v.InGame() && v.session != nil {
			players = append(players, v)
		}
	}
	for _, v := range players {
		if _, err := v.inv.Create(int32(id), int64(count)); err != nil { //nolint:gosec // an int32 from parseInt
			s.log.Warn("game: //give_item_to_all failed", append(v.logArgs(), "item", id, "count", count, "err", err)...)
			continue
		}
		s.itemsChanged(v, true, nil)
		v.SendMessage(fmt.Sprintf("Admin spawned %d %s in your inventory.", count, t.Name))
	}
	p.SendMessage(fmt.Sprintf("%d players rewarded with %s", len(players), t.Name))
}

// gmCreateCoin is //create_coin <name> [count] (Mobius AdminCreateItem):
// the coin goes into the GM's inventory, then the item creation page
// shows. An unknown name with a count does nothing.
func (s *Server) gmCreateCoin(p *Player, args []string) {
	switch len(args) {
	case 2:
		if id := adminCoins[strings.ToLower(args[0])]; id > 0 {
			n, ok := javaParseInt(args[1])
			if !ok {
				gmSysMessage(p, "Specify a valid number.")
				break
			}
			s.gmGiveItem(p, p, int32(id), int32(n), -1, false) //nolint:gosec // an int32 from parseInt
		}
	case 1:
		s.gmGiveItem(p, p, int32(adminCoins[strings.ToLower(args[0])]), 1, -1, false) //nolint:gosec // a coin ID
	}
	s.ShowAdminPage(p, "itemcreation.htm")
}

// --- NPC tools ---

// gmRecallNpc is //recall_npc (Mobius AdminTeleport.recallNPC): the GM's
// target NPC moves to the GM for good: its spawn is deleted and a new one,
// with the same respawn delay, is made where the GM stands. A raid boss's
// spawn moves the same way in the raid spawner and is stored
// (moveRaidSpawn), silently as in Mobius; grand bosses and minions cannot be
// moved.
func (s *Server) gmRecallNpc(p *Player, _ []string) {
	n, ok := p.Target.(*Npc)
	switch {
	case ok && n != nil && n.Template.Type == "RaidBoss":
		if n.spawn == nil || n.boss.raid == nil {
			gmSysMessage(p, "Incorrect raid spawn.")
			s.log.Warn("game: //recall_npc of a raid boss without a spawn", append(p.logArgs(), "npc", n.Template.ID)...)
			return
		}
		if s.moveRaidSpawn(n, p.Position()) == nil {
			p.SendSystemMessage(npcSysMsgTargetNotFound)
			return
		}
		s.log.Info("GM moved a raid boss spawn", append(p.logArgs(), "npc", n.Template.ID, "pos", p.Position())...)
		return
	case !ok || n == nil || n.Template.Type == "GrandBoss" || n.Leader() != nil:
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	sp := n.spawn
	if sp == nil || sp.deleted() {
		gmSysMessage(p, "Incorrect monster spawn.")
		s.log.Warn("game: //recall_npc of an NPC without a spawn", append(p.logArgs(), "npc", n.ID())...)
		return
	}
	respawn := int((sp.respawnMin + sp.respawnMax) / 2 / time.Second)
	oldID, tmpl := n.ID(), sp.tmpl
	s.deleteSpawn(sp)
	lo, hi := spawnRespawnDelays(respawn, 0)
	ns := &npcSpawn{tmpl: tmpl, point: p.Position(), respawnMin: lo, respawnMax: hi, gm: true, index: -1}
	s.addSpawnSlot(ns)
	if s.spawnSlot(ns) == nil {
		s.deleteSpawn(ns)
		gmSysMessage(p, "Target is not in game.")
		return
	}
	ns.respawn = lo > 0
	gmSysMessage(p, fmt.Sprintf("Created %s on %d.", tmpl.Name, oldID))
}

// npcTemplateByName returns the NPC template with the name, ignoring case
// (Mobius NpcData.getTemplateByName), or nil.
func (s *Server) npcTemplateByName(name string) *data.NpcTemplate {
	if s.worldData == nil {
		return nil
	}
	for i := range s.worldData.Npcs {
		if t := &s.worldData.Npcs[i]; strings.EqualFold(t.Name, name) {
			return t
		}
	}
	return nil
}

// gmListSpawns is //list_spawns <npcId|name> [index] and //list_positions
// (Mobius AdminSpawn findNpcs): the spawns of the NPC, numbered, with their
// points (//list_positions: where their NPC is); with an index, the GM
// teleports to that one. An NPC without a spawn is found in the world.
func (s *Server) gmListSpawns(positions bool) GMCommand {
	return func(p *Player, args []string) {
		npcID, index := 0, -1
		hasIndex := len(args) > 1 && adminNumeric(args[len(args)-1])
		search := args
		if hasIndex {
			search = args[:len(args)-1]
		}
		name := strings.TrimSpace(strings.Join(search, " "))
		found := true
		if adminNumeric(name) {
			npcID, found = javaParseInt(name)
		} else if t := s.npcTemplateByName(name); t != nil {
			npcID = t.ID
		} else {
			found = false
		}
		if found && hasIndex {
			index, _ = javaParseInt(args[len(args)-1])
		}
		s.adminFindNpcs(p, npcID, index, positions)
	}
}

// adminFindNpcs is AdminSpawn.findNpcs.
func (s *Server) adminFindNpcs(p *Player, npcID, index int, positions bool) {
	var spawns []*npcSpawn
	for _, sp := range s.npc.spawns {
		if sp.tmpl.ID == npcID {
			spawns = append(spawns, sp)
		}
	}
	for i, sp := range spawns {
		at := sp.point
		if sp.territory != nil {
			at = sp.lastPoint
		}
		shown := at
		if positions && sp.npc != nil {
			shown = sp.npc.Position()
		}
		switch {
		case index == i+1:
			s.teleportAround(p, data.Location{X: int(shown.X), Y: int(shown.Y), Z: int(shown.Z)}, maxOffsetOnTeleport)
			gmSysMessage(p, "You have been teleported.")
		case index < 0:
			p.SendMessage(fmt.Sprintf("%d - %s (Spawn ID: %d X: %d Y: %d Z: %d Heading: %d): %d %d %d",
				i+1, sp.tmpl.Name, sp.tmpl.ID, at.X, at.Y, at.Z, at.Heading, shown.X, shown.Y, shown.Z))
		}
	}
	if len(spawns) > 0 {
		return
	}
	var n *Npc
	for o := range s.world.Objects() {
		if v, ok := o.(*Npc); ok && v.Template.ID == npcID {
			n = v
			break
		}
	}
	if n == nil {
		p.SendMessage("The specified teleport target does not exist.")
		return
	}
	pos := n.Position()
	s.teleportAround(p, data.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z)}, maxOffsetOnTeleport)
	p.SendMessage("The current spawn is not stored.")
}

// --- Server tools ---

// gmServerInfo is //serverinfo (Mobius AdminServerInfo): the server info
// page. The memory, thread and garbage collector rows show the Go
// runtime's numbers: the heap in use, the memory taken from the system and
// the rest of it; the goroutines (none of them daemons) and the system
// threads made; the collections and their total pause.
func (s *Server) gmServerInfo(p *Player, args []string) {
	if len(args) > 0 {
		return // Mobius matches the whole command
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	const mb = 0x100000
	goroutines := strconv.Itoa(runtime.NumGoroutine())
	all, gms := 0, 0
	detached := 0
	ips := make(map[string]bool)
	for v := range s.Players() {
		all++
		if v.session == nil {
			detached++
			continue
		}
		if v.IsGM() {
			gms++
		}
		ips[gmRemoteHost(v.session.RemoteAddr())] = true
	}
	gt := s.world.GameTime()
	dayNight := "Day"
	if gt.IsNight() {
		dayNight = "Night"
	}
	pathfinding := "Enabled"
	if _, none := s.geo.Terrain().(geo.None); none {
		pathfinding = "Disabled"
	}
	up := s.Now().Sub(s.adminWorld.started)
	days := int(up / (24 * time.Hour))
	up -= time.Duration(days) * 24 * time.Hour
	hours := int(up / time.Hour)
	up -= time.Duration(hours) * time.Hour
	pg := s.pageFile(p, 0, "admin/serverinfo.htm")
	for _, r := range [][2]string{
		{"%os_name%", runtime.GOOS},
		{"%os_ver%", runtime.GOARCH + " (" + runtime.Version() + ")"},
		{"%slots%", strconv.Itoa(all) + "/" + strconv.Itoa(int(s.gs.Options.MaxPlayers))},
		{"%gameTime%", fmt.Sprintf("%d:%d", gt.Hour(), gt.Minute())},
		{"%dayNight%", dayNight},
		{"%geodata%", pathfinding},
		{"%serverTime%", s.Now().Format("03:04 PM")},
		{"%serverUpTime%", fmt.Sprintf("%d Days, %d Hours, %d Minutes", days, hours, int(up/time.Minute))},
		{"%onlineAll%", strconv.Itoa(all)},
		{"%offlineTrade%", strconv.Itoa(detached)},
		{"%onlineGM%", strconv.Itoa(gms)},
		{"%onlineReal%", strconv.Itoa(len(ips))},
		{"%usedMem%", strconv.FormatUint(mem.HeapAlloc/mb, 10) + " Mb"},
		{"%freeMem%", strconv.FormatUint((mem.Sys-mem.HeapAlloc)/mb, 10) + " Mb"},
		{"%totalMem%", strconv.FormatUint(mem.Sys/mb, 10) + " Mb"},
		{"%live%", goroutines},
		{"%nondaemon%", goroutines},
		{"%daemon%", "0"},
		{"%peak%", "n/a"},
		{"%totalstarted%", strconv.Itoa(pprof.Lookup("threadcreate").Count())},
		{"%gcol%", "Go " + runtime.Version()},
		{"%colcount%", strconv.FormatUint(uint64(mem.NumGC), 10)},
		{"%coltime%", strconv.FormatUint(mem.PauseTotalNs/uint64(time.Millisecond), 10)},
	} {
		pg.replace(r[0], r[1])
	}
	s.sendPage(p, pg)
}

// tellGMs sends text to every GM in the game (Mobius
// AdminData.broadcastMessageToGMs).
func (s *Server) tellGMs(text string) {
	for v := range s.Players() {
		if v.IsGM() {
			v.SendMessage(text)
		}
	}
}

// gmReloadMenuPage is //reload_menu_page <page>: a page of the reload menu.
func (s *Server) gmReloadMenuPage(p *Player, args []string) {
	page := 0
	if len(args) > 0 {
		page, _ = javaParseInt(args[0])
	}
	s.showReloadMenu(p, page)
}

// showReloadMenu is AdminReload.showReloadMenu: reload.htm with a value box
// and the reload links, 20 to a page, and Prev and Next.
func (s *Server) showReloadMenu(p *Player, page int) {
	total := len(adminReloadMenu)
	pages := (total + adminReloadPageSize - 1) / adminReloadPageSize
	page = min(max(page, 0), pages-1)
	var b strings.Builder
	b.WriteString(`<table width=260 cellspacing=0 cellpadding=0><tr><td fixwidth=50>Value:</td><td fixwidth=150 height=25><edit var="value" width=148></td></tr></table><br>`)
	for _, c := range adminReloadMenu[page*adminReloadPageSize : min((page+1)*adminReloadPageSize, total)] {
		b.WriteString(`<a action="bypass ` + c[1] + `">` + c[0] + `</a><br1>`)
	}
	if pages > 1 {
		button := func(value string, to int, ok bool) string {
			action := ""
			if ok {
				action = "bypass -h admin_reload_menu_page " + strconv.Itoa(to)
			}
			return `<td width=80 align=center><button value="` + value + `" action="` + action + `" width=75 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2"></td>`
		}
		b.WriteString(`<br><table width=260><tr>` + button("Prev", page-1, page > 0))
		fmt.Fprintf(&b, `<td width=100 align=center><font color="LEVEL">Page %d / %d</font></td>`, page+1, pages)
		b.WriteString(button("Next", page+1, page < pages-1) + `</tr></table>`)
	}
	s.sendPage(p, s.pageFile(p, 0, "admin/reload.htm").replace("%buttons%", b.String()))
}

// gmReload is //reload <type> [value] (Mobius AdminReload): html (or htm)
// reads the pages again (one page or folder with a value), quest (or
// scripts) loads the scripts again; the other types of Mobius answer that
// they are not available yet, and without a type the menu shows.
func (s *Server) gmReload(p *Player, args []string) {
	if len(args) == 0 {
		s.showReloadMenu(p, 0)
		return
	}
	switch typ := strings.ToLower(args[0]); typ {
	case "htm", "html":
		s.reloadHTML(p, args[1:])
	case "quest", "scripts":
		s.reloadScripts(p, args[1:])
		return // the warning follows the reload
	case "henna":
		s.reloadHennas(p) // henna.go
	default:
		known := false
		for _, c := range adminReloadMenu {
			if f := strings.Fields(c[1]); len(f) > 1 && f[1] == typ {
				known = true
			}
		}
		if known {
			p.SendMessage("Reloading " + typ + " is not available yet.")
		} else {
			p.SendMessage(adminReloadUsage)
		}
		return
	}
	gmSysMessage(p, adminReloadWarning)
}

// reloadHTML is //reload html [path]: without a path every page is read
// again on its next use (Mobius's lazy cache: the cache is emptied, so it
// reports no files), else the page, or the pages of the folder, at path
// below the HTML folder.
func (s *Server) reloadHTML(p *Player, args []string) {
	l := s.dialog.html
	if len(args) == 0 {
		clear(l.pages)
		gmSysMessage(p, "Cache[HTML]: 0.0 megabytes on 0 files loaded")
		s.tellGMs(p.Name() + ": Reloaded Htms.")
		return
	}
	rel, ok := cleanHTMLPath(args[0])
	var info os.FileInfo
	if ok {
		var err error
		info, err = os.Stat(filepath.Join(l.dir, filepath.FromSlash(rel)))
		ok = err == nil
	}
	if !ok {
		gmSysMessage(p, "File or Directory does not exist.")
		return
	}
	if info.IsDir() {
		for k := range l.pages {
			if strings.HasPrefix(k, rel+"/") {
				delete(l.pages, k)
			}
		}
	} else {
		l.reload(rel)
	}
	s.tellGMs(p.Name() + ": Reloaded Htm File:" + path.Base(rel) + ".")
}

// reloadScripts is //reload quest [name|id] and //reload scripts: the
// scripts load again off the loop and are swapped in (as //quest_reload);
// the quest states stay, as they live in Go. L2Go reloads all scripts, also
// for a single quest.
func (s *Server) reloadScripts(p *Player, args []string) {
	if s.scripts == nil {
		p.SendMessage("This server has no scripts.")
		return
	}
	engine, value := s.scripts, strings.Join(args, " ")
	go func() {
		ctx, cancel := context.WithTimeout(s.background(), saveTimeout)
		defer cancel()
		prepared, err := engine.Prepare(ctx)
		s.loop.Post(func() {
			if err != nil {
				gmSysMessage(p, "Script reload failed: "+err.Error())
				return
			}
			engine.Swap(prepared)
			switch {
			case value == "":
				gmSysMessage(p, "All scripts have been reloaded.")
				s.tellGMs(p.Name() + ": Reloaded Quests.")
			case adminNumeric(value):
				s.tellGMs(p.Name() + ": Reloaded Quest ID:" + value + ".")
			default:
				s.tellGMs(p.Name() + ": Reloaded Quest Name:" + value + ".")
			}
			gmSysMessage(p, adminReloadWarning)
		})
	}()
}
