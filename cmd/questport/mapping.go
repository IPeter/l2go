package main

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// mapEntry maps one Java method, chosen by receiver class, name and
// argument types, to Lua.
//
// Receivers: "Quest" for the inherited Quest/Script methods (unqualified
// calls), "static:Class" for static calls, otherwise a class name matched
// against the receiver's type and its supertypes (see typeChain).
//
// Argument patterns: int, num, dbl (floating only), bool, str, any, obj,
// seq (array or list), pos (Location or world object), a class name, and
// a trailing "T..." for varargs. Unknown argument types match anything.
//
// Templates: $r is the receiver, $b the object a view belongs to (falls
// back to $r), $1..$9 the arguments, $N* the varargs from N on (arrays
// unpacked unless keep is set).
type mapEntry struct {
	recv    string
	name    string
	args    []string
	lua     string
	alt     string // template when the receiver is not a view with a base
	stmt    string // template when the result is unused
	ret     string
	world   bool   // touches the world: constructor code with it moves to q:on_start
	feature string // q:requires feature
	keep    bool   // varargs arrays stay tables (registration functions)
	view    bool   // the result is a view of the receiver: same object, other methods
	fn      func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx
}

func m(recv, name, args, lua, ret string) *mapEntry {
	e := &mapEntry{recv: recv, name: name, lua: lua, ret: ret}
	if args != "" {
		e.args = strings.Split(args, ",")
	}
	return e
}

func (e *mapEntry) w() *mapEntry            { e.world = true; return e }
func (e *mapEntry) feat(f string) *mapEntry { e.feature = f; return e }
func (e *mapEntry) k() *mapEntry            { e.keep = true; return e }
func (e *mapEntry) st(s string) *mapEntry   { e.stmt = s; return e }
func (e *mapEntry) or(alt string) *mapEntry { e.alt = alt; return e }
func (e *mapEntry) asView() *mapEntry       { e.view = true; return e }

func fnm(recv, name, args, ret string, fn func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx) *mapEntry {
	e := m(recv, name, args, "", ret)
	e.fn = fn
	return e
}

// same returns the receiver itself, typed as ret (casts like asPlayer(),
// enum getters like getId()).
func same(ret string) func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	return func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		r := recv
		r.t = t.retType(ret, recv, args)
		return r
	}
}

// argAs returns the first argument, typed as ret.
func argAs(ret string) func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	return func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		r := args[0]
		r.t = t.retType(ret, recv, args)
		return r
	}
}

// specialCall rewrites call chains that only make sense as a whole:
// stream idioms and world scans.
func (t *translator) specialCall(c *callExpr) (lx, bool) {
	// list.stream().mapToInt(i -> i).toArray(): the list itself.
	if c.name == "toArray" && len(c.args) == 0 {
		if m, ok := unparen(c.x).(*callExpr); ok && m.name == "mapToInt" && len(m.args) == 1 && isIdentityLambda(m.args[0]) {
			if s, ok := unparen(m.x).(*callExpr); ok && s.name == "stream" && len(s.args) == 0 && s.x != nil {
				l := t.expr(s.x)
				l.t = arrayOf(tyInt)
				return l, true
			}
		}
	}
	// World.getVisibleObjects().stream().anyMatch(o -> o.getId() == ID):
	// is an NPC with this template ID in the world?
	if c.name == "anyMatch" && len(c.args) == 1 {
		if s, ok := unparen(c.x).(*callExpr); ok && s.name == "stream" && len(s.args) == 0 {
			if v, ok := unparen(s.x).(*callExpr); ok && v.name == "getVisibleObjects" && len(v.args) == 0 && isIdent(v.x, "World") {
				if l, ok := unparen(c.args[0]).(*lambdaExpr); ok && len(l.params) == 1 && l.body != nil {
					if b, ok := unparen(l.body).(*binaryExpr); ok && b.op == "==" {
						if g, ok := unparen(b.x).(*callExpr); ok && g.name == "getId" && len(g.args) == 0 && isIdent(g.x, l.params[0]) {
							id := t.expr(b.y)
							t.useAPI("world.npc_exists")
							return lx{code: "world.npc_exists(" + id.code + ")", prec: precAtom, t: tyBool, call: true}, true
						}
					}
				}
			}
		}
	}
	// World.getVisibleObjects(object, Npc.class): the visible objects of a
	// class around an object.
	if c.name == "getVisibleObjects" && len(c.args) == 2 && isIdent(c.x, "World") {
		if cl, ok := unparen(c.args[1]).(*classLitExpr); ok {
			obj := t.expr(c.args[0])
			t.useAPI("world.visible_objects")
			return lx{code: "world.visible_objects(" + obj.code + ", " + luaQuote(cl.typ.name) + ")", prec: precAtom, t: classType("List", classType(cl.typ.name)), call: true, safe: true}, true
		}
	}
	return lx{}, false
}

func isIdent(e expr, name string) bool {
	id, ok := unparen(e).(*identExpr)
	return ok && id.name == name
}

func isIdentityLambda(e expr) bool {
	l, ok := unparen(e).(*lambdaExpr)
	return ok && len(l.params) == 1 && l.body != nil && isIdent(l.body, l.params[0])
}

// tableOf builds a Lua array of the arguments.
func tableOf(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.code
	}
	return lx{code: "{" + strings.Join(parts, ", ") + "}", prec: precAtom, t: classType("List", elemOf(args)), safe: true}
}

// asListFn is Arrays.asList. A single array of objects (Integer[],
// String[]) is passed as the varargs array itself: the list is a view of
// it, so shuffling the list shuffles the array. An array of primitives
// is one element, as in Java.
func asListFn(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	if len(args) == 1 && isArray(args[0].t) && args[0].t.elem != nil && args[0].t.elem.kind != tkPrim && !isUnknown(args[0].t.elem) {
		r := args[0]
		r.t = classType("List", args[0].t.elem)
		return r
	}
	return tableOf(t, c, recv, args, asStmt)
}

func elemOf(args []lx) *jtype {
	if len(args) > 0 {
		return args[0].t
	}
	return tyUnknown
}

