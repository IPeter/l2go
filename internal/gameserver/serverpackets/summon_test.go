package serverpackets

import "testing"

// The opcodes are those of Mobius ServerPackets, the constants those of
// Mobius Summon.getSummonType and MountType.
func TestSummonOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"Ride", OpRide, 0x86},
		{"PetStatusShow", OpPetStatusShow, 0xb0},
		{"PetInfo", OpPetInfo, 0xb1},
		{"PetItemList", OpPetItemList, 0xb2},
		{"PetInventoryUpdate", OpPetInventoryUpdate, 0xb3},
		{"PetStatusUpdate", OpPetStatusUpdate, 0xb5},
		{"PetDelete", OpPetDelete, 0xb6},
		{"SetSummonRemainTime", OpSetSummonRemainTime, 0xd1},
		{"Servitor.getSummonType", int(SummonTypeServitor), 1},
		{"Pet.getSummonType", int(SummonTypePet), 2},
		{"MountType.NONE", MountNone, 0},
		{"MountType.STRIDER", MountStrider, 1},
		{"MountType.WYVERN", MountWyvern, 2},
		{"teleported", int(SummonAppearanceTeleported), 0},
		{"default", int(SummonAppearanceDefault), 1},
		{"summoned", int(SummonAppearanceSummoned), 2},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// summonTestWolf is a pet wolf with a value of its own in every field.
func summonTestWolf() PetInfo {
	return PetInfo{
		Type: SummonTypePet, ObjectID: 0x10000020, DisplayID: 12077,
		X: -84000, Y: 244000, Z: -3700, Heading: 16384,
		MAtkSpd: 333, PAtkSpd: 278,
		RunSpd: 160, WalkSpd: 60, SwimRunSpd: 51, SwimWalkSpd: 52, FlyRunSpd: 53, FlyWalkSpd: 54,
		MoveMultiplier: 1.25, AttackSpeedMultiplier: 1.5, CollisionRadius: 9, CollisionHeight: 12.5,
		Weapon: 2375, Armor: 2376,
		ShowName: true, Running: true, InCombat: false, Dead: false,
		Summoned: false, Appearance: SummonAppearanceDefault,
		Name: "Fang", Title: "Owner",
		PvPFlag: 1, Karma: 7, CurFed: 3000, MaxFed: 3600,
		CurHP: 900, MaxHP: 1000, CurMP: 300, MaxMP: 400, SP: 55, Level: 15,
		Exp: 50000, ExpThisLevel: 40000, ExpNextLevel: 60000,
		CurLoad: 120, MaxLoad: 5000,
		PAtk: 101, PDef: 102, MAtk: 103, MDef: 104, Accuracy: 105, Evasion: 106, Critical: 107,
		Speed: 200, AbnormalEffects: 0x0004, Mountable: false,
		InWater: true, Flying: false, Team: 1,
		SoulShotsPerHit: 2, SpiritShotsPerHit: 3,
	}
}

// summonPetInfoGolden is Mobius PetInfo.writeImpl for p, with the bytes the
// packet computes given: the appearance byte, the environment byte and the
// "exp for this level" long.
func summonPetInfoGolden(p PetInfo, appearance, environment byte, expThisLevel int64) []byte {
	var g m3Golden
	g.c(0xb1)
	g.d(int32(p.Type)).d(p.ObjectID).d(p.DisplayID + 1000000)
	g.d(0) // 1=attackable
	g.d(p.X).d(p.Y).d(p.Z).d(p.Heading)
	g.d(0)
	g.d(p.MAtkSpd).d(p.PAtkSpd)
	g.d(p.RunSpd).d(p.WalkSpd).d(p.SwimRunSpd).d(p.SwimWalkSpd)
	g.d(p.FlyRunSpd).d(p.FlyWalkSpd).d(p.FlyRunSpd).d(p.FlyWalkSpd)
	g.f(p.MoveMultiplier).f(float64(p.AttackSpeedMultiplier))
	g.f(float64(p.CollisionRadius)).f(float64(p.CollisionHeight))
	g.d(p.Weapon).d(p.Armor).d(0)
	g.c(summonBool(p.ShowName)).c(summonBool(p.Running)).c(summonBool(p.InCombat)).c(summonBool(p.Dead))
	g.c(appearance)
	g.s(p.Name).s(p.Title)
	g.d(1)
	g.d(p.PvPFlag).d(p.Karma)
	g.d(p.CurFed).d(p.MaxFed)
	g.d(p.CurHP).d(p.MaxHP).d(p.CurMP).d(p.MaxMP)
	g.d(p.SP).d(p.Level)
	g.q(p.Exp).q(expThisLevel).q(p.ExpNextLevel)
	g.d(p.CurLoad).d(p.MaxLoad)
	g.d(p.PAtk).d(p.PDef).d(p.MAtk).d(p.MDef)
	g.d(p.Accuracy).d(p.Evasion).d(p.Critical)
	g.d(p.Speed).d(p.PAtkSpd).d(p.MAtkSpd)
	g.d(p.AbnormalEffects)
	g.h(int16(summonBool(p.Mountable)))
	g.c(environment)
	g.h(0)
	g.c(p.Team)
	g.d(p.SoulShotsPerHit).d(p.SpiritShotsPerHit)
	return g.b
}

