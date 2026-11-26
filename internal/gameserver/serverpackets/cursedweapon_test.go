package serverpackets

import "testing"

// Mobius ExCursedWeaponList.writeImpl: c 0xFE, h 0x45, d count, d item ID
// per weapon, in the manager's map order.
func TestExCursedWeaponListPacket(t *testing.T) {
	var g m3Golden
	g.c(0xfe).h(0x45).d(2).d(8689).d(8190)
	m3Check(t, "ExCursedWeaponList", NewExCursedWeaponListPacket([]int32{8689, 8190}), g.b)
	m3Check(t, "empty ExCursedWeaponList", NewExCursedWeaponListPacket(nil), unhex(t, "fe 4500 00000000"))
}

// Mobius ExCursedWeaponLocation.writeImpl: c 0xFE, h 0x46, d count, then
// d item ID, d activated (1 held, 0 on the ground), d x, y, z per weapon;
// an empty list writes two zero ints.
func TestExCursedWeaponLocationPacket(t *testing.T) {
	list := []CursedWeaponLocation{
		{ItemID: 8689, X: 1, Y: -2, Z: 3},
		{ItemID: 8190, Activated: true, X: 83000, Y: 148000, Z: -3400},
	}
	var g m3Golden
	g.c(0xfe).h(0x46).d(2)
	g.d(8689).d(0).d(1).d(-2).d(3)
	g.d(8190).d(1).d(83000).d(148000).d(-3400)
	m3Check(t, "ExCursedWeaponLocation", NewExCursedWeaponLocationPacket(list), g.b)
	m3Check(t, "empty ExCursedWeaponLocation", NewExCursedWeaponLocationPacket(nil), unhex(t, "fe 4600 00000000 00000000"))
}

// Mobius ExRedSky.writeImpl: c 0xFE, h 0x40, d seconds.
func TestExRedSkyPacket(t *testing.T) {
	m3Check(t, "ExRedSky", NewExRedSkyPacket(10), unhex(t, "fe 4000 0a000000"))
}
