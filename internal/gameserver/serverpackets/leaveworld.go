package serverpackets

// OpLeaveWorld is the opcode of LeaveWorld.
const OpLeaveWorld = 0x7e

// NewLeaveWorldPacket builds LeaveWorld, the answer to Logout. The client
// closes the game; Mobius closes the connection shortly after.
func NewLeaveWorldPacket() []byte {
	return []byte{OpLeaveWorld}
}
