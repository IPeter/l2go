package game

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The henna tests: Plan 14 M11c's demo and rules in a game server, with
// the real dye data and the real Symbol Maker script.

// npcMarsden is a Symbol Maker. classWarrior (first class) and
// classGladiator (second class) are in quest_test.go and
// clan_academy_test.go.
const npcMarsden = 31046

// Dyes of the tests and their dye items. The first-class dyes: Dye of STR
// (Str+1 Con-3, 10 dyes and 37,000 adena), Str+1 Dex-3 and Str-3 Con+1.
// The second-class dyes: Str+1 Con-3 (10 dyes, 87,000 adena), Str+1 Dex-3,
// Str-3 Con+1 and Con+1 Dex-3.
const (
	dyeSTR    = 1
	dyeSTRDEX = 2
	dyeCONSTR = 3

	itemDyeSTR = 4445
	itemDyeDEX = 4446
	itemDyeCON = 4447

	dye2STR    = 37
	dye2STRDEX = 38
	dye2CONSTR = 39
	dye2CONDEX = 40

	itemDye2STR  = 4481
	itemDye2DEX  = 4482
	itemDye2CON  = 4483
	itemDye2CON2 = 4484
)

var loadTestHennas = sync.OnceValues(func() (*data.Hennas, error) {
	return data.LoadHennas(filepath.Join("..", "..", "data"))
})

// newHennaEnv starts a server with the dye data, the Symbol Maker script and
// its pages, and no transaction flood limit.
func newHennaEnv(t *testing.T) *itemEnv {
	t.Helper()
	scripts := map[string]string{}
	for _, name := range []string{"symbol_maker.lua", "symbol_maker.htm", "symbol_maker-1.htm", "symbol_maker-2.htm", "symbol_maker-3.htm"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "scripts", "npcs", "symbol_maker", name))
		if err != nil {
			t.Fatal(err)
		}
		scripts["npcs/symbol_maker/"+name] = string(b)
	}
	e := &itemEnv{testEnv: newDialogEnv(t, dialogEnvOptions{
		scripts:   scripts,
		configure: func(gs *config.GameServer) { gs.Economy.FloodTransaction = 0 },
	})}
	table, err := loadTestHennas()
	if err != nil {
		t.Fatalf("load the hennas: %v", err)
	}
	e.onLoop(func() { e.srv.henna.table = table })
	return e
}

// carried is a carried stack for enterWith.
func carried(itemID int32, count int64) storage.Item {
	return storage.Item{ItemID: itemID, Count: count, Slot: storage.PaperdollNone, Mana: storage.NoMana}
}

// marsden spawns Marsden next to p and talks to him: his page shows and he
// is p's last NPC.
func (e *itemEnv) marsden(sess *fakeSession, p *Player) *Npc {
	e.t.Helper()
	n := e.spawnNpc(npcMarsden, eastOf(e.position(p), 50))
	ps := e.talk(sess, p, n)
	if pg := onePage(e.t, ps); pg.ObjectID != int32(n.ID()) {
		e.t.Fatalf("Marsden's page %+v", pg)
	}
	return n
}

// enterHenna enters a Human Fighter at tiVillage with the items and lets
// the delayed enter world packets pass.
func (e *itemEnv) enterHenna(account string, items ...storage.Item) (*fakeSession, *Player) {
	e.t.Helper()
	sess, p := e.enterWith(account, tiVillage, items...)
	e.advance(time.Second)
	sess.take(e.t)
	return sess, p
}

// changeClass changes p's class and returns what p got, the delayed skill
// lists included.
func (e *itemEnv) changeClass(sess *fakeSession, p *Player, classID int) []testclient.Packet {
	e.t.Helper()
	e.onLoop(func() { e.srv.ChangeClass(p, classID) })
	ps := sess.take(e.t)
	for range 10 {
		e.advance(world.DefaultTickInterval)
		ps = append(ps, sess.take(e.t)...)
	}
	return ps
}

