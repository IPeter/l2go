package data

import (
	"fmt"
	"strings"
)

// NpcTemplate is an NPC or monster template, from the Mobius NPC data
// (dist/game/data/stats/npcs/*.xml, not custom/). Values are resolved as
// Mobius resolves them (NpcData, NpcTemplate and CreatureTemplate): an
// attribute missing in the XML holds the Mobius default, so a zero value
// here is a real zero. Two exceptions depend on server configuration and
// stay raw: CorpseTime (0: the configured default) and AI.AggroRange
// (Mobius caps it for monsters at its MaxAggroRange setting).
//
// Mobius keeps some numbers in smaller types than the XML: HP, MP and their
// regeneration as float32, and P./M. attack and defence as int (truncated).
// The values here are the XML's; the stat code converts them.
type NpcTemplate struct {
	ID int `json:"id"`
	// DisplayID is the ID the client shows (the NPC's own ID by default).
	DisplayID int `json:"displayId"`
	Level     int `json:"level"`
	// Type is the Mobius class of the NPC (Monster, Folk, Teleporter, ...),
	// one of the usable classes of NpcTypesFile.
	Type                 string `json:"type"`
	Name                 string `json:"name,omitempty"`
	UsingServerSideName  bool   `json:"usingServerSideName,omitempty"`
	Title                string `json:"title,omitempty"`
	UsingServerSideTitle bool   `json:"usingServerSideTitle,omitempty"`
	// Race is RaceNone by default; Sex is MALE, FEMALE or ETC (the
	// default).
	Race Race   `json:"race"`
	Sex  string `json:"sex"`

	Equipment NpcEquipment `json:"equipment,omitzero"`
	Acquire   NpcAcquire   `json:"acquire,omitzero"`
	Stats     NpcStats     `json:"stats"`
	Status    NpcStatus    `json:"status"`
	Collision NpcCollision `json:"collision"`
	AI        NpcAI        `json:"ai"`
	Shots     NpcShots     `json:"shots,omitzero"`
	// CorpseTime is how long the corpse stays, in seconds; 0 means the
	// configured default (Mobius DefaultCorpseTime, 7).
	CorpseTime int `json:"corpseTime,omitempty"`
	// ExCrtEffect and SNpcPropHpRate are read but unused by Mobius.
	ExCrtEffect    bool    `json:"exCrtEffect,omitempty"`
	SNpcPropHpRate float64 `json:"sNpcPropHpRate,omitempty"`

	// Parameters, Skills and Drops are kept for later milestones (AI,
	// skills, drops); they are not used yet.
	Parameters NpcParameters `json:"parameters,omitzero"`
	Skills     []NpcSkill    `json:"skills,omitempty"`
	Drops      NpcDrops      `json:"drops,omitzero"`
}

// NpcEquipment holds the item IDs the NPC shows (0: none).
type NpcEquipment struct {
	Chest         int `json:"chest,omitempty"`
	RHand         int `json:"rhand,omitempty"`
	LHand         int `json:"lhand,omitempty"`
	WeaponEnchant int `json:"weaponEnchant,omitempty"`
}

// NpcAcquire is the reward for killing the NPC.
type NpcAcquire struct {
	Exp        float64 `json:"exp,omitempty"`
	SP         float64 `json:"sp,omitempty"`
	RaidPoints float64 `json:"raidPoints,omitempty"`
}

// NpcStats are the base stats of the NPC. NpcData reads no magic attack
// speed, magic critical rate or breath for NPCs, so every NPC has the
// CreatureTemplate defaults: 333, 5 and 100.
type NpcStats struct {
	STR int `json:"str"`
	INT int `json:"int"`
	DEX int `json:"dex"`
	WIT int `json:"wit"`
	CON int `json:"con"`
	MEN int `json:"men"`

	Vitals    NpcVitals    `json:"vitals"`
	Attack    NpcAttack    `json:"attack"`
	Defence   NpcDefence   `json:"defence"`
	Speed     NpcSpeed     `json:"speed"`
	Attribute NpcAttribute `json:"attribute,omitzero"`
	// HitTime is the attack animation time in ms, 0 when not given
	// (Mobius reads it but does not use it yet).
	HitTime int `json:"hitTime,omitempty"`
}

// NpcVitals are the maximum HP and MP and their regeneration per tick.
type NpcVitals struct {
	HP      float64 `json:"hp"`
	HPRegen float64 `json:"hpRegen"`
	MP      float64 `json:"mp"`
	MPRegen float64 `json:"mpRegen"`
}

