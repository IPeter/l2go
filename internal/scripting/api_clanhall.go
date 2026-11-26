package scripting

import (
	lua "github.com/yuin/gopher-lua"
)

// The clan hall side of the scripts (Plan 14 M11j): the contestable halls'
// siege scripts (scripts/conquerablehalls/) reach their hall and its siege
// through the clanhall module. The game runs the engine (Mobius
// ClanHallSiegeEngine) and asks the hall's script at its steps through
// on_event with the events "siege_prepare_owner", "siege_start_siege"
// (answer "replace" to do the step yourself), "siege_siege_starts",
// "siege_siege_ends", "siege_end_siege" (before the default end) and
// "siege_winner" (answer the winning clan's ID as text); npc and player are
// nil for them.

// ClanHallWorld is the part of a QuestWorld behind the clanhall module.
type ClanHallWorld interface {
	// ScriptClanHall returns the hall with the ID, nil for none.
	ScriptClanHall(id int) ClanHallRef
	// NpcClanHall returns the hall an NPC belongs to (Mobius
	// getClanHall / getConquerableHall), nil for none.
	NpcClanHall(npc NpcRef) ClanHallRef
	// ClanInfo returns what the hall scripts read of a clan.
	ClanInfo(clanID int32) (ClanInfo, bool)
}

// ClanInfo is what the hall scripts read of a clan.
type ClanInfo struct {
	ID         int32
	Name       string
	Level      int
	Members    int
	LeaderID   int32
	LeaderName string
	CastleID   int32
	HallID     int32
	Dissolving bool
}

// ClanHallRef is a clan hall as the scripts see it (Mobius ClanHall and
// SiegableHall with its ClanHallSiegeEngine). Its methods run on the world
// loop, while a handler runs.
type ClanHallRef interface {
	HallID() int
	HallName() string
	OwnerID() int32
	IsSiegable() bool
	// SiegeDate is the next siege's start in Unix seconds, 0 for none.
	SiegeDate() int64
	// SiegeStatus is "registering", "waiting_battle" or "running".
	SiegeStatus() string
	SetSiegeStatus(status string)
	Attackers() []int32
	IsAttacker(clanID int32) bool
	AddAttacker(clanID int32)
	RemoveAttacker(clanID int32)
	ClearAttackers()
	// Free takes the hall from its owner (SiegableHall.free and the owner's
	// setHideoutId(0)), saved.
	Free()
	BanishForeigners()
	BanishNonParticipants()
	OpenCloseDoor(doorID int, open bool)
	SetZoneActive(active bool)
	UpdateNextSiege()
	// CloseRegistration is the end of prepareOwner: the announcement, the
	// waiting status and the siege an hour later.
	CloseRegistration()
	// Schedule runs a step after ms: "prepare_owner", "start_siege" (the
	// default start), "end_siege"; it replaces the pending step.
	Schedule(step string, ms int64)
	CancelSiegeTask()
	// StartSiegeDefault is the engine's startSiege; CancelSiege its "lack of
	// interest" branch; EndSiege the end, with the mission accomplished when
	// accomplished.
	StartSiegeDefault()
	CancelSiege()
	EndSiege(accomplished bool)
	Var(name string) (string, bool)
	// SetVar keeps a value with the hall, saved; "" deletes it.
	SetVar(name, value string)
	VarNames() []string
	// Spawn places an NPC for the siege, respawning after respawnMs (0
	// never) until Despawn or the siege's end.
	Spawn(npcID int, loc Location, respawnMs int64) NpcRef
	Despawn(objectID int32)
	// ZoneSpawn is a point of the hall zone's spawns of a kind ("" normal,
	// "challenger", "banish", ...) by index.
	ZoneSpawn(kind string, index int) (Location, bool)
	// TeleportClan sends the clan's members online to loc.
	TeleportClan(clanID int32, loc Location)
	TeleportToTown(objectID int32)
	// CheckTeleportZone runs the hall teleport zone's 30 s task
	// (ResidenceHallTeleportZone.checkTeleportTask).
	CheckTeleportZone(index int)
	ShowSiegeInfo(p PlayerRef)
}

