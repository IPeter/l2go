package testclient

import (
	"fmt"
	"reflect"
	"testing"
)

const (
	testMeID    int32 = 0x10000001
	testOtherID int32 = 0x10000002
	testNpcID   int32 = 0x60000001
)

// testUserInfoAt returns the UserInfo of the test character "Hero" at pos.
func testUserInfoAt(pos Position) UserInfo {
	return UserInfo{
		X: pos.X, Y: pos.Y, Z: pos.Z, ObjectID: testMeID, Name: "Hero", Level: 1,
		MaxHP: 126, HP: 120, MaxMP: 38, MP: 30, MaxCP: 63, CP: 60,
		RunSpd: 115, WalkSpd: 80, MoveMultiplier: 1, AttackSpeedMultiplier: 1,
		CollisionRadius: 9, CollisionHeight: 23, ClassID: 0, Running: true,
	}
}

// testOtherPlayer returns the CharInfo of another player, "Pal", at pos.
func testOtherPlayer(pos Position) CharInfoPacket {
	return CharInfoPacket{
		X: pos.X, Y: pos.Y, Z: pos.Z, ObjectID: testOtherID, Name: "Pal", Title: "friend",
		Standing: true, Running: true, Heading: 1000, MoveMultiplier: 1, AttackSpeedMultiplier: 1,
	}
}

func applyAll(w *World, ps ...Packet) {
	for _, p := range ps {
		w.apply(p)
	}
}

func TestWorldTracksObjects(t *testing.T) {
	w := newWorld()
	start := Position{-71338, 258271, -3104}
	npc := testGatekeeper(testNpcID)
	applyAll(w,
		CharSelected{Name: "Hero", ObjectID: testMeID, X: 1, Y: 2, Z: 3},
		testUserInfoAt(start),
		npc,
		testOtherPlayer(Position{-71000, 258000, -3100}),
		ServerObjectInfo{ObjectID: 0x60000002, NpcTypeID: 1000000 + 31111, X: 5, Y: 6, Z: 7, Attackable: true, HP: 10, MaxHP: 20},
		RelationChanged{ObjectID: testOtherID, Relation: 0x480, AutoAttackable: true},
		RelationChanged{ObjectID: 0x7fffffff}, // unknown objects are ignored
	)

	me := w.Me()
	if me.ObjectID != testMeID || me.Name != "Hero" || me.Pos != start || !me.HasUserInfo {
		t.Errorf("Me = %+v", me)
	}
	if me.HP != 120 || me.MaxHP != 126 || me.MP != 30 || me.CP != 60 || !me.Running {
		t.Errorf("Me status = %+v", me)
	}

	objs := w.Objects()
	if len(objs) != 3 || objs[0].ObjectID != testOtherID || objs[1].ObjectID != testNpcID || objs[2].ObjectID != 0x60000002 {
		t.Fatalf("Objects() = %+v", objs)
	}
	other, ok := w.FindPlayer("Pal")
	if !ok || other.Kind != KindPlayer || other.Title != "friend" || other.Heading != 1000 || other.Sitting ||
		other.Relation != 0x480 || !other.AutoAttackable {
		t.Errorf("FindPlayer(Pal) = %+v, %v", other, ok)
	}
	if _, ok := other.Info.(CharInfoPacket); !ok {
		t.Errorf("player Info is %T", other.Info)
	}
	g, ok := w.FindNpc(30006)
	if !ok || g.ObjectID != testNpcID || g.Kind != KindNpc || g.Name != "Roxxy" || g.Pos != npc.Position() {
		t.Errorf("FindNpc(30006) = %+v, %v", g, ok)
	}
	if s, ok := w.Object(0x60000002); !ok || s.Kind != KindStatic || s.NpcID != 31111 || !s.Attackable || s.HP != 10 || s.MaxHP != 20 {
		t.Errorf("static object = %+v, %v", s, ok)
	}
	if ps, ns := w.Players(), w.Npcs(); len(ps) != 1 || len(ns) != 2 {
		t.Errorf("Players %d, Npcs %d", len(ps), len(ns))
	}

	w.apply(DeleteObject{ObjectID: testOtherID, Disappear: 1})
	if _, ok := w.Object(testOtherID); ok {
		t.Error("the player is still known after DeleteObject")
	}
	if _, ok := w.FindPlayer("Pal"); ok {
		t.Error("FindPlayer finds a deleted player")
	}
}

func TestWorldFindNpcNearest(t *testing.T) {
	w := newWorld()
	w.apply(testUserInfoAt(Position{0, 0, 0}))
	far := testGatekeeper(0x60000010)
	far.X, far.Y, far.Z = 1000, 0, 0
	near := testGatekeeper(0x60000020)
	near.X, near.Y, near.Z = 0, 300, 0
	tie := testGatekeeper(0x60000005)
	tie.X, tie.Y, tie.Z = 300, 0, 0
	applyAll(w, far, near, tie)
	if g, ok := w.FindNpc(30006); !ok || g.ObjectID != 0x60000005 {
		t.Errorf("FindNpc = %d, %v; want the nearest with the lower ID, 0x60000005", g.ObjectID, ok)
	}
	if _, ok := w.FindNpc(30007); ok {
		t.Error("FindNpc found an NPC that is not there")
	}
}

