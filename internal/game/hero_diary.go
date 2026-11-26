package game

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/olympiad"
	"github.com/IPeter/l2go/internal/storage"
)

// The heroes' pages of the hero list (Mobius RequestBypassToServer "_match"
// and "_diary", Hero.showHeroFights and showHeroDiary): a hero's fights
// before this month, 20 a page, and its diary with its words, 10 a page.

const (
	heroHistoryPage = "olympiad/herohistory.htm"
	heroDiaryPage   = "olympiad/herodiary.htm"
)

// heroPageArgs reads "?class=<n>&page=<n>" of a hero page bypass.
func heroPageArgs(command string) (class, page int, ok bool) {
	_, params, found := strings.Cut(command, "?")
	if !found {
		return 0, 0, false
	}
	parts := strings.Split(params, "&")
	if len(parts) < 2 {
		return 0, 0, false
	}
	val := func(s string) (int, bool) {
		_, v, ok := strings.Cut(s, "=")
		if !ok {
			return 0, false
		}
		n, err := strconv.Atoi(v)
		return n, err == nil
	}
	class, ok1 := val(parts[0])
	page, ok2 := val(parts[1])
	return class, page, ok1 && ok2
}

// heroPageBypass handles "_match?class=<n>&page=<n>" and
// "_diary?class=<n>&page=<n>": the page of the class's current hero.
func (s *Server) heroPageBypass(p *Player, command string) {
	class, page, ok := heroPageArgs(command)
	if !ok {
		return
	}
	id := s.olympiad.heroes.ByClass(class)
	if id <= 0 {
		return
	}
	if strings.HasPrefix(command, "_match") {
		s.showHeroFights(p, class, id, page)
	} else {
		s.showHeroDiary(p, class, id, page)
	}
}

// heroButton is a Prev or Next button of the hero pages.
func heroButton(value, bypass string, class, page int) string {
	return `<button value="` + value + `" action="bypass ` + bypass + `?class=` + strconv.Itoa(class) + `&page=` + strconv.Itoa(page) +
		`" width=65 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2">`
}

// fightTime is Mobius calcFightTime: mm:ss.
func fightTime(ms int64) string {
	sec := ms / 1000
	return fmt.Sprintf("%02d:%02d", (sec%3600)/60, sec%60)
}

// showHeroFights is Mobius showHeroFights.
func (s *Server) showHeroFights(p *Player, class int, charID int32, page int) {
	fights, ok := s.olympiad.fights[charID]
	if !ok {
		return
	}
	content, ok := s.htmlFile(p, heroHistoryPage)
	if !ok {
		return
	}
	pg := newPage(0, content)
	hero, _ := s.olympiad.heroes.Get(charID)
	pg.replace("%heroname%", hero.Name)
	win, loss, draw := 0, 0, 0
	type row struct {
		start, result, opponent, class, time string
		classed                              bool
	}
	var rows []row
	for _, f := range fights {
		r := row{start: s.calendar.In(time.UnixMilli(f.Start)).Format("2006-01-02 15:04"), time: fightTime(f.Time), classed: f.Classed}
		mine := 1
		if f.CharTwoID == charID {
			mine = 2
		}
		if mine == 1 {
			r.opponent, r.class = f.CharTwoName, s.adminClassName(int(f.CharTwoClass))
		} else {
			r.opponent, r.class = f.CharOneName, s.adminClassName(int(f.CharOneClass))
		}
		switch {
		case f.Winner == 0:
			r.result = `<font color="ffff00">draw</font>`
			draw++
		case int(f.Winner) == mine:
			r.result = `<font color="00ff00">victory</font>`
			win++
		default:
			r.result = `<font color="ff0000">loss</font>`
			loss++
		}
		rows = append(rows, r)
	}
	if len(rows) > 0 {
		const perPage = 20
		var b strings.Builder
		color := true
		breakAt := 0
		counter := 0
		for i := max(page-1, 0) * perPage; i < len(rows); i++ {
			breakAt = i
			r := rows[i]
			b.WriteString("<tr><td>")
			if color {
				b.WriteString(`<table width=270 bgcolor="131210">`)
			} else {
				b.WriteString("<table width=270>")
			}
			kind := `<font color="999999">non-cls<font>`
			if r.classed {
				kind = `<font color="FFFF99">cls</font>`
			}
			b.WriteString(`<tr><td width=220><font color="LEVEL">` + r.start + "</font>&nbsp;&nbsp;" + r.result + "</td><td width=50 align=right>" + kind + "</td></tr>")
			b.WriteString("<tr><td width=220>vs " + r.opponent + " (" + r.class + ")</td><td width=50 align=right>(" + r.time + ")</td></tr>")
			b.WriteString("<tr><td colspan=2>&nbsp;</td></tr></table>")
			b.WriteString("</td></tr>")
			color = !color
			counter++
			if counter >= perPage {
				break
			}
		}
		prev, next := "", ""
		if breakAt < len(rows)-1 {
			prev = heroButton("Prev", "_match", class, page+1)
		}
		if page > 1 {
			next = heroButton("Next", "_match", class, page-1)
		}
		pg.replace("%buttprev%", prev).replace("%buttnext%", next).replace("%list%", b.String())
	} else {
		pg.replace("%list%", "").replace("%buttprev%", "").replace("%buttnext%", "")
	}
	pg.replace("%win%", strconv.Itoa(win)).replace("%draw%", strconv.Itoa(draw)).replace("%loos%", strconv.Itoa(loss))
	s.sendPage(p, pg)
}

