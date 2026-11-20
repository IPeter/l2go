package game

import "github.com/IPeter/l2go/internal/stats"

// Kill rewards (decision F9): Mobius Attackable.calculateRewards for
// players without a party. Parties (M9), summons, overhit (M7), champions,
// vitality and premium are left out.

// rewardShare is one player's share of a kill: the damage it did.
type rewardShare struct {
	p      *Player
	damage int64
}

// npcRewards is Attackable.calculateRewards: the drops go to the player
// that did the most damage (or, if it is offline or there is none, to the
// main damage dealer or the killer: Creature.doDie's lastAttacker), then
// every player that did more than 1 damage within the party range gets its
// share of the exp and SP (stats.CombatRules.SoloReward) when it is alive
// and in the corpse's surrounding regions.
func (s *Server) npcRewards(n *Npc, killer Creature) {
	hate := n.ai.hate
	if len(hate) == 0 {
		return
	}
	partyRange := float64(s.combatSettings().PartyRange)
	pos := n.Position()
	var shares []rewardShare
	var total, maxDamage int64
	var maxDealer *Player
	for _, a := range hate {
		p, ok := a.Attacker.(*Player)
		if !ok || p == nil || a.Damage <= 1 {
			continue
		}
		if pos.Distance3D(p.Position()) > partyRange {
			continue
		}
		total += a.Damage
		shares = append(shares, rewardShare{p: p, damage: a.Damage})
		if a.Damage > maxDamage {
			maxDealer, maxDamage = p, a.Damage
		}
	}

	lastAttacker := killer
	if n.IsMonster() {
		if main := s.mainDamageDealer(n); main != nil {
			lastAttacker = main
		}
	}
	if maxDealer != nil && maxDealer.InGame() {
		s.npcDropItems(n, maxDealer)
	} else {
		s.npcDropItems(n, lastAttacker)
	}

	m := stats.MonsterReward{Level: n.Level(), Exp: n.Template.Acquire.Exp, SP: n.Template.Acquire.SP}
	for _, sh := range shares {
		p := sh.p
		if !p.InGame() || p.IsDead() || !inSurroundingRegion(n, p) {
			continue
		}
		level := p.Level()
		s.AddExpAndSP(p, s.combat.SoloReward(p.stats, m, sh.damage, total))
		if p.Level() != level {
			s.RequestSave(p)
		}
	}
}

// mainDamageDealer is Mobius Attackable.getMainDamageDealer: the attacker
// that did the most damage within the party range (the first of equal
// ones), or nil.
func (s *Server) mainDamageDealer(n *Npc) Creature {
	partyRange := float64(s.combatSettings().PartyRange)
	pos := n.Position()
	var dealer Creature
	var damage int64
	for _, a := range n.ai.hate {
		if a.Attacker == nil || a.Damage <= damage {
			continue
		}
		if pos.Distance3D(a.Attacker.Position()) < partyRange {
			dealer, damage = a.Attacker, a.Damage
		}
	}
	return dealer
}