var mappings = []*mapEntry{
	// --- Quest: registration (only while the scripts load) ---
	m("Quest", "registerQuestItems", "int...", "q:register_quest_items($1*)", "").k(),
	m("Quest", "addStartNpc", "int...", "q:add_start_npc($1*)", "").k(),
	m("Quest", "addTalkId", "int...", "q:add_talk_id($1*)", "").k(),
	m("Quest", "addFirstTalkId", "int...", "q:add_first_talk_id($1*)", "").k(),
	m("Quest", "addKillId", "int...", "q:add_kill_id($1*)", "").k(),
	m("Quest", "addAttackId", "int...", "q:add_attack_id($1*)", "").k(),
	m("Quest", "addSpawnId", "int...", "q:add_spawn_id($1*)", "").k(),
	m("Quest", "addSkillSeeId", "int...", "q:add_skill_see_id($1*)", "").k(),
	m("Quest", "addSpellFinishedId", "int...", "q:add_spell_finished_id($1*)", "").k(),
	m("Quest", "addAggroRangeEnterId", "int...", "q:add_aggro_range_enter_id($1*)", "").k(),
	m("Quest", "addEnterZoneId", "int...", "q:add_enter_zone_id($1*)", "").k(),
	m("Quest", "addExitZoneId", "int...", "q:add_exit_zone_id($1*)", "").k(),
	m("Quest", "addItemTalkId", "int...", "q:add_item_talk_id($1*)", "").k(),
	m("Quest", "addFactionCallId", "int...", "q:add_faction_call_id($1*)", "").k(),
	m("Quest", "addCreatureSeeId", "int...", "q:add_creature_see_id($1*)", "").k(),
	m("Quest", "addMoveFinishedId", "int...", "q:add_move_finished_id($1*)", "").k(),
	m("Quest", "addNpcHateId", "int...", "q:add_npc_hate_id($1*)", "").k(),
	m("Quest", "addTeleportId", "int...", "q:add_teleport_id($1*)", "").k(),
	m("Quest", "setOnEnterWorld", "bool", "q:set_on_enter_world($1)", ""),
	m("Quest", "setCustom", "bool", "q:set_custom($1)", ""),

	// --- Quest: states, pages ---
	m("Quest", "getQuestState", "Player,bool", "q:get_quest_state($1, $2)", "QuestState"),
	m("Quest", "newQuestState", "Player", "q:new_quest_state($1)", "QuestState"),
	m("Quest", "getNoQuestMsg", "Player", "q:get_no_quest_msg($1)", "String"),
	m("Quest", "getAlreadyCompletedMsg", "Player", "q:get_already_completed_msg($1)", "String"),
	m("Quest", "getHtm", "Player,str", "q:get_htm($1, $2)", "String"),
	m("Quest", "showHtmlFile", "Player,str", "q:show_html_file($1, $2)", "String"),
	m("Quest", "showHtmlFile", "Player,str,Npc", "q:show_html_file($1, $2, $3)", "String"),
	m("Quest", "getName", "", "q:name()", "String"),
	m("Quest", "getId", "", "q:id()", "int"),
	m("Quest", "getDescription", "", "q:description()", "String"),
	m("Quest", "getRegisteredItemIds", "", "q:get_registered_item_ids()", "int[]"),
	m("Quest", "notifyEvent", "str,Npc,Player", "q:notify_event($1, $2, $3)", ""),
	m("Quest", "isCustomQuest", "", "q:is_custom()", "boolean"),

	// --- Quest: items and rewards ---
	m("Quest", "giveItems", "Player,int,int", "q:give_items($1, $2, $3)", ""),
	m("Quest", "giveItems", "Player,int,int,int", "q:give_items($1, $2, $3, $4)", ""),
	m("Quest", "giveItems", "Player,ItemHolder", "q:give_items($1, $2)", ""),
	m("Quest", "rewardItems", "Player,int,int", "q:reward_items($1, $2, $3)", ""),
	m("Quest", "rewardItems", "Player,ItemHolder", "q:reward_items($1, $2)", ""),
	m("Quest", "giveAdena", "Player,int,bool", "q:give_adena($1, $2, $3)", ""),
	m("Quest", "takeItems", "Player,int,int", "q:take_items($1, $2, $3)", "boolean"),
	m("Quest", "takeItems", "Player,int,int...", "q:take_items_ids($1, $2, $3*)", "boolean"),
	m("Quest", "takeItems", "Player,ItemHolder", "q:take_items($1, $2)", "boolean"),
	m("Quest", "takeItem", "Player,ItemHolder", "q:take_items($1, $2)", "boolean"),
	m("Quest", "hasItem", "Player,ItemHolder", "q:get_quest_items_count($1, $2.id) >= $2.count", "boolean"),
	m("Quest", "getQuestItemsCount", "Player,int...", "q:get_quest_items_count($1, $2*)", "long"),
	m("Quest", "hasQuestItems", "Player,int...", "q:has_quest_items($1, $2*)", "boolean"),
	m("Quest", "hasAtLeastOneQuestItem", "Player,int...", "q:has_at_least_one_quest_item($1, $2*)", "boolean"),
	m("Quest", "ownsAtLeastOneItem", "Player,int...", "q:owns_at_least_one_item($1, $2*)", "boolean"),
	m("Quest", "getEnchantLevel", "Player,int", "q:get_enchant_level($1, $2)", "int"),
	m("Quest", "getItemEquipped", "Player,int", "q:get_item_equipped($1, $2)", "int"),
	m("Quest", "giveItemRandomly", "Player,int,int,int,num,bool", "q:give_item_randomly($1, nil, $2, $3, $3, $4, $5, $6)", "boolean"),
	m("Quest", "giveItemRandomly", "Player,Npc,int,int,int,num,bool", "q:give_item_randomly($1, $2, $3, $4, $4, $5, $6, $7)", "boolean"),
	m("Quest", "giveItemRandomly", "Player,Npc,int,int,int,int,num,bool", "q:give_item_randomly($1, $2, $3, $4, $5, $6, $7, $8)", "boolean"),
	m("Quest", "removeRegisteredQuestItems", "Player", "q:remove_registered_quest_items($1)", ""),

	// --- Quest: sounds, experience ---
	m("Quest", "playSound", "Player,any", "q:play_sound($1, $2)", ""),
	m("Quest", "addExpAndSp", "Player,int,int", "q:add_exp_and_sp($1, $2, $3)", ""),

	// --- Random ---
	m("Quest", "getRandom", "dbl", "rnd.double($1)", "double"),
	m("Quest", "getRandom", "int", "rnd($1)", "int"),
	m("Quest", "getRandom", "int,int", "rnd($1, $2)", "int"),
	m("Quest", "getRandomBoolean", "", "rnd.bool()", "boolean"),
	fnm("Quest", "getRandomEntry", "seq", "", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("rnd.entry")
		return lx{code: "rnd.entry(" + args[0].code + ")", prec: precAtom, t: elemType(args[0].t), call: true}
	}),
	fnm("Quest", "getRandomEntry", "any...", "", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("rnd.entry")
		tbl := tableOf(t, c, recv, args, false)
		return lx{code: "rnd.entry(" + tbl.code + ")", prec: precAtom, t: elemOf(args), call: true}
	}),
	m("static:Rnd", "get", "dbl", "rnd.double($1)", "double"),
	m("static:Rnd", "get", "int", "rnd($1)", "int"),
	m("static:Rnd", "get", "int,int", "rnd($1, $2)", "int"),
	m("static:Rnd", "nextInt", "int", "rnd($1)", "int"),
	m("static:Rnd", "nextDouble", "", "rnd.double()", "double"),
	m("static:Rnd", "nextBoolean", "", "rnd.bool()", "boolean"),
	m("static:Rnd", "nextGaussian", "", "rnd.gaussian()", "double"),

	// --- Quest timers ---
	m("Quest", "startQuestTimer", "str,int,Npc,Player", "q:start_quest_timer($1, $2, $3, $4)", "").w(),
	m("Quest", "startQuestTimer", "str,int,Npc,Player,bool", "q:start_quest_timer($1, $2, $3, $4, $5)", "").w(),
	m("Quest", "getQuestTimer", "str,Npc,Player", "q:get_quest_timer($1, $2, $3)", "QuestTimer"),
	m("Quest", "cancelQuestTimer", "str,Npc,Player", "q:cancel_quest_timer($1, $2, $3)", ""),
	m("Quest", "cancelQuestTimers", "str", "q:cancel_quest_timers($1)", ""),
	m("QuestTimer", "cancel", "", "$r:cancel()", ""),
	m("QuestTimer", "cancelAndRemove", "", "$r:cancel()", ""),

	// --- Party helpers ---
	m("Quest", "getRandomPartyMember", "Player", "q:get_random_party_member($1)", "Player"),
	m("Quest", "getRandomPartyMember", "Player,int", "q:get_random_party_member($1, $2)", "Player"),
	m("Quest", "getRandomPartyMember", "Player,Npc", "q:get_random_party_member($1, $2)", "Player"),
	m("Quest", "getRandomPartyMember", "Player,str,str", "q:get_random_party_member($1, $2, $3)", "Player"),
	m("Quest", "getRandomPartyMemberState", "Player,int", "q:get_random_party_member_state($1, $2)", "Player"),
	m("Quest", "getRandomPartyMemberState", "Player,int,int,Npc", "q:get_random_party_member_state($1, $2, $3, $4)", "QuestState"),
	m("Quest", "executeForEachPlayer", "Player,Npc,bool,bool,bool", "q:execute_for_each_player($1, $2, $3, $4, $5)", ""),
	m("Quest", "getClanLeaderQuestState", "Player,Npc", "q:get_clan_leader_quest_state($1, $2)", "QuestState").feat("clans"),
	m("Quest", "setQuestToClanMembers", "Player", "q:set_quest_to_clan_members($1)", "").feat("clans"),
	m("Quest", "finishQuestToClan", "Player", "q:finish_quest_to_clan($1)", "").feat("clans"),

	// --- Spawns ---
	m("Quest", "addSpawn", "int,int,int,int,int", "q:add_spawn($1, {x = $2, y = $3, z = $4, heading = $5})", "Npc").w(),
	m("Quest", "addSpawn", "int,int,int,int,int,bool,int", "q:add_spawn($1, {x = $2, y = $3, z = $4, heading = $5}, $6, $7)", "Npc").w(),
	m("Quest", "addSpawn", "int,int,int,int,int,bool,int,bool", "q:add_spawn($1, {x = $2, y = $3, z = $4, heading = $5}, $6, $7, $8)", "Npc").w(),
	m("Quest", "addSpawn", "int,pos", "q:add_spawn($1, $2)", "Npc").w(),
	m("Quest", "addSpawn", "int,pos,bool", "q:add_spawn($1, $2, false, 0, $3)", "Npc").w(),
	m("Quest", "addSpawn", "int,pos,bool,int", "q:add_spawn($1, $2, $3, $4)", "Npc").w(),
	m("Quest", "addSpawn", "int,pos,bool,int,bool", "q:add_spawn($1, $2, $3, $4, $5)", "Npc").w(),
	m("Quest", "addSpawn", "Npc,int,pos,bool,int", "q:add_spawn($2, $3, $4, $5, false, $1)", "Npc").w(),
	m("Quest", "addMinion", "Monster,int", "$1:add_minion($2)", "Npc").w(),

	// --- Radar, screen, camera, teleports, doors ---
	m("Quest", "addRadar", "Player,int,int,int", "q:add_radar($1, $2, $3, $4)", ""),
	m("Quest", "removeRadar", "Player,int,int,int", "q:remove_radar($1, $2, $3, $4)", ""),
	m("Quest", "clearRadar", "Player", "q:clear_radar($1)", ""),
	m("Quest", "showOnScreenMsg", "Player,str,int", "q:show_on_screen_msg($1, $2, $3)", ""),
	m("Quest", "teleportPlayer", "Player,Location,int", "$1:teleport($2)", ""),
	m("Quest", "teleportPlayer", "Player,Location,int,bool", "$1:teleport($2)", ""),
	m("Quest", "getGameTicks", "", "time.game_ticks()", "int"),
	m("Quest", "openDoor", "int,int", "q:open_door($1)", "").w().feat("doors"),
	m("Quest", "closeDoor", "int,int", "q:close_door($1)", "").w().feat("doors"),

	// --- QuestState ---
	m("QuestState", "getState", "", "$r:get_state()", "int"),
	m("QuestState", "isCreated", "", "$r:is_created()", "boolean"),
	m("QuestState", "isStarted", "", "$r:is_started()", "boolean"),
	m("QuestState", "isCompleted", "", "$r:is_completed()", "boolean"),
	m("QuestState", "setState", "int", "$r:set_state($1)", ""),
	m("QuestState", "setState", "int,bool", "$r:set_state($1, $2)", ""),
	m("QuestState", "startQuest", "", "$r:start_quest()", ""),
	m("QuestState", "getCond", "", "$r:get_cond()", "int"),
	m("QuestState", "isCond", "int", "$r:is_cond($1)", "boolean"),
	m("QuestState", "setCond", "int", "$r:set_cond($1)", ""),
	m("QuestState", "setCond", "int,bool", "$r:set_cond($1, $2)", ""),
	m("QuestState", "set", "str,any", "$r:set($1, $2)", ""),
	m("QuestState", "get", "str", "$r:get($1)", "String"),
	m("QuestState", "getInt", "str", "$r:get_int($1)", "int"),
	m("QuestState", "unset", "str", "$r:unset($1)", ""),
	m("QuestState", "isSet", "str", "$r:is_set($1)", "boolean"),
	m("QuestState", "getMemoState", "", "$r:get_memo_state()", "int"),
	m("QuestState", "setMemoState", "int", "$r:set_memo_state($1)", ""),
	m("QuestState", "isMemoState", "int", "$r:is_memo_state($1)", "boolean"),
	m("QuestState", "getMemoStateEx", "int", "$r:get_memo_state_ex($1)", "int"),
	m("QuestState", "setMemoStateEx", "int,int", "$r:set_memo_state_ex($1, $2)", ""),
	m("QuestState", "isMemoStateEx", "int,int", "$r:is_memo_state_ex($1, $2)", "boolean"),
	m("QuestState", "exitQuest", "bool", "$r:exit_quest($1)", ""),
	m("QuestState", "exitQuest", "bool,bool", "$r:exit_quest($1, $2)", ""),
	m("QuestState", "exitQuest", "QuestType", "$r:exit_quest($1)", ""),
	m("QuestState", "exitQuest", "QuestType,bool", "$r:exit_quest($1, $2)", ""),
	m("QuestState", "isNowAvailable", "", "$r:is_now_available()", "boolean"),
	m("QuestState", "setRestartTime", "", "$r:set_restart_time()", ""),
	m("QuestState", "getPlayer", "", "$r:get_player()", "Player"),
	m("QuestState", "getQuestName", "", "$r:get_quest_name()", "String"),
	m("QuestState", "showQuestionMark", "int", "$r:show_question_mark($1)", ""),
	m("QuestState", "playTutorialVoice", "str", "$r:play_tutorial_voice($1)", ""),
	m("QuestState", "showTutorialHTML", "str", "$r:show_tutorial_html($1)", ""),
	m("QuestState", "closeTutorialHtml", "", "$r:close_tutorial_html()", ""),
	m("QuestState", "onTutorialClientEvent", "int", "$r:on_tutorial_client_event($1)", ""),
	m("QuestState", "addNotifyOfDeath", "Creature", "$r:add_notify_of_death($1)", ""),
	m("QuestState", "setIsExitQuestOnCleanUp", "bool", "$r:set_exit_quest_on_clean_up($1)", ""),

	// --- Player ---
	m("Player", "getLevel", "", "$r:level()", "int"),
	m("Player", "getRace", "", "$r:race()", "Race"),
	m("Player", "getPlayerClass", "", "$r:class_id()", "PlayerClass"),
	m("Player", "getClassId", "", "$r:class_id()", "PlayerClass"),
	m("Player", "getBaseClass", "", "$r:base_class()", "int"),
	m("Player", "isMageClass", "", "$r:is_mage_class()", "boolean"),
	m("Player", "isInCategory", "any", "$r:is_in_category($1)", "boolean"),
	m("Player", "setPlayerClass", "int", "$r:set_class_id($1)", ""),
	m("Player", "setClassId", "int", "$r:set_class_id($1)", ""),
	m("Player", "setBaseClass", "int", "$r:set_base_class($1)", ""),
	m("Player", "broadcastUserInfo", "", "$r:broadcast_user_info()", ""),
	m("Player", "isNewbie", "", "$r:is_newbie()", "boolean"),
	m("Player", "getQuestState", "str", "$r:get_quest_state($1)", "QuestState"),
	m("Player", "isInParty", "", "$r:is_in_party()", "boolean"),
	m("Player", "getParty", "", "$r:party()", "Party").asView(),
	m("Player", "getClan", "", "$r:clan()", "Clan").feat("clans"),
	m("Player", "getClanId", "", "$r:clan_id()", "int").feat("clans"),
	m("Player", "isClanLeader", "", "$r:is_clan_leader()", "boolean").feat("clans"),
	m("Player", "isSubClassActive", "", "$r:is_sub_class_active()", "boolean").feat("subclass"),
	m("Player", "isNoble", "", "$r:is_noble()", "boolean").feat("noblesse"),
	m("Player", "setNoble", "bool", "$r:set_noble($1)", "").feat("noblesse"),
	m("Player", "sendMessage", "str", "$r:send_message($1)", ""),
	m("Player", "getAdena", "", "$r:adena()", "long"),
	m("Player", "getSkillLevel", "int", "$r:skill_level($1)", "int"),
	m("Player", "getKarma", "", "$r:karma()", "int"),
	m("Player", "getPkKills", "", "$r:pk_kills()", "int"),
	m("Player", "isGM", "", "$r:is_gm()", "boolean"),
	m("Player", "getSex", "", "$r:sex()", "int"),
	m("Player", "getVariables", "", "$r", "PlayerVariables").asView(),
	m("Player", "getInventory", "", "$r", "Inventory").asView(),
	m("Player", "getAppearance", "", "$r", "PlayerAppearance").asView(),
	m("Player", "getWeightPenalty", "", "$r:weight_penalty()", "int"),
	m("Player", "isOnline", "", "$r:is_online()", "boolean"),
	m("Player", "getSummon", "", "$r:summon()", "Summon").feat("pets"),
	m("Player", "hasSummon", "", "$r:summon() ~= nil", "boolean").feat("pets"),
	m("Player", "getPet", "", "$r:summon()", "Summon").feat("pets"),
	m("Player", "isSitting", "", "$r:is_sitting()", "boolean"),
	m("Player", "getRecomHave", "", "$r:recom_have()", "int"),
	m("Player", "isHero", "", "$r:is_hero()", "boolean"),
	m("Player", "getActiveWeaponItem", "", "$r:active_weapon()", "Weapon"),
	m("Player", "setPkKills", "int", "$r:set_pk_kills($1)", ""),
	m("Player", "updateUserInfo", "", "$r:update_user_info()", ""),
	m("Player", "addNotifyQuestOfDeath", "QuestState", "$1:add_notify_of_death($r)", ""),
	m("Player", "destroyItemByItemId", "any,int,int,any,bool", "q:take_items($r, $2, $3)", "boolean"),
	m("Player", "getAllActiveQuests", "", "$r:active_quests()", "List"),
	m("Player", "getSponsor", "", "$r:sponsor()", "int").feat("clans"),
	m("Player", "getApprentice", "", "$r:apprentice()", "int").feat("clans"),
	m("Player", "getPledgeType", "", "$r:pledge_type()", "int").feat("clans"),
	m("Player", "getClassIndex", "", "$r:class_index()", "int").feat("subclass"),
	m("Player", "getTotalSubClasses", "", "$r:total_sub_classes()", "int").feat("subclass"),
	m("PlayerAppearance", "isFemale", "", "$b:sex() == 1", "boolean"),
	m("PlayerAppearance", "getSex", "", "$b:sex() == 1", "boolean"),
	m("PlayerVariables", "getInt", "str,int", "$b:var_int($1, $2)", "int"),
	m("PlayerVariables", "getInt", "str", "$b:var_int($1)", "int"),
	m("PlayerVariables", "getLong", "str,int", "$b:var_int($1, $2)", "long"),
	m("PlayerVariables", "getBoolean", "str,bool", "$b:var_bool($1, $2)", "boolean"),
	m("PlayerVariables", "getBoolean", "str", "$b:var_bool($1)", "boolean"),
	m("PlayerVariables", "getString", "str,str", "$b:var($1, $2)", "String"),
	m("PlayerVariables", "getString", "str", "$b:var($1)", "String"),
	m("PlayerVariables", "set", "str,any", "$b:set_var($1, $2)", ""),
	m("PlayerVariables", "remove", "str", "$b:unset_var($1)", ""),
	m("PlayerVariables", "hasVariable", "str", "$b:has_var($1)", "boolean"),
	m("PlayerVariables", "contains", "str", "$b:has_var($1)", "boolean"),
	m("Inventory", "getPaperdollItemId", "int", "q:get_item_equipped($b, $1)", "int"),
	m("Inventory", "getItemByItemId", "int", "$b:get_item($1)", "Item"),
	m("Inventory", "getInventoryItemCount", "int,int", "q:get_quest_items_count($b, $1)", "long"),
	m("Party", "getMembers", "", "$b:party_members()", "List<Player>").or("$r:members()"),
	m("Party", "getPartyMembers", "", "$b:party_members()", "List<Player>").or("$r:members()"),
	m("Party", "getLeader", "", "$b:party_leader()", "Player").or("$r:leader()"),
	m("Party", "getMemberCount", "", "#$b:party_members()", "int").or("#$r:members()"),
	m("Party", "isLeader", "Player", "$b:party_leader() == $1", "boolean").or("$r:leader() == $1"),
	m("Party", "getLeaderObjectId", "", "$b:party_leader():object_id()", "int").or("$r:leader():object_id()"),
	m("Clan", "getLevel", "", "$r:level()", "int").feat("clans"),
	m("Clan", "getLeaderId", "", "$r:leader_id()", "int").feat("clans"),
	m("Clan", "getId", "", "$r:id()", "int").feat("clans"),
	m("Clan", "getName", "", "$r:name()", "String").feat("clans"),
	m("Clan", "getReputationScore", "", "$r:reputation()", "int").feat("clans"),
	m("Clan", "addReputationScore", "int", "$r:add_reputation($1)", "").feat("clans"),
	m("Clan", "getHideoutId", "", "$r:hideout_id()", "int").feat("clan_halls"),
	m("Clan", "getLeader", "", "$r:leader()", "ClanMember").feat("clans"),
	m("ClanMember", "getPlayer", "", "$r:player()", "Player").feat("clans"),
	fnm("Clan", "broadcastToOnlineMembers", "any", "", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		if args[0].packet != "PledgeShowInfoUpdate" {
			return t.todo(c.line, "call Clan.broadcastToOnlineMembers", "Clan.broadcastToOnlineMembers of another packet")
		}
		t.consumePacket(args[0])
		t.requires("clans")
		t.useAPI("Clan:broadcast_pledge_info_update")
		return lx{code: asAtom(recv) + ":broadcast_pledge_info_update()", prec: precAtom, t: tyVoid, call: true}
	}),
	m("static:CHSiegeManager", "getInstance", "", "clan_halls", "CHSiegeManager").asView().feat("clan_halls"),
	m("CHSiegeManager", "getSiegableHall", "int", "clan_halls.get_siegable_hall($1)", "SiegableHall").feat("clan_halls"),
	m("SiegableHall", "getSiege", "", "$r", "ClanHallSiege").asView(),
	m("ClanHallSiege", "getAttackers", "", "$b:siege_attackers()", "List").feat("clan_halls"),
	m("ClanMember", "isOnline", "", "$r:is_online()", "boolean").feat("clans"),

	// --- Creatures and world objects ---
	m("WorldObject", "getObjectId", "", "$r:object_id()", "int"),
	m("WorldObject", "getX", "", "$r:x()", "int"),
	m("WorldObject", "getY", "", "$r:y()", "int"),
	m("WorldObject", "getZ", "", "$r:z()", "int"),
	m("WorldObject", "getHeading", "", "$r:heading()", "int"),
	m("WorldObject", "getLocation", "", "$r:location()", "Location"),
	m("WorldObject", "getName", "", "$r:name()", "String"),
	m("WorldObject", "isPlayer", "", "$r:is_player()", "boolean"),
	m("WorldObject", "isNpc", "", "$r:is_npc()", "boolean"),
	m("WorldObject", "isMonster", "", "$r:is_monster()", "boolean"),
	m("WorldObject", "isAttackable", "", "$r:is_attackable()", "boolean"),
	m("WorldObject", "isSummon", "", "$r:is_summon()", "boolean"),
	m("WorldObject", "isPlayable", "", "$r:is_playable()", "boolean"),
	fnm("WorldObject", "asPlayer", "", "Player", same("Player")),
	fnm("WorldObject", "asNpc", "", "Npc", same("Npc")),
	fnm("WorldObject", "asMonster", "", "Monster", same("Monster")),
	fnm("WorldObject", "asAttackable", "", "Attackable", same("Attackable")),
	fnm("WorldObject", "asCreature", "", "Creature", same("Creature")),
	fnm("WorldObject", "asSummon", "", "Summon", same("Summon")),
	m("WorldObject", "getId", "", "$r:id()", "int"),
	m("WorldObject", "decayMe", "", "$r:delete()", "").w(),
	m("WorldObject", "getActingPlayer", "", "$r:acting_player()", "Player"),
	m("WorldObject", "calculateDistance3D", "pos", "$r:distance_3d($1)", "double"),
	m("WorldObject", "calculateDistance2D", "pos", "$r:distance_2d($1)", "double"),
	m("WorldObject", "isInsideRadius2D", "pos,int", "$r:is_inside_radius_2d($1, $2)", "boolean"),
	m("WorldObject", "isInsideRadius3D", "pos,int", "$r:is_inside_radius_3d($1, $2)", "boolean"),
	m("WorldObject", "getInstanceId", "", "0", "int"),
	m("Creature", "getTarget", "", "$r:target()", "WorldObject"),
	m("Creature", "setTarget", "WorldObject", "$r:set_target($1)", ""),
	m("Creature", "isDead", "", "$r:is_dead()", "boolean"),
	m("Creature", "isAlikeDead", "", "$r:is_dead()", "boolean"),
	m("Creature", "getCurrentHp", "", "$r:hp()", "double"),
	m("Creature", "getMaxHp", "", "$r:max_hp()", "int"),
	m("Creature", "getCurrentMp", "", "$r:mp()", "double"),
	m("Creature", "getMaxMp", "", "$r:max_mp()", "int"),
	m("Creature", "teleToLocation", "int,int,int", "$r:teleport($1, $2, $3)", "").w(),
	m("Creature", "teleToLocation", "Location", "$r:teleport($1)", "").w(),
	m("Creature", "teleToLocation", "Location,bool", "$r:teleport($1)", "").w(),
	m("Creature", "teleToLocation", "int,int,int,int", "$r:teleport($1, $2, $3)", "").w(),
	m("Creature", "doCast", "Skill", "$r:cast($1)", "").w(),
	m("Creature", "doDie", "any", "$r:die($1)", "").w(),
	m("Creature", "isInCombat", "", "$r:is_in_combat()", "boolean"),
	m("Creature", "setRunning", "", "$r:set_running()", ""),
	m("Creature", "setWalking", "", "$r:set_walking()", ""),
	m("Creature", "getLevel", "", "$r:level()", "int"),
	m("Creature", "getAI", "", "$r", "AI").asView(),
	m("Creature", "stopSkillEffects", "any", "$r:stop_skill_effects($1)", ""),
	m("Creature", "isAffectedBySkill", "int", "$r:is_affected_by_skill($1)", "boolean"),
	m("Creature", "isRunning", "", "$r:is_running()", "boolean"),
	m("Creature", "getSummoner", "", "$r:summoner()", "Creature"),
	m("Creature", "isInsideZone", "any", "$r:is_inside_zone($1)", "boolean"),
	m("Creature", "isInvul", "", "$r:is_invul()", "boolean"),
	m("Creature", "setInvul", "bool", "$r:set_invul($1)", ""),
	m("Creature", "setImmobilized", "bool", "$r:set_immobilized($1)", ""),
	m("Creature", "isMoving", "", "$r:is_moving()", "boolean"),
	m("Creature", "reduceCurrentHp", "num,Creature,Skill", "$r:reduce_hp($1, $2, $3)", ""),

	// --- Npc ---
	m("Npc", "getId", "", "$r:id()", "int"),
	m("Npc", "getNpcId", "", "$r:id()", "int"),
	m("Npc", "deleteMe", "", "$r:delete()", "").w(),
	m("Npc", "decayMe", "", "$r:delete()", "").w(),
	m("Npc", "getScriptValue", "", "$r:get_script_value()", "int"),
	m("Npc", "setScriptValue", "int", "$r:set_script_value($1)", ""),
	m("Npc", "isScriptValue", "int", "$r:is_script_value($1)", "boolean"),
	m("Npc", "broadcastSay", "any,str", "$r:broadcast_say($1, $2)", ""),
	m("Npc", "showChatWindow", "Player", "$r:show_chat_window($1)", ""),
	m("Npc", "showChatWindow", "Player,int", "$r:show_chat_window($1, $2)", ""),
	m("Npc", "getVariables", "", "$r", "NpcVariables").asView(),
	m("Npc", "getSpawn", "", "$r", "Spawn").asView(),
	m("Npc", "isInsideRadius", "pos,int,bool,bool", "$r:is_inside_radius_3d($1, $2)", "boolean"),
	m("Npc", "getTemplate", "", "$r", "NpcTemplate").asView(),
	m("Npc", "isVillageMaster", "", "$r:is_type(\"VillageMaster\")", "boolean"),
	m("NpcTemplate", "getId", "", "$b:id()", "int"),
	m("NpcTemplate", "getLevel", "", "$b:level()", "int"),
	m("NpcTemplate", "getName", "", "$b:name()", "String"),
	m("NpcTemplate", "getDisplayId", "", "$b:display_id()", "int"),
	m("NpcVariables", "getInt", "str,int", "$b:var_int($1, $2)", "int"),
	m("NpcVariables", "getInt", "str", "$b:var_int($1)", "int"),
	m("NpcVariables", "getLong", "str,int", "$b:var_int($1, $2)", "long"),
	m("NpcVariables", "getBoolean", "str,bool", "$b:var_bool($1, $2)", "boolean"),
	m("NpcVariables", "getString", "str,str", "$b:var($1, $2)", "String"),
	m("NpcVariables", "getObject", "str,any", "$b:var($1)", "Object"),
	m("NpcVariables", "set", "str,any", "$b:set_var($1, $2)", ""),
	m("NpcVariables", "remove", "str", "$b:unset_var($1)", ""),
	m("Spawn", "stopRespawn", "", "$b:stop_respawn()", ""),
	m("Spawn", "startRespawn", "", "$b:start_respawn()", ""),
	m("Spawn", "getLocation", "", "$b:spawn_location()", "Location"),
	m("Attackable", "addDamageHate", "Creature,int,int", "$r:add_damage_hate($1, $2, $3)", ""),
	m("Attackable", "clearAggroList", "", "$r:clear_aggro_list()", ""),
	m("Attackable", "getMostHated", "", "$r:most_hated()", "Creature"),
	m("Monster", "dropItem", "Player,int,int", "$r:drop_item($1, $2, $3)", "").w(),
	m("Npc", "dropItem", "Player,int,int", "$r:drop_item($1, $2, $3)", "").w(),
	m("Monster", "getMinionList", "", "$r", "MinionList").asView(),
	m("Monster", "hasMinions", "", "$r:has_minions()", "boolean"),
	m("MinionList", "getSpawnedMinions", "", "$b:spawned_minions()", "List<Monster>"),
	m("Npc", "abortAttack", "", "$r:abort_attack()", ""),
	m("Npc", "isSpawned", "", "$r:is_spawned()", "boolean"),
	m("Npc", "isSweepActive", "", "$r:is_sweep_active()", "boolean"),
	m("Attackable", "getSpoilerObjectId", "", "$r:spoiler_object_id()", "int"),
	m("Npc", "getAttackByList", "", "$r", "AttackByList").asView(),
	m("AttackByList", "remove", "any", "$b:remove_attacker($1)", "boolean"),
	m("Summon", "getControlObjectId", "", "$r:control_object_id()", "int").feat("pets"),
	m("AI", "setIntentionAttack", "Creature", "$b:attack($1)", ""),
	m("AI", "setIntentionCast", "Skill,WorldObject", "$b:cast($1, $2)", "").w(),
	m("Attackable", "setCanReturnToSpawnPoint", "bool", "$r:set_can_return_to_spawn_point($1)", ""),
	m("Creature", "isCastingNow", "", "$r:is_casting_now()", "boolean"),
	m("Creature", "isSkillDisabled", "Skill", "$r:is_skill_disabled($1)", "boolean"),
	m("Creature", "setXYZ", "int,int,int", "$r:set_xyz($1, $2, $3)", "").w(),
	m("Creature", "revalidateZone", "bool", "$r:revalidate_zone()", ""),
	m("Creature", "getEffectList", "", "$r", "EffectList").asView(),
	m("EffectList", "getBuffInfoByAbnormalType", "any", "$b:buff_info_by_abnormal_type($1)", "BuffInfo"),
	fnm("Creature", "addStatFunc", "any", "", addStatFuncFn),
	m("AI", "setIntention", "any", "$b:set_intention($1)", ""),
	m("AI", "setIntention", "any,any", "$b:set_intention($1, $2)", ""),
	m("AI", "setIntention", "any,any,any", "$b:set_intention($1, $2, $3)", ""),
	m("AI", "setIntentionMoveTo", "Location", "$b:move_to($1)", ""),
	m("AI", "getIntention", "", "$b:intention()", "Intention"),
	m("AI", "stopFollow", "", "$b:stop_follow()", ""),
	m("AI", "startFollow", "Creature", "$b:follow($1)", ""),
	m("AI", "clientStopMoving", "any", "$b:stop_moving()", ""),

	// --- Items, skills and holders ---
	m("Item", "getId", "", "$r:id()", "int"),
	m("Item", "getObjectId", "", "$r:object_id()", "int"),
	m("Item", "getCount", "", "$r:count()", "long"),
	m("Item", "getEnchantLevel", "", "$r:enchant_level()", "int"),
	m("Item", "isEquipped", "", "$r:is_equipped()", "boolean"),
	m("ItemHolder", "getId", "", "$r.id", "int"),
	m("ItemHolder", "getCount", "", "$r.count", "long"),
	m("ItemChanceHolder", "getId", "", "$r.id", "int"),
	m("ItemChanceHolder", "getCount", "", "$r.count", "long"),
	m("ItemChanceHolder", "getChance", "", "$r.chance", "double"),
	m("SkillHolder", "getSkillId", "", "$r.id", "int"),
	m("SkillHolder", "getSkillLevel", "", "$r.level", "int"),
	m("SkillHolder", "getSkill", "", "$r", "Skill"),
	m("Skill", "getId", "", "$r.id", "int"),
	m("Skill", "getLevel", "", "$r.level", "int"),
	m("Skill", "getMpConsume", "", "$r.mp_consume", "int"),
	m("Skill", "getHpConsume", "", "$r.hp_consume", "int"),
	m("Location", "getX", "", "$r.x", "int"),
	m("Location", "getY", "", "$r.y", "int"),
	m("Location", "getZ", "", "$r.z", "int"),
	m("Location", "getHeading", "", "$r.heading", "int"),
	m("ZoneType", "getId", "", "$r:id()", "int"),
	m("Weapon", "getItemType", "", "$r:item_type()", "WeaponType"),
	m("Weapon", "getId", "", "$r:id()", "int"),

	// --- Enums ---
	fnm("PlayerClass", "getId", "", "int", same("int")),
	fnm("PlayerClass", "ordinal", "", "int", same("int")),
	m("PlayerClass", "level", "", "ClassId.level($r)", "int"),
	m("PlayerClass", "getLevel", "", "ClassId.level($r)", "int"),
	m("PlayerClass", "isMage", "", "ClassId.is_mage($r)", "boolean"),
	m("PlayerClass", "getRace", "", "ClassId.race($r)", "Race"),
	m("PlayerClass", "getParent", "", "ClassId.parent($r)", "PlayerClass"),
	m("PlayerClass", "childOf", "any", "ClassId.child_of($r, $1)", "boolean"),
	m("PlayerClass", "equalsOrChildOf", "any", "ClassId.equals_or_child_of($r, $1)", "boolean"),
	fnm("PlayerClass", "equals", "any", "boolean", equalsFn),
	fnm("static:PlayerClass", "getPlayerClass", "int", "PlayerClass", argAs("PlayerClass")),
	fnm("Race", "ordinal", "", "int", same("int")),
	fnm("Race", "equals", "any", "boolean", equalsFn),
	fnm("QuestSound", "getSoundName", "", "String", same("String")),
	fnm("ChatType", "ordinal", "", "int", same("int")),
	fnm("SystemMessageId", "getId", "", "int", same("int")),

	// --- Java: strings ---
	fnm("String", "equals", "any", "boolean", equalsFn),
	m("String", "equalsIgnoreCase", "str", "J.equals_ignore_case($r, $1)", "boolean"),
	m("String", "startsWith", "str", "J.starts_with($r, $1)", "boolean"),
	m("String", "endsWith", "str", "J.ends_with($r, $1)", "boolean"),
	m("String", "contains", "str", "J.str_contains($r, $1)", "boolean"),
	m("String", "substring", "int", "J.substring($r, $1)", "String"),
	m("String", "substring", "int,int", "J.substring($r, $1, $2)", "String"),
	m("String", "indexOf", "str", "J.index_of($r, $1)", "int"),
	m("String", "indexOf", "str,int", "J.index_of($r, $1, $2)", "int"),
	m("String", "lastIndexOf", "str", "J.last_index_of($r, $1)", "int"),
	m("String", "replace", "str,str", "J.replace($r, $1, $2)", "String"),
	m("String", "trim", "", "J.trim($r)", "String"),
	m("String", "toLowerCase", "", "string.lower($r)", "String"),
	m("String", "toUpperCase", "", "string.upper($r)", "String"),
	m("String", "charAt", "int", "J.char_at($r, $1)", "char"),
	m("String", "concat", "str", "$r .. $1", "String"),
	fnm("String", "split", "str", "String[]", splitFn),
	fnm("String", "replaceAll", "str,str", "String", regexReplaceFn),
	fnm("String", "replaceFirst", "str,str", "String", regexReplaceFn),
	fnm("String", "length", "", "int", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: "#" + asAtom(recv), prec: precUnary, t: tyInt, safe: true, pure: recv.pure}
	}),
	fnm("String", "isEmpty", "", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: paren(recv, precCmp+1) + ` == ""`, prec: precCmp, t: tyBool, pure: recv.pure}
	}),
	fnm("String", "isBlank", "", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("J.trim")
		return lx{code: "J.trim(" + recv.code + `) == ""`, prec: precCmp, t: tyBool}
	}),
	fnm("static:String", "valueOf", "any", "String", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return t.toStringValue(args[0])
	}),
	fnm("static:Integer", "toString", "any", "String", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return t.toStringValue(args[0])
	}),
	fnm("static:Long", "toString", "any", "String", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return t.toStringValue(args[0])
	}),
	fnm("static:String", "format", "str,any...", "String", formatFn),

	// --- Java: numbers ---
	m("static:Integer", "parseInt", "str", "J.parse_int($1)", "int"),
	m("static:Integer", "valueOf", "str", "J.parse_int($1)", "int"),
	fnm("static:Integer", "valueOf", "int", "int", argAs("int")),
	m("static:Long", "parseLong", "str", "J.parse_long($1)", "long"),
	m("static:Long", "valueOf", "str", "J.parse_long($1)", "long"),
	fnm("static:Long", "valueOf", "int", "long", argAs("long")),
	m("static:Double", "parseDouble", "str", "J.parse_double($1)", "double"),
	m("static:Boolean", "parseBoolean", "str", "J.parse_bool($1)", "boolean"),
	fnm("Number", "intValue", "", "int", same("int")),
	fnm("Number", "longValue", "", "long", same("long")),
	m("Number", "doubleValue", "", "$r", "double"),
	fnm("Integer", "equals", "any", "boolean", equalsFn),
	m("static:Math", "min", "num,num", "math.min($1, $2)", "num"),
	m("static:Math", "max", "num,num", "math.max($1, $2)", "num"),
	m("static:Math", "abs", "num", "math.abs($1)", "num"),
	m("static:Math", "floor", "num", "math.floor($1)", "double"),
	m("static:Math", "ceil", "num", "math.ceil($1)", "double"),
	m("static:Math", "round", "num", "J.round($1)", "long"),
	m("static:Math", "sqrt", "num", "math.sqrt($1)", "double"),
	m("static:Math", "pow", "num,num", "math.pow($1, $2)", "double"),
	m("static:Math", "random", "", "rnd.double()", "double"),
	m("static:MathUtil", "pow", "num,num", "math.pow($1, $2)", "double"),
	m("static:CommonUtil", "constrain", "num,num,num", "J.constrain($1, $2, $3)", "num"),
	m("static:MathUtil", "clamp", "num,num,num", "J.constrain($1, $2, $3)", "num"),
	m("static:System", "currentTimeMillis", "", "time.now_ms()", "long"),
	m("static:StringUtil", "isNumeric", "str", "J.is_numeric($1)", "boolean"),
	m("static:StringUtil", "isDigit", "str", "J.is_numeric($1)", "boolean"),
	m("static:Util", "isDigit", "str", "J.is_numeric($1)", "boolean"),
	m("static:ArrayUtil", "contains", "seq,any", "J.contains($1, $2)", "boolean"),
	m("static:CommonUtil", "contains", "seq,any", "J.contains($1, $2)", "boolean"),
	m("static:LocationUtil", "checkIfInRange", "int,WorldObject,WorldObject,bool", "util.in_range($1, $2, $3, $4)", "boolean"),
	m("static:Util", "checkIfInRange", "int,WorldObject,WorldObject,bool", "util.in_range($1, $2, $3, $4)", "boolean"),
	m("static:LocationUtil", "calculateDistance", "WorldObject,WorldObject,bool,bool", "util.distance($1, $2, $3)", "double"),

	// --- Java: collections ---
	fnm("static:Arrays", "asList", "any...", "List", asListFn),
	fnm("static:Arrays", "copyOf", "seq,int", "Array", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("J.copy_of")
		return lx{code: "J.copy_of(" + args[0].code + ", " + args[1].code + ")", prec: precAtom, t: args[0].t, call: true, safe: true}
	}),
	fnm("static:String", "join", "str,seq", "String", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("table.concat")
		return lx{code: "table.concat(" + args[1].code + ", " + args[0].code + ")", prec: precAtom, t: tyString, call: true, safe: true}
	}),
	m("List", "toString", "", "J.list_to_string($r)", "String"),
	fnm("static:List", "of", "any...", "List", tableOf),
	fnm("static:Set", "of", "any...", "Set", setOf),
	m("static:Arrays", "fill", "seq,any", "J.fill($1, $2)", ""),
	m("static:Arrays", "sort", "seq", "table.sort($1)", ""),
	m("static:Collections", "shuffle", "seq", "J.shuffle($1)", ""),
	m("static:Collections", "sort", "seq", "table.sort($1)", ""),
	fnm("Array", "clone", "", "", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("J.copy")
		return lx{code: "J.copy(" + recv.code + ")", prec: precAtom, t: recv.t, call: true, safe: true}
	}),

	fnm("Map", "get", "any", "V", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: asAtom(recv) + "[" + args[0].code + "]", prec: precAtom, t: recv.t.arg(1), pure: recv.pure && args[0].pure}
	}),
	fnm("Map", "put", "any,any", "V", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		if asStmt {
			return lx{code: asAtom(recv) + "[" + args[0].code + "] = " + args[1].code, call: true}
		}
		t.useAPI("J.put")
		return lx{code: "J.put(" + recv.code + ", " + args[0].code + ", " + args[1].code + ")", prec: precAtom, t: recv.t.arg(1), call: true}
	}),
	fnm("Map", "putIfAbsent", "any,any", "V", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("J.put_if_absent")
		return lx{code: "J.put_if_absent(" + recv.code + ", " + args[0].code + ", " + args[1].code + ")", prec: precAtom, t: recv.t.arg(1), call: true}
	}),
	fnm("Map", "containsKey", "any", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: asAtom(recv) + "[" + args[0].code + "] ~= nil", prec: precCmp, t: tyBool, pure: recv.pure && args[0].pure}
	}),
	fnm("Map", "remove", "any", "V", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		if asStmt {
			return lx{code: asAtom(recv) + "[" + args[0].code + "] = nil", call: true}
		}
		t.useAPI("J.remove_key")
		return lx{code: "J.remove_key(" + recv.code + ", " + args[0].code + ")", prec: precAtom, t: recv.t.arg(1), call: true}
	}),
	m("Map", "getOrDefault", "any,any", "J.get_or_default($r, $1, $2)", "V"),
	m("Map", "containsValue", "any", "J.contains_value($r, $1)", "boolean"),
	m("Map", "size", "", "J.size($r)", "int"),
	fnm("Map", "isEmpty", "", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: "next(" + recv.code + ") == nil", prec: precCmp, t: tyBool, pure: recv.pure}
	}),
	m("Map", "clear", "", "J.clear($r)", ""),
	fnm("Map", "keySet", "", "List<K>", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("J.keys")
		return lx{code: "J.keys(" + recv.code + ")", prec: precAtom, t: classType("List", recv.t.arg(0)), call: true, safe: true}
	}),
	fnm("Map", "values", "", "List<V>", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("J.values")
		return lx{code: "J.values(" + recv.code + ")", prec: precAtom, t: classType("List", recv.t.arg(1)), call: true, safe: true}
	}),
	fnm("Entry", "getKey", "", "K", entryPart(true)),
	fnm("Entry", "getValue", "", "V", entryPart(false)),

	fnm("List", "get", "int", "E", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		idx := t.indexPlusOne(c.args[0])
		return lx{code: asAtom(recv) + "[" + idx.code + "]", prec: precAtom, t: recv.t.arg(0), pure: recv.pure && idx.pure}
	}),
	fnm("List", "add", "any", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		if asStmt {
			return lx{code: "table.insert(" + recv.code + ", " + args[0].code + ")", call: true}
		}
		t.useAPI("J.add")
		return lx{code: "J.add(" + recv.code + ", " + args[0].code + ")", prec: precAtom, t: tyBool, call: true}
	}),
	fnm("List", "add", "int,any", "", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		idx := t.indexPlusOne(c.args[0])
		return lx{code: "table.insert(" + recv.code + ", " + idx.code + ", " + args[1].code + ")", prec: precAtom, call: true}
	}),
	fnm("List", "set", "int,any", "E", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		idx := t.indexPlusOne(c.args[0])
		if asStmt {
			return lx{code: asAtom(recv) + "[" + idx.code + "] = " + args[1].code, call: true}
		}
		return t.todo(c.line, "List.set as a value", "List.set used as a value")
	}),
	fnm("List", "size", "", "int", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: "#" + asAtom(recv), prec: precUnary, t: tyInt, safe: true, pure: recv.pure}
	}),
	fnm("List", "isEmpty", "", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: "#" + asAtom(recv) + " == 0", prec: precCmp, t: tyBool, pure: recv.pure}
	}),
	m("List", "contains", "any", "J.contains($r, $1)", "boolean"),
	fnm("List", "remove", "int", "E", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		if isIntegral(args[0].t) && !isA(args[0].t, "Integer") {
			idx := t.indexPlusOne(c.args[0])
			return lx{code: "table.remove(" + recv.code + ", " + idx.code + ")", prec: precAtom, t: recv.t.arg(0), call: true}
		}
		t.useAPI("J.remove_value")
		return lx{code: "J.remove_value(" + recv.code + ", " + args[0].code + ")", prec: precAtom, t: tyBool, call: true}
	}),
	m("List", "remove", "any", "J.remove_value($r, $1)", "boolean"),
	m("List", "indexOf", "any", "J.index_of_value($r, $1)", "int"),
	m("List", "clear", "", "J.clear($r)", ""),
	m("List", "addAll", "any", "J.add_all($r, $1)", "boolean"),
	fnm("List", "toArray", "any...", "Array", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		t.useAPI("J.copy")
		return lx{code: "J.copy(" + recv.code + ")", prec: precAtom, t: arrayOf(recv.t.arg(0)), call: true, safe: true}
	}),
	m("List", "getFirst", "", "$r[1]", "E"),
	fnm("Set", "add", "any", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		if asStmt {
			return lx{code: asAtom(recv) + "[" + args[0].code + "] = true", call: true}
		}
		t.useAPI("J.set_add")
		return lx{code: "J.set_add(" + recv.code + ", " + args[0].code + ")", prec: precAtom, t: tyBool, call: true}
	}),
	fnm("Set", "contains", "any", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: asAtom(recv) + "[" + args[0].code + "] ~= nil", prec: precCmp, t: tyBool, pure: recv.pure && args[0].pure}
	}),
	fnm("Set", "remove", "any", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		if asStmt {
			return lx{code: asAtom(recv) + "[" + args[0].code + "] = nil", call: true}
		}
		t.useAPI("J.set_remove")
		return lx{code: "J.set_remove(" + recv.code + ", " + args[0].code + ")", prec: precAtom, t: tyBool, call: true}
	}),
	m("Set", "size", "", "J.size($r)", "int"),
	fnm("Set", "isEmpty", "", "boolean", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return lx{code: "next(" + recv.code + ") == nil", prec: precCmp, t: tyBool, pure: recv.pure}
	}),
	m("Set", "clear", "", "J.clear($r)", ""),

	// --- World, global variables, data ---
	m("static:World", "getInstance", "", "world", "WorldView").asView(),
	m("WorldView", "getPlayer", "any", "world.get_player($1)", "Player"),
	m("WorldView", "findObject", "int", "world.find_object($1)", "WorldObject"),
	m("WorldView", "getNpc", "int", "world.find_object($1)", "Npc"),
	m("static:World", "getPlayer", "any", "world.get_player($1)", "Player"),
	m("static:World", "findObject", "int", "world.find_object($1)", "WorldObject"),
	m("static:GlobalVariablesManager", "getInstance", "", "globals", "GlobalVariables").asView(),
	m("GlobalVariables", "getLong", "str,int", "globals.get_long($1, $2)", "long").w(),
	m("GlobalVariables", "getInt", "str,int", "globals.get_long($1, $2)", "int").w(),
	m("GlobalVariables", "getString", "str,str", "globals.get($1, $2)", "String").w(),
	m("GlobalVariables", "getBoolean", "str,bool", "globals.get_bool($1, $2)", "boolean").w(),
	m("GlobalVariables", "set", "str,any", "globals.set($1, $2)", "").w(),
	m("GlobalVariables", "remove", "str", "globals.remove($1)", "").w(),
	m("GlobalVariables", "hasVariable", "str", "globals.has($1)", "boolean").w(),
	m("static:SkillData", "getInstance", "", "skills", "SkillData").asView(),
	m("SkillData", "getSkill", "int,int", "skills.get($1, $2)", "Skill"),
	m("static:ItemData", "getInstance", "", "items", "ItemData").asView(),
	m("ItemData", "getTemplate", "int", "items.get($1)", "ItemTemplate"),
	m("ItemTemplate", "getName", "", "$r.name", "String"),
	m("ItemTemplate", "getId", "", "$r.id", "int"),
	m("ItemTemplate", "isStackable", "", "$r.stackable", "boolean"),
	m("Skill", "applyEffects", "Creature,Creature", "skills.apply_effects($r, $1, $2)", "").w(),
	m("static:SevenSigns", "getInstance", "", "seven_signs", "SevenSigns").asView().feat("seven_signs"),
	m("SevenSigns", "getPlayerCabal", "int", "seven_signs.player_cabal($1)", "int").feat("seven_signs"),
	m("SevenSigns", "getSealOwner", "int", "seven_signs.seal_owner($1)", "int").feat("seven_signs"),
	m("SevenSigns", "isSealValidationPeriod", "", "seven_signs.is_seal_validation_period()", "boolean").feat("seven_signs"),

	// --- Packets: broadcastPacket / sendPacket take the packets built with new ---
	fnm("WorldObject", "broadcastPacket", "any", "", broadcastPacketFn),
	fnm("Creature", "broadcastPacket", "any", "", broadcastPacketFn),
	fnm("Player", "sendPacket", "any", "", sendPacketFn),
	fnm("SystemMessage", "addString", "any", "SystemMessage", packetArg),
	fnm("SystemMessage", "addInt", "any", "SystemMessage", packetArg),
	fnm("SystemMessage", "addLong", "any", "SystemMessage", packetArg),
	fnm("SystemMessage", "addItemName", "any", "SystemMessage", packetArg),
	fnm("SystemMessage", "addNpcName", "any", "SystemMessage", packetArg),
	fnm("SystemMessage", "addSkillName", "any", "SystemMessage", packetArg),
	fnm("SystemMessage", "addPcName", "any", "SystemMessage", packetArg),
	fnm("SystemMessage", "addNumber", "any", "SystemMessage", packetArg),
	fnm("static:SystemMessage", "getSystemMessage", "any", "SystemMessage", func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		return t.packetValue(c.line, "SystemMessage", args)
	}),
}

