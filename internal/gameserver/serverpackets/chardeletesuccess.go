package serverpackets

// OpCharDeleteSuccess is the opcode of CharDeleteSuccess.
const OpCharDeleteSuccess = 0x23

// NewCharDeleteSuccessPacket builds CharDeleteSuccess: the character is
// marked for deletion. Mobius sends CharSelectionInfo after it.
func NewCharDeleteSuccessPacket() []byte {
	return []byte{OpCharDeleteSuccess}
}
