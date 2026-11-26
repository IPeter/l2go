package game

import (
	"time"

	"github.com/IPeter/l2go/internal/craft"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/world"
)

// The private workshop (Mobius RequestActionUse actions 37 and 51 and the
// RequestRecipeShop* packets): a crafter sits in the store mode of M9's
// private stores (StoreManufacture) and offers recipes of his books for a
// fee; customers order with a second click on him. The list and its
// message stay while the crafter is in the game and are not saved.

// workshopMsgCannotOpenHere is "You cannot open a Private Workshop here.".
const workshopMsgCannotOpenHere serverpackets.SystemMessageID = 1297

// workshopOrderRange is how close a customer must be to order (Mobius
// RequestRecipeShopMakeItem: checkIfInRange(150, ...) in 3D, plus both
// collision radii).
const workshopOrderRange = 150

// actionDwarvenManufacture is RequestActionUse action 37: a dead player or a
// rider (mount.go) gets ActionFailed (no buff stores yet); others get the
// setup window of a dwarven workshop.
func (s *Server) actionDwarvenManufacture(p *Player, _ clientpackets.RequestActionUse) {
	if p.IsDead() || s.mounted(p) {
		p.SendActionFailed()
		return
	}
	s.openWorkshopSetup(p, true)
}

// actionCommonManufacture is RequestActionUse action 51: the setup window
// of a common workshop.
func (s *Server) actionCommonManufacture(p *Player, _ clientpackets.RequestActionUse) {
	if p.IsDead() {
		p.SendActionFailed()
		return
	}
	s.openWorkshopSetup(p, false)
}

// openWorkshopSetup closes a store p runs (UserInfo and CharInfo show it),
// stands p up and sends the setup window. Unlike the other stores Mobius
// sets no setup store mode.
func (s *Server) openWorkshopSetup(p *Player, dwarven bool) {
	if p.InStoreMode() {
		p.store.typ = StoreNone
		p.BroadcastUserInfo()
	}
	if p.Sitting {
		s.standUp(p)
	}
	p.Send(s.workshopManageList(p, dwarven))
}

// recipeIsDwarven reports whether a recipe belongs to the dwarven book.
func (s *Server) recipeIsDwarven(id int32) bool {
	r, ok := s.Recipe(int(id))
	return ok && r.Dwarven
}

// workshopManageList builds the setup window (Mobius RecipeShopManageList):
// the dwarven book for a dwarven window when p has Create Item, else the
// common book; the list first loses the other book's recipes and those p
// forgot.
func (s *Server) workshopManageList(p *Player, dwarven bool) []byte {
	book := p.RecipeBook(false)
	if dwarven && p.CraftLevel(true) >= 1 {
		book = p.RecipeBook(true)
	}
	p.craft.shop = craft.FitShopList(p.craft.shop, dwarven, s.recipeIsDwarven, p.HasRecipe)
	return serverpackets.NewRecipeShopManageListPacket(serverpackets.RecipeShopManageList{
		SellerID:  p.ObjectID(),
		Adena:     itemCount32(p.inv.Adena()),
		Dwarven:   dwarven,
		RecipeIDs: book,
		Items:     workshopPacketItems(p.craft.shop),
	})
}

// workshopPacketItems is a workshop list as the packets write it.
func workshopPacketItems(list []craft.ShopItem) []serverpackets.RecipeShopItem {
	out := make([]serverpackets.RecipeShopItem, len(list))
	for i, e := range list {
		out[i] = serverpackets.RecipeShopItem{RecipeID: e.RecipeID, Price: itemCount32(e.Price)}
	}
	return out
}

// workshopSellList builds the window of owner's workshop as viewer sees it
// (Mobius RecipeShopSellList).
func (s *Server) workshopSellList(viewer, owner *Player) []byte {
	return serverpackets.NewRecipeShopSellListPacket(serverpackets.RecipeShopSellList{
		ManufacturerID: owner.ObjectID(),
		CurMP:          javaInt32(owner.CurMP),
		MaxMP:          int32(owner.stats.MaxMP()), //nolint:gosec // MP fits
		Adena:          itemCount32(viewer.inv.Adena()),
		Items:          workshopPacketItems(owner.craft.shop),
	})
}

// workshopItemInfoPacket is RecipeShopItemInfo of a recipe of crafter's
// workshop, with his MP.
func workshopItemInfoPacket(crafter *Player, recipeID int32) []byte {
	return serverpackets.NewRecipeShopItemInfoPacket(serverpackets.RecipeShopItemInfo{
		ManufacturerID: crafter.ObjectID(),
		RecipeID:       recipeID,
		CurMP:          javaInt32(crafter.CurMP),
		MaxMP:          int32(crafter.stats.MaxMP()), //nolint:gosec // MP fits
	})
}

