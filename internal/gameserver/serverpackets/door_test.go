package serverpackets

import "testing"

// Golden tests for the door, static object and fence packets. Each comment
// is the Mobius writeImpl field list.

func TestStaticObjectInfoPacket(t *testing.T) {
	// c 0x99, d static ID, d object ID.
	var g m3Golden
	g.c(0x99).d(24190001).d(0x60000010)
	m3Check(t, "StaticObjectInfo", NewStaticObjectInfoPacket(StaticObjectInfo{StaticID: 24190001, ObjectID: 0x60000010}), g.b)
}

func TestDoorStatusUpdatePacket(t *testing.T) {
	// c 0x4D, d object ID, d !isOpen, d damage, d isEnemy, d door ID, d max
	// HP, d (int) current HP.
	var g m3Golden
	g.c(0x4d).d(0x60000010).d(1).d(2).d(0).d(24190001).d(158250).d(100000)
	m3Check(t, "DoorStatusUpdate", NewDoorStatusUpdatePacket(DoorStatusUpdate{
		ObjectID: 0x60000010, Closed: true, Damage: 2, DoorID: 24190001, MaxHP: 158250, CurHP: 100000,
	}), g.b)
	var o m3Golden
	o.c(0x4d).d(7).d(0).d(0).d(1).d(8).d(1).d(1)
	m3Check(t, "DoorStatusUpdate open", NewDoorStatusUpdatePacket(DoorStatusUpdate{ObjectID: 7, Enemy: true, DoorID: 8, MaxHP: 1, CurHP: 1}), o.b)
}

func TestShowTownMapPacket(t *testing.T) {
	// c 0xDE, S texture, d x, d y.
	var g m3Golden
	g.c(0xde).s("town_map.town_map_talking_t00").d(365).d(152)
	m3Check(t, "ShowTownMap", NewShowTownMapPacket(ShowTownMap{Texture: "town_map.town_map_talking_t00", X: 365, Y: 152}), g.b)
}

func TestExColosseumFenceInfoPacket(t *testing.T) {
	// c 0xFE, h 0x09, d object ID, d client state, d x, y, z, d width, d
	// length.
	var g m3Golden
	g.c(0xfe).h(0x09).d(0x60000020).d(2).d(-114552).d(-251256).d(-2992).d(100).d(150)
	m3Check(t, "ExColosseumFenceInfo", NewExColosseumFenceInfoPacket(ExColosseumFenceInfo{
		ObjectID: 0x60000020, State: 2, X: -114552, Y: -251256, Z: -2992, Width: 100, Length: 150,
	}), g.b)
}

func TestExServerPrimitivePacket(t *testing.T) {
	// c 0xFE, h 0x24, S name, d x, y, z, d 65535, d 65535, d count; points:
	// c 1, S name, d R, d G, d B, d colored, d x, y, z; lines: c 2, the same
	// and d x2, y2, z2.
	var g m3Golden
	g.c(0xfe).h(0x24).s("Door1").d(10).d(20).d(-16000).d(65535).d(65535).d(2)
	g.c(1).s("p").d(0x12).d(0x34).d(0x56).d(1).d(1).d(2).d(3)
	g.c(2).s("").d(0xff).d(0).d(0).d(0).d(4).d(5).d(6).d(7).d(8).d(9)
	m3Check(t, "ExServerPrimitive", NewExServerPrimitivePacket(ExServerPrimitive{
		Name: "Door1", X: 10, Y: 20, Z: -16000,
		Points: []PrimitivePoint{{Name: "p", Color: 0x123456, NameColored: true, X: 1, Y: 2, Z: 3}},
		Lines:  []PrimitiveLine{{PrimitivePoint: PrimitivePoint{Color: 0xff0000, X: 4, Y: 5, Z: 6}, X2: 7, Y2: 8, Z2: 9}},
	}), g.b)
}
