package game

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// tradeEnv is two players at the same spot, ready to trade: a ("trader1")
// carries 1,000 adena, b ("trader2") a Long Sword.
type tradeEnv struct {
	*itemEnv
	sa, sb *fakeSession
	a, b   *Player
}

func newTradeEnv(t *testing.T, configure func(gs *config.GameServer)) *tradeEnv {
	t.Helper()
	e := newItemEnv(t, func(gs *config.GameServer) {
		gs.Economy.FloodTransaction = 0
		if configure != nil {
			configure(gs)
		}
	})
	sa, a := e.enterWith("trader1", world.Position{}, storage.Item{ItemID: itemAdena, Count: 1000})
	sb, b := e.enterWith("trader2", world.Position{}, storage.Item{ItemID: itemLongSword, Count: 1})
	e.advance(time.Second) // the rest of the enter-world sequence
	sa.take(t)
	sb.take(t)
	return &tradeEnv{itemEnv: e, sa: sa, sb: sb, a: a, b: b}
}

// send sends a client packet of sess and returns what each player got.
func (e *tradeEnv) send(sess *fakeSession, op byte, body []byte) (toA, toB []testclient.Packet) {
	e.t.Helper()
	e.srv.Handle(sess, op, body)
	e.sync()
	return e.sa.take(e.t), e.sb.take(e.t)
}

// open makes a ask b to trade and b accept.
func (e *tradeEnv) open() {
	e.t.Helper()
	toA, toB := e.send(e.sa, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	checkNames(e.t, "a after the request", toA, "SystemMessage")
	checkNames(e.t, "b after the request", toB, "SendTradeRequest")
	if got := shopOne[testclient.SendTradeRequest](e.t, "b", toB).SenderID; got != e.a.ObjectID() {
		e.t.Fatalf("SendTradeRequest names %d, want %d", got, e.a.ObjectID())
	}
	toA, toB = e.send(e.sb, clientpackets.OpAnswerTradeRequest, ints(1))
	checkNames(e.t, "a after the answer", toA, "SystemMessage", "TradeStart")
	checkNames(e.t, "b after the answer", toB, "SystemMessage", "TradeStart")
	if got := shopOne[testclient.TradeStart](e.t, "a", toA).PartnerID; got != e.b.ObjectID() {
		e.t.Fatalf("a's TradeStart names partner %d, want %d", got, e.b.ObjectID())
	}
}

// add puts count of the item into sess's trade.
func (e *tradeEnv) add(sess *fakeSession, objectID, count int32) (toA, toB []testclient.Packet) {
	e.t.Helper()
	return e.send(sess, clientpackets.OpAddTradeItem, ints(0, objectID, count))
}

// done presses OK (true) or Cancel in sess's trade window.
func (e *tradeEnv) done(sess *fakeSession, confirm bool) (toA, toB []testclient.Packet) {
	e.t.Helper()
	v := int32(0)
	if confirm {
		v = 1
	}
	return e.send(sess, clientpackets.OpTradeDone, ints(v))
}

// objectOf returns the object ID of p's item of the template.
func (e *tradeEnv) objectOf(p *Player, itemID int32) int32 {
	e.t.Helper()
	it := e.itemOf(p, itemID)
	if it == nil {
		e.t.Fatalf("%s has no item %d", p.Name(), itemID)
	}
	return it.ObjectID()
}

// trading reports whether p has a trade open, read on the loop.
func (e *tradeEnv) trading(p *Player) bool {
	var on bool
	e.onLoop(func() { on = e.srv.Trading(p) })
	return on
}

// stored returns how many of the template the stored character carries,
// once its pending writes are done.
func (e *tradeEnv) stored(p *Player, itemID int32) int64 {
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
		if it.ItemID == itemID {
			n += it.Count
		}
	}
	return n
}

// checkTradeDone fails unless ps hold exactly one TradeDone with result.
func checkTradeDone(t *testing.T, who string, ps []testclient.Packet, result int32) {
	t.Helper()
	if got := shopOne[testclient.TradeDone](t, who, ps).Result; got != result {
		t.Fatalf("%s: TradeDone(%d), want %d", who, got, result)
	}
}

// checkTradeMsgs fails unless the system messages among ps are want, in order.
func checkTradeMsgs(t *testing.T, who string, ps []testclient.Packet, want ...int32) {
	t.Helper()
	if got := sysMsgIDsOf(ps); !slices.Equal(got, want) {
		t.Fatalf("%s: system messages %v, want %v (packets %q)", who, got, want, testclient.Names(ps))
	}
}

