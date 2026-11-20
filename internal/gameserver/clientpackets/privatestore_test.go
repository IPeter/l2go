package clientpackets

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

// Header sizes of the private store lists: two ints before the entries,
// one for SetPrivateStoreListBuy.
const (
	storeListHeader    = 8
	storeListBuyHeader = 4
)

// RequestPrivateStoreManageSell, RequestPrivateStoreQuitSell,
// RequestPrivateStoreManageBuy and RequestPrivateStoreQuitBuy: readImpl
// reads nothing (Manage Sell has its reads commented out), so any body
// parses.
func TestParseEmptyStorePackets(t *testing.T) {
	for _, body := range [][]byte{nil, {}, {1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}} {
		ms, err := ParseRequestPrivateStoreManageSell(body)
		m3Parsed(t, "RequestPrivateStoreManageSell", ms, err, RequestPrivateStoreManageSell{})
		qs, err := ParseRequestPrivateStoreQuitSell(body)
		m3Parsed(t, "RequestPrivateStoreQuitSell", qs, err, RequestPrivateStoreQuitSell{})
		mb, err := ParseRequestPrivateStoreManageBuy(body)
		m3Parsed(t, "RequestPrivateStoreManageBuy", mb, err, RequestPrivateStoreManageBuy{})
		qb, err := ParseRequestPrivateStoreQuitBuy(body)
		m3Parsed(t, "RequestPrivateStoreQuitBuy", qb, err, RequestPrivateStoreQuitBuy{})
	}
}

// SetPrivateStoreMsgSell and SetPrivateStoreMsgBuy.readImpl: S message,
// read with readString, which stops at the end of the data without
// failing. The length limit is checked by runImpl.
func TestParseStoreMessages(t *testing.T) {
	long := "Selling cheap soulshots, 29 c" + "haracters"
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{"a message", new(m3Body).s("Cheap SS").b, "Cheap SS"},
		{"empty", new(m3Body).s("").b, ""},
		{"empty body", nil, ""},
		{"no terminator", new(m3Body).s("WTB").b[:6], "WTB"},
		{"an odd last byte", append(new(m3Body).s("WTB").b[:6], 'x'), "WTB"},
		{"trailing data", append(new(m3Body).s("B>").b, 'x', 0), "B>"},
		{"too long for runImpl", new(m3Body).s(long).b, long},
		{"non-Latin", new(m3Body).s("Árvíztűrő").b, "Árvíztűrő"},
	}
	for _, tt := range tests {
		s, err := ParseSetPrivateStoreMsgSell(tt.body)
		m3Parsed(t, "SetPrivateStoreMsgSell "+tt.name, s, err, SetPrivateStoreMsgSell{Message: tt.want})
		b, err := ParseSetPrivateStoreMsgBuy(tt.body)
		m3Parsed(t, "SetPrivateStoreMsgBuy "+tt.name, b, err, SetPrivateStoreMsgBuy{Message: tt.want})
	}
}