// clanHallFuncs is the clanhall module: get(id), of_npc(npc) and
// clan(clan_id).
func (v *vm) clanHallFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			const fn = "clanhall.get"
			h := worldAs[ClanHallWorld](v, L, fn).ScriptClanHall(int(checkInt(L, 1, fn)))
			v.pushHall(L, h)
			return 1
		},
		"of_npc": func(L *lua.LState) int {
			const fn = "clanhall.of_npc"
			n := optNpcArg(L, 1, fn)
			if n == nil {
				L.ArgError(1, fn+": an NPC expected")
				return 0
			}
			v.pushHall(L, worldAs[ClanHallWorld](v, L, fn).NpcClanHall(n))
			return 1
		},
		"clan": func(L *lua.LState) int {
			const fn = "clanhall.clan"
			c, ok := worldAs[ClanHallWorld](v, L, fn).ClanInfo(checkObjectID(L, 1, fn))
			if !ok {
				L.Push(lua.LNil)
				return 1
			}
			t := L.NewTable()
			t.RawSetString("id", lua.LNumber(c.ID))
			t.RawSetString("name", lua.LString(c.Name))
			t.RawSetString("level", lua.LNumber(c.Level))
			t.RawSetString("members", lua.LNumber(c.Members))
			t.RawSetString("leader_id", lua.LNumber(c.LeaderID))
			t.RawSetString("leader_name", lua.LString(c.LeaderName))
			t.RawSetString("castle_id", lua.LNumber(c.CastleID))
			t.RawSetString("hall_id", lua.LNumber(c.HallID))
			t.RawSetString("dissolving", lua.LBool(c.Dissolving))
			L.Push(t)
			return 1
		},
	}
}

func (v *vm) pushHall(L *lua.LState, h ClanHallRef) {
	if h == nil {
		L.Push(lua.LNil)
		return
	}
	L.Push(v.newHall(L, h))
}

