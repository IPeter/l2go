package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpRequestActionUse is the opcode of RequestActionUse.
const OpRequestActionUse = 0x45

// Action IDs of RequestActionUse (the action window).
const (
	ActionUseSitStand = 0
	ActionUseWalkRun  = 1
)

// RequestActionUse is a click on an action in the action window.
type RequestActionUse struct {
	ActionID int32
	// CtrlPressed is set when the int the client sends is exactly 1, as in
	// Mobius.
	CtrlPressed bool
	// ShiftPressed is set when the byte the client sends is exactly 1.
	ShiftPressed bool
}

// ParseRequestActionUse parses a RequestActionUse body (Mobius
// RequestActionUse.readImpl).
func ParseRequestActionUse(data []byte) (RequestActionUse, error) {
	r := packets.NewReader(data)
	a := RequestActionUse{
		ActionID:     r.ReadInt32(),
		CtrlPressed:  r.ReadInt32() == 1,
		ShiftPressed: r.ReadUint8() == 1,
	}
	if err := r.Err(); err != nil {
		return RequestActionUse{}, fmt.Errorf("RequestActionUse: %w", err)
	}
	return a, nil
}
