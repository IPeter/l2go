package serverpackets

import "testing"

func TestExSendManorListPacket(t *testing.T) {
	// Mobius ExSendManorList.writeImpl: c 0xFE, h 0x1B, d count, then per
	// castle, sorted by ID: d castle ID, S lower-case name.
	m3Check(t, "empty ExSendManorList", NewExSendManorListPacket(nil), unhex(t, "fe 1b00 00000000"))

	castles := []ManorCastle{{3, "Giran"}, {1, "GLUDIO"}, {2, "Dion"}}
	var g m3Golden
	g.c(0xfe).h(0x1b).d(3)
	g.d(1).s("gludio").d(2).s("dion").d(3).s("giran")
	m3Check(t, "ExSendManorList", NewExSendManorListPacket(castles), g.b)
	if castles[0].ID != 3 || castles[1].Name != "GLUDIO" {
		t.Errorf("the input was changed: %+v", castles)
	}
}

// The default list is the Mobius castle table.
func TestExSendManorListInterludeCastles(t *testing.T) {
	var g m3Golden
	g.c(0xfe).h(0x1b).d(9)
	for i, name := range []string{"gludio", "dion", "giran", "oren", "aden", "innadril", "goddard", "rune", "schuttgart"} {
		g.d(int32(i + 1)).s(name)
	}
	m3Check(t, "ExSendManorList", NewExSendManorListPacket(InterludeCastles()), g.b)
}
