package game

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/conditions"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// The item feature (items*.go, inventory*.go, ground.go): the inventory and
// warehouse of internal/inventory in the game (Plan 08 Phase 14). It loads
// a character's items, sends InventoryUpdate and ItemList the way Mobius
// times them (decision I13), keeps the weight penalty (I5) and the mana of
// shadow items (I12), handles UseItem, RequestUnEquipItem, RequestDropItem,
// RequestDestroyItem, RequestItemList and picking items up, and asks for a
// save at the end of every tick that changed items (I2).
func init() { registerFeature("items", setupItems) }

const (
	// inventoryUpdateDelay is how long Mobius collects inventory changes
	// before it sends them in one InventoryUpdate (Player.sendInventoryUpdate).
	inventoryUpdateDelay = 100 * time.Millisecond
	// itemListDelay is how long after the last request Mobius sends ItemList
	// (Player.sendItemList).
	itemListDelay = 250 * time.Millisecond
	// inventoryBlockTime is how long RequestItemList is ignored after a shop
	// or warehouse window opened (Player.setInventoryBlockingStatus).
	inventoryBlockTime = 1500 * time.Millisecond
	// manaConsumptionRate is how often a worn shadow item loses one mana
	// (ItemManaTaskManager.MANA_CONSUMPTION_RATE).
	manaConsumptionRate = time.Minute
	// illegalActionKickDelay is how long after an illegal action the player
	// is kicked (PunishmentManager.handleIllegalPlayerAction, DefaultPunish
	// KICK).
	illegalActionKickDelay = 5 * time.Second
)

// System messages of the item feature (Mobius SystemMessageId).
const (
	sysMsgObtainedS2S1             serverpackets.SystemMessageID = 29   // "You have obtained $s2 $s1."
	sysMsgObtainedS1               serverpackets.SystemMessageID = 30   // "You have obtained $s1."
	sysMsgEquippedS1               serverpackets.SystemMessageID = 49   // "You have equipped your $s1."
	sysMsgEarnedS1Adena            serverpackets.SystemMessageID = 52   // "You have earned $s1 adena."
	sysMsgEarnedS2S1               serverpackets.SystemMessageID = 53   // "You have earned $s2 $s1(s)."
	sysMsgEarnedS1                 serverpackets.SystemMessageID = 54   // "You have earned $s1."
	sysMsgFailedPickUpS1           serverpackets.SystemMessageID = 56   // "You have failed to pick up $s1."
	sysMsgFailedEarnAdena          serverpackets.SystemMessageID = 58   // "You have failed to earn $s1 adena."
	sysMsgCannotDiscard            serverpackets.SystemMessageID = 98   // "This item cannot be discarded."
	sysMsgInventoryFull            serverpackets.SystemMessageID = 129  // "Your inventory is full."
	sysMsgCannotUseQuestItems      serverpackets.SystemMessageID = 148  // "You cannot use quest items."
	sysMsgTooFarToDiscard          serverpackets.SystemMessageID = 151  // "That is too far from you to discard."
	sysMsgDestroyCountIncorrect    serverpackets.SystemMessageID = 163  // "You cannot destroy it because the number is incorrect."
	sysMsgEquippedS1S2             serverpackets.SystemMessageID = 368  // "Equipped +$s1 $s2."
	sysMsgObtainedEnchantedS1S2    serverpackets.SystemMessageID = 369  // "You have obtained a +$s1 $s2."
	sysMsgAdenaLimit               serverpackets.SystemMessageID = 403  // "You have exceeded your out-of-pocket adena limit."
	sysMsgDisarmed                 serverpackets.SystemMessageID = 417  // "$s1 has been disarmed."
	sysMsgTooEncumbered            serverpackets.SystemMessageID = 555  // "You cannot move due to the weight of your inventory."
	sysMsgCannotDiscardOrExchange  serverpackets.SystemMessageID = 603  // "That item cannot be discarded or exchanged."
	sysMsgEquipmentRemoved         serverpackets.SystemMessageID = 1064 // "The equipment, +$s1 $s2, has been removed."
	sysMsgEquipConditionNotMet     serverpackets.SystemMessageID = 1518 // "You do not meet the required condition to equip that item."
	sysMsgAttentionPickedUp        serverpackets.SystemMessageID = 1533 // "Attention: $s1 picked up $s2."
	sysMsgAttentionPickedUpEnchant serverpackets.SystemMessageID = 1534 // "Attention: $s1 picked up +$s2 $s3."
	sysMsgHeroWeaponsNoDestroy     serverpackets.SystemMessageID = 1845 // "Hero weapons cannot be destroyed."
	sysMsgMana10                   serverpackets.SystemMessageID = 1979 // "$s1's remaining mana is now 10."
	sysMsgMana5                    serverpackets.SystemMessageID = 1980 // "$s1's remaining mana is now 5."
	sysMsgMana1                    serverpackets.SystemMessageID = 1981 // "$s1's remaining mana is now 1. It will disappear soon."
	sysMsgMana0                    serverpackets.SystemMessageID = 1982 // "$s1's remaining mana is now 0, and the item has disappeared."
)

