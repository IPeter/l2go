package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/craft"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/testclient"
)

// workshopList sends RequestRecipeShopListSet with recipe, cost pairs.
func (e *itemEnv) workshopList(sess *fakeSession, pairs ...int32) []testclient.Packet {
	body := append([]int32{int32(len(pairs) / 2)}, pairs...)
	return e.handle(sess, clientpackets.OpRequestRecipeShopListSet, ints(body...))
}

// workshopOrder sends RequestRecipeShopMakeItem for a recipe of crafter's
// workshop, after the flood limit passed.
func (e *itemEnv) workshopOrder(sess *fakeSession, crafter *Player, recipeID int32) []testclient.Packet {
	e.advance(time.Second)
	sess.take(e.t)
	return e.handle(sess, clientpackets.OpRequestRecipeShopMakeItem, ints(crafter.ObjectID(), recipeID, 0))
}

// workshopCrafter enters a crafter with Create Item 1 and the Wooden Arrow
// recipe, and opens a workshop that offers it for fee.
func (e *itemEnv) workshopCrafter(t *testing.T, fee int32) (*fakeSession, *Player) {
	t.Helper()
	sa, a := e.craftEnter("crafter", storeAt(0, 0), itemAdena, 100)
	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	e.advance(time.Second)
	sa.take(t)
	e.workshopList(sa, recipeWoodenArrow, fee)
	if got := e.storeType(a); got != StoreManufacture {
		t.Fatalf("store type %d, want the workshop", got)
	}
	return sa, a
}

