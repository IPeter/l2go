package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpQuestList is the opcode of QuestList.
const OpQuestList = 0x80

// QuestListEntry is one active quest.
type QuestListEntry struct {
	ID int32
	// Cond is the current step. With the top bit set, the other bits mark
	// the steps done or active one by one, so skipped steps can be hidden
	// (Mobius "__compltdStateFlags").
	Cond int32
}

// NewQuestListPacket builds QuestList (Mobius QuestList.writeImpl), the
// player's active quests. An empty list is valid.
func NewQuestListPacket(quests []QuestListEntry) []byte {
	w := packets.NewWriter(3 + len(quests)*8)
	w.WriteUint8(OpQuestList)
	putCount16(w, len(quests))
	for _, q := range quests {
		w.WriteInt32(q.ID)
		w.WriteInt32(q.Cond)
	}
	return w.Bytes()
}
