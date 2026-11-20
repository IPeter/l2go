L2Go
====

L2Go is an open-source Lineage ][ server emulator written in Go. It speaks the **Interlude** protocol: game client protocol **746** (login protocol revision `0xc621`).

Status
----

**In development.** A real Interlude client can log in, create a character and enter the world: walk around, see other players, chat, target and talk to NPCs, travel with gatekeepers, and come back where it left. There is no fighting and no trading yet.

Logging in and the character screen, the parity checklist from the move to Interlude (each item is covered by the end-to-end test in `internal/e2e`):

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

In the world (Milestones 3 and 4, ported from L2J Mobius):

| Feature | What works |
|---|---|
| Entering the world | Selecting a character and Mobius's enter-world sequence. The character stands where it left the game, with its equipment on. The character window (Alt+T) shows the stats Mobius computes from the class, level, base stats and equipped items (no skills or buffs yet). |
| Movement | Walking by mouse click or keyboard with Mobius's `ValidatePosition` policy, sitting and standing, walking or running, and teleporting with the loading screen. With geodata (`geodata.dir` pointing to L2J `.l2j` tiles), heights follow the ground, a mouse walk stops at a wall and a click on a closed cell is refused. |
| Seeing others | Players and NPCs appear and disappear as in Mobius: the world is cut into regions of 2048×2048 units with 16 height layers, and everyone sees the 3×3×3 regions around them. |
| Chat | General chat (heard within 1,250 units), shout (`!`) and trade (`+`, from level 20) heard in the speaker's map region by default, or in the whole world, from GMs only, or not at all (`chat.shout`, `chat.trade`), and whispers (`"name text`) with Mobius's messages for players who are not online. A Lua handler (`events.on_chat`) can change or drop messages. |
| Restart and logout | Back to the character screen, or out of the game. Position, HP, MP and CP are saved then, every 15 minutes, and when the server stops. Selecting a character while the same account is in the game disconnects the other session first. |
| Deleting and restoring characters | A deleted character stays in the list with its timer for 7 days (`world.deleteCharAfterDays`) and can be restored until then. |
| NPCs and spawns | All NPCs and monsters of the Mobius spawn lists, with their names and titles. Respawn timers; NPCs that exist only by day or by night follow the game clock. NPCs stand still for now: they do not walk around or fight. |
| Targeting | Clicking an NPC or a player targets it, with the level-difference colour and its HP bar. Social actions (greet, bow, dance, ...) are shown to the players around. |
| Dialogs | Clicking the target again walks up to the NPC and opens its dialog, with Mobius's page rules and placeholders. Links that were not on the last page, and clicks too far from the NPC, are refused. Shops, warehouses and trainers answer with a "not available yet" page, the quest link with Mobius's "no quests" page. The pages come from `make data` (see [Game data](#game-data)). |
| Gatekeepers | Teleport lists with their prices: free up to level 40 (`npcs.maxFreeTeleportLevel`), otherwise paid in adena. |
| Zones and map regions | Every Mobius zone is loaded, and the server knows which zones each player is in. In water the character swims, at swim speed. `/loc` names the nearest town, and the map window (Alt+M) opens. |
| Lua NPC scripts | Scripts answer the first click on an NPC, its quest link and their own dialog links with a page, inline HTML or a message, and can make the NPC speak. Like every Lua script, they reload without a restart (see [Lua scripting](#lua-scripting)). |
| GM commands | For characters with access level 1 or more ([how](#connecting-an-interlude-client)): `//teleport x y z` (also `//move_to`), `//loc`, `//gmspeed n`, `//heal [name or radius]`, `//invis` and `//vis`, `//online`, `//spawn <npcId or name> [count]`, `//delete [range]` (the targeted NPC, or the NPCs around, with their spawns), `//respawn`, `//npcinfo`, and shift-click on an NPC for its info page. |

What does not work yet:

- Fighting: monsters can be targeted but not attacked. No damage, death, experience or drops (Milestone 6).
- Items and shops: no buying, selling, equipping, using or dropping items, and no warehouses (Milestone 5).
- Skills, quests, parties, clans and the rest of the game, which come after those.
- Pathfinding: with geodata a mouse walk stops at the first obstacle instead of going around it. Without geodata (the default) heights come from the client and the spawn data, and every place is walkable.

Requirements
----

- Go 1.27 or newer.
- Docker with Docker Compose, for MongoDB and for running the whole stack. Both servers store their data in MongoDB.
- To play: an Interlude client (protocol 746).
- For the NPC dialogs: a checkout of [L2J Mobius](https://gitlab.com/MobiusDevelopment/L2J_Mobius) in `L2J_Mobius/`, from which `make data` copies the pages (see [Game data](#game-data)).

Run with Docker Compose
----

```sh
make data      # once, for the NPC dialogs: copies the pages from L2J_Mobius/ into data/html
make up        # docker compose up --build: MongoDB, the login server and game server 1
make up PUBLIC_IP=203.0.113.7   # announce the game server at another address
make logs      # follow the logs (docker compose logs -f)
make down      # stop; accounts and characters stay in the mongo-data volume

L2GO_LOG_LEVEL=debug make up   # log every packet as hex
```

The image contains the game data: `data/`, and the dialog pages in `data/html/` when that folder exists. `make up` rebuilds the image, so it picks up what `make data` wrote. Without the pages, NPCs have no dialogs to show. When the stack stops, the game server first saves every player in the world.

Ports on the host:

| Port | What |
|---|---|
| `2106` | Login server, for clients. |
| `7777` | Game server 1, for clients. |
| `27017` | MongoDB, published on `127.0.0.1` only (for `make integration` and local tools). |

The login ↔ game server link (port 9413) stays on the compose network. The servers read `deploy/compose/server.json`, which uses the compose service names (`mongo`, `login`) as host names.

Clients can connect from any machine. After login, the server list tells the client which IPv4 address to use for the game server, so that address must be reachable from every client. `make up` detects this machine's LAN IP and passes it as `L2GO_PUBLIC_IP`, which `deploy/compose/server.json` uses for `internalIP` and `externalIP` (config values can use `${NAME}` and `${NAME:-default}`). Every machine on the network, and this machine itself, can reach that address. With plain `docker compose up` the address is `127.0.0.1`, which only works for a client on this machine. For players on the internet, use `make up PUBLIC_IP=<public IP>` and forward ports 2106 and 7777 on the router.

### Connecting an Interlude client

1. Point the client at the login server without changing the client: map the login host name from `ServerAddr` in the client's `system/l2.ini` to the server machine in the hosts file of the client machine (on Windows `C:\Windows\System32\drivers\etc\hosts`), for example `192.168.1.87 l2authd.lineage2.com`. The host name is the usual retail one; check your client if it differs.
2. L2Go answers the GameGuard step of the login handshake but does not run GameGuard itself.
3. Log in with any user name and password. The first login creates the account (`autoCreate`). Account names are case-insensitive.
4. The server list shows server 1 as up and sends the client to the announced address (the server machine's LAN IP with `make up`), port 7777.
5. If the game server log shows a wrong protocol version, add the client's version to `allowedProtocols` (Interlude clients use 746; some builds use 737, 740 or 744).

To make an account an admin (sees and joins testing servers) or to ban it:

```sh
docker compose exec mongo mongosh l2go-login --eval 'db.accounts.updateOne({username: "name"}, {$set: {access_level: 1}})'   # -1 bans
```

GM rights belong to the character, as in L2J Mobius. To make a character a GM (the `//` commands), set its access level while it is not in the game, or its next save writes the old value back. `name_lower` is the character name in lower case:

```sh
docker compose exec mongo mongosh l2go-server --eval 'db.characters.updateOne({name_lower: "name"}, {$set: {access_level: 1}})'
```

### Smoke test

With the stack running, `make smoke` walks two fake clients through the whole flow with new random accounts and characters each time: the character screen (login, server list, character list, creation, logging in again, wrong password), then the world (walking, seeing each other, chat, NPC dialogs, a Lua NPC, restart, delete and restore, a second login). The steps that need a GM or adena skip, since the test cannot reach into the server's store. Like a real client, it connects to the game server at the address the server list announces. `L2GO_LOGIN_ADDR` (default `127.0.0.1:2106`) points it at another login server; `L2GO_GAME_ADDR` overrides the announced game server address.

Build and run without Docker
----

```sh
docker compose up -d mongo     # MongoDB only (or bring your own)
make build                     # or: go build -o bin/l2go ./cmd/l2go
make data                      # once, for the NPC dialogs (needs L2J_Mobius/, see Game data)

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

Stop a server with Ctrl+C (or SIGTERM). It closes all connections and exits cleanly; the game server saves every player in the world first. `SIGHUP` makes the game server reload its Lua scripts.

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
| `dataDir` | `"data"` | Game data folder (see `data/README.md`), relative to the working directory. The NPC dialog pages are in its `html` subfolder. |
| `maxCharactersPerAccount` | | Characters per account; `0` means no limit. The example configs use `7`, like L2J Mobius. |
| `options.maxPlayers` | | Player limit shown in the server list; `0` means no limit. A full server only takes accounts above player level. |
| `options.testing` | | `true` hides the server from players: only accounts above player level see and join it. |
| `scripts.dir` | `"scripts"` | Lua scripts folder, relative to the working directory. |
| `scripts.devReload` | | `true` reloads the scripts when a file changes. |
| `scripts.eventTimeout` | `"50ms"` | Time limit for the script handlers of one event. |

The game rules of a game server are in more sections: `world`, `chat`, `npcs`, `geodata`, `items`, `economy`, `rates`, `combat` and `pvp`. A section or key left out keeps its default, the L2J Mobius value. Most keys are L2J Mobius settings with the same name in camel case; the tables name the Mobius key where it differs. Durations are strings such as `"1s"` or `"10m"`. Some keys are read only once the feature using them is built.

`world`:

| Key | Default | Meaning |
|---|---|---|
| `tickInterval` | `"100ms"` | Tick of the world loop (the Mobius game clock tick). |
| `autosaveInterval` | `"15m"` | How often the players in the world are saved. They are also saved when they leave the world and when the server stops; `"0s"` saves only then. |
| `deleteCharAfterDays` | `7` | How long a deleted character can be restored (0-3650 days). After that it is removed for good the next time its account's character list is shown. `0` deletes at once. |
| `maxRunSpeed` | `250` | Highest run speed of players. |
| `spawnProtection` | `"10m"` | How long a player that entered the world is safe from aggressive monsters. Its first click, attack or walk ends it, with a message outside peace zones. `"0s"` turns it off. |
| `gridsAlwaysOn` | `false` | `true` keeps every world region active, so monsters think even where no player is. |
| `gridTurnOnTime`, `gridTurnOffTime` | `"1s"`, `"90s"` | Delay before the regions next to a player become active, and before regions with no player around become inactive (Mobius `GridNeighborTurnOnTime`/`TurnOffTime`). |

`chat`:

| Key | Default | Meaning |
|---|---|---|
| `generalRange` | `1250` | How far normal chat is heard, in game units. |
| `shout` | `"on"` | Who hears shout chat (`!`), Mobius `GlobalChat`: `"on"` the players in the speaker's map region and instance (Mobius's default), `"global"` every player in the game (players below GM level may then speak there once every 500 ms), `"gm"` like `"on"` but only GMs are heard, `"off"` nobody. |
| `trade` | `"on"` | The same for trade chat (`+`), Mobius `TradeChat`. Only characters of level 20 and up can use it. |
| `maxLength` | `105` | Longest message in characters (500 when it links an item). Longer ones are refused, except from GMs. |

`npcs`:

| Key | Default | Meaning |
|---|---|---|
| `spawnCustom` | `false` | `true` also spawns the custom spawn lists (Mobius `spawns/custom/`). The Interlude datapack has none at the moment. |
| `maxFreeTeleportLevel` | `40` | Players up to this level use the normal gatekeeper lists for free. |
| `interactionRange` | `250` | How close a player must be to an NPC to talk to it, in game units. |
| `defaultCorpseTime`, `spoiledCorpseExtendTime` | `"7s"`, `"10s"` | How long a dead NPC's corpse stays, and how much longer when it was spoiled. |
| `maxAggroRange`, `maxDriftRange` | `450`, `300` | Cap of the monsters' aggro range, and how far they wander from their spawn. |
| `mobAggroInPeaceZone` | `true` | Aggressive monsters also attack players in peace zones (Mobius `AltMobAgroInPeaceZone`). |
| `aggroDistanceCheck`, `aggroDistanceCheckRange` | `false`, `2000` | `true` makes monsters give up when pulled farther than the range from their spawn (Mobius `AggroDistanceCheckEnabled`). |
| `attackablesCampPlayerCorpses` | `false` | `true` keeps monsters next to the player they killed. |

`geodata`:

| Key | Default | Meaning |
|---|---|---|
| `dir` | `""` | Folder of L2J geodata tiles, one `X_Y.l2j` file per map tile, as Mobius reads them. The files are not part of L2Go or Mobius: bring your own. With them, ground heights, walls and closed cells count for movement, teleports and spawns; a tile without a file has no geodata. Empty, or a folder without tiles, means no geodata: heights come from the client and the spawns, and everything is walkable and visible, as in Mobius without geodata files. A folder that does not exist stops the game server. |

`items`:

| Key | Default | Meaning |
|---|---|---|
| `allowDiscardItem` | `true` | Players may drop and destroy items. |
| `autoDestroyDroppedItemAfter`, `autoDestroyHerbTime` | `"600s"`, `"60s"` | Items, and herbs, on the ground vanish after this time; `"0s"` keeps them. |
| `listOfProtectedItems` | `[]` | Item IDs that never vanish from the ground. |
| `destroyPlayerDroppedItem`, `destroyEquipableItem` | `false`, `false` | Also let items dropped by players, and equipable ones among them, vanish. |
| `multipleItemDrop` | `true` | A count of a non-stackable item drops as that many items. |
| `maxAdena` | `2000000000` | Most adena a player can hold. |
| `warehouseSlotsNoDwarf`, `warehouseSlotsDwarf` | `100`, `120` | Private warehouse size (Mobius `MaximumWarehouseSlotsForNoDwarf`/`ForDwarf`). |

`economy`:

| Key | Default | Meaning |
|---|---|---|
| `altKarmaPlayerCanShop`, `altKarmaPlayerCanUseWarehouse` | `true`, `true` | Players with karma may use shops, and warehouses. |
| `onlyGMItemsFree` | `true` | Only GMs get shop items for free. |
| `multisellAmountLimit` | `10000` | Most exchanges in one multisell request. |
| `customMultisell` | `false` | Also load the custom multisell lists (Mobius `CustomMultisellLoad`). Their NPCs are custom spawns, off by default. |
| `floodDropItem`, `floodTransaction`, `floodMultisell` | `"1s"` each | Shortest time between two item drops, two trades or shop actions, and two multisell exchanges; `"0s"` means no limit (Mobius `FloodProtector*Interval`). |

`rates` (Mobius `Rates.ini`):

| Key | Default | Meaning |
|---|---|---|
| `xp`, `sp` | `1`, `1` | Exp and SP from monsters (Mobius `RateXp`, `RateSp`). |
| `deathDropChance`, `deathDropAmount` | `1`, `1` | Chance and amount of monster drops (Mobius `DeathDrop*Multiplier`). |
| `herbDropChance`, `herbDropAmount` | `1`, `1` | The same for herbs. |
| `dropChanceByItem`, `dropAmountByItem` | `{"57": 1}`, adena and the grand boss jewels at `1` | Item ID to multiplier; a listed item uses this instead of the general rate. A map in the file replaces the default one (Mobius `Drop*MultiplierByItemId`). |
| `dropMaxOccurrencesNormal` | `2` | Most drops under 100 % chance from one monster. |
| `dropAdenaMinLevelDifference`, `dropAdenaMaxLevelDifference`, `dropAdenaMinLevelGapChance` | `8`, `15`, `10` | Adena drop chance falls from 100 % to this percent as the killer is min to max levels above the monster. Max must be above min. |
| `dropItemMinLevelDifference`, `dropItemMaxLevelDifference`, `dropItemMinLevelGapChance` | `5`, `10`, `10` | The same for items. |
| `monsterExpMaxLevelDifference` | `11` | Killers this many levels above a monster get no exp. |
| `karmaLost`, `karmaExpLost` | `-1`, `1` | Rate that divides the exp burning karma (`-1` means `xp`), and multiplier of the exp lost on death with karma (Mobius `RateKarmaLost`, `RateKarmaExpLost`). |

`combat`:

| Key | Default | Meaning |
|---|---|---|
| `randomizeAutoAttackDamage` | `true` | Random factors on auto-attack damage. |
| `shieldBlocks` | `false` | Damage of a blocked hit: `true` P.Atk minus shield defence, `false` P.Atk / (shield defence + P.Def) (Mobius `AltShieldBlocks`). |
| `perfectShieldBlockRate` | `10` | Percent of shield blocks that are perfect (Mobius `AltPerfectShieldBlockRate`). |
| `cancelByHit` | `"cast"` | What a hit interrupts: `"cast"`, `"bow"`, `"all"` or `"none"` (Mobius `AltGameCancelByHit`). |
| `distanceBowDamage` | `false` | Bow damage depends on distance. |
| `hpRegenMultiplier`, `mpRegenMultiplier`, `cpRegenMultiplier` | `100` each | Player regeneration, in percent. |
| `respawnRestoreHP`, `respawnRestoreCP`, `respawnRestoreMP` | `65`, `0`, `0` | Percent of HP, CP and MP a player revives with (0-100). |
| `delevel` | `true` | Death exp loss may take a level away. |
| `deathPenaltyChance` | `20` | Percent chance that a death raises the Death Penalty. |
| `partyRange` | `1500` | How close party members must be to share exp and items (Mobius `AltPartyRange`). The only party range key. |
| `randomRespawnInTown` | `true` | Restart at a random point of the town (Mobius `RandomRespawnInTownEnabled`). |
| `autoLoot`, `autoLootHerbs` | `false`, `false` | Monster drops, and herbs, go straight into the killer's inventory. |

`pvp` (Mobius `PVP.ini`, and the karma drop rates of `Rates.ini`):

| Key | Default | Meaning |
|---|---|---|
| `pvpVsNormalTime`, `pvpVsPvpTime` | `"120s"`, `"60s"` | How long a player stays flagged after hitting a player who is not flagged, or one who is. |
| `minimumPkRequiredToDrop` | `6` | PK count from which a player with karma drops items on death (Mobius `MinimumPKRequiredToDrop`). |
| `karmaDropLimit` | `10` | Most items dropped on one death. |
| `karmaRateDrop`, `karmaRateDropItem`, `karmaRateDropEquip`, `karmaRateDropEquipWeapon` | `40`, `50`, `40`, `10` | Percent chance to drop at all, and per unequipped item, equipped item and equipped weapon. |
| `listOfNonDroppableItems`, `listOfPetItems` | Mobius's lists | Item IDs a player with karma never drops. |
| `canGMDropEquipment` | `false` | GMs with karma drop items too. |
| `awardPKKillPVPPoint` | `false` | Killing a player with karma gives a PvP point. |
| `karmaPlayerCanBeKilledInPeaceZone` | `false` | Players with karma can be attacked in peace zones (Mobius `AltKarmaPlayerCanBeKilledInPeaceZone`). |

Lua scripting
----

Game rules and content that admins want to change are Lua scripts in `scripts/`, run by the game server on its world loop in a sandbox (Lua 5.1, no file or OS access, a time limit per event). The hooks today:

- Character creation: `scripts/events/character_create.lua` refuses names with forbidden words (the L2J Mobius `ForbiddenNames` list) and is meant to be edited.
- Chat: `events.on_chat` handlers can change or drop a message.
- NPC scripts, in Mobius's style: a script answers the first click on an NPC, its quest link and its own dialog links (`bypass -h Script <name> <event>`) with a page, inline HTML or a message, and can make the NPC speak. `scripts/npcs/talking_island_guard/` is an example.

```lua
events.on_character_create(function(ev)
  if ev.name:lower():find("admin", 1, true) then
    return events.reject("incorrect_name")
  end
end)

local guard = script.new("TalkingIslandGuard")

guard:on_first_talk({ 30733, 31032 }, function(npc, player)
  if player:level() < 20 then
    return "welcome.htm", { name = player:name() } -- a page next to the script
  end
end)
```

- Reload without a restart, in either of two ways. The game server watches the folder when `scripts.devReload` or `-dev-reload` is on (it is on in Docker Compose, which mounts `./scripts`), so saving a file is enough. Or send `SIGHUP`: `docker compose kill -s HUP game` (`make reload-scripts`), or `kill -HUP <pid>`. A broken script never replaces working ones; the errors are logged with file and line.
- `make scripts-check` loads every script and reports problems as `file:line: message`. CI runs it, and `go test ./...` loads every script too.
- `scripts/types/l2go.lua` gives editor autocomplete through the Lua Language Server.

See [`scripts/README.md`](scripts/README.md) for the API, the sandbox and how to test scripts with `internal/scripting/scripttest`.

Game data
----

`data/` holds the static game data the game server loads at startup: the player classes with their per-level stats, the stat bonus and experience tables, all items, and the world: NPC and monster templates, spawns, teleporters, zones and map regions. The JSON files are generated by `cmd/datagen` from the L2J Mobius Interlude datapack and committed, so the server does not need Mobius at runtime.

The NPC dialog pages are not in git: they are NCSoft's dialogue text, not functional values. `make data` copies them from a local Mobius checkout into `data/html/` (git-ignored), where the game server reads them (`<dataDir>/html`). The Docker image includes them when the folder exists.

Clone [L2J Mobius](https://gitlab.com/MobiusDevelopment/L2J_Mobius) into `L2J_Mobius/` (also git-ignored), then:

```sh
make data                                             # the Go enums of internal/data, all of data/, and data/html
make data MOBIUS=/path/to/L2J_Mobius_CT_0_Interlude   # from another checkout
go run ./cmd/datagen html                             # only the dialog pages; the committed files stay as they are
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
make data           # regenerate data/ and data/html from L2J_Mobius/ (see Game data)
make bots           # load test against the running stack (see below)
make refdiff        # compare L2Go with the Mobius reference server (see below)
```

`go test ./...` includes `internal/e2e`, which starts the real login server and game server in one process (in-memory storage, the real `data/` and `scripts/`) and walks two fake Interlude clients from `internal/testclient` through the F1-F11 checklist above and the world features of Milestones 3 and 4. `make smoke` runs the same flow against a running stack.

Lint and vulnerability tools are pinned in `tools/go.mod` and run with `go tool -modfile=tools/go.mod`, so nothing has to be installed globally.

### How the game server works

- **One owner for the world.** One goroutine, the world loop (`internal/world`), owns all world state: the players in the world, NPCs, regions, timers and the Lua state. Each connection reads and decodes its packets on its own goroutine and hands them to the loop; its writer goroutine encrypts and sends what the loop queued. Characters are loaded off the loop, and saved off it from snapshots taken on it. Code that touches the world runs on the loop: in a packet handler, a timer, a tick hook or a function given to `Server.OnLoop`. Never make the loop wait for the database or the network.
- **Features register themselves.** Movement, chat, NPCs, dialogs and the other features live in their own files of `internal/game`. Each one registers from `init` with `registerFeature(name, setup)`, and `setup` adds its packet handlers (`Register`, `RegisterEx`), GM commands (`RegisterGMCommand`), user commands such as `/loc` (`RegisterUserCommand`) and hooks (`OnPlayerEnter`, `OnPlayerLeave`, `OnTick`, ...). There is no central list to edit; `go doc ./internal/game` has the details.
- **Time is injectable.** The loop takes a clock, so tests use `world.NewFakeClock` or `testing/synctest` instead of sleeping.
- **Mobius is the reference.** Packet layouts come from the Mobius `writeImpl`/`readImpl` methods, golden tests are built from them by hand, and `refdiff` compares whole packet streams with a running Mobius.

### Load test: `make bots`

`cmd/bots` runs N fake players (default 50) against the running stack until Ctrl+C. Each logs in with its own account, creates a Human Fighter if it has none, enters the world, and every few seconds walks to a random point nearby, says something in general chat, or targets and talks to the nearest NPC. Every 10 seconds it prints the bots in the world, the errors, the packets per second and the latency percentiles of each kind of request (move, say, target, talk, login, enter).

```sh
make bots N=500 BOTS_ARGS='-duration 5m -rate 50'      # 500 bots for 5 minutes, 50 new ones per second
make bots BOTS_ARGS='-login 127.0.0.1:12106 -server 2'  # against the Mobius reference server
go run ./cmd/bots -h                                    # every flag
```

### Reference diff: `make refdiff`

`cmd/refdiff` runs a scripted client scenario against L2Go and against the L2J Mobius reference server at the same time. It normalizes both packet streams (object IDs, session keys, names, game time) and prints the differences side by side. Mobius is the reference for packet order and values.

```sh
make up                                                  # L2Go: login 2106, game 7777
make reference-up                                        # Mobius: login 127.0.0.1:12106, game 127.0.0.1:17777
make refdiff                                             # SCENARIO=login-create-enter
make refdiff SCENARIO=chat REFDIFF_ARGS='-ignore NpcSay'
go run ./cmd/refdiff -list                               # the scenarios
```

The scenarios are `login-create-enter`, `enter-two-players`, `chat`, `restart` and `npc-talk`. `-save file` records the normalized stream of one side as a golden file, and `-golden file` compares L2Go with it without Mobius running. The exit code is 0 when the streams match and 1 when they differ.

The reference stack is Mobius's login and game servers with MariaDB, built from the checkout in `L2J_Mobius/` (`make reference-up`, `reference-logs`, `reference-down`, `reference-build`). It is its own compose project, `l2go-reference`, so it never touches the `l2go` stack, and it listens on `127.0.0.1` only. The first start compiles Mobius and takes several minutes. See [`deploy/reference/README.md`](deploy/reference/README.md).

### Layout

```
cmd/l2go/                           command line: login, game, scripts check
cmd/datagen/                        converts the L2J Mobius datapack into data/ (and copies data/html)
cmd/bots/                           load-test bots
cmd/refdiff/                        reference diff against Mobius: scenarios, recording, comparison
internal/config/                    config file loading and validation
internal/packets/                   framing, packet Reader/Writer, UTF-16 strings
internal/crypt/blowfish/            Lineage II Blowfish (little-endian x/crypto/blowfish)
internal/crypt/logincrypt/          login packet encryption: static key, XOR pass, session key, checksum
internal/crypt/rsakey/              RSA key pool, modulus scrambling, raw RSA, credentials block
internal/crypt/gamecrypt/           game packet XOR cipher
internal/session/                   session keys from login to game server
internal/link/                      login <-> game server link
internal/data/                      game data loader: players, items, world
internal/netutil/                   connection registry, accept loop
internal/storage/                   data models and store interfaces
internal/storage/memstore/          in-memory store (tests)
internal/storage/mongostore/        MongoDB store
internal/storage/storagetest/       conformance tests for every store
internal/loginserver/               login server
internal/gameserver/                game server network layer: connections, session states, character screen
internal/gameserver/serverpackets/  game packets the server sends
internal/gameserver/clientpackets/  game packets the client sends
internal/gameserver/htmlutil/       the links of dialog pages (the bypass link cache)
internal/game/                      the game on the world loop: players, NPCs, movement, chat, dialogs, ...
internal/world/                     world loop, clock and timers, regions and visibility, broadcasting
internal/stats/                     stat engine: Mobius formulas and modifiers
internal/geo/                       geodata: L2J tiles, heights, walls, line of sight, paths
internal/scripting/                 Lua engine: sandbox, loader, events, NPC scripts, reload
internal/scripting/scripttest/      test harness for Lua scripts
internal/testclient/                fake Interlude client for the tests, the bots and refdiff
internal/e2e/                       full-flow test of both servers, and the smoke test
scripts/                            Lua scripts (see scripts/README.md)
data/                               game data (see data/README.md)
deploy/compose/                     config for docker compose
deploy/reference/                   Mobius reference stack (see deploy/reference/README.md)
extra/default_config_file/          example config
tools/                              pinned lint and vulnerability tools
```

Disclaimer
----

This work is for nonprofit educational purposes only. No copyright infringement is intended.

Using this work to run a private server, is at your own risk.

NCSOFT©, the Interlocking NC Logo, PLAYNC, Lineage, Team Lineage II, Lineage II, Goddess of Destruction, Valiance, Truly Free, L2 Store, L2 Galleria, NCoin, Path to Awakening, and all associated logos and designs are trademarks or registered trademarks of NCSOFT Corporation.

License
----

L2Go is under the GNU GPL v3.0 license (see `LICENSE`).

Credits: [L2J Mobius](https://gitlab.com/MobiusDevelopment/L2J_Mobius) is the reference for the Interlude protocol (packet layouts, encryption, game rules), the source of the game data in `data/` and of the dialog pages `make data` copies.

Made with love by Frostwind <hi@frostwind.me>
