package game

import "github.com/IPeter/l2go/internal/gameserver/serverpackets"

// The live party window: PartySmallWindowUpdate when a member's CP, HP or MP
// crosses one of the bar's 352 steps (Mobius Player.broadcastStatusUpdate,
// needCpUpdate, needHpUpdate, needMpUpdate) and on a class change, and
// PartySpelled with the member's buffs (EffectList.updateEffectIcons).

// partyBars are a player's bar thresholds (Mobius _cpUpdateIncCheck and
// friends), set from the maxima the first time they are needed (Mobius
// initCharStatusUpdateValues runs when the player is made).
type partyBars struct {
	ready      bool
	cp, hp, mp npcHPBarCheck
}

// partyStatusUpdate runs after a player's StatusUpdate (flushStatus): the
// CP and HP checks always run, as in Mobius, the MP check only for a party
// member whose CP and HP did not cross a step; a crossing sends the other
// members PartySmallWindowUpdate.
func (s *Server) partyStatusUpdate(p *Player) {
	v := p.stats.View()
	b := &p.party.bars
	if !b.ready {
		b.cp.reset(float64(v.MaxCP))
		b.hp.reset(float64(v.MaxHP))
		b.mp.reset(float64(v.MaxMP))
		b.ready = true
	}
	cp := b.cp.need(p.CurCP, float64(v.MaxCP))
	hp := b.hp.need(p.CurHP, float64(v.MaxHP))
	pt := p.party.party
	if pt != nil && (cp || hp || b.mp.need(p.CurMP, float64(v.MaxMP))) {
		pt.BroadcastExcept(p, serverpackets.NewPartySmallWindowUpdatePacket(partyMember(p)))
	}
}

// partyClassChanged updates the class icon in the party window (Mobius
// setPlayerClass: PartySmallWindowUpdate to the whole party).
func (s *Server) partyClassChanged(p *Player) {
	if pt := p.party.party; pt != nil {
		pt.Broadcast(serverpackets.NewPartySmallWindowUpdatePacket(partyMember(p)))
	}
}

// partySpelled sends a member's party icons to the whole party (Mobius
// updateEffectIcons: PartySpelled when the owner is in a party).
func (s *Server) partySpelled(p *Player, effects []serverpackets.AbnormalStatusEntry) {
	pt := p.party.party
	if pt == nil {
		return
	}
	pt.Broadcast(serverpackets.NewPartySpelledPacket(serverpackets.PartySpelled{
		Type:     serverpackets.PartySpelledPlayer,
		ObjectID: p.ObjectID(),
		Effects:  effects,
	}))
}
