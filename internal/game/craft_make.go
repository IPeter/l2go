package game

import (
	"fmt"

	"github.com/IPeter/l2go/internal/craft"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/storage"
)

// Making items (Mobius RecipeManager.requestMakeItem, requestManufactureItem
// and RecipeItemMaker with AltGameCreation = False): the checks of the
// maker's constructor, then finishCrafting at once. A workshop order is the
// same with the customer as the target: his materials and adena, the
// crafter's skill, MP and recipe.

// System messages of making items.
const (
	craftMsgCreatedForS1       serverpackets.SystemMessageID = 1145 // "$s2 has been created for $s1 after the payment of $s3 adena is received."
	craftMsgS1CreatedS2        serverpackets.SystemMessageID = 1146 // "$s1 created $s2 after receiving $s3 adena."
	craftMsgCreatedCountForS1  serverpackets.SystemMessageID = 1147 // "$s2 $s3 have been created for $s1 at the price of $s4 adena."
	craftMsgS1CreatedCount     serverpackets.SystemMessageID = 1148 // "$s1 created $s2 $s3 at the price of $s4 adena."
	craftMsgAttemptForS1Failed serverpackets.SystemMessageID = 1149 // "The attempt to create $s2 for $s1 at the price of $s3 adena has failed."
	craftMsgS1FailedToCreate   serverpackets.SystemMessageID = 1150 // "$s1 has failed to create $s2 at the price of $s3 adena."
)

// maker is one craft (Mobius RecipeItemMaker): crafter makes r for target,
// who is the crafter himself when he crafts from his own book.
type maker struct {
	s               *Server
	crafter, target *Player
	r               *data.Recipe
	// price is the workshop fee the target pays; 0 for his own craft.
	price int64
}

// workshop reports whether the craft is a workshop order.
func (m *maker) workshop() bool { return m.crafter != m.target }

// requestMakeItem is Mobius RecipeManager.requestMakeItem: p crafts from its
// own book. Not in combat stance unless CraftingInCombat; the recipe must
// be valid ("No recipe for") and in p's books (else an illegal action).
func (s *Server) requestMakeItem(p *Player, id int32) {
	if s.InAttackStance(p) && !s.gs.Crafting.CraftingInCombat {
		p.SendSystemMessage(storeMsgInCombat)
		return
	}
	r := s.validRecipe(p, id)
	if r == nil {
		return
	}
	if !p.HasRecipe(id) {
		s.illegalAction(p, "a craft of a recipe it does not have")
		return
	}
	if m := s.newMaker(p, r, p); m != nil {
		m.finish()
	}
}

// requestManufactureItem is Mobius RecipeManager.requestManufactureItem:
// customer orders recipe id from crafter's workshop. The recipe must be
// valid and in the crafter's books, else the customer is punished.
func (s *Server) requestManufactureItem(crafter *Player, id int32, customer *Player) {
	r := s.validRecipe(customer, id)
	if r == nil {
		return
	}
	if !crafter.HasRecipe(id) {
		s.illegalAction(customer, "a workshop order of a recipe the crafter does not have")
		return
	}
	if m := s.newMaker(crafter, r, customer); m != nil {
		m.finish()
	}
}

// validRecipe is Mobius RecipeData.getValidRecipeList: a known recipe with
// ingredients, else p is told "RecipeData: No recipe for: <id>".
func (s *Server) validRecipe(p *Player, id int32) *data.Recipe {
	r, ok := s.Recipe(int(id))
	if !ok || !craft.Valid(r) {
		p.SendMessage(fmt.Sprintf("RecipeData: No recipe for: %d", id))
		return nil
	}
	return r
}

// newMaker runs the checks of Mobius RecipeItemMaker's constructor in order
// and returns nil when one fails (the target then gets the craft window with
// a failure): a dead crafter or target, either one in a request or a trade,
// a recipe beyond the crafter's create level; for a workshop order the fee
// of the crafter's list and the target's adena; the materials, then the HP
// and MP the crafter has. Mobius aborts twice when the materials or the
// stats fall short, so the target gets two failed windows then.
func (s *Server) newMaker(crafter *Player, r *data.Recipe, target *Player) *maker {
	m := &maker{s: s, crafter: crafter, target: target, r: r}
	fail := func(p *Player) *maker {
		if p != nil {
			p.SendActionFailed()
		}
		m.abort()
		return nil
	}
	switch {
	case crafter.IsDead():
		return fail(crafter)
	case target.IsDead():
		return fail(target)
	case s.ProcessingTransaction(target):
		return fail(target)
	case s.ProcessingTransaction(crafter):
		return fail(crafter)
	case craft.CheckMake(r, crafter.CraftLevel(r.Dwarven)) != craft.MakeOK:
		return fail(crafter)
	}
	if m.workshop() {
		price, ok := craft.ShopPrice(crafter.craft.shop, int32(r.ID)) //nolint:gosec // recipe IDs are small
		if !ok {
			return fail(nil)
		}
		m.price = price
		if target.inv.Adena() < price {
			target.SendSystemMessage(sysMsgNotEnoughAdena)
			return fail(nil)
		}
	}
	if !m.checkMaterials() {
		return fail(nil)
	}
	if !m.useStats(false) {
		return fail(nil)
	}
	m.updateMakeInfo(true)
	m.sendCurMP()
	m.sendCurLoad()
	return m
}

