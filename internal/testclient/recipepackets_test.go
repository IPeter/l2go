package testclient

import (
	"errors"
	"strings"
	"testing"
)

// The Mobius recipe packets, transcribed from their writeImpl.

// mobiusRecipeBook writes the book section of RecipeBookItemList and
// RecipeShopManageList: d count, per recipe d ID, d count (1-based).
func mobiusRecipeBook(w *wire, ids []int32) {
	w.i32(int32(len(ids)))
	for i, id := range ids {
		w.i32(id)
		w.i32(int32(i + 1))
	}
}

// mobiusRecipeShopItems writes a workshop list: d count, per entry d recipe,
// d 0, d cost.
func mobiusRecipeShopItems(w *wire, items []RecipeShopItem) {
	w.i32(int32(len(items)))
	for _, it := range items {
		w.i32(it.RecipeID)
		w.i32(0)
		w.i32(it.Price)
	}
}

// bool32 is Java's writeInt(boolean).
func bool32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

func TestDecodeRecipePackets(t *testing.T) {
	book := msg(0xd6).i32(0).i32(345)
	mobiusRecipeBook(book, []int32{5, 1})
	manage := msg(0xd8).i32(testMeID).i32(1000).i32(bool32(false))
	mobiusRecipeBook(manage, []int32{1, 7})
	mobiusRecipeShopItems(manage, []RecipeShopItem{{7, 500}})
	sell := msg(0xd9).i32(testOtherID).i32(99).i32(300).i32(5000)
	mobiusRecipeShopItems(sell, []RecipeShopItem{{7, 500}, {1, 0}})
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"RecipeBookItemList", book.b, RecipeBookItemList{Dwarven: true, MaxMP: 345, RecipeIDs: []int32{5, 1}}},
		{"empty common RecipeBookItemList", msg(0xd6).i32(1).i32(3).i32(0).b, RecipeBookItemList{MaxMP: 3}},
		{"RecipeItemMakeInfo", msg(0xd7).i32(1).i32(0).i32(120).i32(345).i32(1).b,
			RecipeItemMakeInfo{RecipeID: 1, Dwarven: true, CurMP: 120, MaxMP: 345, Success: true}},
		{"RecipeShopManageList", manage.b, RecipeShopManageList{SellerID: testMeID, Adena: 1000, Dwarven: true,
			RecipeIDs: []int32{1, 7}, Items: []RecipeShopItem{{7, 500}}}},
		{"RecipeShopSellList", sell.b, RecipeShopSellList{ManufacturerID: testOtherID, CurMP: 99, MaxMP: 300, Adena: 5000,
			Items: []RecipeShopItem{{7, 500}, {1, 0}}}},
		{"RecipeShopItemInfo", msg(0xda).i32(testOtherID).i32(7).i32(99).i32(300).i32(-1).b,
			RecipeShopItemInfo{ManufacturerID: testOtherID, RecipeID: 7, CurMP: 99, MaxMP: 300}},
		{"RecipeShopMsg", msg(0xdb).i32(testOtherID).str("Arrows").b, RecipeShopMsg{ObjectID: testOtherID, Message: "Arrows"}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}
	l := tests[4].want.(RecipeShopSellList)
	if it, ok := l.Find(1); !ok || it.Price != 0 {
		t.Errorf("Find(1) = %+v, %v", it, ok)
	}
	if _, ok := l.Find(2); ok {
		t.Error("Find(2) found an entry")
	}
}

func TestDecodeRecipePacketErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		body   []byte
		substr string
	}{
		{"book type 2", msg(0xd6).i32(2).i32(0).i32(0).b, "RecipeBookItemList: book type is 2"},
		{"misnumbered recipe", msg(0xd6).i32(0).i32(0).i32(1).i32(5).i32(2).b, "recipe 5 is numbered 2"},
		{"negative count", msg(0xd6).i32(0).i32(0).i32(-1).b, "RecipeBookItemList: recipe count -1"},
		{"workshop entry not 0", msg(0xd9).i32(1).i32(0).i32(0).i32(0).i32(1).i32(7).i32(3).i32(9).b,
			"workshop entry 7 has 3"},
		{"item info last field", msg(0xda).i32(1).i32(2).i32(3).i32(4).i32(0).b, "Mobius writes 0xffffffff"},
		{"make info short", msg(0xd7).i32(1).i32(0).b, "RecipeItemMakeInfo: packets: short packet"},
		{"manage list extra byte", append(msg(0xd8).i32(1).i32(0).i32(0).i32(0).i32(0).b, 0), "RecipeShopManageList: 1 bytes after"},
	} {
		_, err := DecodeServerPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
			continue
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v does not wrap ErrMalformed", tt.name, err)
		}
	}
}
