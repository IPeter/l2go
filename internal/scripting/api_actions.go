package scripting

import (
	"math"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// The NPC AI, skill and item parts of the quest API: the Npc and Player
// methods over NpcActions, QuestSkills, NpcStatus, PlayerCombat and
// PlayerNoble, and the skills and items modules.

// worldAs returns the quest world as T for fn, or raises an error saying
// the server cannot do what yet.
func worldAs[T any](v *vm, L *lua.LState, fn string) T {
	w := v.needWorld(L, fn)
	x, ok := w.(T)
	if !ok {
		L.RaiseError("%s: this server cannot do that yet", fn)
	}
	return x
}

// checkObject returns argument n, a Player or an Npc (as PlayerRef or
// NpcRef).
func checkObject(L *lua.LState, n int, fn string) any {
	if obj := objectArg(L, n); obj != nil {
		return obj
	}
	if ud, ok := L.Get(n).(*lua.LUserData); ok {
		switch ud.Value.(type) {
		case *playerBox, *npcBox:
			L.ArgError(n, fn+": this object was passed to an earlier handler; keep its object_id() instead")
			return nil
		}
	}
	L.ArgError(n, fn+": a player or an NPC expected, got "+L.Get(n).Type().String())
	return nil
}

// optSkillArg returns argument n, a Skill; false for nil.
func optSkillArg(L *lua.LState, n int, fn string) (Skill, bool) {
	switch x := L.Get(n).(type) {
	case *lua.LNilType:
		return Skill{}, false
	case *lua.LUserData:
		if s, ok := x.Value.(Skill); ok {
			return s, true
		}
	}
	L.ArgError(n, fn+": a skill (from skills.get) expected, got "+L.Get(n).Type().String())
	return Skill{}, false
}

// skillIDArg returns argument n, a skill ID or a Skill (its ID).
func skillIDArg(L *lua.LState, n int, fn string) int {
	if ud, ok := L.Get(n).(*lua.LUserData); ok {
		if s, ok := ud.Value.(Skill); ok {
			return s.ID
		}
	}
	return int(checkInt(L, n, fn))
}

// moveLocationArg reads the position of npc:move_to: x, y, z numbers, a
// position table, a Player or an Npc.
func moveLocationArg(L *lua.LState, fn string) Location {
	if L.Get(2).Type() == lua.LTNumber {
		return Location{X: int(checkInt(L, 2, fn)), Y: int(checkInt(L, 3, fn)), Z: int(checkInt(L, 4, fn))}
	}
	return locationArg(L, 2, fn)
}

// magicSkillUse returns x:magic_skill_use(target, skill_id, level[,
// hit_time_ms, reuse_delay_ms]) for a Player or an Npc.
func (v *vm) magicSkillUse(self func(L *lua.LState, method string) any, kind string) lua.LGFunction {
	return func(L *lua.LState) int {
		fn := kind + ":magic_skill_use"
		caster := self(L, "magic_skill_use")
		target := checkObject(L, 2, fn)
		id := int(checkInt(L, 3, fn))
		level := int(checkInt(L, 4, fn))
		hit := optInt(L, 5, fn, 0)
		reuse := optInt(L, 6, fn, 0)
		worldAs[NpcActions](v, L, fn).MagicSkillUse(caster, target, id, level,
			time.Duration(hit)*time.Millisecond, time.Duration(reuse)*time.Millisecond)
		return 0
	}
}

// npcAIMethods are the Npc methods of the AI, status, spawn and skills.
func (v *vm) npcAIMethods() map[string]lua.LGFunction {
	status := func(method string, get func(NpcStatus) lua.LValue) lua.LGFunction {
		return func(L *lua.LState) int {
			L.Push(get(npcAs[NpcStatus](L, method, "status")))
			return 1
		}
	}
	plain := func(method string, do func(a NpcActions, n NpcRef)) lua.LGFunction {
		return func(L *lua.LState) int {
			n := checkNpc(L, method)
			do(worldAs[NpcActions](v, L, "npc:"+method), n)
			return 0
		}
	}
	withTarget := func(method string, do func(a NpcActions, n NpcRef, target any)) lua.LGFunction {
		return func(L *lua.LState) int {
			n := checkNpc(L, method)
			target := checkObject(L, 2, "npc:"+method)
			do(worldAs[NpcActions](v, L, "npc:"+method), n, target)
			return 0
		}
	}
	return map[string]lua.LGFunction{
		"level":  status("level", func(s NpcStatus) lua.LValue { return lua.LNumber(s.Level()) }),
		"hp":     status("hp", func(s NpcStatus) lua.LValue { return lua.LNumber(s.CurrentHP()) }),
		"max_hp": status("max_hp", func(s NpcStatus) lua.LValue { return lua.LNumber(s.MaxHP()) }),
		"attack": withTarget("attack", func(a NpcActions, n NpcRef, t any) { a.NpcAttack(n, t) }),
		"add_damage_hate": func(L *lua.LState) int {
			const fn = "npc:add_damage_hate"
			n := checkNpc(L, "add_damage_hate")
			target := checkObject(L, 2, fn)
			damage, hate := checkInt(L, 3, fn), checkInt(L, 4, fn)
			worldAs[NpcActions](v, L, fn).NpcAddDamageHate(n, target, damage, hate)
			return 0
		},
		"remove_attacker": withTarget("remove_attacker", func(a NpcActions, n NpcRef, t any) { a.NpcRemoveAttacker(n, t) }),
		"abort_attack":    plain("abort_attack", func(a NpcActions, n NpcRef) { a.NpcAbortAttack(n) }),
		"set_running":     plain("set_running", func(a NpcActions, n NpcRef) { a.NpcSetRunning(n) }),
		"stop_respawn":    plain("stop_respawn", func(a NpcActions, n NpcRef) { a.NpcSetRespawn(n, false) }),
		"start_respawn":   plain("start_respawn", func(a NpcActions, n NpcRef) { a.NpcSetRespawn(n, true) }),
		"move_to": func(L *lua.LState) int {
			n := checkNpc(L, "move_to")
			loc := moveLocationArg(L, "npc:move_to")
			worldAs[NpcActions](v, L, "npc:move_to").NpcMoveTo(n, loc)
			return 0
		},
		"die": func(L *lua.LState) int {
			n := checkNpc(L, "die")
			var killer any = n
			if L.Get(2) != lua.LNil {
				killer = checkObject(L, 2, "npc:die")
			}
			worldAs[NpcActions](v, L, "npc:die").NpcDie(n, killer)
			return 0
		},
		"cast": func(L *lua.LState) int {
			const fn = "npc:cast"
			n := checkNpc(L, "cast")
			sk, ok := optSkillArg(L, 2, fn)
			var target any
			if L.Get(3) != lua.LNil {
				target = checkObject(L, 3, fn)
			}
			a := worldAs[NpcActions](v, L, fn)
			if ok {
				a.NpcCast(n, sk, target)
			}
			return 0
		},
		"magic_skill_use": v.magicSkillUse(func(L *lua.LState, method string) any { return checkNpc(L, method) }, "npc"),
		"set_target": func(L *lua.LState) int {
			n := checkNpc(L, "set_target")
			var target any
			if L.Get(2) != lua.LNil {
				target = checkObject(L, 2, "npc:set_target")
			}
			worldAs[NpcActions](v, L, "npc:set_target").NpcSetTarget(n, target)
			return 0
		},
		"teleport": func(L *lua.LState) int {
			n := checkNpc(L, "teleport")
			loc := moveLocationArg(L, "npc:teleport")
			worldAs[NpcActions](v, L, "npc:teleport").NpcTeleport(n, loc)
			return 0
		},
		"drop_item": func(L *lua.LState) int {
			const fn = "npc:drop_item"
			n := checkNpc(L, "drop_item")
			p := checkPlayerArg(L, 2, fn)
			id := int(checkInt(L, 3, fn))
			count := optInt(L, 4, fn, 1)
			a := worldAs[NpcActions](v, L, fn)
			if v.validItem(L, fn, id, count) {
				a.NpcDropItem(n, p, id, count)
			}
			return 0
		},
	}
}

// playerAIMethods are the Player methods of skills, the weapon, the target
// and noblesse.
func (v *vm) playerAIMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"magic_skill_use": v.magicSkillUse(func(L *lua.LState, method string) any { return checkPlayer(L, method) }, "player"),
		"is_affected_by_skill": func(L *lua.LState) int {
			const fn = "player:is_affected_by_skill"
			p := checkPlayer(L, "is_affected_by_skill")
			id := skillIDArg(L, 2, fn)
			L.Push(lua.LBool(worldAs[QuestSkills](v, L, fn).IsAffectedBySkill(p, id)))
			return 1
		},
		"active_weapon": func(L *lua.LState) int {
			if id := playerAs[PlayerCombat](L, "active_weapon", "weapon").ActiveWeaponID(); id > 0 {
				L.Push(lua.LNumber(id))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"target": func(L *lua.LState) int {
			L.Push(v.objectValue(L, playerAs[PlayerCombat](L, "target", "target").TargetObject()))
			return 1
		},
		"is_noble": func(L *lua.LState) int {
			p := checkPlayer(L, "is_noble")
			nb, ok := p.(PlayerNoble)
			L.Push(lua.LBool(ok && nb.IsNoble()))
			return 1
		},
		"set_noble": func(L *lua.LState) int {
			nb := playerAs[PlayerNoble](L, "set_noble", "noblesse")
			nb.SetNoble(L.ToBool(2))
			return 0
		},
		"pk_kills": func(L *lua.LState) int {
			L.Push(lua.LNumber(playerAs[PlayerStatus](L, "pk_kills", "status").PkKills()))
			return 1
		},
		"set_pk_kills": func(L *lua.LState) int {
			ps := playerAs[PlayerStatus](L, "set_pk_kills", "status")
			n := checkInt(L, 2, "player:set_pk_kills")
			if n < 0 || n > math.MaxInt32 {
				L.ArgError(2, "player:set_pk_kills: the count must be from 0 to 2147483647")
			}
			ps.SetPkKills(int(n))
			return 0
		},
		"update_user_info": func(L *lua.LState) int {
			playerAs[PlayerStatus](L, "update_user_info", "status").UpdateUserInfo()
			return 0
		},
		"active_quests": func(L *lua.LState) int {
			ids := playerAs[PlayerStatus](L, "active_quests", "status").ActiveQuests()
			t := L.CreateTable(len(ids), 0)
			for i, id := range ids {
				t.RawSetInt(i+1, lua.LNumber(id))
			}
			L.Push(t)
			return 1
		},
	}
}

// skillsFuncs is the skills module: skills.get(id, level) and
// skills.apply_effects(skill, effector, effected).
func (v *vm) skillsFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			const fn = "skills.get"
			id := int(checkInt(L, 1, fn))
			level := int(optInt(L, 2, fn, 1))
			if worldAs[QuestSkills](v, L, fn).SkillExists(id, level) {
				L.Push(newValue(L, Skill{ID: id, Level: level}, skillType))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"apply_effects": func(L *lua.LState) int {
			const fn = "skills.apply_effects"
			sk, ok := optSkillArg(L, 1, fn)
			effector := checkObject(L, 2, fn)
			effected := checkObject(L, 3, fn)
			w := worldAs[QuestSkills](v, L, fn)
			if ok {
				w.ApplyEffects(sk, effector, effected)
			}
			return 0
		},
	}
}

// itemsFuncs is the items module: items.get(id).
func (v *vm) itemsFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			const fn = "items.get"
			id := int(checkInt(L, 1, fn))
			t, ok := worldAs[QuestItems](v, L, fn).ItemTemplate(id)
			if !ok {
				L.Push(lua.LNil)
				return 1
			}
			tbl := L.CreateTable(0, 3)
			tbl.RawSetString("id", lua.LNumber(t.ID))
			tbl.RawSetString("name", lua.LString(t.Name))
			tbl.RawSetString("stackable", lua.LBool(t.Stackable))
			L.Push(tbl)
			return 1
		},
	}
}
