package serverpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// ProtocolRevision is the login protocol revision Init announces.
const ProtocolRevision = 0x0000c621

// Sizes of the Init fields.
const (
	ScrambledModulusSize = 128
	BlowfishKeySize      = 16
)

// ggConstants are the four GameGuard values Mobius sends in Init.
var ggConstants = [4]uint32{0x29dd954e, 0x77c39cfc, 0x97adb620, 0x07bde0f7}

// NewInit builds Init, the first packet of a login connection. It carries
// the connection's session ID, the scrambled RSA modulus (128 bytes) and
// the Blowfish session key (16 bytes). It is encrypted with the static key
// and an XOR pass instead of the session key.
func NewInit(sessionID int32, scrambledModulus, blowfishKey []byte) ([]byte, error) {
	if len(scrambledModulus) != ScrambledModulusSize {
		return nil, fmt.Errorf("serverpackets: Init: modulus is %d bytes, want %d", len(scrambledModulus), ScrambledModulusSize)
	}
	if len(blowfishKey) != BlowfishKeySize {
		return nil, fmt.Errorf("serverpackets: Init: Blowfish key is %d bytes, want %d", len(blowfishKey), BlowfishKeySize)
	}
	w := packets.NewWriter(1 + 4 + 4 + ScrambledModulusSize + 4*len(ggConstants) + BlowfishKeySize + 1)
	w.WriteUint8(OpInit)
	w.WriteInt32(sessionID)
	w.WriteUint32(ProtocolRevision)
	w.WriteBytes(scrambledModulus)
	for _, v := range ggConstants {
		w.WriteUint32(v)
	}
	w.WriteBytes(blowfishKey)
	w.WriteUint8(0) // terminates the key
	return w.Bytes(), nil
}
