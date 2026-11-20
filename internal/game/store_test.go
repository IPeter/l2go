package game

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// itemStem is a stackable, tradeable item of the store tests (itemVarnish,
// 1865, is another).
const itemStem = 1864

// storeAt is a place of the store tests: dx, dy from the human fighters'
// first creation point, where no NPC stands.
func storeAt(dx, dy int32) world.Position {
	return world.Position{X: tiStart.X + dx, Y: tiStart.Y + dy, Z: tiStart.Z}
}

// storeItems are stored items for enterWith.
func storeItems(pairs ...int64) []storage.Item {
	var items []storage.Item
	for i := 0; i+1 < len(pairs); i += 2 {
		items = append(items, storage.Item{ItemID: int32(pairs[i]), Count: pairs[i+1]})
	}
	return items
}

// storeType returns p's store mode, read on the loop.
func (e *itemEnv) storeType(p *Player) StoreType {
	var t StoreType
	e.onLoop(func() { t = p.StoreType() })
	return t
}

// storeObject returns the object ID of p's first item of the template.
func (e *itemEnv) storeObject(p *Player, itemID int32) int32 {
	e.t.Helper()
	it := e.itemOf(p, itemID)
	if it == nil {
		e.t.Fatalf("%s has no item %d", p.Name(), itemID)
	}
	return it.ObjectID()
}

// storeAction sends RequestActionUse with the action.
func (e *itemEnv) storeAction(sess *fakeSession, action int32) []testclient.Packet {
	return e.handle(sess, clientpackets.OpRequestActionUse, actionBody(action))
}

// storeClick clicks on the player o (Action).
func (e *itemEnv) storeClick(sess *fakeSession, o *Player) []testclient.Packet {
	return e.handle(sess, clientpackets.OpAction, npcActionBody(o.ObjectID(), false))
}

// storeSellList sends SetPrivateStoreListSell with object ID, count, price
// triples.
func (e *itemEnv) storeSellList(sess *fakeSession, packageSale bool, triples ...int32) []testclient.Packet {
	flag := int32(0)
	if packageSale {
		flag = 1
	}
	body := append([]int32{flag, int32(len(triples) / 3)}, triples...)
	return e.handle(sess, clientpackets.OpSetPrivateStoreListSell, ints(body...))
}

// storeBuyList sends SetPrivateStoreListBuy with item ID, count, price
// triples.
func (e *itemEnv) storeBuyList(sess *fakeSession, triples ...int32) []testclient.Packet {
	w := packets.NewWriter(4 + 16*len(triples)/3)
	w.WriteInt32(int32(len(triples) / 3))
	for i := 0; i+2 < len(triples); i += 3 {
		w.WriteInt32(triples[i])
		w.WriteUint16(0)
		w.WriteUint16(0)
		w.WriteInt32(triples[i+1])
		w.WriteInt32(triples[i+2])
	}
	return e.handle(sess, clientpackets.OpSetPrivateStoreListBuy, w.Bytes())
}

// storeBuy sends RequestPrivateStoreBuy with object ID, count, price triples.
func (e *itemEnv) storeBuy(sess *fakeSession, owner *Player, triples ...int32) []testclient.Packet {
	body := append([]int32{owner.ObjectID(), int32(len(triples) / 3)}, triples...)
	return e.handle(sess, clientpackets.OpRequestPrivateStoreBuy, ints(body...))
}

// storeSell sends RequestPrivateStoreSell with object ID, item ID, count,
// price quadruples.
func (e *itemEnv) storeSell(sess *fakeSession, owner *Player, quads ...int32) []testclient.Packet {
	w := packets.NewWriter(8 + 20*len(quads)/4)
	w.WriteInt32(owner.ObjectID())
	w.WriteInt32(int32(len(quads) / 4))
	for i := 0; i+3 < len(quads); i += 4 {
		w.WriteInt32(quads[i])
		w.WriteInt32(quads[i+1])
		w.WriteUint16(0)
		w.WriteUint16(0)
		w.WriteInt32(quads[i+2])
		w.WriteInt32(quads[i+3])
	}
	return e.handle(sess, clientpackets.OpRequestPrivateStoreSell, w.Bytes())
}

