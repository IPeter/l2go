package testclient

import (
	"context"

	"github.com/IPeter/l2go/internal/packets"
)

// Game client opcodes of trades and private stores, from Mobius
// ClientPackets.
const (
	OpTradeRequest                  byte = 0x15
	OpAddTradeItem                  byte = 0x16
	OpTradeDone                     byte = 0x17
	OpAnswerTradeRequest            byte = 0x44
	OpRequestPrivateStoreManageSell byte = 0x73
	OpSetPrivateStoreListSell       byte = 0x74
	OpRequestPrivateStoreQuitSell   byte = 0x76
	OpSetPrivateStoreMsgSell        byte = 0x77
	OpRequestPrivateStoreBuy        byte = 0x79
	OpRequestPrivateStoreManageBuy  byte = 0x90
	OpSetPrivateStoreListBuy        byte = 0x91
	OpRequestPrivateStoreQuitBuy    byte = 0x93
	OpSetPrivateStoreMsgBuy         byte = 0x94
	OpRequestPrivateStoreSell       byte = 0x96
)

// Action IDs of RequestActionUse that open the setup window of a store.
const (
	ActionPrivateStoreSell int32 = 10
	ActionPrivateStoreBuy  int32 = 28
)

// StoreSellOffer is one inventory item to offer in a sell store
// (SetPrivateStoreListSell).
type StoreSellOffer struct {
	ObjectID int32
	Count    int32
	Price    int32 // per unit
}

// StoreBuyOffer is one item a buy store wants (SetPrivateStoreListBuy).
type StoreBuyOffer struct {
	ItemID int32 // the template ID
	Count  int32
	Price  int32 // per unit
}

// StorePurchase is one item to buy from a sell store
// (RequestPrivateStoreBuy).
type StorePurchase struct {
	ObjectID int32 // the item in the seller's inventory
	Count    int32
	Price    int32 // the unit price the store shows
}

// StoreSale is one item to sell to a buy store (RequestPrivateStoreSell).
type StoreSale struct {
	ObjectID int32 // the item in the player's own inventory
	ItemID   int32
	Count    int32
	Price    int32 // the unit price the store shows
}

// encodeSetPrivateStoreListSell builds SetPrivateStoreListSell: d package
// sale, d count, then per item d object ID, d count, d price.
func encodeSetPrivateStoreListSell(packageSale bool, items []StoreSellOffer) []byte {
	w := packets.NewWriter(9 + 12*len(items))
	w.WriteUint8(OpSetPrivateStoreListSell)
	w.WriteInt32(boolInt32(packageSale))
	w.WriteInt32(int32(len(items))) //nolint:gosec // a test client sends short lists
	for _, it := range items {
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.Count)
		w.WriteInt32(it.Price)
	}
	return w.Bytes()
}

// encodeSetPrivateStoreListBuy builds SetPrivateStoreListBuy: d count, then
// per item d item ID, h 0 and h 0 (which Mobius ignores), d count, d price.
func encodeSetPrivateStoreListBuy(items []StoreBuyOffer) []byte {
	w := packets.NewWriter(5 + 16*len(items))
	w.WriteUint8(OpSetPrivateStoreListBuy)
	w.WriteInt32(int32(len(items))) //nolint:gosec // a test client sends short lists
	for _, it := range items {
		w.WriteInt32(it.ItemID)
		w.WriteUint16(0)
		w.WriteUint16(0)
		w.WriteInt32(it.Count)
		w.WriteInt32(it.Price)
	}
	return w.Bytes()
}

// encodeRequestPrivateStoreBuy builds RequestPrivateStoreBuy: d store
// player, d count, then per item d object ID, d count, d price.
func encodeRequestPrivateStoreBuy(storeID int32, items []StorePurchase) []byte {
	w := packets.NewWriter(9 + 12*len(items))
	w.WriteUint8(OpRequestPrivateStoreBuy)
	w.WriteInt32(storeID)
	w.WriteInt32(int32(len(items))) //nolint:gosec // a test client sends short lists
	for _, it := range items {
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.Count)
		w.WriteInt32(it.Price)
	}
	return w.Bytes()
}

// encodeRequestPrivateStoreSell builds RequestPrivateStoreSell: d store
// player, d count, then per item d object ID, d item ID, h 0 and h 0
// (which Mobius ignores), d count, d price.
func encodeRequestPrivateStoreSell(storeID int32, items []StoreSale) []byte {
	w := packets.NewWriter(9 + 20*len(items))
	w.WriteUint8(OpRequestPrivateStoreSell)
	w.WriteInt32(storeID)
	w.WriteInt32(int32(len(items))) //nolint:gosec // a test client sends short lists
	for _, it := range items {
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteUint16(0)
		w.WriteUint16(0)
		w.WriteInt32(it.Count)
		w.WriteInt32(it.Price)
	}
	return w.Bytes()
}

