package game

import (
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
)

// handleUseItem handles UseItem (0x14) in Mobius's order: an unknown item
// does nothing, a quest item is refused, a dead player does nothing, the
// item's conditions must hold unless it is worn; an equipable item goes on
// or comes off (useEquippableItem), a lure goes into the left hand of a
// fishing rod, and other items run their handler (Book here, the item
// skills and shots in item_skill.go).
func (s *Server) handleUseItem(p *Player, body []byte) {
	req, err := clientpackets.ParseUseItem(body)
	if err != nil {
		s.log.Debug("game: bad UseItem", append(p.logArgs(), "err", err)...)
		return
	}
	it := p.inv.ByObjectID(req.ObjectID)
	if it == nil {
		return
	}
	if s.Trading(p) { // trade.go; pet collars are exempt once there are pets
		p.SendSystemMessage(tradeMsgCannotUseItems)
		return
	}
	if p.InStoreMode() { // store.go; pet collars are exempt once there are pets
		p.SendSystemMessage(storeMsgNoItemUse)
		p.SendActionFailed()
		return
	}
	t := it.Template()
	if t.Type2 == typeQuest {
		p.SendSystemMessage(sysMsgCannotUseQuestItems)
		return
	}
	// Mobius returns quietly for a dead player (isAlikeDead) before its
	// "cannot be used" message.
	if p.IsDead() {
		return
	}
	if !it.Equipped() && !s.checkItemCondition(p, it) {
		return
	}
	if !s.itemUseReady(p, it) { // item_skill.go: stun, sleep, item reuse
		return
	}
	s.onActionRequest(p)
	if it.Equipable() {
		if !s.equipWhileCasting(p) {
			s.useEquippableItem(p, it)
		}
		return
	}
	if w := p.inv.Paperdoll(storage.PaperdollRHand); w != nil && w.Template().Weapon != nil &&
		w.Template().Weapon.Type == data.WeaponFishingrod && isLure(it.ItemID()) {
		p.inv.SetPaperdoll(storage.PaperdollLHand, it)
		p.inv.TakeUpdates()
		p.BroadcastUserInfo()
		s.SendItemList(p, false)
		s.RequestSave(p)
		return
	}
	switch {
	case t.Etc != nil && t.Etc.Handler == "Book":
		s.useBook(p, it)
	case t.Etc != nil && t.Etc.Handler != "":
		if !s.useItemHandler(p, it, req.CtrlPressed) { // item_skill.go
			s.log.Debug("game: item handler not available yet", append(p.logArgs(), "item", it.ItemID(), "handler", t.Etc.Handler)...)
		}
	}
}

// isLure reports whether an item is a lure (the ranges of Mobius UseItem).
func isLure(id int32) bool {
	return (id >= 6519 && id <= 6527) || (id >= 7610 && id <= 7613) || (id >= 7807 && id <= 7809) ||
		(id >= 8484 && id <= 8486) || (id >= 8505 && id <= 8513)
}

// useEquippableItem puts it on or takes it off (Mobius
// Player.useEquippableItem): the message, a shadow item's mana, UserInfo
// and CharInfo, the InventoryUpdate of every item that moved, and
// ExStorageMaxCount when the inventory limit changed. Loop-only.
func (s *Server) useEquippableItem(p *Player, it *inventory.Item) {
	if it.Container() != &p.inv.Container {
		return
	}
	oldLimit := p.stats.InventoryLimit()
	name := s.itemNameParam(it.ItemID())
	if it.Equipped() {
		if it.Enchant() > 0 {
			p.SendSystemMessage(sysMsgEquipmentRemoved, serverpackets.NumberParam(it.Enchant()), name)
		} else {
			p.SendSystemMessage(sysMsgDisarmed, name)
		}
		p.inv.UnequipItem(it)
	} else {
		p.inv.Equip(it, inventory.Wearer{GM: p.IsGM()})
		if it.Equipped() {
			if it.Enchant() > 0 {
				p.SendSystemMessage(sysMsgEquippedS1S2, serverpackets.NumberParam(it.Enchant()), name)
			} else {
				p.SendSystemMessage(sysMsgEquippedS1, name)
			}
			s.decreaseMana(p, it)
			s.weaponEquipped(p, it) // shot.go: a weapon recharges the shots
		} else {
			p.SendSystemMessage(sysMsgEquipConditionNotMet)
		}
	}
	s.refreshExpertisePenalty(p) // penalty.go
	p.BroadcastUserInfo()
	s.itemsChanged(p, false, nil)
	if p.stats.InventoryLimit() != oldLimit {
		p.Send(s.storageMaxCount(p))
	}
}

