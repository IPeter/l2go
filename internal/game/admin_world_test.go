package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// adminRealPage returns the Mobius page rel of data/html when it is there
// (it is not in git), else fallback, a copy of its relevant lines.
func adminRealPage(rel, fallback string) string {
	b, err := os.ReadFile(filepath.Join("..", "..", "data", "html", filepath.FromSlash(rel)))
	if err != nil {
		return fallback
	}
	return string(b)
}

// adminWorldPages are the pages of the world tool tests: the real Mobius
// pages when data/html has them, else copies of their buttons.
func adminWorldPages() map[string]string {
	return map[string]string{
		"admin/main_menu.htm": adminRealPage("admin/main_menu.htm", `<html><title>Main Admin Menu</title><body>`+
			`<td><edit var="qbox" width=120 height=12></td>`+
			`<td><button value="Item" action="bypass admin_create_item $qbox" width=65 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2"></td>`+
			`<td><button value="Teleport" action="bypass admin_move_to $qbox" width=65 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2"></td>`+
			`<td><button value="Spawn" action="bypass admin_spawn_monster $qbox" width=65 height=21 back="L2UI_ch3.smallbutton2_over" fore="L2UI_ch3.smallbutton2"></td>`+
			`</body></html>`),
		"admin/teleports/TownAreas/giran.htm": adminRealPage("admin/teleports/TownAreas/giran.htm", "<html><title>Admin Teleports</title>\r\n<body>\r\n<center>\r\n"+
			"<font color=\"aadd77\">Giran Castle Town</font>\r\n<table width=256>\r\n<tr>\r\n"+
			`<td align=center><button action="bypass -h admin_move_to 82698 148638 -3473" value="Town Center" width=95 height=21 back="bigbutton_over" fore="bigbutton"></td>`+"\r\n"+
			`<td align=center><button action="bypass -h admin_move_to 79768 146559 -3525" value="Weapon Shop" width=95 height=21 back="bigbutton_over" fore="bigbutton"></td>`+"\r\n"+
			"</tr>\r\n</table>\r\n</center>\r\n</body></html>"),
		"admin/gmshops.htm": adminRealPage("admin/gmshops.htm", `<html><title>Shop</title><body>`+
			`<button action="bypass admin_html gmstore/gear/weapons.htm" value="Weapons" width="95" height="21" back="bigbutton_over" fore="bigbutton">`+
			`<button action="bypass admin_html gmstore/gear/armor.htm" value="Armor" width="95" height="21" back="bigbutton_over" fore="bigbutton">`+
			`</body></html>`),
		"admin/gmstore/gear/weapons.htm": adminRealPage("admin/gmstore/gear/weapons.htm", `<html><title>Shop</title><body>`+
			`<td align=center><button action="bypass -h admin_buy 9997" value="A Grade" width="95" height="21" back="bigbutton_over" fore="bigbutton"></td>`+
			`<td align=center><button action="bypass -h admin_buy 10007" value="Raid Boss Weapons" width="95" height="21" back="bigbutton_over" fore="bigbutton"></td>`+
			`</body></html>`),
		"admin/gmstore/gear/armor.htm": adminRealPage("admin/gmstore/gear/armor.htm", `<html><title>Shop</title><body>`+
			`<td align=center><button action="bypass -h admin_buy 9956" value="S Grade Sets" width="95" height="21" back="bigbutton_over" fore="bigbutton"></td>`+
			`</body></html>`),
		"admin/gm_menu.htm":      adminRealPage("admin/gm_menu.htm", `<html><title>GM Menu</title><body></body></html>`),
		"admin/itemcreation.htm": adminRealPage("admin/itemcreation.htm", `<html><title>Items</title><body></body></html>`),
		"admin/search.htm": adminRealPage("admin/search.htm", `<html><title>Admin Item Search</title><body>`+
			`<button value="Search" action="bypass admin_search $word"><table>%items%</table>Page: %pages%</body></html>`),
		"admin/serverinfo.htm": adminRealPage("admin/serverinfo.htm", `<html><title>Admin Server Info</title><body>`+
			`%usedMem% %freeMem% %totalMem% %live% %nondaemon% %daemon% %peak% %totalstarted% %gcol% %colcount% %coltime% `+
			`%onlineAll% %offlineTrade% %onlineGM% %onlineReal% %os_name% %os_ver% %slots% %serverUpTime% %serverTime% `+
			`%gameTime% (%dayNight%) %geodata%</body></html>`),
		"admin/reload.htm": adminRealPage("admin/reload.htm", `<html><head><title>Reload Panel</title></head><body><center>%buttons%</center></body></html>`),
	}
}

