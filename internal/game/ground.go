package game

import (
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

const (
	// dropMaxDistance and dropMaxHeight are how far from the player an item
	// may be dropped: 150 units in 2D, 50 in height (Mobius RequestDropItem).
	dropMaxDistance = 150
	dropMaxHeight   = 50
	// dropProtectionTime is how long a monster drop is kept for its killer
	// (Mobius DropProtection).
	dropProtectionTime = 15 * time.Second
	// pickUpOffset is how far from an item a player walking to it stops
	// (Mobius setIntentionPickUp: moveToPawn(object, 20)); pickUpRange is how
	// close it must be to pick it up (PlayerAI.thinkPickUp:
	// maybeMoveToPawn(target, 36)).
	pickUpOffset = 20
	pickUpRange  = 36
	// pickUpTimeout gives up a walk up to an item that never ends.
	pickUpTimeout = time.Minute
	// defaultGroundTime is how long a drop stays when
	// AutoDestroyDroppedItemAfter is 0 but the item is timed anyway (Mobius
	// ItemsAutoDestroyTaskManager: one hour).
	defaultGroundTime = time.Hour
	// bigAdenaDrop is the adena drop Mobius logs and tells the GMs about.
	bigAdenaDrop = 1_000_000
)

// GroundItem is an item lying on the ground (Mobius Item at
// ItemLocation.VOID in the world). It keeps the item's object ID. Ground
// items are never saved: a restart clears the ground, as in Mobius
// (SaveDroppedItem = False).
type GroundItem struct {
	world.Base
	item *inventory.Item

	// dropper is the character that dropped the item while it enters the
	// world: the players who see it fall get DropItem, later ones SpawnItem.
	dropper world.ObjectID
	// protectedFor may pick the item up alone until protectedUntil (Mobius
	// DropProtection, for monster drops).
	protectedFor   world.ObjectID
	protectedUntil time.Time
	timer          world.TimerID // the auto-destroy timer, or 0
}

var _ InfoProvider = (*GroundItem)(nil)

// Item returns the item that lies there.
func (g *GroundItem) Item() *inventory.Item { return g.item }

// packet is DropItem or SpawnItem for the item.
func (g *GroundItem) packet() []byte {
	pos := g.Position()
	t := g.item.Template()
	return serverpackets.NewGroundItemPacket(serverpackets.GroundItem{
		ObjectID:  g.item.ObjectID(),
		ItemID:    int32(t.DisplayID), //nolint:gosec // item IDs fit in int32
		X:         pos.X,
		Y:         pos.Y,
		Z:         pos.Z,
		Stackable: g.item.Stackable(),
		Count:     itemCount32(g.item.Count()),
		DropperID: int32(g.dropper),
	})
}

// InfoFor returns what a player that starts to see the item gets (Mobius
// Item.sendInfo): DropItem while it falls, SpawnItem afterwards.
func (g *GroundItem) InfoFor(*Player) [][]byte { return [][]byte{g.packet()} }

// GroundDrop says how an item comes onto the ground.
type GroundDrop struct {
	// Dropper dropped the item: a player, or an NPC for monster drops. The
	// players around see it fall from it. Nil spawns the item without a
	// dropper.
	Dropper world.Object
	// ProtectFor may pick the item up alone for 15 s (Mobius DropProtection,
	// a monster drop's killer); nil for none.
	ProtectFor *Player
	// PlayerDrop marks an item a player dropped: it stays until a restart
	// unless DestroyPlayerDroppedItem is set. Other drops vanish after
	// AutoDestroyDroppedItemAfter (herbs after AutoDestroyHerbTime).
	PlayerDrop bool
}

// DropOnGround puts an item that is in no container onto the ground at pos
// (Mobius Item.dropMe and the auto-destroy rules of decision I6): the
// players around see it fall, and it vanishes when its time is up. A
// dropper places it as far as the geodata lets it go from the dropper.
// Loop-only.
func (s *Server) DropOnGround(it *inventory.Item, pos world.Position, d GroundDrop) (*GroundItem, error) {
	if it.Container() != nil {
		return nil, inventory.ErrInContainer
	}
	instance := int32(0)
	if d.Dropper != nil {
		from := d.Dropper.Position()
		to := s.geo.ValidLocation(geo.Point{X: from.X, Y: from.Y, Z: from.Z}, geo.Point{X: pos.X, Y: pos.Y, Z: pos.Z})
		pos = world.Position{X: to.X, Y: to.Y, Z: to.Z}
		instance = targetInstance(d.Dropper)
	}
	g := &GroundItem{Base: world.NewBase(world.ObjectID(it.ObjectID()), pos), item: it}
	_ = g.SetInstanceID(instance) // not in the world yet
	if d.Dropper != nil {
		g.dropper = d.Dropper.ID()
	}
	if d.ProtectFor != nil {
		g.protectedFor, g.protectedUntil = d.ProtectFor.ID(), s.Now().Add(dropProtectionTime)
	}
	if err := s.AddObject(g); err != nil {
		return nil, err
	}
	g.dropper = 0
	if after, ok := s.groundTime(it.Template(), d.PlayerDrop); ok {
		g.timer = s.loop.After(after, func() {
			g.timer = 0
			if g.InWorld() {
				s.RemoveObject(g)
			}
		})
	}
	return g, nil
}

// groundTime returns how long an item stays on the ground, or false when it
// stays until a restart (Mobius Player.dropItem, Npc.dropItem and
// ItemsAutoDestroyTaskManager).
func (s *Server) groundTime(t *data.Item, playerDrop bool) (time.Duration, bool) {
	cfg := s.gs.Items
	after := time.Duration(cfg.AutoDestroyDroppedItemAfter)
	herbTime := time.Duration(cfg.AutoDestroyHerbTime)
	if slices.Contains(cfg.ListOfProtectedItems, t.ID) {
		return 0, false
	}
	herb := t.ExImmediateEffect
	if playerDrop {
		if after <= 0 || !cfg.DestroyPlayerDroppedItem || (t.Equipable() && !cfg.DestroyEquipableItem) {
			return 0, false
		}
	} else if (after <= 0 || herb) && (herbTime <= 0 || !herb) {
		return 0, false
	}
	switch {
	case t.AutoDestroyTime > 0:
		return time.Duration(t.AutoDestroyTime) * time.Millisecond, true
	case herb:
		return herbTime, true
	case after <= 0:
		return defaultGroundTime, true
	}
	return after, true
}

// removeGroundItem takes an item off the ground.
func (s *Server) removeGroundItem(g *GroundItem) {
	s.loop.Cancel(g.timer)
	g.timer = 0
	if g.InWorld() {
		s.RemoveObject(g)
	}
}

// handleDropItem handles RequestDropItem (0x12) with Mobius's checks in
// order. A worn item comes off first (InventoryUpdate, UserInfo, ItemList),
// then the item, or the dropped part of a stack under a new object ID,
// falls where the client asked.
func (s *Server) handleDropItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestDropItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestDropItem", append(p.logArgs(), "err", err)...)
		return
	}
	if p.IsDead() || !p.InWorld() {
		return
	}
	if s.flood(&p.items.nextDrop, time.Duration(s.gs.Economy.FloodDropItem)) {
		return
	}
	cfg := s.gs.Items
	it := p.inv.ByObjectID(req.ObjectID)
	if it == nil || req.Count == 0 || (!cfg.AllowDiscardItem && !p.IsGM()) || !it.Template().Droppable || p.Zones.Inside(ZoneNoItemDrop) {
		p.SendSystemMessage(sysMsgCannotDiscard)
		return
	}
	t := it.Template()
	if t.QuestItem {
		return
	}
	count := int64(req.Count)
	if count > it.Count() {
		p.SendSystemMessage(sysMsgCannotDiscard)
		return
	}
	if s.SpawnProtected(p) && !p.IsGM() {
		p.SendSystemMessage(sysMsgCannotDiscard)
		return
	}
	if count < 0 {
		s.illegalAction(p, "RequestDropItem with a negative count")
		return
	}
	if !it.Stackable() && count > 1 {
		s.illegalAction(p, "RequestDropItem of more than one item that does not stack")
		return
	}
	if t.Type2 == typeQuest && !p.IsGM() {
		p.SendSystemMessage(sysMsgCannotDiscardOrExchange)
		return
	}
	pos := p.Position()
	dx, dy := float64(req.X-pos.X), float64(req.Y-pos.Y)
	if math.Hypot(dx, dy) > dropMaxDistance || math.Abs(float64(req.Z-pos.Z)) > dropMaxHeight {
		p.SendSystemMessage(sysMsgTooFarToDiscard)
		return
	}
	if it.Equipped() {
		p.inv.UnequipSlot(it.Slot())
		s.itemsChanged(p, false, nil)
		p.BroadcastUserInfo()
		s.SendItemList(p, true)
	}
	dropped, err := p.inv.Drop(req.ObjectID, count)
	if err != nil {
		s.log.Warn("game: cannot drop an item", append(p.logArgs(), "itemObjectId", req.ObjectID, "count", count, "err", err)...)
		p.SendActionFailed()
		return
	}
	s.forgetMana(dropped)
	if _, err := s.DropOnGround(dropped, world.Position{X: req.X, Y: req.Y, Z: req.Z}, GroundDrop{Dropper: p, PlayerDrop: true}); err != nil {
		s.log.Error("game: a dropped item cannot go into the world", append(p.logArgs(), "itemObjectId", dropped.ObjectID(), "err", err)...)
	}
	s.itemsChanged(p, true, nil)
	if dropped.ItemID() == inventory.AdenaID && dropped.Count() >= bigAdenaDrop {
		s.log.Warn("character dropped a lot of adena", append(p.logArgs(), "count", dropped.Count(), "x", req.X, "y", req.Y, "z", req.Z)...)
	}
}

