package game

import (
	"context"
	"fmt"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
)

// The clan warehouse (Plan 12 Phase 7 item 8, Mobius ClanWarehouse, the
// bypass handler ClanWarehouse and the clan branches of
// SendWareHouseDepositList and SendWareHouseWithDrawList), on M5's
// warehouse with the clan as the owner of its items (location CLANWH):
// members of a clan of level 1 or higher deposit, for the same fee as the
// private warehouse; the withdrawal window needs the warehouse privilege,
// and only the leader withdraws unless clan.membersCanWithdrawFromWarehouse
// lets members with the privilege do it; 200 slots
// (items.warehouseSlotsClan). Every move is one joint save of the player
// and the clan's items (decision T5).
//
// The stored items of every clan are read at startup, after the clans (as
// Mobius restores each clan's warehouse with the clan); a clan's container
// is built from them the first time the clan uses its warehouse.
func init() { registerFeature("clanWarehouse", setupClanWarehouse) }

// System messages of the clan warehouse.
const (
	clanWHMsgNoRight        serverpackets.SystemMessageID = 709  // "You do not have the right to use the clan warehouse."
	clanWHMsgNeedsLevel1    serverpackets.SystemMessageID = 710  // "Only clans of clan level 1 or higher can use a clan warehouse."
	clanWHMsgOnlyLeaderTake serverpackets.SystemMessageID = 1039 // "Items left at the clan hall warehouse can only be retrieved by the clan leader. ..."
)

// clanWarehouseFeature is the clan warehouse state on the Server
// (Server.clanWH). Loop-only once the server runs.
type clanWarehouseFeature struct {
	// stored are the items each clan had stored at startup, until its
	// container is built.
	stored map[int32][]storage.Item
	// warehouses are the clans' containers, by clan ID.
	warehouses map[int32]*inventory.Container
	// active holds the players whose clan warehouse window opened, with the
	// clan's ID (Mobius Player.getActiveWarehouse).
	active map[*Player]int32
}

// setupClanWarehouse reads the stored items of every clan (the clans are
// loaded by then: features set up in name order) and the player hooks.
func setupClanWarehouse(s *Server) error {
	f := &s.clanWH
	f.stored = make(map[int32][]storage.Item)
	f.warehouses = make(map[int32]*inventory.Container)
	f.active = make(map[*Player]int32)
	s.OnPlayerLeave(func(p *Player) { delete(f.active, p) })
	if s.clan.store == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), clanLoadTimeout)
	defer cancel()
	n := 0
	for _, c := range sortedClans(s.clan.clans) {
		items, err := s.clan.store.ClanItems(ctx, c.ID)
		if err != nil {
			return fmt.Errorf("game: load the warehouse of clan %d: %w", c.ID, err)
		}
		if len(items) > 0 {
			f.stored[c.ID] = items
			n += len(items)
		}
	}
	if n > 0 {
		s.log.Info("clan warehouses loaded", "clans", len(f.stored), "items", n)
	}
	return nil
}

// ClanWarehouse returns c's warehouse, building it from the stored items
// the first time. Move items with Transfer and write the moves with
// saveClanWarehouseMove. Loop-only.
func (s *Server) ClanWarehouse(c *Clan) *inventory.Container {
	f := &s.clanWH
	if wh := f.warehouses[c.ID]; wh != nil {
		return wh
	}
	opts := inventory.Options{Templates: s.data, IDs: s.items.ids, Config: s.items.cfg}
	wh := inventory.NewClanWarehouse(func() int { return s.gs.Items.WarehouseSlotsClan }, opts)
	for _, si := range wh.Load(f.stored[c.ID]) {
		s.log.Warn("game: clan warehouse item without a template, kept but not shown", "clan", c.Name, "item", si.ItemID, "itemObjectId", si.ObjectID)
	}
	delete(f.stored, c.ID)
	f.warehouses[c.ID] = wh
	return wh
}

// clanWarehouseOpen reports whether p's active warehouse is a clan
// warehouse (the deposit and withdrawal packets then go to this file).
func (s *Server) clanWarehouseOpen(p *Player) bool {
	_, ok := s.clanWH.active[p]
	return ok
}

// closeClanWarehouse forgets p's clan warehouse window: another warehouse
// became the active one.
func (s *Server) closeClanWarehouse(p *Player) { delete(s.clanWH.active, p) }

// openClanWarehouse makes c's warehouse p's active one.
func (s *Server) openClanWarehouse(p *Player, c *Clan) {
	delete(s.warehouse.active, p)
	s.clanWH.active[p] = c.ID
}

