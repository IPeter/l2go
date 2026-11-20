package scripting

import (
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// rndFuncs is the rnd module: rnd(n), rnd(a, b) (through its metatable),
// rnd.bool(), rnd.double([max]) and rnd.entry(t), over the engine's random
// source with Mobius Rnd's rules.
func (v *vm) rndFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"bool": func(L *lua.LState) int {
			L.Push(lua.LBool(v.quest.rand.Bool()))
			return 1
		},
		"double": func(L *lua.LState) int {
			if L.Get(1) == lua.LNil {
				L.Push(lua.LNumber(v.quest.rand.Float64()))
			} else {
				L.Push(lua.LNumber(v.quest.rand.GetFloat64(float64(L.CheckNumber(1)))))
			}
			return 1
		},
		"entry": func(L *lua.LState) int {
			t := L.CheckTable(1)
			n := t.Len()
			if n == 0 {
				L.Push(lua.LNil)
				return 1
			}
			L.Push(t.RawGetInt(1 + v.quest.rand.Get(n)))
			return 1
		},
	}
}

// rndCall is rnd(n) and rnd(a, b) (called as __call, so argument 1 is the
// rnd table): Mobius getRandom(int) and getRandom(int, int).
func (v *vm) rndCall(L *lua.LState) int {
	a := checkInt(L, 2, "rnd")
	if L.GetTop() < 3 {
		if a > math.MaxInt32 {
			L.Push(lua.LNumber(v.quest.rand.GetInt64(a)))
		} else {
			L.Push(lua.LNumber(v.quest.rand.Get(int(a))))
		}
		return 1
	}
	b := checkInt(L, 3, "rnd")
	if a < math.MinInt32 || b > math.MaxInt32 {
		L.Push(lua.LNumber(v.quest.rand.BetweenInt64(a, b)))
	} else {
		L.Push(lua.LNumber(v.quest.rand.Between(int(a), int(b))))
	}
	return 1
}

// mathRandom replaces math.random with the engine's source: () in [0, 1),
// (n) in [1, n], (m, n) in [m, n], as Lua defines them.
func (v *vm) mathRandom(L *lua.LState) int {
	switch L.GetTop() {
	case 0:
		L.Push(lua.LNumber(v.quest.rand.Float64()))
	case 1:
		n := L.CheckInt(1)
		if n < 1 {
			L.ArgError(1, "interval is empty")
			return 0
		}
		L.Push(lua.LNumber(1 + v.quest.rand.Get(n)))
	default:
		lo, hi := L.CheckInt(1), L.CheckInt(2)
		if lo > hi {
			L.ArgError(2, "interval is empty")
			return 0
		}
		L.Push(lua.LNumber(v.quest.rand.Between(lo, hi)))
	}
	return 1
}

// timeFuncs is the time module: time.now_ms() on the loop clock.
func (v *vm) timeFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"now_ms": func(L *lua.LState) int {
			L.Push(lua.LNumber(v.needWorld(L, "time.now_ms").Now().UnixMilli()))
			return 1
		},
	}
}

// worldFuncs is the world module: world.find_object(id) and
// world.get_player(id).
func (v *vm) worldFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"find_object": func(L *lua.LState) int {
			id := checkInt(L, 1, "world.find_object")
			L.Push(v.objectValue(L, v.needWorld(L, "world.find_object").FindObject(int32(id))))
			return 1
		},
		"get_player": func(L *lua.LState) int {
			id := checkInt(L, 1, "world.get_player")
			obj := v.needWorld(L, "world.get_player").FindObject(int32(id))
			if p, ok := obj.(PlayerRef); ok {
				L.Push(v.newPlayer(L, p))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
	}
}

// objectValue returns the Lua value of a world object: a Player, an Npc or
// nil.
func (v *vm) objectValue(L *lua.LState, obj any) lua.LValue {
	switch x := obj.(type) {
	case PlayerRef:
		return v.newPlayer(L, x)
	case NpcRef:
		return v.newNpc(L, x)
	}
	return lua.LNil
}

// globalsFuncs is the globals module: Mobius GlobalVariablesManager.
func (v *vm) globalsFuncs() map[string]lua.LGFunction {
	get := func(L *lua.LState, fn string) (string, bool) {
		return v.needWorld(L, fn).GlobalVar(L.CheckString(1))
	}
	number := func(fn string) lua.LGFunction {
		return func(L *lua.LState) int {
			s, ok := get(L, fn)
			def := optInt(L, 2, fn, 0)
			n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
			if !ok || err != nil {
				n = def
			}
			L.Push(lua.LNumber(n))
			return 1
		}
	}
	return map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			if s, ok := get(L, "globals.get"); ok {
				L.Push(lua.LString(s))
			} else {
				L.Push(L.Get(2))
			}
			return 1
		},
		"get_int":  number("globals.get_int"),
		"get_long": number("globals.get_long"),
		"get_bool": func(L *lua.LState) int {
			s, ok := get(L, "globals.get_bool")
			if ok {
				L.Push(lua.LBool(strings.EqualFold(s, "true")))
			} else {
				L.Push(lua.LBool(L.ToBool(2)))
			}
			return 1
		},
		"set": func(L *lua.LState) int {
			name := L.CheckString(1)
			v.needWorld(L, "globals.set").SetGlobalVar(name, valueString(L, 2, "globals.set"))
			return 0
		},
	}
}

