package testclient

import (
	"reflect"
	"slices"
	"testing"
)

// testMonster is an attackable monster as NpcInfo shows it.
func testMonster(objectID int32, pos Position) NpcInfo {
	return NpcInfo{
		ObjectID: objectID, NpcTypeID: 1000000 + 20120, Attackable: true,
		X: pos.X, Y: pos.Y, Z: pos.Z, RunSpd: 120, WalkSpd: 50, MoveMultiplier: 1, AttackSpeedMultiplier: 1,
		CollisionRadius: 10, CollisionHeight: 12, ShowName: 1, Name: "Wolf",
		CollisionRadius2: 10, CollisionHeight2: 12,
	}
}

func TestWorldGroundItems(t *testing.T) {
	w := newWorld()
	w.apply(testUserInfoAt(Position{0, 0, 0}))
	applyAll(w,
		SpawnItem{ObjectID: 0x10000031, ItemID: 57, X: 500, Y: 0, Z: 0, Stackable: true, Count: 10},
		DropItem{DropperID: 0x60000005, ObjectID: 0x10000032, ItemID: 57, X: 100, Y: 0, Z: 0, Stackable: true, Count: 25, Unknown: 1},
		DropItem{DropperID: testMeID, ObjectID: 0x10000033, ItemID: 1060, X: 10, Y: 10, Z: 0, Stackable: true, Count: 3, Unknown: 1},
	)
	got := w.GroundItems()
	want := []GroundItem{
		{ObjectID: 0x10000031, ItemID: 57, Pos: Position{500, 0, 0}, Stackable: true, Count: 10},
		{ObjectID: 0x10000032, ItemID: 57, Pos: Position{100, 0, 0}, Stackable: true, Count: 25, DropperID: 0x60000005},
		{ObjectID: 0x10000033, ItemID: 1060, Pos: Position{10, 10, 0}, Stackable: true, Count: 3, DropperID: testMeID},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GroundItems()\n got %+v\nwant %+v", got, want)
	}
	if g, ok := w.FindGroundItem(57); !ok || g.ObjectID != 0x10000032 {
		t.Errorf("FindGroundItem(57) = %+v, %v; want the nearer adena", g, ok)
	}
	if _, ok := w.FindGroundItem(1); ok {
		t.Error("FindGroundItem(1) found an item")
	}
	// Ground items are not objects.
	if n := len(w.Objects()); n != 0 {
		t.Errorf("%d objects, want 0", n)
	}

	// GetItem and DeleteObject take items away.
	w.apply(GetItem{PlayerID: testMeID, ObjectID: 0x10000032, X: 100})
	w.apply(DeleteObject{ObjectID: 0x10000031, Disappear: 1})
	if g := w.GroundItems(); len(g) != 1 || g[0].ObjectID != 0x10000033 {
		t.Errorf("after GetItem and DeleteObject: %+v", g)
	}
	if _, ok := w.GroundItem(0x10000032); ok {
		t.Error("the picked up item is still on the ground")
	}
	if g, ok := w.GroundItem(0x10000033); !ok || g.ItemID != 1060 {
		t.Errorf("GroundItem = %+v, %v", g, ok)
	}

	w.apply(RestartResponse{OK: true})
	if n := len(w.GroundItems()); n != 0 {
		t.Errorf("%d ground items after the restart", n)
	}
}

