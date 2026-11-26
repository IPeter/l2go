package testclient

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

// The Mobius pet and servitor packets, transcribed from their writeImpl
// and readImpl line by line.

// mobiusPetInfo is PetInfo.writeImpl.
func mobiusPetInfo(p PetInfo) []byte {
	w := msg(0xb1)
	w.i32(p.SummonType)
	w.i32(p.ObjectID)
	w.i32(p.NpcTypeID)       // displayId + 1000000
	w.i32(b2i(p.Attackable)) // 1=attackable
	w.i32(p.X).i32(p.Y).i32(p.Z)
	w.i32(p.Heading)
	w.i32(p.Unknown) // 0
	w.i32(p.MAtkSpd).i32(p.PAtkSpd)
	w.i32(p.RunSpd).i32(p.WalkSpd).i32(p.SwimRunSpd).i32(p.SwimWalkSpd)
	w.i32(p.FlyRunSpd).i32(p.FlyWalkSpd).i32(p.FlyRunSpd2).i32(p.FlyWalkSpd2)
	w.f64(p.MoveMultiplier).f64(p.AttackSpeedMultiplier)
	w.f64(p.CollisionRadius).f64(p.CollisionHeight)
	w.i32(p.RHand).i32(p.Chest).i32(p.LHand)
	w.u8(b2u8(p.ShowName)).u8(b2u8(p.Running)).u8(b2u8(p.InCombat)).u8(b2u8(p.AlikeDead))
	w.u8(p.Summoned) // 0=teleported 1=default 2=summoned
	w.str(p.Name).str(p.Title)
	w.i32(p.TitleColor) // 1
	w.i32(p.PvpFlag).i32(p.Karma)
	w.i32(p.CurFed).i32(p.MaxFed)
	w.i32(p.CurHP).i32(p.MaxHP).i32(p.CurMP).i32(p.MaxMP)
	w.i32(p.SP).i32(p.Level)
	w.i64(p.Exp).i64(p.ExpThisLevel).i64(p.ExpNextLevel)
	w.i32(p.CurLoad).i32(p.MaxLoad)
	w.i32(p.PAtk).i32(p.PDef).i32(p.MAtk).i32(p.MDef)
	w.i32(p.Accuracy).i32(p.Evasion).i32(p.Critical)
	w.i32(p.Speed).i32(p.AtkSpd).i32(p.CastSpd)
	w.i32(p.AbnormalEffects)
	w.u16(uint16(b2i(p.Mountable))) // c2 ride button
	w.u8(p.Environment)
	w.u16(p.Unknown2) // ??
	w.u8(p.Team)
	w.i32(p.SoulShotsPerHit).i32(p.SpiritShotsPerHit)
	return w.b
}

// mobiusPetItem is one item of PetItemList and PetInventoryUpdate.
func mobiusPetItem(w *wire, it PetItem) {
	w.u16(it.Type1).i32(it.ObjectID).i32(it.ItemID).i32(it.Count)
	w.u16(it.Type2).u16(it.Reserved1).u16(uint16(b2i(it.Equipped)))
	w.i32(it.BodyPart).u16(it.Enchant).u16(it.Reserved2)
}

func mobiusPetItemList(items ...PetItem) []byte {
	w := msg(0xb2).u16(uint16(len(items)))
	for _, it := range items {
		mobiusPetItem(w, it)
	}
	return w.b
}

func mobiusPetInventoryUpdate(items ...PetItemChange) []byte {
	w := msg(0xb3).u16(uint16(len(items)))
	for _, it := range items {
		w.u16(it.Change)
		mobiusPetItem(w, it.PetItem)
	}
	return w.b
}

func mobiusPetStatusUpdate(p PetStatusUpdate) []byte {
	return msg(0xb5).i32(p.SummonType).i32(p.ObjectID).i32(p.X).i32(p.Y).i32(p.Z).
		str(p.Title).i32(p.CurFed).i32(p.MaxFed).
		i32(p.CurHP).i32(p.MaxHP).i32(p.CurMP).i32(p.MaxMP).i32(p.Level).
		i64(p.Exp).i64(p.ExpThisLevel).i64(p.ExpNextLevel).b
}

func mobiusRide(r Ride) []byte {
	return msg(0x86).i32(r.ObjectID).i32(b2i(r.Mounted)).i32(r.MountType).i32(r.MountNpcTypeID).
		i32(r.X).i32(r.Y).i32(r.Z).b
}

