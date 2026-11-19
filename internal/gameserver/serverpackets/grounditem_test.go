package serverpackets

import "testing"

// Mobius DropItem.writeImpl: c 0x0C; d dropper object ID, item object ID,
// display ID; d x, y, z; d stackable; d count; d 1.
func TestDropItemPacket(t *testing.T) {
	var g m3Golden
	g.c(0x0c).d(0x60000001).d(0x10000005).d(57)
	g.d(-84023).d(244581).d(-3730)
	g.d(1).d(1234).d(1)
	m3Check(t, "adena from a monster", NewDropItemPacket(m5GroundAdena()), g.b)

	sword := GroundItem{ObjectID: 0x10000006, ItemID: 2369, X: 10, Y: -20, Z: 30, Count: 1, DropperID: 0x10000001}
	m3Check(t, "a sword from a player", NewDropItemPacket(sword), unhex(t, `
		0c
		01000010 06000010 41090000
		0a000000 ecffffff 1e000000
		00000000 01000000 01000000`))
}

// Mobius SpawnItem.writeImpl: c 0x0B; d object ID, display ID; d x, y, z;
// d stackable; d count; d 0. The dropper is not sent.
func TestSpawnItemPacket(t *testing.T) {
	var g m3Golden
	g.c(0x0b).d(0x10000005).d(57)
	g.d(-84023).d(244581).d(-3730)
	g.d(1).d(1234).d(0)
	m3Check(t, "adena", NewSpawnItemPacket(m5GroundAdena()), g.b)

	sword := GroundItem{ObjectID: 0x10000006, ItemID: 2369, X: 10, Y: -20, Z: 30, Count: 1}
	m3Check(t, "a sword", NewSpawnItemPacket(sword), unhex(t, `
		0b
		06000010 41090000
		0a000000 ecffffff 1e000000
		00000000 01000000 00000000`))
}

// Mobius Item.sendInfo: DropItem while the dropper is set, SpawnItem once
// it is back to 0.
func TestGroundItemPacket(t *testing.T) {
	dropped := m5GroundAdena()
	m3Check(t, "being dropped", NewGroundItemPacket(dropped), NewDropItemPacket(dropped))

	lying := dropped
	lying.DropperID = 0
	m3Check(t, "lying on the ground", NewGroundItemPacket(lying), NewSpawnItemPacket(lying))
}
