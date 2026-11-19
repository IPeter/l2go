package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpFinishRotating is the opcode of FinishRotating.
const OpFinishRotating = 0x4b

// FinishRotating reports that the player stopped turning with the
// keyboard. Mobius sets the heading to Degree and echoes it to the viewers
// as StopRotation.
type FinishRotating struct {
	Degree  int32
	Unknown int32 // read and ignored by Mobius
}

// ParseFinishRotating parses a FinishRotating body (Mobius
// FinishRotating.readImpl).
func ParseFinishRotating(data []byte) (FinishRotating, error) {
	r := packets.NewReader(data)
	f := FinishRotating{
		Degree:  r.ReadInt32(),
		Unknown: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return FinishRotating{}, fmt.Errorf("FinishRotating: %w", err)
	}
	return f, nil
}