// newAdminWorldEnv starts a server with the world tool pages and the
// economy data, without the transaction flood limit.
func newAdminWorldEnv(t *testing.T) *itemEnv {
	t.Helper()
	e := &itemEnv{testEnv: newDialogEnv(t, dialogEnvOptions{
		pages:     adminWorldPages(),
		configure: func(gs *config.GameServer) { gs.Economy.FloodTransaction = 0 },
	})}
	eco := testEconomy(t)
	e.onLoop(func() { e.srv.economy = eco })
	return e
}

// teleportedTo checks that ps teleports the GM to x y z with the
// //move_to answer, then lets the client appear there.
func (e *itemEnv) teleportedTo(t *testing.T, sess *fakeSession, what string, ps []testclient.Packet, x, y, z int32) {
	t.Helper()
	tps := only[testclient.TeleportToLocation](ps)
	// The Z lands on the ground below the given one.
	if len(tps) != 1 || tps[0].X != x || tps[0].Y != y || tps[0].Z < z-64 || tps[0].Z > z+64 {
		t.Fatalf("%s: teleports %+v (messages %q), want one to %d %d %d", what, tps, messages(ps), x, y, z)
	}
	e.srv.Handle(sess, clientpackets.OpAppearing, nil)
	e.sync()
	sess.take(t)
}

// The buttons of the admin pages send their arguments: the town buttons
// their written coordinates, the QuickBox buttons what the client put in
// place of $qbox. The whole bypass reaches the GM command.
func TestAdminButtonArguments(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, p := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})

	// Admin Teleports > Town Areas > Giran > Town Center.
	gmRun(e.testEnv, gs, "html teleports/TownAreas/giran.htm")
	adminPage(t, "the Giran page", gs.take(t), "Admin Teleports")
	ps := e.bypass(gs, "admin_move_to 82698 148638 -3473")
	if msgs := messages(ps); !slices.Equal(msgs, []string{"You have been teleported to 82698 148638 -3473"}) {
		t.Errorf("Town Center: messages %q", msgs)
	}
	e.teleportedTo(t, gs, "Town Center", ps, 82698, 148638, -3473)

	// The main page's QuickBox with coordinates, then an item.
	gmRun(e.testEnv, gs, "admin")
	gs.take(t)
	e.teleportedTo(t, gs, "QuickBox Teleport", e.bypass(gs, "admin_move_to -71000 258000 -3100"), -71000, 258000, -3100)
	before := e.countOf(p, adenaID)
	e.bypass(gs, "admin_create_item 57 100")
	if got := e.countOf(p, adenaID); got != before+100 {
		t.Errorf("QuickBox Item 57 100: %d adena, want %d", got, before+100)
	}

	// The bare command (the main page's Teleport button, or an empty
	// QuickBox) opens the teleport page, as Mobius AdminTeleport does.
	ps = e.bypass(gs, "admin_move_to")
	if msgs := messages(ps); len(msgs) != 0 {
		t.Errorf("bare admin_move_to: messages %q, want the teleport page", msgs)
	}
	onePage(t, ps)
}

