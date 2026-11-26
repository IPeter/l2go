package game

import (
	"github.com/IPeter/l2go/internal/augment"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
)

// The packets of the augmentation windows (Plan 14, M11d), in Mobius's
// order (RequestConfirmTargetItem, RequestConfirmRefinerItem,
// RequestConfirmGemStone, RequestRefine, RequestConfirmCancelItem,
// RequestRefineCancel), with decision X10: the NPC the player talked to
// last must be a blacksmith that offers augmentation, in the player's
// instance and within interaction range (GMs need none, as for selling).

// augmentItem is an item of p's inventory as the augmentation rules see it.
func augmentItem(p *Player, it *inventory.Item) augment.Item {
	return augment.Item{
		Template:     it.Template(),
		Owned:        it.Container() == &p.inv.Container,
		Carried:      it.Loc() == storage.LocInventory,
		Worn:         it.Equipped(),
		Augmentation: augment.Augmentation(it.Augmentation()),
		Shadow:       it.Shadow(),
	}
}

// blacksmithNear reports whether p may use the augmentation windows
// (decision X10).
func (s *Server) blacksmithNear(p *Player) bool {
	if p.IsGM() {
		return true
	}
	n := s.Npc(p.LastNpc)
	if n == nil || !n.InWorld() || n.Template == nil || !s.augment.tables.Blacksmith(n.Template.ID) {
		return false
	}
	return p.InstanceID() == n.InstanceID() && p.Position().InRange3D(n.Position(), s.npcInteractionRange())
}

// sendAugmentMsg sends one of the augmentation system messages, nothing
// for 0.
func sendAugmentMsg(p *Player, id int) {
	if id != 0 {
		p.SendSystemMessage(serverpackets.SystemMessageID(id)) //nolint:gosec // a message ID
	}
}

// refineItems looks up the items of a refine request; nil for one that p
// does not carry, as Mobius returns then without an answer.
func refineItems(p *Player, ids ...int32) []*inventory.Item {
	items := make([]*inventory.Item, len(ids))
	for i, id := range ids {
		if items[i] = p.inv.ByObjectID(id); items[i] == nil {
			return nil
		}
	}
	return items
}

// handleConfirmTargetItem handles RequestConfirmTargetItem (D0 29): the
// item put into the augmentation window is accepted with
// ExPutItemResultForVariationMake, or refused with "Once an item is
// augmented, it cannot be augmented again." or "This is not a suitable
// item.".
func (s *Server) handleConfirmTargetItem(p *Player, body []byte) {
	id, err := clientpackets.ParseRequestObjectID(body)
	if err != nil {
		s.log.Debug("game: bad RequestConfirmTargetItem", append(p.logArgs(), "err", err)...)
		return
	}
	t := s.augment.tables
	items := refineItems(p, id)
	if t == nil || items == nil {
		return
	}
	it := items[0]
	if !s.blacksmithNear(p) {
		p.SendSystemMessage(augment.MsgNotSuitable)
		return
	}
	res := t.Check(s.augment.cfg, augment.Request{Player: s.augmentPlayer(p), Target: augmentItem(p, it)})
	sendAugmentMsg(p, res.PlayerMsg)
	if !res.Valid {
		if it.Augmented() {
			p.SendSystemMessage(augment.MsgAlreadyAugmented)
		} else {
			p.SendSystemMessage(augment.MsgNotSuitable)
		}
		return
	}
	p.Send(serverpackets.NewExPutItemResultForVariationMakePacket(it.ObjectID(), it.ItemID()))
}

