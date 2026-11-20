package game

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// TestMerchantPriceConfig: a merchant takes the config of the town zone in
// its 32,768-unit zone region, the lowest ID first, else the default.
func TestMerchantPriceConfig(t *testing.T) {
	e := newShopEnv(t, nil)
	for _, tc := range []struct {
		name string
		pos  world.Position
		want int
	}{
		{"Talking Island", tiVillage, 2},
		{"Gludio", world.Position{X: -14225, Y: 123540, Z: -3121}, 7},
		{"Giran", world.Position{X: 83400, Y: 147943, Z: -3404}, 0},
		{"Dion", world.Position{X: 15670, Y: 142983, Z: -2705}, 8},
		{"Rune", world.Position{X: 43799, Y: -47727, Z: -798}, 13},
		{"nowhere", world.Position{X: -100000, Y: -200000, Z: 0}, 18},
	} {
		var got *data.PriceConfig
		e.onLoop(func() { got = e.srv.priceConfigAt(tc.pos) })
		if got == nil || got.ID != tc.want {
			t.Errorf("%s: price config %+v, want %d", tc.name, got, tc.want)
		}
	}

	// Lector gets his at spawn; an NPC that is no merchant gets none.
	lector := e.spawnNpc(npcLector, eastOf(shopPos, 50))
	guard := e.spawnNpc(npcGuard, eastOf(shopPos, 80))
	e.onLoop(func() {
		if lector.priceConfig == nil || lector.priceConfig.ID != 2 || merchantTaxRate(lector) != 0.15 {
			t.Errorf("Lector's price config %+v", lector.priceConfig)
		}
		if guard.priceConfig != nil {
			t.Errorf("a guard has the price config %+v", guard.priceConfig)
		}
	})
}

