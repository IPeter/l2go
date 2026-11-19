package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpEnchantResult is the opcode of EnchantResult.
const OpEnchantResult = 0x81

// NewEnchantResultPacket builds EnchantResult (Mobius
// EnchantResult.writeImpl). Mobius sends the new enchant level after a
// successful enchant and 0 otherwise, for example when a warehouse transfer
// cancels an enchant in progress.
func NewEnchantResultPacket(result int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpEnchantResult)
	w.WriteInt32(result)
	return w.Bytes()
}
