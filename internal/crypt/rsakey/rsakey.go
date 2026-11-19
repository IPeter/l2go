// Package rsakey implements the RSA part of the Interlude login: a pool of
// pre-generated key pairs, the modulus scrambling of the Init packet, raw
// (unpadded) RSA and the layout of the login credentials block.
//
// Server side: New builds the pool at startup, Pool.Pick chooses a pair per
// connection, KeyPair.ScrambledModulus goes into Init, KeyPair.DecryptRaw
// opens the RequestAuthLogin block and ParseCredentials reads the account
// name and password from it.
//
// Client side (test clients): PublicKeyFromScrambled reads the key from
// Init, CredentialsBlock builds the plain block and EncryptRaw encrypts it.
//
// The layout follows L2J Mobius (ScrambledKeyPair, RequestAuthLogin).
package rsakey

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand/v2"
	"strings"
)

const (
	// Bits is the RSA modulus size.
	Bits = 1024

	// BlockSize is the size of one RSA block and of the modulus in bytes.
	BlockSize = Bits / 8

	// PublicExponent is the public exponent (F4) the client assumes.
	PublicExponent = 65537

	// Credentials layouts, from Mobius RequestAuthLogin.
	userOffset         = 0x5e
	userSize           = 14
	passwordOffset     = 0x6c
	passwordSize       = 16
	newUserOffset      = 0x4e
	newUserSize        = 50
	newUserTailOffset  = 0xce
	newUserTailSize    = 14
	newPasswordOffset  = 0xdc
	newPasswordSize    = 16
	maxUserSize        = userSize
	maxNewUserSize     = newUserSize + newUserTailSize
	credentialsSizeOld = BlockSize
	credentialsSizeNew = 2 * BlockSize
)

var (
	// ErrBlockSize is returned for RSA input that is not a whole, non-empty
	// number of BlockSize blocks, and for a modulus or credentials block of
	// the wrong size.
	ErrBlockSize = errors.New("rsakey: invalid block size")

	// ErrRange is returned when an RSA block, read as a big-endian number,
	// is not smaller than the modulus.
	ErrRange = errors.New("rsakey: block out of range for the modulus")

	// ErrCredentials is returned by CredentialsBlock for an account name or
	// password that does not fit the layout.
	ErrCredentials = errors.New("rsakey: invalid credentials")
)

// KeyPair is a 1024-bit RSA key pair with its scrambled modulus. It is safe
// for concurrent use.
type KeyPair struct {
	priv      *rsa.PrivateKey
	scrambled []byte
}

// Generate creates a new KeyPair with a 1024-bit modulus and e = 65537.
// Generating keys is slow; call it at startup (see New).
func Generate() (*KeyPair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, Bits)
	if err != nil {
		return nil, fmt.Errorf("rsakey: generate: %w", err)
	}
	return NewKeyPair(priv)
}

// NewKeyPair wraps an existing private key. The modulus must be exactly
// 1024 bits and the public exponent 65537.
func NewKeyPair(priv *rsa.PrivateKey) (*KeyPair, error) {
	if priv == nil || priv.N == nil || priv.D == nil {
		return nil, errors.New("rsakey: incomplete private key")
	}
	if priv.N.BitLen() != Bits {
		return nil, fmt.Errorf("rsakey: modulus is %d bits, want %d", priv.N.BitLen(), Bits)
	}
	if priv.E != PublicExponent {
		return nil, fmt.Errorf("rsakey: public exponent is %d, want %d", priv.E, PublicExponent)
	}
	mod := priv.N.FillBytes(make([]byte, BlockSize))
	scrambled, err := Scramble(mod)
	if err != nil {
		return nil, err
	}
	return &KeyPair{priv: priv, scrambled: scrambled}, nil
}

// PublicKey returns the public half of the pair.
func (k *KeyPair) PublicKey() *rsa.PublicKey { return &k.priv.PublicKey }

// ScrambledModulus returns a copy of the scrambled modulus that the Init
// packet carries (BlockSize bytes).
func (k *KeyPair) ScrambledModulus() []byte { return bytes.Clone(k.scrambled) }

// DecryptRaw decrypts ciphertext with raw RSA (m = c^d mod n, no padding),
// like Java's "RSA/ECB/nopadding". The input is one or more BlockSize-byte
// blocks; each is decrypted on its own and the plain blocks are returned in
// order, each left-padded with zeros to BlockSize bytes.
//
// The exponentiation is blinded with a random factor, so its timing does
// not depend on the private exponent and the input together.
func (k *KeyPair) DecryptRaw(ciphertext []byte) ([]byte, error) {
	if err := checkBlocks(ciphertext); err != nil {
		return nil, err
	}
	n, d := k.priv.N, k.priv.D
	e := big.NewInt(int64(k.priv.E))
	out := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += BlockSize {
		c := new(big.Int).SetBytes(ciphertext[i : i+BlockSize])
		if c.Cmp(n) >= 0 {
			return nil, fmt.Errorf("%w (block %d)", ErrRange, i/BlockSize)
		}
		r, rInv, err := blindingFactor(n)
		if err != nil {
			return nil, err
		}
		// m = ((c * r^e)^d * r^-1) mod n = c^d mod n
		c.Mul(c, new(big.Int).Exp(r, e, n)).Mod(c, n)
		m := new(big.Int).Exp(c, d, n)
		m.Mul(m, rInv).Mod(m, n)
		m.FillBytes(out[i : i+BlockSize])
	}
	return out, nil
}

