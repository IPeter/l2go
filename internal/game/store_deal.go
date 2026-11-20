package game

import (
	"cmp"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/world"
)

// The purchases at private stores (Mobius RequestPrivateStoreBuy,
// RequestPrivateStoreSell and TradeList.privateStoreBuy/privateStoreSell).
// Each purchase ends with one joint save of both players (decision T5).

// Results of storeBuy (Mobius privateStoreBuy).
const (
	storeDealDone     = iota // the items moved
	storeDealCanceled        // nothing moved: adena, weight, slots, or a locked list
	storeDealFailed          // the list or the request was invalid
)

// storeRequest is an entry of a purchase as the purchase changes it (Mobius
// RequestTrade).
type storeRequest struct {
	objectID int32
	itemID   int32
	count    int64
	price    int64
}

// storeStoreOwner returns the store owner a purchase of p names, when p may
// deal with him (Mobius RequestPrivateStoreBuy and RequestPrivateStoreSell):
// a player in the game, within 250 of p in 3D, in p's instance. Otherwise
// nil, and p gets nothing.
func (s *Server) storeStoreOwner(p *Player, id int32) *Player {
	owner, ok := s.Player(world.ObjectID(id))
	if !ok || !p.Position().InRange3D(owner.Position(), s.npcInteractionRange()) || owner.InstanceID() != p.InstanceID() {
		return nil
	}
	return owner
}

