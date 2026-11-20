package recording

import (
	"reflect"
	"strings"
	"testing"

	"github.com/IPeter/l2go/cmd/refdiff/scenario"
	"github.com/IPeter/l2go/internal/testclient"
)

func TestNormalizeMonsterCoordinates(t *testing.T) {
	const (
		player = int32(0x10000001)
		mob    = int32(0x60000001)
		guard  = int32(0x60000002)
	)
	c := newCapture([]scenario.Identity{alice}, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		srvPacket("CharSelected", testclient.CharSelected{ObjectID: player, X: -71338, Y: 258271, Z: -3104}),
		srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: mob, NpcTypeID: 1020481, Attackable: true, X: -74000, Y: 254000, Z: -3200, Heading: 1234}),
		srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: guard, NpcTypeID: 1030009, X: -71424, Y: 258336, Z: -3104, Heading: 42000}),
		// The client packets keep the moves apart from a fight run.
		srvPacket("MoveToPawn", testclient.MoveToPawn{ObjectID: player, TargetID: mob, Distance: 36,
			X: -71338, Y: 258271, Z: -3104, TargetX: -74000, TargetY: 254000, TargetZ: -3200}),
		cliPacket("ValidatePosition"),
		srvPacket("MoveToLocation", testclient.MoveToLocation{ObjectID: mob, DestX: -74100, DestY: 254100, DestZ: -3200, X: -74000, Y: 254000, Z: -3200}),
	}})
	n := newNormalizer(c)
	rec := Normalize(c)
	es := rec.Streams[0].Entries
	checkFields(t, es[1], map[string]string{"ObjectID": "#npc1", "X": phMonster, "Y": phMonster, "Z": phMonster, "Heading": phMonster})
	// A town NPC keeps its spawn point; the scenarios ignore it by default.
	checkFields(t, es[2], map[string]string{"X": "-71424", "Heading": "42000"})
	checkFields(t, es[3], map[string]string{"X": "start", "TargetX": phMonster, "TargetID": "#npc1", "Distance": "36"})
	checkFields(t, es[5], map[string]string{"ObjectID": "#npc1", "DestX": phMonster, "X": phMonster})

	// The monster swings at alice: its own position is random, alice's is
	// relative to her start. (A recording shows Attack only inside a
	// Fight entry; the rule keeps a fight's sort keys and -full output
	// stable.)
	ls := n.leaves("Attack", testclient.Attack{AttackerID: mob, Hits: []testclient.Hit{{TargetID: player, Damage: 3}},
		X: -74000, Y: 254000, Z: -3200, TargetX: -71330, TargetY: 258271, TargetZ: -3104})
	got := map[string]string{}
	for _, l := range ls {
		got[l.path] = l.val
	}
	for path, want := range map[string]string{"X": phMonster, "Z": phMonster, "TargetX": "start+8", "TargetY": "start"} {
		if got[path] != want {
			t.Errorf("Attack.%s = %s, want %s", path, got[path], want)
		}
	}
}

// dropCapture: alice drops part of her adena, bob sees it fall and picks
// it up; then a monster drops loot near bob. The start points are given.
func dropCapture(aStart, bStart testclient.Position, aliceID, bobID, coin, mob, loot int32) *scenario.Capture {
	drop := testclient.DropItem{DropperID: aliceID, ObjectID: coin, ItemID: 57, X: aStart.X + 5, Y: aStart.Y, Z: aStart.Z, Stackable: true, Count: 100}
	mobDrop := testclient.DropItem{DropperID: mob, ObjectID: loot, ItemID: 57, X: bStart.X - 33, Y: bStart.Y + 61, Z: bStart.Z + 20, Stackable: true, Count: 17}
	return newCapture([]scenario.Identity{alice, bob},
		scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
			srvPacket("CharSelected", testclient.CharSelected{ObjectID: aliceID, X: aStart.X, Y: aStart.Y, Z: aStart.Z}),
			srvPacket("DropItem", drop),
			srvPacket("GetItem", testclient.GetItem{PlayerID: bobID, ObjectID: coin, X: drop.X, Y: drop.Y, Z: drop.Z}),
		}},
		scenario.Stream{Name: "game bob", Packets: []scenario.Packet{
			srvPacket("CharSelected", testclient.CharSelected{ObjectID: bobID, X: bStart.X, Y: bStart.Y, Z: bStart.Z}),
			srvPacket("DropItem", drop),
			srvPacket("GetItem", testclient.GetItem{PlayerID: bobID, ObjectID: coin, X: drop.X, Y: drop.Y, Z: drop.Z}),
			srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: mob, NpcTypeID: 1020481, Attackable: true}),
			srvPacket("DropItem", mobDrop),
		}},
	)
}

