// Package logincrypt implements the packet encryption of the Interlude
// login protocol, for both ends of the connection.
//
// The rules follow L2J Mobius (LoginEncryption and NewCrypt):
//
//   - The first server packet (Init) is encrypted with the static Blowfish
//     key StaticKey after an XOR pass (EncXORPass). It carries the 16-byte
//     session key that encrypts everything after it.
//   - Every later packet, in both directions, gets a checksum in its last
//     4-byte word (AppendChecksum) and is encrypted with the session key.
//   - Packets are zero padded to whole Blowfish blocks. See StaticPacketSize
//     and SessionPacketSize for the exact sizes.
//
// The server uses Session: EncryptServerPacket for outgoing packets and
// DecryptClientPacket for incoming ones. A test client uses ClientSession,
// which does the reverse: DecryptServerPacket, SetSessionKey and
// EncryptClientPacket.
//
// The packet bodies handled here exclude the 2-byte length prefix; framing
// is done by internal/packets.
package logincrypt

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/IPeter/l2go/internal/crypt/blowfish"
)

const (
	// BlockSize is the Blowfish block size. Encrypted packets are a whole
	// number of blocks.
	BlockSize = blowfish.BlockSize

	// KeySize is the length of the per-connection session key.
	KeySize = 16

	// ChecksumSize is the length of the checksum word.
	ChecksumSize = 4

	// Overheads from Mobius LoginEncryption: room reserved after the body
	// before padding, and the extra block added after padding.
	staticHeaderSize  = 8
	sessionHeaderSize = 4
	trailerSize       = 8
)

// StaticKey is the Blowfish key of the first server packet.
var StaticKey = [KeySize]byte{
	0x6b, 0x60, 0xcb, 0x5b, 0x82, 0xce, 0x90, 0xb1,
	0xcc, 0x2b, 0x6c, 0x55, 0x6c, 0x6c, 0x6c, 0x6c,
}

var (
	// ErrSize is returned for packets whose length the protocol does not
	// allow: empty, not a whole number of blocks, or too short.
	ErrSize = errors.New("logincrypt: invalid packet size")

	// ErrChecksum is returned when a decrypted packet has a wrong checksum.
	ErrChecksum = errors.New("logincrypt: invalid checksum")

	// ErrNoSessionKey is returned by ClientSession when a packet needs the
	// session key before SetSessionKey was called.
	ErrNoSessionKey = errors.New("logincrypt: session key not set")
)

// staticCipher is shared by all connections. blowfish.Cipher is safe for
// concurrent use.
var staticCipher = mustCipher(StaticKey[:])

func mustCipher(key []byte) *blowfish.Cipher {
	c, err := blowfish.NewCipher(key)
	if err != nil {
		panic(err)
	}
	return c
}

// paddedSize returns the encrypted size of an n-byte body after reserving
// header bytes, as Mobius LoginEncryption.encryptedSize does: the sum is
// padded up to the next block boundary (a whole extra block when it is
// already aligned), then one more block is added.
func paddedSize(n, header int) int {
	size := n + header
	size += BlockSize - size%BlockSize
	return size + trailerSize
}

// StaticPacketSize returns the encrypted size of an n-byte body sent as the
// first server packet (static key and XOR pass).
func StaticPacketSize(n int) int { return paddedSize(n, staticHeaderSize) }

// SessionPacketSize returns the encrypted size of an n-byte body sent with
// the session key and a checksum.
func SessionPacketSize(n int) int { return paddedSize(n, sessionHeaderSize) }

// NewSessionKey returns a random session key read from rnd. A nil rnd
// means crypto/rand.
func NewSessionKey(rnd io.Reader) ([KeySize]byte, error) {
	var key [KeySize]byte
	if rnd == nil {
		rnd = rand.Reader
	}
	if _, err := io.ReadFull(rnd, key[:]); err != nil {
		return key, fmt.Errorf("logincrypt: session key: %w", err)
	}
	return key, nil
}

