package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// Sub-opcodes of the skill enchant requests (Mobius ExClientPackets
// REQUEST_EX_ENCHANT_SKILL_INFO and REQUEST_EX_ENCHANT_SKILL).
const (
	ExOpRequestExEnchantSkillInfo = 0x06
	ExOpRequestExEnchantSkill     = 0x07
)

// RequestExEnchantSkill is a click in the trainer's skill enchant window:
// RequestExEnchantSkillInfo asks for the cost and chance of an enchant
// level, RequestExEnchantSkill enchants. Both carry the same fields.
type RequestExEnchantSkill struct {
	SkillID, Level int32
}

// ParseRequestExEnchantSkill parses the body of RequestExEnchantSkillInfo
// or RequestExEnchantSkill, the data after the sub-opcode (Mobius readImpl
// of both): d skill ID, d level.
func ParseRequestExEnchantSkill(data []byte) (RequestExEnchantSkill, error) {
	r := packets.NewReader(data)
	req := RequestExEnchantSkill{SkillID: r.ReadInt32(), Level: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestExEnchantSkill{}, fmt.Errorf("RequestExEnchantSkill: %w", err)
	}
	return req, nil
}
