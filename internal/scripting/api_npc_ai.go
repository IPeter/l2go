package scripting

import (
	"context"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// The NPC control part of the AI API (Plan 13, Phase 10.6): handles,
// variables, the core AI, intentions, the hate list, spawn data, template
// parameters, leaders and minions, script events and chests.

// npcControlMethods are the Npc methods of the AI API.
func (v *vm) npcControlMethods() map[string]lua.LGFunction {
	// control returns the NPC and the world's NpcControl for method.
	control := func(L *lua.LState, method string) (NpcRef, NpcControl) {
		n := checkNpc(L, method)
		return n, worldAs[NpcControl](v, L, "npc:"+method)
	}
	setter := func(method string, set func(c NpcControl, n NpcRef, on bool)) lua.LGFunction {
		return func(L *lua.LState) int {
			n, c := control(L, method)
			on := true
			if L.GetTop() >= 2 {
				on = L.ToBool(2)
			}
			set(c, n, on)
			return 0
		}
	}
	getter := func(method string, get func(c NpcControl, n NpcRef) bool) lua.LGFunction {
		return func(L *lua.LState) int {
			n, c := control(L, method)
			L.Push(lua.LBool(get(c, n)))
			return 1
		}
	}
	plain := func(method string, do func(c NpcControl, n NpcRef)) lua.LGFunction {
		return func(L *lua.LState) int {
			n, c := control(L, method)
			do(c, n)
			return 0
		}
	}
	withTarget := func(method string, do func(c NpcControl, n NpcRef, target any)) lua.LGFunction {
		return func(L *lua.LState) int {
			n, c := control(L, method)
			do(c, n, checkObject(L, 2, "npc:"+method))
			return 0
		}
	}
	m := map[string]lua.LGFunction{
		"valid": func(L *lua.LState) int {
			b := npcHandle(L, 1)
			if b == nil {
				L.RaiseError("valid must be called on an NPC: npc:valid()")
			}
			L.Push(lua.LBool(b.valid()))
			return 1
		},

		"set_walking":           plain("set_walking", func(c NpcControl, n NpcRef) { c.NpcSetWalking(n) }),
		"is_running":            getter("is_running", func(c NpcControl, n NpcRef) bool { return c.NpcIsRunning(n) }),
		"disable_core_ai":       setter("disable_core_ai", func(c NpcControl, n NpcRef, on bool) { c.NpcDisableCoreAI(n, on) }),
		"is_core_ai_disabled":   getter("is_core_ai_disabled", func(c NpcControl, n NpcRef) bool { return c.NpcIsCoreAIDisabled(n) }),
		"set_immobilized":       setter("set_immobilized", func(c NpcControl, n NpcRef, on bool) { c.NpcSetImmobilized(n, on) }),
		"set_invul":             setter("set_invul", func(c NpcControl, n NpcRef, on bool) { c.NpcSetInvul(n, on) }),
		"is_invul":              getter("is_invul", func(c NpcControl, n NpcRef) bool { return c.NpcIsInvul(n) }),
		"set_mortal":            setter("set_mortal", func(c NpcControl, n NpcRef, on bool) { c.NpcSetMortal(n, on) }),
		"is_mortal":             getter("is_mortal", func(c NpcControl, n NpcRef) bool { return c.NpcIsMortal(n) }),
		"set_random_walking":    setter("set_random_walking", func(c NpcControl, n NpcRef, on bool) { c.NpcSetRandomWalking(n, on) }),
		"return_home":           plain("return_home", func(c NpcControl, n NpcRef) { c.NpcReturnHome(n) }),
		"stop_follow":           plain("stop_follow", func(c NpcControl, n NpcRef) { c.NpcStopFollow(n) }),
		"stop_moving":           plain("stop_moving", func(c NpcControl, n NpcRef) { c.NpcStopMoving(n) }),
		"follow":                withTarget("follow", func(c NpcControl, n NpcRef, t any) { c.NpcFollow(n, t) }),
		"is_moving":             getter("is_moving", func(c NpcControl, n NpcRef) bool { return c.NpcIsMoving(n) }),
		"is_in_combat":          getter("is_in_combat", func(c NpcControl, n NpcRef) bool { return c.NpcIsInCombat(n) }),
		"is_casting_now":        getter("is_casting_now", func(c NpcControl, n NpcRef) bool { return c.NpcIsCastingNow(n) }),
		"clear_aggro_list":      plain("clear_aggro_list", func(c NpcControl, n NpcRef) { c.NpcClearAggroList(n) }),
		"stop_hating":           withTarget("stop_hating", func(c NpcControl, n NpcRef, t any) { c.NpcStopHating(n, t) }),
		"intention":             v.npcIntention,
		"set_intention":         v.npcSetIntention,
		"most_hated":            v.npcMostHated,
		"hate_list":             v.npcHateList,
		"reduce_hate":           v.npcReduceHate,
		"set_heading":           v.npcSetHeading,
		"spawn_location":        v.npcSpawnLocation,
		"stays_in_spawn_loc":    v.npcStaysInSpawnLoc,
		"ai_value":              v.npcAIValue,
		"has_ai_value":          v.npcHasAIValue,
		"param":                 v.npcParam,
		"param_int":             v.npcParamInt,
		"leader":                v.npcLeader,
		"minions":               v.npcMinions,
		"has_minions":           v.npcHasMinions,
		"add_minion":            v.npcAddMinion,
		"broadcast_event":       v.npcBroadcastEvent,
		"send_event":            v.npcSendEvent,
		"is_box":                v.chestGetter("is_box", ChestActions.ChestIsBox),
		"is_interacted":         v.chestGetter("is_interacted", ChestActions.ChestIsInteracted),
		"set_interacted":        v.chestAction("set_interacted", ChestActions.ChestSetInteracted),
		"treasure_bomb":         v.chestAction("treasure_bomb", ChestActions.ChestTreasureBomb),
		"is_inside_zone":        v.npcIsInsideZone,
		"calculate_distance_2d": v.distanceMethod("npc", false),
		"calculate_distance_3d": v.distanceMethod("npc", true),
	}
	for name, fn := range v.npcVarMethods() {
		m[name] = fn
	}
	return m
}

// --- Variables ---

// npcVarMethods are npc:var, var_int, var_bool, set_var, unset_var and
// has_var: Mobius NpcVariables, kept by the engine per object ID until the
// NPC is gone.
func (v *vm) npcVarMethods() map[string]lua.LGFunction {
	vars := func(L *lua.LState, method string, create bool) map[string]string {
		n := checkNpc(L, method)
		e := v.engine
		if e == nil {
			L.RaiseError("npc:%s: the world cannot be used while the scripts load; do it in q:on_start", method)
			return nil
		}
		m := e.npcVars[n.ObjectID()]
		if m == nil && create {
			m = make(map[string]string)
			e.npcVars[n.ObjectID()] = m
		}
		return m
	}
	return map[string]lua.LGFunction{
		"var": func(L *lua.LState) int {
			if s, ok := vars(L, "var", false)[L.CheckString(2)]; ok {
				L.Push(lua.LString(s))
			} else {
				L.Push(L.Get(3))
			}
			return 1
		},
		"var_int": func(L *lua.LState) int {
			s, ok := vars(L, "var_int", false)[L.CheckString(2)]
			def := optInt(L, 3, "npc:var_int", 0)
			n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
			if !ok || err != nil {
				n = def
			}
			L.Push(lua.LNumber(n))
			return 1
		},
		"var_bool": func(L *lua.LState) int {
			s, ok := vars(L, "var_bool", false)[L.CheckString(2)]
			if !ok {
				L.Push(lua.LBool(L.ToBool(3)))
			} else {
				L.Push(lua.LBool(strings.EqualFold(s, "true")))
			}
			return 1
		},
		"set_var": func(L *lua.LState) int {
			m := vars(L, "set_var", true)
			m[L.CheckString(2)] = valueString(L, 3, "npc:set_var")
			return 0
		},
		"unset_var": func(L *lua.LState) int {
			delete(vars(L, "unset_var", false), L.CheckString(2))
			return 0
		},
		"has_var": func(L *lua.LState) int {
			_, ok := vars(L, "has_var", false)[L.CheckString(2)]
			L.Push(lua.LBool(ok))
			return 1
		},
	}
}

// --- Intentions and the hate list ---

// intentions are Mobius's Intention names.
var intentions = []string{"IDLE", "ACTIVE", "REST", "ATTACK", "CAST", "MOVE_TO", "FOLLOW", "PICK_UP", "INTERACT"}

func (v *vm) npcIntention(L *lua.LState) int {
	n := checkNpc(L, "intention")
	L.Push(lua.LString(worldAs[NpcControl](v, L, "npc:intention").NpcIntention(n)))
	return 1
}

// npcSetIntention is npc:set_intention(intention[, arg]): arg is a
// player, an NPC or a position.
func (v *vm) npcSetIntention(L *lua.LState) int {
	const fn = "npc:set_intention"
	n := checkNpc(L, "set_intention")
	name := L.CheckString(2)
	known := false
	for _, i := range intentions {
		known = known || i == name
	}
	if !known {
		L.ArgError(2, fn+": unknown intention "+strconv.Quote(name)+"; use an Intention value")
	}
	var arg any
	switch L.Get(3).(type) {
	case *lua.LNilType:
	case *lua.LTable:
		loc := locationArg(L, 3, fn)
		arg = &loc
	default:
		arg = checkObject(L, 3, fn)
	}
	worldAs[NpcControl](v, L, fn).NpcSetIntention(n, name, arg)
	return 0
}

func (v *vm) npcMostHated(L *lua.LState) int {
	n := checkNpc(L, "most_hated")
	L.Push(v.objectValue(L, worldAs[NpcControl](v, L, "npc:most_hated").NpcMostHated(n)))
	return 1
}

// npcHateList is npc:hate_list(): { {target =, hate =, damage =}, ... },
// most hated first.
func (v *vm) npcHateList(L *lua.LState) int {
	n := checkNpc(L, "hate_list")
	list := worldAs[NpcControl](v, L, "npc:hate_list").NpcHateList(n)
	t := L.CreateTable(len(list), 0)
	for _, h := range list {
		target := v.objectValue(L, h.Target)
		if target == lua.LNil {
			continue
		}
		row := L.CreateTable(0, 3)
		row.RawSetString("target", target)
		row.RawSetString("hate", lua.LNumber(h.Hate))
		row.RawSetString("damage", lua.LNumber(h.Damage))
		t.Append(row)
	}
	L.Push(t)
	return 1
}

func (v *vm) npcReduceHate(L *lua.LState) int {
	const fn = "npc:reduce_hate"
	n := checkNpc(L, "reduce_hate")
	target := checkObject(L, 2, fn)
	amount := checkInt(L, 3, fn)
	worldAs[NpcControl](v, L, fn).NpcReduceHate(n, target, amount)
	return 0
}

func (v *vm) npcSetHeading(L *lua.LState) int {
	const fn = "npc:set_heading"
	n := checkNpc(L, "set_heading")
	h := int(checkInt(L, 2, fn))
	worldAs[NpcControl](v, L, fn).NpcSetHeading(n, h)
	return 0
}

// --- Spawn data and template parameters ---

// npcSpawnLocation is npc:spawn_location(): the spawn point, nil without
// a spawn.
func (v *vm) npcSpawnLocation(L *lua.LState) int {
	si := npcAs[NpcSpawnInfo](L, "spawn_location", "spawn data")
	loc, ok := si.SpawnLocation()
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(locationTable(L, loc))
	return 1
}

// npcStaysInSpawnLoc is Mobius staysInSpawnLoc: the NPC stands on its
// spawn point's x and y.
func (v *vm) npcStaysInSpawnLoc(L *lua.LState) int {
	si := npcAs[NpcSpawnInfo](L, "stays_in_spawn_loc", "spawn data")
	spawn, ok := si.SpawnLocation()
	here, located := locationOf(si)
	L.Push(lua.LBool(ok && located && spawn.X == here.X && spawn.Y == here.Y))
	return 1
}

func (v *vm) npcAIValue(L *lua.LState) int {
	si := npcAs[NpcSpawnInfo](L, "ai_value", "spawn data")
	if val, ok := si.AIValue(L.CheckString(2)); ok {
		L.Push(lua.LNumber(val))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

func (v *vm) npcHasAIValue(L *lua.LState) int {
	n := checkNpc(L, "has_ai_value")
	si, ok := n.(NpcSpawnInfo)
	has := false
	if ok {
		_, has = si.AIValue(L.CheckString(2))
	}
	L.Push(lua.LBool(has))
	return 1
}

// npcParam is npc:param(name[, default]): a template parameter as a
// string.
func (v *vm) npcParam(L *lua.LState) int {
	tp := npcAs[NpcTemplateParams](L, "param", "template parameters")
	if s, ok := tp.TemplateParam(L.CheckString(2)); ok {
		L.Push(lua.LString(s))
	} else {
		L.Push(L.Get(3))
	}
	return 1
}

// npcParamInt is npc:param_int(name[, default]).
func (v *vm) npcParamInt(L *lua.LState) int {
	tp := npcAs[NpcTemplateParams](L, "param_int", "template parameters")
	def := optInt(L, 3, "npc:param_int", 0)
	s, ok := tp.TemplateParam(L.CheckString(2))
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if !ok || err != nil {
		n = def
	}
	L.Push(lua.LNumber(n))
	return 1
}

// --- Leaders and minions ---

func (v *vm) npcLeader(L *lua.LState) int {
	n := checkNpc(L, "leader")
	L.Push(v.newNpc(L, worldAs[NpcMinions](v, L, "npc:leader").NpcLeader(n)))
	return 1
}

func (v *vm) npcMinions(L *lua.LState) int {
	n := checkNpc(L, "minions")
	L.Push(v.npcList(L, worldAs[NpcMinions](v, L, "npc:minions").NpcMinionsOf(n)))
	return 1
}

func (v *vm) npcHasMinions(L *lua.LState) int {
	n := checkNpc(L, "has_minions")
	L.Push(lua.LBool(len(worldAs[NpcMinions](v, L, "npc:has_minions").NpcMinionsOf(n)) > 0))
	return 1
}

func (v *vm) npcAddMinion(L *lua.LState) int {
	const fn = "npc:add_minion"
	n := checkNpc(L, "add_minion")
	id := int(checkInt(L, 2, fn))
	L.Push(v.newNpc(L, worldAs[NpcMinions](v, L, fn).NpcAddMinion(n, id)))
	return 1
}

// npcList returns NPCs as a Lua list.
func (v *vm) npcList(L *lua.LState, npcs []NpcRef) *lua.LTable {
	t := L.CreateTable(len(npcs), 0)
	for _, n := range npcs {
		if n != nil {
			t.Append(v.newNpc(L, n))
		}
	}
	return t
}

// --- Script events ---

// npcBroadcastEvent is npc:broadcast_event(name, radius, reference):
// on_event_received for every NPC within radius whose template a script
// listens for (Mobius broadcastEvent). The handlers run after this one.
func (v *vm) npcBroadcastEvent(L *lua.LState) int {
	const fn = "npc:broadcast_event"
	n := checkNpc(L, "broadcast_event")
	name := L.CheckString(2)
	radius := int(checkInt(L, 3, fn))
	reference := objectArg(L, 4)
	e := v.engine
	scan := worldAs[WorldScan](v, L, fn)
	if e == nil {
		return 0
	}
	for _, other := range scan.NpcsAround(n, radius) {
		if other == nil || other.ObjectID() == n.ObjectID() || !e.listens(hookEventReceived, other.ID()) {
			continue
		}
		_ = e.EventReceived(context.Background(), name, n, other, reference)
	}
	return 0
}

// npcSendEvent is npc:send_event(name, receiver, reference) (Mobius
// sendScriptEvent).
func (v *vm) npcSendEvent(L *lua.LState) int {
	const fn = "npc:send_event"
	n := checkNpc(L, "send_event")
	name := L.CheckString(2)
	receiver := optNpcArg(L, 3, fn)
	reference := objectArg(L, 4)
	v.needWorld(L, fn)
	if receiver != nil && v.engine != nil {
		_ = v.engine.EventReceived(context.Background(), name, n, receiver, reference)
	}
	return 0
}

// --- Chests ---

func (v *vm) chestGetter(method string, get func(ChestActions, NpcRef) bool) lua.LGFunction {
	return func(L *lua.LState) int {
		n := checkNpc(L, method)
		L.Push(lua.LBool(get(worldAs[ChestActions](v, L, "npc:"+method), n)))
		return 1
	}
}

func (v *vm) chestAction(method string, do func(ChestActions, NpcRef)) lua.LGFunction {
	return func(L *lua.LState) int {
		n := checkNpc(L, method)
		do(worldAs[ChestActions](v, L, "npc:"+method), n)
		return 0
	}
}

// --- Zones and distances ---

func (v *vm) npcIsInsideZone(L *lua.LState) int {
	const fn = "npc:is_inside_zone"
	n := checkNpc(L, "is_inside_zone")
	id := int(checkInt(L, 2, fn))
	L.Push(lua.LBool(worldAs[ZoneControl](v, L, fn).ZoneIsInside(id, n)))
	return 1
}

// distanceMethod returns x:calculate_distance_2d/3d(other) for a Player or
// an Npc; other may be a position table.
func (v *vm) distanceMethod(kind string, withZ bool) lua.LGFunction {
	return func(L *lua.LState) int {
		var self any
		method := "calculate_distance_2d"
		if withZ {
			method = "calculate_distance_3d"
		}
		if kind == "npc" {
			self = checkNpc(L, method)
		} else {
			self = checkPlayer(L, method)
		}
		a, ok := locationOf(self)
		if !ok {
			L.RaiseError("%s:%s(): this %s has no position", kind, method, kind)
		}
		b := locationArg(L, 2, kind+":"+method)
		L.Push(lua.LNumber(distance(a, b, withZ)))
		return 1
	}
}

// locationTable returns a position table {x =, y =, z =, heading =}.
func locationTable(L *lua.LState, loc Location) *lua.LTable {
	t := L.CreateTable(0, 4)
	t.RawSetString("x", lua.LNumber(loc.X))
	t.RawSetString("y", lua.LNumber(loc.Y))
	t.RawSetString("z", lua.LNumber(loc.Z))
	t.RawSetString("heading", lua.LNumber(loc.Heading))
	return t
}

// --- Player and Skill additions ---

// playerControlMethods are the Player methods of the AI API.
func (v *vm) playerControlMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"hp": func(L *lua.LState) int {
			L.Push(lua.LNumber(playerAs[PlayerHP](L, "hp", "HP").CurrentHP()))
			return 1
		},
		"max_hp": func(L *lua.LState) int {
			L.Push(lua.LNumber(playerAs[PlayerHP](L, "max_hp", "HP").MaxHP()))
			return 1
		},
		"summon": func(L *lua.LState) int {
			p := checkPlayer(L, "summon")
			if s, ok := p.(PlayerSummon); ok {
				L.Push(v.objectValue(L, s.SummonObject()))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"is_inside_zone": func(L *lua.LState) int {
			const fn = "player:is_inside_zone"
			p := checkPlayer(L, "is_inside_zone")
			id := int(checkInt(L, 2, fn))
			L.Push(lua.LBool(worldAs[ZoneControl](v, L, fn).ZoneIsInside(id, p)))
			return 1
		},
		"calculate_distance_2d": v.distanceMethod("player", false),
		"calculate_distance_3d": v.distanceMethod("player", true),
	}
}

// skillTargetType is skill:target_type().
func (v *vm) skillTargetType(L *lua.LState) int {
	sk := valueOf[Skill](L, "Skill", "target_type")
	L.Push(lua.LString(worldAs[SkillTargets](v, L, "skill:target_type").SkillTargetType(sk)))
	return 1
}