// utilFuncs is the util module: util.in_range(range, a, b, include_z).
func (v *vm) utilFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"in_range": func(L *lua.LState) int {
			r := int(checkInt(L, 1, "util.in_range"))
			L.Push(lua.LBool(inRange(r, objectArg(L, 2), objectArg(L, 3), L.ToBool(4))))
			return 1
		},
	}
}

// classFuncs is the ClassId table's helpers; its constants come from
// Options.Constants.Classes.
func (v *vm) classFuncs() map[string]lua.LGFunction {
	info := func(L *lua.LState, fn string) (ClassInfo, bool) {
		id := int(checkInt(L, 1, fn))
		for _, c := range v.quest.consts.Classes {
			if c.ID == id {
				return c, true
			}
		}
		return ClassInfo{}, false
	}
	return map[string]lua.LGFunction{
		"level": func(L *lua.LState) int {
			c, ok := info(L, "ClassId.level")
			if !ok {
				L.ArgError(1, "ClassId.level: unknown class")
			}
			L.Push(lua.LNumber(c.Level))
			return 1
		},
		"race": func(L *lua.LState) int {
			c, ok := info(L, "ClassId.race")
			if !ok {
				L.ArgError(1, "ClassId.race: unknown class")
			}
			L.Push(lua.LNumber(c.Race))
			return 1
		},
		"parent": func(L *lua.LState) int {
			c, ok := info(L, "ClassId.parent")
			if !ok || c.Parent < 0 {
				L.Push(lua.LNil)
			} else {
				L.Push(lua.LNumber(c.Parent))
			}
			return 1
		},
		"is_mage": func(L *lua.LState) int {
			c, _ := info(L, "ClassId.is_mage")
			L.Push(lua.LBool(c.Mage))
			return 1
		},
	}
}

