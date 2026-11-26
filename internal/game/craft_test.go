package game

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/craft"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The items and recipes of the crafting tests (Mobius Recipes.xml recipe 1).
const (
	itemRecipeWoodenArrow = 1666 // Recipe: Wooden Arrow
	itemIronOre           = 1869
	recipeWoodenArrow     = 1
)

// newCraftEnv starts a server with the skill data and the recipes of data/.
func newCraftEnv(t *testing.T) *itemEnv {
	t.Helper()
	e := &itemEnv{testEnv: newItemSkillEnv(t, testWorldData(t), nil)}
	var ok bool
	e.onLoop(func() { _, ok = e.srv.Recipe(recipeWoodenArrow) })
	if !ok {
		t.Fatal("the recipes of data/ are not loaded")
	}
	return e
}

// craftEnter enters a level 20 Human Fighter with full MP and extra items
// (item ID, count pairs) at pos.
func (e *itemEnv) craftEnter(account string, pos world.Position, pairs ...int64) (*fakeSession, *Player) {
	e.t.Helper()
	items := storeItems(pairs...)
	for i := range items {
		items[i].Slot, items[i].Mana = storage.PaperdollNone, storage.NoMana
	}
	sess, p := e.enterChar(testChar{account: account, name: account, pos: pos, level: 20, items: items})
	e.onLoop(func() { p.SetCurrentMP(float64(p.stats.MaxMP())) })
	sess.take(e.t)
	return sess, p
}

// giveSkill gives p a skill level, not saved (//setskill).
func (e *itemEnv) giveSkill(p *Player, id, level int) {
	e.t.Helper()
	e.onLoop(func() {
		sk, ok := e.srv.Skill(id, level)
		if !ok {
			e.t.Fatalf("no skill %d level %d", id, level)
		}
		e.srv.AddSkill(p, sk, false)
	})
}

// book returns a copy of p's book, read on the loop.
func (e *itemEnv) book(p *Player, dwarven bool) []int32 {
	var b []int32
	e.onLoop(func() { b = slices.Clone(p.RecipeBook(dwarven)) })
	return b
}

// curMP returns p's MP, read on the loop.
func (e *itemEnv) curMP(p *Player) float64 {
	var mp float64
	e.onLoop(func() { mp = p.CurMP })
	return mp
}

// useRecipeItem gives p one recipe item and uses it.
func (e *itemEnv) useRecipeItem(sess *fakeSession, p *Player, itemID int32) []testclient.Packet {
	e.t.Helper()
	it := e.give(p, itemID, 1)
	e.advance(time.Second) // the InventoryUpdate of the new item goes out
	sess.take(e.t)
	return e.useItem(sess, it.ObjectID())
}

// withRecipes replaces the recipe data on the loop.
func (e *itemEnv) withRecipes(list ...data.Recipe) {
	e.t.Helper()
	r, err := data.NewRecipes(list)
	if err != nil {
		e.t.Fatal(err)
	}
	e.onLoop(func() { e.srv.UseRecipes(r) })
}

// craftRecipe returns a copy of a recipe of the data.
func (e *itemEnv) craftRecipe(id int) data.Recipe {
	e.t.Helper()
	var r *data.Recipe
	e.onLoop(func() { r, _ = e.srv.Recipe(id) })
	if r == nil {
		e.t.Fatalf("no recipe %d", id)
	}
	return *r
}

// handleWithRand runs a handler on the loop with src as the random source,
// and returns what sess got.
func (e *itemEnv) handleWithRand(sess *fakeSession, src rnd.Source, fn func()) []testclient.Packet {
	e.t.Helper()
	e.withRand(src, fn)
	e.sync()
	return sess.take(e.t)
}

// savedRecipes returns the stored recipe books of p once its saves are
// written.
func (e *itemEnv) savedRecipes(p *Player) storage.RecipeBook {
	e.t.Helper()
	e.advance(time.Second) // the saves asked for go to the saver at the end of the tick
	if err := e.srv.saver.wait(e.ctx, p.ObjectID()); err != nil {
		e.t.Fatalf("wait for the saves of %s: %v", p.Name(), err)
	}
	c, err := e.store.Get(e.ctx, p.ObjectID())
	if err != nil {
		e.t.Fatal(err)
	}
	return c.Recipes
}

