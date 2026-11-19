package xor

import (
	"bytes"
	"math/rand/v2"
	"testing"
)

// Golden vectors worked out from Encryption.java (L2J Mobius C1) with the
// static key 94 35 00 00 a1 6c 54 87. They are consecutive packets on one
// connection, so the key rolls between them.
var golden = []struct{ plain, cipher []byte }{
	{
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
		[]byte{0x95, 0xa2, 0xa1, 0xa5, 0x01, 0x6b, 0x38, 0xb7, 0x2a},
	},
	{
		// Key bytes 0-3 are now 0x3594 + 9 = 0x359d.
		[]byte{0x0e},
		[]byte{0x93},
	},
}

func TestEncryptGolden(t *testing.T) {
	c := NewCipher(StaticKey)
	for i, g := range golden {
		b := bytes.Clone(g.plain)
		c.Encrypt(b)
		if !bytes.Equal(b, g.cipher) {
			t.Errorf("packet %d: Encrypt = % x, want % x", i, b, g.cipher)
		}
	}
}

func TestDecryptGolden(t *testing.T) {
	c := NewCipher(StaticKey)
	for i, g := range golden {
		b := bytes.Clone(g.cipher)
		c.Decrypt(b)
		if !bytes.Equal(b, g.plain) {
			t.Errorf("packet %d: Decrypt = % x, want % x", i, b, g.plain)
		}
	}
}

func TestSinglePacketGolden(t *testing.T) {
	// AuthLogin-like opcode followed by zeros: the running XOR shows.
	b := []byte{0x08, 0x00, 0x00, 0x00}
	NewCipher(StaticKey).Encrypt(b)
	if want := []byte{0x9c, 0xa9, 0xa9, 0xa9}; !bytes.Equal(b, want) {
		t.Errorf("Encrypt = % x, want % x", b, want)
	}
}

// Input and output keys roll independently.
func TestDirectionsAreIndependent(t *testing.T) {
	c := NewCipher(StaticKey)
	c.Encrypt(make([]byte, 100))
	b := bytes.Clone(golden[0].cipher)
	c.Decrypt(b)
	if !bytes.Equal(b, golden[0].plain) {
		t.Errorf("Decrypt after Encrypt used a rolled key: % x", b)
	}
}

func TestRoundTripManyPackets(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	server := NewCipher(StaticKey)
	client := NewCipher(StaticKey)
	for i := range 100 {
		plain := make([]byte, rng.IntN(300))
		for j := range plain {
			plain[j] = byte(rng.Uint32())
		}
		b := bytes.Clone(plain)
		if i%2 == 0 {
			server.Encrypt(b)
			client.Decrypt(b)
		} else {
			client.Encrypt(b)
			server.Decrypt(b)
		}
		if !bytes.Equal(b, plain) {
			t.Fatalf("packet %d (%d bytes): round trip failed", i, len(plain))
		}
	}
}

func TestKeyRollWraps(t *testing.T) {
	c := NewCipher([KeySize]byte{0xff, 0xff, 0xff, 0xff, 1, 2, 3, 4})
	c.Encrypt(make([]byte, 2))
	if want := [KeySize]byte{0x01, 0x00, 0x00, 0x00, 1, 2, 3, 4}; c.out != want {
		t.Errorf("out key = % x, want % x", c.out, want)
	}
	if c.in != [KeySize]byte{0xff, 0xff, 0xff, 0xff, 1, 2, 3, 4} {
		t.Errorf("in key changed: % x", c.in)
	}
}

func TestEmptyPacket(t *testing.T) {
	c := NewCipher(StaticKey)
	c.Encrypt(nil)
	c.Decrypt([]byte{})
	if c.in != StaticKey || c.out != StaticKey {
		t.Errorf("keys changed on empty packets: in % x, out % x", c.in, c.out)
	}
}

func BenchmarkEncrypt(b *testing.B) {
	c := NewCipher(StaticKey)
	buf := make([]byte, 256)
	b.SetBytes(int64(len(buf)))
	for b.Loop() {
		c.Encrypt(buf)
	}
}
