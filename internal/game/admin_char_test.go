package game

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// adminCharTestPages are short stand-ins for Mobius's character pages, with
// their placeholders and buttons.
var adminCharTestPages = map[string]string{
	"admin/charinfo.htm": `<html><title>Information Panel: %name%</title><body>` +
		`<button value="Edit" action="bypass admin_edit_character">` +
		`<button value="Lv/Exp/Sp" action="bypass admin_add_exp_sp_to_character">` +
		`Level %level% Class %class% Base %baseclass% Exp %xp% SP %sp% Account %account% ` +
		`PvP %pvpkills% PK %pkkills% Karma %karma% Noblesse %noblesse% Access %access%</body></html>`,
	"admin/charedit.htm": `<html><title>Editing %name%</title><body>` +
		`<button value="Back" action="bypass admin_character_info %name%">` +
		`PvP %pvpkills% PK %pkkills%</body></html>`,
	"admin/expsp.htm": `<html><body>Editing %name% Level: %level% Class: %class% Exp: %xp% SP: %sp%` +
		`<edit var="exp_to_add"><edit var="sp_to_add">` +
		`<button value="Add" action="bypass -h admin_add_exp_sp $exp_to_add $sp_to_add">` +
		`<button value="Remove" action="bypass -h admin_remove_exp_sp $exp_to_add $sp_to_add">` +
		`<edit var="lv"><button value="Set Level" action="bypass -h admin_set_level $lv">` +
		`</body></html>`,
	"admin/charlist.htm":    `<html><title>Character List</title><body>%players%%pages%</body></html>`,
	"admin/charfind.htm":    `<html><title>Character Search</title><body>%results%Found %number% character%end%</body></html>`,
	"admin/ipfind.htm":      `<html><title>Ip Menu</title><body>%ip%: %results%Found %number% character%end%</body></html>`,
	"admin/accountinfo.htm": `<html><title>Account Details: %account%</title><body>%player%: %characters%</body></html>`,
	"admin/dualbox.htm":     `<html><title>Find Dualbox Panel</title><body>(%multibox%) %results% %strict%</body></html>`,
	"admin/punishments/punishment-info.htm": `<html><title>Punishment Information Panel</title><body>` +
		`%affect_type%: %player_name% [%affects%]%punishments%</body></html>`,
	"jail_in.htm":  `<html><title>Jail</title><body>Reason: %reason% Applied by: %punishedBy%</body></html>`,
	"jail_out.htm": `<html><title>Free</title><body>Out.</body></html>`,
}

