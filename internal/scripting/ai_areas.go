package scripting

import (
	lua "github.com/yuin/gopher-lua"
)

// The calls the area scripts of Plan 13, Phase 10.7 (Mobius ai/areas)
// need beyond the Phase 10.6 API: skill kinds and cast checks, effect
// levels, the weight penalty, and zones as scripts query them (NPCs inside,
// the zones at a point, zone parameters, enabling a zone).

// SkillKinds tells what kind a skill is (Mobius Skill.isMagic).
type SkillKinds interface {
	SkillIsMagic(skill Skill) bool
}

// NpcCasting is the cast side of an NPC beyond NpcActions.
type NpcCasting interface {
	// NpcCheckDoCastConditions is Mobius checkDoCastConditions(skill): the
	// skill is not disabled, the NPC has the MP and HP it takes and is not
	// muted for it.
	NpcCheckDoCastConditions(npc NpcRef, skill Skill) bool
	// NpcStopSkillEffects is Mobius stopSkillEffects(REMOVED, skillId).
	NpcStopSkillEffects(npc NpcRef, skillID int)
}

// EffectLevels gives the level of a skill's effect on a creature (Mobius
// getEffectList().getBuffInfoBySkillId(id).getSkill().getLevel()); false
// when the skill's effects are not on it.
type EffectLevels interface {
	EffectLevel(obj any, skillID int) (int, bool)
}

// PlayerWeight is the part of a PlayerRef behind player:weight_penalty()
// (Mobius getWeightPenalty, 0 to 4).
type PlayerWeight interface {
	WeightPenalty() int32
}

// ZoneData are the zones as area scripts query them (Mobius ZoneManager and
// ZoneType).
type ZoneData interface {
	// ZoneNpcs returns the NPCs inside the zone (Mobius getCharactersInside
	// without the players).
	ZoneNpcs(zoneID int) []NpcRef
	// ZonesAt returns the zones containing loc, in zone ID order (Mobius
	// ZoneManager.getZones(x, y, z)).
	ZonesAt(loc Location) []Zone
	// ZoneParam returns a parameter of the zone's data, e.g. "skillIdLvl";
	// false when it has none.
	ZoneParam(zoneID int, name string) (string, bool)
	// SetZoneEnabled is Mobius ZoneType.setEnabled.
	SetZoneEnabled(zoneID int, enabled bool)
}

// areaNpcMethods are the Npc methods of the area scripts.
func (v *vm) areaNpcMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"check_do_cast_conditions": func(L *lua.LState) int {
			const fn = "npc:check_do_cast_conditions"
			n := checkNpc(L, "check_do_cast_conditions")
			sk, ok := optSkillArg(L, 2, fn)
			c := worldAs[NpcCasting](v, L, fn)
			L.Push(lua.LBool(ok && c.NpcCheckDoCastConditions(n, sk)))
			return 1
		},
		"stop_skill_effects": func(L *lua.LState) int {
			const fn = "npc:stop_skill_effects"
			n := checkNpc(L, "stop_skill_effects")
			id := skillIDArg(L, 2, fn)
			worldAs[NpcCasting](v, L, fn).NpcStopSkillEffects(n, id)
			return 0
		},
	}
}

// areaPlayerMethods are the Player methods of the area scripts.
func (v *vm) areaPlayerMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"effect_level": func(L *lua.LState) int {
			const fn = "player:effect_level"
			p := checkPlayer(L, "effect_level")
			id := skillIDArg(L, 2, fn)
			if level, ok := worldAs[EffectLevels](v, L, fn).EffectLevel(p, id); ok {
				L.Push(lua.LNumber(level))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"weight_penalty": func(L *lua.LState) int {
			L.Push(lua.LNumber(playerAs[PlayerWeight](L, "weight_penalty", "weight").WeightPenalty()))
			return 1
		},
	}
}

// skillIsMagic is skill:is_magic().
func (v *vm) skillIsMagic(L *lua.LState) int {
	sk := valueOf[Skill](L, "Skill", "is_magic")
	L.Push(lua.LBool(worldAs[SkillKinds](v, L, "skill:is_magic").SkillIsMagic(sk)))
	return 1
}

// areaZonesFuncs are the zones functions of the area scripts.
func (v *vm) areaZonesFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"npcs_inside": func(L *lua.LState) int {
			const fn = "zones.npcs_inside"
			id := int(checkInt(L, 1, fn))
			L.Push(v.npcList(L, worldAs[ZoneData](v, L, fn).ZoneNpcs(id)))
			return 1
		},
		"at": func(L *lua.LState) int {
			const fn = "zones.at"
			loc := locationArg(L, 1, fn)
			list := worldAs[ZoneData](v, L, fn).ZonesAt(loc)
			t := L.CreateTable(len(list), 0)
			for _, z := range list {
				t.Append(newValue(L, z, zoneType))
			}
			L.Push(t)
			return 1
		},
		"param": func(L *lua.LState) int {
			const fn = "zones.param"
			id := int(checkInt(L, 1, fn))
			name := L.CheckString(2)
			if val, ok := worldAs[ZoneData](v, L, fn).ZoneParam(id, name); ok {
				L.Push(lua.LString(val))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"set_enabled": func(L *lua.LState) int {
			const fn = "zones.set_enabled"
			id := int(checkInt(L, 1, fn))
			worldAs[ZoneData](v, L, fn).SetZoneEnabled(id, L.ToBool(2))
			return 0
		},
	}
}
