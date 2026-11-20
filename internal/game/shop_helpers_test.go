package game

import (
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The shop tests use the economy data of data/ plus testBuyList, a list of
// Lector's with known prices and a limited product.
const (
	testBuyListID   = 99000001
	lectorBuyListID = 3000100 // Lector's first real list
	otherBuyListID  = 3000200 // a list of another merchant

	shopOrePrice       = 100  // itemMithrilOre on testBuyList
	shopSwordPrice     = 500  // itemLongSword on testBuyList
	shopLimitedItem    = 1835 // Soulshot: No Grade, stackable
	shopLimitedPrice   = 10
	shopLimitedCount   = 5
	shopRestockMinutes = 60
)

var testBuyList = data.BuyList{
	ID:     testBuyListID,
	NpcIDs: []int{npcLector},
	Products: []data.BuyListProduct{
		{ItemID: itemMithrilOre, Price: shopOrePrice},
		{ItemID: itemLongSword, Price: shopSwordPrice},
		{ItemID: shopLimitedItem, Price: shopLimitedPrice, Count: shopLimitedCount, RestockDelay: shopRestockMinutes},
	},
}

var loadTestEconomy = sync.OnceValues(func() (*data.Economy, error) {
	e, err := data.LoadEconomy(filepath.Join("..", "..", "data"), data.EconomyRefs{})
	if err != nil {
		return nil, err
	}
	d := e.EconomyData
	d.BuyLists = append(slices.Clone(d.BuyLists), testBuyList)
	return data.NewEconomy(d, data.EconomyRefs{})
})

func testEconomy(t *testing.T) *data.Economy {
	t.Helper()
	e, err := loadTestEconomy()
	if err != nil {
		t.Fatalf("load the economy data: %v", err)
	}
	return e
}

// shopPages are the HTML fixture pages of the shop tests: Lector's page
// offers the shop links.
var shopPages = map[string]string{
	"merchant/30001.htm": `<html><body>%npcname%<br>` +
		`<a action="bypass -h npc_%objectId%_Buy ` + strconv.Itoa(testBuyListID) + `">Buy</a><br>` +
		`<a action="bypass -h npc_%objectId%_Buy ` + strconv.Itoa(lectorBuyListID) + `">Buy weapons</a><br>` +
		`<a action="bypass -h npc_%objectId%_Buy ` + strconv.Itoa(otherBuyListID) + `">Buy elsewhere</a><br>` +
		`<a action="bypass -h npc_%objectId%_Sell">Sell</a><br>` +
		`<a action="bypass -h npc_%objectId%_BuyShadowItem">Shadow</a></body></html>`,
	shadowItemLowLevel: `<html><body>Too young (%objectId%)</body></html>`,
	shadowItemD:        `<html><body>D grade (%objectId%)</body></html>`,
	shadowItemC:        `<html><body>C grade (%objectId%)</body></html>`,
	shadowItemB:        `<html><body>B grade (%objectId%)</body></html>`,
}

// newShopEnv starts a server with the economy data and the shop pages,
// without the transaction flood limit unless configure sets one.
func newShopEnv(t *testing.T, configure func(gs *config.GameServer)) *itemEnv {
	t.Helper()
	e := &itemEnv{testEnv: newDialogEnv(t, dialogEnvOptions{
		pages: shopPages,
		configure: func(gs *config.GameServer) {
			gs.Economy.FloodTransaction = 0
			if configure != nil {
				configure(gs)
			}
		},
	})}
	eco := testEconomy(t)
	// The world's merchants spawned without economy data; the tests spawn
	// their own.
	e.onLoop(func() {
		e.srv.economy = eco
		e.srv.shop.priceZones = nil
	})
	return e
}

// shopPos is where the shop tests stand: in the Talking Island village,
// whose merchants add a 15 % tax.
var shopPos = tiVillage

// shopLector spawns Lector next to shopPos, and makes p target him and
// talk to him (so he is p's last NPC).
func (e *itemEnv) shopLector(sess *fakeSession, p *Player) *Npc {
	e.t.Helper()
	n := e.spawnNpc(npcLector, eastOf(shopPos, 50))
	e.onLoop(func() { e.srv.SetTarget(p, n) })
	e.talk(sess, p, n)
	return n
}

// shopCmd sends the bypass "npc_<n>_<cmd>" and returns the answer.
func (e *itemEnv) shopCmd(sess *fakeSession, n *Npc, cmd string) []testclient.Packet {
	e.t.Helper()
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_"+cmd)
}

// shopBuy sends RequestBuyItem.
func (e *itemEnv) shopBuy(sess *fakeSession, listID int32, items ...clientpackets.BuyItem) []testclient.Packet {
	e.t.Helper()
	w := packets.NewWriter(8 + 8*len(items))
	w.WriteInt32(listID)
	w.WriteInt32(int32(len(items))) //nolint:gosec // short test lists
	for _, it := range items {
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
	}
	return e.handle(sess, clientpackets.OpRequestBuyItem, w.Bytes())
}

// shopSell sends RequestSellItem.
func (e *itemEnv) shopSell(sess *fakeSession, listID int32, items ...clientpackets.SellItem) []testclient.Packet {
	e.t.Helper()
	w := packets.NewWriter(8 + 12*len(items))
	w.WriteInt32(listID)
	w.WriteInt32(int32(len(items))) //nolint:gosec // short test lists
	for _, it := range items {
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
	}
	return e.handle(sess, clientpackets.OpRequestSellItem, w.Bytes())
}

// shopAdena returns p's adena, read on the loop.
func (e *itemEnv) shopAdena(p *Player) int64 {
	var n int64
	e.onLoop(func() { n = p.inv.Adena() })
	return n
}

// countOf returns how many of the template p carries, worn ones included,
// read on the loop. The multisell and warehouse tests use it too.
func (e *itemEnv) countOf(p *Player, itemID int32) int64 {
	var n int64
	e.onLoop(func() { n = p.inv.Count(itemID, -1, true) })
	return n
}

// shopMove puts p at pos.
func (e *itemEnv) shopMove(p *Player, pos world.Position) {
	e.t.Helper()
	e.onLoop(func() {
		if err := e.srv.MoveObject(p, pos); err != nil {
			e.t.Errorf("MoveObject: %v", err)
		}
	})
}

// shopOne returns the only packet of type T in ps.
func shopOne[T testclient.Packet](t *testing.T, what string, ps []testclient.Packet) T {
	t.Helper()
	all := only[T](ps)
	if len(all) != 1 {
		t.Fatalf("%s: got %q, want one %T", what, testclient.Names(ps), *new(T))
	}
	return all[0]
}