// clanWarehouseCommands are the bypasses of the clan warehouse (Mobius
// ClanWarehouse.COMMANDS without the sorted list, which Mobius offers only
// with EnableWarehouseSortingClan, off by default).
var clanWarehouseCommands = map[string]bool{"depositc": true, "withdrawc": true}

// bypassClanWarehouse handles "DepositC" and "WithdrawC" at a warehouse
// keeper (Mobius bypass handler ClanWarehouse): p needs a clan of level 1
// or higher. DepositC lists what p may store; WithdrawC needs the warehouse
// privilege and lists the stored items, or tells p nothing is stored.
func (s *Server) bypassClanWarehouse(p *Player, n *Npc, word string) {
	if !s.npcIs(n, "Warehouse") && !s.npcIs(n, "ClanHallManager") {
		return // Mobius: only warehouse keepers and clan hall managers (isWarehouse)
	}
	c := p.clan.clan
	switch {
	case c == nil:
		p.SendSystemMessage(clanWHMsgNoRight)
		return
	case c.Level == 0:
		p.SendSystemMessage(clanWHMsgNeedsLevel1)
		return
	}
	p.SendActionFailed()
	if word == "depositc" {
		s.openClanWarehouse(p, c)
		s.BlockInventory(p)
		var items []serverpackets.ItemInfo
		for _, it := range p.inv.Items() {
			if clanDepositAvailable(it) {
				items = append(items, warehouseItemInfo(it))
			}
		}
		p.Send(serverpackets.NewWareHouseDepositListPacket(serverpackets.WarehouseList{
			Type:  serverpackets.WarehouseClan,
			Adena: itemCount32(p.inv.Adena()),
			Items: items,
		}))
		return
	}
	if !p.HasClanAccess(social.AccessWarehouse) {
		p.SendSystemMessage(clanWHMsgNoRight)
		return
	}
	s.openClanWarehouse(p, c)
	wh := s.ClanWarehouse(c)
	if wh.Len() == 0 {
		p.SendSystemMessage(warehouseMsgNothingDeposited)
		return
	}
	items := make([]serverpackets.ItemInfo, 0, wh.Len())
	for _, it := range wh.Items() {
		items = append(items, warehouseItemInfo(it))
	}
	p.Send(serverpackets.NewWareHouseWithdrawalListPacket(serverpackets.WarehouseList{
		Type:  serverpackets.WarehouseClan,
		Adena: itemCount32(p.inv.Adena()),
		Items: items,
	}))
}

// clanDepositAvailable reports whether the clan warehouse takes the item
// (Mobius Item.isAvailable with adena allowed and untradable items not,
// then isDepositable(false)): not worn, not a quest item, tradable, not a
// shadow item, depositable.
func clanDepositAvailable(it *inventory.Item) bool {
	t := it.Template()
	return !it.Equipped() && t.Type2 != typeQuest &&
		(t.Type2 != itemType2Money || t.Type1 != itemType1ShieldArmor) &&
		it.Depositable(false)
}

// clanWarehouseUser returns p's clan if its open clan warehouse window is
// that clan's and p may use it here (the keeper and karma checks of Mobius
// SendWareHouseDepositList and SendWareHouseWithDrawList), or nil.
//
// Deviation: a player who left the clan since the window opened can no
// longer use its warehouse; Mobius keeps the old clan's warehouse active.
func (s *Server) clanWarehouseUser(p *Player) *Clan {
	c := p.clan.clan
	if id, ok := s.clanWH.active[p]; !ok || c == nil || c.ID != id || s.clan.clans[id] != c {
		return nil
	}
	if !s.warehouseManagerOK(p) {
		return nil
	}
	if !s.gs.Economy.AltKarmaPlayerCanUseWarehouse && p.Karma() > 0 {
		return nil
	}
	return c
}

