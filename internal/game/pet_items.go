package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/summon"
	"github.com/IPeter/l2go/internal/world"
)

// The pet inventory (Mobius PetInventory): the owner's items at LocPet and
// LocPetEquip, 12 slots (MaximumSlotsForPet), kept with the player as their
// owner, so every move is one save of the player (decision I2 holds without
// a second owner). RequestGiveItemToPet, RequestGetItemFromPet,
// RequestPetUseItem (equipment and food), RequestPetGetItem and
// RequestChangePetName.

// petInventoryLimit is MaximumSlotsForPet (Mobius NpcConfig, 12).
const petInventoryLimit = 12

// System messages of the pet inventory (Mobius SystemMessageId).
const (
	sumMsgFailedPickUp       serverpackets.SystemMessageID = 56   // "You have failed to pick up $s1."
	sumMsgPetCannotCarryItem serverpackets.SystemMessageID = 544  // "Your pet cannot carry this item."
	sumMsgPetCannotCarryMore serverpackets.SystemMessageID = 545  // "Your pet cannot carry any more items. ..."
	sumMsgPetTooEncumbered   serverpackets.SystemMessageID = 546  // "Unable to place item, your pet is too encumbered."
	sumMsgPetNameTooLong     serverpackets.SystemMessageID = 548  // "Your pet's name can be up to 8 characters in length."
	sumMsgSpaceInName        serverpackets.SystemMessageID = 581  // "There is a space in the name."
	sumMsgPetMotionless      serverpackets.SystemMessageID = 590  // "Your pet is motionless and ..."
	sumMsgInvalidPetName     serverpackets.SystemMessageID = 591  // "An invalid character is included in the pet's name."
	sumMsgCannotNamePet      serverpackets.SystemMessageID = 695  // "You cannot set the name of the pet."
	sumMsgPetPickedUp        serverpackets.SystemMessageID = 1020 // "Your pet picked up $s1."
	sumMsgPetPickedUpCount   serverpackets.SystemMessageID = 1021 // "Your pet picked up $s2 $s1(s)."
	sumMsgPetPickedUpAdena   serverpackets.SystemMessageID = 1023 // "Your pet picked up $s1 adena."
	sumMsgPetPutOn           serverpackets.SystemMessageID = 1024 // "Your pet put on $s1."
	sumMsgPetTookOff         serverpackets.SystemMessageID = 1025 // "Your pet took off $s1."
)

// Texts Mobius sends with sendMessage about the pet inventory.
const (
	summonTextPetItemsLeft     = "There are items in your Pet Inventory rendering you unable to sell/trade/drop pet summoning items. Please empty your Pet Inventory."
	summonTextGiveWhileTrading = "You cannot give items while trading."
)

// The pet equipment slots (Pet.getWeapon, getArmor, getJewel).
const (
	petSlotWeapon = iota
	petSlotArmor
	petSlotJewel
	petSlotNone = -1
)

// petSlotOf is the pet slot of a body part.
func petSlotOf(t *data.Item) int {
	switch t.Part() {
	case data.BodyPartRHand, data.BodyPartLRHand:
		return petSlotWeapon
	case data.BodyPartChest, data.BodyPartFullArmor:
		return petSlotArmor
	case data.BodyPartNeck:
		return petSlotJewel
	}
	return petSlotNone
}

// petWornItemID returns the item ID the pet wears in slot, 0 for none.
func (s *Server) petWornItemID(p *Player, slot int) int32 {
	if it := petWorn(p, slot); it != nil {
		return it.ItemID()
	}
	return 0
}

// petWorn returns the item of p's pet inventory worn in slot, nil for none.
func petWorn(p *Player, slot int) *inventory.Item {
	for _, it := range petInventoryItems(p) {
		if it.PetEquipped() && petSlotOf(it.Template()) == slot {
			return it
		}
	}
	return nil
}

// petItemInfo is ItemInfo for a pet inventory item: worn means PET_EQUIP.
func petItemInfo(it *inventory.Item) serverpackets.ItemInfo {
	info := ItemInfo(it)
	info.Equipped = it.PetEquipped()
	return info
}