var (
	// mappingIndex finds entries by receiver and method name.
	mappingIndex = map[string][]*mapEntry{}
	// mappedByName finds instance method entries by name alone, for
	// receivers of unknown type.
	mappedByName = map[string][]*mapEntry{}
)

func init() {
	for _, e := range mappings {
		key := e.recv + "." + e.name
		mappingIndex[key] = append(mappingIndex[key], e)
		if e.recv != "Quest" && !strings.HasPrefix(e.recv, "static:") {
			mappedByName[e.name] = append(mappedByName[e.name], e)
		}
	}
}

// holderFields are public fields of library classes the quests read.
var holderFields = map[string]struct {
	lua string
	t   *jtype
}{}

// argMatches reports whether an argument of type t fits pattern p.
func argMatches(p string, t *jtype) bool {
	if isUnknown(t) {
		return true
	}
	switch p {
	case "any":
		return true
	case "int":
		return isIntegral(t) || isChar(t) || numericEnums[t.name]
	case "num":
		return isNumeric(t)
	case "dbl":
		return isFloating(t)
	case "bool":
		return isBool(t)
	case "str":
		return isString(t) || isChar(t) || t.kind == tkNull
	case "obj":
		return t.kind != tkPrim
	case "seq":
		return isSequence(t) || isSet(t) || t.kind == tkNull
	case "pos":
		return t.kind == tkNull || isA(t, "Location") || isA(t, "WorldObject") || isA(t, "ILocational") || isA(t, "IPositionable")
	}
	if t.kind == tkNull {
		return true
	}
	return isA(t, p)
}

