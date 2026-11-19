package blowfish

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"golang.org/x/crypto/blowfish" //nolint:staticcheck // reference implementation for the L2 wrapper
)

// reverseWords returns a copy of b with the bytes of every 4-byte word
// reversed. It is written differently from the package code on purpose.
func reverseWords(b []byte) []byte {
	out := slices.Clone(b)
	for chunk := range slices.Chunk(out, 4) {
		slices.Reverse(chunk)
	}
	return out
}

func unhex(t testing.TB, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Standard Blowfish vectors (Schneier, vectors.txt). The Lineage II
// variant gives the same result with the bytes of each word reversed.
var vectors = []struct{ key, plain, cipher string }{
	{"0000000000000000", "0000000000000000", "4ef997456198dd78"},
	{"ffffffffffffffff", "ffffffffffffffff", "51866fd5b85ecb8a"},
	{"3000000000000000", "1000000000000001", "7d856f9a613063f2"},
	{"1111111111111111", "1111111111111111", "2466dd878b963c9d"},
	{"0123456789abcdef", "1111111111111111", "61f9c3802281b096"},
	{"fedcba9876543210", "0123456789abcdef", "0aceab0fc6a0a28d"},
	{"7ca110454a1a6e57", "01a1d6d039776742", "59c68245eb05282b"},
}

func TestKnownAnswers(t *testing.T) {
	for _, v := range vectors {
		c, err := NewCipher(unhex(t, v.key))
		if err != nil {
			t.Fatal(err)
		}
		plain := reverseWords(unhex(t, v.plain))
		want := reverseWords(unhex(t, v.cipher))

		got := make([]byte, len(plain))
		if err := c.Encrypt(got, plain); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("key %s: Encrypt = %x, want %x", v.key, got, want)
		}
		back := make([]byte, len(got))
		if err := c.Decrypt(back, got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(back, plain) {
			t.Errorf("key %s: Decrypt = %x, want %x", v.key, back, plain)
		}
	}
}

// The Lineage II cipher equals x/crypto/blowfish on byte-swapped words,
// for any key and any number of blocks.
func TestMatchesStandardBlowfishOnSwappedWords(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range 200 {
		key := make([]byte, 1+rng.IntN(56))
		for j := range key {
			key[j] = byte(rng.Uint32())
		}
		data := make([]byte, BlockSize*rng.IntN(8))
		for j := range data {
			data[j] = byte(rng.Uint32())
		}

		std, err := blowfish.NewCipher(key)
		if err != nil {
			t.Fatal(err)
		}
		want := reverseWords(data)
		for b := 0; b < len(want); b += BlockSize {
			std.Encrypt(want[b:b+BlockSize], want[b:b+BlockSize])
		}
		want = reverseWords(want)

		c, err := NewCipher(key)
		if err != nil {
			t.Fatal(err)
		}
		got := slices.Clone(data)
		if err := c.Encrypt(got, got); err != nil { // in place
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("case %d (key %x): Encrypt = %x, want %x", i, key, got, want)
		}
		if err := c.Decrypt(got, got); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("case %d: Decrypt(Encrypt(x)) = %x, want %x", i, got, data)
		}
	}
}

func TestStaticLoginKey(t *testing.T) {
	// The Interlude static login key (logincrypt.StaticKey, 16 bytes).
	c, err := NewCipher([]byte{
		0x6b, 0x60, 0xcb, 0x5b, 0x82, 0xce, 0x90, 0xb1,
		0xcc, 0x2b, 0x6c, 0x55, 0x6c, 0x6c, 0x6c, 0x6c,
	})
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	plain := []byte("0123456789abcdef")
	enc := make([]byte, len(plain))
	if err := c.Encrypt(enc, plain); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(enc, plain) {
		t.Fatal("Encrypt did not change the data")
	}
	if !bytes.Equal(plain, []byte("0123456789abcdef")) {
		t.Fatal("Encrypt modified src")
	}
	dec := make([]byte, len(enc))
	if err := c.Decrypt(dec, enc); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, plain) {
		t.Errorf("Decrypt = %q, want %q", dec, plain)
	}
}

func TestBlocksAreIndependent(t *testing.T) {
	c, err := NewCipher([]byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	two := make([]byte, 2*BlockSize)
	if err := c.Encrypt(two, two); err != nil {
		t.Fatal(err)
	}
	// ECB: equal plain blocks give equal cipher blocks.
	if !bytes.Equal(two[:BlockSize], two[BlockSize:]) {
		t.Errorf("blocks differ: %x", two)
	}
}

func TestNewCipherKeySize(t *testing.T) {
	for _, n := range []int{0, 57} {
		if _, err := NewCipher(make([]byte, n)); err == nil {
			t.Errorf("NewCipher(%d-byte key) succeeded", n)
		}
	}
	for _, n := range []int{1, 16, 56} {
		if _, err := NewCipher(make([]byte, n)); err != nil {
			t.Errorf("NewCipher(%d-byte key) = %v", n, err)
		}
	}
}

func TestBlockSizeErrors(t *testing.T) {
	c, err := NewCipher([]byte("key"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		dst, src int
	}{
		{"not a whole block", 7, 7},
		{"one block plus one", 9, 9},
		{"dst too short", 8, 16},
	}
	for _, tt := range tests {
		dst, src := make([]byte, tt.dst), make([]byte, tt.src)
		if err := c.Encrypt(dst, src); !errors.Is(err, ErrBlockSize) {
			t.Errorf("%s: Encrypt = %v, want ErrBlockSize", tt.name, err)
		}
		if err := c.Decrypt(dst, src); !errors.Is(err, ErrBlockSize) {
			t.Errorf("%s: Decrypt = %v, want ErrBlockSize", tt.name, err)
		}
	}
	if err := c.Decrypt(nil, nil); err != nil {
		t.Errorf("Decrypt(empty) = %v, want nil", err)
	}
}

func FuzzDecrypt(f *testing.F) {
	f.Add([]byte("key"), []byte{})
	f.Add([]byte("[;'.]94-31==-%&@!^+]\x00"), make([]byte, 16))
	f.Fuzz(func(t *testing.T, key, data []byte) {
		c, err := NewCipher(key)
		if err != nil {
			return
		}
		out := make([]byte, len(data))
		if err := c.Decrypt(out, data); err != nil {
			return
		}
		if err := c.Encrypt(out, out); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out, data) {
			t.Fatalf("Encrypt(Decrypt(x)) = %x, want %x", out, data)
		}
	})
}

func BenchmarkEncrypt(b *testing.B) {
	c, err := NewCipher([]byte("[;'.]94-31==-%&@!^+]\x00"))
	if err != nil {
		b.Fatal(err)
	}
	buf := make([]byte, 256)
	b.SetBytes(int64(len(buf)))
	for b.Loop() {
		if err := c.Encrypt(buf, buf); err != nil {
			b.Fatal(err)
		}
	}
}
