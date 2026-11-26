package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// The requests of pets and servitors (M11f): the pet's inventory, its name,
// and the summon actions of RequestActionUse.

// Opcodes of the pet requests (Mobius ClientPackets).
const (
	OpRequestChangePetName  = 0x89
	OpRequestPetUseItem     = 0x8a
	OpRequestGiveItemToPet  = 0x8b
	OpRequestGetItemFromPet = 0x8c
	OpRequestPetGetItem     = 0x8f
)

// Summon action IDs of RequestActionUse, named after the cases of the
// Mobius Interlude RequestActionUse.runImpl switch.
//
// Mobius validateSummon accepts a servitor for the pet actions too (it
// checks "checkPet && isPet() || isServitor()"), while the servitor actions
// refuse a pet.
const (
	ActionUsePetFollow      = 15 // Change Movement Mode (Pets): follow or stay
	ActionUsePetAttack      = 16 // Attack (Pets); ctrl forces the attack
	ActionUsePetStop        = 17 // Stop (Pets)
	ActionUseUnsummonPet    = 19 // Unsummon Pet
	ActionUseServitorFollow = 21 // Change Movement Mode (Servitors)
	ActionUseServitorAttack = 22 // Attack (Servitors); ctrl forces the attack
	ActionUseServitorStop   = 23 // Stop (Servitors)

	ActionUseWildHogCannonWildCannon = 32 // Wild Hog Cannon - Wild Cannon ("DDMagic")
	ActionUseSoullessToxicSmoke      = 36 // Soulless - Toxic Smoke ("RangeDebuff")
	ActionUseMountDismount           = 38 // Mount/Dismount (strider, wyvern)
	ActionUseSoullessParasiteBurst   = 39 // Soulless - Parasite Burst ("RangeDD")
	ActionUseWildHogCannonAttack     = 41 // Wild Hog Cannon - Attack (skill 4230 on a door or siege flag)
	ActionUseKaiSelfDamageShield     = 42 // Kai the Cat - Self Damage Shield ("HealMagic")
	ActionUseMerrowHydroScrew        = 43 // Merrow the Unicorn - Hydro Screw ("DDMagic")
	ActionUseBigBoomAttack           = 44 // Big Boom - Boom Attack ("DDMagic")
	ActionUseBoxerMasterRecharge     = 45 // Boxer the Unicorn - Master Recharge ("HealMagic" on the owner)
	ActionUseMewMegaStormStrike      = 46 // Mew the Cat - Mega Storm Strike ("DDMagic")
	ActionUseSilhouetteStealBlood    = 47 // Silhouette - Steal Blood ("DDMagic")
	ActionUseMechanicGolemMechCannon = 48 // Mechanic Golem - Mech. Cannon ("DDMagic")
	ActionUseUnsummonServitor        = 52 // Unsummon Servitor
	ActionUseServitorMoveToTarget    = 53 // Move to target (Servitors)
	ActionUsePetMoveToTarget         = 54 // Move to target (Pets)

	ActionUseSiegeGolemSiegeHammer    = 1000 // Siege Golem - Siege Hammer (skill 4079 on a door)
	ActionUseSinEaterBombasticBuster  = 1001 // Sin Eater - Ultimate Bombastic Buster (only an NpcSay)
	ActionUseStriderWildStun          = 1003 // Wind Hatchling/Strider - Wild Stun ("PhysicalSpecial")
	ActionUseStriderWildDefense       = 1004 // Wind Hatchling/Strider - Wild Defense ("Buff" on the owner)
	ActionUseStriderBrightBurst       = 1005 // Star Hatchling/Strider - Bright Burst ("DDMagic")
	ActionUseStriderBrightHeal        = 1006 // Star Hatchling/Strider - Bright Heal ("Heal" on the owner)
	ActionUseFelineQueenBlessing      = 1007 // Feline Queen - Blessing of Queen ("Buff1" on the owner)
	ActionUseFelineQueenGift          = 1008 // Feline Queen - Gift of Queen ("Buff2" on the owner)
	ActionUseFelineQueenCure          = 1009 // Feline Queen - Cure of Queen ("DDMagic")
	ActionUseSeraphimBlessing         = 1010 // Unicorn Seraphim - Blessing of Seraphim ("Buff1" on the owner)
	ActionUseSeraphimGift             = 1011 // Unicorn Seraphim - Gift of Seraphim ("Buff2" on the owner)
	ActionUseSeraphimCure             = 1012 // Unicorn Seraphim - Cure of Seraphim ("DDMagic")
	ActionUseNightshadeCurse          = 1013 // Nightshade - Curse of Shade ("DeBuff1")
	ActionUseNightshadeMassCurse      = 1014 // Nightshade - Mass Curse of Shade ("DeBuff2")
	ActionUseNightshadeSacrifice      = 1015 // Nightshade - Shade Sacrifice ("Heal")
	ActionUseCursedManBlow            = 1016 // Cursed Man - Cursed Blow ("PhysicalSpecial1")
	ActionUseCursedManStrike          = 1017 // Cursed Man - Cursed Strike ("PhysicalSpecial2")
	ActionUseFelineKingSlash          = 1031 // Feline King - Slash ("PhysicalSpecial1")
	ActionUseFelineKingSpinningSlash  = 1032 // Feline King - Spinning Slash ("PhysicalSpecial2")
	ActionUseFelineKingHold           = 1033 // Feline King - Hold of King ("PhysicalSpecial3")
	ActionUseMagnusWhiplash           = 1034 // Magnus the Unicorn - Whiplash ("PhysicalSpecial1")
	ActionUseMagnusTidalWave          = 1035 // Magnus the Unicorn - Tridal Wave ("PhysicalSpecial2")
	ActionUseSpectralLordCorpseKaboom = 1036 // Spectral Lord - Corpse Kaboom ("PhysicalSpecial1")
	ActionUseSpectralLordDicingDeath  = 1037 // Spectral Lord - Dicing Death ("PhysicalSpecial2")
	ActionUseSpectralLordDarkCurse    = 1038 // Spectral Lord - Dark Curse ("PhysicalSpecial3")
	ActionUseSwoopCannonCannonFodder  = 1039 // Swoop Cannon - Cannon Fodder (skill 5110)
	ActionUseSwoopCannonBigBang       = 1040 // Swoop Cannon - Big Bang (skill 5111)
)