func TestShapeIntersectsRect(t *testing.T) {
	square := data.Shape{Type: data.ShapeNPoly, Points: []data.Point{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 100, Y: 100}, {X: 0, Y: 100}}}
	diamond := data.Shape{Type: data.ShapeNPoly, Points: []data.Point{{X: 50, Y: -100}, {X: 200, Y: 50}, {X: 50, Y: 200}, {X: -100, Y: 50}}}
	cuboid := data.Shape{Type: data.ShapeCuboid, Points: []data.Point{{X: 0, Y: 0}, {X: 100, Y: 100}}}
	cylinder := data.Shape{Type: data.ShapeCylinder, Points: []data.Point{{X: 0, Y: 0}}, Radius: 50}
	for _, tc := range []struct {
		name           string
		sh             data.Shape
		x1, y1, x2, y2 int
		want           bool
	}{
		{"polygon inside the rect", square, -10, -10, 200, 200, true},
		{"rect inside the polygon", square, 10, 10, 20, 20, true},
		{"edges cross", square, 50, -50, 60, 500, true},
		{"apart", square, 200, 200, 300, 300, false},
		{"touching edge", square, 100, 0, 200, 100, false},
		{"rect in the diamond's box, outside it", diamond, -100, -100, -60, -60, false},
		{"rect in the diamond", diamond, 40, 40, 60, 60, true},
		{"cuboid overlap", cuboid, 99, 99, 200, 200, true},
		{"cuboid touching", cuboid, 100, 100, 200, 200, false},
		{"cylinder near", cylinder, 30, 30, 100, 100, true},
		{"cylinder corner far", cylinder, 40, 40, 100, 100, false},
	} {
		if got := shapeIntersectsRect(tc.sh, tc.x1, tc.y1, tc.x2, tc.y2); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestTaxedPrice(t *testing.T) {
	for _, tc := range []struct {
		price int64
		rate  float64
		want  int64
	}{
		{768, 0.15, 883}, // 883.2: cut per item, before the count
		{100, 0.15, 114}, // 114.99999999999999 in doubles, as in Java
		{100, 0.20, 120},
		{100, 0, 100},
		{0, 0.5, 0},
	} {
		if got := taxedPrice(tc.price, tc.rate); got != tc.want {
			t.Errorf("taxedPrice(%d, %v) = %d, want %d", tc.price, tc.rate, got, tc.want)
		}
	}
}

// TestShopBuyWindow: "Buy <list>" opens the list with the taxed prices and
// the adena; a list that does not name the merchant stays shut.
func TestShopBuyWindow(t *testing.T) {
	e := newShopEnv(t, nil)
	sess, p := e.enterChar(testChar{account: "buyer", name: "Buyer", pos: shopPos, adena: 5000})
	lector := e.shopLector(sess, p)

	ps := e.shopCmd(sess, lector, "Buy "+strconv.Itoa(lectorBuyListID))
	checkNames(t, "Buy", ps, "BuyList", "ActionFailed", "ActionFailed")
	bl := shopOne[testclient.BuyList](t, "Buy", ps)
	list, _ := testEconomy(t).BuyList(lectorBuyListID)
	if bl.ListID != lectorBuyListID || bl.Adena != 5000 || bl.Listed != len(list.Products) || len(bl.Items) != len(list.Products) {
		t.Fatalf("BuyList %d, adena %d, %d of %d entries; want %d entries", bl.ListID, bl.Adena, len(bl.Items), bl.Listed, len(list.Products))
	}
	for i, pr := range list.Products {
		want := int32(float64(pr.Price) * 1.15)
		if it := bl.Items[i]; it.ItemID != int32(pr.ItemID) || it.Price != want || it.Count != 0 {
			t.Errorf("entry %d: %+v, want item %d at %d", i, it, pr.ItemID, want)
		}
	}
	// Item 1 (Short Sword, 768 adena) costs 883 in Talking Island.
	if it, ok := bl.Find(1); !ok || it.Price != 883 {
		t.Errorf("Short Sword %+v, want 883 adena", it)
	}
	// The inventory list request right after is ignored.
	if ps := e.handle(sess, clientpackets.OpRequestItemList, nil); len(ps) != 0 {
		t.Errorf("RequestItemList after the shop opened: %q", testclient.Names(ps))
	}

	ps = e.shopCmd(sess, lector, "Buy "+strconv.Itoa(otherBuyListID))
	checkNames(t, "another merchant's list", ps, "ActionFailed", "ActionFailed")
}

// TestShopBuy buys from Lector: taxed prices, adena, items, the packets, and
// Mobius's refusals in order.
func TestShopBuy(t *testing.T) {
	e := newShopEnv(t, nil)
	sess, p := e.enterChar(testChar{account: "buyer", name: "Buyer", pos: shopPos, adena: 10000})
	e.shopLector(sess, p)

	ps := e.shopBuy(sess, testBuyListID, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 10}, clientpackets.BuyItem{ItemID: itemLongSword, Count: 1})
	checkNames(t, "buy", ps, "StatusUpdate", "StatusUpdate", "StatusUpdate", "SystemMessage")
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(shopMsgTransactionComplete)}) {
		t.Errorf("messages %v", ids)
	}
	orePrice, swordPrice := taxedPrice(shopOrePrice, 0.15), taxedPrice(shopSwordPrice, 0.15) // 114 and 575
	if got, want := e.shopAdena(p), 10000-10*orePrice-swordPrice; got != want {
		t.Errorf("adena %d, want %d", got, want)
	}
	if e.countOf(p, itemMithrilOre) != 10 || e.countOf(p, itemLongSword) != 1 {
		t.Errorf("bought %d ore and %d swords", e.countOf(p, itemMithrilOre), e.countOf(p, itemLongSword))
	}
	e.clock.Advance(itemListDelay)
	e.sync()
	ps = sess.take(t)
	checkNames(t, "after the buy", ps, "InventoryUpdate", "ItemList")
	if il := shopOne[testclient.ItemList](t, "after the buy", ps); !il.ShowWindow {
		t.Error("the ItemList does not open the window")
	}

	// Too little adena: nothing changes.
	adena := e.shopAdena(p)
	ps = e.shopBuy(sess, testBuyListID, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 10000})
	checkNames(t, "too poor", ps, "SystemMessage", "ActionFailed")
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgNotEnoughAdena)}) {
		t.Errorf("too poor: messages %v", ids)
	}
	if e.shopAdena(p) != adena || e.countOf(p, itemMithrilOre) != 10 {
		t.Error("a refused buy changed the inventory")
	}

	// A count above 10,000 refuses the request.
	ps = e.shopBuy(sess, testBuyListID, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 10001})
	if msgs := messages(ps); len(msgs) != 1 || msgs[0] != "You cannot buy more than 10.000 items." {
		t.Errorf("10001 ore: %q", msgs)
	}

	// Out of range of the merchant: ActionFailed.
	e.shopMove(p, eastOf(shopPos, 400))
	checkNames(t, "far away", e.shopBuy(sess, testBuyListID, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 1}), "ActionFailed")
	e.shopMove(p, shopPos)

	// A list that does not name Lector: ActionFailed.
	checkNames(t, "another merchant's list", e.shopBuy(sess, otherBuyListID, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 1}), "ActionFailed")

	// Two of an item that does not stack: Mobius's message, and a kick.
	ps = e.shopBuy(sess, testBuyListID, clientpackets.BuyItem{ItemID: itemLongSword, Count: 2})
	if ids := sysMsgIDsOf(ps); !slices.Contains(ids, int32(shopMsgQuantityExceeded)) {
		t.Errorf("two swords: messages %v", ids)
	}
	if msgs := messages(ps); len(msgs) != 1 || !strings.Contains(msgs[0], "kicked") {
		t.Errorf("two swords: %q", msgs)
	}
	e.advance(illegalActionKickDelay)
	if _, _, closed := sess.closedWith(t); !closed {
		t.Error("the cheater was not kicked")
	}
}

