---@meta
-- L2Go script API definitions for the Lua Language Server (LuaLS).
--
-- They give autocomplete and type checks in editors. The server never runs
-- this file. A Go test (internal/scripting) fails when a function or method
-- registered in Go is missing here, or one here does not exist in Go.
--
-- Scripts run in a sandbox (Lua 5.1): the base library without dofile,
-- loadfile, load, loadstring, module and collectgarbage, plus string,
-- table and math. There is no os, io, debug or package library.
-- require("lib.name") loads lib/name.lua from the scripts folder, and
-- print() writes to the server log.
--
-- Handlers run on the server's world loop, one at a time, with a time
-- limit (50 ms by default). Player and QuestState values are valid only
-- while the handler that received them runs; within it, one object is
-- always the same value. Npc values are handles: they may be kept in
-- fields across handlers, and stay the same value while the NPC is in the
-- world; npc:valid() tells whether it still is.
--
-- Most NPC events (on_attack, on_kill, on_spawn, on_move_finished,
-- on_event_received, ...) are asynchronous, as in Mobius: one raised while
-- a handler runs (npc:broadcast_event, q:add_spawn) runs after it, in
-- order, with its own time limit. on_npc_hate runs at once.

-------------------------------------------------------------------------------
-- log
-------------------------------------------------------------------------------

---Writes to the server log. Arguments after the message are key, value
---pairs. Every record also gets the script file and line.
---
---```lua
---log.info("character name refused", "name", ev.name, "word", word)
---```
log = {}

---Logs at debug level (shown with `-log-level debug`).
---@param msg string
---@param ... any key, value pairs
function log.debug(msg, ...) end

---Logs at info level.
---@param msg string
---@param ... any key, value pairs
function log.info(msg, ...) end

---Logs at warn level.
---@param msg string
---@param ... any key, value pairs
function log.warn(msg, ...) end

---Logs at error level.
---@param msg string
---@param ... any key, value pairs
function log.error(msg, ...) end

-------------------------------------------------------------------------------
-- events
-------------------------------------------------------------------------------

---Event handlers. Register them while the script loads, at the top level
---of the file. Handlers of one event run in load order (scripts are loaded
---in path order) and share one time limit (50 ms by default).
events = {}

---What events.reject returns. Return it from a handler.
---@class Rejection

---Why character creation was refused. The client shows a fixed message for
---each reason.
---@alias CharacterCreateReason
---| "creation_failed"      # Your character creation has failed.
---| "too_many_characters"  # You cannot create another character.
---| "name_exists"          # This name already exists.
---| "name_too_long"        # The name is longer than 16 characters.
---| "incorrect_name"       # Incorrect name. Please try again.
---| "creation_not_allowed" # Characters cannot be created from this server.

---A character about to be created.
---@class CharacterCreateEvent
---@field account string Account creating the character.
---@field name string Name as the player typed it.
---@field race integer 0 human, 1 elf, 2 dark elf, 3 orc, 4 dwarf.
---@field sex integer 0 male, 1 female.
---@field class_id integer Starting class ID.
---@field hair_style integer Hair style index.
---@field hair_color integer Hair color index.
---@field face integer Face index.

---Runs fn before a new character is accepted. Return
---`events.reject(reason)` to refuse it, or nothing to let it through. The
---first rejection stops the event.
---
---```lua
---events.on_character_create(function(ev)
---  if ev.name:lower():find("admin", 1, true) then
---    return events.reject("incorrect_name")
---  end
---end)
---```
---@param fn fun(ev: CharacterCreateEvent): Rejection?
function events.on_character_create(fn) end

---A chat channel (Mobius ChatType, in lower case).
---@alias ChatType
---| "general"             # Normal chat, heard nearby.
---| "shout"               # "!" in front.
---| "whisper"             # To one player; ev.target is the name.
---| "party"
---| "clan"
---| "gm"
---| "petition_player"
---| "petition_gm"
---| "trade"               # "+" in front.
---| "alliance"
---| "announcement"
---| "boat"
---| "friend"
---| "msnchat"
---| "partymatch_room"
---| "partyroom_commander"
---| "partyroom_all"
---| "hero_voice"
---| "critical_announce"
---| "screen_announce"
---| "battlefield"
---| "mpcc_room"
---| "unknown"             # A type ID the server does not know.

---A chat message about to be sent.
---@class ChatEvent
---@field player Player Who wrote it.
---@field type ChatType The channel.
---@field text string The text, with the replacements of earlier handlers.
---@field target string? Whom a whisper goes to; nil for other channels.

---Runs fn before a chat message is sent, after the server's own checks.
---Return nothing to let it through, a string to replace its text (later
---handlers see the new text), or false (or "") to drop it. The first drop
---stops the event. A failing handler is logged and the message goes on
---with the text so far.
---
---```lua
---events.on_chat(function(ev)
---  if ev.type == "trade" and ev.player:level() < 10 then
---    return false
---  end
---  return (ev.text:gsub("darn", "d**n"))
---end)
---```
---@param fn fun(ev: ChatEvent): string|false|nil
function events.on_chat(fn) end

---Rejects the event being handled. Only inside a handler, and only with a
---reason the event knows; anything else is an error with file and line.
---@param reason CharacterCreateReason
---@return Rejection
function events.reject(reason) end

-------------------------------------------------------------------------------
-- script: NPC scripts
-------------------------------------------------------------------------------

---NPC scripts, following Mobius's Script and Quest classes. A script has a
---name; its pages link to it with `bypass -h Script <name> <event>`.
---
---NPC handlers return, as Mobius's showResult reads it:
---* a page file name ending in `.htm` or `.html`, relative to the folder of
---  the script's file;
---* HTML starting with `<html`, shown as is;
---* other text, shown to the player as a message;
---* `""` to show nothing (the handler did what it needed, e.g. npc:say);
---* `nil` for the default: the NPC's own dialog after a first talk,
---  nothing after a talk or an event.
---A table as a second return value gives values for `%name%` placeholders
---in the page, HTML or text.
---
---```lua
---local s = script.new("NewbieGuide")
---s:on_first_talk({ 30598 }, function(npc, player)
---  if player:level() < 6 then
---    return "welcome.htm", { name = player:name() }
---  end
---end)
---s:on_event(function(event, npc, player)
---  if event == "hello" then
---    npc:say("Hello, " .. player:name() .. "!")
---    return ""
---  end
---end)
---```
script = {}

---What an NPC handler returns; see `script`.
---@alias Reply string|nil

---Placeholder values for a reply: `{ name = "Hero" }` fills `%name%`.
---@alias Placeholders table<string, string|number|boolean>

---A script made by script.new or quest.new (Mobius Script and Quest). Its
---methods are Mobius's in snake case: registration (only while the
---scripts load), handlers, and the quest helpers.
---@class Script
local Script = {}

---Creates a plain script: quest.new(-1, name, ""). Only while the scripts
---load; names are unique across all scripts and may contain letters,
---digits and underscores (1 to 64).
---@param name string as in `Script <name> <event>` links, e.g. "NewbieGuide"
---@return Script
function script.new(name) end

---Returns the script's name.
---@return string
function Script:name() end

---The quest ID; -1 for a plain script.
---@return integer
function Script:id() end

---The quest's name in the quest window ("Letters of Love").
---@return string
function Script:description() end

---Whether set_custom(true) was called.
---@return boolean
function Script:is_custom() end

-- Registration. Each function takes IDs or lists of IDs, as many as you
-- like; IDs given twice count once.

---The NPCs that offer the quest. Their quest links reach on_talk too.
---@param ... integer|integer[]
function Script:add_start_npc(...) end

---The NPCs whose quest link reaches on_talk (Mobius addTalkId).
---@param ... integer|integer[]
function Script:add_talk_id(...) end

---The NPCs whose first click reaches on_first_talk. Only one script may
---own the first talk of an NPC; another script claiming it is a load error.
---@param ... integer|integer[]
function Script:add_first_talk_id(...) end

---The monsters whose death reaches on_kill.
---@param ... integer|integer[]
function Script:add_kill_id(...) end

---The NPCs whose attacks reach on_attack.
---@param ... integer|integer[]
function Script:add_attack_id(...) end

---The NPCs whose spawns (and respawns) reach on_spawn.
---@param ... integer|integer[]
function Script:add_spawn_id(...) end

---The NPCs that see skills (on_skill_see).
---@param ... integer|integer[]
function Script:add_skill_see_id(...) end

---The NPCs whose finished casts reach on_spell_finished.
---@param ... integer|integer[]
function Script:add_spell_finished_id(...) end

---The monsters whose aggro range reaches on_aggro_range_enter.
---@param ... integer|integer[]
function Script:add_aggro_range_enter_id(...) end

---The zones whose entries reach on_enter_zone.
---@param ... integer|integer[]
function Script:add_enter_zone_id(...) end

---The zones whose exits reach on_exit_zone.
---@param ... integer|integer[]
function Script:add_exit_zone_id(...) end

---The items whose quest link reaches on_item_talk.
---@param ... integer|integer[]
function Script:add_item_talk_id(...) end

---The quest's items: taken when the quest ends (exit_quest) or is
---aborted. 0 is ignored.
---@param ... integer|integer[]
function Script:register_quest_items(...) end

---The registered quest items.
---@return integer[]
function Script:get_registered_item_ids() end

---Makes on_enter_world run for every player entering the world.
---@param on? boolean true when left out
function Script:set_on_enter_world(on) end

---A custom quest gets no quest mark, and start_quest does nothing for it.
---@param on? boolean true when left out
function Script:set_custom(on) end

---Features the quest needs that later milestones bring. Until they exist
---the quest is not offered; stored states are kept.
---@param ... "clans"|"clan_halls"|"subclass"|"noblesse"|"pets"|"fishing"|"seven_signs"|"sepulchers"|"grand_bosses"|"doors"
function Script:requires(...) end

-- Handlers. One handler per kind and script. q:on_x(fn) handles every ID
-- registered for x; q:on_x(ids, fn) (the M4 form, kinds with IDs only)
-- registers the IDs and binds fn to them. Arguments come in Mobius's
-- order.

---Answers the quest link at the talk and start NPCs (Mobius onTalk).
---Several scripts may talk at one NPC; the player picks.
---@param npc_ids integer|integer[]|fun(npc: Npc, player: Player): Reply, Placeholders?
---@param fn? fun(npc: Npc, player: Player): Reply, Placeholders?
function Script:on_talk(npc_ids, fn) end

---Answers the first click on the NPC instead of its default dialog (Mobius
---onFirstTalk); nil shows the NPC's own page.
---@param npc_ids integer|integer[]|fun(npc: Npc, player: Player): Reply, Placeholders?
---@param fn? fun(npc: Npc, player: Player): Reply, Placeholders?
function Script:on_first_talk(npc_ids, fn) end

---Handles the `Script <name> <event>` links and the quest timers (Mobius
---onEvent). npc may be nil, and so may player for a timer started without
---one.
---@param fn fun(event: string, npc: Npc?, player: Player?): Reply, Placeholders?
function Script:on_event(fn) end

---A registered monster died; runs 2.5 s after the death, with the acting
---player, unless that player left the world.
---@param npc_ids integer|integer[]|fun(npc: Npc, killer: Player, is_summon: boolean)
---@param fn? fun(npc: Npc, killer: Player, is_summon: boolean)
function Script:on_kill(npc_ids, fn) end

---A registered NPC was hit; skill is nil for a plain hit.
---@param npc_ids integer|integer[]|fun(npc: Npc, attacker: Player, damage: integer, is_summon: boolean, skill: Skill?)
---@param fn? fun(npc: Npc, attacker: Player, damage: integer, is_summon: boolean, skill: Skill?)
function Script:on_attack(npc_ids, fn) end

