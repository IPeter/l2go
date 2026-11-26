package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/summon"
)

// Pet hunger (Mobius Pet.FeedTask and the PetFood item handler): every 10 s
// a living pet eats its level's meal rate (the battle rate while it swings);
// a hungry pet eats its food from its inventory; a starving one may leave.

// System messages of feeding (Mobius SystemMessageId).
const (
	sumMsgUnsuitableTerms   serverpackets.SystemMessageID = 113  // "$s1 cannot be used due to unsuitable terms."
	sumMsgStarvingPetLeft   serverpackets.SystemMessageID = 593  // "Starving, grumpy and fed up, your pet has left."
	sumMsgPetAteStillHungry serverpackets.SystemMessageID = 596  // "Your pet ate a little, but is still hungry."
	sumMsgPetVeryHungry     serverpackets.SystemMessageID = 597  // "Your pet is very hungry. Please be careful."
	sumMsgPetCannotUseItem  serverpackets.SystemMessageID = 972  // "This pet cannot use this item."
	sumMsgPetWasHungryAte   serverpackets.SystemMessageID = 1527 // "Your pet was hungry so it ate $s1."
)

// petStartFeed is Pet.startFeed: the feed task starts over (first run 10 s
// later) for a living pet out with its owner.
func (s *Server) petStartFeed(sm *Summon) {
	s.petStopFeed(sm)
	if sm.npc.Dead || s.SummonOf(sm.owner) != sm {
		return
	}
	sm.pet.feedTimer = s.loop.Every(summon.FeedTick, func() { s.petFeedTick(sm) })
}

// petStopFeed is Pet.stopFeed.
func (s *Server) petStopFeed(sm *Summon) {
	if sm.pet != nil && sm.pet.feedTimer != 0 {
		s.loop.Cancel(sm.pet.feedTimer)
		sm.pet.feedTimer = 0
	}
}

// petFeedTick is Mobius FeedTask.run: the meal goes down; a starving pet
// leaves on 30 % of the ticks ("Starving, grumpy and fed up, your pet has
// left.") or refuses to obey; a pet without food in the data warns when
// hungry; a hungry pet eats the first of its foods it carries ("Your pet
// was hungry so it ate $s1."), and warns again if still starving.
func (s *Server) petFeedTick(sm *Summon) {
	ps := sm.pet
	owner := sm.owner
	if s.SummonOf(owner) != sm || sm.npc.Dead {
		s.petStopFeed(sm)
		return
	}
	l := ps.levelData()
	if l == nil {
		return
	}
	consume := summon.FeedConsume(s.AttackingNow(sm.npc), l.MealInBattle, l.MealInNormal)
	ps.fed = summon.ClampFed(summon.AfterFeedTick(ps.fed, consume), ps.maxMeal())
	s.summonFedChanged(sm)
	if summon.Uncontrollable(ps.fed) {
		if s.rnd.Get(100) < summon.StarvingLeaveChance {
			owner.SendSystemMessage(sumMsgStarvingPetLeft)
			s.unsummonNow(sm)
			s.petStopFeed(sm)
			return
		}
		owner.SendSystemMessage(sumMsgPetUnresponsive)
		return
	}
	if len(ps.data.Food) == 0 {
		if s.petHungry(sm) {
			owner.SendSystemMessage(sumMsgPetVeryHungry)
		}
		return
	}
	var food *inventory.Item
	for _, id := range ps.data.Food {
		if owner.items.pet != nil {
			if food = owner.items.pet.ByItemID(int32(id)); food != nil { //nolint:gosec // item IDs are small
				break
			}
		}
	}
	if food != nil && s.petHungry(sm) {
		owner.SendSystemMessage(sumMsgPetWasHungryAte, serverpackets.ItemNameParam(food.ItemID()))
		s.petEat(sm, food)
	}
	if summon.Uncontrollable(ps.fed) {
		owner.SendSystemMessage(sumMsgPetUnresponsive)
	}
}

// summonFedChanged shows the new meal (Pet.broadcastStatusUpdate): the
// owner's status window, and a starving pet's halved speeds.
func (s *Server) summonFedChanged(sm *Summon) {
	s.summonStatusUpdate(sm)
	s.petHungerSpeed(sm)
}

// petEat is the PetFood handler for a pet eating food from its inventory:
// the food goes, the pet shows the item's skill, its meal grows by the
// skill's feed and it warns when still hungry. It reports whether it ate.
func (s *Server) petEat(sm *Summon, food *inventory.Item) bool {
	ps := sm.pet
	owner := sm.owner
	if !canEatFood(ps.data.Food, int(food.ItemID())) {
		owner.SendSystemMessage(sumMsgPetCannotUseItem)
		return false
	}
	ate := false
	for _, ref := range food.Template().Skills {
		sk, ok := s.Skill(ref.ID, ref.Level)
		if !ok {
			continue
		}
		if !s.petDestroyItem(owner, food, 1) {
			return ate
		}
		ate = true
		s.world.Broadcast(sm.npc, castUsePacket(sm.npc, sm.npc, sk.ID, sk.Level, 0, 0), false)
		ps.fed = summon.ClampFed(ps.fed+sk.Feed*petFoodRate, ps.maxMeal())
		s.summonFedChanged(sm)
		if s.petHungry(sm) {
			owner.SendSystemMessage(sumMsgPetAteStillHungry)
		}
	}
	return ate
}

// canEatFood is Pet.canEatFoodId.
func canEatFood(foods []int, itemID int) bool {
	for _, f := range foods {
		if f == itemID {
			return true
		}
	}
	return false
}

// The stat modifiers of hunger: a starving pet moves at half speed
// (Pet.getRunSpeed and friends × 0.5 while uncontrollable), a hungry one
// attacks and casts at half speed (PetStat.getPAtkSpd and getMAtkSpd / 2).
var (
	petStarvingOwner     = stats.Owner{Kind: stats.OwnerSkill, ID: -1}
	petHungryOwner       = stats.Owner{Kind: stats.OwnerSkill, ID: -2}
	petStarvingModifiers = []stats.Modifier{stats.NewModifier(stats.RunSpd, stats.OpMul, 0.5, petStarvingOwner)}
	petHungryModifiers   = []stats.Modifier{
		stats.NewModifier(stats.PAtkSpd, stats.OpMul, 0.5, petHungryOwner),
		stats.NewModifier(stats.MAtkSpd, stats.OpMul, 0.5, petHungryOwner),
	}
)

// petHungerSpeed puts the hunger modifiers on or off as the meal changes,
// and shows the new speeds.
func (s *Server) petHungerSpeed(sm *Summon) {
	n := sm.npc
	changed := false
	for _, h := range []struct {
		on    bool
		owner stats.Owner
		mods  []stats.Modifier
	}{
		{summon.Uncontrollable(sm.pet.fed), petStarvingOwner, petStarvingModifiers},
		{s.petHungry(sm), petHungryOwner, petHungryModifiers},
	} {
		had := n.Stats.RemoveOwner(h.owner) > 0
		if h.on {
			_ = n.Stats.AddModifiers(h.mods...)
		}
		changed = changed || had != h.on
	}
	if changed {
		n.cacheOK = false
		s.summonBroadcastStatus(sm, 1)
	}
}
