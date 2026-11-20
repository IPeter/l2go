package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The vehicle packets (Plan 13, A14): the boats and their passengers, in
// the layouts of the Mobius Interlude writeImpl methods.

// Opcodes of the vehicle packets (Mobius ServerPackets).
const (
	OpVehicleInfo               = 0x59
	OpVehicleDeparture          = 0x5a
	OpVehicleCheckLocation      = 0x5b
	OpGetOnVehicle              = 0x5c
	OpGetOffVehicle             = 0x5d
	OpMoveToLocationInVehicle   = 0x71
	OpStopMoveInVehicle         = 0x72
	OpValidateLocationInVehicle = 0x73
	OpVehicleStarted            = 0xc0
)

// VehicleInfo shows a boat where it is (Mobius VehicleInfo): d object ID,
// d x, y, z, d heading.
type VehicleInfo struct {
	ObjectID int32
	X, Y, Z  int32
	Heading  int32
}

// NewVehicleInfoPacket builds VehicleInfo.
func NewVehicleInfoPacket(v VehicleInfo) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpVehicleInfo)
	w.WriteInt32(v.ObjectID)
	w.WriteInt32(v.X)
	w.WriteInt32(v.Y)
	w.WriteInt32(v.Z)
	w.WriteInt32(v.Heading)
	return w.Bytes()
}

// VehicleDeparture sends a boat towards a point (Mobius VehicleDeparture):
// d object ID, d move speed, d rotation speed, d destination x, y, z.
type VehicleDeparture struct {
	ObjectID      int32
	MoveSpeed     int32
	RotationSpeed int32
	X, Y, Z       int32
}

// NewVehicleDeparturePacket builds VehicleDeparture.
func NewVehicleDeparturePacket(v VehicleDeparture) []byte {
	w := packets.NewWriter(25)
	w.WriteUint8(OpVehicleDeparture)
	w.WriteInt32(v.ObjectID)
	w.WriteInt32(v.MoveSpeed)
	w.WriteInt32(v.RotationSpeed)
	w.WriteInt32(v.X)
	w.WriteInt32(v.Y)
	w.WriteInt32(v.Z)
	return w.Bytes()
}

// NewVehicleCheckLocationPacket builds VehicleCheckLocation (Mobius
// VehicleCheckLocation): the same fields as VehicleInfo. Mobius never sends
// it.
func NewVehicleCheckLocationPacket(v VehicleInfo) []byte {
	b := NewVehicleInfoPacket(v)
	b[0] = OpVehicleCheckLocation
	return b
}

// VehicleStarted tells whether a boat moves (Mobius VehicleStarted): d
// object ID, d state (1 started, 0 stopped).
type VehicleStarted struct {
	ObjectID int32
	State    int32
}

// NewVehicleStartedPacket builds VehicleStarted.
func NewVehicleStartedPacket(v VehicleStarted) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpVehicleStarted)
	w.WriteInt32(v.ObjectID)
	w.WriteInt32(v.State)
	return w.Bytes()
}

// VehiclePassenger is a player on a boat at a position on its deck: the
// fields of GetOnVehicle and GetOffVehicle (d player ID, d boat ID, d x, y,
// z).
type VehiclePassenger struct {
	ObjectID int32
	BoatID   int32
	X, Y, Z  int32
}

// NewGetOnVehiclePacket builds GetOnVehicle (Mobius GetOnVehicle): the
// player stands on the boat at the deck position.
func NewGetOnVehiclePacket(v VehiclePassenger) []byte {
	return vehiclePassenger(OpGetOnVehicle, v)
}

// NewGetOffVehiclePacket builds GetOffVehicle (Mobius GetOffVehicle): the
// player left the boat for the world position.
func NewGetOffVehiclePacket(v VehiclePassenger) []byte {
	return vehiclePassenger(OpGetOffVehicle, v)
}

func vehiclePassenger(op byte, v VehiclePassenger) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(op)
	w.WriteInt32(v.ObjectID)
	w.WriteInt32(v.BoatID)
	w.WriteInt32(v.X)
	w.WriteInt32(v.Y)
	w.WriteInt32(v.Z)
	return w.Bytes()
}

// MoveToLocationInVehicle shows a passenger walking on the deck (Mobius
// MoveToLocationInVehicle): d player ID, d boat ID, d destination x, y, z,
// d origin x, y, z, all deck positions.
type MoveToLocationInVehicle struct {
	ObjectID            int32
	BoatID              int32
	DestX, DestY, DestZ int32
	X, Y, Z             int32
}

// NewMoveToLocationInVehiclePacket builds MoveToLocationInVehicle.
func NewMoveToLocationInVehiclePacket(m MoveToLocationInVehicle) []byte {
	w := packets.NewWriter(33)
	w.WriteUint8(OpMoveToLocationInVehicle)
	w.WriteInt32(m.ObjectID)
	w.WriteInt32(m.BoatID)
	w.WriteInt32(m.DestX)
	w.WriteInt32(m.DestY)
	w.WriteInt32(m.DestZ)
	w.WriteInt32(m.X)
	w.WriteInt32(m.Y)
	w.WriteInt32(m.Z)
	return w.Bytes()
}

// InVehicleLocation is a passenger's deck position and heading: the fields
// of StopMoveInVehicle and ValidateLocationInVehicle (d player ID, d boat
// ID, d x, y, z, d heading).
type InVehicleLocation struct {
	ObjectID int32
	BoatID   int32
	X, Y, Z  int32
	Heading  int32
}

// NewStopMoveInVehiclePacket builds StopMoveInVehicle (Mobius
// StopMoveInVehicle).
func NewStopMoveInVehiclePacket(v InVehicleLocation) []byte {
	return inVehicleLocation(OpStopMoveInVehicle, v)
}

// NewValidateLocationInVehiclePacket builds ValidateLocationInVehicle
// (Mobius ValidateLocationInVehicle). Mobius never sends it.
func NewValidateLocationInVehiclePacket(v InVehicleLocation) []byte {
	return inVehicleLocation(OpValidateLocationInVehicle, v)
}

func inVehicleLocation(op byte, v InVehicleLocation) []byte {
	w := packets.NewWriter(25)
	w.WriteUint8(op)
	w.WriteInt32(v.ObjectID)
	w.WriteInt32(v.BoatID)
	w.WriteInt32(v.X)
	w.WriteInt32(v.Y)
	w.WriteInt32(v.Z)
	w.WriteInt32(v.Heading)
	return w.Bytes()
}

// BoatSay is a boat announcement (Mobius new CreatureSay(ChatType.BOAT,
// charId, systemMessageId)): a CreatureSay without a speaker whose name is
// the character ID NameID and whose text is the system message MessageID.
type BoatSay struct {
	NameID    int32
	MessageID int32
}

// BoatSayNameID is the character ID of every Mobius ferry announcement.
const BoatSayNameID = 801

// NewBoatSayPacket builds the boat form of CreatureSay: d 0 (no speaker), d
// type BOAT, d character ID, d system message ID.
func NewBoatSayPacket(s BoatSay) []byte {
	w := packets.NewWriter(17)
	w.WriteUint8(OpCreatureSay)
	w.WriteInt32(0)
	w.WriteInt32(int32(ChatBoat))
	w.WriteInt32(s.NameID)
	w.WriteInt32(s.MessageID)
	return w.Bytes()
}
