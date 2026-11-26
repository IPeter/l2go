package game

import (
	"errors"
	"io/fs"
	"time"

	"github.com/IPeter/l2go/internal/craft"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
)

// Crafting (Plan 14 M11b): the recipe books (this file and recipe.go), making
// items from them (craft_make.go) and the private workshop (workshop.go). The
// rules are internal/craft's; Mobius RecipeManager, RecipeData, the Recipes
// item handler and the recipe client packets.
//
// The books are persisted with the character (storage.RecipeBook): the
// common book is shared, the dwarven book belongs to the active class index
// (decision X3). Crafting is instant (AltGameCreation = False), so Mobius's
// "crafting" state and its active makers never outlive a packet and are not
// kept.
func init() { registerFeature("crafting", setupCrafting) }

// handlerRecipes is the item handler of the recipe items (Mobius
// handlers/items/Recipes).
const handlerRecipes = "Recipes"

// System messages of crafting.
const (
	craftMsgAlterWhileCrafting serverpackets.SystemMessageID = 853  // "You may not alter your recipe book while engaged in manufacturing."
	craftMsgCreateLevelTooLow  serverpackets.SystemMessageID = 404  // "Your Create Item level is too low to register this recipe."
	craftMsgAlreadyRegistered  serverpackets.SystemMessageID = 840  // "That recipe is already registered."
	craftMsgS1Added            serverpackets.SystemMessageID = 851  // "$s1 has been added."
	craftMsgMissing            serverpackets.SystemMessageID = 854  // "You are missing $s2 $s1 required to create that."
	craftMsgUpToS1Recipes      serverpackets.SystemMessageID = 894  // "Up to $s1 recipes can be registered."
	craftMsgNoAbility          serverpackets.SystemMessageID = 1061 // "The recipe cannot be registered. You do not have the ability to create items."
	craftMsgBookWhileCasting   serverpackets.SystemMessageID = 1124 // "A recipe book may not be used while using a skill."
	craftMsgNotWhileTrading    serverpackets.SystemMessageID = 1125 // "An item may not be created while engaged in trading."
	craftMsgFailedMixing       serverpackets.SystemMessageID = 719  // "You failed at mixing the item."
)

// craftFeature is the crafting state on the Server (Server.craft).
type craftFeature struct {
	// recipes is the recipe data; nil without it (no recipe registers).
	recipes *data.Recipes
}

// playerCraft is a player's crafting state (Player.craft). The recipe books
// themselves are in Player.char.Recipes, replaced on every change (never
// changed in place), so a snapshot can share them.
type playerCraft struct {
	// shop is the workshop list (Mobius _manufactureItems) and shopTitle
	// its message (Mobius _storeName); both stay while the player is in the
	// game and are not saved (StoreRecipeShopList = False).
	shop      []craft.ShopItem
	shopTitle string
	// nextManufacture is when the next craft or order may come (Mobius
	// FloodProtector Manufacture).
	nextManufacture time.Time
}

// setupCrafting loads the recipes of the data folder and registers the
// recipe packets, the workshop actions and the item handler.
func setupCrafting(s *Server) error {
	if dir := s.settings.DataDir; dir != "" {
		recipes, err := data.LoadRecipes(dir)
		switch {
		case err == nil:
			s.UseRecipes(recipes)
		case errors.Is(err, fs.ErrNotExist):
			s.log.Info("game: no recipes in the data folder", "dir", dir)
		default:
			s.log.Warn("game: recipes not loaded", "err", err)
		}
	}
	return errors.Join(
		s.Register(clientpackets.OpRequestRecipeBookOpen, s.handleRecipeBookOpen),
		s.Register(clientpackets.OpRequestRecipeBookDestroy, s.handleRecipeBookDestroy),
		s.Register(clientpackets.OpRequestRecipeItemMakeInfo, s.handleRecipeItemMakeInfo),
		s.Register(clientpackets.OpRequestRecipeItemMakeSelf, s.handleRecipeItemMakeSelf),
		s.Register(clientpackets.OpRequestRecipeShopMessageSet, s.handleRecipeShopMessageSet),
		s.Register(clientpackets.OpRequestRecipeShopListSet, s.handleRecipeShopListSet),
		s.Register(clientpackets.OpRequestRecipeShopManageQuit, s.handleRecipeShopManageQuit),
		s.Register(clientpackets.OpRequestRecipeShopMakeInfo, s.handleRecipeShopMakeInfo),
		s.Register(clientpackets.OpRequestRecipeShopMakeItem, s.handleRecipeShopMakeItem),
		s.Register(clientpackets.OpRequestRecipeShopManagePrev, s.handleRecipeShopManagePrev),
		s.RegisterActionUse(clientpackets.ActionUseDwarvenManufacture, s.actionDwarvenManufacture),
		s.RegisterActionUse(clientpackets.ActionUseCommonManufacture, s.actionCommonManufacture),
	)
}

