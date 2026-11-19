// Package stats is the stat engine: it computes creature stats the way L2J
// Mobius (Interlude) does, so the numbers match the client and the reference
// server bit for bit.
//
// A creature's stat is a chain of functions applied to a base value, run on
// every read (Mobius Calculator and CreatureStat.calcStat):
//
//   - Modifiers are data: {Stat, Op, Order, Value, Owner, Cond}. Equipped
//     items add them (Set/Add at order 0), and later skills and buffs. They
//     are kept per stat in stable order and removed by owner.
//   - Formulas are code: the formula functions Mobius gives every player
//     and NPC at creation (FuncPAtkMod, FuncMaxHpMul, ...), using the base
//     stat bonus tables of statBonus.xml and the level modifier.
//
// Player and Npc are the stat sets of the two creature kinds. Their getters
// reproduce the Mobius getters (with its casts and rounding points), and
// View returns every number UserInfo, CharInfo and NpcInfo send.
//
// The package has its own input types (ClassTemplate, NpcTemplate, Item,
// BonusTables, Config); the game layer converts the loaded data into them.
// Stat sets are not safe for concurrent use: the world loop owns them.
package stats

import "strconv"

// Stat identifies one creature stat. The values follow the order of Mobius
// Stat.java (its ordinal), and String returns the name the datapack XML uses.
// The constant names are the XML names in Go case.
type Stat uint8