// The GM shop: //gmshop, a gmstore page, admin_buy opens the buylist with
// the GM's adena and no merchant, and buying from it gives the items for
// free (OnlyGMItemsFree).
func TestAdminGMShop(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, gm := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart, adena: 1000})

	gmRun(e.testEnv, gs, "gmshop")
	adminPage(t, "//gmshop", gs.take(t), "Shop")
	adminPage(t, "Weapons", e.bypass(gs, "admin_html gmstore/gear/weapons.htm"), "Shop")

	ps := e.bypass(gs, "admin_buy 9997")
	checkNames(t, "admin_buy 9997", ps, "BuyList", "ActionFailed")
	bl := shopOne[testclient.BuyList](t, "admin_buy 9997", ps)
	list, _ := testEconomy(t).BuyList(9997)
	if bl.ListID != 9997 || bl.Adena != 1000 || len(bl.Items) != len(list.Products) {
		t.Fatalf("BuyList %d, adena %d, %d entries; want 9997, 1000, %d", bl.ListID, bl.Adena, len(bl.Items), len(list.Products))
	}
	for _, it := range bl.Items {
		if it.Price != 0 {
			t.Errorf("GM shop item %d costs %d", it.ItemID, it.Price)
		}
	}

	// Buying: Dragon Slayer (81) and Sword of Miracles (151), for free.
	ps = e.shopBuy(gs, 9997, clientpackets.BuyItem{ItemID: 81, Count: 1}, clientpackets.BuyItem{ItemID: 151, Count: 1})
	if ids := sysMsgIDsOf(ps); !slices.Contains(ids, int32(shopMsgTransactionComplete)) {
		t.Fatalf("GM shop buy: %q (messages %q)", testclient.Names(ps), messages(ps))
	}
	if e.countOf(gm, 81) != 1 || e.countOf(gm, 151) != 1 || e.shopAdena(gm) != 1000 {
		t.Errorf("after the buy: %d of 81, %d of 151, %d adena", e.countOf(gm, 81), e.countOf(gm, 151), e.shopAdena(gm))
	}

	// Armor: an S grade set piece.
	gmRun(e.testEnv, gs, "gmshop")
	gs.take(t)
	adminPage(t, "Armor", e.bypass(gs, "admin_html gmstore/gear/armor.htm"), "Shop")
	checkNames(t, "admin_buy 9956", e.bypass(gs, "admin_buy 9956"), "BuyList", "ActionFailed")
	e.shopBuy(gs, 9956, clientpackets.BuyItem{ItemID: 6373, Count: 1})
	if e.countOf(gm, 6373) != 1 {
		t.Errorf("bought %d of item 6373", e.countOf(gm, 6373))
	}

	// A list that does not exist (Mobius has no 10007 either), and none.
	gmRun(e.testEnv, gs, "buy 10007")
	ps = gs.take(t)
	checkNames(t, "//buy 10007", ps, "SystemMessage", "ActionFailed")
	if msgs := messages(ps); !slices.Equal(msgs, []string{"Buylist 10007 does not exist."}) {
		t.Errorf("//buy 10007: %q", msgs)
	}
	gmRun(e.testEnv, gs, "buy")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"Please specify buylist."}) {
		t.Errorf("//buy: %q", msgs)
	}

	// A player cannot buy from a GM list without a merchant.
	ps2, _ := e.enterChar(testChar{account: "player", name: "Player", pos: tiStart, adena: 1000})
	checkNames(t, "a player's buy", e.shopBuy(ps2, 9997, clientpackets.BuyItem{ItemID: 81, Count: 1}), "ActionFailed")
}

