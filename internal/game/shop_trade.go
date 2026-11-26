package game

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
)

// buyLine is one checked line of a RequestBuyItem.
type buyLine struct {
	product *data.BuyListProduct
	count   int64
	price   int64 // one item with the tax
}

// handleBuyItem handles RequestBuyItem (0x1F) in Mobius's order: flood
// protection; a refused list; the karma rule; a merchant targeted within
// interaction range (GMs need none, and then pay no tax); the list, which
// must name the merchant; each product (on the list, one of a non-stacking
// item, not free unless the buyer is a GM, enough stock, the MaxAdena
// limits); then weight, slots and adena. The items come with their
// InventoryUpdate and load; a limited product sold out meanwhile is paid
// back. StatusUpdate, ItemList (opening the window) and "The transaction is
// complete." end it.
//
// Unlike Mobius, buylist 423 (its community board shop) needs a merchant
// like any other list: there is no community board.
func (s *Server) handleBuyItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestBuyItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestBuyItem", append(p.logArgs(), "err", err)...)
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are buying too fast.")
		return
	}
	if req.Items == nil {
		p.SendMessage("You cannot buy more than 10.000 items.")
		p.SendActionFailed()
		return
	}
	if !s.gs.Economy.AltKarmaPlayerCanShop && p.Karma() > 0 {
		p.SendActionFailed()
		return
	}
	var merchant *Npc
	if !p.IsGM() {
		if merchant = s.shopMerchant(p, p.Target); merchant == nil {
			p.SendActionFailed()
			return
		}
	}
	l := s.buyList(int(req.ListID))
	if l == nil {
		s.illegalAction(p, fmt.Sprintf("sent a false BuyList list_id %d", req.ListID))
		return
	}
	var taxRate float64
	if merchant != nil {
		if !l.AllowsNpc(merchant.Template.ID) {
			p.SendActionFailed()
			return
		}
		taxRate = merchantTaxRate(merchant)
	}

	maxAdena := s.items.cfg.MaxAdena
	var subTotal, weight, slots int64
	lines := make([]buyLine, 0, len(req.Items))
	for _, want := range req.Items {
		pr, ok := l.Product(int(want.ItemID))
		var t *data.Item
		if ok {
			t, ok = s.data.ItemTemplate(pr.ItemID)
		}
		if !ok {
			s.illegalAction(p, fmt.Sprintf("sent a false BuyList list_id %d and item_id %d", req.ListID, want.ItemID))
			return
		}
		count := int64(want.Count)
		if !t.Stackable && count > 1 {
			s.illegalAction(p, "tried to purchase invalid quantity of items at the same time")
			p.SendSystemMessage(shopMsgQuantityExceeded)
			return
		}
		price := productPrice(pr)
		if price == 0 && !p.IsGM() && s.gs.Economy.OnlyGMItemsFree {
			p.SendMessage("Ohh Cheat does not work? You have a problem now!")
			s.illegalAction(p, "tried buy item for 0 adena")
			return
		}
		if pr.LimitedStock() && count > s.stockLeft(l.ID, pr) {
			p.SendActionFailed()
			return
		}
		if maxAdena/count < price {
			s.illegalAction(p, fmt.Sprintf("tried to purchase over %d adena worth of goods", maxAdena))
			return
		}
		price = taxedPrice(price, taxRate)
		subTotal += count * price
		if subTotal > maxAdena {
			s.illegalAction(p, fmt.Sprintf("tried to purchase over %d adena worth of goods", maxAdena))
			return
		}
		weight += count * int64(t.Weight)
		if p.inv.ByItemID(want.ItemID) == nil {
			slots++
		}
		lines = append(lines, buyLine{product: pr, count: count, price: price})
	}

	if !p.IsGM() && (weight > math.MaxInt32 || weight < 0 || !p.inv.ValidateWeight(weight)) {
		p.SendSystemMessage(shopMsgWeightLimit)
		p.SendActionFailed()
		return
	}
	if !p.IsGM() && (slots > math.MaxInt32 || !p.inv.ValidateCapacity(slots)) {
		p.SendSystemMessage(sysMsgInventoryFull)
		p.SendActionFailed()
		return
	}
	if subTotal < 0 || !s.ReduceAdena(p, subTotal, false) {
		p.SendSystemMessage(sysMsgNotEnoughAdena)
		p.SendActionFailed()
		return
	}

	for _, ln := range lines {
		if ln.product.LimitedStock() && !s.takeStock(l.ID, ln.product, ln.count) {
			// Sold out after the check (the request named the product twice):
			// the adena comes back.
			s.AddAdena(p, ln.price*ln.count, true)
			subTotal -= ln.price * ln.count
			continue
		}
		if _, err := p.inv.Create(int32(ln.product.ItemID), ln.count); err != nil { //nolint:gosec // item IDs fit in int32
			s.log.Error("game: cannot give a bought item; the adena goes back", append(p.logArgs(), "item", ln.product.ItemID, "count", ln.count, "err", err)...)
			s.AddAdena(p, ln.price*ln.count, true)
			continue
		}
		// PlayerInventory.addItem: InventoryUpdate and the load.
		s.itemsChanged(p, true, nil)
	}
	if merchant != nil {
		s.merchantTreasury(merchant, subTotal) // merchant_tax.go
	}
	s.finishTransaction(p)
}