// numericEnums are enum classes whose Lua values are numbers.
var numericEnums = map[string]bool{
	"PlayerClass": true, "Race": true, "ChatType": true, "CategoryType": true,
	"SystemMessageId": true, "QuestType": true, "Sex": true,
}

// matches reports whether the entry takes these arguments.
func (e *mapEntry) matches(args []lx) bool {
	n := len(e.args)
	if n > 0 && strings.HasSuffix(e.args[n-1], "...") {
		fixed := n - 1
		if len(args) < fixed {
			return false
		}
		for i := 0; i < fixed; i++ {
			if !argMatches(e.args[i], args[i].t) {
				return false
			}
		}
		vp := strings.TrimSuffix(e.args[n-1], "...")
		rest := args[fixed:]
		if len(rest) == 1 && (isSequence(rest[0].t) || isSet(rest[0].t)) {
			return argMatches(vp, elemType(rest[0].t))
		}
		for _, a := range rest {
			if !argMatches(vp, a.t) {
				return false
			}
		}
		return true
	}
	if len(args) != n {
		return false
	}
	for i := range args {
		if !argMatches(e.args[i], args[i].t) {
			return false
		}
	}
	return true
}

var apiPattern = regexp.MustCompile(`(q|J|rnd|util|world|globals|time|math|table|string|skills|items|ClassId|seven_signs|clan_halls|\$r|\$b|\$1|\$2)([.:])([a-z_0-9]+)\(`)

