package testclient

import "context"

// The Ex server sub-opcodes of fishing (Plan 14, M11g; Mobius ServerPackets
// EX_FISHING_*).
const (
	ExOpFishingStart       uint16 = 0x13
	ExOpFishingEnd         uint16 = 0x14
	ExOpFishingStartCombat uint16 = 0x15
	ExOpFishingHpRegen     uint16 = 0x16
)

// ExOpRequestExFishRanking is the client's fish ranking button (D0 1F).
const ExOpRequestExFishRanking uint16 = 0x1f

func init() {
	exServerPackets[ExOpFishingStart] = serverPacket{"ExFishingStart", readExFishingStart}
	exServerPackets[ExOpFishingEnd] = serverPacket{"ExFishingEnd", readExFishingEnd}
	exServerPackets[ExOpFishingStartCombat] = serverPacket{"ExFishingStartCombat", readExFishingStartCombat}
	exServerPackets[ExOpFishingHpRegen] = serverPacket{"ExFishingHpRegen", readExFishingHpRegen}
}

// ExFishingStart shows a cast: the float at X, Y, Z (FE 13).
type ExFishingStart struct {
	ObjectID   int32
	FishType   int32 // the fish group, -1 when it never bites
	X, Y, Z    int32
	Night      bool
	RankButton bool
}

// PacketName implements Packet.
func (ExFishingStart) PacketName() string { return "ExFishingStart" }

func readExFishingStart(f *fields) Packet {
	var p ExFishingStart
	p.ObjectID = f.i32()
	p.FishType = f.i32()
	p.X, p.Y, p.Z = f.i32(), f.i32(), f.i32()
	p.Night = f.flag8("night lure flag")
	p.RankButton = f.flag8("rank button flag")
	return p
}

// ExFishingEnd ends the fishing (FE 14).
type ExFishingEnd struct {
	ObjectID int32
	Win      bool
}

// PacketName implements Packet.
func (ExFishingEnd) PacketName() string { return "ExFishingEnd" }

func readExFishingEnd(f *fields) Packet {
	var p ExFishingEnd
	p.ObjectID = f.i32()
	p.Win = f.flag8("win flag")
	return p
}

// ExFishingStartCombat opens the fight when a fish bites (FE 15).
type ExFishingStartCombat struct {
	ObjectID  int32
	Time      int32
	HP        int32
	Mode      uint8
	LureType  uint8
	Deceptive uint8
}

// PacketName implements Packet.
func (ExFishingStartCombat) PacketName() string { return "ExFishingStartCombat" }

func readExFishingStartCombat(f *fields) Packet {
	var p ExFishingStartCombat
	p.ObjectID = f.i32()
	p.Time = f.i32()
	p.HP = f.i32()
	p.Mode, p.LureType, p.Deceptive = f.u8(), f.u8(), f.u8()
	return p
}

// ExFishingHpRegen updates the fight (FE 16).
type ExFishingHpRegen struct {
	ObjectID int32
	Time     int32
	HP       int32
	Mode     uint8
	GoodUse  uint8
	Anim     uint8
	Penalty  int32
	BarColor uint8
}

// PacketName implements Packet.
func (ExFishingHpRegen) PacketName() string { return "ExFishingHpRegen" }

func readExFishingHpRegen(f *fields) Packet {
	var p ExFishingHpRegen
	p.ObjectID = f.i32()
	p.Time = f.i32()
	p.HP = f.i32()
	p.Mode, p.GoodUse, p.Anim = f.u8(), f.u8(), f.u8()
	p.Penalty = f.i32()
	p.BarColor = f.u8()
	return p
}

// RequestExFishRanking asks for the running fishing championship's ranking
// (D0 1F); a fisherman's page comes back.
func (c *GameConn) RequestExFishRanking(ctx context.Context) error {
	return c.send(ctx, encodeEx(ExOpRequestExFishRanking))
}