// UseRecipes sets the recipe data (setupCrafting loads it from the data
// folder). Recipes naming items the item data lacks are logged. Loop-only
// (or before Run).
func (s *Server) UseRecipes(r *data.Recipes) {
	s.craft.recipes = r
	if r == nil {
		return
	}
	missing := r.MissingItems(func(id int) bool {
		_, ok := s.data.ItemTemplate(id)
		return ok
	})
	if len(missing) > 0 {
		s.log.Warn("game: recipes name unknown items", "recipes", len(missing))
	}
}

// Recipe returns the recipe with the ID. Loop-only.
func (s *Server) Recipe(id int) (*data.Recipe, bool) { return s.craft.recipes.Recipe(id) }

// CraftLevel returns p's create skill level for a book (Mobius
// getDwarvenCraft, getCommonCraft): 0 without the skill.
func (p *Player) CraftLevel(dwarven bool) int {
	if p.skill == nil {
		return 0
	}
	return p.SkillLevel(craft.SkillID(dwarven))
}

// recipeClassIndex is the class index of p's dwarven book: the active
// class's (decision X3).
func (p *Player) recipeClassIndex() int32 {
	return min(max(p.classIndex(), 0), craft.MaxClassIndex) // skill_persist.go
}

// RecipeBook returns p's dwarven book of the active class, or the common
// book, in registration order. The slice must not be changed. Loop-only.
func (p *Player) RecipeBook(dwarven bool) []int32 {
	if dwarven {
		return p.char.Recipes.Dwarven[p.recipeClassIndex()]
	}
	return p.char.Recipes.Common
}

// HasRecipe reports whether either of p's books holds the recipe (Mobius
// hasRecipeList; the dwarven book of the active class). Loop-only.
func (p *Player) HasRecipe(id int32) bool {
	return craft.Contains(p.RecipeBook(true), id) || craft.Contains(p.RecipeBook(false), id)
}

// setRecipeBook replaces one of p's books. The old slices and map are left
// as they were, for the snapshots that share them.
func (p *Player) setRecipeBook(dwarven bool, book []int32) {
	b := p.char.Recipes
	if !dwarven {
		b.Common = book
		p.char.Recipes = b
		return
	}
	m := make(map[int32][]int32, len(b.Dwarven)+1)
	for ci, ids := range b.Dwarven {
		m[ci] = ids
	}
	if len(book) == 0 {
		delete(m, p.recipeClassIndex())
	} else {
		m[p.recipeClassIndex()] = book
	}
	b.Dwarven = m
	p.char.Recipes = b
}

// registerRecipe puts a recipe into its book (Mobius
// registerDwarvenRecipeList, registerCommonRecipeList) and asks for a save.
func (s *Server) registerRecipe(p *Player, r *data.Recipe) {
	p.setRecipeBook(r.Dwarven, craft.Add(p.RecipeBook(r.Dwarven), int32(r.ID))) //nolint:gosec // recipe IDs are small
	s.RequestSave(p)
}

