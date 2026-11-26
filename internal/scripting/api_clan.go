package scripting

import (
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

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
	// OnlineMembers returns the members in the game, the leader included
	// (Mobius getOnlineMembers(0)).
	OnlineMembers() []PlayerRef
}

// PlayerOnline is the part of a PlayerRef behind player:is_online() (Mobius
// isOnline). A PlayerRef without it counts as online.
type PlayerOnline interface {
	IsOnline() bool
}

// AbnormalChecker is the part of a QuestWorld behind
// player:buff_info_by_abnormal_type(type): whether a PlayerRef or NpcRef
// has an effect of the abnormal type (a Mobius AbnormalType name). Without
// it the answer is nil.
type AbnormalChecker interface {
	HasAbnormalType(obj any, abnormalType string) bool
}

// NpcReturnControl is the part of a QuestWorld behind
// npc:set_can_return_to_spawn_point(bool) (Mobius
// Attackable.setCanReturnToSpawnPoint, which the clan quest Q503 uses).
type NpcReturnControl interface {
	NpcSetCanReturnToSpawnPoint(npc NpcRef, can bool)
}

// npcSetCanReturnToSpawnPoint is npc:set_can_return_to_spawn_point(bool).
func (v *vm) npcSetCanReturnToSpawnPoint(L *lua.LState) int {
	const fn = "npc:set_can_return_to_spawn_point"
	n := checkNpc(L, "set_can_return_to_spawn_point")
	can := L.ToBool(2)
	worldAs[NpcReturnControl](v, L, fn).NpcSetCanReturnToSpawnPoint(n, can)
	return 0
}

// playerIsOnline is player:is_online().
func playerIsOnline(L *lua.LState) int {
	p := checkPlayer(L, "is_online")
	on := true
	if po, ok := p.(PlayerOnline); ok {
		on = po.IsOnline()
	}
	L.Push(lua.LBool(on))
	return 1
}

// playerBuffInfoByAbnormalType is player:buff_info_by_abnormal_type(type):
// true when the player has an effect of the abnormal type, else nil (Mobius
// getEffectList().getBuffInfoByAbnormalType, which scripts compare with
// null).
func (v *vm) playerBuffInfoByAbnormalType(L *lua.LState) int {
	p := checkPlayer(L, "buff_info_by_abnormal_type")
	typ := L.CheckString(2)
	w := v.needWorld(L, "player:buff_info_by_abnormal_type")
	if ac, ok := w.(AbnormalChecker); ok && ac.HasAbnormalType(p, typ) {
		L.Push(lua.LTrue)
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

// clanLeaderQuestState is Mobius Quest.getClanLeaderQuestState: the quest
// state of p's clan leader when the leader stands within the party range of
// npc (p itself when it leads), alive and in the game; nil otherwise or
// without a state.
func (v *vm) clanLeaderQuestState(L *lua.LState, s *npcScript, p PlayerRef, npc NpcRef, fn string) (PlayerRef, QuestState) {
	w := v.needWorld(L, fn)
	r := v.partyRange()
	if pc, ok := p.(PlayerClan); ok && pc.IsClanLeader() && within3D(p, npc, r) {
		return p, w.QuestState(p, s.name)
	}
	c := playerClanOf(p)
	if c == nil {
		return nil, nil
	}
	leader := c.LeaderPlayer()
	if leader == nil {
		return nil, nil
	}
	if d, ok := leader.(PlayerDetails); ok && d.IsDead() {
		return nil, nil
	}
	if !within3D(leader, npc, r) {
		return nil, nil
	}
	return leader, w.QuestState(leader, s.name)
}

// within3D is Mobius isInsideRadius3D(npc, r): the 3D distance is below r
// (without collision); refs without a position count as inside.
func within3D(a, b any, r int) bool {
	la, okA := locationOf(a)
	lb, okB := locationOf(b)
	if !okA || !okB {
		return true
	}
	return distance(la, lb, true) < float64(r)
}

// qGetClanLeaderQuestState is q:get_clan_leader_quest_state(player, npc).
func (v *vm) qGetClanLeaderQuestState(L *lua.LState) int {
	const fn = "q:get_clan_leader_quest_state"
	s := checkScript(L, "get_clan_leader_quest_state")
	p := checkPlayerArg(L, 2, fn)
	npc, _ := objectArg(L, 3).(NpcRef)
	leader, st := v.clanLeaderQuestState(L, s, p, npc, fn)
	if st == nil {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(v.newQuestState(L, leader, st))
	return 1
}

// qSetQuestToClanMembers is q:set_quest_to_clan_members(player) (Mobius
// setQuestToClanMembers): when player leads its clan, every other member in
// the game gets the quest started with the leader's cond. Deviation: Mobius
// shares the leader's state object with them and also writes a started
// state for the offline members; here the members online get their own
// state and the offline ones none (the clan quests read the leader's state
// for members anyway).
func (v *vm) qSetQuestToClanMembers(L *lua.LState) int {
	const fn = "q:set_quest_to_clan_members"
	s := checkScript(L, "set_quest_to_clan_members")
	p := checkPlayerArg(L, 2, fn)
	pc, ok := p.(PlayerClan)
	if !ok || !pc.IsClanLeader() {
		return 0
	}
	c := pc.ScriptClan()
	if c == nil {
		return 0
	}
	w := v.needWorld(L, fn)
	cond := 0
	if lst := w.QuestState(p, s.name); lst != nil {
		cond = lst.Cond()
	}
	for _, m := range c.OnlineMembers() {
		if m == nil || m.ObjectID() == p.ObjectID() {
			continue
		}
		st := w.QuestState(m, s.name)
		if st == nil {
			st = w.NewQuestState(m, s.name)
		}
		if st == nil {
			continue
		}
		st.SetState(StateStarted)
		if cond > 0 {
			st.Set("cond", strconv.Itoa(cond))
		}
	}
	return 0
}

// qFinishQuestToClan is q:finish_quest_to_clan(player) (Mobius
// finishQuestToClan): when player leads its clan, the quest state of every
// other member in the game is deleted. Deviation: the offline members keep
// theirs (set_quest_to_clan_members gives them none).
func (v *vm) qFinishQuestToClan(L *lua.LState) int {
	const fn = "q:finish_quest_to_clan"
	s := checkScript(L, "finish_quest_to_clan")
	p := checkPlayerArg(L, 2, fn)
	pc, ok := p.(PlayerClan)
	if !ok || !pc.IsClanLeader() {
		return 0
	}
	c := pc.ScriptClan()
	if c == nil {
		return 0
	}
	w := v.needWorld(L, fn)
	for _, m := range c.OnlineMembers() {
		if m == nil || m.ObjectID() == p.ObjectID() {
			continue
		}
		if st := w.QuestState(m, s.name); st != nil && st.State() == StateStarted {
			st.Exit(true)
		}
	}
	return 0
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
