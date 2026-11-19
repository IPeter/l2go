package serverpackets

import "testing"

// Mobius ShortcutRegister.writeImpl:
//
//	c 0x44
//	d type ordinal, d slot + page × 12
//	item: d ID
//	skill: d ID, d level, c 0
//	action, macro, recipe: d ID
//	none: nothing
//	d 1
//
// It is shorter than a ShortcutInit entry: an item has no -1, 0, 0, 0, 0
// after its ID, and every type ends with the 1.
func TestShortcutRegisterPacket(t *testing.T) {
	tests := []struct {
		name string
		in   Shortcut
		want string
	}{
		{"item", Shortcut{Type: ShortcutItem, Slot: 3, Page: 2, ID: 0x10000002},
			"44 01000000 1b000000 02000010 01000000"},
		{"skill", Shortcut{Type: ShortcutSkill, Slot: 0, Page: 0, ID: 1177, Level: 5},
			"44 02000000 00000000 99040000 05000000 00 01000000"},
		{"action", Shortcut{Type: ShortcutAction, Slot: 11, Page: 9, ID: 2},
			"44 03000000 77000000 02000000 01000000"},
		{"macro", Shortcut{Type: ShortcutMacro, Slot: 1, Page: 1, ID: 1000},
			"44 04000000 0d000000 e8030000 01000000"},
		{"recipe", Shortcut{Type: ShortcutRecipe, Slot: 2, Page: 0, ID: 17},
			"44 05000000 02000000 11000000 01000000"},
		{"none", Shortcut{Type: ShortcutNone, Slot: 4, Page: 0, ID: 99},
			"44 00000000 04000000 01000000"},
		// Mobius ignores the level of anything but a skill.
		{"item with a level", Shortcut{Type: ShortcutItem, Slot: 0, Page: 0, ID: 7, Level: -1},
			"44 01000000 00000000 07000000 01000000"},
	}
	for _, tt := range tests {
		m67Check(t, tt.name, NewShortcutRegisterPacket(tt.in), m67Hex(t, tt.want))
	}
}

// The last slot of the last page the client has: 9 × 12 + 11.
func TestShortcutRegisterPacketLastSlot(t *testing.T) {
	var g m67Golden
	g.c(0x44).d(2).d(119).d(1204).d(2).c(0).d(1)
	got := NewShortcutRegisterPacket(Shortcut{Type: ShortcutSkill, Slot: 11, Page: 9, ID: 1204, Level: 2})
	m67Check(t, "last slot", got, g.b)
}