// awaitMessage collects sess's packets until one says want (the commands
// that read or write the store answer from another goroutine).
func awaitMessage(t *testing.T, e *testEnv, sess *fakeSession, want string) []testclient.Packet {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var all []testclient.Packet
	for {
		e.sync()
		all = append(all, sess.take(t)...)
		if slices.Contains(messages(all), want) {
			return all
		}
		if time.Now().After(deadline) {
			t.Fatalf("no message %q in %q", want, messages(all))
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// lastUserInfo returns the last UserInfo of ps.
func lastUserInfo(t *testing.T, what string, ps []testclient.Packet) testclient.UserInfo {
	t.Helper()
	uis := only[testclient.UserInfo](ps)
	if len(uis) == 0 {
		t.Fatalf("%s: no UserInfo in %q", what, testclient.Names(ps))
	}
	return uis[len(uis)-1]
}

// storedChar returns the stored character called name.
func storedChar(t *testing.T, e *testEnv, name string) storage.Character {
	t.Helper()
	c, err := e.store.GetByName(e.ctx, name)
	if err != nil {
		t.Fatalf("GetByName(%s): %v", name, err)
	}
	return c
}

// adminCharEnv returns an environment with the character pages, a GM and a
// player the GM targets.
func adminCharEnv(t *testing.T) (*testEnv, *fakeSession, *Player, *fakeSession, *Player) {
	t.Helper()
	pages := make(map[string]string)
	for k, v := range adminTestPages {
		pages[k] = v
	}
	for k, v := range adminCharTestPages {
		pages[k] = v
	}
	e := newDialogEnv(t, dialogEnvOptions{pages: pages})
	gs, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	ps, pl := e.enterChar(testChar{account: "player", name: "Player", pos: eastOf(tiStart, 30)})
	e.onLoop(func() { e.srv.SetTarget(gm, pl) })
	gs.take(t)
	ps.take(t)
	return e, gs, gm, ps, pl
}

// The character page, then its Lv/Exp/Sp button: the exp/SP page, whose
// buttons add exp and set the level, each showing the page again.
func TestAdminCharLevelPages(t *testing.T) {
	e, gs, _, _, pl := adminCharEnv(t)

	gmRun(e, gs, "character_info Player")
	html := adminPage(t, "//character_info", gs.take(t), "Information Panel: Player")
	for _, want := range []string{"Level 1 ", "Class Human Fighter ", "Base Human Fighter ", "Account player ", "Noblesse No ", "Access 0 (User)"} {
		if !strings.Contains(html, want) {
			t.Errorf("//character_info: %q lacks %q", html, want)
		}
	}

	got := e.bypass(gs, "admin_add_exp_sp_to_character")
	pg := onePage(t, got)
	if !strings.Contains(pg.HTML, "Editing Player Level: 1 Class: Human Fighter Exp: 0 SP: 0") {
		t.Errorf("Lv/Exp/Sp button: page %q", pg.HTML)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"Usage: //add_exp_sp exp sp"}) {
		t.Errorf("Lv/Exp/Sp button: messages %q", msgs)
	}

	got = e.bypass(gs, "admin_add_exp_sp 1000 50")
	if pg := onePage(t, got); !strings.Contains(pg.HTML, "Exp: 1000 SP: 50") {
		t.Errorf("Add button: page %q", pg.HTML)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"Added 1000 xp and 50 sp to Player."}) {
		t.Errorf("Add button: messages %q", msgs)
	}

	got = e.bypass(gs, "admin_set_level 20")
	if msgs := messages(got); len(msgs) != 1 || !strings.HasPrefix(msgs[0], "Added ") {
		t.Errorf("Set Level button: messages %q", msgs)
	}
	var level int
	e.onLoop(func() { level = pl.Level() })
	if level != 20 {
		t.Errorf("after Set Level 20: level %d", level)
	}

	got = e.bypass(gs, "admin_remove_exp_sp 1 0")
	if pg := onePage(t, got); !strings.Contains(pg.HTML, "Level: 19 ") {
		t.Errorf("Remove button: page %q", pg.HTML)
	}
}

// The edit page and its Back button, and the answers without a target.
func TestAdminCharEditPage(t *testing.T) {
	e, gs, gm, _, _ := adminCharEnv(t)

	gmRun(e, gs, "edit_character")
	adminPage(t, "//edit_character", gs.take(t), "Editing Player")
	adminPage(t, "Back button", e.bypass(gs, "admin_character_info Player"), "Information Panel: Player")

	gmRun(e, gs, "current_player")
	adminPage(t, "//current_player", gs.take(t), "Information Panel: Gm")

	e.onLoop(func() { e.srv.SetTarget(gm, nil) })
	gs.take(t)
	for _, line := range []string{"edit_character", "character_info", "add_exp_sp_to_character"} {
		gmRun(e, gs, line)
		got := gs.take(t)
		noPage(t, "//"+line+" without a target", got)
		if sms := only[testclient.SystemMessage](got); len(sms) == 0 || sms[0].ID != int32(gmSysMsgInvalidTarget) {
			t.Errorf("//%s without a target: system messages %+v", line, sms)
		}
	}
}

// The edits of the Char menu and the edit page reach the target's UserInfo
// (and its title the players around), with Mobius's messages.
func TestAdminCharEdits(t *testing.T) {
	e, gs, gm, ps, pl := adminCharEnv(t)
	run := func(line string, wantGM, wantTarget []string) {
		t.Helper()
		gmRun(e, gs, line)
		if msgs := messages(gs.take(t)); !slices.Equal(msgs, wantGM) {
			t.Errorf("//%s: GM messages %q, want %q", line, msgs, wantGM)
		}
		if msgs := messages(ps.take(t)); !slices.Equal(msgs, wantTarget) {
			t.Errorf("//%s: target messages %q, want %q", line, msgs, wantTarget)
		}
	}
	run("setpvp 7", []string{"Player's PVP count changed to 7"}, []string{"A GM changed your PVP count to 7"})
	run("setpk 3", []string{"Player's PK count changed to 3"}, []string{"A GM changed your PK count to 3"})
	run("rec 300", []string{"Player's Recommend changed to 300"}, []string{"A GM changed your Recommend points to 300"})
	run("settitle The Brave One", nil, []string{"Your title has been changed by a GM"})
	run("setcolor FF00", nil, []string{"Your name color has been changed by a GM"})
	run("settcolor 0000FF", nil, []string{"Your title color has been changed by a GM"})
	run("sethero", nil, nil)
	run("setsex", nil, []string{"Your gender has been changed by a GM"})
	run("setnoble", []string{"You've changed nobless status of: Player"}, []string{"GM changed your nobless status!"})
	run("setfame 10", []string{"Fame system is not enabled on the server!"}, nil)
	run("setpvp x", []string{"Usage: //setpvp <pvp_count>"}, nil)
	run("settitle", []string{"You need to specify the new title."}, nil)
	run("setcolor nothex", []string{"You need to specify a valid new color."}, nil)

	e.advance(world.GameTick)
	ui := lastUserInfo(t, "the target", ps.take(t))
	if ui.PvpKills != 7 || ui.PkKills != 3 || ui.RecomHave != 255 || ui.Title != "The Brave One" ||
		ui.NameColor != 0xFF00 || ui.TitleColor != 0xFF || !ui.Hero || ui.Sex != 1 {
		t.Errorf("UserInfo pvp %d pk %d rec %d title %q colors %x %x hero %v sex %d",
			ui.PvpKills, ui.PkKills, ui.RecomHave, ui.Title, ui.NameColor, ui.TitleColor, ui.Hero, ui.Sex)
	}
	var noble bool
	var stored storage.Character
	e.onLoop(func() { noble, stored = pl.IsNoble(), pl.Snapshot() })
	if !noble || stored.PvPKills != 7 || stored.PKKills != 3 || stored.RecomHave != 255 || stored.Title != "The Brave One" || stored.Sex != 1 {
		t.Errorf("noble %v, snapshot pvp %d pk %d rec %d title %q sex %d", noble, stored.PvPKills, stored.PKKills, stored.RecomHave, stored.Title, stored.Sex)
	}
	// The GM sees the title in the player's CharInfo.
	if cis := only[testclient.CharInfoPacket](gs.take(t)); len(cis) == 0 || cis[len(cis)-1].Title != "The Brave One" {
		t.Errorf("the GM's CharInfo of the player: %+v", cis)
	}

	// A title longer than 21 characters is cut to 20.
	run("settitle 1234567890123456789012", nil, []string{"Your title has been changed by a GM"})
	e.onLoop(func() { stored = pl.Snapshot() })
	if stored.Title != "12345678901234567890" {
		t.Errorf("a long title became %q", stored.Title)
	}

	// //set_hp, //set_mp and //set_cp.
	run("set_hp 10", nil, nil)
	run("set_cp 5", nil, nil)
	run("set_mp x", []string{"Usage: //set_mp 1000"}, nil)
	var hp, cp float64
	e.onLoop(func() { hp, cp = pl.CurHP, pl.CurCP })
	if hp != 10 || cp != 5 {
		t.Errorf("after //set_hp 10 and //set_cp 5: HP %v CP %v", hp, cp)
	}

	// Without a player target.
	e.onLoop(func() { e.srv.SetTarget(gm, nil) })
	gs.take(t)
	for _, line := range []string{"setpk 1", "setpvp 1", "rec 1", "set_hp 1", "sethero"} {
		gmRun(e, gs, line)
		if sms := only[testclient.SystemMessage](gs.take(t)); len(sms) != 1 || sms[0].ID != int32(gmSysMsgInvalidTarget) {
			t.Errorf("//%s without a target: system messages %+v", line, sms)
		}
	}
	// //setnoble without a target is for the GM itself.
	run("setnoble", []string{"GM changed your nobless status!"}, nil)
}

// //changename renames the target in the store and in the game; a name
// another character has is refused.
func TestAdminCharChangeName(t *testing.T) {
	e, gs, _, ps, pl := adminCharEnv(t)
	gmRun(e, gs, "changename Gm")
	awaitMessage(t, e, gs, "Warning, player Gm already exists")

	gmRun(e, gs, "changename Hero")
	awaitMessage(t, e, gs, "Changed name to Hero")
	if msgs := messages(ps.take(t)); !slices.Equal(msgs, []string{"Your name has been changed by a GM."}) {
		t.Errorf("the target's messages %q", msgs)
	}
	var found *Player
	e.onLoop(func() { found, _ = e.srv.PlayerByName("hero") })
	if found != pl || pl.Name() != "Hero" {
		t.Errorf("PlayerByName(hero) = %v, name %q", found, pl.Name())
	}
	if c := storedChar(t, e, "Hero"); c.ObjectID != pl.ObjectID() {
		t.Errorf("stored Hero is %d, want %d", c.ObjectID, pl.ObjectID())
	}
	e.advance(world.GameTick)
	if ui := lastUserInfo(t, "the renamed player", ps.take(t)); ui.Name != "Hero" {
		t.Errorf("UserInfo name %q", ui.Name)
	}
}

// //kick_menu kicks a player; //ban_char bans a character (offline: in the
// store; online: kicked too), and //unban_char lifts it.
func TestAdminCharKickAndBan(t *testing.T) {
	e, gs, _, ps, _ := adminCharEnv(t)
	vs, _ := e.enterChar(testChar{account: "victim", name: "Victim", pos: eastOf(tiStart, 60)})

	gmRun(e, gs, "kick_menu Player")
	got := gs.take(t)
	if pg := lastPage(t, got); !strings.Contains(pg, "Character Selection Menu") {
		t.Errorf("//kick_menu: page %q", pg)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"You kicked Player from the game."}) {
		t.Errorf("//kick_menu: messages %q", msgs)
	}
	if _, _, closed := ps.closedWith(t); !closed {
		t.Error("the kicked player's connection is open")
	}
	gmRun(e, gs, "kick_menu Player")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"Player Player was not found in the game."}) {
		t.Errorf("//kick_menu of nobody online: messages %q", msgs)
	}

	// Offline.
	gmRun(e, gs, "ban_char Player")
	got = awaitMessage(t, e, gs, "Punishment BAN have been applied to: CHARACTER Player!")
	got = append(got, awaitPage(t, e, gs, got)...) // shown once the character is written
	if c := storedChar(t, e, "Player"); c.AccessLevel != storage.AccessLevelBanned {
		t.Errorf("banned Player has access level %d", c.AccessLevel)
	}
	if pg := lastPage(t, got); !strings.Contains(pg, "CHARACTER: Player [ACCOUNT;CHARACTER;IP;HWID]") ||
		!strings.Contains(pg, `>BAN</font></td><td>never</td><td><a action="bypass -h admin_punishment_remove Player CHARACTER BAN">`) {
		t.Errorf("the punishment page after //ban_char: %q", pg)
	}
	gmRun(e, gs, "ban_char Player")
	awaitMessage(t, e, gs, "Target is already affected by that punishment.")
	gmRun(e, gs, "unban_char Player")
	got = awaitMessage(t, e, gs, "Punishment BAN have been stopped to: CHARACTER Player!")
	awaitPage(t, e, gs, got) // shown once the character is written
	if c := storedChar(t, e, "Player"); c.AccessLevel != storage.AccessLevelPlayer {
		t.Errorf("unbanned Player has access level %d", c.AccessLevel)
	}
	gmRun(e, gs, "ban_char Nobody")
	deadline := time.Now().Add(10 * time.Second)
	for {
		e.sync()
		if sms := only[testclient.SystemMessage](gs.take(t)); len(sms) > 0 {
			if sms[0].ID != int32(sysMsgDoesNotExist) {
				t.Errorf("//ban_char Nobody: system message %+v", sms[0])
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no answer to //ban_char Nobody")
		}
		time.Sleep(2 * time.Millisecond)
	}

	// Online: the player is kicked with ServerClose and saved banned.
	gmRun(e, gs, "ban_char Victim")
	awaitMessage(t, e, gs, "Punishment BAN have been applied to: CHARACTER Victim!")
	if _, _, closed := vs.closedWith(t); !closed {
		t.Error("the banned player's connection is open")
	}
	deadline = time.Now().Add(10 * time.Second)
	for storedChar(t, e, "Victim").AccessLevel != storage.AccessLevelBanned {
		if time.Now().After(deadline) {
			t.Fatal("Victim was not saved banned")
		}
		time.Sleep(2 * time.Millisecond)
	}
	gmRun(e, gs, "ban_hwid x")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"The punishment BAN of HWID is not available."}) {
		t.Errorf("//ban_hwid: messages %q", msgs)
	}
	gmRun(e, gs, "punishment_add Victim CHARACTER BAN 10 cheating")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"Bans for a time are not available; use 0 minutes for a permanent ban."}) {
		t.Errorf("a timed ban: messages %q", msgs)
	}
}

