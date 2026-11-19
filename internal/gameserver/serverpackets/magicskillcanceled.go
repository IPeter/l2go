package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpMagicSkillCanceled is the opcode of MagicSkillCanceled (Mobius
// MAGIC_SKILL_CANCELD).
const OpMagicSkillCanceled = 0x49

// NewMagicSkillCanceledPacket builds MagicSkillCanceled (Mobius
// MagicSkillCanceled.writeImpl): the creature with the given object ID
// stops casting, its cast interrupted (Mobius abortCast).
func NewMagicSkillCanceledPacket(objectID int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpMagicSkillCanceled)
	w.WriteInt32(objectID)
	return w.Bytes()
}