// Demo 3: a crafter opens a workshop (action 37), a customer sees it, orders
// Wooden Arrows and pays the fee; both are saved together.
func TestWorkshopOrder(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0), itemAdena, 100)
	sb, b := e.craftEnter("customer", storeAt(40, 0), itemAdena, 5000, itemStem, 4, itemIronOre, 2)
	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	e.advance(time.Second)
	sa.take(t)
	sb.take(t)

	// The setup window: the dwarven book and an empty list.
	ps := e.storeAction(sa, clientpackets.ActionUseDwarvenManufacture)
	m := shopOne[testclient.RecipeShopManageList](t, "setup window", ps)
	if m.SellerID != a.ObjectID() || !m.Dwarven || !slices.Equal(m.RecipeIDs, []int32{recipeWoodenArrow}) || len(m.Items) != 0 || m.Adena != 100 {
		t.Fatalf("setup window = %+v", m)
	}
	if got := e.storeType(a); got != StoreNone {
		t.Fatalf("store type after the setup window = %d, Mobius sets none", got)
	}
	if ps := e.handle(sa, clientpackets.OpRequestRecipeShopMessageSet, stringBody("Arrows")); len(ps) != 0 {
		t.Errorf("the message sent %q, Mobius sends nothing", testclient.Names(ps))
	}

	// Opening: sitting, the store type in UserInfo and CharInfo, the message.
	ps = e.workshopList(sa, recipeWoodenArrow, 1000)
	checkNames(t, "the crafter opens the workshop", ps, "ChangeWaitType", "UserInfo", "RecipeShopMsg")
	if got := storeUserInfoType(t, ps); got != uint8(StoreManufacture) {
		t.Fatalf("UserInfo store type = %d, want %d", got, StoreManufacture)
	}
	if msg := shopOne[testclient.RecipeShopMsg](t, "message", ps); msg.ObjectID != a.ObjectID() || msg.Message != "Arrows" {
		t.Errorf("message = %+v", msg)
	}
	pb := sb.take(t)
	checkNames(t, "the customer sees it", pb, "ChangeWaitType", "CharInfo", "RecipeShopMsg")
	if ci := shopOne[testclient.CharInfoPacket](t, "CharInfo", pb); ci.PrivateStoreType != uint8(StoreManufacture) {
		t.Errorf("CharInfo store type = %d", ci.PrivateStoreType)
	}

	// The customer's second click opens the workshop.
	e.storeClick(sb, a)
	ps = e.storeClick(sb, a)
	l := shopOne[testclient.RecipeShopSellList](t, "workshop window", ps)
	if l.ManufacturerID != a.ObjectID() || l.Adena != 5000 || l.MaxMP <= 0 || l.CurMP != l.MaxMP {
		t.Errorf("workshop window = %+v", l)
	}
	if it, ok := l.Find(recipeWoodenArrow); !ok || it.Price != 1000 {
		t.Errorf("workshop entry = %+v, %v", it, ok)
	}
	ps = e.handle(sb, clientpackets.OpRequestRecipeShopMakeInfo, ints(a.ObjectID(), recipeWoodenArrow))
	if info := shopOne[testclient.RecipeShopItemInfo](t, "item info", ps); info.ManufacturerID != a.ObjectID() || info.RecipeID != recipeWoodenArrow {
		t.Errorf("item info = %+v", info)
	}
	ps = e.handle(sb, clientpackets.OpRequestRecipeShopManagePrev, nil)
	shopOne[testclient.RecipeShopSellList](t, "the window again", ps)

	// The order.
	mp := e.curMP(a)
	sa.take(t)
	ps = e.workshopOrder(sb, a, recipeWoodenArrow)
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(sysMsgS2S1Disappeared), int32(sysMsgS2S1Disappeared), int32(craftMsgS1CreatedCount), int32(sysMsgEarnedS2S1)}) {
		t.Errorf("customer messages %v", ids)
	}
	created := only[testclient.SystemMessage](ps)[2]
	if texts := created.Texts(); len(texts) != 1 || texts[0] != "crafter" || created.Params[1].Number != 500 || created.Params[3].Number != 1000 {
		t.Errorf("customer's created message = %+v", created)
	}
	infos := only[testclient.RecipeShopItemInfo](ps)
	if len(infos) != 3 {
		t.Errorf("%d RecipeShopItemInfo, want 3 (constructor, reward, finish)", len(infos))
	}
	pa := sa.take(t)
	if ids := msgIDs(pa); !slices.Equal(ids, []int32{int32(craftMsgCreatedCountForS1)}) {
		t.Errorf("crafter messages %v", ids)
	}
	if e.countOf(b, itemWoodenArrow) != 500 || e.countOf(b, itemStem) != 0 || e.countOf(b, itemIronOre) != 0 {
		t.Error("the customer did not get the arrows for his materials")
	}
	if e.countOf(b, itemAdena) != 4000 || e.countOf(a, itemAdena) != 1100 {
		t.Errorf("adena: customer %d, crafter %d; want 4000, 1100", e.countOf(b, itemAdena), e.countOf(a, itemAdena))
	}
	if got := e.curMP(a); got != mp-30 {
		t.Errorf("crafter MP %v, want %v", got, mp-30)
	}
	if e.storeSaved(b, itemWoodenArrow) != 500 || e.storeSaved(a, itemAdena) != 1100 || e.storeSaved(b, itemAdena) != 4000 {
		t.Error("the order was not saved")
	}
	if got := e.storeType(a); got != StoreManufacture {
		t.Errorf("store type after an order = %d, the workshop stays open", got)
	}

	// Quitting closes the workshop.
	e.handle(sa, clientpackets.OpRequestRecipeShopManageQuit, nil)
	e.advance(time.Second) // UserInfo goes out at the end of the tick
	ps = sa.take(t)
	if got := storeUserInfoType(t, ps); got != uint8(StoreNone) || e.storeType(a) != StoreNone {
		t.Errorf("after quitting: UserInfo type %d", got)
	}
	if ps := e.workshopOrder(sb, a, recipeWoodenArrow); len(ps) != 0 {
		t.Errorf("an order at a closed workshop sent %q", testclient.Names(ps))
	}
}

