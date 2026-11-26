package testclient

import (
	"reflect"
	"testing"
)

// AgitDecoInfo decodes from the Mobius writeImpl layout: d hall, 13 bytes.
func TestDecodeAgitDecoInfo(t *testing.T) {
	raw := msg(0xf7).i32(22).raw(2, 1, 1, 2, 1, 0, 2, 2, 1, 1, 1, 2).u8(0).b
	got, err := DecodeServerPacket(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := AgitDecoInfo{HallID: 22, Deco: [13]uint8{2, 1, 1, 2, 1, 0, 2, 2, 1, 1, 1, 2, 0}}
	if !reflect.DeepEqual(got, want) || got.PacketName() != "AgitDecoInfo" {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if _, err := DecodeServerPacket(raw[:len(raw)-1]); err == nil {
		t.Error("a short AgitDecoInfo decoded")
	}
	bad := msg(0xf7).i32(22).raw(3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0).b
	if _, err := DecodeServerPacket(bad); err == nil {
		t.Error("a decoration of 3 decoded")
	}
}