// handleStoreBuy handles RequestPrivateStoreBuy (0x79) in Mobius's order: a
// refused list gets ActionFailed; the 1 s flood guard; the store owner
// (storeStoreOwner) with a sell store or a package sale; buying fewer
// entries than a package holds is punished; then storeBuy. A purchase that
// did not go through gets ActionFailed; a store left empty closes and shows
// it.
func (s *Server) handleStoreBuy(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPrivateStoreBuy(body)
	if err != nil {
		s.log.Debug("game: bad RequestPrivateStoreBuy", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Items == nil {
		p.SendActionFailed()
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are buying items too fast.")
		return
	}
	owner := s.storeStoreOwner(p, req.StorePlayerID)
	if owner == nil {
		return
	}
	t := owner.store.typ
	if t != StoreSell && t != StorePackageSell {
		return
	}
	l := &owner.store.sell
	if t == StorePackageSell && len(l.items) > len(req.Items) {
		s.illegalAction(p, "a partial purchase of a package sale")
		return
	}
	items := make([]storeRequest, len(req.Items))
	for i, ri := range req.Items {
		items[i] = storeRequest{objectID: ri.ObjectID, count: int64(ri.Count), price: int64(ri.Price)}
	}
	// Mobius keeps the entries in a HashSet whose hash is the object ID.
	slices.SortStableFunc(items, func(a, b storeRequest) int { return cmp.Compare(a.objectID, b.objectID) })
	if res := s.storeBuy(p, owner, l, items); res != storeDealDone {
		p.SendActionFailed()
		if res == storeDealFailed {
			s.log.Warn("game: private store purchase failed: invalid list or request", append(p.logArgs(), "store", owner.Name())...)
		}
		return
	}
	s.storeCloseIfEmpty(owner, l)
}

// storeBuy is Mobius TradeList.privateStoreBuy: buyer buys items from
// owner's sell list l. Every entry must name a list item at its price (the
// count is cut to the list's), else it is skipped, or, in a package sale,
// the buyer is punished. A price or total over MaxAdena, or a list item the
// owner no longer has, locks the list. Then the buyer's adena, weight and
// slots are checked; the adena moves, then the items one by one, each with
// its message to both, and the list shrinks. Unlike Mobius, whose request
// entries carry no item ID so that its weight and slot checks never count
// anything, the weight and slots of the items are checked.
func (s *Server) storeBuy(buyer, owner *Player, l *storeList, items []storeRequest) int {
	if l.locked {
		return storeDealCanceled
	}
	if !s.storeListValid(owner, l) {
		l.locked = true
		return storeDealCanceled
	}
	if !buyer.InGame() || !owner.InGame() {
		return storeDealCanceled
	}
	maxAdena := s.items.cfg.MaxAdena
	var total, weight, slots int64
	for i := range items {
		r := &items[i]
		found := false
		for _, ti := range l.items {
			if ti.objectID == r.objectID {
				if ti.price == r.price {
					r.count = min(r.count, ti.count)
					found = true
				}
				break
			}
		}
		if !found {
			if l.packaged {
				s.illegalAction(buyer, "a purchase of part of a package sale")
				return storeDealFailed
			}
			r.count = 0
			continue
		}
		if maxAdena/r.count < r.price {
			l.locked = true
			return storeDealCanceled
		}
		if total += r.count * r.price; total > maxAdena {
			l.locked = true
			return storeDealCanceled
		}
		it := storeCheckItem(owner, r.objectID, r.count)
		if it == nil || !it.Template().Tradable {
			l.locked = true
			return storeDealFailed
		}
		t := it.Template()
		weight += r.count * int64(t.Weight)
		switch {
		case !t.Stackable:
			slots += r.count
		case buyer.inv.ByItemID(it.ItemID()) == nil:
			slots++
		}
	}
	switch {
	case total > buyer.inv.Adena():
		buyer.SendSystemMessage(sysMsgNotEnoughAdena)
		return storeDealCanceled
	case !buyer.inv.ValidateWeight(weight):
		buyer.SendSystemMessage(exchangeMsgWeightLimit)
		return storeDealCanceled
	case !buyer.inv.ValidateCapacity(slots):
		buyer.SendSystemMessage(sysMsgInventoryFull)
		return storeDealCanceled
	}
	if !s.storeMoveAdena(buyer, owner, total) {
		buyer.SendSystemMessage(sysMsgNotEnoughAdena)
		return storeDealCanceled
	}
	res := storeDealDone
	for _, r := range items {
		if r.count == 0 {
			continue
		}
		if storeCheckItem(owner, r.objectID, r.count) == nil {
			l.locked = true
			res = storeDealFailed
			break
		}
		moved, err := owner.inv.Transfer(r.objectID, r.count, &buyer.inv.Container)
		if err != nil {
			s.log.Error("game: a private store item did not move", append(owner.logArgs(), "to", buyer.ObjectID(), "itemObjectId", r.objectID, "count", r.count, "err", err)...)
			res = storeDealFailed
			break
		}
		l.remove(func(ti *storeItem) bool { return ti.objectID == r.objectID }, r.count)
		s.storeDealMessages(owner, buyer, buyer, owner, moved, r.count)
	}
	s.finishStoreDeal(owner, buyer)
	return res
}

// handleStoreSell handles RequestPrivateStoreSell (0x96) in Mobius's order:
// a refused list gets ActionFailed; the 1 s flood guard; the store owner
// (storeStoreOwner) with a buy store; then storeSell. A sale that did not go
// through gets ActionFailed; a store left empty closes and shows it.
func (s *Server) handleStoreSell(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPrivateStoreSell(body)
	if err != nil {
		s.log.Debug("game: bad RequestPrivateStoreSell", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Items == nil {
		p.SendActionFailed()
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are selling items too fast.")
		return
	}
	owner := s.storeStoreOwner(p, req.StorePlayerID)
	if owner == nil || owner.store.typ != StoreBuy {
		return
	}
	l := &owner.store.buy
	items := make([]storeRequest, len(req.Items))
	for i, ri := range req.Items {
		items[i] = storeRequest{objectID: ri.ObjectID, itemID: ri.ItemID, count: int64(ri.Count), price: int64(ri.Price)}
	}
	if !s.storeSell(p, owner, l, items) {
		p.SendActionFailed()
		s.log.Warn("game: private store sale failed: invalid list or request", append(p.logArgs(), "store", owner.Name())...)
		return
	}
	s.storeCloseIfEmpty(owner, l)
}

// storeSell is Mobius TradeList.privateStoreSell: seller sells items to
// owner's buy list l, in the order he sent them. An entry must name a list
// template at its price (the count is cut to the list's), else it is
// skipped; a price or total over MaxAdena locks the list and ends the sale;
// an item the owner cannot pay for, or that the seller cannot hand over (his
// named item, or else his first of the template), is skipped; an item of
// another template is punished. Each item moves with its message to both;
// then the adena goes to the seller. It reports whether anything moved.
func (s *Server) storeSell(seller, owner *Player, l *storeList, items []storeRequest) bool {
	if l.locked || !owner.InGame() || !seller.InGame() {
		return false
	}
	maxAdena := s.items.cfg.MaxAdena
	ok := false
	var total int64
	for _, r := range items {
		found := false
		for _, ti := range l.items {
			if int32(ti.tmpl.ID) == r.itemID { //nolint:gosec // item IDs fit in int32
				if ti.price == r.price {
					r.count = min(r.count, ti.count)
					found = r.count > 0
				}
				break
			}
		}
		if !found {
			continue
		}
		if maxAdena/r.count < r.price {
			l.locked = true
			break
		}
		next := total + r.count*r.price
		if next > maxAdena {
			l.locked = true
			break
		}
		if owner.inv.Adena() < next {
			continue
		}
		objectID := r.objectID
		it := storeCheckItem(seller, objectID, r.count)
		if it == nil {
			// A buy store names one object ID for several items that do not
			// stack: another item of the template goes.
			other := storeSellableItem(seller, int(r.itemID))
			if other == nil {
				continue
			}
			objectID = other.ObjectID()
			if it = storeCheckItem(seller, objectID, r.count); it == nil {
				continue
			}
		}
		if it.ItemID() != r.itemID {
			s.illegalAction(seller, "a private store sale of another item")
			s.finishStoreDeal(owner, seller)
			return false
		}
		if !it.Template().Tradable {
			continue
		}
		moved, err := seller.inv.Transfer(objectID, r.count, &owner.inv.Container)
		if err != nil {
			continue
		}
		l.remove(func(ti *storeItem) bool { return int32(ti.tmpl.ID) == r.itemID }, r.count) //nolint:gosec // item IDs fit in int32
		ok = true
		total = next
		s.storeDealMessages(owner, seller, owner, seller, moved, r.count)
	}
	if total > 0 {
		if total > owner.inv.Adena() {
			s.finishStoreDeal(owner, seller)
			return false
		}
		s.storeMoveAdena(owner, seller, total)
	}
	if ok {
		s.finishStoreDeal(owner, seller)
	}
	return ok
}

// storeListValid is Mobius TradeList.validate for a sell list: the owner is
// in the game and still has every item in its count.
func (s *Server) storeListValid(owner *Player, l *storeList) bool {
	if !s.exchangeOnline(owner) {
		return false
	}
	for _, ti := range l.items {
		if it := storeCheckItem(owner, ti.objectID, ti.count); it == nil || it.Count() < 1 {
			return false
		}
	}
	return true
}

// remove takes count from the first entry match picks, or the entry when
// that is all of it (Mobius TradeList.removeItem).
func (l *storeList) remove(match func(*storeItem) bool, count int64) {
	i := slices.IndexFunc(l.items, match)
	if i < 0 {
		return
	}
	if ti := l.items[i]; ti.count > count {
		ti.count -= count
		return
	}
	l.items = slices.Delete(l.items, i, i+1)
}

// storeMoveAdena moves count adena from payer to payee within their
// inventories (Mobius PlayerInventory.reduceAdena and addAdena, without
// messages). It reports false when payer has too little.
func (s *Server) storeMoveAdena(payer, payee *Player, count int64) bool {
	if count <= 0 {
		return true
	}
	if payer.inv.Adena() < count {
		return false
	}
	if _, err := payer.inv.DestroyByItemID(adenaID, count); err != nil {
		return false
	}
	if _, err := payee.inv.Create(adenaID, count); err != nil {
		s.log.Error("game: private store adena lost", append(payee.logArgs(), "from", payer.ObjectID(), "count", count, "err", err)...)
	}
	return true
}

// storeDealMessages tells the buyer and the seller of one item what moved
// (Mobius privateStoreBuy and privateStoreSell): for a stack the store owner
// gets "$s1 purchased $s3 $s2(s)." or "You have purchased $s3 $s2(s) from
// $s1." and the other player the other one; for other items the messages
// without the count. buyer is the player who got the item, seller the one
// who gave it; first and second are who gets a message first.
func (s *Server) storeDealMessages(first, second, buyer, seller *Player, moved *inventory.Item, count int64) {
	name := s.itemNameParam(moved.ItemID())
	stack := moved.Stackable()
	msg := func(p *Player) {
		var other *Player
		switch p {
		case buyer:
			other = seller
			if stack {
				p.SendSystemMessage(storeMsgYouPurchasedS3, serverpackets.TextParam(other.Name()), name, serverpackets.NumberParam(itemCount32(count)))
			} else {
				p.SendSystemMessage(storeMsgYouPurchasedS2, serverpackets.TextParam(other.Name()), name)
			}
		default:
			other = buyer
			if stack {
				p.SendSystemMessage(storeMsgPurchasedS3S2, serverpackets.TextParam(other.Name()), name, serverpackets.NumberParam(itemCount32(count)))
			} else {
				p.SendSystemMessage(storeMsgPurchasedS2, serverpackets.TextParam(other.Name()), name)
			}
		}
	}
	msg(first)
	msg(second)
}

// finishStoreDeal sends both players what a purchase changed: the
// InventoryUpdate (100 ms later, merged), StatusUpdate with the load and the
// weight penalty; then both are saved in one transaction (decision T5).
func (s *Server) finishStoreDeal(players ...*Player) {
	for _, p := range players {
		s.queueInventoryUpdate(p, s.takeInventoryUpdates(p, nil)...)
		if p.InGame() {
			p.Send(loadStatusUpdate(p))
		}
		s.refreshOverloaded(p)
	}
	s.saveExchange(players)
}

// storeCloseIfEmpty closes a store whose list was bought out: its type goes
// back to none and UserInfo and CharInfo show it; the owner stays seated.
func (s *Server) storeCloseIfEmpty(owner *Player, l *storeList) {
	if len(l.items) == 0 {
		owner.store.typ = StoreNone
		owner.BroadcastUserInfo()
	}
}
