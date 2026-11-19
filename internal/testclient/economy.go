package testclient

import (
	"context"
	"slices"

	"github.com/IPeter/l2go/internal/packets"
)

// Paperdoll slots of UnequipItem (Mobius Inventory.PAPERDOLL_*). Mobius
// reads the slot of RequestUnEquipItem as this index
// (BodyPart.fromPaperdollSlot); it is not the slot order of UserInfo.
const (
	PaperdollUnder   int32 = 0
	PaperdollLEar    int32 = 1
	PaperdollREar    int32 = 2
	PaperdollNeck    int32 = 3
	PaperdollLFinger int32 = 4
	PaperdollRFinger int32 = 5
	PaperdollHead    int32 = 6
	PaperdollRHand   int32 = 7
	PaperdollLHand   int32 = 8
	PaperdollGloves  int32 = 9
	PaperdollChest   int32 = 10
	PaperdollLegs    int32 = 11
	PaperdollFeet    int32 = 12
	PaperdollCloak   int32 = 13
	PaperdollFace    int32 = 14
	PaperdollHair    int32 = 15
	PaperdollHair2   int32 = 16
)

// BuyItem is one product to buy with BuyItems.
type BuyItem struct {
	ItemID int32
	Count  int32
}

// SellItem is one inventory item to sell with SellItems.
type SellItem struct {
	ObjectID int32
	ItemID   int32
	Count    int32
}

// WarehouseItem is one item to store with DepositItems (an inventory item)
// or to take back with WithdrawItems (a stored item).
type WarehouseItem struct {
	ObjectID int32
	Count    int32
}

// encodeInts builds a packet whose fields are all d: UseItem,
// RequestDropItem, RequestDestroyItem, MultiSellChoose, the skill requests
// and RequestShortcutReg.
func encodeInts(op byte, vs ...int32) []byte {
	w := packets.NewWriter(1 + 4*len(vs))
	w.WriteUint8(op)
	for _, v := range vs {
		w.WriteInt32(v)
	}
	return w.Bytes()
}

// encodeRequestBuyItem builds RequestBuyItem: d list, d count, then per
// product d item ID, d count.
func encodeRequestBuyItem(listID int32, items []BuyItem) []byte {
	w := packets.NewWriter(9 + 8*len(items))
	w.WriteUint8(OpRequestBuyItem)
	w.WriteInt32(listID)
	w.WriteInt32(int32(len(items))) //nolint:gosec // a test client sends short lists
	for _, it := range items {
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
	}
	return w.Bytes()
}

// encodeRequestSellItem builds RequestSellItem: d list, d count, then per
// item d object ID, d item ID, d count.
func encodeRequestSellItem(listID int32, items []SellItem) []byte {
	w := packets.NewWriter(9 + 12*len(items))
	w.WriteUint8(OpRequestSellItem)
	w.WriteInt32(listID)
	w.WriteInt32(int32(len(items))) //nolint:gosec // a test client sends short lists
	for _, it := range items {
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
	}
	return w.Bytes()
}

// encodeWarehouseItems builds SendWareHouseDepositList or
// SendWareHouseWithDrawList: d count, then per item d object ID, d count.
func encodeWarehouseItems(op byte, items []WarehouseItem) []byte {
	w := packets.NewWriter(5 + 8*len(items))
	w.WriteUint8(op)
	w.WriteInt32(int32(len(items))) //nolint:gosec // a test client sends short lists
	for _, it := range items {
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.Count)
	}
	return w.Bytes()
}

// UseItem uses the inventory item objectID like a double click (UseItem,
// without ctrl): it puts on or takes off equipment, drinks a potion, reads
// a scroll or charges a shot. The answers depend on the item, so nothing is
// awaited; AwaitItemChange waits for the InventoryUpdate of equipment.
func (c *GameConn) UseItem(ctx context.Context, objectID int32) error {
	return c.send(ctx, encodeInts(OpUseItem, objectID, 0))
}

// UnequipItem takes off what is worn in slot, a Paperdoll constant
// (RequestUnEquipItem). Mobius answers with UserInfo, a system message and,
// 100 ms later, the InventoryUpdate of the item; nothing is awaited.
func (c *GameConn) UnequipItem(ctx context.Context, slot int32) error {
	return c.send(ctx, encodeInt(OpRequestUnEquipItem, slot))
}

// AwaitItemChange waits for the next InventoryUpdate that lists the
// inventory item objectID and returns its entry. Mobius gathers the
// inventory changes of 100 ms into one InventoryUpdate, so it comes after
// the other answers to a request.
func (c *GameConn) AwaitItemChange(ctx context.Context, objectID int32) (ItemChange, error) {
	var change ItemChange
	_, err := Await(ctx, c, func(u InventoryUpdate) bool {
		i := slices.IndexFunc(u.Items, func(ch ItemChange) bool { return ch.ObjectID == objectID })
		if i < 0 {
			return false
		}
		change = u.Items[i]
		return true
	})
	return change, err
}

// DropItem drops count of the inventory item objectID at (x, y, z)
// (RequestDropItem) and returns the DropItem that shows it falling: dropped
// by the character, of the item's template. Part of a stack lands as a new
// object. Mobius refuses a drop with a system message or nothing at all,
// so a refused drop ends with the context.
func (c *GameConn) DropItem(ctx context.Context, objectID, count, x, y, z int32) (DropItem, error) {
	me := c.world.Me().ObjectID
	it, known := c.world.Item(objectID)
	if err := c.send(ctx, encodeInts(OpRequestDropItem, objectID, count, x, y, z)); err != nil {
		return DropItem{}, err
	}
	return Await(ctx, c, func(d DropItem) bool {
		return d.DropperID == me && (!known || d.ItemID == it.ItemID)
	})
}

