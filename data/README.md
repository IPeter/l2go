# Game data

The game server loads these files at startup with `internal/data`, from the
folder set by `dataDir` in its config (default `data`, relative to the
working directory; the Docker image has them in `/opt/l2go/data`). It does
not start if a file is missing or invalid. The files are generated; do not
edit them by hand. A file over 10 MB is stored gzipped (`name.json.gz`); the
loaders read either form.

Player and item data (`data.Load`):

| File | Content |
|---|---|
| `players/classes.json` | All 89 player classes: name, race, parent class, base stats, combat values, speeds, slot defences, collision sizes, creation points, and HP/MP/CP and their regeneration for levels 1-80 |
| `players/initial_equipment.json` | Items each starting class receives, and whether they are equipped |
| `players/stat_bonus.json` | Bonus of each base attribute value 0-99 (STR, INT, DEX, WIT, CON, MEN) |
| `players/experience.json` | Experience per level, maximum player and pet level |
| `players/experience_loss.json`, `players/karma_loss.json` | Experience lost on death and karma loss modifier per level |
| `players/hit_condition_bonus.json` | Accuracy bonus by attack position and condition |
| `items/items.json` | All 9,208 items, one per line: type, body part, item list types, grade, weight, price, flags, weapon/armor/etc values, stat functions, skills and conditions |

World data (`data.LoadWorld`):

| File | Content |
|---|---|
| `npcs/npcs.json` | All 6,519 NPC and monster templates (custom and fake-player NPCs left out), one per line: display ID, level, type, name and title, race, sex, equipment, rewards, base stats, speeds, collision, flags and behaviour settings, plus the skills, behaviour parameters and drops kept for later milestones |
| `npcs/types.json` | The Mobius NPC classes (`Folk`, `Teleporter`, `Monster`, ...) and the classes they extend, so code can check an NPC's type the way Mobius uses `instanceof` |
| `spawns/spawns.json` | The spawn lists, one spawn block per line: its NPC entries with a fixed point or a territory, how many to keep alive, respawn delay and random range, day or night only, and behaviour overrides |
| `spawns/territories.json` | The spawn territories, one per line: shape (polygon with a Z band) and the areas inside where nothing spawns |
| `teleporters/teleporters.json` | The teleport lists of each gatekeeper: list type, destinations and fees |
| `zones/zones.json` | All zones, one per line: type (water, peace, town, ...), shape (polygon, cuboid or cylinder), parameters and spawn points |
| `zones/types.json` | The Mobius zone classes and the classes they extend |
| `mapregions/mapregions.json` | The map regions: town, restart points, the map cells each covers, and races that restart elsewhere |

Economy data (`data.LoadEconomy`, which also checks the items and NPCs it
names when given the item and world data):

