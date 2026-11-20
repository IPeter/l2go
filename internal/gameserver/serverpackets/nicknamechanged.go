package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpNicknameChanged is the opcode of NicknameChanged.
const OpNicknameChanged = 0xcc

// NewNicknameChangedPacket builds NicknameChanged (Mobius
// NicknameChanged.writeImpl): the new title of a player, which the players
// around show under its name (Mobius broadcastTitleInfo).
func NewNicknameChangedPacket(objectID int32, title string) []byte {
	w := packets.NewWriter(1 + 4 + 2*len(title) + 2)
	w.WriteUint8(OpNicknameChanged)
	w.WriteInt32(objectID)
	w.WriteString(title)
	return w.Bytes()
}
