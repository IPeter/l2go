package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The door, static object and fence packets (Plan 13, A12), in the layouts
// of the Mobius Interlude writeImpl methods. Mobius never sends DoorInfo
// (0x4C): a door shows itself with StaticObjectInfo and DoorStatusUpdate.

// Opcodes of the door, static object and fence packets (Mobius
// ServerPackets).
const (
	OpDoorStatusUpdate = 0x4d
	OpStaticObjectInfo = 0x99
	OpShowTownMap      = 0xde
	// ExColosseumFenceInfo and ExServerPrimitive are Ex packets.
	ExColosseumFenceInfoSub = 0x09
	ExServerPrimitiveSub    = 0x24
)

// StaticObjectInfo shows a door or a static object (Mobius
// StaticObjectInfo): d static ID (the door ID or the static object ID), d
// object ID.
type StaticObjectInfo struct {
	StaticID int32
	ObjectID int32
}

// NewStaticObjectInfoPacket builds StaticObjectInfo.
func NewStaticObjectInfoPacket(s StaticObjectInfo) []byte {
	w := packets.NewWriter(9)
	w.WriteUint8(OpStaticObjectInfo)
	w.WriteInt32(s.StaticID)
	w.WriteInt32(s.ObjectID)
	return w.Bytes()
}

// DoorStatusUpdate is a door's state (Mobius DoorStatusUpdate): d object ID,
// d closed, d damage (0 to 6), d enemy (a siege is on), d door ID, d max HP,
// d current HP.
type DoorStatusUpdate struct {
	ObjectID int32
	Closed   bool
	Damage   int32
	Enemy    bool
	DoorID   int32
	MaxHP    int32
	CurHP    int32
}

// NewDoorStatusUpdatePacket builds DoorStatusUpdate.
func NewDoorStatusUpdatePacket(d DoorStatusUpdate) []byte {
	w := packets.NewWriter(29)
	w.WriteUint8(OpDoorStatusUpdate)
	w.WriteInt32(d.ObjectID)
	putBool32(w, d.Closed)
	w.WriteInt32(d.Damage)
	putBool32(w, d.Enemy)
	w.WriteInt32(d.DoorID)
	w.WriteInt32(d.MaxHP)
	w.WriteInt32(d.CurHP)
	return w.Bytes()
}

// ShowTownMap opens a town map (Mobius ShowTownMap): S texture, d x, d y.
// Mobius names the texture "town_map." + the static object's texture.
type ShowTownMap struct {
	Texture string
	X, Y    int32
}

// NewShowTownMapPacket builds ShowTownMap.
func NewShowTownMapPacket(m ShowTownMap) []byte {
	w := packets.NewWriter(11 + 2*len(m.Texture))
	w.WriteUint8(OpShowTownMap)
	w.WriteString(m.Texture)
	w.WriteInt32(m.X)
	w.WriteInt32(m.Y)
	return w.Bytes()
}

// ExColosseumFenceInfo shows a fence (Mobius ExColosseumFenceInfo): d
// object ID, d state (data.FenceClientState), d x, y, z, d width, d length.
type ExColosseumFenceInfo struct {
	ObjectID      int32
	State         int32
	X, Y, Z       int32
	Width, Length int32
}

// NewExColosseumFenceInfoPacket builds ExColosseumFenceInfo.
func NewExColosseumFenceInfoPacket(f ExColosseumFenceInfo) []byte {
	w := newExWriter(ExColosseumFenceInfoSub, 28)
	w.WriteInt32(f.ObjectID)
	w.WriteInt32(f.State)
	w.WriteInt32(f.X)
	w.WriteInt32(f.Y)
	w.WriteInt32(f.Z)
	w.WriteInt32(f.Width)
	w.WriteInt32(f.Length)
	return w.Bytes()
}

// PrimitivePoint is a point ExServerPrimitive draws, with its name over it.
// Color is 0xRRGGBB.
type PrimitivePoint struct {
	Name        string
	Color       int32
	NameColored bool
	X, Y, Z     int32
}

// PrimitiveLine is a line ExServerPrimitive draws from (X, Y, Z) to (X2, Y2,
// Z2), with its name over its middle.
type PrimitiveLine struct {
	PrimitivePoint
	X2, Y2, Z2 int32
}

// ExServerPrimitive draws points and lines on the client (Mobius
// ExServerPrimitive): S name (a later packet of the same name replaces the
// drawing), d x, y, z (the middle of the drawing), d 65535, d 65535, d
// count, then per point c 1, S name, d R, G, B, d name colored, d x, y, z,
// and per line c 2, the same, then d x2, y2, z2. Points come first.
type ExServerPrimitive struct {
	Name    string
	X, Y, Z int32
	Points  []PrimitivePoint
	Lines   []PrimitiveLine
}

// NewExServerPrimitivePacket builds ExServerPrimitive.
func NewExServerPrimitivePacket(p ExServerPrimitive) []byte {
	w := newExWriter(ExServerPrimitiveSub, 26+2*len(p.Name)+45*len(p.Points)+57*len(p.Lines))
	w.WriteString(p.Name)
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
	w.WriteInt32(65535)
	w.WriteInt32(65535)
	putCount32(w, len(p.Points)+len(p.Lines))
	for i := range p.Points {
		w.WriteUint8(1)
		putPrimitivePoint(w, &p.Points[i])
	}
	for i := range p.Lines {
		l := &p.Lines[i]
		w.WriteUint8(2)
		putPrimitivePoint(w, &l.PrimitivePoint)
		w.WriteInt32(l.X2)
		w.WriteInt32(l.Y2)
		w.WriteInt32(l.Z2)
	}
	return w.Bytes()
}

func putPrimitivePoint(w *packets.Writer, p *PrimitivePoint) {
	w.WriteString(p.Name)
	w.WriteInt32(p.Color >> 16 & 0xff)
	w.WriteInt32(p.Color >> 8 & 0xff)
	w.WriteInt32(p.Color & 0xff)
	putBool32(w, p.NameColored)
	w.WriteInt32(p.X)
	w.WriteInt32(p.Y)
	w.WriteInt32(p.Z)
}
