package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpListPartyWating is the opcode of ListPartyWating (Mobius
// PARTY_MATCH_LIST).
const OpListPartyWating = 0x96

// PartyMatchRoom is one party matching room of ListPartyWating.
type PartyMatchRoom struct {
	ID         int32
	Title      string
	Location   int32
	MinLevel   int32
	MaxLevel   int32
	MaxMembers int32
	Members    int32
	Owner      string // the owner's name
}

// NewListPartyWatingPacket builds ListPartyWating (Mobius
// ListPartyWating.writeImpl), the party matching room list: d 1 when there
// are rooms else 0, d count, then per room d ID, S title, d location,
// d min level, d max level, d max members, d members, S owner. The rooms
// are already filtered (Mobius filters them by location and level in
// writeImpl). M9 has no rooms (Plan 12 T10) and sends an empty list.
func NewListPartyWatingPacket(rooms []PartyMatchRoom) []byte {
	w := packets.NewWriter(9 + 40*len(rooms))
	w.WriteUint8(OpListPartyWating)
	putBool32(w, len(rooms) > 0)
	putCount32(w, len(rooms))
	for _, r := range rooms {
		w.WriteInt32(r.ID)
		w.WriteString(r.Title)
		w.WriteInt32(r.Location)
		w.WriteInt32(r.MinLevel)
		w.WriteInt32(r.MaxLevel)
		w.WriteInt32(r.MaxMembers)
		w.WriteInt32(r.Members)
		w.WriteString(r.Owner)
	}
	return w.Bytes()
}