// henna sends a henna packet with one int and returns what p got by the
// end of the next tick: the UserInfo and the item updates come then, and
// the save is handed to the saver.
func (e *itemEnv) henna(sess *fakeSession, op byte, id int32) []testclient.Packet {
	e.t.Helper()
	ps := e.handle(sess, op, ints(id))
	e.advance(world.DefaultTickInterval)
	return append(ps, sess.take(e.t)...)
}

// baseStats returns p's STR, CON and DEX, read on the loop.
func (e *itemEnv) baseStats(p *Player) (str, con, dex int32) {
	e.onLoop(func() {
		v := p.stats.View()
		str, con, dex = v.STR, v.CON, v.DEX
	})
	return str, con, dex
}

func hennaIDs(h testclient.HennaInfo) []int32 {
	var ids []int32
	for _, x := range h.Hennas {
		ids = append(ids, x.DyeID)
	}
	return ids
}

// Demo 1 and 2: a first-class character draws the Dye of STR at a Symbol
// Maker (10 dyes, 37,000 adena; STR +1, CON -3) and removes it (5 dyes
// back for 7,400 adena). The windows open from the script's links.
func TestHennaDrawAndRemove(t *testing.T) {
	e := newHennaEnv(t)
	sess, p := e.enterHenna("dyer", carried(adenaID, 100000), carried(itemDyeSTR, 20))
	e.marsden(sess, p)

	// A base class has no slot and may wear no dye.
	l := shopOne[testclient.HennaEquipList](t, "Draw as a fighter", e.bypass(sess, "Script SymbolMaker Draw"))
	if l.Listed != 0 || len(l.Entries) != 0 || l.Adena != 100000 || l.Slots != 3 {
		t.Errorf("fighter's HennaEquipList %+v", l)
	}
	ps := e.henna(sess, clientpackets.OpRequestHennaEquip, dyeSTR)
	if got := sysMsgIDsOf(ps); !slices.Equal(got, []int32{900}) || len(only[testclient.HennaInfo](ps)) != 0 {
		t.Errorf("draw as a fighter: %q, messages %v; want 900", testclient.Names(ps), got)
	}
	checkNames(t, "draw as a fighter", ps, "SystemMessage", "ActionFailed")

	e.changeClass(sess, p, classWarrior)
	str, con, dex := e.baseStats(p)
	l = shopOne[testclient.HennaEquipList](t, "Draw", e.bypass(sess, "Script SymbolMaker Draw"))
	if want := int32(len(e.srv.henna.table.ForClass(classWarrior))); l.Listed != want || l.Listed < 2 {
		t.Errorf("Listed = %d, want %d", l.Listed, want)
	}
	// Only the dyes whose item the player carries are sent.
	if !slices.Equal(l.Entries, []testclient.HennaEntry{{DyeID: dyeSTR, DyeItemID: itemDyeSTR, Count: 10, Fee: 37000, Allowed: 1}}) {
		t.Errorf("entries %+v", l.Entries)
	}

	info := shopOne[testclient.HennaItemDrawInfo](t, "draw info", e.henna(sess, clientpackets.OpRequestHennaItemInfo, dyeSTR))
	if info.DyeID != dyeSTR || info.Count != 10 || info.Fee != 37000 || !info.Allowed || info.Adena != 100000 ||
		info.STR != (testclient.HennaStat{Now: str, After: uint8(str + 1)}) || info.CON != (testclient.HennaStat{Now: con, After: uint8(con - 3)}) {
		t.Errorf("HennaItemDrawInfo %+v (STR %d, CON %d)", info, str, con)
	}

	ps = e.henna(sess, clientpackets.OpRequestHennaEquip, dyeSTR)
	hi := shopOne[testclient.HennaInfo](t, "draw", ps)
	if hi.STR != 1 || hi.CON != -3 || hi.DEX != 0 || hi.Slots != 3 || !slices.Equal(hennaIDs(hi), []int32{dyeSTR}) {
		t.Errorf("HennaInfo %+v", hi)
	}
	if got := sysMsgIDsOf(ps); !slices.Equal(got, []int32{301, 877}) {
		t.Errorf("draw messages %v, want 301 (10 dyes disappeared), 877", got)
	}
	ui := shopOne[testclient.UserInfo](t, "draw", ps)
	if ui.STR != str+1 || ui.CON != con-3 || ui.DEX != dex {
		t.Errorf("UserInfo STR %d CON %d DEX %d; want %d %d %d", ui.STR, ui.CON, ui.DEX, str+1, con-3, dex)
	}
	if got := e.countOf(p, itemDyeSTR); got != 10 {
		t.Errorf("dyes left %d, want 10", got)
	}
	if got := e.shopAdena(p); got != 63000 {
		t.Errorf("adena %d, want 63000", got)
	}
	e.waitHennaSave(p, map[int32]storage.HennaSlots{0: {dyeSTR, 0, 0}})

	rl := shopOne[testclient.HennaRemoveList](t, "Remove", e.bypass(sess, "Script SymbolMaker Remove"))
	if rl.Adena != 63000 || rl.EmptySlots != 1 || rl.Used != 2 ||
		!slices.Equal(rl.Entries, []testclient.HennaEntry{{DyeID: dyeSTR, DyeItemID: itemDyeSTR, Count: 5, Fee: 7400, Allowed: 1}}) {
		t.Errorf("HennaRemoveList %+v", rl)
	}
	ri := shopOne[testclient.HennaItemRemoveInfo](t, "remove info", e.henna(sess, clientpackets.OpRequestHennaItemRemoveInfo, dyeSTR))
	if ri.Count != 5 || ri.Fee != 7400 || !ri.Allowed ||
		ri.STR != (testclient.HennaStat{Now: str + 1, After: uint8(str)}) || ri.CON != (testclient.HennaStat{Now: con - 3, After: uint8(con)}) {
		t.Errorf("HennaItemRemoveInfo %+v", ri)
	}

	ps = e.henna(sess, clientpackets.OpRequestHennaRemove, dyeSTR)
	hi = shopOne[testclient.HennaInfo](t, "remove", ps)
	if hi.STR != 0 || hi.CON != 0 || len(hi.Hennas) != 0 {
		t.Errorf("HennaInfo after removing %+v", hi)
	}
	if got := sysMsgIDsOf(ps); !slices.Equal(got, []int32{53, 878}) {
		t.Errorf("remove messages %v, want 53 (5 dyes earned), 878", got)
	}
	if s2, c2, _ := e.baseStats(p); s2 != str || c2 != con {
		t.Errorf("STR %d CON %d after removing, want %d %d", s2, c2, str, con)
	}
	if got := e.countOf(p, itemDyeSTR); got != 15 {
		t.Errorf("dyes %d, want 15", got)
	}
	if got := e.shopAdena(p); got != 55600 {
		t.Errorf("adena %d, want 55600", got)
	}
	e.waitHennaSave(p, nil)
}