// TestShopBuyRules: the karma rule, a forged list, the flood limit and a GM
// without a merchant.
func TestShopBuyRules(t *testing.T) {
	e := newShopEnv(t, func(gs *config.GameServer) {
		gs.Economy.AltKarmaPlayerCanShop = false
		gs.Economy.FloodTransaction = config.Duration(time.Second)
	})
	sess, p := e.enterChar(testChar{account: "pk", name: "Pk", pos: shopPos, adena: 1000, karma: 10})
	e.shopLector(sess, p)
	checkNames(t, "karma", e.shopBuy(sess, testBuyListID, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 1}), "ActionFailed")

	// One transaction per second.
	ps := e.shopBuy(sess, testBuyListID, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 1})
	if msgs := messages(ps); len(msgs) != 1 || msgs[0] != "You are buying too fast." {
		t.Errorf("flood: %q", msgs)
	}

	// A GM buys anywhere, untaxed.
	gsess, gm := e.enterChar(testChar{account: "gm", name: "Gm", pos: tiStart, adena: 1000, access: storage.AccessLevelAdmin})
	ps = e.shopBuy(gsess, testBuyListID, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 2})
	if ids := sysMsgIDsOf(ps); !slices.Contains(ids, int32(shopMsgTransactionComplete)) {
		t.Fatalf("GM buy: %q", testclient.Names(ps))
	}
	if got := e.shopAdena(gm); got != 1000-2*shopOrePrice {
		t.Errorf("GM adena %d, want %d", got, 1000-2*shopOrePrice)
	}

	// A list that does not exist is a forged request.
	e.advance(time.Second)
	ps = e.shopBuy(gsess, 123456789, clientpackets.BuyItem{ItemID: itemMithrilOre, Count: 1})
	if msgs := messages(ps); len(msgs) != 1 || !strings.Contains(msgs[0], "kicked") {
		t.Errorf("forged list: %q", msgs)
	}
}

// TestShopStock: a limited product runs out, refuses more than is left,
// leaves the list when sold out, and is full again 60 minutes after its
// first sale; later sales do not move the timer.
func TestShopStock(t *testing.T) {
	e := newShopEnv(t, nil)
	sess, p := e.enterChar(testChar{account: "buyer", name: "Buyer", pos: shopPos, adena: 10000})
	lector := e.shopLector(sess, p)
	shot := func(n int32) clientpackets.BuyItem { return clientpackets.BuyItem{ItemID: shopLimitedItem, Count: n} }
	stock := func(what string) (int32, bool) {
		t.Helper()
		bl := shopOne[testclient.BuyList](t, what, e.shopCmd(sess, lector, "Buy "+strconv.Itoa(testBuyListID)))
		it, ok := bl.Find(shopLimitedItem)
		if len(bl.Items) != bl.Listed {
			t.Errorf("%s: %d entries, %d announced", what, len(bl.Items), bl.Listed)
		}
		return it.Count, ok
	}
	if n, ok := stock("full"); !ok || n != shopLimitedCount {
		t.Fatalf("full stock %d, %v", n, ok)
	}

	e.shopBuy(sess, testBuyListID, shot(2))
	if n, _ := stock("after 2"); n != 3 {
		t.Errorf("stock after buying 2: %d, want 3", n)
	}
	e.advance(30 * time.Minute)
	sess.take(t) // what came due meanwhile
	checkNames(t, "more than is left", e.shopBuy(sess, testBuyListID, shot(4)), "ActionFailed")
	e.shopBuy(sess, testBuyListID, shot(3))
	if _, ok := stock("sold out"); ok {
		t.Error("a sold-out product is listed")
	}
	if e.countOf(p, shopLimitedItem) != 5 {
		t.Errorf("bought %d shots, want 5", e.countOf(p, shopLimitedItem))
	}

	// The timer started with the first sale.
	e.advance(30*time.Minute - time.Second)
	if _, ok := stock("before the restock"); ok {
		t.Error("restocked too early")
	}
	e.advance(time.Second)
	if n, _ := stock("restocked"); n != shopLimitedCount {
		t.Errorf("restocked to %d, want %d", n, shopLimitedCount)
	}

	// One request naming the product twice: each line fits the stock, the
	// second runs out and is paid back.
	adena := e.shopAdena(p)
	ps := e.shopBuy(sess, testBuyListID, shot(3), shot(3))
	price := taxedPrice(shopLimitedPrice, 0.15)
	if got := e.shopAdena(p); got != adena-3*price {
		t.Errorf("adena %d after the double line, want %d", got, adena-3*price)
	}
	if ids := sysMsgIDsOf(ps); !slices.Contains(ids, int32(sysMsgEarnedS1Adena)) {
		t.Errorf("no refund message: %v", ids)
	}
	if _, ok := stock("over-sold"); ok {
		t.Error("an over-sold product is listed")
	}
}

