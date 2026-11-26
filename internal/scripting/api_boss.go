package scripting

import (
	"time"

	lua "github.com/yuin/gopher-lua"
)

// The boss scripts' part of the AI API (Plan 13, Phase 10.7): what Mobius's
// grand boss scripts call beyond NpcControl and NpcMinions.

// BossActions are the NPC, player and skill details the grand boss scripts
// use. Without it the Lua calls raise "this server cannot do that yet".
type BossActions interface {
	// NpcUseMagic is Mobius Attackable.useMagic(skill): the NPC casts the
	// skill at its target when it can (Queen Ant's nurses).
	NpcUseMagic(npc NpcRef, skill Skill)
	// NpcTarget is Mobius getTarget: a PlayerRef, an NpcRef or nil.
	NpcTarget(npc NpcRef) any
	// NpcSetHPMP is Mobius setCurrentHpMp.
	NpcSetHPMP(npc NpcRef, hp, mp float64)
	// NpcMP returns the current and the maximum MP (getCurrentMp, getMaxMp).
	NpcMP(npc NpcRef) (mp, maxMP float64)
	// NpcMinionRespawn is Mobius getLeader().getMinionList().onMinionDie(
	// minion, ms): the dead minion comes back after d while its leader
	// lives, instead of after the default time. Nothing for an NPC without a
	// leader.
	NpcMinionRespawn(minion NpcRef, d time.Duration)
	// NpcSetSpawnLocation is Mobius getSpawn().setLocation(loc): where the
	// NPC returns home to (and respawns, for an NPC with a spawn).
	NpcSetSpawnLocation(npc NpcRef, loc Location)
	// NpcSetIntentionCast is Mobius getAI().setIntentionCast(skill,
	// target): the NPC walks into range and casts.
	NpcSetIntentionCast(npc NpcRef, skill Skill, target any)
	// SkillEffectPoint is skill.getEffectPoint(); SkillCastRange
	// getCastRange().
	SkillEffectPoint(skill Skill) int
	SkillCastRange(skill Skill) int
	// PlayerDie is Mobius doDie(killer) for a player; killer is a PlayerRef,
	// an NpcRef or nil (the player itself).
	PlayerDie(p PlayerRef, killer any)
	// PlayerTeleportToTown is Mobius teleToLocation(TeleportWhereType.TOWN):
	// the nearest town's restart point.
	PlayerTeleportToTown(p PlayerRef)
	// PlayerIsGM is isGM; PlayerIsInvisible isInvisible; PlayerIsHero
	// isHero.
	PlayerIsGM(p PlayerRef) bool
	PlayerIsInvisible(p PlayerRef) bool
	PlayerIsHero(p PlayerRef) bool
}

// npcBossMethods are the Npc methods of the boss scripts.
func (v *vm) npcBossMethods() map[string]lua.LGFunction {
	boss := func(L *lua.LState, method string) (NpcRef, BossActions) {
		n := checkNpc(L, method)
		return n, worldAs[BossActions](v, L, "npc:"+method)
	}
	return map[string]lua.LGFunction{
		"use_magic": func(L *lua.LState) int {
			n, b := boss(L, "use_magic")
			if sk, ok := optSkillArg(L, 2, "npc:use_magic"); ok {
				b.NpcUseMagic(n, sk)
			}
			return 0
		},
		"target": func(L *lua.LState) int {
			n, b := boss(L, "target")
			L.Push(v.objectValue(L, b.NpcTarget(n)))
			return 1
		},
		"set_hp_mp": func(L *lua.LState) int {
			n, b := boss(L, "set_hp_mp")
			b.NpcSetHPMP(n, float64(L.CheckNumber(2)), float64(L.CheckNumber(3)))
			return 0
		},
		"mp": func(L *lua.LState) int {
			n, b := boss(L, "mp")
			mp, _ := b.NpcMP(n)
			L.Push(lua.LNumber(mp))
			return 1
		},
		"max_mp": func(L *lua.LState) int {
			n, b := boss(L, "max_mp")
			_, maxMP := b.NpcMP(n)
			L.Push(lua.LNumber(maxMP))
			return 1
		},
		"minion_respawn": func(L *lua.LState) int {
			n, b := boss(L, "minion_respawn")
			ms := checkInt(L, 2, "npc:minion_respawn")
			b.NpcMinionRespawn(n, time.Duration(ms)*time.Millisecond)
			return 0
		},
		"set_spawn_location": func(L *lua.LState) int {
			n, b := boss(L, "set_spawn_location")
			b.NpcSetSpawnLocation(n, locationArg(L, 2, "npc:set_spawn_location"))
			return 0
		},
		"set_intention_cast": func(L *lua.LState) int {
			const fn = "npc:set_intention_cast"
			n, b := boss(L, "set_intention_cast")
			sk, ok := optSkillArg(L, 2, fn)
			target := checkObject(L, 3, fn)
			if ok {
				b.NpcSetIntentionCast(n, sk, target)
			}
			return 0
		},
		"effect_level": func(L *lua.LState) int {
			const fn = "npc:effect_level"
			n := checkNpc(L, "effect_level")
			id := skillIDArg(L, 2, fn)
			if level, ok := worldAs[EffectLevels](v, L, fn).EffectLevel(n, id); ok {
				L.Push(lua.LNumber(level))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"is_affected_by_skill": func(L *lua.LState) int {
			const fn = "npc:is_affected_by_skill"
			n := checkNpc(L, "is_affected_by_skill")
			id := skillIDArg(L, 2, fn)
			L.Push(lua.LBool(worldAs[QuestSkills](v, L, fn).IsAffectedBySkill(n, id)))
			return 1
		},
	}
}

// playerBossMethods are the Player methods of the boss scripts.
func (v *vm) playerBossMethods() map[string]lua.LGFunction {
	flag := func(method string, get func(BossActions, PlayerRef) bool) lua.LGFunction {
		return func(L *lua.LState) int {
			p := checkPlayer(L, method)
			L.Push(lua.LBool(get(worldAs[BossActions](v, L, "player:"+method), p)))
			return 1
		}
	}
	return map[string]lua.LGFunction{
		"is_gm":        flag("is_gm", BossActions.PlayerIsGM),
		"is_invisible": flag("is_invisible", BossActions.PlayerIsInvisible),
		"is_hero":      flag("is_hero", BossActions.PlayerIsHero),
		"die": func(L *lua.LState) int {
			p := checkPlayer(L, "die")
			var killer any
			if L.Get(2) != lua.LNil {
				killer = checkObject(L, 2, "player:die")
			}
			worldAs[BossActions](v, L, "player:die").PlayerDie(p, killer)
			return 0
		},
		"teleport_to_town": func(L *lua.LState) int {
			p := checkPlayer(L, "teleport_to_town")
			worldAs[BossActions](v, L, "player:teleport_to_town").PlayerTeleportToTown(p)
			return 0
		},
	}
}

// skillCastRange is skill:cast_range().
func (v *vm) skillCastRange(L *lua.LState) int {
	sk := valueOf[Skill](L, "Skill", "cast_range")
	L.Push(lua.LNumber(worldAs[BossActions](v, L, "skill:cast_range").SkillCastRange(sk)))
	return 1
}

// skillEffectPoint is skill:effect_point().
func (v *vm) skillEffectPoint(L *lua.LState) int {
	sk := valueOf[Skill](L, "Skill", "effect_point")
	L.Push(lua.LNumber(worldAs[BossActions](v, L, "skill:effect_point").SkillEffectPoint(sk)))
	return 1
}
