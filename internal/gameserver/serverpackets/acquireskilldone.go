package serverpackets

// OpAcquireSkillDone is the opcode of AcquireSkillDone.
const OpAcquireSkillDone = 0x8e

// NewAcquireSkillDonePacket builds AcquireSkillDone (Mobius
// AcquireSkillDone.writeImpl, the opcode alone). Mobius sends it after the
// system message that a trainer has nothing more to teach the player.
func NewAcquireSkillDonePacket() []byte {
	return []byte{OpAcquireSkillDone}
}
