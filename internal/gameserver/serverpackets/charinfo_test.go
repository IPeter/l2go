package serverpackets

import "testing"

func TestCharInfoPacket(t *testing.T) {
	c := CharInfo{
		X: 100, Y: -200, Z: -3000, VehicleID: 0x70000002, ObjectID: 0x10000005,
		Name: "Elmore", Race: 3, Female: false, BaseClassID: 44,
		Paperdoll: m3FullPaperdoll(), WeaponAugmentationID: 0x0a0b0c0d,
		PvPFlag: 1, Karma: 9, MAtkSpd: 213, PAtkSpd: 325,
		RunSpd: 117, WalkSpd: 70, SwimRunSpd: 51, SwimWalkSpd: 52, FlyRunSpd: 118, FlyWalkSpd: 71,
		MoveMultiplier: 1.25, AttackSpeedMultiplier: 1.21,
		CollisionRadius: 11, CollisionHeight: 28.5,
		HairStyle: 1, HairColor: 2, Face: 0, Title: "Invisible",
		ClanID: 21, ClanCrestID: 22, AllyID: 23, AllyCrestID: 24,
		Sitting: true, Running: true, InCombat: true, Dead: true, Invisible: true,
		MountType: 2, PrivateStoreType: 1,
		Cubics: []int16{4}, InPartyMatchRoom: true, AbnormalEffects: AbnormalStealth,
		RecomLeft: 0x0102, RecomHave: 300, ClassID: 46, MaxCP: 500, CurCP: 499.99,
		EnchantEffect: 16, Team: 1, ClanCrestLargeID: 25, Noble: true, HeroAura: true,
		Fishing: true, FishX: 1, FishY: 2, FishZ: 3,
		NameColor: 0x00ff00, Heading: 32768, PledgeClass: 4, PledgeType: 100,
		TitleColor: 0xffff77, CursedWeaponLevel: 2,
	}

	// Mobius CharInfo.writeImpl, in order.
	var g m3Golden
	g.c(0x03)
	g.d(100).d(-200).d(-3000)     // x, y, z
	g.d(0x70000002).d(0x10000005) // vehicle ID, object ID
	g.s("Elmore")                 // visible name
	g.d(3).d(0).d(44)             // race, female, base class
	// 12 × d item display ID: under, head, rhand, lhand, gloves, chest,
	// legs, feet, cloak, rhand, hair, hair2.
	for _, slot := range []int32{0, 6, 7, 8, 9, 10, 11, 12, 13, 7, 15, 16} {
		g.d(1000 + slot)
	}
	g.hn(4).d(0x0a0b0c0d).hn(12).d(0x0a0b0c0d).hn(4) // C6 shorts, augmentation twice
	g.d(1).d(9)                                      // PvP flag, karma
	g.d(213).d(325)                                  // casting speed, attack speed
	g.d(1).d(9)                                      // PvP flag, karma again
	g.d(117).d(70).d(51).d(52)                       // run, walk, swim run, swim walk
	g.d(118).d(71).d(118).d(71)                      // fly run, fly walk, twice
	g.f(1.25).f(m3AtkSpdMul).f(11).f(28.5)           // move mul, atk speed mul (float), collision
	g.d(1).d(2).d(0)                                 // hair style, hair color, face
	g.s("Invisible")                                 // title
	g.d(21).d(22).d(23).d(24)                        // clan, clan crest, ally, ally crest
	g.d(0)                                           // relation: in RelationChanged
	g.c(0)                                           // standing: 0 while sitting
	g.c(1).c(1).c(1).c(1)                            // running, in combat, dead, invisible
	g.c(2).c(1)                                      // mount type, store type
	g.h(1).h(4)                                      // cubic count, cubic ID
	g.c(1)                                           // in party match room
	g.d(0x00100000)                                  // abnormal effects
	g.c(0x02)                                        // recommendations left: the low byte
	g.h(300)                                         // recommendations received
	g.d(46).d(500).d(499)                            // class ID, max CP, (int) current CP
	g.c(16).c(1)                                     // enchant effect, team
	g.d(25)                                          // large clan crest
	g.c(1).c(1)                                      // noble, hero aura
	g.c(1).d(1).d(2).d(3)                            // fishing, fish x, y, z
	g.d(0x00ff00).d(32768)                           // name color, heading
	g.d(4).d(100).d(0xffff77).d(2)                   // pledge class, pledge type, title color, cursed weapon level
	m3Check(t, "CharInfo", NewCharInfoPacket(c), g.b)
}

// A zero CharInfo: standing (1), nothing equipped, no cubics.
func TestCharInfoPacketZero(t *testing.T) {
	var g m3Golden
	g.c(0x03).dn(5).s("").dn(3)
	g.dn(12)
	g.hn(4).d(0).hn(12).d(0).hn(4)
	g.dn(6).dn(8)
	g.f(0).f(0).f(0).f(0)
	g.dn(3).s("").dn(4).d(0)
	g.c(1).c(0).c(0).c(0).c(0).c(0).c(0)
	g.h(0).c(0).d(0).c(0).h(0)
	g.dn(3).c(0).c(0).d(0).c(0).c(0)
	g.c(0).dn(3).dn(2).dn(4)
	m3Check(t, "zero CharInfo", NewCharInfoPacket(CharInfo{}), g.b)
}
