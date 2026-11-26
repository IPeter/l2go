package craft

import (
	"slices"
	"testing"
)

func TestBuildShopList(t *testing.T) {
	known := func(id int32) bool { return id == 1 || id == 2 || id == 9 }
	const maxAdena = 2000000000
	for _, tt := range []struct {
		name    string
		entries []ShopItem
		want    []ShopItem
		refusal ShopRefusal
	}{
		{"two recipes", []ShopItem{{1, 100}, {9, 0}}, []ShopItem{{1, 100}, {9, 0}}, ShopOK},
		{"a recipe twice keeps its place and the last fee",
			[]ShopItem{{2, 5}, {1, 100}, {2, 7}}, []ShopItem{{2, 7}, {1, 100}}, ShopOK},
		{"MaxAdena exactly", []ShopItem{{1, maxAdena}}, []ShopItem{{1, maxAdena}}, ShopOK},
		{"not in the book", []ShopItem{{1, 100}, {3, 5}}, nil, ShopNotInBook},
		{"fee over MaxAdena", []ShopItem{{1, maxAdena + 1}}, nil, ShopPriceTooHigh},
		{"the first refusal counts", []ShopItem{{1, maxAdena + 1}, {3, 5}}, nil, ShopPriceTooHigh},
	} {
		got, refusal := BuildShopList(tt.entries, known, maxAdena)
		if refusal != tt.refusal || !slices.Equal(got, tt.want) {
			t.Errorf("%s: BuildShopList = %v, %d; want %v, %d", tt.name, got, refusal, tt.want, tt.refusal)
		}
	}
}

// RecipeShopManageList drops the entries of the other book and the recipes
// the crafter forgot.
func TestFitShopList(t *testing.T) {
	list := []ShopItem{{1, 10}, {500, 20}, {2, 30}, {7, 40}}
	isDwarven := func(id int32) bool { return id < 100 }
	known := func(id int32) bool { return id != 2 }
	if got := FitShopList(list, true, isDwarven, known); !slices.Equal(got, []ShopItem{{1, 10}, {7, 40}}) {
		t.Errorf("dwarven window: %v", got)
	}
	if got := FitShopList(list, false, isDwarven, known); !slices.Equal(got, []ShopItem{{500, 20}}) {
		t.Errorf("common window: %v", got)
	}
	if len(list) != 4 || list[1] != (ShopItem{500, 20}) {
		t.Errorf("the list changed: %v", list)
	}
}

func TestShopPriceAndRemove(t *testing.T) {
	list := []ShopItem{{1, 10}, {2, 0}}
	if p, ok := ShopPrice(list, 2); !ok || p != 0 {
		t.Errorf("ShopPrice(2) = %d, %v", p, ok)
	}
	if _, ok := ShopPrice(list, 3); ok {
		t.Error("ShopPrice of a recipe not offered")
	}
	if got := RemoveShopItem(list, 1); !slices.Equal(got, []ShopItem{{2, 0}}) || len(list) != 2 || list[0].RecipeID != 1 {
		t.Errorf("RemoveShopItem = %v, list %v", got, list)
	}
}

// The fee stays with the crafter once materials are used up, and goes back
// when they ran short.
func TestWorkshopOutcomeFee(t *testing.T) {
	if OrderRefunded.FeeKept() || !OrderFailed.FeeKept() || !OrderMade.FeeKept() {
		t.Error("FeeKept: want only a refund to return the fee")
	}
}