// awaitPage collects sess's packets until there is a page in got or them.
func awaitPage(t *testing.T, e *testEnv, sess *fakeSession, got []testclient.Packet) []testclient.Packet {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var more []testclient.Packet
	for len(only[testclient.NpcHtmlMessage](append(slices.Clone(got), more...))) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("no page")
		}
		time.Sleep(2 * time.Millisecond)
		e.sync()
		more = append(more, sess.take(t)...)
	}
	return more
}

// lastPage returns the text of the last page of ps.
func lastPage(t *testing.T, ps []testclient.Packet) string {
	t.Helper()
	pages := only[testclient.NpcHtmlMessage](ps)
	if len(pages) == 0 {
		t.Fatalf("no page in %q", testclient.Names(ps))
	}
	return pages[len(pages)-1].HTML
}

// accessRecorder stands in for the login link.
type accessRecorder struct {
	mu  sync.Mutex
	got []string
	err error
}

func (r *accessRecorder) ChangeAccessLevel(account string, level int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, fmt.Sprintf("%s=%d", account, level))
	return r.err
}

// //ban_acc tells the login server (access level -1) and kicks the
// account's player; //unban_acc sets 0. Without a login link it is not
// available.
func TestAdminCharAccountBan(t *testing.T) {
	e, gs, _, ps, _ := adminCharEnv(t)
	gmRun(e, gs, "ban_acc player")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"Account bans are not available: the game server has no link to the login server."}) {
		t.Errorf("//ban_acc without a link: messages %q", msgs)
	}
	rec := &accessRecorder{}
	e.onLoop(func() { e.srv.adminChar.accounts = rec })
	gmRun(e, gs, "ban_acc Player")
	awaitMessage(t, e, gs, "Punishment BAN have been applied to: ACCOUNT Player!")
	if _, _, closed := ps.closedWith(t); !closed {
		t.Error("the banned account's player is still connected")
	}
	gmRun(e, gs, "unban_acc player")
	awaitMessage(t, e, gs, "Punishment BAN have been stopped to: ACCOUNT player!")
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if !slices.Equal(rec.got, []string{"player=-1", "player=0"}) {
		t.Errorf("the login server was told %q", rec.got)
	}
}