// apiName names the Lua function a template calls, for the report.
func (e *mapEntry) apiName() string {
	tmpl := e.lua
	if tmpl == "" {
		return ""
	}
	if strings.HasPrefix(tmpl, "rnd(") {
		return "rnd"
	}
	mm := apiPattern.FindStringSubmatch(tmpl)
	if mm == nil {
		return ""
	}
	owner := mm[1]
	switch owner {
	case "$r", "$b", "$1", "$2":
		owner = e.recv
		if e.recv == "Quest" {
			owner = "Player"
		}
	}
	return owner + mm[2] + mm[3]
}

// args translates call arguments.
func (t *translator) args(es []expr) []lx {
	out := make([]lx, len(es))
	for i, e := range es {
		out[i] = t.expr(e)
	}
	return out
}

// call translates a method call. asStmt is set when the result is unused.
func (t *translator) call(c *callExpr, asStmt bool) lx {
	if c.x == nil && c.name == "super" {
		return t.todo(c.line, "super(...)", "constructor call super(...)")
	}
	if r, ok := t.specialCall(c); ok {
		return r
	}
	if c.x != nil {
		switch x := unparen(c.x).(type) {
		case *superExpr:
			return t.superCall(c, asStmt)
		case *classLitExpr:
			if c.name == "getSimpleName" || c.name == "getName" {
				return lx{code: luaQuote(x.typ.name), prec: precAtom, t: tyString, safe: true, pure: true, strLit: true}
			}
		case *callExpr:
			if c.name == "getSimpleName" && x.name == "getClass" && (x.x == nil || isThis(x.x)) {
				t.useAPI("q:name")
				return lx{code: "q:name()", prec: precAtom, t: tyString, call: true, safe: true}
			}
		}
	}
	if c.x == nil || isThis(c.x) {
		if t.fn != nil && t.fn.inner != nil && !t.fn.static {
			if mi := pickMethod(t.fn.inner.methods[c.name], c.args); mi != nil {
				args := t.methodArgs(mi, c.args)
				return lx{code: "self:" + mi.lua + "(" + strings.Join(args, ", ") + ")", prec: precAtom, t: mi.ret, call: true, safe: isNumeric(mi.ret)}
			}
		}
		if mi := pickMethod(t.methods[c.name], c.args); mi != nil {
			t.upval(mi.lua)
			args := t.methodArgs(mi, c.args)
			return lx{code: mi.lua + "(" + strings.Join(args, ", ") + ")", prec: precAtom, t: mi.ret, call: true, safe: isNumeric(mi.ret) && mi.ret.kind == tkPrim}
		}
		if t.fn != nil && t.fn.inner != nil {
			// A static method of an inner class called from inside it.
			if mi := pickMethod(t.fn.inner.methods[c.name], c.args); mi != nil {
				args := t.methodArgs(mi, c.args)
				return lx{code: t.fn.inner.lua + "." + mi.lua + "(" + strings.Join(args, ", ") + ")", prec: precAtom, t: mi.ret, call: true}
			}
		}
		return t.mapped([]string{"Quest"}, lx{}, c, asStmt)
	}
	if id, ok := unparen(c.x).(*identExpr); ok && t.lookupLocal(id.name) == nil && t.fields[id.name] == nil && !t.innerField(id.name) {
		if ic := t.inner[id.name]; ic != nil {
			if mi := pickMethod(ic.methods[c.name], c.args); mi != nil {
				args := t.methodArgs(mi, c.args)
				return lx{code: ic.lua + "." + mi.lua + "(" + strings.Join(args, ", ") + ")", prec: precAtom, t: mi.ret, call: true}
			}
		}
		if id.name == "Quest" || id.name == "Script" || id.name == "AbstractSagaQuest" {
			return t.mapped([]string{"Quest"}, lx{}, c, asStmt)
		}
		if isTypeName(id.name) {
			return t.mapped([]string{"static:" + id.name}, lx{}, c, asStmt)
		}
	}
	if c.name == "forEach" && len(c.args) == 1 {
		if _, ok := unparen(c.args[0]).(*lambdaExpr); ok && asStmt {
			return t.forEachLambda(c)
		}
	}
	recv := t.expr(c.x)
	if recv.isType != "" {
		return t.mapped([]string{"static:" + recv.isType}, lx{}, c, asStmt)
	}
	if strings.HasPrefix(recv.code, "J.todo(") {
		// One marker for the whole chain.
		return recv
	}
	if recv.t != nil {
		if ic := t.inner[recv.t.name]; ic != nil {
			if mi := pickMethod(ic.methods[c.name], c.args); mi != nil {
				args := t.methodArgs(mi, c.args)
				return lx{code: asAtom(recv) + ":" + mi.lua + "(" + strings.Join(args, ", ") + ")", prec: precAtom, t: mi.ret, call: true, safe: isNumeric(mi.ret) && mi.ret.kind == tkPrim}
			}
		}
	}
	if recv.packet != "" {
		return t.mapped([]string{recv.packet}, recv, c, asStmt)
	}
	if recv.entryOf != "" {
		return t.mapped([]string{"Entry"}, recv, c, asStmt)
	}
	chain := typeChain(recv.t)
	if isUnknown(recv.t) {
		chain = nil
	}
	return t.mapped(chain, recv, c, asStmt)
}

