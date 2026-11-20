package game

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// deposit sends SendWareHouseDepositList with object ID and count pairs.
func (e *itemEnv) deposit(sess *fakeSession, pairs ...int32) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	return e.handle(sess, clientpackets.OpSendWareHouseDepositList, ints(append([]int32{int32(len(pairs) / 2)}, pairs...)...))
}

// withdraw sends SendWareHouseWithDrawList with object ID and count pairs.
func (e *itemEnv) withdraw(sess *fakeSession, pairs ...int32) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	return e.handle(sess, clientpackets.OpSendWareHouseWithDrawList, ints(append([]int32{int32(len(pairs) / 2)}, pairs...)...))
}

// storedCount returns how many of the template p's warehouse holds.
func (e *itemEnv) storedCount(p *Player, itemID int32) int64 {
	var n int64
	e.onLoop(func() { n = p.Warehouse().Count(itemID, -1, true) })
	return n
}

// exchangeObject returns the object ID of the first item of the template p
// carries.
func (e *itemEnv) exchangeObject(p *Player, itemID int32) int32 {
	e.t.Helper()
	it := e.itemOf(p, itemID)
	if it == nil {
		e.t.Fatalf("no item %d", itemID)
	}
	return it.ObjectID()
}

// TestWarehouseDeposit: DepositP lists what may be stored; a deposit costs
// 30 adena per entry, from the adena not being stored; a full warehouse, too
// little adena or the wrong NPC store nothing.
func TestWarehouseDeposit(t *testing.T) {
	e := newExchangeEnv(t, func(gs *config.GameServer) { gs.Items.WarehouseSlotsNoDwarf = 3 })
	keeper := e.spawnNpc(npcWilford, eastOf(tiVillage, 60))
	lector := e.spawnNpc(npcLector, eastOf(tiVillage, -60))
	sess, p := e.enterWith("keeper", tiVillage,
		storage.Item{ItemID: itemAdena, Count: 1000},
		storage.Item{ItemID: itemMithrilOre, Count: 10},
		storage.Item{ItemID: itemDarinsLetter, Count: 1},
		storage.Item{ItemID: itemLongSword, Count: 1},
		storage.Item{ItemID: itemLongSword, Count: 1},
		storage.Item{ItemID: itemLongSword, Count: 1})

	ps := e.exchangeBypass(sess, p, keeper, "DepositP")
	lists := only[testclient.WareHouseDepositList](ps)
	if len(lists) != 1 || lists[0].Type != testclient.WarehousePrivate || lists[0].Adena != 1000 {
		t.Fatalf("WareHouseDepositList %+v in %q", lists, testclient.Names(ps))
	}
	var listed []int32
	for _, it := range lists[0].Items {
		listed = append(listed, it.ItemID)
	}
	for _, id := range []int32{itemAdena, itemMithrilOre, itemLongSword} {
		if !slices.Contains(listed, id) {
			t.Errorf("listed %v, want item %d", listed, id)
		}
	}
	for _, id := range []int32{itemSquireSword, itemSquireShirt, itemSquirePants, itemDarinsLetter} {
		if slices.Contains(listed, id) {
			t.Errorf("listed %v: a worn or quest item %d", listed, id)
		}
	}
	if len(only[testclient.ActionFailed](ps)) == 0 {
		t.Error("no ActionFailed")
	}

	adena, ore := e.exchangeObject(p, itemAdena), e.exchangeObject(p, itemMithrilOre)
	ps = e.deposit(sess, ore, 4, adena, 100)
	if len(only[testclient.StatusUpdate](ps)) != 1 || len(sysMsgIDsOf(ps)) != 0 {
		t.Errorf("deposit: %q, messages %v", testclient.Names(ps), sysMsgIDsOf(ps))
	}
	if e.countOf(p, itemAdena) != 840 || e.countOf(p, itemMithrilOre) != 6 || e.storedCount(p, itemAdena) != 100 || e.storedCount(p, itemMithrilOre) != 4 {
		t.Errorf("after the deposit: adena %d, ore %d; stored adena %d, ore %d; want 840, 6, 100, 4",
			e.countOf(p, itemAdena), e.countOf(p, itemMithrilOre), e.storedCount(p, itemAdena), e.storedCount(p, itemMithrilOre))
	}
	e.advance(inventoryUpdateDelay)
	if ius := only[testclient.InventoryUpdate](sess.take(t)); len(ius) != 1 || len(ius[0].Items) != 2 {
		t.Errorf("InventoryUpdate %+v, want the adena and ore changed", ius)
	}

	// Two swords do not fit next to the adena and ore (3 slots).
	var swords []int32
	e.onLoop(func() {
		for _, it := range p.inv.AllByItemID(itemLongSword) {
			swords = append(swords, it.ObjectID())
		}
	})
	ps = e.deposit(sess, swords[0], 1, swords[1], 1)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(exchangeMsgQuantityExceeded)}) || e.countOf(p, itemLongSword) != 3 || e.countOf(p, itemAdena) != 840 {
		t.Errorf("full warehouse: messages %v, swords %d, adena %d", ids, e.countOf(p, itemLongSword), e.countOf(p, itemAdena))
	}
	// Ore stacks onto the stored ore: no new slot.
	if e.deposit(sess, ore, 1); e.storedCount(p, itemMithrilOre) != 5 || e.countOf(p, itemAdena) != 810 {
		t.Errorf("ore onto the stack: stored %d, adena %d", e.storedCount(p, itemMithrilOre), e.countOf(p, itemAdena))
	}

	// Not enough adena besides the adena being stored.
	ps = e.deposit(sess, adena, 790, ore, 1)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgNotEnoughAdena)}) || e.countOf(p, itemAdena) != 810 || e.countOf(p, itemMithrilOre) != 5 {
		t.Errorf("short of the fee: messages %v, adena %d, ore %d", ids, e.countOf(p, itemAdena), e.countOf(p, itemMithrilOre))
	}

	// The last NPC is no warehouse keeper.
	e.onLoop(func() { p.LastNpc = lector.ID() })
	ps = e.deposit(sess, ore, 1)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(warehouseMsgTooFar)}) || e.countOf(p, itemMithrilOre) != 5 {
		t.Errorf("at a merchant: messages %v, ore %d", ids, e.countOf(p, itemMithrilOre))
	}
	// A worn item is skipped and costs nothing.
	e.onLoop(func() { p.LastNpc = keeper.ID() })
	worn := e.exchangeObject(p, itemSquireSword)
	if e.deposit(sess, worn, 1); e.countOf(p, itemSquireSword) != 1 || e.countOf(p, itemAdena) != 810 {
		t.Errorf("worn sword: carried %d, adena %d", e.countOf(p, itemSquireSword), e.countOf(p, itemAdena))
	}
}

