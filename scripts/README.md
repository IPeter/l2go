Lua scripts
====

Game content that changes often is written here in Lua. The game server loads every script at startup and can reload them without a restart. Networking, data loading and anything that runs every tick stay in Go.

The hooks today: character creation, chat, NPC scripts (first talk, quest-link talk and dialog links), quests (quest states, items, kills and the other world events, timers), the scripted monster, area and boss AI, and the ferry schedules. Commands come as the game grows.

Layout
----

```
events/          event rules, e.g. character_create.lua, and the seasonal events, one folder each: events/L2Day/L2Day.lua
npcs/            NPC scripts, one folder each with its pages, e.g. talking_island_guard/
quests/          quests, one folder each, named after the Mobius class: quests/Q00001_LettersOfLove/
village_master/  the village master scripts: class changes, clan and alliance pages
ai/monsters/     monster scripts from Mobius ai/others, one file each: chests.lua, town_pets.lua, ...
ai/areas/        area scripts, one folder per area: ai/areas/<Area>/<Class>.lua, or <Area>/<Class>/<Class>.lua
ai/bosses/       boss scripts, one file per Mobius class (QueenAnt.lua, ZakenDoor.lua, ...), with their pages
vehicles/        the five ferry schedules, e.g. talking_gludin.lua
lib/             shared helpers, loaded only through require("lib.name"), e.g. lib/java.lua
types/           API definitions for the editor (never run)
```

A `commands/` folder appears when that feature exists. Every `.lua` file outside `lib/` and `types/` runs when the scripts load, in path order (`events/a.lua` before `events/b.lua` before `npcs/x.lua`). Files and folders starting with `.` are skipped, and so is everything that is not `.lua` (pages are read by the game server when it shows them).

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
- Keep state you need after a reload or restart out of Lua globals: every reload starts from a fresh Lua state. Quest states, player variables and `globals` are saved by the server.

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

A table as the second return value fills `%name%` placeholders: `return "welcome.htm", { name = player:name() }`. The server's own placeholders are filled too, as Mobius fills them: `%objectId%` (the NPC) and `%playername%` in a page file, `%playername%` in returned HTML.

A page name is looked up next to the script first, then in the server's HTML folder (`data/html`), so a script can also answer with a shared Mobius page, for example `"default/30598.htm"`; a name starting with `data/html/` always means the HTML folder. A page that does not exist shows nothing (and is logged). A handler that fails shows players nothing (the first talk falls back to the NPC's own page); GMs get a "Script error" page with the error.

How links reach scripts:

- `bypass -h Script <name> <event>` (also `npc_%objectId%_Script <name> <event>`) runs `on_event` of script `<name>` with the NPC the player talked to last, if the player still stands within talking range of it (Mobius `processScriptEvent`); otherwise nothing happens.
- `bypass -h Script <name>` runs `on_talk` of `<name>` at the page's NPC.
- `bypass -h Script` alone is the NPC's quest link: with one script talking at the NPC it answers; with several the player picks one from a list; with none the server shows `data/html/noquest.htm`.
- `npc_%objectId%_Chat 0` on an NPC whose first talk a script owns runs `on_first_talk` again.

Every link a page offers is remembered for the player; a link that the last page did not offer, or clicked too far from the NPC, is ignored.

### Shipped NPC scripts

