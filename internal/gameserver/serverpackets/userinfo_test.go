package serverpackets

import (
	"math"
	"testing"
)

// m3FullPaperdoll fills every slot: slot i holds object ID 0x10000100+i and
// item ID 1000+i.
func m3FullPaperdoll() Paperdoll {
	var p Paperdoll
	for slot := range p {
		p[slot] = EquippedItem{ObjectID: 0x10000100 + int32(slot), ItemID: 1000 + int32(slot)}
	}
	return p
}

// float32(1.21) widened to a double, as Java widens the float
// getAttackSpeedMultiplier result in writeDouble.
var m3AtkSpdMul = math.Float64frombits(0x3FF35C2900000000)

func TestUserInfoPacket(t *testing.T) {
	u := UserInfo{
		X: -71338, Y: 258271, Z: -3104, VehicleID: 0x70000001, ObjectID: 0x10000001,
		Name: "Frostwind", Race: 1, Female: true, BaseClassID: 18, Level: 20, Exp: 123456789012,
		STR: 23, DEX: 34, CON: 24, INT: 37, WIT: 22, MEN: 25,
		MaxHP: 402, CurHP: 380.5, MaxMP: 210, CurMP: 99.49, SP: 12345,
		CurLoad: 1000, MaxLoad: 69000, WeaponEquipped: true,
		Paperdoll: m3FullPaperdoll(), WeaponAugmentationID: 0x01020304,
		PAtk: 35, PAtkSpd: 330, PDef: 70, Evasion: 36, Accuracy: 33, Critical: 41,
		MAtk: 25, MAtkSpd: 222, MDef: 52, PvPFlag: 1, Karma: 7,
		RunSpd: 125, WalkSpd: 80, SwimRunSpd: 51, SwimWalkSpd: 52, FlyRunSpd: 130, FlyWalkSpd: 81,
		MoveMultiplier: 1.1, AttackSpeedMultiplier: 1.21,
		CollisionRadius: 7.5, CollisionHeight: 24,
		HairStyle: 2, HairColor: 3, Face: 1, GM: true, Title: "Tester",
		ClanID: 11, ClanCrestID: 12, AllyID: 13, AllyCrestID: 14,
		Relation:  UserRelationLeader | UserRelationAttacker,
		MountType: 1, PrivateStoreType: 3, DwarvenCraft: true, PKKills: 4, PvPKills: 5,
		Cubics: []int16{1, 3}, InPartyMatchRoom: true, AbnormalEffects: AbnormalStealth | 0x2,
		InWater: true, ClanPrivileges: 0x7ffffe, RecomLeft: 3, RecomHave: 200,
		MountNpcID: 12526, InventoryLimit: 80, ClassID: 19, MaxCP: 250, CurCP: 249.9,
		EnchantEffect: 7, Team: 2, ClanCrestLargeID: 15, Noble: true, HeroAura: true,
		Fishing: true, FishX: 100, FishY: -200, FishZ: 300,
		NameColor: 0xffffff, Running: true, PledgeClass: 2, PledgeType: -1,
		TitleColor: 0xffff77, CursedWeaponLevel: 6,
	}

	// Mobius UserInfo.writeImpl, in order.
	var g m3Golden
	g.c(0x04)
	g.d(-71338).d(258271).d(-3104)        // x, y, z
	g.d(0x70000001).d(0x10000001)         // vehicle ID, object ID
	g.s("Frostwind")                      // visible name
	g.d(1).d(1).d(18).d(20)               // race, female, base class, level
	g.q(123456789012)                     // exp
	g.d(23).d(34).d(24).d(37).d(22).d(25) // STR, DEX, CON, INT, WIT, MEN
	g.d(402).d(381)                       // max HP, Math.round(current HP 380.5)
	g.d(210).d(99)                        // max MP, Math.round(current MP 99.49)
	g.d(12345).d(1000).d(69000)           // SP, current load, max load
	g.d(40)                               // 40 with a weapon
	// 17 × d object ID: under, rear, lear, neck, rfinger, lfinger, head,
	// rhand, lhand, gloves, chest, legs, feet, cloak, rhand, hair, face.
	order := []int32{0, 2, 1, 3, 5, 4, 6, 7, 8, 9, 10, 11, 12, 13, 7, 15, 14}
	for _, slot := range order {
		g.d(0x10000100 + slot)
	}
	for _, slot := range order { // 17 × d item display ID, same slots
		g.d(1000 + slot)
	}
	g.hn(14).d(0x01020304).hn(12).d(0x01020304).hn(4) // C6 shorts, augmentation twice
	g.d(35).d(330).d(70).d(36).d(33).d(41)            // P.Atk, atk speed, P.Def, evasion, accuracy, crit
	g.d(25).d(222).d(330).d(52)                       // M.Atk, casting speed, atk speed again, M.Def
	g.d(1).d(7)                                       // PvP flag, karma
	g.d(125).d(80).d(51).d(52)                        // run, walk, swim run, swim walk
	g.d(130).d(81).d(130).d(81)                       // fly run, fly walk, twice
	g.f(1.1).f(m3AtkSpdMul).f(7.5).f(24)              // move mul, atk speed mul (float), collision
	g.d(2).d(3).d(1)                                  // hair style, hair color, face
	g.d(1)                                            // GM (builder level)
	g.s("Tester")                                     // title
	g.d(11).d(12).d(13).d(14)                         // clan, clan crest, ally, ally crest
	g.d(0x1c0)                                        // relation
	g.c(1).c(3).c(1)                                  // mount type, store type, dwarven craft
	g.d(4).d(5)                                       // PK kills, PvP kills
	g.h(2).h(1).h(3)                                  // cubic count, cubic IDs
	g.c(1)                                            // in party match room
	g.d(0x00100002)                                   // abnormal effects
	g.c(1)                                            // in water
	g.d(0x7ffffe)                                     // clan privileges
	g.h(3).h(200)                                     // recommendations left, received
	g.d(1012526)                                      // mount NPC ID + 1,000,000
	g.h(80)                                           // inventory limit
	g.d(19).d(0)                                      // class ID, special effects
	g.d(250).d(249)                                   // max CP, (int) current CP 249.9
	g.c(7).c(2)                                       // enchant effect, team
	g.d(15)                                           // large clan crest
	g.c(1).c(1)                                       // noble, hero aura
	g.c(1).d(100).d(-200).d(300)                      // fishing, fish x, y, z
	g.d(0xffffff)                                     // name color
	g.c(1)                                            // running
	g.d(2).d(-1).d(0xffff77).d(6)                     // pledge class, pledge type, title color, cursed weapon level
	m3Check(t, "UserInfo", NewUserInfoPacket(u), g.b)
}

// A zero UserInfo: no weapon (20), no mount (0), no cubics.
func TestUserInfoPacketZero(t *testing.T) {
	var g m3Golden
	g.c(0x04).dn(5).s("")
	g.dn(4).q(0).dn(6)
	g.dn(7).d(20)
	g.dn(17).dn(17)
	g.hn(14).d(0).hn(12).d(0).hn(4)
	g.dn(10).dn(2).dn(8)
	g.f(0).f(0).f(0).f(0)
	g.dn(3).d(0).s("")
	g.dn(4).d(0).c(0).c(0).c(0).dn(2)
	g.h(0)
	g.c(0).d(0).c(0).d(0).h(0).h(0)
	g.d(0).h(0).d(0).d(0).d(0).d(0)
	g.c(0).c(0).d(0).c(0).c(0)
	g.c(0).dn(3).d(0).c(0).dn(4)
	m3Check(t, "zero UserInfo", NewUserInfoPacket(UserInfo{}), g.b)
}
