package scenario

import (
	"context"
	"fmt"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// The Milestone 5 scenarios (Plan 08, "Test strategy"): equipment, ground
// items, shops, multisell and the warehouse. Except items-equip, which
// uses the starting items, each starts with a GM setup (Session.Prepare)
// that gives the items with //create_item and moves the character next to
// the NPC with //move_to, so that both servers start from the same state.

// useItem is a step in which p uses (equips or takes off) the inventory
// item *it and waits for its InventoryUpdate entry. it is read when the
// step runs, so an earlier step can fill it in.
func useItem(p *Player, name string, it *testclient.Item) Step {
	return Step{name, func(ctx context.Context) error {
		if err := p.Game().UseItem(ctx, it.ObjectID); err != nil {
			return err
		}
		_, err := p.Game().AwaitItemChange(ctx, it.ObjectID)
		return err
	}}
}

// findItems is a step that looks up the first inventory item of each item
// ID into the matching *testclient.Item.
func findItems(p *Player, items map[int32]*testclient.Item) Step {
	return Step{p.Role + " finds the items in the inventory", func(context.Context) error {
		for id, dst := range items {
			it, err := inventoryItem(p, id)
			if err != nil {
				return err
			}
			*dst = it
		}
		return nil
	}}
}

func itemsEquip(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var sword, dagger, pants testclient.Item
	return s.Steps(ctx,
		Idle(time.Second),
		findItems(a, map[int32]*testclient.Item{itemSquireSword: &sword, itemDagger: &dagger, itemSquirePants: &pants}),
		Step{"alice takes off the sword (RequestUnEquipItem)", func(ctx context.Context) error {
			if err := a.Game().UnequipItem(ctx, testclient.PaperdollRHand); err != nil {
				return err
			}
			_, err := a.Game().AwaitItemChange(ctx, sword.ObjectID)
			return err
		}},
		Idle(settle),
		useItem(a, "alice equips the dagger", &dagger),
		Idle(settle),
		useItem(a, "alice equips the sword, which takes the dagger off", &sword),
		Idle(settle),
		useItem(a, "alice takes off the pants (UseItem)", &pants),
		Idle(settle),
		useItem(a, "alice puts the pants on again", &pants),
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func itemsBow(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	err := s.Prepare(ctx, a,
		giveItem(a, itemBow, 1),
		giveItem(a, itemWoodenArrow, 100),
	)
	if err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var bow, arrows testclient.Item
	equipped := func(ctx context.Context, it *testclient.Item, want bool) error {
		return poll(ctx, func() bool {
			cur, ok := a.Game().World().Item(it.ObjectID)
			return ok && cur.Equipped == want
		})
	}
	return s.Steps(ctx,
		Idle(time.Second),
		findItems(a, map[int32]*testclient.Item{itemBow: &bow, itemWoodenArrow: &arrows}),
		useItem(a, "alice equips the bow", &bow),
		Step{"the arrows go on with the bow", func(ctx context.Context) error { return equipped(ctx, &arrows, true) }},
		Idle(settle),
		useItem(a, "alice takes off the bow", &bow),
		Step{"the arrows come off with the bow", func(ctx context.Context) error { return equipped(ctx, &arrows, false) }},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

// dropAdena is how much adena items-drop-pickup drops of the 1,000 it gives.
const dropAdena = 100

func itemsDropPickup(ctx context.Context, s *Session) error {
	a, b := s.Player("alice"), s.Player("bob")
	if err := s.Prepare(ctx, a, giveItem(a, itemAdena, 1000), gmMoveTo(a, spotDropAlice)); err != nil {
		return err
	}
	if err := s.Prepare(ctx, b, gmMoveTo(b, spotDropBob)); err != nil {
		return err
	}
	if err := enterBoth(ctx, s, a, b); err != nil {
		return err
	}
	var dropped testclient.DropItem
	err := s.Steps(ctx,
		Step{fmt.Sprintf("alice drops %d of her adena", dropAdena), func(ctx context.Context) error {
			adena, err := inventoryItem(a, itemAdena)
			if err != nil {
				return err
			}
			pos := a.Game().World().Me().Pos
			dropped, err = a.Game().DropItem(ctx, adena.ObjectID, dropAdena, pos.X, pos.Y, pos.Z)
			return err
		}},
		Step{"bob sees the adena fall", func(ctx context.Context) error {
			_, err := testclient.Await(ctx, b.Game(), func(d testclient.DropItem) bool { return d.ObjectID == dropped.ObjectID })
			return err
		}},
		Idle(settle),
		Step{"bob picks the adena up", func(ctx context.Context) error {
			_, err := b.Game().PickUp(ctx, dropped.ObjectID)
			return err
		}},
		Step{"alice sees bob pick it up", func(ctx context.Context) error {
			_, err := testclient.Await(ctx, a.Game(), func(g testclient.GetItem) bool { return g.ObjectID == dropped.ObjectID })
			return err
		}},
		Idle(time.Second),
	)
	if err != nil {
		return err
	}
	return leaveBoth(ctx, s, a, b)
}

// Purchases and sales of shop-buy-sell.
var (
	shopBuy = []testclient.BuyItem{{ItemID: itemWoodenArrow, Count: 20}, {ItemID: itemHealingPotion, Count: 2}}
	// shopSell is what is sold back: item ID and count.
	shopSell = []testclient.BuyItem{{ItemID: itemWoodenArrow, Count: 10}, {ItemID: itemHealingPotion, Count: 1}}
)

func shopBuySell(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	if err := s.Prepare(ctx, a, giveItem(a, itemAdena, 10000), gmMoveTo(a, spotKaterina)); err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var (
		page testclient.NpcHtmlMessage
		buy  testclient.BuyList
		sell testclient.SellList
	)
	talk := Step{"alice talks to Katerina", func(ctx context.Context) error {
		var err error
		page, err = talkTo(ctx, s, a, npcKaterina)
		return err
	}}
	return s.Steps(ctx,
		Idle(time.Second),
		talk,
		Step{"alice opens the buy list", func(ctx context.Context) error {
			if err := openLink(ctx, a, page, "_Buy"); err != nil {
				return err
			}
			var err error
			buy, err = testclient.Await[testclient.BuyList](ctx, a.Game(), nil)
			if err == nil {
				s.Note("buy list %d: %d products", buy.ListID, len(buy.Items))
			}
			return err
		}},
		Idle(settle),
		Step{"alice buys arrows and potions", func(ctx context.Context) error {
			_, err := a.Game().BuyItems(ctx, buy.ListID, shopBuy)
			return err
		}},
		Idle(floodGap),
		talk,
		Step{"alice opens the sell list", func(ctx context.Context) error {
			if err := openLink(ctx, a, page, "_Sell"); err != nil {
				return err
			}
			var err error
			sell, err = testclient.Await[testclient.SellList](ctx, a.Game(), nil)
			return err
		}},
		Idle(settle),
		Step{"alice sells some of them back", func(ctx context.Context) error {
			var items []testclient.SellItem
			for _, it := range shopSell {
				inv, err := inventoryItem(a, it.ItemID)
				if err != nil {
					return err
				}
				items = append(items, testclient.SellItem{ObjectID: inv.ObjectID, ItemID: it.ItemID, Count: it.Count})
			}
			_, err := a.Game().SellItems(ctx, sell.ListID, items)
			return err
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

func multisellGalladucci(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	err := s.Prepare(ctx, a,
		giveItem(a, itemCrystalC, 573),
		giveItem(a, itemCrystalD, 2865),
		gmMoveTo(a, spotGalladucci),
	)
	if err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var (
		page testclient.NpcHtmlMessage
		list testclient.MultiSellList
	)
	return s.Steps(ctx,
		Idle(time.Second),
		Step{"alice talks to Galladucci", func(ctx context.Context) error {
			var err error
			page, err = talkTo(ctx, s, a, npcGalladucci)
			return err
		}},
		Step{"alice opens the multisell list", func(ctx context.Context) error {
			if err := openLink(ctx, a, page, "multisell"); err != nil {
				return err
			}
			var err error
			list, err = a.Game().AwaitMultiSellList(ctx)
			return err
		}},
		Idle(settle),
		Step{"alice exchanges the crystals for the first entry", func(ctx context.Context) error {
			if len(list.Entries) == 0 {
				return fmt.Errorf("multisell list %d has no entries", list.ListID)
			}
			e := list.Entries[0]
			if len(e.Products) > 0 {
				s.Note("multisell %d entry %d: item %d", list.ListID, e.EntryID, e.Products[0].ItemID)
			}
			_, err := a.Game().MultiSellChoose(ctx, list.ListID, e.EntryID, 1)
			return err
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}

// Deposits of warehouse-private: item ID and count, withdrawn again.
var warehouseItems = []testclient.BuyItem{{ItemID: itemAdena, Count: 1000}, {ItemID: itemWoodenArrow, Count: 50}}

func warehousePrivate(ctx context.Context, s *Session) error {
	a := s.Player("alice")
	err := s.Prepare(ctx, a,
		giveItem(a, itemAdena, 10000),
		giveItem(a, itemWoodenArrow, 100),
		gmMoveTo(a, spotWilford),
	)
	if err != nil {
		return err
	}
	if err := s.Enter(ctx, a); err != nil {
		return err
	}
	var (
		page     testclient.NpcHtmlMessage
		withdraw testclient.WareHouseWithdrawalList
	)
	talk := Step{"alice talks to Wilford", func(ctx context.Context) error {
		var err error
		page, err = talkTo(ctx, s, a, npcWilford)
		return err
	}}
	return s.Steps(ctx,
		Idle(time.Second),
		talk,
		Step{"alice opens the deposit window", func(ctx context.Context) error {
			if err := openLink(ctx, a, page, "_DepositP"); err != nil {
				return err
			}
			_, err := testclient.Await[testclient.WareHouseDepositList](ctx, a.Game(), nil)
			return err
		}},
		Idle(settle),
		Step{"alice deposits adena and arrows", func(ctx context.Context) error {
			var items []testclient.WarehouseItem
			for _, it := range warehouseItems {
				inv, err := inventoryItem(a, it.ItemID)
				if err != nil {
					return err
				}
				items = append(items, testclient.WarehouseItem{ObjectID: inv.ObjectID, Count: it.Count})
			}
			_, err := a.Game().DepositItems(ctx, items)
			return err
		}},
		Idle(floodGap),
		talk,
		Step{"alice opens the withdrawal window", func(ctx context.Context) error {
			if err := openLink(ctx, a, page, "_WithdrawP"); err != nil {
				return err
			}
			var err error
			withdraw, err = testclient.Await[testclient.WareHouseWithdrawalList](ctx, a.Game(), nil)
			return err
		}},
		Idle(settle),
		Step{"alice takes them back", func(ctx context.Context) error {
			var items []testclient.WarehouseItem
			for _, it := range warehouseItems {
				stored, ok := withdraw.FindItem(it.ItemID)
				if !ok {
					return fmt.Errorf("the warehouse has no item %d", it.ItemID)
				}
				items = append(items, testclient.WarehouseItem{ObjectID: stored.ObjectID, Count: it.Count})
			}
			_, _ = a.Game().Drain()
			_, err := a.Game().WithdrawItems(ctx, items)
			return err
		}},
		Idle(time.Second),
		Step{"alice logs out", a.Logout},
	)
}