// abort is RecipeItemMaker.abort: the target gets the window with a failure.
func (m *maker) abort() { m.updateMakeInfo(false) }

// updateMakeInfo sends the target his craft window: RecipeItemMakeInfo for
// his own craft, RecipeShopItemInfo with the crafter's MP for an order.
func (m *maker) updateMakeInfo(success bool) {
	if !m.workshop() {
		m.target.Send(recipeMakeInfoPacket(m.target, m.r, success))
		return
	}
	m.target.Send(workshopItemInfoPacket(m.crafter, int32(m.r.ID))) //nolint:gosec // recipe IDs are small
}

// sendCurMP sends the target StatusUpdate with his MP (Mobius updateCurMp).
func (m *maker) sendCurMP() {
	m.target.Send(serverpackets.NewStatusUpdatePacket(serverpackets.StatusUpdate{
		ObjectID: m.target.ObjectID(),
		Values:   []serverpackets.StatusValue{{Attr: serverpackets.StatusCurMP, Value: javaInt32(m.target.CurMP)}},
	}))
}

// sendCurLoad sends the target StatusUpdate with his load (Mobius
// updateCurLoad).
func (m *maker) sendCurLoad() { m.target.Send(loadStatusUpdate(m.target)) }

// checkMaterials is Mobius listItems(false): the target has every
// ingredient, else he is told the first one missing and the craft aborts.
func (m *maker) checkMaterials() bool {
	short, missing := craft.FirstShortfall(m.r, func(itemID int) int64 {
		if it := m.target.inv.ByItemID(int32(itemID)); it != nil { //nolint:gosec // item IDs fit
			return it.Count()
		}
		return 0
	})
	if !missing {
		return true
	}
	m.target.SendSystemMessage(craftMsgMissing, m.s.itemNameParam(int32(short.ItemID)), //nolint:gosec // item IDs fit
		serverpackets.NumberParam(itemCount32(short.Missing)))
	m.abort()
	return false
}

// useStats is Mobius calculateStatUse: each statUse the crafter cannot pay
// tells the target "Not enough HP." or "Not enough MP." and aborts; with
// reduce the others are paid, HP as damage the crafter does to himself.
func (m *maker) useStats(reduce bool) bool {
	ok := true
	for _, st := range m.r.StatUse {
		one := data.Recipe{StatUse: []data.RecipeStat{st}}
		if short := craft.StatShortfalls(&one, m.crafter.CurHP, m.crafter.CurMP); len(short) > 0 {
			if short[0] == craft.NotEnoughHP {
				m.target.SendSystemMessage(sysMsgNotEnoughHP)
			} else {
				m.target.SendSystemMessage(sysMsgNotEnoughMP)
			}
			m.abort()
			ok = false
			continue
		}
		if !reduce {
			continue
		}
		switch st.Stat {
		case data.RecipeStatHP:
			m.crafter.ReduceHP(float64(st.Value), m.crafter)
		case data.RecipeStatMP:
			m.crafter.ReduceMP(float64(st.Value))
		}
	}
	return ok
}

// finish is Mobius run and finishCrafting: crafting must be on; the crafter
// pays HP and MP; a customer pays the fee (refunded when his materials fall
// short); the materials go, each with its message; the success roll gives
// the product or the failure messages; the target gets his inventory and
// the window. A workshop order is saved jointly; an own craft at the end of
// the tick.
func (m *maker) finish() {
	s := m.s
	if !s.gs.Crafting.Enabled {
		m.target.SendMessage("Item creation is currently disabled.")
		m.abort()
		return
	}
	m.useStats(true)
	paid := m.workshop() && m.price > 0
	if paid {
		if !s.storeMoveAdena(m.target, m.crafter, m.price) {
			m.target.SendSystemMessage(sysMsgNotEnoughAdena)
			m.abort()
			m.save()
			return
		}
		s.craftAdenaMoved(m.target, m.crafter)
	}
	if m.takeMaterials() {
		success := craft.Succeeds(m.r, s.rnd)
		if success {
			m.reward()
		} else {
			m.failed()
		}
		s.sendInventoryNow(m.target)
		m.updateMakeInfo(success)
	} else if paid {
		s.storeMoveAdena(m.crafter, m.target, m.price)
		s.craftAdenaMoved(m.crafter, m.target)
	}
	m.sendCurMP()
	m.sendCurLoad()
	m.save()
}

