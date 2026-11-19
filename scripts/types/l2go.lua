---@meta
-- L2Go script API definitions for the Lua Language Server (LuaLS).
--
-- They give autocomplete and type checks in editors. The server never runs
-- this file. A Go test (internal/scripting) fails when a function
-- registered in Go is missing here, or one here does not exist in Go.
--
-- Scripts run in a sandbox (Lua 5.1): the base library without dofile,
-- loadfile, load, loadstring, module and collectgarbage, plus string,
-- table and math. There is no os, io, debug or package library.
-- require("lib.name") loads lib/name.lua from the scripts folder, and
-- print() writes to the server log.

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
---in path order) and share one time limit (50 ms by default). The first
---rejection stops the event.
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
---`events.reject(reason)` to refuse it, or nothing to let it through.
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

---Rejects the event being handled. Only inside a handler, and only with a
---reason the event knows; anything else is an error with file and line.
---@param reason CharacterCreateReason
---@return Rejection
function events.reject(reason) end
