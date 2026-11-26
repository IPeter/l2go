package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// What the Olympiad changes in other features: who may attack whom
// (Player.isAutoAttackable), blessed spiritshots in a match
// (BlessedSpiritShot), and who may stay in a stadium (OlympiadStadiumZone).

// sysMsgNoItemInOlympiad is "You cannot use that item in a Grand Olympiad
// Games match.".
const sysMsgNoItemInOlympiad serverpackets.SystemMessageID = 1508

// olympiadAttack is the Olympiad's part of Mobius isAutoAttackable for a
// player attacker: ok is false when the Olympiad does not decide; then
// attackable says whether a fighter may hit p (both fighting in the same
// started match).
func (s *Server) olympiadAttack(p, attacker *Player) (attackable, ok bool) {
	sa := s.olyOf(attacker)
	if sa == nil || !sa.inMode {
		return false, false
	}
	sp := s.olyOf(p)
	return sp != nil && sp.inMode && sp.start && sa.gameID == sp.gameID, true
}

// olympiadBlocksBlessedShots refuses a blessed spiritshot in a match
// (OlympiadDisableBlessedSpiritShots).
func (s *Server) olympiadBlocksBlessedShots(p *Player) bool {
	if !p.InOlympiadMode() || !s.olympiad.cfg.DisableBlessedSpiritShots {
		return false
	}
	p.SendSystemMessage(sysMsgNoItemInOlympiad)
	return true
}

// inOlympiadStadium is Mobius playerInStadia: p is inside an olympiad
// stadium zone.
func (s *Server) inOlympiadStadium(p *Player) bool {
	return s.zoneAt(p.Position(), "OlympiadStadiumZone") != nil
}

// olympiadZoneEntered is OlympiadStadiumZone.onEnter for a player: only
// fighters, observers and GMs may be there; anyone else goes to the
// nearest town.
func (s *Server) olympiadZoneEntered(p *Player, zoneType string) {
	if zoneType != "OlympiadStadiumZone" || p.IsGM() || p.InOlympiadMode() || p.InObserverMode() {
		return
	}
	if sm := s.SummonOf(p); sm != nil {
		s.Unsummon(p)
	}
	if loc, ok := s.restartLocation(p); ok {
		p.after(0, func() { s.teleportAround(p, loc, maxOffsetOnTeleport) })
	}
}
