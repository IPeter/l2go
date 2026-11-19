package serverpackets

import "testing"

// Golden tests for the movement and world packets. Each comment is the
// Mobius writeImpl field list.

func TestMoveToLocationPacket(t *testing.T) {
	// c 0x01, d object ID, d dest x, y, z, d current x, y, z.
	m := MoveToLocation{ObjectID: 0x10000001, X: 1, Y: 2, Z: 3, DestX: -100, DestY: 200, DestZ: -300}
	var g m3Golden
	g.c(0x01).d(0x10000001).d(-100).d(200).d(-300).d(1).d(2).d(3)
	m3Check(t, "MoveToLocation", NewMoveToLocationPacket(m), g.b)
}

func TestStopMovePacket(t *testing.T) {
	// c 0x47, d object ID, d x, y, z, d heading.
	s := StopMove{ObjectID: 0x10000001, X: -71338, Y: 258271, Z: -3104, Heading: 49152}
	var g m3Golden
	g.c(0x47).d(0x10000001).d(-71338).d(258271).d(-3104).d(49152)
	m3Check(t, "StopMove", NewStopMovePacket(s), g.b)
}

func TestDeleteObjectPacket(t *testing.T) {
	// c 0x12, d object ID, d 1.
	m3Check(t, "DeleteObject", NewDeleteObjectPacket(0x10000001), unhex(t, "12 01000010 01000000"))
}

func TestTeleportToLocationPacket(t *testing.T) {
	// c 0x28, d object ID, d x, y, z, d 0 (fade), d heading.
	tp := TeleportToLocation{ObjectID: 0x10000001, X: 83400, Y: 147943, Z: -3404, Heading: 100}
	var g m3Golden
	g.c(0x28).d(0x10000001).d(83400).d(147943).d(-3404).d(0).d(100)
	m3Check(t, "TeleportToLocation", NewTeleportToLocationPacket(tp), g.b)
}

func TestChangeMoveTypePacket(t *testing.T) {
	// c 0x2E, d object ID, d 1 running or 0 walking, d 0.
	m3Check(t, "running", NewChangeMoveTypePacket(ChangeMoveType{ObjectID: 5, Running: true}),
		unhex(t, "2e 05000000 01000000 00000000"))
	m3Check(t, "walking", NewChangeMoveTypePacket(ChangeMoveType{ObjectID: 5}),
		unhex(t, "2e 05000000 00000000 00000000"))
}

func TestChangeWaitTypePacket(t *testing.T) {
	// c 0x2F, d object ID, d wait type, d x, y, z.
	c := ChangeWaitType{ObjectID: 0x10000001, Type: WaitSitting, X: 10, Y: -20, Z: 30}
	var g m3Golden
	g.c(0x2f).d(0x10000001).d(0).d(10).d(-20).d(30)
	m3Check(t, "sitting", NewChangeWaitTypePacket(c), g.b)

	c.Type = WaitStanding
	var g2 m3Golden
	g2.c(0x2f).d(0x10000001).d(1).d(10).d(-20).d(30)
	m3Check(t, "standing", NewChangeWaitTypePacket(c), g2.b)

	if WaitStartFakeDeath != 2 || WaitStopFakeDeath != 3 {
		t.Errorf("fake death wait types = %d, %d; want 2, 3", WaitStartFakeDeath, WaitStopFakeDeath)
	}
}

func TestStartRotationPacket(t *testing.T) {
	// c 0x62, d object ID, d degree, d side, d speed.
	s := StartRotation{ObjectID: 0x10000001, Degree: 16384, Side: -1, Speed: 0}
	var g m3Golden
	g.c(0x62).d(0x10000001).d(16384).d(-1).d(0)
	m3Check(t, "StartRotation", NewStartRotationPacket(s), g.b)
}

func TestStopRotationPacket(t *testing.T) {
	// c 0x63, d object ID, d degree, d speed, c 0.
	s := StopRotation{ObjectID: 0x10000001, Degree: 32768, Speed: 0}
	var g m3Golden
	g.c(0x63).d(0x10000001).d(32768).d(0).c(0)
	m3Check(t, "StopRotation", NewStopRotationPacket(s), g.b)
}

func TestValidateLocationPacket(t *testing.T) {
	// c 0x61, d object ID, d x, y, z, d heading.
	v := ValidateLocation{ObjectID: 0x10000001, X: -71338, Y: 258271, Z: -3104, Heading: 12345}
	var g m3Golden
	g.c(0x61).d(0x10000001).d(-71338).d(258271).d(-3104).d(12345)
	m3Check(t, "ValidateLocation", NewValidateLocationPacket(v), g.b)
}
