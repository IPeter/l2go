package serverpackets

import "testing"

// Mobius ExSetCompassZoneCode.writeImpl: c 0xFE, h 0x32; d zone type.
func TestExSetCompassZoneCodePacket(t *testing.T) {
	m67Check(t, "peace", NewExSetCompassZoneCodePacket(CompassPeace), m67Hex(t, "fe 3200 0c000000"))
	m67Check(t, "general", NewExSetCompassZoneCodePacket(CompassGeneral), m67Hex(t, "fe 3200 0f000000"))
}
