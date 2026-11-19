Lua scripts
====

Game content that changes often is written here in Lua. The game server loads every script at startup and can reload them without a restart. Networking, data loading and anything that runs every tick stay in Go.

The hooks today: character creation, chat, and NPC scripts (first talk, quest-link talk and dialog links). Quests, timers and commands come as the game grows.

Layout
----

```
events/     event rules, e.g. character_create.lua
npcs/       NPC scripts, one folder each with its pages, e.g. talking_island_guard/
lib/        shared helpers, loaded only through require("lib.name")
types/      API definitions for the editor (never run)
```

More folders (`quests/`, `commands/`) appear when those features exist. Every `.lua` file outside `lib/` and `types/` runs when the scripts load, in path order (`events/a.lua` before `events/b.lua` before `npcs/x.lua`). Files and folders starting with `.` are skipped, and so is everything that is not `.lua` (pages are read by the game server when it shows them).

Writing a script
----

A script registers handlers at its top level:

```lua
-- events/no_gm_names.lua
local reserved = { "gm", "admin" }

events.on_character_create(function(ev)
  local name = ev.name:lower()
  for _, word in ipairs(reserved) do
    if name:find(word, 1, true) then
      log.info("reserved name", "name", ev.name)
      return events.reject("incorrect_name")
    end
  end
end)
```

