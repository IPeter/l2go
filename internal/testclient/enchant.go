package testclient

import "context"

// Enchanting and crystallization (M11a).

// OpChooseInventoryItem is the server packet that opens the enchant window.
const OpChooseInventoryItem byte = 0x6f

// Client opcodes of enchanting and crystallization.
const (
	OpRequestEnchantItem     byte = 0x58
	OpRequestCrystallizeItem byte = 0x72
)

// ChooseInventoryItem opens the enchant window for the scroll ItemID
// (Mobius ChooseInventoryItem, 0x6F): the client answers with
// RequestEnchantItem.
type ChooseInventoryItem struct {
	ItemID int32
}

// PacketName returns the Mobius class name.
func (ChooseInventoryItem) PacketName() string { return "ChooseInventoryItem" }

func readChooseInventoryItem(f *fields) Packet { return ChooseInventoryItem{ItemID: f.i32()} }

// UseEnchantScroll uses the scroll objectID (UseItem) and returns the
// ChooseInventoryItem that opens the enchant window. Mobius refuses
// silently while the character casts, so a refused scroll ends with the
// context.
func (c *GameConn) UseEnchantScroll(ctx context.Context, objectID int32) (ChooseInventoryItem, error) {
	if err := c.UseItem(ctx, objectID); err != nil {
		return ChooseInventoryItem{}, err
	}
	return Await[ChooseInventoryItem](ctx, c, nil)
}

// EnchantItem enchants the inventory item objectID with the scroll used
// before (RequestEnchantItem) and returns the EnchantResult: the new level
// after a success, 0 otherwise. The InventoryUpdate follows it.
func (c *GameConn) EnchantItem(ctx context.Context, objectID int32) (EnchantResult, error) {
	if err := c.send(ctx, encodeInt(OpRequestEnchantItem, objectID)); err != nil {
		return EnchantResult{}, err
	}
	return Await[EnchantResult](ctx, c, nil)
}

// CrystallizeItem crystallizes count of the inventory item objectID
// (RequestCrystallizeItem) and returns its entry in the InventoryUpdate that
// follows, ItemRemoved. Mobius refuses with a system message and
// ActionFailed, or nothing at all, so a refused request ends with the
// context.
func (c *GameConn) CrystallizeItem(ctx context.Context, objectID, count int32) (ItemChange, error) {
	if err := c.send(ctx, encodeInts(OpRequestCrystallizeItem, objectID, count)); err != nil {
		return ItemChange{}, err
	}
	return c.AwaitItemChange(ctx, objectID)
}
