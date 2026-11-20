package game

import (
	"errors"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/world"
)

// The direct trade between two players (trade*.go, Plan 12 Phase 5): the
// request (TradeRequest, AnswerTradeRequest), the trade window (AddTradeItem,
// TradeDone) and the exchange (Mobius TradeList, Player.onTrade*). The
// exchange itself is in trade_exchange.go, shared with the private stores.
func init() { registerFeature("trade", setupTrade) }

// tradeMaxDistance is how close the partners must stand to request and
// confirm a trade (Mobius TradeRequest, TradeDone: 150, in 3D).
const tradeMaxDistance = 150

// System messages of the trade.
const (
	tradeMsgNothingHappened serverpackets.SystemMessageID = 61   // "Nothing happened."
	tradeMsgInvalidTarget   serverpackets.SystemMessageID = 109  // "Invalid target."
	tradeMsgRequested       serverpackets.SystemMessageID = 118  // "You have requested a trade with $s1."
	tradeMsgDenied          serverpackets.SystemMessageID = 119  // "$s1 has denied your request to trade."
	tradeMsgBegin           serverpackets.SystemMessageID = 120  // "You begin trading with $s1."
	tradeMsgConfirmed       serverpackets.SystemMessageID = 121  // "$s1 has confirmed the trade."
	tradeMsgNoLongerAdjust  serverpackets.SystemMessageID = 122  // "You may no longer adjust items in the trade because the trade has been confirmed."
	tradeMsgSuccessful      serverpackets.SystemMessageID = 123  // "Your trade is successful."
	tradeMsgCanceled        serverpackets.SystemMessageID = 124  // "$s1 has canceled the trade."
	tradeMsgAlreadyTrading  serverpackets.SystemMessageID = 142  // "You are already trading with someone."
	tradeMsgPartnerTrading  serverpackets.SystemMessageID = 143  // "$s1 is already trading with another person. Please try again later."
	tradeMsgNotOnline       serverpackets.SystemMessageID = 145  // "That player is not online."
	tradeMsgCannotUseItems  serverpackets.SystemMessageID = 149  // "You cannot pick up or use items while trading."
	tradeMsgIgnoreList      serverpackets.SystemMessageID = 619  // "$s1 has placed you on his/her Ignore List."
	tradeMsgStoreNoDiscard  serverpackets.SystemMessageID = 1065 // "While operating a private store or workshop, you cannot discard, destroy, or trade an item."
)

// tradeFeature is the trade state on the Server (Server.trade). Loop-only.
type tradeFeature struct {
	// lists holds the open trade of each trading player (Mobius
	// Player.getActiveTradeList for a trade).
	lists map[*Player]*tradeList
	// requests are the request slots the trade uses (trade_request.go).
	requests tradeRequests
}

// tradeList is one side of a trade (Mobius TradeList without the store
// parts): what the owner gives the partner.
type tradeList struct {
	owner, partner *Player
	items          []tradeItem
	confirmed      bool
	locked         bool
}

// tradeItem is an item of a trade list (Mobius TradeItem): the item as it
// was when it was added, and the count to give.
type tradeItem struct {
	exchangeItem
	info serverpackets.ItemInfo
}

// setupTrade registers the trade packets and the leave hook.
func setupTrade(s *Server) error {
	s.trade.lists = make(map[*Player]*tradeList)
	s.trade.requests = newTradeRequests()
	s.OnPlayerLeave(s.tradeLeave)
	return errors.Join(
		s.Register(clientpackets.OpTradeRequest, s.handleTradeRequest),
		s.Register(clientpackets.OpAnswerTradeRequest, s.handleAnswerTradeRequest),
		s.Register(clientpackets.OpAddTradeItem, s.handleAddTradeItem),
		s.Register(clientpackets.OpTradeDone, s.handleTradeDone),
	)
}

// Trading reports whether p has a trade window open (Mobius
// getActiveTradeList() != null for a trade). Loop-only.
func (s *Server) Trading(p *Player) bool { return s.trade.lists[p] != nil }

// ProcessingTransaction reports whether p is in a request or a trade
// (Mobius isProcessingTransaction): it was asked and has not answered, its
// own request is waiting, or a trade window is open. Item handlers refuse
// to drop, destroy or enchant then. Loop-only.
func (s *Server) ProcessingTransaction(p *Player) bool {
	return s.processingRequest(p) || s.Trading(p)
}

// inStoreMode reports whether p runs a private store (Mobius isInStoreMode,
// Player.InStoreMode). Loop-only.
func (s *Server) inStoreMode(p *Player) bool { return p.InStoreMode() }

