package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpStartRotating is the opcode of StartRotating.
const OpStartRotating = 0x4a

// StartRotating reports that the player starts turning with the keyboard.
// Mobius echoes it to the viewers as StartRotation.
type StartRotating struct {
	Degree int32
	Side   int32
}

// ParseStartRotating parses a StartRotating body (Mobius
// StartRotating.readImpl).
func ParseStartRotating(data []byte) (StartRotating, error) {
	r := packets.NewReader(data)
	s := StartRotating{
		Degree: r.ReadInt32(),
		Side:   r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return StartRotating{}, fmt.Errorf("StartRotating: %w", err)
	}
	return s, nil
}
