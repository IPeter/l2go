package game

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// payingEnv makes every level pay for teleports.
func payingEnv(gs *config.GameServer) { gs.NPCs.MaxFreeTeleportLevel = 0 }

// roxxyList returns the gatekeeper's NORMAL list from the world data.
func roxxyList(t *testing.T) *data.TeleportList {
	t.Helper()
	list, ok := testWorldData(t).TeleportList(npcRoxxy, data.TeleportNormal)
	if !ok || len(list.Locations) == 0 {
		t.Fatal("the world data has no NORMAL teleport list for Roxxy")
	}
	return list
}

// wantLocations is the %locations% text Mobius builds for list: fee shows
// the price as "<fee> Adena", or nothing when fee returns 0.
func wantLocations(npc string, list *data.TeleportList, fee func(data.TeleportLocation) int) string {
	var b strings.Builder
	for i, loc := range list.Locations {
		name := teleportLocName(loc)
		text := name
		if f := fee(loc); f != 0 {
			text += " - " + strconv.Itoa(f) + " Adena"
		}
		b.WriteString(`<a action="bypass -h npc_` + npc + `_teleport NORMAL ` + strconv.Itoa(i) + `" msg="811;` + name + `">` + text + `</a><br1>`)
	}
	return b.String()
}

// sysMsgIDs returns the IDs of the system messages in ps.
func sysMsgIDs(ps []testclient.Packet) []int32 {
	var ids []int32
	for _, m := range only[testclient.SystemMessage](ps) {
		ids = append(ids, m.ID)
	}
	return ids
}

func TestTeleportListAndFreeTeleport(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart, adena: 50000})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	rid := strconv.Itoa(int(roxxy.ID()))
	list := roxxyList(t)

	e.talk(sess, p, roxxy)
	// Level 1 travels for free: no prices.
	ps := e.bypass(sess, "npc_"+rid+"_showTeleports")
	want := "<html><body>Region where teleporting is possible<br>" + wantLocations(rid, list, func(data.TeleportLocation) int { return 0 }) + "<br></body></html>"
	wantPage(t, "teleport list", onePage(t, ps), roxxy.ID(), want)

	ps = e.bypass(sess, "npc_"+rid+"_teleport NORMAL 0")
	tps := only[testclient.TeleportToLocation](ps)
	dest := list.Locations[0]
	if len(tps) != 1 || tps[0].Position() != (testclient.Position{X: int32(dest.X), Y: int32(dest.Y), Z: int32(dest.Z) + teleportZOffset}) {
		t.Fatalf("teleport packets %q, TeleportToLocation %+v, want one to %+v", testclient.Names(ps), tps, dest)
	}
	if ids := sysMsgIDs(ps); len(ids) != 0 {
		t.Errorf("a free teleport sent the system messages %v", ids)
	}
	if ps[len(ps)-1].PacketName() != "ActionFailed" {
		t.Errorf("teleport packets %q, want ActionFailed last", testclient.Names(ps))
	}
	e.advance(inventoryUpdateDelay)
	if ius := only[testclient.InventoryUpdate](sess.take(t)); len(ius) != 0 {
		t.Errorf("a free teleport changed the inventory: %+v", ius)
	}
}

