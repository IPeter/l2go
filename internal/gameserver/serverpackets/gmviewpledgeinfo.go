package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpGMViewPledgeInfo is the opcode of GMViewPledgeInfo.
const OpGMViewPledgeInfo = 0x90

// NewGMViewPledgeInfoPacket builds GMViewPledgeInfo, a clan as //pledge info
// shows it to a GM (Mobius GMViewPledgeInfo.writeImpl): the player the GM
// looked at, the clan's state as in PledgeShowMemberListAll, then every
// member of every unit, offline ones with object ID 0.
func NewGMViewPledgeInfoPacket(playerName string, c ClanInfo, members []PledgeMember) []byte {
	w := packets.NewWriter(96 + 2*len(playerName) + 2*len(c.Name) + 2*len(c.LeaderName) + 2*len(c.AllyName) + len(members)*48)
	w.WriteUint8(OpGMViewPledgeInfo)
	w.WriteString(playerName)
	w.WriteInt32(c.ID)
	w.WriteInt32(0)
	w.WriteString(c.Name)
	w.WriteString(c.LeaderName)
	putClanState(w, c)
	putCount32(w, len(members))
	for _, m := range members {
		w.WriteString(m.Name)
		w.WriteInt32(m.Level)
		w.WriteInt32(m.ClassID)
		putBool32(w, m.Female)
		w.WriteInt32(m.Race)
		if m.Online {
			w.WriteInt32(m.ObjectID)
		} else {
			w.WriteInt32(0)
		}
		putBool32(w, m.HasSponsor)
	}
	return w.Bytes()
}