// msgIDs returns the IDs of the system messages in ps.
func msgIDs(ps []testclient.Packet) []int32 { return sysMsgIDsOf(ps) }

// hasText reports whether a system message of ps carries the text.
func hasText(ps []testclient.Packet, text string) bool {
	for _, m := range only[testclient.SystemMessage](ps) {
		for _, s := range m.Texts() {
			if strings.Contains(s, text) {
				return true
			}
		}
	}
	return false
}

// Demo 1: a player with Create Item uses "Recipe: Wooden Arrow" (1666): it
// is added to the dwarven book, the item is used up and the book is saved.
// The refusals come in Mobius's order.
func TestCraftRegisterRecipe(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0))

	// Without Create Item.
	ps := e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(craftMsgNoAbility)}) {
		t.Fatalf("without the skill: messages %v", ids)
	}
	if n := e.countOf(a, itemRecipeWoodenArrow); n != 1 {
		t.Fatalf("the refused recipe item is gone: %d left", n)
	}

	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	ps = e.useItem(sa, e.itemOf(a, itemRecipeWoodenArrow).ObjectID())
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(craftMsgS1Added)}) {
		t.Fatalf("register: messages %v", ids)
	}
	if m := shopOne[testclient.SystemMessage](t, "added", ps); len(m.Params) != 1 || m.Params[0].Number != itemRecipeWoodenArrow {
		t.Errorf("added message = %+v, want the recipe item's name", m)
	}
	checkNames(t, "register", ps, "StatusUpdate", "SystemMessage")
	if n := e.countOf(a, itemRecipeWoodenArrow); n != 0 {
		t.Errorf("%d recipe items left, want it used up", n)
	}
	if b := e.book(a, true); !slices.Equal(b, []int32{recipeWoodenArrow}) {
		t.Errorf("dwarven book = %v, want [1]", b)
	}
	if b := e.book(a, false); len(b) != 0 {
		t.Errorf("common book = %v, want none", b)
	}
	saved := e.savedRecipes(a)
	if !slices.Equal(saved.Dwarven[0], []int32{recipeWoodenArrow}) || len(saved.Common) != 0 {
		t.Errorf("saved books = %+v", saved)
	}

	// Registered already.
	ps = e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(craftMsgAlreadyRegistered)}) {
		t.Fatalf("again: messages %v", ids)
	}

	// The book opens with it (RequestRecipeBookOpen, 0 is dwarven).
	ps = e.handle(sa, clientpackets.OpRequestRecipeBookOpen, ints(0))
	book := shopOne[testclient.RecipeBookItemList](t, "dwarven book", ps)
	if !book.Dwarven || !slices.Equal(book.RecipeIDs, []int32{recipeWoodenArrow}) || book.MaxMP <= 0 {
		t.Errorf("dwarven book = %+v", book)
	}
	ps = e.handle(sa, clientpackets.OpRequestRecipeBookOpen, ints(1))
	if book := shopOne[testclient.RecipeBookItemList](t, "common book", ps); book.Dwarven || len(book.RecipeIDs) != 0 {
		t.Errorf("common book = %+v", book)
	}
}