---A registered NPC spawned or respawned.
---@param npc_ids integer|integer[]|fun(npc: Npc)
---@param fn? fun(npc: Npc)
function Script:on_spawn(npc_ids, fn) end

---A registered NPC saw a skill.
---@param npc_ids integer|integer[]|fun(npc: Npc, caster: Player, skill: Skill, targets: (Player|Npc)[], is_summon: boolean)
---@param fn? fun(npc: Npc, caster: Player, skill: Skill, targets: (Player|Npc)[], is_summon: boolean)
function Script:on_skill_see(npc_ids, fn) end

---A registered NPC finished casting.
---@param npc_ids integer|integer[]|fun(npc: Npc, player: Player, skill: Skill)
---@param fn? fun(npc: Npc, player: Player, skill: Skill)
function Script:on_spell_finished(npc_ids, fn) end

---A player came into a registered monster's aggro range.
---@param npc_ids integer|integer[]|fun(npc: Npc, player: Player, is_summon: boolean)
---@param fn? fun(npc: Npc, player: Player, is_summon: boolean)
function Script:on_aggro_range_enter(npc_ids, fn) end

---A creature entered a registered zone.
---@param zone_ids integer|integer[]|fun(creature: Player|Npc, zone: Zone)
---@param fn? fun(creature: Player|Npc, zone: Zone)
function Script:on_enter_zone(zone_ids, fn) end

---A creature left a registered zone.
---@param zone_ids integer|integer[]|fun(creature: Player|Npc, zone: Zone)
---@param fn? fun(creature: Player|Npc, zone: Zone)
function Script:on_exit_zone(zone_ids, fn) end

---The quest link of a registered item.
---@param item_ids integer|integer[]|fun(item: Item, player: Player): Reply, Placeholders?
---@param fn? fun(item: Item, player: Player): Reply, Placeholders?
function Script:on_item_talk(item_ids, fn) end

---A player whose quest state asked for it died. Without this handler the
---script's on_event("", killer NPC or nil, player) runs, as in Mobius.
---@param fn fun(killer: Player|Npc|nil, victim: Player, st: QuestState?)
function Script:on_death(fn) end

---A player entered the world (with set_on_enter_world(true)).
---@param fn fun(player: Player)
function Script:on_enter_world(fn) end

---get_random_party_member(player, npc) asks this for each candidate
---(Mobius checkPartyMember(Player, Npc)).
---@param fn fun(player: Player, npc: Npc?): boolean
function Script:on_check_party_member(fn) end

---get_random_party_member_state asks this for each candidate (Mobius
---checkPartyMember(QuestState, Npc)).
---@param fn fun(st: QuestState, npc: Npc?): boolean
function Script:on_check_party_member_state(fn) end

---execute_for_each_player calls this for each player (Mobius
---actionForEachPlayer).
---@param fn fun(player: Player, npc: Npc?, is_summon: boolean)
function Script:on_action_for_each_player(fn) end

---What on_start handlers get.
---@class StartEvent
---@field reload boolean true after a reload: the world kept the NPCs the scripts spawned before, but not their timers

---Runs on the world loop once the scripts are in place, at startup and
---after each reload: what Mobius scripts do in their constructors (spawns,
---timers, global variables). Loading itself cannot touch the world. After
---a reload, ev.reload is true: a boss script then finds its live boss
---(grandboss.boss, world.npcs_by_id) and starts its timers again instead
---of spawning a second one.
---@param fn fun(ev: StartEvent)
function Script:on_start(fn) end

-- The AI handlers (Mobius Quest callbacks).

---The monsters whose faction calls reach on_faction_call.
---@param ... integer|integer[]
function Script:add_faction_call_id(...) end

---The NPCs whose sightings reach on_creature_see.
---@param ... integer|integer[]
function Script:add_creature_see_id(...) end

---The NPCs whose script events reach on_event_received.
---@param ... integer|integer[]
function Script:add_event_received_id(...) end

---The NPCs whose arrivals reach on_move_finished.
---@param ... integer|integer[]
function Script:add_move_finished_id(...) end

---The walkers whose route nodes reach on_node_arrived.
---@param ... integer|integer[]
function Script:add_node_arrived_id(...) end

---The walkers whose route ends reach on_route_finished.
---@param ... integer|integer[]
function Script:add_route_finished_id(...) end

---The NPCs whose teleports reach on_teleport.
---@param ... integer|integer[]
function Script:add_teleport_id(...) end

---The monsters whose hate on_npc_hate decides.
---@param ... integer|integer[]
function Script:add_npc_hate_id(...) end

---A monster was called to help caller against attacker (Mobius
---onFactionCall).
---@param npc_ids integer|integer[]|fun(npc: Npc, caller: Npc, attacker: Player, is_summon: boolean)
---@param fn? fun(npc: Npc, caller: Npc, attacker: Player, is_summon: boolean)
function Script:on_faction_call(npc_ids, fn) end

---An NPC saw a player or an NPC come into range (Mobius onCreatureSee).
---@param npc_ids integer|integer[]|fun(npc: Npc, creature: Player|Npc)
---@param fn? fun(npc: Npc, creature: Player|Npc)
function Script:on_creature_see(npc_ids, fn) end

---An NPC got a script event from npc:broadcast_event or npc:send_event
---(Mobius onEventReceived).
---@param npc_ids integer|integer[]|fun(event: string, sender: Npc, receiver: Npc, reference: Player|Npc|nil)
---@param fn? fun(event: string, sender: Npc, receiver: Npc, reference: Player|Npc|nil)
function Script:on_event_received(npc_ids, fn) end

---An NPC arrived where it was sent (Mobius onMoveFinished).
---@param npc_ids integer|integer[]|fun(npc: Npc)
---@param fn? fun(npc: Npc)
function Script:on_move_finished(npc_ids, fn) end

---A walker reached a node of its route (Mobius onNodeArrived).
---@param npc_ids integer|integer[]|fun(npc: Npc)
---@param fn? fun(npc: Npc)
function Script:on_node_arrived(npc_ids, fn) end

---A walker reached the end of its route (Mobius onRouteFinished).
---@param npc_ids integer|integer[]|fun(npc: Npc)
---@param fn? fun(npc: Npc)
function Script:on_route_finished(npc_ids, fn) end

---An NPC was teleported (Mobius onTeleport).
---@param npc_ids integer|integer[]|fun(npc: Npc)
---@param fn? fun(npc: Npc)
function Script:on_teleport(npc_ids, fn) end

---Decides at once whether a monster may hate a player: return false to
---veto (Mobius onNpcHate). Anything else allows.
---@param npc_ids integer|integer[]|fun(npc: Npc, player: Player, is_summon: boolean): boolean?
---@param fn? fun(npc: Npc, player: Player, is_summon: boolean): boolean?
function Script:on_npc_hate(npc_ids, fn) end

---Game night began (true, 00:00) or ended (false, 06:00) (Mobius
---OnDayNightChange, a global event: every script with this handler gets
---it).
---@param fn fun(is_night: boolean)
function Script:on_day_night_change(fn) end

-- Quest helpers (Mobius Quest methods). They need the world, so they work
-- in handlers, not while the scripts load.

---The player's state of this quest; with create, a new CREATED state when
---there is none (Mobius getQuestState).
---@param player Player
---@param create? boolean
---@return QuestState?
function Script:get_quest_state(player, create) end

---A new CREATED state of this quest for the player.
---@param player Player
---@return QuestState
function Script:new_quest_state(player) end

---The quest state of the player's clan leader (the player's own when it
---leads), when the leader is in the game, alive and within the party range
---of npc; else nil (Mobius getClanLeaderQuestState).
---@param player Player
---@param npc Npc?
---@return QuestState?
function Script:get_clan_leader_quest_state(player, npc) end

---When the player leads its clan, the other members in the game get this
---quest started with the leader's cond (Mobius setQuestToClanMembers).
---@param player Player
function Script:set_quest_to_clan_members(player) end

---When the player leads its clan, the other members in the game lose their
---state of this quest (Mobius finishQuestToClan).
---@param player Player
function Script:finish_quest_to_clan(player) end

