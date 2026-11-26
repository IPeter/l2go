package game

import (
	"errors"
	"unicode/utf16"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
)

// The private stores (store*.go, Plan 12 Phase 6): a seated player sells
// items of his inventory (a sell store, or a package sale that is bought
// whole) or buys items (a buy store). This file opens, sets up and closes
// stores (Mobius Player.tryOpenPrivateSellStore, tryOpenPrivateBuyStore,
// canOpenPrivateStore and the store client packets); store_view.go shows
// them to other players and store_deal.go runs the purchases (Mobius
// TradeList.privateStoreBuy and privateStoreSell).
//
// There is no offline trade (decision T11): a store closes when its owner
// leaves the game.
func init() { registerFeature("stores", setupStores) }

// StoreType is a player's private store mode (Mobius PrivateStoreType). Its
// value is what UserInfo and CharInfo carry.
type StoreType uint8

// The store modes.
const (
	StoreNone        StoreType = 0
	StoreSell        StoreType = 1 // a sell store is open
	StoreSellManage  StoreType = 2 // the setup window of a sell store is open
	StoreBuy         StoreType = 3 // a buy store is open
	StoreBuyManage   StoreType = 4 // the setup window of a buy store is open
	StoreManufacture StoreType = 5 // a recipe shop, which comes with crafting (M11)
	StorePackageSell StoreType = 8 // a package sale is open: the buyer takes everything
)

// System messages of the private stores.
const (
	storeMsgPurchasedS2    serverpackets.SystemMessageID = 378  // "$s1 purchased $s2."
	storeMsgPurchasedS3S2  serverpackets.SystemMessageID = 380  // "$s1 purchased $s3 $s2(s)."
	storeMsgYouPurchasedS2 serverpackets.SystemMessageID = 559  // "You have purchased $s2 from $s1."
	storeMsgYouPurchasedS3 serverpackets.SystemMessageID = 561  // "You have purchased $s3 $s2(s) from $s1."
	storeMsgPriceTooHigh   serverpackets.SystemMessageID = 720  // "The purchase price is higher than the amount of money that you have and so you cannot open a personal store."
	storeMsgInCombat       serverpackets.SystemMessageID = 1135 // "While you are engaged in combat, you cannot operate a private store or private workshop."
	storeMsgCannotOpenHere serverpackets.SystemMessageID = 1296 // "You cannot open a Private Store here."
	storeMsgNoItemUse      serverpackets.SystemMessageID = 1891 // "You may not use items in a private store or private work shop."
)

// storeMsgQuantityExceeds is "You have exceeded the quantity that can be
// inputted.".
const storeMsgQuantityExceeds = exchangeMsgQuantityExceeded

// storeFeature is the private store state on the Server (Server.stores).
// Loop-only.
type storeFeature struct {
	// approaches are the players walking up to a store to open it (Mobius
	// INTERACT intention on a player).
	approaches map[*Player]*storeApproach
}

// playerStore is a player's private store (Mobius Player
// _privateStoreType, getSellList, getBuyList). The lists and their messages
// stay while the player is in the game, so that the next store starts from
// them.
type playerStore struct {
	typ       StoreType
	sell, buy storeList
}

// storeList is the item list of a store (Mobius TradeList as a store).
type storeList struct {
	items []*storeItem
	// title is the store message (Mobius TradeList title; empty for null).
	title string
	// packaged is set for a package sale.
	packaged bool
	// locked is set when a purchase found the list broken (an overflow, an
	// item the owner no longer has): nothing more is bought from it.
	locked bool
}

// storeItem is an entry of a store list (Mobius TradeItem): in a sell list
// an item of the owner's inventory as it was when it was offered, in a buy
// list an item template (objectID 0).
type storeItem struct {
	objectID int32
	tmpl     *data.Item
	enchant  int32
	custom2  int16
	count    int64
	price    int64 // per unit
}

// clear empties the list and unlocks it (Mobius TradeList.clear); the title
// stays.
func (l *storeList) clear() {
	l.items = nil
	l.locked = false
}

