package testclient

import (
	"reflect"
	"testing"
)

// The door, static object and fence packets decode from the Mobius
// writeImpl layouts.
func TestDecodeDoorPackets(t *testing.T) {
	const door, fence = 0x60000010, 0x60000020
	for _, tt := range []struct {
		name string
		raw  []byte
		want Packet
	}{
		// StaticObjectInfo: d static ID, d object ID.
		{"StaticObjectInfo", msg(0x99).i32(24190001).i32(door).b, StaticObjectInfo{StaticID: 24190001, ObjectID: door}},
		// DoorStatusUpdate: d object ID, d closed, d damage, d enemy, d door ID, d max HP, d HP.
		{"DoorStatusUpdate", msg(0x4d).i32(door).i32(1).i32(0).i32(0).i32(24190001).i32(158250).i32(158250).b,
			DoorStatusUpdate{ObjectID: door, Closed: true, DoorID: 24190001, MaxHP: 158250, CurHP: 158250}},
		// ShowTownMap: S texture, d x, d y.
		{"ShowTownMap", msg(0xde).str("town_map.town_map_talking_t00").i32(365).i32(152).b,
			ShowTownMap{Texture: "town_map.town_map_talking_t00", X: 365, Y: 152}},
		// ExColosseumFenceInfo: d object ID, d state, d x, y, z, d width, d length.
		{"ExColosseumFenceInfo", msg(0xfe).u16(0x09).i32(fence).i32(2).i32(-114552).i32(-251256).i32(-2992).i32(100).i32(100).b,
			ExColosseumFenceInfo{ObjectID: fence, State: 2, X: -114552, Y: -251256, Z: -2992, Width: 100, Length: 100}},
		// ExServerPrimitive: S name, d x, y, z, d 65535 twice, d count, shapes.
		{"ExServerPrimitive", msg(0xfe).u16(0x24).str("Door1").i32(1).i32(2).i32(-16000).i32(65535).i32(65535).i32(1).
			u8(2).str("").i32(255).i32(0).i32(0).i32(0).i32(4).i32(5).i32(6).i32(7).i32(8).i32(9).b,
			ExServerPrimitive{Name: "Door1", X: 1, Y: 2, Z: -16000, Shapes: []PrimitiveShape{
				{Line: true, R: 255, X: 4, Y: 5, Z: 6, X2: 7, Y2: 8, Z2: 9},
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
		if got.PacketName() != tt.name {
			t.Errorf("%s: named %q", tt.name, got.PacketName())
		}
		if _, err := DecodeServerPacket(tt.raw[:len(tt.raw)-1]); err == nil {
			t.Errorf("%s: a short packet decoded", tt.name)
		}
	}
	// A damage above 6 and a closed flag of 2 are not Mobius's.
	for _, raw := range [][]byte{
		msg(0x4d).i32(door).i32(1).i32(7).i32(0).i32(1).i32(1).i32(1).b,
		msg(0x4d).i32(door).i32(2).i32(0).i32(0).i32(1).i32(1).i32(1).b,
	} {
		if _, err := DecodeServerPacket(raw); err == nil {
			t.Errorf("malformed DoorStatusUpdate % x decoded", raw)
		}
	}
}