---The "no quest" page (data/html/noquest.htm, or Mobius's text).
---@param player Player?
---@return string html
function Script:get_no_quest_msg(player) end

---The "already completed" page (data/html/alreadycompleted.htm, or
---Mobius's text).
---@param player Player?
---@return string html
function Script:get_already_completed_msg(player) end

---A page of this script as HTML: next to the script, then in
---data/html/scripts/<the script's folder>/. nil if there is none.
---@param player Player?
---@param file string
---@return string?
function Script:get_htm(player, file) end

---Shows a page of this script to the player and returns its HTML (nil if
---there is no such page).
---@param player Player
---@param file string
---@param npc? Npc
---@return string?
function Script:show_html_file(player, file, npc) end

---Gives items with the "You have earned" messages (Mobius giveItems): the
---quest item drop multiplier for quest items, enchant for all but adena.
---Counts below 1 and unknown items are ignored. The item may also be given
---as one table {id =, count =} (Mobius's ItemHolder), the enchant after it.
---@param player Player
---@param item_id integer|{id: integer, count: integer}
---@param count? integer
---@param enchant? integer
function Script:give_items(player, item_id, count, enchant) end

---Gives items with the quest reward rates (Mobius rewardItems). The item
---may also be given as one table {id =, count =}.
---@param player Player
---@param item_id integer|{id: integer, count: integer}
---@param count? integer
function Script:reward_items(player, item_id, count) end

---Gives adena, with the quest reward rate when apply_rates is set.
---@param player Player
---@param count integer
---@param apply_rates boolean
function Script:give_adena(player, count, apply_rates) end

---Takes count items (-1: all of them); a worn item comes off first.
---Returns true, as Mobius does. The item may also be given as one table
---{id =, count =} (Mobius takeItem(player, ItemHolder)).
---@param player Player
---@param item_id integer|{id: integer, count: integer}
---@param count? integer
---@return boolean
function Script:take_items(player, item_id, count) end

---Mobius takeItems(player, amount, ids...): count of each item.
---@param player Player
---@param count integer
---@param ... integer|integer[] item IDs
---@return boolean
function Script:take_items_ids(player, count, ...) end

---The summed count of the items (Mobius getQuestItemsCount).
---@param player Player
---@param ... integer|integer[] item IDs
---@return integer
function Script:get_quest_items_count(player, ...) end

---Whether the player has all of the items; false for none given.
---@param player Player
---@param ... integer|integer[] item IDs
---@return boolean
function Script:has_quest_items(player, ...) end

---Whether the player has at least one of the items.
---@param player Player
---@param ... integer|integer[] item IDs
---@return boolean
function Script:has_at_least_one_quest_item(player, ...) end

---Mobius giveItemRandomly: with chance (0 to 1) gives min..max items (the
---quest item drop multiplier applied), never past limit (0: no limit).
---With play_sound: the item sound, or the middle sound when the limit is
---reached. Returns true when the limit is reached, or, without a limit,
---when items were given.
---@param player Player
---@param npc Npc?
---@param item_id integer
---@param min integer
---@param max integer
---@param limit integer
---@param chance number
---@param play_sound boolean
---@return boolean
function Script:give_item_randomly(player, npc, item_id, min, max, limit, chance, play_sound) end

---Adds exp and sp with the quest rates.
---@param player Player
---@param exp integer
---@param sp integer
function Script:add_exp_and_sp(player, exp, sp) end

---Plays a sound to the player: a QuestSound value or any sound name.
---@param player Player
---@param sound string
function Script:play_sound(player, sound) end

---The item ID in a paperdoll slot, 0 if empty.
---@param player Player
---@param slot integer a Paperdoll value
---@return integer
function Script:get_item_equipped(player, slot) end

---The enchant level of the player's item, 0 if none.
---@param player Player
---@param item_id integer
---@return integer
function Script:get_enchant_level(player, item_id) end

---Starts a timer that runs on_event(name, npc, player) after ms (every ms
---with repeating). A timer with the same name, NPC and player already
---running is left alone; nil matches only nil. Timers stop when their NPC
---or player leaves the world and on reload, and are not saved.
---@param name string
---@param ms integer
---@param npc Npc?
---@param player Player?
---@param repeating? boolean
function Script:start_quest_timer(name, ms, npc, player, repeating) end

---The running timer with this name, NPC and player, or nil.
---@param name string
---@param npc Npc?
---@param player Player?
---@return QuestTimer?
function Script:get_quest_timer(name, npc, player) end

---Cancels the timer with this name, NPC and player.
---@param name string
---@param npc Npc?
---@param player Player?
function Script:cancel_quest_timer(name, npc, player) end

---Cancels every timer of this script with the name.
---@param name string
function Script:cancel_quest_timers(name) end

---Mobius getRandomPartyMember: (player) any member; (player, cond) or
---(player, var, value) a member whose state matches; (player, npc) a
---member near npc that on_check_party_member accepts (the highest of 1000
---rolls). Without a party, the player or nil.
---@param player Player?
---@param cond_var_or_npc? integer|string|Npc
---@param value? string
---@return Player?
function Script:get_random_party_member(player, cond_var_or_npc, value) end

---Mobius getRandomPartyMemberState: (player, state) → a member with the
---state; (player, cond, chance, npc) → the state of a member at cond (-1:
---started) near npc that on_check_party_member_state accepts, the player
---counting chance times. Without a party, the player's.
---@param player Player?
---@param state_or_cond integer
---@param chance? integer
---@param npc? Npc
---@return Player|QuestState|nil
function Script:get_random_party_member_state(player, state_or_cond, chance, npc) end

---Runs on_action_for_each_player for the player, or with include_party for
---each member of its party (command channels come with M11).
---@param player Player
---@param npc Npc?
---@param is_summon boolean
---@param include_party boolean
---@param include_command_channel boolean
function Script:execute_for_each_player(player, npc, is_summon, include_party, include_command_channel) end

---Spawns an NPC (Mobius addSpawn) and returns it. A heading of -1 is a
---random heading.
---@param npc_id integer
---@param pos {x: integer, y: integer, z: integer, heading?: integer}|Player|Npc
---@param random_offset? boolean
---@param despawn_ms? integer 0: stays
---@param summon? boolean
---@return Npc?
function Script:add_spawn(npc_id, pos, random_offset, despawn_ms, summon) end

---Adds a radar mark.
---@param player Player
---@param x integer
---@param y integer
---@param z integer
function Script:add_radar(player, x, y, z) end

---Removes a radar mark.
---@param player Player
---@param x integer
---@param y integer
---@param z integer
function Script:remove_radar(player, x, y, z) end

---Removes every radar mark.
---@param player Player
function Script:clear_radar(player) end

---Shows text on the player's screen for ms.
---@param player Player
---@param text string
---@param ms integer
function Script:show_on_screen_msg(player, text, ms) end

-------------------------------------------------------------------------------
-- quest: quest scripts
-------------------------------------------------------------------------------

---Quest scripts (Mobius Quest). A quest is a Script with an ID.
---
---```lua
---local q = quest.new(1, "Q00001_LettersOfLove", "Letters of Love")
---q:add_start_npc(30048)
---q:on_talk(function(npc, player)
---  local st = q:get_quest_state(player, true)
---  if st:is_created() then return "30048-02.htm" end
---  return q:get_no_quest_msg(player)
---end)
---```
---@class questlib
---@field DF_REWARD_35 table<integer, integer> Dimensional Diamonds by class (2nd class change).
---@field DF_REWARD_37 table<integer, integer> Dimensional Diamonds by race.
---@field DF_REWARD_39 table<integer, integer> Dimensional Diamonds by class.
quest = {}

---Creates a quest. Only while the scripts load; names are unique across
---all scripts, IDs across all quests.
---@param id integer the quest ID (1 or more), or -1 for a plain script
---@param name string as in `Script <name> <event>` links, e.g. "Q00001_LettersOfLove"
---@param description string the name in the quest window
---@return Script
function quest.new(id, name, description) end

-------------------------------------------------------------------------------
-- QuestState
-------------------------------------------------------------------------------

---One player's state of one quest (Mobius QuestState), saved by the
---server. Valid only while the handler that got it runs. Variables are
---strings (names up to 20 characters, values up to 255).
---@class QuestState
local QuestState = {}

---@return string
function QuestState:get_quest_name() end

---@return Player
function QuestState:get_player() end

---State.CREATED, STARTED or COMPLETED.
---@return integer
function QuestState:get_state() end

---@return boolean
function QuestState:is_created() end

---@return boolean
function QuestState:is_started() end

---@return boolean
function QuestState:is_completed() end

---@param state integer a State value
function QuestState:set_state(state) end

---From CREATED (and not for a custom quest): cond 1, STARTED, the accept
---sound.
function QuestState:start_quest() end

---The cond; 0 unless the quest is started.
---@return integer
function QuestState:get_cond() end

---Whether the cond is n, whatever the state (Mobius quirk).
---@param n integer
---@return boolean
function QuestState:is_cond(n) end

---Sets the cond (only while started), with the middle sound when asked.
---Updates the quest log and the quest mark.
---@param n integer
---@param play_middle? boolean
function QuestState:set_cond(n, play_middle) end

---Sets a variable; numbers are stored as Java writes them.
---@param name string
---@param value string|number|nil
function QuestState:set(name, value) end

---@param name string
---@return string?
function QuestState:get(name) end

---The variable as an integer; 0 when unset or not a number.
---@param name string
---@return integer
function QuestState:get_int(name) end

---@param name string
function QuestState:unset(name) end

---@param name string
---@return boolean
function QuestState:is_set(name) end

---The memo state; 0 unless started.
---@return integer
function QuestState:get_memo_state() end

---@param value integer
function QuestState:set_memo_state(value) end

---@param value integer
---@return boolean
function QuestState:is_memo_state(value) end

---@param slot integer
---@return integer
function QuestState:get_memo_state_ex(slot) end

---@param slot integer
---@param value integer
function QuestState:set_memo_state_ex(slot, value) end

---@param slot integer
---@param value integer
---@return boolean
function QuestState:is_memo_state_ex(slot, value) end

---Ends a started quest: its quest items are taken; repeatable (true or
---QuestType.REPEATABLE) deletes the state, otherwise it becomes COMPLETED
---(QuestType.DAILY also sets the restart time). With play_finish, the
---finish sound.
---@param repeatable boolean|integer true, false or a QuestType value
---@param play_finish? boolean
function QuestState:exit_quest(repeatable, play_finish) end

---Whether a daily quest can be taken again.
---@return boolean
function QuestState:is_now_available() end

---Sets the restart time to the next 06:30 server time.
function QuestState:set_restart_time() end

---Shows a tutorial question mark.
---@param n integer
function QuestState:show_question_mark(n) end

---Plays a tutorial voice.
---@param voice string
function QuestState:play_tutorial_voice(voice) end

---Shows a page of Q00255_Tutorial in the tutorial window.
---@param file string
function QuestState:show_tutorial_html(file) end

---Closes the tutorial window.
function QuestState:close_tutorial_html() end

---Enables a tutorial client event.
---@param n integer
function QuestState:on_tutorial_client_event(n) end

---A running quest timer.
---@class QuestTimer
local QuestTimer = {}

---@return string
function QuestTimer:name() end

---@return boolean
function QuestTimer:is_repeating() end

---Whether the timer still runs.
---@return boolean
function QuestTimer:is_active() end

---Stops the timer.
function QuestTimer:cancel() end

---An item, as on_item_talk gets it.
---@class Item
local Item = {}

---The item template ID.
---@return integer
function Item:id() end

---@return integer
function Item:object_id() end

---@return integer
function Item:count() end

---A zone, as the zone handlers get it.
---@class Zone
local Zone = {}

---@return integer
function Zone:id() end

---The zone type, e.g. "peace".
---@return string
function Zone:type() end

---A skill, as the attack and skill handlers get it and skills.get returns
---it.
---@class Skill
local Skill = {}

---@return integer
function Skill:id() end

---@return integer
function Skill:level() end

---The skill's target type, Mobius's TargetType name, e.g. "ONE" or
---"UNLOCKABLE".
---@return string
function Skill:target_type() end

---The skill's effect point (Mobius getEffectPoint): below 0 for harmful
---skills, above 0 for helpful ones.
---@return integer
function Skill:effect_point() end

---The skill's cast range (Mobius getCastRange); -1 for none.
---@return integer
function Skill:cast_range() end

---Whether the skill is magic (Mobius isMagic).
---@return boolean
function Skill:is_magic() end

---Whether an effect of the skill is of the type, a Mobius EffectType name
---such as "AGGRESSION" (Mobius hasEffectType).
---@param effect_type string
---@return boolean
function Skill:has_effect_type(effect_type) end

-------------------------------------------------------------------------------
-- Player and Npc
-------------------------------------------------------------------------------

---A player, as handlers receive it. Valid only while that handler runs:
---keep player:object_id() instead of the player.
---@class Player
local Player = {}

---@return integer
function Player:object_id() end

---@return string
function Player:name() end

---@return integer
function Player:level() end

---The current class ID.
---@return integer
function Player:class_id() end

---0 human, 1 elf, 2 dark elf, 3 orc, 4 dwarf.
---@return integer
function Player:race() end

---The adena the player carries.
---@return integer
function Player:adena() end

---How many items of the template the player carries, worn ones included:
---the stack of a stackable item, else the number of items.
---@param item_id integer
---@return integer
function Player:item_count(item_id) end

---Gives the player items, with the "You have earned ..." message.
---@param item_id integer
---@param count? integer At least 1; 1 when left out.
---@return boolean given
function Player:give_items(item_id, count) end

---Takes items from the player; worn ones come off first. With fewer than
---count it takes what there is and returns false.
---@param item_id integer
---@param count? integer -1 (the default) takes all of them.
---@return boolean had_enough
function Player:take_items(item_id, count) end

---The player's state of a quest, by the quest's name.
---@param quest string e.g. "Q00001_LettersOfLove"
---@return QuestState?
function Player:get_quest_state(quest) end

---0 male, 1 female.
---@return integer
function Player:sex() end

---The base class ID.
---@return integer
function Player:base_class() end

---Changes the class (Mobius setPlayerClass): the class change effect, the
---system message, the new skills and the skill list.
---@param id integer a ClassId value
function Player:set_class_id(id) end

---Sets the base class (Mobius setBaseClass).
---@param id integer a ClassId value
function Player:set_base_class(id) end

---Sends UserInfo to the player and CharInfo to the players around it.
function Player:broadcast_user_info() end

---@return boolean
function Player:is_mage_class() end

---@return boolean
function Player:is_newbie() end

---@return boolean
function Player:is_dead() end

---@param category integer a Category value
---@return boolean
function Player:is_in_category(category) end

---A player variable (saved with the character), or default.
---@param name string
---@param default? string
---@return string?
function Player:var(name, default) end

---A player variable as an integer, or default (0).
---@param name string
---@param default? integer
---@return integer
function Player:var_int(name, default) end

---A player variable as a boolean ("true" in any case), or default.
---@param name string
---@param default? boolean
---@return boolean
function Player:var_bool(name, default) end

---Sets a player variable.
---@param name string
---@param value string|number|boolean
function Player:set_var(name, value) end

---Whether the player is in a party.
---@return boolean
function Player:is_in_party() end

---The player's party, nil without one: the leader, the members in party
---order (leader first) and the loot mode (0 finders keepers, 1 random,
---2 random including spoil, 3 by turn, 4 by turn including spoil).
---@return {leader: Player, members: Player[], loot: integer}|nil
function Player:party() end

---The party's members, leader first; without a party, the player alone.
---@return Player[]
function Player:party_members() end

---A clan member's leader: its object ID, its name, and its player while it
---is in the game (nil offline).
---@class ClanLeader
---@field object_id fun(self: ClanLeader): integer
---@field name fun(self: ClanLeader): string
---@field player fun(self: ClanLeader): Player|nil

---A clan, called with ":": id, name, level, reputation, leader_id,
---leader(), add_reputation(points) (negative takes; with Mobius's messages
---and the clan window update) and broadcast_pledge_info_update().
---@class Clan
---@field id fun(self: Clan): integer
---@field name fun(self: Clan): string
---@field level fun(self: Clan): integer
---@field reputation fun(self: Clan): integer
---@field leader_id fun(self: Clan): integer
---@field leader fun(self: Clan): ClanLeader
---@field add_reputation fun(self: Clan, points: integer)
---@field broadcast_pledge_info_update fun(self: Clan)

---The player's clan, nil without one.
---@return Clan|nil
function Player:clan() end

---Whether the player leads its clan.
---@return boolean
function Player:is_clan_leader() end

---Whether the player is in the game (Mobius isOnline).
---@return boolean
function Player:is_online() end

---Opens the symbol maker's "draw a symbol" window (Mobius
---player.sendPacket(new HennaEquipList(player))).
function Player:show_henna_equip_list() end

---Opens the symbol maker's "delete a symbol" window (Mobius
---player.sendPacket(new HennaRemoveList(player))).
function Player:show_henna_remove_list() end

---Whether the player holds a clan privilege, by Mobius's ClanAccess name
---("CASTLE_VAULT", "CASTLE_OPEN_DOOR", ...; Mobius hasAccess). The clan
---leader holds all of them; a player without a clan none. An unknown name
---is an error.
---@param name string
---@return boolean
function Player:has_clan_privilege(name) end

---The player's clan ID, 0 without a clan (Mobius getClanId).
---@return integer
function Player:clan_id() end

---true when the player has an effect of the abnormal type (an AbnormalType
---value), else nil (Mobius getEffectList().getBuffInfoByAbnormalType).
---@param abnormal_type string
---@return true|nil
function Player:buff_info_by_abnormal_type(abnormal_type) end

---Whether the player plays a subclass (Mobius isSubClassActive).
---@return boolean
function Player:is_sub_class_active() end

---The active class index: 0 for the base class, 1-3 for a subclass (Mobius
---getClassIndex).
---@return integer
function Player:class_index() end

---How many subclasses the player has (Mobius getTotalSubClasses).
---@return integer
function Player:total_sub_classes() end

---Whether the player is a noble.
---@return boolean
function Player:is_noble() end

---Makes the player a noble or not (Mobius setNoble): the flag and the
---noble skills, then the skill list.
---@param noble boolean
function Player:set_noble(noble) end

---Whether a skill's effects are on the player (Mobius isAffectedBySkill).
---@param skill integer|Skill a skill ID or a skill
---@return boolean
function Player:is_affected_by_skill(skill) end

---The item ID of the weapon in the right hand, nil without one (Mobius
---getActiveWeaponItem().getId(); Mobius gives a player without a weapon its
---fists, which this leaves out).
---@return integer?
function Player:active_weapon() end

---The player's target, nil without one (Mobius getTarget).
---@return Player|Npc|nil
function Player:target() end

---Shows a skill animation of the player on target to the players around,
---the player included (Mobius broadcastPacket(new MagicSkillUse(player,
---target, skill_id, level, hit_time, reuse_delay))). It only shows; nothing
---is cast.
---@param target Player|Npc
---@param skill_id integer
---@param level integer
---@param hit_time? integer milliseconds, 0 when left out
---@param reuse_delay? integer milliseconds, 0 when left out
function Player:magic_skill_use(target, skill_id, level, hit_time, reuse_delay) end

---The player's PK count (Mobius getPkKills).
---@return integer
function Player:pk_kills() end

---Sets the PK count (Mobius setPkKills); player:update_user_info() shows
---it.
---@param count integer
function Player:set_pk_kills(count) end

---Sends the player its UserInfo (Mobius updateUserInfo).
function Player:update_user_info() end

---The IDs of the player's started quests (Mobius getAllActiveQuests).
---@return integer[]
function Player:active_quests() end

---Shows a social action to the players around.
---@param action integer
function Player:social_action(action) end

---Teleports the player.
---@param x integer|{x: integer, y: integer, z: integer}|Player|Npc
---@param y? integer
---@param z? integer
function Player:teleport(x, y, z) end

---Sends a text message.
---@param text string
function Player:send_message(text) end

---Sends a system message with its parameters.
---@param id integer a SystemMessageId value
---@param ... string|number
function Player:send_system_message(id, ...) end

---@return integer
function Player:x() end

---@return integer
function Player:y() end

---@return integer
function Player:z() end

---@return integer
function Player:heading() end

---Whether other is within radius, on the ground.
---@param other Player|Npc
---@param radius number
---@return boolean
function Player:is_inside_radius_2d(other, radius) end

---Whether other is within radius, height included.
---@param other Player|Npc
---@param radius number
---@return boolean
function Player:is_inside_radius_3d(other, radius) end

---The current HP (Mobius getCurrentHp).
---@return number
function Player:hp() end

---The maximum HP (Mobius getMaxHp).
---@return number
function Player:max_hp() end

---The player's summon or pet, nil for none (Mobius getSummon).
---@return Player|Npc|nil
function Player:summon() end

---Whether the player is inside the zone.
---@param zone_id integer
---@return boolean
function Player:is_inside_zone(zone_id) end

---The distance to a position, a player or an NPC, without height.
---@param other Player|Npc|Position
---@return number
function Player:calculate_distance_2d(other) end

---The distance to a position, a player or an NPC, with height.
---@param other Player|Npc|Position
---@return number
function Player:calculate_distance_3d(other) end

---The level of the skill's effect on the player, nil when it has none
---(Mobius getEffectList().getBuffInfoBySkillId(id).getSkill().getLevel()).
---@param skill integer|Skill a skill ID or a skill
---@return integer?
function Player:effect_level(skill) end

---The weight penalty, 0 to 4 (Mobius getWeightPenalty).
---@return integer
function Player:weight_penalty() end

---Whether the player is a GM (Mobius isGM).
---@return boolean
function Player:is_gm() end

---Whether the player is invisible (Mobius isInvisible, a GM's //invis).
---@return boolean
function Player:is_invisible() end

---Whether the player is a hero (Mobius isHero).
---@return boolean
function Player:is_hero() end

---Kills the player (Mobius doDie(killer)).
---@param killer? Player|Npc the player itself when left out
function Player:die(killer) end

---Teleports the player to the nearest town (Mobius
---teleToLocation(TeleportWhereType.TOWN)).
function Player:teleport_to_town() end

---Whether the player holds a cursed weapon (Mobius
---isCursedWeaponEquipped): Demonic Sword Zariche or Blood Sword Akamanah.
---@return boolean
function Player:is_cursed_weapon_equipped() end

---true: tells a player from an NPC where a handler gets either (Mobius
---isPlayer).
---@return boolean
function Player:is_player() end

---Whether the player is in its attack stance (Mobius isInCombat).
---@return boolean
function Player:is_in_combat() end

---Whether the player rides a strider or a wyvern (Mobius isMounted).
---@return boolean
function Player:is_mounted() end

---What the player rides (Mobius getMountType().ordinal()): 0 nothing, 1 a
---strider, 2 a wyvern.
---@return integer
function Player:mount_type() end

---The NPC ID of what the player rides, 0 on foot (Mobius getMountNpcId).
---@return integer
function Player:mount_npc_id() end

---The level of what the player rides, 0 on foot (Mobius getMountLevel).
---@return integer
function Player:mount_level() end

---Gets the player off its mount (Mobius dismount): false where it may not
---(over a no-landing zone, too high).
---@return boolean
function Player:dismount() end

---Mounts the player on an NPC at the player's level (Mobius mount(npcId,
---controlItemObjId, useFood)): the weapon and shield come off; with use_food
---the mount eats. false when the weapon cannot come off.
---@param npc_id integer
---@param control_item? integer the collar's object ID, 0 for none
---@param use_food? boolean
---@return boolean
function Player:mount(npc_id, control_item, use_food) end