// storeSaved returns how many of the template the stored character carries,
// once its pending saves are written.
func (e *itemEnv) storeSaved(p *Player, itemID int32) int64 {
	e.t.Helper()
	if err := e.srv.saver.wait(e.ctx, p.ObjectID()); err != nil {
		e.t.Fatalf("wait for the saves of %s: %v", p.Name(), err)
	}
	c, err := e.store.Get(e.ctx, p.ObjectID())
	if err != nil {
		e.t.Fatalf("Get %s: %v", p.Name(), err)
	}
	var n int64
	for _, it := range c.Items {
		if it.ItemID == itemID && it.Normalized().Loc != storage.LocWarehouse {
			n += it.Count
		}
	}
	return n
}

// storeKicked reports whether ps tell the player it is kicked for an
// illegal action.
func storeKicked(ps []testclient.Packet) bool {
	for _, m := range only[testclient.SystemMessage](ps) {
		for _, s := range m.Texts() {
			if strings.Contains(s, "kicked for illegal action") {
				return true
			}
		}
	}
	return false
}

// storeUserInfoType returns the store type of the last UserInfo in ps.
func storeUserInfoType(t *testing.T, ps []testclient.Packet) uint8 {
	t.Helper()
	uis := only[testclient.UserInfo](ps)
	if len(uis) == 0 {
		t.Fatalf("no UserInfo in %q", testclient.Names(ps))
	}
	return uis[len(uis)-1].PrivateStoreType
}