func TestNormalizeDrops(t *testing.T) {
	a := Normalize(dropCapture(testclient.Position{X: -71000, Y: 258100, Z: -3104}, testclient.Position{X: -70900, Y: 258100, Z: -3104},
		0x10000001, 0x10000002, 0x10000003, 0x60000001, 0x10000004))
	b := Normalize(dropCapture(testclient.Position{X: -71300, Y: 258200, Z: -3100}, testclient.Position{X: -71250, Y: 258000, Z: -3100},
		0x20000001, 0x20000005, 0x20000002, 0x20000009, 0x20000003))
	if !reflect.DeepEqual(a.Streams, b.Streams) {
		t.Fatalf("normalized streams differ:\n%+v\n%+v", a.Streams, b.Streams)
	}
	al, bl := a.Streams[0].Entries, a.Streams[1].Entries
	checkFields(t, al[1], map[string]string{"ObjectID": "#item1", "DropperID": "#player1", "X": "start+5", "ItemID": "57", "Count": "100"})
	checkFields(t, al[2], map[string]string{"ObjectID": "#item1", "PlayerID": "#player2", "X": "start+5"})
	checkFields(t, bl[2], map[string]string{"X": "start+5"})
	checkFields(t, bl[4], map[string]string{
		"DropperID": "#npc1", "ItemID": phDrop, "Count": phDrop, "Stackable": phDrop, "X": phDrop, "Z": phDrop,
	})
}