// pickUp is a player walking up to an item to pick it up (Mobius PICK_UP
// intention).
type pickUp struct {
	g       *GroundItem
	dest    world.Position
	moving  bool
	started time.Time
}

// itemClick is a click on an item on the ground (Mobius ItemClick and
// setIntentionPickUp): the player walks up to it (MoveToPawn, 20 units)
// and picks it up once it is close enough.
func (s *Server) itemClick(p *Player, g *GroundItem) {
	delete(s.items.pickups, p)
	if p.Sitting || !g.InWorld() {
		p.SendActionFailed()
		return
	}
	pu := &pickUp{g: g, started: s.Now()}
	s.items.pickups[p] = pu
	if !s.MovePlayerTo(p, g.Position(), pickUpOffset) {
		// Already there (the arrival hook picked it up) or cannot walk.
		if s.items.pickups[p] == pu {
			delete(s.items.pickups, p)
			if s.pickUpReached(p, g) {
				s.doPickUp(p, g)
			}
		}
		return
	}
	if m := p.Move; m != nil {
		pu.dest, pu.moving = m.Destination, true
	}
	pos, to := p.Position(), g.Position()
	s.world.Broadcast(p, serverpackets.NewMoveToPawnPacket(serverpackets.MoveToPawn{
		ObjectID: p.ObjectID(),
		TargetID: g.item.ObjectID(),
		Distance: pickUpOffset,
		X:        pos.X, Y: pos.Y, Z: pos.Z,
		TargetX: to.X, TargetY: to.Y, TargetZ: to.Z,
	}), true)
}

