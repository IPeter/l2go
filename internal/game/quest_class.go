package game

import (
	"math"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
)

// The class change service (decision Q13) and the quest item helpers
// (Mobius Quest giveItems, rewardItems, takeItems, addExpAndSp).

// System messages of the class change.
const (
	sysMsgClassTransfer      serverpackets.SystemMessageID = 1308 // "Congratulations. You've completed a class transfer."
	sysMsgThirdClassTransfer serverpackets.SystemMessageID = 1606 // "Congratulations! You've completed the third class transfer quest."
)

const (
	// classChangeSkill is the skill whose effect a class change shows
	// (Mobius setPlayerClass: MagicSkillUse 5103).
	classChangeSkill = 5103
	// classChangeSkillListDelay is when the skill list follows a class
	// change.
	classChangeSkillListDelay = 100 * time.Millisecond
)

// ChangeClass changes p's class (Mobius Player.setPlayerClass): the class
// change effect to p and the players around, the new class template, the
// system message, UserInfo and CharInfo, and the skill list 100 ms later.
// Academy, subclasses, hennas and the party and clan windows join with
// their features. Loop-only.
func (s *Server) ChangeClass(p *Player, id int) {
	class, ok := s.data.ClassTemplate(id)
	tmpl := s.classes[id]
	if !ok || tmpl == nil {
		s.log.Warn("game: class change to an unknown class", append(p.logArgs(), "class", id)...)
		return
	}
	pos := p.Position()
	if p.InWorld() {
		s.world.Broadcast(p, serverpackets.NewMagicSkillUsePacket(serverpackets.MagicSkillUse{
			CasterID: p.ObjectID(), TargetID: p.ObjectID(),
			SkillID: classChangeSkill, SkillLevel: 1,
			X: pos.X, Y: pos.Y, Z: pos.Z,
			TargetX: pos.X, TargetY: pos.Y, TargetZ: pos.Z,
		}), true)
	}
	if err := p.stats.SetTemplate(tmpl); err != nil {
		s.log.Error("game: class change failed", append(p.logArgs(), "class", id, "err", err)...)
		return
	}
	p.char.ClassID = int32(id) //nolint:gosec // class IDs are small
	p.class = class
	if data.PlayerClass(id).Level() == 3 { //nolint:gosec // class IDs are small
		p.SendSystemMessage(sysMsgThirdClassTransfer)
	} else {
		p.SendSystemMessage(sysMsgClassTransfer)
	}
	s.partyClassChanged(p) // party_window.go
	s.clanClassChanged(p)  // clan_members.go
	p.BroadcastUserInfo()
	p.after(classChangeSkillListDelay, p.SendSkillList)
	s.RequestSave(p)
	s.scheduleQuestSave(p)
}

// questRewardCount applies the quest reward rates to count (Mobius
// rewardItems).
func (s *Server) questRewardCount(t *data.Item, count int64) int64 {
	r := s.quest.settings
	rate := r.RewardRate
	switch {
	case t.ID == adenaID:
		rate = r.RewardAdenaRate
	case r.UseRewardMultipliers && t.Etc != nil:
		switch t.Etc.Type {
		case data.EtcPotion:
			rate = r.RewardPotionRate
		case data.EtcScrlEnchantWp, data.EtcScrlEnchantAm, data.EtcScroll:
			rate = r.RewardScrollRate
		case data.EtcRecipe:
			rate = r.RewardRecipeRate
		case data.EtcMaterial:
			rate = r.RewardMaterialRate
		}
	case r.UseRewardMultipliers:
		rate = 1
	}
	return clampCount(float64(count) * rate)
}

func clampCount(v float64) int64 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int64(v)
}

// giveQuestItems is Mobius giveItems (reward false: the quest item
// multiplier for quest items, the enchant level for everything but adena)
// and rewardItems (reward true: the reward rates), with "You have earned"
// messages.
func (s *Server) giveQuestItems(p *Player, itemID int, count int64, enchant int, reward bool) {
	t, ok := s.data.ItemTemplate(itemID)
	if !ok || count <= 0 {
		return
	}
	switch {
	case reward:
		count = s.questRewardCount(t, count)
	case t.QuestItem:
		count = clampCount(float64(count) * s.quest.settings.ItemDropAmount)
	}
	if count <= 0 {
		return
	}
	if itemID == adenaID {
		s.AddAdena(p, count, true)
		return
	}
	ench := int32(-1)
	if enchant > 0 {
		ench = int32(enchant) //nolint:gosec // small
	}
	if _, err := s.AddItem(p, int32(itemID), count, ench, true, true); err != nil { //nolint:gosec // item IDs fit in int32
		s.log.Warn("game: a quest could not give an item", append(p.logArgs(), "item", itemID, "count", count, "err", err)...)
	}
}

// takeQuestItems is Mobius takeItems(player, id, count): count items of
// the template, all of them for a negative count; a worn item comes off
// first, with its InventoryUpdate and UserInfo.
func (s *Server) takeQuestItems(p *Player, itemID int, count int64) {
	if p.inv.Inventory == nil || count == 0 {
		return
	}
	items := append([]*inventory.Item(nil), p.inv.AllByItemID(int32(itemID))...) //nolint:gosec // item IDs fit in int32
	var taken int64
	for _, it := range items {
		n := it.Count()
		if count >= 0 {
			n = min(n, count-taken)
		}
		if n <= 0 {
			continue
		}
		if it.Equipped() {
			p.inv.UnequipSlot(it.Slot())
			s.sendInventoryUpdateNow(p, s.takeInventoryUpdates(p, nil))
			p.RefreshStats()
			p.BroadcastUserInfo()
		}
		if it.ItemID() == adenaID {
			s.ReduceAdena(p, n, true)
		} else {
			s.DestroyItem(p, it, n, true)
		}
		taken += n
	}
}

// addQuestExpAndSP is Mobius Quest.addExpAndSp: the reward with the quest
// exp and SP rates.
func (s *Server) addQuestExpAndSP(p *Player, exp, sp int64) {
	r := s.quest.settings
	exp = int64(float64(exp) * r.RewardXPRate)
	sp = int64(float64(sp) * r.RewardSPRate)
	s.AddExpAndSP(p, statsReward(exp, sp))
}
