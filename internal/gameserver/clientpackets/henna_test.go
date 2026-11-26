package clientpackets

import (
	"math"
	"strings"
	"testing"
)

// The henna packets (Mobius ClientPackets 0xBA-0xBF), each a single d.
var hennaParsers = []struct {
	name  string
	op    int
	want  int
	parse func([]byte) (RequestHenna, error)
}{
	{"RequestHennaItemList", OpRequestHennaItemList, 0xba, ParseRequestHennaItemList},
	{"RequestHennaItemInfo", OpRequestHennaItemInfo, 0xbb, ParseRequestHennaItemInfo},
	{"RequestHennaEquip", OpRequestHennaEquip, 0xbc, ParseRequestHennaEquip},
	{"RequestHennaRemoveList", OpRequestHennaRemoveList, 0xbd, ParseRequestHennaRemoveList},
	{"RequestHennaItemRemoveInfo", OpRequestHennaItemRemoveInfo, 0xbe, ParseRequestHennaItemRemoveInfo},
	{"RequestHennaRemove", OpRequestHennaRemove, 0xbf, ParseRequestHennaRemove},
}

func TestParseRequestHenna(t *testing.T) {
	for _, p := range hennaParsers {
		if p.op != p.want {
			t.Errorf("%s opcode = %#x, want %#x", p.name, p.op, p.want)
		}
		for _, id := range []int32{0, 1, 180, -1, math.MaxInt32} {
			got, err := p.parse(new(m67Body).d(id).b)
			m67Parsed(t, p.name, got, err, RequestHenna{SymbolID: id})
		}
		got, err := p.parse(new(m67Body).d(7).c(0xff).b)
		m67Parsed(t, p.name+" with trailing data", got, err, RequestHenna{SymbolID: 7})
		m67Truncations(t, p.name, new(m67Body).d(7).b, p.parse)
		if _, err := p.parse(nil); err == nil || !strings.Contains(err.Error(), p.name) {
			t.Errorf("%s: error %v does not name the packet", p.name, err)
		}
	}
}

func FuzzParseRequestHenna(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(1).b)
	f.Add(new(m67Body).d(-1).c(1).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, p := range hennaParsers {
			got, err := p.parse(data)
			if !m67Fuzzed(t, data, got, err, 4, false, nil) {
				continue
			}
			if got.SymbolID != m67Int(data, 0) {
				t.Errorf("%s % x: got %+v", p.name, data, got)
			}
		}
	})
}
