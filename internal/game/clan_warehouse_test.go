package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// clanWHItem is a carried item for clanChar.items.
func clanWHItem(itemID int32, count int64) storage.Item {
	return storage.Item{ItemID: itemID, Count: count, Slot: storage.PaperdollNone, Mana: storage.NoMana}
}

// newClanWHEnv is newClanEnv without flood protection and with a clan
// warehouse of slots slots.
func newClanWHEnv(t *testing.T, prev *testEnv, slots int, configure func(gs *config.GameServer)) *testEnv {
	t.Helper()
	return newClanEnv(t, prev, time.Time{}, func(gs *config.GameServer) {
		gs.Economy.FloodTransaction = 0
		gs.Items.WarehouseSlotsClan = slots
		if configure != nil {
			configure(gs)
		}
	})
}

// clanWHBypass makes p click a warehouse keeper's command and returns what
// p got.
func (e *testEnv) clanWHBypass(sess *fakeSession, p *Player, keeper *Npc, command string) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.onLoop(func() {
		p.LastNpc = keeper.ID()
		e.srv.npcCommand(p, keeper, command)
	})
	return sess.take(e.t)
}

// clanWHMove sends SendWareHouseDepositList (deposit) or
// SendWareHouseWithDrawList with object ID and count pairs.
func (e *testEnv) clanWHMove(sess *fakeSession, deposit bool, pairs ...int32) []testclient.Packet {
	e.t.Helper()
	var op byte = clientpackets.OpSendWareHouseWithDrawList
	if deposit {
		op = clientpackets.OpSendWareHouseDepositList
	}
	return e.clanSend(sess, op, clanInts(append([]int32{int32(len(pairs) / 2)}, pairs...)...))
}

// clanStored returns how many of the template c's warehouse holds.
func (e *testEnv) clanStored(c *Clan, itemID int32) int64 {
	var n int64
	e.onLoop(func() { n = e.srv.ClanWarehouse(c).Count(itemID, -1, true) })
	return n
}

// carriedObject returns the object ID of p's first item of the template.
func (e *testEnv) carriedObject(p *Player, itemID int32) int32 {
	e.t.Helper()
	var id int32
	e.onLoop(func() {
		if it := p.inv.ByItemID(itemID); it != nil {
			id = it.ObjectID()
		}
	})
	if id == 0 {
		e.t.Fatalf("%s carries no item %d", p.Name(), itemID)
	}
	return id
}

// storedClanItems returns the clan's items in the store after its writes.
func (e *testEnv) storedClanItems(c *Clan, ids ...int32) map[int32]storage.Item {
	e.t.Helper()
	e.flushClanWrites(append(ids, c.ID)...)
	items, err := e.store.Clans().ClanItems(e.ctx, c.ID)
	if err != nil {
		e.t.Fatalf("ClanItems: %v", err)
	}
	out := make(map[int32]storage.Item)
	for _, it := range items {
		if it.Loc != storage.LocClanWarehouse {
			e.t.Errorf("clan item %+v not in the clan warehouse", it)
		}
		out[it.ItemID] = it
	}
	return out
}

// storedItemCount is the count of the template in a stored character's
// inventory.
func storedItemCount(ch storage.Character, itemID int32) int64 {
	var n int64
	for _, it := range ch.Items {
		if it.ItemID == itemID && it.Loc != storage.LocWarehouse {
			n += it.Count
		}
	}
	return n
}

