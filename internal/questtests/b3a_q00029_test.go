package questtests

import "testing"

// Chest caught with a bait of earth: Willie and Anabel, Plated Leather Gloves.
func TestB3aQ029ChestCaughtWithABaitOfEarth(t *testing.T) {
	b3aQ027WalkChest(t, b3aQ027Chest{
		name: "Q00029_ChestCaughtWithABaitOfEarth", baitQuest: "Q00052_WilliesSpecialBait", id: 29, level: 48,
		fisher: 31574, other: 30909, chest: 6507, item: 7627, reward: 2455,
	})
}
