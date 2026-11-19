package mongostore

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/IPeter/l2go/internal/storage"
)

// These tests need no database: they check the stored forms through BSON.

func TestItemDocRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		in      storage.Item
		want    storage.Item // as Get returns it
		locData int32
	}{
		{
			name:    "worn",
			in:      storage.Item{ObjectID: 11, ItemID: 2369, Count: 1, Enchant: 4, Slot: storage.PaperdollRHand, Mana: storage.NoMana},
			want:    storage.Item{ObjectID: 11, ItemID: 2369, Count: 1, Enchant: 4, Slot: storage.PaperdollRHand, Loc: storage.LocPaperdoll, Mana: storage.NoMana},
			locData: 7,
		},
		{
			name: "worn under, slot 0",
			in:   storage.Item{ObjectID: 12, ItemID: 1146, Count: 1, Slot: storage.PaperdollUnder, Loc: storage.LocPaperdoll, Mana: storage.NoMana},
			want: storage.Item{ObjectID: 12, ItemID: 1146, Count: 1, Slot: storage.PaperdollUnder, Loc: storage.LocPaperdoll, Mana: storage.NoMana},
		},
		{
			name: "carried",
			in:   storage.Item{ObjectID: 13, ItemID: 57, Count: 5_000_000_000_000, Slot: storage.PaperdollNone, Mana: storage.NoMana},
			want: storage.Item{ObjectID: 13, ItemID: 57, Count: 5_000_000_000_000, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: storage.NoMana},
		},
		{
			name: "warehouse, zero slot",
			in:   storage.Item{ObjectID: 14, ItemID: 1835, Count: 300, Loc: storage.LocWarehouse, Mana: storage.NoMana, CustomType1: 1, CustomType2: 2},
			want: storage.Item{ObjectID: 14, ItemID: 1835, Count: 300, Slot: storage.PaperdollNone, Loc: storage.LocWarehouse, Mana: storage.NoMana, CustomType1: 1, CustomType2: 2},
		},
		{
			name: "shadow item",
			in:   storage.Item{ObjectID: 15, ItemID: 6730, Count: 1, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: 30},
			want: storage.Item{ObjectID: 15, ItemID: 6730, Count: 1, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: 30},
		},
	}
	for _, tt := range tests {
		d := toItemDoc(99, tt.in)
		if d.OwnerID != 99 || d.LocData != tt.locData || d.Loc != string(tt.want.Loc) {
			t.Errorf("%s: document owner %d, loc %s, loc_data %d; want 99, %s, %d", tt.name, d.OwnerID, d.Loc, d.LocData, tt.want.Loc, tt.locData)
		}
		raw, err := bson.Marshal(d)
		if err != nil {
			t.Fatalf("%s: Marshal: %v", tt.name, err)
		}
		var back itemDoc
		if err := bson.Unmarshal(raw, &back); err != nil {
			t.Fatalf("%s: Unmarshal: %v", tt.name, err)
		}
		if got := back.item(); got != tt.want {
			t.Errorf("%s: read back %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestItemDocLocData(t *testing.T) {
	tests := []struct {
		name string
		doc  itemDoc
		want storage.Item
	}{
		{
			// L2Go never writes one; it loads as a carried item.
			name: "paperdoll slot out of range",
			doc:  itemDoc{ObjectID: 1, Loc: "PAPERDOLL", LocData: 40, ManaLeft: -1},
			want: storage.Item{ObjectID: 1, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: -1},
		},
		{
			name: "loc_data of a carried item",
			doc:  itemDoc{ObjectID: 2, Loc: "INVENTORY", LocData: 7, ManaLeft: -1},
			want: storage.Item{ObjectID: 2, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: -1},
		},
		{
			name: "loc_data of a warehouse item",
			doc:  itemDoc{ObjectID: 3, Loc: "WAREHOUSE", LocData: 7, ManaLeft: -1},
			want: storage.Item{ObjectID: 3, Slot: storage.PaperdollNone, Loc: storage.LocWarehouse, Mana: -1},
		},
	}
	for _, tt := range tests {
		if got := tt.doc.item(); got != tt.want {
			t.Errorf("%s: item() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// TestLegacyItems decodes items as L2Go stored them inside a character
// before Milestone 5 (the slot as an int8) and checks what the migration
// makes of them.
func TestLegacyItems(t *testing.T) {
	type m4ItemDoc struct {
		ObjectID int32 `bson:"object_id"`
		ItemID   int32 `bson:"item_id"`
		Count    int64 `bson:"count"`
		Enchant  int32 `bson:"enchant"`
		Slot     int8  `bson:"slot"`
	}
	raw, err := bson.Marshal(bson.D{{Key: "items", Value: []m4ItemDoc{
		{ObjectID: 11, ItemID: 2369, Count: 1, Enchant: 2, Slot: 7},
		{ObjectID: 12, ItemID: 57, Count: 3_000_000_000, Slot: -1},
		{ObjectID: 13, ItemID: 1146, Count: 1, Slot: 0},
		{ObjectID: 14, ItemID: 10, Count: 1, Slot: 17},
	}}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var doc struct {
		Items []legacyItemDoc `bson:"items"`
	}
	if err := bson.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	want := []storage.Item{
		{ObjectID: 11, ItemID: 2369, Count: 1, Enchant: 2, Slot: storage.PaperdollRHand, Loc: storage.LocPaperdoll, Mana: storage.NoMana},
		{ObjectID: 12, ItemID: 57, Count: 3_000_000_000, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: storage.NoMana},
		{ObjectID: 13, ItemID: 1146, Count: 1, Slot: storage.PaperdollUnder, Loc: storage.LocPaperdoll, Mana: storage.NoMana},
		{ObjectID: 14, ItemID: 10, Count: 1, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: storage.NoMana},
	}
	if len(doc.Items) != len(want) {
		t.Fatalf("decoded %d items, want %d", len(doc.Items), len(want))
	}
	for i := range want {
		if got := doc.Items[i].item(); got != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, got, want[i])
		}
	}
}

func TestTransactionsError(t *testing.T) {
	tests := []struct {
		name  string
		hello helloReply
		ok    bool
		say   string // part of the error
	}{
		{"replica set primary", helloReply{SetName: "rs0", IsWritablePrimary: true}, true, ""},
		{"mongos", helloReply{Msg: "isdbgrid", IsWritablePrimary: true}, true, ""},
		{"secondary", helloReply{SetName: "rs0"}, false, "not the primary"},
		{"replica set not initiated", helloReply{IsReplicaSet: true}, false, "rs.initiate"},
		{"standalone", helloReply{IsWritablePrimary: true}, false, "--replSet rs0"},
	}
	for _, tt := range tests {
		err := tt.hello.transactionsError()
		if (err == nil) != tt.ok {
			t.Errorf("%s: error = %v, want ok %v", tt.name, err, tt.ok)
			continue
		}
		if err != nil && !strings.Contains(err.Error(), tt.say) {
			t.Errorf("%s: error %q does not say %q", tt.name, err, tt.say)
		}
	}
}