// RequestTrade asks the player objectID to trade (TradeRequest). Mobius
// sends him SendTradeRequest and answers the requester with a system
// message, or refuses with one; nothing is awaited.
func (c *GameConn) RequestTrade(ctx context.Context, objectID int32) error {
	return c.send(ctx, encodeInt(OpTradeRequest, objectID))
}

// AnswerTradeRequest answers SendTradeRequest (AnswerTradeRequest) without
// waiting: accept opens the trade, unless the request expired.
func (c *GameConn) AnswerTradeRequest(ctx context.Context, accept bool) error {
	return c.send(ctx, encodeInt(OpAnswerTradeRequest, boolInt32(accept)))
}

// AcceptTrade accepts SendTradeRequest and returns the TradeStart that
// opens the window. An expired request ends with the context.
func (c *GameConn) AcceptTrade(ctx context.Context) (TradeStart, error) {
	if err := c.AnswerTradeRequest(ctx, true); err != nil {
		return TradeStart{}, err
	}
	return Await[TradeStart](ctx, c, nil)
}

// RequestAddTradeItem sends AddTradeItem without waiting, for negative
// tests. Mobius only logs tradeID.
func (c *GameConn) RequestAddTradeItem(ctx context.Context, tradeID, objectID, count int32) error {
	return c.send(ctx, encodeInts(OpAddTradeItem, tradeID, objectID, count))
}

// AddTradeItem puts count of the inventory item objectID into the open
// trade (AddTradeItem) and returns the TradeOwnAdd that shows it. Mobius
// sends TradeUpdate after it. A refused item ends with the context.
func (c *GameConn) AddTradeItem(ctx context.Context, objectID, count int32) (TradeOwnAdd, error) {
	if err := c.RequestAddTradeItem(ctx, 0, objectID, count); err != nil {
		return TradeOwnAdd{}, err
	}
	return Await(ctx, c, func(a TradeOwnAdd) bool { return a.Item.ObjectID == objectID })
}

// ConfirmTrade presses OK in the trade window (TradeDone 1) without
// waiting: the partner gets TradeOtherDone, or, when he confirmed already,
// both get the exchange and TradeDone.
func (c *GameConn) ConfirmTrade(ctx context.Context) error {
	return c.send(ctx, encodeInt(OpTradeDone, 1))
}

// CancelTrade presses Cancel in the trade window (TradeDone 0) and returns
// the server TradeDone that closes it.
func (c *GameConn) CancelTrade(ctx context.Context) (TradeDone, error) {
	if err := c.send(ctx, encodeInt(OpTradeDone, 0)); err != nil {
		return TradeDone{}, err
	}
	return Await[TradeDone](ctx, c, nil)
}

// OpenSellStoreSetup opens the setup window of a sell store with the
// action button (RequestActionUse 10) and returns it. A refusal (a system
// message and ActionFailed) ends with the context.
func (c *GameConn) OpenSellStoreSetup(ctx context.Context) (PrivateStoreManageListSell, error) {
	if err := c.RequestActionUse(ctx, ActionPrivateStoreSell, false, false); err != nil {
		return PrivateStoreManageListSell{}, err
	}
	return Await[PrivateStoreManageListSell](ctx, c, nil)
}

// OpenBuyStoreSetup opens the setup window of a buy store with the action
// button (RequestActionUse 28) and returns it.
func (c *GameConn) OpenBuyStoreSetup(ctx context.Context) (PrivateStoreManageListBuy, error) {
	if err := c.RequestActionUse(ctx, ActionPrivateStoreBuy, false, false); err != nil {
		return PrivateStoreManageListBuy{}, err
	}
	return Await[PrivateStoreManageListBuy](ctx, c, nil)
}

// RequestPrivateStoreManageSell sends RequestPrivateStoreManageSell without
// waiting.
func (c *GameConn) RequestPrivateStoreManageSell(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestPrivateStoreManageSell))
}

// RequestPrivateStoreManageBuy sends RequestPrivateStoreManageBuy without
// waiting.
func (c *GameConn) RequestPrivateStoreManageBuy(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestPrivateStoreManageBuy))
}

// QuitSellStore closes the sell store or its setup window
// (RequestPrivateStoreQuitSell) without waiting.
func (c *GameConn) QuitSellStore(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestPrivateStoreQuitSell))
}