// setupStores registers the store packets and actions, the leave hook and
// the walks up to stores.
func setupStores(s *Server) error {
	s.stores.approaches = make(map[*Player]*storeApproach)
	s.OnPlayerLeave(s.storeLeave)
	s.OnTick(s.checkStoreApproaches)
	return errors.Join(
		s.Register(clientpackets.OpRequestPrivateStoreManageSell, s.handleStoreManageSell),
		s.Register(clientpackets.OpSetPrivateStoreListSell, s.handleSetStoreListSell),
		s.Register(clientpackets.OpRequestPrivateStoreQuitSell, s.handleStoreQuit),
		s.Register(clientpackets.OpSetPrivateStoreMsgSell, s.handleSetStoreMsgSell),
		s.Register(clientpackets.OpRequestPrivateStoreBuy, s.handleStoreBuy),
		s.Register(clientpackets.OpRequestPrivateStoreManageBuy, s.handleStoreManageBuy),
		s.Register(clientpackets.OpSetPrivateStoreListBuy, s.handleSetStoreListBuy),
		s.Register(clientpackets.OpRequestPrivateStoreQuitBuy, s.handleStoreQuit),
		s.Register(clientpackets.OpSetPrivateStoreMsgBuy, s.handleSetStoreMsgBuy),
		s.Register(clientpackets.OpRequestPrivateStoreSell, s.handleStoreSell),
		s.RegisterActionUse(clientpackets.ActionUsePrivateStoreSell, s.actionStoreSell),
		s.RegisterActionUse(clientpackets.ActionUsePrivateStoreBuy, s.actionStoreBuy),
	)
}

// InStoreMode reports whether p runs a private store or has its setup
// window open (Mobius isInStoreMode). Loop-only.
func (p *Player) InStoreMode() bool { return p.store.typ != StoreNone }

// StoreType returns p's private store mode. Loop-only.
func (p *Player) StoreType() StoreType { return p.store.typ }

// storeLeave is the OnPlayerLeave hook: the store closes (decision T11, no
// offline trade) and a walk up to a store ends.
func (s *Server) storeLeave(p *Player) {
	delete(s.stores.approaches, p)
	p.store.typ = StoreNone
}

// refuseStoreRestart refuses RequestRestart in store mode (Mobius
// RequestRestart): "Cannot restart while trading." and RestartResponse
// (false). It reports whether it refused.
func (s *Server) refuseStoreRestart(p *Player) bool {
	if !p.InStoreMode() {
		return false
	}
	p.SendMessage("Cannot restart while trading.")
	p.Send(serverpackets.NewRestartResponsePacket(false))
	return true
}

// actionStoreSell is RequestActionUse action 10 (Mobius RequestActionUse:
// tryOpenPrivateSellStore). A dead player gets ActionFailed, as for every
// action.
func (s *Server) actionStoreSell(p *Player, _ clientpackets.RequestActionUse) {
	if p.IsDead() {
		p.SendActionFailed()
		return
	}
	s.tryOpenSellStore(p, false)
}

// actionStoreBuy is RequestActionUse action 28 (tryOpenPrivateBuyStore).
func (s *Server) actionStoreBuy(p *Player, _ clientpackets.RequestActionUse) {
	if p.IsDead() {
		p.SendActionFailed()
		return
	}
	s.tryOpenBuyStore(p)
}

// handleStoreManageSell handles RequestPrivateStoreManageSell (0x73): a
// dead player gets ActionFailed (no olympiad yet), others open the setup
// window of a sell store.
func (s *Server) handleStoreManageSell(p *Player, _ []byte) {
	if p.IsDead() {
		p.SendActionFailed()
		return
	}
	s.tryOpenSellStore(p, false)
}

// handleStoreManageBuy handles RequestPrivateStoreManageBuy (0x90): a dead
// player gets nothing, others open the setup window of a buy store.
func (s *Server) handleStoreManageBuy(p *Player, _ []byte) {
	if p.IsDead() {
		return
	}
	s.tryOpenBuyStore(p)
}

// tryOpenSellStore is Mobius Player.tryOpenPrivateSellStore: where a store
// may open, a running sell store (or its setup window) is taken down, and
// without another store p stands up and gets the setup window. Where none
// may open, a store p runs closes (and p stands up); otherwise p is told
// when the zone forbids stores, and gets ActionFailed.
func (s *Server) tryOpenSellStore(p *Player, packageSale bool) {
	if s.canOpenStore(p) {
		if t := p.store.typ; t == StoreSell || t == StoreSellManage || t == StorePackageSell {
			p.store.typ = StoreNone
		}
		if p.store.typ == StoreNone {
			if p.Sitting {
				s.standUp(p)
			}
			p.store.typ = StoreSellManage
			p.Send(s.storeManageListSell(p, packageSale))
		}
		return
	}
	if p.store.typ != StoreNone && !p.IsDead() {
		p.store.typ = StoreNone
		if p.Sitting {
			s.standUp(p)
		}
		return
	}
	if p.Zones.Inside(ZoneNoStore) {
		p.SendSystemMessage(storeMsgCannotOpenHere)
	}
	p.SendActionFailed()
}