// NpcAttack holds the attack stats. Defaults: Critical 4, Speed 300, Type
// FIST, Range 40, Width 120.
type NpcAttack struct {
	Physical float64 `json:"physical"`
	Magical  float64 `json:"magical"`
	// Random is the random damage spread.
	Random   int     `json:"random"`
	Critical int     `json:"critical"`
	Accuracy float64 `json:"accuracy"`
	// Speed is the attack speed.
	Speed      int `json:"speed"`
	ReuseDelay int `json:"reuseDelay,omitempty"`
	// Type is the weapon type the NPC attacks with (BOW makes it ranged).
	Type     WeaponType `json:"type"`
	Range    int        `json:"range"`
	Distance int        `json:"distance,omitempty"`
	// Width is the attack angle in degrees.
	Width int `json:"width"`
}

// NpcDefence holds the defence stats.
type NpcDefence struct {
	Physical   float64 `json:"physical"`
	Magical    float64 `json:"magical"`
	Evasion    int     `json:"evasion,omitempty"`
	Shield     int     `json:"shield,omitempty"`
	ShieldRate int     `json:"shieldRate,omitempty"`
}

// NpcSpeed holds the move speeds. Mobius defaults: walk 50, run 120, a
// ground speed of 0 or less becomes 0.1, and swim and fly speeds default
// to the ground speeds.
type NpcSpeed struct {
	Walk     float64 `json:"walk"`
	Run      float64 `json:"run"`
	SwimWalk float64 `json:"swimWalk"`
	SwimRun  float64 `json:"swimRun"`
	FlyWalk  float64 `json:"flyWalk"`
	FlyRun   float64 `json:"flyRun"`
}

// NpcAttribute holds elemental attack and defence (unused in Interlude
// data, kept for completeness).
type NpcAttribute struct {
	// AttackType is FIRE, WATER, WIND, EARTH, DARK or HOLY, or empty.
	AttackType  string `json:"attackType,omitempty"`
	AttackValue int    `json:"attackValue,omitempty"`
	Fire        int    `json:"fire,omitempty"`
	Water       int    `json:"water,omitempty"`
	Wind        int    `json:"wind,omitempty"`
	Earth       int    `json:"earth,omitempty"`
	Holy        int    `json:"holy,omitempty"`
	Dark        int    `json:"dark,omitempty"`
	Default     int    `json:"default,omitempty"`
}

// NpcStatus holds the NPC's flags, with Mobius defaults resolved:
// attackable, targetable, talkable, showName and randomAnimation default to
// true; undying defaults to true except for the Monster, RaidBoss and
// GrandBoss types; randomWalk defaults to true except for Guard; canMove is
// also true when the walk speed is 0.1 or less, as Mobius computes it.
type NpcStatus struct {
	Unique          bool `json:"unique,omitempty"`
	Attackable      bool `json:"attackable"`
	Targetable      bool `json:"targetable"`
	Talkable        bool `json:"talkable"`
	Undying         bool `json:"undying"`
	ShowName        bool `json:"showName"`
	RandomWalk      bool `json:"randomWalk"`
	RandomAnimation bool `json:"randomAnimation"`
	Flying          bool `json:"flying,omitempty"`
	CanMove         bool `json:"canMove"`
	NoSleepMode     bool `json:"noSleepMode,omitempty"`
	PassableDoor    bool `json:"passableDoor,omitempty"`
	HasSummoner     bool `json:"hasSummoner,omitempty"`
	CanBeSown       bool `json:"canBeSown,omitempty"`
}

// NpcCollision is the NPC's collision cylinder; the grown values are used
// when it is enlarged.
type NpcCollision struct {
	Radius      float64 `json:"radius"`
	RadiusGrown float64 `json:"radiusGrown,omitempty"`
	Height      float64 `json:"height"`
	HeightGrown float64 `json:"heightGrown,omitempty"`
}

// NpcAI holds the AI settings, kept for the AI milestone. Defaults: Type
// FIGHTER, IsAggressive true, MinSkillChance 7, MaxSkillChance 15.
type NpcAI struct {
	// Type is a Mobius AIType constant (FIGHTER, ARCHER, MAGE, ...).
	Type                  string `json:"type"`
	AggroRange            int    `json:"aggroRange,omitempty"`
	ClanHelpRange         int    `json:"clanHelpRange,omitempty"`
	IsChaos               bool   `json:"isChaos,omitempty"`
	IsAggressive          bool   `json:"isAggressive"`
	MinSkillChance        int    `json:"minSkillChance"`
	MaxSkillChance        int    `json:"maxSkillChance"`
	PrimarySkillID        int    `json:"primarySkillId,omitempty"`
	ShortRangeSkillID     int    `json:"shortRangeSkillId,omitempty"`
	ShortRangeSkillChance int    `json:"shortRangeSkillChance,omitempty"`
	LongRangeSkillID      int    `json:"longRangeSkillId,omitempty"`
	LongRangeSkillChance  int    `json:"longRangeSkillChance,omitempty"`
	// Clans are the NPC's clan names, IgnoreNpcIDs the NPCs its clan
	// ignores, in XML order.
	Clans        []string `json:"clans,omitempty"`
	IgnoreNpcIDs []int    `json:"ignoreNpcIds,omitempty"`
}