// handleUnEquipItem handles RequestUnEquipItem (0x11): the client names a
// paperdoll slot (Mobius Interlude BodyPart.fromPaperdollSlot). Arrows in
// the left hand stay, and only GMs take off weapons that must stay worn.
func (s *Server) handleUnEquipItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestUnEquipItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestUnEquipItem", append(p.logArgs(), "err", err)...)
		return
	}
	s.log.Debug("RequestUnEquipItem", append(p.logArgs(), "slot", req.Slot)...)
	if req.Slot < 0 || req.Slot >= storage.PaperdollTotalSlots {
		return
	}
	slot := storage.PaperdollSlot(req.Slot)
	it := p.inv.Paperdoll(slot)
	if it == nil {
		return
	}
	t := it.Template()
	switch {
	case slot == storage.PaperdollLHand && t.Etc != nil:
		return // arrows and lures
	case p.IsDead():
		return
	case t.Weapon != nil && t.Weapon.ForceEquip && !p.IsGM():
		p.SendMessage("That item cannot be taken off.")
		return
	}
	off := p.inv.UnequipItem(it)
	p.BroadcastUserInfo()
	if len(off) == 0 {
		return
	}
	first := off[0]
	name := s.itemNameParam(first.ItemID())
	if first.Enchant() > 0 {
		p.SendSystemMessage(sysMsgEquipmentRemoved, serverpackets.NumberParam(first.Enchant()), name)
	} else {
		p.SendSystemMessage(sysMsgDisarmed, name)
	}
	s.itemsChanged(p, false, nil)
}

// handleDestroyItem handles RequestDestroyItem (0x59) in Mobius's order. A
// worn item comes off first; both InventoryUpdates go at once, as in
// Mobius, then StatusUpdate with the load and "... has disappeared.".
func (s *Server) handleDestroyItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestDestroyItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestDestroyItem", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Count < 1 {
		p.SendSystemMessage(sysMsgDestroyCountIncorrect)
		if req.Count < 0 {
			s.illegalAction(p, "RequestDestroyItem with a negative count")
		}
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are destroying items too fast.")
		return
	}
	if s.ProcessingTransaction(p) || p.InStoreMode() { // trade.go
		p.SendSystemMessage(tradeMsgStoreNoDiscard)
		return
	}
	it := p.inv.ByObjectID(req.ObjectID)
	if it == nil {
		p.SendSystemMessage(sysMsgCannotDiscard)
		return
	}
	t := it.Template()
	if !p.IsGM() && !t.Destroyable {
		if t.Hero {
			p.SendSystemMessage(sysMsgHeroWeaponsNoDestroy)
		} else {
			p.SendSystemMessage(sysMsgCannotDiscard)
		}
		return
	}
	count := int64(req.Count)
	if !it.Stackable() && count > 1 {
		s.illegalAction(p, "RequestDestroyItem of more than one item that does not stack")
		return
	}
	count = min(count, it.Count())
	name := s.itemNameParam(it.ItemID())
	if it.Equipped() {
		if it.Enchant() > 0 {
			p.SendSystemMessage(sysMsgEquipmentRemoved, serverpackets.NumberParam(it.Enchant()), name)
		} else {
			p.SendSystemMessage(sysMsgDisarmed, name)
		}
		p.inv.UnequipSlot(it.Slot())
		s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
		// Not in Mobius: the paperdoll others see changed.
		p.RefreshStats()
		p.BroadcastUserInfo()
	}
	if _, err := p.inv.Destroy(it, count); err != nil {
		return
	}
	s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
	p.Send(loadStatusUpdate(p))
	if count > 1 {
		p.SendSystemMessage(sysMsgS2S1Disappeared, name, serverpackets.NumberParam(itemCount32(count)))
	} else {
		p.SendSystemMessage(sysMsgS1Disappeared, name)
	}
	s.forgetMana(it)
	s.refreshOverloaded(p)
	s.RequestSave(p)
}

