# Mobius reference stack

A local L2J Mobius Interlude server (login server, game server, MariaDB), built from the Mobius checkout in `L2J_Mobius/`. It is the oracle for `cmd/refdiff`: the same `internal/testclient` scenario runs against Mobius and against L2Go, and the decoded packet streams are compared. Mobius is the reference for packet order and values (Plan 04, "Testing strategy"; Plan 05, Phase 0).

It is its own compose project, `l2go-reference` (`name:` in `docker-compose.yml`, plus `-p l2go-reference` in every Makefile target), with its own containers, network and volumes. It never touches the `l2go` stack, and both can run at the same time: all its ports are on 127.0.0.1 and differ from L2Go's.

## Commands

| Command | What it does |
|---|---|
| `make reference-up` | Builds the image if needed and starts the stack. The first run also compiles Mobius. |
| `make reference-logs` | Follows the logs of every service. |
| `make reference-down` | Stops the stack. The database and the build stay in their volumes. |
| `make reference-build` | Rebuilds the image, stops the two Mobius servers and recompiles Mobius. Start again with `make reference-up`. |

For anything else, use the same project and file as the Makefile:

```sh
docker compose -p l2go-reference -f deploy/reference/docker-compose.yml ps
```

`ps` shows a server as `healthy` once it listens. For the game server that means it has finished loading.

## Addresses

| | |
|---|---|
| Login server | `127.0.0.1:12106` |
| Game server | `127.0.0.1:17777`, announced in the server list as server 2 ("Sieghardt") |

- Logging in with an unknown account creates it with the password given (`AutoCreateAccounts`).
- Game packets are Blowfish-encrypted (`PacketEncryption = True`), like L2Go's. The protocol revision is 746.
- MariaDB is not published to the host:

  ```sh
  docker compose -p l2go-reference -f deploy/reference/docker-compose.yml exec mariadb mariadb -ul2jmobius -pl2jmobius l2jmobiusinterlude
  ```

## First run

`make reference-up` on a clean machine:

1. Pulls `eclipse-temurin:27-jdk` and `mariadb:11`, and downloads Apache Ant.
2. Initializes MariaDB with Mobius's SQL.
3. Compiles Mobius with Ant (about 1,600 Java files) and copies the datapack (about 17,500 files) into a volume.
4. Starts the login server, then the game server. The game server loads the datapack and compiles about 870 Java scripts in memory before it opens its port.

Expect roughly 5–10 minutes, most of it the image pulls, the compile and the game server's first load. These are estimates, not measurements. Later starts skip steps 1–3 and take about as long as the game server's load (a minute or two).

`make reference-up` returns once the containers are started. Follow the progress with `make reference-logs` and wait for `healthy` in `ps` before running `refdiff`.

## How it works

| Service | What it does |
|---|---|
| `mariadb` | MariaDB 11. Mobius keeps login and game tables in one database, `l2jmobiusinterlude` (user and password `l2jmobius`). On its first start, `mariadb/initdb/10-mobius-sql.sh` loads `dist/db_installer/sql/login/*.sql` and then `.../game/*.sql`, in the same order as Mobius's DatabaseInstaller. |
| `mobius-build` | One-shot, image `mobius/Dockerfile` (`eclipse-temurin:27-jdk` with Apache Ant 1.10.15). `mobius/build.sh` runs Mobius's `build.xml`, target `jar`, with `-Dbuild=/tmp/mobius-ant` (the default, `../build`, would be inside the read-only checkout). It then lays the server out in the `mobius-server` volume, like Mobius's release zip: `libs/` (LoginServer.jar, GameServer.jar and `dist/libs`), `login/` and `game/` (copies of `dist/login` and `dist/game`, with `config/` kept as `config.default/`). If a build is already there, it only reports whether the checkout changed since. |
| `mobius-login`, `mobius-game` | Same image. `mobius/start.sh` rebuilds `config/` from `config.default/` and our overrides at every start. Then it runs Mobius's start command from the server folder: `java $(cat java.cfg) -jar ../libs/LoginServer.jar` (or `GameServer.jar`). |

The checkout (`MOBIUS_DIR`) is mounted read-only and never modified.

No registration step is needed for the game server. Mobius ships `game/config/hexid.txt` (server 2), and its `login/gameservers.sql` registers the same hex ID, so the game server authenticates as server 2. With `AcceptNewGameServer = True`, the login server would also accept an unknown one.

## Config overrides

Everything is Mobius's default except these keys. Each file in `config/login/` and `config/game/` is named after the Mobius config file it changes.

