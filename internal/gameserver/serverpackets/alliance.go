package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// Opcodes of the alliance packets (Mobius ServerPackets). AllyCrest is in
// crest.go.
const (
	OpAskJoinAlly  = 0xa8
	OpAllianceInfo = 0xb4
)

// NewAskJoinAllyPacket builds AskJoinAlly, an invitation of a clan leader
// into an alliance (Mobius AskJoinAlly.writeImpl).
func NewAskJoinAllyPacket(requestorID int32, requestorName string) []byte {
	w := packets.NewWriter(7 + 2*len(requestorName))
	w.WriteUint8(OpAskJoinAlly)
	w.WriteInt32(requestorID)
	w.WriteString(requestorName)
	return w.Bytes()
}

// AllianceClan is one member clan of AllianceInfo.
type AllianceClan struct {
	Name       string
	Level      int32
	LeaderName string
	Members    int32 // all members (Mobius ClanInfo.getTotal)
	Online     int32 // members online
}

// AllianceInfo is the alliance window, the answer to RequestAllyInfo.
type AllianceInfo struct {
	Name       string
	LeaderClan string // the name of the alliance leader's clan
	LeaderName string // the name of that clan's leader
	Clans      []AllianceClan
}

// NewAllianceInfoPacket builds AllianceInfo (Mobius AllianceInfo.writeImpl).
// The alliance totals are the sums over the clans, as Mobius counts them.
func NewAllianceInfoPacket(a AllianceInfo) []byte {
	var total, online int32
	for _, c := range a.Clans {
		total += c.Members
		online += c.Online
	}
	w := packets.NewWriter(32 + 2*(len(a.Name)+len(a.LeaderClan)+len(a.LeaderName)) + len(a.Clans)*64)
	w.WriteUint8(OpAllianceInfo)
	w.WriteString(a.Name)
	w.WriteInt32(total)
	w.WriteInt32(online)
	w.WriteString(a.LeaderClan)
	w.WriteString(a.LeaderName)
	putCount32(w, len(a.Clans))
	for _, c := range a.Clans {
		w.WriteString(c.Name)
		w.WriteInt32(0)
		w.WriteInt32(c.Level)
		w.WriteString(c.LeaderName)
		w.WriteInt32(c.Members)
		w.WriteInt32(c.Online)
	}
	return w.Bytes()
}