// A recipe above the skill level, and a full book (DwarfRecipeLimit plus
// the stat), are refused with their messages.
func TestCraftRegisterLimits(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0))
	e.giveSkill(a, craft.SkillCreateDwarven, 1)

	arrow := e.craftRecipe(recipeWoodenArrow)
	level2 := arrow
	level2.CraftLevel = 2
	e.withRecipes(level2)
	ps := e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(craftMsgCreateLevelTooLow)}) {
		t.Fatalf("level too low: messages %v", ids)
	}

	e.withRecipes(arrow)
	e.onLoop(func() {
		e.srv.gs.Crafting.DwarfRecipeLimit = 1
		a.setRecipeBook(true, []int32{77})
	})
	ps = e.handle(sa, clientpackets.OpUseItem, ints(e.itemOf(a, itemRecipeWoodenArrow).ObjectID(), 0))
	m := shopOne[testclient.SystemMessage](t, "book full", ps)
	if m.ID != int32(craftMsgUpToS1Recipes) || len(m.Params) != 1 || m.Params[0].Number != 1 {
		t.Fatalf("book full: %+v", m)
	}

	// Crafting turned off: the text, and the item stays.
	e.onLoop(func() {
		e.srv.gs.Crafting.DwarfRecipeLimit = 50
		e.srv.gs.Crafting.Enabled = false
	})
	ps = e.handle(sa, clientpackets.OpUseItem, ints(e.itemOf(a, itemRecipeWoodenArrow).ObjectID(), 0))
	if !hasText(ps, "Crafting is disabled") || e.countOf(a, itemRecipeWoodenArrow) != 1 {
		t.Errorf("disabled: %q", testclient.Names(ps))
	}
}

// The dwarven book belongs to the active class index, the common book is
// shared (decision X3); both are saved by class index.
func TestCraftBooksByClassIndex(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0))
	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	e.onLoop(func() { a.setRecipeBook(false, []int32{851}) })

	e.onLoop(func() { a.char.ClassIndex = 2 })
	if b := e.book(a, true); len(b) != 0 {
		t.Errorf("class index 2: dwarven book = %v, want empty", b)
	}
	if b := e.book(a, false); !slices.Equal(b, []int32{851}) {
		t.Errorf("class index 2: common book = %v, want the shared one", b)
	}
	e.onLoop(func() { a.setRecipeBook(true, []int32{5}) })
	e.onLoop(func() { a.char.ClassIndex = 0 })
	if b := e.book(a, true); !slices.Equal(b, []int32{1}) {
		t.Errorf("back on the base class: dwarven book = %v", b)
	}
	e.onLoop(func() { e.srv.RequestSave(a) })
	saved := e.savedRecipes(a)
	if !slices.Equal(saved.Dwarven[0], []int32{1}) || !slices.Equal(saved.Dwarven[2], []int32{5}) || !slices.Equal(saved.Common, []int32{851}) {
		t.Errorf("saved books = %+v", saved)
	}
}

// RequestRecipeBookDestroy takes a recipe out of the book with its
// shortcuts and its workshop entry, and sends the book again.
func TestCraftDestroyRecipe(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0))
	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	e.onLoop(func() {
		e.srv.registerShortcut(a, shortcut{slot: 2, page: 0, typ: storage.ShortcutRecipe, id: recipeWoodenArrow, level: -1})
		a.craft.shop = []craft.ShopItem{{RecipeID: recipeWoodenArrow, Price: 10}}
	})
	e.advance(time.Second)
	sa.take(t)

	ps := e.handle(sa, clientpackets.OpRequestRecipeBookDestroy, ints(recipeWoodenArrow))
	book := shopOne[testclient.RecipeBookItemList](t, "book", ps)
	if !book.Dwarven || len(book.RecipeIDs) != 0 {
		t.Errorf("book after the delete = %+v", book)
	}
	if bar := shopOne[testclient.ShortcutInit](t, "shortcuts", ps); len(bar.Shortcuts) != 0 {
		t.Errorf("shortcuts left: %+v", bar.Shortcuts)
	}
	var shop []craft.ShopItem
	e.onLoop(func() { shop = a.craft.shop })
	if len(shop) != 0 {
		t.Errorf("workshop list = %v, want the recipe gone", shop)
	}
	if saved := e.savedRecipes(a); !saved.Empty() {
		t.Errorf("saved books = %+v, want empty", saved)
	}
	// A recipe not in the book: nothing (the flood limit passed).
	e.advance(time.Second)
	if ps := e.handle(sa, clientpackets.OpRequestRecipeBookDestroy, ints(recipeWoodenArrow)); len(only[testclient.RecipeBookItemList](ps)) != 0 {
		t.Errorf("deleting a recipe again sent %q", testclient.Names(ps))
	}
}

