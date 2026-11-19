package rsakey

import (
	"bytes"
	"crypto/rsa"
	"errors"
	"math/big"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
)

// testKey is one generated key pair shared by the tests, since generating
// keys is slow.
var testKey = sync.OnceValues(Generate)

func key(t testing.TB) *KeyPair {
	t.Helper()
	k, err := testKey()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return k
}

func randBlock(r *rand.Rand, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(r.Uint32())
	}
	return b
}

// sparse returns a BlockSize-byte slice that is zero except at the given
// offsets.
func sparse(vals map[int]byte) []byte {
	b := make([]byte, BlockSize)
	for off, v := range vals {
		b[off] = v
	}
	return b
}

// Scramble vectors worked out by hand from ScrambledKeyPair.scrambleModulus
// on inputs with only a few non-zero bytes. Each comment follows the four
// steps.
func TestScrambleGolden(t *testing.T) {
	tests := []struct {
		name    string
		in, out []byte
	}{
		{
			// 1: m[0]=02, m[4d]=01. 2: m[0d]^=m[4d] -> 01.
			// 3: m[34..37] are zero. 4: m[40]^=m[0] -> 02, m[4d]^=m[0d] -> 00.
			name: "swap and fold",
			in:   sparse(map[int]byte{0x00: 0x01, 0x4d: 0x02}),
			out:  sparse(map[int]byte{0x00: 0x02, 0x0d: 0x01, 0x40: 0x02}),
		},
		{
			// 1: nothing to swap. 2: m[34]^=m[74] -> 30.
			// 3: m[0d]^=m[34] -> 30. 4: m[4d]^=m[0d] -> 30, m[74]^=m[34] -> 10.
			name: "step three",
			in:   sparse(map[int]byte{0x34: 0x10, 0x74: 0x20}),
			out:  sparse(map[int]byte{0x0d: 0x30, 0x34: 0x30, 0x4d: 0x30, 0x74: 0x10}),
		},
		{
			// 1: m[4d..50] = 01 02 03 04, m[0..3] = 0.
			// 2: m[0d..10] ^= m[4d..50] -> 01 02 03 04.
			// 3: m[34..37] are zero. 4: m[4d..50] ^= m[0d..10] -> 0.
			name: "four byte swap",
			in:   sparse(map[int]byte{0x00: 0x01, 0x01: 0x02, 0x02: 0x03, 0x03: 0x04}),
			out:  sparse(map[int]byte{0x0d: 0x01, 0x0e: 0x02, 0x0f: 0x03, 0x10: 0x04}),
		},
	}
	for _, tt := range tests {
		in := bytes.Clone(tt.in)
		got, err := Scramble(in)
		if err != nil {
			t.Fatalf("%s: Scramble: %v", tt.name, err)
		}
		if !bytes.Equal(got, tt.out) {
			t.Errorf("%s: Scramble = %x, want %x", tt.name, got, tt.out)
		}
		if !bytes.Equal(in, tt.in) {
			t.Errorf("%s: Scramble modified its input", tt.name)
		}
		back, err := Unscramble(got)
		if err != nil {
			t.Fatalf("%s: Unscramble: %v", tt.name, err)
		}
		if !bytes.Equal(back, tt.in) {
			t.Errorf("%s: Unscramble = %x, want %x", tt.name, back, tt.in)
		}
	}
}

func TestScrambleRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for i := range 500 {
		mod := randBlock(r, BlockSize)
		s, err := Scramble(mod)
		if err != nil {
			t.Fatal(err)
		}
		back, err := Unscramble(s)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(back, mod) {
			t.Fatalf("case %d: Unscramble(Scramble(x)) = %x, want %x", i, back, mod)
		}
	}
	for _, n := range []int{0, 127, 129, 256} {
		if _, err := Scramble(make([]byte, n)); !errors.Is(err, ErrBlockSize) {
			t.Errorf("Scramble(%d bytes) = %v, want ErrBlockSize", n, err)
		}
		if _, err := Unscramble(make([]byte, n)); !errors.Is(err, ErrBlockSize) {
			t.Errorf("Unscramble(%d bytes) = %v, want ErrBlockSize", n, err)
		}
	}
}