// DestroyItem destroys count of the inventory item objectID
// (RequestDestroyItem) and returns its entry in the InventoryUpdate that
// follows: ItemRemoved, or ItemModified with the count left. Mobius refuses
// with a system message, so a refused request ends with the context.
func (c *GameConn) DestroyItem(ctx context.Context, objectID, count int32) (ItemChange, error) {
	if err := c.send(ctx, encodeInts(OpRequestDestroyItem, objectID, count)); err != nil {
		return ItemChange{}, err
	}
	return c.AwaitItemChange(ctx, objectID)
}

// PickUp picks up the ground item objectID with a click on it (Action,
// from the position the world model knows) and returns the character's
// GetItem for it. The character walks to the item first. Mobius refuses
// with a system message and ActionFailed (someone else's loot, a full
// inventory), so a refused pick-up ends with the context.
func (c *GameConn) PickUp(ctx context.Context, objectID int32) (GetItem, error) {
	me := c.world.Me().ObjectID
	if err := c.Action(ctx, objectID, false); err != nil {
		return GetItem{}, err
	}
	return Await(ctx, c, func(g GetItem) bool { return g.ObjectID == objectID && g.PlayerID == me })
}

// RequestItemList asks for the inventory (ItemList comes back, with the
// window shown).
func (c *GameConn) RequestItemList(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestItemList))
}

// RequestBuyItem sends RequestBuyItem for buylist listID without waiting,
// for negative tests.
func (c *GameConn) RequestBuyItem(ctx context.Context, listID int32, items []BuyItem) error {
	return c.send(ctx, encodeRequestBuyItem(listID, items))
}

// BuyItems buys from buylist listID, opened before with BuyList
// (RequestBuyItem), and returns the ItemList Mobius sends once the items
// are in the inventory. Mobius refuses a purchase with a system message
// and ActionFailed, so a refused one ends with the context; negative tests
// use RequestBuyItem.
func (c *GameConn) BuyItems(ctx context.Context, listID int32, items []BuyItem) (ItemList, error) {
	if err := c.RequestBuyItem(ctx, listID, items); err != nil {
		return ItemList{}, err
	}
	return Await[ItemList](ctx, c, nil)
}

// RequestSellItem sends RequestSellItem for buylist listID without
// waiting, for negative tests.
func (c *GameConn) RequestSellItem(ctx context.Context, listID int32, items []SellItem) error {
	return c.send(ctx, encodeRequestSellItem(listID, items))
}

// SellItems sells inventory items to the merchant whose SellList is open
// (RequestSellItem) and returns the ItemList Mobius sends after the sale.
// listID is the SellList's ListID, 0 in Mobius; any other value must name
// a buylist of the merchant. A refused sale ends with the context; negative
// tests use RequestSellItem.
func (c *GameConn) SellItems(ctx context.Context, listID int32, items []SellItem) (ItemList, error) {
	if err := c.RequestSellItem(ctx, listID, items); err != nil {
		return ItemList{}, err
	}
	return Await[ItemList](ctx, c, nil)
}

// AwaitMultiSellList waits for the pages of the next multisell list, up to
// the one with Finished set, and returns them as one list: Entries holds
// the entries of every page in order and Page the number of the last page.
// Mobius sends a page per 40 entries.
func (c *GameConn) AwaitMultiSellList(ctx context.Context) (MultiSellList, error) {
	var all MultiSellList
	for {
		p, err := Await[MultiSellList](ctx, c, nil)
		if err != nil {
			return all, err
		}
		all.ListID, all.Page, all.PageSize = p.ListID, p.Page, p.PageSize
		all.Entries = append(all.Entries, p.Entries...)
		if p.Finished {
			all.Finished = true
			return all, nil
		}
	}
}

// MultiSellChoose exchanges amount times the entry entryID of the
// multisell list listID (MultiSellChoose) and returns the ItemList Mobius
// sends after the exchange. A refused exchange ends with the context.
func (c *GameConn) MultiSellChoose(ctx context.Context, listID, entryID, amount int32) (ItemList, error) {
	if err := c.send(ctx, encodeInts(OpMultiSellChoose, listID, entryID, amount)); err != nil {
		return ItemList{}, err
	}
	return Await[ItemList](ctx, c, nil)
}

// DepositItems stores inventory items in the warehouse whose
// WareHouseDepositList is open (SendWareHouseDepositList) and returns the
// InventoryUpdate that takes them out of the inventory: the first one that
// lists one of them. A refused deposit ends with the context.
func (c *GameConn) DepositItems(ctx context.Context, items []WarehouseItem) (InventoryUpdate, error) {
	if err := c.send(ctx, encodeWarehouseItems(OpSendWareHouseDepositList, items)); err != nil {
		return InventoryUpdate{}, err
	}
	return Await(ctx, c, func(u InventoryUpdate) bool {
		return slices.ContainsFunc(u.Items, func(ch ItemChange) bool {
			return slices.ContainsFunc(items, func(it WarehouseItem) bool { return it.ObjectID == ch.ObjectID })
		})
	})
}

// WithdrawItems takes stored items from the warehouse whose
// WareHouseWithdrawalList is open (SendWareHouseWithDrawList) and returns
// the next InventoryUpdate, which adds them. A withdrawn item can join a
// stack of the inventory or get a new object ID, so the update is not
// matched by object ID: Drain first when an older one may still be queued.
// A refused withdrawal ends with the context.
func (c *GameConn) WithdrawItems(ctx context.Context, items []WarehouseItem) (InventoryUpdate, error) {
	if err := c.send(ctx, encodeWarehouseItems(OpSendWareHouseWithDrawList, items)); err != nil {
		return InventoryUpdate{}, err
	}
	return Await[InventoryUpdate](ctx, c, nil)
}