// takeMaterials is Mobius listItems(true): the materials are checked again,
// then each is destroyed with "$s2 $s1 has disappeared." or "$s1 has
// disappeared.". It reports false when one fell short.
func (m *maker) takeMaterials() bool {
	if !m.checkMaterials() {
		return false
	}
	for _, in := range craft.Materials(m.r) {
		id, n := int32(in.ItemID), int64(in.Count) //nolint:gosec // item IDs fit
		if _, err := m.target.inv.DestroyByItemID(id, n); err != nil {
			m.s.log.Error("game: a craft material was not taken", append(m.target.logArgs(), "item", id, "count", n, "err", err)...)
		}
		m.s.sendDisappeared(m.target, id, n)
	}
	return true
}

// reward is Mobius rewardPlayer: the target gets the product; for an order
// the crafter and the customer are told who made what for how much; the
// target gets "You have earned ..." and the window with a success.
func (m *maker) reward() {
	s, target := m.s, m.target
	itemID, count := int32(m.r.Product.ItemID), int64(m.r.Product.Count) //nolint:gosec // item IDs fit
	if _, err := target.inv.Create(itemID, count); err != nil {
		s.log.Error("game: a crafted item was not created", append(target.logArgs(), "item", itemID, "count", count, "err", err)...)
	}
	name := s.itemNameParam(itemID)
	if m.workshop() {
		price := serverpackets.NumberParam(itemCount32(m.price))
		if count == 1 {
			m.crafter.SendSystemMessage(craftMsgCreatedForS1, serverpackets.TextParam(target.Name()), name, price)
			target.SendSystemMessage(craftMsgS1CreatedS2, serverpackets.TextParam(m.crafter.Name()), name, price)
		} else {
			n := serverpackets.NumberParam(itemCount32(count))
			m.crafter.SendSystemMessage(craftMsgCreatedCountForS1, serverpackets.TextParam(target.Name()), n, name, price)
			target.SendSystemMessage(craftMsgS1CreatedCount, serverpackets.TextParam(m.crafter.Name()), n, name, price)
		}
	}
	if count > 1 {
		target.SendSystemMessage(sysMsgEarnedS2S1, name, serverpackets.NumberParam(itemCount32(count)))
	} else {
		target.SendSystemMessage(sysMsgEarnedS1, name)
	}
	m.updateMakeInfo(true)
}

// failed tells a failed craft: for an order both sides with the fee, else
// "You failed at mixing the item.".
func (m *maker) failed() {
	if !m.workshop() {
		m.target.SendSystemMessage(craftMsgFailedMixing)
		return
	}
	name := m.s.itemNameParam(int32(m.r.Product.ItemID)) //nolint:gosec // item IDs fit
	price := serverpackets.NumberParam(itemCount32(m.price))
	m.crafter.SendSystemMessage(craftMsgAttemptForS1Failed, serverpackets.TextParam(m.target.Name()), name, price)
	m.target.SendSystemMessage(craftMsgS1FailedToCreate, serverpackets.TextParam(m.crafter.Name()), name, price)
}

// save writes the craft: an order in one transaction with both players (the
// fee and the items, decision T5), an own craft at the end of the tick.
func (m *maker) save() {
	s := m.s
	if !m.workshop() {
		s.refreshOverloaded(m.target)
		s.RequestSave(m.target)
		return
	}
	for _, p := range []*Player{m.crafter, m.target} {
		s.queueInventoryUpdate(p, s.takeInventoryUpdates(p, nil)...)
		s.refreshOverloaded(p)
	}
	s.SaveJoint([]*Player{m.crafter, m.target}, storage.Batch{})
}

// craftAdenaMoved sends what a fee or its refund changed, as Mobius
// Player.transferItem does: the payer's InventoryUpdate (queued) and load,
// then the payee's.
func (s *Server) craftAdenaMoved(payer, payee *Player) {
	for _, p := range []*Player{payer, payee} {
		s.queueInventoryUpdate(p, s.takeInventoryUpdates(p, nil)...)
		if p.InGame() {
			p.Send(loadStatusUpdate(p))
		}
	}
}

// sendInventoryNow sends p's waiting InventoryUpdate and then ItemList at
// once (Mobius finishCrafting: flushInventoryUpdate, then ItemList, since
// the craft window counts the client's inventory).
func (s *Server) sendInventoryNow(p *Player) {
	s.queueInventoryUpdate(p, s.takeInventoryUpdates(p, nil)...)
	st := &p.items
	s.loop.Cancel(st.iuTimer)
	st.iuTimer = 0
	if items := st.iu; len(items) > 0 {
		st.iu = nil
		p.Send(serverpackets.NewInventoryUpdatePacket(serverpackets.InventoryUpdate{Items: items}))
	}
	p.Send(serverpackets.NewItemListPacket(serverpackets.ItemList{Items: p.inv.ItemInfos()}))
}