// NpcShots are the soulshot and spiritshot settings.
type NpcShots struct {
	Soul         int `json:"soul,omitempty"`
	Spirit       int `json:"spirit,omitempty"`
	ShotChance   int `json:"shotChance,omitempty"`
	SpiritChance int `json:"spiritChance,omitempty"`
}

// NpcParameters are the AI parameters of the template's <parameters>, by
// name, kept raw for the AI and scripts. Mobius keeps them in one map; a
// name is used by only one of the three kinds.
type NpcParameters struct {
	Params  map[string]string      `json:"params,omitempty"`
	Skills  map[string]NpcSkill    `json:"skills,omitempty"`
	Minions map[string][]NpcMinion `json:"minions,omitempty"`
}

// NpcSkill is a skill ID and level.
type NpcSkill struct {
	ID    int `json:"id"`
	Level int `json:"level"`
}

// NpcMinion is one entry of a minion list (usually "Privates").
type NpcMinion struct {
	NpcID int `json:"npcId"`
	Count int `json:"count"`
	Max   int `json:"max,omitempty"`
	// RespawnTime is in seconds.
	RespawnTime int `json:"respawnTime"`
	WeightPoint int `json:"weightPoint,omitempty"`
}

// NpcDrops are the drop and spoil lists, in XML order. Mobius sorts them
// when it loads them (items by chance, high to low; group items by chance,
// low to high), with a stable sort; the drop code does the same.
type NpcDrops struct {
	// Items are the ungrouped drops, Groups the grouped ones.
	Items  []NpcDropItem  `json:"items,omitempty"`
	Groups []NpcDropGroup `json:"groups,omitempty"`
	Spoil  []NpcDropItem  `json:"spoil,omitempty"`
}

// NpcDropGroup is a drop group: one roll of Chance percent picks at most
// one of its items.
type NpcDropGroup struct {
	Chance float64       `json:"chance"`
	Items  []NpcDropItem `json:"items"`
}

// NpcDropItem is one possible drop. Chance is in percent. Items Mobius does
// not know are kept; the drop code skips them as Mobius does. The amount is
// Mobius's Rnd.get(Min, Max), which is Min when Min >= Max (one entry of
// the data has Min above Max).
type NpcDropItem struct {
	ItemID int     `json:"itemId"`
	Min    int     `json:"min"`
	Max    int     `json:"max"`
	Chance float64 `json:"chance"`
}

// Constants of the small Mobius enums the NPC data names (Race and
// WeaponType are the generated enums): actor/enums/player/Sex.java,
// actor/enums/npc/AIType.java and the attribute types NpcData reads.
var (
	npcSexes       = setOf("MALE", "FEMALE", "ETC")
	npcAITypes     = setOf("FIGHTER", "ARCHER", "BALANCED", "MAGE", "HEALER", "CORPSE")
	attributeTypes = setOf("FIRE", "WATER", "WIND", "EARTH", "DARK", "HOLY")
)