// typeQuest is the Type2 of quest items (Mobius ItemTemplate.TYPE2_QUEST).
const typeQuest = 3

// weightPenaltyOwner owns the modifiers of the weight penalty (skill 4270).
var weightPenaltyOwner = stats.Owner{Kind: stats.OwnerSkill, ID: data.SkillWeightPenalty}

// itemsFeature is the item feature's state on the Server (Server.items).
// The options are set before setup; the rest is loop-only.
type itemsFeature struct {
	pool       *storage.IDPool
	passives   *data.Passives
	categories *data.Categories

	ids   inventory.IDSource // pool as an IDSource, nil without one
	cfg   inventory.Config
	conds conditions.Table
	// weightMods are skill 4270's modifiers per weight penalty level 1-4.
	weightMods [5][]stats.Modifier

	// saveRequests are the players whose items changed this tick, in the
	// order of their first change (RequestSave).
	saveRequests []*Player
	// mana holds the worn shadow items whose mana runs down.
	mana map[*inventory.Item]manaTask
	// pickups are the walks up to items on the ground.
	pickups map[*Player]*pickUp
}

// manaTask is the timer of a shadow item's next mana loss.
type manaTask struct {
	p     *Player
	timer world.TimerID
}

// playerItems is the item feature's state of a player (Player.items).
type playerItems struct {
	warehouse *inventory.Container
	tracker   *inventory.Tracker
	// unknown are the stored items the data has no template for: kept and
	// saved as they are, never shown (Mobius skips them).
	unknown []storage.Item

	iu      []serverpackets.InventoryItem // the InventoryUpdate waiting to be sent
	iuTimer world.TimerID
	ilTimer world.TimerID
	ilOpen  bool

	weightPenalty int32 // 0-4 (Mobius curWeightPenalty)
	overloaded    bool  // the load is above the maximum: no walking
	blockedUntil  time.Time

	nextDrop        time.Time // flood protection of RequestDropItem
	nextTransaction time.Time // flood protection of destroying and trading
}