// blindingFactor returns a random r in [1, n) that is invertible mod n, and
// its inverse.
func blindingFactor(n *big.Int) (r, rInv *big.Int, err error) {
	for range 16 {
		r, err = rand.Int(rand.Reader, n)
		if err != nil {
			return nil, nil, fmt.Errorf("rsakey: blinding: %w", err)
		}
		if r.Sign() == 0 {
			continue
		}
		if rInv = new(big.Int).ModInverse(r, n); rInv != nil {
			return r, rInv, nil
		}
	}
	return nil, nil, errors.New("rsakey: blinding: no invertible factor found")
}

// EncryptRaw encrypts plain with raw RSA (c = m^e mod n, no padding), the
// client side of DecryptRaw. The input is one or more BlockSize-byte
// blocks, each smaller than the modulus when read as a big-endian number.
func EncryptRaw(pub *rsa.PublicKey, plain []byte) ([]byte, error) {
	if pub == nil || pub.N == nil {
		return nil, errors.New("rsakey: nil public key")
	}
	if err := checkBlocks(plain); err != nil {
		return nil, err
	}
	if pub.N.BitLen() > Bits {
		return nil, fmt.Errorf("rsakey: modulus is %d bits, want at most %d", pub.N.BitLen(), Bits)
	}
	e := big.NewInt(int64(pub.E))
	out := make([]byte, len(plain))
	for i := 0; i < len(plain); i += BlockSize {
		m := new(big.Int).SetBytes(plain[i : i+BlockSize])
		if m.Cmp(pub.N) >= 0 {
			return nil, fmt.Errorf("%w (block %d)", ErrRange, i/BlockSize)
		}
		new(big.Int).Exp(m, e, pub.N).FillBytes(out[i : i+BlockSize])
	}
	return out, nil
}

func checkBlocks(b []byte) error {
	if len(b) == 0 || len(b)%BlockSize != 0 {
		return fmt.Errorf("%w: %d bytes, want a multiple of %d", ErrBlockSize, len(b), BlockSize)
	}
	return nil
}

// Scramble returns the scrambled form of a big-endian, BlockSize-byte
// modulus, exactly as Mobius ScrambledKeyPair.scrambleModulus does:
//
//  1. swap bytes 0x00-0x03 with bytes 0x4d-0x50;
//  2. XOR the first 0x40 bytes with the last 0x40 bytes;
//  3. XOR bytes 0x0d-0x10 with bytes 0x34-0x37;
//  4. XOR the last 0x40 bytes with the first 0x40 bytes.
//
// mod is not modified.
func Scramble(mod []byte) ([]byte, error) {
	if len(mod) != BlockSize {
		return nil, fmt.Errorf("%w: modulus is %d bytes, want %d", ErrBlockSize, len(mod), BlockSize)
	}
	m := bytes.Clone(mod)
	for i := range 4 {
		m[i], m[0x4d+i] = m[0x4d+i], m[i]
	}
	for i := range 0x40 {
		m[i] ^= m[0x40+i]
	}
	for i := range 4 {
		m[0x0d+i] ^= m[0x34+i]
	}
	for i := range 0x40 {
		m[0x40+i] ^= m[i]
	}
	return m, nil
}

// Unscramble reverses Scramble: it runs the four steps backwards. scrambled
// is not modified.
func Unscramble(scrambled []byte) ([]byte, error) {
	if len(scrambled) != BlockSize {
		return nil, fmt.Errorf("%w: modulus is %d bytes, want %d", ErrBlockSize, len(scrambled), BlockSize)
	}
	m := bytes.Clone(scrambled)
	for i := range 0x40 {
		m[0x40+i] ^= m[i]
	}
	for i := range 4 {
		m[0x0d+i] ^= m[0x34+i]
	}
	for i := range 0x40 {
		m[i] ^= m[0x40+i]
	}
	for i := range 4 {
		m[i], m[0x4d+i] = m[0x4d+i], m[i]
	}
	return m, nil
}

// PublicKeyFromScrambled returns the public key whose scrambled modulus is
// scrambled, with e = 65537. It is what a client does with the key in Init.
func PublicKeyFromScrambled(scrambled []byte) (*rsa.PublicKey, error) {
	mod, err := Unscramble(scrambled)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(mod)
	if n.BitLen() != Bits {
		return nil, fmt.Errorf("rsakey: modulus is %d bits, want %d", n.BitLen(), Bits)
	}
	return &rsa.PublicKey{N: n, E: PublicExponent}, nil
}

