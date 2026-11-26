package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/skills"
)

// The summon item handlers (Mobius SummonItems, PetFood, BeastSoulShot,
// BeastSpiritShot) and the summon's automatic shots (Summon.rechargeShots).

// The item handler names of the summon items.
const (
	handlerSummonItems     = "SummonItems"
	handlerPetFood         = "PetFood"
	handlerBeastSoulShot   = "BeastSoulShot"
	handlerBeastSpiritShot = "BeastSpiritShot"
)

// System messages of the summon items (Mobius SystemMessageId).
const (
	sumMsgNoPetsNow           serverpackets.SystemMessageID = 574  // "Pets and Servitors are not available at this time."
	sumMsgPetPowerOfSpirit    serverpackets.SystemMessageID = 1576 // "Pet uses the power of spirit."
	sumMsgNoShotsDeadPet      serverpackets.SystemMessageID = 1598 // "Soulshots and spiritshots are not available for a dead pet or servitor. Sad, isn't it?"
	sumMsgNotEnoughPetSpirits serverpackets.SystemMessageID = 1700 // "You don't have enough spiritshots needed for a pet/servitor."
	sumMsgNotEnoughPetShots   serverpackets.SystemMessageID = 1701 // "You don't have enough soulshots needed for a pet/servitor."
)

// runSummonItemHandler runs the summon item handlers; known is false for
// any other handler.
func (s *Server) runSummonItemHandler(p *Player, it *inventory.Item, ctrl bool) (used, known bool) {
	switch it.Template().Etc.Handler {
	case handlerSummonItems:
		return s.useSummonItem(p, it, ctrl), true
	case handlerPetFood:
		return s.usePetFoodAsPlayer(p, it), true
	case handlerBeastSoulShot:
		return s.useBeastShot(p, it, false), true
	case handlerBeastSpiritShot:
		return s.useBeastShot(p, it, true), true
	}
	return false, false
}

// useSummonItem is Mobius SummonItems.onItemUse: a collar's summoning skill
// is cast unless the player cannot summon now; the collar goes with it to
// SummonPet (PetItemHolder).
func (s *Server) useSummonItem(p *Player, it *inventory.Item, ctrl bool) bool {
	if s.CastingNow(p) {
		return false
	}
	if p.Sitting {
		p.SendSystemMessage(sysMsgCannotMoveSitting)
		return false
	}
	if s.SummonOf(p) != nil || s.mounted(p) {
		p.SendSystemMessage(sumMsgYouAlreadyHaveAPet)
		return false
	}
	if s.AttackingNow(p) || s.InCombat(p) {
		p.SendSystemMessage(sumMsgCannotSummonInCombat)
		return false
	}
	if s.Trading(p) || p.InStoreMode() {
		p.SendSystemMessage(sumMsgCannotSummonInTrade)
		return false
	}
	if s.summons.data == nil || !s.summons.data.IsPetItem(int(it.ItemID())) {
		return false
	}
	s.summonOwnerOf(p).collar = it.ObjectID()
	return s.useItemSkills(p, it, ctrl)
}

// usePetFoodAsPlayer is PetFood.onItemUse for a player: a rider feeds its
// mount; anyone else gets "$s1 cannot be used due to unsuitable terms.".
func (s *Server) usePetFoodAsPlayer(p *Player, it *inventory.Item) bool {
	if len(normalSkills(it.Template())) == 0 {
		return false
	}
	return s.mountEat(p, it) // mount.go
}

// useBeastShot is Mobius BeastSoulShot and BeastSpiritShot.onItemUse: the
// summon's shots per hit are used up and it is charged ("Pet uses the power
// of spirit."), the glow of the shot's skill on the summon to everyone.
func (s *Server) useBeastShot(p *Player, it *inventory.Item, spirit bool) bool {
	sm := s.SummonOf(p)
	if sm == nil {
		p.SendSystemMessage(sumMsgNoPetsNow)
		return false
	}
	if sm.npc.Dead {
		p.SendSystemMessage(sumMsgNoShotsDeadPet)
		return false
	}
	refs := normalSkills(it.Template())
	if len(refs) == 0 {
		return false
	}
	ss, sps := s.summonShotsPerHit(sm)
	count := int64(ss)
	kind := skills.Soulshot
	notEnough := sumMsgNotEnoughPetShots
	if spirit {
		count = int64(sps)
		kind = skills.Spiritshot
		if id := it.ItemID(); id == 6647 || id == 20334 {
			kind = skills.BlessedSpiritshot
		}
		notEnough = sumMsgNotEnoughPetSpirits
	}
	if it.Count() < count {
		if !s.disableAutoShot(p, it.ItemID()) {
			p.SendSystemMessage(notEnough)
		}
		return false
	}
	if sm.npc.skill.ChargedShot(kind) {
		return false
	}
	if !s.DestroyItem(p, it, count, false) {
		if !s.disableAutoShot(p, it.ItemID()) {
			p.SendSystemMessage(notEnough)
		}
		return false
	}
	p.SendSystemMessage(sumMsgPetPowerOfSpirit)
	sm.npc.skill.SetChargedShot(kind, true)
	n := sm.npc
	s.world.Broadcast(p, castUsePacket(n, n, refs[0].ID, refs[0].Level, 0, 0), true)
	return true
}

// rechargeSummonShots is Summon.rechargeShots(true, true) when it comes
// out: the owner's automatic beast shots charge it.
func (s *Server) rechargeSummonShots(sm *Summon) { s.summonRechargeShots(sm, true, true) }

// summonRechargeShots is Summon.rechargeShots: each automatic shot of the
// owner with a summon action runs its handler; one that is gone leaves the
// set.
func (s *Server) summonRechargeShots(sm *Summon, physical, magic bool) {
	p := sm.owner
	if p == nil || p.inv.Inventory == nil {
		return
	}
	for _, id := range p.itemSkill.autoShots.list() {
		it := p.inv.ByItemID(id)
		if it == nil {
			p.itemSkill.autoShots.remove(id)
			continue
		}
		t := it.Template()
		if t.Etc == nil {
			continue
		}
		switch {
		case magic && t.DefaultAction == data.ActionSummonSpiritshot:
			s.useBeastShot(p, it, true)
		case physical && t.DefaultAction == data.ActionSummonSoulshot:
			s.useBeastShot(p, it, false)
		}
	}
}