// handleRecipeShopListSet handles RequestRecipeShopListSet (0xB2) in Mobius's
// order: an entry naming an unknown recipe drops the packet (Mobius fails to
// read it); a refused list closes the store; in combat stance, in a no-store
// zone or where canOpenStore refuses, p gets ActionFailed (with a message
// for the first two); a recipe p does not know or a fee over MaxAdena is an
// illegal action; then the workshop opens: p sits, UserInfo and CharInfo
// show it and p and the players around get its message.
func (s *Server) handleRecipeShopListSet(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestRecipeShopListSet(body)
	if err != nil {
		s.log.Debug("game: bad RequestRecipeShopListSet", append(p.logArgs(), "err", err)...)
		return
	}
	for _, it := range req.Items {
		if _, ok := s.Recipe(int(it.RecipeID)); !ok {
			s.log.Debug("game: RequestRecipeShopListSet with an unknown recipe", append(p.logArgs(), "recipe", it.RecipeID)...)
			return
		}
	}
	if req.Refused {
		p.store.typ = StoreNone
		p.BroadcastUserInfo()
		return
	}
	switch {
	case s.InAttackStance(p):
		p.SendSystemMessage(storeMsgInCombat)
		p.SendActionFailed()
		return
	case p.Zones.Inside(ZoneNoStore):
		p.SendSystemMessage(workshopMsgCannotOpenHere)
		p.SendActionFailed()
		return
	case !s.canOpenStore(p):
		p.SendActionFailed()
		return
	}
	entries := make([]craft.ShopItem, len(req.Items))
	for i, it := range req.Items {
		entries[i] = craft.ShopItem{RecipeID: it.RecipeID, Price: int64(it.Cost)}
	}
	list, refusal := craft.BuildShopList(entries, p.HasRecipe, s.items.cfg.MaxAdena)
	p.craft.shop = nil
	switch refusal {
	case craft.ShopNotInBook:
		s.illegalAction(p, "a workshop recipe it does not have")
		return
	case craft.ShopPriceTooHigh:
		s.illegalAction(p, "a workshop fee over MaxAdena")
		return
	}
	p.craft.shop = list
	p.store.typ = StoreManufacture
	s.sitDown(p)
	s.openStore(p, serverpackets.NewRecipeShopMsgPacket(p.ObjectID(), p.craft.shopTitle))
}

// handleRecipeShopMessageSet handles RequestRecipeShopMessageSet (0xB1): the
// workshop's message; one over 29 characters is punished. Mobius sends
// nothing back.
func (s *Server) handleRecipeShopMessageSet(p *Player, body []byte) {
	req, _ := clientpackets.ParseRequestRecipeShopMessageSet(body)
	if !s.storeMessageOK(p, req.Message) {
		return
	}
	p.craft.shopTitle = req.Message
}

// handleRecipeShopManageQuit handles RequestRecipeShopManageQuit (0xB3): the
// store closes, UserInfo and CharInfo show it and p stands up.
func (s *Server) handleRecipeShopManageQuit(p *Player, _ []byte) {
	p.store.typ = StoreNone
	p.BroadcastUserInfo()
	s.standUp(p)
}

// handleRecipeShopMakeInfo handles RequestRecipeShopMakeInfo (0xB5): the
// craft window of a recipe at a workshop, anywhere in the world (Mobius
// checks no range).
func (s *Server) handleRecipeShopMakeInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestRecipeShopMakeInfo(body)
	if err != nil {
		s.log.Debug("game: bad RequestRecipeShopMakeInfo", append(p.logArgs(), "err", err)...)
		return
	}
	shop, ok := s.Player(world.ObjectID(req.ShopID))
	if !ok || shop.store.typ != StoreManufacture {
		return
	}
	p.Send(workshopItemInfoPacket(shop, req.RecipeID))
}

// handleRecipeShopMakeItem handles RequestRecipeShopMakeItem (0xB6) in
// Mobius's order: the manufacture flood limit; the crafter in the game and
// in p's instance; p not in store mode ("You cannot create items while
// trading."); the crafter's workshop open; within 150 of p; then the order
// (craft_make.go).
func (s *Server) handleRecipeShopMakeItem(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestRecipeShopMakeItem(body)
	if err != nil {
		s.log.Debug("game: bad RequestRecipeShopMakeItem", append(p.logArgs(), "err", err)...)
		return
	}
	if s.flood(&p.craft.nextManufacture, time.Duration(s.gs.Crafting.FloodManufacture)) {
		return
	}
	crafter, ok := s.Player(world.ObjectID(req.ShopID))
	if !ok || (crafter.InstanceID() != p.InstanceID() && p.InstanceID() != -1) {
		return
	}
	if p.InStoreMode() {
		p.SendMessage("You cannot create items while trading.")
		return
	}
	if crafter.store.typ != StoreManufacture {
		return
	}
	if !s.workshopInReach(p, crafter) {
		return
	}
	s.requestManufactureItem(crafter, req.RecipeID, p)
}

// workshopInReach is Mobius checkIfInRange(150, customer, crafter, true): the
// same instance, and within 150 plus both collision radii in 3D.
func (s *Server) workshopInReach(customer, crafter *Player) bool {
	if customer.InstanceID() != crafter.InstanceID() {
		return false
	}
	reach := float64(workshopOrderRange) + float64(int(customer.lastView.CollisionRadius)) + float64(int(crafter.lastView.CollisionRadius))
	return customer.Position().Distance3D(crafter.Position()) <= reach
}

// handleRecipeShopManagePrev handles RequestRecipeShopManagePrev (0xB7): the
// workshop window of p's target again; a dead p or a target that is not a
// player gets ActionFailed.
func (s *Server) handleRecipeShopManagePrev(p *Player, _ []byte) {
	owner, ok := p.Target.(*Player)
	if p.IsDead() || !ok || owner == nil {
		p.SendActionFailed()
		return
	}
	p.Send(s.workshopSellList(p, owner))
}
