package questtests

import "testing"

// Help the Sister!: Cooper, a Crafted Dagger, 30 map pieces, Galladucci.
func TestB3aQ043HelpTheSister(t *testing.T) {
	b3aQ042Walk(t, b3aQ042PetQuest{
		name: "Q00043_HelpTheSister", start: 30829, helper: 30097, level: 26,
		tool: 220, piece: 7550, whole: 7551, ticket: 7584, monsters: []int{20171, 20197},
	})
}
