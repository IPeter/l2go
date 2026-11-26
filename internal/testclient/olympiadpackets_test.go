package testclient

import (
	"bytes"
	"reflect"
	"testing"
)

// The Olympiad packets as Mobius writes them.
func TestOlympiadPackets(t *testing.T) {
	str := func(s string) []byte {
		var b []byte
		for _, r := range s {
			b = append(b, byte(r), 0)
		}
		return append(b, 0, 0)
	}
	cat := func(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
	for _, c := range []struct {
		raw  []byte
		want Packet
	}{
		{cat([]byte{0xfe, 0x23, 0x00, 1, 0, 0, 0}, str("Ab"), []byte{88, 0, 0, 0}, str("C"), []byte{5, 0, 0, 0}, str(""), []byte{0, 0, 0, 0, 2, 0, 0, 0}),
			ExHeroList{Heroes: []HeroListEntry{{Name: "Ab", ClassID: 88, ClanName: "C", ClanCrestID: 5, Count: 2}}}},
		{cat([]byte{0xfe, 0x29, 0x00, 2, 7, 0, 0, 0}, str("Ab"), []byte{88, 0, 0, 0, 10, 0, 0, 0, 20, 0, 0, 0, 3, 0, 0, 0, 4, 0, 0, 0}),
			ExOlympiadUserInfo{Side: 2, ObjectID: 7, Name: "Ab", ClassID: 88, HP: 10, MaxHP: 20, CP: 3, MaxCP: 4}},
		{[]byte{0xfe, 0x2a, 0x00, 7, 0, 0, 0, 1, 0, 0, 0, 0xb4, 0x04, 0, 0, 2, 0, 0xff, 0xff, 0xff, 0xff},
			ExOlympiadSpelledInfo{ObjectID: 7, Effects: []SpelledEffect{{SkillID: 1204, Level: 2, Seconds: -1}}}},
		{[]byte{0xfe, 0x2b, 0x00, 3}, ExOlympiadMode{Mode: 3}},
		{[]byte{0xfe, 0x2c, 0x00}, ExOlympiadMatchEnd{}},
		{[]byte{0xdf, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 0, 0xc0, 0}, ObservationEnter{X: 1, Y: 2, Z: 3}},
		{[]byte{0xe0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}, ObservationExit{X: 1, Y: 2, Z: 3}},
	} {
		got, err := DecodeServerPacket(c.raw)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("DecodeServerPacket(% x) = %+v, %v; want %+v", c.raw, got, err, c.want)
		}
	}
	if got := encodeEx(ExOpRequestOlympiadObserverEnd); !bytes.Equal(got, []byte{0xd0, 0x12, 0x00}) {
		t.Errorf("RequestOlympiadObserverEnd = % x", got)
	}
}