| Folder | Script | NPCs | What it shows |
|---|---|---|---|
| `npcs/talking_island_guard/` | `TalkingIslandGuard` | Talking Island guards | The example above. |
| `npcs/symbol_maker/` | `SymbolMaker` | The 11 Symbol Makers | Mobius's `ai/others/SymbolMaker`: the symbol pages; "Draw" and "Remove" open the dye windows (`player:show_henna_equip_list()`, `show_henna_remove_list()`). |
| `npcs/asamah/` | `Asamah` | Asamah (32115) | Mobius's `ai/others/Asamah`: the Elroki pages and the capture skill explanations. |
| `npcs/kier/` | `Kier` | Kierre's statue (32022) | Mobius's `ai/others/Kier`. |
| `npcs/monument_of_heroes/` | `MonumentOfHeroes` | The 5 Monuments of Heroes | Mobius's `ai/others/MonumentOfHeroes`: the links of the game's `olympiad/hero_main.htm` (a hero or a noble sees it): an unclaimed hero claims the status (`hero.claim`), a hero takes one Infinity weapon and one Wings of Destiny Circlet. The Grand Olympiad Manager's own pages (`OlympiadNoble`, `Olympiad`) are commands of the game, as in Mobius's `OlympiadManager` NPC class. |
| `npcs/castle_chamberlain/` | `CastleChamberlain` | The 9 chamberlains | Mobius's `ai/others/CastleChamberlain`: the lord's report, the tax within the Seal of Strife's limit, the vault (deposit and withdraw between the inventory and the treasury), the gates, banishing foreigners, the siege window, the general items buylist and the Lord's Crown; members of the owner clan by their clan privileges. Not yet: renting castle functions and their teleports and buffs, trap upgrades (their pages show, the confirmation answers "not available"); door upgrades work (`castle:set_door_upgrade`), the manor (M11k: its pages only), the dawn ticket sale (M11l). |
| `npcs/castle_siege_manager/` | `CastleSiegeManager` | The 9 castle siege managers, 2 clan hall ones | Mobius's `ai/others/CastleSiegeManager`: the lord's pages, the siege window (`castle:show_siege_info`) for registering. The clan hall managers answer "not available" until M11j. |
| `npcs/castle_mercenary_manager/` | `CastleMercenaryManager` | The 9 mercenary managers | Mobius's `ai/others/CastleMercenaryManager`: the ticket buylists (`npc:show_buy_window`, without the tax shown) for the lord, GMs and members with `CASTLE_MERCENARIES`, the limits per castle, the Seal of Strife's pages. |
| `npcs/castle_teleporter/` | `CastleTeleporter` | The 9 mass gatekeepers | Mobius's `ai/others/CastleTeleporter`: after 30 s (8 min in a siege without control towers) everyone in the castle's teleport zone goes inside, with a shout to the map region. |
| `npcs/castle_warehouse/` | `CastleWarehouse` | The 9 castle warehouse keepers | Mobius's `ai/others/CastleWarehouse`: the warehouse pages (the windows are the game's), the Blood Alliance pages (the clan's count is not kept: always 0). |
| `npcs/castle_blacksmith/` | `CastleBlacksmith` | The 9 castle blacksmiths | Mobius's `ai/others/CastleBlacksmith`: the crop pages and their multisell for the lord, GMs and members with `CASTLE_MANOR`. |
| `npcs/castle_court_magician/` | `CastleCourtMagician` | The 9 court magicians | Mobius's `ai/others/CastleCourtMagician`: the clan gate to the leader and the talisman trade (its items are not in the Interlude data, so it always asks for more epaulettes). |
| `npcs/wyvern_manager/` | `WyvernManager` | The 10 castle and 2 clan hall wyvern managers | Mobius's `ai/others/WyvernManager`: the lord's pages; the lord on a strider of level 55 or more trades it and 25 B-grade crystals for a wyvern (`player:dismount`, `player:mount`). |

The castle scripts serve Mobius's pages from `data/html/scripts/ai/others/<Class>/` (they return `data/html/...` page names); `castle:seal_owner` and `castle:seven_signs_period` are the Seven Signs' (the `sevensigns` module below).

`village_master/` holds Mobius's 16 village master scripts, ported by hand with the same names: the first and second class changes (`ElfHumanFighterChange1`, `ElfHumanWizardChange1`, `DarkElfChange1`, `OrcChange1`, `DwarfBlacksmithChange1`, `DwarfWarehouseChange1` and their `...Change2`, plus `ElfHumanClericChange2`), the newbie village headmasters (`FirstClassTransferTalk`) and the clan and alliance pages (`ClanMaster`, `AllianceMaster`; the clan, unit, clan skill and alliance actions of their pages, `npc_%objectId%_create_clan`, `..._create_ally` and the others, are commands of the game's village masters, as in Mobius `VillageMaster`). `lib/class_master.lua` holds their shared class change step. Their pages are Mobius's, in `data/html/scripts/village_master/`.

