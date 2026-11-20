package game

import (
	"errors"
	"math"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
)

// The warehouse feature (warehouse*.go, Plan 08 Phase 16, decision I11):
// the private warehouse. The bypasses DepositP and WithdrawP open its
// windows (Mobius bypass handler PrivateWarehouse); SendWareHouseDepositList
// stores items for a fee and SendWareHouseWithDrawList takes them back
// (Mobius SendWareHouseDepositList, SendWareHouseWithDrawList).
func init() { registerFeature("warehouse", setupWarehouse) }

// warehouseDepositFee is the adena a deposit costs per stored entry (Mobius
// SendWareHouseDepositList).
const warehouseDepositFee = 30

// System messages of the warehouse.
const (
	warehouseMsgNothingDeposited serverpackets.SystemMessageID = 282 // "You have not deposited any items in your warehouse."
	warehouseMsgTooFar           serverpackets.SystemMessageID = 763 // "You failed at sending the package because you are too far from the warehouse."
)

// Item list types the deposit window checks (Mobius ItemTemplate).
const (
	itemType1ShieldArmor = 1 // TYPE1_SHIELD_ARMOR
	itemType2Money       = 4 // TYPE2_MONEY
)

// warehouseFeature is the warehouse state on the Server (Server.warehouse).
// Loop-only.
type warehouseFeature struct {
	// active holds the players whose private warehouse window opened (Mobius
	// Player.getActiveWarehouse).
	active map[*Player]bool
}

// setupWarehouse registers the warehouse packets and the player hooks.
func setupWarehouse(s *Server) error {
	s.warehouse.active = make(map[*Player]bool)
	s.OnPlayerLeave(func(p *Player) { delete(s.warehouse.active, p) })
	return errors.Join(
		s.Register(clientpackets.OpSendWareHouseDepositList, s.handleWarehouseDeposit),
		s.Register(clientpackets.OpSendWareHouseWithDrawList, s.handleWarehouseWithdraw),
	)
}

// bypassDepositP handles "DepositP" (Mobius PrivateWarehouse): ActionFailed,
// the private warehouse becomes the active one, RequestItemList is ignored
// for 1.5 s, and the deposit window lists what p may store.
func (s *Server) bypassDepositP(p *Player) {
	p.SendActionFailed()
	s.warehouse.active[p] = true
	s.BlockInventory(p)
	var items []serverpackets.ItemInfo
	for _, it := range p.inv.Items() {
		if depositAvailable(it) {
			items = append(items, warehouseItemInfo(it))
		}
	}
	p.Send(serverpackets.NewWareHouseDepositListPacket(serverpackets.WarehouseList{
		Type:  serverpackets.WarehousePrivate,
		Adena: itemCount32(p.inv.Adena()),
		Items: items,
	}))
}

// bypassWithdrawP handles "WithdrawP" (Mobius PrivateWarehouse with
// EnableWarehouseSortingPrivate off): ActionFailed, the private warehouse
// becomes the active one, and the withdrawal window lists its items, or p
// is told it has stored nothing.
func (s *Server) bypassWithdrawP(p *Player) {
	p.SendActionFailed()
	s.warehouse.active[p] = true
	wh := p.Warehouse()
	if wh == nil || wh.Len() == 0 {
		p.SendSystemMessage(warehouseMsgNothingDeposited)
		return
	}
	items := make([]serverpackets.ItemInfo, 0, wh.Len())
	for _, it := range wh.Items() {
		items = append(items, warehouseItemInfo(it))
	}
	p.Send(serverpackets.NewWareHouseWithdrawalListPacket(serverpackets.WarehouseList{
		Type:  serverpackets.WarehousePrivate,
		Adena: itemCount32(p.inv.Adena()),
		Items: items,
	}))
}

// warehouseItemInfo is an item as the warehouse windows list it: Mobius
// writes the template ID, not the display ID.
func warehouseItemInfo(it *inventory.Item) serverpackets.ItemInfo {
	info := ItemInfo(it)
	info.ItemID = it.ItemID()
	return info
}

// depositAvailable reports whether the private warehouse takes the item
// (Mobius Item.isAvailable with adena and untradable items allowed, then
// isDepositable(true)): not worn, not a quest item, depositable.
func depositAvailable(it *inventory.Item) bool {
	t := it.Template()
	return !it.Equipped() && t.Type2 != typeQuest &&
		!(t.Type2 == itemType2Money && t.Type1 == itemType1ShieldArmor) &&
		it.Depositable(true)
}

// warehouseManagerOK reports whether p may use the warehouse here: its last
// NPC is a warehouse keeper it can interact with (standing, within 250), or
// p is a GM. Otherwise p is told it is too far.
func (s *Server) warehouseManagerOK(p *Player) bool {
	if p.IsGM() {
		return true
	}
	var n *Npc
	if p.LastNpc != 0 {
		n = s.Npc(p.LastNpc)
	}
	if n == nil || !s.npcIs(n, "Warehouse") || !s.npcCanInteract(p, n) {
		p.SendSystemMessage(warehouseMsgTooFar)
		return false
	}
	return true
}