| File | Content |
|---|---|
| `buylists/buylists.json` | All 616 NPC shops, each list on a line followed by its products one per line: the NPCs that may open it (none: GM shop) and, per product, the item, its price before tax (the item's reference price when the list gives none) and, for limited items, the stock and restock delay |
| `multisell/multisell.json` | The 90 multisell lists (custom ones left out), each list on a line followed by its entries one per line: the NPCs that may open it, whether it applies taxes and keeps enchant levels, and the ingredients and products of each entry (item or special points ID, count, enchant level, tax and kept-ingredient flags) |
| `merchant/price_configs.json` | The 27 merchant tax regions, one per line: base tax, the castle whose tax applies too, the town zone whose merchants use it, and which one is the default |

Skill data (`data.LoadSkills`, which resolves all 15,311 skill levels and
14,550 enchant levels at load; `CheckItems`, `CheckItemSkills` and
`CheckNpcSkills` check the references from and to the item and NPC data):

| File | Content |
|---|---|
| `skills/skills.json` | All 2,694 skills (custom ones left out), one per line, kept as Mobius writes them: the `<table>` columns, the values (literals or table references: operate and target type, MP/HP cost, hit, cool and reuse times, power, magic level, abnormal type, level and time, ...), the condition blocks as and/or/not trees of typed conditions with their messages, the effects by scope with their parameters and stat functions, and the enchant routes (levels 101-130 and 141-170) with the values and blocks they replace |
| `skilltrees/skilltrees.json` | The 95 skill trees, each tree's header on a line followed by its skills one per line: the 89 class trees with the parent class each extends, and the fishing, pledge (clan), noble, hero and game master trees; per skill the level needed, SP, auto-get and learned-by flags, required items, races and social class |
| `armorsets/armorsets.json` | The 51 armor sets, one per line: chest and the other parts, the skills of the full set, with a shield and at +6, and the stat bonuses |

Passive skills (`data.LoadPassives`) and categories (`data.LoadCategories`):

| File | Content |
|---|---|
| `skills/passives.json` | 59 passive skills, one per line: those NPC templates name that have effects, plus Weight Penalty (4270) and Death Penalty (5076); per level, the stat functions of the Buff effects and the DefenceTrait and AttackTrait values as the XML writes them; effects left out are named per skill (none in the current datapack) |
| `categories/categories.json` | The 160 categories (`FIGHTER_GROUP`, `MAGE_GROUP`, ...), one per line, with their class (or NPC) IDs |

The JSON schema is defined by the types in `internal/data`. The starting
class templates character creation uses are built from `classes.json`.

## Dialog pages: `html/`

`html/` holds the pages NPCs show: about 3,100 dialog pages from Mobius's
`dist/game/data/html/`, and about 11,300 quest and script pages from
`dist/game/data/scripts/`, kept with their paths under `html/scripts/`. The
game server reads them from `<dataDir>/html`.

Unlike the JSON files, the pages are not committed: they are NCSoft's
dialogue text, not functional game values. The folder is ignored by git, and
`make data` copies it from the Mobius checkout (see below). The Docker image
includes it when it exists; without it, NPCs have no dialogs to show.

## Source and license

The files are generated by `cmd/datagen` from the L2J Mobius Interlude
datapack (`L2J_Mobius_CT_0_Interlude`):

- `dist/game/data/stats/players/templates/**/*.xml`, `classList.xml`,
  `initialEquipment.xml`, `experience.xml`, `experienceLoss.xml` and
  `karmaLoss.xml` in `dist/game/data/stats/players/`
- `dist/game/data/stats/statBonus.xml` and `hitConditionBonus.xml`
- `dist/game/data/stats/items/*.xml` (not `custom/`)
- The race and parent of each class, which Mobius defines in code:
  `java/org/l2jmobius/gameserver/entity/actor/enums/player/PlayerClass.java`
  and `.../enums/creature/Race.java`
- `dist/game/data/stats/npcs/*.xml` (not `custom/`, no fake players)
- `dist/game/data/spawns/**/*.xml` and `dist/game/data/teleporters/**/*.xml`
- `dist/game/data/zones/*.xml` and `dist/game/data/mapregion/*.xml`
- `dist/game/data/buylists/*.xml`, `dist/game/data/multisell/*.xml` (not
  `custom/`) and `dist/game/data/MerchantPriceConfig.xml`
- `dist/game/data/stats/skills/*.xml` (not `custom/`, which Mobius loads only
  with `CustomSkillsLoad`), `dist/game/data/stats/players/skillTrees/**/*.xml`
  and `dist/game/data/stats/armorsets/*.xml`
- `dist/game/data/CategoryData.xml`
- The skill enums (abnormal, target and operate types, effect scopes,
  traits, abnormal visual effects, social classes, instance and category
  types), from their Java sources in `java/org/l2jmobius/gameserver/`; the
  effect names, from the handler list in
  `dist/game/data/scripts/handlers/EffectMasterHandler.java`; and the
  condition names, from `java/org/l2jmobius/gameserver/util/DocumentBase.java`
- The NPC and zone classes, from the Java sources in
  `java/org/l2jmobius/gameserver/entity/actor/instance/` and
  `.../entity/zone/type/`
- `html/`: copies of `dist/game/data/html/**` and of the `.htm` and `.html`
  files in `dist/game/data/scripts/**`

L2J Mobius source files are licensed under MIT or GPLv3, as stated in each
file header. Its XML data files carry no header; they describe functional
game values (stats, coordinates, item IDs and names) that the Lineage II
client expects. L2Go is licensed under GPLv3.

## Regenerating

Clone L2J Mobius into `L2J_Mobius/` (it is ignored by git), then run from the
repository root:

```sh
make data
```

It runs `go run ./cmd/datagen -enums` (the Go enums of `internal/data`, from
the Mobius sources; `go generate ./internal/data` does the same) and then
`go run ./cmd/datagen` (every file of `data/`, `html/` included). To read
another checkout: `make data MOBIUS=/path/to/L2J_Mobius_CT_0_Interlude`.

`go run ./cmd/datagen -list` prints the domains (`categories`, `economy`,
`html`, `items`, `passives`, `players`, `skills`, `world`); naming some (for example `go run ./cmd/datagen html` or
`go run ./cmd/datagen players items`) regenerates only those. The output is
sorted and does not depend on the time or file order, so running it again on
the same datapack gives identical files. Commit the JSON files; `html/` stays
out of git.