// handleItemList handles RequestItemList (0x0F): the ItemList with the
// window open, unless a shop or warehouse window opened less than 1.5 s ago.
func (s *Server) handleItemList(p *Player, body []byte) {
	if _, err := clientpackets.ParseRequestItemList(body); err != nil {
		return
	}
	if s.Now().Before(p.items.blockedUntil) {
		return
	}
	s.SendItemList(p, true)
}

// useBook is Mobius's Book handler: the help page html/help/<itemId>.htm,
// bound to the item, then ActionFailed.
func (s *Server) useBook(p *Player, it *inventory.Item) {
	id := strconv.Itoa(int(it.ItemID()))
	content, ok := s.htmlFile(p, "help/"+id+".htm")
	if !ok {
		content = "<html><body>My Text is missing:<br>data/html/help/" + id + ".htm</body></html>"
	}
	pg := newPage(0, content)
	pg.itemID = it.ItemID()
	s.sendPage(p, pg)
	p.SendActionFailed()
}

// decreaseMana takes one mana from a shadow item (Mobius Item.decreaseMana):
// at 10, 5 and 1 the owner is warned; at 0 the item comes off and is
// destroyed; otherwise a worn item loses the next one a minute later.
// Loop-only.
func (s *Server) decreaseMana(p *Player, it *inventory.Item) {
	if !it.Shadow() {
		return
	}
	it.SetMana(max(it.Mana()-1, 0))
	name := s.itemNameParam(it.ItemID())
	switch it.Mana() {
	case 10:
		p.SendSystemMessage(sysMsgMana10, name)
	case 5:
		p.SendSystemMessage(sysMsgMana5, name)
	case 1:
		p.SendSystemMessage(sysMsgMana1, name)
	}
	inWarehouse := it.Loc() == storage.LocWarehouse
	if it.Mana() > 0 {
		if _, consuming := s.items.mana[it]; !consuming && it.Equipped() {
			s.items.mana[it] = manaTask{p: p, timer: s.loop.After(manaConsumptionRate, func() { s.manaTick(p, it) })}
		}
		if inWarehouse {
			p.items.warehouse.TakeUpdates()
			s.RequestSave(p)
		} else {
			s.itemsChanged(p, false, nil)
		}
		return
	}
	p.SendSystemMessage(sysMsgMana0, name)
	s.forgetMana(it)
	if it.Equipped() {
		p.inv.UnequipSlot(it.Slot())
		s.itemsChanged(p, false, nil)
		p.BroadcastUserInfo()
	}
	if inWarehouse {
		if _, err := p.items.warehouse.Destroy(it, it.Count()); err == nil {
			s.RequestSave(p)
		}
		return
	}
	if _, err := p.inv.Destroy(it, it.Count()); err == nil {
		s.itemsChanged(p, true, nil)
	}
}

// manaTick is the minute timer of a worn shadow item (Mobius
// ItemManaTaskManager): the item loses a mana while p still owns it.
func (s *Server) manaTick(p *Player, it *inventory.Item) {
	delete(s.items.mana, it)
	if !p.InGame() {
		return
	}
	if c := it.Container(); c != &p.inv.Container && c != p.items.warehouse {
		return
	}
	s.decreaseMana(p, it)
}

// forgetMana stops the mana timer of an item that is gone.
func (s *Server) forgetMana(it *inventory.Item) {
	if task, ok := s.items.mana[it]; ok {
		s.loop.Cancel(task.timer)
		delete(s.items.mana, it)
	}
}