// setupItems registers the item packets, the GM item commands and the hooks.
func setupItems(s *Server) error {
	f := &s.items
	f.cfg = inventory.Config{
		MultipleItemDrop:  s.gs.Items.MultipleItemDrop,
		MaxAdena:          s.gs.Items.MaxAdena,
		AutoLootSlotLimit: s.gs.Items.AutoLootSlotLimit,
	}
	if f.cfg.MaxAdena <= 0 {
		f.cfg.MaxAdena = inventory.DefaultMaxAdena
	}
	conds, err := conditions.NewTable(s.data.Items)
	if err != nil {
		s.log.Warn("game: item conditions that cannot be read are left out", "err", err)
	}
	f.conds = conds
	if f.pool == nil && s.ids != nil {
		f.pool = storage.NewIDPool(s.ids, storage.IDPoolOptions{Logger: s.log})
		ctx, cancel := context.WithTimeout(context.Background(), storage.DefaultIDPoolTimeout)
		err := f.pool.Fill(ctx)
		cancel()
		if err != nil {
			s.log.Error("game: cannot fill the object ID pool; new items fail until a refill works", "err", err)
		}
	}
	if f.pool != nil {
		f.ids = f.pool
	}
	f.weightMods = s.weightPenaltyModifiers()
	f.mana = make(map[*inventory.Item]manaTask)
	f.pickups = make(map[*Player]*pickUp)

	s.OnPlayerEnter(s.itemsEnter)
	s.OnPlayerLeave(s.itemsLeave)
	s.OnTick(s.checkPickUps)
	s.OnPlayerArrived(s.pickUpArrived)
	s.loop.OnTickEnd(func(time.Time) { s.flushSaveRequests() })
	return errors.Join(
		s.Register(clientpackets.OpUseItem, s.handleUseItem),
		s.Register(clientpackets.OpRequestUnEquipItem, s.handleUnEquipItem),
		s.Register(clientpackets.OpRequestDropItem, s.handleDropItem),
		s.Register(clientpackets.OpRequestDestroyItem, s.handleDestroyItem),
		s.Register(clientpackets.OpRequestItemList, s.handleItemList),
		s.registerItemGMCommands(),
	)
}

// weightPenaltyModifiers converts skill 4270's levels into modifiers.
func (s *Server) weightPenaltyModifiers() [5][]stats.Modifier {
	var mods [5][]stats.Modifier
	if s.items.passives == nil {
		return mods
	}
	for level := 1; level <= 4; level++ {
		l, ok := s.items.passives.Level(data.SkillWeightPenalty, level)
		if !ok {
			s.log.Warn("game: the passive skill data has no weight penalty level", "level", level)
			continue
		}
		for _, fn := range l.Funcs {
			st, ok1 := stats.ParseStat(fn.Stat.String())
			op, ok2 := stats.ParseOp(fn.Op.String())
			if !ok1 || !ok2 {
				s.log.Warn("game: weight penalty function not supported", "stat", fn.Stat.String(), "op", fn.Op.String())
				continue
			}
			mods[level] = append(mods[level], stats.Modifier{Stat: st, Op: op, Order: fn.Order, Value: fn.Value, Owner: weightPenaltyOwner})
		}
	}
	return mods
}

// loadItems builds the inventory and the warehouse of a loaded character
// (Mobius Player.restore): worn items go on with the equip rules and add
// their stats, and the weight penalty is set. It touches no world state.
func (s *Server) loadItems(p *Player, items []storage.Item) {
	f := &s.items
	tracker := inventory.NewTracker()
	opts := inventory.Options{Templates: s.data, IDs: f.ids, Tracker: tracker, Config: f.cfg}
	inv := inventory.NewInventory(p.stats, opts)
	inv.AddListener(&inventory.StatsListener{Stats: p.stats, OnError: func(it *inventory.Item, err error) {
		s.log.Warn("game: equipped item has no usable stats", append(p.logArgs(), "item", it.ItemID(), "slot", it.Slot(), "err", err)...)
	}})
	wh := inventory.NewWarehouse(func() int { return s.warehouseLimit(p) }, opts)

	items = slices.Clone(items)
	for i := range items {
		// Only shadow items have mana (Mobius -1 for the others); a stored 0
		// on another item would make it a spent shadow item.
		if t, ok := s.data.ItemTemplate(int(items[i].ItemID)); ok && t.Duration <= 0 {
			items[i].Mana = storage.NoMana
		}
	}
	unknown := inv.Load(items, inventory.Wearer{GM: p.char.AccessLevel >= storage.AccessLevelAdmin})
	unknown = append(unknown, wh.Load(items)...)
	for _, si := range unknown {
		s.log.Warn("game: item without a template, kept but not shown", append(p.logArgs(), "item", si.ItemID, "itemObjectId", si.ObjectID)...)
	}
	p.inv = Inventory{inv}
	p.items.warehouse, p.items.tracker, p.items.unknown = wh, tracker, unknown
	s.refreshOverloaded(p)
}