// builtinConstants are the constant tables that are the same on every
// server (Mobius enums and constants).
var builtinConstants = map[string]map[string]any{
	"State":     {"CREATED": StateCreated, "STARTED": StateStarted, "COMPLETED": StateCompleted},
	"QuestType": {"REPEATABLE": questTypeRepeatable, "ONE_TIME": questTypeOneTime, "DAILY": questTypeDaily},
	"QuestSound": {
		"ITEMSOUND_QUEST_ACCEPT": "ItemSound.quest_accept", "ITEMSOUND_QUEST_MIDDLE": "ItemSound.quest_middle",
		"ITEMSOUND_QUEST_FINISH": "ItemSound.quest_finish", "ITEMSOUND_QUEST_ITEMGET": "ItemSound.quest_itemget",
		"ITEMSOUND_QUEST_TUTORIAL": "ItemSound.quest_tutorial", "ITEMSOUND_QUEST_GIVEUP": "ItemSound.quest_giveup",
		"ITEMSOUND_QUEST_BEFORE_BATTLE": "ItemSound.quest_before_battle", "ITEMSOUND_QUEST_JACKPOT": "ItemSound.quest_jackpot",
		"ITEMSOUND_QUEST_FANFARE_1": "ItemSound.quest_fanfare_1", "ITEMSOUND_QUEST_FANFARE_2": "ItemSound.quest_fanfare_2",
		"ITEMSOUND_QUEST_FANFARE_MIDDLE": "ItemSound.quest_fanfare_middle", "ITEMSOUND_ARMOR_WOOD": "ItemSound.armor_wood_3",
		"ITEMSOUND_ARMOR_CLOTH": "ItemSound.item_drop_equip_armor_cloth", "AMDSOUND_ED_CHIMES": "AmdSound.ed_chimes_05",
		"HORROR_01": "horror_01", "AMBSOUND_HORROR_01": "AmbSound.dd_horror_01", "AMBSOUND_HORROR_03": "AmbSound.d_horror_03",
		"AMBSOUND_HORROR_15": "AmbSound.d_horror_15", "ITEMSOUND_ARMOR_LEATHER": "ItemSound.itemdrop_armor_leather",
		"ITEMSOUND_WEAPON_SPEAR": "ItemSound.itemdrop_weapon_spear", "AMBSOUND_MT_CREAK": "AmbSound.mt_creak01",
		"AMBSOUND_EG_DRON": "AmbSound.eg_dron_02", "SKILLSOUND_HORROR_02": "SkillSound5.horror_02",
		"CHRSOUND_MHFIGHTER_CRY": "ChrSound.MHFighter_cry", "AMDSOUND_WIND_LOOT": "AmdSound.d_wind_loot_02",
		"INTERFACESOUND_CHARSTAT_OPEN": "InterfaceSound.charstat_open_01", "AMDSOUND_HORROR_02": "AmdSound.dd_horror_02",
		"CHRSOUND_FDELF_CRY": "ChrSound.FDElf_Cry", "AMBSOUND_WINGFLAP": "AmbSound.t_wingflap_04",
		"AMBSOUND_THUNDER": "AmbSound.thunder_02", "AMBSOUND_DRONE": "AmbSound.ed_drone_02",
		"AMBSOUND_CRYSTAL_LOOP": "AmbSound.cd_crystal_loop", "AMBSOUND_PERCUSSION_01": "AmbSound.dt_percussion_01",
		"AMBSOUND_PERCUSSION_02": "AmbSound.ac_percussion_02", "ITEMSOUND_BROKEN_KEY": "ItemSound2.broken_key",
		"ITEMSOUND_SIREN": "ItemSound3.sys_siren", "ITEMSOUND_ENCHANT_SUCCESS": "ItemSound3.sys_enchant_success",
		"ITEMSOUND_ENCHANT_FAILED": "ItemSound3.sys_enchant_failed", "ITEMSOUND_SOW_SUCCESS": "ItemSound3.sys_sow_success",
		"SKILLSOUND_HORROR_1": "SkillSound5.horror_01", "SKILLSOUND_HORROR_2": "SkillSound5.horror_02",
		"SKILLSOUND_ANTARAS_FEAR": "SkillSound3.antaras_fear", "SKILLSOUND_JEWEL_CELEBRATE": "SkillSound2.jewel.celebrate",
		"SKILLSOUND_LIQUID_MIX": "SkillSound5.liquid_mix_01", "SKILLSOUND_LIQUID_SUCCESS": "SkillSound5.liquid_success_01",
		"SKILLSOUND_LIQUID_FAIL": "SkillSound5.liquid_fail_01", "ETCSOUND_ELROKI_SONG_FULL": "EtcSound.elcroki_song_full",
		"ETCSOUND_ELROKI_SONG_1ST": "EtcSound.elcroki_song_1st", "ETCSOUND_ELROKI_SONG_2ND": "EtcSound.elcroki_song_2nd",
		"ETCSOUND_ELROKI_SONG_3RD": "EtcSound.elcroki_song_3rd",
		"BS01_A":                   "BS01_A", "BS02_A": "BS02_A", "BS03_A": "BS03_A", "BS04_A": "BS04_A", "BS06_A": "BS06_A",
		"BS07_A": "BS07_A", "BS08_A": "BS08_A", "BS01_D": "BS01_D", "BS02_D": "BS02_D", "BS05_D": "BS05_D", "BS07_D": "BS07_D",
	},
	"Paperdoll": {
		"UNDER": 0, "LEAR": 1, "REAR": 2, "NECK": 3, "LFINGER": 4, "RFINGER": 5, "HEAD": 6, "RHAND": 7, "LHAND": 8,
		"GLOVES": 9, "CHEST": 10, "LEGS": 11, "FEET": 12, "CLOAK": 13, "FACE": 14, "HAIR": 15, "HAIR2": 16, "TOTALSLOTS": 17,
	},
	"ChatType": {
		"GENERAL": 0, "SHOUT": 1, "WHISPER": 2, "PARTY": 3, "CLAN": 4, "GM": 5, "PETITION_PLAYER": 6, "PETITION_GM": 7,
		"TRADE": 8, "ALLIANCE": 9, "ANNOUNCEMENT": 10, "BOAT": 11, "FRIEND": 12, "MSNCHAT": 13, "PARTYMATCH_ROOM": 14,
		"PARTYROOM_COMMANDER": 15, "PARTYROOM_ALL": 16, "HERO_VOICE": 17, "CRITICAL_ANNOUNCE": 18, "SCREEN_ANNOUNCE": 19,
		"BATTLEFIELD": 20, "MPCC_ROOM": 21, "NPC_GENERAL": 0, "NPC_SHOUT": 1,
	},
	"Race": {
		"HUMAN": 0, "ELF": 1, "DARK_ELF": 2, "ORC": 3, "DWARF": 4, "ANIMAL": 5, "BEAST": 6, "BUG": 7, "CASTLE_GUARD": 8,
		"CONSTRUCT": 9, "DEMONIC": 10, "DIVINE": 11, "DRAGON": 12, "ELEMENTAL": 13, "ETC": 14, "FAIRY": 15, "GIANT": 16,
		"HUMANOID": 17, "MERCENARY": 18, "NONE": 19, "PLANT": 20, "SIEGE_WEAPON": 21, "UNDEAD": 22,
	},
	// Mobius configuration the quests read, with its defaults. The game
	// sets the server's values through Constants.Tables["Config"].
	"Config": {
		"ALT_PARTY_RANGE": defaultPartyRange, "ALT_VILLAGES_REPEATABLE_QUEST_REWARD": false, "DISABLE_TUTORIAL": false,
		"RAID_MIN_RESPAWN_MULTIPLIER": 1.0, "RAID_MAX_RESPAWN_MULTIPLIER": 1.0,
	},
	// Filled by the game from its data (datagen).
	"Category":        {},
	"SystemMessageId": {},
}

