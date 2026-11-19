package serverpackets

import "testing"

// Mobius ExAutoSoulShot.writeImpl: c 0xFE, h 0x12; d itemId, d type (1 on,
// 0 off).
func TestExAutoSoulShotPacket(t *testing.T) {
	// Soulshot: No Grade on.
	m67Check(t, "on",
		NewExAutoSoulShotPacket(ExAutoSoulShot{ItemID: 1835, Enabled: true}),
		m67Hex(t, "fe 1200 2b070000 01000000"))
	// Blessed Spiritshot: No Grade off.
	m67Check(t, "off",
		NewExAutoSoulShotPacket(ExAutoSoulShot{ItemID: 3947}),
		m67Hex(t, "fe 1200 6b0f0000 00000000"))

	var g m67Golden
	g.c(0xfe).h(0x12).d(5790).d(1)
	m67Check(t, "beast soulshot", NewExAutoSoulShotPacket(ExAutoSoulShot{ItemID: 5790, Enabled: true}), g.b)
}
