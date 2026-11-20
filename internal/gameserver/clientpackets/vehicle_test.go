package clientpackets

import "testing"

func TestVehicleOpcodes(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want int
	}{
		{"RequestGetOnVehicle", OpRequestGetOnVehicle, 0x42},
		{"RequestGetOffVehicle", OpRequestGetOffVehicle, 0x43},
		{"RequestMoveToLocationInVehicle", OpRequestMoveToLocationInVehicle, 0x5c},
		{"CannotMoveAnymoreInVehicle", OpCannotMoveAnymoreInVehicle, 0x5d},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// Mobius RequestGetOnVehicle / RequestGetOffVehicle.readImpl: d boat ID, d
// x, y, z (16 bytes).
func TestParseVehicleRequest(t *testing.T) {
	var b m3Body
	b.d(0x10000020).d(10).d(-20).d(30)
	want := VehicleRequest{BoatID: 0x10000020, X: 10, Y: -20, Z: 30}
	got, err := ParseVehicleRequest(b.b)
	m3Parsed(t, "vehicle request", got, err, want)
	got, err = ParseVehicleRequest(append(b.b, 1))
	m3Parsed(t, "vehicle request with trailing data", got, err, want)
	m3Truncations(t, "vehicle request", b.b, ParseVehicleRequest)
}

// Mobius RequestMoveToLocationInVehicle.readImpl: d boat ID, d target x, y,
// z, d origin x, y, z (28 bytes).
func TestParseMoveToLocationInVehicle(t *testing.T) {
	var b m3Body
	b.d(0x10000020).d(50).d(-60).d(70).d(1).d(2).d(3)
	want := MoveToLocationInVehicle{BoatID: 0x10000020, TargetX: 50, TargetY: -60, TargetZ: 70, OriginX: 1, OriginY: 2, OriginZ: 3}
	got, err := ParseMoveToLocationInVehicle(b.b)
	m3Parsed(t, "RequestMoveToLocationInVehicle", got, err, want)
	m3Truncations(t, "RequestMoveToLocationInVehicle", b.b, ParseMoveToLocationInVehicle)
}

// Mobius CannotMoveAnymoreInVehicle.readImpl: d boat ID, d x, y, z, d
// heading (20 bytes).
func TestParseCannotMoveAnymoreInVehicle(t *testing.T) {
	var b m3Body
	b.d(0x10000020).d(5).d(-6).d(7).d(1024)
	want := CannotMoveAnymoreInVehicle{BoatID: 0x10000020, X: 5, Y: -6, Z: 7, Heading: 1024}
	got, err := ParseCannotMoveAnymoreInVehicle(b.b)
	m3Parsed(t, "CannotMoveAnymoreInVehicle", got, err, want)
	m3Truncations(t, "CannotMoveAnymoreInVehicle", b.b, ParseCannotMoveAnymoreInVehicle)
}

func FuzzParseVehicleRequests(f *testing.F) {
	var b m3Body
	b.d(1).d(2).d(3).d(4).d(5).d(6).d(7)
	m3Fuzz(f, ParseVehicleRequest, 16, b.b)
}
