package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewLoginOk builds LoginOk with the login key pair of the session.
func NewLoginOk(loginOK1, loginOK2 int32) []byte {
	w := packets.NewWriter(49)
	w.WriteUint8(OpLoginOk)
	w.WriteInt32(loginOK1)
	w.WriteInt32(loginOK2)
	w.WriteUint32(0)
	w.WriteUint32(0)
	w.WriteUint32(0x000003ea)
	w.WriteUint32(0)
	w.WriteUint32(0)
	w.WriteUint32(0)
	w.WriteBytes(make([]byte, 16))
	return w.Bytes()
}
