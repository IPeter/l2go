package serverpackets

import "testing"

// Golden tests for the vehicle packets. Each comment is the Mobius
// writeImpl field list.

func TestVehicleInfoPacket(t *testing.T) {
	// c 0x59, d object ID, d x, y, z, d heading.
	v := VehicleInfo{ObjectID: 0x10000020, X: -96622, Y: 261660, Z: -3610, Heading: 32768}
	var g m3Golden
	g.c(0x59).d(0x10000020).d(-96622).d(261660).d(-3610).d(32768)
	m3Check(t, "VehicleInfo", NewVehicleInfoPacket(v), g.b)

	// VehicleCheckLocation: c 0x5B and the same fields.
	var c m3Golden
	c.c(0x5b).d(0x10000020).d(-96622).d(261660).d(-3610).d(32768)
	m3Check(t, "VehicleCheckLocation", NewVehicleCheckLocationPacket(v), c.b)
}

func TestVehicleDeparturePacket(t *testing.T) {
	// c 0x5A, d object ID, d move speed, d rotation speed, d x, y, z.
	v := VehicleDeparture{ObjectID: 0x10000020, MoveSpeed: 180, RotationSpeed: 800, X: -121385, Y: 261660, Z: -3610}
	var g m3Golden
	g.c(0x5a).d(0x10000020).d(180).d(800).d(-121385).d(261660).d(-3610)
	m3Check(t, "VehicleDeparture", NewVehicleDeparturePacket(v), g.b)
}

func TestVehicleStartedPacket(t *testing.T) {
	// c 0xC0, d object ID, d state.
	var g m3Golden
	g.c(0xc0).d(0x10000020).d(1)
	m3Check(t, "VehicleStarted", NewVehicleStartedPacket(VehicleStarted{ObjectID: 0x10000020, State: 1}), g.b)
}

func TestGetOnOffVehiclePackets(t *testing.T) {
	// c 0x5C / 0x5D, d player ID, d boat ID, d x, y, z.
	v := VehiclePassenger{ObjectID: 0x10000001, BoatID: 0x10000020, X: 10, Y: -20, Z: 30}
	var on, off m3Golden
	on.c(0x5c).d(0x10000001).d(0x10000020).d(10).d(-20).d(30)
	off.c(0x5d).d(0x10000001).d(0x10000020).d(10).d(-20).d(30)
	m3Check(t, "GetOnVehicle", NewGetOnVehiclePacket(v), on.b)
	m3Check(t, "GetOffVehicle", NewGetOffVehiclePacket(v), off.b)
}

func TestMoveToLocationInVehiclePacket(t *testing.T) {
	// c 0x71, d player ID, d boat ID, d destination x, y, z, d origin x, y, z.
	m := MoveToLocationInVehicle{ObjectID: 0x10000001, BoatID: 0x10000020, DestX: 50, DestY: -60, DestZ: 70, X: 1, Y: 2, Z: 3}
	var g m3Golden
	g.c(0x71).d(0x10000001).d(0x10000020).d(50).d(-60).d(70).d(1).d(2).d(3)
	m3Check(t, "MoveToLocationInVehicle", NewMoveToLocationInVehiclePacket(m), g.b)
}

func TestInVehicleLocationPackets(t *testing.T) {
	// c 0x72 / 0x73, d player ID, d boat ID, d x, y, z, d heading.
	v := InVehicleLocation{ObjectID: 0x10000001, BoatID: 0x10000020, X: 5, Y: -6, Z: 7, Heading: 1024}
	var stop, validate m3Golden
	stop.c(0x72).d(0x10000001).d(0x10000020).d(5).d(-6).d(7).d(1024)
	validate.c(0x73).d(0x10000001).d(0x10000020).d(5).d(-6).d(7).d(1024)
	m3Check(t, "StopMoveInVehicle", NewStopMoveInVehiclePacket(v), stop.b)
	m3Check(t, "ValidateLocationInVehicle", NewValidateLocationInVehiclePacket(v), validate.b)
}

func TestBoatSayPacket(t *testing.T) {
	// new CreatureSay(ChatType.BOAT, 801, THE_FERRY_IS_LEAVING_FOR_GLUDIN_HARBOR):
	// c 0x4A, d 0 (no sender), d 11 (BOAT), d 801, d 985.
	var g m3Golden
	g.c(0x4a).d(0).d(11).d(801).d(985)
	m3Check(t, "boat CreatureSay", NewBoatSayPacket(BoatSay{NameID: BoatSayNameID, MessageID: 985}), g.b)
}
