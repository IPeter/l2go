package testclient

import (
	"bytes"
	"testing"
)

// The vehicle packets decode from the Mobius writeImpl layouts.
func TestDecodeVehiclePackets(t *testing.T) {
	const boat, me = 0x10000020, 0x10000001
	for _, tt := range []struct {
		name string
		raw  []byte
		want Packet
	}{
		// VehicleInfo: d object ID, d x, y, z, d heading.
		{"VehicleInfo", msg(0x59).i32(boat).i32(-96622).i32(261660).i32(-3610).i32(32768).b,
			VehicleInfo{ObjectID: boat, X: -96622, Y: 261660, Z: -3610, Heading: 32768}},
		{"VehicleCheckLocation", msg(0x5b).i32(boat).i32(1).i32(2).i32(3).i32(4).b,
			VehicleCheckLocation{VehicleInfo{ObjectID: boat, X: 1, Y: 2, Z: 3, Heading: 4}}},
		// VehicleDeparture: d object ID, d move speed, d rotation speed, d x, y, z.
		{"VehicleDeparture", msg(0x5a).i32(boat).i32(180).i32(800).i32(-121385).i32(261660).i32(-3610).b,
			VehicleDeparture{ObjectID: boat, MoveSpeed: 180, RotationSpeed: 800, X: -121385, Y: 261660, Z: -3610}},
		// VehicleStarted: d object ID, d state.
		{"VehicleStarted", msg(0xc0).i32(boat).i32(1).b, VehicleStarted{ObjectID: boat, State: 1}},
		// GetOnVehicle / GetOffVehicle: d player ID, d boat ID, d x, y, z.
		{"GetOnVehicle", msg(0x5c).i32(me).i32(boat).i32(10).i32(-20).i32(30).b,
			GetOnVehicle{ObjectID: me, BoatID: boat, X: 10, Y: -20, Z: 30}},
		{"GetOffVehicle", msg(0x5d).i32(me).i32(boat).i32(-96777).i32(258970).i32(-3623).b,
			GetOffVehicle{ObjectID: me, BoatID: boat, X: -96777, Y: 258970, Z: -3623}},
		// MoveToLocationInVehicle: d player ID, d boat ID, d dest x, y, z, d origin x, y, z.
		{"MoveToLocationInVehicle", msg(0x71).i32(me).i32(boat).i32(50).i32(-60).i32(70).i32(1).i32(2).i32(3).b,
			MoveToLocationInVehicle{ObjectID: me, BoatID: boat, DestX: 50, DestY: -60, DestZ: 70, X: 1, Y: 2, Z: 3}},
		// StopMoveInVehicle / ValidateLocationInVehicle: d player ID, d boat ID, d x, y, z, d heading.
		{"StopMoveInVehicle", msg(0x72).i32(me).i32(boat).i32(5).i32(-6).i32(7).i32(1024).b,
			StopMoveInVehicle{ObjectID: me, BoatID: boat, X: 5, Y: -6, Z: 7, Heading: 1024}},
		{"ValidateLocationInVehicle", msg(0x73).i32(me).i32(boat).i32(5).i32(-6).i32(7).i32(1024).b,
			ValidateLocationInVehicle{StopMoveInVehicle{ObjectID: me, BoatID: boat, X: 5, Y: -6, Z: 7, Heading: 1024}}},
	} {
		got, err := DecodeServerPacket(tt.raw)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
		if got.PacketName() != tt.name {
			t.Errorf("%s: named %q", tt.name, got.PacketName())
		}
		if _, err := DecodeServerPacket(tt.raw[:len(tt.raw)-1]); err == nil {
			t.Errorf("%s: a short packet decoded", tt.name)
		}
	}
}

// The vehicle requests use the Mobius readImpl layouts.
func TestEncodeVehicleRequests(t *testing.T) {
	const boat = 0x10000020
	for _, tt := range []struct {
		name      string
		got, want []byte
	}{
		{"RequestGetOnVehicle", encodeVehicleRequest(OpRequestGetOnVehicle, boat, Position{10, -20, 30}),
			msg(0x42).i32(boat).i32(10).i32(-20).i32(30).b},
		{"RequestGetOffVehicle", encodeVehicleRequest(OpRequestGetOffVehicle, boat, Position{1, 2, 3}),
			msg(0x43).i32(boat).i32(1).i32(2).i32(3).b},
		{"RequestMoveToLocationInVehicle", encodeMoveToLocationInVehicle(boat, Position{50, -60, 70}, Position{1, 2, 3}),
			msg(0x5c).i32(boat).i32(50).i32(-60).i32(70).i32(1).i32(2).i32(3).b},
		{"CannotMoveAnymoreInVehicle", encodeCannotMoveAnymoreInVehicle(boat, Position{5, -6, 7}, 1024),
			msg(0x5d).i32(boat).i32(5).i32(-6).i32(7).i32(1024).b},
	} {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s = % x, want % x", tt.name, tt.got, tt.want)
		}
		if clientPacketNames[tt.got[0]] != tt.name {
			t.Errorf("%s is recorded as %q", tt.name, clientPacketNames[tt.got[0]])
		}
	}
}