func TestKeyPair(t *testing.T) {
	k := key(t)
	pub := k.PublicKey()
	if pub.N.BitLen() != Bits || pub.E != PublicExponent {
		t.Fatalf("key is %d bits with e=%d, want %d bits with e=%d", pub.N.BitLen(), pub.E, Bits, PublicExponent)
	}
	mod := pub.N.FillBytes(make([]byte, BlockSize))
	want, err := Scramble(mod)
	if err != nil {
		t.Fatal(err)
	}
	got := k.ScrambledModulus()
	if !bytes.Equal(got, want) {
		t.Errorf("ScrambledModulus = %x, want %x", got, want)
	}
	got[0] ^= 0xff
	if bytes.Equal(k.ScrambledModulus(), got) {
		t.Error("ScrambledModulus returned the internal slice")
	}

	// The client recovers the same public key from the scrambled modulus.
	clientPub, err := PublicKeyFromScrambled(k.ScrambledModulus())
	if err != nil {
		t.Fatal(err)
	}
	if !clientPub.Equal(pub) {
		t.Errorf("PublicKeyFromScrambled: N=%x E=%d, want N=%x E=%d", clientPub.N, clientPub.E, pub.N, pub.E)
	}
}

// Textbook RSA example (p=61, q=53): n=3233, e=17, d=2753, and
// 65^17 mod 3233 = 2790. It checks the raw arithmetic and the big-endian
// block layout.
func TestRawRSATextbook(t *testing.T) {
	n := big.NewInt(3233)
	plain := make([]byte, BlockSize)
	plain[BlockSize-1] = 65
	want := make([]byte, BlockSize)
	want[BlockSize-2], want[BlockSize-1] = 0x0a, 0xe6 // 2790

	got, err := EncryptRaw(&rsa.PublicKey{N: n, E: 17}, plain)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("EncryptRaw = %x, want %x", got, want)
	}

	k := &KeyPair{priv: &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: n, E: 17}, D: big.NewInt(2753)}}
	back, err := k.DecryptRaw(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(back, plain) {
		t.Errorf("DecryptRaw = %x, want %x", back, plain)
	}
}

// DecryptRaw (blinded) gives the same result as plain c^d mod n.
func TestDecryptRawMatchesTextbook(t *testing.T) {
	k := key(t)
	r := rand.New(rand.NewPCG(3, 4))
	for i := range 20 {
		c := randBlock(r, BlockSize)
		c[0] &= 0x7f // below 2^1023 <= n
		got, err := k.DecryptRaw(c)
		if err != nil {
			t.Fatal(err)
		}
		m := new(big.Int).Exp(new(big.Int).SetBytes(c), k.priv.D, k.priv.N)
		if want := m.FillBytes(make([]byte, BlockSize)); !bytes.Equal(got, want) {
			t.Fatalf("case %d: DecryptRaw = %x, want %x", i, got, want)
		}
	}
}

func TestRawRoundTrip(t *testing.T) {
	k := key(t)
	pub, err := PublicKeyFromScrambled(k.ScrambledModulus())
	if err != nil {
		t.Fatal(err)
	}
	r := rand.New(rand.NewPCG(5, 6))
	for i := range 20 {
		plain := randBlock(r, BlockSize*(1+i%2))
		for off := 0; off < len(plain); off += BlockSize {
			plain[off] &= 0x7f
		}
		enc, err := EncryptRaw(pub, plain)
		if err != nil {
			t.Fatal(err)
		}
		dec, err := k.DecryptRaw(enc)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(dec, plain) {
			t.Fatalf("case %d: DecryptRaw(EncryptRaw(x)) = %x, want %x", i, dec, plain)
		}
	}
}

func TestRawErrors(t *testing.T) {
	k := key(t)
	for _, n := range []int{0, 1, 127, 129, 255} {
		if _, err := k.DecryptRaw(make([]byte, n)); !errors.Is(err, ErrBlockSize) {
			t.Errorf("DecryptRaw(%d bytes) = %v, want ErrBlockSize", n, err)
		}
		if _, err := EncryptRaw(k.PublicKey(), make([]byte, n)); !errors.Is(err, ErrBlockSize) {
			t.Errorf("EncryptRaw(%d bytes) = %v, want ErrBlockSize", n, err)
		}
	}
	tooBig := bytes.Repeat([]byte{0xff}, BlockSize)
	if _, err := k.DecryptRaw(tooBig); !errors.Is(err, ErrRange) {
		t.Errorf("DecryptRaw(0xff...) = %v, want ErrRange", err)
	}
	if _, err := EncryptRaw(k.PublicKey(), tooBig); !errors.Is(err, ErrRange) {
		t.Errorf("EncryptRaw(0xff...) = %v, want ErrRange", err)
	}
	// The modulus itself is out of range too.
	mod := k.PublicKey().N.FillBytes(make([]byte, BlockSize))
	if _, err := k.DecryptRaw(mod); !errors.Is(err, ErrRange) {
		t.Errorf("DecryptRaw(n) = %v, want ErrRange", err)
	}
	// An error in the second block is reported.
	two := append(make([]byte, BlockSize), tooBig...)
	if _, err := k.DecryptRaw(two); !errors.Is(err, ErrRange) {
		t.Errorf("DecryptRaw(ok, 0xff...) = %v, want ErrRange", err)
	}
	if _, err := EncryptRaw(nil, make([]byte, BlockSize)); err == nil {
		t.Error("EncryptRaw(nil key) succeeded")
	}
}

