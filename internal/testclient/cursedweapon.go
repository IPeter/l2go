package testclient

import "context"

// The cursed weapon packets (Plan 14 M11m) in the Mobius Interlude layouts.

// Ex sub-opcodes of the cursed weapon packets.
const (
	// Client: RequestCursedWeaponList and RequestCursedWeaponLocation.
	ExOpRequestCursedWeaponList     uint16 = 0x22
	ExOpRequestCursedWeaponLocation uint16 = 0x23
	// Server: ExCursedWeaponList and ExCursedWeaponLocation.
	ExOpCursedWeaponList     uint16 = 0x45
	ExOpCursedWeaponLocation uint16 = 0x46
)

// The decoder entries and names are added here rather than in the shared
// tables, so the cursed weapon packets stay in one file.
func init() {
	exServerPackets[ExOpCursedWeaponList] = serverPacket{"ExCursedWeaponList", readExCursedWeaponList}
	exServerPackets[ExOpCursedWeaponLocation] = serverPacket{"ExCursedWeaponLocation", readExCursedWeaponLocation}
	exClientPacketNames[ExOpRequestCursedWeaponList] = "RequestCursedWeaponList"
	exClientPacketNames[ExOpRequestCursedWeaponLocation] = "RequestCursedWeaponLocation"
}

// RequestCursedWeaponList asks for the cursed weapons (D0:22);
// ExCursedWeaponList comes back.
func (c *GameConn) RequestCursedWeaponList(ctx context.Context) error {
	return c.send(ctx, encodeEx(ExOpRequestCursedWeaponList))
}

// RequestCursedWeaponLocation asks where the cursed weapons are (D0:23);
// ExCursedWeaponLocation comes back when one is in the world.
func (c *GameConn) RequestCursedWeaponLocation(ctx context.Context) error {
	return c.send(ctx, encodeEx(ExOpRequestCursedWeaponLocation))
}

// ExCursedWeaponList lists the cursed weapons' item IDs (FE:45).
type ExCursedWeaponList struct {
	ItemIDs []int32
}

// PacketName implements Packet.
func (ExCursedWeaponList) PacketName() string { return "ExCursedWeaponList" }

func readExCursedWeaponList(f *fields) Packet {
	var p ExCursedWeaponList
	n := f.count32("count")
	for i := 0; i < n && f.ok(); i++ {
		p.ItemIDs = append(p.ItemIDs, f.i32())
	}
	return p
}

// CursedWeaponLocation is one weapon of ExCursedWeaponLocation.
type CursedWeaponLocation struct {
	ItemID int32
	// Activated is set when a player holds the weapon.
	Activated bool
	X, Y, Z   int32
}

// ExCursedWeaponLocation tells where the cursed weapons are (FE:46).
type ExCursedWeaponLocation struct {
	Weapons []CursedWeaponLocation
}

// PacketName implements Packet.
func (ExCursedWeaponLocation) PacketName() string { return "ExCursedWeaponLocation" }

func readExCursedWeaponLocation(f *fields) Packet {
	var p ExCursedWeaponLocation
	n := f.count32("count")
	if n == 0 {
		// An empty list is written as two zero ints.
		if v := f.i32(); f.ok() && v != 0 {
			f.failf("empty list ends with %d, want 0", v)
		}
		return p
	}
	for i := 0; i < n && f.ok(); i++ {
		var w CursedWeaponLocation
		w.ItemID = f.i32()
		w.Activated = f.flag32("activated")
		w.X, w.Y, w.Z = f.i32(), f.i32(), f.i32()
		p.Weapons = append(p.Weapons, w)
	}
	return p
}
