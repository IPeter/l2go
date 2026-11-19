package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// LoginFailReason is the reason code of the game server's LoginFail.
type LoginFailReason int32

// LoginFail reasons (Mobius gameserver LoginFail.java). Only the one L2Go
// sends is listed.
const (
	// LoginFailSystemErrorLoginLater: "System error, please log in again
	// later." Mobius sends it when the login server refuses the session
	// keys.
	LoginFailSystemErrorLoginLater LoginFailReason = 1
)

// NewLoginFailPacket builds LoginFail, which refuses AuthLogin. The server
// closes the connection after sending it.
func NewLoginFailPacket(reason LoginFailReason) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpLoginFail)
	w.WriteInt32(int32(reason))
	return w.Bytes()
}