// Demo 3 and the class change: a first class has two slots, a second class
// three; a class change takes off the dyes the new class may not wear,
// with the usual refund and fee.
func TestHennaSlotsByClass(t *testing.T) {
	e := newHennaEnv(t)
	sess, p := e.enterHenna("slots", carried(adenaID, 1_000_000),
		carried(itemDyeSTR, 10), carried(itemDyeDEX, 10), carried(itemDyeCON, 10),
		carried(itemDye2STR, 10), carried(itemDye2DEX, 10), carried(itemDye2CON, 10), carried(itemDye2CON2, 10))
	e.changeClass(sess, p, classWarrior)
	e.marsden(sess, p)

	for _, id := range []int32{dyeSTR, dyeSTRDEX} {
		if ps := e.henna(sess, clientpackets.OpRequestHennaEquip, id); !slices.Contains(sysMsgIDsOf(ps), 877) {
			t.Fatalf("draw %d: %q", id, testclient.Names(ps))
		}
	}
	ps := e.henna(sess, clientpackets.OpRequestHennaEquip, dyeCONSTR)
	if got := sysMsgIDsOf(ps); !slices.Equal(got, []int32{900}) {
		t.Errorf("a third dye on a first class: messages %v, want 900", got)
	}
	if got := e.countOf(p, itemDyeCON); got != 10 {
		t.Errorf("the refused dye's items %d, want 10", got)
	}

	// Gladiator: the first-class dyes come off (5 dyes back, 7,400 adena
	// each); then three second-class dyes fit, a fourth does not.
	adena := e.shopAdena(p)
	ps = e.changeClass(sess, p, classGladiator)
	infos := only[testclient.HennaInfo](ps)
	if len(infos) != 2 || len(infos[1].Hennas) != 0 {
		t.Errorf("class change HennaInfos %+v", infos)
	}
	if got := sysMsgIDsOf(ps); !slices.Equal(got, []int32{1308, 53, 878, 53, 878}) {
		t.Errorf("class change messages %v", got)
	}
	if got := e.countOf(p, itemDyeSTR) + e.countOf(p, itemDyeDEX); got != 10 {
		t.Errorf("dyes back %d, want 5 + 5", got)
	}
	if got := e.shopAdena(p); got != adena-2*7400 {
		t.Errorf("adena %d, want %d", got, adena-2*7400)
	}

	for _, id := range []int32{dye2STR, dye2STRDEX, dye2CONSTR} {
		if ps := e.henna(sess, clientpackets.OpRequestHennaEquip, id); !slices.Contains(sysMsgIDsOf(ps), 877) {
			t.Fatalf("draw %d as a Gladiator: %q %v", id, testclient.Names(ps), sysMsgIDsOf(ps))
		}
	}
	ps = e.henna(sess, clientpackets.OpRequestHennaEquip, dye2CONDEX)
	if got := sysMsgIDsOf(ps); !slices.Equal(got, []int32{900}) {
		t.Errorf("a fourth dye: messages %v, want 900", got)
	}
	var worn [3]int32
	var str, con, dex int
	e.onLoop(func() {
		worn = p.henna.worn.IDs()
		b := p.henna.worn.Bonus()
		str, con, dex = b.STR, b.CON, b.DEX
	})
	// 37 Str+1 Con-3, 38 Str+1 Dex-3, 39 Str-3 Con+1.
	if worn != [3]int32{dye2STR, dye2STRDEX, dye2CONSTR} || str != -1 || con != -2 || dex != -3 {
		t.Errorf("worn %v, bonus STR %d CON %d DEX %d", worn, str, con, dex)
	}
	rl := shopOne[testclient.HennaRemoveList](t, "Remove", e.bypass(sess, "Script SymbolMaker Remove"))
	if rl.EmptySlots != 0 || rl.Used != 3 || len(rl.Entries) != 3 || rl.Entries[0].Fee != 17400 {
		t.Errorf("HennaRemoveList %+v", rl)
	}
}

