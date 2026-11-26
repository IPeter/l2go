package clientpackets

import "testing"

// Mobius RequestRecipeBookOpen.readImpl: _isDwarvenCraft = readInt() == 0.
func TestParseRequestRecipeBookOpen(t *testing.T) {
	for _, tt := range []struct {
		v    int32
		want bool
	}{{0, true}, {1, false}, {7, false}, {-1, false}} {
		got, err := ParseRequestRecipeBookOpen(new(m3Body).d(tt.v).b)
		m3Parsed(t, "RequestRecipeBookOpen", got, err, RequestRecipeBookOpen{Dwarven: tt.want})
	}
	m3Truncations(t, "RequestRecipeBookOpen", new(m3Body).d(0).b, ParseRequestRecipeBookOpen)
}

// RequestRecipeBookDestroy, RequestRecipeItemMakeInfo and
// RequestRecipeItemMakeSelf read one int; extra bytes are ignored.
func TestParseRecipeID(t *testing.T) {
	full := new(m3Body).d(871).b
	got, err := ParseRecipeID(full)
	m3Parsed(t, "recipe ID", got, err, RecipeID{ID: 871})
	got, err = ParseRecipeID(append(full, 1, 2))
	m3Parsed(t, "recipe ID with extra bytes", got, err, RecipeID{ID: 871})
	m3Truncations(t, "recipe ID", full, ParseRecipeID)
}

// RequestRecipeShopMessageSet.readImpl: S name, read with readString.
func TestParseRequestRecipeShopMessageSet(t *testing.T) {
	for _, tt := range []struct {
		name string
		body []byte
		want string
	}{
		{"a message", new(m3Body).s("Arrows 5k").b, "Arrows 5k"},
		{"empty body", nil, ""},
		{"no terminator", new(m3Body).s("WTS").b[:6], "WTS"},
	} {
		got, err := ParseRequestRecipeShopMessageSet(tt.body)
		m3Parsed(t, "RequestRecipeShopMessageSet "+tt.name, got, err, RequestRecipeShopMessageSet{Message: tt.want})
	}
}

// RequestRecipeShopListSet.readImpl: d count, which must be 1 to
// MAX_ITEM_IN_PACKET and match the rest exactly (8 bytes per entry); per
// entry d recipe, d cost; a negative cost sets _items to null.
func TestParseRequestRecipeShopListSet(t *testing.T) {
	two := new(m3Body).d(2).d(1).d(500).d(7).d(0).b
	got, err := ParseRequestRecipeShopListSet(two)
	m3Parsed(t, "two entries", got, err, RequestRecipeShopListSet{Items: []RecipeShopListItem{{1, 500}, {7, 0}}})

	for _, tt := range []struct {
		name string
		body []byte
		want RequestRecipeShopListSet
	}{
		{"no entries", new(m3Body).d(0).b, RequestRecipeShopListSet{Refused: true}},
		{"negative count", new(m3Body).d(-1).d(1).d(2).b, RequestRecipeShopListSet{Refused: true}},
		{"count above the data", new(m3Body).d(3).d(1).d(500).d(7).d(0).b, RequestRecipeShopListSet{Refused: true}},
		{"extra bytes", append(two, 0), RequestRecipeShopListSet{Refused: true}},
		{"too many", new(m3Body).d(MaxItemsInPacket + 1).b, RequestRecipeShopListSet{Refused: true}},
		{"negative cost first", new(m3Body).d(2).d(1).d(-5).d(7).d(0).b, RequestRecipeShopListSet{Items: []RecipeShopListItem{}, Refused: true}},
		{"negative cost second", new(m3Body).d(2).d(1).d(5).d(7).d(-1).b,
			RequestRecipeShopListSet{Items: []RecipeShopListItem{{1, 5}}, Refused: true}},
	} {
		got, err := ParseRequestRecipeShopListSet(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestRecipeShopListSet header", new(m3Body).d(1).b, ParseRequestRecipeShopListSet)
}

// RequestRecipeShopMakeInfo.readImpl: d player, d recipe.
func TestParseRequestRecipeShopMakeInfo(t *testing.T) {
	full := new(m3Body).d(0x10000002).d(7).b
	got, err := ParseRequestRecipeShopMakeInfo(full)
	m3Parsed(t, "RequestRecipeShopMakeInfo", got, err, RequestRecipeShopMakeInfo{ShopID: 0x10000002, RecipeID: 7})
	m3Truncations(t, "RequestRecipeShopMakeInfo", full, ParseRequestRecipeShopMakeInfo)
}

// RequestRecipeShopMakeItem.readImpl: d player, d recipe, d unknown.
func TestParseRequestRecipeShopMakeItem(t *testing.T) {
	full := new(m3Body).d(0x10000002).d(7).d(1000).b
	got, err := ParseRequestRecipeShopMakeItem(full)
	m3Parsed(t, "RequestRecipeShopMakeItem", got, err, RequestRecipeShopMakeItem{ShopID: 0x10000002, RecipeID: 7, Unknown: 1000})
	m3Truncations(t, "RequestRecipeShopMakeItem", full, ParseRequestRecipeShopMakeItem)
}

func TestRecipeOpcodes(t *testing.T) {
	for _, tt := range []struct {
		name string
		got  int
		want int
	}{
		{"RequestRecipeBookOpen", OpRequestRecipeBookOpen, 0xac},
		{"RequestRecipeBookDestroy", OpRequestRecipeBookDestroy, 0xad},
		{"RequestRecipeItemMakeInfo", OpRequestRecipeItemMakeInfo, 0xae},
		{"RequestRecipeItemMakeSelf", OpRequestRecipeItemMakeSelf, 0xaf},
		{"RequestRecipeShopMessageSet", OpRequestRecipeShopMessageSet, 0xb1},
		{"RequestRecipeShopListSet", OpRequestRecipeShopListSet, 0xb2},
		{"RequestRecipeShopManageQuit", OpRequestRecipeShopManageQuit, 0xb3},
		{"RequestRecipeShopMakeInfo", OpRequestRecipeShopMakeInfo, 0xb5},
		{"RequestRecipeShopMakeItem", OpRequestRecipeShopMakeItem, 0xb6},
		{"RequestRecipeShopManagePrev", OpRequestRecipeShopManagePrev, 0xb7},
		{"action 37", ActionUseDwarvenManufacture, 37},
		{"action 51", ActionUseCommonManufacture, 51},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}