func TestNormalizeAdena(t *testing.T) {
	const (
		player = int32(0x10000001)
		other  = int32(0x10000002)
		coin   = int32(0x10000010)
		arrows = int32(0x10000011)
	)
	adena := func(n int32) testclient.Item { return testclient.Item{ObjectID: coin, ItemID: 57, Count: n} }
	c := newCapture([]scenario.Identity{alice, bob},
		scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
			srvPacket("CharSelected", testclient.CharSelected{ObjectID: player}),
			srvPacket("ItemList", testclient.ItemList{Items: []testclient.Item{adena(10000), {ObjectID: arrows, ItemID: 17, Count: 20}}}),
			srvPacket("BuyList", testclient.BuyList{Adena: 10000, ListID: 3000400}),
			srvPacket("InventoryUpdate", testclient.InventoryUpdate{Items: []testclient.ItemChange{
				{Change: testclient.ItemModified, Item: adena(9946)},
				{Change: testclient.ItemModified, Item: testclient.Item{ObjectID: arrows, ItemID: 17, Count: 40}},
			}}),
			srvPacket("SellList", testclient.SellList{Adena: 9946}),
			srvPacket("SystemMessage", testclient.SystemMessage{ID: 28, Params: []testclient.SMParam{{Type: testclient.SMParamInt, Number: 54}}}),
		}},
		scenario.Stream{Name: "game alice 2", Packets: []scenario.Packet{
			srvPacket("ItemList", testclient.ItemList{Items: []testclient.Item{adena(9946)}}),
			srvPacket("WareHouseDepositList", testclient.WareHouseDepositList{WarehouseList: testclient.WarehouseList{
				Type: testclient.WarehousePrivate, Adena: 9946, Items: []testclient.WarehouseListItem{{ObjectID: coin, ItemID: 57, Count: 9946}},
			}}),
			srvPacket("WareHouseWithdrawalList", testclient.WareHouseWithdrawalList{WarehouseList: testclient.WarehouseList{
				Type: testclient.WarehousePrivate, Adena: 8916, Items: []testclient.WarehouseListItem{{ObjectID: coin + 1, ItemID: 57, Count: 1000}},
			}}),
		}},
		scenario.Stream{Name: "game bob", Packets: []scenario.Packet{
			srvPacket("ItemList", testclient.ItemList{Items: []testclient.Item{{ObjectID: other, ItemID: 2369, Count: 1}}}),
			srvPacket("InventoryUpdate", testclient.InventoryUpdate{Items: []testclient.ItemChange{
				{Change: testclient.ItemAdded, Item: testclient.Item{ObjectID: coin + 2, ItemID: 57, Count: 100}},
			}}),
		}},
	)
	rec := Normalize(c)
	a, a2, b := rec.Streams[0].Entries, rec.Streams[1].Entries, rec.Streams[2].Entries
	// The items are sorted by item ID: the arrows (17) before the adena.
	checkFields(t, a[1], map[string]string{"Items[0].Count": "20", "Items[1].Count": "adena0"})
	checkFields(t, a[2], map[string]string{"Adena": "adena0", "ListID": "3000400"})
	checkFields(t, a[3], map[string]string{"Items[0].Count": "40", "Items[1].Count": "adena0-54"})
	checkFields(t, a[4], map[string]string{"Adena": "adena0-54"})
	// Amounts that are no balance stay.
	checkFields(t, a[5], map[string]string{"Params[0].Number": "54"})
	// The second connection keeps the role's start.
	checkFields(t, a2[0], map[string]string{"Items[0].Count": "adena0-54"})
	checkFields(t, a2[1], map[string]string{"Adena": "adena0-54", "Items[0].Count": "adena0-54"})
	// The warehouse's own adena is no balance of the player.
	checkFields(t, a2[2], map[string]string{"Adena": "adena0-1084", "Items[0].Count": "1000"})
	checkFields(t, b[1], map[string]string{"Items[0].Count": "adena0+100"})
}

func TestNormalizeCurrentValues(t *testing.T) {
	const npc = int32(0x60000001)
	attrs := func(kv ...int32) []testclient.StatusAttribute {
		var out []testclient.StatusAttribute
		for i := 0; i+1 < len(kv); i += 2 {
			out = append(out, testclient.StatusAttribute{ID: kv[i], Value: kv[i+1]})
		}
		return out
	}
	c := newCapture(nil, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		srvPacket("StatusUpdate", testclient.StatusUpdate{ObjectID: npc, Attributes: attrs(testclient.StatusCurHP, 120, testclient.StatusMaxHP, 120)}),
		srvPacket("StatusUpdate", testclient.StatusUpdate{ObjectID: npc, Attributes: attrs(testclient.StatusMaxHP, 120, testclient.StatusCurHP, 97)}),
		srvPacket("StatusUpdate", testclient.StatusUpdate{ObjectID: npc, Attributes: attrs(testclient.StatusCurMP, 30, testclient.StatusLevel, 2)}),
		srvPacket("UserInfo", testclient.UserInfo{HP: 50, MaxHP: 80, MP: 30, MaxMP: 30, CP: 0, MaxCP: 40}),
	}})
	es := Normalize(c).Streams[0].Entries
	checkFields(t, es[0], map[string]string{"Attributes[0].Value": "120", "Attributes[1].Value": "120"})
	checkFields(t, es[1], map[string]string{"Attributes[0].Value": "120", "Attributes[1].Value": phCurrent})
	// Without the maximum in the packet the value can't be told full.
	checkFields(t, es[2], map[string]string{"Attributes[0].Value": phCurrent, "Attributes[1].Value": "2"})
	checkFields(t, es[3], map[string]string{"HP": phCurrent, "MaxHP": "80", "MP": "30", "CP": phCurrent})
}