// Every product of the GM lists (the buylists that name no NPC) is an
// item the server knows, so the GM shop windows are complete.
func TestAdminGMShopListsHaveTheirItems(t *testing.T) {
	d := testGameData(t)
	var missing []string
	for _, l := range testEconomy(t).BuyLists {
		if len(l.NpcIDs) > 0 || l.ID == testBuyListID {
			continue
		}
		for _, pr := range l.Products {
			if _, ok := d.ItemTemplate(pr.ItemID); !ok {
				missing = append(missing, strconv.Itoa(l.ID)+":"+strconv.Itoa(pr.ItemID))
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("GM shop products without an item (list:item): %v", missing)
	}
}

// adminWorldPair enters a GM, "Gm", and a player, "Player", at tiStart and
// drops their packets.
func adminWorldPair(t *testing.T, e *itemEnv) (gs, ps *fakeSession, gm, pl *Player) {
	t.Helper()
	gs, gm = e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	ps, pl = e.enterChar(testChar{account: "player", name: "Player", pos: tiStart})
	gs.take(t)
	ps.take(t)
	return gs, ps, gm, pl
}

// announcesOf returns the announcements in ps, a critical one with "!".
func announcesOf(ps []testclient.Packet) []string {
	var out []string
	for _, s := range only[testclient.CreatureSay](ps) {
		switch s.Type {
		case testclient.ChatAnnouncement:
			out = append(out, s.Text)
		case testclient.ChatCriticalAnnounce:
			out = append(out, "!"+s.Text)
		}
	}
	return out
}

// //announce, //announce_crit and //announce_screen reach every player
// (the GM too), and the GM menu shows again.
func TestAdminAnnounce(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, ps, _, _ := adminWorldPair(t, e)

	gmRun(e.testEnv, gs, "announce Hello  world")
	got := gs.take(t)
	adminPage(t, "//announce", got, "GM Menu")
	if a := announcesOf(got); !slices.Equal(a, []string{"Hello world"}) {
		t.Errorf("the GM's announcements %q", a)
	}
	if a := announcesOf(ps.take(t)); !slices.Equal(a, []string{"Hello world"}) {
		t.Errorf("the player's announcements %q", a)
	}

	gmRun(e.testEnv, gs, "announce_crit Danger")
	gs.take(t)
	if a := announcesOf(ps.take(t)); !slices.Equal(a, []string{"!Danger"}) {
		t.Errorf("//announce_crit: %q", a)
	}

	gmRun(e.testEnv, gs, "announce_screen On screen")
	gs.take(t)
	sms := only[testclient.ExShowScreenMessage](ps.take(t))
	if len(sms) != 1 || sms[0].Text != "On screen" || sms[0].Time != 10000 || sms[0].Position != 2 {
		t.Errorf("//announce_screen: %+v", sms)
	}

	gmRun(e.testEnv, gs, "announce")
	got = gs.take(t)
	noPage(t, "//announce without text", got)
	if says := sysSays(got); !slices.Equal(says, []string{"Syntax: //announce <text to announce here>"}) {
		t.Errorf("//announce without text: %q", says)
	}
}

// The announcement list: added announcements show to entering players, an
// auto one comes after its delays as often as asked, and they can be shown,
// edited, restarted and removed with Mobius's answers.
func TestAdminAnnouncements(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, ps, gm, pl := adminWorldPair(t, e)
	run := func(line string) []testclient.Packet {
		t.Helper()
		gs.take(t)
		gmRun(e.testEnv, gs, line)
		return gs.take(t)
	}

	page := adminPage(t, "the empty list", run("announces list"), "Announcements")
	if !strings.Contains(page, "There are no announcements.") || !strings.Contains(page, "bypass admin_announces add") {
		t.Errorf("the empty list: %q", page)
	}
	adminPage(t, "the Add button", e.bypass(gs, "admin_announces add"), "Announcements")

	got := run("announces add NORMAL 0 0 0 Welcome to L2Go")
	if says := sysSays(got); !slices.Equal(says, []string{"Announcement has been successfully added!"}) {
		t.Errorf("add: %q", says)
	}
	if page := adminPage(t, "add", got, "Announcements"); !strings.Contains(page, "bypass admin_announces show 1") {
		t.Errorf("the list after add: %q", page)
	}
	run("announces add critical 0 0 0 Be nice")
	for _, bad := range []string{"announces add NORMAL x 0 0 Text", "announces add NORMAL 0 0 0"} {
		if says := sysSays(run(bad)); !slices.Equal(says, []string{"Syntax: //announces add <type> <delay> <repeat> <text>"}) {
			t.Errorf("//%s: %q", bad, says)
		}
	}
	if says := sysSays(run("announces add AUTO_NORMAL 0 5 0 Too often")); !slices.Equal(says, []string{"Delay cannot be less then 10 seconds!"}) {
		t.Errorf("a short delay: %q", says)
	}

	// An entering player gets the normal, then the critical announcements,
	// with its own name.
	ps.take(t)
	e.onLoop(func() { e.srv.showAnnouncements(pl) })
	got = ps.take(t)
	if a := announcesOf(got); !slices.Equal(a, []string{"Welcome to L2Go", "!Be nice"}) {
		t.Errorf("on entering: %q", a)
	}
	if says := only[testclient.CreatureSay](got); len(says) == 0 || says[0].Name != "Player" {
		t.Errorf("on entering: %+v", says)
	}

	// Show, edit and remove.
	page = adminPage(t, "show 1", e.bypass(gs, "admin_announces show 1"), "Announcements")
	if !strings.Contains(page, "Welcome to L2Go") || !strings.Contains(page, "NORMAL") {
		t.Errorf("show 1: %q", page)
	}
	if says := sysSays(run("announces edit 1 AUTO_NORMAL 0 10 0")); !slices.Equal(says, []string{"Announce type can be changed only to NORMAL or CRITICAL!"}) {
		t.Errorf("edit 1 to auto: %q", says)
	}
	run("announces edit 1 CRITICAL 0 0 0")
	e.onLoop(func() {
		if a := e.srv.announcementByID(1); a.typ != announceCritical || a.content != "Welcome to L2Go" || a.author != gm.Name() {
			t.Errorf("after the edit: %+v", *a)
		}
	})
	if says := sysSays(run("announces remove 2")); !slices.Equal(says, []string{"Announcement has been successfully removed!"}) {
		t.Errorf("remove 2: %q", says)
	}
	if says := sysSays(run("announces remove 2")); !slices.Equal(says, []string{"Announcement does not exist!"}) {
		t.Errorf("remove 2 again: %q", says)
	}
	if says := sysSays(run("announces restart 1")); !slices.Equal(says, []string{"This option has effect only on auto announcements!"}) {
		t.Errorf("restart 1: %q", says)
	}

	// An auto announcement: after 5 s, then every 10 s, twice.
	run("announces add AUTO_CRITICAL 5 10 2 Tick")
	ps.take(t)
	var heard []string
	for _, d := range []time.Duration{4 * time.Second, time.Second, 10 * time.Second, 10 * time.Second, 30 * time.Second} {
		e.advance(d)
		heard = append(heard, announcesOf(ps.take(t))...)
		heard = append(heard, "|")
	}
	if want := []string{"|", "!Tick", "|", "!Tick", "|", "|", "|"}; !slices.Equal(heard, want) {
		t.Errorf("the auto announcement: %q, want %q", heard, want)
	}
	if says := sysSays(run("announces restart 3")); !slices.Equal(says, []string{"Auto announcement has been successfully restarted"}) {
		t.Errorf("restart 3: %q", says)
	}
	e.advance(5 * time.Second)
	if a := announcesOf(ps.take(t)); !slices.Equal(a, []string{"!Tick"}) {
		t.Errorf("after the restart: %q", a)
	}
	run("announces remove 3")
	e.advance(time.Minute)
	if a := announcesOf(ps.take(t)); len(a) != 0 {
		t.Errorf("after the removal: %q", a)
	}
}

// //search lists the items whose name has the word, with create buttons
// and page links; //search alone shows the empty page.
func TestAdminSearch(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, _, gm, _ := adminWorldPair(t, e)

	gmRun(e.testEnv, gs, "search")
	if page := adminPage(t, "//search", gs.take(t), "Admin Item Search"); strings.Contains(page, "%items%") {
		t.Errorf("//search: %q", page)
	}

	dragon, _ := e.data.ItemTemplate(81)
	gmRun(e.testEnv, gs, "search slayer")
	page := adminPage(t, "//search slayer", gs.take(t), "Admin Item Search")
	if !strings.Contains(page, `action="bypass admin_create_item 81"`) || !strings.Contains(page, `<font color="LEVEL">Slayer</font>`) {
		t.Errorf("//search slayer: %q", page)
	}
	e.bypass(gs, "admin_create_item 81")
	if n := e.countOf(gm, 81); n != 1 {
		t.Errorf("the Add button of %s: %d of it", dragon.Name, n)
	}

	gmRun(e.testEnv, gs, "search sword")
	page = adminPage(t, "//search sword", gs.take(t), "Admin Item Search")
	if !strings.Contains(page, `bypass -h admin_search sword 2`) || strings.Count(page, `value="Add"`) != adminSearchPageLimit {
		t.Errorf("//search sword: %q", page)
	}
	adminPage(t, "page 2", e.bypass(gs, "admin_search sword 2"), "Admin Item Search")

	gmRun(e.testEnv, gs, "search zzqqzz")
	if page := adminPage(t, "//search zzqqzz", gs.take(t), "Admin Item Search"); !strings.Contains(page, "No items found with word zzqqzz.") {
		t.Errorf("//search zzqqzz: %q", page)
	}
}

// //give_item_to_all gives every other player the items; //create_coin
// gives the GM a coin by name and shows the item creation page.
func TestAdminGiveItemToAllAndCoins(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, ps, gm, pl := adminWorldPair(t, e)

	gmBefore, plBefore := e.countOf(gm, adenaID), e.countOf(pl, adenaID)
	gmRun(e.testEnv, gs, "give_item_to_all 57 100")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"1 players rewarded with Adena"}) {
		t.Errorf("//give_item_to_all: %q", msgs)
	}
	if msgs := messages(ps.take(t)); !slices.Equal(msgs, []string{"Admin spawned 100 Adena in your inventory."}) {
		t.Errorf("the player: %q", msgs)
	}
	if e.countOf(gm, adenaID) != gmBefore || e.countOf(pl, adenaID) != plBefore+100 {
		t.Errorf("adena: GM %d (was %d), player %d (was %d)", e.countOf(gm, adenaID), gmBefore, e.countOf(pl, adenaID), plBefore)
	}
	for line, want := range map[string]string{
		"give_item_to_all 99999999": "This item doesn't exist.",
		"give_item_to_all 81 11":    "This item does not stack - Creation aborted.",
		"give_item_to_all x":        "Specify a valid number.",
	} {
		gmRun(e.testEnv, gs, line)
		if says := sysSays(gs.take(t)); !slices.Equal(says, []string{want}) {
			t.Errorf("//%s: %q", line, says)
		}
	}

	gmRun(e.testEnv, gs, "create_coin AncientAdena 5")
	adminPage(t, "//create_coin", gs.take(t), "Items")
	if n := e.countOf(gm, 5575); n != 5 {
		t.Errorf("//create_coin AncientAdena 5: %d ancient adena", n)
	}
	gmRun(e.testEnv, gs, "create_coin Nothing")
	got := gs.take(t)
	adminPage(t, "//create_coin Nothing", got, "Items")
	if msgs := messages(got); !slices.Equal(msgs, []string{"This item doesn't exist."}) {
		t.Errorf("//create_coin Nothing: %q", msgs)
	}
}