// dimensionalDiamondRewards are Mobius Quest.DF_REWARD_35, _37 and _39:
// Dimensional Diamonds for the second class change quests, by class (35,
// 39) or race (37).
var dimensionalDiamondRewards = map[string]map[int]int{
	"DF_REWARD_35": {1: 61, 4: 45, 7: 128, 11: 168, 15: 49, 19: 61, 22: 128, 26: 168, 29: 49, 32: 61, 35: 128, 39: 168, 42: 49, 45: 61, 47: 61, 50: 49, 54: 85, 56: 85},
	"DF_REWARD_37": {0: 96, 1: 102, 2: 98, 3: 109, 4: 50},
	"DF_REWARD_39": {1: 72, 4: 104, 7: 96, 11: 122, 15: 60, 19: 72, 22: 96, 26: 122, 29: 45, 32: 104, 35: 96, 39: 122, 42: 60, 45: 64, 47: 72, 50: 92, 54: 82, 56: 23},
}

// constantTables returns the constant tables of this vm: the built-in ones
// with the game's on top.
func (v *vm) constantTables() map[string]map[string]any {
	tables := make(map[string]map[string]any, len(builtinConstants))
	for name, t := range builtinConstants {
		tables[name] = maps.Clone(t)
	}
	for name, t := range v.quest.consts.Tables {
		if name == "Config" {
			maps.Copy(tables[name], t)
			continue
		}
		tables[name] = maps.Clone(t)
	}
	classes := make(map[string]any, len(v.quest.consts.Classes))
	for _, c := range v.quest.consts.Classes {
		classes[c.Name] = c.ID
	}
	tables["ClassId"] = classes
	return tables
}

// toNumber returns a constant value as a number.
func toNumber(val any) (float64, bool) {
	switch x := val.(type) {
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

// initConstants makes the constant tables globals: read-only, and reading
// a name a table does not have is an error. ClassId keeps its helper
// functions.
func (v *vm) initConstants(L *lua.LState) {
	tables := v.constantTables()
	for _, name := range slices.Sorted(maps.Keys(tables)) {
		t, ok := L.GetGlobal(name).(*lua.LTable)
		if !ok {
			t = L.NewTable()
			L.SetGlobal(name, t)
		}
		for key, val := range tables[name] {
			t.RawSetString(key, v.toLua(L, val))
		}
		L.SetMetatable(t, strictMeta(L, name))
	}
	q, _ := L.GetGlobal("quest").(*lua.LTable)
	if q != nil {
		for name, rewards := range dimensionalDiamondRewards {
			t := L.CreateTable(0, len(rewards))
			for k, val := range rewards {
				t.RawSetInt(k, lua.LNumber(val))
			}
			q.RawSetString(name, t)
		}
	}
}

// strictMeta returns the metatable of a constant table.
func strictMeta(L *lua.LState, name string) *lua.LTable {
	mt := L.NewTable()
	L.SetField(mt, "__index", L.NewFunction(func(L *lua.LState) int {
		L.RaiseError("unknown constant %s.%s", name, L.ToStringMeta(L.Get(2)).String())
		return 0
	}))
	L.SetField(mt, "__newindex", L.NewFunction(func(L *lua.LState) int {
		L.RaiseError("%s is read-only", name)
		return 0
	}))
	L.SetField(mt, "__metatable", lua.LFalse)
	return mt
}

// initRnd makes rnd callable and puts the engine's source behind
// math.random.
func (v *vm) initRnd(L *lua.LState) {
	if t, ok := L.GetGlobal("rnd").(*lua.LTable); ok {
		mt := L.NewTable()
		L.SetField(mt, "__call", L.NewFunction(v.rndCall))
		L.SetField(mt, "__metatable", lua.LFalse)
		L.SetMetatable(t, mt)
	}
	m := L.GetGlobal(lua.MathLibName)
	L.SetField(m, "random", L.NewFunction(v.mathRandom))
	L.SetField(m, "randomseed", lua.LNil)
}
