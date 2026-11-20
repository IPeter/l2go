package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/world"
)

// The party's share of a kill and of loot (decisions T8 and T9): Mobius
// Attackable.calculateRewards for parties (584–681), Party.distributeXpAndSp,
// distributeItem and distributeAdena.

// System messages of the party loot.
const (
	partyMsgObtainedCount        serverpackets.SystemMessageID = 299 // "$s1 has obtained $s3 $s2."
	partyMsgObtained             serverpackets.SystemMessageID = 300 // "$s1 has obtained $s2."
	partyMsgObtainedCountSweeper serverpackets.SystemMessageID = 608 // "$s1 has obtained $s3 $s2 by using Sweeper."
	partyMsgObtainedSweeper      serverpackets.SystemMessageID = 609 // "$s1 has obtained $s2 by using Sweeper."
)

// partyRange is AltPartyRange.
func (s *Server) partyRange() int { return s.combatSettings().PartyRange }

// partyInRange is LocationUtil.checkIfInRange(AltPartyRange, a, b, true):
// the same instance and a 3D distance within the range plus both collision
// radii.
func (s *Server) partyInRange(a, b world.Object) bool {
	if a == nil || b == nil || targetInstance(a) != targetInstance(b) {
		return false
	}
	radius := 0
	if c, ok := a.(Creature); ok {
		radius += attackCollision(c)
	}
	if c, ok := b.(Creature); ok {
		radius += attackCollision(c)
	}
	return a.Position().Distance3D(b.Position()) <= float64(s.partyRange()+radius)
}

// mostDamageParty is the damagingParties part of calculateRewards: of the
// parties of the players on n's aggro list, the one whose members did the
// most damage (each member's damage when above 1), the first of equal ones.
func (s *Server) mostDamageParty(n *Npc) (*Party, int64) {
	var best *Party
	var bestDamage int64
	seen := make(map[*Party]bool)
	for _, a := range n.ai.hate {
		p, ok := a.Attacker.(*Player)
		if !ok || p == nil || p.party.party == nil || seen[p.party.party] {
			continue
		}
		pt := p.party.party
		seen[pt] = true
		var damage int64
		for _, m := range pt.members {
			if ai := n.aggroOf(m); ai != nil && ai.Damage > 1 {
				damage += ai.Damage
			}
		}
		if best == nil || damage > bestDamage {
			best, bestDamage = pt, damage
		}
	}
	return best, bestDamage
}

// partyKillRewards is the party branch of calculateRewards for the party of
// a rewarded attacker: its living members within the party range (3D,
// strictly closer) are rewarded, the damage of those on the reward list
// counts (and every member on it leaves the list, done), the party level is
// the highest rewarded level, and the party's exp and SP (for that level
// and the party's damage, times the party's part of the damage once more,
// as Mobius does) are shared out when the party did damage.
func (s *Server) partyKillRewards(n *Npc, pt *Party, shares []rewardShare, done map[*Player]bool, total int64, m stats.MonsterReward) {
	rng := float64(s.partyRange())
	pos := n.Position()
	var partyDamage int64
	partyLevel := 0
	var rewarded []*Player
	for _, member := range pt.members {
		if member.IsDead() {
			continue
		}
		near := pos.Distance3D(member.Position()) < rng
		if i := shareIndex(shares, member); i >= 0 && !done[member] {
			if near {
				partyDamage += shares[i].damage
				rewarded = append(rewarded, member)
				partyLevel = max(partyLevel, member.Level())
			}
			done[member] = true
		} else if near {
			rewarded = append(rewarded, member)
			partyLevel = max(partyLevel, member.Level())
		}
	}
	mul := social.PartyDamageMul(partyDamage, total)
	exp, sp := s.combat.ExpSpShare(m, partyLevel, partyDamage, total)
	exp, sp = exp*mul, sp*mul
	if partyDamage > 0 {
		s.distributeXpAndSp(n, rewarded, partyLevel, exp, sp)
	}
}

func shareIndex(shares []rewardShare, p *Player) int {
	for i := range shares {
		if shares[i].p == p {
			return i
		}
	}
	return -1
}

// distributeXpAndSp is Party.distributeXpAndSp with the configured cutoff
// (decision T8): each valid living member gets its share through its
// expSpRate (stats.CombatRules.ExpSPRate), the cutoff and
// PlayerStat.addExpAndSp; a living member that is cut off gets nothing (an
// empty addExpAndSp, which refreshes its UserInfo).
func (s *Server) distributeXpAndSp(n *Npc, rewarded []*Player, topLevel int, exp, sp float64) {
	rules := s.party.exp
	members := make([]social.ExpMember, len(rewarded))
	for i, m := range rewarded {
		members[i] = social.ExpMember{Level: m.Level(), SameInstance: m.InstanceID() == n.InstanceID(), Dead: m.IsDead()}
	}
	valid := rules.Valid(members, topLevel)
	out := rules.Distribute(members, topLevel, exp, sp, func(i int, v float64) float64 {
		return s.combat.ExpSPRate(rewarded[i].stats, v)
	})
	for i, m := range rewarded {
		if m.IsDead() || !m.InGame() {
			continue
		}
		level := m.Level()
		switch {
		case out[i].Get:
			s.AddExpAndSP(m, s.combat.AddExpAndSP(m.stats, out[i].Exp, out[i].SP))
		case !valid[i]:
			s.AddExpAndSP(m, stats.Reward{})
		}
		if m.Level() != level {
			s.RequestSave(m)
		}
	}
}