// adminTestSpawn makes a spawn slot of Roxxy dx east of tiStart, with a
// respawn delay of 30 s, and returns it with its NPC.
func adminTestSpawn(t *testing.T, e *itemEnv, dx int32) (*npcSpawn, *Npc) {
	t.Helper()
	var sp *npcSpawn
	var n *Npc
	e.onLoop(func() {
		tmpl, _ := e.srv.worldData.Npc(npcRoxxy)
		sp = &npcSpawn{tmpl: tmpl, point: eastOf(tiStart, dx), respawnMin: 30 * time.Second, respawnMax: 30 * time.Second, index: -1}
		e.srv.addSpawnSlot(sp)
		n = e.srv.spawnSlot(sp)
		sp.respawn = true
	})
	if n == nil {
		t.Fatal("Roxxy did not spawn")
	}
	return sp, n
}

// //list_spawns lists the spawns of an NPC by ID or name, and teleports to
// one by its number; an NPC without a spawn is found in the world.
func TestAdminListSpawns(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, _, _, _ := adminWorldPair(t, e)
	sp, n := adminTestSpawn(t, e, 300)

	index := 0
	e.onLoop(func() {
		for _, o := range e.srv.npc.spawns {
			if o.tmpl.ID == npcRoxxy {
				index++
			}
			if o == sp {
				break
			}
		}
	})
	pt := sp.point
	want := fmt.Sprintf("%d - %s (Spawn ID: %d X: %d Y: %d Z: %d Heading: %d): %d %d %d", index, n.Name(), npcRoxxy, pt.X, pt.Y, pt.Z, pt.Heading, pt.X, pt.Y, pt.Z)
	for _, line := range []string{"list_spawns 30006", "list_spawns " + n.Name(), "list_positions 30006"} {
		gmRun(e.testEnv, gs, line)
		if msgs := messages(gs.take(t)); !slices.Contains(msgs, want) {
			t.Errorf("//%s: %q, want %q in it", line, msgs, want)
		}
	}

	gmRun(e.testEnv, gs, "list_spawns 30006 "+strconv.Itoa(index))
	got := gs.take(t)
	tps := only[testclient.TeleportToLocation](got)
	if len(tps) != 1 || tps[0].X < pt.X-maxOffsetOnTeleport || tps[0].X > pt.X+maxOffsetOnTeleport ||
		tps[0].Y < pt.Y-maxOffsetOnTeleport || tps[0].Y > pt.Y+maxOffsetOnTeleport {
		t.Fatalf("goSpawn: teleports %+v, want one near %v", tps, pt)
	}
	if says := sysSays(got); !slices.Equal(says, []string{"You have been teleported."}) {
		t.Errorf("goSpawn: %q", says)
	}
	e.srv.Handle(gs, clientpackets.OpAppearing, nil)
	e.sync()
	gs.take(t)

	gmRun(e.testEnv, gs, "list_spawns Nobody At All")
	if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{"The specified teleport target does not exist."}) {
		t.Errorf("an unknown name: %q", msgs)
	}

	// An NPC of no spawn slot.
	e.onLoop(func() {
		for _, o := range e.srv.npc.spawns {
			if o.tmpl.ID == npcMartin {
				t.Fatalf("Martin has a spawn slot")
			}
		}
	})
	e.spawnNpc(npcMartin, eastOf(tiStart, 600))
	gmRun(e.testEnv, gs, "list_spawns "+strconv.Itoa(npcMartin))
	got = gs.take(t)
	if msgs := messages(got); !slices.Equal(msgs, []string{"The current spawn is not stored."}) || len(only[testclient.TeleportToLocation](got)) != 1 {
		t.Errorf("an NPC without a spawn: %q, %q", msgs, testclient.Names(got))
	}
}