// pickUpArrived is the OnPlayerArrived hook: a player that reached the item
// it walked to picks it up.
func (s *Server) pickUpArrived(p *Player) {
	pu := s.items.pickups[p]
	if pu == nil {
		return
	}
	delete(s.items.pickups, p)
	if s.pickUpReached(p, pu.g) {
		s.doPickUp(p, pu.g)
	}
}

// checkPickUps is the tick hook of the walks up to items: one that stopped
// next to its item picks it up, one that walks elsewhere or lost its item
// gives up (Mobius thinkPickUp).
func (s *Server) checkPickUps(now time.Time) {
	for p, pu := range s.items.pickups {
		switch {
		case !p.InGame() || !p.InWorld() || !pu.g.InWorld():
			delete(s.items.pickups, p)
		case p.Move == nil:
			delete(s.items.pickups, p)
			if s.pickUpReached(p, pu.g) {
				s.doPickUp(p, pu.g)
			}
		case !pu.moving || p.Move.Destination != pu.dest, now.Sub(pu.started) > pickUpTimeout:
			delete(s.items.pickups, p)
		}
	}
}

// pickUpReached reports whether p stands close enough to pick g up (Mobius
// maybeMoveToPawn(target, 36): the offset plus p's collision radius, in
// 2D).
func (s *Server) pickUpReached(p *Player, g *GroundItem) bool {
	reach := float64(pickUpRange) + float64(int(p.lastView.CollisionRadius))
	return p.Position().Distance2D(g.Position()) <= reach
}

