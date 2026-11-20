package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// The server packets of a direct trade between two players.

// Opcodes of the trade packets (Mobius ServerPackets).
const (
	OpTradeStart       = 0x1e
	OpTradeOwnAdd      = 0x20
	OpTradeOtherAdd    = 0x21
	OpTradeDone        = 0x22
	OpSendTradeRequest = 0x5e
	OpTradeUpdate      = 0x74
	OpTradeOtherDone   = 0x7c // Mobius TRADE_PRESS_OTHER_OK
)

// Results of TradeDone.
const (
	// TradeDoneCanceled closes the trade window without a trade
	// (Player.onTradeCancel).
	TradeDoneCanceled int32 = 0
	// TradeDoneFinished closes it after the exchange, or after a failed
	// weight or slot check (Player.onTradeFinish).
	TradeDoneFinished int32 = 1
)

// Entry sizes of the trade lists.
const (
	tradeStartItemSize  = 28
	tradeUpdateItemSize = 30
	tradeAddSize        = 31 // the whole TradeOwnAdd or TradeOtherAdd packet
)

// Change types of TradeUpdate.
const (
	tradeUpdateRemove = 2
	tradeUpdateModify = 3
)

// TradeStart opens the trade window.
type TradeStart struct {
	PartnerID int32
	// Items are the inventory items the player can trade (Mobius
	// getAvailableItems(true, GM restriction, false): not equipped, not
	// quest items, tradeable unless a GM may trade anything). Mobius sends
	// the template ID as the item ID and leaves out CustomType1, Equipped,
	// AugmentationID and Mana.
	Items []ItemInfo
}

// NewTradeStartPacket builds TradeStart (Mobius TradeStart.writeImpl).
// Mobius writes nothing when the player has no trade partner; the caller
// does not send it then.
func NewTradeStartPacket(s TradeStart) []byte {
	w := packets.NewWriter(7 + len(s.Items)*tradeStartItemSize)
	w.WriteUint8(OpTradeStart)
	w.WriteInt32(s.PartnerID)
	putCount16(w, len(s.Items))
	for _, it := range s.Items {
		putInt16(w, it.Type1)
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
		putInt16(w, it.Type2)
		w.WriteUint16(0)
		w.WriteInt32(it.BodyPart)
		putInt16(w, it.Enchant)
		w.WriteUint16(0)
		putInt16(w, it.CustomType2)
	}
	return w.Bytes()
}

// NewTradeOwnAddPacket builds TradeOwnAdd (Mobius TradeOwnAdd.writeImpl):
// the item the player put into the trade, shown on his side. it.Count is
// the count added; Equipped, AugmentationID and Mana are not sent.
func NewTradeOwnAddPacket(it ItemInfo) []byte {
	return newTradeAddPacket(OpTradeOwnAdd, it)
}

// NewTradeOtherAddPacket builds TradeOtherAdd (Mobius
// TradeOtherAdd.writeImpl): the item the partner put into the trade, with
// the same fields as TradeOwnAdd.
func NewTradeOtherAddPacket(it ItemInfo) []byte {
	return newTradeAddPacket(OpTradeOtherAdd, it)
}

func newTradeAddPacket(op uint8, it ItemInfo) []byte {
	w := packets.NewWriter(tradeAddSize)
	w.WriteUint8(op)
	w.WriteUint16(1) // one item
	putInt16(w, it.Type1)
	w.WriteInt32(it.ObjectID)
	w.WriteInt32(it.ItemID)
	w.WriteInt32(it.Count)
	putInt16(w, it.Type2)
	putInt16(w, it.CustomType1)
	w.WriteInt32(it.BodyPart)
	putInt16(w, it.Enchant)
	w.WriteUint16(0)
	putInt16(w, it.CustomType2)
	return w.Bytes()
}

// NewTradeDonePacket builds TradeDone (Mobius TradeDone.writeImpl): it
// closes the trade window. result is TradeDoneCanceled or
// TradeDoneFinished.
func NewTradeDonePacket(result int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpTradeDone)
	w.WriteInt32(result)
	return w.Bytes()
}

// NewSendTradeRequestPacket builds SendTradeRequest (Mobius
// SendTradeRequest.writeImpl): the question "senderID asks you to trade".
func NewSendTradeRequestPacket(senderID int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpSendTradeRequest)
	w.WriteInt32(senderID)
	return w.Bytes()
}

// NewTradeOtherDonePacket builds TradeOtherDone (Mobius
// TradeOtherDone.writeImpl, TRADE_PRESS_OTHER_OK): the partner confirmed.
// It has no fields.
func NewTradeOtherDonePacket() []byte {
	return []byte{OpTradeOtherDone}
}

// TradeUpdateItem is one entry of the player's own trade list in
// TradeUpdate.
type TradeUpdateItem struct {
	// Item is the item in the trade. Item.Count is the count in the trade
	// list; Mobius sends the template ID as the item ID and leaves out
	// CustomType1, Equipped, CustomType2, AugmentationID and Mana.
	Item ItemInfo
	// InventoryCount is the count of the item's object in the inventory, 0
	// if it is not there.
	InventoryCount int32
	// Stackable is the template's stackable flag.
	Stackable bool
}

// NewTradeUpdatePacket builds TradeUpdate (Mobius TradeUpdate.writeImpl):
// it updates the inventory side of the player's trade window for every item
// of his trade list. An entry shows what is left: a stackable item with
// some left is modified (3) to that count; otherwise it is removed (2),
// with a count of 1 when nothing is left.
func NewTradeUpdatePacket(items []TradeUpdateItem) []byte {
	w := packets.NewWriter(3 + len(items)*tradeUpdateItemSize)
	w.WriteUint8(OpTradeUpdate)
	putCount16(w, len(items))
	for _, u := range items {
		it := u.Item
		left := u.InventoryCount - it.Count
		change := uint16(tradeUpdateRemove)
		if left == 0 {
			left = 1
		} else if u.Stackable {
			change = tradeUpdateModify
		}
		w.WriteUint16(change)
		putInt16(w, it.Type1)
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(left)
		putInt16(w, it.Type2)
		w.WriteUint16(0)
		w.WriteInt32(it.BodyPart)
		putInt16(w, it.Enchant)
		w.WriteUint16(0)
		w.WriteUint16(0)
	}
	return w.Bytes()
}
