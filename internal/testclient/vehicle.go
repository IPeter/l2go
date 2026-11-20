package testclient

import (
	"context"

	"github.com/IPeter/l2go/internal/packets"
)

// The vehicle packets (Plan 13, A14): boats, their passengers and the
// requests to board, leave and walk on a boat, in the Mobius Interlude
// layouts. The boat announcements are CreatureSay of type ChatBoat.

// Game server opcodes of the vehicle packets (Mobius ServerPackets).
const (
	OpVehicleInfo               byte = 0x59
	OpVehicleDeparture          byte = 0x5a
	OpVehicleCheckLocation      byte = 0x5b
	OpGetOnVehicle              byte = 0x5c
	OpGetOffVehicle             byte = 0x5d
	OpMoveToLocationInVehicle   byte = 0x71
	OpStopMoveInVehicle         byte = 0x72
	OpValidateLocationInVehicle byte = 0x73
	OpVehicleStarted            byte = 0xc0
)

// Game client opcodes of the vehicle requests (Mobius ClientPackets).
const (
	OpRequestGetOnVehicle            byte = 0x42
	OpRequestGetOffVehicle           byte = 0x43
	OpRequestMoveToLocationInVehicle byte = 0x5c
	OpCannotMoveAnymoreInVehicle     byte = 0x5d
)

// The decoder and recorder entries are added here rather than in the shared
// tables, so the vehicle packets stay in one file.
func init() {
	for op, p := range map[byte]serverPacket{
		OpVehicleInfo:               {"VehicleInfo", readVehicleInfo},
		OpVehicleDeparture:          {"VehicleDeparture", readVehicleDeparture},
		OpVehicleCheckLocation:      {"VehicleCheckLocation", readVehicleCheckLocation},
		OpGetOnVehicle:              {"GetOnVehicle", readGetOnVehicle},
		OpGetOffVehicle:             {"GetOffVehicle", readGetOffVehicle},
		OpMoveToLocationInVehicle:   {"MoveToLocationInVehicle", readMoveToLocationInVehicle},
		OpStopMoveInVehicle:         {"StopMoveInVehicle", readStopMoveInVehicle},
		OpValidateLocationInVehicle: {"ValidateLocationInVehicle", readValidateLocationInVehicle},
		OpVehicleStarted:            {"VehicleStarted", readVehicleStarted},
	} {
		serverPackets[op] = p
	}
	for op, name := range map[byte]string{
		OpRequestGetOnVehicle:            "RequestGetOnVehicle",
		OpRequestGetOffVehicle:           "RequestGetOffVehicle",
		OpRequestMoveToLocationInVehicle: "RequestMoveToLocationInVehicle",
		OpCannotMoveAnymoreInVehicle:     "CannotMoveAnymoreInVehicle",
	} {
		clientPacketNames[op] = name
	}
}

// VehicleInfo shows a boat where it is (0x59).
type VehicleInfo struct {
	ObjectID int32
	X, Y, Z  int32
	Heading  int32
}

// PacketName implements Packet.
func (VehicleInfo) PacketName() string { return "VehicleInfo" }

// Position returns where the boat is.
func (v VehicleInfo) Position() Position { return Position{v.X, v.Y, v.Z} }

func readVehicleInfo(f *fields) Packet {
	var p VehicleInfo
	p.ObjectID = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Heading = f.i32()
	return p
}

// VehicleCheckLocation has VehicleInfo's fields (0x5b); Mobius never sends
// it.
type VehicleCheckLocation struct{ VehicleInfo }

// PacketName implements Packet.
func (VehicleCheckLocation) PacketName() string { return "VehicleCheckLocation" }

func readVehicleCheckLocation(f *fields) Packet {
	return VehicleCheckLocation{readVehicleInfo(f).(VehicleInfo)}
}

// VehicleDeparture sends a boat towards a point (0x5a).
type VehicleDeparture struct {
	ObjectID      int32
	MoveSpeed     int32
	RotationSpeed int32
	X, Y, Z       int32
}

// PacketName implements Packet.
func (VehicleDeparture) PacketName() string { return "VehicleDeparture" }

// Destination returns where the boat sails to.
func (v VehicleDeparture) Destination() Position { return Position{v.X, v.Y, v.Z} }

func readVehicleDeparture(f *fields) Packet {
	var p VehicleDeparture
	p.ObjectID = f.i32()
	p.MoveSpeed = f.i32()
	p.RotationSpeed = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	return p
}

// VehicleStarted tells whether a boat moves (0xc0): State 1 started, 0
// stopped.
type VehicleStarted struct {
	ObjectID int32
	State    int32
}

// PacketName implements Packet.
func (VehicleStarted) PacketName() string { return "VehicleStarted" }

func readVehicleStarted(f *fields) Packet {
	return VehicleStarted{ObjectID: f.i32(), State: f.i32()}
}

// GetOnVehicle shows a player standing on a boat at a deck position (0x5c).
type GetOnVehicle struct {
	ObjectID int32
	BoatID   int32
	X, Y, Z  int32
}

// PacketName implements Packet.
func (GetOnVehicle) PacketName() string { return "GetOnVehicle" }

