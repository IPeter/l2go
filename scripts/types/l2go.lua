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
-- limit (50 ms by default). Player and Npc values are valid only while the
-- handler that received them runs.

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

---A script made by script.new.
---@class Script
local Script = {}

---Creates a script. Only while the scripts load; names are unique across
---all scripts and may contain letters, digits and underscores (1 to 64).
---@param name string as in `Script <name> <event>` links, e.g. "NewbieGuide"
---@return Script
function script.new(name) end

---Returns the script's name.
---@return string
function Script:name() end

---Answers the first click on these NPCs instead of their default dialog
---(Mobius onFirstTalk). Only one script may own the first talk of an NPC;
---a second registration is a load error.
---@param npc_ids integer|integer[] NPC template IDs
---@param fn fun(npc: Npc, player: Player): Reply, Placeholders?
function Script:on_first_talk(npc_ids, fn) end

---Answers the `Script <name>` link (the quest link) at these NPCs
---(Mobius onTalk). Several scripts may talk at one NPC; the player picks.
---@param npc_ids integer|integer[] NPC template IDs
---@param fn fun(npc: Npc, player: Player): Reply, Placeholders?
function Script:on_talk(npc_ids, fn) end

---Handles the `Script <name> <event>` links of the script's pages (Mobius
---onEvent). npc is the NPC the player talks to; it may be nil.
---@param fn fun(event: string, npc: Npc?, player: Player): Reply, Placeholders?
function Script:on_event(fn) end

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