func TestNormalizeDamageMessages(t *testing.T) {
	msg := func(id int32, params ...testclient.SMParam) scenario.Packet {
		return srvPacket("SystemMessage", testclient.SystemMessage{ID: id, Params: params})
	}
	num := func(v int64) testclient.SMParam { return testclient.SMParam{Type: testclient.SMParamInt, Number: v} }
	npc := testclient.SMParam{Type: testclient.SMParamNpcName, Number: 1020481}
	c := newCapture(nil, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
		msg(35, num(12)), msg(36, npc, num(4)), msg(95, num(35), num(2)),
	}})
	es := Normalize(c).Streams[0].Entries
	checkFields(t, es[0], map[string]string{"Params[0].Number": phDamage})
	checkFields(t, es[1], map[string]string{"Params[0].Number": "1020481", "Params[1].Number": phDamage})
	checkFields(t, es[2], map[string]string{"Params[0].Number": "35", "Params[1].Number": "2"})
}

func TestNormalizeSortsItemPackets(t *testing.T) {
	change := func(ch uint16, obj, itemID, count int32) testclient.ItemChange {
		return testclient.ItemChange{Change: ch, Item: testclient.Item{ObjectID: obj, ItemID: itemID, Count: count}}
	}
	capture := func(rev bool) *scenario.Capture {
		u := []testclient.ItemChange{change(testclient.ItemRemoved, 0x10000001, 1060, 0), change(testclient.ItemModified, 0x10000002, 17, 10)}
		s := []testclient.SellListItem{{ObjectID: 0x10000003, ItemID: 1060}, {ObjectID: 0x10000002, ItemID: 17}}
		w := []testclient.WarehouseListItem{{ObjectID: 0x10000004, ItemID: 57}, {ObjectID: 0x10000005, ItemID: 17}}
		if rev {
			u[0], u[1] = u[1], u[0]
			s[0], s[1] = s[1], s[0]
			w[0], w[1] = w[1], w[0]
		}
		// The IDs are named before these packets, so the order shows in
		// the values only.
		return newCapture(nil, scenario.Stream{Name: "game alice", Packets: []scenario.Packet{
			srvPacket("ItemList", testclient.ItemList{Items: []testclient.Item{
				{ObjectID: 0x10000001, ItemID: 1060}, {ObjectID: 0x10000002, ItemID: 17}, {ObjectID: 0x10000003, ItemID: 1060},
				{ObjectID: 0x10000004, ItemID: 57}, {ObjectID: 0x10000005, ItemID: 17},
			}}),
			srvPacket("InventoryUpdate", testclient.InventoryUpdate{Items: u}),
			srvPacket("SellList", testclient.SellList{Items: s}),
			srvPacket("WareHouseWithdrawalList", testclient.WareHouseWithdrawalList{WarehouseList: testclient.WarehouseList{Items: w}}),
		}})
	}
	a, b := Normalize(capture(false)), Normalize(capture(true))
	if !reflect.DeepEqual(a.Streams, b.Streams) {
		t.Fatalf("normalized streams differ:\n%+v\n%+v", a.Streams, b.Streams)
	}
	es := a.Streams[0].Entries
	checkFields(t, es[1], map[string]string{"Items[0].ItemID": "17", "Items[1].ItemID": "1060"})
	checkFields(t, es[2], map[string]string{"Items[0].ItemID": "17", "Items[1].ItemID": "1060"})
	checkFields(t, es[3], map[string]string{"Items[0].ItemID": "17", "Items[1].ItemID": "57", "Type": "0"})
}