func TestPaidTeleport(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, configure: payingEnv})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart, adena: 50000})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	rid := strconv.Itoa(int(roxxy.ID()))
	list := roxxyList(t)
	dest := list.Locations[0]
	if dest.FeeItemID != adenaID || dest.FeeCount <= 0 {
		t.Fatalf("Roxxy's first destination %+v has no adena fee", dest)
	}

	// The list shows the prices.
	e.talk(sess, p, roxxy)
	ps := e.bypass(sess, "npc_"+rid+"_showTeleports")
	want := "<html><body>Region where teleporting is possible<br>" + wantLocations(rid, list, func(loc data.TeleportLocation) int { return loc.FeeCount }) + "<br></body></html>"
	wantPage(t, "teleport list with prices", onePage(t, ps), roxxy.ID(), want)

	// Teleporting takes the fee: "<fee> adena disappeared.", then the
	// teleport, then ActionFailed; the InventoryUpdate follows 100 ms later.
	ps = e.bypass(sess, "npc_"+rid+"_teleport NORMAL 0")
	msgs := only[testclient.SystemMessage](ps)
	if len(msgs) != 1 || msgs[0].ID != int32(sysMsgAdenaDisappeared) || len(msgs[0].Params) != 1 || msgs[0].Params[0].Number != int64(dest.FeeCount) {
		t.Errorf("fee messages %+v, want %d adena disappeared", msgs, dest.FeeCount)
	}
	names := testclient.Names(ps)
	if i, j := slices.Index(names, "SystemMessage"), slices.Index(names, "TeleportToLocation"); i < 0 || j < i {
		t.Errorf("packets %q: want the fee message before TeleportToLocation", names)
	}
	if len(only[testclient.InventoryUpdate](ps)) != 0 {
		t.Error("InventoryUpdate came at once; Mobius sends it 100 ms later")
	}
	e.advance(inventoryUpdateDelay)
	ius := only[testclient.InventoryUpdate](sess.take(t))
	left := int32(50000 - dest.FeeCount)
	if len(ius) != 1 || len(ius[0].Items) != 1 || ius[0].Items[0].Change != testclient.ItemModified ||
		ius[0].Items[0].ItemID != adenaID || ius[0].Items[0].Count != left {
		t.Errorf("InventoryUpdate %+v, want the adena modified to %d", ius, left)
	}

	// The fee is saved.
	saved := e.store.waitSave(t, p.ObjectID())
	var adena int64
	for _, it := range saved.Items {
		if it.ItemID == adenaID {
			adena = it.Count
		}
	}
	if adena != int64(left) {
		t.Errorf("saved adena %d, want %d", adena, left)
	}
}

func TestTeleportWithoutEnoughAdena(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, configure: payingEnv})
	sess, p := e.enterChar(testChar{account: "poor", name: "Poor", pos: tiStart, adena: 100})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	rid := strconv.Itoa(int(roxxy.ID()))
	start := e.position(p)

	e.talk(sess, p, roxxy)
	e.bypass(sess, "npc_"+rid+"_showTeleports")
	ps := e.bypass(sess, "npc_"+rid+"_teleport NORMAL 0")
	// Mobius says it twice: reduceAdena, then the teleporter.
	if ids := sysMsgIDs(ps); !slices.Equal(ids, []int32{int32(sysMsgNotEnoughAdena), int32(sysMsgNotEnoughAdena)}) {
		t.Errorf("system messages %v, want \"You do not have enough adena.\" twice", ids)
	}
	if len(only[testclient.TeleportToLocation](ps)) != 0 || e.position(p) != start {
		t.Error("teleported without the fee")
	}
	var adena int64
	e.onLoop(func() { adena = p.Inventory().itemByItemID(adenaID).Count })
	if adena != 100 {
		t.Errorf("adena %d after a refused teleport, want 100", adena)
	}
}

func TestTeleportFeeTakesTheLastAdena(t *testing.T) {
	list := roxxyList(t)
	fee := int64(list.Locations[0].FeeCount)
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, configure: payingEnv})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart, adena: fee})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	rid := strconv.Itoa(int(roxxy.ID()))

	e.talk(sess, p, roxxy)
	e.bypass(sess, "npc_"+rid+"_showTeleports")
	e.bypass(sess, "npc_"+rid+"_teleport NORMAL 0")
	e.advance(inventoryUpdateDelay)
	ius := only[testclient.InventoryUpdate](sess.take(t))
	if len(ius) != 1 || len(ius[0].Items) != 1 || ius[0].Items[0].Change != testclient.ItemRemoved || ius[0].Items[0].ItemID != adenaID {
		t.Errorf("InventoryUpdate %+v, want the adena removed", ius)
	}
	var left *Item
	e.onLoop(func() { left = p.Inventory().itemByItemID(adenaID) })
	if left != nil {
		t.Errorf("adena left in the inventory: %+v", left.Item)
	}
}

func TestTeleportDiscountOnMondayEvenings(t *testing.T) {
	list := roxxyList(t)
	monday := time.Date(2026, 11, 23, 20, 30, 0, 0, time.UTC)
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, configure: payingEnv, start: monday})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart, adena: 50000})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	rid := strconv.Itoa(int(roxxy.ID()))
	e.talk(sess, p, roxxy)
	ps := e.bypass(sess, "npc_"+rid+"_showTeleports")
	want := wantLocations(rid, list, func(loc data.TeleportLocation) int { return loc.FeeCount / 2 })
	if pg := onePage(t, ps); !strings.Contains(pg.HTML, want) {
		t.Errorf("Monday evening list %q, want the half prices %q", pg.HTML, want)
	}
}