// A sell store: the setup window, the message, opening (sitting, the store
// type, the message to everyone), the window a second click opens, a
// purchase with its messages and the joint save, a wrong price, and the last
// items, after which the store closes.
func TestStoreSell(t *testing.T) {
	e := newItemEnv(t, nil)
	sa, a := e.enterWith("seller", storeAt(0, 0), storeItems(itemMithrilOre, 100, itemLongSword, 1)...)
	sb, b := e.enterWith("buyer", storeAt(40, 0), storeItems(itemAdena, 100000)...)
	sa.take(t)
	ore, sword := e.storeObject(a, itemMithrilOre), e.storeObject(a, itemLongSword)

	ps := e.storeAction(sa, clientpackets.ActionUsePrivateStoreSell)
	m := shopOne[testclient.PrivateStoreManageListSell](t, "setup window", ps)
	_, oreOK := m.FindAvailable(ore)
	_, swordOK := m.FindAvailable(sword)
	if !oreOK || !swordOK || len(m.Items) != 0 || m.PackageSale {
		t.Fatalf("setup window = %+v, want the ore and the sword available", m)
	}
	for _, it := range m.Available {
		if it.ItemID == itemSquireSword || it.ItemID == 10 { // worn; the dagger is not tradeable
			t.Fatalf("setup window offers item %d", it.ItemID)
		}
	}
	if got := e.storeType(a); got != StoreSellManage {
		t.Fatalf("store type after the setup window = %d, want %d", got, StoreSellManage)
	}
	ps = e.handle(sa, clientpackets.OpSetPrivateStoreMsgSell, stringBody("Cheap ore"))
	if msg := shopOne[testclient.PrivateStoreMsgSell](t, "message", ps); msg.Message != "Cheap ore" || msg.ObjectID != a.ObjectID() {
		t.Fatalf("message = %+v", msg)
	}

	sb.take(t)
	ps = e.storeSellList(sa, false, ore, 50, 10, sword, 1, 500)
	checkNames(t, "seller opens the store", ps, "ChangeWaitType", "UserInfo", "PrivateStoreMsgSell")
	if got := storeUserInfoType(t, ps); got != uint8(StoreSell) {
		t.Fatalf("UserInfo store type = %d, want %d", got, StoreSell)
	}
	pb := sb.take(t)
	checkNames(t, "the buyer sees the store open", pb, "ChangeWaitType", "CharInfo", "PrivateStoreMsgSell")
	if ci := shopOne[testclient.CharInfoPacket](t, "CharInfo", pb); ci.PrivateStoreType != uint8(StoreSell) {
		t.Fatalf("CharInfo store type = %d, want %d", ci.PrivateStoreType, StoreSell)
	}

	e.storeClick(sb, a)
	ps = e.storeClick(sb, a)
	l := shopOne[testclient.PrivateStoreListSell](t, "store window", ps)
	if it, ok := l.Find(ore); !ok || len(l.Items) != 2 || l.Adena != 100000 || it.Count != 50 || it.Price != 10 || it.ReferencePrice != 1000 || it.ItemID != itemMithrilOre {
		t.Fatalf("store window = %+v", l)
	}

	sa.take(t)
	ps = e.storeBuy(sb, a, ore, 20, 10)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(storeMsgYouPurchasedS3)}) {
		t.Fatalf("buyer messages = %v", ids)
	}
	if ids := sysMsgIDsOf(sa.take(t)); !slices.Equal(ids, []int32{int32(storeMsgPurchasedS3S2)}) {
		t.Fatalf("seller messages = %v", ids)
	}
	if e.countOf(b, itemMithrilOre) != 20 || e.shopAdena(b) != 99800 || e.countOf(a, itemMithrilOre) != 80 || e.shopAdena(a) != 200 {
		t.Fatal("the purchase did not move 20 ore for 200 adena")
	}
	if e.storeSaved(a, itemMithrilOre) != 80 || e.storeSaved(a, itemAdena) != 200 || e.storeSaved(b, itemMithrilOre) != 20 || e.storeSaved(b, itemAdena) != 99800 {
		t.Fatal("the purchase was not saved for both")
	}

	e.advance(time.Second) // the transaction flood guard
	ps = e.storeBuy(sb, a, sword, 1, 499)
	if len(sysMsgIDsOf(ps)) != 0 || e.countOf(a, itemLongSword) != 1 {
		t.Fatalf("a wrong price bought the sword: %q", testclient.Names(ps))
	}

	e.advance(time.Second)
	sa.take(t)
	ps = e.storeBuy(sb, a, sword, 1, 500, ore, 40, 10) // only 30 ore are left
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(storeMsgYouPurchasedS3), int32(storeMsgYouPurchasedS2)}) {
		t.Fatalf("buyer messages = %v", ids)
	}
	if e.countOf(b, itemMithrilOre) != 50 || e.countOf(b, itemLongSword) != 1 || e.shopAdena(a) != 1000 {
		t.Fatal("the rest of the store did not move")
	}
	e.advance(200 * time.Millisecond) // the end of the tick sends UserInfo
	pa := sa.take(t)
	if got := storeUserInfoType(t, pa); got != uint8(StoreNone) || e.storeType(a) != StoreNone {
		t.Fatalf("the empty store did not close: UserInfo type %d", got)
	}
	var sitting bool
	e.onLoop(func() { sitting = a.Sitting })
	if !sitting {
		t.Error("the owner of a closed store stood up")
	}
}

