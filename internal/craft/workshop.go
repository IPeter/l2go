package craft

import "slices"

// The private workshop (Mobius ManufactureItem, RequestRecipeShopListSet and
// RecipeShopManageList): a crafter sits and offers recipes of his books, each
// for a fee the customer pays before the item is made. The fee stays with
// the crafter when the craft fails and goes back when the customer's
// materials run short after paying.

// ShopItem is a recipe a workshop offers and its fee (Mobius
// ManufactureItem).
type ShopItem struct {
	RecipeID int32
	Price    int64
}

// ShopRefusal is why a workshop list is refused (Mobius
// RequestRecipeShopListSet.runImpl); both are punished as illegal actions.
type ShopRefusal int

// The answers of BuildShopList.
const (
	ShopOK ShopRefusal = iota
	// ShopNotInBook: an entry names a recipe neither book holds.
	ShopNotInBook
	// ShopPriceTooHigh: a fee is above MaxAdena.
	ShopPriceTooHigh
)

// BuildShopList builds the list a workshop opens with from the client's
// entries, in Mobius's order: each entry must name a recipe the crafter
// knows and a fee of at most maxAdena, else the list is refused. An entry
// for a recipe listed before replaces its fee and keeps its place (Mobius
// keeps the list in a LinkedHashMap by recipe ID).
func BuildShopList(entries []ShopItem, known func(recipeID int32) bool, maxAdena int64) ([]ShopItem, ShopRefusal) {
	var out []ShopItem
	for _, e := range entries {
		if !known(e.RecipeID) {
			return nil, ShopNotInBook
		}
		if e.Price > maxAdena {
			return nil, ShopPriceTooHigh
		}
		out = putShopItem(out, e)
	}
	return out, ShopOK
}

// putShopItem is LinkedHashMap.put: a recipe in the list gets the new fee
// in its place, a new one goes to the end.
func putShopItem(list []ShopItem, e ShopItem) []ShopItem {
	if i := slices.IndexFunc(list, func(x ShopItem) bool { return x.RecipeID == e.RecipeID }); i >= 0 {
		list[i] = e
		return list
	}
	return append(list, e)
}

// FitShopList is what opening the setup window does to a workshop's list
// (Mobius RecipeShopManageList's constructor): the entries of the other book
// and those of recipes the crafter no longer knows leave it. isDwarven tells
// a recipe's book. The list is not changed; the result is a new slice.
func FitShopList(list []ShopItem, dwarven bool, isDwarven func(recipeID int32) bool, known func(recipeID int32) bool) []ShopItem {
	out := make([]ShopItem, 0, len(list))
	for _, e := range list {
		if isDwarven(e.RecipeID) == dwarven && known(e.RecipeID) {
			out = append(out, e)
		}
	}
	return out
}

// ShopPrice returns the fee of a recipe the workshop offers.
func ShopPrice(list []ShopItem, recipeID int32) (int64, bool) {
	i := slices.IndexFunc(list, func(x ShopItem) bool { return x.RecipeID == recipeID })
	if i < 0 {
		return 0, false
	}
	return list[i].Price, true
}

// RemoveShopItem returns the list without the recipe (Mobius
// RequestRecipeBookDestroy, getManufactureItems().remove); list is not
// changed.
func RemoveShopItem(list []ShopItem, recipeID int32) []ShopItem {
	return slices.DeleteFunc(slices.Clone(list), func(x ShopItem) bool { return x.RecipeID == recipeID })
}

// WorkshopOutcome is how a workshop order ends once the customer paid
// (Mobius finishCrafting): who keeps the fee follows from it.
type WorkshopOutcome int

// The outcomes of a paid order.
const (
	// OrderRefunded: the customer's materials ran short after paying; the
	// fee goes back and nothing is made.
	OrderRefunded WorkshopOutcome = iota + 1
	// OrderFailed: the materials are used up, nothing is made, the crafter
	// keeps the fee.
	OrderFailed
	// OrderMade: the materials are used up and the customer gets the item;
	// the crafter keeps the fee.
	OrderMade
)

// FeeKept reports whether the crafter keeps the fee.
func (o WorkshopOutcome) FeeKept() bool { return o == OrderFailed || o == OrderMade }
