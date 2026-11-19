package serverpackets

import "testing"

// The M6 and M7 packets that are the opcode and at most one object ID.

// Mobius AutoAttackStart.writeImpl: c 0x2B; d objectId (the creature that
// enters its combat stance).
func TestAutoAttackStartPacket(t *testing.T) {
	m67Check(t, "player", NewAutoAttackStartPacket(0x10000001), m67Hex(t, "2b 01000010"))
	m67Check(t, "monster", NewAutoAttackStartPacket(0x60000123), m67Hex(t, "2b 23010060"))
}

// Mobius AutoAttackStop.writeImpl: c 0x2C; d objectId.
func TestAutoAttackStopPacket(t *testing.T) {
	m67Check(t, "player", NewAutoAttackStopPacket(0x10000001), m67Hex(t, "2c 01000010"))
	m67Check(t, "monster", NewAutoAttackStopPacket(0x60000123), m67Hex(t, "2c 23010060"))
}

// Mobius Revive.writeImpl: c 0x07; d objectId.
func TestRevivePacket(t *testing.T) {
	m67Check(t, "Revive", NewRevivePacket(0x10000001), m67Hex(t, "07 01000010"))
}

// Mobius MagicSkillCanceled.writeImpl: c 0x49; d objectId.
func TestMagicSkillCanceledPacket(t *testing.T) {
	m67Check(t, "MagicSkillCanceled", NewMagicSkillCanceledPacket(0x10000001), m67Hex(t, "49 01000010"))
}

// Mobius AcquireSkillDone.writeImpl: c 0x8E and nothing else.
func TestAcquireSkillDonePacket(t *testing.T) {
	m67Check(t, "AcquireSkillDone", NewAcquireSkillDonePacket(), m67Hex(t, "8e"))
}
