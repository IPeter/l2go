package serverpackets

import (
	"slices"
	"testing"
	"time"
)

// Mobius SiegeInfo.writeImpl (castle):
//
//	c 0xc9
//	d castle ID
//	d ownerId == player.clanId && player.isClanLeader
//	d ownerId
//	ownerId > 0 ? S clan name, S leader name, d ally ID, S ally name
//	            : S "", S "", d 0, S ""
//	d now (seconds)
//	!registrationOver && leader of the owner clan ?
//		d 0, d hours, d time per hour
//		: d siege date (seconds), d 0
func TestSiegeInfoPacket(t *testing.T) {
	// No owner: empty names whatever the owner fields hold.
	var g m3Golden
	g.c(0xc9).d(1).d(0).d(0)
	g.s("").s("").d(0).s("")
	g.d(1700000000).d(1700600000).d(0)
	m3Check(t, "no owner",
		NewSiegeInfoPacket(SiegeInfo{
			ResidenceID: 1, OwnerClanName: "Stale", OwnerAllyID: 9, OwnerAllyName: "Stale",
			Now: 1700000000, SiegeDate: 1700600000,
		}), g.b)

	// An owner seen by another player: the date, no controls.
	g = m3Golden{}
	g.c(0xc9).d(5).d(0).d(0x10000001)
	g.s("Lions").s("Leo").d(0x10000001).s("Pride")
	g.d(1700000000).d(1700600000).d(0)
	m3Check(t, "owner, other viewer",
		NewSiegeInfoPacket(SiegeInfo{
			ResidenceID: 5, OwnerClanID: 0x10000001, OwnerClanName: "Lions", OwnerLeaderName: "Leo",
			OwnerAllyID: 0x10000001, OwnerAllyName: "Pride",
			Now: 1700000000, SiegeDate: 1700600000,
		}), g.b)

	// The owner's leader after the registration: controls, the date.
	g = m3Golden{}
	g.c(0xc9).d(5).d(1).d(0x10000001)
	g.s("Lions").s("Leo").d(0).s("")
	g.d(1700000000).d(1700600000).d(0)
	m3Check(t, "owner leader, registration over",
		NewSiegeInfoPacket(SiegeInfo{
			ResidenceID: 5, OwnerLeader: true, OwnerClanID: 0x10000001, OwnerClanName: "Lions", OwnerLeaderName: "Leo",
			Now: 1700000000, SiegeDate: 1700600000,
		}), g.b)

	// The owner's leader choosing the hour: d 0, the count, the times; no
	// date.
	g = m3Golden{}
	g.c(0xc9).d(5).d(1).d(0x10000001)
	g.s("Lions").s("Leo").d(0).s("")
	g.d(1700000000).d(0).d(2).d(1700564400).d(1700589600)
	m3Check(t, "owner leader choosing",
		NewSiegeInfoPacket(SiegeInfo{
			ResidenceID: 5, OwnerLeader: true, OwnerClanID: 0x10000001, OwnerClanName: "Lions", OwnerLeaderName: "Leo",
			Now: 1700000000, SiegeDate: 1700600000,
			ChooseTime: true, TimeChoices: []int32{1700564400, 1700589600},
		}), g.b)

	// Choosing with no configured hours still writes d 0, d 0.
	g = m3Golden{}
	g.c(0xc9).d(5).d(1).d(7)
	g.s("A").s("B").d(0).s("")
	g.d(10).d(0).d(0)
	m3Check(t, "choosing, no hours",
		NewSiegeInfoPacket(SiegeInfo{
			ResidenceID: 5, OwnerLeader: true, OwnerClanID: 7, OwnerClanName: "A", OwnerLeaderName: "B",
			Now: 10, SiegeDate: 99, ChooseTime: true,
		}), g.b)
}

// Mobius sets MINUTE and SECOND to 0 and HOUR_OF_DAY to each hour, on the
// siege date's day in the server's time zone.
func TestSiegeHourChoices(t *testing.T) {
	loc := time.FixedZone("CET", 3600)
	date := time.Date(2026, 11, 28, 20, 37, 15, 500e6, loc)
	got := SiegeHourChoices(date, []int{16, 20})
	want := []int32{
		int32(time.Date(2026, 11, 28, 16, 0, 0, 0, loc).Unix()),
		int32(time.Date(2026, 11, 28, 20, 0, 0, 0, loc).Unix()),
	}
	if !slices.Equal(got, want) {
		t.Errorf("SiegeHourChoices = %v, want %v", got, want)
	}
	if want[1]-want[0] != 4*3600 {
		t.Errorf("choices %v are not 4 hours apart", want)
	}
	if got := SiegeHourChoices(date, nil); len(got) != 0 {
		t.Errorf("no hours: %v", got)
	}
}