---An NPC handle, as handlers receive it. It may be kept across handlers:
---while the NPC is in the world it is the same value. Once the NPC is gone
---(dead and decayed, deleted) every method but valid() raises an error.
---@class Npc
local Npc = {}

---A position: a table with x, y, z and optionally heading.
---@class Position
---@field x integer
---@field y integer
---@field z integer
---@field heading? integer

---Whether the NPC is still in the world (a dead NPC is, until it decays).
---@return boolean
function Npc:valid() end

---@return integer
function Npc:object_id() end

---The NPC template ID, e.g. 30598 for a Newbie Guide.
---@return integer
function Npc:id() end

---@return string
function Npc:name() end

---Says text in normal NPC chat to the players around the NPC.
---@param text string
function Npc:say(text) end

---Says text in a chat channel (Mobius broadcastSay).
---@param chat_type integer a ChatType value
---@param text string
function Npc:broadcast_say(chat_type, text) end

---Sends text in a chat channel to every player in the NPC's map region
---(Mobius: NpcSay to the players whose getMapRegionLocId is the NPC's).
---@param chat_type integer a ChatType value
---@param text string
function Npc:broadcast_say_to_region(chat_type, text) end

---A rented castle function (Mobius Castle.CastleFunction): its level, fee,
---the ms between two payments, and the end of the paid time (ms, and its
---day, month 1-12 and year in server time).
---@class CastleFunction
---@field level integer
---@field lease integer
---@field rate integer
---@field end_time integer
---@field end_day integer
---@field end_month integer
---@field end_year integer

---A castle (Mobius Castle and its Siege), called with ":". It is valid while
---the handler runs.
---
---* `id()`, `name()`, `owner_id()` (0 without an owner), `owner()` (a Clan or nil)
---* `tax_percent()`, `set_tax_percent(n)` (saved; 0-100), `max_tax()` (the
---  chamberlain's limit for the Seal of Strife's owner: 15, Dawn 25, Dusk 5)
---* `treasury()`, `deposit(player, n)` and `withdraw(player, n)`: adena
---  between the player and the treasury, saved together; false when nothing
---  moved (too little adena, or no owner)
---* `siege_date()` (seconds since the epoch, 0 for none),
---  `is_siege_in_progress()`, `is_registration_over()`, `is_zone_active()`,
---  `control_tower_count()`, `show_siege_info(player)` (the SiegeInfo window)
---* `get_function(type)`: a CastleFunction or nil (1 teleport, 2 HP, 3 MP,
---  4 exp, 5 support)
---* `seal_owner(seal)` (1 avarice, 2 gnosis, 3 strife; 0 none, 1 Dusk, 2
---  Dawn) and `seven_signs_period()` (0 recruiting, 1 competition, 2
---  results, 3 seal validation): Mobius SevenSigns, for the castle NPCs
---* `open_close_door(player, door_id, open)` (only for the owner clan),
---  `banish_foreigners()`, `oust_all_players()`
---* `door_upgrade_level(door_id)` (the castle door's HP ratio, 1 without an
---  upgrade; nil for another door) and `set_door_upgrade(door_id, level)`
---  (Mobius Castle.setDoorUpgrade: the ratio and full HP, saved)
---@class Castle
---@field id fun(self: Castle): integer
---@field name fun(self: Castle): string
---@field owner_id fun(self: Castle): integer
---@field owner fun(self: Castle): Clan|nil
---@field tax_percent fun(self: Castle): integer
---@field set_tax_percent fun(self: Castle, percent: integer)
---@field max_tax fun(self: Castle): integer
---@field treasury fun(self: Castle): integer
---@field deposit fun(self: Castle, player: Player, amount: integer): boolean
---@field withdraw fun(self: Castle, player: Player, amount: integer): boolean
---@field siege_date fun(self: Castle): integer
---@field is_siege_in_progress fun(self: Castle): boolean
---@field is_registration_over fun(self: Castle): boolean
---@field is_zone_active fun(self: Castle): boolean
---@field control_tower_count fun(self: Castle): integer
---@field get_function fun(self: Castle, type: integer): CastleFunction|nil
---@field seal_owner fun(self: Castle, seal: integer): integer
---@field seven_signs_period fun(self: Castle): integer
---@field open_close_door fun(self: Castle, player: Player, door_id: integer, open: boolean)
---@field door_upgrade_level fun(self: Castle, door_id: integer): integer|nil
---@field set_door_upgrade fun(self: Castle, door_id: integer, level: integer)
---@field banish_foreigners fun(self: Castle)
---@field oust_all_players fun(self: Castle)
---@field show_siege_info fun(self: Castle, player: Player)

---The castle the NPC belongs to (Mobius Npc.getCastle: its town's castle,
---else the nearest), nil without castles.
---@return Castle|nil
function Npc:castle() end

---Whether the player leads the clan that owns the NPC's castle (Mobius
---Npc.isMyLord).
---@param player Player
---@return boolean
function Npc:is_my_lord(player) end