// petItemListPacket builds PetItemList of p's pet inventory.
func (s *Server) petItemListPacket(p *Player) []byte {
	items := petInventoryItems(p)
	infos := make([]serverpackets.ItemInfo, 0, len(items))
	for _, it := range items {
		infos = append(infos, petItemInfo(it))
	}
	return serverpackets.NewPetItemListPacket(serverpackets.PetItemList{Items: infos})
}

// petItemsChanged sends the owner PetInventoryUpdate of the pet inventory's
// changes, its own InventoryUpdate, and asks for a save; the pet's windows
// follow its new load.
func (s *Server) petItemsChanged(p *Player) {
	if pet := p.items.pet; pet != nil {
		var out []serverpackets.InventoryItem
		for _, u := range pet.TakeUpdates() {
			out = append(out, serverpackets.InventoryItem{Change: serverpackets.ItemChange(u.Change), Item: petItemInfo(u.Item)})
		}
		if len(out) > 0 && p.InGame() {
			p.Send(serverpackets.NewPetInventoryUpdatePacket(serverpackets.PetInventoryUpdate{Items: out}))
		}
	}
	s.ItemsChanged(p)
	if sm := s.SummonOf(p); sm != nil && sm.pet != nil {
		s.summonBroadcastStatus(sm, 1)
	}
}

// petLoad is the weight the pet carries (PetInventory.getTotalWeight).
func (s *Server) petLoad(sm *Summon) int32 {
	var w int64
	for _, it := range petInventoryItems(sm.owner) {
		w += int64(it.Template().Weight) * it.Count()
	}
	return int32(min(w, 1<<31-1)) //nolint:gosec // clamped
}

// petMaxLoad is the pet's weight limit (Pet.getMaxLoad: the pet data's load
// through the weight limit stat).
func (s *Server) petMaxLoad(sm *Summon) int32 {
	return javaInt32(sm.npc.Stats.Calc(stats.WeightLimit, float64(sm.pet.data.Load)))
}

// petRefreshOverloaded is Pet.refreshOverloaded: the weight penalty skill
// 4270 at the level of the load ratio.
func (s *Server) petRefreshOverloaded(sm *Summon) {
	maxLoad := int64(s.petMaxLoad(sm))
	if maxLoad <= 0 {
		return
	}
	ratio := int64(s.petLoad(sm)) * 1000 / maxLoad
	level := 0
	switch {
	case ratio < 500 || sm.owner.admin.diet:
	case ratio < 666:
		level = 1
	case ratio < 800:
		level = 2
	case ratio < 1000:
		level = 3
	default:
		level = 4
	}
	if level == sm.pet.weightPenalty {
		return
	}
	sm.pet.weightPenalty = level
	n := sm.npc
	n.Stats.RemoveOwner(weightPenaltyOwner)
	if level > 0 {
		_ = n.Stats.AddModifiers(s.items.weightMods[level]...)
	}
	n.cacheOK = false
}

// petUnsummonNotice is Summon.unSummon's message about items left in the
// pet inventory.
func (s *Server) petUnsummonNotice(sm *Summon) {
	if len(petInventoryItems(sm.owner)) > 0 {
		sm.owner.SendMessage(summonTextPetItemsLeft)
	}
}

// petDestroyItem takes count of an item out of the pet inventory (Mobius
// Pet.destroyItem), with PetInventoryUpdate.
func (s *Server) petDestroyItem(p *Player, it *inventory.Item, count int64) bool {
	pet := p.items.pet
	if pet == nil {
		return false
	}
	if _, err := pet.Destroy(it, count); err != nil {
		return false
	}
	s.petItemsChanged(p)
	return true
}

// petItemsToOwner is PetInventory.transferItemsToOwner: every pet item goes
// back into the owner's inventory.
func (s *Server) petItemsToOwner(p *Player) {
	pet := p.items.pet
	if pet == nil || p.inv.Inventory == nil {
		return
	}
	for _, it := range append([]*inventory.Item(nil), pet.Items()...) {
		if _, err := pet.Transfer(it.ObjectID(), it.Count(), &p.inv.Container); err != nil {
			s.log.Warn("game: cannot give a pet item back", append(p.logArgs(), "item", it.ItemID(), "err", err)...)
		}
	}
	s.petItemsChanged(p)
}

