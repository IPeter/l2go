package questtests

import "testing"

// Help the Son!: Lundy, a Work Hammer, 30 gemstone fragments, Drikus.
func TestB3aQ044HelpTheSon(t *testing.T) {
	b3aQ042Walk(t, b3aQ042PetQuest{
		name: "Q00044_HelpTheSon", start: 30827, helper: 30505, level: 24,
		tool: 168, piece: 7552, whole: 7553, ticket: 7585, monsters: []int{20919, 20920, 20921},
	})
}
