package clientpackets

// OpRequestSkillCoolTime is the opcode of RequestSkillCoolTime.
const OpRequestSkillCoolTime = 0x9d

// RequestSkillCoolTime asks for the skill reuse times (the server answers
// with SkillCoolTime). It has no fields.
type RequestSkillCoolTime struct{}

// ParseRequestSkillCoolTime parses a RequestSkillCoolTime body. Like Mobius
// it reads nothing, so it never fails.
func ParseRequestSkillCoolTime([]byte) (RequestSkillCoolTime, error) {
	return RequestSkillCoolTime{}, nil
}