// handleWarehouseDeposit handles SendWareHouseDepositList (0x31) in Mobius's
// order: flood, the open window, the keeper, karma; then every item is
// checked, the slots counted and the fee (30 adena per stored entry, from
// the adena not being stored) taken before the items move. Items the
// warehouse does not take are skipped and cost nothing.
func (s *Server) handleWarehouseDeposit(p *Player, body []byte) {
	req, err := clientpackets.ParseSendWareHouseDepositList(body)
	if err != nil {
		s.log.Debug("game: bad SendWareHouseDepositList", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Items == nil {
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are depositing items too fast.")
		return
	}
	wh := p.Warehouse()
	if wh == nil || !s.warehouse.active[p] || !s.warehouseManagerOK(p) {
		return
	}
	if !s.gs.Economy.AltKarmaPlayerCanUseWarehouse && p.Karma() > 0 {
		return
	}
	if hasDuplicateWarehouseItems(req.Items) {
		s.log.Warn("game: deposit lists an item twice", p.logArgs()...)
		return
	}
	adena := p.inv.Adena()
	var slots, chargeable int64
	for _, ri := range req.Items {
		it := depositCheckItem(p, ri.ObjectID, int64(ri.Count))
		if it == nil {
			s.log.Warn("game: deposit of an item the player cannot move", append(p.logArgs(), "itemObjectId", ri.ObjectID, "count", ri.Count)...)
			return
		}
		if !depositAvailable(it) {
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
	fee := chargeable * warehouseDepositFee
	if adena < fee || !s.ReduceAdena(p, fee, false) {
		p.SendSystemMessage(sysMsgNotEnoughAdena)
		return
	}
	for _, ri := range req.Items {
		it := p.inv.ByObjectID(ri.ObjectID)
		if it == nil || !depositAvailable(it) {
			continue
		}
		if _, err := p.inv.Transfer(ri.ObjectID, int64(ri.Count), wh); err != nil {
			s.log.Error("game: deposit failed", append(p.logArgs(), "itemObjectId", ri.ObjectID, "count", ri.Count, "err", err)...)
		}
	}
	s.ItemsChanged(p)
}

// handleWarehouseWithdraw handles SendWareHouseWithDrawList (0x32) in
// Mobius's order: flood, the open window, the keeper, karma; an item that is
// not stored, or not that many, is an illegal action; then the inventory
// slots and the weight; then the items move.
func (s *Server) handleWarehouseWithdraw(p *Player, body []byte) {
	req, err := clientpackets.ParseSendWareHouseWithDrawList(body)
	if err != nil {
		s.log.Debug("game: bad SendWareHouseWithDrawList", append(p.logArgs(), "err", err)...)
		return
	}
	if req.Items == nil {
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		p.SendMessage("You are withdrawing items too fast.")
		return
	}
	wh := p.Warehouse()
	if wh == nil || !s.warehouse.active[p] || !s.warehouseManagerOK(p) {
		return
	}
	if !s.gs.Economy.AltKarmaPlayerCanUseWarehouse && p.Karma() > 0 {
		return
	}
	if hasDuplicateWarehouseItems(req.Items) {
		s.log.Warn("game: withdrawal lists an item twice", p.logArgs()...)
		return
	}
	var weight, slots int64
	for _, ri := range req.Items {
		it := wh.ByObjectID(ri.ObjectID)
		if it == nil || it.Count() < int64(ri.Count) {
			s.illegalAction(p, "withdrawal of an item not in the warehouse")
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
	for _, ri := range req.Items {
		if ri.Count == 0 {
			continue // Mobius would make an item of count 0
		}
		if _, err := wh.Transfer(ri.ObjectID, int64(ri.Count), &p.inv.Container); err != nil {
			s.log.Error("game: withdrawal failed", append(p.logArgs(), "itemObjectId", ri.ObjectID, "count", ri.Count, "err", err)...)
		}
	}
	s.ItemsChanged(p)
}

// hasDuplicateWarehouseItems reports whether the list names an object twice.
// Mobius would check each against the item as the earlier entries left it
// and stop halfway; an honest client never sends one.
func hasDuplicateWarehouseItems(items []clientpackets.WarehouseItem) bool {
	seen := make(map[int32]bool, len(items))
	for _, it := range items {
		if seen[it.ObjectID] {
			return true
		}
		seen[it.ObjectID] = true
	}
	return false
}

// depositCheckItem returns p's carried item with the object ID if p may
// move count of it (Mobius Player.checkItemManipulation): it must exist,
// count must be at least 0, at most 1 for an item that does not stack, and
// at most what it has. Pets, mounts, enchanting and augmentations come
// later.
func depositCheckItem(p *Player, objectID int32, count int64) *inventory.Item {
	it := p.inv.ByObjectID(objectID)
	if it == nil || count < 0 || (count > 1 && !it.Stackable()) || count > it.Count() || count > math.MaxInt32 {
		return nil
	}
	return it
}
