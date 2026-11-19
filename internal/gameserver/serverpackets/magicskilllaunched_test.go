package serverpackets

import "testing"

// Mobius MagicSkillLaunched.writeImpl:
//
//	c 0x76
//	d caster objectId, skillId, skillLevel
//	d targets.size(); per target d objectId

func TestMagicSkillLaunchedPacket(t *testing.T) {
	m := MagicSkillLaunched{
		CasterID: 0x10000001, SkillID: 1177, SkillLevel: 1,
		TargetIDs: []int32{0x60000001},
	}
	m67Check(t, "one target", NewMagicSkillLaunchedPacket(m), m67Hex(t, `
		76
		01000010
		99040000
		01000000
		01000000
		01000060`))

	// An area skill hits several targets, in the given order.
	m.TargetIDs = []int32{0x60000003, 0x60000001, 0x60000002}
	var g m67Golden
	g.c(0x76).d(0x10000001).d(1177).d(1)
	g.d(3).d(0x60000003).d(0x60000001).d(0x60000002)
	m67Check(t, "three targets", NewMagicSkillLaunchedPacket(m), g.b)

	// Mobius's constructors without targets send the caster.
	self := MagicSkillLaunched{CasterID: 0x10000001, SkillID: 2046, SkillLevel: 1, TargetIDs: []int32{0x10000001}}
	g = m67Golden{}
	g.c(0x76).d(0x10000001).d(2046).d(1)
	g.d(1).d(0x10000001)
	m67Check(t, "self", NewMagicSkillLaunchedPacket(self), g.b)
}

// An empty target list is sent as it is, with a count of 0.
func TestMagicSkillLaunchedPacketNoTargets(t *testing.T) {
	m := MagicSkillLaunched{CasterID: 0x10000001, SkillID: 1177, SkillLevel: 1}
	m67Check(t, "no targets", NewMagicSkillLaunchedPacket(m),
		m67Hex(t, "76 01000010 99040000 01000000 00000000"))
}
