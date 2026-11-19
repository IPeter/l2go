package clientpackets

import (
	"errors"
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestAcquireSkillInfo is the opcode of RequestAcquireSkillInfo.
const OpRequestAcquireSkillInfo = 0x6b

// Acquire skill types, the serverpackets.AcquireSkillType values the client
// sends back (the ordinals of Mobius AcquireSkillType).
const (
	AcquireSkillClass   = 0
	AcquireSkillFishing = 1
	AcquireSkillPledge  = 2
)

// ErrUnknownAcquireSkillType reports a skill type that is not an
// AcquireSkill value. Mobius fails reading such a packet (its
// AcquireSkillType lookup throws) and drops it.
var ErrUnknownAcquireSkillType = errors.New("clientpackets: unknown acquire skill type")

// RequestAcquireSkillInfo asks what learning a skill costs, when the player
// picks it in the trainer's list. The answer is AcquireSkillInfo.
type RequestAcquireSkillInfo struct {
	// SkillID and Level are not checked here. Mobius ignores the request
	// when either is not positive.
	SkillID int32
	Level   int32
	Type    int32 // one of the AcquireSkill values
}

// ParseRequestAcquireSkillInfo parses a RequestAcquireSkillInfo body
// (Mobius RequestAcquireSkillInfo.readImpl): d skill ID, d level, d type. A
// type that is not an AcquireSkill value fails with
// ErrUnknownAcquireSkillType.
func ParseRequestAcquireSkillInfo(data []byte) (RequestAcquireSkillInfo, error) {
	r := packets.NewReader(data)
	req := RequestAcquireSkillInfo{
		SkillID: r.ReadInt32(),
		Level:   r.ReadInt32(),
		Type:    r.ReadInt32(),
	}
	err := r.Err()
	if err == nil {
		err = checkAcquireSkillType(req.Type)
	}
	if err != nil {
		return RequestAcquireSkillInfo{}, fmt.Errorf("RequestAcquireSkillInfo: %w", err)
	}
	return req, nil
}

// checkAcquireSkillType returns an error wrapping ErrUnknownAcquireSkillType
// when t is not an AcquireSkill value.
func checkAcquireSkillType(t int32) error {
	if t < AcquireSkillClass || t > AcquireSkillPledge {
		return fmt.Errorf("%w: %d", ErrUnknownAcquireSkillType, t)
	}
	return nil
}