// handleConfirmRefinerItem handles RequestConfirmRefinerItem (D0 2A): the
// life stone is accepted with ExPutIntensiveResultForVariationMake, which
// names the gemstones and their count, or refused with "This is not a
// suitable item.".
func (s *Server) handleConfirmRefinerItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestConfirmRefinerItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestConfirmRefinerItem", append(p.logArgs(), "err", err)...)
		return
	}
	t := s.augment.tables
	items := refineItems(p, req.TargetObjectID, req.RefinerObjectID)
	if t == nil || items == nil {
		return
	}
	target, refiner := items[0], items[1]
	if !s.blacksmithNear(p) {
		p.SendSystemMessage(augment.MsgNotSuitable)
		return
	}
	ri := augmentItem(p, refiner)
	res := t.Check(s.augment.cfg, augment.Request{Player: s.augmentPlayer(p), Target: augmentItem(p, target), Refiner: &ri})
	sendAugmentMsg(p, res.PlayerMsg)
	if !res.Valid {
		p.SendSystemMessage(augment.MsgNotSuitable)
		return
	}
	grade := target.Template().Grade
	p.Send(serverpackets.NewExPutIntensiveResultForVariationMakePacket(serverpackets.ExPutIntensiveResultForVariationMake{
		RefinerObjectID: refiner.ObjectID(),
		LifeStoneItemID: refiner.ItemID(),
		GemstoneItemID:  int32(t.GemstoneID(grade)),                         //nolint:gosec // an item ID
		GemstoneCount:   int32(t.GemstoneCount(grade, res.LifeStone.Grade)), //nolint:gosec // a small count
	}))
}

// refineCheck runs the checks of a request with gemstones
// (isValid(player, item, refiner, gemstones)), sending the player-state
// message.
func (s *Server) refineCheck(p *Player, target, refiner, gem *inventory.Item) augment.Result {
	ri, gi := augmentItem(p, refiner), augmentItem(p, gem)
	res := s.augment.tables.Check(s.augment.cfg, augment.Request{
		Player: s.augmentPlayer(p), Target: augmentItem(p, target), Refiner: &ri, Gemstone: &gi, GemCount: gem.Count(),
	})
	sendAugmentMsg(p, res.PlayerMsg)
	return res
}

// handleConfirmGemStone handles RequestConfirmGemStone (D0 2B): the
// gemstones are accepted with ExPutCommissionResultForVariationMake when
// the count is the one the item needs ("Gemstone quantity is incorrect."
// otherwise), or refused with "This is not a suitable item.".
func (s *Server) handleConfirmGemStone(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestRefine(body)
	if err != nil {
		s.log.Debug("game: bad RequestConfirmGemStone", append(p.logArgs(), "err", err)...)
		return
	}
	t := s.augment.tables
	items := refineItems(p, req.TargetObjectID, req.RefinerObjectID, req.GemstoneObjectID)
	if t == nil || items == nil {
		return
	}
	target, refiner, gem := items[0], items[1], items[2]
	if !s.blacksmithNear(p) {
		p.SendSystemMessage(augment.MsgNotSuitable)
		return
	}
	res := s.refineCheck(p, target, refiner, gem)
	if !res.Valid {
		p.SendSystemMessage(augment.MsgNotSuitable)
		return
	}
	if int(req.GemstoneCount) != t.GemstoneCount(target.Template().Grade, res.LifeStone.Grade) {
		p.SendSystemMessage(augment.MsgGemstoneCount)
		return
	}
	p.Send(serverpackets.NewExPutCommissionResultForVariationMakePacket(gem.ObjectID(), req.GemstoneCount))
}

// handleRefine handles RequestRefine (D0 2C), Mobius RequestRefine.runImpl:
// a refused request gets ExVariationResult 0 and "Augmentation failed due
// to inappropriate conditions."; otherwise a worn item comes off (its
// InventoryUpdate at once, then UserInfo), the life stone and the
// gemstones are used up, the item gets a random augmentation of the life
// stone (internal/augment), and ExVariationResult with the two options,
// the item's InventoryUpdate and StatusUpdate with the load follow.
func (s *Server) handleRefine(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestRefine(body)
	if err != nil {
		s.log.Debug("game: bad RequestRefine", append(p.logArgs(), "err", err)...)
		return
	}
	t := s.augment.tables
	items := refineItems(p, req.TargetObjectID, req.RefinerObjectID, req.GemstoneObjectID)
	if t == nil || items == nil {
		return
	}
	target, refiner, gem := items[0], items[1], items[2]
	fail := func() {
		p.Send(serverpackets.NewExVariationResultPacket(0, 0, 0))
		p.SendSystemMessage(augment.MsgInappropriateRefine)
	}
	if !s.blacksmithNear(p) {
		fail()
		return
	}
	res := s.refineCheck(p, target, refiner, gem)
	if !res.Valid {
		fail()
		return
	}
	ls := res.LifeStone
	tmpl := target.Template()
	if int(req.GemstoneCount) != t.GemstoneCount(tmpl.Grade, ls.Grade) {
		fail()
		return
	}
	if target.Equipped() {
		p.inv.UnequipSlot(target.Slot())
		s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
		p.BroadcastUserInfo()
	}
	if !s.DestroyItem(p, refiner, 1, false) {
		return
	}
	if !s.DestroyItem(p, gem, int64(req.GemstoneCount), false) {
		return
	}
	magic := tmpl.Weapon != nil && tmpl.Weapon.Magic
	aug := t.Generate(s.augment.rnd, s.augment.cfg, ls.ItemID, ls.Grade, magic)
	target.SetAugmentation(int32(aug))
	s.log.Debug("item augmented", append(p.logArgs(), "item", target.ItemID(), "itemObjectId", target.ObjectID(), "lifeStone", ls.ItemID, "augmentation", int32(aug))...)
	p.Send(serverpackets.NewExVariationResultPacket(int32(aug.Stat12()), int32(aug.Stat34()), 1)) //nolint:gosec // option IDs are below 2^16
	s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
	p.Send(loadStatusUpdate(p))
	s.RequestSave(p)
}