// Refusals: too few dyes or adena, a dye the class may not wear (an exploit
// in Mobius: the player is kicked), an unknown dye, and removing without
// the fee or a dye not worn.
func TestHennaRefused(t *testing.T) {
	e := newHennaEnv(t)
	sess, p := e.enterHenna("refused", carried(adenaID, 40000), carried(itemDyeSTR, 9), carried(itemDye2STR, 10))
	e.changeClass(sess, p, classWarrior)
	e.marsden(sess, p)

	ps := e.henna(sess, clientpackets.OpRequestHennaEquip, dyeSTR)
	if got := sysMsgIDsOf(ps); !slices.Equal(got, []int32{899}) || len(only[testclient.HennaInfo](ps)) != 0 {
		t.Errorf("9 dyes: %q %v; want 899", testclient.Names(ps), got)
	}
	checkNames(t, "9 dyes", ps, "SystemMessage", "ActionFailed")

	e.give(p, itemDyeSTR, 1)
	e.onLoop(func() { e.srv.ReduceAdena(p, 3001, false) }) // 36,999 left
	sess.take(t)
	if got := sysMsgIDsOf(e.henna(sess, clientpackets.OpRequestHennaEquip, dyeSTR)); !slices.Equal(got, []int32{899}) {
		t.Errorf("36,999 adena: messages %v; want 899", got)
	}

	ps = e.henna(sess, clientpackets.OpRequestHennaEquip, 99999)
	checkNames(t, "unknown dye", ps, "ActionFailed")
	checkNames(t, "unknown dye info", e.henna(sess, clientpackets.OpRequestHennaItemInfo, 99999), "ActionFailed")
	checkNames(t, "dye 0 info", e.henna(sess, clientpackets.OpRequestHennaItemInfo, 0), "ActionFailed")
	checkNames(t, "dye 0 remove info", e.henna(sess, clientpackets.OpRequestHennaItemRemoveInfo, 0))
	checkNames(t, "not worn", e.henna(sess, clientpackets.OpRequestHennaRemove, dyeSTR), "ActionFailed")

	// Enough adena now: draw, then fail to pay the removal.
	e.onLoop(func() { e.srv.AddAdena(p, 1, false) })
	if ps := e.henna(sess, clientpackets.OpRequestHennaEquip, dyeSTR); !slices.Contains(sysMsgIDsOf(ps), 877) {
		t.Fatalf("draw: %q", testclient.Names(ps))
	}
	ps = e.henna(sess, clientpackets.OpRequestHennaRemove, dyeSTR)
	if got := sysMsgIDsOf(ps); !slices.Equal(got, []int32{279}) || len(only[testclient.HennaInfo](ps)) != 0 {
		t.Errorf("remove without the fee: %q %v; want 279", testclient.Names(ps), got)
	}

	// A second-class dye on a Warrior: refused and punished.
	ps = e.henna(sess, clientpackets.OpRequestHennaEquip, dye2STR)
	if got := sysMsgIDsOf(ps); !slices.Contains(got, 899) || len(only[testclient.HennaInfo](ps)) != 0 {
		t.Errorf("forbidden dye: %q, messages %v; want 899", testclient.Names(ps), got)
	}
	e.advance(illegalActionKickDelay)
	if _, _, closed := sess.closedWith(t); !closed {
		t.Error("the player who asked for a forbidden dye was not kicked")
	}
}

