package serverpackets

import "testing"

func TestEtcStatusUpdatePacket(t *testing.T) {
	// Mobius EtcStatusUpdate.writeImpl: c 0xF3, d charges, d weight
	// penalty, d chat blocked, d danger area, d grade penalty, d charm of
	// courage, d death penalty level.
	m3Check(t, "plain EtcStatusUpdate", NewEtcStatusUpdatePacket(EtcStatusUpdate{}),
		unhex(t, "f3 00000000 00000000 00000000 00000000 00000000 00000000 00000000"))

	e := EtcStatusUpdate{
		Charges: 7, WeightPenalty: 2, ChatBlocked: true, DangerArea: true,
		GradePenalty: true, CharmOfCourage: true, DeathPenaltyLevel: 15,
	}
	var g m3Golden
	g.c(0xf3).d(7).d(2).d(1).d(1).d(1).d(1).d(15)
	m3Check(t, "EtcStatusUpdate", NewEtcStatusUpdatePacket(e), g.b)
}
