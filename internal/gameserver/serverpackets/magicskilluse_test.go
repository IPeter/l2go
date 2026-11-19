package serverpackets

import (
	"testing"
	"time"
)

// Mobius MagicSkillUse.writeImpl:
//
//	c 0x48
//	d caster objectId, target objectId
//	d skillId, skillLevel, hitTime, reuseDelay
//	d caster x, y, z
//	critical ? d 1, h 0 : d 0
//	d target x, y, z

// m67WindStrike is a mystic casting Wind Strike on a monster.
func m67WindStrike() MagicSkillUse {
	return MagicSkillUse{
		CasterID: 0x10000001, TargetID: 0x60000001,
		SkillID: 1177, SkillLevel: 1,
		HitTime: 4 * time.Second, ReuseDelay: 6 * time.Second,
		X: -84023, Y: 244581, Z: -3730,
		TargetX: -84100, TargetY: 244600, TargetZ: -3728,
	}
}

func TestMagicSkillUsePacket(t *testing.T) {
	var g m67Golden
	g.c(0x48).d(0x10000001).d(0x60000001)
	g.d(1177).d(1).d(4000).d(6000)
	g.d(-84023).d(244581).d(-3730)
	g.d(0)
	g.d(-84100).d(244600).d(-3728)
	m67Check(t, "Wind Strike", NewMagicSkillUsePacket(m67WindStrike()), g.b)

	// The same layout written out.
	m := MagicSkillUse{
		CasterID: 1, TargetID: 2, SkillID: 3, SkillLevel: 4,
		HitTime: 5 * time.Millisecond, ReuseDelay: 6 * time.Millisecond,
		X: 7, Y: 8, Z: 9, TargetX: 10, TargetY: 11, TargetZ: 12,
	}
	m67Check(t, "written out", NewMagicSkillUsePacket(m), m67Hex(t, `
		48
		01000000 02000000
		03000000 04000000 05000000 06000000
		07000000 08000000 09000000
		00000000
		0a000000 0b000000 0c000000`))
}

// A critical cast adds a zero short after the flag.
func TestMagicSkillUsePacketCritical(t *testing.T) {
	m := m67WindStrike()
	m.Critical = true
	var g m67Golden
	g.c(0x48).d(0x10000001).d(0x60000001)
	g.d(1177).d(1).d(4000).d(6000)
	g.d(-84023).d(244581).d(-3730)
	g.d(1).h(0)
	g.d(-84100).d(244600).d(-3728)
	m67Check(t, "critical", NewMagicSkillUsePacket(m), g.b)
}

// A buff on oneself: the caster is the target. The times are whole
// milliseconds, cut toward zero; no reuse is 0.
func TestMagicSkillUsePacketSelf(t *testing.T) {
	m := MagicSkillUse{
		CasterID: 0x10000001, TargetID: 0x10000001,
		SkillID: 1068, SkillLevel: 2, // Might
		HitTime: 1500*time.Millisecond + 999*time.Microsecond,
		X:       -71338, Y: 258271, Z: -3104,
		TargetX: -71338, TargetY: 258271, TargetZ: -3104,
	}
	var g m67Golden
	g.c(0x48).d(0x10000001).d(0x10000001)
	g.d(1068).d(2).d(1500).d(0)
	g.d(-71338).d(258271).d(-3104)
	g.d(0)
	g.d(-71338).d(258271).d(-3104)
	m67Check(t, "self buff", NewMagicSkillUsePacket(m), g.b)
}