// Decision X10: drawing and removing need the last NPC to be a Symbol Maker
// within 250 (3D); GMs need none.
func TestHennaNeedsSymbolMaker(t *testing.T) {
	e := newHennaEnv(t)
	sess, p := e.enterHenna("far", carried(adenaID, 200000), carried(itemDyeSTR, 30))
	e.changeClass(sess, p, classWarrior)

	refused := func(what string) {
		t.Helper()
		ps := e.henna(sess, clientpackets.OpRequestHennaEquip, dyeSTR)
		checkNames(t, what, ps, "ActionFailed")
	}
	refused("no NPC")

	lector := e.spawnNpc(npcLector, eastOf(tiVillage, -60))
	e.talk(sess, p, lector)
	refused("a merchant")

	m := e.marsden(sess, p)
	var at world.Position
	e.onLoop(func() { at = m.Position() })
	e.shopMove(p, eastOf(at, 251))
	sess.take(t)
	refused("251 away")

	e.shopMove(p, eastOf(at, 240))
	sess.take(t)
	if ps := e.henna(sess, clientpackets.OpRequestHennaEquip, dyeSTR); !slices.Contains(sysMsgIDsOf(ps), 877) {
		t.Fatalf("240 away: %q", testclient.Names(ps))
	}
	e.shopMove(p, eastOf(at, 400))
	sess.take(t)
	checkNames(t, "remove far away", e.henna(sess, clientpackets.OpRequestHennaRemove, dyeSTR), "ActionFailed")
	// The windows themselves need no NPC, as in Mobius.
	shopOne[testclient.HennaEquipList](t, "list far away", e.henna(sess, clientpackets.OpRequestHennaItemList, 0))
	shopOne[testclient.HennaRemoveList](t, "remove list far away", e.henna(sess, clientpackets.OpRequestHennaRemoveList, 0))

	// A GM draws anywhere.
	gs, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart,
		items: []storage.Item{carried(itemDyeSTR, 10)}, adena: 50000})
	e.changeClass(gs, gm, classWarrior)
	if ps := e.henna(gs, clientpackets.OpRequestHennaEquip, dyeSTR); !slices.Contains(sysMsgIDsOf(ps), 877) {
		t.Errorf("GM: %q", testclient.Names(ps))
	}
}

