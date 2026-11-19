package serverpackets

import "testing"

// Mobius EnchantResult.writeImpl: c 0x81; d result.
func TestEnchantResultPacket(t *testing.T) {
	m3Check(t, "cancelled", NewEnchantResultPacket(0), unhex(t, "81 00000000"))
	m3Check(t, "+4", NewEnchantResultPacket(4), unhex(t, "81 04000000"))
}
