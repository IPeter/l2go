package testclient

import (
	"reflect"
	"testing"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
)

// The summon packets round-trip between the server's codecs and the
// client's: what the server builds, the client decodes field by field, and
// what the client sends, the server parses.

func TestSummonServerCodecsRoundTrip(t *testing.T) {
	pet := serverpackets.PetInfo{
		Type: serverpackets.SummonTypePet, ObjectID: 0x10000020, DisplayID: 12077,
		X: -71300, Y: 258250, Z: -3104, Heading: 16384, MAtkSpd: 333, PAtkSpd: 278,
		RunSpd: 160, WalkSpd: 60, SwimRunSpd: 61, SwimWalkSpd: 62, FlyRunSpd: 63, FlyWalkSpd: 64,
		MoveMultiplier: 1.25, AttackSpeedMultiplier: 1.5, CollisionRadius: 9, CollisionHeight: 12.5,
		Weapon: 2375, Armor: 2376, ShowName: true, Running: true, InCombat: true,
		Appearance: serverpackets.SummonAppearanceDefault, Name: "Fang", Title: "Hero",
		PvPFlag: 1, Karma: 2, CurFed: 3000, MaxFed: 3600, CurHP: 900, MaxHP: 1000, CurMP: 300, MaxMP: 400,
		SP: 55, Level: 15, Exp: 100, ExpThisLevel: 200, ExpNextLevel: 300, CurLoad: 120, MaxLoad: 5000,
		PAtk: 101, PDef: 102, MAtk: 103, MDef: 104, Accuracy: 105, Evasion: 106, Critical: 107,
		Speed: 200, AbnormalEffects: 4, Mountable: true, Flying: true, Team: 2,
		SoulShotsPerHit: 2, SpiritShotsPerHit: 3,
	}
	checkDecode(t, "PetInfo", serverpackets.NewPetInfoPacket(pet), PetInfo{
		SummonType: SummonPet, ObjectID: 0x10000020, NpcTypeID: 1012077,
		X: -71300, Y: 258250, Z: -3104, Heading: 16384, MAtkSpd: 333, PAtkSpd: 278,
		RunSpd: 160, WalkSpd: 60, SwimRunSpd: 61, SwimWalkSpd: 62,
		FlyRunSpd: 63, FlyWalkSpd: 64, FlyRunSpd2: 63, FlyWalkSpd2: 64,
		MoveMultiplier: 1.25, AttackSpeedMultiplier: 1.5, CollisionRadius: 9, CollisionHeight: 12.5,
		RHand: 2375, Chest: 2376, ShowName: true, Running: true, InCombat: true,
		Summoned: 1, Name: "Fang", Title: "Hero", TitleColor: 1,
		PvpFlag: 1, Karma: 2, CurFed: 3000, MaxFed: 3600, CurHP: 900, MaxHP: 1000, CurMP: 300, MaxMP: 400,
		SP: 55, Level: 15, Exp: 100, ExpThisLevel: 100, ExpNextLevel: 300, CurLoad: 120, MaxLoad: 5000,
		PAtk: 101, PDef: 102, MAtk: 103, MDef: 104, Accuracy: 105, Evasion: 106, Critical: 107,
		Speed: 200, AtkSpd: 278, CastSpd: 333, AbnormalEffects: 4, Mountable: true, Environment: 2, Team: 2,
		SoulShotsPerHit: 2, SpiritShotsPerHit: 3,
	})

	// Other players see a summon as NpcInfo.
	summon := serverpackets.SummonInfo{
		ObjectID: 0x10000021, DisplayID: 14001, Attackable: true, X: 1, Y: 2, Z: 3, Heading: 4,
		MAtkSpd: 5, PAtkSpd: 6, RunSpd: 7, WalkSpd: 8, SwimRunSpd: 9, SwimWalkSpd: 10,
		MoveMultiplier: 1, AttackSpeedMultiplier: 1, CollisionRadius: 10, CollisionHeight: 20.5,
		Weapon: 13, Armor: 14, Dead: true, Summoned: true, Name: "Kat the Cat", Title: "Hero",
		PvPFlag: 1, Karma: 15, AbnormalEffects: 2, Invisible: true, ViewerGM: true, InWater: true,
		Team: 1, EnchantEffect: 16,
	}
	n := NpcInfo{
		ObjectID: 0x10000021, NpcTypeID: 1014001, Attackable: true, X: 1, Y: 2, Z: 3, Heading: 4,
		MAtkSpd: 5, PAtkSpd: 6, RunSpd: 7, WalkSpd: 8, SwimRunSpd: 9, SwimWalkSpd: 10,
		MoveMultiplier: 1, AttackSpeedMultiplier: 1, CollisionRadius: 10, CollisionHeight: 20.5,
		RHand: 13, Chest: 14, ShowName: 1, Running: true, AlikeDead: true, Summoned: 2,
		Name: "Kat the Cat", Title: "Hero", TitleColor: 1, PvpFlag: 1, Karma: 15,
		AbnormalEffects: 2 | 0x00100000, Environment: 1, Team: 1,
		CollisionRadius2: 10, CollisionHeight2: 20.5, EnchantEffect: 16,
	}
	checkDecode(t, "SummonInfo", serverpackets.NewSummonInfoPacket(summon), n)

	sword := serverpackets.ItemInfo{
		Type1: 0, ObjectID: 0x10000002, ItemID: 2369, Count: 1, Type2: 0, CustomType1: 11,
		Equipped: true, BodyPart: 0x80, Enchant: 3, CustomType2: 12, AugmentationID: 0x1234, Mana: 45,
	}
	food := serverpackets.ItemInfo{Type1: 4, ObjectID: 0x10000041, ItemID: 2515, Count: 20, Type2: 5, Mana: -1}
	petSword := PetItem{ObjectID: 0x10000002, ItemID: 2369, Count: 1, Equipped: true, BodyPart: 0x80, Enchant: 3}
	petFood := PetItem{Type1: 4, ObjectID: 0x10000041, ItemID: 2515, Count: 20, Type2: 5}
	checkDecode(t, "PetItemList", serverpackets.NewPetItemListPacket(serverpackets.PetItemList{
		Items: []serverpackets.ItemInfo{sword, food},
	}), PetItemList{Items: []PetItem{petSword, petFood}})
	checkDecode(t, "PetInventoryUpdate", serverpackets.NewPetInventoryUpdatePacket(serverpackets.PetInventoryUpdate{
		Items: []serverpackets.InventoryItem{
			{Change: serverpackets.ItemModified, Item: sword},
			{Change: serverpackets.ItemRemoved, Item: food},
		},
	}), PetInventoryUpdate{Items: []PetItemChange{
		{Change: ItemModified, PetItem: petSword},
		{Change: ItemRemoved, PetItem: petFood},
	}})

	checkDecode(t, "PetStatusShow", serverpackets.NewPetStatusShowPacket(serverpackets.PetStatusShow{
		Type: serverpackets.SummonTypeServitor,
	}), PetStatusShow{SummonType: SummonServitor})
	checkDecode(t, "PetStatusUpdate", serverpackets.NewPetStatusUpdatePacket(serverpackets.PetStatusUpdate{
		Type: serverpackets.SummonTypePet, ObjectID: 0x10000020, X: 1, Y: 2, Z: 3, Title: "Hero",
		CurFed: 4, MaxFed: 5, CurHP: 6, MaxHP: 7, CurMP: 8, MaxMP: 9, Level: 10,
		Exp: 100, ExpThisLevel: 200, ExpNextLevel: 300,
	}), PetStatusUpdate{
		SummonType: SummonPet, ObjectID: 0x10000020, X: 1, Y: 2, Z: 3, Title: "Hero",
		CurFed: 4, MaxFed: 5, CurHP: 6, MaxHP: 7, CurMP: 8, MaxMP: 9, Level: 10,
		Exp: 100, ExpThisLevel: 200, ExpNextLevel: 300,
	})
	checkDecode(t, "PetDelete", serverpackets.NewPetDeletePacket(serverpackets.PetDelete{
		Type: serverpackets.SummonTypePet, ObjectID: 0x10000020,
	}), PetDelete{SummonType: SummonPet, ObjectID: 0x10000020})
	checkDecode(t, "SetSummonRemainTime", serverpackets.NewSetSummonRemainTimePacket(serverpackets.SetSummonRemainTime{
		MaxTime: 1200, RemainingTime: 600,
	}), SetSummonRemainTime{MaxTime: 1200, RemainingTime: 600})
	checkDecode(t, "Ride", serverpackets.NewRidePacket(serverpackets.Ride{
		ObjectID: 0x10000001, Mounted: true, MountType: serverpackets.MountWyvern, MountNpcID: 12621, X: 1, Y: 2, Z: 3,
	}), Ride{ObjectID: 0x10000001, Mounted: true, MountType: MountWyvern, MountNpcTypeID: 1012621, X: 1, Y: 2, Z: 3})
	checkDecode(t, "Ride dismount", serverpackets.NewRidePacket(serverpackets.Ride{ObjectID: 0x10000001}),
		Ride{ObjectID: 0x10000001, MountNpcTypeID: 1000000})

	// The opcodes agree.
	ops := []struct {
		name         string
		server, mine byte
	}{
		{"Ride", serverpackets.OpRide, OpRide},
		{"PetStatusShow", serverpackets.OpPetStatusShow, OpPetStatusShow},
		{"PetInfo", serverpackets.OpPetInfo, OpPetInfo},
		{"PetItemList", serverpackets.OpPetItemList, OpPetItemList},
		{"PetInventoryUpdate", serverpackets.OpPetInventoryUpdate, OpPetInventoryUpdate},
		{"PetStatusUpdate", serverpackets.OpPetStatusUpdate, OpPetStatusUpdate},
		{"PetDelete", serverpackets.OpPetDelete, OpPetDelete},
		{"SetSummonRemainTime", serverpackets.OpSetSummonRemainTime, OpSetSummonRemainTime},
	}
	for _, o := range ops {
		if o.server != o.mine {
			t.Errorf("%s: server opcode %#x, client %#x", o.name, o.server, o.mine)
		}
	}
}