func TestTeleporterRefusals(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages, configure: payingEnv})
	sess, p := e.enterChar(testChar{account: "pk", name: "Pk", pos: tiStart, adena: 50000, karma: 10})
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiStart, 50))
	rid := strconv.Itoa(int(roxxy.ID()))
	offer := func(links ...string) {
		var b strings.Builder
		for _, l := range links {
			b.WriteString(`<a action="bypass -h npc_` + rid + `_` + l + `">x</a>`)
		}
		e.onLoop(func() { p.HTML.Set(HTMLScopeNpc, roxxy.ID(), b.String()) })
	}

	// A player with karma is sent away from the normal lists.
	offer("teleport NORMAL 0", "showNoblesSelect", "teleport NOBLES 0", "teleport NORMAL 999", "teleport NORMAL")
	ps := e.bypass(sess, "npc_"+rid+"_teleport NORMAL 0")
	if msgs := messages(ps); !slices.Equal(msgs, []string{"Go away, you're not welcome here."}) {
		t.Errorf("karma teleport messages %q", msgs)
	}
	if len(only[testclient.TeleportToLocation](ps)) != 0 {
		t.Error("a player with karma was teleported")
	}
	// Nobody is a noble yet.
	wantPage(t, "showNoblesSelect", onePage(t, e.bypass(sess, "npc_"+rid+"_showNoblesSelect")), roxxy.ID(), "<html><body>Roxxy: nobles only.</body></html>")
	// Unknown lists and locations, and a missing location, do nothing.
	for _, cmd := range []string{"teleport NOBLES 0", "teleport NORMAL 999", "teleport NORMAL"} {
		offer(cmd)
		ps := e.bypass(sess, "npc_"+rid+"_"+cmd)
		if len(only[testclient.TeleportToLocation](ps)) != 0 || len(only[testclient.NpcHtmlMessage](ps)) != 0 {
			t.Errorf("%q answered %q", cmd, testclient.Names(ps))
		}
	}
	var adena int64
	e.onLoop(func() { adena = p.Inventory().itemByItemID(adenaID).Count })
	if adena != 50000 {
		t.Errorf("adena %d after refused teleports, want 50000", adena)
	}
}

func TestDestroyItemByItemID(t *testing.T) {
	const gatePass = 6651 // Noblesse Gate Pass, a stackable teleport fee item
	e := newDialogEnv(t, dialogEnvOptions{pages: dialogPages})
	sess, p := e.enterChar(testChar{account: "hero", name: "Hero", pos: tiStart, items: []storage.Item{{ItemID: gatePass, Count: 3, Slot: storage.PaperdollNone}}})
	var ok1, ok2, ok3 bool
	e.onLoop(func() {
		ok1 = e.srv.destroyItemByItemID(p, gatePass, 5, true) // too many
		ok2 = e.srv.destroyItemByItemID(p, gatePass, 2, true)
		ok3 = e.srv.destroyItemByItemID(p, 9999, 1, true) // none
	})
	if ok1 || !ok2 || ok3 {
		t.Errorf("destroy results %v %v %v, want false true false", ok1, ok2, ok3)
	}
	ps := sess.take(t)
	if ids := sysMsgIDs(ps); !slices.Equal(ids, []int32{int32(sysMsgIncorrectCount), int32(sysMsgS2S1Disappeared), int32(sysMsgIncorrectCount)}) {
		t.Errorf("system messages %v", ids)
	}
	sus := only[testclient.StatusUpdate](ps)
	if len(sus) != 1 || sus[0].ObjectID != p.ObjectID() {
		t.Errorf("StatusUpdate %+v, want the player's load", sus)
	}
	e.advance(inventoryUpdateDelay)
	ius := only[testclient.InventoryUpdate](sess.take(t))
	if len(ius) != 1 || len(ius[0].Items) != 1 || ius[0].Items[0].Count != 1 || ius[0].Items[0].Change != testclient.ItemModified {
		t.Errorf("InventoryUpdate %+v, want one pass left", ius)
	}
}
