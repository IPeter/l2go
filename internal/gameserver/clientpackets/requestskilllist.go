package clientpackets

// OpRequestSkillList is the opcode of RequestSkillList.
const OpRequestSkillList = 0x3f

// RequestSkillList asks for the player's skills (the server answers with
// SkillList). It has no fields.
type RequestSkillList struct{}

// ParseRequestSkillList parses a RequestSkillList body. Like Mobius it reads
// nothing, so it never fails.
func ParseRequestSkillList([]byte) (RequestSkillList, error) {
	return RequestSkillList{}, nil
}