// Position returns the deck position.
func (g GetOnVehicle) Position() Position { return Position{g.X, g.Y, g.Z} }

func readGetOnVehicle(f *fields) Packet {
	return GetOnVehicle{ObjectID: f.i32(), BoatID: f.i32(), X: f.i32(), Y: f.i32(), Z: f.i32()}
}

// GetOffVehicle shows a player leaving a boat for a world position (0x5d).
type GetOffVehicle struct {
	ObjectID int32
	BoatID   int32
	X, Y, Z  int32
}

// PacketName implements Packet.
func (GetOffVehicle) PacketName() string { return "GetOffVehicle" }

// Position returns the world position.
func (g GetOffVehicle) Position() Position { return Position{g.X, g.Y, g.Z} }

func readGetOffVehicle(f *fields) Packet {
	return GetOffVehicle{ObjectID: f.i32(), BoatID: f.i32(), X: f.i32(), Y: f.i32(), Z: f.i32()}
}

// MoveToLocationInVehicle shows a passenger walking on the deck (0x71).
type MoveToLocationInVehicle struct {
	ObjectID            int32
	BoatID              int32
	DestX, DestY, DestZ int32
	X, Y, Z             int32
}

// PacketName implements Packet.
func (MoveToLocationInVehicle) PacketName() string { return "MoveToLocationInVehicle" }

func readMoveToLocationInVehicle(f *fields) Packet {
	var p MoveToLocationInVehicle
	p.ObjectID = f.i32()
	p.BoatID = f.i32()
	p.DestX = f.i32()
	p.DestY = f.i32()
	p.DestZ = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	return p
}

// StopMoveInVehicle stops a passenger on the deck (0x72).
type StopMoveInVehicle struct {
	ObjectID int32
	BoatID   int32
	X, Y, Z  int32
	Heading  int32
}

// PacketName implements Packet.
func (StopMoveInVehicle) PacketName() string { return "StopMoveInVehicle" }

func readStopMoveInVehicle(f *fields) Packet {
	return StopMoveInVehicle{ObjectID: f.i32(), BoatID: f.i32(), X: f.i32(), Y: f.i32(), Z: f.i32(), Heading: f.i32()}
}

// ValidateLocationInVehicle has StopMoveInVehicle's fields (0x73); Mobius
// never sends it.
type ValidateLocationInVehicle struct{ StopMoveInVehicle }

// PacketName implements Packet.
func (ValidateLocationInVehicle) PacketName() string { return "ValidateLocationInVehicle" }

func readValidateLocationInVehicle(f *fields) Packet {
	return ValidateLocationInVehicle{readStopMoveInVehicle(f).(StopMoveInVehicle)}
}

// encodeVehicleRequest builds RequestGetOnVehicle or RequestGetOffVehicle:
// the boat and a position.
func encodeVehicleRequest(op byte, boatID int32, pos Position) []byte {
	w := packets.NewWriter(17)
	w.WriteUint8(op)
	w.WriteInt32(boatID)
	writePosition(w, pos)
	return w.Bytes()
}

// encodeMoveToLocationInVehicle builds RequestMoveToLocationInVehicle.
func encodeMoveToLocationInVehicle(boatID int32, target, origin Position) []byte {
	w := packets.NewWriter(29)
	w.WriteUint8(OpRequestMoveToLocationInVehicle)
	w.WriteInt32(boatID)
	writePosition(w, target)
	writePosition(w, origin)
	return w.Bytes()
}

// encodeCannotMoveAnymoreInVehicle builds CannotMoveAnymoreInVehicle.
func encodeCannotMoveAnymoreInVehicle(boatID int32, pos Position, heading int32) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpCannotMoveAnymoreInVehicle)
	w.WriteInt32(boatID)
	writePosition(w, pos)
	w.WriteInt32(heading)
	return w.Bytes()
}

// GetOnVehicle asks to board the boat, standing at the deck position deck
// (RequestGetOnVehicle). The server answers with GetOnVehicle to everyone
// around, or ActionFailed; nothing is awaited.
func (c *GameConn) GetOnVehicle(ctx context.Context, boatID int32, deck Position) error {
	return c.send(ctx, encodeVehicleRequest(OpRequestGetOnVehicle, boatID, deck))
}

// GetOffVehicle asks to leave the boat for the world position to
// (RequestGetOffVehicle). Nothing is awaited.
func (c *GameConn) GetOffVehicle(ctx context.Context, boatID int32, to Position) error {
	return c.send(ctx, encodeVehicleRequest(OpRequestGetOffVehicle, boatID, to))
}

// MoveInVehicle walks on the deck from origin to target
// (RequestMoveToLocationInVehicle). Nothing is awaited.
func (c *GameConn) MoveInVehicle(ctx context.Context, boatID int32, target, origin Position) error {
	return c.send(ctx, encodeMoveToLocationInVehicle(boatID, target, origin))
}

// CannotMoveAnymoreInVehicle tells the server the character stopped on the
// deck at pos.
func (c *GameConn) CannotMoveAnymoreInVehicle(ctx context.Context, boatID int32, pos Position, heading int32) error {
	return c.send(ctx, encodeCannotMoveAnymoreInVehicle(boatID, pos, heading))
}
