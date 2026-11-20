package scripting

import (
	lua "github.com/yuin/gopher-lua"
)

// PlayerParty is a PlayerRef that can be in a party (Plan 12 Phase 3). A
// player without it is always alone, as before parties existed.
type PlayerParty interface {
	// PartyMembers returns the members of the player's party in party
	// order, the leader first; nil without a party.
	PartyMembers() []PlayerRef
	// PartyLootMode returns the party's loot mode (Mobius
	// PartyDistributionType: 0 finders keepers, 1 random, 2 random
	// including spoil, 3 by turn, 4 by turn including spoil), -1 without a
	// party.
	PartyLootMode() int
	// QuestRangeTarget returns what the party quest helpers measure the
	// distance from (Mobius getRandomPartyMember: the player's target, or
	// the player itself without one).
	QuestRangeTarget() any
	// SameInstance reports whether other is in the player's instance.
	SameInstance(other PlayerRef) bool
}

// partyOf returns the members of p's party, nil when p has none.
func partyOf(p PlayerRef) []PlayerRef {
	if pp, ok := p.(PlayerParty); ok {
		return pp.PartyMembers()
	}
	return nil
}

// playerPartyMethods are the party methods of Player objects: is_in_party,
// party ({leader = Player, members = {Player...}, loot = mode} or nil) and
// party_members (the members, or the player alone).
func (v *vm) playerPartyMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"is_in_party": func(L *lua.LState) int {
			p := checkPlayer(L, "is_in_party")
			L.Push(lua.LBool(len(partyOf(p)) > 0))
			return 1
		},
		"party": func(L *lua.LState) int {
			p := checkPlayer(L, "party")
			members := partyOf(p)
			if len(members) == 0 {
				L.Push(lua.LNil)
				return 1
			}
			t := L.CreateTable(0, 3)
			t.RawSetString("leader", v.newPlayer(L, members[0]))
			t.RawSetString("members", v.playerList(L, members))
			t.RawSetString("loot", lua.LNumber(p.(PlayerParty).PartyLootMode()))
			L.Push(t)
			return 1
		},
		"party_members": func(L *lua.LState) int {
			p := checkPlayer(L, "party_members")
			members := partyOf(p)
			if len(members) == 0 {
				members = []PlayerRef{p}
			}
			L.Push(v.playerList(L, members))
			return 1
		},
	}
}

// partyCandidate reports whether member may be picked by p's party helpers
// (Mobius getRandomPartyMember(player, var, value) and
// getRandomPartyMemberState(player, state)): within the party range (3D,
// no collision) of p's target, or of p without one, and in p's instance.
func (v *vm) partyCandidate(p, member PlayerRef) bool {
	pp, ok := p.(PlayerParty)
	if !ok {
		return true
	}
	target := pp.QuestRangeTarget()
	if target == nil {
		target = p
	}
	lt, okT := locationOf(target)
	lm, okM := locationOf(member)
	if okT && okM && distance(lt, lm, true) > float64(v.partyRange()) {
		return false
	}
	return pp.SameInstance(member)
}