// superCall handles super.method(...): the Mobius defaults do nothing.
func (t *translator) superCall(c *callExpr, asStmt bool) lx {
	if _, ok := callbacks[c.name]; ok {
		if asStmt {
			return lx{}
		}
		return lx{code: "nil", prec: precAtom, t: tyNull, nilLit: true}
	}
	return t.mapped([]string{"Quest"}, lx{}, c, asStmt)
}

// pickMethod chooses a declared method for a call: by arity, then by
// argument types.
func pickMethod(cands []*methodInfo, args []expr) *methodInfo {
	var byArity []*methodInfo
	for _, mi := range cands {
		if len(mi.params) == len(args) || (mi.varargs && len(args) >= len(mi.params)-1) {
			byArity = append(byArity, mi)
		}
	}
	if len(byArity) == 0 {
		return nil
	}
	return byArity[0]
}

// methodArgs translates the arguments of a call to a script method. Java
// varargs become one table.
func (t *translator) methodArgs(mi *methodInfo, es []expr) []string {
	args := t.args(es)
	out := make([]string, 0, len(args))
	n := len(mi.params)
	for i, a := range args {
		if mi.varargs && i == n-1 {
			if len(args) == n && (isSequence(a.t) || isUnknown(a.t)) {
				out = append(out, a.code)
				break
			}
			var rest []string
			for _, r := range args[i:] {
				rest = append(rest, r.code)
			}
			out = append(out, "{"+strings.Join(rest, ", ")+"}")
			break
		}
		out = append(out, a.code)
	}
	return out
}

// mapped looks the call up in the mapping table for each receiver kind in
// chain.
func (t *translator) mapped(chain []string, recv lx, c *callExpr, asStmt bool) lx {
	args := t.args(c.args)
	for _, kind := range chain {
		for _, e := range mappingIndex[kind+"."+c.name] {
			if e.matches(args) {
				return t.apply(e, recv, args, c, asStmt)
			}
		}
	}
	if len(chain) == 0 || (len(chain) == 1 && chain[0] == "Object") {
		// Unknown receiver type: use the method if only one receiver kind
		// has it.
		var found *mapEntry
		kinds := map[string]bool{}
		for _, e := range mappedByName[c.name] {
			if e.matches(args) {
				if found == nil {
					found = e
				}
				kinds[e.recv] = true
			}
		}
		if found != nil && len(kinds) == 1 {
			return t.apply(found, recv, args, c, asStmt)
		}
	}
	owner := "?"
	if len(chain) > 0 {
		owner = strings.TrimPrefix(chain[0], "static:")
	}
	key := fmt.Sprintf("%s.%s", owner, c.name)
	if len(chain) > 0 && chain[0] == "Quest" {
		key = "Quest." + c.name
	}
	types := make([]string, len(args))
	for i, a := range args {
		types[i] = a.t.String()
	}
	return t.todo(c.line, "call "+key, fmt.Sprintf("unmapped call %s(%s)", key, strings.Join(types, ", ")))
}

