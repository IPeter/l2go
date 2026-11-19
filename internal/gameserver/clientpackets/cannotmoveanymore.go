package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpCannotMoveAnymore is the opcode of CannotMoveAnymore.
const OpCannotMoveAnymore = 0x36

// CannotMoveAnymore reports that the client stopped the player because
// something blocks the way.
type CannotMoveAnymore struct {
	X, Y, Z int32
	Heading int32
}

// ParseCannotMoveAnymore parses a CannotMoveAnymore body (Mobius
// CannotMoveAnymore.readImpl).
func ParseCannotMoveAnymore(data []byte) (CannotMoveAnymore, error) {
	r := packets.NewReader(data)
	c := CannotMoveAnymore{
		X:       r.ReadInt32(),
		Y:       r.ReadInt32(),
		Z:       r.ReadInt32(),
		Heading: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return CannotMoveAnymore{}, fmt.Errorf("CannotMoveAnymore: %w", err)
	}
	return c, nil
}