// doPickUp is Mobius Player.doPickupItem: checks, GetItem to the player and
// the players around, the item leaves the ground, weapons and armor are
// announced, and the item goes into the inventory (adena onto the stack;
// arrows for a worn bow go on).
func (s *Server) doPickUp(p *Player, g *GroundItem) {
	if p.IsDead() {
		return
	}
	p.Send(stopMovePacket(p))
	if !g.InWorld() {
		p.SendActionFailed()
		return
	}
	it := g.item
	if g.protectedFor != 0 && g.protectedFor != p.ID() && s.Now().Before(g.protectedUntil) {
		p.SendActionFailed()
		p.SendSystemMessage(sysMsgFailedPickUpS1, s.itemNameParam(it.ItemID()))
		return
	}
	if !p.inv.ValidateCapacityFor(it) {
		p.SendActionFailed()
		p.SendSystemMessage(sysMsgInventoryFull)
		return
	}
	if p.Invisible() && !p.IsGM() {
		return
	}
	pos := g.Position()
	s.world.Broadcast(p, serverpackets.NewGetItemPacket(serverpackets.GetItem{
		PlayerID: p.ObjectID(), ObjectID: it.ObjectID(), X: pos.X, Y: pos.Y, Z: pos.Z,
	}), true)
	s.removeGroundItem(g)

	t := it.Template()
	if t.ExImmediateEffect && t.Etc != nil {
		// Herbs take effect at once and never enter the inventory.
		s.useHerb(p, it) // item_skill.go
		return
	}
	if t.Weapon != nil || t.Armor != nil {
		name := serverpackets.TextParam(p.Name())
		if it.Enchant() > 0 {
			s.world.Broadcast(p, serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: sysMsgAttentionPickedUpEnchant,
				Params: []serverpackets.SystemMessageParam{name, serverpackets.NumberParam(it.Enchant()), s.itemNameParam(it.ItemID())}}), true)
		} else {
			s.world.Broadcast(p, serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: sysMsgAttentionPickedUp,
				Params: []serverpackets.SystemMessageParam{name, s.itemNameParam(it.ItemID())}}), true)
		}
	}
	if it.ItemID() == inventory.AdenaID && p.inv.AdenaItem() != nil {
		s.AddAdena(p, it.Count(), true)
		return
	}
	s.addItemObject(p, it)
	if w := p.inv.Paperdoll(storage.PaperdollRHand); w != nil && w.Template().Weapon != nil &&
		w.Template().Weapon.Type == data.WeaponBow && t.Etc != nil && t.Etc.Type == data.EtcArrow {
		s.checkAndEquipArrows(p)
	}
}

// addItemObject puts an item that is in no container into p's inventory
// with Mobius Player.addItem(process, Item, reference, true): the message,
// InventoryUpdate and StatusUpdate with the load.
func (s *Server) addItemObject(p *Player, it *inventory.Item) {
	name := s.itemNameParam(it.ItemID())
	switch {
	case it.Count() > 1:
		p.SendSystemMessage(sysMsgObtainedS2S1, name, serverpackets.NumberParam(itemCount32(it.Count())))
	case it.Enchant() > 0:
		p.SendSystemMessage(sysMsgObtainedEnchantedS1S2, serverpackets.NumberParam(it.Enchant()), name)
	default:
		p.SendSystemMessage(sysMsgObtainedS1, name)
	}
	if _, err := p.inv.Add(it); err != nil {
		s.log.Error("game: cannot add an item", append(p.logArgs(), "itemObjectId", it.ObjectID(), "err", err)...)
		return
	}
	s.itemsChanged(p, true, it)
}

// checkAndEquipArrows puts arrows of the bow's grade into an empty left
// hand and sends ItemList (Mobius checkAndEquipArrows).
func (s *Server) checkAndEquipArrows(p *Player) {
	if p.inv.Paperdoll(storage.PaperdollLHand) != nil {
		return
	}
	w := p.inv.Paperdoll(storage.PaperdollRHand)
	if w == nil {
		return
	}
	if arrow := p.inv.FindArrowForBow(w.Template()); arrow != nil {
		p.inv.SetPaperdoll(storage.PaperdollLHand, arrow)
		p.inv.TakeUpdates()
		p.Send(serverpackets.NewItemListPacket(serverpackets.ItemList{Items: p.inv.ItemInfos()}))
		p.BroadcastUserInfo()
		s.RequestSave(p)
	}
}
