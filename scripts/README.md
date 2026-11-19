Lua scripts
====

Game content that changes often is written here in Lua. The game server loads every script at startup and can reload them without a restart. Networking, data loading and anything that runs every tick stay in Go.

Today there is one hook: character creation. More (NPC dialogs, quests, timers, commands) come as the game grows.

Layout
----

```
events/     event rules, e.g. character_create.lua
lib/        shared helpers, loaded only through require("lib.name")
types/      API definitions for the editor (never run)
```

More folders (`quests/`, `npcs/`, `commands/`) appear when those features exist. Every `.lua` file outside `lib/` and `types/` runs when the scripts load, in path order (`events/a.lua` before `events/b.lua` before `quests/x.lua`). Files and folders starting with `.` are skipped.

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

- A handler returns `events.reject(reason)` to refuse, or nothing to allow.
- All handlers of an event run in load order until one rejects.
- Handlers of one event share a time limit (`scripts.eventTimeout`, 50 ms by default). An endless loop is stopped, logged with its file and line, and the event gets the default result (allowed).
- A Lua error in a handler is logged with file and line; the other handlers still run.
- Keep state you need after a reload or restart out of Lua globals: every reload starts from a fresh Lua state. (Saved state comes with the quest API.)

API
----

The full API with types is in `types/l2go.lua`. A Go test checks that it lists every function the server registers.

### `events`

| Function | Meaning |
|---|---|
| `events.on_character_create(fn)` | `fn(ev)` runs before a new character is accepted, after the server's own checks (name length and letters, appearance, starting class, characters per account, name taken). |
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

A reload builds a fresh Lua state and loads every script. Only if all of them load does it replace the running scripts; otherwise the old scripts keep running and the errors are logged with file and line.

- **On save:** run the game server with `-dev-reload` or set `scripts.devReload` in the config. (The Docker Compose config has it on.)
- **By hand:** send `SIGHUP` to the game server: `kill -HUP <pid>`, or with Docker Compose `make reload-scripts` (`docker compose kill -s HUP game`).

Checking
----

```sh
make scripts-check          # or: go run ./cmd/l2go scripts check -dir scripts
```

This loads every script in the sandbox without starting a server and prints each problem as `file:line: message`. It exits non-zero on any error, and CI runs it. `go test ./...` also loads every script here.

Testing
----

Test scripts from Go with `internal/scripting/scripttest`: load the scripts, fire events, check the results and what was logged.

```go
func TestNoGMNames(t *testing.T) {
	h := scripttest.Load(t, "../../scripts", "events/no_gm_names.lua")
	res := h.CharacterCreate(scripting.CharacterCreate{Account: "acc", Name: "TheGM"})
	if !res.Rejected || res.Reason != scripting.ReasonIncorrectName {
		t.Errorf("got %+v", res)
	}
}
```

- `scripttest.Load(t, dir, files...)` loads only the given scripts (all of them when none are given); `lib/` is always available.
- `scripttest.LoadSource(t, map[string]string{...})` loads scripts written inline.
- `Fire` / `CharacterCreate` fail the test if a handler errors; `FireAllowErrors` returns the error instead.
- `Logs()` returns what the scripts logged.

The tests for the default scripts are in `internal/scripting/scripts_test.go`.

Editor support
----

`types/l2go.lua` is a [Lua Language Server](https://luals.github.io/) definitions file (`---@meta`), and `.luarc.json` in the repository root sets Lua 5.1 and hides the libraries the sandbox removes. With the LuaLS extension (VS Code, Neovim, ...) and the repository opened as the workspace, you get autocomplete and type checks for the whole API.