// SummonActionSkill is what a summon skill action of RequestActionUse casts
// (Mobius RequestActionUse.useSkill).
type SummonActionSkill struct {
	// Param is the name of the NPC template parameter that holds the skill
	// (Mobius getParameters().getSkillHolder(name)); "" when SkillID is set.
	// An action does nothing for a summon whose template lacks it.
	Param string
	// SkillID is the skill of the actions that name one; its level is the
	// highest the summon has learned (PetDataTable for a pet, PetSkillData
	// for a servitor), and the action does nothing at level 0.
	SkillID int32
	// OnOwner casts on the player instead of the player's target.
	OnOwner bool
	// Pet is the validateSummon checkPet flag: true accepts a pet or a
	// servitor, false only a servitor.
	Pet bool
	// DoorOnly is set for the actions that need a door target: Siege Hammer
	// (silently nothing otherwise) and the Wild Hog Cannon attack, which also
	// takes a siege flag and else sends INVALID_TARGET.
	DoorOnly bool
}

// SummonActionSkills maps the summon skill action IDs to their skills, as
// the Mobius Interlude RequestActionUse switch casts them. Actions 1001 (an
// NpcSay only), the movement actions and mounting are not here.
var SummonActionSkills = map[int32]SummonActionSkill{
	ActionUseWildHogCannonWildCannon:  {Param: "DDMagic"},
	ActionUseSoullessToxicSmoke:       {Param: "RangeDebuff"},
	ActionUseSoullessParasiteBurst:    {Param: "RangeDD"},
	ActionUseWildHogCannonAttack:      {SkillID: 4230, DoorOnly: true},
	ActionUseKaiSelfDamageShield:      {Param: "HealMagic"},
	ActionUseMerrowHydroScrew:         {Param: "DDMagic"},
	ActionUseBigBoomAttack:            {Param: "DDMagic"},
	ActionUseBoxerMasterRecharge:      {Param: "HealMagic", OnOwner: true},
	ActionUseMewMegaStormStrike:       {Param: "DDMagic"},
	ActionUseSilhouetteStealBlood:     {Param: "DDMagic"},
	ActionUseMechanicGolemMechCannon:  {Param: "DDMagic"},
	ActionUseSiegeGolemSiegeHammer:    {SkillID: 4079, DoorOnly: true},
	ActionUseStriderWildStun:          {Param: "PhysicalSpecial", Pet: true},
	ActionUseStriderWildDefense:       {Param: "Buff", OnOwner: true, Pet: true},
	ActionUseStriderBrightBurst:       {Param: "DDMagic", Pet: true},
	ActionUseStriderBrightHeal:        {Param: "Heal", OnOwner: true, Pet: true},
	ActionUseFelineQueenBlessing:      {Param: "Buff1", OnOwner: true},
	ActionUseFelineQueenGift:          {Param: "Buff2", OnOwner: true},
	ActionUseFelineQueenCure:          {Param: "DDMagic"},
	ActionUseSeraphimBlessing:         {Param: "Buff1", OnOwner: true},
	ActionUseSeraphimGift:             {Param: "Buff2", OnOwner: true},
	ActionUseSeraphimCure:             {Param: "DDMagic"},
	ActionUseNightshadeCurse:          {Param: "DeBuff1"},
	ActionUseNightshadeMassCurse:      {Param: "DeBuff2"},
	ActionUseNightshadeSacrifice:      {Param: "Heal"},
	ActionUseCursedManBlow:            {Param: "PhysicalSpecial1"},
	ActionUseCursedManStrike:          {Param: "PhysicalSpecial2"},
	ActionUseFelineKingSlash:          {Param: "PhysicalSpecial1"},
	ActionUseFelineKingSpinningSlash:  {Param: "PhysicalSpecial2"},
	ActionUseFelineKingHold:           {Param: "PhysicalSpecial3"},
	ActionUseMagnusWhiplash:           {Param: "PhysicalSpecial1"},
	ActionUseMagnusTidalWave:          {Param: "PhysicalSpecial2"},
	ActionUseSpectralLordCorpseKaboom: {Param: "PhysicalSpecial1"},
	ActionUseSpectralLordDicingDeath:  {Param: "PhysicalSpecial2"},
	ActionUseSpectralLordDarkCurse:    {Param: "PhysicalSpecial3"},
	ActionUseSwoopCannonCannonFodder:  {SkillID: 5110},
	ActionUseSwoopCannonBigBang:       {SkillID: 5111},
}

