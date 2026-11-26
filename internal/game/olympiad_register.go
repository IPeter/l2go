package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/olympiad"
)

// Registration for the Olympiad (Mobius Olympiad.registerNoble,
// unRegisterNoble, removeDisconnectedCompetitor) and the waiting lists.

// sendOlympiadRefusal tells p why the Olympiad refused it.
func (s *Server) sendOlympiadRefusal(p *Player, r olympiad.Refusal) {
	switch {
	case r.Text != "":
		p.SendMessage(r.Text)
	case r.Item != 0:
		p.SendSystemMessage(serverpackets.SystemMessageID(r.Msg), s.itemNameParam(int32(r.Item))) //nolint:gosec // an item ID
	case r.Msg != 0:
		p.SendSystemMessage(serverpackets.SystemMessageID(r.Msg))
	}
}

// inOlympiadGame reports whether p was picked for a match that has not
// ended (Mobius: _playerOneID or _playerTwoID of a game).
func (s *Server) inOlympiadGame(charID int32) bool {
	for _, g := range s.olympiad.games {
		if g != nil && (g.ids[0] == charID || g.ids[1] == charID) {
			return true
		}
	}
	return false
}

// olympiadRegistered is Mobius isRegistered: p waits on a list.
func (s *Server) olympiadRegistered(p *Player) bool {
	return s.olympiad.queues.IsRegistered(p.ClassID(), p.ObjectID())
}

// olympiadRegisteredInComp is Mobius isRegisteredInComp: p waits, or (in
// the competition) was picked for a match.
func (s *Server) olympiadRegisteredInComp(p *Player) bool {
	return s.olympiadRegistered(p) || (s.olympiad.inComp && s.inOlympiadGame(p.ObjectID()))
}

// registerNoble is Mobius Olympiad.registerNoble for p, classed or not.
func (s *Server) registerNoble(p *Player, classed bool) bool {
	f := &s.olympiad
	a := olympiad.Applicant{
		CharID: p.ObjectID(), Name: p.Name(), ClassID: p.ClassID(),
		Noble: p.IsNoble(), OnBaseClass: p.BaseClassID() == p.ClassID(),
		CursedWeapon:  int(p.CursedWeaponEquippedID()),
		InventorySize: p.inv.Len(), InventoryLimit: int(p.stats.View().InventoryLimit),
	}
	before := f.nobles.Len()
	msg, refused := olympiad.Register(f.cfg, f.nobles, &f.queues, a, classed, f.inComp,
		f.inComp && s.inOlympiadGame(p.ObjectID()), s.untilCompEnd())
	if f.nobles.Len() != before {
		s.saveOlympiad() // a new noble of the period
	}
	if refused.Refused() {
		s.sendOlympiadRefusal(p, refused)
		return false
	}
	p.SendSystemMessage(serverpackets.SystemMessageID(msg))
	return true
}

// unregisterNoble is Mobius Olympiad.unRegisterNoble for p.
func (s *Server) unregisterNoble(p *Player) bool {
	f := &s.olympiad
	msg, refused := olympiad.Unregister(&f.queues, p.ClassID(), p.ObjectID(), p.IsNoble(), f.inComp, s.inOlympiadGame(p.ObjectID()))
	if refused.Refused() {
		s.sendOlympiadRefusal(p, refused)
		return false
	}
	p.SendSystemMessage(serverpackets.SystemMessageID(msg))
	return true
}

// olympiadLeaveForSubclass is the subclass hook of the village master's
// pages (Mobius VillageMaster: unRegisterNoble when isRegisteredInComp).
func (s *Server) olympiadLeaveForSubclass(p *Player) {
	if s.olympiadRegisteredInComp(p) {
		s.unregisterNoble(p)
	}
}

// olympiadLeave is the leave hook: a waiting noble leaves its list, a
// fighter's game learns it is gone (Mobius Logout unRegisterNoble, then
// deleteMe removeDisconnectedCompetitor), and a spectator leaves its
// stadium.
func (s *Server) olympiadLeave(p *Player) {
	f := &s.olympiad
	st := s.olyOf(p)
	if st != nil && st.observer {
		s.removeSpectator(st.gameID, p) // olympiad_observe.go
	}
	if st != nil && st.gameID >= 0 && !st.observer {
		if g := f.games[st.gameID]; g != nil {
			g.handleDisconnect(p)
		}
	}
	f.queues.Remove(p.ClassID(), p.ObjectID())
	delete(f.players, p)
}
