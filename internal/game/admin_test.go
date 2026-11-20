package game

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// adminTestPages are short stand-ins for Mobius's admin pages (the real
// ones are not in git), with the same kinds of links.
var adminTestPages = map[string]string{
	"admin/main_menu.htm": `<html><title>Main Admin Menu</title><body>` +
		`<button value="Game" action="bypass admin_admin2">` +
		`<button value="Help" action="bypass admin_help">` +
		`<edit var="qbox"><button value="Nothing" action="bypass admin_no_such_command $qbox">` +
		`<button value="Kill" action="bypass -h admin_kill_menu"></body></html>`,
	"admin/game_menu.htm":  `<html><title>Game Panel</title><body><button value="Main" action="bypass admin_admin"></body></html>`,
	"admin/gm_menu.htm":    `<html><title>GM Menu</title><body><button value="Invis" action="bypass -h admin_invis_menu"></body></html>`,
	"admin/help.htm":       `<html><title>Admin Help</title><body><a action="bypass admin_html help/main_menu.htm">Introduction</a></body></html>`,
	"admin/move.htm":       `<html><title>Teleport Menu</title><body><button value="North" action="bypass -h admin_gonorth $offset"></body></html>`,
	"admin/charmanage.htm": `<html><body>Character Selection Menu</body></html>`,
}

// adminPage checks that ps has exactly one page, not bound to an NPC, with
// title, and returns its text.
func adminPage(t *testing.T, what string, ps []testclient.Packet, title string) string {
	t.Helper()
	pg := onePage(t, ps)
	if pg.ObjectID != 0 || !strings.Contains(pg.HTML, "<title>"+title+"</title>") {
		t.Errorf("%s: page of %d %q, want the admin page %q", what, pg.ObjectID, pg.HTML, title)
	}
	return pg.HTML
}

// //admin shows a GM the panel's main page (and the page's path, as for
// every page); its buttons run the GM commands, an unknown one gets
// Mobius's answer, and players get nothing.
func TestAdminPanel(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: adminTestPages})
	gs, _ := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	ps, pl := e.enterChar(testChar{account: "player", name: "Player", pos: eastOf(tiStart, 30)})

	gs.take(t)
	gmRun(e, gs, "admin")
	got := gs.take(t)
	adminPage(t, "//admin", got, "Main Admin Menu")
	if says := only[testclient.CreatureSay](got); len(says) != 1 || says[0].Name != "HTML" || says[0].Text != "html/admin/main_menu.htm" {
		t.Errorf("//admin debug lines %+v, want one for html/admin/main_menu.htm", says)
	}

	// The buttons: a menu, and back.
	adminPage(t, "Game button", e.bypass(gs, "admin_admin2"), "Game Panel")
	adminPage(t, "Main button", e.bypass(gs, "admin_admin"), "Main Admin Menu")
	// A command the page does not offer is dropped.
	noPage(t, "a bypass of no page", e.bypass(gs, "admin_admin7"))
	// A button of a command the server does not have yet.
	got = e.bypass(gs, "admin_no_such_command Bob")
	noPage(t, "no_such_command", got)
	if msgs := messages(got); !slices.Equal(msgs, []string{"The command 'no_such_command' does not exist!"}) {
		t.Errorf("no_such_command: messages %q", msgs)
	}
	adminPage(t, "Help button", e.bypass(gs, "admin_help"), "Admin Help")

	// A player gets nothing from //admin, nor from the page's buttons.
	ps.take(t)
	gmRun(e, ps, "admin")
	got = ps.take(t)
	noPage(t, "a player's //admin", got)
	if msgs := messages(got); len(msgs) != 0 {
		t.Errorf("a player's //admin: messages %q", msgs)
	}
	e.onLoop(func() { e.srv.ShowAdminPage(pl, "main_menu.htm") })
	ps.take(t)
	for _, cmd := range []string{"admin_admin2", "admin_no_such_command x"} {
		got = e.bypass(ps, cmd)
		noPage(t, "a player's "+cmd, got)
		if msgs := messages(got); len(msgs) != 0 {
			t.Errorf("a player's %s: messages %q", cmd, msgs)
		}
	}
}

func TestAdminMenus(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: adminTestPages})
	gs, _ := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	for _, tc := range []struct{ line, title string }{
		{"admin1", "Main Admin Menu"},
		{"admin2", "Game Panel"},
		{"admin7", "GM Menu"},
		{"help", "Admin Help"},
		{"tele", "Teleport Menu"},
	} {
		gs.take(t)
		gmRun(e, gs, tc.line)
		adminPage(t, "//"+tc.line, gs.take(t), tc.title)
	}
	// A menu whose page is missing gets AdminHtml's text.
	gmRun(e, gs, "admin4")
	if pg := onePage(t, gs.take(t)); pg.HTML != "<html><body>My text is missing:<br>data/html/admin/server_menu.htm</body></html>" {
		t.Errorf("//admin4 without its page: %q", pg.HTML)
	}
	// Page commands do nothing with arguments, as in Mobius.
	gmRun(e, gs, "help me")
	noPage(t, "//help me", gs.take(t))
}

