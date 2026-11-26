package game

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// The Olympiad's GM commands (Mobius AdminAdmin): //checkolympiad shows
// the period's end and the next weekly change, //saveolymp saves the
// Olympiad, //endolympiad ends the period now and makes the heroes,
// //givehero lets an unclaimed hero (the target, or the GM) claim the
// status. //sethero is in admin_char.go.

// registerOlympiadGM registers the commands.
func (s *Server) registerOlympiadGM() error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("checkolympiad", admin, s.gmCheckOlympiad),
		s.RegisterGMCommand("saveolymp", admin, s.gmSaveOlympiad),
		s.RegisterGMCommand("endolympiad", admin, s.gmEndOlympiad),
		s.RegisterGMCommand("givehero", admin, s.gmGiveHero),
	)
}

// olympiadDateTime is Mobius TimeUtil.getDateTimeString: dd/MM/yyyy
// HH:mm:ss in the server's zone.
func (s *Server) olympiadDateTime(t time.Time) string {
	return s.calendar.In(t).Format("02/01/2006 15:04:05")
}

// secondsRe and millisRe are what //checkolympiad cuts from its durations.
var (
	secondsRe = regexp.MustCompile(`,? \d+ seconds?`)
	millisRe  = regexp.MustCompile(`,? \d+ milliseconds?`)
)

// gmCheckOlympiad is //checkolympiad: the current time, the period's end
// and the next weekly change with the time left.
func (s *Server) gmCheckOlympiad(p *Player, _ []string) {
	f := &s.olympiad
	now := s.Now()
	left := func(t time.Time) string {
		d := formatDuration(t.Sub(now).Milliseconds())
		return millisRe.ReplaceAllString(secondsRe.ReplaceAllString(d, ""), "")
	}
	weekly := now.Add(f.state.UntilWeeklyChange(now))
	var b strings.Builder
	b.WriteString(`<html><title>Olympiad Period</title><body>`)
	b.WriteString(`<center><table width=280>`)
	b.WriteString(`<tr><td width=60><button value="Main" action="bypass admin_admin" width=60 height=15 back="sek.cbui94" fore="sek.cbui92"></td>`)
	b.WriteString(`<td width=150><center>Olympiad Info</center></td>`)
	b.WriteString(`<td width=60><button value="Back" action="bypass admin_admin2" width=60 height=15 back="sek.cbui94" fore="sek.cbui92"></td></tr>`)
	b.WriteString(`</table></center><br>`)
	b.WriteString(`<center><table width=280 bgcolor="000000">`)
	b.WriteString(`<tr><td width=120>Current Time:</td><td width=140><font color="LEVEL">` + s.olympiadDateTime(now))
	b.WriteString(`</font></td></tr></table></center><br><br>`)
	b.WriteString(`<center><table width=280 bgcolor="000000">`)
	b.WriteString(`<tr><td width=120>Olympiad Period Ends:</td><td width=120><font color="LEVEL">` + s.olympiadDateTime(f.state.OlympiadEnd))
	b.WriteString(`</font></td></tr>`)
	b.WriteString(`<tr><td width=120>Time Left:</td><td width=120><font color="LEVEL">` + left(f.state.OlympiadEnd))
	b.WriteString(`</font></td></tr></table></center><br>`)
	b.WriteString(`<center><table width=280 bgcolor="000000">`)
	b.WriteString(`<tr><td width=120>Next Weekly Change:</td><td width=120><font color="LEVEL">` + s.olympiadDateTime(weekly))
	b.WriteString(`</font></td></tr>`)
	b.WriteString(`<tr><td width=120>Time Left:</td><td width=120><font color="LEVEL">` + left(weekly))
	b.WriteString(`</font></td></tr></table></center><br>`)
	b.WriteString(`</body></html>`)
	s.sendPage(p, newPage(0, b.String()))
}

// gmSaveOlympiad is //saveolymp.
func (s *Server) gmSaveOlympiad(p *Player, _ []string) {
	s.saveOlympiad()
	p.SendMessage("olympiad system saved.")
}

// gmEndOlympiad is //endolympiad (Mobius manualSelectHeroes).
func (s *Server) gmEndOlympiad(p *Player, _ []string) {
	if s.olympiad.cfg.Enabled {
		s.endOlympiadNow()
	}
	p.SendMessage("Heroes formed.")
}

// gmGiveHero is //givehero: the target player, or the GM when the target is
// no player, claims its hero status.
func (s *Server) gmGiveHero(p *Player, _ []string) {
	if p.Target == nil {
		p.SendSystemMessage(gmSysMsgInvalidTarget)
		return
	}
	t := p
	if tp, ok := gmTargetPlayer(p); ok {
		t = tp
	}
	f := &s.olympiad
	switch {
	case f.heroes.IsHero(t.ObjectID()):
		p.SendMessage("This player has already claimed the hero status.")
	case !f.heroes.IsUnclaimed(t.ObjectID()):
		p.SendMessage("This player cannot claim the hero status.")
	default:
		s.claimHero(t)
	}
}
