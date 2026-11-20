package game

import (
	"slices"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
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
