package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpValidatePosition is the opcode of ValidatePosition.
const OpValidatePosition = 0x48

// ValidatePosition reports where the client shows the player. The client
// sends it regularly while moving and when it stops.
type ValidatePosition struct {
	X, Y, Z   int32
	Heading   int32
	VehicleID int32 // read but not used by Mobius
}

// ParseValidatePosition parses a ValidatePosition body (Mobius
// ValidatePosition.readImpl).
func ParseValidatePosition(data []byte) (ValidatePosition, error) {
	r := packets.NewReader(data)
	v := ValidatePosition{
		X:         r.ReadInt32(),
		Y:         r.ReadInt32(),
		Z:         r.ReadInt32(),
		Heading:   r.ReadInt32(),
		VehicleID: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return ValidatePosition{}, fmt.Errorf("ValidatePosition: %w", err)
	}
	return v, nil
}