// RequestChangePetName names the player's pet. The name is not checked
// here: Mobius refuses a pet that has a name, a name longer than 8
// characters, an invalid or taken name.
type RequestChangePetName struct {
	Name string
}

// ParseRequestChangePetName parses a RequestChangePetName body (Mobius
// RequestChangePetName.readImpl).
func ParseRequestChangePetName(data []byte) (RequestChangePetName, error) {
	r := packets.NewReader(data)
	p := RequestChangePetName{Name: r.ReadString()}
	if err := r.Err(); err != nil {
		return RequestChangePetName{}, fmt.Errorf("RequestChangePetName: %w", err)
	}
	return p, nil
}

// RequestPetUseItem is a double click on an item of the pet's inventory.
// Mobius reads only the object ID.
type RequestPetUseItem struct {
	ObjectID int32
}

// ParseRequestPetUseItem parses a RequestPetUseItem body (Mobius
// RequestPetUseItem.readImpl).
func ParseRequestPetUseItem(data []byte) (RequestPetUseItem, error) {
	r := packets.NewReader(data)
	p := RequestPetUseItem{ObjectID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestPetUseItem{}, fmt.Errorf("RequestPetUseItem: %w", err)
	}
	return p, nil
}

// RequestGiveItemToPet moves Count of an inventory item to the pet's
// inventory. The values are not checked here.
type RequestGiveItemToPet struct {
	ObjectID int32
	Count    int32
}

// ParseRequestGiveItemToPet parses a RequestGiveItemToPet body (Mobius
// RequestGiveItemToPet.readImpl).
func ParseRequestGiveItemToPet(data []byte) (RequestGiveItemToPet, error) {
	r := packets.NewReader(data)
	p := RequestGiveItemToPet{ObjectID: r.ReadInt32(), Count: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestGiveItemToPet{}, fmt.Errorf("RequestGiveItemToPet: %w", err)
	}
	return p, nil
}

// RequestGetItemFromPet moves Count of a pet's item to the owner's
// inventory. The values are not checked here.
type RequestGetItemFromPet struct {
	ObjectID int32
	Count    int32
	// Unknown is a word Mobius reads and ignores ("0 for most trades").
	Unknown int32
}

// ParseRequestGetItemFromPet parses a RequestGetItemFromPet body (Mobius
// RequestGetItemFromPet.readImpl).
func ParseRequestGetItemFromPet(data []byte) (RequestGetItemFromPet, error) {
	r := packets.NewReader(data)
	p := RequestGetItemFromPet{ObjectID: r.ReadInt32(), Count: r.ReadInt32(), Unknown: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestGetItemFromPet{}, fmt.Errorf("RequestGetItemFromPet: %w", err)
	}
	return p, nil
}

// RequestPetGetItem sends the pet to pick up the ground item ObjectID.
type RequestPetGetItem struct {
	ObjectID int32
}

// ParseRequestPetGetItem parses a RequestPetGetItem body (Mobius
// RequestPetGetItem.readImpl).
func ParseRequestPetGetItem(data []byte) (RequestPetGetItem, error) {
	r := packets.NewReader(data)
	p := RequestPetGetItem{ObjectID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestPetGetItem{}, fmt.Errorf("RequestPetGetItem: %w", err)
	}
	return p, nil
}