// TestTradeSuccess: request, accept, both add, both confirm; the items
// change sides with Mobius's packets and are stored by one joint save.
func TestTradeSuccess(t *testing.T) {
	e := newTradeEnv(t, nil)
	e.open()
	adena, sword := e.objectOf(e.a, itemAdena), e.objectOf(e.b, itemLongSword)

	toA, toB := e.add(e.sa, adena, 500)
	checkNames(t, "a after adding", toA, "TradeOwnAdd", "TradeUpdate")
	checkNames(t, "b after a added", toB, "TradeOtherAdd")
	if own := shopOne[testclient.TradeOwnAdd](t, "a", toA).Item; own.ObjectID != adena || own.Count != 500 {
		t.Fatalf("TradeOwnAdd = %+v, want 500 of %d", own, adena)
	}
	upd := shopOne[testclient.TradeUpdate](t, "a", toA)
	if len(upd.Items) != 1 || upd.Items[0].Count != 500 || upd.Items[0].Change != testclient.TradeUpdateModify {
		t.Fatalf("TradeUpdate = %+v, want 500 adena left (modified)", upd)
	}
	toA, toB = e.add(e.sb, sword, 1)
	checkNames(t, "b after adding", toB, "TradeOwnAdd", "TradeUpdate")
	checkNames(t, "a after b added", toA, "TradeOtherAdd")

	toA, toB = e.done(e.sa, true)
	checkNames(t, "a after confirming", toA)
	checkNames(t, "b after a confirmed", toB, "SystemMessage", "TradeOtherDone")
	checkTradeMsgs(t, "b", toB, int32(tradeMsgConfirmed))

	toA, toB = e.done(e.sb, true)
	checkNames(t, "a after the exchange", toA, "StatusUpdate", "TradeDone", "SystemMessage")
	checkNames(t, "b after the exchange", toB, "StatusUpdate", "TradeDone", "SystemMessage")
	checkTradeDone(t, "a", toA, 1)
	checkTradeMsgs(t, "a", toA, int32(tradeMsgSuccessful))
	e.advance(300 * time.Millisecond)
	checkNames(t, "a later", e.sa.take(t), "InventoryUpdate", "ItemList")
	checkNames(t, "b later", e.sb.take(t), "InventoryUpdate", "ItemList")

	if e.trading(e.a) || e.trading(e.b) {
		t.Fatal("a trade is still open after the exchange")
	}
	if got := e.countOf(e.a, itemAdena); got != 500 {
		t.Errorf("a has %d adena, want 500", got)
	}
	if got := e.countOf(e.b, itemAdena); got != 500 {
		t.Errorf("b has %d adena, want 500", got)
	}
	if e.countOf(e.a, itemLongSword) != 1 || e.countOf(e.b, itemLongSword) != 0 {
		t.Error("the Long Sword did not move from b to a")
	}
	if got := e.stored(e.a, itemAdena); got != 500 {
		t.Errorf("stored a has %d adena, want 500", got)
	}
	if got := e.stored(e.b, itemAdena); got != 500 {
		t.Errorf("stored b has %d adena, want 500", got)
	}
	if e.stored(e.a, itemLongSword) != 1 || e.stored(e.b, itemLongSword) != 0 {
		t.Error("the stored Long Sword did not move from b to a")
	}
}

