package questtests

import "testing"

// Chest caught with a bait of fire: Linnaeus and Rukal, a Necklace of Protection.
func TestB3aQ030ChestCaughtWithABaitOfFire(t *testing.T) {
	b3aQ027WalkChest(t, b3aQ027Chest{
		name: "Q00030_ChestCaughtWithABaitOfFire", baitQuest: "Q00053_LinnaeusSpecialBait", id: 30, level: 60,
		fisher: 31577, other: 30629, chest: 6511, item: 7628, reward: 916,
	})
}
