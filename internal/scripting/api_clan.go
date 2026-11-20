package scripting

import lua "github.com/yuin/gopher-lua"

// PlayerClan is the clan side of a PlayerRef (Mobius Player.getClan and
// isClanLeader). A PlayerRef without it is in no clan.
type PlayerClan interface {
	// ScriptClan returns the player's clan, nil without one.
	ScriptClan() ClanRef
	// IsClanLeader reports whether the player leads its clan.
	IsClanLeader() bool
}

// ClanRef is a clan as the scripts see it (Mobius Clan). Its methods run on
// the world loop, like every script handler.
type ClanRef interface {
	ClanID() int32
	ClanName() string
	ClanLevel() int
	Reputation() int
	// LeaderID and LeaderName are the leader's object ID and name;
	// LeaderPlayer is the leader in the game, nil when it is offline.
	LeaderID() int32
	LeaderName() string
	LeaderPlayer() PlayerRef
	// AddReputation changes the reputation (Mobius addReputationScore, with
	// its messages and PledgeShowInfoUpdate), negative to take.
	AddReputation(points int)
	// BroadcastPledgeInfo sends the members online PledgeShowInfoUpdate
	// (Mobius broadcastToOnlineMembers(new PledgeShowInfoUpdate(clan))).
	BroadcastPledgeInfo()
}

// playerClanOf returns the player's clan, nil without one.
func playerClanOf(p PlayerRef) ClanRef {
	if pc, ok := p.(PlayerClan); ok {
		return pc.ScriptClan()
	}
	return nil
}

// playerClanMethod is player:clan(): the clan as a table of methods, nil
// without a clan.
func (v *vm) playerClanMethod(L *lua.LState) int {
	c := playerClanOf(checkPlayer(L, "clan"))
	if c == nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(v.newClan(L, c))
	return 1
}

// playerIsClanLeader is player:is_clan_leader().
func playerIsClanLeader(L *lua.LState) int {
	p := checkPlayer(L, "is_clan_leader")
	pc, ok := p.(PlayerClan)
	L.Push(lua.LBool(ok && pc.IsClanLeader()))
	return 1
}

// newClan builds the Lua clan object: a table of methods, called with ":"
// (the table itself is ignored):
//
//	clan:id(), clan:name(), clan:level(), clan:reputation(),
//	clan:leader_id(), clan:add_reputation(n),
//	clan:broadcast_pledge_info_update(),
//	clan:leader() -- a member: member:object_id(), member:name(),
//	              -- member:player() (nil while offline)
func (v *vm) newClan(L *lua.LState, c ClanRef) lua.LValue {
	t := L.NewTable()
	num := func(f func() int) *lua.LFunction {
		return L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNumber(f()))
			return 1
		})
	}
	t.RawSetString("id", num(func() int { return int(c.ClanID()) }))
	t.RawSetString("name", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(c.ClanName()))
		return 1
	}))
	t.RawSetString("level", num(c.ClanLevel))
	t.RawSetString("reputation", num(c.Reputation))
	t.RawSetString("leader_id", num(func() int { return int(c.LeaderID()) }))
	t.RawSetString("add_reputation", L.NewFunction(func(L *lua.LState) int {
		c.AddReputation(int(L.CheckNumber(2)))
		return 0
	}))
	t.RawSetString("broadcast_pledge_info_update", L.NewFunction(func(L *lua.LState) int {
		c.BroadcastPledgeInfo()
		return 0
	}))
	t.RawSetString("leader", L.NewFunction(func(L *lua.LState) int {
		m := L.NewTable()
		m.RawSetString("object_id", num(func() int { return int(c.LeaderID()) }))
		m.RawSetString("name", L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LString(c.LeaderName()))
			return 1
		}))
		m.RawSetString("player", L.NewFunction(func(L *lua.LState) int {
			L.Push(v.newPlayer(L, c.LeaderPlayer()))
			return 1
		}))
		L.Push(m)
		return 1
	}))
	return t
}
