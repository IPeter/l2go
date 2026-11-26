package testclient

import (
	"bytes"
	"errors"
	"testing"
)

// The fishing packets as Mobius writes them.
func TestFishingPackets(t *testing.T) {
	for _, c := range []struct {
		raw  []byte
		want Packet
	}{
		{[]byte{0xfe, 0x13, 0x00, 1, 0, 0, 0x10, 0xff, 0xff, 0xff, 0xff, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 1, 0},
			ExFishingStart{ObjectID: 0x10000001, FishType: -1, X: 1, Y: 2, Z: 3, Night: true}},
		{[]byte{0xfe, 0x14, 0x00, 9, 0, 0, 0, 1}, ExFishingEnd{ObjectID: 9, Win: true}},
		{[]byte{0xfe, 0x15, 0x00, 9, 0, 0, 0, 24, 0, 0, 0, 100, 0, 0, 0, 1, 2, 1},
			ExFishingStartCombat{ObjectID: 9, Time: 24, HP: 100, Mode: 1, LureType: 2, Deceptive: 1}},
		{[]byte{0xfe, 0x16, 0x00, 9, 0, 0, 0, 23, 0, 0, 0, 60, 0, 0, 0, 0, 1, 1, 2, 0, 0, 0, 1},
			ExFishingHpRegen{ObjectID: 9, Time: 23, HP: 60, GoodUse: 1, Anim: 1, Penalty: 2, BarColor: 1}},
	} {
		got, err := DecodeServerPacket(c.raw)
		if err != nil || got != c.want {
			t.Errorf("DecodeServerPacket(% x) = %+v, %v; want %+v", c.raw, got, err, c.want)
		}
	}
	// A flag that is not 0 or 1 is malformed.
	if _, err := DecodeServerPacket([]byte{0xfe, 0x14, 0x00, 9, 0, 0, 0, 2}); !errors.Is(err, ErrMalformed) {
		t.Errorf("a win flag of 2: err = %v, want ErrMalformed", err)
	}
	if got := encodeEx(ExOpRequestExFishRanking); !bytes.Equal(got, []byte{0xd0, 0x1f, 0x00}) {
		t.Errorf("RequestExFishRanking = % x", got)
	}
}
