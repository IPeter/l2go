package game

import (
	"slices"

	"github.com/IPeter/l2go/internal/stats"
)

// Kill rewards (decision F9): Mobius Attackable.calculateRewards, the
// party branch in party_reward.go (Plan 12 T8, T9). A summon's damage counts
// as its owner's (M11f); overhit (M7), champions, vitality and premium are
// left out.

// rewardShare is one player's share of a kill: the damage it did.
type rewardShare struct {
	p      *Player
	damage int64
}

// npcRewards is Attackable.calculateRewards: the drops go to the leader of
// the party whose members did the most damage when that beats every single
// player, else to the player that did the most damage (or, if it is offline
// or there is none, to the main damage dealer or the killer: Creature.doDie's
// lastAttacker). Then every player that did more than 1 damage within the
// party range gets its share of the exp and SP: alone
// (stats.CombatRules.SoloReward) when it is alive and in the corpse's
// surrounding regions, in a party through the party's share
// (partyKillRewards), once per party.
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
		// A summon's damage counts as its owner's (Mobius asPlayer, summon.go).
		p := s.actingPlayer(a.Attacker)
		if p == nil || a.Damage <= 1 {
			continue
		}
		if pos.Distance3D(p.Position()) > partyRange {
			continue
		}
		total += a.Damage
		i := slices.IndexFunc(shares, func(sh rewardShare) bool { return sh.p == p })
		if i < 0 {
			i = len(shares)
			shares = append(shares, rewardShare{p: p})
		}
		shares[i].damage += a.Damage
		if shares[i].damage > maxDamage {
			maxDealer, maxDamage = p, shares[i].damage
		}
	}

	lastAttacker := killer
	if n.IsMonster() {
		if main := s.mainDamageDealer(n); main != nil {
			lastAttacker = main
		}
	}
	if pt, damage := s.mostDamageParty(n); pt != nil && damage > maxDamage {
		s.npcDropItems(n, pt.Leader())
		s.extraDrops(n, pt.Leader()) // npc_reward_extra.go: Mobius doEventDrop
	} else {
		if maxDealer != nil && maxDealer.InGame() {
			s.npcDropItems(n, maxDealer)
		} else {
			s.npcDropItems(n, lastAttacker)
		}
		s.extraDrops(n, lastAttacker)
	}

	m := stats.MonsterReward{Level: n.Level(), Exp: n.Template.Acquire.Exp, SP: n.Template.Acquire.SP}
	done := make(map[*Player]bool) // rewarded through their party
	for _, sh := range shares {
		p := sh.p
		if done[p] {
			continue
		}
		if pt := p.party.party; pt != nil {
			s.partyKillRewards(n, pt, shares, done, total, m)
			continue
		}
		if !p.InGame() || p.IsDead() || !inSurroundingRegion(n, p) {
			continue
		}
		level := p.Level()
		s.AddExpAndSP(p, s.servitorExpPenalty(p, s.combat.SoloReward(p.stats, m, sh.damage, total))) // summon.go
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
