// Package blowfish implements the Blowfish variant used by the Lineage II
// login protocol.
//
// It is standard Blowfish (golang.org/x/crypto/blowfish) with each 32-bit
// half of a block read and written in little-endian order instead of
// big-endian. Blocks are processed independently (ECB).
package blowfish

import (
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/blowfish" //nolint:staticcheck // the Lineage II protocol requires Blowfish
)

// BlockSize is the Blowfish block size in bytes.
const BlockSize = blowfish.BlockSize

// ErrBlockSize is returned when the data is not a whole number of blocks or
// dst is shorter than src.
var ErrBlockSize = errors.New("blowfish: data is not a multiple of the block size")

// Cipher is a Lineage II Blowfish cipher. It is safe for concurrent use.
type Cipher struct {
	b *blowfish.Cipher
}

// NewCipher returns a cipher for key, which must be 1 to 56 bytes long.
func NewCipher(key []byte) (*Cipher, error) {
	b, err := blowfish.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("blowfish: %w", err)
	}
	return &Cipher{b: b}, nil
}

// Encrypt encrypts src into dst block by block. len(src) must be a multiple
// of BlockSize. dst and src may be the same slice.
func (c *Cipher) Encrypt(dst, src []byte) error {
	return c.process(dst, src, c.b.Encrypt)
}

// Decrypt decrypts src into dst block by block. len(src) must be a multiple
// of BlockSize. dst and src may be the same slice.
func (c *Cipher) Decrypt(dst, src []byte) error {
	return c.process(dst, src, c.b.Decrypt)
}

func (c *Cipher) process(dst, src []byte, fn func(dst, src []byte)) error {
	if len(src)%BlockSize != 0 || len(dst) < len(src) {
		return fmt.Errorf("%w: src %d bytes, dst %d bytes", ErrBlockSize, len(src), len(dst))
	}
	var block [BlockSize]byte
	for i := 0; i < len(src); i += BlockSize {
		swapWords(block[:], src[i:i+BlockSize])
		fn(block[:], block[:])
		swapWords(dst[i:i+BlockSize], block[:])
	}
	return nil
}

// swapWords reverses the byte order of both 32-bit words of a block.
func swapWords(dst, src []byte) {
	l := binary.LittleEndian.Uint32(src[0:4])
	r := binary.LittleEndian.Uint32(src[4:8])
	binary.BigEndian.PutUint32(dst[0:4], l)
	binary.BigEndian.PutUint32(dst[4:8], r)
}