// TestShopSell sells to Lector: the sell window, half the reference price,
// only what was removed is paid, the MaxAdena check and the merchant rule.
func TestShopSell(t *testing.T) {
	e := newShopEnv(t, nil)
	sess, p := e.enterWith("seller", shopPos, storage.Item{ItemID: itemMithrilOre, Count: 10}, storage.Item{ItemID: itemDarinsLetter, Count: 1})
	lector := e.shopLector(sess, p)
	ore := e.itemOf(p, itemMithrilOre)
	oreT, _ := e.data.ItemTemplate(itemMithrilOre)
	price := int64(oreT.Price / 2)

	ps := e.shopCmd(sess, lector, "Sell")
	checkNames(t, "Sell", ps, "SellList", "ActionFailed", "ActionFailed")
	sl := shopOne[testclient.SellList](t, "Sell", ps)
	if it, ok := sl.Find(ore.ObjectID()); !ok || it.Price != int32(price) || it.Count != 10 || it.ItemID != itemMithrilOre {
		t.Errorf("ore in the SellList: %+v, %v", it, ok)
	}
	sword := e.itemOf(p, itemSquireSword)
	if _, ok := sl.Find(sword.ObjectID()); ok {
		t.Error("the worn sword is offered")
	}
	if _, ok := sl.Find(e.itemOf(p, itemDarinsLetter).ObjectID()); ok {
		t.Error("a quest item is offered")
	}

	// Sell 4 ore, and 20 that are not there: only the 4 are paid.
	ps = e.shopSell(sess, 0, clientpackets.SellItem{ObjectID: ore.ObjectID(), ItemID: itemMithrilOre, Count: 4}, clientpackets.SellItem{ObjectID: ore.ObjectID(), ItemID: itemMithrilOre, Count: 20})
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(shopMsgTransactionComplete)}) {
		t.Errorf("sell: messages %v", ids)
	}
	if e.countOf(p, itemMithrilOre) != 6 || e.shopAdena(p) != 4*price {
		t.Errorf("after selling: %d ore, %d adena; want 6, %d", e.countOf(p, itemMithrilOre), e.shopAdena(p), 4*price)
	}
	e.clock.Advance(itemListDelay)
	e.sync()
	ps = sess.take(t)
	if il := shopOne[testclient.ItemList](t, "after the sale", ps); !il.ShowWindow {
		t.Error("the ItemList does not open the window")
	}
	for _, iu := range only[testclient.InventoryUpdate](ps) {
		for _, c := range iu.Items {
			if c.ObjectID == ore.ObjectID() {
				t.Errorf("the sale announced the ore in an InventoryUpdate: %+v", c)
			}
		}
	}

	// Near MaxAdena nothing is sold.
	e.onLoop(func() { e.srv.AddAdena(p, e.srv.items.cfg.MaxAdena-p.inv.Adena()-1, false) })
	sess.take(t)
	ps = e.shopSell(sess, 0, clientpackets.SellItem{ObjectID: ore.ObjectID(), ItemID: itemMithrilOre, Count: 6})
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgAdenaLimit)}) {
		t.Errorf("over MaxAdena: messages %v", ids)
	}
	if e.countOf(p, itemMithrilOre) != 6 {
		t.Error("ore was sold over the adena limit")
	}

	// Away from the last merchant: ActionFailed (decision I9).
	e.shopMove(p, eastOf(shopPos, 400))
	checkNames(t, "far away", e.shopSell(sess, 0, clientpackets.SellItem{ObjectID: ore.ObjectID(), ItemID: itemMithrilOre, Count: 1}), "ActionFailed")
}

// TestBuyShadowItem shows the shadow item page for the player's level.
func TestBuyShadowItem(t *testing.T) {
	e := newShopEnv(t, nil)
	for _, tc := range []struct {
		level int32
		want  string
	}{
		{20, "Too young"},
		{40, "D grade"},
		{46, "C grade"},
		{52, "B grade"},
	} {
		name := "lv" + strconv.Itoa(int(tc.level))
		sess, p := e.enterChar(testChar{account: name, name: name, pos: shopPos, level: tc.level})
		lector := e.shopLector(sess, p)
		pg := onePage(t, e.shopCmd(sess, lector, "BuyShadowItem"))
		if want := "<html><body>" + tc.want + " (" + strconv.Itoa(int(lector.ID())) + ")</body></html>"; pg.HTML != want {
			t.Errorf("level %d: %q, want %q", tc.level, pg.HTML, want)
		}
	}
}