// handleTradeRequest handles TradeRequest (0x15) with Mobius's checks in
// order (TradeRequest.java 56–152): the target exists, is around and in p's
// instance (else quietly nothing), is not p, is a player; karma; store mode;
// p busy; the target busy; trade refusal; the target's block list; 150
// units. Then the target gets SendTradeRequest and p "You have requested a
// trade with $s1.". Access levels, the olympiad and jail come later.
func (s *Server) handleTradeRequest(p *Player, body []byte) {
	req, err := clientpackets.ParseTradeRequest(body)
	if err != nil {
		s.log.Debug("game: bad TradeRequest", append(p.logArgs(), "err", err)...)
		return
	}
	if !p.InWorld() {
		return
	}
	target, ok := s.world.Find(world.ObjectID(req.ObjectID))
	if !ok || !inSurroundingRegion(p, target) || targetInstance(target) != p.InstanceID() {
		return
	}
	if target == p {
		p.SendSystemMessage(sysMsgIncorrectTarget)
		return
	}
	partner, ok := target.(*Player)
	if !ok {
		p.SendSystemMessage(tradeMsgInvalidTarget)
		return
	}
	if !s.gs.Economy.AltKarmaPlayerCanTrade && p.Karma() > 0 {
		p.SendMessage("You cannot trade while you are in a chaotic state.")
		return
	}
	if !s.gs.Economy.AltKarmaPlayerCanTrade && partner.Karma() > 0 {
		p.SendMessage("You cannot request a trade while your target is in a chaotic state.")
		return
	}
	if p.InStoreMode() || partner.InStoreMode() {
		p.SendSystemMessage(tradeMsgStoreNoDiscard)
		return
	}
	if s.ProcessingTransaction(p) {
		p.SendSystemMessage(tradeMsgAlreadyTrading)
		return
	}
	if s.processingRequest(partner) || s.ProcessingTransaction(partner) {
		p.SendSystemMessage(tradeMsgPartnerTrading, serverpackets.TextParam(partner.Name()))
		return
	}
	if partner.TradeRefusal() {
		p.SendMessage("That person is in trade refusal mode.")
		return
	}
	if partner.Blocks(p) { // friends.go
		p.SendSystemMessage(tradeMsgIgnoreList, serverpackets.TextParam(partner.Name()))
		return
	}
	if p.Position().Distance3D(partner.Position()) > tradeMaxDistance {
		p.SendSystemMessage(sysMsgTargetOutOfRange)
		return
	}
	s.onTransactionRequest(p, partner)
	partner.Send(serverpackets.NewSendTradeRequestPacket(p.ObjectID()))
	p.SendSystemMessage(tradeMsgRequested, serverpackets.TextParam(partner.Name()))
}

// handleAnswerTradeRequest handles AnswerTradeRequest (0x44) as Mobius does:
// without a requester (none, it left, or its request expired) p's window
// closes with "That player is not online."; an acceptance of a live request
// starts the trade for both, anything else tells the requester that p
// denied. The request slots are cleared either way.
func (s *Server) handleAnswerTradeRequest(p *Player, body []byte) {
	req, err := clientpackets.ParseAnswerTradeRequest(body)
	if err != nil {
		s.log.Debug("game: bad AnswerTradeRequest", append(p.logArgs(), "err", err)...)
		return
	}
	partner := s.activeRequester(p)
	if partner == nil || !s.exchangeOnline(partner) {
		p.Send(serverpackets.NewTradeDonePacket(serverpackets.TradeDoneCanceled))
		p.SendSystemMessage(tradeMsgNotOnline)
		s.setActiveRequester(p, nil)
		return
	}
	if req.Accept && !s.requestExpired(partner) {
		s.startTrade(p, partner)
	} else {
		partner.SendSystemMessage(tradeMsgDenied, serverpackets.TextParam(p.Name()))
	}
	s.setActiveRequester(p, nil)
	s.onTransactionResponse(partner)
}

// startTrade opens the trade window of p, then of partner (Mobius
// Player.startTrade).
func (s *Server) startTrade(p, partner *Player) {
	s.onTradeStart(p, partner)
	s.onTradeStart(partner, p)
}

// onTradeStart gives p a new trade list with partner, "You begin trading
// with $s1." and TradeStart with the items p can trade (Mobius
// Player.onTradeStart).
func (s *Server) onTradeStart(p, partner *Player) {
	s.trade.lists[p] = &tradeList{owner: p, partner: partner}
	p.SendSystemMessage(tradeMsgBegin, serverpackets.PlayerNameParam(partner.Name()))
	var items []serverpackets.ItemInfo
	for _, it := range p.inv.Items() {
		if tradeAvailable(it) {
			info := ItemInfo(it)
			info.ItemID = it.ItemID() // Mobius writes the template ID here
			items = append(items, info)
		}
	}
	p.Send(serverpackets.NewTradeStartPacket(serverpackets.TradeStart{PartnerID: partner.ObjectID(), Items: items}))
}