// handleRequestGiveItemToPet handles RequestGiveItemToPet (0x8B): count of
// an item goes from the owner's inventory into the pet's (Mobius
// RequestGiveItemToPet).
func (s *Server) handleRequestGiveItemToPet(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestGiveItemToPet(body)
	if err != nil {
		s.log.Debug("game: bad RequestGiveItemToPet", append(p.logArgs(), "err", err)...)
		return
	}
	sm := s.SummonOf(p)
	if req.Count <= 0 || sm == nil || sm.pet == nil || p.items.pet == nil {
		return
	}
	if p.char.Karma > 0 { // AltKarmaPlayerCanTrade = False
		return
	}
	if p.InStoreMode() {
		p.SendMessage(summonTextGiveWhileTrading)
		return
	}
	it := p.inv.ByObjectID(req.ObjectID)
	if it == nil || int64(req.Count) > it.Count() || it.Equipped() {
		return
	}
	t := it.Template()
	if t.Hero || !t.Droppable || !t.Destroyable || !t.Tradable {
		p.SendSystemMessage(sumMsgPetCannotCarryItem)
		return
	}
	if sm.npc.Dead {
		p.SendSystemMessage(sumMsgPetMotionless)
		return
	}
	pet := p.items.pet
	slots := int64(1)
	if t.Stackable && pet.ByItemID(it.ItemID()) != nil {
		slots = 0
	}
	if !pet.ValidateCapacity(slots) {
		p.SendSystemMessage(sumMsgPetCannotCarryMore)
		return
	}
	if int64(s.petLoad(sm))+int64(t.Weight)*int64(req.Count) > int64(s.petMaxLoad(sm)) {
		p.SendSystemMessage(sumMsgPetTooEncumbered)
		return
	}
	if _, err := p.inv.Transfer(req.ObjectID, int64(req.Count), pet); err != nil {
		s.log.Debug("game: invalid item transfer to a pet", append(p.logArgs(), "err", err)...)
		return
	}
	s.petItemsChanged(p)
}

// handleRequestGetItemFromPet handles RequestGetItemFromPet (0x8C): count of
// a pet item goes back into the owner's inventory; a worn one comes off
// first.
func (s *Server) handleRequestGetItemFromPet(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestGetItemFromPet(body)
	if err != nil {
		s.log.Debug("game: bad RequestGetItemFromPet", append(p.logArgs(), "err", err)...)
		return
	}
	sm := s.SummonOf(p)
	if req.Count <= 0 || sm == nil || sm.pet == nil || p.items.pet == nil {
		return
	}
	pet := p.items.pet
	it := pet.ByObjectID(req.ObjectID)
	if it == nil || int64(req.Count) > it.Count() {
		return
	}
	if it.PetEquipped() {
		s.petUnequip(sm, it)
	}
	if _, err := pet.Transfer(req.ObjectID, int64(req.Count), &p.inv.Container); err != nil {
		s.log.Debug("game: invalid item transfer from a pet", append(p.logArgs(), "err", err)...)
		return
	}
	s.petItemsChanged(p)
}