// A failed workshop craft keeps the fee and uses up the materials; a
// customer whose materials fall short after paying gets the fee back; one
// without the fee is told so; one out of reach or naming a recipe the
// workshop does not offer gets nothing.
func TestWorkshopFees(t *testing.T) {
	e := newCraftEnv(t)
	arrow := e.craftRecipe(recipeWoodenArrow)
	arrow.SuccessRate = 50
	e.withRecipes(arrow, e.craftRecipe(2))
	sa, a := e.workshopCrafter(t, 1000)
	sb, b := e.craftEnter("customer", storeAt(40, 0), itemAdena, 5000, itemStem, 4, itemIronOre, 2)
	sa.take(t)

	// Failure: the fee stays with the crafter.
	e.advance(time.Second)
	sb.take(t)
	ps := e.handleWithRand(sb, rnd.NewScripted(t, rnd.Get(100, 50)), func() { e.srv.requestManufactureItem(a, recipeWoodenArrow, b) })
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(sysMsgS2S1Disappeared), int32(sysMsgS2S1Disappeared), int32(craftMsgS1FailedToCreate)}) {
		t.Errorf("failure: customer messages %v", ids)
	}
	if ids := msgIDs(sa.take(t)); !slices.Equal(ids, []int32{int32(craftMsgAttemptForS1Failed)}) {
		t.Errorf("failure: crafter messages %v", ids)
	}
	if e.countOf(b, itemAdena) != 4000 || e.countOf(a, itemAdena) != 1100 || e.countOf(b, itemStem) != 0 || e.countOf(b, itemWoodenArrow) != 0 {
		t.Error("a failed order must keep the fee and the materials")
	}
	if e.storeSaved(a, itemAdena) != 1100 || e.storeSaved(b, itemAdena) != 4000 {
		t.Error("the failed order was not saved")
	}

	// Missing materials before paying: nothing moves.
	ps = e.workshopOrder(sb, a, recipeWoodenArrow)
	if m := shopOne[testclient.SystemMessage](t, "missing", ps); m.ID != int32(craftMsgMissing) {
		t.Errorf("missing: %+v", m)
	}
	if e.countOf(b, itemAdena) != 4000 {
		t.Error("a customer without materials paid")
	}

	// Not enough adena for the fee.
	e.give(b, itemStem, 4)
	e.give(b, itemIronOre, 2)
	e.onLoop(func() {
		if _, err := b.inv.DestroyByItemID(itemAdena, 3500); err != nil {
			t.Fatal(err)
		}
		e.srv.ItemsChanged(b)
	})
	ps = e.workshopOrder(sb, a, recipeWoodenArrow)
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(sysMsgNotEnoughAdena)}) {
		t.Errorf("no fee: messages %v", ids)
	}

	// Out of reach (150 plus the collision radii).
	sc, c := e.craftEnter("faraway", storeAt(400, 0), itemAdena, 5000, itemStem, 4, itemIronOre, 2)
	if ps := e.workshopOrder(sc, a, recipeWoodenArrow); len(only[testclient.RecipeShopItemInfo](ps)) != 0 || e.countOf(c, itemStem) != 4 {
		t.Errorf("an order from afar: %q", testclient.Names(ps))
	}

	// A recipe the crafter does not know is the customer's illegal action.
	if ps := e.workshopOrder(sb, a, 2); !storeKicked(ps) {
		t.Errorf("an order of a recipe the crafter does not know: %q", testclient.Names(ps))
	}
}

