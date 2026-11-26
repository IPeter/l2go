package main

import (
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/testclient"
)

func TestStoreKeeper(t *testing.T) {
	var keepers []int
	for id := range 30 {
		if storeKeeper(id, 0.1) {
			keepers = append(keepers, id)
		}
	}
	if !slices.Equal(keepers, []int{9, 19, 29}) {
		t.Errorf("keepers of 0.1 = %v, want 9, 19, 29", keepers)
	}
	for id := range 5 {
		if storeKeeper(id, 0) || !storeKeeper(id, 1) {
			t.Errorf("bot %d: share 0 or 1 gives the wrong answer", id)
		}
	}
}

func TestNearestStore(t *testing.T) {
	me := testclient.Position{}
	player := func(id, x int32, store uint8) testclient.Object {
		o := testclient.Object{ObjectID: id, Kind: testclient.KindPlayer, Info: testclient.CharInfoPacket{PrivateStoreType: store}}
		o.Pos = testclient.Position{X: x}
		return o
	}
	objs := []testclient.Object{
		player(1, 50, 0),                 // no store
		player(2, 300, storeTypeSell),    // a sell store
		player(3, 200, 3),                // a buy store
		player(4, 300, storeTypePackage), // a package sale, as near as 2
		player(5, 2000, storeTypeSell),   // too far
	}
	o, ok := nearestStore(me, objs, 1500)
	if !ok || o.ObjectID != 2 {
		t.Errorf("nearestStore = %d, %v; want 2", o.ObjectID, ok)
	}
	if _, ok := nearestStore(me, objs[:1], 1500); ok {
		t.Error("a player without a store was taken")
	}
}

func TestPurchase(t *testing.T) {
	l := testclient.PrivateStoreListSell{Adena: 1000, Items: []testclient.StoreSellItem{
		{ObjectID: 7, Count: 20, Price: 10},
		{ObjectID: 8, Count: 1, Price: 500},
	}}
	if got := purchase(l, 3); !slices.Equal(got, []testclient.StorePurchase{{ObjectID: 7, Count: 3, Price: 10}}) {
		t.Errorf("purchase = %+v, want 3 of the first item", got)
	}
	l.PackageSale = true
	want := []testclient.StorePurchase{{ObjectID: 7, Count: 20, Price: 10}, {ObjectID: 8, Count: 1, Price: 500}}
	if got := purchase(l, 3); !slices.Equal(got, want) {
		t.Errorf("package purchase = %+v, want everything", got)
	}
	l.Adena = 600
	if got := purchase(l, 3); got != nil {
		t.Errorf("a package over the adena: %+v, want nothing", got)
	}
	if got := purchase(testclient.PrivateStoreListSell{Adena: 1000}, 3); got != nil {
		t.Errorf("an empty store: %+v", got)
	}
}

func TestParseFlagsStores(t *testing.T) {
	cfg, err := parseFlags([]string{"-stores", "0.1", "-buy", "0.2"}, io.Discard)
	if err != nil || cfg.stores != 0.1 || cfg.buy != 0.2 {
		t.Fatalf("store flags: %+v, %v", cfg, err)
	}
	for _, args := range [][]string{
		{"-stores", "-0.1"},
		{"-stores", "1.5"},
		{"-buy", "-0.1"},
		{"-say", "0.5", "-talk", "0.3", "-buy", "0.3"},
	} {
		if _, err := parseFlags(args, io.Discard); err == nil {
			t.Errorf("parseFlags(%q) accepted it", args)
		}
	}
	logger := slog.New(slog.DiscardHandler)
	if b := newBot(9, cfg, nil, logger); !b.keeper {
		t.Error("bot 9 of 0.1 does not keep a store")
	}
	cfg.mode = modeHunt
	if b := newBot(9, cfg, nil, logger); b.keeper {
		t.Error("a hunting bot keeps a store")
	}
}
