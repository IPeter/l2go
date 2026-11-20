package serverpackets

// ExOpShowQuestMark is the sub-opcode of ExShowQuestMark.
const ExOpShowQuestMark = 0x1a

// NewExShowQuestMarkPacket builds ExShowQuestMark (Mobius
// ExShowQuestMark.writeImpl): d quest ID. The client marks the quest in the
// quest log as updated. Mobius sends it after QuestList when a quest's cond
// is set to a positive value.
func NewExShowQuestMarkPacket(questID int32) []byte {
	w := newExWriter(ExOpShowQuestMark, 4)
	w.WriteInt32(questID)
	return w.Bytes()
}