// //recall_npc moves the target NPC's spawn to the GM for good, with its
// respawn delay.
func TestAdminRecallNpc(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, _, gm, _ := adminWorldPair(t, e)

	gmRun(e.testEnv, gs, "recall_npc")
	if ids := sysMsgIDsOf(gs.take(t)); !slices.Equal(ids, []int32{int32(gmSysMsgInvalidTarget)}) {
		t.Errorf("no target: %v", ids)
	}

	sp, n := adminTestSpawn(t, e, 300)
	e.onLoop(func() { e.srv.SetTarget(gm, n) })
	gs.take(t)
	gmRun(e.testEnv, gs, "recall_npc")
	if says := sysSays(gs.take(t)); !slices.Equal(says, []string{fmt.Sprintf("Created %s on %d.", n.Name(), n.ID())}) {
		t.Errorf("//recall_npc: %q", says)
	}
	e.onLoop(func() {
		if !sp.deleted() {
			t.Error("the old spawn was not deleted")
		}
		at := gm.Position()
		var moved *npcSpawn
		for _, o := range e.srv.npc.spawns {
			if o.tmpl.ID == npcRoxxy && o.point == at {
				moved = o
			}
		}
		if moved == nil || moved.npc == nil || !moved.npc.InWorld() || moved.respawnMin != 30*time.Second || !moved.respawn {
			t.Errorf("the new spawn: %+v", moved)
		}
	})

	// An NPC without a spawn.
	other := e.spawnNpc(npcMartin, eastOf(tiStart, 100))
	e.onLoop(func() { e.srv.SetTarget(gm, other) })
	gs.take(t)
	gmRun(e.testEnv, gs, "recall_npc")
	if says := sysSays(gs.take(t)); !slices.Equal(says, []string{"Incorrect monster spawn."}) {
		t.Errorf("an NPC without a spawn: %q", says)
	}
}