// tryOpenBuyStore is Mobius Player.tryOpenPrivateBuyStore: like
// tryOpenSellStore for a buy store, except that a store p runs stays when
// none may open here.
func (s *Server) tryOpenBuyStore(p *Player) {
	if s.canOpenStore(p) {
		if t := p.store.typ; t == StoreBuy || t == StoreBuyManage {
			p.store.typ = StoreNone
		}
		if p.store.typ == StoreNone {
			if p.Sitting {
				s.standUp(p)
			}
			p.store.typ = StoreBuyManage
			p.Send(s.storeManageListBuy(p))
		}
		return
	}
	if p.Zones.Inside(ZoneNoStore) {
		p.SendSystemMessage(storeMsgCannotOpenHere)
	}
	p.SendActionFailed()
}

// canOpenStore is Mobius Player.canOpenPrivateStore: not within 50 of a
// sitting player or 100 of an NPC (decision T11, "You cannot open a Private
// Store here." otherwise), alive, not in a no-store zone, not casting. The
// olympiad does not exist yet; a rider may not (mount.go).
func (s *Server) canOpenStore(p *Player) bool {
	cfg := s.gs.Stores
	if (cfg.MinRangeFromNpc > 0 || cfg.MinRangeFromPlayer > 0) && s.storeTooClose(p) {
		p.SendSystemMessage(storeMsgCannotOpenHere)
		return false
	}
	return !p.IsDead() && !s.mounted(p) && !p.Zones.Inside(ZoneNoStore) && !s.CastingNow(p)
}

// storeTooClose reports whether a creature around p keeps a store away
// (Mobius World.getFirstVisibleObject with getMinShopDistance): an NPC
// within MinRangeFromNpc or a sitting player within MinRangeFromPlayer, in
// 3D, both collision radii added to the range.
func (s *Server) storeTooClose(p *Player) bool {
	cfg := s.gs.Stores
	pos := p.Position()
	own := int(p.lastView.CollisionRadius)
	for o := range s.world.Visible(p) {
		var r, radius int
		switch c := o.(type) {
		case *Npc:
			r, radius = cfg.MinRangeFromNpc, int(c.statView().CollisionRadius)
		case *Player:
			if c.Sitting {
				r = cfg.MinRangeFromPlayer
			}
			radius = int(c.lastView.CollisionRadius)
		}
		if r > 0 && pos.Distance3D(o.Position()) <= float64(r+own+radius) {
			return true
		}
	}
	return false
}

// sellStoreLimit is how many items a sell store holds (Mobius
// getPrivateSellStoreLimit): 4 for dwarves, 3 for the others, plus the
// P_SELL_LIM stat.
func (s *Server) sellStoreLimit(p *Player) int {
	n := s.gs.Stores.SellSlotsOther
	if p.char.Race == stats.RaceDwarf {
		n = s.gs.Stores.SellSlotsDwarf
	}
	return n + int(p.stats.Calc(stats.PrivateSellLimit, 0))
}

// buyStoreLimit is how many items a buy store holds (Mobius
// getPrivateBuyStoreLimit): 5 for dwarves, 4 for the others, plus the
// P_BUY_LIM stat.
func (s *Server) buyStoreLimit(p *Player) int {
	n := s.gs.Stores.BuySlotsOther
	if p.char.Race == stats.RaceDwarf {
		n = s.gs.Stores.BuySlotsDwarf
	}
	return n + int(p.stats.Calc(stats.PrivateBuyLimit, 0))
}

// storeSetupRefused runs Mobius's checks of SetPrivateStoreListSell and
// SetPrivateStoreListBuy before the list: in combat stance (no duels yet),
// in a no-store zone, or where canOpenStore refuses, p gets its setup window
// back (manage) and ActionFailed, with a message for the first two. It
// reports whether it refused.
func (s *Server) storeSetupRefused(p *Player, manage func() []byte) bool {
	switch {
	case s.InAttackStance(p):
		p.SendSystemMessage(storeMsgInCombat)
		p.Send(manage())
	case p.Zones.Inside(ZoneNoStore):
		p.Send(manage())
		p.SendSystemMessage(storeMsgCannotOpenHere)
	case !s.canOpenStore(p):
		p.Send(manage())
	default:
		return false
	}
	p.SendActionFailed()
	return true
}

