package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// The client packets of the recipe books, crafting and the private workshop
// (Plan 14 M11b).

// Opcodes of the recipe packets (Mobius ClientPackets). 0xB4
// (RequestRecipeShopManageList) stays unhandled, as in Mobius.
const (
	OpRequestRecipeBookOpen       = 0xac
	OpRequestRecipeBookDestroy    = 0xad
	OpRequestRecipeItemMakeInfo   = 0xae
	OpRequestRecipeItemMakeSelf   = 0xaf
	OpRequestRecipeShopMessageSet = 0xb1
	OpRequestRecipeShopListSet    = 0xb2
	OpRequestRecipeShopManageQuit = 0xb3
	OpRequestRecipeShopMakeInfo   = 0xb5
	OpRequestRecipeShopMakeItem   = 0xb6
	OpRequestRecipeShopManagePrev = 0xb7
)

// Action IDs of RequestActionUse that open the setup window of a workshop.
const (
	ActionUseDwarvenManufacture = 37
	ActionUseCommonManufacture  = 51
)

// RecipeShopMessageMaxLength is the longest workshop message Mobius takes,
// in UTF-16 code units (RequestRecipeShopMessageSet.MAX_MSG_LENGTH). A
// longer one is punished as an illegal action.
const RecipeShopMessageMaxLength = 29

// recipeShopListItemSize is the BATCH_LENGTH of RequestRecipeShopListSet: d
// recipe, d cost.
const recipeShopListItemSize = 8

// RequestRecipeBookOpen opens a recipe book.
type RequestRecipeBookOpen struct {
	// Dwarven is set when the int the client sends is 0.
	Dwarven bool
}

// ParseRequestRecipeBookOpen parses a RequestRecipeBookOpen body (Mobius
// readImpl: d book, 0 for the dwarven one).
func ParseRequestRecipeBookOpen(data []byte) (RequestRecipeBookOpen, error) {
	r := packets.NewReader(data)
	v := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestRecipeBookOpen{}, fmt.Errorf("RequestRecipeBookOpen: %w", err)
	}
	return RequestRecipeBookOpen{Dwarven: v == 0}, nil
}

// RecipeID is the body of the recipe packets that carry one recipe ID:
// RequestRecipeBookDestroy, RequestRecipeItemMakeInfo and
// RequestRecipeItemMakeSelf.
type RecipeID struct {
	ID int32
}

// ParseRecipeID parses a body of one recipe ID (d ID).
func ParseRecipeID(data []byte) (RecipeID, error) {
	r := packets.NewReader(data)
	id := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RecipeID{}, fmt.Errorf("recipe packet: %w", err)
	}
	return RecipeID{ID: id}, nil
}

// RequestRecipeShopMessageSet sets the message over a workshop.
type RequestRecipeShopMessageSet struct {
	// Message is read like SetPrivateStoreMsgSell.Message.
	Message string
}

// ParseRequestRecipeShopMessageSet parses a RequestRecipeShopMessageSet
// body (S message). It never fails.
func ParseRequestRecipeShopMessageSet(data []byte) (RequestRecipeShopMessageSet, error) {
	return RequestRecipeShopMessageSet{Message: htmlActionText(data)}, nil
}

// RecipeShopListItem is one recipe a workshop offers.
type RecipeShopListItem struct {
	RecipeID int32
	Cost     int32
}

// RequestRecipeShopListSet opens a workshop with the entries of its setup
// window.
type RequestRecipeShopListSet struct {
	// Items are the entries read, in order.
	Items []RecipeShopListItem
	// Refused is set when Mobius refuses the list (its _items is null): no
	// entries, more than MaxItemsInPacket, a count that does not match the
	// data, or a negative cost. For a negative cost, Items holds the entries
	// before it, which Mobius had read: one naming an unknown recipe makes
	// Mobius drop the packet before the cost is seen.
	Refused bool
}

// ParseRequestRecipeShopListSet parses a RequestRecipeShopListSet body
// (Mobius readImpl): d count, then per entry d recipe, d cost. Only a body
// too short for the count is an error.
func ParseRequestRecipeShopListSet(data []byte) (RequestRecipeShopListSet, error) {
	r := packets.NewReader(data)
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return RequestRecipeShopListSet{}, fmt.Errorf("RequestRecipeShopListSet: %w", err)
	}
	if !itemBatchFits(r, n, recipeShopListItemSize) {
		return RequestRecipeShopListSet{Refused: true}, nil
	}
	p := RequestRecipeShopListSet{Items: make([]RecipeShopListItem, 0, n)}
	for range n {
		it := RecipeShopListItem{RecipeID: r.ReadInt32(), Cost: r.ReadInt32()}
		if it.Cost < 0 {
			p.Refused = true
			return p, nil
		}
		p.Items = append(p.Items, it)
	}
	return p, nil
}

// RequestRecipeShopMakeInfo asks for the craft window of a workshop recipe.
type RequestRecipeShopMakeInfo struct {
	ShopID   int32 // the crafter
	RecipeID int32
}

// ParseRequestRecipeShopMakeInfo parses a RequestRecipeShopMakeInfo body
// (d crafter, d recipe).
func ParseRequestRecipeShopMakeInfo(data []byte) (RequestRecipeShopMakeInfo, error) {
	r := packets.NewReader(data)
	p := RequestRecipeShopMakeInfo{ShopID: r.ReadInt32(), RecipeID: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestRecipeShopMakeInfo{}, fmt.Errorf("RequestRecipeShopMakeInfo: %w", err)
	}
	return p, nil
}

// RequestRecipeShopMakeItem orders an item at a workshop.
type RequestRecipeShopMakeItem struct {
	ShopID   int32 // the crafter
	RecipeID int32
	// Unknown is an int Mobius reads and ignores (the fee the client saw).
	Unknown int32
}

// ParseRequestRecipeShopMakeItem parses a RequestRecipeShopMakeItem body (d
// crafter, d recipe, d unknown).
func ParseRequestRecipeShopMakeItem(data []byte) (RequestRecipeShopMakeItem, error) {
	r := packets.NewReader(data)
	p := RequestRecipeShopMakeItem{ShopID: r.ReadInt32(), RecipeID: r.ReadInt32(), Unknown: r.ReadInt32()}
	if err := r.Err(); err != nil {
		return RequestRecipeShopMakeItem{}, fmt.Errorf("RequestRecipeShopMakeItem: %w", err)
	}
	return p, nil
}