// A chat ban stops the player's chat until it ends; //unban_chat ends it
// at once.
func TestAdminCharChatBan(t *testing.T) {
	e, gs, _, ps, _ := adminCharEnv(t)
	gmRun(e, gs, "ban_chat Player 10")
	got := ps.take(t)
	if msgs := messages(got); !slices.Equal(msgs, []string{"You've been chat banned for 10 minute's."}) {
		t.Errorf("the chat banned player's messages %q", msgs)
	}
	if sms := only[testclient.SystemMessage](got); len(sms) == 0 || sms[0].ID != int32(sysMsgChatDisabled) {
		t.Errorf("the chat banned player's system messages %+v", sms)
	}
	if etc := only[testclient.EtcStatusUpdate](got); len(etc) != 1 || !etc[0].ChatBlocked {
		t.Errorf("EtcStatusUpdate after the chat ban: %+v", etc)
	}
	if msgs := messages(gs.take(t)); !slices.Contains(msgs, "Punishment CHAT_BAN have been applied to: CHARACTER Player!") {
		t.Errorf("the GM's messages %q", msgs)
	}

	say := func(text string) []testclient.Packet {
		e.srv.Handle(ps, clientpackets.OpSay2, chatBody(serverpackets.ChatGeneral, text, ""))
		e.sync()
		return ps.take(t)
	}
	got = say("hello")
	if sms := only[testclient.SystemMessage](got); len(sms) != 1 || sms[0].ID != int32(sysMsgChattingProhibited) || len(only[testclient.CreatureSay](got)) != 0 {
		t.Errorf("a chat banned player's chat: %q %+v", testclient.Names(got), sms)
	}

	e.advance(10*time.Minute + time.Second)
	if sms := only[testclient.SystemMessage](ps.take(t)); len(sms) != 1 || sms[0].ID != int32(sysMsgChatEnabled) {
		t.Errorf("at the end of the chat ban: system messages %+v", sms)
	}
	if got = say("free"); len(only[testclient.CreatureSay](got)) != 1 {
		t.Errorf("chat after the ban: %q", testclient.Names(got))
	}

	gmRun(e, gs, "ban_chat Player")
	if msgs := messages(ps.take(t)); !slices.Equal(msgs, []string{"You've been chat banned forever."}) {
		t.Errorf("a permanent chat ban: messages %q", msgs)
	}
	gmRun(e, gs, "unban_chat Player")
	if sms := only[testclient.SystemMessage](ps.take(t)); len(sms) != 1 || sms[0].ID != int32(sysMsgChatEnabled) {
		t.Errorf("//unban_chat: system messages %+v", sms)
	}
	if got = say("free again"); len(only[testclient.CreatureSay](got)) != 1 {
		t.Errorf("chat after //unban_chat: %q", testclient.Names(got))
	}
}

