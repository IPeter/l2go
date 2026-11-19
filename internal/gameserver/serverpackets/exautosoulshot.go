package serverpackets

// ExOpAutoSoulShot is the sub-opcode of ExAutoSoulShot.
const ExOpAutoSoulShot = 0x12

// ExAutoSoulShot turns the automatic use of a soulshot or spiritshot on or
// off; the client lights up or dims the item on the shortcut bar. Mobius
// sends it for RequestAutoSoulShot, when it turns a shot off by itself, and
// for every shot still on after it rebuilds the bar with ShortcutInit.
type ExAutoSoulShot struct {
	ItemID  int32
	Enabled bool
}

// NewExAutoSoulShotPacket builds ExAutoSoulShot (Mobius
// ExAutoSoulShot.writeImpl): the item ID, then 1 for on or 0 for off.
func NewExAutoSoulShotPacket(e ExAutoSoulShot) []byte {
	w := newExWriter(ExOpAutoSoulShot, 8)
	w.WriteInt32(e.ItemID)
	putBool32(w, e.Enabled)
	return w.Bytes()
}
