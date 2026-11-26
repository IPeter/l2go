package testclient

import (
	"errors"
	"reflect"
	"testing"
)

// The cursed weapon packets decode from the Mobius writeImpl layouts.
func TestDecodeCursedWeaponPackets(t *testing.T) {
	for _, tt := range []struct {
		name string
		raw  []byte
		want Packet
	}{
		// ExCursedWeaponList: d count, d item ID per weapon.
		{"ExCursedWeaponList", msg(0xfe).u16(0x45).i32(2).i32(8689).i32(8190).b,
			ExCursedWeaponList{ItemIDs: []int32{8689, 8190}}},
		// ExCursedWeaponLocation: d count, then d item ID, d activated,
		// d x, y, z per weapon.
		{"ExCursedWeaponLocation", msg(0xfe).u16(0x46).i32(2).
			i32(8689).i32(0).i32(1).i32(-2).i32(3).
			i32(8190).i32(1).i32(83000).i32(148000).i32(-3400).b,
			ExCursedWeaponLocation{Weapons: []CursedWeaponLocation{
				{ItemID: 8689, X: 1, Y: -2, Z: 3},
				{ItemID: 8190, Activated: true, X: 83000, Y: 148000, Z: -3400},
			}}},
		// An empty location list is two zero ints.
		{"ExCursedWeaponLocation", msg(0xfe).u16(0x46).i32(0).i32(0).b, ExCursedWeaponLocation{}},
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
	if _, err := DecodeServerPacket(msg(0xfe).u16(0x46).i32(1).i32(8190).i32(2).i32(0).i32(0).i32(0).b); !errors.Is(err, ErrMalformed) {
		t.Errorf("activated 2: %v, want ErrMalformed", err)
	}
}

func TestCursedWeaponRequestNames(t *testing.T) {
	for raw, want := range map[uint16]string{
		ExOpRequestCursedWeaponList:     "RequestCursedWeaponList",
		ExOpRequestCursedWeaponLocation: "RequestCursedWeaponLocation",
	} {
		if got := clientPacketName(encodeEx(raw)); got != want {
			t.Errorf("clientPacketName(% x) = %q, want %q", encodeEx(raw), got, want)
		}
	}
}