// The stats, in Mobius order. Comments give the Mobius name where the XML
// name does not make it obvious.
const (
	MaxHP                    Stat = iota // MAX_HP
	MaxMP                                // MAX_MP
	MaxCP                                // MAX_CP
	MaxRecoverableHP                     // MAX_RECOVERABLE_HP
	MaxRecoverableMP                     // MAX_RECOVERABLE_MP
	MaxRecoverableCP                     // MAX_RECOVERABLE_CP
	RegHP                                // REGENERATE_HP_RATE
	RegCP                                // REGENERATE_CP_RATE
	RegMP                                // REGENERATE_MP_RATE
	ManaCharge                           // MANA_CHARGE
	HealEffect                           // HEAL_EFFECT
	PDef                                 // POWER_DEFENCE
	MDef                                 // MAGIC_DEFENCE
	PAtk                                 // POWER_ATTACK
	MAtk                                 // MAGIC_ATTACK
	PhysicalSkillPower                   // PHYSICAL_SKILL_POWER
	PAtkSpd                              // POWER_ATTACK_SPEED
	MAtkSpd                              // MAGIC_ATTACK_SPEED (casting speed)
	AtkReuse                             // ATK_REUSE
	PReuse                               // P_REUSE
	MReuse                               // MAGIC_REUSE_RATE
	DReuse                               // DANCE_REUSE
	SDef                                 // SHIELD_DEFENCE
	CritDmg                              // CRITICAL_DAMAGE
	CritDmgPos                           // CRITICAL_DAMAGE_POS
	CritDmgAdd                           // CRITICAL_DAMAGE_ADD
	MCritPower                           // MAGIC_CRIT_DMG
	PvpPhysDmg                           // PVP_PHYSICAL_DMG
	PvpMagicalDmg                        // PVP_MAGICAL_DMG
	PvpPhysSkillsDmg                     // PVP_PHYS_SKILL_DMG
	PvpPhysDef                           // PVP_PHYSICAL_DEF
	PvpMagicalDef                        // PVP_MAGICAL_DEF
	PvpPhysSkillsDef                     // PVP_PHYS_SKILL_DEF
	PvePhysDmg                           // PVE_PHYSICAL_DMG
	PvePhysSkillsDmg                     // PVE_PHYS_SKILL_DMG
	PveBowDmg                            // PVE_BOW_DMG
	PveBowSkillsDmg                      // PVE_BOW_SKILL_DMG
	PveMagicalDmg                        // PVE_MAGICAL_DMG
	REvas                                // EVASION_RATE
	PSkillEvas                           // P_SKILL_EVASION
	DefCritRate                          // DEFENCE_CRITICAL_RATE
	DefCritRateAdd                       // DEFENCE_CRITICAL_RATE_ADD
	DefCritDamage                        // DEFENCE_CRITICAL_DAMAGE
	DefCritDamageAdd                     // DEFENCE_CRITICAL_DAMAGE_ADD
	RShld                                // SHIELD_RATE
	CritRate                             // CRITICAL_RATE
	CritRatePos                          // CRITICAL_RATE_POS
	BlowRate                             // BLOW_RATE
	MCritRate                            // MCRITICAL_RATE
	RExp                                 // EXPSP_RATE
	BonusExp                             // BONUS_EXP
	BonusSP                              // BONUS_SP
	BonusDropAdena                       // BONUS_DROP_ADENA
	BonusDropAmount                      // BONUS_DROP_AMOUNT
	BonusDropRate                        // BONUS_DROP_RATE
	BonusSpoilRate                       // BONUS_SPOIL_RATE
	Cancel                               // ATTACK_CANCEL
	AccCombat                            // ACCURACY_COMBAT
	PAtkRange                            // POWER_ATTACK_RANGE
	MAtkRange                            // MAGIC_ATTACK_RANGE
	AtkCountMax                          // ATTACK_COUNT_MAX
	RunSpd                               // MOVE_SPEED: scales every movement speed
	STR                                  // STAT_STR
	CON                                  // STAT_CON
	DEX                                  // STAT_DEX
	INT                                  // STAT_INT
	WIT                                  // STAT_WIT
	MEN                                  // STAT_MEN
	Breath                               // BREATH
	Fall                                 // FALL
	DamageZoneVuln                       // DAMAGE_ZONE_VULN
	MovementVuln                         // MOVEMENT_VULN
	CancelVuln                           // CANCEL_VULN
	DebuffVuln                           // DEBUFF_VULN
	BuffVuln                             // BUFF_VULN
	FireRes                              // FIRE_RES
	WindRes                              // WIND_RES
	WaterRes                             // WATER_RES
	EarthRes                             // EARTH_RES
	HolyRes                              // HOLY_RES
	DarkRes                              // DARK_RES
	MagicSuccRes                         // MAGIC_SUCCESS_RES
	DebuffImmunity                       // DEBUFF_IMMUNITY
	FirePower                            // FIRE_POWER
	WaterPower                           // WATER_POWER
	WindPower                            // WIND_POWER
	EarthPower                           // EARTH_POWER
	HolyPower                            // HOLY_POWER
	DarkPower                            // DARK_POWER
	CancelProf                           // CANCEL_PROF
	ReflectDam                           // REFLECT_DAMAGE_PERCENT
	ReflectSkillMagic                    // REFLECT_SKILL_MAGIC
	ReflectSkillPhysic                   // REFLECT_SKILL_PHYSIC
	VengeanceMdam                        // VENGEANCE_SKILL_MAGIC_DAMAGE
	VengeancePdam                        // VENGEANCE_SKILL_PHYSICAL_DAMAGE
	AbsorbDam                            // ABSORB_DAMAGE_PERCENT
	TransDam                             // TRANSFER_DAMAGE_PERCENT
	ManaShield                           // MANA_SHIELD_PERCENT
	TransDamToPlayer                     // TRANSFER_DAMAGE_TO_PLAYER
	AbsorbDamMana                        // ABSORB_MANA_DAMAGE_PERCENT
	WeightLimit                          // WEIGHT_LIMIT
	WeightPenalty                        // WEIGHT_PENALTY
	InventoryLimit                       // INV_LIM
	WhLimit                              // WH_LIM
	FreightLimit                         // FREIGHT_LIM
	PrivateSellLimit                     // P_SELL_LIM
	PrivateBuyLimit                      // P_BUY_LIM
	DwarfRecipeLimit                     // REC_D_LIM
	CommonRecipeLimit                    // REC_C_LIM
	PhysicalMPConsumeRate                // PHYSICAL_MP_CONSUME_RATE
	MagicalMPConsumeRate                 // MAGICAL_MP_CONSUME_RATE
	DanceMPConsumeRate                   // DANCE_MP_CONSUME_RATE
	BowMPConsumeRate                     // BOW_MP_CONSUME_RATE
	MPConsume                            // MP_CONSUME
	ShieldDefAngle                       // SHIELD_DEFENCE_ANGLE
	SkillCritical                        // SKILL_CRITICAL
	SkillCriticalProbability             // SKILL_CRITICAL_PROBABILITY
	VitalityConsumeRate                  // VITALITY_CONSUME_RATE
	ReduceExpLostByPvp                   // REDUCE_EXP_LOST_BY_PVP
	ReduceExpLostByMob                   // REDUCE_EXP_LOST_BY_MOB
	ReduceExpLostByRaid                  // REDUCE_EXP_LOST_BY_RAID
	ReduceDeathPenaltyByPvp              // REDUCE_DEATH_PENALTY_BY_PVP
	ReduceDeathPenaltyByMob              // REDUCE_DEATH_PENALTY_BY_MOB
	ReduceDeathPenaltyByRaid             // REDUCE_DEATH_PENALTY_BY_RAID
	FishingExpertise                     // FISHING_EXPERTISE
	RandomDamage                         // RANDOM_DAMAGE
)