// checkNpcs checks the NPC templates and indexes them:
//
//   - IDs are positive and unique, the display ID is positive and the
//     level fits Mobius's byte (0-127);
//   - the type is a usable class of NpcTypesFile, and the race, attack
//     type, sex, AI type and attribute type are Mobius constants;
//   - collision sizes and speeds are not negative;
//   - skills, minions and drops have sane numbers (IDs positive, amounts
//     and chances not negative), and minions name existing NPCs.
func (w *World) checkNpcs(p *problems) {
	if len(w.Npcs) == 0 {
		p.addf("%s: no NPC templates", NpcsFile)
	}
	for i := range w.Npcs {
		n := &w.Npcs[i]
		where := fmt.Sprintf("%s: NPC %d (%q)", NpcsFile, n.ID, n.Name)
		if _, dup := w.npcs[n.ID]; dup {
			p.addf("%s: duplicate NPC ID", where)
			continue
		}
		w.npcs[n.ID] = i
		if n.ID <= 0 {
			p.addf("%s: NPC ID must be positive", where)
		}
		if n.DisplayID <= 0 {
			p.addf("%s: display ID must be positive (got %d)", where, n.DisplayID)
		}
		if n.Level < 0 || n.Level > 127 {
			p.addf("%s: level %d is out of range 0-127", where, n.Level)
		}
		if !w.npcTypes.usable(n.Type) {
			p.addf("%s: unknown NPC type %q (not a usable class of %s)", where, n.Type, NpcTypesFile)
		}
		if !n.Race.Valid() {
			p.addf("%s: unknown race %v", where, n.Race)
		}
		if !n.Stats.Attack.Type.Valid() {
			p.addf("%s: unknown attack type %v", where, n.Stats.Attack.Type)
		}
		checkConst(p, where, "sex", n.Sex, npcSexes)
		checkConst(p, where, "AI type", n.AI.Type, npcAITypes)
		if t := n.Stats.Attribute.AttackType; t != "" {
			checkConst(p, where, "attribute type", t, attributeTypes)
		}
		c := n.Collision
		if c.Radius < 0 || c.RadiusGrown < 0 || c.Height < 0 || c.HeightGrown < 0 {
			p.addf("%s: collision sizes must not be negative (got %+v)", where, c)
		}
		s := n.Stats.Speed
		if s.Walk < 0 || s.Run < 0 || s.SwimWalk < 0 || s.SwimRun < 0 || s.FlyWalk < 0 || s.FlyRun < 0 {
			p.addf("%s: speeds must not be negative (got %+v)", where, s)
		}
		for _, sk := range n.Skills {
			if sk.ID <= 0 || sk.Level <= 0 {
				p.addf("%s: skill %d level %d: ID and level must be positive", where, sk.ID, sk.Level)
			}
		}
		for name, sk := range n.Parameters.Skills {
			if sk.ID <= 0 || sk.Level <= 0 {
				p.addf("%s: parameter skill %s: ID and level must be positive (got %+v)", where, name, sk)
			}
		}
		checkDrops(p, where+": drop", n.Drops.Items)
		checkDrops(p, where+": spoil", n.Drops.Spoil)
		for j, g := range n.Drops.Groups {
			if g.Chance < 0 || len(g.Items) == 0 {
				p.addf("%s: drop group %d: needs items and a chance that is not negative (got %v)", where, j, g.Chance)
			}
			checkDrops(p, fmt.Sprintf("%s: drop group %d", where, j), g.Items)
		}
	}
	for i := range w.Npcs {
		n := &w.Npcs[i]
		for name, list := range n.Parameters.Minions {
			for _, m := range list {
				if _, ok := w.npcs[m.NpcID]; !ok {
					p.addf("%s: NPC %d: minions %s: NPC %d does not exist", NpcsFile, n.ID, name, m.NpcID)
				}
				if m.Count <= 0 || m.RespawnTime < 0 {
					p.addf("%s: NPC %d: minions %s: NPC %d needs a positive count and a respawn time that is not negative", NpcsFile, n.ID, name, m.NpcID)
				}
			}
		}
	}
}

// checkConst reports a value that is not one of the constants of an enum.
func checkConst(p *problems, where, what, v string, set map[string]bool) {
	if !set[v] {
		p.addf("%s: unknown %s %q", where, what, v)
	}
}

func checkDrops(p *problems, where string, items []NpcDropItem) {
	for _, it := range items {
		if it.ItemID <= 0 || it.Min < 0 || it.Max < 0 || it.Chance < 0 {
			p.addf("%s: item %d: bad numbers (min %d, max %d, chance %v)", where, it.ItemID, it.Min, it.Max, it.Chance)
		}
	}
}

// Npc returns the NPC template with the given ID.
func (w *World) Npc(id int) (*NpcTemplate, bool) {
	i, ok := w.npcs[id]
	if !ok {
		return nil, false
	}
	return &w.Npcs[i], true
}

// Spawnable reports whether the spawn lists may spawn NPC id at startup,
// following Mobius SpawnData.checkTemplate: the template exists and is not
// of type SiegeGuard or RaidBoss (raid bosses and siege guards are spawned
// by their own managers). Fake player NPCs are not in the data at all.
func (w *World) Spawnable(id int) bool {
	n, ok := w.Npc(id)
	if !ok {
		return false
	}
	return !strings.EqualFold(n.Type, "SiegeGuard") && !strings.EqualFold(n.Type, "RaidBoss")
}