// Demo 2: crafting Wooden Arrows takes 4 Stems, 2 Iron Ore and 30 MP and
// gives 500 Wooden Arrows, with Mobius's packets in order.
func TestCraftSelf(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0), itemStem, 10, itemIronOre, 2)
	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	e.advance(time.Second)
	sa.take(t)
	mp := e.curMP(a)

	ps := e.handle(sa, clientpackets.OpRequestRecipeItemMakeSelf, ints(recipeWoodenArrow))
	var got []string
	for _, p := range ps {
		switch p.(type) {
		case testclient.RecipeItemMakeInfo, testclient.SystemMessage, testclient.InventoryUpdate, testclient.ItemList:
			got = append(got, p.PacketName())
		}
	}
	want := []string{
		"RecipeItemMakeInfo",             // the constructor's
		"SystemMessage", "SystemMessage", // the stems and the ore disappeared
		"SystemMessage",      // you have earned
		"RecipeItemMakeInfo", // rewardPlayer's
		"InventoryUpdate", "ItemList",
		"RecipeItemMakeInfo", // finishCrafting's
	}
	if !slices.Equal(got, want) {
		t.Fatalf("craft packets %q, want %q", got, want)
	}
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(sysMsgS2S1Disappeared), int32(sysMsgS2S1Disappeared), int32(sysMsgEarnedS2S1)}) {
		t.Errorf("messages %v", ids)
	}
	for _, info := range only[testclient.RecipeItemMakeInfo](ps) {
		if info.RecipeID != recipeWoodenArrow || !info.Dwarven || !info.Success {
			t.Errorf("RecipeItemMakeInfo = %+v", info)
		}
	}
	if n := e.countOf(a, itemWoodenArrow); n != 500 {
		t.Errorf("%d Wooden Arrows, want 500", n)
	}
	if s, o := e.countOf(a, itemStem), e.countOf(a, itemIronOre); s != 6 || o != 0 {
		t.Errorf("%d stems and %d ore left, want 6 and 0", s, o)
	}
	if got := e.curMP(a); got != mp-30 {
		t.Errorf("MP %v, want %v", got, mp-30)
	}
	e.advance(time.Second) // the save goes to the saver at the end of the tick
	if n := e.storeSaved(a, itemWoodenArrow); n != 500 {
		t.Errorf("saved %d Wooden Arrows, want 500", n)
	}
}

// The checks before a craft: the materials (the first one missing, two
// failed windows, nothing taken), MP, the book, combat stance, store mode
// and the flood limit.
func TestCraftSelfRefusals(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0), itemStem, 3, itemIronOre, 2)
	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	e.advance(time.Second)
	sa.take(t)

	ps := e.handle(sa, clientpackets.OpRequestRecipeItemMakeSelf, ints(recipeWoodenArrow))
	m := shopOne[testclient.SystemMessage](t, "missing", ps)
	if m.ID != int32(craftMsgMissing) || len(m.Params) != 2 || m.Params[0].Number != itemStem || m.Params[1].Number != 1 {
		t.Errorf("missing message = %+v, want 1 stem", m)
	}
	infos := only[testclient.RecipeItemMakeInfo](ps)
	if len(infos) != 2 || infos[0].Success || infos[1].Success {
		t.Errorf("windows %+v, want two failures", infos)
	}
	if e.countOf(a, itemStem) != 3 || e.countOf(a, itemIronOre) != 2 {
		t.Error("materials were taken")
	}

	// Not enough MP: set on the loop right before the request, so that no
	// regeneration comes between.
	e.give(a, itemStem, 1)
	e.advance(time.Second)
	sa.take(t)
	var mp float64
	e.onLoop(func() {
		a.SetCurrentMP(29)
		e.srv.handleRecipeItemMakeSelf(a, ints(recipeWoodenArrow))
		mp = a.CurMP
	})
	ps = sa.take(t)
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(sysMsgNotEnoughMP)}) {
		t.Errorf("MP short: messages %v", ids)
	}
	if e.countOf(a, itemStem) != 4 || mp != 29 {
		t.Errorf("an MP-short craft took something: %d stems, %v MP", e.countOf(a, itemStem), mp)
	}

	// The flood limit: a second request within 300 ms does nothing.
	e.onLoop(func() { a.SetCurrentMP(500) })
	if ps := e.handle(sa, clientpackets.OpRequestRecipeItemMakeSelf, ints(recipeWoodenArrow)); len(only[testclient.RecipeItemMakeInfo](ps)) != 0 {
		t.Errorf("a flooding craft sent %q", testclient.Names(ps))
	}

	// In combat stance.
	e.advance(time.Second)
	e.onLoop(func() { e.srv.addStance(a) })
	ps = e.handle(sa, clientpackets.OpRequestRecipeItemMakeSelf, ints(recipeWoodenArrow))
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(storeMsgInCombat)}) {
		t.Errorf("in combat: messages %v", ids)
	}
	e.onLoop(func() { e.srv.dropStance(a) })

	// A recipe not in the book is an illegal action.
	e.advance(30 * time.Second)
	ps = e.handle(sa, clientpackets.OpRequestRecipeItemMakeSelf, ints(2))
	if !storeKicked(ps) {
		t.Errorf("a recipe not in the book: %q, want a kick", testclient.Names(ps))
	}
}

