package serverpackets

import "testing"

func TestQuestListPacket(t *testing.T) {
	// Mobius QuestList.writeImpl: c 0x80, h count, then per quest d ID,
	// d cond (or the completed-steps flags).
	m3Check(t, "empty QuestList", NewQuestListPacket(nil), unhex(t, "80 0000"))

	quests := []QuestListEntry{
		{ID: 255, Cond: 3},
		{ID: 1, Cond: -0x7ffffc01}, // 0x800003ff: the top bit and steps 1-10
	}
	var g m3Golden
	g.c(0x80).h(2).d(255).d(3).d(1).d(-0x7ffffc01)
	m3Check(t, "QuestList", NewQuestListPacket(quests), g.b)
}