// handleRequestPetUseItem handles RequestPetUseItem (0x8A): the pet puts on
// or takes off a piece of its equipment, or eats its food (Mobius
// RequestPetUseItem).
func (s *Server) handleRequestPetUseItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPetUseItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestPetUseItem", append(p.logArgs(), "err", err)...)
		return
	}
	sm := s.SummonOf(p)
	if sm == nil || sm.pet == nil || p.items.pet == nil {
		return
	}
	it := p.items.pet.ByObjectID(req.ObjectID)
	if it == nil {
		return
	}
	t := it.Template()
	if !t.ForNpc {
		p.SendSystemMessage(sumMsgPetCannotUseItem)
		return
	}
	if p.IsDead() || sm.npc.Dead {
		p.SendSystemMessage(sumMsgUnsuitableTerms, serverpackets.ItemNameParam(it.ItemID()))
		return
	}
	if it.Equipable() {
		if t.Conditions == nil {
			p.SendSystemMessage(sumMsgPetCannotUseItem)
			return
		}
		if it.PetEquipped() {
			s.petUnequip(sm, it)
			p.SendSystemMessage(sumMsgPetTookOff, serverpackets.ItemNameParam(it.ItemID()))
		} else {
			s.petEquip(sm, it)
			p.SendSystemMessage(sumMsgPetPutOn, serverpackets.ItemNameParam(it.ItemID()))
		}
		p.items.pet.TakeUpdates()
		s.ItemsChanged(p)
		p.Send(s.petItemListPacket(p))
		s.summonBroadcastStatus(sm, 1)
		return
	}
	if t.Etc != nil && t.Etc.Handler == handlerPetFood {
		if s.petEat(sm, it) {
			s.summonBroadcastStatus(sm, 1)
		}
		return
	}
	p.SendSystemMessage(sumMsgPetCannotUseItem)
}

// petEquip puts it on the pet: what it wore in the slot comes off, and the
// item's stats go on the pet.
func (s *Server) petEquip(sm *Summon, it *inventory.Item) {
	slot := petSlotOf(it.Template())
	if old := petWorn(sm.owner, slot); old != nil && slot != petSlotNone {
		s.petUnequip(sm, old)
	}
	sm.owner.items.pet.SetPetEquipped(it, true)
	if mods, err := stats.ItemModifiers(stats.ItemOwner(it.ObjectID()), inventory.StatsItem(it.Template(), int(it.Enchant()))); err == nil {
		_ = sm.npc.Stats.AddModifiers(mods...)
	}
	sm.npc.cacheOK = false
}

// petUnequip takes it off the pet with its stats.
func (s *Server) petUnequip(sm *Summon, it *inventory.Item) {
	sm.owner.items.pet.SetPetEquipped(it, false)
	sm.npc.Stats.RemoveOwner(stats.ItemOwner(it.ObjectID()))
	sm.npc.cacheOK = false
}

// petWornStats puts the stats of the items the pet wears on it when it
// comes out.
func (s *Server) petWornStats(sm *Summon) {
	for _, it := range petInventoryItems(sm.owner) {
		if !it.PetEquipped() {
			continue
		}
		if mods, err := stats.ItemModifiers(stats.ItemOwner(it.ObjectID()), inventory.StatsItem(it.Template(), int(it.Enchant()))); err == nil {
			_ = sm.npc.Stats.AddModifiers(mods...)
		}
	}
	sm.npc.cacheOK = false
}

// handleRequestChangePetName handles RequestChangePetName (0x89): a pet
// without a name gets one of 1 to 8 letters and digits (Mobius
// RequestChangePetName; names are not checked against other pets').
func (s *Server) handleRequestChangePetName(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestChangePetName(body)
	if err != nil {
		s.log.Debug("game: bad RequestChangePetName", append(p.logArgs(), "err", err)...)
		return
	}
	sm := s.SummonOf(p)
	if sm == nil {
		return
	}
	if sm.pet == nil {
		p.SendMessage(summonTextNoPet)
		return
	}
	name := req.Name
	switch {
	case sm.name != "":
		p.SendSystemMessage(sumMsgCannotNamePet)
	case len([]rune(name)) == 0 || len([]rune(name)) > 8:
		p.SendSystemMessage(sumMsgPetNameTooLong)
	case containsSpace(name):
		p.SendSystemMessage(sumMsgSpaceInName)
	case !validPetName(name):
		p.SendSystemMessage(sumMsgInvalidPetName)
	default:
		sm.name = name
		s.petStoreMe(sm)
		s.summonBroadcastStatus(sm, 1)
	}
}

func containsSpace(name string) bool {
	for _, r := range name {
		if r == ' ' {
			return true
		}
	}
	return false
}