The Mobius ports keep their page names and links. Where Mobius asks for quest states or items, which the Lua API does not have yet, the scripts take the branch of a player who never took the quest (marked in the code).

### Quests: `quest`

Quests follow Mobius's `Quest` and `QuestState` classes: the method names are Mobius's in snake case (`getQuestItemsCount` is `q:get_quest_items_count`), with the same arguments in the same order, so a Mobius quest reads almost line by line. `cmd/questport` translates Mobius's quests this way; `quests/Q00001_LettersOfLove` and `quests/Q00401_PathOfTheWarrior` are ported by hand and are the reference.

```lua
-- quests/Q00001_LettersOfLove/Q00001_LettersOfLove.lua (shortened)
local q = quest.new(1, "Q00001_LettersOfLove", "Letters of Love")
local DARIN, ROXXY = 30048, 30006
local DARIN_LETTER = 687

q:register_quest_items(DARIN_LETTER)
q:add_start_npc(DARIN)
q:add_talk_id(DARIN, ROXXY)

q:on_event(function(event, npc, player)
  local st = q:get_quest_state(player, false)
  if st ~= nil and event == "30048-06.htm" then
    st:start_quest()              -- cond 1, STARTED, the accept sound
    q:give_items(player, DARIN_LETTER, 1)
  end
  return event                    -- the page the link named
end)

q:on_talk(function(npc, player)
  local st = q:get_quest_state(player, true)
  if st:is_created() then
    return player:level() < 2 and "30048-01.htm" or "30048-02.htm"
  elseif st:is_started() and npc:id() == ROXXY and st:is_cond(1) then
    st:set_cond(2, true)          -- the quest log and the middle sound
    return "30006-01.htm"
  end
  return q:get_no_quest_msg(player)
end)
```

- `quest.new(id, name, description)` makes a quest: `id` ≥ 1 is listed in the quest log and can be aborted; `name` is the Mobius class name, used in `Script <name> <event>` links; `description` is the quest window's name. `script.new(name)` is `quest.new(-1, name, "")`, so everything here works for plain scripts too.
- **Registration**, while the scripts load: `add_start_npc`, `add_talk_id`, `add_first_talk_id` (one owning script per NPC), `add_kill_id`, `add_attack_id`, `add_spawn_id`, `add_skill_see_id`, `add_spell_finished_id`, `add_aggro_range_enter_id`, `add_enter_zone_id`, `add_exit_zone_id`, `add_item_talk_id`. Each takes IDs or lists, as many as you like; duplicates count once. Plus `register_quest_items(...)` (taken on exit and abort), `set_custom(true)` (no quest mark, no `start_quest`), `set_on_enter_world(true)`, and `requires("clans", ...)` for quests that need a later feature (`clans`, `clan_halls`, `subclass`, `noblesse`, `pets`, `fishing`, `seven_signs`, `sepulchers`, `grand_bosses`, `doors`): they are not offered until it exists (`clans` does since M9, `sepulchers` and `grand_bosses` since M10, `subclass` and `noblesse` since M11e).
- **Handlers**, one per kind: `q:on_x(fn)` handles every ID registered for x; `q:on_x(ids, fn)` registers the IDs and binds `fn` to them (the NPC script form above).

