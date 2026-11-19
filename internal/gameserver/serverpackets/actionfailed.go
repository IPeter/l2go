package serverpackets

// OpActionFailed is the opcode of ActionFailed.
const OpActionFailed = 0x25

// NewActionFailedPacket builds ActionFailed. It tells the client that its
// last request was refused, which unlocks its controls; a request answered
// with nothing can leave the client stuck.
func NewActionFailedPacket() []byte {
	return []byte{OpActionFailed}
}
