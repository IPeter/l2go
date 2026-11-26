// Package summon holds the pure rules of pets, servitors, baby pets and
// cubics (Plan 14 M11f, decision X2): the servitor's life and upkeep, pet
// hunger and feeding, pet experience and levels, pet skill levels, the
// owner's experience share, cubic skill selection and targeting rules, and
// the summon actions of the action window. Nothing here touches the world:
// internal/game adapts these rules to the loop, the packets and storage.
//
// The Mobius sources are named on each rule: entity/actor/Summon.java,
// instance/Pet.java, Servitor.java, BabyPet.java, Cubic.java,
// stat/PetStat.java, data/xml/PetSkillData.java and
// network/clientpackets/RequestActionUse.java.
package summon

// Kind is the kind of a summon (Mobius Summon's subclasses).
type Kind uint8

// The summon kinds.
const (
	// Servitor is a summon of a skill (Mobius Servitor).
	Servitor Kind = iota + 1
	// Pet is a summon of a collar item (Mobius Pet).
	Pet
	// BabyPet is a pet that heals and buffs its owner (Mobius BabyPet).
	BabyPet
)

// String returns Mobius's class name of the kind.
func (k Kind) String() string {
	switch k {
	case Servitor:
		return "Servitor"
	case Pet:
		return "Pet"
	case BabyPet:
		return "BabyPet"
	}
	return "?"
}

// IsPet reports a pet or a baby pet (Mobius isPet).
func (k Kind) IsPet() bool { return k == Pet || k == BabyPet }

// SummonType is the number the packets and the action window use for the
// kind (Mobius getSummonType): 1 for a servitor, 2 for a pet.
func (k Kind) SummonType() int32 {
	if k.IsPet() {
		return 2
	}
	return 1
}

// KindOfType returns the kind of an NPC template type ("Servitor", "Pet",
// "BabyPet"), false for any other type.
func KindOfType(typ string) (Kind, bool) {
	switch typ {
	case "Servitor":
		return Servitor, true
	case "Pet":
		return Pet, true
	case "BabyPet":
		return BabyPet, true
	}
	return 0, false
}

// SinEaterID is the Sin Eater (Mobius SIN_EATER_ID), a pet whose level
// follows its owner's when it first comes out.
const SinEaterID = 12564

// PassiveSummons are the summons that never attack with the attack button
// (Mobius Summon.PASSIVE_SUMMONS: Sin Eater, Wyvern, Big Boom...).
var PassiveSummons = []int{
	12564, 12621, 14702, 14703, 14704, 14705, 14706, 14707, 14708, 14709, 14710, 14711,
	14712, 14713, 14714, 14715, 14716, 14717, 14718, 14719, 14720, 14721, 14722, 14723,
	14724, 14725, 14726, 14727, 14728, 14729, 14730, 14731, 14732, 14733, 14734, 14735, 14736,
}

// IsPassive reports a passive summon.
func IsPassive(npcID int) bool {
	for _, id := range PassiveSummons {
		if id == npcID {
			return true
		}
	}
	return false
}

// Ranges of the summon rules.
const (
	// MinAttackRange is the smallest attack range of a summon (Mobius
	// Summon.setAttackRange: 36, melee).
	MinAttackRange = 36
	// MaxCommandRange is how far from its owner a summon still takes an
	// attack order (Summon.canAttack: 3,000).
	MaxCommandRange = 3000
	// FollowGiveUpRange is how far a summon follows before it gives up
	// (CreatureFollowTaskManager: 3,000).
	FollowGiveUpRange = 3000
	// FollowMinRange and FollowMaxRange bound the random distance a
	// following summon keeps (Rnd.get(50, 100)).
	FollowMinRange, FollowMaxRange = 50, 100
	// AvoidRadius is how far a summon steps around its owner when hit near
	// it (SummonAI.AVOID_RADIUS).
	AvoidRadius = 70
	// PickUpRange is the range of an INTERACT or PICK_UP order (36).
	PickUpRange = 36
	// LevelGapControl is the level gap above its owner at which a pet stops
	// obeying (Summon.canAttack and RequestActionUse.canControl: 20).
	LevelGapControl = 20
	// SpawnOffset is the random offset from its owner a servitor appears at
	// (Summon's constructor: Rnd.get(-100, 100)).
	SpawnOffset = 100
)

// Pet spawn offsets from its owner (SummonPet.onStart: x + 50, y + 100).
const (
	PetSpawnDX = 50
	PetSpawnDY = 100
)
