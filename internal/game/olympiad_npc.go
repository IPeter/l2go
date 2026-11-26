package game

import (
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The Grand Olympiad Managers and the Monuments of Heroes (Mobius
// entity/actor/instance/OlympiadManager, both of that type): a noble sees
// noble_main.htm, a hero or a noble at a monument hero_main.htm (Mobius
// Npc.showChatWindow); their links are "OlympiadDesc", "OlympiadNoble" and
// "Olympiad" commands. The monuments' hero claim and items are the Lua
// script MonumentOfHeroes.

// The Olympiad NPCs.
const (
	olympiadManagerNpc = 31688 // Grand Olympiad Manager
	olympiadHTML       = "olympiad/"
)

// monumentNpcs are the Monuments of Heroes.
var monumentNpcs = []int{31690, 31769, 31770, 31771, 31772}

// olympiadFirstPage is the page of Npc.showChatWindow's switch for the
// Olympiad NPCs, "" for another NPC or a player who sees the NPC's own.
func olympiadFirstPage(p *Player, npcID int) string {
	switch {
	case npcID == olympiadManagerNpc && p.IsNoble():
		return olympiadHTML + "noble_main.htm"
	case slices.Contains(monumentNpcs, npcID) && (p.IsHero() || p.IsNoble()):
		return olympiadHTML + "hero_main.htm"
	}
	return ""
}

// olympiadPage sends an Olympiad page of n with %objectId% filled.
func (s *Server) olympiadPage(p *Player, n *Npc, file string) *htmlPage {
	return s.pageFile(p, n.ID(), olympiadHTML+file).replaceObjectID(n.ID())
}

// olympiadManagerCommand runs the OlympiadManager's own bypasses (Mobius
// OlympiadManager.onBypassFeedback); false hands the command on.
func (s *Server) olympiadManagerCommand(p *Player, n *Npc, command string) bool {
	switch {
	case strings.HasPrefix(command, "OlympiadDesc"):
		// "OlympiadDesc 2a": the page noble_desc2a.htm; 0 is the main page.
		rest := strings.TrimSpace(strings.TrimPrefix(command, "OlympiadDesc"))
		file := "noble_desc" + rest + ".htm"
		if rest == "0" {
			file = "noble_main.htm"
		}
		s.sendPage(p, s.olympiadPage(p, n, file))
	case strings.HasPrefix(command, "OlympiadNoble"):
		s.olympiadNobleCommand(p, n, strings.TrimSpace(strings.TrimPrefix(command, "OlympiadNoble")))
	case strings.HasPrefix(command, "Olympiad"):
		s.olympiadInfoCommand(p, n, strings.TrimSpace(strings.TrimPrefix(command, "Olympiad")))
	default:
		return false
	}
	return true
}

// olympiadNobleCommand is "OlympiadNoble <n>": the noble's own matters.
// A subclass, a player who is no noble or not in its third class get their
// pages instead.
func (s *Server) olympiadNobleCommand(p *Player, n *Npc, arg string) {
	f := &s.olympiad
	if p.IsSubClassActive() {
		s.sendPage(p, s.olympiadPage(p, n, "noble_desc5.htm"))
		return
	}
	if !p.IsNoble() || classLevelOf(p.ClassID()) < 3 {
		s.sendPage(p, s.olympiadPage(p, n, "noble_desc6.htm"))
		return
	}
	val, err := strconv.Atoi(arg)
	if err != nil {
		return
	}
	switch val {
	case 1:
		s.unregisterNoble(p)
	case 2:
		classed, nonClassed := 0, 0
		if f.inComp {
			classed, nonClassed = f.queues.Counts()
		}
		pg := s.olympiadPage(p, n, "noble_registered.htm")
		if d := f.cfg.RegDisplay; d > 0 {
			fewer, more := "Fewer than "+strconv.Itoa(d), "More than "+strconv.Itoa(d)
			pick := func(c int) string {
				if c < d {
					return fewer
				}
				return more
			}
			pg.replace("%listClassed%", pick(classed)).replace("%listNonClassedTeam%", fewer).replace("%listNonClassed%", pick(nonClassed))
		} else {
			pg.replace("%listClassed%", strconv.Itoa(classed)).replace("%listNonClassedTeam%", "0").replace("%listNonClassed%", strconv.Itoa(nonClassed))
		}
		s.sendPage(p, pg)
	case 3:
		s.sendPage(p, s.olympiadPage(p, n, "noble_points1.htm").replace("%points%", strconv.Itoa(f.nobles.Points(p.ObjectID()))))
	case 4:
		s.registerNoble(p, false)
	case 5:
		s.registerNoble(p, true)
	case 6:
		if f.unclaimed[p.ObjectID()] > 0 {
			s.sendPage(p, s.olympiadPage(p, n, "noble_settle.htm"))
		} else {
			s.sendPage(p, s.olympiadPage(p, n, "noble_nopoints.htm"))
		}
	case 7:
		s.OpenMultisell(p, n, 102, false)
	case 8:
		s.OpenMultisell(p, n, 103, false)
	case 9:
		s.sendPage(p, s.olympiadPage(p, n, "noble_points2.htm").replace("%points%", strconv.Itoa(f.nobles.LastPoints(p.ObjectID()))))
	case 10:
		s.settleOlympiadPasses(p)
	default:
		s.log.Warn("game: unknown OlympiadNoble request", append(p.logArgs(), "value", val)...)
	}
}

// settleOlympiadPasses is "OlympiadNoble 10": the passes the noble earned
// come as CompRewardItem, × GPPerPoint again as Mobius does.
func (s *Server) settleOlympiadPasses(p *Player) {
	f := &s.olympiad
	passes := f.unclaimed[p.ObjectID()]
	if passes <= 0 {
		return
	}
	delete(f.unclaimed, p.ObjectID())
	if _, err := s.AddItem(p, int32(f.cfg.CompRewardItem), passes*int64(f.cfg.GPPerPoint), -1, true, false); err != nil { //nolint:gosec // an item ID
		s.log.Warn("game: cannot settle the olympiad passes", append(p.logArgs(), "passes", passes, "err", err)...)
	}
	s.RequestSave(p)
	s.saveOlympiad()
}

// olympiadInfoCommand is "Olympiad <n>": the arena list (1), a class
// ranking (2_<class>), watching an arena (3_<arena>), the hero list (4).
func (s *Server) olympiadInfoCommand(p *Player, n *Npc, arg string) {
	f := &s.olympiad
	if arg == "" {
		return
	}
	val, err := strconv.Atoi(arg[:1])
	if err != nil {
		return
	}
	param := ""
	if len(arg) > 2 {
		param = arg[2:]
	}
	switch val {
	case 1:
		pg := s.olympiadPage(p, n, "olympiad_observe1.htm")
		titles := s.olympiadTitles()
		for i := range len(f.stadiums) {
			title, ok := titles[i]
			if !ok {
				title = `\&$906;`
			}
			pg.replace("%title"+strconv.Itoa(i+1)+"%", title)
		}
		s.sendPage(p, pg)
	case 2:
		class, err := strconv.Atoi(param)
		if err != nil || !((class >= 88 && class <= 118) || (class >= 131 && class <= 134) || class == 136) {
			return
		}
		pg := s.olympiadPage(p, n, "olympiad_ranking.htm")
		names := f.nobles.ClassLeaders(class, f.cfg.ShowMonthlyWinners)
		for i := 1; i <= 10; i++ {
			place, rank := "", ""
			if i <= len(names) {
				place, rank = strconv.Itoa(i), names[i-1]
			}
			pg.replace("%place"+strconv.Itoa(i)+"%", place).replace("%rank"+strconv.Itoa(i)+"%", rank)
		}
		s.sendPage(p, pg)
	case 3:
		id, err := strconv.Atoi(param)
		if err != nil {
			return
		}
		s.addSpectator(id, p, true)
	case 4:
		p.Send(s.heroListPacket()) // hero.go
	default:
		s.log.Warn("game: unknown Olympiad request", append(p.logArgs(), "value", val)...)
	}
}

// classLevelOf is Mobius PlayerClass.level(): 0 for a starting class, 3 for
// a third one.
func classLevelOf(classID int) int { return data.PlayerClass(classID).Level() }
