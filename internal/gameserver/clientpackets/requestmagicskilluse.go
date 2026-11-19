package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestMagicSkillUse is the opcode of RequestMagicSkillUse.
const OpRequestMagicSkillUse = 0x2f

// RequestMagicSkillUse asks to use a skill, from the skill window or the
// shortcut bar. The client sends no level: the player uses the level it
// knows.
type RequestMagicSkillUse struct {
	SkillID int32
	// CtrlPressed forces the skill on the target (Mobius forceUse). It is
	// set when the int the client sends is not 0, as in Mobius.
	CtrlPressed bool
	// ShiftPressed casts without moving to the target (Mobius dontMove). It
	// is set when the byte the client sends is not 0.
	ShiftPressed bool
}

// ParseRequestMagicSkillUse parses a RequestMagicSkillUse body (Mobius
// RequestMagicSkillUse.readImpl): d skill ID, d ctrl, c shift.
func ParseRequestMagicSkillUse(data []byte) (RequestMagicSkillUse, error) {
	r := packets.NewReader(data)
	req := RequestMagicSkillUse{
		SkillID:      r.ReadInt32(),
		CtrlPressed:  r.ReadInt32() != 0,
		ShiftPressed: r.ReadUint8() != 0,
	}
	if err := r.Err(); err != nil {
		return RequestMagicSkillUse{}, fmt.Errorf("RequestMagicSkillUse: %w", err)
	}
	return req, nil
}
