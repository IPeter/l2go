package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/world"
)

// The store windows: the setup windows of the owner, and the windows other
// players open with a second click on a store (Mobius PlayerClick, the
// INTERACT intention and Player.doInteract).

// storeApproach is a player walking up to a store to open it.
type storeApproach struct {
	owner *Player
	// dest is where the walk goes; another destination means the player
	// walks elsewhere. moving is false when the walk had no MoveState.
	dest    world.Position
	moving  bool
	started time.Time
}

// interactStore is a click on a store owner who is p's target (Mobius
// setIntentionInteract and PlayerAI.thinkInteract): within reach p opens
// the store, otherwise p walks up to it (MoveToPawn to p and the players
// around) and opens it on arrival (checkStoreApproaches). A sitting player
// does not move.
func (s *Server) interactStore(p, owner *Player) {
	delete(s.stores.approaches, p)
	if s.storeReached(p, owner) {
		s.storeInteract(p, owner)
		return
	}
	if p.Sitting {
		return
	}
	s.playerIntendInteract(p) // the INTERACT intention ends an attack
	if !s.MovePlayerTo(p, owner.Position(), targetApproachOffset) {
		return
	}
	a := &storeApproach{owner: owner, started: s.Now()}
	if m := p.Move; m != nil {
		a.dest, a.moving = s.walkGoal(p, m), true
	}
	s.stores.approaches[p] = a
	s.broadcastPawnWalk(p, owner, targetApproachOffset) // pathing.go
}

// checkStoreApproaches is the tick hook of the walks up to stores: a player
// that reached the store, or stopped, opens it when it is within reach; one
// that walks elsewhere, lost the owner as its target, or whose owner left
// gives up.
func (s *Server) checkStoreApproaches(now time.Time) {
	for p, a := range s.stores.approaches {
		o := a.owner
		switch {
		case !p.InGame() || !p.InWorld() || p.Target != world.Object(o) || !o.InWorld():
			delete(s.stores.approaches, p)
		case s.storeReached(p, o):
			delete(s.stores.approaches, p)
			s.storeInteract(p, o)
		case p.Move == nil, !a.moving || s.walkGoal(p, p.Move) != a.dest, now.Sub(a.started) > targetApproachTimeout:
			delete(s.stores.approaches, p)
		}
	}
}

// storeReached reports whether p is close enough to open owner's store
// (Mobius maybeMoveToPawn with offset 36: within the offset plus both
// collision radii, in 2D).
func (s *Server) storeReached(p, owner *Player) bool {
	reach := float64(targetApproachOffset) + float64(int(p.lastView.CollisionRadius)) + float64(int(owner.lastView.CollisionRadius))
	return p.Position().Distance2D(owner.Position()) <= reach
}

// storeInteract is Mobius Player.doInteract on a player: a walking p stops
// (the intention becomes idle), gets ActionFailed, and then the window of
// the store owner runs now, if any.
func (s *Server) storeInteract(p, owner *Player) {
	if p.Move != nil {
		s.playerIntendIdle(p)
	}
	p.SendActionFailed()
	switch owner.store.typ {
	case StoreSell, StorePackageSell:
		p.Send(s.storeListSell(p, owner))
	case StoreBuy:
		p.Send(s.storeListBuy(p, owner))
	}
}

// storePacketItem is an entry of a store list as the store packets write it.
func storePacketItem(ti *storeItem) serverpackets.StoreItem {
	t := ti.tmpl
	//nolint:gosec // list types, IDs and masks fit; counts and prices are below MaxAdena
	return serverpackets.StoreItem{
		ObjectID:       ti.objectID,
		ItemID:         int32(t.ID),
		Count:          itemCount32(ti.count),
		Type2:          int16(t.Type2),
		BodyPart:       int32(t.BodyPartMask),
		Enchant:        int16(min(ti.enchant, 0x7fff)),
		CustomType2:    ti.custom2,
		Price:          itemCount32(ti.price),
		ReferencePrice: int32(t.Price),
	}
}

// storeInventoryItem is an inventory item as the store packets write it,
// with count and price.
func storeInventoryItem(it *inventory.Item, count, price int64) serverpackets.StoreItem {
	return storePacketItem(&storeItem{
		objectID: it.ObjectID(),
		tmpl:     it.Template(),
		enchant:  it.Enchant(),
		custom2:  int16(it.Storage().CustomType2), //nolint:gosec // custom types are small
		count:    count,
		price:    price,
	})
}

// storeUpdateSellList fits p's sell list to its inventory (Mobius
// TradeList.updateItems): an item that is gone, or an entry without a
// count, leaves the list; an entry for more than the item has is cut to
// it.
func storeUpdateSellList(p *Player) {
	l := &p.store.sell
	l.items = slices.DeleteFunc(l.items, func(ti *storeItem) bool {
		it := p.inv.ByObjectID(ti.objectID)
		if it == nil || ti.count < 1 {
			return true
		}
		ti.count = min(ti.count, it.Count())
		return false
	})
}

