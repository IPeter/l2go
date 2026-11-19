package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpMoveToLocation is the opcode of the client MoveToLocation
// (MoveBackwardToLocation in older clients).
const OpMoveToLocation = 0x01

// Movement modes of MoveToLocation. Mobius treats every mode other than
// MoveModeMouse as keyboard movement.
const (
	MoveModeKeyboard = 0 // cursor keys
	MoveModeMouse    = 1 // a click
)

// MoveToLocation asks to move the player to a target position.
type MoveToLocation struct {
	TargetX int32
	TargetY int32
	TargetZ int32
	// The position where the client thinks the player is. Mobius uses it
	// only for keyboard movement, when within 1,000 units of the server's
	// position.
	OriginX int32
	OriginY int32
	OriginZ int32
	Mode    int32 // MoveModeKeyboard or MoveModeMouse
}

// ParseMoveToLocation parses a MoveToLocation body (Mobius
// MoveToLocation.readImpl).
func ParseMoveToLocation(data []byte) (MoveToLocation, error) {
	r := packets.NewReader(data)
	m := MoveToLocation{
		TargetX: r.ReadInt32(),
		TargetY: r.ReadInt32(),
		TargetZ: r.ReadInt32(),
		OriginX: r.ReadInt32(),
		OriginY: r.ReadInt32(),
		OriginZ: r.ReadInt32(),
		Mode:    r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return MoveToLocation{}, fmt.Errorf("MoveToLocation: %w", err)
	}
	return m, nil
}
