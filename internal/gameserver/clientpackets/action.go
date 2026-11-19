package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpAction is the opcode of Action.
const OpAction = 0x04

// Action IDs of Action and AttackRequest.
const (
	ActionClick      = 0 // a plain click
	ActionShiftClick = 1 // a click with Shift held: a GM gets the object's info
)

// Action is a click on an object. The first click targets it; a click on
// the current target interacts with it (talks to an NPC, for example).
type Action struct {
	ObjectID int32
	// OriginX, OriginY and OriginZ are the player's position as the client
	// sees it. Mobius reads and ignores them.
	OriginX, OriginY, OriginZ int32
	// ActionID is ActionClick or ActionShiftClick. Mobius answers any other
	// value with ActionFailed.
	ActionID uint8
}

// Shift reports whether the click was a Shift-click.
func (a Action) Shift() bool { return a.ActionID == ActionShiftClick }

// ParseAction parses an Action body (Mobius Action.readImpl).
func ParseAction(data []byte) (Action, error) {
	r := packets.NewReader(data)
	a := Action{
		ObjectID: r.ReadInt32(),
		OriginX:  r.ReadInt32(),
		OriginY:  r.ReadInt32(),
		OriginZ:  r.ReadInt32(),
		ActionID: r.ReadUint8(),
	}
	if err := r.Err(); err != nil {
		return Action{}, fmt.Errorf("Action: %w", err)
	}
	return a, nil
}
