package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpHennaInfo is the opcode of HennaInfo.
const OpHennaInfo = 0xe4

// hennaSlots is the number of dye slots Mobius sends.
const hennaSlots = 3

// HennaInfo is the player's dyes and the stat changes they give. The stat
// changes can be negative.
type HennaInfo struct {
	INT    int8
	STR    int8
	CON    int8
	MEN    int8
	DEX    int8
	WIT    int8
	DyeIDs []int32 // the dyes worn
}

// NewHennaInfoPacket builds HennaInfo (Mobius HennaInfo.writeImpl). No dyes
// is valid.
func NewHennaInfoPacket(h HennaInfo) []byte {
	w := packets.NewWriter(15 + len(h.DyeIDs)*8)
	w.WriteUint8(OpHennaInfo)
	putInt8(w, h.INT)
	putInt8(w, h.STR)
	putInt8(w, h.CON)
	putInt8(w, h.MEN)
	putInt8(w, h.DEX)
	putInt8(w, h.WIT)
	w.WriteInt32(hennaSlots)
	putCount32(w, len(h.DyeIDs))
	for _, id := range h.DyeIDs {
		w.WriteInt32(id)
		w.WriteInt32(1)
	}
	return w.Bytes()
}
