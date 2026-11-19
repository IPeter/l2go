package serverpackets

import (
	"math"
	"testing"
)

// Mobius BuyList.writeImpl:
//
//	c 0x11
//	d adena, list ID
//	h product count (every product)
//	per product in stock or not limited:
//	  h type1; d 0 (object ID); d item ID; d count left (0 if not limited)
//	  h type2; h 0 (equipped)
//	  d body part mask, 0 if type1 is 4; h 0 (enchant); h 0; h 0
//	  d (int) (price * (1 + tax)); for items 3960 to 4026
//	    (int) (price * RATE_SIEGE_GUARDS_PRICE * (1 + tax))
func TestBuyListPacket(t *testing.T) {
	l := BuyList{
		ListID: 300001, Adena: 5000, TaxRate: 0.1, SiegeGuardsPriceRate: 1.5,
		Products: []BuyListProduct{
			// A sword: 138 * 1.1 = 151.8, cut to 151.
			{ItemID: 2369, Type1: 0, Type2: 0, BodyPart: 0x80, Price: 138},
			// Shots with stock left. Type 1 4 sends no body part.
			{ItemID: 1835, Type1: 4, Type2: 5, BodyPart: 0x100, Price: 7, LimitedStock: true, Count: 300},
			// Sold out: counted, but not written.
			{ItemID: 1060, Type1: 4, Type2: 5, Price: 30, LimitedStock: true, Count: 0},
			// The first and last tickets: 10000 * 1.5 * 1.1 = 16500.
			{ItemID: 3960, Type1: 4, Type2: 5, Price: 10000},
			{ItemID: 4026, Type1: 4, Type2: 5, Price: 10000},
			// The items around them have no rate: 10000 * 1.1 = 11000.
			{ItemID: 3959, Type1: 4, Type2: 5, Price: 10000},
			{ItemID: 4027, Type1: 4, Type2: 5, Price: 10000},
			// Not limited: the count is not sent. 26 * 1.1 = 28.6.
			{ItemID: 1146, Type1: 1, Type2: 1, BodyPart: 0x400, Price: 26, Count: 9},
			// A negative stock is none left.
			{ItemID: 1147, Type1: 1, Type2: 1, BodyPart: 0x800, Price: 29, LimitedStock: true, Count: -3},
		},
	}
	var g m3Golden
	g.c(0x11).d(5000).d(300001).h(9)
	product := func(type1 int16, itemID, count int32, type2 int16, bodyPart, price int32) {
		g.h(type1).d(0).d(itemID).d(count)
		g.h(type2).h(0)
		g.d(bodyPart).h(0).h(0).h(0)
		g.d(price)
	}
	product(0, 2369, 0, 0, 0x80, 151)
	product(4, 1835, 300, 5, 0, 7)
	product(4, 3960, 0, 5, 0, 16500)
	product(4, 4026, 0, 5, 0, 16500)
	product(4, 3959, 0, 5, 0, 11000)
	product(4, 4027, 0, 5, 0, 11000)
	product(1, 1146, 0, 1, 0x400, 28)
	m3Check(t, "BuyList", NewBuyListPacket(l), g.b)

	m3Check(t, "empty BuyList",
		NewBuyListPacket(BuyList{ListID: 7, Adena: 100}),
		unhex(t, "11 64000000 07000000 0000"))
}

// The price BuyList shows. Mobius multiplies a ticket's price by the rate in
// Java float precision, where 16,777,217 is 16,777,216; the Java int cast
// saturates and turns NaN into 0.
func TestBuyListPrice(t *testing.T) {
	tests := []struct {
		name          string
		itemID, price int32
		tax           float64
		rate          float32
		want          int32
	}{
		{"no tax", 2369, 138, 0, 1, 138},
		{"tax", 2369, 138, 0.1, 1, 151},
		{"a ticket in float precision", 3960, 16777217, 0, 1, 16777216},
		{"another item in double precision", 57, 16777217, 0, 1, 16777217},
		{"a ticket at half price", 4000, 1000, 0, 0.5, 500},
		{"a ticket with rate and tax", 4000, 1000, 0.5, 2, 3000},
		{"no rate for other items", 1835, 1000, 0, 2, 1000},
		{"a ticket at rate 0", 3960, 1000, 0.1, 0, 0},
		{"saturated", 2369, 2000000000, 0.5, 1, math.MaxInt32},
		{"NaN tax", 2369, 138, math.NaN(), 1, 0},
	}
	for _, tt := range tests {
		got := buyListPrice(BuyListProduct{ItemID: tt.itemID, Price: tt.price}, tt.tax, tt.rate)
		if got != tt.want {
			t.Errorf("%s: price %d, want %d", tt.name, got, tt.want)
		}
	}
}
