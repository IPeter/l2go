package testclient

import (
	"reflect"
	"testing"
)

// The siege window packets decode from the Mobius writeImpl layouts.
func TestDecodeSiegePackets(t *testing.T) {
	const lions, hawks = 0x10000001, 0x10000002
	for _, tt := range []struct {
		name string
		raw  []byte
		want Packet
	}{
		// SiegeInfo without an owner: d castle, d 0, d 0, S "", S "", d 0,
		// S "", d now, d date, d 0.
		{"SiegeInfo", msg(0xc9).i32(1).i32(0).i32(0).str("").str("").i32(0).str("").
			i32(1700000000).i32(1700600000).i32(0).b,
			SiegeInfo{CastleID: 1, Now: 1700000000, SiegeDate: 1700600000}},
		// The owner's leader choosing the hour: d 0, d n, the times.
		{"SiegeInfo", msg(0xc9).i32(5).i32(1).i32(lions).str("Lions").str("Leo").i32(lions).str("Pride").
			i32(1700000000).i32(0).i32(2).i32(1700564400).i32(1700589600).b,
			SiegeInfo{
				CastleID: 5, OwnerLeader: true, OwnerClanID: lions, OwnerClanName: "Lions", OwnerLeaderName: "Leo",
				OwnerAllyID: lions, OwnerAllyName: "Pride", Now: 1700000000, TimeChoices: []int32{1700564400, 1700589600},
			}},
		// SiegeAttackerList: d castle, d 0, d 1, d 0, d n, d n, clans
		// without the type.
		{"SiegeAttackerList", msg(0xca).i32(3).i32(0).i32(1).i32(0).i32(1).i32(1).
			i32(hawks).str("Hawks").str("Kite").i32(0x500).i32(0).i32(lions).str("Pride").str("").i32(0x600).b,
			SiegeAttackerList{CastleID: 3, Head: [3]int32{0, 1, 0}, Clans: []SiegeClan{
				{ClanID: hawks, ClanName: "Hawks", LeaderName: "Kite", CrestID: 0x500, AllyID: lions, AllyName: "Pride", AllyCrestID: 0x600},
			}}},
		// SiegeDefenderList: the same with d type after the signed time.
		{"SiegeDefenderList", msg(0xcb).i32(3).i32(0).i32(1).i32(0).i32(2).i32(2).
			i32(lions).str("Lions").str("Leo").i32(0).i32(0).i32(1).i32(0).str("").str("").i32(0).
			i32(hawks).str("Hawks").str("Kite").i32(0).i32(0).i32(2).i32(0).str("").str("").i32(0).b,
			SiegeDefenderList{CastleID: 3, Head: [3]int32{0, 1, 0}, Clans: []SiegeClan{
				{ClanID: lions, ClanName: "Lions", LeaderName: "Leo", Type: SiegeDefenderOwner},
				{ClanID: hawks, ClanName: "Hawks", LeaderName: "Kite", Type: SiegeDefenderWaiting},
			}}},
		{"SiegeDefenderList", msg(0xcb).i32(3).i32(0).i32(1).i32(0).i32(0).i32(0).b,
			SiegeDefenderList{CastleID: 3, Head: [3]int32{0, 1, 0}}},
	} {
		got, err := DecodeServerPacket(tt.raw)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
		if got.PacketName() != tt.name {
			t.Errorf("%s: named %q", tt.name, got.PacketName())
		}
		if _, err := DecodeServerPacket(tt.raw[:len(tt.raw)-1]); err == nil {
			t.Errorf("%s: a short packet decoded", tt.name)
		}
	}

	// Layouts that are not Mobius's: an owner flag of 2, a date with time
	// choices, two different clan counts.
	for _, raw := range [][]byte{
		msg(0xc9).i32(1).i32(2).i32(0).str("").str("").i32(0).str("").i32(1).i32(2).i32(0).b,
		msg(0xc9).i32(1).i32(1).i32(0).str("").str("").i32(0).str("").i32(1).i32(2).i32(1).i32(3).b,
		msg(0xca).i32(3).i32(0).i32(1).i32(0).i32(1).i32(0).b,
	} {
		if _, err := DecodeServerPacket(raw); err == nil {
			t.Errorf("malformed siege packet % x decoded", raw)
		}
	}
}

// Clan finds a line by clan name.
func TestSiegeListClan(t *testing.T) {
	l := SiegeDefenderList{Clans: []SiegeClan{{ClanID: 1, ClanName: "Lions"}, {ClanID: 2, ClanName: "Hawks"}}}
	if c, ok := l.Clan("Hawks"); !ok || c.ClanID != 2 {
		t.Errorf("Clan(Hawks) = %+v, %v", c, ok)
	}
	if _, ok := (SiegeAttackerList{}).Clan("Hawks"); ok {
		t.Error("Clan found in an empty list")
	}
}