// summonBool is a Java boolean as a byte.
func summonBool(v bool) byte {
	if v {
		return 1
	}
	return 0
}

func TestPetInfoPacket(t *testing.T) {
	wolf := summonTestWolf()
	m3Check(t, "PetInfo pet", NewPetInfoPacket(wolf), summonPetInfoGolden(wolf, 1, 1, 40000))

	// A servitor that just appeared, flying, whose experience is below the
	// start of its level: the summon animation wins over the value given,
	// and Mobius sends the experience as the 0% mark.
	s := wolf
	s.Type = SummonTypeServitor
	s.Name = ""
	s.Summoned = true
	s.Appearance = SummonAppearanceTeleported
	s.InWater, s.Flying = false, true
	s.ShowName, s.Running, s.InCombat, s.Dead, s.Mountable = false, false, true, true, true
	s.Exp, s.ExpThisLevel = 100, 200
	m3Check(t, "PetInfo servitor", NewPetInfoPacket(s), summonPetInfoGolden(s, 2, 2, 100))

	// Water wins over flying; teleported shows no animation.
	s.Summoned, s.InWater = false, true
	m3Check(t, "PetInfo teleported", NewPetInfoPacket(s), summonPetInfoGolden(s, 0, 1, 100))

	// Land, and an experience exactly at the 0% mark.
	s.InWater, s.Flying = false, false
	s.Exp, s.ExpThisLevel = 200, 200
	m3Check(t, "PetInfo on land", NewPetInfoPacket(s), summonPetInfoGolden(s, 0, 0, 200))
}

func TestPetInfoPacketSize(t *testing.T) {
	// Mobius sends nothing after the spiritshots: the size is that of the
	// transcribed writeImpl.
	p := PetInfo{Name: "ab", Title: "cde"}
	got := len(NewPetInfoPacket(p))
	want := len(summonPetInfoGolden(p, 0, 0, 0))
	if got != want {
		t.Errorf("PetInfo size %d, want %d", got, want)
	}
}

// summonTestInfo is a servitor as other players see it.
func summonTestInfo() SummonInfo {
	return SummonInfo{
		ObjectID: 0x10000021, DisplayID: 14001, Attackable: true,
		X: 1, Y: 2, Z: 3, Heading: 4,
		MAtkSpd: 5, PAtkSpd: 6,
		RunSpd: 7, WalkSpd: 8, SwimRunSpd: 9, SwimWalkSpd: 10, FlyRunSpd: 11, FlyWalkSpd: 12,
		MoveMultiplier: 1.5, AttackSpeedMultiplier: 0.75, CollisionRadius: 10, CollisionHeight: 20.5,
		Weapon: 13, Armor: 14, InCombat: true, Dead: false,
		Summoned: false, Appearance: SummonAppearanceDefault,
		Name: "Kat the Cat", Title: "Owner",
		PvPFlag: 2, Karma: 15, AbnormalEffects: 0x0002,
		InWater: false, Flying: true, Team: 2, EnchantEffect: 16,
	}
}

