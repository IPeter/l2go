package serverpackets

import "testing"

func TestAbnormalStatusUpdatePacket(t *testing.T) {
	// Mobius AbnormalStatusUpdate.writeImpl: c 0x7F, h count, then per
	// effect d skill display ID, h display level, d seconds left.
	m3Check(t, "empty AbnormalStatusUpdate", NewAbnormalStatusUpdatePacket(nil), unhex(t, "7f 0000"))

	effects := []AbnormalStatusEntry{
		{SkillID: 1204, SkillLevel: 2, Seconds: 1150},
		{SkillID: 4342, SkillLevel: 1, Seconds: -1},
	}
	var g m3Golden
	g.c(0x7f).h(2)
	g.d(1204).h(2).d(1150)
	g.d(4342).h(1).d(-1)
	m3Check(t, "AbnormalStatusUpdate", NewAbnormalStatusUpdatePacket(effects), g.b)
}