| File | Key | Value | Why |
|---|---|---|---|
| login `Server.ini` | `LoginHostname` | `*` | The game server connects over the compose network (default `127.0.0.1`). |
| | `AcceptNewGameServer` | `True` | Mobius default, kept explicit. |
| | `AutoCreateAccounts` | `True` | Mobius default, kept explicit. |
| login and game `Database.ini` | `URL` | Mobius's URL with host `mariadb` | The compose database (default `localhost`). |
| | `Login`, `Password` | `l2jmobius` | Default `root` with an empty password. |
| login and game `Interface.ini` | `EnableGUI` | `False` | No GUI in a container. |
| game `Server.ini` | `LoginHost` | `mobius-login` | The login service (default `127.0.0.1`). |
| | `GameserverPort` | `17777` | The port the login server announces; published as `127.0.0.1:17777` (default 7777). |
| | `PacketEncryption` | `True` | The client path matches L2Go's (default `False`). |
| | `AllowedProtocolRevisions` | `746` | Mobius default (Interlude), kept explicit. |
| game `ipconfig.xml` | whole file | `127.0.0.1` | The address announced to every client. Without this file, Mobius configures the network itself and asks `checkip.amazonaws.com`. |

To change a setting, add `Key = Value` to the matching file in `config/login/` or `config/game/`, then restart that server:

```sh
docker compose -p l2go-reference -f deploy/reference/docker-compose.yml restart mobius-game
```

`start.sh` logs every key it sets. A key that Mobius's file does not have is appended, with a warning (check the name).

## Logs

- `make reference-logs` shows the console output of every service: the Ant build, the MariaDB initialization, the overrides `start.sh` applied, and Mobius's console log.
- Mobius's own log files (`java0.log`, `error0.log`, `chat0.log`, ...) are in the `mobius-server` volume, under `/srv/mobius/login/log/` and `/srv/mobius/game/log/`:

  ```sh
  docker compose -p l2go-reference -f deploy/reference/docker-compose.yml exec mobius-game tail -f log/java0.log
  ```

  A rebuild (`make reference-build`) replaces the server folders and their logs.

## Reset the database

```sh
make reference-down
docker volume rm l2go-reference_mariadb-data
make reference-up    # MariaDB reloads Mobius's SQL on its first start
```

Do the same if the first initialization failed (for example with a wrong `MOBIUS_DIR`). MariaDB runs its init scripts only on an empty volume.

To remove everything, the database and the build: `docker compose -p l2go-reference -f deploy/reference/docker-compose.yml down -v`.

## Another or updated Mobius checkout

`make reference-up` never recompiles an existing build. When the checkout has changed, `mobius-build` says so in the logs. To rebuild:

```sh
make reference-build
make reference-up
```

`MOBIUS_DIR` selects another checkout, preferably as an absolute path (a relative one resolves from `deploy/reference/`):

```sh
MOBIUS_DIR=/path/to/L2J_Mobius_CT_0_Interlude make reference-build reference-up
```

To use another Ant version, change `ANT_VERSION` in `mobius/Dockerfile`.

## Memory

The game server uses Mobius's `game/java.cfg`: `-Xms2g -Xmx4g` with ZGC. The login server uses `-Xms128m -Xmx256m`. Give Docker at least 6 GB. To use other options, set `MOBIUS_GAME_JAVA_OPTS` (or `MOBIUS_LOGIN_JAVA_OPTS`). It replaces the whole of `java.cfg`, so keep Mobius's other options:

```sh
MOBIUS_GAME_JAVA_OPTS="-server -Dfile.encoding=UTF-8 -Djava.util.logging.manager=org.l2jmobius.log.ServerLogManager -Dorg.slf4j.simpleLogger.log.com.zaxxer.hikari=warn -XX:+UseZGC -Xms1g -Xmx2g" make reference-up
```

## GM characters

Mobius takes the GM level from the character (`AccessLevels.xml`; 100 is Master). Run this while the character is offline:

```sh
docker compose -p l2go-reference -f deploy/reference/docker-compose.yml exec mariadb \
  mariadb -ul2jmobius -pl2jmobius l2jmobiusinterlude \
  -e "UPDATE characters SET accesslevel = 100 WHERE char_name = 'Name'"
```

## Notes

- Mobius ships no geodata (`data/geodata/` holds only a readme), so the reference runs without geodata, like L2Go's `geo.None`.
- The stack is for local comparison only: fixed passwords, ports on 127.0.0.1.