| Handler | Arguments | Returns |
|---|---|---|
| `on_talk` | `npc, player` | a reply |
| `on_first_talk` | `npc, player` | a reply; `nil` shows the NPC's own page |
| `on_event` | `event, npc, player` (links and timers; `npc`, `player` may be nil) | a reply |
| `on_kill` | `npc, killer, is_summon` — 2.5 s after the death | – |
| `on_attack` | `npc, attacker, damage, is_summon, skill` | – |
| `on_spawn` | `npc` | – |
| `on_skill_see` | `npc, caster, skill, targets, is_summon` | – |
| `on_spell_finished` | `npc, player, skill` | – |
| `on_aggro_range_enter` | `npc, player, is_summon` | – |
| `on_enter_zone`, `on_exit_zone` | `creature, zone` | – |
| `on_item_talk` | `item, player` | a reply |
| `on_death` | `killer, victim, st` | – |
| `on_enter_world` | `player` | – |
| `on_check_party_member`, `on_check_party_member_state` | `player, npc` / `st, npc` | boolean |
| `on_action_for_each_player` | `player, npc, is_summon` | – |
| `on_start` | – (after the scripts are in place, at startup and after each reload: spawns, timers, global variables) | – |

**Quest state.** `q:get_quest_state(player, create)` returns the player's `QuestState` (nil when there is none and `create` is false). The state lives in the server and is saved with the character, so it survives reloads and restarts; keep progress there, never in Lua variables. Its methods: `get_state()` / `is_created()` / `is_started()` / `is_completed()` / `set_state(State.X)`, `start_quest()`, `get_cond()` (0 unless started), `is_cond(n)` (whatever the state, as Mobius), `set_cond(n, play_middle)`, `set` / `get` / `get_int` / `unset` / `is_set` (string values; names up to 20 characters, values up to 255), the memo states, `exit_quest(repeatable, play_finish)` (true deletes the state, false keeps it COMPLETED; or a `QuestType`), `is_now_available()` / `set_restart_time()` (daily quests, 06:30 server time), `get_player()`, and the tutorial methods.

**Helpers** (Mobius `Quest` methods): `give_items`, `reward_items` (quest rates), `give_adena`, `take_items` (-1 takes all; these three also take Mobius's `ItemHolder` as `{id = , count = }`), `take_items_ids`, `get_quest_items_count`, `has_quest_items`, `has_at_least_one_quest_item`, `give_item_randomly(player, npc, id, min, max, limit, chance, play_sound)`, `add_exp_and_sp`, `play_sound(player, QuestSound.X)`, `get_item_equipped(player, Paperdoll.RHAND)`, `get_enchant_level`, `get_no_quest_msg`, `get_already_completed_msg`, `get_htm`, `show_html_file`, `add_spawn` (heading -1 is random), `add_radar` / `remove_radar` / `clear_radar`, `show_on_screen_msg`, and the party helpers `get_random_party_member`, `get_random_party_member_state`, `execute_for_each_player` (over the player's party, as Mobius).

**Timers.** `q:start_quest_timer(name, ms, npc, player, repeating)` runs `on_event(name, npc, player)` later; a reply goes to the timer's player. A timer is known by its name, NPC and player (nil matches only nil); starting one that runs already does nothing. A one-shot timer is gone before its handler runs, so the handler may start it again. `get_quest_timer`, `cancel_quest_timer`, `cancel_quest_timers(name)`. Timers stop when their NPC or player leaves the world and on reload; they are not saved.

**Values.** Within one handler call, one object is always the same Lua value, so `st == other_st` and `player == st:get_player()` work as in Java. Players, NPCs and quest states expire when the handler returns: keep object IDs and get the objects back with `world.find_object(id)`.

**Other modules.**