// TestTradeDecline: a refusal tells the requester; a new request works.
func TestTradeDecline(t *testing.T) {
	e := newTradeEnv(t, nil)
	e.send(e.sa, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	toA, toB := e.send(e.sb, clientpackets.OpAnswerTradeRequest, ints(0))
	checkTradeMsgs(t, "a", toA, int32(tradeMsgDenied))
	checkNames(t, "b", toB)
	if texts := shopOne[testclient.SystemMessage](t, "a", toA).Texts(); !slices.Equal(texts, []string{"trader2"}) {
		t.Errorf("denial names %q, want trader2", texts)
	}
	toA, _ = e.send(e.sa, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	checkTradeMsgs(t, "a asking again", toA, int32(tradeMsgRequested))
}

// TestTradeCancel: Cancel closes both windows with "$s1 has canceled the
// trade." naming the one who cancelled; nothing moves.
func TestTradeCancel(t *testing.T) {
	e := newTradeEnv(t, nil)
	e.open()
	e.add(e.sa, e.objectOf(e.a, itemAdena), 300)
	toA, toB := e.done(e.sa, false)
	checkTradeDone(t, "a", toA, 0)
	checkTradeDone(t, "b", toB, 0)
	checkTradeMsgs(t, "a", toA, int32(tradeMsgCanceled))
	checkTradeMsgs(t, "b", toB, int32(tradeMsgCanceled))
	if texts := shopOne[testclient.SystemMessage](t, "b", toB).Texts(); !slices.Equal(texts, []string{"trader1"}) {
		t.Errorf("cancel names %q, want trader1", texts)
	}
	if e.trading(e.a) || e.trading(e.b) {
		t.Fatal("a trade is still open after the cancel")
	}
	if got := e.countOf(e.a, itemAdena); got != 1000 {
		t.Errorf("a has %d adena, want 1000", got)
	}
}

// TestTradeDistance: a request from over 150 units is refused; a partner
// who walked away cancels the trade on confirmation.
func TestTradeDistance(t *testing.T) {
	e := newTradeEnv(t, nil)
	pos := e.position(e.a)
	far := world.Position{X: pos.X + 200, Y: pos.Y, Z: pos.Z}
	e.shopMove(e.b, far)
	e.sa.take(t)
	e.sb.take(t)
	toA, toB := e.send(e.sa, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	checkTradeMsgs(t, "a asking from afar", toA, int32(sysMsgTargetOutOfRange))
	checkNames(t, "b", toB)

	e.shopMove(e.b, pos)
	e.sa.take(t)
	e.sb.take(t)
	e.open()
	e.shopMove(e.b, far)
	e.sa.take(t)
	e.sb.take(t)
	toA, toB = e.done(e.sa, true)
	checkTradeDone(t, "a", toA, 0)
	checkTradeDone(t, "b", toB, 0)
	if e.trading(e.a) || e.trading(e.b) {
		t.Fatal("a trade is still open")
	}
}

// TestTradeOverweight: the receiver's weight limit fails the exchange with
// the weight message and TradeDone(1) without success; nothing moves.
func TestTradeOverweight(t *testing.T) {
	e := newTradeEnv(t, nil)
	var limit, load int64
	e.onLoop(func() { limit, load = int64(e.b.stats.WeightLimit()), e.b.inv.Weight() })
	tmpl, ok := e.data.ItemTemplate(itemMithrilOre)
	if !ok || tmpl.Weight <= 0 {
		t.Fatal("no Mithril Ore with a weight")
	}
	n := (limit-load)/int64(tmpl.Weight) + 1
	ore := e.give(e.a, itemMithrilOre, n)
	e.sync()
	e.sa.take(t)
	e.open()
	e.add(e.sa, ore.ObjectID(), int32(n))
	e.done(e.sb, true)
	toA, toB := e.done(e.sa, true)
	checkTradeMsgs(t, "a", toA, int32(exchangeMsgWeightLimit))
	checkTradeMsgs(t, "b", toB, int32(exchangeMsgWeightLimit))
	checkTradeDone(t, "a", toA, 1)
	checkTradeDone(t, "b", toB, 1)
	if got := e.countOf(e.b, itemMithrilOre); got != 0 {
		t.Errorf("b got %d ore", got)
	}
	if got := e.countOf(e.a, itemMithrilOre); got != n {
		t.Errorf("a has %d ore, want %d", got, n)
	}
}

// TestTradeInventoryFull: a receiver without a free slot fails the exchange
// with "Your inventory is full." to both and TradeDone(1).
func TestTradeInventoryFull(t *testing.T) {
	e := newTradeEnv(t, nil)
	e.onLoop(func() {
		for e.a.inv.Len() < e.a.stats.InventoryLimit() {
			if _, err := e.a.inv.Create(itemMagicRing, 1); err != nil { // light, so the weight fits
				t.Errorf("Create: %v", err)
				return
			}
		}
		e.srv.ItemsChanged(e.a)
	})
	e.advance(300 * time.Millisecond)
	e.sa.take(t)
	e.sb.take(t)
	e.open()
	e.add(e.sb, e.objectOf(e.b, itemLongSword), 1)
	e.done(e.sa, true)
	toA, toB := e.done(e.sb, true)
	checkTradeMsgs(t, "a", toA, int32(sysMsgInventoryFull))
	checkTradeMsgs(t, "b", toB, int32(sysMsgInventoryFull))
	checkTradeDone(t, "a", toA, 1)
	checkTradeDone(t, "b", toB, 1)
	if e.countOf(e.b, itemLongSword) != 1 || e.countOf(e.a, itemLongSword) != 0 {
		t.Error("the Long Sword moved")
	}
}

// TestTradePartnerLeaves: a partner who leaves mid-trade cancels it for
// both; the one left behind is told.
func TestTradePartnerLeaves(t *testing.T) {
	e := newTradeEnv(t, nil)
	e.open()
	e.add(e.sb, e.objectOf(e.b, itemLongSword), 1)
	e.sa.take(t)
	e.onLoop(func() { e.srv.Leave(e.b, nil) })
	toA := e.sa.take(t)
	checkTradeDone(t, "a", toA, 0)
	checkTradeMsgs(t, "a", toA, int32(tradeMsgCanceled))
	if texts := shopOne[testclient.SystemMessage](t, "a", toA).Texts(); !slices.Equal(texts, []string{"trader2"}) {
		t.Errorf("cancel names %q, want trader2", texts)
	}
	if e.trading(e.a) {
		t.Fatal("a still trades")
	}
	if got := e.stored(e.b, itemLongSword); got != 1 {
		t.Errorf("stored b has %d Long Swords, want 1", got)
	}
}

// TestTradeExpiry: an answer after 15 s finds no requester: the window
// closes with "That player is not online." and nobody trades.
func TestTradeExpiry(t *testing.T) {
	e := newTradeEnv(t, nil)
	e.send(e.sa, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	e.advance(requestTimeout + time.Second)
	e.sa.take(t)
	e.sb.take(t)
	toA, toB := e.send(e.sb, clientpackets.OpAnswerTradeRequest, ints(1))
	checkTradeDone(t, "b", toB, 0)
	checkTradeMsgs(t, "b", toB, int32(tradeMsgNotOnline))
	checkNames(t, "a", toA)
	if e.trading(e.a) || e.trading(e.b) {
		t.Fatal("a trade opened after the request expired")
	}
	toA, _ = e.send(e.sa, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	checkTradeMsgs(t, "a asking again", toA, int32(tradeMsgRequested))
}

// TestTradeBusy: a player asked by someone else, or trading, cannot be
// asked; one who asked cannot ask again.
func TestTradeBusy(t *testing.T) {
	e := newTradeEnv(t, nil)
	sc, c := e.enterWith("trader3", world.Position{})
	e.sync()
	e.sa.take(t)
	e.sb.take(t)
	e.send(e.sa, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	toA, _ := e.send(e.sa, clientpackets.OpTradeRequest, ints(c.ObjectID()))
	checkTradeMsgs(t, "a asking twice", toA, int32(tradeMsgAlreadyTrading))
	e.srv.Handle(sc, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	e.sync()
	if got := sysMsgIDsOf(sc.take(t)); !slices.Equal(got, []int32{int32(tradeMsgPartnerTrading)}) {
		t.Fatalf("c asking a busy b: messages %v", got)
	}
}

// TestTradeAddRules: TradeList.addItem's refusals, and no changes once a
// side confirmed.
func TestTradeAddRules(t *testing.T) {
	e := newTradeEnv(t, nil)
	quest := e.give(e.a, itemDarinsLetter, 1)
	bow := e.give(e.a, itemBow, 1)
	e.sync()
	e.sa.take(t)
	e.open()
	adena := e.objectOf(e.a, itemAdena)
	for _, c := range []struct {
		name            string
		objectID, count int32
	}{
		{"a quest item", quest.ObjectID(), 1},
		{"more than owned", adena, 1001},
		{"two of an item that does not stack", bow.ObjectID(), 2},
	} {
		toA, toB := e.add(e.sa, c.objectID, c.count)
		if len(only[testclient.TradeOwnAdd](toA)) != 0 || len(toB) != 0 {
			t.Errorf("%s: a got %q, b got %q, want nothing added", c.name, testclient.Names(toA), testclient.Names(toB))
		}
	}
	toA, _ := e.add(e.sa, e.objectOf(e.b, itemLongSword), 1)
	checkTradeMsgs(t, "an item of the partner", toA, int32(tradeMsgNothingHappened))
	e.add(e.sa, adena, 10)
	toA, toB := e.add(e.sa, adena, 10)
	if len(toA) != 0 || len(toB) != 0 {
		t.Errorf("adding twice: a got %q, b got %q", testclient.Names(toA), testclient.Names(toB))
	}
	e.done(e.sb, true)
	toA, _ = e.add(e.sa, bow.ObjectID(), 1)
	checkTradeMsgs(t, "after b confirmed", toA, int32(tradeMsgNoLongerAdjust))
}

// TestTradeGuards: while trading, items cannot be used, dropped or
// destroyed; a pending request blocks drops too.
func TestTradeGuards(t *testing.T) {
	e := newTradeEnv(t, func(gs *config.GameServer) { gs.Economy.FloodDropItem = 0 })
	bow := e.give(e.a, itemBow, 1)
	e.sync()
	e.sa.take(t)
	e.send(e.sa, clientpackets.OpTradeRequest, ints(e.b.ObjectID()))
	pos := e.position(e.a)
	adena := e.objectOf(e.a, itemAdena)
	toA, _ := e.send(e.sa, clientpackets.OpRequestDropItem, ints(adena, 10, pos.X, pos.Y, pos.Z))
	checkTradeMsgs(t, "drop while asking", toA, int32(tradeMsgStoreNoDiscard))
	e.send(e.sb, clientpackets.OpAnswerTradeRequest, ints(1))

	toA, _ = e.send(e.sa, clientpackets.OpUseItem, ints(bow.ObjectID(), 0))
	checkTradeMsgs(t, "use while trading", toA, int32(tradeMsgCannotUseItems))
	toA, _ = e.send(e.sa, clientpackets.OpRequestDropItem, ints(adena, 10, pos.X, pos.Y, pos.Z))
	checkTradeMsgs(t, "drop while trading", toA, int32(tradeMsgStoreNoDiscard))
	toA, _ = e.send(e.sa, clientpackets.OpRequestDestroyItem, ints(bow.ObjectID(), 1))
	checkTradeMsgs(t, "destroy while trading", toA, int32(tradeMsgStoreNoDiscard))
	if e.countOf(e.a, itemBow) != 1 || e.countOf(e.a, itemAdena) != 1000 {
		t.Error("an item changed while trading")
	}
}

// TestTradeFlood: a second TradeDone within FloodTransaction is refused.
func TestTradeFlood(t *testing.T) {
	e := newTradeEnv(t, func(gs *config.GameServer) { gs.Economy.FloodTransaction = config.Duration(time.Second) })
	e.open()
	_, toB := e.done(e.sa, true)
	checkTradeMsgs(t, "b", toB, int32(tradeMsgConfirmed))
	toA, toB := e.done(e.sa, false)
	if len(toB) != 0 || len(only[testclient.TradeDone](toA)) != 0 {
		t.Fatalf("a cancel within 1 s went through: a got %q, b got %q", testclient.Names(toA), testclient.Names(toB))
	}
	if !e.trading(e.a) {
		t.Fatal("the trade closed")
	}
	e.advance(time.Second)
	toA, _ = e.done(e.sa, false)
	checkTradeDone(t, "a", toA, 0)
}

// errTradeInjected fails a joint save for good (the saver gives up on
// storage.ErrNotFound).
var errTradeInjected = fmt.Errorf("injected joint save failure: %w", storage.ErrNotFound)

// tradeFailingBatches is a storage.BatchWriter whose writes all fail.
type tradeFailingBatches struct{}

func (tradeFailingBatches) WriteBatch(context.Context, storage.Batch) error { return errTradeInjected }

// TestTradeJointSaveFailure: when the joint save of an exchange fails for
// good, both players are disconnected without a save, and after a reload
// both inventories are as before the trade.
func TestTradeJointSaveFailure(t *testing.T) {
	e := newTradeEnv(t, nil)
	e.open()
	e.add(e.sa, e.objectOf(e.a, itemAdena), 500)
	e.add(e.sb, e.objectOf(e.b, itemLongSword), 1)
	e.done(e.sa, true)
	sv := e.srv.saver
	sv.mu.Lock()
	sv.batches = tradeFailingBatches{}
	sv.mu.Unlock()
	e.done(e.sb, true)

	for _, sess := range []*fakeSession{e.sa, e.sb} {
		sess.waitFor(t, sess.account+" disconnected", func(s *fakeSession) bool { return s.closed })
	}
	e.sync()
	if got := e.stored(e.a, itemAdena); got != 1000 {
		t.Errorf("stored a has %d adena, want 1000", got)
	}
	if got := e.stored(e.b, itemAdena); got != 0 {
		t.Errorf("stored b has %d adena, want 0", got)
	}
	if e.stored(e.a, itemLongSword) != 0 || e.stored(e.b, itemLongSword) != 1 {
		t.Error("the stored Long Sword moved")
	}

	a := e.enter(newSession("trader1"), 0)
	b := e.enter(newSession("trader2"), 0)
	if got := e.countOf(a, itemAdena); got != 1000 {
		t.Errorf("reloaded a has %d adena, want 1000", got)
	}
	if e.countOf(a, itemLongSword) != 0 || e.countOf(b, itemLongSword) != 1 || e.countOf(b, itemAdena) != 0 {
		t.Error("the reloaded inventories changed")
	}
}