// fightCapture is alice's kill of a monster with the given swings: each
// swing is an Attack, a StatusUpdate and a hit or miss message.
func fightCapture(swings int, misses ...int) *scenario.Capture {
	const (
		player = int32(0x10000001)
		mob    = int32(0x60000001)
	)
	ps := []scenario.Packet{
		srvPacket("CharSelected", testclient.CharSelected{ObjectID: player}),
		srvPacket("NpcInfo", testclient.NpcInfo{ObjectID: mob, NpcTypeID: 1020481, Attackable: true}),
		cliPacket("AttackRequest"),
		srvPacket("MoveToPawn", testclient.MoveToPawn{ObjectID: player, TargetID: mob}),
		srvPacket("AutoAttackStart", testclient.AutoAttackStart{ObjectID: player}),
	}
	for i := range swings {
		hit := testclient.Hit{TargetID: mob, Damage: int32(10 + i)}
		msg := testclient.SystemMessage{ID: 35, Params: []testclient.SMParam{{Type: testclient.SMParamInt, Number: int64(hit.Damage)}}}
		for _, m := range misses {
			if m == i {
				hit = testclient.Hit{TargetID: mob, Flags: testclient.HitMiss}
				msg = testclient.SystemMessage{ID: 43}
			}
		}
		ps = append(ps,
			srvPacket("Attack", testclient.Attack{AttackerID: player, Hits: []testclient.Hit{hit}}),
			srvPacket("StatusUpdate", testclient.StatusUpdate{ObjectID: mob, Attributes: []testclient.StatusAttribute{
				{ID: testclient.StatusCurHP, Value: int32(100 - 10*i)}, {ID: testclient.StatusMaxHP, Value: 100},
			}}),
			srvPacket("SystemMessage", msg),
		)
	}
	ps = append(ps,
		srvPacket("SystemMessage", testclient.SystemMessage{ID: 95, Params: []testclient.SMParam{
			{Type: testclient.SMParamInt, Number: 35}, {Type: testclient.SMParamInt, Number: 2},
		}}),
		srvPacket("Die", testclient.Die{ObjectID: mob}),
	)
	return newCapture([]scenario.Identity{alice}, scenario.Stream{Name: "game alice", Packets: ps})
}

func TestNormalizeCollapsesFights(t *testing.T) {
	short := Normalize(fightCapture(3))
	long := Normalize(fightCapture(6, 1, 4))
	var names []string
	for _, e := range short.Streams[0].Entries {
		names = append(names, e.Name)
	}
	// The MoveToPawn before AutoAttackStart is a run without a swing: it
	// stays.
	want := []string{"CharSelected", "NpcInfo", "AttackRequest", "MoveToPawn", "AutoAttackStart", FightEntry, "SystemMessage", "Die"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("entries %q, want %q", names, want)
	}
	checkFields(t, short.Streams[0].Entries[5], map[string]string{
		"Attackers": "#player1", "Targets": "#npc1", "Swings": "3", "Packets": "9",
	})
	res := Diff(short, long, Options{Ignore: []string{"Fight.Swings", "Fight.Packets"}})
	if !res.Identical() {
		t.Errorf("fights of 3 and 6 swings differ beyond the counts: %+v", res.Summary)
	}
	if res := Diff(short, long, Options{}); res.Summary.Changed != 1 || res.Summary.Fields != 2 {
		t.Errorf("without the ignores: %+v, want one Fight with two counts changed", res.Summary)
	}
}

func TestCollapseFightsKeepsOtherRuns(t *testing.T) {
	es := []Entry{
		{Dir: DirServer, Name: "StatusUpdate"},
		{Dir: DirServer, Name: "SystemMessage", Fields: []Field{{Path: "ID", Value: "43"}}},
		{Dir: DirServer, Name: "SystemMessage", Fields: []Field{{Path: "ID", Value: "95"}}},
		{Dir: DirClient, Name: "Attack"},
	}
	if got := collapseFights(es); !reflect.DeepEqual(got, es) {
		t.Errorf("collapseFights changed a run without a server Attack: %+v", got)
	}
	if s := nameSet([]string{"#npc2", "#npc1", "#npc2"}); s != "#npc1 #npc2" {
		t.Errorf("nameSet = %q", s)
	}
	if !strings.Contains(FightEntry, "Fight") {
		t.Error("FightEntry renamed")
	}
}