// TestWarehouseWithdraw: WithdrawP with nothing stored says so; then the
// list, a withdrawal onto a stack, the weight check and a forged item.
func TestWarehouseWithdraw(t *testing.T) {
	e := newExchangeEnv(t, nil)
	keeper := e.spawnNpc(npcWilford, eastOf(tiVillage, 60))
	sess, p := e.enterWith("keeper", tiVillage,
		storage.Item{ItemID: itemAdena, Count: 1000},
		storage.Item{ItemID: itemMithrilOre, Count: 10})

	ps := e.exchangeBypass(sess, p, keeper, "WithdrawP")
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(warehouseMsgNothingDeposited)}) || len(only[testclient.WareHouseWithdrawalList](ps)) != 0 {
		t.Errorf("empty warehouse: %q, messages %v", testclient.Names(ps), ids)
	}

	e.exchangeBypass(sess, p, keeper, "DepositP")
	e.deposit(sess, e.exchangeObject(p, itemMithrilOre), 10)
	lists := only[testclient.WareHouseWithdrawalList](e.exchangeBypass(sess, p, keeper, "WithdrawP"))
	if len(lists) != 1 || len(lists[0].Items) != 1 || lists[0].Items[0].ItemID != itemMithrilOre || lists[0].Items[0].Count != 10 || lists[0].Adena != 970 {
		t.Fatalf("WareHouseWithdrawalList %+v", lists)
	}
	stored := lists[0].Items[0].ObjectID

	if e.withdraw(sess, stored, 4); e.countOf(p, itemMithrilOre) != 4 || e.storedCount(p, itemMithrilOre) != 6 || e.countOf(p, itemAdena) != 970 {
		t.Errorf("withdrawal: carried %d, stored %d, adena %d; want 4, 6, 970 (free)", e.countOf(p, itemMithrilOre), e.storedCount(p, itemMithrilOre), e.countOf(p, itemAdena))
	}
	if e.withdraw(sess, stored, 2); e.countOf(p, itemMithrilOre) != 6 {
		t.Errorf("onto the carried stack: %d ore", e.countOf(p, itemMithrilOre))
	}

	// Weight: fill up to the maximum, then the stored ore does not fit.
	var maxLoad, load int64
	e.onLoop(func() { maxLoad, load = int64(p.stats.WeightLimit()), p.inv.Weight() })
	e.give(p, itemMithrilOre, (maxLoad-load)/2)
	ps = e.withdraw(sess, stored, 4)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(exchangeMsgWeightLimit)}) || e.storedCount(p, itemMithrilOre) != 4 {
		t.Errorf("over the weight limit: messages %v, stored %d", ids, e.storedCount(p, itemMithrilOre))
	}

	// More than is stored is a forged request.
	ps = e.withdraw(sess, stored, 5)
	if texts := exchangeTexts(ps); len(texts) != 1 || e.storedCount(p, itemMithrilOre) != 4 {
		t.Errorf("forged withdrawal: texts %q, stored %d", texts, e.storedCount(p, itemMithrilOre))
	}
}

