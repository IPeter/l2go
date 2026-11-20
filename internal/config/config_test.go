package config

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("Default().Validate() = %v", err)
	}
}

// The shipped example file must stay in sync with the built-in defaults.
func TestDefaultFileMatchesDefault(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "extra", "default_config_file", "server.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.Path = ""
	if want := Default(); !reflect.DeepEqual(cfg, want) {
		t.Errorf("default file = %+v\nwant %+v", cfg, want)
	}
}

func TestComposeFileIsValid(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "deploy", "compose", "server.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got := cfg.LoginServer.GameServerAddr(); got != "login:9413" {
		t.Errorf("GameServerAddr = %q, want login:9413", got)
	}
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("Load(missing) = %+v, want Default()", cfg)
	}
	if cfg.Path != "" {
		t.Errorf("Path = %q, want empty for defaults", cfg.Path)
	}
}

func TestLoadSetsPath(t *testing.T) {
	path := writeFile(t, `{"gameservers": []}`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Path != path {
		t.Errorf("Path = %q, want %q", cfg.Path, path)
	}
}

// B11: JSON errors used to be ignored and gave a zero config.
func TestLoadReportsJSONErrors(t *testing.T) {
	tests := map[string]string{
		"syntax":         `{"loginserver": {"host": "x",}}`,
		"wrong type":     `{"loginserver": {"host": 5}}`,
		"unknown field":  `{"loginserver": {"hots": "x"}}`,
		"trailing data":  `{} {}`,
		"empty":          ``,
		"removed cache":  `{"gameservers": [{"id": 1, "cache": {}}]}`,
		"bad duration":   `{"gameservers": [{"id": 1, "scripts": {"eventTimeout": "fast"}}]}`,
		"number timeout": `{"gameservers": [{"id": 1, "scripts": {"eventTimeout": 50}}]}`,
		"unknown script": `{"gameservers": [{"id": 1, "scripts": {"folder": "x"}}]}`,
		"unknown world":  `{"gameservers": [{"id": 1, "world": {"tick": "100ms"}}]}`,
		"bad tick":       `{"gameservers": [{"id": 1, "world": {"tickInterval": "often"}}]}`,
		"string range":   `{"gameservers": [{"id": 1, "chat": {"generalRange": "far"}}]}`,
		"old chat flag":  `{"gameservers": [{"id": 1, "chat": {"globalShout": true}}]}`,
		"bool chat mode": `{"gameservers": [{"id": 1, "chat": {"shout": true}}]}`,
		"unknown npcs":   `{"gameservers": [{"id": 1, "npcs": {"range": 1}}]}`,
		"unknown geo":    `{"gameservers": [{"id": 1, "geodata": {"path": "x"}}]}`,
		"unknown items":  `{"gameservers": [{"id": 1, "items": {"maxAdenaCount": 1}}]}`,
		"unknown rates":  `{"gameservers": [{"id": 1, "rates": {"RateXp": 1}}]}`,
		"string rate":    `{"gameservers": [{"id": 1, "rates": {"xp": "x10"}}]}`,
		"bad item rate":  `{"gameservers": [{"id": 1, "rates": {"dropAmountByItem": {"adena": 1}}}]}`,
		"item rate list": `{"gameservers": [{"id": 1, "rates": {"dropChanceByItem": "57,1"}}]}`,
		"seconds number": `{"gameservers": [{"id": 1, "pvp": {"pvpVsNormalTime": 120000}}]}`,
		"string item ID": `{"gameservers": [{"id": 1, "pvp": {"listOfPetItems": ["2375"]}}]}`,
		"unknown combat": `{"gameservers": [{"id": 1, "combat": {"altPartyRange": 1500}}]}`,
		"unknown econ":   `{"gameservers": [{"id": 1, "economy": {"customMultisellLoad": true}}]}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := writeFile(t, content)
			_, err := Load(path)
			if err == nil {
				t.Fatal("Load succeeded, want an error")
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("error %q does not name the file", err)
			}
		})
	}
}

func TestLoadUnreadableFile(t *testing.T) {
	// A directory cannot be read as a file; this must not fall back to defaults.
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("Load(directory) succeeded, want an error")
	}
}

func TestParseAppliesDefaults(t *testing.T) {
	cfg, err := Parse([]byte(`{"loginserver": {"host": "h"}}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	l := cfg.LoginServer
	if l.ClientListen != DefaultClientListen || l.GameServerListen != DefaultGameServerListen {
		t.Errorf("listen = %q, %q; want defaults", l.ClientListen, l.GameServerListen)
	}
	if !reflect.DeepEqual(l.InternalNetworks, DefaultInternalNetworks) {
		t.Errorf("InternalNetworks = %v, want %v", l.InternalNetworks, DefaultInternalNetworks)
	}

	// Scripts settings left out get their defaults.
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}, {"id": 2, "scripts": {"dir": "/srv/lua", "devReload": true, "eventTimeout": "2s"}}]}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := cfg.GameServers[0].Scripts, (Scripts{Dir: DefaultScriptsDir, EventTimeout: DefaultEventTimeout}); got != want {
		t.Errorf("default scripts = %+v, want %+v", got, want)
	}
	if got, want := cfg.GameServers[1].Scripts, (Scripts{Dir: "/srv/lua", DevReload: true, EventTimeout: Duration(2 * time.Second)}); got != want {
		t.Errorf("scripts = %+v, want %+v", got, want)
	}

	// An explicit empty list is kept: no client counts as internal.
	cfg, err = Parse([]byte(`{"loginserver": {"internalNetworks": []}}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if n := cfg.LoginServer.InternalNetworks; n == nil || len(n) != 0 {
		t.Errorf("InternalNetworks = %#v, want empty non-nil", n)
	}
}

// The game rule sections default to Mobius's values when left out, field by
// field, while explicit false and 0 are kept.
func TestParseGameSections(t *testing.T) {
	tests := []struct {
		name string
		json string
		want func(*GameServer)
	}{
		{"left out", `{"id": 1}`, func(*GameServer) {}},
		{"empty sections", `{"id": 1, "world": {}, "chat": {}, "npcs": {}, "geodata": {}}`, func(*GameServer) {}},
		{"some fields", `{"id": 1, "world": {"maxRunSpeed": 300}, "chat": {"trade": "global"}}`, func(g *GameServer) {
			g.World.MaxRunSpeed = 300
			g.Chat.Trade = ChatModeGlobal
		}},
		{"chat modes in Mobius case", `{"id": 1, "chat": {"shout": "OFF", "trade": " Gm "}}`, func(g *GameServer) {
			g.Chat.Shout = ChatModeOff
			g.Chat.Trade = ChatModeGM
		}},
		{"explicit zeros", `{"id": 1, "world": {"deleteCharAfterDays": 0, "autosaveInterval": "0s"}, "npcs": {"maxFreeTeleportLevel": 0}}`, func(g *GameServer) {
			g.World.DeleteCharAfterDays = 0
			g.World.AutosaveInterval = 0
			g.NPCs.MaxFreeTeleportLevel = 0
		}},
		{"everything", `{"id": 1,
			"world": {"_comment": "w", "tickInterval": "50ms", "autosaveInterval": "1m", "deleteCharAfterDays": 3, "maxRunSpeed": 200, "spawnProtection": "0s"},
			"chat": {"generalRange": 900, "shout": "global", "trade": "off", "maxLength": 80},
			"npcs": {"spawnCustom": true, "maxFreeTeleportLevel": 20, "interactionRange": 150},
			"geodata": {"dir": "/srv/geo"}}`, func(g *GameServer) {
			g.World.Comment, g.World.TickInterval, g.World.AutosaveInterval = "w", Duration(50*time.Millisecond), Duration(time.Minute)
			g.World.DeleteCharAfterDays, g.World.MaxRunSpeed, g.World.SpawnProtection = 3, 200, 0
			g.Chat = Chat{GeneralRange: 900, Shout: ChatModeGlobal, Trade: ChatModeOff, MaxLength: 80}
			g.NPCs.SpawnCustom, g.NPCs.MaxFreeTeleportLevel, g.NPCs.InteractionRange = true, 20, 150
			g.Geodata = Geodata{Dir: "/srv/geo"}
		}},
		{"every new key", `{"id": 1,
			"world": {"gridsAlwaysOn": true, "gridTurnOnTime": "2s", "gridTurnOffTime": "1m"},
			"npcs": {"defaultCorpseTime": "5s", "spoiledCorpseExtendTime": "0s", "maxAggroRange": 600, "maxDriftRange": 0,
				"mobAggroInPeaceZone": false, "aggroDistanceCheck": true, "aggroDistanceCheckRange": 3000, "attackablesCampPlayerCorpses": true},
			"items": {"_comment": "i", "allowDiscardItem": false, "autoDestroyDroppedItemAfter": "0s", "autoDestroyHerbTime": "30s", "listOfProtectedItems": [57, 1060],
				"destroyPlayerDroppedItem": true, "destroyEquipableItem": true, "multipleItemDrop": false, "maxAdena": 99999999999,
				"warehouseSlotsNoDwarf": 50, "warehouseSlotsDwarf": 60},
			"economy": {"altKarmaPlayerCanShop": false, "altKarmaPlayerCanUseWarehouse": false, "onlyGMItemsFree": false, "multisellAmountLimit": 1,
				"customMultisell": true, "floodDropItem": "0s", "floodTransaction": "500ms", "floodMultisell": "2s"},
			"rates": {"xp": 10, "sp": 2.5, "deathDropChance": 3, "deathDropAmount": 4, "herbDropChance": 0, "herbDropAmount": 0.5,
				"dropChanceByItem": {}, "dropAmountByItem": {"57": 5},
				"dropMaxOccurrencesNormal": 0, "dropAdenaMinLevelDifference": 1, "dropAdenaMaxLevelDifference": 2, "dropAdenaMinLevelGapChance": 0,
				"dropItemMinLevelDifference": 3, "dropItemMaxLevelDifference": 4, "dropItemMinLevelGapChance": 100,
				"monsterExpMaxLevelDifference": 0, "karmaLost": 2, "karmaExpLost": 0},
			"combat": {"randomizeAutoAttackDamage": false, "shieldBlocks": true, "perfectShieldBlockRate": 0, "cancelByHit": " ALL ", "distanceBowDamage": true,
				"hpRegenMultiplier": 200, "mpRegenMultiplier": 0, "cpRegenMultiplier": 50, "respawnRestoreHP": 100, "respawnRestoreCP": 10, "respawnRestoreMP": 20,
				"delevel": false, "deathPenaltyChance": 0, "partyRange": 1600, "randomRespawnInTown": false, "autoLoot": true, "autoLootHerbs": true},
			"pvp": {"pvpVsNormalTime": "1m", "pvpVsPvpTime": "0s", "minimumPkRequiredToDrop": 0, "karmaDropLimit": 1, "karmaRateDrop": 100,
				"karmaRateDropItem": 0, "karmaRateDropEquip": 1, "karmaRateDropEquipWeapon": 2, "listOfNonDroppableItems": [], "listOfPetItems": [1],
				"canGMDropEquipment": true, "awardPKKillPVPPoint": true, "karmaPlayerCanBeKilledInPeaceZone": true}}`, func(g *GameServer) {
			g.World.GridsAlwaysOn = true
			g.World.GridTurnOnTime = Duration(2 * time.Second)
			g.World.GridTurnOffTime = Duration(time.Minute)
			g.NPCs = NPCs{MaxFreeTeleportLevel: DefaultMaxFreeTeleportLevel, InteractionRange: DefaultNPCInteractionRange,
				DefaultCorpseTime: Duration(5 * time.Second), MaxAggroRange: 600,
				AggroDistanceCheck: true, AggroDistanceCheckRange: 3000, AttackablesCampPlayerCorpses: true}
			g.Items = Items{Comment: "i", AutoDestroyHerbTime: Duration(30 * time.Second), ListOfProtectedItems: []int{57, 1060},
				DestroyPlayerDroppedItem: true, DestroyEquipableItem: true, MaxAdena: 99999999999,
				WarehouseSlotsNoDwarf: 50, WarehouseSlotsDwarf: 60}
			g.Economy = Economy{MultisellAmountLimit: 1, CustomMultisell: true, FloodTransaction: Duration(500 * time.Millisecond), FloodMultisell: Duration(2 * time.Second)}
			g.Rates = Rates{XP: 10, SP: 2.5, DeathDropChance: 3, DeathDropAmount: 4, HerbDropAmount: 0.5,
				DropChanceByItem: ItemRates{}, DropAmountByItem: ItemRates{57: 5},
				DropAdenaMinLevelDifference: 1, DropAdenaMaxLevelDifference: 2,
				DropItemMinLevelDifference: 3, DropItemMaxLevelDifference: 4, DropItemMinLevelGapChance: 100,
				KarmaLost: 2}
			g.Combat = Combat{ShieldBlocks: true, CancelByHit: CancelByHitAll, DistanceBowDamage: true,
				HPRegenMultiplier: 200, CPRegenMultiplier: 50, RespawnRestoreHP: 100, RespawnRestoreCP: 10, RespawnRestoreMP: 20,
				PartyRange: 1600, AutoLoot: true, AutoLootHerbs: true}
			g.PvP = PvP{PvPVsNormalTime: Duration(time.Minute), KarmaDropLimit: 1, KarmaRateDrop: 100,
				KarmaRateDropEquip: 1, KarmaRateDropEquipWeapon: 2, ListOfNonDroppableItems: []int{}, ListOfPetItems: []int{1},
				CanGMDropEquipment: true, AwardPKKillPVPPoint: true, KarmaPlayerCanBeKilledInPeaceZone: true}
		}},
		// A map or list in the file replaces the default one; Mobius's
		// per-item rates override the general rate, so merging would keep
		// overrides the file left out.
		{"item rates replace the default map", `{"id": 1, "rates": {"dropAmountByItem": {"57": 2}}}`, func(g *GameServer) {
			g.Rates.DropAmountByItem = ItemRates{57: 2}
		}},
		{"item lists replace the defaults", `{"id": 1, "pvp": {"listOfPetItems": [4422]}}`, func(g *GameServer) {
			g.PvP.ListOfPetItems = []int{4422}
		}},
		{"explicit false and 0 in new sections", `{"id": 1, "combat": {"randomizeAutoAttackDamage": false, "respawnRestoreHP": 0}, "economy": {"onlyGMItemsFree": false}}`, func(g *GameServer) {
			g.Combat.RandomizeAutoAttackDamage = false
			g.Combat.RespawnRestoreHP = 0
			g.Economy.OnlyGMItemsFree = false
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse([]byte(`{"gameservers": [` + tt.json + `]}`))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			want := GameServer{
				World: DefaultWorld(), Chat: DefaultChat(), NPCs: DefaultNPCs(),
				Items: DefaultItems(), Economy: DefaultEconomy(), Rates: DefaultRates(), Combat: DefaultCombat(), PvP: DefaultPvP(),
			}
			tt.want(&want)
			got := cfg.GameServers[0]
			if got.World != want.World || got.Chat != want.Chat || got.NPCs != want.NPCs || got.Geodata != want.Geodata {
				t.Errorf("sections = %+v %+v %+v %+v\nwant %+v %+v %+v %+v",
					got.World, got.Chat, got.NPCs, got.Geodata, want.World, want.Chat, want.NPCs, want.Geodata)
			}
			if !reflect.DeepEqual(got.Items, want.Items) {
				t.Errorf("Items = %+v\nwant %+v", got.Items, want.Items)
			}
			if got.Economy != want.Economy {
				t.Errorf("Economy = %+v\nwant %+v", got.Economy, want.Economy)
			}
			if !reflect.DeepEqual(got.Rates, want.Rates) {
				t.Errorf("Rates = %+v\nwant %+v", got.Rates, want.Rates)
			}
			if got.Combat != want.Combat {
				t.Errorf("Combat = %+v\nwant %+v", got.Combat, want.Combat)
			}
			if !reflect.DeepEqual(got.PvP, want.PvP) {
				t.Errorf("PvP = %+v\nwant %+v", got.PvP, want.PvP)
			}
			// The other fields still get their usual defaults.
			if got.ID != 1 || got.DataDir != DefaultDataDir || got.Scripts.Dir != DefaultScriptsDir {
				t.Errorf("other fields: id %d dataDir %q scripts %q", got.ID, got.DataDir, got.Scripts.Dir)
			}
		})
	}
}

func TestDefaultGameSections(t *testing.T) {
	gs := Default().GameServers[0]
	want := World{
		TickInterval:        Duration(100 * time.Millisecond),
		AutosaveInterval:    Duration(15 * time.Minute),
		DeleteCharAfterDays: 7,
		MaxRunSpeed:         250,
		SpawnProtection:     Duration(10 * time.Minute),
		GridsAlwaysOn:       false,
		GridTurnOnTime:      Duration(time.Second),
		GridTurnOffTime:     Duration(90 * time.Second),
	}
	if gs.World != want {
		t.Errorf("World = %+v, want %+v", gs.World, want)
	}
	if want := (Chat{GeneralRange: 1250, Shout: ChatModeOn, Trade: ChatModeOn, MaxLength: 105}); gs.Chat != want {
		t.Errorf("Chat = %+v, want %+v", gs.Chat, want)
	}
	wantNPCs := NPCs{
		MaxFreeTeleportLevel:    40,
		InteractionRange:        250,
		DefaultCorpseTime:       Duration(7 * time.Second),
		SpoiledCorpseExtendTime: Duration(10 * time.Second),
		MaxAggroRange:           450,
		MaxDriftRange:           300,
		MobAggroInPeaceZone:     true,
		AggroDistanceCheckRange: 2000,
	}
	if gs.NPCs != wantNPCs {
		t.Errorf("NPCs = %+v, want %+v", gs.NPCs, wantNPCs)
	}
	if gs.Geodata != (Geodata{}) {
		t.Errorf("Geodata = %+v, want no geodata", gs.Geodata)
	}

	// The values of the Mobius Interlude .ini files.
	wantItems := Items{
		AllowDiscardItem:            true,
		AutoDestroyDroppedItemAfter: Duration(600 * time.Second),
		AutoDestroyHerbTime:         Duration(60 * time.Second),
		ListOfProtectedItems:        []int{},
		MultipleItemDrop:            true,
		MaxAdena:                    2000000000,
		WarehouseSlotsNoDwarf:       100,
		WarehouseSlotsDwarf:         120,
	}
	if !reflect.DeepEqual(gs.Items, wantItems) {
		t.Errorf("Items = %+v, want %+v", gs.Items, wantItems)
	}
	wantEconomy := Economy{
		AltKarmaPlayerCanShop:         true,
		AltKarmaPlayerCanUseWarehouse: true,
		OnlyGMItemsFree:               true,
		MultisellAmountLimit:          10000,
		FloodDropItem:                Duration(time.Second),
		FloodTransaction:              Duration(time.Second),
		FloodMultisell:                Duration(time.Second),
	}
	if gs.Economy != wantEconomy {
		t.Errorf("Economy = %+v, want %+v", gs.Economy, wantEconomy)
	}
	wantRates := Rates{
		XP: 1, SP: 1,
		DeathDropChance: 1, DeathDropAmount: 1, HerbDropChance: 1, HerbDropAmount: 1,
		DropChanceByItem:            ItemRates{57: 1},
		DropAmountByItem:            ItemRates{57: 1, 6656: 1, 6657: 1, 6658: 1, 6659: 1, 6660: 1, 6661: 1, 6662: 1, 8191: 1},
		DropMaxOccurrencesNormal:    2,
		DropAdenaMinLevelDifference: 8, DropAdenaMaxLevelDifference: 15, DropAdenaMinLevelGapChance: 10,
		DropItemMinLevelDifference: 5, DropItemMaxLevelDifference: 10, DropItemMinLevelGapChance: 10,
		MonsterExpMaxLevelDifference: 11,
		KarmaLost:                    -1,
		KarmaExpLost:                 1,
	}
	if !reflect.DeepEqual(gs.Rates, wantRates) {
		t.Errorf("Rates = %+v, want %+v", gs.Rates, wantRates)
	}
	wantCombat := Combat{
		RandomizeAutoAttackDamage: true,
		PerfectShieldBlockRate:    10,
		CancelByHit:               CancelByHitCast,
		HPRegenMultiplier:         100,
		MPRegenMultiplier:         100,
		CPRegenMultiplier:         100,
		RespawnRestoreHP:          65,
		Delevel:                   true,
		DeathPenaltyChance:        20,
		PartyRange:                1500,
		RandomRespawnInTown:       true,
	}
	if gs.Combat != wantCombat {
		t.Errorf("Combat = %+v, want %+v", gs.Combat, wantCombat)
	}
	wantPvP := PvP{
		PvPVsNormalTime:          Duration(120 * time.Second),
		PvPVsPvPTime:             Duration(60 * time.Second),
		MinimumPkRequiredToDrop:  6,
		KarmaDropLimit:           10,
		KarmaRateDrop:            40,
		KarmaRateDropItem:        50,
		KarmaRateDropEquip:       40,
		KarmaRateDropEquipWeapon: 10,
		ListOfNonDroppableItems:  []int{57, 1147, 425, 1146, 461, 10, 2368, 7, 6, 2370, 2369, 6842, 6611, 6612, 6613, 6614, 6615, 6616, 6617, 6618, 6619, 6620, 6621, 7694, 8181, 5575, 7694},
		ListOfPetItems:           []int{2375, 3500, 3501, 3502, 4422, 4423, 4424, 4425, 6648, 6649, 6650},
	}
	if !reflect.DeepEqual(gs.PvP, wantPvP) {
		t.Errorf("PvP = %+v, want %+v", gs.PvP, wantPvP)
	}
}

// Each call returns fresh maps and slices, so a game server that changes
// its config cannot change another's defaults.
func TestDefaultsDoNotShare(t *testing.T) {
	a, b := Default(), Default()
	a.GameServers[0].Rates.DropAmountByItem[57] = 9
	a.GameServers[0].PvP.ListOfPetItems[0] = 1
	a.GameServers[0].Items.ListOfProtectedItems = append(a.GameServers[0].Items.ListOfProtectedItems, 1)
	if b.GameServers[0].Rates.DropAmountByItem[57] != 1 || b.GameServers[0].PvP.ListOfPetItems[0] != 2375 ||
		len(b.GameServers[0].Items.ListOfProtectedItems) != 0 {
		t.Error("two Default() calls share maps or slices")
	}
}

func TestKarmaLostRate(t *testing.T) {
	r := DefaultRates()
	r.XP = 7
	if got := r.KarmaLostRate(); got != 7 {
		t.Errorf("KarmaLostRate with -1 = %v, want rates.xp 7", got)
	}
	r.KarmaLost = 2
	if got := r.KarmaLostRate(); got != 2 {
		t.Errorf("KarmaLostRate = %v, want 2", got)
	}
}

func TestParseAcceptsComments(t *testing.T) {
	_, err := Parse([]byte(`{"_comment": "a", "loginserver": {"_comment": "b"}, "gameservers": [{"_comment": "c"}]}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
		want   string // substring of the error; empty means valid
	}{
		{"default", func(*Config) {}, ""},
		{"empty host", func(c *Config) { c.LoginServer.Host = "" }, "loginserver.host"},
		{"bad client listen", func(c *Config) { c.LoginServer.ClientListen = "2106" }, "clientListen"},
		{"client listen port 0", func(c *Config) { c.LoginServer.ClientListen = ":0" }, "clientListen"},
		{"bad game server listen", func(c *Config) { c.LoginServer.GameServerListen = ":x" }, "gameServerListen"},
		{"bad network", func(c *Config) { c.LoginServer.InternalNetworks = []string{"10.0.0.0"} }, "internalNetworks"},
		{"login db no name", func(c *Config) { c.LoginServer.Database.Name = "" }, "loginserver.database"},
		{"login db no host", func(c *Config) { c.LoginServer.Database.Host = "" }, "loginserver.database"},
		{"login db bad port", func(c *Config) { c.LoginServer.Database.Port = 70000 }, "loginserver.database"},
		{"login db uri", func(c *Config) {
			c.LoginServer.Database = Database{Name: "x", URI: "mongodb://user:pw@db:27017"}
		}, ""},
		{"login db bad uri", func(c *Config) {
			c.LoginServer.Database = Database{Name: "x", URI: "http://db"}
		}, "loginserver.database"},
		{"id 0", func(c *Config) { c.GameServers[0].ID = 0 }, "gameservers[0].id"},
		{"id 256", func(c *Config) { c.GameServers[0].ID = 256 }, "gameservers[0].id"},
		{"duplicate id", func(c *Config) {
			c.GameServers = append(c.GameServers, c.GameServers[0])
		}, "used twice"},
		{"no name", func(c *Config) { c.GameServers[0].Name = " " }, "gameservers[0].name"},
		{"bad internal ip", func(c *Config) { c.GameServers[0].InternalIP = "localhost" }, "internalIP"},
		{"ipv6 external ip", func(c *Config) { c.GameServers[0].ExternalIP = "::1" }, "externalIP"},
		{"port 0", func(c *Config) { c.GameServers[0].Port = 0 }, "gameservers[0].port"},
		{"game db", func(c *Config) { c.GameServers[0].Database.Name = "" }, "gameservers[0].database"},
		{"no scripts dir", func(c *Config) { c.GameServers[0].Scripts.Dir = "" }, "gameservers[0].scripts.dir"},
		{"zero event timeout", func(c *Config) { c.GameServers[0].Scripts.EventTimeout = 0 }, "scripts.eventTimeout"},
		{"negative event timeout", func(c *Config) {
			c.GameServers[0].Scripts.EventTimeout = Duration(-time.Millisecond)
		}, "scripts.eventTimeout"},
		{"no game servers", func(c *Config) { c.GameServers = nil }, ""},
		{"zero tick", func(c *Config) { c.GameServers[0].World.TickInterval = 0 }, "gameservers[0].world.tickInterval"},
		{"negative autosave", func(c *Config) {
			c.GameServers[0].World.AutosaveInterval = Duration(-time.Second)
		}, "world.autosaveInterval"},
		{"autosave off", func(c *Config) { c.GameServers[0].World.AutosaveInterval = 0 }, ""},
		{"delete at once", func(c *Config) { c.GameServers[0].World.DeleteCharAfterDays = 0 }, ""},
		{"negative delete days", func(c *Config) { c.GameServers[0].World.DeleteCharAfterDays = -1 }, "world.deleteCharAfterDays"},
		{"huge delete days", func(c *Config) {
			c.GameServers[0].World.DeleteCharAfterDays = MaxDeleteCharAfterDays + 1
		}, "world.deleteCharAfterDays"},
		{"zero run speed", func(c *Config) { c.GameServers[0].World.MaxRunSpeed = 0 }, "world.maxRunSpeed"},
		{"spawn protection off", func(c *Config) { c.GameServers[0].World.SpawnProtection = 0 }, ""},
		{"negative spawn protection", func(c *Config) {
			c.GameServers[0].World.SpawnProtection = Duration(-time.Second)
		}, "world.spawnProtection"},
		{"zero chat range", func(c *Config) { c.GameServers[0].Chat.GeneralRange = 0 }, "chat.generalRange"},
		{"negative chat length", func(c *Config) { c.GameServers[0].Chat.MaxLength = -5 }, "chat.maxLength"},
		{"global shout", func(c *Config) { c.GameServers[0].Chat.Shout = ChatModeGlobal }, ""},
		{"trade for GMs", func(c *Config) { c.GameServers[0].Chat.Trade = ChatModeGM }, ""},
		{"shout off", func(c *Config) { c.GameServers[0].Chat.Shout = ChatModeOff }, ""},
		{"unknown shout mode", func(c *Config) { c.GameServers[0].Chat.Shout = "everyone" }, "chat.shout"},
		{"no trade mode", func(c *Config) { c.GameServers[0].Chat.Trade = "" }, "chat.trade"},
		{"no free teleports", func(c *Config) { c.GameServers[0].NPCs.MaxFreeTeleportLevel = 0 }, ""},
		{"negative free teleport level", func(c *Config) {
			c.GameServers[0].NPCs.MaxFreeTeleportLevel = -1
		}, "npcs.maxFreeTeleportLevel"},
		{"zero interaction range", func(c *Config) { c.GameServers[0].NPCs.InteractionRange = 0 }, "npcs.interactionRange"},
		{"geodata dir", func(c *Config) { c.GameServers[0].Geodata.Dir = "/srv/geodata" }, ""},
		{"negative grid turn on", func(c *Config) {
			c.GameServers[0].World.GridTurnOnTime = Duration(-time.Second)
		}, "world.gridTurnOnTime"},
		{"negative grid turn off", func(c *Config) {
			c.GameServers[0].World.GridTurnOffTime = Duration(-time.Second)
		}, "world.gridTurnOffTime"},
		{"grids at once", func(c *Config) {
			c.GameServers[0].World.GridTurnOnTime, c.GameServers[0].World.GridTurnOffTime = 0, 0
		}, ""},
		{"negative corpse time", func(c *Config) {
			c.GameServers[0].NPCs.DefaultCorpseTime = Duration(-time.Second)
		}, "npcs.defaultCorpseTime"},
		{"negative spoil extend", func(c *Config) {
			c.GameServers[0].NPCs.SpoiledCorpseExtendTime = Duration(-time.Second)
		}, "npcs.spoiledCorpseExtendTime"},
		{"negative aggro range", func(c *Config) { c.GameServers[0].NPCs.MaxAggroRange = -1 }, "npcs.maxAggroRange"},
		{"negative drift range", func(c *Config) { c.GameServers[0].NPCs.MaxDriftRange = -1 }, "npcs.maxDriftRange"},
		{"zero aggro check range", func(c *Config) { c.GameServers[0].NPCs.AggroDistanceCheckRange = 0 }, "npcs.aggroDistanceCheckRange"},
		{"items never destroyed", func(c *Config) { c.GameServers[0].Items.AutoDestroyDroppedItemAfter = 0 }, ""},
		{"negative auto destroy", func(c *Config) {
			c.GameServers[0].Items.AutoDestroyDroppedItemAfter = Duration(-time.Second)
		}, "items.autoDestroyDroppedItemAfter"},
		{"negative herb time", func(c *Config) {
			c.GameServers[0].Items.AutoDestroyHerbTime = Duration(-time.Second)
		}, "items.autoDestroyHerbTime"},
		{"negative protected item", func(c *Config) {
			c.GameServers[0].Items.ListOfProtectedItems = []int{57, -1}
		}, "items.listOfProtectedItems"},
		{"zero max adena", func(c *Config) { c.GameServers[0].Items.MaxAdena = 0 }, "items.maxAdena"},
		{"zero warehouse", func(c *Config) { c.GameServers[0].Items.WarehouseSlotsNoDwarf = 0 }, "items.warehouseSlotsNoDwarf"},
		{"negative dwarf warehouse", func(c *Config) { c.GameServers[0].Items.WarehouseSlotsDwarf = -1 }, "items.warehouseSlotsDwarf"},
		{"zero multisell limit", func(c *Config) { c.GameServers[0].Economy.MultisellAmountLimit = 0 }, "economy.multisellAmountLimit"},
		{"no flood limit", func(c *Config) { c.GameServers[0].Economy.FloodDropItem = 0 }, ""},
		{"negative flood drop", func(c *Config) {
			c.GameServers[0].Economy.FloodDropItem = Duration(-time.Second)
		}, "economy.floodDropItem"},
		{"negative flood transaction", func(c *Config) {
			c.GameServers[0].Economy.FloodTransaction = Duration(-time.Second)
		}, "economy.floodTransaction"},
		{"negative flood multisell", func(c *Config) {
			c.GameServers[0].Economy.FloodMultisell = Duration(-time.Second)
		}, "economy.floodMultisell"},
		{"x10 rates", func(c *Config) { c.GameServers[0].Rates.XP, c.GameServers[0].Rates.SP = 10, 10 }, ""},
		{"negative xp", func(c *Config) { c.GameServers[0].Rates.XP = -1 }, "rates.xp"},
		{"NaN sp", func(c *Config) { c.GameServers[0].Rates.SP = math.NaN() }, "rates.sp"},
		{"negative drop chance", func(c *Config) { c.GameServers[0].Rates.DeathDropChance = -1 }, "rates.deathDropChance"},
		{"negative drop amount", func(c *Config) { c.GameServers[0].Rates.DeathDropAmount = -1 }, "rates.deathDropAmount"},
		{"negative herb chance", func(c *Config) { c.GameServers[0].Rates.HerbDropChance = -1 }, "rates.herbDropChance"},
		{"negative herb amount", func(c *Config) { c.GameServers[0].Rates.HerbDropAmount = -1 }, "rates.herbDropAmount"},
		{"item rate for item 0", func(c *Config) {
			c.GameServers[0].Rates.DropChanceByItem = ItemRates{0: 1}
		}, "rates.dropChanceByItem"},
		{"negative item rate", func(c *Config) {
			c.GameServers[0].Rates.DropAmountByItem = ItemRates{57: -2}
		}, "rates.dropAmountByItem[57]"},
		{"no item rates", func(c *Config) { c.GameServers[0].Rates.DropAmountByItem = nil }, ""},
		{"negative max occurrences", func(c *Config) {
			c.GameServers[0].Rates.DropMaxOccurrencesNormal = -1
		}, "rates.dropMaxOccurrencesNormal"},
		{"adena gap max not above min", func(c *Config) {
			c.GameServers[0].Rates.DropAdenaMaxLevelDifference = 8
		}, "rates.dropAdenaMaxLevelDifference"},
		{"negative adena gap min", func(c *Config) {
			c.GameServers[0].Rates.DropAdenaMinLevelDifference = -1
		}, "rates.dropAdenaMinLevelDifference"},
		{"adena gap chance over 100", func(c *Config) {
			c.GameServers[0].Rates.DropAdenaMinLevelGapChance = 101
		}, "rates.dropAdenaMinLevelGapChance"},
		{"item gap max below min", func(c *Config) {
			c.GameServers[0].Rates.DropItemMaxLevelDifference = 4
		}, "rates.dropItemMaxLevelDifference"},
		{"negative item gap min", func(c *Config) {
			c.GameServers[0].Rates.DropItemMinLevelDifference = -1
		}, "rates.dropItemMinLevelDifference"},
		{"negative item gap chance", func(c *Config) {
			c.GameServers[0].Rates.DropItemMinLevelGapChance = -1
		}, "rates.dropItemMinLevelGapChance"},
		{"negative exp level difference", func(c *Config) {
			c.GameServers[0].Rates.MonsterExpMaxLevelDifference = -1
		}, "rates.monsterExpMaxLevelDifference"},
		{"zero karma lost", func(c *Config) { c.GameServers[0].Rates.KarmaLost = 0 }, "rates.karmaLost"},
		{"karma lost -1 with xp 0", func(c *Config) { c.GameServers[0].Rates.XP = 0 }, "rates.karmaLost"},
		{"karma lost 0.5", func(c *Config) { c.GameServers[0].Rates.KarmaLost = 0.5 }, ""},
		{"negative karma exp lost", func(c *Config) { c.GameServers[0].Rates.KarmaExpLost = -1 }, "rates.karmaExpLost"},
		{"perfect block over 100", func(c *Config) {
			c.GameServers[0].Combat.PerfectShieldBlockRate = 101
		}, "combat.perfectShieldBlockRate"},
		{"cancel by bow", func(c *Config) { c.GameServers[0].Combat.CancelByHit = CancelByHitBow }, ""},
		{"cancel by nothing", func(c *Config) { c.GameServers[0].Combat.CancelByHit = CancelByHitNone }, ""},
		{"unknown cancel by hit", func(c *Config) { c.GameServers[0].Combat.CancelByHit = "skill" }, "combat.cancelByHit"},
		{"negative hp regen", func(c *Config) { c.GameServers[0].Combat.HPRegenMultiplier = -1 }, "combat.hpRegenMultiplier"},
		{"negative mp regen", func(c *Config) { c.GameServers[0].Combat.MPRegenMultiplier = -1 }, "combat.mpRegenMultiplier"},
		{"infinite cp regen", func(c *Config) {
			c.GameServers[0].Combat.CPRegenMultiplier = math.Inf(1)
		}, "combat.cpRegenMultiplier"},
		{"restore hp over 100", func(c *Config) { c.GameServers[0].Combat.RespawnRestoreHP = 101 }, "combat.respawnRestoreHP"},
		{"negative restore cp", func(c *Config) { c.GameServers[0].Combat.RespawnRestoreCP = -1 }, "combat.respawnRestoreCP"},
		{"restore mp NaN", func(c *Config) { c.GameServers[0].Combat.RespawnRestoreMP = math.NaN() }, "combat.respawnRestoreMP"},
		{"death penalty over 100", func(c *Config) { c.GameServers[0].Combat.DeathPenaltyChance = 101 }, "combat.deathPenaltyChance"},
		{"zero party range", func(c *Config) { c.GameServers[0].Combat.PartyRange = 0 }, "combat.partyRange"},
		{"negative pvp time", func(c *Config) {
			c.GameServers[0].PvP.PvPVsNormalTime = Duration(-time.Second)
		}, "pvp.pvpVsNormalTime"},
		{"negative pvp vs pvp time", func(c *Config) {
			c.GameServers[0].PvP.PvPVsPvPTime = Duration(-time.Second)
		}, "pvp.pvpVsPvpTime"},
		{"negative pk to drop", func(c *Config) { c.GameServers[0].PvP.MinimumPkRequiredToDrop = -1 }, "pvp.minimumPkRequiredToDrop"},
		{"negative karma drop limit", func(c *Config) { c.GameServers[0].PvP.KarmaDropLimit = -1 }, "pvp.karmaDropLimit"},
		{"karma drop over 100", func(c *Config) { c.GameServers[0].PvP.KarmaRateDrop = 101 }, "pvp.karmaRateDrop"},
		{"karma item drop negative", func(c *Config) { c.GameServers[0].PvP.KarmaRateDropItem = -1 }, "pvp.karmaRateDropItem"},
		{"karma equip drop over 100", func(c *Config) { c.GameServers[0].PvP.KarmaRateDropEquip = 101 }, "pvp.karmaRateDropEquip"},
		{"karma weapon drop over 100", func(c *Config) {
			c.GameServers[0].PvP.KarmaRateDropEquipWeapon = 101
		}, "pvp.karmaRateDropEquipWeapon"},
		{"negative non-droppable item", func(c *Config) {
			c.GameServers[0].PvP.ListOfNonDroppableItems = []int{-57}
		}, "pvp.listOfNonDroppableItems"},
		{"negative pet item", func(c *Config) { c.GameServers[0].PvP.ListOfPetItems = []int{-1} }, "pvp.listOfPetItems"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(&cfg)
			err := cfg.Validate()
			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate() = %v, want nil", err)
			case tt.want != "" && err == nil:
				t.Fatalf("Validate() = nil, want error containing %q", tt.want)
			case tt.want != "" && !strings.Contains(err.Error(), tt.want):
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	cfg := Default()
	cfg.LoginServer.Host = ""
	cfg.GameServers[0].Port = 0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() = nil")
	}
	for _, want := range []string{"loginserver.host", "gameservers[0].port"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q is missing %q", err, want)
		}
	}
}