func TestNewKeyPairChecks(t *testing.T) {
	small := new(big.Int).Lsh(big.NewInt(1), 511)
	small.Add(small, big.NewInt(1))
	n1024 := k1024(t)
	tests := []struct {
		name string
		priv *rsa.PrivateKey
	}{
		{"nil", nil},
		{"no modulus", &rsa.PrivateKey{D: big.NewInt(1)}},
		{"no private exponent", &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: n1024, E: PublicExponent}}},
		{"512 bits", &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: small, E: PublicExponent}, D: big.NewInt(1)}},
		{"e=3", &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: n1024, E: 3}, D: big.NewInt(1)}},
	}
	for _, tt := range tests {
		if _, err := NewKeyPair(tt.priv); err == nil {
			t.Errorf("NewKeyPair(%s) succeeded", tt.name)
		}
	}
}

// k1024 returns some 1024-bit number.
func k1024(t testing.TB) *big.Int {
	t.Helper()
	n := new(big.Int).Lsh(big.NewInt(1), Bits-1)
	return n.Add(n, big.NewInt(12345))
}

func TestPublicKeyFromScrambledErrors(t *testing.T) {
	if _, err := PublicKeyFromScrambled(make([]byte, BlockSize)); err == nil {
		t.Error("PublicKeyFromScrambled(zero modulus) succeeded")
	}
	if _, err := PublicKeyFromScrambled(make([]byte, 64)); !errors.Is(err, ErrBlockSize) {
		t.Errorf("PublicKeyFromScrambled(64 bytes) = %v, want ErrBlockSize", err)
	}
}

func TestPool(t *testing.T) {
	if _, err := New(0); err == nil {
		t.Error("New(0) succeeded")
	}
	p, err := New(3)
	if err != nil {
		t.Fatal(err)
	}
	if p.Len() != 3 {
		t.Fatalf("Len = %d, want 3", p.Len())
	}
	seen := map[*KeyPair]bool{}
	for range 200 {
		k := p.Pick()
		if k == nil {
			t.Fatal("Pick returned nil")
		}
		seen[k] = true
	}
	if len(seen) < 2 {
		t.Errorf("Pick returned %d distinct pairs in 200 picks, want more than one", len(seen))
	}
	mods := map[string]bool{}
	for k := range seen {
		mods[string(k.ScrambledModulus())] = true
	}
	if len(mods) != len(seen) {
		t.Error("two pairs in the pool share a modulus")
	}
}

