package serverpackets

import "testing"

// Mobius ChooseInventoryItem.writeImpl: c 0x6F; d item ID.
func TestChooseInventoryItemPacket(t *testing.T) {
	m3Check(t, "Scroll: Enchant Weapon (Grade D)", NewChooseInventoryItemPacket(955), unhex(t, "6f bb030000"))
	m3Check(t, "Blessed Scroll: Enchant Weapon (Grade D)", NewChooseInventoryItemPacket(6575), unhex(t, "6f af190000"))
}