func TestMongoURI(t *testing.T) {
	tests := []struct {
		db   Database
		want string
	}{
		{Database{Host: "127.0.0.1", Port: 27017}, "mongodb://127.0.0.1:27017"},
		{Database{Host: "mongo", Port: 1, User: "u", Password: "p@ss"}, "mongodb://u:p%40ss@mongo:1"},
		{Database{Host: "::1", Port: 27017}, "mongodb://[::1]:27017"},
		{Database{URI: "mongodb+srv://x", Host: "ignored"}, "mongodb+srv://x"},
	}
	for _, tt := range tests {
		if got := tt.db.MongoURI(); got != tt.want {
			t.Errorf("%+v.MongoURI() = %q, want %q", tt.db, got, tt.want)
		}
	}
}

func TestAddresses(t *testing.T) {
	l := LoginServer{Host: "login.example", GameServerListen: "0.0.0.0:9500"}
	if got, want := l.GameServerAddr(), "login.example:9500"; got != want {
		t.Errorf("GameServerAddr = %q, want %q", got, want)
	}
	if got, want := (GameServer{Port: 7777}).ListenAddr(), ":7777"; got != want {
		t.Errorf("ListenAddr = %q, want %q", got, want)
	}
}

func TestGameServerByID(t *testing.T) {
	cfg := Default()
	cfg.GameServers = append(cfg.GameServers, GameServer{ID: 5, Name: "Five"})
	if gs, ok := cfg.GameServerByID(5); !ok || gs.Name != "Five" {
		t.Errorf("GameServerByID(5) = %+v, %v", gs, ok)
	}
	if _, ok := cfg.GameServerByID(2); ok {
		t.Error("GameServerByID(2) found a server")
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join(".l2go", "config", "server.json")) {
		t.Errorf("DefaultPath = %q", path)
	}
}

func TestDurationText(t *testing.T) {
	var d Duration
	if err := d.UnmarshalText([]byte("1m30s")); err != nil || time.Duration(d) != 90*time.Second {
		t.Errorf("UnmarshalText(1m30s) = %v, %v", time.Duration(d), err)
	}
	if err := d.UnmarshalText([]byte("soon")); err == nil {
		t.Error("UnmarshalText(soon) succeeded")
	}
	if b, err := DefaultEventTimeout.MarshalText(); err != nil || string(b) != "50ms" {
		t.Errorf("MarshalText = %q, %v; want 50ms", b, err)
	}
}

func TestErrorsWrap(t *testing.T) {
	_, err := Load(t.TempDir())
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		t.Errorf("Load(directory) error %v does not wrap *os.PathError", err)
	}
}
