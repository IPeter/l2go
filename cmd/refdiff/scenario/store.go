package scenario

import (
	"context"
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The Milestone 9 private store scenario (Plan 12 Phase 6): a sell store,
// a purchase from it and the store closing when it is sold out.

// itemStem is Stem, a stackable, tradeable material the store sells.
const itemStem int32 = 1864

// The sell store of private-store: what it offers, at which price, and what
// the first purchase takes (the second takes the rest).
const (
	storeStems      = 10
	storeStemPrice  = 5
	storeFirstBuy   = 4
	storeMessageTxt = "refdiff stems"
)

// System messages of a purchase: "$s1 purchased $s3 $s2(s)." to the store
// owner, "You have purchased $s3 $s2(s) from $s1." to the buyer.
const (
	msgPurchasedCount    int32 = 380
	msgYouPurchasedCount int32 = 561
)

// awaitSysMsg waits for p to receive the system message id.
func awaitSysMsg(ctx context.Context, p *Player, id int32) error {
	_, err := testclient.Await(ctx, p.Game(), func(m testclient.SystemMessage) bool { return m.ID == id })
	if err != nil {
		return fmt.Errorf("%s did not get system message %d: %w", p.Role, id, err)
	}
	return nil
}

// buyStems is a step in which bob buys count Stems from alice's store and
// both get the purchase message.
func buyStems(a, b *Player, stems *testclient.Item, count int32) Step {
	return Step{fmt.Sprintf("bob buys %d Stems", count), func(ctx context.Context) error {
		err := b.Game().BuyFromStore(ctx, a.ObjectID(), []testclient.StorePurchase{
			{ObjectID: stems.ObjectID, Count: count, Price: storeStemPrice},
		})
		if err != nil {
			return err
		}
		if err := awaitSysMsg(ctx, b, msgYouPurchasedCount); err != nil {
			return err
		}
		return awaitSysMsg(ctx, a, msgPurchasedCount)
	}}
}

func privateStore(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := s.Prepare(ctx, a, giveItem(a, itemStem, 20), gmMoveTo(a, spotDropAlice)); err != nil {
		return err
	}
	if err := s.Prepare(ctx, b, giveItem(b, itemAdena, 1000), gmMoveTo(b, spotDropBob)); err != nil {
		return err
	}
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	var stems testclient.Item
	err := s.Steps(ctx,
		findItems(a, map[int32]*testclient.Item{itemStem: &stems}),
		Step{"alice opens the setup window of a sell store", func(ctx context.Context) error {
			_, err := a.Game().OpenSellStoreSetup(ctx)
			return err
		}},
		Idle(settle),
		Step{"alice sets the store message", func(ctx context.Context) error {
			if err := a.Game().SetSellStoreMessage(ctx, storeMessageTxt); err != nil {
				return err
			}
			_, err := testclient.Await(ctx, a.Game(), func(m testclient.PrivateStoreMsgSell) bool { return m.Message == storeMessageTxt })
			return err
		}},
		Step{"alice opens the store", func(ctx context.Context) error {
			_, err := a.Game().OpenSellStore(ctx, false, []testclient.StoreSellOffer{
				{ObjectID: stems.ObjectID, Count: storeStems, Price: storeStemPrice},
			})
			return err
		}},
		Step{"bob sees the store message", func(ctx context.Context) error {
			id := a.ObjectID()
			_, err := testclient.Await(ctx, b.Game(), func(m testclient.PrivateStoreMsgSell) bool { return m.ObjectID == id })
			return err
		}},
		Idle(time.Second),
		Step{"bob targets alice", func(ctx context.Context) error {
			_, err := b.Game().Target(ctx, a.ObjectID())
			return err
		}},
		Step{"bob walks up to the store and opens it", func(ctx context.Context) error {
			l, err := b.Game().ViewSellStore(ctx, a.ObjectID())
			if err == nil {
				s.Note("store window: %d items, bob has %d adena", len(l.Items), l.Adena)
			}
			return err
		}},
		Idle(settle),
		buyStems(a, b, &stems, storeFirstBuy),
		Idle(floodGap),
		buyStems(a, b, &stems, storeStems-storeFirstBuy),
		Step{"the sold-out store closes", func(ctx context.Context) error {
			_, err := testclient.Await(ctx, a.Game(), func(u testclient.UserInfo) bool { return u.PrivateStoreType == 0 })
			return err
		}},
		Idle(time.Second),
	)
	if err != nil {
		return err
	}
	return leaveBoth(ctx, s, a, b)
}
