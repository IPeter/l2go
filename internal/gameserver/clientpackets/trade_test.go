package clientpackets

import (
	"math"
	"testing"
)

// The opcodes are those of Mobius ClientPackets.
func TestTradeAndStoreOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"TradeRequest", OpTradeRequest, 0x15},
		{"AddTradeItem", OpAddTradeItem, 0x16},
		{"TradeDone", OpTradeDone, 0x17},
		{"AnswerTradeRequest", OpAnswerTradeRequest, 0x44},
		{"RequestPrivateStoreManageSell", OpRequestPrivateStoreManageSell, 0x73},
		{"SetPrivateStoreListSell", OpSetPrivateStoreListSell, 0x74},
		{"RequestPrivateStoreQuitSell", OpRequestPrivateStoreQuitSell, 0x76},
		{"SetPrivateStoreMsgSell", OpSetPrivateStoreMsgSell, 0x77},
		{"RequestPrivateStoreBuy", OpRequestPrivateStoreBuy, 0x79},
		{"RequestPrivateStoreManageBuy", OpRequestPrivateStoreManageBuy, 0x90},
		{"SetPrivateStoreListBuy", OpSetPrivateStoreListBuy, 0x91},
		{"RequestPrivateStoreQuitBuy", OpRequestPrivateStoreQuitBuy, 0x93},
		{"SetPrivateStoreMsgBuy", OpSetPrivateStoreMsgBuy, 0x94},
		{"RequestPrivateStoreSell", OpRequestPrivateStoreSell, 0x96},
		// RequestActionUse.runImpl: case 10 "Private Store - Sell", case 28
		// "Private Store - Buy".
		{"action 10", ActionUsePrivateStoreSell, 10},
		{"action 28", ActionUsePrivateStoreBuy, 28},
		// SetPrivateStoreMsgSell/Buy.MAX_MSG_LENGTH.
		{"MAX_MSG_LENGTH", PrivateStoreMessageMaxLength, 29},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// Mobius TradeRequest.readImpl: d object ID.
func TestParseTradeRequest(t *testing.T) {
	got, err := ParseTradeRequest(new(m3Body).d(0x10000002).b)
	m3Parsed(t, "TradeRequest", got, err, TradeRequest{ObjectID: 0x10000002})
	got, err = ParseTradeRequest(new(m3Body).d(7).c(1).b)
	m3Parsed(t, "trailing data", got, err, TradeRequest{ObjectID: 7})
	m3Truncations(t, "TradeRequest", new(m3Body).d(7).b, ParseTradeRequest)
}

// Mobius AddTradeItem.readImpl: d trade ID, d object ID, d count. The count
// is checked by runImpl.
func TestParseAddTradeItem(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want AddTradeItem
	}{
		{"one sword", new(m3Body).d(0).d(0x10000002).d(1).b, AddTradeItem{ObjectID: 0x10000002, Count: 1}},
		{"adena", new(m3Body).d(3).d(0x10000004).d(math.MaxInt32).b,
			AddTradeItem{TradeID: 3, ObjectID: 0x10000004, Count: math.MaxInt32}},
		{"count 0", new(m3Body).d(0).d(5).d(0).b, AddTradeItem{ObjectID: 5}},
		{"a negative count", new(m3Body).d(0).d(5).d(-1).b, AddTradeItem{ObjectID: 5, Count: -1}},
		{"trailing data", new(m3Body).d(1).d(2).d(3).c(4).b, AddTradeItem{TradeID: 1, ObjectID: 2, Count: 3}},
	}
	for _, tt := range tests {
		got, err := ParseAddTradeItem(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "AddTradeItem", new(m3Body).d(1).d(2).d(3).b, ParseAddTradeItem)
}

// Mobius TradeDone.readImpl: d response; runImpl confirms on 1 and cancels
// on anything else.
func TestParseTradeDone(t *testing.T) {
	tests := []struct {
		name string
		v    int32
		want bool
	}{
		{"confirm", 1, true},
		{"cancel", 0, false},
		{"2 cancels", 2, false},
		{"-1 cancels", -1, false},
		{"256 cancels", 256, false},
	}
	for _, tt := range tests {
		got, err := ParseTradeDone(new(m3Body).d(tt.v).b)
		m3Parsed(t, tt.name, got, err, TradeDone{Confirm: tt.want})
	}
	got, err := ParseTradeDone(new(m3Body).d(1).c(0).b)
	m3Parsed(t, "trailing data", got, err, TradeDone{Confirm: true})
	m3Truncations(t, "TradeDone", new(m3Body).d(1).b, ParseTradeDone)
}

// Mobius AnswerTradeRequest.readImpl: d response; runImpl starts the trade
// on 1 only.
func TestParseAnswerTradeRequest(t *testing.T) {
	tests := []struct {
		name string
		v    int32
		want bool
	}{
		{"accept", 1, true},
		{"refuse", 0, false},
		{"2 refuses", 2, false},
		{"0x101 refuses", 0x101, false},
	}
	for _, tt := range tests {
		got, err := ParseAnswerTradeRequest(new(m3Body).d(tt.v).b)
		m3Parsed(t, tt.name, got, err, AnswerTradeRequest{Accept: tt.want})
	}
	m3Truncations(t, "AnswerTradeRequest", new(m3Body).d(1).b, ParseAnswerTradeRequest)
}

func FuzzParseTradeRequest(f *testing.F) {
	m3Fuzz(f, ParseTradeRequest, 4, new(m3Body).d(0x10000002).b)
}

func FuzzParseAddTradeItem(f *testing.F) {
	m3Fuzz(f, ParseAddTradeItem, 12, new(m3Body).d(0).d(0x10000002).d(1).b)
}

func FuzzParseTradeDone(f *testing.F) {
	m3Fuzz(f, ParseTradeDone, 4, new(m3Body).d(1).b)
}

func FuzzParseAnswerTradeRequest(f *testing.F) {
	m3Fuzz(f, ParseAnswerTradeRequest, 4, new(m3Body).d(1).b)
}