func TestAdminHTML(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: adminTestPages})
	gs, _ := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})

	gmRun(e, gs, "html")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"Usage: //html path"}) {
		t.Errorf("//html: messages %q", msgs)
	}
	for _, path := range []string{"nosuch.htm", "../noquest.htm"} {
		gmRun(e, gs, "html "+path)
		if pg := onePage(t, gs.take(t)); pg.HTML != "<html><body>My text is missing:<br>data/html/admin/"+path+"</body></html>" {
			t.Errorf("//html %s: %q", path, pg.HTML)
		}
	}
	gmRun(e, gs, "html help.htm")
	adminPage(t, "//html help.htm", gs.take(t), "Admin Help")

	// //loadhtml reads an edited page again; //html then shows the new one.
	var dir string
	e.onLoop(func() { dir = e.srv.settings.HTMLDir })
	writeDialogFile(t, filepath.Join(dir, "admin", "help.htm"), `<html><title>New Help</title><body></body></html>`)
	gmRun(e, gs, "html help.htm")
	adminPage(t, "//html help.htm before //loadhtml", gs.take(t), "Admin Help")
	gmRun(e, gs, "loadhtml data/html/admin/help.htm")
	adminPage(t, "//loadhtml", gs.take(t), "New Help")
	gmRun(e, gs, "html help.htm")
	adminPage(t, "//html help.htm after //loadhtml", gs.take(t), "New Help")
	gmRun(e, gs, "loadhtml admin/help.htm")
	if pg := onePage(t, gs.take(t)); pg.HTML != "<html><body>My text is missing:<br>admin/help.htm</body></html>" {
		t.Errorf("//loadhtml outside the data folder: %q", pg.HTML)
	}
}

func TestAdminGoKillAndInvisMenu(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: adminTestPages})
	gs, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})

	// //gonorth 300 teleports the GM 300 units north, then shows the
	// teleport menu.
	gmRun(e, gs, "gonorth 300")
	got := gs.take(t)
	adminPage(t, "//gonorth", got, "Teleport Menu")
	tps := only[testclient.TeleportToLocation](got)
	if len(tps) != 1 || tps[0].X != tiStart.X || tps[0].Y != tiStart.Y-300 {
		t.Errorf("//gonorth 300: teleports %+v, want one to %d %d", tps, tiStart.X, tiStart.Y-300)
	}
	e.srv.Handle(gs, clientpackets.OpAppearing, nil)
	e.sync()
	gs.take(t)
	gmRun(e, gs, "goeast x")
	got = gs.take(t)
	noPage(t, "//goeast x", got)
	if msgs := messages(got); !slices.Equal(msgs, []string{"Usage: //go<north|south|east|west|up|down> [offset] (default 150)"}) {
		t.Errorf("//goeast x: messages %q", msgs)
	}

	// //kill_menu without a target: "Invalid target." and the main page.
	gmRun(e, gs, "kill_menu")
	got = gs.take(t)
	adminPage(t, "//kill_menu", got, "Main Admin Menu")
	if sms := only[testclient.SystemMessage](got); len(sms) != 1 || sms[0].ID != int32(gmSysMsgInvalidTarget) {
		t.Errorf("//kill_menu without a target: system messages %+v", sms)
	}

	// //invis_menu switches invisibility and shows the GM menu.
	for _, want := range []string{"Now, you cannot be seen.", "Now, you can be seen."} {
		gmRun(e, gs, "invis_menu")
		got = gs.take(t)
		adminPage(t, "//invis_menu", got, "GM Menu")
		if msgs := messages(got); !slices.Equal(msgs, []string{want}) {
			t.Errorf("//invis_menu: messages %q, want %q", msgs, want)
		}
	}
	var invisible bool
	e.onLoop(func() { invisible = gm.Invisible() })
	if invisible {
		t.Error("the GM is invisible after two //invis_menu")
	}

	// //goto_char_menu with nobody of that name.
	gmRun(e, gs, "goto_char_menu Nobody")
	if sms := only[testclient.SystemMessage](gs.take(t)); len(sms) != 1 || sms[0].ID != int32(adminSysMsgNotCurrentlyOnline) {
		t.Errorf("//goto_char_menu Nobody: system messages %+v", sms)
	}
}