// handleSetStoreListSell handles SetPrivateStoreListSell (0x74) in Mobius's
// order: a refused list closes the setup with "Incorrect item count."; then
// the setup checks (storeSetupRefused) and the slots; then the list is
// built, every price checked against MaxAdena with the adena p carries; then
// p sits, its store type shows in UserInfo and CharInfo, and p and the
// players around get the store message.
func (s *Server) handleSetStoreListSell(p *Player, body []byte) {
	req, err := clientpackets.ParseSetPrivateStoreListSell(body)
	if err != nil {
		s.log.Debug("game: bad SetPrivateStoreListSell", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Items == nil {
		p.SendSystemMessage(sysMsgIncorrectCount)
		p.store.typ = StoreNone
		p.BroadcastUserInfo()
		return
	}
	manage := func() []byte { return s.storeManageListSell(p, req.PackageSale) }
	if s.storeSetupRefused(p, manage) {
		return
	}
	if len(req.Items) > s.sellStoreLimit(p) {
		p.Send(manage())
		p.SendSystemMessage(storeMsgQuantityExceeds)
		return
	}
	l := &p.store.sell
	l.clear()
	l.packaged = req.PackageSale
	maxAdena := s.items.cfg.MaxAdena
	total := p.inv.Adena()
	for _, ri := range req.Items {
		count, price := int64(ri.Count), int64(ri.Price)
		if maxAdena/count < price {
			p.SendSystemMessage(sysMsgAdenaLimit)
			return
		}
		storeAddSellItem(p, l, ri.ObjectID, count, price, maxAdena)
		if total += count * price; total > maxAdena {
			p.SendSystemMessage(sysMsgAdenaLimit)
			return
		}
	}
	s.sitDown(p)
	p.store.typ = StoreSell
	if req.PackageSale {
		p.store.typ = StorePackageSell
	}
	s.openStore(p, serverpackets.NewPrivateStoreMsgSellPacket(p.ObjectID(), l.title))
}

// storeAddSellItem adds count of p's item to a sell list (Mobius
// TradeList.addItem): an item p does not carry, one that is not tradeable
// or a quest item, a count it does not have, more than one of an item that
// does not stack, a price that overflows MaxAdena and an item already in
// the list are left out without a word. Unlike Mobius, a worn item is left
// out too: Mobius would sell it off its owner's body.
func storeAddSellItem(p *Player, l *storeList, objectID int32, count, price, maxAdena int64) {
	it := p.inv.ByObjectID(objectID)
	if it == nil || it.Equipped() || !it.Template().Tradable || it.Template().QuestItem || augmentLocked(it) {
		return
	}
	if count <= 0 || count > it.Count() || (!it.Stackable() && count > 1) || maxAdena/count < price {
		return
	}
	for _, ti := range l.items {
		if ti.objectID == objectID {
			return
		}
	}
	l.items = append(l.items, &storeItem{
		objectID: objectID,
		tmpl:     it.Template(),
		enchant:  it.Enchant(),
		custom2:  int16(it.Storage().CustomType2), //nolint:gosec // custom types are small
		count:    count,
		price:    price,
	})
}

// handleSetStoreListBuy handles SetPrivateStoreListBuy (0x91) in Mobius's
// order: a refused list closes the setup; then the setup checks, the list
// is cleared and the slots checked; a price or total over MaxAdena is
// punished as an illegal action; a total over the adena p carries is
// refused with the setup window; then p sits, its store type shows and the
// store message goes out.
func (s *Server) handleSetStoreListBuy(p *Player, body []byte) {
	req, err := clientpackets.ParseSetPrivateStoreListBuy(body)
	if err != nil {
		s.log.Debug("game: bad SetPrivateStoreListBuy", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Items == nil {
		p.store.typ = StoreNone
		p.BroadcastUserInfo()
		return
	}
	manage := func() []byte { return s.storeManageListBuy(p) }
	if s.storeSetupRefused(p, manage) {
		return
	}
	l := &p.store.buy
	l.clear()
	if len(req.Items) > s.buyStoreLimit(p) {
		p.Send(manage())
		p.SendSystemMessage(storeMsgQuantityExceeds)
		return
	}
	maxAdena := s.items.cfg.MaxAdena
	var total int64
	for _, ri := range req.Items {
		count, price := int64(ri.Count), int64(ri.Price)
		if maxAdena/count < price {
			s.illegalAction(p, "a buy store price over MaxAdena")
			return
		}
		s.storeAddBuyItem(l, ri.ItemID, count, price)
		if total += count * price; total > maxAdena {
			s.illegalAction(p, "a buy store total over MaxAdena")
			return
		}
	}
	if total > p.inv.Adena() {
		p.Send(manage())
		p.SendSystemMessage(storeMsgPriceTooHigh)
		return
	}
	s.sitDown(p)
	p.store.typ = StoreBuy
	s.openStore(p, serverpackets.NewPrivateStoreMsgBuyPacket(p.ObjectID(), l.title))
}

// storeAddBuyItem adds an item template to a buy list (Mobius
// TradeList.addItemByItemId): an unknown template, one that is not
// tradeable or a quest item, and more than one of an item that does not
// stack are left out without a word. The price was checked already.
func (s *Server) storeAddBuyItem(l *storeList, itemID int32, count, price int64) {
	t, ok := s.data.ItemTemplate(int(itemID))
	if !ok || !t.Tradable || t.QuestItem || (!t.Stackable && count > 1) {
		return
	}
	l.items = append(l.items, &storeItem{tmpl: t, count: count, price: price})
}

// openStore shows a store that just opened: p's UserInfo and the CharInfo
// the players around see, with the store type, then the store message to p
// and them (Mobius broadcastUserInfo and broadcastPacket, in that order).
func (s *Server) openStore(p *Player, msg []byte) {
	p.BroadcastUserInfo()
	s.world.FlushUpdates()
	s.world.Broadcast(p, msg, true)
}

// handleStoreQuit handles RequestPrivateStoreQuitSell (0x76) and
// RequestPrivateStoreQuitBuy (0x93): the store closes, p stands up and
// UserInfo and CharInfo show it.
func (s *Server) handleStoreQuit(p *Player, _ []byte) {
	p.store.typ = StoreNone
	s.standUp(p)
	p.BroadcastUserInfo()
}

// handleSetStoreMsgSell handles SetPrivateStoreMsgSell (0x77): the message
// over p's sell store; one over 29 characters is punished. p gets the
// message back.
func (s *Server) handleSetStoreMsgSell(p *Player, body []byte) {
	req, _ := clientpackets.ParseSetPrivateStoreMsgSell(body)
	if !s.storeMessageOK(p, req.Message) {
		return
	}
	p.store.sell.title = req.Message
	p.Send(serverpackets.NewPrivateStoreMsgSellPacket(p.ObjectID(), req.Message))
}

// handleSetStoreMsgBuy handles SetPrivateStoreMsgBuy (0x94), like
// handleSetStoreMsgSell for the buy store.
func (s *Server) handleSetStoreMsgBuy(p *Player, body []byte) {
	req, _ := clientpackets.ParseSetPrivateStoreMsgBuy(body)
	if !s.storeMessageOK(p, req.Message) {
		return
	}
	p.store.buy.title = req.Message
	p.Send(serverpackets.NewPrivateStoreMsgBuyPacket(p.ObjectID(), req.Message))
}

// storeMessageOK punishes a store message longer than 29 UTF-16 units as an
// illegal action (Mobius MAX_MSG_LENGTH, Java String.length) and reports
// whether the message may stay.
func (s *Server) storeMessageOK(p *Player, msg string) bool {
	if len(utf16.Encode([]rune(msg))) > clientpackets.PrivateStoreMessageMaxLength {
		s.illegalAction(p, "a private store message over 29 characters")
		return false
	}
	return true
}

// storeMsgPacket is what a player who starts to see p learns about its
// store (Mobius Player.sendInfo): the message of an open sell store,
// package sale or buy store; nil without one.
func (p *Player) storeMsgPacket() []byte {
	switch p.store.typ {
	case StoreSell, StorePackageSell:
		return serverpackets.NewPrivateStoreMsgSellPacket(p.ObjectID(), p.store.sell.title)
	case StoreBuy:
		return serverpackets.NewPrivateStoreMsgBuyPacket(p.ObjectID(), p.store.buy.title)
	case StoreManufacture:
		return serverpackets.NewRecipeShopMsgPacket(p.ObjectID(), p.craft.shopTitle) // workshop.go
	}
	return nil
}

// storeItemAvailable reports whether an item may be offered or listed
// (Mobius Item.isAvailable(owner, false, false)): not worn, not a quest
// item, not money of the armor list type, not adena, tradeable. Pets and
// the item a skill in progress consumes come later.
func storeItemAvailable(it *inventory.Item) bool {
	return tradeAvailable(it) && it.ItemID() != adenaID
}

// storeCheckItem returns p's item with the object ID when count of it may
// change hands (Mobius Player.checkItemManipulation, depositCheckItem); a
// worn item never does (see storeAddSellItem).
func storeCheckItem(p *Player, objectID int32, count int64) *inventory.Item {
	it := depositCheckItem(p, objectID, count)
	if it == nil || it.Equipped() {
		return nil
	}
	return it
}