// summonInfoGolden is Mobius AbstractNpcInfo.SummonInfo.writeImpl for s,
// with the computed appearance, abnormal effects and environment bytes.
func summonInfoGolden(s SummonInfo, appearance byte, effects int32, environment byte) []byte {
	var g m3Golden
	g.c(0x16)
	g.d(s.ObjectID).d(s.DisplayID + 1000000)
	g.d(int32(summonBool(s.Attackable)))
	g.d(s.X).d(s.Y).d(s.Z).d(s.Heading)
	g.d(0)
	g.d(s.MAtkSpd).d(s.PAtkSpd)
	g.d(s.RunSpd).d(s.WalkSpd).d(s.SwimRunSpd).d(s.SwimWalkSpd)
	g.d(s.FlyRunSpd).d(s.FlyWalkSpd).d(s.FlyRunSpd).d(s.FlyWalkSpd)
	g.f(s.MoveMultiplier).f(float64(s.AttackSpeedMultiplier))
	g.f(float64(s.CollisionRadius)).f(float64(s.CollisionHeight))
	g.d(s.Weapon) // right hand
	g.d(s.Armor)  // chest
	g.d(0)        // left hand
	g.c(1)        // name above char
	g.c(1)        // always running
	g.c(summonBool(s.InCombat)).c(summonBool(s.Dead))
	g.c(appearance)
	g.s(s.Name).s(s.Title)
	g.d(1) // title color
	g.d(s.PvPFlag).d(s.Karma)
	g.d(effects)
	g.dn(4) // clan, crest, ally, ally crest
	g.c(environment)
	g.c(s.Team)
	g.f(float64(s.CollisionRadius)).f(float64(s.CollisionHeight))
	g.d(s.EnchantEffect)
	g.d(0)
	return g.b
}

func TestSummonInfoPacket(t *testing.T) {
	s := summonTestInfo()
	m3Check(t, "SummonInfo", NewSummonInfoPacket(s), summonInfoGolden(s, 1, 0x0002, 2))

	// Invisible: only a GM viewer sees the stealth effect.
	s.Invisible = true
	m3Check(t, "SummonInfo invisible", NewSummonInfoPacket(s), summonInfoGolden(s, 1, 0x0002, 2))
	s.ViewerGM = true
	m3Check(t, "SummonInfo invisible to a GM", NewSummonInfoPacket(s), summonInfoGolden(s, 1, 0x0002|0x00100000, 2))

	// Summoned, in water, dead, not attackable.
	s = summonTestInfo()
	s.Summoned, s.Appearance = true, SummonAppearanceTeleported
	s.InWater, s.Dead, s.InCombat, s.Attackable = true, true, false, false
	m3Check(t, "SummonInfo summoned", NewSummonInfoPacket(s), summonInfoGolden(s, 2, 0x0002, 1))

	// On land, teleported.
	s.Summoned, s.InWater, s.Flying = false, false, false
	m3Check(t, "SummonInfo teleported", NewSummonInfoPacket(s), summonInfoGolden(s, 0, 0x0002, 0))
}

// The NpcInfo reader must accept SummonInfo: it has the same length as an
// NpcInfo with the same strings.
func TestSummonInfoSameSizeAsNpcInfo(t *testing.T) {
	s := NewSummonInfoPacket(SummonInfo{Name: "a", Title: "bc"})
	n := NewNpcInfoPacket(NpcInfo{Name: "a", Title: "bc"})
	if len(s) != len(n) {
		t.Errorf("SummonInfo %d bytes, NpcInfo %d", len(s), len(n))
	}
}

func TestPetItemListPacket(t *testing.T) {
	m3Check(t, "empty PetItemList", NewPetItemListPacket(PetItemList{}), unhex(t, "b2 0000"))

	// Mobius PetItemList.writeImpl: c 0xB2, h count, then per item h type1,
	// d object ID, d item ID, d count, h type2, h 0, h equipped, d body part,
	// h enchant, h 0. The custom types, augmentation and mana of m5Sword
	// must not show.
	l := PetItemList{Items: []ItemInfo{m5Sword(), m5Shots()}}
	var g m3Golden
	g.c(0xb2).h(2)
	g.h(0).d(0x10000002).d(2369).d(1).h(0).h(0).h(1).d(0x80).h(3).h(0)
	g.h(4).d(0x10000003).d(1835).d(300).h(5).h(0).h(0).d(0).h(0).h(0)
	m3Check(t, "PetItemList", NewPetItemListPacket(l), g.b)
}

