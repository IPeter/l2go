package serverpackets

import (
	"time"

	"github.com/IPeter/l2go/internal/packets"
)

// Opcodes of the siege window packets (Mobius ServerPackets).
const (
	OpSiegeInfo         = 0xc9
	OpSiegeAttackerList = 0xca
	OpSiegeDefenderList = 0xcb
)

// The type of a clan on the defender list (Mobius SiegeDefenderList: the
// SiegeClanType cases). Mobius's comments on 2 and 3 are swapped; the
// values are these.
const (
	SiegeDefenderTypeOther    = 0 // a type the defender list does not know (Mobius default)
	SiegeDefenderTypeOwner    = 1 // the castle owner
	SiegeDefenderTypeWaiting  = 2 // registered, waiting for the owner's approval
	SiegeDefenderTypeApproved = 3 // an approved defender
)

// SiegeInfo is the siege window of a castle (or of a siegable clan hall:
// the same layout with ChooseTime always false).
type SiegeInfo struct {
	// ResidenceID is the castle ID (the clan hall ID for a hall).
	ResidenceID int32
	// OwnerLeader is set when the viewer leads the owner clan: it shows the
	// owner controls (Mobius: ownerId == player.getClanId() &&
	// player.isClanLeader()).
	OwnerLeader bool
	// OwnerClanID is 0 when the residence has no owner. Then Mobius writes
	// empty names and ally ID 0, whatever the owner fields below hold.
	OwnerClanID     int32
	OwnerClanName   string
	OwnerLeaderName string
	OwnerAllyID     int32
	OwnerAllyName   string
	// Now is the current time in Unix seconds.
	Now int32
	// SiegeDate is the siege start in Unix seconds, sent when ChooseTime is
	// false.
	SiegeDate int32
	// ChooseTime lets the viewer pick the siege hour: Mobius sets it for
	// the owner clan's leader while the registration is not over. Then the
	// packet carries TimeChoices instead of SiegeDate.
	ChooseTime bool
	// TimeChoices are the selectable siege start times in Unix seconds
	// (see SiegeHourChoices).
	TimeChoices []int32
}

// NewSiegeInfoPacket builds SiegeInfo (Mobius SiegeInfo.writeImpl, castle
// branch):
//
//	c 0xc9, d residence, d owner leader, d owner clan,
//	S clan name, S leader name, d ally ID, S ally name (empty without owner),
//	d now, then d siege date, d 0; or, choosing the time: d 0, d n, n times d time.
func NewSiegeInfoPacket(s SiegeInfo) []byte {
	clanName, leaderName, allyID, allyName := s.OwnerClanName, s.OwnerLeaderName, s.OwnerAllyID, s.OwnerAllyName
	if s.OwnerClanID <= 0 {
		clanName, leaderName, allyID, allyName = "", "", 0, ""
	}
	w := packets.NewWriter(37 + 2*(len(clanName)+len(leaderName)+len(allyName)) + 4*len(s.TimeChoices))
	w.WriteUint8(OpSiegeInfo)
	w.WriteInt32(s.ResidenceID)
	putBool32(w, s.OwnerLeader)
	w.WriteInt32(s.OwnerClanID)
	w.WriteString(clanName)
	w.WriteString(leaderName)
	w.WriteInt32(allyID)
	w.WriteString(allyName)
	w.WriteInt32(s.Now)
	if s.ChooseTime {
		w.WriteInt32(0)
		putCount32(w, len(s.TimeChoices))
		for _, t := range s.TimeChoices {
			w.WriteInt32(t)
		}
	} else {
		w.WriteInt32(s.SiegeDate)
		w.WriteInt32(0)
	}
	return w.Bytes()
}

// SiegeHourChoices returns the siege start times the owner may choose from,
// in Unix seconds, as Mobius SiegeInfo computes them: the siege date's day
// in its location, at each of hours (FeatureConfig.SIEGE_HOUR_LIST) on the
// full hour.
func SiegeHourChoices(siegeDate time.Time, hours []int) []int32 {
	y, m, d := siegeDate.Date()
	out := make([]int32, 0, len(hours))
	for _, h := range hours {
		out = append(out, int32(time.Date(y, m, d, h, 0, 0, 0, siegeDate.Location()).Unix())) //nolint:gosec // Java (int) cast of seconds
	}
	return out
}

