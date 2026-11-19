package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpCharDeleteFail is the opcode of CharDeleteFail.
const OpCharDeleteFail = 0x24

// CharDeleteFailReason is the reason code of CharDeleteFail. The client
// shows a fixed message for each.
type CharDeleteFailReason int32

// Reason codes (Mobius CharDeleteFail.java).
const (
	CharDeleteFailDeletionFailed CharDeleteFailReason = 1 // "Character deletion has failed."
	CharDeleteFailClanMember     CharDeleteFailReason = 2 // "You may not delete a clan member. ..."
	CharDeleteFailClanLeader     CharDeleteFailReason = 3 // "Clan leaders may not be deleted. ..."
)

// NewCharDeleteFailPacket builds CharDeleteFail. Mobius sends
// CharSelectionInfo after it.
func NewCharDeleteFailPacket(reason CharDeleteFailReason) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpCharDeleteFail)
	w.WriteInt32(int32(reason))
	return w.Bytes()
}