// Mobius SetPrivateStoreListSell.readImpl: d package sale (== 1), d count;
// per item d object ID, d count, d price (BATCH_LENGTH 12, which must match
// the rest of the body exactly). An object ID or count below 1, or a
// negative price, drops the list.
func TestParseSetPrivateStoreListSell(t *testing.T) {
	single := []int32{0}
	pkg := []int32{1}
	refused := SetPrivateStoreListSell{}
	refusedPkg := SetPrivateStoreListSell{PackageSale: true}
	tests := []struct {
		name string
		body []byte
		want SetPrivateStoreListSell
	}{
		{"two items", m5List(single, 2, []int32{0x10000002, 1, 500000}, []int32{0x10000003, 300, 8}),
			SetPrivateStoreListSell{Items: []SetPrivateStoreListSellItem{
				{ObjectID: 0x10000002, Count: 1, Price: 500000},
				{ObjectID: 0x10000003, Count: 300, Price: 8},
			}}},
		{"a package sale", m5List(pkg, 1, []int32{0x10000003, 300, 8}),
			SetPrivateStoreListSell{PackageSale: true, Items: []SetPrivateStoreListSellItem{{ObjectID: 0x10000003, Count: 300, Price: 8}}}},
		{"package flag 2", m5List([]int32{2}, 1, []int32{0x10000003, 1, 1}),
			SetPrivateStoreListSell{Items: []SetPrivateStoreListSellItem{{ObjectID: 0x10000003, Count: 1, Price: 1}}}},
		{"price 0", m5List(single, 1, []int32{0x10000003, 1, 0}),
			SetPrivateStoreListSell{Items: []SetPrivateStoreListSellItem{{ObjectID: 0x10000003, Count: 1}}}},
		{"the largest price", m5List(single, 1, []int32{0x10000003, math.MaxInt32, math.MaxInt32}),
			SetPrivateStoreListSell{Items: []SetPrivateStoreListSellItem{{ObjectID: 0x10000003, Count: math.MaxInt32, Price: math.MaxInt32}}}},
		{"the most entries", m5List(single, MaxItemsInPacket, m5Repeat(MaxItemsInPacket, []int32{9, 1, 2})...),
			SetPrivateStoreListSell{Items: slices.Repeat([]SetPrivateStoreListSellItem{{ObjectID: 9, Count: 1, Price: 2}}, MaxItemsInPacket)}},
		{"object ID 0", m5List(single, 1, []int32{0, 1, 1}), refused},
		{"count 0", m5List(pkg, 1, []int32{0x10000003, 0, 1}), refusedPkg},
		{"a negative price", m5List(single, 1, []int32{0x10000003, 1, -1}), refused},
		{"a bad second entry", m5List(single, 2, []int32{0x10000002, 1, 1}, []int32{0x10000003, -5, 1}), refused},
		{"no entries", m5List(pkg, 0), refusedPkg},
		{"a negative count", m5List(single, -1, []int32{0x10000002, 1, 1}), refused},
		{"too many entries", m5List(single, MaxItemsInPacket+1, m5Repeat(MaxItemsInPacket+1, []int32{9, 1, 2})...), refused},
		{"a size over the data", m5List(single, 2, []int32{0x10000002, 1, 1}), refused},
		{"a trailing byte", append(m5List(single, 1, []int32{0x10000002, 1, 1}), 0), refused},
	}
	for _, tt := range tests {
		got, err := ParseSetPrivateStoreListSell(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "SetPrivateStoreListSell", new(m3Body).d(0).d(1).b, ParseSetPrivateStoreListSell)
}

// Mobius SetPrivateStoreListBuy.readImpl: d count; per item d item ID, h
// and h (ignored), d count, d price. The size check uses a BATCH_LENGTH of
// 12 and only as an upper bound (count * 12 > remaining drops the list), so
// trailing bytes are ignored and a body that ends inside an entry throws.
func TestParseSetPrivateStoreListBuy(t *testing.T) {
	refused := SetPrivateStoreListBuy{}
	one := []int32{1835, 0, 1000, 5}
	tests := []struct {
		name string
		body []byte
		want SetPrivateStoreListBuy
	}{
		{"two items", m5List(nil, 2, one, []int32{1463, 0, 500, 3}),
			SetPrivateStoreListBuy{Items: []SetPrivateStoreListBuyItem{
				{ItemID: 1835, Count: 1000, Price: 5},
				{ItemID: 1463, Count: 500, Price: 3},
			}}},
		{"the ignored shorts", m5List(nil, 1, []int32{1835, 0x00070003, 10, 5}),
			SetPrivateStoreListBuy{Items: []SetPrivateStoreListBuyItem{{ItemID: 1835, Count: 10, Price: 5}}}},
		{"price 0", m5List(nil, 1, []int32{1835, 0, 10, 0}),
			SetPrivateStoreListBuy{Items: []SetPrivateStoreListBuyItem{{ItemID: 1835, Count: 10}}}},
		{"trailing data", append(m5List(nil, 1, one), 1, 2, 3),
			SetPrivateStoreListBuy{Items: []SetPrivateStoreListBuyItem{{ItemID: 1835, Count: 1000, Price: 5}}}},
		{"the most entries", m5List(nil, MaxItemsInPacket, m5Repeat(MaxItemsInPacket, one)...),
			SetPrivateStoreListBuy{Items: slices.Repeat([]SetPrivateStoreListBuyItem{{ItemID: 1835, Count: 1000, Price: 5}}, MaxItemsInPacket)}},
		{"item ID 0", m5List(nil, 1, []int32{0, 0, 1, 1}), refused},
		{"count 0", m5List(nil, 1, []int32{1835, 0, 0, 1}), refused},
		{"a negative price", m5List(nil, 1, []int32{1835, 0, 1, -1}), refused},
		{"a bad second entry", m5List(nil, 2, one, []int32{1835, 0, -1, 1}), refused},
		{"no entries", m5List(nil, 0), refused},
		{"a negative count", m5List(nil, -1, one), refused},
		{"too many entries", m5List(nil, MaxItemsInPacket+1, m5Repeat(MaxItemsInPacket+1, one)...), refused},
		// 2 entries need 24 bytes for the check; 23 are there.
		{"under 12 bytes per entry", append(m5List(nil, 2, one), 1, 2, 3, 4, 5, 6, 7), refused},
		// The check passes with 24 bytes, but the first entry is refused
		// before the second one runs short.
		{"a bad entry before the end", append(m5List(nil, 2, []int32{0, 0, 1, 1}), 1, 2, 3, 4, 5, 6, 7, 8), refused},
	}
	for _, tt := range tests {
		got, err := ParseSetPrivateStoreListBuy(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "SetPrivateStoreListBuy", new(m3Body).d(1).b, ParseSetPrivateStoreListBuy)

	// The check passes (2 * 12 <= 24) but the second entry ends after its
	// item ID and shorts: Mobius fails to read the packet.
	short := append(m5List(nil, 2, one), 1, 0, 0, 0, 0, 0, 0, 0)
	got, err := ParseSetPrivateStoreListBuy(short)
	m3Failed(t, "a body that ends inside an entry", got, err)
	// The same with 3 entries in 36 bytes: the third entry is cut.
	got, err = ParseSetPrivateStoreListBuy(append(m5List(nil, 3, one, one), 1, 2, 3, 4))
	m3Failed(t, "36 bytes for 3 entries", got, err)
}

// Mobius RequestPrivateStoreBuy.readImpl: d store player, d count; per item
// d object ID, d count, d price (BATCH_LENGTH 12, exact). An object ID or
// count below 1, or a negative price, drops the list.
func TestParseRequestPrivateStoreBuy(t *testing.T) {
	const store = 0x10000009
	header := []int32{store}
	refused := RequestPrivateStoreBuy{StorePlayerID: store}
	tests := []struct {
		name string
		body []byte
		want RequestPrivateStoreBuy
	}{
		{"two items", m5List(header, 2, []int32{0x10000102, 1, 500000}, []int32{0x10000103, 100, 8}),
			RequestPrivateStoreBuy{StorePlayerID: store, Items: []RequestPrivateStoreBuyItem{
				{ObjectID: 0x10000102, Count: 1, Price: 500000},
				{ObjectID: 0x10000103, Count: 100, Price: 8},
			}}},
		// Mobius's inverted RequestTrade.equals keeps both entries.
		{"a duplicate", m5List(header, 2, []int32{0x10000103, 1, 8}, []int32{0x10000103, 2, 8}),
			RequestPrivateStoreBuy{StorePlayerID: store, Items: []RequestPrivateStoreBuyItem{
				{ObjectID: 0x10000103, Count: 1, Price: 8},
				{ObjectID: 0x10000103, Count: 2, Price: 8},
			}}},
		{"price 0", m5List(header, 1, []int32{0x10000103, 1, 0}),
			RequestPrivateStoreBuy{StorePlayerID: store, Items: []RequestPrivateStoreBuyItem{{ObjectID: 0x10000103, Count: 1}}}},
		{"store 0", m5List([]int32{0}, 1, []int32{0x10000103, 1, 1}),
			RequestPrivateStoreBuy{Items: []RequestPrivateStoreBuyItem{{ObjectID: 0x10000103, Count: 1, Price: 1}}}},
		{"the most entries", m5List(header, MaxItemsInPacket, m5Repeat(MaxItemsInPacket, []int32{9, 1, 2})...),
			RequestPrivateStoreBuy{StorePlayerID: store, Items: slices.Repeat([]RequestPrivateStoreBuyItem{{ObjectID: 9, Count: 1, Price: 2}}, MaxItemsInPacket)}},
		{"object ID 0", m5List(header, 1, []int32{0, 1, 1}), refused},
		{"count 0", m5List(header, 1, []int32{0x10000103, 0, 1}), refused},
		{"a negative price", m5List(header, 1, []int32{0x10000103, 1, -8}), refused},
		{"a bad second entry", m5List(header, 2, []int32{0x10000102, 1, 1}, []int32{-1, 1, 1}), refused},
		{"no entries", m5List(header, 0), refused},
		{"too many entries", m5List(header, MaxItemsInPacket+1, m5Repeat(MaxItemsInPacket+1, []int32{9, 1, 2})...), refused},
		{"a size over the data", m5List(header, 2, []int32{0x10000102, 1, 1}), refused},
		{"a trailing byte", append(m5List(header, 1, []int32{0x10000102, 1, 1}), 0), refused},
		{"sell entries", m5List(header, 1, []int32{0x10000102, 1835, 0, 1, 1}), refused},
	}
	for _, tt := range tests {
		got, err := ParseRequestPrivateStoreBuy(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestPrivateStoreBuy", new(m3Body).d(store).d(1).b, ParseRequestPrivateStoreBuy)
}

// Mobius RequestPrivateStoreSell.readImpl: d store player, d count; per
// item d object ID, d item ID, h and h (ignored), d count, d price
// (BATCH_LENGTH 20, exact). An object ID, item ID or count below 1, or a
// negative price, drops the list.
func TestParseRequestPrivateStoreSell(t *testing.T) {
	const store = 0x10000009
	header := []int32{store}
	refused := RequestPrivateStoreSell{StorePlayerID: store}
	tests := []struct {
		name string
		body []byte
		want RequestPrivateStoreSell
	}{
		{"two items", m5List(header, 2, []int32{0x10000202, 1835, 0, 1000, 5}, []int32{0x10000203, 1463, 0x00010002, 500, 3}),
			RequestPrivateStoreSell{StorePlayerID: store, Items: []RequestPrivateStoreSellItem{
				{ObjectID: 0x10000202, ItemID: 1835, Count: 1000, Price: 5},
				{ObjectID: 0x10000203, ItemID: 1463, Count: 500, Price: 3},
			}}},
		{"price 0", m5List(header, 1, []int32{0x10000202, 1835, 0, 1, 0}),
			RequestPrivateStoreSell{StorePlayerID: store, Items: []RequestPrivateStoreSellItem{{ObjectID: 0x10000202, ItemID: 1835, Count: 1}}}},
		{"the most entries", m5List(header, MaxItemsInPacket, m5Repeat(MaxItemsInPacket, []int32{9, 57, 0, 1, 2})...),
			RequestPrivateStoreSell{StorePlayerID: store, Items: slices.Repeat([]RequestPrivateStoreSellItem{{ObjectID: 9, ItemID: 57, Count: 1, Price: 2}}, MaxItemsInPacket)}},
		{"object ID 0", m5List(header, 1, []int32{0, 1835, 0, 1, 1}), refused},
		{"item ID 0", m5List(header, 1, []int32{0x10000202, 0, 0, 1, 1}), refused},
		{"count 0", m5List(header, 1, []int32{0x10000202, 1835, 0, 0, 1}), refused},
		{"a negative price", m5List(header, 1, []int32{0x10000202, 1835, 0, 1, -1}), refused},
		{"a bad second entry", m5List(header, 2, []int32{0x10000202, 1835, 0, 1, 1}, []int32{0x10000203, -1, 0, 1, 1}), refused},
		{"no entries", m5List(header, 0), refused},
		{"too many entries", m5List(header, MaxItemsInPacket+1, m5Repeat(MaxItemsInPacket+1, []int32{9, 57, 0, 1, 2})...), refused},
		{"a size over the data", m5List(header, 2, []int32{0x10000202, 1835, 0, 1, 1}), refused},
		{"a trailing byte", append(m5List(header, 1, []int32{0x10000202, 1835, 0, 1, 1}), 0), refused},
		{"buy entries", m5List(header, 1, []int32{0x10000202, 1, 1}), refused},
	}
	for _, tt := range tests {
		got, err := ParseRequestPrivateStoreSell(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestPrivateStoreSell", new(m3Body).d(store).d(1).b, ParseRequestPrivateStoreSell)
}

func FuzzParseSetPrivateStoreListSell(f *testing.F) {
	m5FuzzList(f, ParseSetPrivateStoreListSell,
		func(p SetPrivateStoreListSell) []SetPrivateStoreListSellItem { return p.Items },
		storeListHeader, storeListSellItemSize,
		func(it SetPrivateStoreListSellItem) bool { return it.ObjectID >= 1 && it.Count >= 1 && it.Price >= 0 },
		m5List([]int32{1}, 2, []int32{0x10000002, 1, 500000}, []int32{0x10000003, 300, 8}),
		m5List([]int32{0}, 1, []int32{0x10000003, 1, -1}),
		m5List([]int32{0}, -1),
		[]byte{0xff})
}

func FuzzParseRequestPrivateStoreBuy(f *testing.F) {
	m5FuzzList(f, ParseRequestPrivateStoreBuy,
		func(p RequestPrivateStoreBuy) []RequestPrivateStoreBuyItem { return p.Items },
		storeListHeader, storeBuyRequestItemSize,
		func(it RequestPrivateStoreBuyItem) bool { return it.ObjectID >= 1 && it.Count >= 1 && it.Price >= 0 },
		m5List([]int32{0x10000009}, 2, []int32{0x10000102, 1, 500000}, []int32{0x10000103, 100, 8}),
		m5List([]int32{0x10000009}, 1, []int32{0, 1, 1}),
		m5List([]int32{0x10000009}, MaxItemsInPacket+1),
		[]byte{0xff})
}

func FuzzParseRequestPrivateStoreSell(f *testing.F) {
	m5FuzzList(f, ParseRequestPrivateStoreSell,
		func(p RequestPrivateStoreSell) []RequestPrivateStoreSellItem { return p.Items },
		storeListHeader, storeSellRequestItemSize,
		func(it RequestPrivateStoreSellItem) bool {
			return it.ObjectID >= 1 && it.ItemID >= 1 && it.Count >= 1 && it.Price >= 0
		},
		m5List([]int32{0x10000009}, 2, []int32{0x10000202, 1835, 0, 1000, 5}, []int32{0x10000203, 1463, 0, 500, 3}),
		m5List([]int32{0x10000009}, 1, []int32{0x10000202, 0, 0, 1, 1}),
		m5List([]int32{0x10000009}, 0),
		[]byte{0xff})
}

// FuzzParseSetPrivateStoreListBuy checks the looser size rule of
// SetPrivateStoreListBuy: a failure is ErrShortPacket with the zero value,
// and an accepted list has 1 to MaxItemsInPacket valid entries of 16 bytes
// that fit in the body, with the count read from its first four bytes.
func FuzzParseSetPrivateStoreListBuy(f *testing.F) {
	one := []int32{1835, 0, 1000, 5}
	f.Add([]byte{})
	f.Add(m5List(nil, 2, one, []int32{1463, 0, 500, 3}))
	f.Add(append(m5List(nil, 1, one), 1, 2, 3))
	f.Add(append(m5List(nil, 2, one), 1, 0, 0, 0, 0, 0, 0, 0))
	f.Add(m5List(nil, 1, []int32{1835, 0, 1, -1}))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseSetPrivateStoreListBuy(data)
		if err != nil {
			if !errors.Is(err, packets.ErrShortPacket) || !reflect.DeepEqual(got, SetPrivateStoreListBuy{}) {
				t.Fatalf("%d bytes: %+v, %v", len(data), got, err)
			}
			if len(data) >= storeListBuyHeader {
				n := int(packets.NewReader(data).ReadInt32())
				if n < 1 || n > MaxItemsInPacket || storeListBuyHeader+n*storeListBuyItemSize <= len(data) {
					t.Errorf("%d bytes, count %d: %v", len(data), n, err)
				}
			}
			return
		}
		if len(data) < storeListBuyHeader {
			t.Fatalf("%d bytes parsed", len(data))
		}
		if got.Items == nil {
			return
		}
		n := len(got.Items)
		if n < 1 || n > MaxItemsInPacket || storeListBuyHeader+n*storeListBuyItemSize > len(data) ||
			int(packets.NewReader(data).ReadInt32()) != n {
			t.Errorf("%d bytes: %d entries accepted", len(data), n)
		}
		for _, it := range got.Items {
			if it.ItemID < 1 || it.Count < 1 || it.Price < 0 {
				t.Errorf("entry %+v accepted", it)
			}
		}
	})
}

func FuzzParseSetPrivateStoreMsgSell(f *testing.F) {
	m3Fuzz(f, ParseSetPrivateStoreMsgSell, 0, new(m3Body).s("Cheap SS").b, []byte{0x41})
}

func FuzzParseSetPrivateStoreMsgBuy(f *testing.F) {
	m3Fuzz(f, ParseSetPrivateStoreMsgBuy, 0, new(m3Body).s("WTB").b, []byte{0x41})
}
