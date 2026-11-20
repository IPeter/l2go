package testclient

// The door, static object and fence packets (Plan 13, A12) in the Mobius
// Interlude layouts. Mobius never sends DoorInfo (0x4C); a door shows
// itself with StaticObjectInfo and DoorStatusUpdate.

// Game server opcodes of the door, static object and fence packets (Mobius
// ServerPackets).
const (
	OpDoorStatusUpdate byte = 0x4d
	OpStaticObjectInfo byte = 0x99
	OpShowTownMap      byte = 0xde
)

// Ex sub-opcodes of the fence and drawing packets.
const (
	ExOpColosseumFenceInfo uint16 = 0x09
	ExOpServerPrimitive    uint16 = 0x24
)

// The decoder entries are added here rather than in the shared tables, so
// the door packets stay in one file.
func init() {
	serverPackets[OpDoorStatusUpdate] = serverPacket{"DoorStatusUpdate", readDoorStatusUpdate}
	serverPackets[OpStaticObjectInfo] = serverPacket{"StaticObjectInfo", readStaticObjectInfo}
	serverPackets[OpShowTownMap] = serverPacket{"ShowTownMap", readShowTownMap}
	exServerPackets[ExOpColosseumFenceInfo] = serverPacket{"ExColosseumFenceInfo", readExColosseumFenceInfo}
	exServerPackets[ExOpServerPrimitive] = serverPacket{"ExServerPrimitive", readExServerPrimitive}
}

// StaticObjectInfo shows a door or a static object (0x99).
type StaticObjectInfo struct {
	// StaticID is the door ID or the static object ID of the data.
	StaticID int32
	ObjectID int32
}

// PacketName implements Packet.
func (StaticObjectInfo) PacketName() string { return "StaticObjectInfo" }

func readStaticObjectInfo(f *fields) Packet {
	var p StaticObjectInfo
	p.StaticID = f.i32()
	p.ObjectID = f.i32()
	return p
}

// DoorStatusUpdate is a door's state (0x4D).
type DoorStatusUpdate struct {
	ObjectID int32
	Closed   bool
	// Damage is 0 (whole) to 6.
	Damage int32
	Enemy  bool
	DoorID int32
	MaxHP  int32
	CurHP  int32
}

// PacketName implements Packet.
func (DoorStatusUpdate) PacketName() string { return "DoorStatusUpdate" }

func readDoorStatusUpdate(f *fields) Packet {
	var p DoorStatusUpdate
	p.ObjectID = f.i32()
	p.Closed = f.flag32("closed")
	p.Damage = f.i32()
	p.Enemy = f.flag32("enemy")
	p.DoorID = f.i32()
	p.MaxHP = f.i32()
	p.CurHP = f.i32()
	if f.ok() && (p.Damage < 0 || p.Damage > 6) {
		f.failf("damage %d, want 0 to 6", p.Damage)
	}
	return p
}

// ShowTownMap opens a town map (0xDE).
type ShowTownMap struct {
	Texture string
	X, Y    int32
}

// PacketName implements Packet.
func (ShowTownMap) PacketName() string { return "ShowTownMap" }

func readShowTownMap(f *fields) Packet {
	var p ShowTownMap
	p.Texture = f.str()
	p.X = f.i32()
	p.Y = f.i32()
	return p
}

// ExColosseumFenceInfo shows a fence (FE:09).
type ExColosseumFenceInfo struct {
	ObjectID int32
	// State is 0 hidden, 1 opened, 2 closed.
	State         int32
	X, Y, Z       int32
	Width, Length int32
}

// PacketName implements Packet.
func (ExColosseumFenceInfo) PacketName() string { return "ExColosseumFenceInfo" }

func readExColosseumFenceInfo(f *fields) Packet {
	var p ExColosseumFenceInfo
	p.ObjectID = f.i32()
	p.State = f.i32()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	p.Width = f.i32()
	p.Length = f.i32()
	if f.ok() && (p.State < 0 || p.State > 2) {
		f.failf("state %d, want 0 to 2", p.State)
	}
	return p
}

// PrimitiveShape is a point (Line false) or a line ExServerPrimitive draws.
// R, G, B are the colour.
type PrimitiveShape struct {
	Line        bool
	Name        string
	R, G, B     int32
	NameColored bool
	X, Y, Z     int32
	// X2, Y2, Z2 end a line.
	X2, Y2, Z2 int32
}

// ExServerPrimitive draws points and lines (FE:24).
type ExServerPrimitive struct {
	Name    string
	X, Y, Z int32
	Shapes  []PrimitiveShape
}

// PacketName implements Packet.
func (ExServerPrimitive) PacketName() string { return "ExServerPrimitive" }

func readExServerPrimitive(f *fields) Packet {
	var p ExServerPrimitive
	p.Name = f.str()
	p.X = f.i32()
	p.Y = f.i32()
	p.Z = f.i32()
	if a, b := f.i32(), f.i32(); f.ok() && (a != 65535 || b != 65535) {
		f.failf("display range %d, %d, want 65535, 65535", a, b)
	}
	n := f.count32("count")
	for i := 0; i < n && f.ok(); i++ {
		var s PrimitiveShape
		switch kind := f.u8(); kind {
		case 1:
		case 2:
			s.Line = true
		default:
			f.failf("shape type %d", kind)
		}
		s.Name = f.str()
		s.R, s.G, s.B = f.i32(), f.i32(), f.i32()
		s.NameColored = f.flag32("name colored")
		s.X, s.Y, s.Z = f.i32(), f.i32(), f.i32()
		if s.Line {
			s.X2, s.Y2, s.Z2 = f.i32(), f.i32(), f.i32()
		}
		p.Shapes = append(p.Shapes, s)
	}
	return p
}