// NumStats is the number of stats (Mobius Stat.NUM_STATS, 126).
const NumStats = int(RandomDamage) + 1

// statNames are the XML names (Stat.getValue), indexed by Stat.
var statNames = [NumStats]string{
	MaxHP:                    "maxHp",
	MaxMP:                    "maxMp",
	MaxCP:                    "maxCp",
	MaxRecoverableHP:         "maxRecoverableHp",
	MaxRecoverableMP:         "maxRecoverableMp",
	MaxRecoverableCP:         "maxRecoverableCp",
	RegHP:                    "regHp",
	RegCP:                    "regCp",
	RegMP:                    "regMp",
	ManaCharge:               "manaCharge",
	HealEffect:               "healEffect",
	PDef:                     "pDef",
	MDef:                     "mDef",
	PAtk:                     "pAtk",
	MAtk:                     "mAtk",
	PhysicalSkillPower:       "physicalSkillPower",
	PAtkSpd:                  "pAtkSpd",
	MAtkSpd:                  "mAtkSpd",
	AtkReuse:                 "atkReuse",
	PReuse:                   "pReuse",
	MReuse:                   "mReuse",
	DReuse:                   "dReuse",
	SDef:                     "sDef",
	CritDmg:                  "critDmg",
	CritDmgPos:               "critDmgPos",
	CritDmgAdd:               "critDmgAdd",
	MCritPower:               "mCritPower",
	PvpPhysDmg:               "pvpPhysDmg",
	PvpMagicalDmg:            "pvpMagicalDmg",
	PvpPhysSkillsDmg:         "pvpPhysSkillsDmg",
	PvpPhysDef:               "pvpPhysDef",
	PvpMagicalDef:            "pvpMagicalDef",
	PvpPhysSkillsDef:         "pvpPhysSkillsDef",
	PvePhysDmg:               "pvePhysDmg",
	PvePhysSkillsDmg:         "pvePhysSkillsDmg",
	PveBowDmg:                "pveBowDmg",
	PveBowSkillsDmg:          "pveBowSkillsDmg",
	PveMagicalDmg:            "pveMagicalDmg",
	REvas:                    "rEvas",
	PSkillEvas:               "pSkillEvas",
	DefCritRate:              "defCritRate",
	DefCritRateAdd:           "defCritRateAdd",
	DefCritDamage:            "defCritDamage",
	DefCritDamageAdd:         "defCritDamageAdd",
	RShld:                    "rShld",
	CritRate:                 "critRate",
	CritRatePos:              "critRatePos",
	BlowRate:                 "blowRate",
	MCritRate:                "mCritRate",
	RExp:                     "rExp",
	BonusExp:                 "bonusExp",
	BonusSP:                  "bonusSp",
	BonusDropAdena:           "bonusDropAdena",
	BonusDropAmount:          "bonusDropAmount",
	BonusDropRate:            "bonusDropRate",
	BonusSpoilRate:           "bonusSpoilRate",
	Cancel:                   "cancel",
	AccCombat:                "accCombat",
	PAtkRange:                "pAtkRange",
	MAtkRange:                "mAtkRange",
	AtkCountMax:              "atkCountMax",
	RunSpd:                   "runSpd",
	STR:                      "STR",
	CON:                      "CON",
	DEX:                      "DEX",
	INT:                      "INT",
	WIT:                      "WIT",
	MEN:                      "MEN",
	Breath:                   "breath",
	Fall:                     "fall",
	DamageZoneVuln:           "damageZoneVuln",
	MovementVuln:             "movementVuln",
	CancelVuln:               "cancelVuln",
	DebuffVuln:               "debuffVuln",
	BuffVuln:                 "buffVuln",
	FireRes:                  "fireRes",
	WindRes:                  "windRes",
	WaterRes:                 "waterRes",
	EarthRes:                 "earthRes",
	HolyRes:                  "holyRes",
	DarkRes:                  "darkRes",
	MagicSuccRes:             "magicSuccRes",
	DebuffImmunity:           "debuffImmunity",
	FirePower:                "firePower",
	WaterPower:               "waterPower",
	WindPower:                "windPower",
	EarthPower:               "earthPower",
	HolyPower:                "holyPower",
	DarkPower:                "darkPower",
	CancelProf:               "cancelProf",
	ReflectDam:               "reflectDam",
	ReflectSkillMagic:        "reflectSkillMagic",
	ReflectSkillPhysic:       "reflectSkillPhysic",
	VengeanceMdam:            "vengeanceMdam",
	VengeancePdam:            "vengeancePdam",
	AbsorbDam:                "absorbDam",
	TransDam:                 "transDam",
	ManaShield:               "manaShield",
	TransDamToPlayer:         "transDamToPlayer",
	AbsorbDamMana:            "absorbDamMana",
	WeightLimit:              "weightLimit",
	WeightPenalty:            "weightPenalty",
	InventoryLimit:           "inventoryLimit",
	WhLimit:                  "whLimit",
	FreightLimit:             "freightLimit",
	PrivateSellLimit:         "privateSellLimit",
	PrivateBuyLimit:          "privateBuyLimit",
	DwarfRecipeLimit:         "dwarfRecipeLimit",
	CommonRecipeLimit:        "commonRecipeLimit",
	PhysicalMPConsumeRate:    "physicalMpConsumeRate",
	MagicalMPConsumeRate:     "magicalMpConsumeRate",
	DanceMPConsumeRate:       "danceMpConsumeRate",
	BowMPConsumeRate:         "bowMpConsumeRate",
	MPConsume:                "mpConsume",
	ShieldDefAngle:           "shieldDefAngle",
	SkillCritical:            "skillCritical",
	SkillCriticalProbability: "skillCriticalProbability",
	VitalityConsumeRate:      "vitalityConsumeRate",
	ReduceExpLostByPvp:       "reduceExpLostByPvp",
	ReduceExpLostByMob:       "reduceExpLostByMob",
	ReduceExpLostByRaid:      "reduceExpLostByRaid",
	ReduceDeathPenaltyByPvp:  "reduceDeathPenaltyByPvp",
	ReduceDeathPenaltyByMob:  "reduceDeathPenaltyByMob",
	ReduceDeathPenaltyByRaid: "reduceDeathPenaltyByRaid",
	FishingExpertise:         "fishingExpertise",
	RandomDamage:             "randomDamage",
}