| Module | Functions |
|---|---|
| `rnd` | `rnd(n)` → [0, n) (0 for n ≤ 0); `rnd(a, b)` → [a, b]; `rnd.bool()`, `rnd.double([max])`, `rnd.entry(list)`. Mobius `Rnd` rules from the server's one random source; `math.random` uses it too (still [1, n]), and `math.randomseed` is gone. |
| `time` | `time.now_ms()`: the world clock. |
| `world` | `world.find_object(id)`, `world.get_player(id)`, `world.npc_exists(npc_id)`. |
| `skills` | `skills.get(id, level)` → Skill or nil; `skills.apply_effects(skill, effector, effected)`. |
| `items` | `items.get(id)` → `{id, name, stackable}` or nil. |
| `globals` | `get(name, default)`, `get_int`, `get_long`, `get_bool`, `set(name, value)`: saved server-wide variables. |
| `hero` | `hero.is_hero(player)`, `hero.is_unclaimed(player)`, `hero.claim(player)`: the Olympiad's heroes (Mobius `Hero.isHero`, `isUnclaimedHero`, `claimHero`). |
| `util` | `util.in_range(range, a, b, include_z)`. |
| `sevensigns` | Mobius `SevenSigns` (cabals 0 none, 1 Dusk, 2 Dawn; seals 1-3; periods 0-3): `period()`, `cycle()`, `seal_owner(seal)`, `highest_cabal()`, `cabal(player)`, `seal(player)`, `contribution(player)`, `join(player, cabal, seal)`, `contribute(player, blue, green, red)`, `take_reward(player)`, `clan_castle(player)`, `config()`, `ticket_count(castle_id)` / `set_ticket_count`, and Mobius's Player item calls the priests use: `destroy_item`, `add_item`, `reduce_adena`, `send_disappeared`, `has_room`. Changes are saved at once. |

**Constants** are read-only global tables with Mobius names; a name a table does not have is an error: `State`, `QuestType`, `QuestSound` (sound names), `Paperdoll`, `ChatType`, `Race`, `ClassId` (with `ClassId.level(id)`, `race`, `parent`, `is_mage`), `Category`, `SystemMessageId`, and `Config` (`ALT_PARTY_RANGE`, ...). `quest.DF_REWARD_35`, `_37` and `_39` are Mobius's Dimensional Diamond tables.

**Java helpers.** `local J = require("lib.java")` gives translated code Java's library behaviour (0-based `J.substring`, `J.idiv`, `J.split`, sorted `J.spairs`, ...); see `lib/java.lua`.

The world is only there in handlers: a quest helper, `time`, `world`, `globals`, `skills` or `items` called while the scripts load is an error (use `on_start`).

### AI scripts and ferries: `ai/` and `vehicles/`

Mobius's scripted AI (`dist/game/data/scripts/ai/`) and its ferry schedules (`scripts/vehicles/`) are ported one Lua file per Mobius class, named after it. They are plain scripts (`script.new`) with Mobius's AI callbacks. The monster AI itself (thinking, skills, minions, raid bosses, the grand boss store) is Go; Lua runs only on the events and timers of the NPCs a script registered, never per tick.

| Folder | What | Mobius source |
|---|---|---|
| `ai/monsters/` | 13 monster scripts, file names in snake case: `chests.lua`, `fairy_trees.lua`, `flee_monsters.lua`, `gordon.lua`, `non_lethalable_npcs.lua`, `polymorphing_angel.lua`, `polymorphing_on_attack.lua`, `see_through_silent_move.lua`, `timak_orc_troop_leader.lua`, `town_pets.lua`, `turek_orcs.lua`, `varka_ketra.lua`, `wasted_corpse.lua` | `ai/others` |
| `ai/areas/<Area>/` | 22 area scripts: `<Class>.lua` next to each other (`CrumaTower/CrumaTower.lua`, `PaganTemple/PaganKeys.lua`), or `<Class>/<Class>.lua` with its pages where Mobius gives the class a folder (`ImperialTomb/FourSepulchers/FourSepulchers.lua`) | `ai/areas`, without FeedableBeasts, Tunatun and Venom (M11) |
| `ai/bosses/` | `QueenAnt`, `Core`, `Orfen`, `Zaken`, `ZakenDoor`, `Baium`, `Antharas`, `Valakas`, `ValakasTeleporters`, `Sailren`, `DrChaos`, with their pages | `ai/bosses`, without Frintezza (M11) |
| `vehicles/` | The five ferry schedules: `talking_gludin.lua`, `giran_talking.lua`, `gludin_rune.lua`, `rune_primeval.lua`, `innadril_tour.lua` | `vehicles/Boat*.java` |