// A failed craft loses the materials and gives nothing (Rnd.get(100) <
// successRate).
func TestCraftSelfFails(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0), itemStem, 8, itemIronOre, 4)
	e.giveSkill(a, craft.SkillCreateDwarven, 1)
	r := e.craftRecipe(recipeWoodenArrow)
	r.SuccessRate = 70
	e.withRecipes(r)
	e.useRecipeItem(sa, a, itemRecipeWoodenArrow)
	e.advance(time.Second)
	sa.take(t)

	ps := e.handleWithRand(sa, rnd.NewScripted(t, rnd.Get(100, 70)), func() { e.srv.requestMakeItem(a, recipeWoodenArrow) })
	if ids := msgIDs(ps); !slices.Equal(ids, []int32{int32(sysMsgS2S1Disappeared), int32(sysMsgS2S1Disappeared), int32(craftMsgFailedMixing)}) {
		t.Errorf("failure messages %v", ids)
	}
	if infos := only[testclient.RecipeItemMakeInfo](ps); len(infos) != 2 || !infos[0].Success || infos[1].Success {
		t.Errorf("windows %+v, want the constructor's success and the failure", infos)
	}
	if e.countOf(a, itemWoodenArrow) != 0 || e.countOf(a, itemStem) != 4 || e.countOf(a, itemIronOre) != 2 {
		t.Error("a failed craft must take the materials and give nothing")
	}

	ps = e.handleWithRand(sa, rnd.NewScripted(t, rnd.Get(100, 69)), func() { e.srv.requestMakeItem(a, recipeWoodenArrow) })
	if e.countOf(a, itemWoodenArrow) != 500 {
		t.Errorf("a roll of 69 at 70 %% failed: %q", testclient.Names(ps))
	}
}

// The Dwarven Craft and Common Craft skills open the book, not in store
// mode.
func TestCraftOpenBookEffect(t *testing.T) {
	e := newCraftEnv(t)
	sa, a := e.craftEnter("crafter", storeAt(0, 0))
	e.onLoop(func() { a.skill.OpenRecipeBook(false) })
	if b := shopOne[testclient.RecipeBookItemList](t, "common book", sa.take(t)); b.Dwarven {
		t.Errorf("book = %+v, want the common one", b)
	}
	e.onLoop(func() {
		a.store.typ = StoreManufacture
		a.skill.OpenRecipeBook(true)
		a.store.typ = StoreNone
	})
	if ids := msgIDs(sa.take(t)); !slices.Equal(ids, []int32{int32(craftMsgNotWhileTrading)}) {
		t.Errorf("in store mode: messages %v", ids)
	}
}