// //serverinfo fills every number of the page.
func TestAdminServerInfo(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, _, _, _ := adminWorldPair(t, e)
	gmRun(e.testEnv, gs, "serverinfo")
	page := adminPage(t, "//serverinfo", gs.take(t), "Admin Server Info")
	for _, ph := range []string{"%usedMem%", "%live%", "%gcol%", "%onlineAll%", "%slots%", "%serverUpTime%", "%gameTime%", "%geodata%"} {
		if strings.Contains(page, ph) {
			t.Errorf("//serverinfo left %s: %q", ph, page)
		}
	}
	if !strings.Contains(page, "0 Days, 0 Hours, 0 Minutes") || !strings.Contains(page, "Disabled") {
		t.Errorf("//serverinfo: %q", page)
	}
}

// //reload_menu shows the reload links in pages; //reload html reads the
// pages again, and the types the server cannot reload yet say so.
func TestAdminReload(t *testing.T) {
	e := newAdminWorldEnv(t)
	gs, _, _, _ := adminWorldPair(t, e)

	gmRun(e.testEnv, gs, "reload_menu")
	page := adminPage(t, "//reload_menu", gs.take(t), "Reload Panel")
	if !strings.Contains(page, "bypass admin_reload access") || !strings.Contains(page, "admin_reload_menu_page 1") {
		t.Errorf("//reload_menu: %q", page)
	}
	page = adminPage(t, "Next", e.bypass(gs, "admin_reload_menu_page 1"), "Reload Panel")
	if !strings.Contains(page, "bypass admin_reload html $value") || !strings.Contains(page, "Page 2 / 3") {
		t.Errorf("page 2: %q", page)
	}
	page = adminPage(t, "Next", e.bypass(gs, "admin_reload_menu_page 2"), "Reload Panel")
	if !strings.Contains(page, "bypass admin_reload quest $value") || !strings.Contains(page, "bypass admin_reload zone") {
		t.Errorf("page 3: %q", page)
	}

	// A page read again after it changed on the disk.
	var dir string
	e.onLoop(func() { dir = e.srv.settings.HTMLDir })
	gmRun(e.testEnv, gs, "search")
	gs.take(t)
	writeDialogFile(t, filepath.Join(dir, "admin", "search.htm"), `<html><title>Changed</title><body>%items%%pages%</body></html>`)
	gmRun(e.testEnv, gs, "search")
	adminPage(t, "the cached page", gs.take(t), "Admin Item Search")
	gmRun(e.testEnv, gs, "reload html admin/search.htm")
	got := gs.take(t)
	if msgs := messages(got); !slices.Equal(msgs, []string{"Gm: Reloaded Htm File:search.htm."}) {
		t.Errorf("//reload html admin/search.htm: %q", msgs)
	}
	if says := sysSays(got); !slices.Equal(says, []string{adminReloadWarning}) {
		t.Errorf("//reload html admin/search.htm: %q", says)
	}
	gmRun(e.testEnv, gs, "search")
	adminPage(t, "the reloaded page", gs.take(t), "Changed")

	gmRun(e.testEnv, gs, "reload html")
	got = gs.take(t)
	if says := sysSays(got); !slices.Equal(says, []string{"Cache[HTML]: 0.0 megabytes on 0 files loaded", adminReloadWarning}) {
		t.Errorf("//reload html: %q", says)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"Gm: Reloaded Htms."}) {
		t.Errorf("//reload html: %q", msgs)
	}

	for line, want := range map[string][]string{
		"reload html no/such.htm": {"File or Directory does not exist.", adminReloadWarning},
	} {
		gmRun(e.testEnv, gs, line)
		if says := sysSays(gs.take(t)); !slices.Equal(says, want) {
			t.Errorf("//%s: %q", line, says)
		}
	}
	for line, want := range map[string]string{
		"reload npc":     "Reloading npc is not available yet.",
		"reload nothing": adminReloadUsage,
		"reload quest":   "This server has no scripts.",
	} {
		gmRun(e.testEnv, gs, line)
		if msgs := messages(gs.take(t)); !slices.Equal(msgs, []string{want}) {
			t.Errorf("//%s: %q", line, msgs)
		}
	}
}

// //reload quest loads the scripts again and swaps them in.
func TestAdminReloadScripts(t *testing.T) {
	e := newDialogEnv(t, dialogEnvOptions{scripts: map[string]string{}})
	gs, _ := e.enterChar(testChar{account: "gm", name: "Gm", access: storage.AccessLevelAdmin, pos: tiStart})
	gs.take(t)
	gmRun(e, gs, "reload quest")
	gs.waitFor(t, "the reload's answers", func(s *fakeSession) bool { return len(s.bodies) >= 3 })
	got := gs.take(t)
	if says := sysSays(got); !slices.Equal(says, []string{"All scripts have been reloaded.", adminReloadWarning}) {
		t.Errorf("//reload quest: %q", says)
	}
	if msgs := messages(got); !slices.Equal(msgs, []string{"Gm: Reloaded Quests."}) {
		t.Errorf("//reload quest: %q", msgs)
	}
}