// A package sale is bought whole: fewer entries than the package, or an
// entry at another price, kicks the buyer and moves nothing.
func TestStorePackageSale(t *testing.T) {
	e := newItemEnv(t, nil)
	sa, a := e.enterWith("seller", storeAt(0, 0), storeItems(itemMithrilOre, 100, itemStem, 10)...)
	s1, b1 := e.enterWith("partial", storeAt(40, 0), storeItems(itemAdena, 1000)...)
	s2, b2 := e.enterWith("cheater", storeAt(0, 40), storeItems(itemAdena, 1000)...)
	s3, b3 := e.enterWith("buyer", storeAt(-40, 0), storeItems(itemAdena, 1000)...)
	ore, stem := e.storeObject(a, itemMithrilOre), e.storeObject(a, itemStem)

	e.storeAction(sa, clientpackets.ActionUsePrivateStoreSell)
	ps := e.storeSellList(sa, true, ore, 50, 10, stem, 10, 5)
	if got := storeUserInfoType(t, ps); got != uint8(StorePackageSell) {
		t.Fatalf("UserInfo store type = %d, want %d", got, StorePackageSell)
	}
	s3.take(t)
	e.storeClick(s3, a)
	if l := shopOne[testclient.PrivateStoreListSell](t, "package window", e.storeClick(s3, a)); !l.PackageSale || len(l.Items) != 2 {
		t.Fatalf("package window = %+v", l)
	}

	if ps := e.storeBuy(s1, a, ore, 50, 10); !storeKicked(ps) {
		t.Fatalf("buying part of a package: %q, want a kick", testclient.Names(ps))
	}
	if ps := e.storeBuy(s2, a, ore, 50, 10, stem, 10, 4); !storeKicked(ps) {
		t.Fatalf("buying a package at another price: %q, want a kick", testclient.Names(ps))
	}
	if e.shopAdena(b1) != 1000 || e.shopAdena(b2) != 1000 || e.countOf(a, itemMithrilOre) != 100 {
		t.Fatal("a refused package purchase moved something")
	}

	ps = e.storeBuy(s3, a, ore, 50, 10, stem, 10, 5)
	if storeKicked(ps) || e.shopAdena(b3) != 450 || e.countOf(b3, itemMithrilOre) != 50 || e.countOf(b3, itemStem) != 10 {
		t.Fatalf("the whole package did not sell: %q", testclient.Names(ps))
	}
	if e.storeType(a) != StoreNone {
		t.Error("the sold package did not close")
	}
}

