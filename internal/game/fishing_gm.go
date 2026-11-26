package game

import (
	"errors"
	"strconv"

	"github.com/IPeter/l2go/internal/fishing"
	"github.com/IPeter/l2go/internal/storage"
)

// The fishing GM commands (decision X9; Mobius has none): a week of the
// championship cannot wait for real time.
//
//   - //fishchamp shows the week's end and the ranking of the running week
//     and of last week's winners.
//   - //fishchamp_end ends the running week now, as its Tuesday 19:00 would.

// registerFishingGM registers the commands.
func (s *Server) registerFishingGM() error {
	admin := storage.AccessLevelAdmin
	return errors.Join(
		s.RegisterGMCommand("fishchamp", admin, s.gmFishChamp),
		s.RegisterGMCommand("fishchamp_end", admin, s.gmFishChampEnd),
	)
}

// gmFishChamp is //fishchamp.
func (s *Server) gmFishChamp(p *Player, _ []string) {
	champ := s.fishing.champ
	gmSysMessage(p, "Fishing championship ends "+s.calendar.In(champ.End()).Format("2006-01-02 15:04:05 MST")+
		" ("+strconv.FormatInt(champ.MinutesLeft(s.Now()), 10)+" minutes).")
	running, winners := champ.Ranking()
	show := func(title string, list []fishing.Fisher) {
		gmSysMessage(p, title+": "+strconv.Itoa(len(list)))
		for _, f := range list {
			gmSysMessage(p, f.Name+" "+fishing.JavaDouble(f.Length)+" (reward "+strconv.Itoa(f.Reward)+")")
		}
	}
	show("Running week", running)
	show("Last week's winners", winners)
}

// gmFishChampEnd is //fishchamp_end.
func (s *Server) gmFishChampEnd(p *Player, _ []string) {
	s.finishChampionship()
	gmSysMessage(p, "The fishing championship week ended; the next ends "+
		s.calendar.In(s.fishing.champ.End()).Format("2006-01-02 15:04:05 MST")+".")
}