func TestWorldMovement(t *testing.T) {
	w := newWorld()
	applyAll(w, testUserInfoAt(Position{0, 0, 0}), testOtherPlayer(Position{100, 100, 0}))

	steps := []struct {
		name  string
		p     Packet
		id    int32
		check func(m Motion) bool
	}{
		{"MoveToLocation", MoveToLocation{ObjectID: testOtherID, DestX: 500, DestY: 600, DestZ: 0, X: 110, Y: 120, Z: 0}, testOtherID,
			func(m Motion) bool {
				return m.Moving && m.Pos == Position{110, 120, 0} && m.Dest == Position{500, 600, 0}
			}},
		{"StopMove", StopMove{ObjectID: testOtherID, X: 300, Y: 310, Z: 0, Heading: 7}, testOtherID,
			func(m Motion) bool { return !m.Moving && m.Pos == Position{300, 310, 0} && m.Heading == 7 }},
		{"ChangeMoveType", ChangeMoveType{ObjectID: testOtherID, Running: false}, testOtherID,
			func(m Motion) bool { return !m.Running }},
		{"ChangeWaitType sit", ChangeWaitType{ObjectID: testOtherID, WaitType: WaitSitting, X: 301, Y: 311}, testOtherID,
			func(m Motion) bool { return m.Sitting && m.Pos == Position{301, 311, 0} }},
		{"ChangeWaitType stand", ChangeWaitType{ObjectID: testOtherID, WaitType: WaitStanding, X: 301, Y: 311}, testOtherID,
			func(m Motion) bool { return !m.Sitting }},
		{"StopRotation", StopRotation{ObjectID: testOtherID, Degree: 16000}, testOtherID,
			func(m Motion) bool { return m.Heading == 16000 }},
		{"TargetSelected", TargetSelected{ObjectID: testOtherID, TargetID: testMeID}, testOtherID,
			func(m Motion) bool { return m.TargetID == testMeID }},
		{"TargetUnselected", TargetUnselected{ObjectID: testOtherID}, testOtherID,
			func(m Motion) bool { return m.TargetID == 0 }},
		{"own MoveToLocation", MoveToLocation{ObjectID: testMeID, DestX: 50, X: 1, Y: 2, Z: 3}, testMeID,
			func(m Motion) bool { return m.Moving && m.Pos == Position{1, 2, 3} && m.Dest == Position{50, 0, 0} }},
		{"own ValidateLocation", ValidateLocation{ObjectID: testMeID, X: 40, Y: 0, Z: 0, Heading: 9}, testMeID,
			func(m Motion) bool { return m.Pos == Position{40, 0, 0} && m.Heading == 9 }},
		{"own MoveToPawn", MoveToPawn{ObjectID: testMeID, TargetID: testNpcID, Distance: 100, X: 41, TargetX: 900}, testMeID,
			func(m Motion) bool { return m.Moving && m.Pos == Position{41, 0, 0} && m.Dest == Position{900, 0, 0} }},
		{"own TeleportToLocation", TeleportToLocation{ObjectID: testMeID, X: 1000, Y: 2000, Z: -3000, Heading: 5}, testMeID,
			func(m Motion) bool { return !m.Moving && m.Pos == Position{1000, 2000, -3000} && m.Heading == 5 }},
		{"MyTargetSelected", MyTargetSelected{ObjectID: testOtherID}, testMeID,
			func(m Motion) bool { return m.TargetID == testOtherID }},
		{"own TargetUnselected", TargetUnselected{ObjectID: testMeID}, testMeID,
			func(m Motion) bool { return m.TargetID == 0 }},
	}
	for _, s := range steps {
		w.apply(s.p)
		var m Motion
		if s.id == testMeID {
			m = w.Me().Motion
		} else {
			o, ok := w.Object(s.id)
			if !ok {
				t.Fatalf("%s: object %d unknown", s.name, s.id)
			}
			m = o.Motion
		}
		if !s.check(m) {
			t.Errorf("%s: motion %+v", s.name, m)
		}
	}

	// Movement of an object the client does not know changes nothing.
	before := w.Objects()
	w.apply(MoveToLocation{ObjectID: 0x7fffffff, DestX: 1})
	if !reflect.DeepEqual(before, w.Objects()) {
		t.Error("a move of an unknown object changed the world")
	}
}