// A buy store: the setup window, opening, the window a seller opens (his
// item and count), sales cut to what the store still wants, both saved, and
// the store closing once it has everything. A total over the owner's adena
// is refused, a price over MaxAdena punished.
func TestStoreBuy(t *testing.T) {
	e := newItemEnv(t, nil)
	sa, a := e.enterWith("owner", storeAt(0, 0), storeItems(itemAdena, 1000, itemStem, 5)...)
	sb, b := e.enterWith("seller", storeAt(40, 0), storeItems(itemMithrilOre, 50)...)
	sc, _ := e.enterWith("greedy", storeAt(-300, 0), storeItems(itemAdena, 10)...)
	ore := e.storeObject(b, itemMithrilOre)

	ps := e.handle(sa, clientpackets.OpRequestPrivateStoreManageBuy, nil)
	m := shopOne[testclient.PrivateStoreManageListBuy](t, "setup window", ps)
	var listed []int32
	for _, it := range m.Inventory {
		listed = append(listed, it.ItemID)
	}
	if !slices.Contains(listed, itemStem) || slices.Contains(listed, itemAdena) || slices.Contains(listed, itemSquireSword) || m.Adena != 1000 {
		t.Fatalf("setup window lists %v, want the stem, without adena and worn items", listed)
	}

	ps = e.storeBuyList(sa, itemMithrilOre, 60, 20) // 1200 adena
	checkNames(t, "a buy store over the owner's adena", ps, "PrivateStoreManageListBuy", "SystemMessage")
	if ids := sysMsgIDsOf(ps); ids[0] != int32(storeMsgPriceTooHigh) {
		t.Fatalf("message = %v", ids)
	}
	sb.take(t)
	ps = e.storeBuyList(sa, itemMithrilOre, 30, 20)
	checkNames(t, "owner opens the store", ps, "ChangeWaitType", "UserInfo", "PrivateStoreMsgBuy")
	if got := storeUserInfoType(t, ps); got != uint8(StoreBuy) {
		t.Fatalf("UserInfo store type = %d, want %d", got, StoreBuy)
	}

	e.storeClick(sb, a)
	l := shopOne[testclient.PrivateStoreListBuy](t, "store window", e.storeClick(sb, a))
	if it, ok := l.FindItem(itemMithrilOre); !ok || it.ObjectID != ore || it.Count != 30 || it.StoreCount != 30 || it.Price != 20 {
		t.Fatalf("store window = %+v", l)
	}

	sa.take(t)
	ps = e.storeSell(sb, a, ore, itemMithrilOre, 10, 20)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(storeMsgPurchasedS3S2)}) {
		t.Fatalf("seller messages = %v", ids)
	}
	if ids := sysMsgIDsOf(sa.take(t)); !slices.Equal(ids, []int32{int32(storeMsgYouPurchasedS3)}) {
		t.Fatalf("owner messages = %v", ids)
	}
	if e.countOf(a, itemMithrilOre) != 10 || e.shopAdena(a) != 800 || e.countOf(b, itemMithrilOre) != 40 || e.shopAdena(b) != 200 {
		t.Fatal("the sale did not move 10 ore for 200 adena")
	}
	if e.storeSaved(a, itemMithrilOre) != 10 || e.storeSaved(b, itemAdena) != 200 {
		t.Fatal("the sale was not saved for both")
	}

	e.advance(time.Second)
	e.storeSell(sb, a, ore, itemMithrilOre, 25, 20) // the store wants 20 more
	if e.countOf(a, itemMithrilOre) != 30 || e.shopAdena(a) != 400 || e.countOf(b, itemMithrilOre) != 20 || e.shopAdena(b) != 600 {
		t.Fatal("the second sale was not cut to what the store wanted")
	}
	if e.storeType(a) != StoreNone {
		t.Error("the full buy store did not close")
	}

	e.storeAction(sc, clientpackets.ActionUsePrivateStoreBuy)
	if ps := e.storeBuyList(sc, itemMithrilOre, 2, 1_500_000_000); !storeKicked(ps) {
		t.Fatalf("a price over MaxAdena: %q, want a kick", testclient.Names(ps))
	}
}

// Where a store may not open: a no-store zone, near an NPC, near a sitting
// player.
func TestStoreZones(t *testing.T) {
	e := newItemEnv(t, nil)
	giran := world.Position{X: 80000, Y: 145500, Z: -3500} // Giran - Luxury noshop
	sz, z := e.enterWith("zoned", giran, storeItems(itemMithrilOre, 10)...)
	sa, _ := e.enterWith("sitter", storeAt(0, 0))
	sb, b := e.enterWith("neighbour", storeAt(40, 0), storeItems(itemMithrilOre, 10)...)
	sn, n := e.enterWith("npcside", storeAt(1000, 0), storeItems(itemMithrilOre, 10)...)

	refused := func(what string, ps []testclient.Packet) {
		t.Helper()
		if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(storeMsgCannotOpenHere)}) || len(only[testclient.ActionFailed](ps)) != 1 {
			t.Fatalf("%s: %q %v, want \"You cannot open a Private Store here.\" and ActionFailed", what, testclient.Names(ps), ids)
		}
	}
	refused("no-store zone", e.storeAction(sz, clientpackets.ActionUsePrivateStoreSell))
	if e.storeType(z) != StoreNone {
		t.Fatal("a store opened in a no-store zone")
	}
	ps := e.storeSellList(sz, false, e.storeObject(z, itemMithrilOre), 1, 1)
	checkNames(t, "setting up in a no-store zone", ps, "PrivateStoreManageListSell", "SystemMessage", "ActionFailed")

	e.spawnNpc(npcGuard, storeAt(1080, 0))
	refused("next to an NPC", e.storeAction(sn, clientpackets.ActionUsePrivateStoreBuy))
	if e.storeType(n) != StoreNone {
		t.Fatal("a store opened next to an NPC")
	}

	e.storeAction(sa, clientpackets.ActionUseSitStand)
	refused("next to a sitting player", e.storeAction(sb, clientpackets.ActionUsePrivateStoreSell))
	e.advance(3 * time.Second)
	e.storeAction(sa, clientpackets.ActionUseSitStand)
	e.advance(3 * time.Second)
	sb.take(t)
	shopOne[testclient.PrivateStoreManageListSell](t, "next to a standing player", e.storeAction(sb, clientpackets.ActionUsePrivateStoreSell))
	if e.storeType(b) != StoreSellManage {
		t.Fatal("the setup window did not open next to a standing player")
	}
}

