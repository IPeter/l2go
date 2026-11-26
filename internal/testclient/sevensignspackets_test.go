package testclient

import (
	"reflect"
	"testing"
)

// SSQStatus decodes from the Mobius writeImpl layout of each page.
func TestDecodeSSQStatus(t *testing.T) {
	for _, tt := range []struct {
		name string
		raw  []byte
		want SSQStatus
	}{
		{"page 1", msg(0xf5).u8(1).u8(1).i32(7).i32(1176).i32(1286).u8(2).u8(3).i32(15).i32(120).
			i32(100).i32(60).i32(160).u8(31).i32(400).i32(0).i32(400).u8(71).b,
			SSQStatus{Page: 1, Period: 1, Cycle: 7, PeriodMsg: 1176, PeriodEndMsg: 1286, Cabal: 2, Seal: 3,
				StonesTurnedIn: 15, AncientAdenaToCollect: 120, DuskStoneScore: 100, DuskFestivalScore: 60, DuskTotal: 160,
				DuskPercent: 31, DawnStoneScore: 400, DawnTotal: 400, DawnPercent: 71}},
		{"page 2", msg(0xf5).u8(2).u8(3).u16(1).u8(2).
			u8(1).i32(60).i32(75).u8(2).str("Ann").str("Bo").i32(0).u8(1).str("").
			u8(5).i32(150).i32(0).u8(0).i32(300).u8(1).str("Zed").b,
			SSQStatus{Page: 2, Period: 3, Festivals: []SSQFestival{
				{ID: 1, LevelScore: 60, DuskScore: 75, DuskMembers: []string{"Ann", "Bo"}, DawnMembers: []string{""}},
				{ID: 5, LevelScore: 150, DuskMembers: []string{}, DawnScore: 300, DawnMembers: []string{"Zed"}},
			}}},
		{"page 3", msg(0xf5).u8(3).u8(0).u8(10).u8(35).u8(2).u8(1).u8(2).u8(0).u8(50).u8(3).u8(1).u8(33).u8(25).b,
			SSQStatus{Page: 3, KeepPercent: 10, ClaimPercent: 35, Seals: []SSQSealVote{
				{Seal: 1, Owner: 2, DawnPercent: 50}, {Seal: 3, Owner: 1, DuskPercent: 33, DawnPercent: 25},
			}}},
		{"page 4", msg(0xf5).u8(4).u8(1).u8(2).u8(2).u8(1).u8(2).u16(1290).u8(3).u8(0).u16(1292).b,
			SSQStatus{Page: 4, Period: 1, Winner: 2, Predictions: []SSQSealPrediction{
				{Seal: 1, Owner: 2, Msg: 1290}, {Seal: 3, Owner: 0, Msg: 1292},
			}}},
	} {
		got, err := DecodeServerPacket(tt.raw)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
		if got.PacketName() != "SSQStatus" {
			t.Errorf("%s: named %q", tt.name, got.PacketName())
		}
		if _, err := DecodeServerPacket(tt.raw[:len(tt.raw)-1]); err == nil {
			t.Errorf("%s: a short packet decoded", tt.name)
		}
	}
	if got, err := DecodeServerPacket(msg(0xf5).u8(0).u8(2).b); err != nil || !reflect.DeepEqual(got, SSQStatus{Period: 2}) {
		t.Errorf("page 0: %+v, %v", got, err)
	}
	if _, err := DecodeServerPacket(msg(0xf5).u8(2).u8(0).u16(2).u8(0).b); err == nil {
		t.Error("a festival page not starting with 1 decoded")
	}
	if clientPacketNames[OpRequestSSQStatus] != "RequestSSQStatus" {
		t.Errorf("0xC7 is named %q", clientPacketNames[OpRequestSSQStatus])
	}
}
