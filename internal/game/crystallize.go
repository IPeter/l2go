package game

import (
	"time"

	"github.com/IPeter/l2go/internal/enchant"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// Crystallization (Mobius RequestCrystallizeItem): a player with
// Crystallize (248), a dwarf skill, breaks an item into the crystals of its
// grade.

// skillCrystallize is Mobius CommonSkill.CRYSTALLIZE.
const skillCrystallize = 248

// System messages of crystallization (Mobius SystemMessageId).
const (
	sysMsgCrystallizeLevelTooLow serverpackets.SystemMessageID = 562  // "You may not crystallize this item. Your crystallization skill level is too low."
	sysMsgS1Crystallized         serverpackets.SystemMessageID = 1258 // "$s1 has been crystallized."
)

// handleCrystallizeItem handles RequestCrystallizeItem (0x72) in Mobius's
// order: the flood limit; a count below 1 is an illegal action; no store or
// trade; the skill; the item (not a hero item, nor an augmented one unless
// AltAllowAugmentDestroy); a shadow or time-limited item is refused quietly, an item
// without crystals logged; the skill level for the grade (2 for C, 3 for B,
// 4 for A, 5 for S). Then a worn item comes off (its InventoryUpdate at
// once, then the message), the item is destroyed (InventoryUpdate at once),
// and the crystals of its grade and level are added (a delayed
// InventoryUpdate and the load), with "$s1 has been crystallized." and "You
// have earned $s2 $s1(s).", then UserInfo and CharInfo.
func (s *Server) handleCrystallizeItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestCrystallizeItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestCrystallizeItem", append(p.logArgs(), "err", err)...)
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are crystallizing too fast.")
		return
	}
	if req.Count < 1 {
		s.illegalAction(p, "RequestCrystallizeItem with a count below 1")
		return
	}
	if p.InStoreMode() || s.Trading(p) { // store.go, trade.go
		p.SendSystemMessage(tradeMsgStoreNoDiscard)
		return
	}
	level := p.SkillLevel(skillCrystallize)
	if level <= 0 {
		p.SendSystemMessage(sysMsgCrystallizeLevelTooLow)
		p.SendActionFailed()
		if p.char.Race != stats.RaceDwarf && p.ClassID() != 117 && p.ClassID() != 55 {
			s.log.Info("game: crystallize without the skill", append(p.logArgs(), "class", p.ClassID())...)
		}
		return
	}
	it := p.inv.ByObjectID(req.ObjectID)
	if it == nil || it.Template().Hero || (!s.augment.cfg.AllowDestroy && it.Augmented()) {
		p.SendActionFailed()
		return
	}
	count := min(int64(req.Count), it.Count())
	t := it.Template()
	if it.Shadow() || t.Time > 0 {
		return
	}
	if !t.Crystallizable() {
		s.log.Warn("game: crystallize of an item without crystals", append(p.logArgs(), "item", it.ItemID())...)
		return
	}
	if !enchant.CanCrystallize(t.Grade, level) {
		p.SendSystemMessage(sysMsgCrystallizeLevelTooLow)
		p.SendActionFailed()
		return
	}
	name := s.itemNameParam(it.ItemID())
	if it.Equipped() {
		p.inv.UnequipSlot(it.Slot())
		s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
		if it.Enchant() > 0 {
			p.SendSystemMessage(sysMsgEquipmentRemoved, serverpackets.NumberParam(it.Enchant()), name)
		} else {
			p.SendSystemMessage(sysMsgDisarmed, name)
		}
	}
	crystals := enchant.CrystalCount(t, int(it.Enchant()))
	if _, err := p.inv.Destroy(it, count); err != nil {
		p.SendActionFailed()
		return
	}
	s.forgetMana(it)
	s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
	crystalID := int32(t.Grade.CrystalItemID()) //nolint:gosec // crystal item IDs are small
	if _, err := p.inv.Create(crystalID, int64(crystals)); err != nil {
		s.log.Error("game: crystals not given", append(p.logArgs(), "item", crystalID, "count", crystals, "err", err)...)
	} else {
		s.queueInventoryUpdate(p, s.takeInventoryUpdates(p, nil)...)
		p.Send(loadStatusUpdate(p))
	}
	p.SendSystemMessage(sysMsgS1Crystallized, name)
	p.SendSystemMessage(sysMsgEarnedS2S1, s.itemNameParam(crystalID), serverpackets.NumberParam(int32(crystals))) //nolint:gosec // far below 2^31
	p.BroadcastUserInfo()
	s.refreshOverloaded(p)
	s.RequestSave(p)
}