func TestWorldStatusAndItems(t *testing.T) {
	w := newWorld()
	items := testItems()
	applyAll(w,
		testUserInfoAt(Position{}),
		testGatekeeper(testNpcID),
		StatusUpdate{ObjectID: testNpcID, Attributes: []StatusAttribute{{StatusCurHP, 80}, {StatusMaxHP, 100}}},
		StatusUpdate{ObjectID: testMeID, Attributes: []StatusAttribute{{StatusCurHP, 90}, {StatusMaxCP, 70}, {StatusCurMP, 5}}},
		ItemList{Items: items},
	)
	if g, _ := w.Object(testNpcID); g.HP != 80 || g.MaxHP != 100 {
		t.Errorf("NPC HP %d/%d", g.HP, g.MaxHP)
	}
	if me := w.Me(); me.HP != 90 || me.MaxCP != 70 || me.MP != 5 || me.MaxHP != 126 {
		t.Errorf("Me HP %d/%d MP %d CP max %d", me.HP, me.MaxHP, me.MP, me.MaxCP)
	}
	if n := w.ItemCount(57); n != 5000 {
		t.Errorf("adena %d, want 5000", n)
	}

	adena := items[3]
	adena.Count = 4000
	shield := Item{Type1: 1, ObjectID: 0x10000020, ItemID: 18, Count: 1, Type2: 1, BodyPart: 0x0100, Mana: -1}
	w.apply(InventoryUpdate{Items: []ItemChange{
		{Change: ItemModified, Item: adena},
		{Change: ItemRemoved, Item: items[0]},
		{Change: ItemAdded, Item: shield},
	}})
	got := w.Items()
	want := []Item{items[1], items[2], adena, shield}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Items after InventoryUpdate:\n got %+v\nwant %+v", got, want)
	}
	if n := w.ItemCount(57); n != 4000 {
		t.Errorf("adena %d, want 4000", n)
	}
	got[0].Count = 99
	if w.Items()[0].Count == 99 {
		t.Error("Items returned the world's own slice")
	}
}

func TestWorldLogsAndReset(t *testing.T) {
	w := newWorld()
	say := CreatureSay{ObjectID: testOtherID, Type: ChatShout, Name: "Pal", Text: "hi"}
	npcSay := NpcSay{ObjectID: testNpcID, Type: ChatGeneral, NpcTypeID: 1000000 + 30006, Text: "Welcome"}
	sm := SystemMessage{ID: SysMsgWelcome}
	page := NpcHtmlMessage{ObjectID: testNpcID, HTML: "<html>1</html>"}
	applyAll(w, CharSelected{ObjectID: testMeID}, testOtherPlayer(Position{}), say, npcSay, sm, page)

	if c := w.Chat(); len(c) != 1 || c[0] != say {
		t.Errorf("Chat() = %+v", c)
	}
	if n := w.NpcSays(); len(n) != 1 || n[0] != npcSay {
		t.Errorf("NpcSays() = %+v", n)
	}
	if s := w.SystemMessages(); len(s) != 1 || s[0].ID != SysMsgWelcome {
		t.Errorf("SystemMessages() = %+v", s)
	}
	if h, ok := w.HTML(); !ok || h != page {
		t.Errorf("HTML() = %+v, %v", h, ok)
	}

	// A refused restart keeps the world; an accepted one clears it.
	w.apply(RestartResponse{OK: false})
	if w.Me().ObjectID != testMeID || len(w.Objects()) != 1 {
		t.Error("a refused restart cleared the world")
	}
	w.apply(RestartResponse{OK: true})
	if w.Me().ObjectID != 0 || len(w.Objects()) != 0 || len(w.Chat()) != 0 {
		t.Errorf("after restart: me %+v, objects %d", w.Me(), len(w.Objects()))
	}
	if _, ok := w.HTML(); ok {
		t.Error("the dialog page survived the restart")
	}

	// Selecting a character starts over too.
	applyAll(w, testOtherPlayer(Position{}), CharSelected{ObjectID: 0x10000009, Name: "Next"})
	if me := w.Me(); me.ObjectID != 0x10000009 || me.Name != "Next" || len(w.Objects()) != 0 {
		t.Errorf("after CharSelected: me %+v, objects %d", me, len(w.Objects()))
	}
}

func TestWorldLogLimit(t *testing.T) {
	w := newWorld()
	for i := range logLimit + 5 {
		w.apply(CreatureSay{Text: fmt.Sprint(i)})
	}
	c := w.Chat()
	if len(c) != logLimit || c[0].Text != "5" || c[len(c)-1].Text != fmt.Sprint(logLimit+4) {
		t.Errorf("chat log has %d lines from %q to %q", len(c), c[0].Text, c[len(c)-1].Text)
	}
}

func TestPositionDistance(t *testing.T) {
	if d := (Position{0, 0, 0}).DistanceTo(Position{3, 4, 12}); d != 13 {
		t.Errorf("distance %v, want 13", d)
	}
	// No int32 overflow far apart.
	if d := (Position{-2000000000, 0, 0}).DistanceTo(Position{2000000000, 0, 0}); d != 4e9 {
		t.Errorf("distance %v, want 4e9", d)
	}
}
