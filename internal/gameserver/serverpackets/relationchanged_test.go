package serverpackets

import "testing"

func TestRelationChangedPacket(t *testing.T) {
	r := RelationChanged{
		ObjectID:       0x10000005,
		Relation:       RelationLeader | RelationInSiege | RelationEnemy,
		AutoAttackable: true,
		Karma:          120,
		PvPFlag:        1,
	}
	// Mobius RelationChanged.writeImpl: c 0xCE, d object ID, d relation,
	// d auto attackable, d karma, d PvP flag.
	var g m3Golden
	g.c(0xce).d(0x10000005).d(0x1280).d(1).d(120).d(1)
	m3Check(t, "RelationChanged", NewRelationChangedPacket(r), g.b)

	m3Check(t, "plain RelationChanged", NewRelationChangedPacket(RelationChanged{ObjectID: 7}),
		unhex(t, "ce 07000000 00000000 00000000 00000000 00000000"))
}

// The relation bits are those of Mobius RelationChanged.
func TestRelationBits(t *testing.T) {
	bits := map[string]int{
		"RelationPvPFlag":   RelationPvPFlag,
		"RelationHasKarma":  RelationHasKarma,
		"RelationLeader":    RelationLeader,
		"RelationInSiege":   RelationInSiege,
		"RelationAttacker":  RelationAttacker,
		"RelationAlly":      RelationAlly,
		"RelationEnemy":     RelationEnemy,
		"RelationMutualWar": RelationMutualWar,
		"Relation1SidedWar": Relation1SidedWar,
	}
	want := map[string]int{
		"RelationPvPFlag":   0x2,
		"RelationHasKarma":  0x4,
		"RelationLeader":    0x80,
		"RelationInSiege":   0x200,
		"RelationAttacker":  0x400,
		"RelationAlly":      0x800,
		"RelationEnemy":     0x1000,
		"RelationMutualWar": 0x8000,
		"Relation1SidedWar": 0x10000,
	}
	for name, v := range bits {
		if v != want[name] {
			t.Errorf("%s = %#x, want %#x", name, v, want[name])
		}
	}
}
