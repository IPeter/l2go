package serverpackets

// GroundItem is an item lying on the ground, as DropItem and SpawnItem show
// it.
type GroundItem struct {
	ObjectID int32
	ItemID   int32 // the display ID of the item template
	X, Y, Z  int32
	// Stackable makes the client show Count. The count is sent for every
	// item.
	Stackable bool
	Count     int32
	// DropperID is the object ID of the character that dropped the item, or
	// 0. Mobius keeps it only while the dropped item enters the world: the
	// players who see it fall get DropItem, later viewers SpawnItem.
	DropperID int32
}

// NewGroundItemPacket builds the packet that shows a ground item to a
// player, picked as Mobius Item.sendInfo does: DropItem when DropperID is
// set, SpawnItem otherwise.
func NewGroundItemPacket(it GroundItem) []byte {
	if it.DropperID != 0 {
		return NewDropItemPacket(it)
	}
	return NewSpawnItemPacket(it)
}
