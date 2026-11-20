package game

import "github.com/IPeter/l2go/internal/scripting"

// The party as the Lua quest helpers see it (scripting.PlayerParty):
// player:is_in_party(), player:party(), player:party_members() and the
// q:get_random_party_member* helpers.

var _ scripting.PlayerParty = (*Player)(nil)

// PartyMembers returns the members of the player's party, the leader
// first; nil without a party. Loop-only.
func (p *Player) PartyMembers() []scripting.PlayerRef {
	pt := p.party.party
	if pt == nil {
		return nil
	}
	out := make([]scripting.PlayerRef, len(pt.members))
	for i, m := range pt.members {
		out[i] = m
	}
	return out
}

// PartyLootMode returns the loot mode of the player's party, -1 without
// one. Loop-only.
func (p *Player) PartyLootMode() int {
	if pt := p.party.party; pt != nil {
		return int(pt.loot.Mode)
	}
	return -1
}

// questPoint is a position the quest helpers measure distances from.
type questPoint scripting.Location

// Location returns the position (scripting.Located).
func (q questPoint) Location() scripting.Location { return scripting.Location(q) }

// QuestRangeTarget returns the position of the player's target, nil
// without one (Mobius getRandomPartyMember measures from it). Loop-only.
func (p *Player) QuestRangeTarget() any {
	if p.Target == nil {
		return nil
	}
	pos := p.Target.Position()
	return questPoint{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z), Heading: int(pos.Heading)}
}

// SameInstance reports whether other is a player in p's instance.
// Loop-only.
func (p *Player) SameInstance(other scripting.PlayerRef) bool {
	o, ok := other.(*Player)
	return ok && o.InstanceID() == p.InstanceID()
}