// checksumOK reports whether b has a length that can carry a checksum, the
// same test as Mobius NewCrypt.verifyChecksum: a multiple of 4 and longer
// than the checksum itself.
func checksumOK(b []byte) bool {
	return len(b)%ChecksumSize == 0 && len(b) > ChecksumSize
}

// xorWords returns the XOR of the little-endian 32-bit words of b.
func xorWords(b []byte) uint32 {
	var sum uint32
	for i := 0; i+ChecksumSize <= len(b); i += ChecksumSize {
		sum ^= binary.LittleEndian.Uint32(b[i:])
	}
	return sum
}

// AppendChecksum stores the checksum of b in its last word, as Mobius
// NewCrypt.appendChecksum does: the XOR of all little-endian words before
// the last one. len(b) must be a multiple of 4 and greater than 4.
func AppendChecksum(b []byte) error {
	if !checksumOK(b) {
		return fmt.Errorf("%w: %d bytes for a checksum", ErrSize, len(b))
	}
	end := len(b) - ChecksumSize
	binary.LittleEndian.PutUint32(b[end:], xorWords(b[:end]))
	return nil
}

// VerifyChecksum reports whether the last word of b is the checksum of the
// words before it, as Mobius NewCrypt.verifyChecksum does.
func VerifyChecksum(b []byte) bool {
	if !checksumOK(b) {
		return false
	}
	end := len(b) - ChecksumSize
	return xorWords(b[:end]) == binary.LittleEndian.Uint32(b[end:])
}

// xorPassOK reports whether b can take an XOR pass: whole words, with the
// key word at len-8 after the first word.
func xorPassOK(b []byte) bool {
	return len(b)%4 == 0 && len(b) >= 12
}

// EncXORPass applies the XOR pass of the first server packet to b in place,
// as Mobius NewCrypt.encXORPass does with offset 0.
//
// The first word stays as it is. For each word from offset 4 up to len-8,
// key grows by the plain word (wrapping at 32 bits) and the word is XORed
// with the new key. The final key is stored at len-8. The last word is left
// unchanged. len(b) must be a multiple of 4 and at least 12.
func EncXORPass(b []byte, key uint32) error {
	if !xorPassOK(b) {
		return fmt.Errorf("%w: %d bytes for the XOR pass", ErrSize, len(b))
	}
	end := len(b) - 8
	for pos := 4; pos < end; pos += 4 {
		w := binary.LittleEndian.Uint32(b[pos:])
		key += w
		binary.LittleEndian.PutUint32(b[pos:], w^key)
	}
	binary.LittleEndian.PutUint32(b[end:], key)
	return nil
}

// DecXORPass reverses EncXORPass in place. It reads the final key at len-8
// and walks the words backwards down to offset 4. The key word itself is
// left in b.
func DecXORPass(b []byte) error {
	if !xorPassOK(b) {
		return fmt.Errorf("%w: %d bytes for the XOR pass", ErrSize, len(b))
	}
	end := len(b) - 8
	key := binary.LittleEndian.Uint32(b[end:])
	for pos := end - 4; pos >= 4; pos -= 4 {
		w := binary.LittleEndian.Uint32(b[pos:]) ^ key
		key -= w
		binary.LittleEndian.PutUint32(b[pos:], w)
	}
	return nil
}

// blocksOK reports whether b is a non-empty whole number of blocks.
func blocksOK(b []byte) bool {
	return len(b) > 0 && len(b)%BlockSize == 0
}

// sealSession pads body, appends the checksum and encrypts it with c.
func sealSession(c *blowfish.Cipher, body []byte) ([]byte, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: empty body", ErrSize)
	}
	buf := make([]byte, SessionPacketSize(len(body)))
	copy(buf, body)
	if err := AppendChecksum(buf); err != nil {
		return nil, err
	}
	if err := c.Encrypt(buf, buf); err != nil {
		return nil, fmt.Errorf("logincrypt: %w", err)
	}
	return buf, nil
}

// openSession decrypts b in place with c and verifies its checksum.
func openSession(c *blowfish.Cipher, b []byte) error {
	if !blocksOK(b) {
		return fmt.Errorf("%w: %d bytes", ErrSize, len(b))
	}
	if err := c.Decrypt(b, b); err != nil {
		return fmt.Errorf("logincrypt: %w", err)
	}
	if !VerifyChecksum(b) {
		return ErrChecksum
	}
	return nil
}