// apply writes a call through a mapping entry.
func (t *translator) apply(e *mapEntry, recv lx, args []lx, c *callExpr, asStmt bool) lx {
	if e.world && t.fn != nil {
		t.fn.world = true
	}
	t.requires(e.feature)
	if e.fn != nil {
		r := e.fn(t, c, recv, args, asStmt)
		if r.code == "" && !asStmt {
			return t.todo(c.line, "call "+e.recv+"."+e.name, "no Lua equivalent for "+e.recv+"."+e.name)
		}
		if r.t == nil || (isUnknown(r.t) && e.ret != "") {
			r.t = t.retType(e.ret, recv, args)
		}
		return r
	}
	tmpl := e.lua
	if asStmt && e.stmt != "" {
		tmpl = e.stmt
	}
	if e.alt != "" && recv.base == "" {
		tmpl = e.alt
	}
	code, ok := t.expand(tmpl, recv, args, e.keep)
	if !ok {
		return t.todo(c.line, "call "+e.recv+"."+e.name, "argument with side effects used twice in "+e.recv+"."+e.name)
	}
	t.useAPI(e.apiName())
	ret := t.retType(e.ret, recv, args)
	res := lx{code: code, prec: precAtom, t: ret, call: true}
	if e.view {
		res.base = recv.code
		if recv.base != "" {
			res.base = recv.base
		}
		if tmpl == "$r" {
			res.base = recv.code
			res.call = recv.call
		}
		if !strings.Contains(tmpl, "$") {
			res.call = false
			res.base = ""
		}
	}
	if !strings.HasSuffix(code, ")") || strings.HasPrefix(tmpl, "#") || isCmpTemplate(tmpl) {
		res.call = false
		switch {
		case strings.HasPrefix(tmpl, "#"):
			res.prec = precUnary
		case isCmpTemplate(tmpl):
			res.prec = precCmp
		case strings.Contains(tmpl, " .. "):
			res.prec = precConcat
		}
	}
	if isNumeric(ret) && ret.kind == tkPrim {
		res.safe = true
	}
	return res
}

// isCmpTemplate reports whether a template's top level is a comparison.
func isCmpTemplate(tmpl string) bool {
	for _, op := range []string{" == ", " ~= ", " >= ", " <= ", " < ", " > "} {
		if strings.Contains(tmpl, op) {
			return true
		}
	}
	return false
}

// retType resolves an entry's result type name.
func (t *translator) retType(name string, recv lx, args []lx) *jtype {
	switch name {
	case "":
		return tyVoid
	case "int":
		return tyInt
	case "long":
		return tyLong
	case "double":
		return tyDouble
	case "boolean":
		return tyBool
	case "String":
		return tyString
	case "char":
		return tyChar
	case "num":
		if len(args) >= 2 {
			return binaryNumeric(args[0].t, args[1].t)
		}
		if len(args) == 1 {
			return args[0].t
		}
		return tyUnknown
	case "V":
		return recv.t.arg(1)
	case "E":
		return recv.t.arg(0)
	case "K":
		return recv.t.arg(0)
	case "int[]":
		return arrayOf(tyInt)
	case "String[]":
		return arrayOf(tyString)
	case "List<Player>":
		return classType("List", classType("Player"))
	case "List<Monster>":
		return classType("List", classType("Monster"))
	case "List<K>":
		return classType("List", recv.t.arg(0))
	case "List<V>":
		return classType("List", recv.t.arg(1))
	case "Array", "List", "Set":
		return classType(name, elemOf(args))
	}
	return classType(name)
}

// expand fills a template. It reports false when an argument with side
// effects would be evaluated twice and cannot be hoisted.
func (t *translator) expand(tmpl string, recv lx, args []lx, keep bool) (string, bool) {
	// Arguments used twice are evaluated once.
	uses := map[int]int{}
	for i := 0; i+1 < len(tmpl); i++ {
		if tmpl[i] == '$' && tmpl[i+1] >= '1' && tmpl[i+1] <= '9' {
			uses[int(tmpl[i+1]-'1')]++
		}
	}
	for i, n := range uses {
		if n > 1 && i < len(args) && !args[i].pure {
			tmp := t.temp("v")
			if !t.hoist("local " + tmp + " = " + args[i].code) {
				return "", false
			}
			args[i] = atom(tmp, args[i].t)
		}
	}
	var b strings.Builder
	for i := 0; i < len(tmpl); i++ {
		ch := tmpl[i]
		if ch != '$' || i+1 >= len(tmpl) {
			b.WriteByte(ch)
			continue
		}
		nx := tmpl[i+1]
		switch {
		case nx == 'r':
			b.WriteString(asAtom(recv))
			i++
		case nx == 'b':
			if recv.base != "" {
				b.WriteString(recv.base)
			} else {
				b.WriteString(asAtom(recv))
			}
			i++
		case nx >= '1' && nx <= '9':
			n := int(nx - '1')
			i++
			if i+1 < len(tmpl) && tmpl[i+1] == '*' {
				i++
				rest := args[min(n, len(args)):]
				if len(rest) == 0 {
					s := b.String()
					s = strings.TrimSuffix(s, ", ")
					b.Reset()
					b.WriteString(s)
					continue
				}
				parts := make([]string, len(rest))
				for j, a := range rest {
					parts[j] = a.code
					if len(rest) == 1 && !keep && (isSequence(a.t) || isSet(a.t)) {
						if isSet(a.t) {
							t.useAPI("J.keys")
							parts[j] = "unpack(J.keys(" + a.code + "))"
						} else {
							parts[j] = "unpack(" + a.code + ")"
						}
					}
					if keep && isSet(a.t) {
						t.useAPI("J.keys")
						parts[j] = "J.keys(" + a.code + ")"
					}
				}
				b.WriteString(strings.Join(parts, ", "))
				continue
			}
			if n < len(args) {
				if i+1 < len(tmpl) && (tmpl[i+1] == '.' || tmpl[i+1] == ':') {
					// A field or method of the argument: $1.id, $1:name().
					b.WriteString(asAtom(args[n]))
				} else {
					b.WriteString(args[n].code)
				}
			}
		default:
			b.WriteByte(ch)
		}
	}
	return b.String(), true
}

// --- Custom mappings ---

func equalsFn(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	return lx{code: paren(recv, precCmp+1) + " == " + paren(args[0], precCmp+1), prec: precCmp, t: tyBool, pure: recv.pure && args[0].pure}
}

// entryPart reads the key or value of a Map.Entry loop variable.
func entryPart(key bool) func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	return func(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
		name := recv.base
		typ := recv.t.arg(1)
		if key {
			name = recv.entryOf
			typ = recv.t.arg(0)
		}
		return lx{code: name, prec: precAtom, t: typ, pure: true}
	}
}

func setOf(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = "[" + a.code + "] = true"
	}
	return lx{code: "{" + strings.Join(parts, ", ") + "}", prec: precAtom, t: classType("Set", elemOf(args)), safe: true}
}

// toStringValue is String.valueOf(x).
func (t *translator) toStringValue(x lx) lx {
	switch {
	case isString(x.t) || isChar(x.t):
		return x
	case isFloating(x.t):
		t.useAPI("J.str")
		return lx{code: "J.str(" + x.code + ")", prec: precAtom, t: tyString, call: true, safe: true}
	}
	return lx{code: "tostring(" + x.code + ")", prec: precAtom, t: tyString, call: true, safe: true}
}

var regexMeta = regexp.MustCompile(`[\\.\[\]{}()*+?^$|]`)

// splitFn translates String.split: only literal separators.
func splitFn(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	sep, ok := unparen(c.args[0]).(*literalExpr)
	if !ok || sep.kind != litString {
		return t.todo(c.line, "split with a variable pattern", "String.split with a non-literal pattern")
	}
	pattern := sep.value
	if regexMeta.MatchString(pattern) {
		// A few regular expressions are just literal characters.
		switch pattern {
		case `\.`:
			pattern = "."
		case `\|`:
			pattern = "|"
		case `\s+`, `\s`:
			// Whitespace: runs of it with \s+, each char with \s.
			t.useAPI("J.split_ws")
			code := "J.split_ws(" + recv.code + ")"
			if pattern == `\s` {
				code = "J.split_ws(" + recv.code + ", true)"
			}
			return lx{code: code, prec: precAtom, t: arrayOf(tyString), call: true, safe: true}
		default:
			return t.todo(c.line, "split with a regex", "String.split with the regular expression "+sep.value)
		}
	}
	t.useAPI("J.split")
	return lx{code: "J.split(" + recv.code + ", " + luaQuote(pattern) + ")", prec: precAtom, t: arrayOf(tyString), call: true, safe: true}
}

// regexToLuaPattern maps the few Java regular expressions quests pass to
// replaceAll (with an empty replacement) to equal Lua patterns. Keys are
// the decoded string values.
var regexToLuaPattern = map[string]string{
	`\D`:      "%D",
	`[^\d ]`:  "[^%d ]",
	`[^0-9]`:  "%D",
	`[^0-9 ]`: "[^%d ]",
}

// regexReplaceFn translates replaceAll/replaceFirst when the pattern is a
// literal without regular expression characters, or a known character
// class removal (regexToLuaPattern).
func regexReplaceFn(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	pat, ok := unparen(c.args[0]).(*literalExpr)
	if ok && pat.kind == litString && c.name == "replaceAll" {
		// Character class removals the quests use, as fixed Lua patterns.
		if lp, known := regexToLuaPattern[pat.value]; known {
			if rep, ok := unparen(c.args[1]).(*literalExpr); ok && rep.kind == litString && rep.value == "" {
				t.useAPI("string.gsub")
				return lx{code: "(string.gsub(" + recv.code + ", " + luaQuote(lp) + `, ""))`, prec: precAtom, t: tyString, call: true, safe: true}
			}
		}
	}
	if !ok || pat.kind != litString || regexMeta.MatchString(pat.value) {
		return t.todo(c.line, "String."+c.name+" with a regex", "String."+c.name+" with a regular expression")
	}
	// Unlike String.replace, the replacement follows the regex rules ($0, \x).
	fn := "J.replace_all"
	if c.name == "replaceFirst" {
		fn = "J.replace_first"
	}
	t.useAPI(fn)
	return lx{code: fn + "(" + recv.code + ", " + args[0].code + ", " + args[1].code + ")", prec: precAtom, t: tyString, call: true, safe: true}
}

// formatFn translates String.format with a literal format of %d and %s.
func formatFn(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	f, ok := unparen(c.args[0]).(*literalExpr)
	if !ok || f.kind != litString || strings.ContainsAny(strings.NewReplacer("%d", "", "%s", "", "%%", "").Replace(f.value), "%") {
		return t.todo(c.line, "String.format", "String.format with this format")
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.code
		if i > 0 && !isIntegral(a.t) && !isString(a.t) {
			parts[i] = "tostring(" + a.code + ")"
		}
	}
	t.useAPI("string.format")
	return lx{code: "string.format(" + strings.Join(parts, ", ") + ")", prec: precAtom, t: tyString, call: true, safe: true}
}

// forEachLambda writes list.forEach(x -> { ... }) as a loop.
func (t *translator) forEachLambda(c *callExpr) lx {
	l := unparen(c.args[0]).(*lambdaExpr)
	if len(l.params) != 1 {
		return t.todo(c.line, "forEach lambda", "forEach with a lambda of several parameters")
	}
	coll := t.expr(c.x)
	t.flushPre()
	t.flushTodoComments()
	t.pushScope()
	lv := t.declare(l.params[0], elemType(coll.t))
	if isSet(coll.t) {
		t.useAPI("J.spairs")
		t.w.linef("for %s in J.spairs(%s) do", lv.lua, coll.code)
	} else {
		t.w.linef("for _, %s in ipairs(%s) do", lv.lua, coll.code)
	}
	t.w.ind++
	t.fn.targets = append(t.fn.targets, &jumpTarget{loop: true, luaLoop: true})
	if l.block != nil {
		// A lambda's return ends the iteration: like continue.
		if hasReturn(l.block) {
			t.fn.targets = t.fn.targets[:len(t.fn.targets)-1]
			t.popScope()
			t.w.ind--
			t.w.line("end")
			return t.todo(c.line, "forEach lambda with return", "forEach lambda that returns")
		}
		t.stmtList(l.block.stmts)
	} else {
		r := t.expr(l.body)
		t.writeExprStmt(r)
	}
	t.fn.targets = t.fn.targets[:len(t.fn.targets)-1]
	t.w.ind--
	t.w.line("end")
	t.popScope()
	return lx{}
}