// //jail sends the player to jail 2 s later with Mobius's page, and
// //unjail brings it out.
func TestAdminCharJail(t *testing.T) {
	e, gs, _, ps, pl := adminCharEnv(t)
	gmRun(e, gs, "jail Player")
	got := ps.take(t)
	if pg := lastPage(t, got); !strings.Contains(pg, "Reason: Jailed by admin Applied by: Gm") {
		t.Errorf("the jail page %q", pg)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"You've been jailed forever."}) {
		t.Errorf("the jailed player's messages %q", msgs)
	}
	e.advance(2 * time.Second)
	tps := only[testclient.TeleportToLocation](ps.take(t))
	if len(tps) != 1 || tps[0].X != jailInLoc.X || tps[0].Y != jailInLoc.Y {
		t.Fatalf("teleports into the jail: %+v", tps)
	}
	e.srv.Handle(ps, clientpackets.OpAppearing, nil)
	e.sync()
	ps.take(t)
	var jailed bool
	e.onLoop(func() { jailed = pl.jailed() })
	if !jailed {
		t.Error("the player is not jailed")
	}
	// No whispers out of the jail.
	e.srv.Handle(ps, clientpackets.OpSay2, chatBody(serverpackets.ChatWhisper, "help", "Gm"))
	e.sync()
	if msgs := messages(ps.take(t)); !slices.Equal(msgs, []string{"You can not chat with players outside of the jail."}) {
		t.Errorf("a whisper from jail: messages %q", msgs)
	}

	gmRun(e, gs, "unjail Player")
	if pg := lastPage(t, ps.take(t)); !strings.Contains(pg, "Out.") {
		t.Errorf("the page out of jail %q", pg)
	}
	e.advance(2 * time.Second)
	if tps := only[testclient.TeleportToLocation](ps.take(t)); len(tps) != 1 || tps[0].X != jailOutLoc.X {
		t.Errorf("teleports out of the jail: %+v", tps)
	}
	gmRun(e, gs, "unjail Player")
	awaitMessage(t, e, gs, "Target is not affected by that punishment!")
}

