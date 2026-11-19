L2Go
====

L2Go is an open-source Lineage ][ server emulator written in Go.

Status
----

**In development.** L2Go currently speaks the **C1 (Harbingers of War)** protocol, game protocol 419. The move to Interlude (protocol 746) is planned and will replace C1.

What works with a C1 client:

- Login with automatic account creation (bcrypt passwords, MongoDB storage), access levels, banned accounts.
- Server list and server selection (`PlayOk` / `PlayFail`), including "testing" servers for staff only.
- Game server handshake (`ProtocolVersion`, `CryptInit`), an empty character list, the character creation screen, `CharCreateOk` and `CharCreateFail`.
- Lua scripting: character creation rules live in `scripts/` and reload without a restart.

What does not work yet:

- Characters are not saved; the list is always empty.
- No world: selecting a character and entering the game are not implemented.
- The login ↔ game server link only registers the game server. Session keys are not checked between the two, and the server list does not show live status or player counts.

Requirements
----

- Go 1.27 or newer.
- MongoDB for the login server. The easiest way is Docker (`docker compose`), see below.

Build and run
----

```sh
make build                     # or: go build -o bin/l2go ./cmd/l2go

bin/l2go login                 # login server
bin/l2go game -id 1            # game server 1 (start the login server first)
bin/l2go scripts check         # load every Lua script without starting a server
```

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `-config path` (login, game) | `~/.l2go/config/server.json` | Config file. If it does not exist, built-in defaults are used. |
| `-log-level level` | `info` | `debug`, `info`, `warn` or `error`. `debug` logs every packet as hex. |
| `-id N` (game) | `1` | Which game server from the config to run. |
| `-dev-reload` (game) | off | Reload the Lua scripts when a file changes (same as `scripts.devReload`). |
| `-dir path` (scripts check) | `scripts` | Scripts folder to check. |

Run the game server from the repository root (or set `scripts.dir`), so it finds the `scripts/` folder.

Stop a server with Ctrl+C (or SIGTERM). It closes all connections and exits cleanly. `SIGHUP` makes the game server reload its Lua scripts.

To run against a local MongoDB without the rest of the stack: `docker compose up -d mongo`.

Run everything with Docker Compose
----

```sh
make up        # docker compose up --build: MongoDB, login server, game server 1
make logs      # follow the logs
make down      # stop (accounts stay in the mongo-data volume)

L2GO_LOG_LEVEL=debug make up   # with packet dumps
```

The client connects to `127.0.0.1:2106` (login) and the server list points it to `127.0.0.1:7777` (game). The servers read `deploy/compose/server.json`, which uses the compose service names (`mongo`, `login`) as host names.

Inside Docker, the login server sees clients coming from the Docker gateway address, not `127.0.0.1`. So in the compose config both `internalIP` and `externalIP` of game server 1 are `127.0.0.1`. For players on other machines, set `externalIP` to the host's address.

Configuration
----

One JSON file configures the login server and all game servers. The default is in `extra/default_config_file/server.json`; copy it to `~/.l2go/config/server.json` and edit it. Unknown keys and invalid values are reported at startup. A `_comment` key with any text is allowed in each object.

`loginserver`:

| Key | Default | Meaning |
|---|---|---|
| `host` | `127.0.0.1` | Where game servers reach the login server. |
| `clientListen` | `:2106` | Listen address for clients. |
| `gameServerListen` | `:9413` | Listen address for game servers. Game servers dial `host` on this port. |
| `autoCreate` | `true` | Create an account on the first login with an unknown username. |
| `internalNetworks` | `["127.0.0.0/8"]` | Clients from these networks (CIDR) get a game server's `internalIP`, everyone else its `externalIP`. |
| `database` | | MongoDB: `name`, plus either `uri` or `host`, `port`, `user`, `password`. |

`gameservers` (a list):

| Key | Meaning |
|---|---|
| `id` | Server ID, 1-255. Shown in the server list and used by `l2go game -id`. |
| `name` | Server name (for the logs; the client shows names from its own files). |
| `secret` | Shared secret for the login server link. Not used yet. |
| `internalIP`, `externalIP` | IPv4 addresses announced to clients, see `internalNetworks`. |
| `port` | Game port: the server listens on it and announces it. |
| `database` | MongoDB settings, same format as above. Not used yet. |
| `options.maxPlayers` | Shown in the server list. |
| `options.testing` | Only accounts above player access level may join. |
| `scripts.dir` | Lua scripts folder, relative to the working directory. Default `scripts`. |
| `scripts.devReload` | Reload the scripts when a file changes. Default `false`. |
| `scripts.eventTimeout` | Time limit for the script handlers of one event, e.g. `"50ms"` (the default). |

Lua scripting
----

Game rules that admins want to change are Lua scripts in `scripts/`, run by the game server in a sandbox (Lua 5.1, no file or OS access, a time limit per event). Today's hook is character creation; `scripts/events/character_create.lua` refuses names with forbidden words (the L2J Mobius `ForbiddenNames` list) and is meant to be edited.

```lua
events.on_character_create(function(ev)
  if ev.name:lower():find("admin", 1, true) then
    return events.reject("incorrect_name")
  end
end)
```

- Reload without a restart: save with `-dev-reload` (on in Docker Compose), or send `SIGHUP` (`make reload-scripts` for Docker Compose). A broken script never replaces working ones; the errors are logged with file and line.
- `make scripts-check` loads every script and reports problems as `file:line: message`. CI runs it, and `go test ./...` loads every script too.
- `scripts/types/l2go.lua` gives editor autocomplete through the Lua Language Server.

See `scripts/README.md` for the API, the sandbox and how to test scripts with `internal/scripting/scripttest`.

Development
----

```sh
make test         # unit tests, no database needed
make race         # with the race detector, random order
make cover        # coverage report
make fuzz         # every fuzz target for 60s (FUZZTIME=10s make fuzz)
make lint           # golangci-lint
make vuln           # govulncheck
make scripts-check  # load every Lua script in scripts/
make integration    # MongoDB tests; starts the compose mongo service
```

Lint and vulnerability tools are pinned in `tools/go.mod` and run with `go tool -modfile=tools/go.mod`, so nothing has to be installed globally.

Layout:

```
cmd/l2go/                     command line (login, game)
internal/config/              config file loading and validation
internal/packets/             framing, packet Reader/Writer, UTF-16 strings
internal/crypt/blowfish/      Lineage II Blowfish (byte-swapped x/crypto/blowfish)
internal/crypt/checksum/      login packet checksum
internal/crypt/xor/           C1 game packet XOR cipher
internal/netutil/             connection registry, accept loop
internal/storage/             data models and store interfaces
internal/storage/memstore/    in-memory store (tests)
internal/storage/mongostore/  MongoDB store
internal/loginserver/         login server
internal/gameserver/          game server
internal/scripting/           Lua engine: sandbox, loader, events, reload
internal/scripting/scripttest/  test harness for Lua scripts
scripts/                      Lua scripts (see scripts/README.md)
deploy/compose/               config for docker compose
```

Disclaimer
----

This work is for nonprofit educational purposes only. No copyright infringement is intended.

Using this work to run a private server, is at your own risk.

NCSOFT©, the Interlocking NC Logo, PLAYNC, Lineage, Team Lineage II, Lineage II, Goddess of Destruction, Valiance, Truly Free, L2 Store, L2 Galleria, NCoin, Path to Awakening, and all associated logos and designs are trademarks or registered trademarks of NCSOFT Corporation.

License
----

L2Go is under the GNU GPL v3.0 license.

Made with love by Frostwind <hi@frostwind.me>