// validPetName is Mobius PetNameTable.isValidPetName with the default
// PetNameTemplate ([A-Za-z0-9]{2,8}).
func validPetName(name string) bool {
	if len(name) < 2 || len(name) > 8 {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// handleRequestPetGetItem handles RequestPetGetItem (0x8F): the pet walks
// to a ground item and picks it up (Mobius RequestPetGetItem and
// SummonAI.thinkPickUp).
func (s *Server) handleRequestPetGetItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestPetGetItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestPetGetItem", append(p.logArgs(), "err", err)...)
		return
	}
	sm := s.SummonOf(p)
	o, ok := s.World().Find(world.ObjectID(req.ObjectID))
	g, isItem := o.(*GroundItem)
	if !ok || !isItem || sm == nil || sm.pet == nil || sm.npc.Dead || npcMovementDisabled(sm.npc) || summon.Uncontrollable(sm.pet.fed) {
		p.SendActionFailed()
		return
	}
	sm.ai.followTarget = nil
	sm.ai.intention = summonPickingUp
	sm.ai.pickUp = g
	s.summonThink(sm)
}

// petThinkPickUp is SummonAI.thinkPickUp: the pet walks up to the item and
// picks it up.
func (s *Server) petThinkPickUp(sm *Summon) {
	g, ok := sm.ai.pickUp.(*GroundItem)
	if !ok || g == nil || !g.InWorld() {
		sm.ai.pickUp = nil
		s.summonSetActive(sm)
		return
	}
	n := sm.npc
	if !within2D(n.Position(), g.Position(), summon.PickUpRange+int(npcCollision(n))) {
		s.NpcMoveToPawn(n, g, summon.PickUpRange)
		return
	}
	sm.ai.pickUp = nil
	follow := sm.follow
	s.summonSetIdle(sm)
	s.petPickUp(sm, g)
	if follow {
		s.summonSetFollowStatus(sm, true)
	}
}

// petPickUp is Mobius Pet.doPickupItem: the pet stops, the protected or
// too heavy item stays, otherwise it leaves the ground into the pet
// inventory ("Your pet picked up $s1.").
func (s *Server) petPickUp(sm *Summon, g *GroundItem) {
	p := sm.owner
	n := sm.npc
	s.world.Broadcast(n, npcStopMovePacket(n), false)
	it := g.item
	if g.protectedFor != 0 && g.protectedFor != p.ID() && s.Now().Before(g.protectedUntil) && !s.inLooterParty(p, g.protectedFor) {
		p.SendActionFailed()
		p.SendSystemMessage(sumMsgFailedPickUp, s.itemNameParam(it.ItemID()))
		return
	}
	pet := p.items.pet
	slots := int64(1)
	if it.Stackable() && pet.ByItemID(it.ItemID()) != nil {
		slots = 0
	}
	if !pet.ValidateCapacity(slots) {
		p.SendActionFailed()
		p.SendSystemMessage(sumMsgPetTooEncumbered)
		return
	}
	pos := g.Position()
	s.world.Broadcast(n, serverpackets.NewGetItemPacket(serverpackets.GetItem{
		PlayerID: int32(n.ID()), ObjectID: it.ObjectID(), X: pos.X, Y: pos.Y, Z: pos.Z,
	}), false)
	s.removeGroundItem(g)
	name := s.itemNameParam(it.ItemID())
	switch {
	case it.ItemID() == inventory.AdenaID:
		p.SendSystemMessage(sumMsgPetPickedUpAdena, serverpackets.NumberParam(itemCount32(it.Count())))
	case it.Count() > 1:
		p.SendSystemMessage(sumMsgPetPickedUpCount, serverpackets.NumberParam(itemCount32(it.Count())), name)
	default:
		p.SendSystemMessage(sumMsgPetPickedUp, name)
	}
	if _, err := pet.Add(it); err != nil {
		s.log.Warn("game: a pet cannot take an item", append(p.logArgs(), "item", it.ItemID(), "err", err)...)
		return
	}
	s.petItemsChanged(p)
}