// testWolf is a pet wolf as its owner sees it.
func testWolf() PetInfo {
	return PetInfo{
		SummonType: SummonPet, ObjectID: 0x10000020, NpcTypeID: 1000000 + 12077,
		X: -71300, Y: 258250, Z: -3104, Heading: 16384,
		MAtkSpd: 333, PAtkSpd: 278, RunSpd: 160, WalkSpd: 60, SwimRunSpd: 160, SwimWalkSpd: 60,
		MoveMultiplier: 1, AttackSpeedMultiplier: 1, CollisionRadius: 9, CollisionHeight: 12.5,
		ShowName: true, Running: true, Summoned: 1, Title: "Hero", TitleColor: 1,
		CurFed: 3000, MaxFed: 3600, CurHP: 900, MaxHP: 1000, CurMP: 300, MaxMP: 400, Level: 15,
		Exp: 50000, ExpThisLevel: 40000, ExpNextLevel: 60000, MaxLoad: 5000,
		PAtk: 101, PDef: 102, MAtk: 103, MDef: 104, Accuracy: 105, Evasion: 106, Critical: 107,
		Speed: 160, AtkSpd: 278, CastSpd: 333, SoulShotsPerHit: 1, SpiritShotsPerHit: 1,
	}
}

func TestSummonPacketsRoundTrip(t *testing.T) {
	var p PetInfo
	fillDistinct(&p)
	checkDecode(t, "PetInfo", mobiusPetInfo(p), p)

	w := testWolf()
	checkDecode(t, "PetInfo wolf", mobiusPetInfo(w), w)
	if w.NpcID() != 12077 || !w.IsPet() || w.Position() != (Position{-71300, 258250, -3104}) {
		t.Errorf("NpcID %d, IsPet %v, Position %v", w.NpcID(), w.IsPet(), w.Position())
	}

	var a, b PetItem
	fillDistinct(&a)
	fillDistinct(&b)
	b.ObjectID++
	checkDecode(t, "PetItemList", mobiusPetItemList(a, b), PetItemList{Items: []PetItem{a, b}})
	checkDecode(t, "empty PetItemList", mobiusPetItemList(), PetItemList{})
	changes := []PetItemChange{{Change: ItemAdded, PetItem: a}, {Change: ItemRemoved, PetItem: b}}
	checkDecode(t, "PetInventoryUpdate", mobiusPetInventoryUpdate(changes...), PetInventoryUpdate{Items: changes})

	var s PetStatusUpdate
	fillDistinct(&s)
	checkDecode(t, "PetStatusUpdate", mobiusPetStatusUpdate(s), s)

	var r Ride
	fillDistinct(&r)
	checkDecode(t, "Ride", mobiusRide(r), r)
	r.MountNpcTypeID = 1000000 + 12526
	if r.MountNpcID() != 12526 {
		t.Errorf("MountNpcID %d", r.MountNpcID())
	}

	checkDecode(t, "PetStatusShow", msg(0xb0).i32(2).b, PetStatusShow{SummonType: SummonPet})
	checkDecode(t, "PetDelete", msg(0xb6).i32(1).i32(0x10000021).b, PetDelete{SummonType: SummonServitor, ObjectID: 0x10000021})
	checkDecode(t, "SetSummonRemainTime", msg(0xd1).i32(1200).i32(600).b, SetSummonRemainTime{MaxTime: 1200, RemainingTime: 600})
}

func TestPetItemFind(t *testing.T) {
	l := PetItemList{Items: []PetItem{{ObjectID: 1, ItemID: 2515}, {ObjectID: 2, ItemID: 57}}}
	if it, ok := l.Find(2); !ok || it.ItemID != 57 {
		t.Errorf("Find(2) = %+v, %v", it, ok)
	}
	if it, ok := l.FindItem(2515); !ok || it.ObjectID != 1 {
		t.Errorf("FindItem(2515) = %+v, %v", it, ok)
	}
	if _, ok := l.Find(3); ok {
		t.Error("Find(3) found an item")
	}
	if _, ok := l.FindItem(3); ok {
		t.Error("FindItem(3) found an item")
	}
	u := PetInventoryUpdate{Items: []PetItemChange{{Change: ItemModified, PetItem: PetItem{ObjectID: 5, Count: 7}}}}
	if ch, ok := u.Find(5); !ok || ch.Count != 7 || ch.Change != ItemModified {
		t.Errorf("Find(5) = %+v, %v", ch, ok)
	}
	if _, ok := u.Find(6); ok {
		t.Error("Find(6) found an item")
	}
}