// storeManageListSell builds the setup window of p's sell store (Mobius
// PrivateStoreManageListSell): the list is fitted to the inventory first;
// the available items are those p may offer (storeItemAvailable), a
// stackable one less the count the list holds, at the reference price.
func (s *Server) storeManageListSell(p *Player, packageSale bool) []byte {
	storeUpdateSellList(p)
	l := &p.store.sell
	var available []serverpackets.StoreItem
	for _, it := range p.inv.Items() {
		if !storeItemAvailable(it) {
			continue
		}
		count := it.Count()
		if it.Stackable() {
			if i := slices.IndexFunc(l.items, func(ti *storeItem) bool { return ti.tmpl.ID == it.Template().ID }); i >= 0 {
				if count <= l.items[i].count {
					continue
				}
				count -= l.items[i].count
			}
		}
		available = append(available, storeInventoryItem(it, count, int64(it.Template().Price)))
	}
	items := make([]serverpackets.StoreItem, 0, len(l.items))
	for _, ti := range l.items {
		items = append(items, storePacketItem(ti))
	}
	return serverpackets.NewPrivateStoreManageListSellPacket(serverpackets.PrivateStoreManageListSell{
		ObjectID:    p.ObjectID(),
		PackageSale: packageSale,
		Adena:       itemCount32(p.inv.Adena()),
		Available:   available,
		Items:       items,
	})
}

// storeManageListBuy builds the setup window of p's buy store (Mobius
// PrivateStoreManageListBuy): one item per template that p may offer
// (getUniqueItems(false, true, true, false)), and the buy list.
func (s *Server) storeManageListBuy(p *Player) []byte {
	var inv []serverpackets.StoreItem
	for _, it := range p.inv.Items() {
		if it.ItemID() == adenaID || !storeItemAvailable(it) {
			continue
		}
		if slices.ContainsFunc(inv, func(x serverpackets.StoreItem) bool { return x.ItemID == it.ItemID() }) {
			continue
		}
		inv = append(inv, storeInventoryItem(it, it.Count(), 0))
	}
	l := &p.store.buy
	items := make([]serverpackets.StoreItem, 0, len(l.items))
	for _, ti := range l.items {
		items = append(items, storePacketItem(ti))
	}
	return serverpackets.NewPrivateStoreManageListBuyPacket(serverpackets.PrivateStoreManageListBuy{
		ObjectID:  p.ObjectID(),
		Adena:     itemCount32(p.inv.Adena()),
		Inventory: inv,
		Items:     items,
	})
}

// storeListSell builds the window of owner's sell store as viewer sees it
// (Mobius PrivateStoreListSell): the list as it is, and the adena viewer
// carries.
func (s *Server) storeListSell(viewer, owner *Player) []byte {
	l := &owner.store.sell
	items := make([]serverpackets.StoreItem, 0, len(l.items))
	for _, ti := range l.items {
		items = append(items, storePacketItem(ti))
	}
	return serverpackets.NewPrivateStoreListSellPacket(serverpackets.PrivateStoreListSell{
		SellerID:    owner.ObjectID(),
		PackageSale: l.packaged,
		Adena:       itemCount32(viewer.inv.Adena()),
		Items:       items,
	})
}

// storeListBuy builds the window of owner's buy store as viewer sees it
// (Mobius PrivateStoreListBuy and TradeList.getAvailableItems): each entry
// names viewer's item of the template and how many of it viewer can sell
// (PlayerInventory.adjustAvailableItem), 0 when viewer has none that is not
// worn.
func (s *Server) storeListBuy(viewer, owner *Player) []byte {
	l := &owner.store.buy
	items := make([]serverpackets.StoreBuyItem, 0, len(l.items))
	for _, ti := range l.items {
		e := serverpackets.StoreBuyItem{StoreItem: storePacketItem(ti), StoreCount: itemCount32(ti.count)}
		e.ObjectID, e.Enchant, e.Count = 0, 0, 0
		if it := storeSellableItem(viewer, ti.tmpl.ID); it != nil {
			e.ObjectID = it.ObjectID()
			e.Enchant = int16(min(it.Enchant(), 0x7fff)) //nolint:gosec // clamped
			e.Count = itemCount32(min(ti.count, it.Count()))
		}
		items = append(items, e)
	}
	return serverpackets.NewPrivateStoreListBuyPacket(serverpackets.PrivateStoreListBuy{
		StorePlayerID: owner.ObjectID(),
		Adena:         itemCount32(viewer.inv.Adena()),
		Items:         items,
	})
}

// storeSellableItem returns p's item of the template that a buy store may
// get: the first that is not worn (Mobius adjustAvailableItem takes the
// first of the template, which may be the worn one), or nil.
func storeSellableItem(p *Player, templateID int) *inventory.Item {
	for _, it := range p.inv.AllByItemID(int32(templateID)) { //nolint:gosec // item IDs fit in int32
		if !it.Equipped() {
			return it
		}
	}
	return nil
}