// handleConfirmCancelItem handles RequestConfirmCancelItem (D0 2D): the
// augmented item put into the cancel window is answered with
// ExPutItemResultForVariationCancel and the fee, or refused with
// "Augmentation removal can only be done on an augmented item." or "This
// is not a suitable item.".
func (s *Server) handleConfirmCancelItem(p *Player, body []byte) {
	id, err := clientpackets.ParseRequestObjectID(body)
	if err != nil {
		s.log.Debug("game: bad RequestConfirmCancelItem", append(p.logArgs(), "err", err)...)
		return
	}
	t := s.augment.tables
	if t == nil {
		return
	}
	st := s.augmentPlayer(p)
	if msg, ok := augment.CheckPlayer(st); !ok {
		sendAugmentMsg(p, msg)
		return
	}
	it := p.inv.ByObjectID(id)
	if it == nil || !s.blacksmithNear(p) {
		return
	}
	cc := t.CheckCancel(s.augment.cfg, st, augmentItem(p, it))
	sendAugmentMsg(p, cc.Msg)
	if !cc.Valid {
		return
	}
	p.Send(serverpackets.NewExPutItemResultForVariationCancelPacket(it.ObjectID(), cc.Fee))
}

// handleRefineCancel handles RequestRefineCancel (D0 2E), Mobius
// RequestRefineCancel.runImpl: a refused request gets
// ExVariationCancelResult 0 (with "Augmentation removal can only be done on
// an augmented item." for an item without one); otherwise the fee is paid
// ("$s1 adena disappeared."), a worn item comes off, the augmentation goes,
// and ExVariationCancelResult 1 and the InventoryUpdate follow.
func (s *Server) handleRefineCancel(p *Player, body []byte) {
	id, err := clientpackets.ParseRequestObjectID(body)
	if err != nil {
		s.log.Debug("game: bad RequestRefineCancel", append(p.logArgs(), "err", err)...)
		return
	}
	t := s.augment.tables
	refuse := func() { p.Send(serverpackets.NewExVariationCancelResultPacket(0)) }
	it := p.inv.ByObjectID(id)
	if t == nil || it == nil || !s.blacksmithNear(p) {
		refuse()
		return
	}
	cc := t.CheckRefineCancel(s.augment.cfg, s.augmentPlayer(p), augmentItem(p, it))
	sendAugmentMsg(p, cc.Msg)
	sendAugmentMsg(p, cc.PlayerMsg)
	if !cc.Valid {
		refuse()
		return
	}
	if !s.ReduceAdena(p, cc.Fee, true) {
		refuse()
		return
	}
	worn := it.Equipped()
	if worn {
		p.inv.UnequipSlot(it.Slot())
	}
	it.SetAugmentation(0)
	s.log.Debug("augmentation removed", append(p.logArgs(), "item", it.ItemID(), "itemObjectId", it.ObjectID(), "fee", cc.Fee)...)
	p.Send(serverpackets.NewExVariationCancelResultPacket(1))
	s.itemsChanged(p, false, nil)
	if worn {
		// Not in Mobius: the paperdoll and the stats changed.
		p.BroadcastUserInfo()
	}
}
