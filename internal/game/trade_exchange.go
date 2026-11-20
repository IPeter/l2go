package game

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
)

// The item exchange between two players (Plan 12 decision T5): the checks
// and the moves Mobius TradeList shares between the direct trade and the
// private stores (calcItemsWeight, countItemsSlots, validate,
// TransferItems), and the end of an exchange: the packets and the joint
// save. All are loop-only.

// exchangeItem is count of an inventory item that moves to another player
// (Mobius TradeItem without the price).
type exchangeItem struct {
	objectID int32
	template *data.Item
	count    int64
}

// exchangeWeight returns the weight of the items (Mobius
// TradeList.calcItemsWeight): template weight × count, capped at the int
// range.
func exchangeWeight(items []exchangeItem) int64 {
	var w int64
	for _, it := range items {
		w += int64(it.template.Weight) * it.count
		if w >= math.MaxInt32 {
			return math.MaxInt32
		}
	}
	return w
}

// exchangeSlots returns the inventory slots the items take at receiver
// (Mobius TradeList.countItemsSlots): an item that does not stack takes one
// per count, a stackable one takes one unless receiver holds it already.
func exchangeSlots(items []exchangeItem, receiver *Player) int64 {
	var slots int64
	for _, it := range items {
		switch {
		case !it.template.Stackable:
			slots += it.count
		case receiver.inv.ByItemID(int32(it.template.ID)) == nil: //nolint:gosec // item IDs fit in int32
			slots++
		}
	}
	return slots
}

// exchangeValid reports whether owner still has every item in the counts
// given (Mobius TradeList.validate: checkItemManipulation for each, and an
// owner who is still in the game).
func (s *Server) exchangeValid(owner *Player, items []exchangeItem) bool {
	if !s.exchangeOnline(owner) {
		return false
	}
	for _, it := range items {
		if item := depositCheckItem(owner, it.objectID, it.count); item == nil || item.Count() < 1 {
			s.log.Warn("game: an item of the exchange is no longer valid", append(owner.logArgs(), "itemObjectId", it.objectID, "count", it.count)...)
			return false
		}
	}
	return true
}

// exchangeOnline reports whether p is the player in the game under its
// object ID (Mobius World.getPlayer(objectId) != null).
func (s *Server) exchangeOnline(p *Player) bool {
	if p == nil || !p.InGame() {
		return false
	}
	q, ok := s.Player(p.ID())
	return ok && q == p
}

// exchangeFits checks the weight and the slots of a two-way exchange in
// Mobius's order (TradeList.doExchange): a gets bItems, b gets aItems. A
// failed check sends "You have exceeded the weight limit." or "Your
// inventory is full." to b, then a, and returns false. The weights and slots
// are those before anything moves, as in Mobius.
func (s *Server) exchangeFits(a *Player, aItems []exchangeItem, b *Player, bItems []exchangeItem) bool {
	if !a.inv.ValidateWeight(exchangeWeight(bItems)) || !b.inv.ValidateWeight(exchangeWeight(aItems)) {
		b.SendSystemMessage(exchangeMsgWeightLimit)
		a.SendSystemMessage(exchangeMsgWeightLimit)
		return false
	}
	if !a.inv.ValidateCapacity(exchangeSlots(bItems, a)) || !b.inv.ValidateCapacity(exchangeSlots(aItems, b)) {
		b.SendSystemMessage(sysMsgInventoryFull)
		a.SendSystemMessage(sysMsgInventoryFull)
		return false
	}
	return true
}

// transferExchangeItems moves the items from giver's inventory to receiver's
// (Mobius TradeList.TransferItems): a whole item that does not stack keeps
// its object ID, a stack goes onto receiver's stack or into a new one. An
// item that is gone stops the moves and the result is false. A worn item
// comes off first, and the giver's UserInfo and CharInfo follow at the end
// of the tick. The InventoryUpdates are recorded by the inventories and sent
// by finishExchange.
func (s *Server) transferExchangeItems(giver, receiver *Player, items []exchangeItem) bool {
	for _, it := range items {
		old := giver.inv.ByObjectID(it.objectID)
		if old == nil {
			return false
		}
		worn := old.Equipped()
		if _, err := giver.inv.Transfer(it.objectID, it.count, &receiver.inv.Container); err != nil {
			s.log.Error("game: an item of the exchange did not move", append(giver.logArgs(),
				"to", receiver.ObjectID(), "itemObjectId", it.objectID, "count", it.count, "err", err)...)
			return false
		}
		if worn {
			s.forgetMana(old)
			giver.RefreshStats()
			giver.BroadcastUserInfo()
		}
	}
	return true
}

// finishExchange sends each player what the exchange changed (Mobius
// TradeList.doExchange after the moves): the InventoryUpdate (100 ms later,
// merged), the full ItemList (the client drops part of the update when the
// trade window closes, TradeList.java 611), StatusUpdate with the load and
// the weight penalty. Then every player of the exchange is saved in one
// transaction (decision T5): a crash can lose all of the exchange or none of
// it, never one side.
func (s *Server) finishExchange(players ...*Player) {
	for _, p := range players {
		s.queueInventoryUpdate(p, s.takeInventoryUpdates(p, nil)...)
		s.SendItemList(p, false)
		if p.InGame() {
			p.Send(loadStatusUpdate(p))
		}
		s.refreshOverloaded(p)
	}
	s.saveExchange(players)
}

// saveExchange saves the players of an exchange in one transaction
// (decision T5, Server.SaveJoint). A save that fails for good disconnects
// them, so that they reload what the database has: both sides of the
// exchange, or neither.
func (s *Server) saveExchange(players []*Player) {
	s.SaveJoint(players, storage.Batch{})
}

// exchangeItemOf returns the exchange entry for count of it.
func exchangeItemOf(it *inventory.Item, count int64) exchangeItem {
	return exchangeItem{objectID: it.ObjectID(), template: it.Template(), count: count}
}