// partyLooter is Party.getActualLooter: the member the loot mode picks
// among those with room for the item and within the party range of ref,
// else p.
func (s *Server) partyLooter(pt *Party, p *Player, itemID int32, spoil bool, ref world.Object) *Player {
	cands := make([]social.LootCandidate, len(pt.members))
	for i, m := range pt.members {
		cands[i] = social.LootCandidate{HasSpace: m.inv.ValidateCapacityByItemID(itemID), InRange: s.partyInRange(ref, m)}
	}
	if i := pt.loot.Looter(s.rnd, cands, spoil); i >= 0 {
		return pt.members[i]
	}
	return p
}

// partyDistributeItem is Party.distributeItem for a kill's or a sweep's
// item: adena is split among the members around target, anything else goes
// to the looter the mode picks, and the other members are told who got it.
// It is how auto-loot and Sweeper give items to a party member.
func (s *Server) partyDistributeItem(p *Player, itemID int32, count int64, spoil bool, target Creature) {
	pt := p.party.party
	if pt == nil {
		return
	}
	if itemID == adenaID {
		s.partyDistributeAdena(pt, count, target)
		return
	}
	var ref world.Object = p
	if target != nil {
		ref = target
	}
	looter := s.partyLooter(pt, p, itemID, spoil, ref)
	if _, err := s.AddItem(looter, itemID, count, -1, true, spoil); err != nil {
		s.log.Warn("game: cannot give a party member its loot", append(looter.logArgs(), "item", itemID, "count", count, "err", err)...)
	}
	pt.BroadcastExcept(looter, partyLootMessage(looter, s.itemNameParam(itemID), count, spoil))
}

// partyLootMessage is what the other members learn about a looter's item.
func partyLootMessage(looter *Player, item serverpackets.SystemMessageParam, count int64, spoil bool) []byte {
	name := serverpackets.TextParam(looter.Name())
	switch {
	case count > 1 && spoil:
		return sysMsgPacket(partyMsgObtainedCountSweeper, name, item, serverpackets.NumberParam(itemCount32(count)))
	case count > 1:
		return sysMsgPacket(partyMsgObtainedCount, name, item, serverpackets.NumberParam(itemCount32(count)))
	case spoil:
		return sysMsgPacket(partyMsgObtainedSweeper, name, item)
	}
	return sysMsgPacket(partyMsgObtained, name, item)
}

// partyDistributeAdena is Party.distributeAdena: the members within the
// party range of ref share the adena evenly, the remainder 1 each from a
// random member on (social.SplitAdena), each with its "You have earned"
// message.
func (s *Server) partyDistributeAdena(pt *Party, adena int64, ref world.Object) {
	members := pt.Members()
	in := make([]bool, len(members))
	for i, m := range members {
		in[i] = s.partyInRange(ref, m)
	}
	for i, share := range social.SplitAdena(s.rnd, adena, in) {
		if share > 0 {
			s.AddAdena(members[i], share, true)
		}
	}
}

// partyPickUp is the party end of Player.doPickupItem (Party.distributeItem
// for an item on the ground): adena is split among the members around the
// picker and the item vanishes; anything else goes to the looter the mode
// picks around the picker, and the other members are told. Reports false
// when p is in no party.
func (s *Server) partyPickUp(p *Player, it *inventory.Item) bool {
	pt := p.party.party
	if pt == nil {
		return false
	}
	if it.ItemID() == adenaID {
		s.partyDistributeAdena(pt, it.Count(), p)
		return true
	}
	itemID, count := it.ItemID(), it.Count()
	looter := s.partyLooter(pt, p, itemID, false, p)
	s.addItemObject(looter, it)
	pt.BroadcastExcept(looter, partyLootMessage(looter, s.itemNameParam(itemID), count, false))
	return true
}

// inLooterParty is Player.isInLooterParty (and DropProtection's party
// rule): the player whose ID owns a drop is in p's party.
func (s *Server) inLooterParty(p *Player, owner world.ObjectID) bool {
	if p.party.party == nil {
		return false
	}
	o, ok := s.Player(owner)
	return ok && p.party.party.Contains(o)
}

// pickUpCapacityChecked reports whether picking up checks the picker's own
// inventory (Mobius doPickupItem: alone, or in a finders keepers party).
func pickUpCapacityChecked(p *Player) bool {
	return p.party.party == nil || p.party.party.loot.Mode == social.FindersKeepers
}