---Opens a buylist of this merchant NPC (Mobius Merchant.showBuyWindow);
---apply_tax (default true) only changes the tax the window shows. Nothing
---happens for an NPC that is not a merchant or a list that does not name it.
---@param player Player
---@param list_id integer
---@param apply_tax? boolean
function Npc:show_buy_window(player, list_id, apply_tax) end

---@return boolean
function Npc:is_dead() end

---Removes the NPC from the world (Mobius deleteMe).
function Npc:delete() end

---Shows the player the NPC's own dialog.
---@param player Player
function Npc:show_chat_window(player) end

---Shows a social action to the players around.
---@param action integer
function Npc:social_action(action) end

---The NPC's script value (Mobius getScriptValue).
---@return integer
function Npc:get_script_value() end

---@param value integer
function Npc:set_script_value(value) end

---@param value integer
---@return boolean
function Npc:is_script_value(value) end

---Whether the NPC is of a Mobius NPC class, e.g. "Monster" or
---"VillageMasterPriest".
---@param class string
---@return boolean
function Npc:is_type(class) end

---@return integer
function Npc:x() end

---@return integer
function Npc:y() end

---@return integer
function Npc:z() end

---@return integer
function Npc:heading() end

---@param other Player|Npc
---@param radius number
---@return boolean
function Npc:is_inside_radius_2d(other, radius) end

---@param other Player|Npc
---@param radius number
---@return boolean
function Npc:is_inside_radius_3d(other, radius) end

---The NPC's level (Mobius getLevel).
---@return integer
function Npc:level() end

---The current HP (Mobius getCurrentHp).
---@return number
function Npc:hp() end

---The maximum HP (Mobius getMaxHp).
---@return number
function Npc:max_hp() end

---Makes a monster attack target (Mobius
---getAI().setIntention(ATTACK, target)). NPCs that cannot fight ignore it.
---@param target Player|Npc
function Npc:attack(target) end

---Adds damage and hate for target to a monster's hate list (Mobius
---Attackable.addDamageHate). NPCs that cannot fight ignore it.
---@param target Player|Npc
---@param damage integer
---@param hate integer
function Npc:add_damage_hate(target, damage, hate) end

---Takes target off the NPC's attack-by list (Mobius
---getAttackByList().remove(target)).
---@param target Player|Npc
function Npc:remove_attacker(target) end

---Stops the NPC's swing (Mobius abortAttack).
function Npc:abort_attack() end

---Makes the NPC run instead of walk (Mobius setRunning).
function Npc:set_running() end

---Walks the NPC to a place (Mobius getAI().setIntention(MOVE_TO, loc)).
---@param x integer|{x: integer, y: integer, z: integer}|Player|Npc
---@param y? integer
---@param z? integer
function Npc:move_to(x, y, z) end

---Kills the NPC (Mobius doDie(killer)).
---@param killer? Player|Npc the NPC itself when left out
function Npc:die(killer) end

---Stops the NPC's spawn from bringing it back (Mobius
---getSpawn().stopRespawn()).
function Npc:stop_respawn() end

---Makes the NPC's spawn bring it back once it is gone (Mobius
---getSpawn().startRespawn()); an NPC from q:add_spawn comes back at once,
---where it was spawned.
function Npc:start_respawn() end

---Casts a skill (Mobius doCast(skill)) at target, or at the NPC's current
---target. A nil skill does nothing.
---@param skill Skill? from skills.get
---@param target? Player|Npc
function Npc:cast(skill, target) end

---Shows a skill animation of the NPC on target to the players around
---(Mobius broadcastPacket(new MagicSkillUse(npc, target, skill_id, level,
---hit_time, reuse_delay))). It only shows; nothing is cast.
---@param target Player|Npc
---@param skill_id integer
---@param level integer
---@param hit_time? integer milliseconds, 0 when left out
---@param reuse_delay? integer milliseconds, 0 when left out
function Npc:magic_skill_use(target, skill_id, level, hit_time, reuse_delay) end

---Sets the NPC's target, which npc:cast without a target casts at (Mobius
---setTarget).
---@param target Player|Npc|nil
function Npc:set_target(target) end

---Teleports the NPC (Mobius teleToLocation).
---@param x integer|{x: integer, y: integer, z: integer}|Player|Npc
---@param y? integer
---@param z? integer
function Npc:teleport(x, y, z) end

---With false, the monster no longer walks back to its spawn point when it
---has nothing to fight, for as long as it lives, respawns included (Mobius
---Attackable.setCanReturnToSpawnPoint).
---@param can boolean
function Npc:set_can_return_to_spawn_point(can) end

---Drops items on the ground around the NPC, kept for player for 15 s
---(Mobius dropItem(player, item_id, count)).
---@param player Player
---@param item_id integer
---@param count? integer 1 when left out
function Npc:drop_item(player, item_id, count) end

-- NPC variables (Mobius NpcVariables). Values are strings; they last
-- until the NPC is gone (a respawn starts empty) and survive reloads.

---A variable, or default when it is not set.
---@param name string
---@param default? string
---@return string?
function Npc:var(name, default) end

---A variable as an integer, or default (0) when it is not set.
---@param name string
---@param default? integer
---@return integer
function Npc:var_int(name, default) end

---A variable as a boolean ("true"), or default when it is not set.
---@param name string
---@param default? boolean
---@return boolean
function Npc:var_bool(name, default) end

---@param name string
---@param value string|number|boolean
function Npc:set_var(name, value) end

---@param name string
function Npc:unset_var(name) end

---@param name string
---@return boolean
function Npc:has_var(name) end

-- The NPC's AI (Mobius Npc, Attackable, Creature and their AI).

---Makes the NPC walk instead of run (Mobius setWalking).
function Npc:set_walking() end

---@return boolean
function Npc:is_running() end

---Stops (true) or restarts (false) the NPC's own AI; scripts move it
---meanwhile (Mobius disableCoreAI).
---@param disabled? boolean true when left out
function Npc:disable_core_ai(disabled) end

---@return boolean
function Npc:is_core_ai_disabled() end

---@param immobilized? boolean true when left out
function Npc:set_immobilized(immobilized) end

---@param invul? boolean true when left out
function Npc:set_invul(invul) end

---@return boolean
function Npc:is_invul() end

---An immortal NPC keeps 1 HP (Mobius setMortal).
---@param mortal? boolean true when left out
function Npc:set_mortal(mortal) end

---@return boolean
function Npc:is_mortal() end

---@param enabled? boolean true when left out
function Npc:set_random_walking(enabled) end

---Back to the spawn point, hate forgotten (Mobius Attackable.returnHome).
function Npc:return_home() end

---The AI intention, an Intention value such as Intention.ATTACK.
---@return string
function Npc:intention() end

---Sets the AI intention (Mobius getAI().setIntention), with a target or a
---position when it takes one.
---@param intention string an Intention value
---@param arg? Player|Npc|Position
function Npc:set_intention(intention, arg) end

---@param target Player|Npc
function Npc:follow(target) end

function Npc:stop_follow() end

---Stops where it is (Mobius clientStopMoving).
function Npc:stop_moving() end

---@return boolean
function Npc:is_moving() end

---@return boolean
function Npc:is_in_combat() end

---@return boolean
function Npc:is_casting_now() end

---Turns the NPC.
---@param heading integer
function Npc:set_heading(heading) end

---An entry of npc:hate_list().
---@class HateEntry
---@field target Player|Npc
---@field hate integer
---@field damage integer

---The aggro list, most hated first (Mobius getAggroList).
---@return HateEntry[]
function Npc:hate_list() end

---The most hated creature, nil for none (Mobius getMostHated).
---@return Player|Npc|nil
function Npc:most_hated() end

---Forgets every hate (Mobius clearAggroList).
function Npc:clear_aggro_list() end

---Forgets the hate for target (Mobius stopHating).
---@param target Player|Npc
function Npc:stop_hating(target) end

---Lowers the hate for target (Mobius reduceHate).
---@param target Player|Npc
---@param amount integer
function Npc:reduce_hate(target, amount) end

-- Spawn data and template parameters.

---The spawn point, nil for an NPC without a spawn.
---@return Position?
function Npc:spawn_location() end

---Whether the NPC stands on its spawn point's x and y (Mobius
---staysInSpawnLoc).
---@return boolean
function Npc:stays_in_spawn_loc() end

---A value of the spawn's AI data, nil when it has none (Mobius
---getAIValue).
---@param name string
---@return integer?
function Npc:ai_value(name) end

---@param name string
---@return boolean
function Npc:has_ai_value(name) end

---A template parameter (Mobius getParameters()), or default.
---@param name string
---@param default? string
---@return string?
function Npc:param(name, default) end

---A template parameter as an integer, or default (0).
---@param name string
---@param default? integer
---@return integer
function Npc:param_int(name, default) end

-- Leaders and minions (Mobius MinionList).

---The NPC's leader, nil for none.
---@return Npc?
function Npc:leader() end

---The leader's living minions.
---@return Npc[]
function Npc:minions() end

---@return boolean
function Npc:has_minions() end

---Spawns a minion of the template for this leader (Mobius addMinion).
---@param npc_id integer
---@return Npc?
function Npc:add_minion(npc_id) end

-- Script events.

---Sends on_event_received(name, npc, receiver, reference) to every NPC
---within radius whose template a script listens for (Mobius
---broadcastEvent). The handlers run after the current one.
---@param name string
---@param radius integer
---@param reference? Player|Npc
function Npc:broadcast_event(name, radius, reference) end

---Sends on_event_received(name, npc, receiver, reference) to one NPC
---(Mobius sendScriptEvent).
---@param name string
---@param receiver Npc
---@param reference? Player|Npc
function Npc:send_event(name, receiver, reference) end

