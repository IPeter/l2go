package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestAcquireSkill is the opcode of RequestAcquireSkill.
const OpRequestAcquireSkill = 0x6c

// RequestAcquireSkill asks to learn a skill at a trainer.
type RequestAcquireSkill struct {
	// SkillID and Level are not checked here. Mobius punishes a level
	// outside 1-1000 or an ID below 1 as an illegal action.
	SkillID int32
	Level   int32
	Type    int32 // one of the AcquireSkill values
}

// ParseRequestAcquireSkill parses a RequestAcquireSkill body (Mobius
// RequestAcquireSkill.readImpl): d skill ID, d level, d type. A type that
// is not an AcquireSkill value fails with ErrUnknownAcquireSkillType.
func ParseRequestAcquireSkill(data []byte) (RequestAcquireSkill, error) {
	r := packets.NewReader(data)
	req := RequestAcquireSkill{
		SkillID: r.ReadInt32(),
		Level:   r.ReadInt32(),
		Type:    r.ReadInt32(),
	}
	err := r.Err()
	if err == nil {
		err = checkAcquireSkillType(req.Type)
	}
	if err != nil {
		return RequestAcquireSkill{}, fmt.Errorf("RequestAcquireSkill: %w", err)
	}
	return req, nil
}
