package game

import (
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/storage"
)

// The level page of the admin panel (an L2Go page, not in Mobius): one-click
// buttons that set or change the level of the GM's player target, or of the
// GM when no player is targeted, with //set_level and //add_level, then show
// the page again with the new level. //admin's main page gets a button to it.
func init() { registerFeature("admin_level", setupAdminLevel) }

// adminLevelPresets are the levels of the page's "set level" buttons.
var adminLevelPresets = []int{1, 10, 20, 40, 52, 61, 76, 78, 80}

// adminLevelButtonRow is the row added to //admin's main page: the Level
// and Class (admin_class.go) buttons.
const adminLevelButtonRow = `<center><table width=270><tr><td>` +
	`<button value="Level" action="bypass -h admin_level_menu" width=65 height=21 back="sek.cbui94" fore="sek.cbui92">` +
	`</td><td>` +
	`<button value="Class" action="bypass -h admin_class_menu" width=65 height=21 back="sek.cbui94" fore="sek.cbui92">` +
	`</td></tr></table></center>`

func setupAdminLevel(s *Server) error {
	return s.RegisterGMCommand("level_menu", storage.AccessLevelAdmin, s.gmAdminLevelMenu)
}

// adminAddLevelButton puts the Level button row into the main admin page,
// before its last </body> (at the end when it has none).
func adminAddLevelButton(content string) string {
	i := strings.LastIndex(strings.ToLower(content), "</body>")
	if i < 0 {
		return content + adminLevelButtonRow
	}
	return content[:i] + adminLevelButtonRow + content[i:]
}

// gmAdminLevelMenu is //level_menu: the level page. //level_menu set <level>
// and //level_menu add <n> first run //set_level and //add_level on the GM's
// player target, or on the GM without one; //level_menu skills gives it all
// its missing class skills (//give_all_skills).
func (s *Server) gmAdminLevelMenu(p *Player, args []string) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		t = p
	}
	if len(args) > 0 {
		switch args[0] {
		case "set":
			s.gmSetLevelOf(p, t, args[1:])
		case "add":
			n := 0
			if len(args) > 1 {
				n, _ = strconv.Atoi(args[1])
			}
			switch top := s.MaxLevel() - 1; {
			case n == 0:
				p.SendMessage("Wrong Number Format")
			case t.Level()+n > top:
				p.SendMessage("The level cannot go beyond " + strconv.Itoa(top) + ".")
			default:
				s.AddLevel(t, n)
			}
		case "skills":
			// //give_all_skills on the target or the GM: every class skill
			// missing at the current level.
			n := s.giveAvailableSkills(t, false, false)
			gmSysMessage(p, "You gave "+strconv.Itoa(n)+" skills to "+t.Name())
			t.SendSkillList()
		}
	}
	s.sendPage(p, newPage(0, s.adminLevelPage(p, t)))
}

// adminLevelPage is the level page for the GM p acting on t.
func (s *Server) adminLevelPage(p, t *Player) string {
	button := func(value, action string, width int) string {
		return `<button value="` + value + `" action="bypass -h admin_` + action + `" width=` +
			strconv.Itoa(width) + ` height=21 back="sek.cbui94" fore="sek.cbui92">`
	}
	who := t.Name()
	if t == p {
		who += " (yourself)"
	}
	var b strings.Builder
	b.WriteString(`<html><title>Level</title><body><center>`)
	b.WriteString(`<font color="LEVEL">` + who + `</font><br1>`)
	b.WriteString(`Level: <font color="LEVEL">` + strconv.Itoa(t.Level()) + `</font>`)
	b.WriteString(` (max ` + strconv.Itoa(s.MaxLevel()-1) + `)<br1>`)
	b.WriteString(`Target a player to change their level, else your own.<br>`)
	b.WriteString(`<table width=270><tr>`)
	shown := 0
	for _, level := range adminLevelPresets {
		if level > s.MaxLevel() {
			continue
		}
		if shown > 0 && shown%3 == 0 {
			b.WriteString(`</tr><tr>`)
		}
		v := strconv.Itoa(level)
		b.WriteString(`<td>` + button("Set level "+v, "level_menu set "+v, 85) + `</td>`)
		shown++
	}
	b.WriteString(`</tr></table><br>`)
	b.WriteString(`<table width=270><tr>`)
	b.WriteString(`<td>` + button("Level -1", "level_menu add -1", 85) + `</td>`)
	b.WriteString(`<td>` + button("Level +1", "level_menu add 1", 85) + `</td>`)
	b.WriteString(`</tr></table><br>`)
	b.WriteString(`<table width=270><tr><td>Level</td><td><edit var="level" width=60 height=12></td>`)
	b.WriteString(`<td>` + button("Set", "level_menu set $level", 60) + `</td></tr></table><br>`)
	b.WriteString(button("Give all skills", "level_menu skills", 130) + `<br>`)
	b.WriteString(button("Back", "admin", 65))
	b.WriteString(`</center></body></html>`)
	return b.String()
}