// TestWarehouseRelog: stored items come back after a relog, saved through
// the item diff with the warehouse location.
func TestWarehouseRelog(t *testing.T) {
	e := newExchangeEnv(t, nil)
	keeper := e.spawnNpc(npcWilford, eastOf(tiVillage, 60))
	sess, p := e.enterWith("keeper", tiVillage,
		storage.Item{ItemID: itemAdena, Count: 1000},
		storage.Item{ItemID: itemMithrilOre, Count: 10},
		storage.Item{ItemID: itemLongSword, Count: 1, Enchant: 2})
	e.exchangeBypass(sess, p, keeper, "DepositP")
	sword := e.exchangeObject(p, itemLongSword)
	e.deposit(sess, e.exchangeObject(p, itemMithrilOre), 3, sword, 1)

	e.srv.Logout(sess)
	if err := e.srv.saver.wait(e.ctx, p.ObjectID()); err != nil {
		t.Fatalf("wait: %v", err)
	}
	e.store.mu.Lock()
	var saved []storage.Item
	for _, ch := range e.store.changes {
		saved = append(saved, ch.Upserts...)
	}
	e.store.mu.Unlock()
	if !slices.ContainsFunc(saved, func(it storage.Item) bool {
		return it.ObjectID == sword && it.Loc == storage.LocWarehouse && it.Enchant == 2
	}) {
		t.Errorf("saved upserts %+v, want the sword in the warehouse", saved)
	}

	_, p2 := e.enterWithSession(newSession("keeper"))
	if e.storedCount(p2, itemMithrilOre) != 3 || e.storedCount(p2, itemLongSword) != 1 || e.countOf(p2, itemMithrilOre) != 7 || e.countOf(p2, itemAdena) != 940 {
		t.Errorf("after a relog: stored ore %d, sword %d; carried ore %d, adena %d",
			e.storedCount(p2, itemMithrilOre), e.storedCount(p2, itemLongSword), e.countOf(p2, itemMithrilOre), e.countOf(p2, itemAdena))
	}
	var enchant int32
	e.onLoop(func() {
		if it := p2.Warehouse().ByObjectID(sword); it != nil {
			enchant = it.Enchant()
		}
	})
	if enchant != 2 {
		t.Errorf("the stored sword is +%d, want +2 with its object ID", enchant)
	}
}
