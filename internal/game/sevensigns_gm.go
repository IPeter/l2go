package game

import (
	"errors"
	"strconv"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/sevensigns"
	"github.com/IPeter/l2go/internal/storage"
)

// The Seven Signs GM commands: Mobius AdminMammon (//mammon_find,
// //mammon_respawn) and, an L2Go addition (decision X9), //ssq_change, which
// runs the next period change now, and //ssq_info.

func (s *Server) registerSevenSignsGM() error {
	add := func(name string, fn GMCommand) error {
		return s.RegisterGMCommand(name, storage.AccessLevelAdmin, fn)
	}
	return errors.Join(
		add("mammon_find", s.gmMammonFind),
		add("mammon_respawn", s.gmMammonRespawn),
		add("ssq_change", func(p *Player, _ []string) {
			s.changeSevenSignsPeriodNow()
			s.gmSevenSignsInfo(p, nil)
		}),
		add("ssq_info", s.gmSevenSignsInfo),
	)
}

// gmMammonFind is //mammon_find [1|2]: where the Blacksmith and the
// Merchant of Mammon stand, and a teleport to one (1 the blacksmith, 2 the
// merchant). Mobius reads the period when the handler loads; here the
// period now.
func (s *Server) gmMammonFind(p *Player, args []string) {
	idx := -1
	if len(args) > 0 {
		v, err := strconv.Atoi(args[0])
		if err != nil {
			p.SendMessage("Usage: //mammon_find [teleportIndex] (where 1 = Blacksmith, 2 = Merchant)")
			return
		}
		idx = v
	}
	if !s.sevenSignsSealValidation() {
		p.SendSystemMessage(serverpackets.SystemMessageID(sevensigns.MsgQuestEventPeriod))
		return
	}
	for i, m := range []struct {
		id   int
		name string
	}{{sevensigns.MammonBlacksmithID, "Blacksmith of Mammon"}, {sevensigns.MammonMerchantID, "Merchant of Mammon"}} {
		if len(s.autoSpawnGroups(m.id)) == 0 {
			p.SendMessage(m.name + " isn't registered for spawn.")
			continue
		}
		n := s.autoSpawnNpc(m.id)
		if n == nil {
			continue
		}
		pos := n.Position()
		p.SendMessage(m.name + ": " + strconv.Itoa(int(pos.X)) + " " + strconv.Itoa(int(pos.Y)) + " " + strconv.Itoa(int(pos.Z)))
		if idx == i+1 {
			if err := s.Teleport(p, pos); err != nil {
				s.log.Warn("game: //mammon_find teleport failed", append(p.logArgs(), "err", err)...)
			}
		}
	}
}

// gmMammonRespawn is //mammon_respawn: when the Merchant and the Blacksmith
// of Mammon move next.
func (s *Server) gmMammonRespawn(p *Player, _ []string) {
	if !s.sevenSignsSealValidation() {
		p.SendSystemMessage(serverpackets.SystemMessageID(sevensigns.MsgQuestEventPeriod))
		return
	}
	for _, m := range []struct {
		id   int
		name string
	}{{sevensigns.MammonMerchantID, "Merchant of Mammon"}, {sevensigns.MammonBlacksmithID, "Blacksmith of Mammon"}} {
		groups := s.autoSpawnGroups(m.id)
		if len(groups) == 0 {
			p.SendMessage(m.name + " isn't registered for spawn.")
			continue
		}
		var left int64
		if g := groups[0]; g.timer != 0 {
			left = g.nextAt.Sub(s.Now()).Milliseconds()
		}
		p.SendMessage("The " + m.name + " will respawn in " + strconv.FormatInt(left/60000, 10) + " minute(s).")
	}
}

// gmSevenSignsInfo is //ssq_info: the period, the scores and the seals.
func (s *Server) gmSevenSignsInfo(p *Player, _ []string) {
	st := s.sevenSigns.state
	if st == nil {
		p.SendMessage("The Seven Signs are not loaded.")
		return
	}
	p.SendMessage("Seven Signs: cycle " + strconv.Itoa(st.Cycle) + ", " + st.Period.Name() + " until " + s.Calendar().In(s.sevenSigns.next).Format("2006-01-02 15:04:05"))
	p.SendMessage("Dawn " + strconv.Itoa(st.CurrentScore(sevensigns.CabalDawn)) + ", Dusk " + strconv.Itoa(st.CurrentScore(sevensigns.CabalDusk)) +
		"; leader: " + st.HighestCabal().Name())
	for _, seal := range sevensigns.Seals {
		p.SendMessage(seal.Name(false) + ": " + st.SealOwner(seal).Name())
	}
}