// newHall builds the Lua hall object: a table of methods, called with ":"
// (the table itself is ignored), like the castle object.
func (v *vm) newHall(L *lua.LState, h ClanHallRef) lua.LValue {
	t := L.NewTable()
	fn := func(name string, f lua.LGFunction) { t.RawSetString(name, L.NewFunction(f)) }
	num := func(name string, get func() int64) {
		fn(name, func(L *lua.LState) int {
			L.Push(lua.LNumber(get()))
			return 1
		})
	}
	flag := func(name string, get func() bool) {
		fn(name, func(L *lua.LState) int {
			L.Push(lua.LBool(get()))
			return 1
		})
	}
	act := func(name string, do func()) {
		fn(name, func(L *lua.LState) int {
			do()
			return 0
		})
	}
	clanArg := func(name string, do func(int32)) {
		fn(name, func(L *lua.LState) int {
			do(checkObjectID(L, 2, "hall:"+name))
			return 0
		})
	}
	num("id", func() int64 { return int64(h.HallID()) })
	fn("name", func(L *lua.LState) int {
		L.Push(lua.LString(h.HallName()))
		return 1
	})
	num("owner_id", func() int64 { return int64(h.OwnerID()) })
	flag("is_siegable", h.IsSiegable)
	num("siege_date", h.SiegeDate)
	fn("status", func(L *lua.LState) int {
		L.Push(lua.LString(h.SiegeStatus()))
		return 1
	})
	flag("is_registering", func() bool { return h.SiegeStatus() == "registering" })
	flag("is_waiting_battle", func() bool { return h.SiegeStatus() == "waiting_battle" })
	flag("is_in_siege", func() bool { return h.SiegeStatus() == "running" })
	fn("set_status", func(L *lua.LState) int {
		s := L.CheckString(2)
		if s != "registering" && s != "waiting_battle" && s != "running" {
			L.ArgError(2, "hall:set_status: registering, waiting_battle or running")
			return 0
		}
		h.SetSiegeStatus(s)
		return 0
	})
	fn("attackers", func(L *lua.LState) int {
		out := L.NewTable()
		for i, id := range h.Attackers() {
			out.RawSetInt(i+1, lua.LNumber(id))
		}
		L.Push(out)
		return 1
	})
	num("attacker_count", func() int64 { return int64(len(h.Attackers())) })
	fn("is_attacker", func(L *lua.LState) int {
		L.Push(lua.LBool(h.IsAttacker(checkObjectID(L, 2, "hall:is_attacker"))))
		return 1
	})
	clanArg("add_attacker", h.AddAttacker)
	clanArg("remove_attacker", h.RemoveAttacker)
	act("clear_attackers", h.ClearAttackers)
	act("free", h.Free)
	act("banish_foreigners", h.BanishForeigners)
	act("banish_non_participants", h.BanishNonParticipants)
	fn("open_close_door", func(L *lua.LState) int {
		h.OpenCloseDoor(int(checkInt(L, 2, "hall:open_close_door")), L.ToBool(3))
		return 0
	})
	fn("set_zone_active", func(L *lua.LState) int {
		h.SetZoneActive(L.ToBool(2))
		return 0
	})
	act("update_next_siege", h.UpdateNextSiege)
	act("close_registration", h.CloseRegistration)
	fn("schedule", func(L *lua.LState) int {
		const name = "hall:schedule"
		step := L.CheckString(2)
		if step != "prepare_owner" && step != "start_siege" && step != "end_siege" {
			L.ArgError(2, name+": prepare_owner, start_siege or end_siege")
			return 0
		}
		h.Schedule(step, checkInt(L, 3, name))
		return 0
	})
	act("cancel_siege_task", h.CancelSiegeTask)
	act("start_siege", h.StartSiegeDefault)
	act("cancel_siege", h.CancelSiege)
	fn("end_siege", func(L *lua.LState) int {
		h.EndSiege(L.ToBool(2))
		return 0
	})
	fn("get_var", func(L *lua.LState) int {
		if s, ok := h.Var(L.CheckString(2)); ok {
			L.Push(lua.LString(s))
		} else {
			L.Push(L.Get(3))
		}
		return 1
	})
	fn("set_var", func(L *lua.LState) int {
		h.SetVar(L.CheckString(2), valueString(L, 3, "hall:set_var"))
		return 0
	})
	fn("var_names", func(L *lua.LState) int {
		out := L.NewTable()
		for i, n := range h.VarNames() {
			out.RawSetInt(i+1, lua.LString(n))
		}
		L.Push(out)
		return 1
	})
	fn("spawn", func(L *lua.LState) int {
		const name = "hall:spawn"
		id := int(checkInt(L, 2, name))
		loc := locationArg(L, 3, name)
		n := h.Spawn(id, loc, optInt(L, 4, name, 0))
		if n == nil {
			L.Push(lua.LNil)
		} else {
			L.Push(v.newNpc(L, n))
		}
		return 1
	})
	clanArg("despawn", h.Despawn)
	fn("zone_spawn", func(L *lua.LState) int {
		loc, ok := h.ZoneSpawn(L.OptString(2, ""), int(optInt(L, 3, "hall:zone_spawn", 0)))
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(locationTable(L, loc))
		return 1
	})
	fn("teleport_clan", func(L *lua.LState) int {
		const name = "hall:teleport_clan"
		h.TeleportClan(checkObjectID(L, 2, name), locationArg(L, 3, name))
		return 0
	})
	clanArg("teleport_to_town", h.TeleportToTown)
	fn("check_teleport_zone", func(L *lua.LState) int {
		h.CheckTeleportZone(int(checkInt(L, 2, "hall:check_teleport_zone")))
		return 0
	})
	fn("show_siege_info", func(L *lua.LState) int {
		h.ShowSiegeInfo(checkPlayerArg(L, 2, "hall:show_siege_info"))
		return 0
	})
	return t
}