// The character lists: //show_characters, //find_character, //find_ip,
// //find_account and //find_dualbox.
func TestAdminCharLists(t *testing.T) {
	e, gs, _, _, _ := adminCharEnv(t)
	gmRun(e, gs, "show_characters page=0")
	html := adminPage(t, "//show_characters", gs.take(t), "Character List")
	if !strings.Contains(html, `<a action="bypass -h admin_character_info Player">Player</a></td><td width=110>Human Fighter</td><td width=40>1</td>`) ||
		!strings.Contains(html, ">Gm</a>") {
		t.Errorf("//show_characters: %q", html)
	}
	gmRun(e, gs, "find_character LAY")
	html = adminPage(t, "//find_character", gs.take(t), "Character Search")
	if !strings.Contains(html, ">Player</a>") || strings.Contains(html, ">Gm</a>") || !strings.HasSuffix(html, "Found 1 character.</body></html>") {
		t.Errorf("//find_character LAY: %q", html)
	}
	gmRun(e, gs, "find_ip nonsense")
	got := gs.take(t)
	adminPage(t, "//find_ip nonsense", got, "Character List")
	if msgs := messages(got); !slices.Equal(msgs, []string{"Usage: //find_ip <www.xxx.yyy.zzz>"}) {
		t.Errorf("//find_ip nonsense: messages %q", msgs)
	}
	gmRun(e, gs, "find_ip 10.0.0.1")
	if html = adminPage(t, "//find_ip", gs.take(t), "Ip Menu"); !strings.HasSuffix(html, "Found 0 characters. Maybe they got d/c? :)</body></html>") {
		t.Errorf("//find_ip 10.0.0.1: %q", html)
	}
	gmRun(e, gs, "find_account Player")
	got = awaitPage(t, e, gs, nil)
	if html = lastPage(t, got); !strings.Contains(html, "<title>Account Details: player</title>") {
		t.Errorf("//find_account: %q", html)
	}
	gmRun(e, gs, "find_dualbox 1")
	if html = adminPage(t, "//find_dualbox", gs.take(t), "Find Dualbox Panel"); !strings.Contains(html, "(1)") ||
		!strings.Contains(html, `<a action="bypass -h admin_find_ip test/gm">test/gm (1)</a><br1>`) {
		t.Errorf("//find_dualbox 1: %q", html)
	}
	gmRun(e, gs, "find_dualbox 0")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"Usage: //find_dualbox [number > 0]"}) {
		t.Errorf("//find_dualbox 0: messages %q", msgs)
	}
}