// The clan warehouse: only members of a level 1+ clan open it; the deposit
// list leaves out worn and quest items; deposits cost 30 adena per entry and
// fill at most the clan's slots; the withdrawal window needs the warehouse
// privilege, and only the leader withdraws; every move is one joint save of
// the player and the clan's items, which survive a restart.
func TestClanWarehouse(t *testing.T) {
	e := newClanWHEnv(t, nil, 3, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40, items: []storage.Item{
		clanWHItem(itemAdena, 10000), clanWHItem(itemMithrilOre, 10), clanWHItem(itemLongSword, 1),
		clanWHItem(itemLongSword, 1), clanWHItem(itemDarinsLetter, 1),
	}})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 30, items: []storage.Item{
		clanWHItem(itemAdena, 1000), clanWHItem(itemMithrilOre, 5),
	}})
	sessX, x := e.enterClanChar(clanChar{account: "xena", name: "Xena", level: 30})
	keeper := e.spawnNpc(npcWilford, eastOf(tiVillage, 60))
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, social.UnitMain, 1)

	clanWantMsg(t, "no clan", e.clanWHBypass(sessX, x, keeper, "DepositC"), int32(clanWHMsgNoRight))
	clanWantMsg(t, "clan level 0", e.clanWHBypass(sessB, b, keeper, "DepositC"), int32(clanWHMsgNeedsLevel1))
	e.onLoop(func() { e.srv.ChangeClanLevel(c, 1) })

	// The deposit window.
	ps := e.clanWHBypass(sessA, a, keeper, "DepositC")
	lists := only[testclient.WareHouseDepositList](ps)
	if len(lists) != 1 || lists[0].Type != testclient.WarehouseClan || lists[0].Adena != 10000 || len(only[testclient.ActionFailed](ps)) == 0 {
		t.Fatalf("WareHouseDepositList %+v in %q", lists, testclient.Names(ps))
	}
	var listed []int32
	for _, it := range lists[0].Items {
		listed = append(listed, it.ItemID)
	}
	if !slices.Contains(listed, itemMithrilOre) || !slices.Contains(listed, itemAdena) || slices.Contains(listed, itemDarinsLetter) || slices.Contains(listed, itemSquireSword) {
		t.Errorf("listed %v: want ore and adena, no quest or worn item", listed)
	}

	// A member deposits: 2 entries, 60 adena, one joint save.
	e.clanWHBypass(sessB, b, keeper, "DepositC")
	e.flushClanWrites(b.ObjectID(), c.ID)
	singles := e.store.saveCount(b.ObjectID())
	ps = e.clanWHMove(sessB, true, e.carriedObject(b, itemMithrilOre), 4, e.carriedObject(b, itemAdena), 100)
	if len(only[testclient.StatusUpdate](ps)) != 1 || len(sysMsgs(ps)) != 0 {
		t.Errorf("deposit: %q, messages %v", testclient.Names(ps), sysMsgs(ps))
	}
	if e.itemCount(b, itemAdena) != 840 || e.itemCount(b, itemMithrilOre) != 1 || e.clanStored(c, itemAdena) != 100 || e.clanStored(c, itemMithrilOre) != 4 {
		t.Errorf("after the deposit: carried adena %d ore %d, stored adena %d ore %d; want 840, 1, 100, 4",
			e.itemCount(b, itemAdena), e.itemCount(b, itemMithrilOre), e.clanStored(c, itemAdena), e.clanStored(c, itemMithrilOre))
	}
	stored := e.storedClanItems(c, b.ObjectID())
	if stored[itemMithrilOre].Count != 4 || stored[itemAdena].Count != 100 {
		t.Errorf("stored clan items %+v", stored)
	}
	if ch := e.storedCharacter(b.ObjectID()); storedItemCount(ch, itemMithrilOre) != 1 || storedItemCount(ch, itemAdena) != 840 {
		t.Errorf("stored Bob: ore %d adena %d", storedItemCount(ch, itemMithrilOre), storedItemCount(ch, itemAdena))
	}
	if n := e.store.saveCount(b.ObjectID()); n != singles {
		t.Errorf("the deposit was saved on its own (%d single saves, want %d): not a joint save", n, singles)
	}

	// Slots: two swords do not fit next to the ore and adena (3 slots).
	e.clanWHBypass(sessA, a, keeper, "DepositC")
	var swords []int32
	e.onLoop(func() {
		for _, it := range a.inv.AllByItemID(itemLongSword) {
			swords = append(swords, it.ObjectID())
		}
	})
	ps = e.clanWHMove(sessA, true, swords[0], 1, swords[1], 1)
	if ids := sysMsgs(ps); !slices.Equal(ids, []int32{int32(exchangeMsgQuantityExceeded)}) || e.itemCount(a, itemLongSword) != 2 || e.itemCount(a, itemAdena) != 10000 {
		t.Errorf("full clan warehouse: messages %v, swords %d, adena %d", ids, e.itemCount(a, itemLongSword), e.itemCount(a, itemAdena))
	}
	if e.clanWHMove(sessA, true, swords[0], 1); e.clanStored(c, itemLongSword) != 1 || e.itemCount(a, itemAdena) != 9970 {
		t.Errorf("one sword: stored %d, adena %d", e.clanStored(c, itemLongSword), e.itemCount(a, itemAdena))
	}

	// The withdrawal window needs the warehouse privilege.
	e.onLoop(func() { b.clan.privs = 0 })
	ps = e.clanWHBypass(sessB, b, keeper, "WithdrawC")
	if clanWantMsg(t, "no privilege", ps, int32(clanWHMsgNoRight)); len(only[testclient.WareHouseWithdrawalList](ps)) != 0 {
		t.Error("withdrawal window without the privilege")
	}
	e.onLoop(func() { b.clan.privs = 1 << social.AccessWarehouse })
	lists2 := only[testclient.WareHouseWithdrawalList](e.clanWHBypass(sessB, b, keeper, "WithdrawC"))
	if len(lists2) != 1 || lists2[0].Type != testclient.WarehouseClan || len(lists2[0].Items) != 3 || lists2[0].Adena != 840 {
		t.Fatalf("WareHouseWithdrawalList %+v", lists2)
	}
	ore, ok := lists2[0].FindItem(itemMithrilOre)
	if !ok {
		t.Fatal("no ore listed")
	}
	// Only the leader withdraws.
	ps = e.clanWHMove(sessB, false, ore.ObjectID, 1)
	if ids := sysMsgs(ps); !slices.Equal(ids, []int32{int32(clanWHMsgOnlyLeaderTake)}) || e.clanStored(c, itemMithrilOre) != 4 || e.itemCount(b, itemMithrilOre) != 1 {
		t.Errorf("member withdrawal: messages %v, stored %d, carried %d", ids, e.clanStored(c, itemMithrilOre), e.itemCount(b, itemMithrilOre))
	}
	e.clanWHBypass(sessA, a, keeper, "WithdrawC")
	e.flushClanWrites(a.ObjectID(), c.ID)
	singles = e.store.saveCount(a.ObjectID())
	e.clanWHMove(sessA, false, ore.ObjectID, 3)
	if e.clanStored(c, itemMithrilOre) != 1 || e.itemCount(a, itemMithrilOre) != 13 {
		t.Errorf("leader withdrawal: stored %d, carried %d; want 1, 13", e.clanStored(c, itemMithrilOre), e.itemCount(a, itemMithrilOre))
	}
	if stored := e.storedClanItems(c, a.ObjectID()); stored[itemMithrilOre].Count != 1 || stored[itemLongSword].Count != 1 {
		t.Errorf("stored clan items after the withdrawal %+v", stored)
	}
	if ch := e.storedCharacter(a.ObjectID()); storedItemCount(ch, itemMithrilOre) != 13 {
		t.Errorf("stored Leader ore %d, want 13", storedItemCount(ch, itemMithrilOre))
	}
	if n := e.store.saveCount(a.ObjectID()); n != singles {
		t.Errorf("the withdrawal was saved on its own (%d single saves, want %d)", n, singles)
	}
	// Opening the private warehouse ends the clan window: a deposit then
	// goes to the private warehouse.
	e.onLoop(func() {
		e.srv.bypassDepositP(a)
		if e.srv.clanWarehouseOpen(a) {
			t.Error("the clan warehouse is still open after DepositP")
		}
	})

	// After a restart the clan warehouse is back.
	e.clanLogout(sessA, a)
	e.clanLogout(sessB, b)
	e.flushClanWrites(c.ID)
	e.stop()
	e2 := newClanWHEnv(t, e, 3, nil)
	var c2 *Clan
	e2.onLoop(func() { c2 = e2.srv.Clan(c.ID) })
	if c2 == nil || e2.clanStored(c2, itemMithrilOre) != 1 || e2.clanStored(c2, itemAdena) != 100 || e2.clanStored(c2, itemLongSword) != 1 {
		t.Errorf("after the restart: clan %v", c2)
	}
}

