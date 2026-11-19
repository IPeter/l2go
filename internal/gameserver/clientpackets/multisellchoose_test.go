package clientpackets

import "testing"

// Mobius MultiSellChoose.readImpl: d list ID; d entry ID; d amount
// (12 bytes). The values are not checked when parsing.
const m5MultiSellChooseSize = 12

func TestParseMultiSellChoose(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want MultiSellChoose
	}{
		{"one exchange", new(m3Body).d(300006).d(200003).d(1).b,
			MultiSellChoose{ListID: 300006, EntryID: 200003, Amount: 1}},
		{"amount 0", new(m3Body).d(300006).d(100000).d(0).b,
			MultiSellChoose{ListID: 300006, EntryID: 100000}},
		{"negative amount", new(m3Body).d(1).d(2).d(-5).b,
			MultiSellChoose{ListID: 1, EntryID: 2, Amount: -5}},
		{"trailing data", new(m3Body).d(1).d(2).d(3).c(4).b,
			MultiSellChoose{ListID: 1, EntryID: 2, Amount: 3}},
	}
	for _, tt := range tests {
		got, err := ParseMultiSellChoose(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "MultiSellChoose", new(m3Body).d(1).d(2).d(3).b, ParseMultiSellChoose)
}

func FuzzParseMultiSellChoose(f *testing.F) {
	m3Fuzz(f, ParseMultiSellChoose, m5MultiSellChooseSize, new(m3Body).d(300006).d(200003).d(1).b)
}
