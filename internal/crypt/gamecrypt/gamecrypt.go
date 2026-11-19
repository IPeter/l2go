// Package gamecrypt implements the XOR cipher of the Interlude game
// protocol, for both ends of the connection.
//
// The rules follow L2J Mobius (Encryption and BlowFishKeygen):
//
//   - The key is 16 bytes: 8 random bytes followed by KeyTail (NewKey). The
//     KeyPacket sends only the first 8 bytes; the client appends the same
//     tail (KeyFromHalf).
//   - Each byte is XORed with key[i&15] and with the previous encrypted
//     byte.
//   - After each packet, key bytes 8-11, read as a little-endian uint32,
//     grow by the packet length. Input and output keys roll separately.
//   - On the server, the first Encrypt call (the KeyPacket) leaves the data
//     plain and turns the cipher on; Decrypt does nothing until then.
//
// The server uses NewServerCipher. A test client uses NewClientCipher,
// which is on from the start, because the client builds its key from the
// KeyPacket it has already read.
package gamecrypt

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// KeySize is the cipher key length in bytes.
	KeySize = 16

	// HalfSize is the length of the random part of the key, which the
	// KeyPacket carries.
	HalfSize = 8

	// offsetIndex is where the rolling offset lives in the key.
	offsetIndex = 8
)

// KeyTail is the fixed second half of every key.
var KeyTail = [KeySize - HalfSize]byte{0xc8, 0x27, 0x93, 0x01, 0xa1, 0x6c, 0x31, 0x97}

// NewKey returns a new key: HalfSize bytes read from rnd followed by
// KeyTail. A nil rnd means crypto/rand.
func NewKey(rnd io.Reader) ([KeySize]byte, error) {
	var key [KeySize]byte
	if rnd == nil {
		rnd = rand.Reader
	}
	if _, err := io.ReadFull(rnd, key[:HalfSize]); err != nil {
		return key, fmt.Errorf("gamecrypt: key: %w", err)
	}
	copy(key[HalfSize:], KeyTail[:])
	return key, nil
}

// KeyFromHalf returns the full key for the HalfSize bytes a client reads
// from the KeyPacket.
func KeyFromHalf(half []byte) ([KeySize]byte, error) {
	var key [KeySize]byte
	if len(half) != HalfSize {
		return key, fmt.Errorf("gamecrypt: key half is %d bytes, want %d", len(half), HalfSize)
	}
	copy(key[:], half)
	copy(key[HalfSize:], KeyTail[:])
	return key, nil
}

// Cipher holds the input and output key state of one connection. It is not
// safe for concurrent use.
type Cipher struct {
	in, out [KeySize]byte
	enabled bool
}

// NewServerCipher returns the server cipher for key. Its first Encrypt
// call leaves the data plain (the KeyPacket) and turns the cipher on.
func NewServerCipher(key [KeySize]byte) *Cipher {
	return &Cipher{in: key, out: key}
}

// NewClientCipher returns the client cipher for key. It is on from the
// start: the client creates it after reading the plain KeyPacket.
func NewClientCipher(key [KeySize]byte) *Cipher {
	return &Cipher{in: key, out: key, enabled: true}
}

// Enabled reports whether Encrypt and Decrypt change data.
func (c *Cipher) Enabled() bool { return c.enabled }

// Encrypt encrypts b in place with the output key and rolls that key. If
// the cipher is not on yet, it turns it on and leaves b unchanged.
func (c *Cipher) Encrypt(b []byte) {
	if !c.enabled {
		c.enabled = true
		return
	}
	if len(b) == 0 {
		return
	}
	var prev byte
	for i := range b {
		prev = b[i] ^ c.out[i&15] ^ prev
		b[i] = prev
	}
	roll(&c.out, len(b))
}

// Decrypt decrypts b in place with the input key and rolls that key. It
// does nothing while the cipher is off.
func (c *Cipher) Decrypt(b []byte) {
	if !c.enabled || len(b) == 0 {
		return
	}
	var prev byte
	for i := range b {
		enc := b[i]
		b[i] = enc ^ c.in[i&15] ^ prev
		prev = enc
	}
	roll(&c.in, len(b))
}

// roll adds n to the little-endian uint32 at key[8:12], wrapping like the
// Java int does.
func roll(key *[KeySize]byte, n int) {
	v := binary.LittleEndian.Uint32(key[offsetIndex:])
	binary.LittleEndian.PutUint32(key[offsetIndex:], v+uint32(n)) //nolint:gosec // packet sizes fit; wraps like the client does
}
