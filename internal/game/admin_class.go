package game

import (
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
)

// The class page of the admin panel (an L2Go page, not in Mobius): one-click
// buttons that change the class of the GM's player target, or of the GM
// when no player is targeted (as //setclass, then RewardSkills), and turn
// its noblesse (//setnoble) and hero status (//sethero) on or off, then show
// the page again with the new state. //admin's main page gets a button to it.
func init() { registerFeature("admin_class", setupAdminClass) }

// adminClassRaces are the race groups of the "All classes" list, in order.
var adminClassRaces = []struct {
	race data.Race
	name string
}{
	{data.RaceHuman, "Human"},
	{data.RaceElf, "Elf"},
	{data.RaceDarkElf, "Dark Elf"},
	{data.RaceOrc, "Orc"},
	{data.RaceDwarf, "Dwarf"},
}

func setupAdminClass(s *Server) error {
	return s.RegisterGMCommand("class_menu", storage.AccessLevelAdmin, s.gmAdminClassMenu)
}

// gmAdminClassMenu is //class_menu: the class page. //class_menu set <id>
// changes the class of the GM's player target, or of the GM without one;
// //class_menu noble on|off and //class_menu hero on|off set its noblesse
// and hero status; //class_menu all [race] shows every class of a race
// (Human first) instead of the class page.
func (s *Server) gmAdminClassMenu(p *Player, args []string) {
	t, ok := gmTargetPlayer(p)
	if !ok {
		t = p
	}
	if len(args) > 0 {
		on := len(args) > 1 && args[1] == "on"
		switch args[0] {
		case "set":
			if len(args) < 2 {
				break
			}
			id, err := strconv.Atoi(args[1])
			if err != nil {
				break
			}
			if _, ok := s.data.ClassTemplate(id); !ok {
				p.SendMessage("No class " + args[1] + ".")
				break
			}
			s.ChangeClass(t, id)
			s.RewardSkills(t)
		case "noble":
			if t.IsNoble() != on {
				t.SetNoble(on)
				t.SendMessage("GM changed your nobless status!")
			}
		case "hero":
			if t.IsHero() != on {
				s.setHero(t, on)
				t.BroadcastUserInfo()
			}
		case "all":
			race := 0
			if len(args) > 1 {
				race, _ = strconv.Atoi(args[1])
			}
			s.sendPage(p, newPage(0, s.adminAllClassesPage(p, t, race)))
			return
		}
	}
	s.sendPage(p, newPage(0, s.adminClassPage(p, t)))
}

// adminClassButton is an admin panel button running //action.
func adminClassButton(value, action string, width int) string {
	return `<button value="` + value + `" action="bypass -h admin_` + action + `" width=` +
		strconv.Itoa(width) + ` height=21 back="sek.cbui94" fore="sek.cbui92">`
}

// adminClassName is the name of the class id, such as "Human Fighter".
func (s *Server) adminClassName(id int) string {
	if c, ok := s.data.ClassTemplate(id); ok && c.Name != "" {
		return c.Name
	}
	return data.PlayerClass(id).String()
}

// adminClassHeader is the start of the class pages: who, class and level.
func (s *Server) adminClassHeader(b *strings.Builder, title string, p, t *Player) {
	who := t.Name()
	if t == p {
		who += " (yourself)"
	}
	b.WriteString(`<html><title>` + title + `</title><body><center>`)
	b.WriteString(`<font color="LEVEL">` + who + `</font><br1>`)
	b.WriteString(`Class: <font color="LEVEL">` + s.adminClassName(t.ClassID()) + `</font><br1>`)
	b.WriteString(`Level: <font color="LEVEL">` + strconv.Itoa(t.Level()) + `</font><br1>`)
}

// adminClassButtons writes a two-column table of class buttons.
func (s *Server) adminClassButtons(b *strings.Builder, classes []data.PlayerClass) {
	b.WriteString(`<table width=270><tr>`)
	for i, c := range classes {
		if i > 0 && i%2 == 0 {
			b.WriteString(`</tr><tr>`)
		}
		id := strconv.Itoa(int(c))
		b.WriteString(`<td>` + adminClassButton(s.adminClassName(int(c)), "class_menu set "+id, 130) + `</td>`)
	}
	b.WriteString(`</tr></table>`)
}

// adminClassPage is the class page for the GM p acting on t.
func (s *Server) adminClassPage(p, t *Player) string {
	var b strings.Builder
	s.adminClassHeader(&b, "Class", p, t)
	b.WriteString(`Target a player to change their class, else your own.<br>`)
	var next []data.PlayerClass
	for _, c := range data.PlayerClasses() {
		if parent, ok := c.Parent(); ok && int(parent) == t.ClassID() {
			if _, ok := s.data.ClassTemplate(int(c)); ok {
				next = append(next, c)
			}
		}
	}
	if len(next) == 0 {
		b.WriteString(`No next class.<br>`)
	} else {
		b.WriteString(`Next class:<br1>`)
		s.adminClassButtons(&b, next)
	}
	b.WriteString(`<br>` + adminClassButton("All classes", "class_menu all", 130) + `<br>`)
	onOff := func(label, cmd string, state bool) {
		st := "No"
		if state {
			st = "Yes"
		}
		b.WriteString(`<table width=270><tr><td width=110>` + label + `: <font color="LEVEL">` + st + `</font></td>`)
		b.WriteString(`<td>` + adminClassButton("On", "class_menu "+cmd+" on", 60) + `</td>`)
		b.WriteString(`<td>` + adminClassButton("Off", "class_menu "+cmd+" off", 60) + `</td></tr></table>`)
	}
	onOff("Noble", "noble", t.IsNoble())
	onOff("Hero", "hero", t.IsHero())
	b.WriteString(`<br>` + adminClassButton("Back", "admin", 65))
	b.WriteString(`</center></body></html>`)
	return b.String()
}

// adminAllClassesPage lists every class of one race group (an index into
// adminClassRaces), with a button row for the other groups, so each page
// stays well under the client's page size.
func (s *Server) adminAllClassesPage(p, t *Player, race int) string {
	if race < 0 || race >= len(adminClassRaces) {
		race = 0
	}
	var b strings.Builder
	s.adminClassHeader(&b, "All classes", p, t)
	b.WriteString(`<br><table width=270><tr>`)
	for i, r := range adminClassRaces {
		if i == 3 {
			b.WriteString(`</tr><tr>`)
		}
		b.WriteString(`<td>` + adminClassButton(r.name, "class_menu all "+strconv.Itoa(i), 85) + `</td>`)
	}
	b.WriteString(`</tr></table><br>`)
	b.WriteString(`<font color="LEVEL">` + adminClassRaces[race].name + `</font><br1>`)
	var classes []data.PlayerClass
	for _, c := range data.PlayerClasses() {
		if c.Race() != adminClassRaces[race].race {
			continue
		}
		if _, ok := s.data.ClassTemplate(int(c)); ok {
			classes = append(classes, c)
		}
	}
	s.adminClassButtons(&b, classes)
	b.WriteString(`<br>` + adminClassButton("Back", "class_menu", 65))
	b.WriteString(`</center></body></html>`)
	return b.String()
}
