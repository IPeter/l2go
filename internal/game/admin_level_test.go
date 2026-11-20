package game

import (
	"strconv"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// adminLevelShown checks that ps is the level page for name at level.
func adminLevelShown(t *testing.T, what string, html, name string, level string) {
	t.Helper()
	for _, want := range []string{`<font color="LEVEL">` + name, `Level: <font color="LEVEL">` + level + `</font>`} {
		if !strings.Contains(html, want) {
			t.Errorf("%s: page %q lacks %q", what, html, want)
		}
	}
}

// The Level button of //admin opens the level page, whose buttons set and
// change the level of the GM (no target) or of the targeted player, each
// showing the page again; players get nothing.
func TestAdminLevelMenu(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: adminTestPages})
	gs, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	ps, pl := e.enterChar(testChar{account: "player", name: "Player", pos: eastOf(tiStart, 30)})
	gs.take(t)
	ps.take(t)
	levelOf := func(p *Player) int {
		var l int
		e.onLoop(func() { l = p.Level() })
		return l
	}

	gmRun(e, gs, "admin")
	html := adminPage(t, "//admin", gs.take(t), "Main Admin Menu")
	if !strings.Contains(html, `action="bypass -h admin_level_menu"`) || !strings.Contains(html, "admin_kill_menu") {
		t.Fatalf("//admin: page %q lacks the Level button or its own buttons", html)
	}

	// No target: the GM's own level.
	html = adminPage(t, "Level button", e.bypass(gs, "admin_level_menu"), "Level")
	adminLevelShown(t, "Level button", html, "Gm (yourself)", "1")
	for _, want := range []string{"admin_level_menu set 40", "admin_level_menu add 1", "admin_level_menu add -1", "admin_level_menu set $level", "bypass -h admin_admin\""} {
		if !strings.Contains(html, want) {
			t.Errorf("level page lacks %q", want)
		}
	}
	html = adminPage(t, "Set level 40", e.bypass(gs, "admin_level_menu set 40"), "Level")
	adminLevelShown(t, "Set level 40", html, "Gm (yourself)", "40")
	if l := levelOf(gm); l != 40 {
		t.Errorf("after Set level 40: GM level %d", l)
	}

	// A player target.
	e.onLoop(func() { e.srv.SetTarget(gm, pl) })
	gs.take(t)
	html = adminPage(t, "Set level 40 on the target", e.bypass(gs, "admin_level_menu set 40"), "Level")
	adminLevelShown(t, "Set level 40 on the target", html, "Player", "40")
	if l := levelOf(pl); l != 40 {
		t.Errorf("after Set level 40: target level %d", l)
	}
	adminLevelShown(t, "Level +1", onePage(t, e.bypass(gs, "admin_level_menu add 1")).HTML, "Player", "41")
	adminLevelShown(t, "Level -1", onePage(t, e.bypass(gs, "admin_level_menu add -1")).HTML, "Player", "40")
	// The edit box, as the client fills it in.
	adminLevelShown(t, "Set with 33", onePage(t, e.bypass(gs, "admin_level_menu set 33")).HTML, "Player", "33")
	if l := levelOf(pl); l != 33 {
		t.Errorf("after Set 33: target level %d", l)
	}
	// Beyond the max level: refused, the page shows again.
	top := e.srv.MaxLevel() - 1
	e.bypass(gs, "admin_level_menu set "+strconv.Itoa(top))
	got := e.bypass(gs, "admin_level_menu add 1")
	adminLevelShown(t, "Level +1 at the top", onePage(t, got).HTML, "Player", strconv.Itoa(top))
	if l := levelOf(gm); l != 40 {
		t.Errorf("the target's level changes moved the GM to %d", l)
	}

	// Back.
	adminPage(t, "Back button", e.bypass(gs, "admin_admin"), "Main Admin Menu")

	// A player gets nothing.
	gmRun(e, ps, "level_menu")
	noPage(t, "a player's //level_menu", ps.take(t))
	e.onLoop(func() { e.srv.sendPage(pl, newPage(0, e.srv.adminLevelPage(pl, pl))) })
	ps.take(t)
	noPage(t, "a player's Set level 40", e.bypass(ps, "admin_level_menu set 40"))
	if l := levelOf(pl); l != top {
		t.Errorf("a player's Set level 40: level %d", l)
	}
}