// The slots of a store: 3 sell and 4 buy items for a human, plus the
// configured counts.
func TestStoreSlotLimits(t *testing.T) {
	e := newItemEnv(t, func(gs *config.GameServer) { gs.Stores.BuySlotsOther = 2 })
	sa, a := e.enterWith("seller", storeAt(0, 0), storeItems(itemMithrilOre, 10, itemStem, 10, itemVarnish, 10, itemLongSword, 1, itemAdena, 1000)...)
	ore, stem, varnish, sword := e.storeObject(a, itemMithrilOre), e.storeObject(a, itemStem), e.storeObject(a, itemVarnish), e.storeObject(a, itemLongSword)

	e.storeAction(sa, clientpackets.ActionUsePrivateStoreSell)
	ps := e.storeSellList(sa, false, ore, 1, 1, stem, 1, 1, varnish, 1, 1, sword, 1, 1)
	checkNames(t, "four sell items", ps, "PrivateStoreManageListSell", "SystemMessage")
	if ids := sysMsgIDsOf(ps); ids[0] != int32(storeMsgQuantityExceeds) {
		t.Fatalf("message = %v", ids)
	}
	ps = e.storeSellList(sa, false, ore, 1, 1, stem, 1, 1, varnish, 1, 1)
	if got := storeUserInfoType(t, ps); got != uint8(StoreSell) {
		t.Fatalf("three sell items: store type %d", got)
	}
	e.advance(3 * time.Second)
	e.handle(sa, clientpackets.OpRequestPrivateStoreQuitSell, nil)
	e.advance(3 * time.Second)

	e.storeAction(sa, clientpackets.ActionUsePrivateStoreBuy)
	ps = e.storeBuyList(sa, itemMithrilOre, 1, 1, itemStem, 1, 1, itemVarnish, 1, 1)
	checkNames(t, "three buy items", ps, "PrivateStoreManageListBuy", "SystemMessage")
	ps = e.storeBuyList(sa, itemMithrilOre, 1, 1, itemStem, 1, 1)
	if got := storeUserInfoType(t, ps); got != uint8(StoreBuy) {
		t.Fatalf("two buy items: store type %d", got)
	}
}

