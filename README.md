L2Go
====

L2Go is an open-source Lineage ][ server emulator written in Go. It speaks the **Interlude** protocol: game client protocol **746** (login protocol revision `0xc621`).

Status
----

**In development.** A real Interlude client can log in, pick a server, see its characters and create new ones. The world itself (entering the game) is not there yet.

What works, the parity checklist from the move to Interlude (each item is covered by the end-to-end test in `internal/e2e`):

| # | Feature |
|---|---|
| F1 | Login server `Init`: session ID, scrambled RSA key, GameGuard constants, a per-session Blowfish key (static key plus XOR pass for the first packet). |
| F2 | GameGuard handshake (`AuthGameGuard` → `GGAuth`). L2Go does not run GameGuard itself. |
| F3 | Login with RSA-encrypted credentials (128-byte and the newer 256-byte block). Automatic account creation, bcrypt passwords, case-insensitive account names, access levels: `LoginOk`, `LoginFail`, `AccountKicked` for banned accounts. |
| F4 | Server list in the Interlude format with each game server's live status (up/down) and player count from the login ↔ game server link. Internal and external IPs per client network; "testing" servers only for staff. |
| F5 | Server selection: session keys checked, `PlayOk` with the play keys or `PlayFail`. |
| F6 | Game servers register with the login server over the link with an HMAC-SHA256 challenge on a shared secret, report their player count and reconnect with backoff. |
| F7 | `ProtocolVersion` check against the allowed versions in the config (default 746). Version -2 is a ping. |
| F8 | `KeyPacket` with a random per-session key; 16-byte XOR game packet encryption. |
| F9 | `AuthLogin`: the game server checks the session keys with the login server (each set works once), then sends `CharSelectionInfo` with the account's saved characters and their equipment. |
| F10 | Character creation screen: `NewCharacterSuccess` with the 9 starting classes and their base stats. |
| F11 | `CharacterCreate`: the L2J Mobius rules (name, appearance, class, characters per account, unique name), then the Lua hook; `CharCreateOk` or `CharCreateFail` with the Interlude reason codes. The character is saved in MongoDB with its start location, level 1 HP/MP/CP and starting equipment, and the updated list follows. |

Also: Lua scripting for game rules, with live reload.

What does not work yet (the client shows the buttons, the server logs the packets and ignores them):

- Deleting and restoring characters.
- Selecting a character and entering the world, and everything after that.

Requirements
----

- Go 1.27 or newer.
- Docker with Docker Compose, for MongoDB and for running the whole stack. Both servers store their data in MongoDB.
- To play: an Interlude client (protocol 746).

Run with Docker Compose
----

```sh
make up        # docker compose up --build: MongoDB, the login server and game server 1
make logs      # follow the logs (docker compose logs -f)
make down      # stop; accounts and characters stay in the mongo-data volume

L2GO_LOG_LEVEL=debug make up   # log every packet as hex
```

Ports on the host:

| Port | What |
|---|---|
| `2106` | Login server, for clients. |
| `7777` | Game server 1, for clients. |
| `27017` | MongoDB, published on `127.0.0.1` only (for `make integration` and local tools). |

The login ↔ game server link (port 9413) stays on the compose network. The servers read `deploy/compose/server.json`, which uses the compose service names (`mongo`, `login`) as host names.

Inside Docker the login server sees clients coming from the Docker gateway, not `127.0.0.1`, so they get the game server's `externalIP`. In the compose config both `internalIP` and `externalIP` are `127.0.0.1`, which works for a client on the same machine. For players on other machines, set `externalIP` to the host's address.

### Connecting an Interlude client

1. Point the client at the login server. The client reads the login server address from `ServerAddr` in `system/l2.ini`. Either set it to `127.0.0.1` (the retail `l2.ini` is encrypted, so this needs an `l2.ini` editor or a patched `system` folder), or keep the host name and map it in your hosts file, for example `127.0.0.1 l2authd.lineage2.com`.
2. Use a `system` folder with GameGuard disabled. L2Go answers the GameGuard handshake but does not run or check GameGuard.
3. Log in with any user name and password. The first login creates the account (`autoCreate`). Account names are case-insensitive.
4. The server list shows server 1 as up and sends the client to `127.0.0.1:7777`.

To make an account an admin (sees and joins testing servers) or to ban it:

```sh
docker compose exec mongo mongosh l2go-login --eval 'db.accounts.updateOne({username: "name"}, {$set: {access_level: 1}})'   # -1 bans
```

### Smoke test

With the stack running, `make smoke` walks a fake client through the whole flow (login, server list, character list, creation, logging in again, wrong password) with a new random account and character each time. `L2GO_LOGIN_ADDR` (default `127.0.0.1:2106`) and `L2GO_GAME_ADDR` (default `127.0.0.1:7777`) point it at another stack.

Build and run without Docker
----

```sh
docker compose up -d mongo     # MongoDB only (or bring your own)
make build                     # or: go build -o bin/l2go ./cmd/l2go

bin/l2go login                 # login server
bin/l2go game -id 1            # game server 1
bin/l2go scripts check         # load every Lua script without starting a server
```

Start the login server first. A game server that cannot reach it keeps retrying the link, and the server list shows it as down until it registers. A link error that retrying cannot fix (wrong `secret`, unknown server ID) stops the game server.

Run the game server from the repository root (or set `dataDir` and `scripts.dir`), so it finds the `data/` and `scripts/` folders.

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `-config path` (login, game) | `~/.l2go/config/server.json` | Config file. If it does not exist, built-in defaults are used. |
| `-log-level level` | `info` | `debug`, `info`, `warn` or `error`. `debug` logs every packet as hex. |
| `-id N` (game) | `1` | Which game server from the config to run. |
| `-dev-reload` (game) | off | Reload the Lua scripts when a file changes (same as `scripts.devReload`). |
| `-dir path` (scripts check) | `scripts` | Scripts folder to check. |

Stop a server with Ctrl+C (or SIGTERM). It closes all connections and exits cleanly. `SIGHUP` makes the game server reload its Lua scripts.

Configuration
----

One JSON file configures the login server and all game servers; both read the same file. An example is in `extra/default_config_file/server.json` (copy it to `~/.l2go/config/server.json` and edit it), and the Docker Compose one is `deploy/compose/server.json`. Unknown keys and invalid values are reported at startup. A `_comment` key with any text is allowed in each object.

The "Default" column is what a key gets when it is left out of the file. Keys without a default are empty, zero or `false` when left out. When there is no config file at all, the built-in defaults are the values in `extra/default_config_file/server.json`.

`loginserver`:

| Key | Default | Meaning |
|---|---|---|
| `host` | | Where game servers reach the login server. Required. |
| `clientListen` | `:2106` | Listen address for clients. |
| `gameServerListen` | `:9413` | Listen address for the game server link. Game servers dial `host` on this port. |
| `autoCreate` | | `true` creates an account on the first login with an unknown user name. |
| `internalNetworks` | `["127.0.0.0/8"]` | Clients from these networks (CIDR) get a game server's `internalIP`, everyone else its `externalIP`. |
| `database` | | MongoDB for accounts: `name`, plus either `uri` (`mongodb://` or `mongodb+srv://`) or `host`, `port`, `user`, `password`. |
| `rsaKeyPairs` | `10` | RSA key pairs generated at startup for the login handshake (1-100). Each connection picks one at random. |
| `sessionTTL` | `"2m"` | How long the session keys of a login stay valid after they were issued or last used (server list, server selection, the game server's check). |

`gameservers` (a list, one entry per game server):

| Key | Default | Meaning |
|---|---|---|
| `id` | | Server ID, 1-255. Shown in the server list and used by `l2go game -id`. |
| `name` | | Server name, for the logs (the client shows names from its own files). |
| `secret` | | Shared secret of the login server link. Required. The login server checks it with an HMAC challenge; it never goes over the wire. Change `CHANGE_ME_PLEASE`. |
| `internalIP`, `externalIP` | | IPv4 addresses announced to clients, see `internalNetworks`. |
| `port` | | Game port: the server listens on it and announces it. |
| `database` | | MongoDB for characters and the object ID counter, same format as above. |
| `allowedProtocols` | `[746]` | Client protocol versions the server accepts. Others get `KeyPacket` with result 0 and are disconnected. |
| `dataDir` | `"data"` | Game data folder (see `data/README.md`), relative to the working directory. |
| `maxCharactersPerAccount` | | Characters per account; `0` means no limit. The example configs use `7`, like L2J Mobius. |
| `options.maxPlayers` | | Player limit shown in the server list; `0` means no limit. A full server only takes accounts above player level. |
| `options.testing` | | `true` hides the server from players: only accounts above player level see and join it. |
| `scripts.dir` | `"scripts"` | Lua scripts folder, relative to the working directory. |
| `scripts.devReload` | | `true` reloads the scripts when a file changes. |
| `scripts.eventTimeout` | `"50ms"` | Time limit for the script handlers of one event. |

Lua scripting
----

Game rules that admins want to change are Lua scripts in `scripts/`, run by the game server in a sandbox (Lua 5.1, no file or OS access, a time limit per event). Today's hook is character creation: `scripts/events/character_create.lua` refuses names with forbidden words (the L2J Mobius `ForbiddenNames` list) and is meant to be edited.

```lua
events.on_character_create(function(ev)
  if ev.name:lower():find("admin", 1, true) then
    return events.reject("incorrect_name")
  end
end)
```

- Reload without a restart, in either of two ways. The game server watches the folder when `scripts.devReload` or `-dev-reload` is on (it is on in Docker Compose, which mounts `./scripts`), so saving a file is enough. Or send `SIGHUP`: `docker compose kill -s HUP game` (`make reload-scripts`), or `kill -HUP <pid>`. A broken script never replaces working ones; the errors are logged with file and line.
- `make scripts-check` loads every script and reports problems as `file:line: message`. CI runs it, and `go test ./...` loads every script too.
- `scripts/types/l2go.lua` gives editor autocomplete through the Lua Language Server.

See [`scripts/README.md`](scripts/README.md) for the API, the sandbox and how to test scripts with `internal/scripting/scripttest`.

Game data
----

`data/` holds the static game data the game server loads at startup: the 9 starting class templates, their starting equipment and the items it uses. The JSON files are generated by `cmd/datagen` from the L2J Mobius Interlude datapack and committed, so the server does not need Mobius at runtime:

```sh
go run ./cmd/datagen -mobius L2J_Mobius/L2J_Mobius_CT_0_Interlude -out data
```

See [`data/README.md`](data/README.md) for the files, their source and license.

Development
----

```sh
make test           # unit and end-to-end tests, no database needed
make race           # with the race detector, random order
make cover          # coverage report
make fuzz           # every fuzz target for 60s (FUZZTIME=10s make fuzz)
make lint           # golangci-lint
make vuln           # govulncheck
make scripts-check  # load every Lua script in scripts/
make integration    # MongoDB tests; starts the compose mongo service
make smoke          # full client flow against the running stack (make up first)
```

`go test ./...` includes `internal/e2e`, which starts the real login server and game server in one process (in-memory storage, the real `data/` and `scripts/`) and walks the fake Interlude client in `internal/testclient` through the whole checklist above. `make smoke` runs the same flow against a running stack.

Lint and vulnerability tools are pinned in `tools/go.mod` and run with `go tool -modfile=tools/go.mod`, so nothing has to be installed globally.

Layout:

```
cmd/l2go/                       command line: login, game, scripts check
cmd/datagen/                    converts the L2J Mobius datapack into data/
internal/config/                config file loading and validation
internal/packets/               framing, packet Reader/Writer, UTF-16 strings
internal/crypt/blowfish/        Lineage II Blowfish (little-endian x/crypto/blowfish)
internal/crypt/logincrypt/      login packet encryption: static key, XOR pass, session key, checksum
internal/crypt/rsakey/          RSA key pool, modulus scrambling, raw RSA, credentials block
internal/crypt/gamecrypt/       game packet XOR cipher
internal/session/               session keys from login to game server
internal/link/                  login <-> game server link
internal/data/                  game data loader
internal/netutil/               connection registry, accept loop
internal/storage/               data models and store interfaces
internal/storage/memstore/      in-memory store (tests)
internal/storage/mongostore/    MongoDB store
internal/storage/storagetest/   conformance tests for every store
internal/loginserver/           login server
internal/gameserver/            game server
internal/scripting/             Lua engine: sandbox, loader, events, reload
internal/scripting/scripttest/  test harness for Lua scripts
internal/testclient/            fake Interlude client for end-to-end tests
internal/e2e/                   full-flow test of both servers, and the smoke test
scripts/                        Lua scripts (see scripts/README.md)
data/                           game data (see data/README.md)
deploy/compose/                 config for docker compose
extra/default_config_file/      example config
tools/                          pinned lint and vulnerability tools
```

Disclaimer
----

This work is for nonprofit educational purposes only. No copyright infringement is intended.

Using this work to run a private server, is at your own risk.

NCSOFT©, the Interlocking NC Logo, PLAYNC, Lineage, Team Lineage II, Lineage II, Goddess of Destruction, Valiance, Truly Free, L2 Store, L2 Galleria, NCoin, Path to Awakening, and all associated logos and designs are trademarks or registered trademarks of NCSOFT Corporation.

License
----

L2Go is under the GNU GPL v3.0 license (see `LICENSE`).

Credits: [L2J Mobius](https://gitlab.com/MobiusDevelopment/L2J_Mobius) is the reference for the Interlude protocol (packet layouts, encryption, game rules) and the source of the game data in `data/`.

Made with love by Frostwind <hi@frostwind.me>