// heroTags strips the tags of a hero's words (Mobius replaceAll("<.*?>",
// "")).
var heroTags = regexp.MustCompile(`<.*?>`)

// diaryAction is the text of a diary entry (Mobius loadDiary).
func (s *Server) diaryAction(e storage.HeroDiaryEntry) string {
	switch e.Action {
	case olympiad.DiaryRaidKilled:
		if s.worldData != nil {
			if t, ok := s.worldData.Npc(int(e.Param)); ok {
				return t.Name + " was defeated"
			}
		}
	case olympiad.DiaryHeroGained:
		return "Gained Hero status"
	case olympiad.DiaryCastleTaken:
		return "Castle " + strconv.Itoa(int(e.Param)) + " was successfuly taken"
	}
	return ""
}

// showHeroDiary is Mobius showHeroDiary: shown once the hero has words.
func (s *Server) showHeroDiary(p *Player, class int, charID int32, page int) {
	entries, ok := s.olympiad.diary[charID]
	if !ok {
		return
	}
	hero, ok := s.olympiad.heroes.Get(charID)
	if !ok || !hero.Claimed {
		return
	}
	content, ok := s.htmlFile(p, heroDiaryPage)
	if !ok {
		return
	}
	pg := newPage(0, content)
	pg.replace("%heroname%", hero.Name)
	words := heroTags.ReplaceAllString(hero.Message, "")
	words = strings.ReplaceAll(strings.ReplaceAll(words, "<", "&lt;"), ">", "&gt;")
	pg.replace("%message%", words)
	list := slices.Clone(entries)
	slices.Reverse(list)
	if len(list) > 0 {
		const perPage = 10
		var b strings.Builder
		color := true
		breakAt := 0
		counter := 0
		for i := max(page-1, 0) * perPage; i < len(list); i++ {
			breakAt = i
			e := list[i]
			b.WriteString("<tr><td>")
			if color {
				b.WriteString(`<table width=270 bgcolor="131210">`)
			} else {
				b.WriteString("<table width=270>")
			}
			date := s.calendar.In(time.UnixMilli(e.Time)).Format("2006-01-02 15")
			b.WriteString(`<tr><td width=270><font color="LEVEL">` + date + ":xx</font></td></tr>")
			b.WriteString("<tr><td width=270>" + s.diaryAction(e) + "</td></tr>")
			b.WriteString("<tr><td>&nbsp;</td></tr></table>")
			b.WriteString("</td></tr>")
			color = !color
			counter++
			if counter >= perPage {
				break
			}
		}
		prev, next := "", ""
		if breakAt < len(list)-1 {
			prev = heroButton("Prev", "_diary", class, page+1)
		}
		if page > 1 {
			next = heroButton("Next", "_diary", class, page-1)
		}
		pg.replace("%buttprev%", prev).replace("%buttnext%", next).replace("%list%", b.String())
	} else {
		pg.replace("%list%", "").replace("%buttprev%", "").replace("%buttnext%", "")
	}
	s.sendPage(p, pg)
}