// Store mode refuses restart, item use, destroying and trading, and keeps
// the owner seated; a player coming into view gets the message after the
// CharInfo; a long message is punished; quitting stands the owner up, and
// leaving the game closes the store.
func TestStoreClosing(t *testing.T) {
	e := newItemEnv(t, nil)
	sa, a := e.enterWith("seller", storeAt(0, 0), storeItems(itemMithrilOre, 10)...)
	sb, _ := e.enterWith("trader", storeAt(40, 0))
	ore := e.storeObject(a, itemMithrilOre)
	e.storeAction(sa, clientpackets.ActionUsePrivateStoreSell)
	e.handle(sa, clientpackets.OpSetPrivateStoreMsgSell, stringBody("Ore"))
	e.storeSellList(sa, false, ore, 5, 7)
	e.advance(3 * time.Second)
	sa.take(t)

	e.srv.Restart(sa)
	ps := sa.take(t)
	if rr := shopOne[testclient.RestartResponse](t, "restart", ps); rr.OK || !e.inWorld(a) {
		t.Fatal("restart in store mode was not refused")
	}
	if ps := e.useItem(sa, ore); !slices.Equal(sysMsgIDsOf(ps), []int32{int32(storeMsgNoItemUse)}) {
		t.Fatalf("item use: %q", testclient.Names(ps))
	}
	if ps := e.handle(sa, clientpackets.OpRequestDestroyItem, ints(ore, 1)); !slices.Equal(sysMsgIDsOf(ps), []int32{int32(tradeMsgStoreNoDiscard)}) || e.countOf(a, itemMithrilOre) != 10 {
		t.Fatalf("destroy: %q", testclient.Names(ps))
	}
	if ps := e.handle(sb, clientpackets.OpTradeRequest, ints(a.ObjectID())); !slices.Equal(sysMsgIDsOf(ps), []int32{int32(tradeMsgStoreNoDiscard)}) {
		t.Fatalf("trade request to a store: %q", testclient.Names(ps))
	}
	if ps := e.storeAction(sa, clientpackets.ActionUseSitStand); len(only[testclient.ChangeWaitType](ps)) != 0 {
		t.Fatal("the owner of a store stood up")
	}

	e.enterWith("viewer", storeAt(0, 60))
	var info [][]byte
	e.onLoop(func() {
		c, _ := e.srv.PlayerByName("viewer")
		info = a.InfoFor(c)
	})
	var names []string
	for _, body := range info {
		p, err := testclient.DecodeServerPacket(body)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, p.PacketName())
	}
	if len(names) < 2 || names[0] != "CharInfo" || names[len(names)-1] != "PrivateStoreMsgSell" {
		t.Fatalf("a new viewer gets %q, want CharInfo first and PrivateStoreMsgSell last", names)
	}

	if ps := e.handle(sa, clientpackets.OpSetPrivateStoreMsgSell, stringBody(strings.Repeat("x", 30))); !storeKicked(ps) {
		t.Fatalf("a 30-character message: %q, want a kick", testclient.Names(ps))
	}

	ps = e.handle(sa, clientpackets.OpRequestPrivateStoreQuitSell, nil)
	if cw := shopOne[testclient.ChangeWaitType](t, "quit", ps); cw.WaitType != 1 || e.storeType(a) != StoreNone {
		t.Fatalf("quit: %+v", cw)
	}

	e.advance(3 * time.Second)
	e.storeAction(sa, clientpackets.ActionUsePrivateStoreSell)
	e.storeSellList(sa, false, ore, 5, 7)
	if e.storeType(a) != StoreSell {
		t.Fatal("the store did not open again")
	}
	e.srv.Disconnected(sa)
	if e.storeType(a) != StoreNone {
		t.Fatal("leaving the game did not close the store")
	}
}

// A second click on a store out of reach walks up to it and opens it.
func TestStoreWalkUp(t *testing.T) {
	e := newItemEnv(t, nil)
	sa, a := e.enterWith("seller", storeAt(0, 0), storeItems(itemMithrilOre, 10)...)
	sb, _ := e.enterWith("buyer", storeAt(300, 0))
	e.storeAction(sa, clientpackets.ActionUsePrivateStoreSell)
	e.storeSellList(sa, false, e.storeObject(a, itemMithrilOre), 5, 7)

	e.storeClick(sb, a)
	ps := e.storeClick(sb, a)
	if len(only[testclient.MoveToPawn](ps)) != 1 || len(only[testclient.PrivateStoreListSell](ps)) != 0 {
		t.Fatalf("a click out of reach: %q, want MoveToPawn", testclient.Names(ps))
	}
	var got []testclient.Packet
	for range 50 {
		e.advance(100 * time.Millisecond)
		got = append(got, sb.take(t)...)
		if len(only[testclient.PrivateStoreListSell](got)) > 0 {
			break
		}
	}
	shopOne[testclient.PrivateStoreListSell](t, "after the walk", got)
}
