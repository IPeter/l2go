package serverpackets

// The cursed weapon packets (Plan 14 M11m): the list of cursed weapons and
// where they are, the answers to RequestCursedWeaponList (D0:22) and
// RequestCursedWeaponLocation (D0:23), and the red sky of a monster's drop.

// Sub-opcodes of the cursed weapon packets (Mobius ServerPackets).
const (
	ExOpRedSky               = 0x40
	ExOpCursedWeaponList     = 0x45
	ExOpCursedWeaponLocation = 0x46
)

// NewExCursedWeaponListPacket builds ExCursedWeaponList (Mobius
// ExCursedWeaponList.writeImpl): d count, then d item ID per weapon.
func NewExCursedWeaponListPacket(itemIDs []int32) []byte {
	w := newExWriter(ExOpCursedWeaponList, 4+4*len(itemIDs))
	putCount32(w, len(itemIDs))
	for _, id := range itemIDs {
		w.WriteInt32(id)
	}
	return w.Bytes()
}

// CursedWeaponLocation is one weapon of ExCursedWeaponLocation.
type CursedWeaponLocation struct {
	ItemID int32
	// Activated is set when a player holds the weapon, clear when it lies
	// on the ground.
	Activated bool
	X, Y, Z   int32
}

// NewExCursedWeaponLocationPacket builds ExCursedWeaponLocation (Mobius
// ExCursedWeaponLocation.writeImpl): d count, then d item ID, d activated,
// d x, y, z per weapon. An empty list is written as two zero ints (Mobius
// never sends one: RequestCursedWeaponLocation sends nothing then).
func NewExCursedWeaponLocationPacket(list []CursedWeaponLocation) []byte {
	w := newExWriter(ExOpCursedWeaponLocation, 4+20*len(list))
	if len(list) == 0 {
		w.WriteInt32(0)
		w.WriteInt32(0)
		return w.Bytes()
	}
	putCount32(w, len(list))
	for _, l := range list {
		w.WriteInt32(l.ItemID)
		if l.Activated {
			w.WriteInt32(1)
		} else {
			w.WriteInt32(0)
		}
		w.WriteInt32(l.X)
		w.WriteInt32(l.Y)
		w.WriteInt32(l.Z)
	}
	return w.Bytes()
}

// NewExRedSkyPacket builds ExRedSky (Mobius ExRedSky.writeImpl): the sky
// turns red for seconds.
func NewExRedSkyPacket(seconds int32) []byte {
	w := newExWriter(ExOpRedSky, 4)
	w.WriteInt32(seconds)
	return w.Bytes()
}