// tradeAvailable reports whether TradeStart lists the item (Mobius
// Item.isAvailable(player, true, false)): not worn, not a quest item, not
// money of the armor list type, tradeable. Pets and enchant scrolls in use
// come later; GMs trade restricted items only with GMTradeRestrictedItems,
// which is not there yet.
func tradeAvailable(it *inventory.Item) bool {
	t := it.Template()
	return !it.Equipped() && t.Type2 != typeQuest &&
		(t.Type2 != itemType2Money || t.Type1 != itemType1ShieldArmor) &&
		t.Tradable
}

// handleAddTradeItem handles AddTradeItem (0x16) in Mobius's order: a count
// below 1 and a player without a trade do nothing; a partner who is gone
// cancels the trade with "That player is not online."; once either side
// confirmed nothing more goes in; an item p does not own gets "Nothing
// happened."; then TradeList.addItem's rules. An added item shows as
// TradeOwnAdd and TradeUpdate to p and TradeOtherAdd to the partner.
func (s *Server) handleAddTradeItem(p *Player, body []byte) {
	req, err := clientpackets.ParseAddTradeItem(body)
	if err != nil {
		s.log.Debug("game: bad AddTradeItem", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Count < 1 {
		return
	}
	trade := s.trade.lists[p]
	if trade == nil {
		s.log.Warn("game: AddTradeItem without a trade", append(p.logArgs(), "itemObjectId", req.ObjectID, "tradeId", req.TradeID)...)
		return
	}
	partner := trade.partner
	partnerList := s.trade.lists[partner]
	if !s.exchangeOnline(partner) || partnerList == nil {
		p.SendSystemMessage(tradeMsgNotOnline)
		s.cancelActiveTrade(p)
		return
	}
	if trade.confirmed || partnerList.confirmed {
		p.SendSystemMessage(tradeMsgNoLongerAdjust)
		return
	}
	if p.inv.ByObjectID(req.ObjectID) == nil {
		p.SendSystemMessage(tradeMsgNothingHappened)
		return
	}
	ti := trade.addItem(p, req.ObjectID, int64(req.Count))
	if ti == nil {
		return
	}
	p.Send(serverpackets.NewTradeOwnAddPacket(ti.info))
	p.Send(tradeUpdatePacket(trade))
	partner.Send(serverpackets.NewTradeOtherAddPacket(ti.info))
}

// addItem puts count of the owner's item into the list (Mobius
// TradeList.addItem with price 0): not once locked; the item must be
// tradeable and not a quest item, count at least 1 and at most what it has,
// 1 for an item that does not stack, and not in the list already. Adding
// takes back the owner's confirmation. It returns the new entry, or nil.
func (l *tradeList) addItem(owner *Player, objectID int32, count int64) *tradeItem {
	if l.locked {
		return nil
	}
	it := owner.inv.ByObjectID(objectID)
	if it == nil {
		return nil
	}
	t := it.Template()
	if !t.Tradable || t.QuestItem {
		return nil
	}
	if count <= 0 || count > it.Count() || (!it.Stackable() && count > 1) {
		return nil
	}
	// Mobius refuses MaxAdena / count < price; a trade has no price.
	for _, ti := range l.items {
		if ti.objectID == objectID {
			return nil
		}
	}
	info := ItemInfo(it)
	info.Count = itemCount32(count)
	l.items = append(l.items, tradeItem{exchangeItem: exchangeItemOf(it, count), info: info})
	l.confirmed = false
	return &l.items[len(l.items)-1]
}

// exchangeItems returns the list's items for the exchange.
func (l *tradeList) exchangeItems() []exchangeItem {
	out := make([]exchangeItem, len(l.items))
	for i, ti := range l.items {
		out[i] = ti.exchangeItem
	}
	return out
}

// tradeUpdatePacket is TradeUpdate for the owner's list (Mobius
// TradeUpdate): every item of the list with what is left of it in the
// inventory.
func tradeUpdatePacket(l *tradeList) []byte {
	items := make([]serverpackets.TradeUpdateItem, 0, len(l.items))
	for _, ti := range l.items {
		info := ti.info
		info.ItemID = int32(ti.template.ID) //nolint:gosec // item IDs fit in int32; Mobius writes the template ID here
		var have int32
		if it := l.owner.inv.ByObjectID(ti.objectID); it != nil {
			have = itemCount32(it.Count())
		}
		items = append(items, serverpackets.TradeUpdateItem{Item: info, InventoryCount: have, Stackable: ti.template.Stackable})
	}
	return serverpackets.NewTradeUpdatePacket(items)
}

// handleTradeDone handles TradeDone (0x17) as Mobius does: the 1 s flood
// guard; nothing without an open, unlocked trade; 0 cancels for both. 1
// confirms, unless the partner is gone ("That player is not online."), in
// another instance or over 150 units away, which cancel the trade. The
// enchant check (ignored while either side enchants) and access levels come
// with those features.
func (s *Server) handleTradeDone(p *Player, body []byte) {
	req, err := clientpackets.ParseTradeDone(body)
	if err != nil {
		s.log.Debug("game: bad TradeDone", append(p.logArgs(), "err", err)...)
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are trading too fast.")
		return
	}
	trade := s.trade.lists[p]
	if trade == nil || trade.locked {
		return
	}
	if !req.Confirm {
		s.cancelActiveTrade(p)
		return
	}
	partner := trade.partner
	if !s.exchangeOnline(partner) {
		s.cancelActiveTrade(p)
		p.SendSystemMessage(tradeMsgNotOnline)
		return
	}
	if partner.InstanceID() != p.InstanceID() || p.Position().Distance3D(partner.Position()) > tradeMaxDistance {
		s.cancelActiveTrade(p)
		return
	}
	s.confirmTrade(trade)
}

// confirmTrade confirms l (Mobius TradeList.confirm): the partner is told
// ("$s1 has confirmed the trade." and TradeOtherDone), or, when the partner
// confirmed already, both lists lock, are checked against the inventories
// and the exchange runs.
func (s *Server) confirmTrade(l *tradeList) {
	if l.confirmed {
		return
	}
	partnerList := s.trade.lists[l.partner]
	if partnerList == nil {
		s.log.Warn("game: the trade partner has no trade", l.owner.logArgs()...)
		return
	}
	l.confirmed = true
	if !partnerList.confirmed {
		l.partner.SendSystemMessage(tradeMsgConfirmed, serverpackets.PlayerNameParam(l.owner.Name()))
		l.partner.Send(serverpackets.NewTradeOtherDonePacket())
		return
	}
	partnerList.locked, l.locked = true, true
	if !s.exchangeValid(partnerList.owner, partnerList.exchangeItems()) || !s.exchangeValid(l.owner, l.exchangeItems()) {
		// Mobius leaves both lists locked, so neither window can close
		// until the players leave; the trade is cancelled instead.
		s.cancelActiveTrade(l.owner)
		return
	}
	s.doExchange(l, partnerList)
}

// doExchange runs the exchange of l's owner (who confirmed last) and the
// partner (Mobius TradeList.doExchange): weight, then slots on both sides;
// the partner's items move first, then the owner's; the packets and the
// joint save (finishExchange); then TradeDone(1) to both, with "Your trade
// is successful." when the items moved.
func (s *Server) doExchange(l, partnerList *tradeList) {
	owner, partner := l.owner, partnerList.owner
	mine, theirs := l.exchangeItems(), partnerList.exchangeItems()
	success := false
	if s.exchangeFits(owner, mine, partner, theirs) {
		s.transferExchangeItems(partner, owner, theirs)
		s.transferExchangeItems(owner, partner, mine)
		s.finishExchange(owner, partner)
		success = true
	}
	s.onTradeFinish(partner, success)
	s.onTradeFinish(owner, success)
}

// onTradeFinish closes p's trade window after the exchange (Mobius
// Player.onTradeFinish).
func (s *Server) onTradeFinish(p *Player, success bool) {
	delete(s.trade.lists, p)
	p.Send(serverpackets.NewTradeDonePacket(serverpackets.TradeDoneFinished))
	if success {
		p.SendSystemMessage(tradeMsgSuccessful)
	}
}

// cancelActiveTrade cancels p's trade for both sides (Mobius
// Player.cancelActiveTrade): the partner, then p, get TradeDone(0) and "$s1
// has canceled the trade." with p's name.
func (s *Server) cancelActiveTrade(p *Player) {
	l := s.trade.lists[p]
	if l == nil {
		return
	}
	if l.partner != nil {
		s.onTradeCancel(l.partner, p)
	}
	s.onTradeCancel(p, p)
}

// onTradeCancel closes p's trade window, cancelled by by (Mobius
// Player.onTradeCancel). A partner whose list is not p's partner's is left
// alone.
func (s *Server) onTradeCancel(p, by *Player) {
	l := s.trade.lists[p]
	if l == nil {
		return
	}
	l.locked = true
	delete(s.trade.lists, p)
	if p.InGame() {
		p.Send(serverpackets.NewTradeDonePacket(serverpackets.TradeDoneCanceled))
		p.SendSystemMessage(tradeMsgCanceled, serverpackets.PlayerNameParam(by.Name()))
	}
}

// tradeLeave is the OnPlayerLeave hook (decision T7): the request slots are
// cleared and an open trade is cancelled for both.
func (s *Server) tradeLeave(p *Player) {
	s.setActiveRequester(p, nil)
	s.onTransactionResponse(p)
	s.cancelActiveTrade(p)
}