- Handlers run on the server's world loop, one at a time, so they see a consistent world. Keep them short.
- A handler returns `events.reject(reason)` to refuse, or nothing to allow (chat handlers return a new text or `false`; NPC handlers return a page; see below).
- All handlers of an event run in load order until one rejects or drops it.
- Handlers of one event share a time limit (`scripts.eventTimeout`, 50 ms by default); each NPC handler call has the same limit. An endless loop is stopped, logged with its file and line, and the event gets the default result (allowed, the message unchanged, the NPC's default dialog).
- A Lua error in a handler is logged with file and line; the other handlers still run.
- Keep state you need after a reload or restart out of Lua globals: every reload starts from a fresh Lua state. (Saved state comes with the quest API.)

API
----

The full API with types is in `types/l2go.lua`. A Go test checks that it lists every function and method the server registers.

### `events`

| Function | Meaning |
|---|---|
| `events.on_character_create(fn)` | `fn(ev)` runs before a new character is accepted, after the server's own checks (name length and letters, appearance, starting class, characters per account, name taken). |
| `events.on_chat(fn)` | `fn(ev)` runs before a chat message is sent, after the server's own checks. Return nothing to let it through, a string to replace its text, or `false` (or `""`) to drop it. |
| `events.reject(reason)` | Rejects the event being handled. Only inside a handler, and only with a reason the event knows; anything else is an error. |

`on_character_create` event table:

| Field | Type | Meaning |
|---|---|---|
| `account` | string | Account creating the character. |
| `name` | string | Name as typed. |
| `race` | integer | 0 human, 1 elf, 2 dark elf, 3 orc, 4 dwarf. Taken from the class template, not from the client. |
| `sex` | integer | 0 male, 1 female. |
| `class_id` | integer | Starting class. |
| `hair_style`, `hair_color`, `face` | integer | Appearance. |

Reasons (the client shows a fixed message for each; the server sends the matching Interlude `CharCreateFail` code):

| Reason | Code | Client message |
|---|---|---|
| `creation_failed` | `0x00` | Your character creation has failed. |
| `too_many_characters` | `0x01` | You cannot create another character. Please delete the existing character and try again. |
| `name_exists` | `0x02` | This name already exists. |
| `name_too_long` | `0x03` | Your title cannot exceed 16 characters in length. Please try again. |
| `incorrect_name` | `0x04` | Incorrect name. Please try again. |
| `creation_not_allowed` | `0x05` | Characters cannot be created from this server. |

`on_chat` event table:

| Field | Type | Meaning |
|---|---|---|
| `player` | Player | Who wrote it. |
| `type` | string | The channel, Mobius's `ChatType` in lower case: `general`, `shout`, `whisper`, `party`, `clan`, `trade`, `alliance`, `hero_voice`, ... (`unknown` for an ID the server does not know). |
| `text` | string | The text, with the replacements of earlier handlers. |
| `target` | string or nil | Whom a whisper goes to. |

```lua
-- events/chat_filter.lua
events.on_chat(function(ev)
  if ev.type == "trade" and ev.player:level() < 10 then
    return false -- no trade spam from fresh characters
  end
  return (ev.text:gsub("darn", "d**n"))
end)
```

A chat handler that fails is skipped: the message goes on with the text so far.

### NPC scripts: `script`

NPC scripts follow Mobius's `Script` and `Quest` classes, so Mobius's pages and links carry over. A script has a name, and its pages link to it with `bypass -h Script <name> <event>`.

```lua
-- npcs/talking_island_guard/guard.lua (shortened)
local guard = script.new("TalkingIslandGuard")

guard:on_first_talk({ 30733, 31032 }, function(npc, player)
  if player:level() >= 20 then
    return nil -- the guard's usual dialog
  end
  return "welcome.htm", { name = player:name(), level = player:level() }
end)

guard:on_event(function(event, npc, player)
  if event == "salute" then
    npc:say("Stay on the roads, " .. player:name() .. "!")
    return ""
  end
end)
```

`welcome.htm`, next to the script, links back to it:

```html
<a action="bypass -h Script TalkingIslandGuard salute">Salute the guard.</a>
```

| Function | Meaning |
|---|---|
| `script.new(name)` | Creates a script while the scripts load. Names are unique; letters, digits and underscores. |
| `s:on_first_talk(npc_ids, fn)` | `fn(npc, player)` answers the first click on these NPCs instead of their default dialog (Mobius `onFirstTalk`). Only one script may own an NPC's first talk; a second one is a load error with both places. |
| `s:on_talk(npc_ids, fn)` | `fn(npc, player)` answers the `Script <name>` link, the quest link, at these NPCs (Mobius `onTalk`). Several scripts may talk at one NPC. |
| `s:on_event(fn)` | `fn(event, npc, player)` handles the `Script <name> <event>` links (Mobius `onEvent`). `npc` may be nil. |
| `s:name()` | The script's name. |

`npc_ids` is one NPC template ID or a list of them. Handlers get their arguments in Mobius's order (`npc, player`).

What an NPC handler returns (Mobius's `showResult` rules, checked in this order):

| Return | Shown |
|---|---|
| a name ending in `.htm` or `.html` | That page file, relative to the folder of the script's file. |
| a string starting with `<html` | That HTML, as is. |
| other text | A message to the player. |
| `""` | Nothing: the handler did what it needed (for example `npc:say`). |
| `nil` | The default: the NPC's own dialog after a first talk, nothing after a talk or an event. (In Mobius a first-talk handler that wants the default dialog calls `npc.showChatWindow` and returns null.) |

A table as the second return value fills `%name%` placeholders: `return "welcome.htm", { name = player:name() }`. The server's own placeholders, like `%objectId%`, are filled too.

### `Player` and `Npc`

Handlers get players and NPCs as values with methods. They are valid only while that handler runs; keep `player:object_id()` instead of the player.

| Method | Returns |
|---|---|
| `player:object_id()`, `player:name()`, `player:level()` | integer, string, integer |
| `player:class_id()`, `player:race()` | integer (race: 0 human, 1 elf, 2 dark elf, 3 orc, 4 dwarf) |
| `npc:object_id()`, `npc:id()`, `npc:name()` | integer, integer (the template ID), string |
| `npc:say(text)` | Says `text` in normal NPC chat to the players around. |

### `html`

`html.replace(page, vars)` replaces every `%name%` in `page` with `vars.name`, in one pass.

### `log`

`log.debug`, `log.info`, `log.warn`, `log.error(msg, key, value, ...)` write to the server log. The key/value pairs become log fields, and every record gets the script's file and line. `print(...)` logs at info level.

### Sandbox

Scripts run Lua 5.1 (gopher-lua) with:

- the base library without `dofile`, `loadfile`, `load`, `loadstring`, `module` and `collectgarbage`;
- `string` (`string.rep` results are capped at 1 MiB), `table` and `math`;
- `require("lib.names")`, which loads `lib/names.lua` from the scripts folder once and returns its result. Only dotted names work; paths like `../x` or `/etc/x` are refused, and so is a symlink that leads out of the folder.

There is no `os`, `io`, `debug`, `package` or `coroutine`.

Reloading
----

A reload loads every script into a fresh Lua state next to the running one. Only if all of them load does the server swap it in, between two events on the world loop; otherwise the old scripts keep running and the errors are logged with file and line.

- **On save:** run the game server with `-dev-reload` or set `scripts.devReload` in the config. (The Docker Compose config has it on.)
- **By hand:** send `SIGHUP` to the game server: `kill -HUP <pid>`, or with Docker Compose `make reload-scripts` (`docker compose kill -s HUP game`).

Checking
----

```sh
make scripts-check          # or: go run ./cmd/l2go scripts check -dir scripts
```

This loads every script in the sandbox without starting a server and prints each problem as `file:line: message`, including two scripts claiming the first talk of one NPC. It exits non-zero on any error, and CI runs it. `go test ./...` also loads every script here.

Testing
----

Test scripts from Go with `internal/scripting/scripttest`: load the scripts, fire events, talk to scripted NPCs, and check the results, what NPCs said and what was logged.

```go
func TestNoGMNames(t *testing.T) {
	h := scripttest.Load(t, "../../scripts", "events/no_gm_names.lua")
	res := h.CharacterCreate(scripting.CharacterCreate{Account: "acc", Name: "TheGM"})
	if !res.Rejected || res.Reason != scripting.ReasonIncorrectName {
		t.Errorf("got %+v", res)
	}
}

func TestGuard(t *testing.T) {
	h := scripttest.Load(t, "../../scripts", "npcs/talking_island_guard/guard.lua")
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 5}
	guard := &scripttest.Npc{ObjectID: 2, ID: 30733, Name: "Guard"}
	reply, ok := h.FirstTalk(hero, guard)
	if !ok || reply.Content != "welcome.htm" {
		t.Errorf("got %+v", reply)
	}
	h.Event("TalkingIslandGuard", "salute", hero, guard)
	if said := h.Said(); len(said) != 1 {
		t.Errorf("said %+v", said)
	}
}
```

- `scripttest.Load(t, dir, files...)` loads only the given scripts (all of them when none are given); `lib/` is always available.
- `scripttest.LoadSource(t, map[string]string{...})` loads scripts written inline.
- `Fire`, `CharacterCreate`, `Chat`, `FirstTalk`, `Talk` and `Event` fail the test if a handler errors; `FireAllowErrors` returns the error instead.
- `scripttest.Player` and `scripttest.Npc` are fake players and NPCs; `Ref()` gives them to the engine directly.
- `Said()` returns what NPCs said, `Logs()` what the scripts logged, and `Reload()` reloads like the server.

The tests for the shipped scripts are in `internal/scripting/scripts_test.go`.

Editor support
----

`types/l2go.lua` is a [Lua Language Server](https://luals.github.io/) definitions file (`---@meta`), and `.luarc.json` in the repository root sets Lua 5.1 and hides the libraries the sandbox removes. With the LuaLS extension (VS Code, Neovim, ...) and the repository opened as the workspace, you get autocomplete and type checks for the whole API.
