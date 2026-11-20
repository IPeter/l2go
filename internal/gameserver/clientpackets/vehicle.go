package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// The vehicle requests (Plan 13, A14): boarding, leaving and walking on a
// boat, in the layouts of the Mobius Interlude readImpl methods.

// Opcodes of the vehicle requests (Mobius ClientPackets).
const (
	OpRequestGetOnVehicle            = 0x42
	OpRequestGetOffVehicle           = 0x43
	OpRequestMoveToLocationInVehicle = 0x5c
	OpCannotMoveAnymoreInVehicle     = 0x5d
)

// VehicleRequest is RequestGetOnVehicle and RequestGetOffVehicle: d boat
// object ID, d x, y, z. To get on, the position is where the player stands
// on the deck; to get off, where it steps into the world.
type VehicleRequest struct {
	BoatID  int32
	X, Y, Z int32
}

// ParseVehicleRequest parses a RequestGetOnVehicle or RequestGetOffVehicle
// body (Mobius RequestGetOnVehicle.readImpl, RequestGetOffVehicle.readImpl).
func ParseVehicleRequest(data []byte) (VehicleRequest, error) {
	r := packets.NewReader(data)
	v := VehicleRequest{
		BoatID: r.ReadInt32(),
		X:      r.ReadInt32(),
		Y:      r.ReadInt32(),
		Z:      r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return VehicleRequest{}, fmt.Errorf("vehicle request: %w", err)
	}
	return v, nil
}

// MoveToLocationInVehicle is RequestMoveToLocationInVehicle: d boat object
// ID, d target x, y, z, d origin x, y, z, deck positions.
type MoveToLocationInVehicle struct {
	BoatID                    int32
	TargetX, TargetY, TargetZ int32
	OriginX, OriginY, OriginZ int32
}

// ParseMoveToLocationInVehicle parses a RequestMoveToLocationInVehicle body
// (Mobius RequestMoveToLocationInVehicle.readImpl).
func ParseMoveToLocationInVehicle(data []byte) (MoveToLocationInVehicle, error) {
	r := packets.NewReader(data)
	m := MoveToLocationInVehicle{
		BoatID:  r.ReadInt32(),
		TargetX: r.ReadInt32(),
		TargetY: r.ReadInt32(),
		TargetZ: r.ReadInt32(),
		OriginX: r.ReadInt32(),
		OriginY: r.ReadInt32(),
		OriginZ: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return MoveToLocationInVehicle{}, fmt.Errorf("RequestMoveToLocationInVehicle: %w", err)
	}
	return m, nil
}

// CannotMoveAnymoreInVehicle reports that a passenger stopped on the deck:
// d boat object ID, d x, y, z, d heading.
type CannotMoveAnymoreInVehicle struct {
	BoatID  int32
	X, Y, Z int32
	Heading int32
}

// ParseCannotMoveAnymoreInVehicle parses a CannotMoveAnymoreInVehicle body
// (Mobius CannotMoveAnymoreInVehicle.readImpl).
func ParseCannotMoveAnymoreInVehicle(data []byte) (CannotMoveAnymoreInVehicle, error) {
	r := packets.NewReader(data)
	c := CannotMoveAnymoreInVehicle{
		BoatID:  r.ReadInt32(),
		X:       r.ReadInt32(),
		Y:       r.ReadInt32(),
		Z:       r.ReadInt32(),
		Heading: r.ReadInt32(),
	}
	if err := r.Err(); err != nil {
		return CannotMoveAnymoreInVehicle{}, fmt.Errorf("CannotMoveAnymoreInVehicle: %w", err)
	}
	return c, nil
}