// With clan.membersCanWithdrawFromWarehouse, members with the warehouse
// privilege withdraw; members without it are ignored.
func TestClanWarehouseMembersWithdraw(t *testing.T) {
	e := newClanWHEnv(t, nil, 200, func(gs *config.GameServer) { gs.Clan.MembersCanWithdrawFromWarehouse = true })
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40, items: []storage.Item{
		clanWHItem(itemAdena, 1000), clanWHItem(itemMithrilOre, 10),
	}})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 30})
	keeper := e.spawnNpc(npcWilford, eastOf(tiVillage, 60))
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, social.UnitMain, 1)
	e.onLoop(func() { e.srv.ChangeClanLevel(c, 1) })
	e.clanWHBypass(sessA, a, keeper, "DepositC")
	e.clanWHMove(sessA, true, e.carriedObject(a, itemMithrilOre), 10)

	e.onLoop(func() { b.clan.privs = 1 << social.AccessWarehouse })
	lists := only[testclient.WareHouseWithdrawalList](e.clanWHBypass(sessB, b, keeper, "WithdrawC"))
	if len(lists) != 1 {
		t.Fatal("no withdrawal window")
	}
	ore, _ := lists[0].FindItem(itemMithrilOre)
	if e.clanWHMove(sessB, false, ore.ObjectID, 4); e.itemCount(b, itemMithrilOre) != 4 || e.clanStored(c, itemMithrilOre) != 6 {
		t.Errorf("member with the privilege: carried %d, stored %d", e.itemCount(b, itemMithrilOre), e.clanStored(c, itemMithrilOre))
	}
	e.onLoop(func() { b.clan.privs = 0 })
	ps := e.clanWHMove(sessB, false, ore.ObjectID, 1)
	if len(ps) != 0 || e.itemCount(b, itemMithrilOre) != 4 {
		t.Errorf("member without the privilege: %q, carried %d", testclient.Names(ps), e.itemCount(b, itemMithrilOre))
	}
	// A member who left the clan can no longer use its warehouse.
	e.onLoop(func() { b.clan.privs = 1 << social.AccessWarehouse })
	e.clanWHBypass(sessB, b, keeper, "WithdrawC")
	e.clanSend(sessB, clientpackets.OpRequestWithdrawalPledge, nil)
	if e.clanWHMove(sessB, false, ore.ObjectID, 1); e.itemCount(b, itemMithrilOre) != 4 {
		t.Errorf("ex-member withdrew: carried %d", e.itemCount(b, itemMithrilOre))
	}
}