func hasReturn(n any) bool {
	found := false
	inspect(n, func(x any) bool {
		if _, ok := x.(*returnStmt); ok {
			found = true
		}
		return !found
	})
	return found
}

// --- Objects and packets ---

var packetClasses = map[string]bool{
	"SocialAction": true, "MagicSkillUse": true, "NpcSay": true, "CreatureSay": true,
	"SystemMessage": true, "ExShowScreenMessage": true, "PlaySound": true,
	"NpcHtmlMessage": true, "RadarControl": true, "SpecialCamera": true,
	"ExShowSlideshowKamael": true, "PledgeShowInfoUpdate": true, "InventoryUpdate": true,
	"TeleportToLocation": true, "UserInfo": true, "ActionFailed": true, "StatusUpdate": true,
	"TutorialShowHtml": true, "TutorialShowQuestionMark": true, "TutorialCloseHtml": true,
	"ExShowQuestMark": true, "QuestList": true, "Earthquake": true, "MagicSkillLaunched": true,
}

// packetValue builds the value of a packet created with new. It stays a
// J.todo marker unless broadcastPacket or sendPacket consumes it.
func (t *translator) packetValue(line int, name string, args []lx) lx {
	idx := len(t.res.todos)
	r := t.todo(line, "packet "+name, "packet "+name+" used outside broadcastPacket/sendPacket")
	r.packet = name
	r.pargs = args
	r.intVal = int64(idx)
	r.t = classType(name)
	return r
}

// consumePacket removes the marker of a packet that a mapping handles.
func (t *translator) consumePacket(p lx) {
	idx := int(p.intVal)
	if idx < len(t.res.todos) && strings.HasPrefix(t.res.todos[idx].key, "packet ") {
		line := t.res.todos[idx].line
		t.res.todos = slices.Delete(t.res.todos, idx, idx+1)
		if t.fn != nil {
			for i := len(t.fn.todoLines) - 1; i >= 0; i-- {
				if t.fn.todoLines[i] == line {
					t.fn.todoLines = slices.Delete(t.fn.todoLines, i, i+1)
					break
				}
			}
		}
	}
}

func packetArg(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	t.consumePacket(recv)
	p := t.packetValue(c.line, recv.packet, append(append([]lx{}, recv.pargs...), args...))
	return p
}

func broadcastPacketFn(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	p := args[0]
	pa := p.pargs
	r := asAtom(recv)
	var code string
	switch {
	case p.packet == "SocialAction" && len(pa) == 2:
		t.useAPI("Creature:social_action")
		code = r + ":social_action(" + pa[1].code + ")"
	case p.packet == "MagicSkillUse" && len(pa) >= 6:
		t.useAPI("Creature:magic_skill_use")
		code = asAtom(pa[0]) + ":magic_skill_use(" + joinCodes(pa[1:6]) + ")"
	case p.packet == "MagicSkillUse" && len(pa) == 5:
		t.useAPI("Creature:magic_skill_use")
		code = asAtom(pa[0]) + ":magic_skill_use(" + pa[0].code + ", " + joinCodes(pa[1:5]) + ")"
	case p.packet == "NpcSay" && len(pa) == 3 && !isIntegral(pa[0].t):
		t.useAPI("Npc:broadcast_say")
		code = asAtom(pa[0]) + ":broadcast_say(" + pa[1].code + ", " + pa[2].code + ")"
	case p.packet == "NpcSay" && len(pa) == 4:
		t.useAPI("Npc:broadcast_say")
		code = r + ":broadcast_say(" + pa[1].code + ", " + pa[3].code + ")"
	case p.packet == "CreatureSay" && len(pa) >= 3:
		t.useAPI("Npc:broadcast_say")
		code = r + ":broadcast_say(" + pa[len(pa)-2].code + ", " + pa[len(pa)-1].code + ")"
	case p.packet == "PlaySound" && len(pa) == 1:
		t.useAPI("Creature:broadcast_sound")
		code = r + ":broadcast_sound(" + pa[0].code + ")"
	case p.packet == "Earthquake" && len(pa) >= 5:
		t.useAPI("Creature:earthquake")
		code = r + ":earthquake(" + pa[3].code + ", " + pa[4].code + ")"
	case p.packet == "TeleportToLocation" && len(pa) >= 4:
		// The teleport animation of the object (first argument) to x, y, z;
		// Mobius moves it with setXYZ right after.
		t.useAPI("Creature:broadcast_teleport")
		code = asAtom(pa[0]) + ":broadcast_teleport(" + joinCodes(pa[1:4]) + ")"
	default:
		if p.packet == "" {
			return t.todo(c.line, "broadcastPacket", "broadcastPacket of a value that is not a known packet")
		}
		return t.todo(c.line, "broadcast "+p.packet, "broadcastPacket(new "+p.packet+")")
	}
	t.consumePacket(p)
	return lx{code: code, prec: precAtom, t: tyVoid, call: true}
}

func sendPacketFn(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	p := args[0]
	pa := p.pargs
	r := asAtom(recv)
	var code string
	switch {
	case p.packet == "" && isA(p.t, "SystemMessageId"):
		t.useAPI("Player:send_system_message")
		return lx{code: r + ":send_system_message(" + p.code + ")", prec: precAtom, t: tyVoid, call: true}
	case p.packet == "SystemMessage" && len(pa) >= 1:
		t.useAPI("Player:send_system_message")
		code = r + ":send_system_message(" + joinCodes(pa) + ")"
	case p.packet == "ExShowScreenMessage" && len(pa) == 2:
		t.useAPI("q:show_on_screen_msg")
		code = "q:show_on_screen_msg(" + recv.code + ", " + pa[0].code + ", " + pa[1].code + ")"
	case p.packet == "PlaySound" && len(pa) == 1:
		t.useAPI("q:play_sound")
		code = "q:play_sound(" + recv.code + ", " + pa[0].code + ")"
	case p.packet == "SocialAction" && len(pa) == 2:
		t.review(c.line, "social action sent to one player", "sendPacket(new SocialAction) becomes a broadcast")
		t.useAPI("Creature:social_action")
		code = r + ":social_action(" + pa[1].code + ")"
	case p.packet == "TutorialShowHtml" && len(pa) == 1:
		t.useAPI("Player:show_tutorial_html")
		code = r + ":show_tutorial_html(" + pa[0].code + ")"
	case p.packet == "TutorialCloseHtml":
		t.useAPI("Player:close_tutorial_html")
		code = r + ":close_tutorial_html()"
	case p.packet == "RadarControl" && len(pa) == 5:
		t.useAPI("Player:radar_control")
		code = r + ":radar_control(" + joinCodes(pa) + ")"
	case p.packet == "ExShowSlideshowKamael" && len(pa) == 0:
		t.useAPI("Player:show_slideshow_kamael")
		code = r + ":show_slideshow_kamael()"
	default:
		if p.packet == "" {
			return t.todo(c.line, "sendPacket", "sendPacket of a value that is not a known packet")
		}
		return t.todo(c.line, "send "+p.packet, "sendPacket(new "+p.packet+")")
	}
	t.consumePacket(p)
	return lx{code: code, prec: precAtom, t: tyVoid, call: true}
}

// addStatFuncFn is creature.addStatFunc(new FuncSet(stat, order, owner,
// value, null)): the stat is fixed at value from now on.
func addStatFuncFn(t *translator, c *callExpr, recv lx, args []lx, asStmt bool) lx {
	if !isA(args[0].t, "FuncSet") || len(args[0].pargs) != 2 {
		return t.todo(c.line, "call Creature.addStatFunc", "Creature.addStatFunc with a stat function other than FuncSet")
	}
	t.useAPI("Creature:set_stat")
	return lx{code: asAtom(recv) + ":set_stat(" + joinCodes(args[0].pargs) + ")", prec: precAtom, t: tyVoid, call: true}
}

func joinCodes(xs []lx) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = x.code
	}
	return strings.Join(parts, ", ")
}

// newObject translates new T(...).
func (t *translator) newObject(v *newExpr) lx {
	name := v.typ.name
	if v.anon {
		return t.todo(v.line, "anonymous class", "anonymous class new "+name)
	}
	if ic := t.inner[name]; ic != nil {
		var mi *methodInfo
		for _, c := range ic.ctors {
			if len(c.params) == len(v.args) {
				mi = c
				break
			}
		}
		if mi == nil && (len(ic.ctors) > 0 || len(v.args) > 0) {
			return t.todo(v.line, "inner constructor", "no constructor of "+name+" with these arguments")
		}
		lua := "new"
		if mi != nil {
			lua = mi.lua
		}
		args := t.args(v.args)
		return lx{code: ic.lua + "." + lua + "(" + joinCodes(args) + ")", prec: precAtom, t: classType(name), call: true, safe: true}
	}
	args := t.args(v.args)
	typ := typeFromRef(v.typ)
	switch {
	case mapTypes[name] || listTypes[name] || setTypes[name]:
		if len(args) == 0 || isIntegral(args[0].t) {
			return lx{code: "{}", prec: precAtom, t: typ, safe: true, pure: true}
		}
		if setTypes[name] && !isSet(args[0].t) {
			t.useAPI("J.set_of")
			return lx{code: "J.set_of(" + args[0].code + ")", prec: precAtom, t: typ, call: true, safe: true}
		}
		t.useAPI("J.copy")
		return lx{code: "J.copy(" + args[0].code + ")", prec: precAtom, t: typ, call: true, safe: true}
	case packetClasses[name]:
		return t.packetValue(v.line, name, args)
	}
	switch name {
	case "Location":
		keys := []string{"x", "y", "z", "heading"}
		if len(args) >= 3 && len(args) <= 4 {
			parts := make([]string, len(args))
			for i, a := range args {
				parts[i] = keys[i] + " = " + a.code
			}
			return lx{code: "{" + strings.Join(parts, ", ") + "}", prec: precAtom, t: classType("Location"), safe: true, pure: true}
		}
	case "ItemHolder":
		if len(args) == 2 {
			return lx{code: "{id = " + args[0].code + ", count = " + args[1].code + "}", prec: precAtom, t: classType("ItemHolder"), safe: true, pure: true}
		}
	case "ItemChanceHolder":
		if len(args) == 3 {
			return lx{code: "{id = " + args[0].code + ", chance = " + args[1].code + ", count = " + args[2].code + "}", prec: precAtom, t: classType("ItemChanceHolder"), safe: true, pure: true}
		}
	case "SkillHolder":
		if len(args) == 2 {
			return lx{code: "{id = " + args[0].code + ", level = " + args[1].code + "}", prec: precAtom, t: classType("SkillHolder"), safe: true, pure: true}
		}
		if len(args) == 1 {
			return lx{code: "{id = " + args[0].code + ", level = 1}", prec: precAtom, t: classType("SkillHolder"), safe: true, pure: true}
		}
	case "String":
		if len(args) == 0 {
			return lx{code: `""`, prec: precAtom, t: tyString, safe: true, pure: true}
		}
		return args[0]
	case "Object":
		return lx{code: "{}", prec: precAtom, t: classType("Object"), safe: true}
	case "FuncSet":
		// new FuncSet(stat, order, owner, value, condition): a stat set to a
		// fixed value. Only Creature.addStatFunc consumes it.
		if len(args) == 5 && isNullLit(v.args[4]) {
			return lx{code: "{stat = " + args[0].code + ", value = " + args[3].code + "}", prec: precAtom, t: classType("FuncSet"), safe: true, pargs: []lx{args[0], args[3]}}
		}
	case "StringBuilder":
		if len(args) == 0 {
			return lx{code: `""`, prec: precAtom, t: tyString, safe: true}
		}
	}
	return t.todo(v.line, "new "+name, "new "+name+"("+strconv.Itoa(len(args))+" arguments)")
}
