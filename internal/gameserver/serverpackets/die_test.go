package serverpackets

import "testing"

// Mobius Die.writeImpl:
//
//	c 0x06
//	d objectId
//	d canTeleport
//	canTeleport && clan != null && !jailed ?
//		d hideout, d castle, d siege HQ
//		: d 0, d 0, d 0
//	d sweepable
//	d fixed resurrection

func TestDiePacket(t *testing.T) {
	// A monster with a spoil to sweep.
	m67Check(t, "sweepable monster",
		NewDiePacket(Die{ObjectID: 0x60000001, Sweepable: true}),
		m67Hex(t, `
			06
			01000060
			00000000
			00000000 00000000 00000000
			01000000
			00000000`))

	// A player without a clan, with a feather.
	var g m67Golden
	g.c(0x06).d(0x10000001).d(1)
	g.d(0).d(0).d(0)
	g.d(0).d(1)
	m67Check(t, "player with a feather",
		NewDiePacket(Die{ObjectID: 0x10000001, CanTeleport: true, FixedRes: true}), g.b)

	// A clan member with every option.
	g = m67Golden{}
	g.c(0x06).d(0x10000002).d(1)
	g.d(1).d(1).d(1)
	g.d(0).d(1)
	m67Check(t, "every option",
		NewDiePacket(Die{
			ObjectID: 0x10000002, CanTeleport: true,
			ToClanHall: true, ToCastle: true, ToSiegeHQ: true, FixedRes: true,
		}), g.b)

	// Only the castle: the three clan values keep their order.
	g = m67Golden{}
	g.c(0x06).d(0x10000003).d(1)
	g.d(0).d(1).d(0)
	g.d(0).d(0)
	m67Check(t, "castle only",
		NewDiePacket(Die{ObjectID: 0x10000003, CanTeleport: true, ToCastle: true}), g.b)
}

// Mobius writes the clan options only when the player can teleport.
func TestDiePacketClanOptionsNeedTeleport(t *testing.T) {
	var g m67Golden
	g.c(0x06).d(0x10000004).d(0)
	g.d(0).d(0).d(0)
	g.d(1).d(1)
	m67Check(t, "cannot teleport",
		NewDiePacket(Die{
			ObjectID: 0x10000004, CanTeleport: false,
			ToClanHall: true, ToCastle: true, ToSiegeHQ: true,
			Sweepable: true, FixedRes: true,
		}), g.b)
}
