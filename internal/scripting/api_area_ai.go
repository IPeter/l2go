package scripting

import (
	lua "github.com/yuin/gopher-lua"
)

// The calls of the area scripts of Plan 13 group B's second half (Mobius
// ai/areas from KetraOrcOutpust on) beyond the AI API: an NPC's swing,
// skill reuse and ranges, the template's skill parameters, a player's
// combat stance, the teleport lists, sight between two objects and a
// skill's effect types. ai_areas.go has the first half's calls, api_boss.go
// npc:target, npc:set_hp_mp and npc:max_mp.

// AreaAI is the part of the QuestWorld these area scripts need.
type AreaAI interface {
	// NpcIsAttackingNow is Mobius isAttackingNow: in the middle of a swing.
	NpcIsAttackingNow(npc NpcRef) bool
	// NpcIsSkillDisabled is Mobius isSkillDisabled(skill): under reuse, or
	// all skills disabled.
	NpcIsSkillDisabled(npc NpcRef, skill Skill) bool
	// NpcAggroRange is Mobius getAggroRange; NpcClanHelpRange is
	// getTemplate().getClanHelpRange().
	NpcAggroRange(npc NpcRef) int
	NpcClanHelpRange(npc NpcRef) int
	// PlayerIsInCombat is Mobius isInCombat (the attack stance).
	PlayerIsInCombat(p PlayerRef) bool
	// ShowTeleportList is Mobius TeleporterData.getHolder(npc's template,
	// list).showTeleportList(player, npc); false when the NPC has no such
	// list.
	ShowTeleportList(p PlayerRef, npc NpcRef, list string) bool
	// CanSeeTarget is Mobius GeoEngine.canSeeTarget(from, to); from and to
	// are PlayerRefs or NpcRefs.
	CanSeeTarget(from, to any) bool
	// SkillHasEffectType is Mobius skill.hasEffectType(EffectType.X) by the
	// type's name, e.g. "AGGRESSION".
	SkillHasEffectType(skill Skill, effectType string) bool
}

// NpcTemplateSkills is the part of an NpcRef behind the template's skill
// parameters (Mobius getParameters().getObject(name, SkillHolder.class)).
type NpcTemplateSkills interface {
	TemplateSkill(name string) (Skill, bool)
}

// npcAreaMethods are the Npc methods of these area scripts.
func (v *vm) npcAreaMethods() map[string]lua.LGFunction {
	area := func(L *lua.LState, method string) (NpcRef, AreaAI) {
		n := checkNpc(L, method)
		return n, worldAs[AreaAI](v, L, "npc:"+method)
	}
	return map[string]lua.LGFunction{
		"is_attacking_now": func(L *lua.LState) int {
			n, a := area(L, "is_attacking_now")
			L.Push(lua.LBool(a.NpcIsAttackingNow(n)))
			return 1
		},
		"is_skill_disabled": func(L *lua.LState) int {
			n, a := area(L, "is_skill_disabled")
			sk, ok := optSkillArg(L, 2, "npc:is_skill_disabled")
			// A skill the data lacks (nil) cannot be used.
			L.Push(lua.LBool(!ok || a.NpcIsSkillDisabled(n, sk)))
			return 1
		},
		"aggro_range": func(L *lua.LState) int {
			n, a := area(L, "aggro_range")
			L.Push(lua.LNumber(a.NpcAggroRange(n)))
			return 1
		},
		"clan_help_range": func(L *lua.LState) int {
			n, a := area(L, "clan_help_range")
			L.Push(lua.LNumber(a.NpcClanHelpRange(n)))
			return 1
		},
		"show_teleport_list": func(L *lua.LState) int {
			n, a := area(L, "show_teleport_list")
			p := checkPlayerArg(L, 2, "npc:show_teleport_list")
			L.Push(lua.LBool(a.ShowTeleportList(p, n, L.CheckString(3))))
			return 1
		},
		"param_skill": func(L *lua.LState) int {
			ts := npcAs[NpcTemplateSkills](L, "param_skill", "template skills")
			sk, ok := ts.TemplateSkill(L.CheckString(2))
			if !ok || !worldAs[QuestSkills](v, L, "npc:param_skill").SkillExists(sk.ID, sk.Level) {
				L.Push(lua.LNil)
				return 1
			}
			L.Push(newValue(L, sk, skillType))
			return 1
		},
	}
}

// playerAreaMethods are the Player methods of these area scripts.
func (v *vm) playerAreaMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"is_in_combat": func(L *lua.LState) int {
			p := checkPlayer(L, "is_in_combat")
			L.Push(lua.LBool(worldAs[AreaAI](v, L, "player:is_in_combat").PlayerIsInCombat(p)))
			return 1
		},
	}
}

// worldAreaFuncs are the world functions of these area scripts.
func (v *vm) worldAreaFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"can_see_target": func(L *lua.LState) int {
			const fn = "world.can_see_target"
			from := checkObject(L, 1, fn)
			to := checkObject(L, 2, fn)
			L.Push(lua.LBool(worldAs[AreaAI](v, L, fn).CanSeeTarget(from, to)))
			return 1
		},
	}
}

// skillHasEffectType is skill:has_effect_type(name).
func (v *vm) skillHasEffectType(L *lua.LState) int {
	sk := valueOf[Skill](L, "Skill", "has_effect_type")
	name := L.CheckString(2)
	L.Push(lua.LBool(worldAs[AreaAI](v, L, "skill:has_effect_type").SkillHasEffectType(sk, name)))
	return 1
}