// statByName maps XML names to stats.
var statByName = func() map[string]Stat {
	m := make(map[string]Stat, NumStats)
	for i, name := range statNames {
		m[name] = Stat(i)
	}
	return m
}()

// String returns the XML name of the stat, or "Stat(n)" for an unknown value.
func (s Stat) String() string {
	if s.Valid() {
		return statNames[s]
	}
	return "Stat(" + strconv.Itoa(int(s)) + ")"
}

// Valid reports whether s is one of the NumStats stats.
func (s Stat) Valid() bool { return int(s) < NumStats }

// ParseStat returns the stat with the given XML name (Stat.valueOfXml). The
// match is exact, as in Mobius.
func ParseStat(name string) (Stat, bool) {
	s, ok := statByName[name]
	return s, ok
}

// keepsPositive are the stats Mobius never lets drop to zero or below: a
// non-empty calculator result of 0 or less becomes 1 (CreatureStat.calcStat).
var keepsPositive = [NumStats]bool{
	MaxHP: true, MaxMP: true, MaxCP: true,
	MDef: true, PDef: true, PAtk: true, MAtk: true,
	PAtkSpd: true, MAtkSpd: true, SDef: true,
	CON: true, DEX: true, INT: true, MEN: true, STR: true, WIT: true,
}