// finishTransaction ends a shop transaction as Mobius does: StatusUpdate with
// the load, ItemList opening the window, "The transaction is complete.".
func (s *Server) finishTransaction(p *Player) {
	p.Send(loadStatusUpdate(p))
	s.SendItemList(p, true)
	p.SendSystemMessage(shopMsgTransactionComplete)
}

// handleSellItem handles RequestSellItem (0x1E) in Mobius's order: flood
// protection, a refused list, the karma rule, the merchant, then the
// MaxAdena check over the whole list before anything is sold, then the sale:
// each item still there, sellable and with the count p has is destroyed and
// paid at half its reference price; the rest is skipped and not paid.
// StatusUpdate, ItemList (opening the window) and "The transaction is
// complete." end it.
//
// Decision I9: sold items are destroyed (Mobius moves them into a refund
// list the Interlude client cannot show), and the sell window (list 0)
// needs a merchant too: the last NPC p talked to, within interaction range
// (Mobius checks nothing for list 0). GMs need no merchant, as in Mobius.
func (s *Server) handleSellItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestSellItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestSellItem", append(p.logArgs(), "err", err)...)
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are selling too fast.")
		return
	}
	if req.Items == nil {
		p.SendActionFailed()
		return
	}
	if !s.gs.Economy.AltKarmaPlayerCanShop && p.Karma() > 0 {
		p.SendActionFailed()
		return
	}
	if !p.IsGM() {
		var merchant *Npc
		if req.ListID == 0 {
			merchant = s.shopMerchant(p, s.Npc(p.LastNpc))
		} else {
			merchant = s.shopMerchant(p, p.Target)
		}
		if merchant == nil {
			p.SendActionFailed()
			return
		}
		if req.ListID != 0 {
			l := s.buyList(int(req.ListID))
			if l == nil {
				s.illegalAction(p, fmt.Sprintf("sent a false BuyList list_id %d", req.ListID))
				return
			}
			if !l.AllowsNpc(merchant.Template.ID) {
				p.SendActionFailed()
				return
			}
		}
	} else if req.ListID != 0 && s.buyList(int(req.ListID)) == nil {
		s.illegalAction(p, fmt.Sprintf("sent a false BuyList list_id %d", req.ListID))
		return
	}

	// The limit covers the whole list, against the room left under MaxAdena.
	maxAdena := s.items.cfg.MaxAdena
	var listPrice int64
	for _, sell := range req.Items {
		it := sellableItem(p, sell)
		if it == nil {
			continue
		}
		count := int64(sell.Count)
		price := sellPrice(it)
		listPrice += price * count
		if maxAdena/count < price || listPrice > maxAdena || listPrice > maxAdena-p.inv.Adena() {
			p.SendSystemMessage(sysMsgAdenaLimit)
			return
		}
	}

	var total int64
	var sold []int32
	wornSold := false
	for _, sell := range req.Items {
		it := sellableItem(p, sell)
		if it == nil {
			continue
		}
		count := int64(sell.Count)
		price := sellPrice(it)
		worn := it.Equipped()
		if _, err := p.inv.Destroy(it, count); err != nil {
			continue
		}
		s.forgetMana(it)
		wornSold = wornSold || worn
		sold = append(sold, it.ObjectID())
		total += price * count
	}
	// Mobius's destroyItem sends no InventoryUpdate: the ItemList below
	// shows the sale.
	s.dropInventoryUpdates(p, sold)
	if wornSold {
		// Only a forged request sells a worn item; the paperdoll changed.
		p.RefreshStats()
		p.BroadcastUserInfo()
	}
	s.refreshOverloaded(p)
	s.RequestSave(p)
	s.AddAdena(p, total, false)
	s.finishTransaction(p)
}

// sellableItem returns the item a RequestSellItem line names if p may sell
// it (Mobius checkItemManipulation and isSellable): p carries it, the count
// is no more than it has, more than one only of a stack, and merchants buy
// the template.
func sellableItem(p *Player, sell clientpackets.SellItem) *inventory.Item {
	it := p.inv.ByObjectID(sell.ObjectID)
	if it == nil || !it.Template().Sellable || augmentLocked(it) {
		return nil
	}
	count := int64(sell.Count)
	if count < 0 || (count > 1 && !it.Stackable()) || count > it.Count() {
		return nil
	}
	return it
}

// sellPrice is what a merchant pays for one of the item: half its
// reference price, rounded down.
func sellPrice(it *inventory.Item) int64 { return int64(it.Template().Price / 2) }

// dropInventoryUpdates forgets the recorded and queued InventoryUpdate
// entries of p's items with the given object IDs, for changes Mobius does
// not announce.
func (s *Server) dropInventoryUpdates(p *Player, objectIDs []int32) {
	keep := s.takeInventoryUpdates(p, nil)
	keep = slices.DeleteFunc(keep, func(c serverpackets.InventoryItem) bool { return slices.Contains(objectIDs, c.Item.ObjectID) })
	p.items.iu = slices.DeleteFunc(p.items.iu, func(c serverpackets.InventoryItem) bool { return slices.Contains(objectIDs, c.Item.ObjectID) })
	s.queueInventoryUpdate(p, keep...)
}