// Pool is a fixed set of key pairs generated at startup. It is safe for
// concurrent use.
type Pool struct {
	pairs []*KeyPair
}

// New generates a pool of n key pairs (Mobius keeps 10). It takes a while,
// so call it once at startup.
func New(n int) (*Pool, error) {
	if n < 1 {
		return nil, fmt.Errorf("rsakey: pool size %d, want at least 1", n)
	}
	pairs := make([]*KeyPair, n)
	for i := range pairs {
		k, err := Generate()
		if err != nil {
			return nil, err
		}
		pairs[i] = k
	}
	return &Pool{pairs: pairs}, nil
}

// Len returns the number of key pairs in the pool.
func (p *Pool) Len() int { return len(p.pairs) }

// Pick returns a key pair chosen at random, for one connection.
func (p *Pool) Pick() *KeyPair {
	return p.pairs[mrand.IntN(len(p.pairs))] //nolint:gosec // choosing among our own keys needs no secrecy
}

// Credentials are the account name and password from RequestAuthLogin.
type Credentials struct {
	User     string
	Password string
}

// ParseCredentials reads the account name and password from a decrypted
// credentials block, as Mobius RequestAuthLogin does:
//
//   - 128 bytes (one RSA block): name at 0x5e (14 bytes), password at 0x6c
//     (16 bytes).
//   - 256 bytes (two RSA blocks, the newer client): name at 0x4e (50 bytes)
//     followed by 0xce (14 bytes), password at 0xdc (16 bytes).
//
// Each field is trimmed like Java's String.trim: bytes up to 0x20 (spaces,
// zero bytes and other control characters) are removed from both ends.
// Invalid UTF-8 is replaced with U+FFFD, as Java's decoder does.
func ParseCredentials(plain []byte) (Credentials, error) {
	switch len(plain) {
	case credentialsSizeOld:
		return Credentials{
			User:     field(plain, userOffset, userSize),
			Password: field(plain, passwordOffset, passwordSize),
		}, nil
	case credentialsSizeNew:
		return Credentials{
			User:     field(plain, newUserOffset, newUserSize) + field(plain, newUserTailOffset, newUserTailSize),
			Password: field(plain, newPasswordOffset, newPasswordSize),
		}, nil
	default:
		return Credentials{}, fmt.Errorf("%w: credentials block is %d bytes, want %d or %d",
			ErrBlockSize, len(plain), credentialsSizeOld, credentialsSizeNew)
	}
}

// field returns b[off:off+n] as a trimmed string.
func field(b []byte, off, n int) string {
	f := bytes.TrimFunc(b[off:off+n], func(r rune) bool { return r <= ' ' })
	return strings.ToValidUTF8(string(f), "�")
}

// CredentialsBlock builds the plain credentials block a client encrypts
// with EncryptRaw. size is 128 (one block) or 256 (two blocks, the newer
// client); see ParseCredentials for the layout. In the 256-byte layout a
// name longer than 50 bytes continues in the second field. All other bytes
// are zero, so every block is smaller than any 1024-bit modulus.
//
// The name and password must not start or end with a byte up to 0x20,
// since the server would trim it.
func CredentialsBlock(user, password string, size int) ([]byte, error) {
	if err := checkCredential("account name", user); err != nil {
		return nil, err
	}
	if err := checkCredential("password", password); err != nil {
		return nil, err
	}
	if len(password) > passwordSize {
		return nil, fmt.Errorf("%w: password is %d bytes, max %d", ErrCredentials, len(password), passwordSize)
	}
	var b []byte
	switch size {
	case credentialsSizeOld:
		if len(user) > maxUserSize {
			return nil, fmt.Errorf("%w: account name is %d bytes, max %d", ErrCredentials, len(user), maxUserSize)
		}
		b = make([]byte, size)
		copy(b[userOffset:], user)
		copy(b[passwordOffset:], password)
	case credentialsSizeNew:
		if len(user) > maxNewUserSize {
			return nil, fmt.Errorf("%w: account name is %d bytes, max %d", ErrCredentials, len(user), maxNewUserSize)
		}
		head, tail := user, ""
		if len(user) > newUserSize {
			head, tail = user[:newUserSize], user[newUserSize:]
			if err := checkCredential("account name split", head); err != nil {
				return nil, err
			}
			if err := checkCredential("account name split", tail); err != nil {
				return nil, err
			}
		}
		b = make([]byte, size)
		copy(b[newUserOffset:], head)
		copy(b[newUserTailOffset:], tail)
		copy(b[newPasswordOffset:], password)
	default:
		return nil, fmt.Errorf("%w: credentials block of %d bytes, want %d or %d",
			ErrBlockSize, size, credentialsSizeOld, credentialsSizeNew)
	}
	return b, nil
}

func checkCredential(what, s string) error {
	if s == "" {
		return nil
	}
	if s[0] <= ' ' || s[len(s)-1] <= ' ' {
		return fmt.Errorf("%w: %s starts or ends with a space or control byte", ErrCredentials, what)
	}
	return nil
}
