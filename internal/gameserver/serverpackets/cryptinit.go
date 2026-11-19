package serverpackets

import (
	"github.com/IPeter/l2go/internal/crypt/xor"
	"github.com/IPeter/l2go/internal/packets"
)

// NewCryptInitPacket builds CryptInit, which hands the XOR key to the
// client. It is sent unencrypted.
func NewCryptInitPacket(key [xor.KeySize]byte) []byte {
	w := packets.NewWriter(2 + xor.KeySize)
	w.WriteUint8(0x00) // CryptInit
	w.WriteUint8(0x01) // unknown
	w.WriteBytes(key[:])
	return w.Bytes()
}