// Mobius SiegeAttackerList.writeImpl:
//
//	c 0xca
//	d castle ID, d 0, d 1, d 0
//	size > 0 ? d size, d size, then per clan:
//		d clan ID, S name, S leader, d crest, d 0 (signed time),
//		d ally ID, S ally name, S "" (ally leader), d ally crest
//		: d 0, d 0
func TestSiegeAttackerListPacket(t *testing.T) {
	var g m3Golden
	g.c(0xca).d(3).d(0).d(1).d(0).d(0).d(0)
	m3Check(t, "empty", NewSiegeAttackerListPacket(SiegeAttackerList{ResidenceID: 3}), g.b)

	g = m3Golden{}
	g.c(0xca).d(3).d(0).d(1).d(0).d(2).d(2)
	g.d(0x10000010).s("Wolves").s("Fang").d(0x500).d(0).d(0x10000010).s("Pack").s("").d(0x600)
	g.d(0x10000020).s("Bears").s("Ursa").d(0).d(0).d(0).s("").s("").d(0)
	m3Check(t, "two attackers",
		NewSiegeAttackerListPacket(SiegeAttackerList{ResidenceID: 3, Clans: []SiegeClan{
			// The type is not written in the attacker list.
			{
				ClanID: 0x10000010, ClanName: "Wolves", LeaderName: "Fang", CrestID: 0x500, Type: SiegeDefenderTypeOwner,
				AllyID: 0x10000010, AllyName: "Pack", AllyCrestID: 0x600,
			},
			{ClanID: 0x10000020, ClanName: "Bears", LeaderName: "Ursa"},
		}}), g.b)
}

// Mobius SiegeDefenderList.writeImpl:
//
//	c 0xcb
//	d castle ID, d 0, d 1, d 0
//	size > 0 ? d size, d size, then per clan:
//		d clan ID, S name, S leader, d crest, d 0 (signed time),
//		d type (OWNER 1, DEFENDER_PENDING 2, DEFENDER 3, else 0),
//		d ally ID, S ally name, S "" (ally leader), d ally crest
//		: d 0, d 0
func TestSiegeDefenderListPacket(t *testing.T) {
	var g m3Golden
	g.c(0xcb).d(2).d(0).d(1).d(0).d(0).d(0)
	m3Check(t, "empty", NewSiegeDefenderListPacket(SiegeDefenderList{CastleID: 2}), g.b)

	g = m3Golden{}
	g.c(0xcb).d(2).d(0).d(1).d(0).d(3).d(3)
	g.d(0x10000001).s("Lions").s("Leo").d(0x700).d(0).d(1).d(0x10000001).s("Pride").s("").d(0x701)
	g.d(0x10000002).s("Hawks").s("Kite").d(0).d(0).d(3).d(0x10000001).s("Pride").s("").d(0x701)
	g.d(0x10000003).s("Owls").s("Hoot").d(0).d(0).d(2).d(0).s("").s("").d(0)
	m3Check(t, "owner, approved, waiting",
		NewSiegeDefenderListPacket(SiegeDefenderList{CastleID: 2, Clans: []SiegeClan{
			{
				ClanID: 0x10000001, ClanName: "Lions", LeaderName: "Leo", CrestID: 0x700,
				Type: SiegeDefenderTypeOwner, AllyID: 0x10000001, AllyName: "Pride", AllyCrestID: 0x701,
			},
			{
				ClanID: 0x10000002, ClanName: "Hawks", LeaderName: "Kite",
				Type: SiegeDefenderTypeApproved, AllyID: 0x10000001, AllyName: "Pride", AllyCrestID: 0x701,
			},
			{ClanID: 0x10000003, ClanName: "Owls", LeaderName: "Hoot", Type: SiegeDefenderTypeWaiting},
		}}), g.b)
}

// The opcodes are those of Mobius ServerPackets; the defender types are
// the values Mobius SiegeDefenderList writes.
func TestSiegeOpcodesAndTypes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"SiegeInfo", OpSiegeInfo, 0xc9},
		{"SiegeAttackerList", OpSiegeAttackerList, 0xca},
		{"SiegeDefenderList", OpSiegeDefenderList, 0xcb},
		{"other", SiegeDefenderTypeOther, 0},
		{"OWNER", SiegeDefenderTypeOwner, 1},
		{"DEFENDER_PENDING", SiegeDefenderTypeWaiting, 2},
		{"DEFENDER", SiegeDefenderTypeApproved, 3},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}
