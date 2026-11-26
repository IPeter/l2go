package serverpackets

import "testing"

func TestAgitDecoInfoPacket(t *testing.T) {
	// No function rented: c 0xf7, d hall, 13 zero bytes.
	m3Check(t, "bare AgitDecoInfo", NewAgitDecoInfoPacket(AgitDeco{HallID: 22, Grade: 2}),
		unhex(t, "f7 16000000 00 0000 00 00 00 00 00 0000 00 00"))
	// A grade-2 hall: HP 260 is the large fireplace, MP 15 the small
	// carpet, exp 40 the large chandelier, teleport 1 the small mirror,
	// curtains 2, item creation 3 (large), support 4 (small), front
	// platform 1.
	got := NewAgitDecoInfoPacket(AgitDeco{HallID: 22, Grade: 2, HP: 260, MP: 15, Exp: 40, Teleport: 1, Curtains: 2,
		ItemCreate: 3, Support: 4, FrontPlatform: 1})
	m3Check(t, "AgitDecoInfo", got, unhex(t, "f7 16000000 02 0101 02 01 00 02 02 0101 01 02"))
	// Item creation below level 3 is small whatever the grade (Mobius's
	// grade-0 test never matters).
	got = NewAgitDecoInfoPacket(AgitDeco{HallID: 31, Grade: 0, ItemCreate: 2})
	m3Check(t, "grade 0 AgitDecoInfo", got, unhex(t, "f7 1f000000 00 0000 00 00 00 00 01 0000 00 01"))
}