// //repair moves an offline character to Talking Island, drops its
// shortcuts and puts its items in the inventory.
func TestAdminCharRepair(t *testing.T) {
	e, gs, _, _, _ := adminCharEnv(t)
	c := e.createCharacter("lost", "Lost", classHumanFighter, storage.AccessLevelPlayer)
	c.Shortcuts = []storage.Shortcut{{Slot: 1, Type: storage.ShortcutAction, ID: 2}}
	if err := e.store.Update(e.ctx, &c); err != nil {
		t.Fatal(err)
	}
	gmRun(e, gs, "repair Lost")
	awaitMessage(t, e, gs, "Character Lost repaired.")
	got := storedChar(t, e, "Lost")
	if got.X != adminRepairX || got.Y != adminRepairY || got.Z != adminRepairZ || got.Shortcuts != nil {
		t.Errorf("repaired at %d %d %d with shortcuts %+v", got.X, got.Y, got.Z, got.Shortcuts)
	}
	for _, it := range got.Items {
		if it.Loc != storage.LocInventory {
			t.Errorf("item %d of the repaired character is in %s", it.ItemID, it.Loc)
		}
	}
	gmRun(e, gs, "repair Player")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"The character Player is online; log it out to repair it."}) {
		t.Errorf("//repair of an online player: messages %q", msgs)
	}
}
