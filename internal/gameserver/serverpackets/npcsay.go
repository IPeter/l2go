package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpNpcSay is the opcode of NpcSay.
const OpNpcSay = 0x02

// NpcSay is a line an NPC says, shown in the chat window and over the
// NPC's head. The Interlude packet always carries the text: there are no
// NPC string IDs in this chronicle.
type NpcSay struct {
	ObjectID int32
	// Type is the chat channel. Mobius NPC speech uses ChatGeneral (its
	// NPC_GENERAL) and ChatShout (NPC_SHOUT).
	Type ChatType
	// NpcID is the template's display ID; the client shows the NPC's name
	// from it. The builder adds the 1,000,000 offset the client expects.
	NpcID int32
	Text  string
}

// NewNpcSayPacket builds NpcSay (Mobius NpcSay.writeImpl).
func NewNpcSayPacket(s NpcSay) []byte {
	w := packets.NewWriter(15 + 2*len(s.Text))
	w.WriteUint8(OpNpcSay)
	w.WriteInt32(s.ObjectID)
	w.WriteInt32(int32(s.Type))
	w.WriteInt32(s.NpcID + clientNpcIDBase)
	w.WriteString(s.Text)
	return w.Bytes()
}
