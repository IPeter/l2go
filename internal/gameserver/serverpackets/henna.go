package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The symbol maker's windows (M11c, Mobius ServerPackets HENNA_*). HennaInfo
// is in hennainfo.go.
const (
	OpHennaEquipList      = 0xe2
	OpHennaItemInfo       = 0xe3 // Mobius HennaItemDrawInfo
	OpHennaRemoveList     = 0xe5
	OpHennaItemRemoveInfo = 0xe6
)

// HennaEntry is one dye of HennaEquipList or HennaRemoveList: its symbol and
// item, the dye count and the fee (to draw, or to get back and pay on
// removal).
type HennaEntry struct {
	DyeID     int32
	DyeItemID int32
	Count     int32
	Fee       int32
}

// HennaEquipList is the "draw a symbol" window.
type HennaEquipList struct {
	Adena int32
	// Listed is the number of dyes the class may wear, which Mobius writes
	// as the list's size although it sends only those whose dye item the
	// player carries (Entries); kept for parity.
	Listed int32
	// Entries are the dyes the class may wear and whose item the player
	// carries, by dye ID; each is sent as allowed.
	Entries []HennaEntry
}

// NewHennaEquipListPacket builds HennaEquipList (Mobius
// HennaEquipList.writeImpl): d adena, d 3 (slots), d count, then per dye d
// symbol, d item, d count, d fee, d 1 (allowed).
func NewHennaEquipListPacket(l HennaEquipList) []byte {
	w := packets.NewWriter(13 + 20*len(l.Entries))
	w.WriteUint8(OpHennaEquipList)
	w.WriteInt32(l.Adena)
	w.WriteInt32(hennaSlots)
	w.WriteInt32(l.Listed)
	for _, e := range l.Entries {
		putHennaEntry(w, e)
	}
	return w.Bytes()
}

// HennaRemoveList is the "delete a symbol" window.
type HennaRemoveList struct {
	Adena int32
	// EmptySlots are the player's empty slots (Mobius getHennaEmptySlots);
	// Mobius then writes |EmptySlots - 3| where a count would be, which is
	// not the number of Entries unless the class has three slots.
	EmptySlots int32
	// Entries are the worn dyes in slot order, with the cancel count and
	// fee.
	Entries []HennaEntry
}

// NewHennaRemoveListPacket builds HennaRemoveList (Mobius
// HennaRemoveList.writeImpl): d adena, d empty slots, d |empty slots - 3|,
// then per worn dye d symbol, d item, d cancel count, d cancel fee, d 1.
func NewHennaRemoveListPacket(l HennaRemoveList) []byte {
	w := packets.NewWriter(13 + 20*len(l.Entries))
	w.WriteUint8(OpHennaRemoveList)
	w.WriteInt32(l.Adena)
	w.WriteInt32(l.EmptySlots)
	used := l.EmptySlots - hennaSlots
	if used < 0 {
		used = -used
	}
	w.WriteInt32(used)
	for _, e := range l.Entries {
		putHennaEntry(w, e)
	}
	return w.Bytes()
}

func putHennaEntry(w *packets.Writer, e HennaEntry) {
	w.WriteInt32(e.DyeID)
	w.WriteInt32(e.DyeItemID)
	w.WriteInt32(e.Count)
	w.WriteInt32(e.Fee)
	w.WriteInt32(1)
}

// HennaStat is one base stat of HennaItemInfo and HennaItemRemoveInfo: the
// player's value now and with the dye drawn (or removed). Mobius writes the
// second with writeByte, so only its low 8 bits are sent.
type HennaStat struct {
	Now, After int32
}

// HennaItemInfo is the details of one dye: HennaItemDrawInfo (0xE3, what
// drawing it costs and does) and HennaItemRemoveInfo (0xE6, what removing it
// gives back and does).
type HennaItemInfo struct {
	DyeID     int32
	DyeItemID int32
	// Count and Fee are the wear count and fee (draw), or the cancel count
	// and fee (remove).
	Count   int32
	Fee     int32
	Allowed bool // the class may wear the dye
	Adena   int32
	// The stats in Mobius's order: INT, STR, CON, MEN, DEX, WIT.
	INT, STR, CON, MEN, DEX, WIT HennaStat
}

// NewHennaItemDrawInfoPacket builds HennaItemDrawInfo (Mobius
// HennaItemDrawInfo.writeImpl, opcode 0xE3).
func NewHennaItemDrawInfoPacket(i HennaItemInfo) []byte {
	return hennaItemInfo(OpHennaItemInfo, i)
}

// NewHennaItemRemoveInfoPacket builds HennaItemRemoveInfo (Mobius
// HennaItemRemoveInfo.writeImpl, opcode 0xE6).
func NewHennaItemRemoveInfoPacket(i HennaItemInfo) []byte {
	return hennaItemInfo(OpHennaItemRemoveInfo, i)
}

// hennaItemInfo writes d symbol, d item, d count, d fee, d allowed, d adena,
// then for INT, STR, CON, MEN, DEX and WIT d now and c after.
func hennaItemInfo(op uint8, i HennaItemInfo) []byte {
	w := packets.NewWriter(55)
	w.WriteUint8(op)
	w.WriteInt32(i.DyeID)
	w.WriteInt32(i.DyeItemID)
	w.WriteInt32(i.Count)
	w.WriteInt32(i.Fee)
	putBool32(w, i.Allowed)
	w.WriteInt32(i.Adena)
	for _, st := range []HennaStat{i.INT, i.STR, i.CON, i.MEN, i.DEX, i.WIT} {
		w.WriteInt32(st.Now)
		w.WriteUint8(uint8(st.After)) //nolint:gosec // writeByte keeps the low 8 bits
	}
	return w.Bytes()
}
