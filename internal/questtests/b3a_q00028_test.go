package questtests

import "testing"

// Chest caught with a bait of icy air: O'Fulle and Kiki, an Elven Ring.
func TestB3aQ028ChestCaughtWithABaitOfIcyAir(t *testing.T) {
	b3aQ027WalkChest(t, b3aQ027Chest{
		name: "Q00028_ChestCaughtWithABaitOfIcyAir", baitQuest: "Q00051_OFullesSpecialBait", id: 28, level: 36,
		fisher: 31572, other: 31442, chest: 6503, item: 7626, reward: 881,
	})
}
