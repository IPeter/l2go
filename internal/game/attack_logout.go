package game

import "github.com/IPeter/l2go/internal/gameserver/serverpackets"

// No leaving in combat (decision F14): restart and logout are refused while
// the player is in its combat stance (Mobius RequestRestart, Logout and
// Player.canLogout), and a lost connection leaves the player in the world
// for 15 s more (Mobius Disconnection). GMs may leave at any time (Mobius
// GMRestartFighting, on by default).

// inCombat reports whether p may not leave now (Mobius hasAttackStanceTask
// for a non-GM).
func (s *Server) inCombat(p *Player) bool { return !p.IsGM() && s.InAttackStance(p) }

// refuseRestart refuses RequestRestart in combat: "You cannot restart while
// in combat.", RestartResponse(false) and ActionFailed. It reports whether
// it refused.
func (s *Server) refuseRestart(p *Player) bool {
	if !s.inCombat(p) {
		return false
	}
	p.SendSystemMessage(serverpackets.SysMsgCannotRestartInCombat)
	p.Send(serverpackets.NewRestartResponsePacket(false))
	p.SendActionFailed()
	return true
}

// refuseLogout refuses Logout in combat: "You cannot exit while in combat."
// and ActionFailed. It reports whether it refused.
func (s *Server) refuseLogout(p *Player) bool {
	if !s.inCombat(p) {
		return false
	}
	p.SendSystemMessage(serverpackets.SysMsgCannotExitInCombat)
	p.SendActionFailed()
	return true
}

// lingerInCombat keeps the player of a lost connection in the world for
// attackStanceTime when it is in combat (Mobius Disconnection: it leaves 15
// s later unless it left meanwhile). The session is forgotten at once, so a
// new login of the account kicks the old player as usual. It reports whether
// the player stays.
func (s *Server) lingerInCombat(p *Player) bool {
	if !s.inCombat(p) {
		return false
	}
	if sess := p.session; sess != nil && s.bySession[sess] == p {
		delete(s.bySession, sess)
	}
	p.session = nil
	s.loop.After(attackStanceTime, func() {
		if p.state != stateLeft {
			s.log.Info("player left the world after its combat", p.logArgs()...)
			s.leave(p, true, nil)
		}
	})
	return true
}
