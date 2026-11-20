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
-- limit (50 ms by default). Player, Npc and QuestState values are valid only
-- while the handler that received them runs; within it, one object is
-- always the same value.

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

---Runs on the world loop once the scripts are in place, at startup and
---after each reload: what Mobius scripts do in their constructors (spawns,
---timers, global variables). Loading itself cannot touch the world.
---@param fn fun()
function Script:on_start(fn) end

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
---Counts below 1 and unknown items are ignored.
---@param player Player
---@param item_id integer
---@param count integer
---@param enchant? integer
function Script:give_items(player, item_id, count, enchant) end

---Gives items with the quest reward rates (Mobius rewardItems).
---@param player Player
---@param item_id integer
---@param count integer
function Script:reward_items(player, item_id, count) end

---Gives adena, with the quest reward rate when apply_rates is set.
---@param player Player
---@param count integer
---@param apply_rates boolean
function Script:give_adena(player, count, apply_rates) end

---Takes count items (-1: all of them); a worn item comes off first.
---Returns true, as Mobius does.
---@param player Player
---@param item_id integer
---@param count integer
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
---member near npc that on_check_party_member accepts. Until parties exist,
---the player or nil.
---@param player Player?
---@param cond_var_or_npc? integer|string|Npc
---@param value? string
---@return Player?
function Script:get_random_party_member(player, cond_var_or_npc, value) end

---Mobius getRandomPartyMemberState: (player, state) → a member with the
---state; (player, cond, chance, npc) → the state of a member at cond (-1:
---started) near npc that on_check_party_member_state accepts. Until
---parties exist, the player's.
---@param player Player?
---@param state_or_cond integer
---@param chance? integer
---@param npc? Npc
---@return Player|QuestState|nil
function Script:get_random_party_member_state(player, state_or_cond, chance, npc) end

---Runs on_action_for_each_player for the player (and, once parties
---exist, its party or command channel).
---@param player Player
---@param npc Npc?
---@param is_summon boolean
---@param include_party boolean
---@param include_command_channel boolean
function Script:execute_for_each_player(player, npc, is_summon, include_party, include_command_channel) end

---Spawns an NPC (Mobius addSpawn) and returns it.
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

---A skill, as the attack and skill handlers get it.
---@class Skill
local Skill = {}

---@return integer
function Skill:id() end

---@return integer
function Skill:level() end

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

---Always false until parties exist.
---@return boolean
function Player:is_in_party() end

---Always nil until parties exist.
---@return nil
function Player:party() end

---The party's members; until parties exist, the player alone.
---@return Player[]
function Player:party_members() end

---Always nil until clans exist.
---@return nil
function Player:clan() end

---Always false until clans exist.
---@return boolean
function Player:is_clan_leader() end

---Always false until subclasses exist.
---@return boolean
function Player:is_sub_class_active() end

---Always false until noblesse exists.
---@return boolean
function Player:is_noble() end

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

---An NPC, as handlers receive it. Valid only while that handler runs.
---@class Npc
local Npc = {}

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

---Server settings quests read, with Mobius names: ALT_PARTY_RANGE,
---ALT_VILLAGES_REPEATABLE_QUEST_REWARD, DISABLE_TUTORIAL,
---RAID_MIN_RESPAWN_MULTIPLIER, RAID_MAX_RESPAWN_MULTIPLIER.
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
