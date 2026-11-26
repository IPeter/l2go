package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The private stores of wandering bots (-stores, -buy): a share of the bots
// keep a sell store where they stand, and the others now and then walk up to
// the nearest store and buy from it. Both get their items and adena with the
// GM command //create_item, so the characters need GM rights.

// The goods of the store keepers.
const (
	// storeGoods is what a keeper sells: Stem (1864), a stackable,
	// tradeable material.
	storeGoods int32 = 1864
	// storeStock is how many Stems a keeper creates when it has too few
	// to fill its store.
	storeStock = 100
	// storeOffer is how many Stems a keeper's store offers.
	storeOffer = 20
	// storePrice is the unit price of a Stem in a keeper's store.
	storePrice = 10
	// storeMessage is the message over a keeper's store.
	storeMessage = "Stems, cheap"
	// shopMaxCount is the most Stems a buyer takes at once.
	shopMaxCount = 5
	// shopAdena is the adena a buyer creates when it has too little, and
	// shopMinAdena that limit.
	shopAdena    = 100000
	shopMinAdena = 1000
)

// adenaID is the item ID of adena.
const adenaID int32 = 57

// The store types of UserInfo and CharInfo (Mobius PrivateStoreType).
const (
	storeTypeSell    uint8 = 1
	storeTypePackage uint8 = 8
)

// System messages a buyer waits for: "You have purchased $s2 from $s1." and
// "You have purchased $s3 $s2(s) from $s1.".
const (
	msgYouPurchased      int32 = 559
	msgYouPurchasedCount int32 = 561
)

// storeKeeper reports whether bot id keeps a store when share of the bots
// do: one bot in every 1/share, spread evenly over the IDs.
func storeKeeper(id int, share float64) bool {
	if share <= 0 {
		return false
	}
	return math.Floor(float64(id+1)*share) > math.Floor(float64(id)*share)
}

// keepStore is one action of a store keeper: nothing while its store is
// open; otherwise it creates Stems when it has too few, opens the setup
// window of a sell store and opens the store ("store": the setup window to
// the store message). Where no store may open (a no-store zone, another
// store or an NPC too close) it walks elsewhere and tries again next time.
func (b *bot) keepStore(ctx context.Context, gc *testclient.GameConn) error {
	w := gc.World()
	if w.Me().UserInfo.PrivateStoreType == storeTypeSell || b.position(time.Now()) != b.to {
		return nil // the store is open, or the bot still walks
	}
	if time.Now().Before(b.storeRetryAt) {
		return b.walk(ctx, gc)
	}
	if w.ItemCount(storeGoods) < storeOffer {
		if err := b.create(ctx, gc, storeGoods, storeStock); err != nil {
			return staged("store", err)
		}
	}
	t0 := time.Now()
	if err := gc.RequestActionUse(ctx, testclient.ActionPrivateStoreSell, false, false); err != nil {
		return staged("store", err)
	}
	p, err := gc.WaitFor(ctx, func(p testclient.Packet) bool {
		switch p.(type) {
		case testclient.PrivateStoreManageListSell, testclient.ActionFailed:
			return true
		}
		return false
	})
	if err != nil {
		return staged("store", err)
	}
	m, ok := p.(testclient.PrivateStoreManageListSell)
	if !ok {
		b.stats.Count("store refused")
		return b.walk(ctx, gc)
	}
	var offer *testclient.StoreSellItem
	for i := range m.Available {
		if m.Available[i].ItemID == storeGoods && m.Available[i].Count >= storeOffer {
			offer = &m.Available[i]
			break
		}
	}
	if offer == nil {
		_ = gc.QuitSellStore(ctx)
		return staged("store", fmt.Errorf("%w: no %d Stems to offer", errRefused, storeOffer))
	}
	if err := gc.SetSellStoreMessage(ctx, storeMessage); err != nil {
		return staged("store", err)
	}
	if _, err := gc.OpenSellStore(ctx, false, []testclient.StoreSellOffer{
		{ObjectID: offer.ObjectID, Count: storeOffer, Price: storePrice},
	}); err != nil {
		return staged("store", err)
	}
	b.stats.Observe("store", time.Since(t0))
	b.stats.Count("store")
	return nil
}