Shared libraries in `lib/`:

| File | What |
|---|---|
| `lib/boss.lua` | The grand boss pattern: the statuses, the respawn delay from `GrandBoss.ini` (`respawn_ms`), the restart-safe start (`start`: a live boss after a reload re-arms its timers, a dead one re-arms its respawn timer from the saved time, a living one in the store spawns where it was), `spawn_saved`, `died` (status, timer and respawn time saved at once), `oust`, sounds and desires. |
| `lib/four_sepulchers.lua` | The room spawns of the Four Sepulchers (data from Mobius `FourSepulchers.xml`), for `ai/areas/ImperialTomb/FourSepulchers/`. |
| `lib/boat.lua` | The docks, tickets and ferry announcements, `say`, `with_horn`, `await`, and `route`, which declares a schedule (`vehicles.add_route`) starting 3 minutes after the server. |

- **Events.** Besides the quest handlers above, scripts register `add_faction_call_id`, `add_creature_see_id`, `add_event_received_id`, `add_move_finished_id`, `add_node_arrived_id`, `add_route_finished_id`, `add_teleport_id` and `add_npc_hate_id`, with the handlers `on_faction_call`, `on_creature_see`, `on_event_received`, `on_move_finished`, `on_node_arrived`, `on_route_finished`, `on_teleport` and `on_npc_hate` (which runs at once and can veto a monster's hate by returning `false`). `on_day_night_change(fn)` reaches every script that sets it, at game 00:00 (`true`) and 06:00 (`false`). Events Mobius sends asynchronously (all but `on_npc_hate`) wait until the running handler ends, then run in order, each with its own time limit.
- **Modules.** `zones` (players and NPCs inside, move or oust them, allowed entry to boss zones), `doors` (open, close, state), `grandboss` (saved status and info, the live boss), `cinematics` (camera, earthquake, sound, screen message), `vehicles` (ferry schedules), more of `world` (`npcs_by_id`, `npcs_around`, `players_around`, `visible_objects`, `can_see_target`, `valid_location`) and `time` (`game_time`, `game_hour`, `is_night`). The NPC methods that drive the AI (hate, casts, movement, minions, script values, ...) are in `types/l2go.lua`.
- **Reloads.** A reload keeps the NPCs in the world but drops every timer. `on_start` gets `ev.reload`; a boss script then finds its live boss (`grandboss.boss`, `world.npcs_by_id`) and starts its timers again instead of spawning a second one. A reload does not change running ferries.

Tests: `internal/scripting/ai_*_test.go` run the scripts against the fake world, bosses on the fake clock; `internal/game/boss_script_test.go`, `quest_world_ai_*_test.go` and `vehicle_test.go` run them in a game server.

### Seasonal events: `events/<Event>/`

Mobius's three `LongTimeEvent` scripts, `Christmas`, `L2Day` and `HeavyMedal`, are ported one folder each, with Mobius's pages in `data/html/scripts/events/<Event>/`. `lib/long_time_event.lua` is Mobius's `LongTimeEvent`: `lte.new(name)` makes the script and runs the event's life on its dates (start, drops during the drop period, the NPCs until the end, the start and end announcements, the login notice, the items destroyed at the end; a reload keeps a running event), and `lte.on_event(s, fn)` takes the event's own links, since the library owns `s:on_event` and `s:on_start`. The data (dates, drop list, spawn list, messages) is each event's `config.xml`, converted by `cmd/datagen` into `data/events/longtimeevents.json`; the game keeps the drop list, the notice and the item cleanup, which scripts reach through the `long_events` module. Every shipped date is past: the config's `events.dates` moves an event (see the main README), and `//event_start` / `//event_stop` run one now. Tests: `internal/scripting/long_events_test.go` (each event with `scripttest`) and `internal/game/long_time_event_test.go`.

### `Player` and `Npc`

Handlers get players and NPCs as values with methods. They are valid only while that handler runs; keep `player:object_id()` instead of the player.

| Method | Returns |
|---|---|
| `player:object_id()`, `player:name()`, `player:level()` | integer, string, integer |
| `player:class_id()`, `player:race()` | integer (race: 0 human, 1 elf, 2 dark elf, 3 orc, 4 dwarf) |
| `player:adena()`, `player:item_count(id)`, `player:give_items(id, n)`, `player:take_items(id, n)` | the player's items |
| `player:get_quest_state(name)` | the state of any quest, by name |
| `player:sex()`, `player:base_class()`, `player:is_mage_class()`, `player:is_newbie()`, `player:is_dead()`, `player:is_in_category(Category.X)` | |
| `player:var(name, default)`, `var_int`, `var_bool`, `set_var(name, value)` | player variables, saved with the character |
| `player:set_class_id(ClassId.X)`, `player:set_base_class(id)`, `player:broadcast_user_info()` | the class change (effect, message, skill list) |
| `player:is_in_party()`, `party()`, `party_members()` | the party: `party()` is nil without one, else `{leader, members, loot}`; `party_members()` is the player alone without one |
| `player:is_sub_class_active()`, `class_index()`, `total_sub_classes()` | the subclasses: whether one is played, the active class index (0 for the base class), how many there are |
| `player:clan()`, `player:is_clan_leader()` | the clan (nil without one): `clan:id()`, `name()`, `level()`, `reputation()`, `leader_id()`, `leader()` (`:object_id()`, `:name()`, `:player()`, nil offline), `add_reputation(n)`, `broadcast_pledge_info_update()` |
| `player:is_noble()`, `set_noble(bool)` | noblesse (the flag and the noble skills) |
| `player:show_henna_equip_list()`, `show_henna_remove_list()` | the symbol maker's dye windows (Mobius `HennaEquipList`, `HennaRemoveList`) |
| `player:clan_id()`, `player:has_clan_privilege(name)` | the clan ID (0 without one); a clan privilege by Mobius's `ClanAccess` name (`"CASTLE_VAULT"`, ...; the leader holds all) |
| `player:active_weapon()`, `player:target()` | the right-hand weapon's item ID (nil without one), the target (Player, Npc or nil) |
| `player:is_affected_by_skill(id)`, `player:magic_skill_use(target, id, level, hit_ms, reuse_ms)` | a skill's effects on the player; the skill animation (shown only) |
| `player:pk_kills()`, `set_pk_kills(n)`, `update_user_info()`, `active_quests()` | PK count, UserInfo, the IDs of the started quests |
| `player:social_action(n)`, `teleport(x, y, z)`, `send_message(text)`, `send_system_message(SystemMessageId.X, ...)` | |
| `x()`, `y()`, `z()`, `heading()`, `is_inside_radius_2d(other, r)`, `is_inside_radius_3d(other, r)` | players and NPCs |
| `npc:object_id()`, `npc:id()`, `npc:name()` | integer, integer (the template ID), string |
| `npc:say(text)` | Says `text` in normal NPC chat to the players around. |
| `npc:broadcast_say(ChatType.X, text)`, `npc:show_chat_window(player)`, `npc:social_action(n)`, `npc:delete()` | |
| `npc:is_dead()`, `npc:get_script_value()`, `set_script_value(n)`, `is_script_value(n)`, `npc:is_type("Monster")` | |
| `npc:level()`, `npc:hp()`, `npc:max_hp()` | |
| `npc:set_running()`, `add_damage_hate(target, damage, hate)`, `attack(target)`, `remove_attacker(target)`, `abort_attack()`, `move_to(x, y, z)`, `teleport(x, y, z)`, `die(killer)` | the monster AI and the NPC as Mobius quests drive them |
| `npc:stop_respawn()`, `start_respawn()` | the NPC's spawn (a `q:add_spawn` NPC comes back where it was spawned) |
| `npc:castle()`, `npc:is_my_lord(player)` | the NPC's castle (Mobius `Npc.getCastle`; nil without one), whether the player leads its owner clan. A castle, called with `:`: `id()`, `name()`, `owner_id()`, `owner()` (a clan or nil), `tax_percent()`, `set_tax_percent(n)` (saved), `max_tax()` (15, Dawn 25, Dusk 5), `treasury()`, `deposit(player, n)` / `withdraw(player, n)` (adena between the player and the treasury, saved together; false when nothing moved), `siege_date()` (Unix seconds, 0 for none), `is_siege_in_progress()`, `is_registration_over()`, `is_zone_active()`, `control_tower_count()`, `get_function(type)` (a rented function `{level, lease, rate, end_time, end_day, end_month, end_year}` or nil), `seal_owner(seal)`, `seven_signs_period()`, `open_close_door(player, door_id, open)`, `door_upgrade_level(door_id)` (the door's HP ratio, nil for a door not of the castle), `set_door_upgrade(door_id, level)` (saved), `banish_foreigners()`, `oust_all_players()`, `show_siege_info(player)` (the SiegeInfo window) |
| `npc:show_buy_window(player, list_id[, apply_tax])`, `npc:broadcast_say_to_region(ChatType.X, text)` | a merchant's buylist (Mobius `Merchant.showBuyWindow`; `apply_tax` false shows no tax); NpcSay to every player in the NPC's map region |
| `npc:set_target(target)`, `cast(skills.get(id, level)[, target])`, `magic_skill_use(target, id, level, hit_ms, reuse_ms)`, `drop_item(player, id, count)` | skills and drops |

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