// A packet that does not follow the Mobius layout is malformed.
func TestDecodeSummonPacketsMalformed(t *testing.T) {
	wolf := mobiusPetInfo(testWolf())
	tests := []struct {
		name string
		raw  []byte
	}{
		{"PetInfo short", wolf[:len(wolf)-1]},
		{"PetInfo extra byte", append(bytes.Clone(wolf), 0)},
		{"PetItemList short", mobiusPetItemList(PetItem{})[:20]},
		{"PetItemList equipped 2", msg(0xb2).u16(1).u16(0).i32(1).i32(2).i32(3).u16(0).u16(0).u16(2).i32(0).u16(0).u16(0).b},
		{"PetInventoryUpdate short", mobiusPetInventoryUpdate(PetItemChange{Change: 1})[:10]},
		{"Ride mounted 2", msg(0x86).i32(1).i32(2).i32(1).i32(1012526).i32(0).i32(0).i32(0).b},
		{"PetStatusShow short", msg(0xb0).u16(2).b},
		{"PetDelete short", msg(0xb6).i32(1).b},
		{"SetSummonRemainTime short", msg(0xd1).i32(1).b},
		{"PetStatusUpdate short", mobiusPetStatusUpdate(PetStatusUpdate{})[:30]},
	}
	for _, tt := range tests {
		if p, err := DecodeServerPacket(tt.raw); !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %+v, %v; want ErrMalformed", tt.name, p, err)
		}
	}
}

// The pet requests use the Mobius readImpl layouts.
func TestEncodeSummonRequests(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want []byte
	}{
		// RequestChangePetName: S name.
		{"RequestChangePetName", encodeString(OpRequestChangePetName, "Fang"), msg(0x89).str("Fang").b},
		// RequestPetUseItem: d object ID (Mobius reads nothing more).
		{"RequestPetUseItem", encodeInts(OpRequestPetUseItem, 0x10000030), msg(0x8a).i32(0x10000030).b},
		// RequestGiveItemToPet: d object ID, d amount.
		{"RequestGiveItemToPet", encodeInts(OpRequestGiveItemToPet, 0x10000031, 20), msg(0x8b).i32(0x10000031).i32(20).b},
		// RequestGetItemFromPet: d object ID, d amount, d unknown.
		{"RequestGetItemFromPet", encodeRequestGetItemFromPet(0x10000032, 3), msg(0x8c).i32(0x10000032).i32(3).i32(0).b},
		// RequestPetGetItem: d object ID.
		{"RequestPetGetItem", encodeInts(OpRequestPetGetItem, 0x10000033), msg(0x8f).i32(0x10000033).b},
		// RequestActionUse: d action, d ctrl, c shift.
		{"RequestActionUse", encodeRequestActionUse(ActionPetAttack, true, true), msg(0x45).i32(16).i32(1).u8(1).b},
	} {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s = % x, want % x", tt.name, tt.got, tt.want)
		}
		if got := clientPacketName(tt.got); got != tt.name {
			t.Errorf("%s is recorded as %q", tt.name, got)
		}
	}
	actions := map[string][2]int32{
		"Change Movement Mode (Pets)":      {ActionPetFollow, 15},
		"Attack (Pets)":                    {ActionPetAttack, 16},
		"Stop (Pets)":                      {ActionPetStop, 17},
		"Unsummon Pet":                     {ActionUnsummonPet, 19},
		"Change Movement Mode (Servitors)": {ActionServitorFollow, 21},
		"Attack (Servitors)":               {ActionServitorAttack, 22},
		"Stop (Servitors)":                 {ActionServitorStop, 23},
		"Mount/Dismount":                   {ActionMountDismount, 38},
		"Unsummon Servitor":                {ActionUnsummonServitor, 52},
		"Move to target (Servitors)":       {ActionServitorMoveToTarget, 53},
		"Move to target (Pets)":            {ActionPetMoveToTarget, 54},
	}
	for name, v := range actions {
		if v[0] != v[1] {
			t.Errorf("%s = %d, want %d", name, v[0], v[1])
		}
	}
}

