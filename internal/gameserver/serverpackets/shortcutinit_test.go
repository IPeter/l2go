package serverpackets

import "testing"

func TestShortcutInitPacketEmpty(t *testing.T) {
	m3Check(t, "empty ShortcutInit", NewShortcutInitPacket(nil), unhex(t, "45 00000000"))
}

func TestShortcutInitPacket(t *testing.T) {
	shortcuts := []Shortcut{
		{Type: ShortcutItem, Slot: 3, Page: 2, ID: 0x10000002},
		{Type: ShortcutSkill, Slot: 0, Page: 0, ID: 1177, Level: 5},
		{Type: ShortcutAction, Slot: 11, Page: 9, ID: 2},
		{Type: ShortcutMacro, Slot: 1, Page: 1, ID: 1000},
		{Type: ShortcutRecipe, Slot: 2, Page: 0, ID: 17},
		{Type: ShortcutNone, Slot: 4, Page: 0, ID: 99},
	}
	// Mobius ShortcutInit.writeImpl: c 0x45, d count, then per shortcut
	// d type, d slot + page × 12, and by type:
	//   item:   d ID, d 1, d -1, d 0, d 0, h 0, h 0
	//   skill:  d ID, d level, c 0, d 1
	//   action, macro, recipe: d ID, d 1
	//   none:   nothing
	var g m3Golden
	g.c(0x45).d(6)
	g.d(1).d(27).d(0x10000002).d(1).d(-1).d(0).d(0).h(0).h(0)
	g.d(2).d(0).d(1177).d(5).c(0).d(1)
	g.d(3).d(119).d(2).d(1)
	g.d(4).d(13).d(1000).d(1)
	g.d(5).d(2).d(17).d(1)
	g.d(0).d(4)
	m3Check(t, "ShortcutInit", NewShortcutInitPacket(shortcuts), g.b)
}