// unregisterRecipe takes a recipe out of p's books (Mobius
// unregisterRecipeList): the dwarven book first, else the common one; then
// its shortcuts go.
func (s *Server) unregisterRecipe(p *Player, id int32) {
	if book, ok := craft.Remove(p.RecipeBook(true), id); ok {
		p.setRecipeBook(true, book)
	} else if book, ok := craft.Remove(p.RecipeBook(false), id); ok {
		p.setRecipeBook(false, book)
	} else {
		s.log.Warn("game: removing a recipe the player does not have", append(p.logArgs(), "recipe", id)...)
	}
	for _, sc := range p.itemSkill.bar.all() {
		if sc.typ == storage.ShortcutRecipe && sc.id == id {
			s.deleteShortcut(p, sc.slot, sc.page, nil)
		}
	}
	s.RequestSave(p)
}

// recipeLimit is how many recipes a book holds (Mobius getDwarfRecipeLimit,
// getCommonRecipeLimit): the configured limit plus the REC_D_LIM or
// REC_C_LIM stat.
func (s *Server) recipeLimit(p *Player, dwarven bool) int {
	if dwarven {
		return s.gs.Crafting.DwarfRecipeLimit + int(p.stats.Calc(stats.DwarfRecipeLimit, 0))
	}
	return s.gs.Crafting.CommonRecipeLimit + int(p.stats.Calc(stats.CommonRecipeLimit, 0))
}

// useRecipe is Mobius's Recipes item handler: a recipe item registers its
// recipe when crafting is on, the player knows the book's create skill at
// the recipe's level and the book has room; then the item is used up with
// "$s1 has been added.". It reports whether the item was used.
func (s *Server) useRecipe(p *Player, it *inventory.Item) bool {
	if !s.gs.Crafting.Enabled {
		p.SendMessage("Crafting is disabled, you cannot register this recipe.")
		return false
	}
	r, ok := s.craft.recipes.ByItem(int(it.ItemID()))
	if !ok {
		return false
	}
	switch craft.CheckRegister(r, p.HasRecipe(int32(r.ID)), p.CraftLevel(r.Dwarven), len(p.RecipeBook(r.Dwarven)), s.recipeLimit(p, r.Dwarven)) { //nolint:gosec // recipe IDs are small
	case craft.RegisterKnown:
		p.SendSystemMessage(craftMsgAlreadyRegistered)
		return false
	case craft.RegisterNoAbility:
		p.SendSystemMessage(craftMsgNoAbility)
		return false
	case craft.RegisterLevelTooLow:
		p.SendSystemMessage(craftMsgCreateLevelTooLow)
		return false
	case craft.RegisterBookFull:
		p.SendSystemMessage(craftMsgUpToS1Recipes, serverpackets.NumberParam(int32(s.recipeLimit(p, r.Dwarven)))) //nolint:gosec // small limits
		return false
	}
	s.registerRecipe(p, r)
	itemID := it.ItemID()
	if _, err := p.inv.Destroy(it, 1); err != nil {
		s.log.Error("game: a registered recipe item was not used up", append(p.logArgs(), "item", itemID, "err", err)...)
	}
	s.ItemsChanged(p)
	p.SendSystemMessage(craftMsgS1Added, s.itemNameParam(itemID))
	return true
}

// recipeBookPacket is RecipeBookItemList of p's book.
func recipeBookPacket(p *Player, dwarven bool) []byte {
	return serverpackets.NewRecipeBookItemListPacket(serverpackets.RecipeBookItemList{
		Dwarven:   dwarven,
		MaxMP:     int32(p.stats.MaxMP()), //nolint:gosec // MP fits
		RecipeIDs: p.RecipeBook(dwarven),
	})
}

// openRecipeBook is Mobius RecipeManager.requestBookOpen: p gets the book.
// Crafting never lasts beyond a packet here, so the "while engaged in
// manufacturing" refusal never applies.
func (s *Server) openRecipeBook(p *Player, dwarven bool) {
	p.Send(recipeBookPacket(p, dwarven))
}

