package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpAttackRequest is the opcode of AttackRequest.
const OpAttackRequest = 0x0a

// AttackRequest asks to attack an object, for example with a click on a
// monster that is already the target.
type AttackRequest struct {
	ObjectID int32
	// OriginX, OriginY and OriginZ are the player's position as the client
	// sees it. Mobius reads and ignores them.
	OriginX, OriginY, OriginZ int32
	// AttackID is ActionClick or ActionShiftClick. Mobius reads and ignores
	// it.
	AttackID uint8
}

// ParseAttackRequest parses an AttackRequest body (Mobius
// AttackRequest.readImpl).
func ParseAttackRequest(data []byte) (AttackRequest, error) {
	r := packets.NewReader(data)
	a := AttackRequest{
		ObjectID: r.ReadInt32(),
		OriginX:  r.ReadInt32(),
		OriginY:  r.ReadInt32(),
		OriginZ:  r.ReadInt32(),
		AttackID: r.ReadUint8(),
	}
	if err := r.Err(); err != nil {
		return AttackRequest{}, fmt.Errorf("AttackRequest: %w", err)
	}
	return a, nil
}