// warehouseLimit is the number of slots of p's private warehouse (Mobius
// Player.getWareHouseLimit).
func (s *Server) warehouseLimit(p *Player) int {
	n := s.gs.Items.WarehouseSlotsNoDwarf
	if p.char.Race == stats.RaceDwarf {
		n = s.gs.Items.WarehouseSlotsDwarf
	}
	return n + int(p.stats.Calc(stats.WhLimit, 0))
}

// Warehouse returns p's private warehouse. Move items with the inventory's
// Transfer and call Server.ItemsChanged afterwards.
func (p *Player) Warehouse() *inventory.Container { return p.items.warehouse }

// WeightPenalty returns the weight penalty level, 0-4.
func (p *Player) WeightPenalty() int32 { return p.items.weightPenalty }

// Overloaded reports whether the player carries more than it can: it
// cannot walk (Mobius isOverloaded).
func (p *Player) Overloaded() bool { return p.items.overloaded }

// itemsEnter is the OnPlayerEnter hook: worn shadow items lose one mana
// (Mobius EnterWorld).
func (s *Server) itemsEnter(p *Player) {
	for _, it := range slices.Clone(p.inv.Items()) {
		if it.Shadow() && it.Equipped() {
			s.decreaseMana(p, it)
		}
	}
}

// itemsLeave is the OnPlayerLeave hook: the player's item timers stop and
// waiting packets are dropped. Leaving saves the player itself.
func (s *Server) itemsLeave(p *Player) {
	f := &s.items
	s.loop.Cancel(p.items.iuTimer)
	s.loop.Cancel(p.items.ilTimer)
	p.items.iuTimer, p.items.ilTimer, p.items.iu = 0, 0, nil
	for it, task := range f.mana {
		if task.p == p {
			s.loop.Cancel(task.timer)
			delete(f.mana, it)
		}
	}
	delete(f.pickups, p)
}

// ItemsChanged sends p what changed in its items since the last call: an
// InventoryUpdate 100 ms later (merged with the waiting one), StatusUpdate
// with the load, and the weight penalty when it changed; then it asks for
// a save at the end of the tick. Call it after changing p's inventory or
// warehouse directly. Loop-only.
func (s *Server) ItemsChanged(p *Player) { s.itemsChanged(p, true, nil) }

// itemsChanged is ItemsChanged; load sends StatusUpdate with the load, and
// skip leaves an item out of the InventoryUpdate.
func (s *Server) itemsChanged(p *Player, load bool, skip *inventory.Item) {
	s.queueInventoryUpdate(p, s.takeInventoryUpdates(p, skip)...)
	if load && p.InGame() {
		p.Send(loadStatusUpdate(p))
	}
	s.refreshOverloaded(p)
	s.RequestSave(p)
}

// takeInventoryUpdates returns the changes the inventory recorded as
// InventoryUpdate entries, leaving out skip. The warehouse's own changes
// are dropped: its windows send their own lists.
func (s *Server) takeInventoryUpdates(p *Player, skip *inventory.Item) []serverpackets.InventoryItem {
	if p.items.warehouse != nil {
		p.items.warehouse.TakeUpdates()
	}
	ups := p.inv.TakeUpdates()
	s.itemsRemoved(p, ups) // shortcut.go: the shortcuts of items that left
	out := make([]serverpackets.InventoryItem, 0, len(ups))
	for _, u := range ups {
		if u.Item != skip {
			out = append(out, serverpackets.InventoryItem{Change: serverpackets.ItemChange(u.Change), Item: ItemInfo(u.Item)})
		}
	}
	return out
}

// loadStatusUpdate is StatusUpdate with p's load (Mobius CUR_LOAD).
func loadStatusUpdate(p *Player) []byte {
	return serverpackets.NewStatusUpdatePacket(serverpackets.StatusUpdate{
		ObjectID: p.ObjectID(),
		Values:   []serverpackets.StatusValue{{Attr: serverpackets.StatusCurLoad, Value: p.inv.Load()}},
	})
}

