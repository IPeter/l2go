package serverpackets

import (
	"github.com/IPeter/l2go/internal/crypt/gamecrypt"
	"github.com/IPeter/l2go/internal/packets"
)

// NewKeyPacket builds KeyPacket, the answer to ProtocolVersion. It carries
// the random first half of the XOR key; the client adds the fixed second
// half. ok is false when the protocol version is not accepted; the server
// then closes the connection. KeyPacket is always sent unencrypted.
func NewKeyPacket(key [gamecrypt.KeySize]byte, ok bool, serverID int32) []byte {
	w := packets.NewWriter(23)
	w.WriteUint8(OpKeyPacket)
	if ok {
		w.WriteUint8(1)
	} else {
		w.WriteUint8(0)
	}
	w.WriteBytes(key[:gamecrypt.HalfSize])
	// Mobius writes its PacketEncryption setting here. L2Go always encrypts
	// (retail does too), so this is 1.
	w.WriteInt32(1)
	w.WriteInt32(serverID)
	w.WriteUint8(1)
	w.WriteInt32(0) // obfuscation key: none
	return w.Bytes()
}