// QuitBuyStore closes the buy store or its setup window
// (RequestPrivateStoreQuitBuy) without waiting.
func (c *GameConn) QuitBuyStore(ctx context.Context) error {
	return c.send(ctx, encodeOpcode(OpRequestPrivateStoreQuitBuy))
}

// SetSellStoreMessage sets the message over the sell store
// (SetPrivateStoreMsgSell) without waiting.
func (c *GameConn) SetSellStoreMessage(ctx context.Context, message string) error {
	return c.send(ctx, encodeString(OpSetPrivateStoreMsgSell, message))
}

// SetBuyStoreMessage sets the message over the buy store
// (SetPrivateStoreMsgBuy) without waiting.
func (c *GameConn) SetBuyStoreMessage(ctx context.Context, message string) error {
	return c.send(ctx, encodeString(OpSetPrivateStoreMsgBuy, message))
}

// SetPrivateStoreListSell sends SetPrivateStoreListSell without waiting,
// for negative tests.
func (c *GameConn) SetPrivateStoreListSell(ctx context.Context, packageSale bool, items []StoreSellOffer) error {
	return c.send(ctx, encodeSetPrivateStoreListSell(packageSale, items))
}

// OpenSellStore opens a sell store with items, from its setup window
// (SetPrivateStoreListSell), and returns the PrivateStoreMsgSell Mobius
// broadcasts for it after the UserInfo. A refused store ends with the
// context; negative tests use SetPrivateStoreListSell.
func (c *GameConn) OpenSellStore(ctx context.Context, packageSale bool, items []StoreSellOffer) (PrivateStoreMsgSell, error) {
	me := c.world.Me().ObjectID
	if err := c.SetPrivateStoreListSell(ctx, packageSale, items); err != nil {
		return PrivateStoreMsgSell{}, err
	}
	return Await(ctx, c, func(m PrivateStoreMsgSell) bool { return m.ObjectID == me })
}

// SetPrivateStoreListBuy sends SetPrivateStoreListBuy without waiting, for
// negative tests.
func (c *GameConn) SetPrivateStoreListBuy(ctx context.Context, items []StoreBuyOffer) error {
	return c.send(ctx, encodeSetPrivateStoreListBuy(items))
}

// OpenBuyStore opens a buy store with items, from its setup window
// (SetPrivateStoreListBuy), and returns the PrivateStoreMsgBuy Mobius
// broadcasts for it. A refused store ends with the context.
func (c *GameConn) OpenBuyStore(ctx context.Context, items []StoreBuyOffer) (PrivateStoreMsgBuy, error) {
	me := c.world.Me().ObjectID
	if err := c.SetPrivateStoreListBuy(ctx, items); err != nil {
		return PrivateStoreMsgBuy{}, err
	}
	return Await(ctx, c, func(m PrivateStoreMsgBuy) bool { return m.ObjectID == me })
}

// ViewSellStore clicks on the player objectID who runs a sell store
// (Action) and returns his PrivateStoreListSell. The character walks to
// him first.
func (c *GameConn) ViewSellStore(ctx context.Context, objectID int32) (PrivateStoreListSell, error) {
	if err := c.Action(ctx, objectID, false); err != nil {
		return PrivateStoreListSell{}, err
	}
	return Await(ctx, c, func(l PrivateStoreListSell) bool { return l.SellerID == objectID })
}

// ViewBuyStore clicks on the player objectID who runs a buy store (Action)
// and returns his PrivateStoreListBuy.
func (c *GameConn) ViewBuyStore(ctx context.Context, objectID int32) (PrivateStoreListBuy, error) {
	if err := c.Action(ctx, objectID, false); err != nil {
		return PrivateStoreListBuy{}, err
	}
	return Await(ctx, c, func(l PrivateStoreListBuy) bool { return l.StorePlayerID == objectID })
}

// BuyFromStore buys items from the sell store of storeID
// (RequestPrivateStoreBuy) without waiting: Mobius answers with the
// inventory changes and a system message per item, or refuses with a
// message or ActionFailed.
func (c *GameConn) BuyFromStore(ctx context.Context, storeID int32, items []StorePurchase) error {
	return c.send(ctx, encodeRequestPrivateStoreBuy(storeID, items))
}

// SellToStore sells items to the buy store of storeID
// (RequestPrivateStoreSell) without waiting, like BuyFromStore.
func (c *GameConn) SellToStore(ctx context.Context, storeID int32, items []StoreSale) error {
	return c.send(ctx, encodeRequestPrivateStoreSell(storeID, items))
}
