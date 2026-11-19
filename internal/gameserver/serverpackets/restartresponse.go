package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpRestartResponse is the opcode of RestartResponse.
const OpRestartResponse = 0x5f

// NewRestartResponsePacket builds RestartResponse, the answer to
// RequestRestart. With ok the client goes back to character selection and
// CharSelectionInfo follows; without it the client stays in the world.
func NewRestartResponsePacket(ok bool) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpRestartResponse)
	putBool32(w, ok)
	return w.Bytes()
}