func TestWorldCombatState(t *testing.T) {
	w := newWorld()
	monster := testMonster(0x60000005, Position{100, 0, 0})
	corpse := testMonster(0x60000006, Position{200, 0, 0})
	corpse.AlikeDead, corpse.InCombat = true, true
	applyAll(w, testUserInfoAt(Position{}), monster, corpse, testOtherPlayer(Position{}))

	if o, _ := w.Object(0x60000006); !o.Dead || !o.InCombat {
		t.Errorf("NpcInfo flags not taken: %+v", o)
	}
	if o, _ := w.Object(0x60000005); o.Dead || o.InCombat {
		t.Errorf("monster state %+v", o)
	}

	steps := []struct {
		name  string
		p     Packet
		check func() bool
	}{
		{"own AutoAttackStart", AutoAttackStart{ObjectID: testMeID}, func() bool { return w.Me().InCombat }},
		{"monster AutoAttackStart", AutoAttackStart{ObjectID: 0x60000005}, func() bool {
			o, _ := w.Object(0x60000005)
			return o.InCombat
		}},
		{"monster Die", Die{ObjectID: 0x60000005, Sweepable: true}, func() bool {
			o, _ := w.Object(0x60000005)
			return o.Dead && !w.Me().Dead
		}},
		{"monster Revive", Revive{ObjectID: 0x60000005}, func() bool {
			o, _ := w.Object(0x60000005)
			return !o.Dead
		}},
		{"player Die", Die{ObjectID: testOtherID}, func() bool {
			o, _ := w.Object(testOtherID)
			return o.Dead
		}},
		{"own AutoAttackStop", AutoAttackStop{ObjectID: testMeID}, func() bool { return !w.Me().InCombat }},
		{"own Die", Die{ObjectID: testMeID, CanTeleport: true, FixedRes: true}, func() bool {
			me := w.Me()
			return me.Dead && me.Death == Die{ObjectID: testMeID, CanTeleport: true, FixedRes: true}
		}},
		{"own Revive", Revive{ObjectID: testMeID}, func() bool {
			me := w.Me()
			return !me.Dead && me.Death == Die{}
		}},
	}
	for _, s := range steps {
		w.apply(s.p)
		if !s.check() {
			t.Errorf("%s: me %+v", s.name, w.Me())
		}
	}

	// Packets about unknown objects change nothing.
	before := w.Objects()
	applyAll(w, Die{ObjectID: 0x7fffffff}, AutoAttackStart{ObjectID: 0x7ffffffe}, Revive{ObjectID: 0x7ffffffd})
	if !reflect.DeepEqual(before, w.Objects()) || w.Me().Dead || w.Me().InCombat {
		t.Error("packets about unknown objects changed the world")
	}
}

func TestWorldLoad(t *testing.T) {
	w := newWorld()
	ui := testUserInfoAt(Position{})
	ui.CurLoad, ui.MaxLoad = 1200, 69000
	w.apply(ui)
	if me := w.Me(); me.Load != 1200 || me.MaxLoad != 69000 {
		t.Errorf("load %d/%d after UserInfo", me.Load, me.MaxLoad)
	}
	w.apply(StatusUpdate{ObjectID: testMeID, Attributes: []StatusAttribute{{StatusCurLoad, 1500}, {StatusMaxLoad, 70000}}})
	if me := w.Me(); me.Load != 1500 || me.MaxLoad != 70000 {
		t.Errorf("load %d/%d after StatusUpdate", me.Load, me.MaxLoad)
	}
}

func TestWorldInventoryLookups(t *testing.T) {
	w := newWorld()
	items := testItems()
	w.apply(ItemList{Items: items})
	if it, ok := w.Item(0x10000013); !ok || it.ItemID != 2369 {
		t.Errorf("Item(0x10000013) = %+v, %v", it, ok)
	}
	if _, ok := w.Item(0x10000099); ok {
		t.Error("Item found an object that is not there")
	}
	if it, ok := w.FindItem(57); !ok || it.ObjectID != 0x10000014 || it.Count != 5000 {
		t.Errorf("FindItem(57) = %+v, %v", it, ok)
	}
	if _, ok := w.FindItem(1060); ok {
		t.Error("FindItem found an item that is not there")
	}
}

