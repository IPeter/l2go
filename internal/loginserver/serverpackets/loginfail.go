package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// NewLoginFail builds LoginFail. The Interlude reason is a single byte.
func NewLoginFail(reason LoginFailReason) []byte {
	w := packets.NewWriter(2)
	w.WriteUint8(OpLoginFail)
	w.WriteUint8(uint8(reason))
	return w.Bytes()
}

// NewAccountKicked builds AccountKicked. Its reason is a 32-bit value.
func NewAccountKicked(reason AccountKickedReason) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpAccountKicked)
	w.WriteUint32(uint32(reason))
	return w.Bytes()
}