// SiegeClan is one clan line of SiegeAttackerList and SiegeDefenderList.
type SiegeClan struct {
	ClanID     int32
	ClanName   string
	LeaderName string
	CrestID    int32
	// Type is one of the SiegeDefenderType values; only SiegeDefenderList
	// writes it.
	Type        int32
	AllyID      int32
	AllyName    string
	AllyCrestID int32
}

// putSiegeClan writes one clan line: d clan, S name, S leader, d crest,
// d signed time (always 0 in Mobius), [d type], d ally, S ally name,
// S ally leader name (always empty in Mobius), d ally crest.
func putSiegeClan(w *packets.Writer, c SiegeClan, withType bool) {
	w.WriteInt32(c.ClanID)
	w.WriteString(c.ClanName)
	w.WriteString(c.LeaderName)
	w.WriteInt32(c.CrestID)
	w.WriteInt32(0) // signed time: not stored by Mobius
	if withType {
		w.WriteInt32(c.Type)
	}
	w.WriteInt32(c.AllyID)
	w.WriteString(c.AllyName)
	w.WriteString("") // ally leader name
	w.WriteInt32(c.AllyCrestID)
}

// siegeClanSize estimates the bytes of the clan lines.
func siegeClanSize(clans []SiegeClan) int {
	n := 0
	for _, c := range clans {
		n += 34 + 2*(len(c.ClanName)+len(c.LeaderName)+len(c.AllyName))
	}
	return n
}

// putSiegeListHead writes the head the two lists share: d residence, d 0,
// d 1, d 0, then the clan count twice.
func putSiegeListHead(w *packets.Writer, op uint8, residenceID int32, n int) {
	w.WriteUint8(op)
	w.WriteInt32(residenceID)
	w.WriteInt32(0)
	w.WriteInt32(1)
	w.WriteInt32(0)
	putCount32(w, n)
	putCount32(w, n)
}

// SiegeAttackerList is the attacker tab of the siege window.
type SiegeAttackerList struct {
	// ResidenceID is the castle ID (the clan hall ID for a hall).
	ResidenceID int32
	// Clans are the registered attackers. Mobius skips a clan it cannot
	// find after counting it; pass existing clans only.
	Clans []SiegeClan
}

// NewSiegeAttackerListPacket builds SiegeAttackerList (Mobius
// SiegeAttackerList.writeImpl): the list head, then the clan lines without
// the type.
func NewSiegeAttackerListPacket(l SiegeAttackerList) []byte {
	w := packets.NewWriter(25 + siegeClanSize(l.Clans))
	putSiegeListHead(w, OpSiegeAttackerList, l.ResidenceID, len(l.Clans))
	for _, c := range l.Clans {
		putSiegeClan(w, c, false)
	}
	return w.Bytes()
}

// SiegeDefenderList is the defender tab of a castle's siege window.
type SiegeDefenderList struct {
	CastleID int32
	// Clans are, in Mobius's order, the defender clans (the owner with
	// SiegeDefenderTypeOwner, the approved ones with
	// SiegeDefenderTypeApproved), then the clans waiting for approval with
	// SiegeDefenderTypeWaiting.
	Clans []SiegeClan
}

// NewSiegeDefenderListPacket builds SiegeDefenderList (Mobius
// SiegeDefenderList.writeImpl): the list head, then the clan lines with
// the type.
func NewSiegeDefenderListPacket(l SiegeDefenderList) []byte {
	w := packets.NewWriter(25 + siegeClanSize(l.Clans) + 4*len(l.Clans))
	putSiegeListHead(w, OpSiegeDefenderList, l.CastleID, len(l.Clans))
	for _, c := range l.Clans {
		putSiegeClan(w, c, true)
	}
	return w.Bytes()
}
