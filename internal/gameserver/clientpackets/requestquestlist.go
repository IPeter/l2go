package clientpackets

// OpRequestQuestList is the opcode of RequestQuestList.
const OpRequestQuestList = 0x63

// RequestQuestList asks for the quest list (the server answers with
// QuestList). It has no fields.
type RequestQuestList struct{}

// ParseRequestQuestList parses a RequestQuestList body. Like Mobius it
// reads nothing, so it never fails.
func ParseRequestQuestList([]byte) (RequestQuestList, error) { return RequestQuestList{}, nil }
