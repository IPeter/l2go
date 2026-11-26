package scripting

import lua "github.com/yuin/gopher-lua"

// The calls the monster scripts of Plan 13 group A (Mobius ai/others) need
// beyond the AI API: lethal strikes, Silent Move, the world region, the
// template's minion lists, cursed weapons and telling players from NPCs.

// MonsterFlags is the part of a QuestWorld behind npc:set_lethalable,
// npc:set_see_through_silent_move and npc:is_in_active_region.
type MonsterFlags interface {
	// NpcSetLethalable is Mobius setLethalable: false stops lethal strikes
	// on the NPC.
	NpcSetLethalable(npc NpcRef, on bool)
	// NpcSetSeeThroughSilentMove is Mobius
	// Attackable.setSeeThroughSilentMove: the monster is aggressive towards
	// players in Silent Move too.
	NpcSetSeeThroughSilentMove(npc NpcRef, on bool)
	// NpcInActiveRegion is Mobius getWorldRegion().isActive(): players are
	// near.
	NpcInActiveRegion(npc NpcRef) bool
}

// TemplateMinion is one entry of a template's minion list.
type TemplateMinion struct {
	ID, Count int
}

// NpcTemplateMinions is the part of an NpcRef behind
// npc:param_minions(name) (Mobius getParameters().getMinionList(name)).
type NpcTemplateMinions interface {
	TemplateMinions(name string) []TemplateMinion
}

// PlayerCursedWeapon is the part of a PlayerRef behind
// player:is_cursed_weapon_equipped() (Mobius isCursedWeaponEquipped).
// Without it the answer is false.
type PlayerCursedWeapon interface {
	CursedWeaponEquipped() bool
}

// npcMonsterMethods are the Npc methods of this file.
func (v *vm) npcMonsterMethods() map[string]lua.LGFunction {
	flag := func(method string, set func(MonsterFlags, NpcRef, bool)) lua.LGFunction {
		return func(L *lua.LState) int {
			n := checkNpc(L, method)
			on := true
			if L.GetTop() >= 2 {
				on = L.ToBool(2)
			}
			set(worldAs[MonsterFlags](v, L, "npc:"+method), n, on)
			return 0
		}
	}
	return map[string]lua.LGFunction{
		"set_lethalable":              flag("set_lethalable", MonsterFlags.NpcSetLethalable),
		"set_see_through_silent_move": flag("set_see_through_silent_move", MonsterFlags.NpcSetSeeThroughSilentMove),
		"is_in_active_region": func(L *lua.LState) int {
			n := checkNpc(L, "is_in_active_region")
			L.Push(lua.LBool(worldAs[MonsterFlags](v, L, "npc:is_in_active_region").NpcInActiveRegion(n)))
			return 1
		},
		"param_minions": func(L *lua.LState) int {
			tm := npcAs[NpcTemplateMinions](L, "param_minions", "template minion lists")
			list := tm.TemplateMinions(L.CheckString(2))
			t := L.CreateTable(len(list), 0)
			for _, m := range list {
				row := L.CreateTable(0, 2)
				row.RawSetString("id", lua.LNumber(m.ID))
				row.RawSetString("count", lua.LNumber(m.Count))
				t.Append(row)
			}
			L.Push(t)
			return 1
		},
		"is_player": func(L *lua.LState) int {
			checkNpc(L, "is_player")
			L.Push(lua.LFalse)
			return 1
		},
	}
}

// playerMonsterMethods are the Player methods of this file.
func playerMonsterMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"is_cursed_weapon_equipped": func(L *lua.LState) int {
			p := checkPlayer(L, "is_cursed_weapon_equipped")
			cw, ok := p.(PlayerCursedWeapon)
			L.Push(lua.LBool(ok && cw.CursedWeaponEquipped()))
			return 1
		},
		"is_player": func(L *lua.LState) int {
			checkPlayer(L, "is_player")
			L.Push(lua.LTrue)
			return 1
		},
	}
}