Quests are tested the same way, against a fake world (`h.World()`): fake players carry items, equipment, exp and their quest states (`hero.Quest("Q00001_LettersOfLove")`), and the world records sounds, social actions, messages, quest log updates and timer replies. `h.Kill(player, npc)` runs the kill handlers after Mobius's 2.5 s delay on the fake clock, `h.Advance(d)` runs due quest timers, `h.Start()` the `on_start` handlers, and `h.SetRand(rnd.NewScripted(t, rnd.Get(10, 3)))` forces the random draws (a draw the script does not expect fails the test):

```go
h := scripttest.Load(t, "../../scripts", "quests/Q00401_PathOfTheWarrior/Q00401_PathOfTheWarrior.lua")
hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 19}
auron := &scripttest.Npc{ObjectID: 2, ID: 30010}
h.Talk("Q00401_PathOfTheWarrior", hero, auron)
h.Event("Q00401_PathOfTheWarrior", "30010-06.htm", hero, auron)
h.SetRand(rnd.NewScripted(t, rnd.Get(10, 3)))
h.Kill(hero, &scripttest.Npc{ObjectID: 3, ID: 20035})
```

The walkthroughs of the hand-ported quests are in `internal/scripting/quests_test.go`.

The tests for the shipped scripts are in `internal/scripting/scripts_test.go` (the castle NPCs in `castle_scripts_test.go`) and, for the NPC scripts ported from Mobius, `internal/game/dialog_script_test.go`, which also runs scripts inside a game server.

Editor support
----

`types/l2go.lua` is a [Lua Language Server](https://luals.github.io/) definitions file (`---@meta`), and `.luarc.json` in the repository root sets Lua 5.1 and hides the libraries the sandbox removes. With the LuaLS extension (VS Code, Neovim, ...) and the repository opened as the workspace, you get autocomplete and type checks for the whole API.