// shop is a purchase at the nearest store within -talk-range ("shop": the
// second click, the walk included, to the store window; "buy": the purchase
// to its message): the bot buys up to shopMaxCount of the store's first item,
// or a whole package sale. Without a store in reach it walks.
func (b *bot) shop(ctx context.Context, gc *testclient.GameConn) error {
	w := gc.World()
	store, ok := nearestStore(w.Me().Pos, w.Players(), b.cfg.talkRange)
	if !ok {
		return b.walk(ctx, gc)
	}
	if w.ItemCount(adenaID) < shopMinAdena {
		if time.Now().Before(b.storeRetryAt) {
			return b.walk(ctx, gc)
		}
		if err := b.create(ctx, gc, adenaID, shopAdena); err != nil {
			return staged("shop", err)
		}
	}
	if w.Me().TargetID != store.ObjectID {
		if _, err := gc.Target(ctx, store.ObjectID); err != nil {
			return staged("target", err)
		}
	}
	t0 := time.Now()
	l, err := gc.ViewSellStore(ctx, store.ObjectID)
	if err != nil {
		return staged("shop", err)
	}
	b.stats.Observe("shop", time.Since(t0))
	// The character walked up to the store.
	me := gc.World().Me()
	b.from, b.to, b.departed = me.Pos, me.Pos, time.Now()

	items := purchase(l, b.rng.IntN(shopMaxCount)+1)
	if len(items) == 0 {
		return nil
	}
	t1 := time.Now()
	if err := gc.BuyFromStore(ctx, store.ObjectID, items); err != nil {
		return staged("buy", err)
	}
	if _, err := testclient.Await(ctx, gc, func(m testclient.SystemMessage) bool {
		return m.ID == msgYouPurchased || m.ID == msgYouPurchasedCount
	}); err != nil {
		return staged("buy", err)
	}
	b.stats.Observe("buy", time.Since(t1))
	b.stats.Count("purchase")
	return nil
}

// purchase is what a bot buys from the store window l: everything of a
// package sale, otherwise up to count of the first item; nothing it cannot
// pay.
func purchase(l testclient.PrivateStoreListSell, count int) []testclient.StorePurchase {
	if len(l.Items) == 0 {
		return nil
	}
	var items []testclient.StorePurchase
	var total int64
	for _, it := range l.Items {
		n := it.Count
		if !l.PackageSale {
			n = min(n, int32(count)) //nolint:gosec // count is at most shopMaxCount
		}
		items = append(items, testclient.StorePurchase{ObjectID: it.ObjectID, Count: n, Price: it.Price})
		total += int64(n) * int64(it.Price)
		if !l.PackageSale {
			break
		}
	}
	if total > int64(l.Adena) {
		return nil
	}
	return items
}

// nearestStore returns the player of objs with an open sell store or
// package sale (CharInfo) nearest to me within maxDist. Ties go to the lower
// object ID.
func nearestStore(me testclient.Position, objs []testclient.Object, maxDist float64) (testclient.Object, bool) {
	var (
		best     testclient.Object
		bestDist = math.Inf(1)
		found    bool
	)
	for _, o := range objs {
		ci, ok := o.Info.(testclient.CharInfoPacket)
		if o.Kind != testclient.KindPlayer || !ok || (ci.PrivateStoreType != storeTypeSell && ci.PrivateStoreType != storeTypePackage) {
			continue
		}
		d := o.Pos.DistanceTo(me)
		if d > maxDist {
			continue
		}
		if !found || d < bestDist || (d == bestDist && o.ObjectID < best.ObjectID) {
			best, bestDist, found = o, d, true
		}
	}
	return best, found
}

// create gives the bot count of itemID with //create_item and waits until
// its inventory shows them. Without GM rights nothing comes: the bot then
// leaves stores alone for buffRetry.
func (b *bot) create(ctx context.Context, gc *testclient.GameConn, itemID int32, count int) error {
	want := gc.World().ItemCount(itemID) + int64(count)
	if err := gc.GMCommand(ctx, fmt.Sprintf("create_item %d %d", itemID, count)); err != nil {
		return err
	}
	if err := pollUntil(ctx, func() bool { return gc.World().ItemCount(itemID) >= want }); err != nil {
		b.storeRetryAt = time.Now().Add(buffRetry)
		return err
	}
	return nil
}
