package serverpackets

import "testing"

// Mobius ShortBuffStatusUpdate.writeImpl: c 0xF4; d skillId, d skillLevel,
// d duration.
func TestShortBuffStatusUpdatePacket(t *testing.T) {
	m67Check(t, "potion",
		NewShortBuffStatusUpdatePacket(ShortBuffStatusUpdate{SkillID: 2031, SkillLevel: 1, Seconds: 15}),
		m67Hex(t, "f4 ef070000 01000000 0f000000"))
	// The zero value is Mobius RESET_SHORT_BUFF.
	m67Check(t, "reset",
		NewShortBuffStatusUpdatePacket(ShortBuffStatusUpdate{}),
		m67Hex(t, "f4 00000000 00000000 00000000"))
}
