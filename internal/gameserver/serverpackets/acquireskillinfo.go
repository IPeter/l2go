package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpAcquireSkillInfo is the opcode of AcquireSkillInfo.
const OpAcquireSkillInfo = 0x8b

// The values Mobius sends around every required item of AcquireSkillInfo
// (its Req type and unk).
const (
	acquireSkillItemType    = 99
	acquireSkillItemTrailer = 50
)

// AcquireSkillItem is an item that learning a skill takes, such as a
// spellbook.
type AcquireSkillItem struct {
	ItemID int32
	Count  int64
}

// AcquireSkillInfo is what learning a skill costs, shown when the player
// picks it in the trainer's list (the answer to RequestAcquireSkillInfo).
type AcquireSkillInfo struct {
	SkillID int32
	Level   int32
	SPCost  int32
	Type    AcquireSkillType
	Items   []AcquireSkillItem
}

// NewAcquireSkillInfoPacket builds AcquireSkillInfo (Mobius
// AcquireSkillInfo.writeImpl). Every item goes out as the type 99, the item
// ID, the count as a long, and 50. An empty list is valid.
func NewAcquireSkillInfoPacket(a AcquireSkillInfo) []byte {
	w := packets.NewWriter(21 + 20*len(a.Items))
	w.WriteUint8(OpAcquireSkillInfo)
	w.WriteInt32(a.SkillID)
	w.WriteInt32(a.Level)
	w.WriteInt32(a.SPCost)
	w.WriteInt32(int32(a.Type))
	putCount32(w, len(a.Items))
	for _, it := range a.Items {
		w.WriteInt32(acquireSkillItemType)
		w.WriteInt32(it.ItemID)
		putInt64(w, it.Count)
		w.WriteInt32(acquireSkillItemTrailer)
	}
	return w.Bytes()
}
