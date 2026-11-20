package game

import (
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// The Class button of //admin opens the class page, whose buttons change
// the class of the GM (no target) or of the targeted player, turn noblesse
// and hero status on and off, and list every class by race, each showing
// the page again; players get nothing.
func TestAdminClassMenu(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: adminTestPages})
	gs, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	ps, pl := e.enterChar(testChar{account: "player", name: "Player", pos: eastOf(tiStart, 30)})
	gs.take(t)
	ps.take(t)
	classOf := func(p *Player) int {
		var c int
		e.onLoop(func() { c = p.ClassID() })
		return c
	}
	contains := func(what, html string, wants ...string) {
		t.Helper()
		for _, want := range wants {
			if !strings.Contains(html, want) {
				t.Errorf("%s: page %q lacks %q", what, html, want)
			}
		}
	}

	gmRun(e, gs, "admin")
	html := adminPage(t, "//admin", gs.take(t), "Main Admin Menu")
	contains("//admin", html, `action="bypass -h admin_class_menu"`, `action="bypass -h admin_level_menu"`)

	// No target: the GM, a Human Fighter, with its next classes.
	html = adminPage(t, "Class button", e.bypass(gs, "admin_class_menu"), "Class")
	contains("Class button", html, "Gm (yourself)", `Class: <font color="LEVEL">Human Fighter</font>`,
		"admin_class_menu set 1\"", "admin_class_menu set 4\"", "admin_class_menu set 7\"",
		"admin_class_menu all", "admin_class_menu noble on", "admin_class_menu hero off", "bypass -h admin_admin\"")
	if strings.Contains(html, "admin_class_menu set 10\"") {
		t.Errorf("class page offers the Mage to a Human Fighter: %q", html)
	}

	// A player target: Warrior, then the page again.
	e.onLoop(func() { e.srv.SetTarget(gm, pl) })
	gs.take(t)
	html = adminPage(t, "Warrior", e.bypass(gs, "admin_class_menu set 1"), "Class")
	contains("Warrior", html, `<font color="LEVEL">Player</font>`, `Class: <font color="LEVEL">Warrior</font>`,
		"admin_class_menu set 2\"", "admin_class_menu set 3\"")
	if c := classOf(pl); c != 1 {
		t.Errorf("after Warrior: target class %d", c)
	}
	if c := classOf(gm); c != 0 {
		t.Errorf("after Warrior on the target: GM class %d", c)
	}

	// All classes: every race group, Human first; a race button shows its
	// classes; a class button changes the class.
	html = adminPage(t, "All classes", e.bypass(gs, "admin_class_menu all"), "All classes")
	contains("All classes", html, ">Human<", "admin_class_menu all 1", "admin_class_menu all 2",
		"admin_class_menu all 3", "admin_class_menu all 4", "admin_class_menu set 88\"", "bypass -h admin_class_menu\"")
	for race := range 5 {
		got := adminPage(t, "race page", e.bypass(gs, "admin_class_menu all "+string(rune('0'+race))), "All classes")
		if len(got) > 8000 {
			t.Errorf("race %d page is %d bytes", race, len(got))
		}
		if race == 4 {
			contains("Dwarf", got, "Dwarf", "admin_class_menu set 53\"", "admin_class_menu set 118\"")
		}
	}
	adminPage(t, "Maestro", e.bypass(gs, "admin_class_menu set 118"), "Class")
	if c := classOf(pl); c != 118 {
		t.Errorf("after Maestro: target class %d", c)
	}

	// Noble and Hero on and off.
	state := func() (noble, hero bool) {
		e.onLoop(func() { noble, hero = pl.IsNoble(), pl.IsHero() })
		return
	}
	contains("Noble on", adminPage(t, "Noble on", e.bypass(gs, "admin_class_menu noble on"), "Class"),
		`Noble: <font color="LEVEL">Yes</font>`)
	contains("Hero on", adminPage(t, "Hero on", e.bypass(gs, "admin_class_menu hero on"), "Class"),
		`Hero: <font color="LEVEL">Yes</font>`)
	if n, h := state(); !n || !h {
		t.Errorf("after On: noble %v, hero %v", n, h)
	}
	contains("Noble off", adminPage(t, "Noble off", e.bypass(gs, "admin_class_menu noble off"), "Class"),
		`Noble: <font color="LEVEL">No</font>`)
	contains("Hero off", adminPage(t, "Hero off", e.bypass(gs, "admin_class_menu hero off"), "Class"),
		`Hero: <font color="LEVEL">No</font>`)
	if n, h := state(); n || h {
		t.Errorf("after Off: noble %v, hero %v", n, h)
	}

	// A player gets nothing.
	gmRun(e, ps, "class_menu")
	noPage(t, "a player's //class_menu", ps.take(t))
	e.onLoop(func() { e.srv.sendPage(pl, newPage(0, e.srv.adminClassPage(pl, pl))) })
	ps.take(t)
	noPage(t, "a player's Warrior", e.bypass(ps, "admin_class_menu set 1"))
	noPage(t, "a player's Noble on", e.bypass(ps, "admin_class_menu noble on"))
	if c := classOf(pl); c != 118 {
		t.Errorf("a player's class button: class %d", c)
	}
	if n, _ := state(); n {
		t.Errorf("a player's Noble on made it noble")
	}
}
