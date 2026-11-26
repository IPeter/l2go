package clientpackets

import "testing"

// Mobius RequestProcureCropList.readImpl: d count; per crop d object, d
// item, d manor, d count. The list is dropped unless the count is 1 to
// MAX_ITEM_IN_PACKET, the entries fill the body exactly, and every entry has
// object and item IDs of at least 1 and a manor and count of at least 0.
func TestParseRequestProcureCropList(t *testing.T) {
	var refused RequestProcureCropList
	tests := []struct {
		name string
		body []byte
		want RequestProcureCropList
	}{
		{"two crops", m5List(nil, 2, []int32{0x10000101, 5073, 1, 7}, []int32{0x10000102, 5818, 0, 0}),
			RequestProcureCropList{Items: []ProcureCropItem{{0x10000101, 5073, 1, 7}, {0x10000102, 5818, 0, 0}}}},
		{"object 0", m5List(nil, 1, []int32{0, 5073, 1, 7}), refused},
		{"item 0", m5List(nil, 1, []int32{1, 0, 1, 7}), refused},
		{"negative manor", m5List(nil, 1, []int32{1, 5073, -1, 7}), refused},
		{"negative count", m5List(nil, 1, []int32{1, 5073, 1, -7}), refused},
		{"no entries", m5List(nil, 0), refused},
		{"a size over the data", m5List(nil, 2, []int32{1, 5073, 1, 7}), refused},
	}
	for _, tt := range tests {
		got, err := ParseRequestProcureCropList(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestProcureCropList", new(m3Body).d(1).b, ParseRequestProcureCropList)
}

func FuzzParseRequestProcureCropList(f *testing.F) {
	m5FuzzList(f, ParseRequestProcureCropList,
		func(p RequestProcureCropList) []ProcureCropItem { return p.Items },
		4, procureCropItemSize,
		func(it ProcureCropItem) bool { return it.ObjectID >= 1 && it.ItemID >= 1 && it.ManorID >= 0 && it.Count >= 0 },
		m5List(nil, 1, []int32{1, 5073, 1, 7}), []byte{0x01})
}

// Mobius RequestSetSeed.readImpl: d manor, d count; per seed d id, d sales,
// d price. Seeds with no sales are skipped; a bad entry empties the list.
func TestParseRequestSetSeed(t *testing.T) {
	header := []int32{1}
	refused := RequestSetSeed{ManorID: 1}
	tests := []struct {
		name string
		body []byte
		want RequestSetSeed
	}{
		{"two seeds, one without sales", m5List(header, 3, []int32{5016, 100, 10}, []int32{5650, 0, 30}, []int32{7019, 5, 0}),
			RequestSetSeed{ManorID: 1, Items: []SetSeedItem{{5016, 100, 10}, {7019, 5, 0}}}},
		{"none with sales", m5List(header, 1, []int32{5016, 0, 10}), refused},
		{"seed 0", m5List(header, 1, []int32{0, 1, 10}), refused},
		{"negative sales", m5List(header, 2, []int32{5016, 1, 10}, []int32{5650, -1, 10}), refused},
		{"negative price", m5List(header, 1, []int32{5016, 1, -10}), refused},
		{"a bad size", m5List(header, 2, []int32{5016, 1, 10}), refused},
	}
	for _, tt := range tests {
		got, err := ParseRequestSetSeed(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestSetSeed", new(m3Body).d(1).d(0).b, ParseRequestSetSeed)
}

// Mobius RequestSetCrop.readImpl: d manor, d count; per crop d id, d sales,
// d price, c reward type.
func TestParseRequestSetCrop(t *testing.T) {
	body := new(m3Body).d(2).d(2).d(5073).d(10).d(100).c(1).d(5818).d(0).d(200).c(2).b
	got, err := ParseRequestSetCrop(body)
	m3Parsed(t, "two crops", got, err, RequestSetCrop{ManorID: 2, Items: []SetCropItem{{5073, 10, 100, 1}}})
	bad := new(m3Body).d(2).d(1).d(5073).d(-1).d(100).c(1).b
	got, err = ParseRequestSetCrop(bad)
	m3Parsed(t, "negative sales", got, err, RequestSetCrop{ManorID: 2})
	short := new(m3Body).d(2).d(1).d(5073).d(1).d(100).b
	got, err = ParseRequestSetCrop(short)
	m3Parsed(t, "no reward byte", got, err, RequestSetCrop{ManorID: 2})
	m3Truncations(t, "RequestSetCrop", new(m3Body).d(1).d(0).b, ParseRequestSetCrop)
}

func FuzzParseRequestSetSeed(f *testing.F) {
	m3Fuzz(f, ParseRequestSetSeed, -1, m5List([]int32{1}, 1, []int32{5016, 1, 10}))
}

func FuzzParseRequestSetCrop(f *testing.F) {
	m3Fuzz(f, ParseRequestSetCrop, -1, new(m3Body).d(2).d(1).d(5073).d(10).d(100).c(1).b)
}

// Mobius RequestBuySeed.readImpl: d manor, d count; per seed d id, d count.
func TestParseRequestBuySeed(t *testing.T) {
	header := []int32{1}
	refused := RequestBuySeed{ManorID: 1}
	tests := []struct {
		name string
		body []byte
		want RequestBuySeed
	}{
		{"two seeds", m5List(header, 2, []int32{5016, 10}, []int32{5650, 1}),
			RequestBuySeed{ManorID: 1, Items: []BuySeedItem{{5016, 10}, {5650, 1}}}},
		{"count 0", m5List(header, 1, []int32{5016, 0}), refused},
		{"seed 0", m5List(header, 1, []int32{0, 1}), refused},
		{"no entries", m5List(header, 0), refused},
		{"a trailing byte", append(m5List(header, 1, []int32{5016, 1}), 0), refused},
	}
	for _, tt := range tests {
		got, err := ParseRequestBuySeed(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestBuySeed", new(m3Body).d(1).d(1).b, ParseRequestBuySeed)
}

func FuzzParseRequestBuySeed(f *testing.F) {
	m5FuzzList(f, ParseRequestBuySeed,
		func(p RequestBuySeed) []BuySeedItem { return p.Items },
		8, buySeedItemSize,
		func(it BuySeedItem) bool { return it.SeedID >= 1 && it.Count >= 1 },
		m5List([]int32{1}, 1, []int32{5016, 1}))
}