// queueInventoryUpdate sends changes to p in an InventoryUpdate 100 ms after
// the last change, merged with the changes queued meanwhile (a later change
// of an item replaces an earlier one), as Mobius sendInventoryUpdate does.
func (s *Server) queueInventoryUpdate(p *Player, changes ...serverpackets.InventoryItem) {
	if len(changes) == 0 || p.state == stateLeft {
		return
	}
	st := &p.items
	for _, c := range changes {
		i := slices.IndexFunc(st.iu, func(q serverpackets.InventoryItem) bool { return q.Item.ObjectID == c.Item.ObjectID })
		if i >= 0 {
			st.iu[i] = c
		} else {
			st.iu = append(st.iu, c)
		}
	}
	s.loop.Cancel(st.iuTimer)
	st.iuTimer = s.loop.After(inventoryUpdateDelay, func() {
		st.iuTimer = 0
		items := st.iu
		st.iu = nil
		if p.InGame() && len(items) > 0 {
			p.Send(serverpackets.NewInventoryUpdatePacket(serverpackets.InventoryUpdate{Items: items}))
		}
	})
}

// sendInventoryUpdateNow sends changes at once, as Mobius RequestDestroyItem
// does, and drops waiting entries of the same items so that a later
// merged packet cannot undo them.
func (s *Server) sendInventoryUpdateNow(p *Player, changes []serverpackets.InventoryItem) {
	if len(changes) == 0 {
		return
	}
	p.items.iu = slices.DeleteFunc(p.items.iu, func(q serverpackets.InventoryItem) bool {
		return slices.ContainsFunc(changes, func(c serverpackets.InventoryItem) bool { return c.Item.ObjectID == q.Item.ObjectID })
	})
	p.Send(serverpackets.NewInventoryUpdatePacket(serverpackets.InventoryUpdate{Items: changes}))
}

// SendItemList sends p its ItemList 250 ms after the last call (Mobius
// sendItemList); open makes the client open the inventory window, the last
// call's value counts. Loop-only.
func (s *Server) SendItemList(p *Player, open bool) {
	if p.state == stateLeft {
		return
	}
	st := &p.items
	st.ilOpen = open
	s.loop.Cancel(st.ilTimer)
	st.ilTimer = s.loop.After(itemListDelay, func() {
		st.ilTimer = 0
		if p.InGame() {
			p.Send(serverpackets.NewItemListPacket(serverpackets.ItemList{ShowWindow: st.ilOpen, Items: p.inv.ItemInfos()}))
		}
	})
}

// BlockInventory makes the server ignore p's RequestItemList for 1.5 s, as
// Mobius does after a shop or warehouse window opened
// (setInventoryBlockingStatus). Loop-only.
func (s *Server) BlockInventory(p *Player) { p.items.blockedUntil = s.Now().Add(inventoryBlockTime) }

// refreshOverloaded sets the weight penalty from p's load (Mobius
// refreshOverloaded): (load - weightPenalty stat, base 1) × 1000 / maximum
// below 500 is level 0, below 666 level 1, below 800 level 2, below 1000
// level 3, else 4. A new level applies skill 4270's modifiers, sets the
// overloaded flag (load above the maximum) and, in the game, sends UserInfo,
// CharInfo and EtcStatusUpdate.
func (s *Server) refreshOverloaded(p *Player) {
	maxLoad := int64(p.stats.WeightLimit())
	if maxLoad <= 0 {
		return
	}
	load := p.inv.Weight()
	proc := (load - int64(p.stats.Calc(stats.WeightPenalty, 1))) * 1000 / maxLoad
	var level int32
	switch {
	case proc < 500 || p.admin.diet: // //diet (admin_gm.go)
		level = 0
	case proc < 666:
		level = 1
	case proc < 800:
		level = 2
	case proc < 1000:
		level = 3
	default:
		level = 4
	}
	if level == p.items.weightPenalty {
		return
	}
	p.items.weightPenalty = level
	p.stats.RemoveOwner(weightPenaltyOwner)
	if level > 0 {
		if mods := s.items.weightMods[level]; len(mods) > 0 {
			if err := p.stats.AddModifiers(mods...); err != nil {
				s.log.Warn("game: weight penalty modifiers refused", append(p.logArgs(), "err", err)...)
			}
		}
		p.items.overloaded = load > maxLoad
	} else {
		p.items.overloaded = false
	}
	if p.InGame() {
		p.BroadcastUserInfo()
		p.Send(etcStatusPacket(p))
	}
}