func TestWorldSkillsAndShortcuts(t *testing.T) {
	w := newWorld()
	w.apply(testUserInfoAt(Position{}))
	itemShortcut := Shortcut{Type: ShortcutItem, Slot: 3, ID: 0x10000015, Level: 1,
		Rest: []byte{0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	action := Shortcut{Type: ShortcutAction, Slot: 1, ID: 2, Level: 1}
	applyAll(w,
		SkillList{Skills: []Skill{{Level: 1, ID: 1177}, {Passive: true, Level: 2, ID: 214}}},
		ShortcutInit{Shortcuts: []Shortcut{itemShortcut, action}},
	)
	if w.SkillLevel(1177) != 1 || w.SkillLevel(214) != 2 || w.SkillLevel(1184) != 0 {
		t.Errorf("skill levels %d %d %d", w.SkillLevel(1177), w.SkillLevel(214), w.SkillLevel(1184))
	}
	if s := w.Skills(); len(s) != 2 {
		t.Errorf("Skills() = %+v", s)
	}
	// ShortcutInit is kept sorted by slot.
	if got := w.Shortcuts(); !reflect.DeepEqual(got, []Shortcut{action, itemShortcut}) {
		t.Errorf("Shortcuts() after ShortcutInit = %+v", got)
	}

	// A new skill level replaces the list.
	w.apply(SkillList{Skills: []Skill{{Level: 2, ID: 1177}}})
	if w.SkillLevel(1177) != 2 || w.SkillLevel(214) != 0 {
		t.Errorf("after the second SkillList: %+v", w.Skills())
	}

	// ShortcutRegister adds a slot in order, and replaces a taken one.
	skill := Shortcut{Type: ShortcutSkill, Slot: 2, ID: 1177, Level: 2, Rest: []byte{0, 1, 0, 0, 0}}
	w.apply(ShortcutRegister{skill})
	replaced := Shortcut{Type: ShortcutItem, Slot: 3, ID: 0x10000016, Level: 1}
	w.apply(ShortcutRegister{replaced})
	if got := w.Shortcuts(); !reflect.DeepEqual(got, []Shortcut{action, skill, replaced}) {
		t.Errorf("Shortcuts() after ShortcutRegister = %+v", got)
	}
	if s, ok := w.Shortcut(2); !ok || s.ID != 1177 {
		t.Errorf("Shortcut(2) = %+v, %v", s, ok)
	}
	w.deleteShortcut(2)
	w.deleteShortcut(40) // an empty slot
	if got := w.Shortcuts(); !reflect.DeepEqual(got, []Shortcut{action, replaced}) {
		t.Errorf("Shortcuts() after deleteShortcut = %+v", got)
	}
	if _, ok := w.Shortcut(2); ok {
		t.Error("the deleted shortcut is still there")
	}
	got := w.Shortcuts()
	got[0].ID = 99
	if s, _ := w.Shortcut(1); s.ID != 2 {
		t.Error("Shortcuts returned the world's own slice")
	}
}

func TestWorldBuffsAndShots(t *testing.T) {
	w := newWorld()
	w.apply(testUserInfoAt(Position{}))
	if _, ok := w.ShortBuff(); ok {
		t.Error("a short buff before any came")
	}
	pet := PartySpelled{Kind: SpelledPet, ObjectID: 0x60000009, Effects: []Effect{{SkillID: 1040, Level: 3, Duration: 1200}}}
	applyAll(w,
		AbnormalStatusUpdate{Effects: []Effect{{SkillID: 1040, Level: 3, Duration: 1200}, {SkillID: 1068, Level: 1, Duration: 1100}}},
		ShortBuffStatusUpdate{SkillID: 2031, Level: 1, Duration: 15},
		pet,
		ExAutoSoulShot{ItemID: 3947, Enabled: true},
		ExAutoSoulShot{ItemID: 1835, Enabled: true},
	)
	if e := w.Effects(); len(e) != 2 {
		t.Errorf("Effects() = %+v", e)
	}
	if e, ok := w.Effect(1068); !ok || e.Duration != 1100 {
		t.Errorf("Effect(1068) = %+v, %v", e, ok)
	}
	if _, ok := w.Effect(1204); ok {
		t.Error("Effect found an effect that is not there")
	}
	if s, ok := w.ShortBuff(); !ok || s.SkillID != 2031 || s.Duration != 15 {
		t.Errorf("ShortBuff() = %+v, %v", s, ok)
	}
	if p, ok := w.PartySpelled(0x60000009); !ok || !reflect.DeepEqual(p, pet) {
		t.Errorf("PartySpelled = %+v, %v", p, ok)
	}
	if _, ok := w.PartySpelled(0x60000010); ok {
		t.Error("PartySpelled found an object without buffs")
	}
	if got := w.AutoSoulShots(); !slices.Equal(got, []int32{1835, 3947}) {
		t.Errorf("AutoSoulShots() = %v", got)
	}

	// A new AbnormalStatusUpdate replaces the icons; the reset short buff
	// and a shot turned off are gone.
	applyAll(w,
		AbnormalStatusUpdate{Effects: []Effect{{SkillID: 1068, Level: 1, Duration: 1000}}},
		ShortBuffStatusUpdate{},
		ExAutoSoulShot{ItemID: 3947},
	)
	if e := w.Effects(); len(e) != 1 || e[0].SkillID != 1068 {
		t.Errorf("Effects() = %+v", e)
	}
	if _, ok := w.ShortBuff(); ok {
		t.Error("the short buff survived its reset")
	}
	if got := w.AutoSoulShots(); !slices.Equal(got, []int32{1835}) {
		t.Errorf("AutoSoulShots() = %v", got)
	}

	// Selecting a character starts over.
	w.apply(CharSelected{ObjectID: testMeID})
	if len(w.Effects()) != 0 || len(w.AutoSoulShots()) != 0 || len(w.Skills()) != 0 || len(w.Shortcuts()) != 0 {
		t.Error("buffs, shots, skills or shortcuts survived CharSelected")
	}
	if _, ok := w.PartySpelled(0x60000009); ok {
		t.Error("PartySpelled survived CharSelected")
	}
}