// The fee goes back when the customer's materials fall short after paying:
// a recipe that takes adena (none of Mobius's does, so the test makes one).
func TestWorkshopRefund(t *testing.T) {
	e := newCraftEnv(t)
	arrow := e.craftRecipe(recipeWoodenArrow)
	arrow.Ingredients = []data.RecipeItem{{ItemID: itemStem, Count: 4}, {ItemID: itemAdena, Count: 1000}}
	e.withRecipes(arrow)
	sa, a := e.workshopCrafter(t, 1000)
	sb, b := e.craftEnter("customer", storeAt(40, 0), itemAdena, 1500, itemStem, 4)
	sa.take(t)

	ps := e.workshopOrder(sb, a, recipeWoodenArrow)
	m := shopOne[testclient.SystemMessage](t, "missing after paying", ps)
	if m.ID != int32(craftMsgMissing) || m.Params[0].Number != itemAdena || m.Params[1].Number != 500 {
		t.Errorf("missing message = %+v, want 500 adena", m)
	}
	if e.countOf(b, itemAdena) != 1500 || e.countOf(a, itemAdena) != 100 || e.countOf(b, itemStem) != 4 {
		t.Errorf("after the refund: customer %d adena, crafter %d; want 1500, 100 and the stems kept",
			e.countOf(b, itemAdena), e.countOf(a, itemAdena))
	}
	if e.storeSaved(b, itemAdena) != 1500 || e.storeSaved(a, itemAdena) != 100 {
		t.Error("the refund was not saved")
	}
}

// RequestRecipeShopListSet refusals: a refused list closes the store, a
// recipe not in the book or a fee over MaxAdena is punished, an unknown
// recipe drops the packet; the setup window drops the other book's entries.
func TestWorkshopListSet(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0))
	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	e.advance(time.Second)
	sa.take(t)

	// A negative cost: the list is refused and the store closes.
	e.onLoop(func() { a.store.typ = StoreSellManage })
	e.handle(sa, clientpackets.OpRequestRecipeShopListSet, ints(1, recipeWoodenArrow, -1))
	e.advance(time.Second) // UserInfo goes out at the end of the tick
	ps := sa.take(t)
	if got := storeUserInfoType(t, ps); got != uint8(StoreNone) {
		t.Errorf("refused list: UserInfo type %d", got)
	}
	// An unknown recipe drops the packet.
	if ps := e.workshopList(sa, 999999, 5); len(ps) != 0 || e.storeType(a) != StoreNone {
		t.Errorf("unknown recipe: %q", testclient.Names(ps))
	}

	// The setup window drops entries of the other book and forgotten ones.
	const common = 680 // mk_fish_oil_average, a common recipe
	e.onLoop(func() {
		a.setRecipeBook(false, []int32{common})
		a.craft.shop = []craft.ShopItem{{RecipeID: recipeWoodenArrow, Price: 5}, {RecipeID: common, Price: 7}, {RecipeID: 3, Price: 9}}
	})
	ps = e.storeAction(sa, clientpackets.ActionUseCommonManufacture)
	m := shopOne[testclient.RecipeShopManageList](t, "common setup window", ps)
	if m.Dwarven || !slices.Equal(m.RecipeIDs, []int32{common}) || !slices.Equal(m.Items, []testclient.RecipeShopItem{{RecipeID: common, Price: 7}}) {
		t.Errorf("common setup window = %+v", m)
	}

	// A negative fee second: refused, not punished. A fee over MaxAdena.
	if ps := e.workshopList(sa, recipeWoodenArrow, 5, common, -2); e.storeType(a) != StoreNone || storeKicked(ps) {
		t.Errorf("a negative fee second: %q", testclient.Names(ps))
	}
	e.onLoop(func() { e.srv.items.cfg.MaxAdena = 100 })
	if ps := e.workshopList(sa, recipeWoodenArrow, 101); !storeKicked(ps) || e.storeType(a) != StoreNone {
		t.Errorf("a fee over MaxAdena: %q", testclient.Names(ps))
	}
}

// A recipe the crafter does not have in the list is an illegal action.
func TestWorkshopListNotInBook(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0))
	if ps := e.workshopList(sa, recipeWoodenArrow, 5); !storeKicked(ps) || e.storeType(a) != StoreNone {
		t.Errorf("a recipe not in the book: %q", testclient.Names(ps))
	}
}
