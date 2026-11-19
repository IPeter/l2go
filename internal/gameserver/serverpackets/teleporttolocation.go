package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpTeleportToLocation is the opcode of TeleportToLocation.
const OpTeleportToLocation = 0x28

// TeleportToLocation moves an object to a new position at once. When it is
// the player's own character, the client shows the loading screen and
// answers with Appearing.
type TeleportToLocation struct {
	ObjectID int32
	X, Y, Z  int32
	Heading  int32
}

// NewTeleportToLocationPacket builds TeleportToLocation (Mobius
// TeleportToLocation.writeImpl).
func NewTeleportToLocationPacket(t TeleportToLocation) []byte {
	w := packets.NewWriter(25)
	w.WriteUint8(OpTeleportToLocation)
	w.WriteInt32(t.ObjectID)
	w.WriteInt32(t.X)
	w.WriteInt32(t.Y)
	w.WriteInt32(t.Z)
	w.WriteInt32(0) // 0 fades the screen, 1 is instant; Mobius always fades
	w.WriteInt32(t.Heading)
	return w.Bytes()
}
