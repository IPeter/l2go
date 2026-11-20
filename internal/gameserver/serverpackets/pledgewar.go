package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// Opcodes of the clan war packets (Mobius ServerPackets).
const (
	OpStartPledgeWar     = 0x65
	OpStopPledgeWar      = 0x67
	OpSurrenderPledgeWar = 0x69
)

// The tabs of PledgeReceiveWarList and RequestPledgeWarList.
const (
	WarListDeclared    = 0 // the clans this clan has declared war on
	WarListUnderAttack = 1 // the clans that have declared war on this clan
)

// NewStartPledgeWarPacket builds StartPledgeWar, which asks a clan leader
// to answer a war declaration (Mobius StartPledgeWar.writeImpl): the
// player's name comes first, then the clan's.
func NewStartPledgeWarPacket(clanName, playerName string) []byte {
	return pledgeWarPacket(OpStartPledgeWar, playerName, clanName)
}

// NewStopPledgeWarPacket builds StopPledgeWar (Mobius
// StopPledgeWar.writeImpl): the clan's name, then the player's.
func NewStopPledgeWarPacket(clanName, playerName string) []byte {
	return pledgeWarPacket(OpStopPledgeWar, clanName, playerName)
}

// NewSurrenderPledgeWarPacket builds SurrenderPledgeWar (Mobius
// SurrenderPledgeWar.writeImpl): the clan's name, then the player's.
func NewSurrenderPledgeWarPacket(clanName, playerName string) []byte {
	return pledgeWarPacket(OpSurrenderPledgeWar, clanName, playerName)
}

func pledgeWarPacket(op uint8, first, second string) []byte {
	w := packets.NewWriter(5 + 2*len(first) + 2*len(second))
	w.WriteUint8(op)
	w.WriteString(first)
	w.WriteString(second)
	return w.Bytes()
}

// NewPledgeReceiveWarListPacket builds PledgeReceiveWarList, the clan war
// tab of the clan window (Mobius PledgeReceiveWarList.writeImpl). tab is
// WarListDeclared or WarListUnderAttack; clans are the names of the other
// clans.
//
// Mobius counts the clan IDs of the war list but skips the clans it cannot
// find, so its count can be larger than the entries. The game layer passes
// the names it found, and the count matches them.
func NewPledgeReceiveWarListPacket(tab int32, clans []string) []byte {
	w := newExWriter(ExOpPledgeReceiveWarList, 12+len(clans)*40)
	w.WriteInt32(tab)
	w.WriteInt32(0) // page
	putCount32(w, len(clans))
	for _, name := range clans {
		w.WriteString(name)
		w.WriteInt32(tab)
		w.WriteInt32(tab)
	}
	return w.Bytes()
}
