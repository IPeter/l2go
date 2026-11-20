package game

import "github.com/IPeter/l2go/internal/world"

// Peace zones and the click paths that start an attack (decision F13).
// Peace and no-PvP zones block only a playable attacking a playable;
// monsters fight in towns, and GMs (Mobius allowPeaceAttack) attack
// anywhere.

// insidePeaceZone is Mobius Creature.isInsidePeaceZone(attacker, target) for
// a player attacker: false for a GM; otherwise, between two players, true
// when either stands in a peace or no-PvP zone. With
// KarmaPlayerCanBeKilledInPeaceZone a target with karma, or a flagged
// target of an attacker with karma, may be attacked there.
func (s *Server) insidePeaceZone(p *Player, target Creature) bool {
	if p.IsGM() {
		return false
	}
	tp, ok := target.(*Player)
	if !ok {
		return false
	}
	if s.gs.PvP.KarmaPlayerCanBeKilledInPeaceZone {
		if tp.Karma() > 0 || (p.Karma() > 0 && tp.PvPFlag > 0) {
			return false
		}
	}
	return tp.Zones.Inside(ZonePeace) || p.Zones.Inside(ZonePeace) ||
		tp.Zones.Inside(ZoneNoPvP) || p.Zones.Inside(ZoneNoPvP)
}

// onForcedAttack is Mobius onForcedAttack, an AttackRequest (Ctrl+click) on
// p's target o: a creature outside the peace zone rules becomes p's attack
// target; anything else gets ActionFailed. NPCs can all be attacked this way
// (Mobius AltAttackableNpcs, on by default).
func (s *Server) onForcedAttack(p *Player, o world.Object) {
	t, ok := o.(Creature)
	if !ok {
		p.SendActionFailed()
		return
	}
	if s.insidePeaceZone(p, t) {
		p.SendSystemMessage(sysMsgPeacefulZone)
		p.SendActionFailed()
		return
	}
	s.playerIntendAttack(p, t)
}