func TestParseCredentialsGolden(t *testing.T) {
	old := make([]byte, 128)
	for i := range old {
		old[i] = 0xaa // outside the fields: ignored
	}
	copy(old[0x5e:], "admin\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	copy(old[0x6c:], "  s3cret \x00\x00\x00\x00\x00\x00\x00")

	full := make([]byte, 128)
	copy(full[0x5e:], "abcdefghijklmn")   // 14 bytes, up to 0x6b
	copy(full[0x6c:], "0123456789abcdef") // 16 bytes, up to 0x7b

	newer := make([]byte, 256)
	copy(newer[0x4e:], "averylongaccountname")
	copy(newer[0xce:], "tail")
	copy(newer[0xdc:], "pw")

	invalid := make([]byte, 128)
	copy(invalid[0x5e:], "a\xffb")

	tests := []struct {
		name  string
		block []byte
		want  Credentials
	}{
		{"128-byte block", old, Credentials{User: "admin", Password: "s3cret"}},
		{"full fields", full, Credentials{User: "abcdefghijklmn", Password: "0123456789abcdef"}},
		{"256-byte block", newer, Credentials{User: "averylongaccountnametail", Password: "pw"}},
		{"invalid UTF-8", invalid, Credentials{User: "a�b", Password: ""}},
		{"empty", make([]byte, 256), Credentials{}},
	}
	for _, tt := range tests {
		got, err := ParseCredentials(tt.block)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got != tt.want {
			t.Errorf("%s: ParseCredentials = %+v, want %+v", tt.name, got, tt.want)
		}
	}
	for _, n := range []int{0, 127, 129, 255, 384} {
		if _, err := ParseCredentials(make([]byte, n)); !errors.Is(err, ErrBlockSize) {
			t.Errorf("ParseCredentials(%d bytes) = %v, want ErrBlockSize", n, err)
		}
	}
}

func TestCredentialsBlock(t *testing.T) {
	long64 := strings.Repeat("a", 50) + strings.Repeat("b", 14)
	tests := []struct {
		user, password string
		size           int
	}{
		{"admin", "secret", 128},
		{"abcdefghijklmn", "0123456789abcdef", 128},
		{"", "", 128},
		{"admin", "secret", 256},
		{long64, "0123456789abcdef", 256},
		{strings.Repeat("x", 51), "p", 256},
	}
	for _, tt := range tests {
		b, err := CredentialsBlock(tt.user, tt.password, tt.size)
		if err != nil {
			t.Fatalf("CredentialsBlock(%q, %q, %d): %v", tt.user, tt.password, tt.size, err)
		}
		if len(b) != tt.size {
			t.Fatalf("CredentialsBlock size = %d, want %d", len(b), tt.size)
		}
		got, err := ParseCredentials(b)
		if err != nil {
			t.Fatal(err)
		}
		if got.User != tt.user || got.Password != tt.password {
			t.Errorf("round trip (%d) = %+v, want %q/%q", tt.size, got, tt.user, tt.password)
		}
	}

	bad := []struct {
		name, user, password string
		size                 int
	}{
		{"name too long (128)", "abcdefghijklmno", "p", 128},
		{"name too long (256)", long64 + "c", "p", 256},
		{"password too long", "a", "0123456789abcdefg", 128},
		{"leading space", " a", "p", 128},
		{"trailing zero", "a", "p\x00", 256},
		{"split on a space", strings.Repeat("a", 49) + " b", "p", 256},
	}
	for _, tt := range bad {
		if _, err := CredentialsBlock(tt.user, tt.password, tt.size); !errors.Is(err, ErrCredentials) {
			t.Errorf("%s: CredentialsBlock = %v, want ErrCredentials", tt.name, err)
		}
	}
	for _, size := range []int{-1, 0, 64, 129} {
		if _, err := CredentialsBlock("a", "b", size); !errors.Is(err, ErrBlockSize) {
			t.Errorf("CredentialsBlock(size %d) = %v, want ErrBlockSize", size, err)
		}
	}
}

// The full login path: the client reads the scrambled key from Init,
// encrypts its credentials, and the server decrypts and parses them.
func TestCredentialsEndToEnd(t *testing.T) {
	k := key(t)
	pub, err := PublicKeyFromScrambled(k.ScrambledModulus())
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{128, 256} {
		plain, err := CredentialsBlock("player1", "hunter22", size)
		if err != nil {
			t.Fatal(err)
		}
		enc, err := EncryptRaw(pub, plain)
		if err != nil {
			t.Fatal(err)
		}
		if len(enc) != size {
			t.Fatalf("EncryptRaw returned %d bytes, want %d", len(enc), size)
		}
		dec, err := k.DecryptRaw(enc)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ParseCredentials(dec)
		if err != nil {
			t.Fatal(err)
		}
		if want := (Credentials{User: "player1", Password: "hunter22"}); got != want {
			t.Errorf("%d-byte login: got %+v, want %+v", size, got, want)
		}
	}
}

func FuzzDecryptRaw(f *testing.F) {
	k := key(f)
	f.Add(make([]byte, BlockSize))
	f.Add(bytes.Repeat([]byte{0xff}, BlockSize))
	f.Add(make([]byte, 2*BlockSize))
	f.Add([]byte{1, 2, 3})
	f.Fuzz(func(t *testing.T, data []byte) {
		plain, err := k.DecryptRaw(data)
		if err != nil {
			return
		}
		if len(plain) != len(data) {
			t.Fatalf("DecryptRaw returned %d bytes for %d", len(plain), len(data))
		}
		_, _ = ParseCredentials(plain)
	})
}

func FuzzParseCredentials(f *testing.F) {
	f.Add(make([]byte, 128))
	f.Add(make([]byte, 256))
	f.Add([]byte("short"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseCredentials(data)
	})
}

func FuzzUnscramble(f *testing.F) {
	f.Add(make([]byte, BlockSize))
	f.Add([]byte{1})
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := Unscramble(data)
		if err != nil {
			return
		}
		s, err := Scramble(m)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(s, data) {
			t.Fatalf("Scramble(Unscramble(x)) = %x, want %x", s, data)
		}
		_, _ = PublicKeyFromScrambled(data)
	})
}

func BenchmarkDecryptRaw(b *testing.B) {
	k := key(b)
	c := make([]byte, BlockSize)
	c[BlockSize-1] = 1
	for b.Loop() {
		if _, err := k.DecryptRaw(c); err != nil {
			b.Fatal(err)
		}
	}
}