func TestPetInventoryUpdatePacket(t *testing.T) {
	m3Check(t, "empty PetInventoryUpdate", NewPetInventoryUpdatePacket(PetInventoryUpdate{}), unhex(t, "b3 0000"))

	u := PetInventoryUpdate{Items: []InventoryItem{
		{Change: ItemAdded, Item: m5Sword()},
		{Change: ItemRemoved, Item: m5Shots()},
	}}
	// Mobius PetInventoryUpdate.writeImpl: c 0xB3, h count, then per item
	// h change and the PetItemList item block.
	var g m3Golden
	g.c(0xb3).h(2)
	g.h(1).h(0).d(0x10000002).d(2369).d(1).h(0).h(0).h(1).d(0x80).h(3).h(0)
	g.h(3).h(4).d(0x10000003).d(1835).d(300).h(5).h(0).h(0).d(0).h(0).h(0)
	m3Check(t, "PetInventoryUpdate", NewPetInventoryUpdatePacket(u), g.b)
}

func TestPetStatusShowPacket(t *testing.T) {
	m3Check(t, "PetStatusShow pet", NewPetStatusShowPacket(PetStatusShow{Type: SummonTypePet}), unhex(t, "b0 02000000"))
	m3Check(t, "PetStatusShow servitor", NewPetStatusShowPacket(PetStatusShow{Type: SummonTypeServitor}), unhex(t, "b0 01000000"))
}

func TestPetStatusUpdatePacket(t *testing.T) {
	p := PetStatusUpdate{
		Type: SummonTypePet, ObjectID: 0x10000020, X: -84000, Y: 244000, Z: -3700,
		Title: "Owner", CurFed: 3000, MaxFed: 3600,
		CurHP: 900, MaxHP: 1000, CurMP: 300, MaxMP: 400, Level: 15,
		// Below the 0% mark: PetStatusUpdate sends it unchanged.
		Exp: 100, ExpThisLevel: 200, ExpNextLevel: 300,
	}
	// Mobius PetStatusUpdate.writeImpl: d type, d object ID, d x, y, z,
	// S title, d fed, d max fed, d hp, d max hp, d mp, d max mp, d level,
	// q exp, q exp for this level, q exp for next level.
	var g m3Golden
	g.c(0xb5).d(2).d(0x10000020).d(-84000).d(244000).d(-3700)
	g.s("Owner").d(3000).d(3600).d(900).d(1000).d(300).d(400).d(15)
	g.q(100).q(200).q(300)
	m3Check(t, "PetStatusUpdate", NewPetStatusUpdatePacket(p), g.b)
}

func TestPetDeletePacket(t *testing.T) {
	m3Check(t, "PetDelete", NewPetDeletePacket(PetDelete{Type: SummonTypeServitor, ObjectID: 0x10000021}),
		unhex(t, "b6 01000000 21000010"))
}

func TestSetSummonRemainTimePacket(t *testing.T) {
	// The maximum comes first.
	m3Check(t, "SetSummonRemainTime", NewSetSummonRemainTimePacket(SetSummonRemainTime{MaxTime: 1200, RemainingTime: 600}),
		unhex(t, "d1 b0040000 58020000"))
}

func TestRidePacket(t *testing.T) {
	// Mobius Ride.writeImpl: d object ID, d mounted, d mount type, d mount
	// NPC ID + 1000000, d x, y, z.
	var g m3Golden
	g.c(0x86).d(0x10000001).d(1).d(1).d(12526 + 1000000).d(-84000).d(244000).d(-3700)
	m3Check(t, "Ride strider", NewRidePacket(Ride{
		ObjectID: 0x10000001, Mounted: true, MountType: MountStrider, MountNpcID: 12526,
		X: -84000, Y: 244000, Z: -3700,
	}), g.b)

	// Dismounting: Mobius still adds the offset to the NPC ID 0.
	var d m3Golden
	d.c(0x86).d(0x10000001).d(0).d(0).d(1000000).d(1).d(2).d(3)
	m3Check(t, "Ride dismount", NewRidePacket(Ride{ObjectID: 0x10000001, X: 1, Y: 2, Z: 3}), d.b)
}
