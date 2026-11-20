package game

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

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