// summonParse checks that the server parses what the client sends: the
// opcode, then the body.
func summonParse[T any](t *testing.T, name string, raw []byte, op byte, parse func([]byte) (T, error), want T) {
	t.Helper()
	if raw[0] != op {
		t.Errorf("%s: opcode %#x, the server expects %#x", name, raw[0], op)
	}
	got, err := parse(raw[1:])
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: parsed %+v, want %+v", name, got, want)
	}
}

func TestSummonClientCodecsRoundTrip(t *testing.T) {
	summonParse(t, "RequestChangePetName", encodeString(OpRequestChangePetName, "Fang"),
		clientpackets.OpRequestChangePetName, clientpackets.ParseRequestChangePetName,
		clientpackets.RequestChangePetName{Name: "Fang"})
	summonParse(t, "RequestPetUseItem", encodeInts(OpRequestPetUseItem, 0x10000030),
		clientpackets.OpRequestPetUseItem, clientpackets.ParseRequestPetUseItem,
		clientpackets.RequestPetUseItem{ObjectID: 0x10000030})
	summonParse(t, "RequestGiveItemToPet", encodeInts(OpRequestGiveItemToPet, 0x10000031, 20),
		clientpackets.OpRequestGiveItemToPet, clientpackets.ParseRequestGiveItemToPet,
		clientpackets.RequestGiveItemToPet{ObjectID: 0x10000031, Count: 20})
	summonParse(t, "RequestGetItemFromPet", encodeRequestGetItemFromPet(0x10000032, 3),
		clientpackets.OpRequestGetItemFromPet, clientpackets.ParseRequestGetItemFromPet,
		clientpackets.RequestGetItemFromPet{ObjectID: 0x10000032, Count: 3})
	summonParse(t, "RequestPetGetItem", encodeInts(OpRequestPetGetItem, 0x10000033),
		clientpackets.OpRequestPetGetItem, clientpackets.ParseRequestPetGetItem,
		clientpackets.RequestPetGetItem{ObjectID: 0x10000033})
	summonParse(t, "RequestActionUse", encodeRequestActionUse(ActionServitorAttack, true, true),
		clientpackets.OpRequestActionUse, clientpackets.ParseRequestActionUse,
		clientpackets.RequestActionUse{ActionID: clientpackets.ActionUseServitorAttack, CtrlPressed: true, ShiftPressed: true})

	actions := []struct {
		name         string
		mine, server int32
	}{
		{"pet follow", ActionPetFollow, clientpackets.ActionUsePetFollow},
		{"pet attack", ActionPetAttack, clientpackets.ActionUsePetAttack},
		{"pet stop", ActionPetStop, clientpackets.ActionUsePetStop},
		{"unsummon pet", ActionUnsummonPet, clientpackets.ActionUseUnsummonPet},
		{"servitor follow", ActionServitorFollow, clientpackets.ActionUseServitorFollow},
		{"servitor attack", ActionServitorAttack, clientpackets.ActionUseServitorAttack},
		{"servitor stop", ActionServitorStop, clientpackets.ActionUseServitorStop},
		{"mount", ActionMountDismount, clientpackets.ActionUseMountDismount},
		{"unsummon servitor", ActionUnsummonServitor, clientpackets.ActionUseUnsummonServitor},
		{"servitor move", ActionServitorMoveToTarget, clientpackets.ActionUseServitorMoveToTarget},
		{"pet move", ActionPetMoveToTarget, clientpackets.ActionUsePetMoveToTarget},
	}
	for _, a := range actions {
		if a.mine != a.server {
			t.Errorf("%s: client %d, server %d", a.name, a.mine, a.server)
		}
	}
}