-- Treasure chests (Mobius Chest).

---Whether the chest is a box (a mimic is not).
---@return boolean
function Npc:is_box() end

---@return boolean
function Npc:is_interacted() end

function Npc:set_interacted() end

---The box explodes (Mobius treasureBomb).
function Npc:treasure_bomb() end

---Whether the NPC is inside the zone.
---@param zone_id integer
---@return boolean
function Npc:is_inside_zone(zone_id) end

---The distance to a position, a player or an NPC, without height.
---@param other Player|Npc|Position
---@return number
function Npc:calculate_distance_2d(other) end

---The distance to a position, a player or an NPC, with height.
---@param other Player|Npc|Position
---@return number
function Npc:calculate_distance_3d(other) end

---Whether the NPC could cast the skill now (Mobius checkDoCastConditions):
---not disabled, enough MP and HP, not muted for it. false for a nil skill.
---@param skill Skill? from skills.get
---@return boolean
function Npc:check_do_cast_conditions(skill) end

---Removes the skill's effects from the NPC (Mobius
---stopSkillEffects(REMOVED, id)).
---@param skill integer|Skill a skill ID or a skill
function Npc:stop_skill_effects(skill) end

-- The grand boss scripts' NPC methods.

---Casts the skill at the NPC's target when it can (Mobius
---Attackable.useMagic): not while casting, with the MP for it, not muted.
---A nil skill does nothing.
---@param skill Skill? from skills.get
function Npc:use_magic(skill) end

---The NPC's target (Mobius getTarget), set by npc:set_target or its AI.
---@return Player|Npc|nil
function Npc:target() end

---Sets the current HP and MP (Mobius setCurrentHpMp), at most the maximum.
---@param hp number
---@param mp number
function Npc:set_hp_mp(hp, mp) end

---The current MP (Mobius getCurrentMp).
---@return number
function Npc:mp() end

---The maximum MP (Mobius getMaxMp).
---@return number
function Npc:max_mp() end

---For a dead minion: it comes back after ms while its leader lives,
---instead of after the default time (Mobius
---getLeader().getMinionList().onMinionDie(minion, ms)).
---@param ms integer
function Npc:minion_respawn(ms) end

---Moves the NPC's spawn point, where it walks home to and respawns
---(Mobius getSpawn().setLocation).
---@param pos Position|Player|Npc
function Npc:set_spawn_location(pos) end

---Makes the NPC walk into the skill's range of target and cast it (Mobius
---getAI().setIntentionCast). A nil skill does nothing.
---@param skill Skill? from skills.get
---@param target Player|Npc
function Npc:set_intention_cast(skill, target) end

---The level of the skill's effect on the NPC, nil when it is not on it
---(Mobius getEffectList().getBuffInfoBySkillId(id).getSkill().getLevel()).
---@param skill integer|Skill
---@return integer?
function Npc:effect_level(skill) end

---Whether the skill's effects are on the NPC (Mobius isAffectedBySkill).
---@param skill integer|Skill
---@return boolean
function Npc:is_affected_by_skill(skill) end

-- Monster flags (Mobius ai/others).

---With false, lethal strikes no longer work on the NPC (Mobius
---setLethalable).
---@param lethalable? boolean true when left out
function Npc:set_lethalable(lethalable) end

---With true, the monster is aggressive towards players in Silent Move too
---(Mobius Attackable.setSeeThroughSilentMove).
---@param see? boolean true when left out
function Npc:set_see_through_silent_move(see) end

---Whether the NPC's world region is active: players are near (Mobius
---getWorldRegion().isActive()).
---@return boolean
function Npc:is_in_active_region() end

---A minion list of the template parameters, e.g. "Privates" (Mobius
---getParameters().getMinionList(name)); empty when there is none.
---@param name string
---@return {id: integer, count: integer}[]
function Npc:param_minions(name) end

---false: tells an NPC from a player where a handler gets either (Mobius
---isPlayer).
---@return boolean
function Npc:is_player() end

-- More area script methods (Mobius ai/areas).

---Whether the NPC is in the middle of a swing (Mobius isAttackingNow).
---@return boolean
function Npc:is_attacking_now() end

---Whether the NPC cannot use the skill now (Mobius isSkillDisabled): under
---reuse, or all its skills disabled. true for a nil skill.
---@param skill Skill?
---@return boolean
function Npc:is_skill_disabled(skill) end

---The aggression range (Mobius getAggroRange).
---@return integer
function Npc:aggro_range() end

---The template's clan help range (Mobius getTemplate().getClanHelpRange()).
---@return integer
function Npc:clan_help_range() end

---A skill of the template parameters, e.g. "PhysicalSpecial1" (Mobius
---getParameters().getObject(name, SkillHolder.class)); nil when there is
---none.
---@param name string
---@return Skill?
function Npc:param_skill(name) end

---Shows the player the NPC's teleport list of that name (Mobius
---TeleporterData.getHolder(npc_id, list).showTeleportList(player, npc));
---false when the NPC has no such list.
---@param player Player
---@param list string e.g. "list1"
---@return boolean
function Npc:show_teleport_list(player, list) end

-------------------------------------------------------------------------------
-- rnd, time, world, globals, util
-------------------------------------------------------------------------------

---Random numbers with Mobius Rnd's rules, from the server's one random
---source (tests script it). `math.random` uses the same source;
---`math.randomseed` does not exist.
---
---* `rnd(n)`: an integer in [0, n); 0 when n ≤ 0 (Mobius getRandom(n)).
---  Mind Lua's `math.random(n)`, which is [1, n].
---* `rnd(a, b)`: an integer in [a, b]; a when a ≥ b.
---@class rndlib
---@overload fun(n: integer): integer
---@overload fun(a: integer, b: integer): integer
rnd = {}

---true or false (Mobius getRandomBoolean).
---@return boolean
function rnd.bool() end

---A number in [0, 1), or [0, max) (Mobius Rnd.nextDouble, Rnd.get(double)).
---@param max? number
---@return number
function rnd.double(max) end

---A random element of a list; nil for an empty one (Mobius
---getRandomEntry).
---@generic T
---@param list T[]
---@return T?
function rnd.entry(list) end

---The server clock.
time = {}

---Milliseconds since 1970 on the world clock (System.currentTimeMillis).
---@return integer
function time.now_ms() end

---The game ticks (Mobius GameTimeTaskManager.getGameTicks).
---@return integer
function time.game_ticks() end

---The game time: minutes since game midnight, 0 to 1439 (Mobius
---getGameTime). A game day lasts 4 real hours.
---@return integer
function time.game_time() end

---The game hour, 0 to 23.
---@return integer
function time.game_hour() end

---Whether it is game night (hour < 6).
---@return boolean
function time.is_night() end

---World objects by object ID. Keep object IDs in fields across handlers,
---and turn them back into values here.
world = {}

---The player or NPC with the object ID, or nil (Mobius World.findObject).
---@param object_id integer
---@return Player|Npc|nil
function world.find_object(object_id) end

---The player with the object ID, or nil.
---@param object_id integer
---@return Player?
function world.get_player(object_id) end

---Whether an NPC of the template is in the world (Mobius
---World.getVisibleObjects() with anyMatch on getId()).
---@param npc_id integer
---@return boolean
function world.npc_exists(npc_id) end

---The NPCs of a template in the world. Restart-safe scripts find the
---NPCs they spawned before a reload here.
---@param npc_id integer
---@return Npc[]
function world.npcs_by_id(npc_id) end

---The NPCs within radius of obj, obj left out (Mobius
---World.getVisibleObjectsInRange).
---@param obj Player|Npc
---@param radius? integer the visible range when left out
---@return Npc[]
function world.npcs_around(obj, radius) end

---The players within radius of obj.
---@param obj Player|Npc
---@param radius? integer the visible range when left out
---@return Player[]
function world.players_around(obj, radius) end

---The objects of a Mobius class around obj (World.getVisibleObjects(obj,
---class)): "Player" and "Playable" are players, "Npc" every NPC,
---"Creature" and "WorldObject" both, any other class the NPCs of that
---class (npc:is_type).
---@param obj Player|Npc
---@param class? string "WorldObject" when left out
---@param radius? integer the visible range when left out
---@return (Player|Npc)[]
function world.visible_objects(obj, class, radius) end

---Whether from can see to: doors, fences and geodata between them (Mobius
---GeoEngine.canSeeTarget). Without geodata, true unless a door is closed
---between them.
---@param from Player|Npc
---@param to Player|Npc
---@return boolean
function world.can_see_target(from, to) end

---The last point of a straight walk from from to to, stopped by geodata
---(Mobius GeoEngine.getValidLocation). Without geodata, to.
---@param from Position|Player|Npc
---@param to Position|Player|Npc
---@return Position
function world.valid_location(from, to) end

---The skill data and effects (Mobius SkillData, Skill).
skills = {}

---A level of a skill, nil when it does not exist (Mobius
---SkillData.getSkill(id, level)).
---@param id integer
---@param level? integer 1 when left out
---@return Skill?
function skills.get(id, level) end

---Applies the skill's effects of effector on effected (Mobius
---skill.applyEffects(effector, effected)). A nil skill does nothing.
---@param skill Skill? from skills.get
---@param effector Player|Npc
---@param effected Player|Npc
function skills.apply_effects(skill, effector, effected) end

---An item template, as items.get returns it.
---@class ItemTemplate
---@field id integer
---@field name string
---@field stackable boolean

---The item data (Mobius ItemData).
items = {}

---An item template, nil for an unknown ID (Mobius
---ItemData.getTemplate(id)).
---@param id integer
---@return ItemTemplate?
function items.get(id) end

---Server-wide variables, saved (Mobius GlobalVariablesManager). Values are
---strings.
globals = {}

---@param name string
---@param default? string
---@return string?
function globals.get(name, default) end

---@param name string
---@param default? integer
---@return integer
function globals.get_int(name, default) end

---@param name string
---@param default? integer
---@return integer
function globals.get_long(name, default) end

---@param name string
---@param default? boolean
---@return boolean
function globals.get_bool(name, default) end

---@param name string
---@param value string|number|boolean
function globals.set(name, value) end

---Helpers from Mobius util classes.
util = {}

---Mobius LocationUtil.checkIfInRange: whether a and b are within range of
---each other (collision radii added), with or without height. false when
---one is nil.
---@param range integer
---@param a Player|Npc|nil
---@param b Player|Npc|nil
---@param include_z boolean
---@return boolean
function util.in_range(range, a, b, include_z) end

---Mobius LocationUtil.calculateAngleFrom: the angle in degrees, 0 to 360,
---of the line from a to b.
---@param a Player|Npc|Position
---@param b Player|Npc|Position
---@return number
function util.calculate_angle_from(a, b) end

-------------------------------------------------------------------------------
-- zones, doors, grandboss, cinematics
-------------------------------------------------------------------------------

---Zones as boss scripts use them (Mobius ZoneType, BossZone).
zones = {}

---The players inside the zone.
---@param zone_id integer
---@return Player[]
function zones.players_inside(zone_id) end

---Whether a player, an NPC or a position is inside the zone.
---@param zone_id integer
---@param obj Player|Npc|Position
---@return boolean
function zones.is_inside(zone_id, obj) end

---Teleports every player inside to pos (Mobius movePlayersTo).
---@param zone_id integer
---@param pos Position|Player|Npc
function zones.move_players(zone_id, pos) end

---Sends every player inside to the zone's oust point, or town (Mobius
---oustAllPlayers).
---@param zone_id integer
function zones.oust_all(zone_id) end