// clanWarehouseDeposit is SendWareHouseDepositList with a clan warehouse
// open, after the flood check: as the private deposit (the fee of 30 adena
// per stored entry, the slot check against 200), for the items the clan
// warehouse takes; the move is one joint save.
func (s *Server) clanWarehouseDeposit(p *Player, req []clientpackets.WarehouseItem) {
	if s.Trading(p) {
		return
	}
	c := s.clanWarehouseUser(p)
	if c == nil {
		return
	}
	if hasDuplicateWarehouseItems(req) {
		s.log.Warn("game: clan deposit lists an item twice", p.logArgs()...)
		return
	}
	wh := s.ClanWarehouse(c)
	adena := p.inv.Adena()
	var slots, chargeable int64
	for _, ri := range req {
		it := depositCheckItem(p, ri.ObjectID, int64(ri.Count))
		if it == nil {
			s.log.Warn("game: clan deposit of an item the player cannot move", append(p.logArgs(), "itemObjectId", ri.ObjectID, "count", ri.Count)...)
			return
		}
		if !clanDepositAvailable(it) {
			continue
		}
		chargeable++
		if it.ItemID() == adenaID {
			adena -= int64(ri.Count)
		}
		switch {
		case !it.Stackable():
			slots += int64(ri.Count)
		case wh.ByItemID(it.ItemID()) == nil:
			slots++
		}
	}
	if !wh.ValidateCapacity(slots) {
		p.SendSystemMessage(exchangeMsgQuantityExceeded)
		return
	}
	// The fee is taken without ReduceAdena, whose save request would write
	// it apart from the move.
	fee := chargeable * warehouseDepositFee
	if adena < fee {
		p.SendSystemMessage(sysMsgNotEnoughAdena)
		return
	}
	if fee > 0 {
		if _, err := p.inv.DestroyByItemID(adenaID, fee); err != nil {
			p.SendSystemMessage(sysMsgNotEnoughAdena)
			return
		}
	}
	for _, ri := range req {
		it := p.inv.ByObjectID(ri.ObjectID)
		if it == nil || !clanDepositAvailable(it) {
			continue
		}
		if _, err := p.inv.Transfer(ri.ObjectID, int64(ri.Count), wh); err != nil {
			s.log.Error("game: clan deposit failed", append(p.logArgs(), "clan", c.Name, "itemObjectId", ri.ObjectID, "count", ri.Count, "err", err)...)
		}
	}
	s.saveClanWarehouseMove(p, c, wh)
}

// clanWarehouseWithdraw is SendWareHouseWithDrawList with a clan warehouse
// open, after the flood check: only the leader withdraws ("Items left at
// the clan hall warehouse can only be retrieved by the clan leader."), or,
// with clan.membersCanWithdrawFromWarehouse, members with the warehouse
// privilege (others are ignored); then as the private withdrawal (a forged
// item is an illegal action, then slots, then weight); the move is one
// joint save.
func (s *Server) clanWarehouseWithdraw(p *Player, req []clientpackets.WarehouseItem) {
	c := s.clanWarehouseUser(p)
	if c == nil {
		return
	}
	if s.gs.Clan.MembersCanWithdrawFromWarehouse {
		if !p.HasClanAccess(social.AccessWarehouse) {
			return
		}
	} else if !p.IsClanLeader() {
		p.SendSystemMessage(clanWHMsgOnlyLeaderTake)
		return
	}
	if hasDuplicateWarehouseItems(req) {
		s.log.Warn("game: clan withdrawal lists an item twice", p.logArgs()...)
		return
	}
	wh := s.ClanWarehouse(c)
	var weight, slots int64
	for _, ri := range req {
		it := wh.ByObjectID(ri.ObjectID)
		if it == nil || it.Count() < int64(ri.Count) {
			s.illegalAction(p, "withdrawal of an item not in the clan warehouse")
			return
		}
		weight += int64(ri.Count) * int64(it.Template().Weight)
		switch {
		case !it.Stackable():
			slots += int64(ri.Count)
		case p.inv.ByItemID(it.ItemID()) == nil:
			slots++
		}
	}
	if !p.inv.ValidateCapacity(slots) {
		p.SendSystemMessage(sysMsgInventoryFull)
		return
	}
	if !p.inv.ValidateWeight(weight) {
		p.SendSystemMessage(exchangeMsgWeightLimit)
		return
	}
	for _, ri := range req {
		if ri.Count == 0 {
			continue // Mobius would make an item of count 0
		}
		if _, err := wh.Transfer(ri.ObjectID, int64(ri.Count), &p.inv.Container); err != nil {
			s.log.Error("game: clan withdrawal failed", append(p.logArgs(), "clan", c.Name, "itemObjectId", ri.ObjectID, "count", ri.Count, "err", err)...)
		}
	}
	s.saveClanWarehouseMove(p, c, wh)
}

// saveClanWarehouseMove ends a move between p and c's warehouse: p gets the
// InventoryUpdate and StatusUpdate with its load, and p's pending changes
// and the warehouse's are written in one transaction (decision T5), so that
// a crash keeps the items on one side.
func (s *Server) saveClanWarehouseMove(p *Player, c *Clan, wh *inventory.Container) {
	s.queueInventoryUpdate(p, s.takeInventoryUpdates(p, nil)...)
	wh.TakeUpdates()
	if p.InGame() {
		p.Send(loadStatusUpdate(p))
	}
	s.refreshOverloaded(p)
	var b storage.Batch
	if ch := wh.Tracker().Take(); !noItemChanges(ch) {
		b.Items = []storage.OwnerItemChanges{ownerItemChanges(c.ID, ch)}
	}
	s.SaveJoint([]*Player{p}, b)
}