// etcStatusPacket is EtcStatusUpdate with p's state: the force charges, the
// weight penalty, the danger area icon, the grade penalty, the Charm of
// Courage and the death penalty level.
func etcStatusPacket(p *Player) []byte {
	var charges int32
	if p.skill != nil {
		charges = int32(p.skill.charges) //nolint:gosec // 0-7
	}
	return serverpackets.NewEtcStatusUpdatePacket(serverpackets.EtcStatusUpdate{
		Charges:           charges,
		WeightPenalty:     p.items.weightPenalty,
		ChatBlocked:       p.chatBlockIcon(),
		DangerArea:        p.Zones.Inside(ZoneDangerArea),
		GradePenalty:      p.GradePenalty(),
		CharmOfCourage:    p.CharmOfCourage(),
		DeathPenaltyLevel: p.char.DeathPenaltyLevel,
	})
}

// chatBlockIcon reports whether EtcStatusUpdate shows the chat block icon
// (Mobius: message refusal, a chat ban or silence mode): //silence
// (admin_gm.go), a chat ban (admin_char.go), message refusal (friends.go).
func (p *Player) chatBlockIcon() bool {
	return p.admin.silence || p.chatBanned() || p.social.refuseAll
}

// itemCondition is a player as item conditions test it.
type itemCondition struct {
	p    *Player
	cats *data.Categories
}

func (c itemCondition) Race() data.Race { return data.Race(c.p.char.Race) } //nolint:gosec // races are small

func (c itemCondition) Level() int { return c.p.Level() }

func (c itemCondition) Female() bool { return c.p.Female() }

func (c itemCondition) PKKills() int { return int(c.p.char.PKKills) }

func (c itemCondition) Hero() bool { return false } // no heroes yet

func (c itemCondition) PvPFlagged() bool { return c.p.PvPFlag != 0 }

func (c itemCondition) Clan() conditions.Clan { return conditions.Clan{ID: c.p.char.ClanID} }

func (c itemCondition) InCategory(t data.CategoryType) bool {
	return c.cats != nil && c.cats.InCategory(t, c.p.ClassID())
}

// checkItemCondition tests the <conditions> of it for p and tells p why it
// fails (Mobius ItemTemplate.checkCondition with sendMessage).
func (s *Server) checkItemCondition(p *Player, it *inventory.Item) bool {
	res := s.items.conds.Check(int(it.ItemID()), itemCondition{p: p, cats: s.items.categories})
	if res.OK {
		return true
	}
	switch {
	case res.Msg != "":
		p.SendMessage(res.Msg)
	case res.MsgID != 0:
		var params []serverpackets.SystemMessageParam
		if res.AddName {
			params = append(params, s.itemNameParam(it.ItemID()))
		}
		p.SendSystemMessage(serverpackets.SystemMessageID(res.MsgID), params...) //nolint:gosec // message IDs fit in int32
	}
	return false
}

// illegalAction logs a forged request and kicks the player 5 s later, as
// Mobius's default punishment does (PunishmentManager, KICK).
func (s *Server) illegalAction(p *Player, what string) {
	s.log.Warn("game: illegal action, kicking the player", append(p.logArgs(), "action", what)...)
	p.SendMessage("You will be kicked for illegal action, GM informed.")
	p.after(illegalActionKickDelay, func() {
		sess := p.session
		s.leave(p, true, nil)
		if sess != nil {
			sess.CloseAfter(leaveWorldPacket, 0)
		}
	})
}

// flood reports whether an action limited by interval came too early, and
// otherwise sets when the next may come (Mobius FloodProtectorAction).
func (s *Server) flood(next *time.Time, interval time.Duration) bool {
	if interval <= 0 {
		return false
	}
	now := s.Now()
	if now.Before(*next) {
		return true
	}
	*next = now.Add(interval)
	return false
}