// handleRecipeBookOpen handles RequestRecipeBookOpen (0xAC): not while
// casting ("A recipe book may not be used while using a skill.") nor while
// someone waits for p's answer to a request; else the book.
func (s *Server) handleRecipeBookOpen(p *Player, body []byte) {
	req, err := clientpackets.ParseRequestRecipeBookOpen(body)
	if err != nil {
		s.log.Debug("game: bad RequestRecipeBookOpen", append(p.logArgs(), "err", err)...)
		return
	}
	if s.CastingNow(p) {
		p.SendSystemMessage(craftMsgBookWhileCasting)
		return
	}
	if s.activeRequester(p) != nil {
		p.SendMessage("You may not alter your recipe book while trading.")
		return
	}
	s.openRecipeBook(p, req.Dwarven)
}

// OpenRecipeBook is the effect of the Dwarven Craft (1321) and Common Craft
// (1322) skills (Mobius OpenDwarfRecipeBook, OpenCommonRecipeBook): not in
// store mode ("An item may not be created while engaged in trading."), else
// the book. internal/skills/effects finds it on the skill adapter.
func (a *skillPlayer) OpenRecipeBook(dwarven bool) {
	p, s := a.p, a.srv
	if p == nil || s == nil {
		return
	}
	if p.InStoreMode() {
		p.SendSystemMessage(craftMsgNotWhileTrading)
		return
	}
	s.openRecipeBook(p, dwarven)
}

// handleRecipeBookDestroy handles RequestRecipeBookDestroy (0xAD): within
// the transaction flood limit, a recipe of p's books leaves them (with its
// shortcuts and its workshop entry), and p gets the book again.
func (s *Server) handleRecipeBookDestroy(p *Player, body []byte) {
	req, err := clientpackets.ParseRecipeID(body)
	if err != nil {
		s.log.Debug("game: bad RequestRecipeBookDestroy", append(p.logArgs(), "err", err)...)
		return
	}
	if s.flood(&p.items.nextTransaction, time.Duration(s.gs.Economy.FloodTransaction)) {
		return
	}
	r, ok := s.Recipe(int(req.ID))
	if !ok || !p.HasRecipe(req.ID) {
		return
	}
	s.unregisterRecipe(p, req.ID)
	p.craft.shop = craft.RemoveShopItem(p.craft.shop, req.ID)
	p.Send(recipeBookPacket(p, r.Dwarven))
}

// recipeMakeInfoPacket is RecipeItemMakeInfo of a recipe for p.
func recipeMakeInfoPacket(p *Player, r *data.Recipe, success bool) []byte {
	return serverpackets.NewRecipeItemMakeInfoPacket(serverpackets.RecipeItemMakeInfo{
		RecipeID: int32(r.ID), //nolint:gosec // recipe IDs are small
		Dwarven:  r.Dwarven,
		CurMP:    javaInt32(p.CurMP),
		MaxMP:    int32(p.stats.MaxMP()), //nolint:gosec // MP fits
		Success:  success,
	})
}

// handleRecipeItemMakeInfo handles RequestRecipeItemMakeInfo (0xAE): the
// craft window of a known recipe ID, whatever the books hold (Mobius checks
// nothing more).
func (s *Server) handleRecipeItemMakeInfo(p *Player, body []byte) {
	req, err := clientpackets.ParseRecipeID(body)
	if err != nil {
		s.log.Debug("game: bad RequestRecipeItemMakeInfo", append(p.logArgs(), "err", err)...)
		return
	}
	if r, ok := s.Recipe(int(req.ID)); ok {
		p.Send(recipeMakeInfoPacket(p, r, true))
	}
}

// handleRecipeItemMakeSelf handles RequestRecipeItemMakeSelf (0xAF): within
// the manufacture flood limit and outside store mode ("You cannot create
// items while trading."), p crafts from its own book (craft_make.go).
func (s *Server) handleRecipeItemMakeSelf(p *Player, body []byte) {
	req, err := clientpackets.ParseRecipeID(body)
	if err != nil {
		s.log.Debug("game: bad RequestRecipeItemMakeSelf", append(p.logArgs(), "err", err)...)
		return
	}
	if s.flood(&p.craft.nextManufacture, time.Duration(s.gs.Crafting.FloodManufacture)) {
		return
	}
	if p.InStoreMode() {
		p.SendMessage("You cannot create items while trading.")
		return
	}
	s.requestMakeItem(p, req.ID)
}