// Session is the server side of one login connection. It is not safe for
// concurrent use.
type Session struct {
	session    *blowfish.Cipher
	rnd        io.Reader
	staticSent bool
}

// NewSession returns the server state for a connection whose session key is
// key (KeySize bytes, sent to the client in Init). rnd supplies the XOR key
// of the first packet; nil means crypto/rand.
func NewSession(key []byte, rnd io.Reader) (*Session, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("logincrypt: session key is %d bytes, want %d", len(key), KeySize)
	}
	c, err := blowfish.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("logincrypt: %w", err)
	}
	if rnd == nil {
		rnd = rand.Reader
	}
	return &Session{session: c, rnd: rnd}, nil
}

// EncryptServerPacket returns body padded and encrypted for sending. The
// first call uses the static key with an XOR pass (for Init); later calls
// use the session key with a checksum. body is not modified.
func (s *Session) EncryptServerPacket(body []byte) ([]byte, error) {
	if s.staticSent {
		return sealSession(s.session, body)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("%w: empty body", ErrSize)
	}
	var k [4]byte
	if _, err := io.ReadFull(s.rnd, k[:]); err != nil {
		return nil, fmt.Errorf("logincrypt: XOR key: %w", err)
	}
	buf := make([]byte, StaticPacketSize(len(body)))
	copy(buf, body)
	if err := EncXORPass(buf, binary.LittleEndian.Uint32(k[:])); err != nil {
		return nil, err
	}
	if err := staticCipher.Encrypt(buf, buf); err != nil {
		return nil, fmt.Errorf("logincrypt: %w", err)
	}
	s.staticSent = true
	return buf, nil
}

// DecryptClientPacket decrypts a client packet in place with the session
// key and verifies its checksum. On success b holds the plain body followed
// by its padding and checksum. On error the content of b is undefined and
// the connection should be closed, as Mobius does.
func (s *Session) DecryptClientPacket(b []byte) error {
	return openSession(s.session, b)
}

// ClientSession is the client side of one login connection, used by test
// clients. It is not safe for concurrent use.
type ClientSession struct {
	session      *blowfish.Cipher
	staticOpened bool
}

// NewClientSession returns the client state for a new connection. The
// session key is not known until the first server packet is read.
func NewClientSession() *ClientSession {
	return &ClientSession{}
}

// SetSessionKey sets the session key taken from the Init packet.
func (c *ClientSession) SetSessionKey(key []byte) error {
	if len(key) != KeySize {
		return fmt.Errorf("logincrypt: session key is %d bytes, want %d", len(key), KeySize)
	}
	ci, err := blowfish.NewCipher(key)
	if err != nil {
		return fmt.Errorf("logincrypt: %w", err)
	}
	c.session = ci
	return nil
}

// DecryptServerPacket decrypts a server packet in place. The first packet
// is decrypted with the static key and its XOR pass is undone; later
// packets need the session key and must carry a valid checksum. On success
// b holds the plain body followed by padding (and the XOR key or checksum).
func (c *ClientSession) DecryptServerPacket(b []byte) error {
	if c.staticOpened {
		if c.session == nil {
			return ErrNoSessionKey
		}
		return openSession(c.session, b)
	}
	if !blocksOK(b) || !xorPassOK(b) {
		return fmt.Errorf("%w: %d bytes", ErrSize, len(b))
	}
	if err := staticCipher.Decrypt(b, b); err != nil {
		return fmt.Errorf("logincrypt: %w", err)
	}
	if err := DecXORPass(b); err != nil {
		return err
	}
	c.staticOpened = true
	return nil
}

// EncryptClientPacket returns body padded, given a checksum and encrypted
// with the session key. The padding follows SessionPacketSize. body is not
// modified.
func (c *ClientSession) EncryptClientPacket(body []byte) ([]byte, error) {
	if c.session == nil {
		return nil, ErrNoSessionKey
	}
	return sealSession(c.session, body)
}