// The dyes are stored per class index: the enter world HennaInfo shows the
// base class's, with their stats in CharSelected and UserInfo; the other
// class indexes are kept as stored, and SwapHennas puts them on.
func TestHennaEnterWorldAndClassIndex(t *testing.T) {
	e := newHennaEnv(t)
	ch := e.createCharacter("dyed", "Dyed", classHumanFighter, storage.AccessLevelPlayer)
	ch.ClassID = classWarrior
	ch.X, ch.Y, ch.Z = tiVillage.X, tiVillage.Y, tiVillage.Z
	ch.Hennas = map[int32]storage.HennaSlots{
		0: {dyeSTR, 0, dyeSTRDEX}, // Str+2, Con-3, Dex-3
		1: {dye2CONSTR, 0, 0},     // a subclass's: Str-3, Con+1
	}
	if err := e.store.Update(e.ctx, &ch); err != nil {
		t.Fatal(err)
	}

	sess := newSession("dyed")
	p := e.enter(sess, 0)
	ps := sess.take(t)
	hi := shopOne[testclient.HennaInfo](t, "enter world", ps)
	if hi.STR != 2 || hi.CON != -3 || hi.DEX != -3 || !slices.Equal(hennaIDs(hi), []int32{dyeSTR, dyeSTRDEX}) {
		t.Errorf("enter world HennaInfo %+v", hi)
	}
	// Class index 2 has no dyes: the base stats.
	e.onLoop(func() { e.srv.SwapHennas(p, 2) })
	baseSTR, baseCON, baseDEX := e.baseStats(p)
	cs := shopOne[testclient.CharSelected](t, "select", ps)
	if cs.STR != baseSTR+2 || cs.CON != baseCON-3 || cs.DEX != baseDEX-3 {
		t.Errorf("CharSelected STR %d CON %d DEX %d; want %d %d %d", cs.STR, cs.CON, cs.DEX, baseSTR+2, baseCON-3, baseDEX-3)
	}
	for _, ui := range only[testclient.UserInfo](ps) {
		if ui.STR != baseSTR+2 || ui.CON != baseCON-3 {
			t.Errorf("UserInfo STR %d CON %d", ui.STR, ui.CON)
		}
	}

	// A subclass switch puts on the dyes of its class index.
	sess.take(t)
	e.onLoop(func() { e.srv.SwapHennas(p, 1) })
	hi = shopOne[testclient.HennaInfo](t, "swap", sess.take(t))
	if hi.STR != -3 || hi.CON != 1 || !slices.Equal(hennaIDs(hi), []int32{dye2CONSTR}) {
		t.Errorf("swapped HennaInfo %+v", hi)
	}
	if s, c, _ := e.baseStats(p); s != baseSTR-3 || c != baseCON+1 {
		t.Errorf("swapped STR %d CON %d", s, c)
	}
	e.onLoop(func() { e.srv.SwapHennas(p, 0) })
	sess.take(t)

	// A save keeps every class index; a change touches only the worn one.
	e.marsden(sess, p)
	e.onLoop(func() { e.srv.Save(p, nil) })
	sess.take(t)
	e.waitHennaSave(p, map[int32]storage.HennaSlots{0: {dyeSTR, 0, dyeSTRDEX}, 1: {dye2CONSTR, 0, 0}})
	e.give(p, adenaID, 10000)
	ps = e.henna(sess, clientpackets.OpRequestHennaRemove, dyeSTRDEX)
	if !slices.Contains(sysMsgIDsOf(ps), 878) {
		t.Fatalf("remove: %q", testclient.Names(ps))
	}
	e.waitHennaSave(p, map[int32]storage.HennaSlots{0: {dyeSTR, 0, 0}, 1: {dye2CONSTR, 0, 0}})
}

// waitHennaSave waits for a save of p that stores the dyes want; the saves
// before it (asked for by earlier steps) are skipped.
func (e *itemEnv) waitHennaSave(p *Player, want map[int32]storage.HennaSlots) {
	e.t.Helper()
	var got map[int32]storage.HennaSlots
	for range 20 {
		if got = e.store.waitSave(e.t, p.ObjectID()).Hennas; maps.Equal(got, want) {
			return
		}
	}
	e.t.Errorf("saved hennas %v, want %v", got, want)
}