// TestPetFlow names a pet, moves food to it and back, feeds it, sends it to
// pick up an item and to attack, against a fake server.
func TestPetFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	food := Item{Type1: 4, ObjectID: 0x10000040, ItemID: 2515, Count: 50, Type2: 5, Mana: -1}
	inv := append(testItems(), food)
	food30 := food
	food30.Count = 30
	petFood := PetItem{Type1: 4, ObjectID: 0x10000041, ItemID: 2515, Count: 20, Type2: 5}
	petFood19 := petFood
	petFood19.Count = 19
	named := testWolf()
	named.Name = "Fang"
	const ground int32 = 0x10000050

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, inv, mobiusPetInfo(testWolf()))

		f.expect(msg(0x89).str("Fang").b)
		f.send(mobiusPetInfo(named))

		f.expect(msg(0x8b).i32(food.ObjectID).i32(20).b)
		f.send(mobiusInventoryUpdate(InventoryUpdate{Items: []ItemChange{{Change: ItemModified, Item: food30}}}))
		f.send(mobiusPetInventoryUpdate(PetItemChange{Change: ItemAdded, PetItem: petFood}))

		f.expect(msg(0x8a).i32(petFood.ObjectID).b)
		f.send(mobiusPetInventoryUpdate(PetItemChange{Change: ItemModified, PetItem: petFood19}))

		f.expect(msg(0x8c).i32(petFood.ObjectID).i32(19).i32(0).b)
		f.send(mobiusPetInventoryUpdate(PetItemChange{Change: ItemRemoved, PetItem: petFood19}))

		f.expect(msg(0x8f).i32(ground).b)
		f.send(mobiusPetItemList(petFood))

		f.expect(msg(0x45).i32(16).i32(1).u8(0).b)
		f.send(mobiusPetStatusUpdate(PetStatusUpdate{SummonType: SummonPet, ObjectID: named.ObjectID, Title: "Hero", Level: 15}))
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)

	if err := c.ChangePetName(ctx, "Fang"); err != nil {
		t.Fatal(err)
	}
	if p, err := Await(ctx, c, func(p PetInfo) bool { return p.Name != "" }); err != nil || p != named {
		t.Fatalf("PetInfo after ChangePetName = %+v, %v", p, err)
	}

	if err := c.GiveItemToPet(ctx, food.ObjectID, 20); err != nil {
		t.Fatal(err)
	}
	if u, err := Await[PetInventoryUpdate](ctx, c, nil); err != nil || !reflect.DeepEqual(u.Items, []PetItemChange{{Change: ItemAdded, PetItem: petFood}}) {
		t.Fatalf("PetInventoryUpdate after GiveItemToPet = %+v, %v", u, err)
	}
	if n := c.World().ItemCount(2515); n != 30 {
		t.Errorf("food left %d, want 30", n)
	}

	if err := c.PetUseItem(ctx, petFood.ObjectID); err != nil {
		t.Fatal(err)
	}
	if u, err := Await[PetInventoryUpdate](ctx, c, nil); err != nil || u.Items[0].Count != 19 {
		t.Fatalf("PetInventoryUpdate after PetUseItem = %+v, %v", u, err)
	}

	if err := c.GetItemFromPet(ctx, petFood.ObjectID, 19); err != nil {
		t.Fatal(err)
	}
	if u, err := Await[PetInventoryUpdate](ctx, c, nil); err != nil || u.Items[0].Change != ItemRemoved {
		t.Fatalf("PetInventoryUpdate after GetItemFromPet = %+v, %v", u, err)
	}

	if err := c.PetPickUp(ctx, ground); err != nil {
		t.Fatal(err)
	}
	if l, err := Await[PetItemList](ctx, c, nil); err != nil || len(l.Items) != 1 || l.Items[0] != petFood {
		t.Fatalf("PetItemList after PetPickUp = %+v, %v", l, err)
	}

	if err := c.RequestActionUse(ctx, ActionPetAttack, true, false); err != nil {
		t.Fatal(err)
	}
	if s, err := Await[PetStatusUpdate](ctx, c, nil); err != nil || s.ObjectID != named.ObjectID || s.Level != 15 {
		t.Fatalf("PetStatusUpdate after the attack = %+v, %v", s, err)
	}
}
