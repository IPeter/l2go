package game

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/inventory"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// errInjectedSave is the error of a save the recording store fails.
var errInjectedSave = errors.New("injected save failure")

var (
	loadTestPassives = sync.OnceValues(func() (*data.Passives, error) {
		return data.LoadPassives(filepath.Join("..", "..", "data"))
	})
	loadTestCategories = sync.OnceValues(func() (*data.Categories, error) {
		return data.LoadCategories(filepath.Join("..", "..", "data"))
	})
)

func testPassives(t *testing.T) *data.Passives {
	t.Helper()
	p, err := loadTestPassives()
	if err != nil {
		t.Fatalf("load the passive skills: %v", err)
	}
	return p
}

func testCategories(t *testing.T) *data.Categories {
	t.Helper()
	c, err := loadTestCategories()
	if err != nil {
		t.Fatalf("load the categories: %v", err)
	}
	return c
}

// testLimits are generous inventory limits for inventories without a
// player.
type testLimits struct{}

func (testLimits) InventoryLimit() int { return 100 }
func (testLimits) WeightLimit() int    { return 1 << 30 }

// Items the item tests use (Mobius item IDs).
const (
	itemAdena          = 57
	itemBow            = 14   // Bow (lrhand, no grade)
	itemWoodenArrow    = 17   // Wooden Arrow (no grade)
	itemLeatherShield  = 18   // Leather Shield (lhand)
	itemLongSword      = 2    // Long Sword (rhand)
	itemClaymore       = 70   // Claymore (lrhand)
	itemLeatherShirt   = 22   // Leather Shirt (chest)
	itemLeatherPants   = 29   // Leather Pants (legs)
	itemCompositeArmor = 60   // Composite Armor (onepiece, full armor)
	itemEarring        = 112  // Apprentice's Earring
	itemMagicRing      = 116  // Magic Ring
	itemFishingRod     = 6529 // Baby Duck Rod
	itemLure           = 6519 // Green Colored Lure - Low Grade
	itemFormalWear     = 6408 // Formal Wear (alldress)
	itemLeatherHelmet  = 44   // Leather Helmet (head)
	itemTutorialGuide  = 5588 // a Book (html/help/5588.htm)
	itemShadowSword    = 8822 // Shadow Item: Crimson Sword (mana 90)
	itemMithrilOre     = 1876 // Mithril Ore: stackable, weight 2
	itemDarinsLetter   = 687  // a quest item
)

// itemEnv is a test server with players whose items the tests change.
type itemEnv struct {
	*testEnv
}

// bookPage is the help page of itemTutorialGuide in the item tests.
const bookPage = "<html><body>How to play.</body></html>"

func newItemEnv(t *testing.T, configure func(gs *config.GameServer)) *itemEnv {
	t.Helper()
	return &itemEnv{testEnv: newDialogEnv(t, dialogEnvOptions{
		pages:     map[string]string{"help/5588.htm": bookPage},
		configure: configure,
	})}
}

// enterWith enters a Human Fighter with extra items at pos (zero for the
// first creation point).
func (e *itemEnv) enterWith(account string, pos world.Position, items ...storage.Item) (*fakeSession, *Player) {
	e.t.Helper()
	for i := range items {
		if items[i].Slot == 0 && items[i].Loc == "" {
			items[i].Slot = storage.PaperdollNone
		}
		if items[i].Mana == 0 {
			items[i].Mana = storage.NoMana
		}
	}
	return e.enterChar(testChar{account: account, name: account, pos: pos, items: items})
}

// item returns an item of the template in p's inventory, read on the loop.
func (e *itemEnv) itemOf(p *Player, itemID int32) *inventory.Item {
	var it *inventory.Item
	e.onLoop(func() { it = p.inv.ByItemID(itemID) })
	return it
}

// handle sends a client packet of p and waits until the loop ran it.
func (e *itemEnv) handle(sess *fakeSession, op byte, body []byte) []testclient.Packet {
	e.t.Helper()
	e.srv.Handle(sess, op, body)
	e.sync()
	return sess.take(e.t)
}

// useItem sends UseItem.
func (e *itemEnv) useItem(sess *fakeSession, objectID int32) []testclient.Packet {
	e.t.Helper()
	return e.handle(sess, clientpackets.OpUseItem, ints(objectID, 0))
}

// ints encodes a client packet body of ints.
func ints(vs ...int32) []byte {
	w := packets.NewWriter(4 * len(vs))
	for _, v := range vs {
		w.WriteInt32(v)
	}
	return w.Bytes()
}

// sysMsgIDsOf returns the IDs of the system messages among ps.
func sysMsgIDsOf(ps []testclient.Packet) []int32 {
	var ids []int32
	for _, m := range only[testclient.SystemMessage](ps) {
		ids = append(ids, m.ID)
	}
	return ids
}

// wornIn returns the template ID worn in slot, 0 if none, read on the loop.
func (e *itemEnv) wornIn(p *Player, slot storage.PaperdollSlot) int32 {
	var id int32
	e.onLoop(func() {
		if it := p.inv.Paperdoll(slot); it != nil {
			id = it.ItemID()
		}
	})
	return id
}

// give creates items in p's inventory on the loop and returns the last.
func (e *itemEnv) give(p *Player, itemID int32, count int64) *inventory.Item {
	e.t.Helper()
	var it *inventory.Item
	var err error
	e.onLoop(func() {
		it, err = p.inv.Create(itemID, count)
		e.srv.ItemsChanged(p)
	})
	if err != nil {
		e.t.Fatalf("create item %d: %v", itemID, err)
	}
	return it
}