---Lets the player enter a boss zone for seconds (Mobius
---allowPlayerEntry).
---@param zone_id integer
---@param player Player
---@param seconds integer
function zones.allow_entry(zone_id, player, seconds) end

---Takes the player off the zone's allowed list (Mobius removePlayer).
---@param zone_id integer
---@param player Player
function zones.remove_player(zone_id, player) end

---The NPCs inside the zone (Mobius getCharactersInside without the
---players).
---@param zone_id integer
---@return Npc[]
function zones.npcs_inside(zone_id) end

---The zones containing a position, in zone ID order (Mobius
---ZoneManager.getZones); zone:type() is the Mobius zone type, e.g.
---"EffectZone".
---@param pos Position|Player|Npc
---@return Zone[]
function zones.at(pos) end

---A parameter of the zone's data, e.g. "skillIdLvl" of an EffectZone; nil
---when it has none.
---@param zone_id integer
---@param name string
---@return string?
function zones.param(zone_id, name) end

---Turns a zone on or off (Mobius ZoneType.setEnabled).
---@param zone_id integer
---@param enabled boolean
function zones.set_enabled(zone_id, enabled) end

---Doors (Mobius DoorData, Door).
doors = {}

---Opens a door; false for an unknown door.
---@param door_id integer
---@return boolean
function doors.open(door_id) end

---Closes a door; false for an unknown door.
---@param door_id integer
---@return boolean
function doors.close(door_id) end

---Whether the door is open; nil for an unknown door.
---@param door_id integer
---@return boolean?
function doors.is_open(door_id) end

---Grand boss state, saved (Mobius GrandBossManager). Statuses are the
---script's numbers; Mobius scripts use 0 alive, 1 dead, 2 fighting, 3
---waiting.
grandboss = {}

---The saved status, nil for an unknown boss.
---@param boss_id integer
---@return integer?
function grandboss.status(boss_id) end

---Saves the status at once (Mobius setStatus).
---@param boss_id integer
---@param status integer
function grandboss.set_status(boss_id, status) end

---A grand boss's saved data (Mobius StatSet).
---@class BossInfo
---@field respawn_time integer Unix milliseconds, 0 for none
---@field x integer
---@field y integer
---@field z integer
---@field heading integer
---@field hp number
---@field mp number

---The saved data, nil for an unknown boss.
---@param boss_id integer
---@return BossInfo?
function grandboss.info(boss_id) end

---Saves the data (Mobius setStatSet).
---@param boss_id integer
---@param info BossInfo
function grandboss.set_info(boss_id, info) end

---The live boss, nil when it is not in the world (Mobius getBoss).
---@param boss_id integer
---@return Npc?
function grandboss.boss(boss_id) end

---Registers a spawned boss as the live boss of its template (Mobius
---addBoss).
---@param npc Npc
function grandboss.add_boss(npc) end

---The server side of the seasonal events (Mobius LongTimeEvent), by script
---name: their data from config.xml with the config's dates, the event
---drops, the login notice and the item cleanup. scripts/lib/long_time_event.lua
---drives them; event scripts use that library instead.
long_events = {}

---One NPC of an event's spawn list.
---@class LongEventSpawn
---@field npc integer
---@field x integer
---@field y integer
---@field z integer
---@field heading integer
---@field respawn_ms integer 0: no respawn

---An event's data, its periods in Unix milliseconds.
---@class LongEventConfig
---@field name string
---@field start_ms integer
---@field end_ms integer
---@field drop_start_ms integer
---@field drop_end_ms integer may be before drop_start_ms: no drops
---@field on_enter string
---@field on_end string
---@field spawns LongEventSpawn[]

---The event's data, or nil and why (no such event, bad dates).
---@param name string
---@return LongEventConfig?
---@return string? error
function long_events.config(name) end

---Marks the event running or not (Mobius _active).
---@param name string
---@param active boolean
function long_events.set_active(name, active) end

---@param name string
---@return boolean
function long_events.is_active(name) end

---Starts or stops the event's monster drops (Mobius EventDropManager).
---@param name string
---@param on boolean
function long_events.set_drops(name, on) end

---Shows text to players entering the world from start_ms to end_ms (nil: no
---end); nil text removes it (Mobius EventAnnouncement).
---@param name string
---@param text string?
---@param start_ms? integer
---@param end_ms? integer
function long_events.set_notice(name, text, start_ms, end_ms) end

---Announces text to every player in the game.
---@param text string
function long_events.announce(text) end

---Takes the event's destroyItemsOnEnd items from every character.
---@param name string
function long_events.destroy_items(name) end

---Puts the event's NPCs into the world, each leaving after despawn_ms (0
---or nil: never). The server does it right after the running handler.
---@param name string
---@param despawn_ms? integer
function long_events.spawn(name, despawn_ms) end

---Removes the NPCs long_events.spawn put into the world.
---@param name string
function long_events.despawn(name) end

---The heroes of the Olympiad (Mobius Hero), for the Monument of Heroes.
hero = {}

---Mobius Hero.isHero: a hero of the running cycle that claimed the status.
---@param player Player
---@return boolean
function hero.is_hero(player) end

---Mobius Hero.isUnclaimedHero: a hero of the last period's end that has
---not claimed the status yet.
---@param player Player
---@return boolean
function hero.is_unclaimed(player) end

---Mobius Hero.claimHero: the player takes the hero status (the hero skills
---and aura, the clan reputation, the animation).
---@param player Player
function hero.claim(player) end

---The castle manor (Mobius CastleManorManager), for the manor managers and
---the chamberlains. The manor's windows (the manor_menu_select links) are
---the game's.
manor = {}

