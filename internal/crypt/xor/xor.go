// Package xor implements the 8-byte rolling XOR cipher of the C1 game
// protocol.
//
// Each byte is XORed with a key byte and with the previous encrypted byte.
// After every packet the first four key bytes, read as a little-endian
// uint32, grow by the packet length. Input and output keys roll separately.
package xor

import "encoding/binary"

// KeySize is the cipher key length in bytes.
const KeySize = 8

// StaticKey is the fixed key L2Go sends in CryptInit.
var StaticKey = [KeySize]byte{0x94, 0x35, 0x00, 0x00, 0xa1, 0x6c, 0x54, 0x87}

// Cipher holds the input and output key state of one connection. It is not
// safe for concurrent use.
type Cipher struct {
	in  [KeySize]byte
	out [KeySize]byte
}

// NewCipher returns a cipher whose input and output keys both start as key.
func NewCipher(key [KeySize]byte) *Cipher {
	return &Cipher{in: key, out: key}
}

// Encrypt encrypts b in place with the output key and rolls that key.
func (c *Cipher) Encrypt(b []byte) {
	var prev byte
	for i := range b {
		prev = b[i] ^ c.out[i%KeySize] ^ prev
		b[i] = prev
	}
	roll(&c.out, len(b))
}

// Decrypt decrypts b in place with the input key and rolls that key.
func (c *Cipher) Decrypt(b []byte) {
	var prev byte
	for i := range b {
		enc := b[i]
		b[i] = enc ^ c.in[i%KeySize] ^ prev
		prev = enc
	}
	roll(&c.in, len(b))
}

func roll(key *[KeySize]byte, n int) {
	v := binary.LittleEndian.Uint32(key[:4])
	binary.LittleEndian.PutUint32(key[:4], v+uint32(n)) //nolint:gosec // wraps like the client does
}