---Mobius GeneralConfig.ALLOW_MANOR (the config's manor.allowManor).
---@return boolean
function manor.is_enabled() end

---The NPC's manor_id template parameter: the castle whose manor it serves
---(or -1 without one).
---@param npc Npc
---@return integer
function manor.manor_id(npc) end

---ManorManager.onFirstTalk's lord test: the player is no GM, leads its clan,
---and the clan owns the castle of the NPC's manor_id.
---@param npc Npc
---@param player Player
---@return boolean
function manor.is_lord(npc, player) end

---What the castle's seeds of the running period brought so far (the
---chamberlain's vault page).
---@param castle_id integer
---@return integer
function manor.seed_income(castle_id) end

---The Seven Signs (Mobius SevenSigns), for the priests' dialogs and the
---castle chamberlain. Cabals are 0 none, 1 Dusk, 2 Dawn; seals 1 avarice,
---2 gnosis, 3 strife; periods 0 recruiting, 1 competition, 2 results, 3
---seal validation. Changes are saved at once.
sevensigns = {}

---The current period.
---@return integer
function sevensigns.period() end

---The current cycle (1 for a new server).
---@return integer
function sevensigns.cycle() end

---A seal's owner (Mobius getSealOwner).
---@param seal integer
---@return integer
function sevensigns.seal_owner(seal) end

---The leading cabal, 0 on a tie (Mobius getCabalHighestScore).
---@return integer
function sevensigns.highest_cabal() end

---The player's cabal (Mobius getPlayerCabal).
---@param player Player
---@return integer
function sevensigns.cabal(player) end

---The seal the player fights for, 0 for none.
---@param player Player
---@return integer
function sevensigns.seal(player) end

---The player's contribution score.
---@param player Player
---@return integer
function sevensigns.contribution(player) end

---The player joins the cabal for the seal (Mobius setPlayerInfo).
---@param player Player
---@param cabal integer
---@param seal integer
function sevensigns.join(player, cabal, seal) end

---The stones count for the player's cabal (Mobius addPlayerStoneContrib):
---the contribution they are worth, -1 (nothing changes) above the limit.
---@param player Player
---@param blue integer
---@param green integer
---@param red integer
---@return integer
function sevensigns.contribute(player, blue, green, red) end

---The ancient adena the player may collect; its stones and ancient adena
---are cleared (Mobius getAncientAdenaReward(id, true)).
---@param player Player
---@return integer
function sevensigns.take_reward(player) end

---The castle ID of the player's clan, 0 for none.
---@param player Player
---@return integer
function sevensigns.clan_castle(player) end

---The dialogs' settings: max_player_contrib, join_dawn_fee,
---manors_agreement_id, dawn_ticket_quantity, dawn_ticket_price,
---dawn_ticket_bundle, castle_for_dawn, castle_for_dusk.
---@return table
function sevensigns.config() end

---Mobius Player.destroyItemByItemId: takes count items, with the
---"disappeared" message when send_message is set.
---@param player Player
---@param item_id integer
---@param count integer
---@param send_message boolean
---@return boolean
function sevensigns.destroy_item(player, item_id, count, send_message) end

---Mobius Player.addItem (addAdena, addAncientAdena): gives count items,
---with the "You have earned" message when send_message is set.
---@param player Player
---@param item_id integer
---@param count integer
---@param send_message boolean
---@return boolean
function sevensigns.add_item(player, item_id, count, send_message) end

---Mobius Player.reduceAdena(..., true).
---@param player Player
---@param count integer
---@return boolean
function sevensigns.reduce_adena(player, count) end

---Sends "$s2 $s1 has disappeared." with the item's name and the count.
---@param player Player
---@param item_id integer
---@param count integer
function sevensigns.send_disappeared(player, item_id, count) end

---Whether the inventory has room for slots more items (validateCapacity).
---@param player Player
---@param slots integer
---@return boolean
function sevensigns.has_room(player, slots) end

---A castle's Dawn ticket sales this competition (Castle.getTicketBuyCount).
---@param castle_id integer
---@return integer
function sevensigns.ticket_count(castle_id) end

---Sets a castle's Dawn ticket sales (saved).
---@param castle_id integer
---@param n integer
function sevensigns.set_ticket_count(castle_id, n) end

---What boss fights show. `to` is who sees it: a Player (that player), an
---Npc (the players around it) or a zone ID (the players inside). Skill
---animations are npc:magic_skill_use.
cinematics = {}

---Points the camera at target (Mobius SpecialCamera, same argument
---order). Arguments left out are 0.
---@param to Player|Npc|integer
---@param target Player|Npc
---@param force integer
---@param angle1 integer
---@param angle2 integer
---@param time integer
---@param range integer
---@param duration integer
---@param rel_yaw? integer
---@param rel_pitch? integer
---@param wide? integer
---@param rel_angle? integer
---@param unk? integer
function cinematics.special_camera(to, target, force, angle1, angle2, time, range, duration, rel_yaw, rel_pitch, wide, rel_angle, unk) end

---Shakes the screens (Mobius Earthquake).
---@param to Player|Npc|integer
---@param pos Position|Player|Npc
---@param intensity integer
---@param duration integer seconds
function cinematics.earthquake(to, pos, intensity, duration) end

---Plays a sound, coming from source when given (Mobius PlaySound).
---@param to Player|Npc|integer
---@param sound string e.g. "BS02_A"
---@param source? Player|Npc
function cinematics.sound(to, sound, source) end

---Shows text on screen (Mobius ExShowScreenMessage).
---@param to Player|Npc|integer
---@param text string
---@param ms integer
---@param position? integer 2 (top center) when left out
function cinematics.screen_message(to, text, ms, position) end

---The ferries (Mobius vehicles/Boat*.java). A schedule is data: the game
---runs the boat, its path, the docks and the tickets. scripts/lib/boat.lua
---has the docks, messages and a helper; scripts/vehicles/ the five routes.
vehicles = {}

---A point of a boat path: {x, y, z, speed, rotation}. Without the speeds
---the boat sails at 350 and turns at 4000 (Mobius VehiclePathPoint); a
---speed of 0 teleports it there, facing `rotation`.
---@alias VehiclePoint integer[]

---What the players within the boat broadcast radius of either `near`
---point hear: the boat announcements (system message IDs), then the horn.
---@class VehicleAnnounce
---@field near VehiclePoint[] two points
---@field messages? integer[] system message IDs
---@field horn? VehiclePoint where the horn plays
---@field dock? integer only in `await`: the dock to wait for

---One case of the schedule, done in this order: wait for a free dock
---(`await`: try again every 5 s, announcing the delay on the first try and
---every 36th after), free (`undock`) or take (`dock`) a dock, `say`, the
---`horn` around the boat, take the tickets (`pay`), sail the `path`, and
---run the next step after `wait` ms, or when the path ends without one.
---@class VehicleStep
---@field await? VehicleAnnounce
---@field undock? integer
---@field dock? integer
---@field say? VehicleAnnounce
---@field horn? VehiclePoint
---@field pay? {item: integer, count?: integer, oust: VehiclePoint} item 0: no ticket
---@field path? VehiclePoint[]
---@field wait? integer milliseconds

---A ferry schedule.
---@class VehicleRoute
---@field name string
---@field boat integer the Mobius boat ID
---@field start integer[] {x, y, z, heading}
---@field start_delay? integer ms before the first step
---@field dock? integer the dock the boat holds at startup
---@field steps VehicleStep[]

---Declares a ferry schedule; only while the scripts load. The game starts
---one boat per route at startup (when boats are allowed) and runs its steps
---in a cycle. A reload does not change running boats.
---@param route VehicleRoute
function vehicles.add_route(route) end

-------------------------------------------------------------------------------
-- Constants
-------------------------------------------------------------------------------
-- Global tables of Mobius constants. They are read-only, and reading a
-- name a table does not have is an error.

---Player classes: `ClassId.FIGHTER` is 0, `ClassId.WARRIOR` 1, ... (Mobius
---PlayerClass names), with helpers.
---@class classlib
---@field [string] integer
ClassId = {}

---The class level: 0 for a starting class, 1 after the first change, ...
---@param id integer
---@return integer
function ClassId.level(id) end

---The race of the class (a Race value).
---@param id integer
---@return integer
function ClassId.race(id) end

---The class it changes from; nil for a starting class.
---@param id integer
---@return integer?
function ClassId.parent(id) end

---Whether the class is a mystic.
---@param id integer
---@return boolean
function ClassId.is_mage(id) end

---Quest states.
---@class State
---@field CREATED integer 0
---@field STARTED integer 1
---@field COMPLETED integer 2
State = {}

---How exit_quest ends a quest.
---@class QuestType
---@field REPEATABLE integer 0: the state is deleted
---@field ONE_TIME integer 1: COMPLETED
---@field DAILY integer 2: COMPLETED until 06:30
QuestType = {}

---Quest sounds: the values are sound names, e.g.
---`QuestSound.ITEMSOUND_QUEST_ACCEPT` is "ItemSound.quest_accept".
---@class QuestSound
---@field [string] string
QuestSound = {}

---Paperdoll slots for q:get_item_equipped: UNDER, LEAR, REAR, NECK,
---LFINGER, RFINGER, HEAD, RHAND, LHAND, GLOVES, CHEST, LEGS, FEET, CLOAK,
---FACE, HAIR, HAIR2.
---@class Paperdoll
---@field [string] integer
Paperdoll = {}

---Chat types for npc:broadcast_say: GENERAL, SHOUT, ..., NPC_GENERAL,
---NPC_SHOUT.
---@class ChatTypes
---@field [string] integer
ChatType = {}

---AI intentions for npc:intention() and npc:set_intention: IDLE, ACTIVE,
---REST, ATTACK, CAST, MOVE_TO, FOLLOW, PICK_UP, INTERACT (the values are
---the names).
---@class Intentions
---@field [string] string
Intention = {}

---Races: HUMAN 0, ELF 1, DARK_ELF 2, ORC 3, DWARF 4, then the NPC races.
---@class Race
---@field [string] integer
Race = {}

---Player categories (Mobius CategoryType), for player:is_in_category.
---@class Category
---@field [string] integer
Category = {}

---System messages (Mobius SystemMessageId), for
---player:send_system_message.
---@class SystemMessageId
---@field [string] integer
SystemMessageId = {}

---Abnormal types (Mobius AbnormalType), for
---player:buff_info_by_abnormal_type (the values are the names).
---@class AbnormalType
---@field [string] string
AbnormalType = {}

---Server settings quests read, with Mobius names: ALT_PARTY_RANGE,
---ALT_VILLAGES_REPEATABLE_QUEST_REWARD, DISABLE_TUTORIAL,
---RAID_MIN_RESPAWN_MULTIPLIER, RAID_MAX_RESPAWN_MULTIPLIER,
---RAID_DISABLE_CURSE, AUTO_LOOT; and GrandBoss.ini's (the grandBosses section):
---<BOSS>_SPAWN_INTERVAL and <BOSS>_SPAWN_RANDOM in hours for ANTHARAS,
---VALAKAS, BAIUM, CORE, ORFEN, QUEEN_ANT and ZAKEN, ANTHARAS_WAIT_TIME and
---VALAKAS_WAIT_TIME in minutes, ANTHARAS_, VALAKAS_ and
---BAIUM_RECOGNIZE_HERO.
---@class Config
---@field [string] any
Config = {}

-------------------------------------------------------------------------------
-- html
-------------------------------------------------------------------------------

---Helpers for dialog HTML.
html = {}

---Replaces every `%name%` in page with vars.name, in one pass. Names may
---also be written with their percent signs.
---
---```lua
---return html.replace("<html><body>Hello, %name%!</body></html>", { name = player:name() })
---```
---@param page string
---@param vars Placeholders
---@return string
function html.replace(page, vars) end

-------------------------------------------------------------------------------
-- lib/java.lua: Java helpers for translated quests
-------------------------------------------------------------------------------

---Java library behaviour for quests translated from Mobius, from
---`local J = require("lib.java")`. String indexes are 0-based like Java's;
---lists are 1-based sequences, maps tables, sets tables of `value = true`,
---and nil is Java's null. Where Java throws, the helper raises an error
---starting with the exception name ("NumberFormatException: ...").
---@class JavaLib
---@field todo fun(reason: string, java?: string) Java code the translator could not translate: errors when reached.
---@field idiv fun(a: number, b: number): integer Integer a / b, truncated toward zero; b == 0 is an ArithmeticException.
---@field int fun(x: number, kind?: "long"): integer (int) cast: truncated, NaN is 0, clamped to the int range (the long range with "long").
---@field round fun(x: number): integer Math.round: ties toward positive infinity, NaN is 0.
---@field constrain fun(value: number, min: number, max: number): number MathUtil.clamp: max(min, min(max, value)).
---@field band fun(a: integer, b: integer): integer 32-bit a & b.
---@field bor fun(a: integer, b: integer): integer 32-bit a | b.
---@field bxor fun(a: integer, b: integer): integer 32-bit a ^ b.
---@field bnot fun(a: integer): integer 32-bit ~a.
---@field shl fun(a: integer, n: integer): integer 32-bit a << n.
---@field shr fun(a: integer, n: integer): integer 32-bit a >> n (sign-extending).
---@field ushr fun(a: integer, n: integer): integer 32-bit a >>> n (zero-filling).
---@field parse_int fun(s: string): integer Integer.parseInt: sign and digits only, within the int range.
---@field parse_long fun(s: string): integer Long.parseLong: sign and digits only, within the long range.
---@field parse_double fun(s: string): number Double.parseDouble.
---@field parse_bool fun(s: string?): boolean Boolean.parseBoolean: true only for "true" in any case.
---@field is_numeric fun(s: string?): boolean StringUtil.isNumeric: a non-empty string of digits.
---@field str fun(x: number?): string String.valueOf(double): "1.0", "0.25", "1.0E7", "NaN".
---@field equals_ignore_case fun(s: string, other: string?): boolean String.equalsIgnoreCase.
---@field starts_with fun(s: string, prefix: string): boolean String.startsWith.
---@field ends_with fun(s: string, suffix: string): boolean String.endsWith.
---@field str_contains fun(s: string, part: string): boolean String.contains (plain text).
---@field substring fun(s: string, begin: integer, end_?: integer): string String.substring, 0-based, end exclusive.
---@field index_of fun(s: string, part: string, from?: integer): integer String.indexOf, 0-based, -1 if missing.
---@field last_index_of fun(s: string, part: string): integer String.lastIndexOf, 0-based, -1 if missing.
---@field char_at fun(s: string, i: integer): string String.charAt, 0-based, as a one-byte string.
---@field trim fun(s: string): string String.trim: strips bytes up to the space from both ends.
---@field replace fun(s: string, target: string, replacement: string): string String.replace: every literal occurrence.
---@field replace_first fun(s: string, target: string, replacement: string): string String.replaceFirst with a literal pattern ($0 and \x in the replacement as Java).
---@field replace_all fun(s: string, target: string, replacement: string): string String.replaceAll with a literal pattern ($0 and \x in the replacement as Java).
---@field split fun(s: string, sep: string): string[] String.split with a literal separator; trailing empty strings dropped.
---@field split_ws fun(s: string, single?: boolean): string[] String.split("\\s+") (with single, "\\s"): ASCII whitespace separates; trailing empty strings dropped.
---@field array fun(default: any, n: integer, ...: integer): any[] new T[n][m]...: nested arrays filled with default.
---@field copy fun(t: table): table Shallow copy: clone(), toArray(), copy constructors.
---@field copy_of fun(arr: any[], n: integer, default?: any): any[] Arrays.copyOf: truncated or padded (0, false or nil).
---@field fill fun(arr: any[], value: any) Arrays.fill.
---@field shuffle fun(list: any[]) Collections.shuffle, in place.
---@field contains fun(seq: any[], value: any): boolean List.contains / ArrayUtil.contains.
---@field index_of_value fun(list: any[], value: any): integer List.indexOf, 0-based, -1 if missing.
---@field add fun(list: any[], value: any): boolean List.add as a value: appends, returns true.
---@field add_all fun(list: any[], other: table): boolean List.addAll: appends a list (or a set's elements, sorted); true if changed.
---@field remove_value fun(list: any[], value: any): boolean List.remove(Object): removes the first equal element.
---@field list_to_string fun(list: any[]): string List.toString: "[a, b, c]".
---@field keys fun(t: table): any[] keySet() or a set's elements, as a sorted list.
---@field values fun(t: table): any[] values(), in sorted key order.
---@field spairs fun(t: table): (fun(): any, any) Iterates a map or set in sorted key order.
---@field size fun(t: table): integer Map.size / Set.size.
---@field clear fun(t: table) clear() on a list, map or set.
---@field put fun(map: table, key: any, value: any): any Map.put as a value: returns the previous value.
---@field put_if_absent fun(map: table, key: any, value: any): any Map.putIfAbsent: returns the value that was there.
---@field remove_key fun(map: table, key: any): any Map.remove as a value: returns the removed value.
---@field get_or_default fun(map: table, key: any, default: any): any Map.getOrDefault.
---@field contains_value fun(map: table, value: any): boolean Map.containsValue.
---@field set_of fun(list: any[]): table<any, true> new HashSet<>(list).
---@field set_add fun(set: table, value: any): boolean Set.add as a value: true if new.
---@field set_remove fun(set: table, value: any): boolean Set.remove as a value: true if it was there.
---@field oid fun(obj: Player|Npc|integer|nil): integer? A world object as its object ID, for keeping in a field.
---@field obj fun(id: integer?): Player|Npc|nil A field written by oid back as the object (world.find_object), nil if gone.
---@field instance_of fun(x: any, class: string): boolean x instanceof Class; other NPC classes through npc:is_type.
